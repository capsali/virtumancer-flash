package console

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/capsali/virtumancer/internal/libvirt"
	"github.com/capsali/virtumancer/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now.
		// In production, you'd want to restrict this.
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// noVNC and SPICE require the "binary" subprotocol.
	Subprotocols: []string{"binary"},
}

// wsConnWrapper adapts a websocket.Conn to an io.ReadWriteCloser.
// This is necessary because websocket.Conn does not implement the standard
// io.Reader and io.Writer interfaces directly.
type wsConnWrapper struct {
	*websocket.Conn
	reader io.Reader
}

// Read implements the io.Reader interface. It reads from the current websocket
// message. If the current message is fully read, it fetches the next message.
func (w *wsConnWrapper) Read(p []byte) (n int, err error) {
	// If we have an active reader from a previous call, use it.
	if w.reader != nil {
		n, err = w.reader.Read(p)
		// If we've reached the end of the message, reset the reader.
		if err == io.EOF {
			w.reader = nil
			err = nil // Don't propagate EOF for this single message, wait for next.
		}
		// If we read something, return it immediately, even if there was an error.
		if n > 0 {
			return n, err
		}
	}

	// Get the next message reader from the websocket connection.
	mt, r, err := w.NextReader()
	if err != nil {
		return 0, err
	}

	// We only proxy binary and text messages. Other types are ignored.
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		return 0, nil // Effectively a non-blocking read if message type is wrong.
	}

	w.reader = r
	// Now that we have a new reader, read from it.
	n, err = w.reader.Read(p)
	if err == io.EOF {
		// We've finished this message, reset for the next Read call.
		w.reader = nil
		err = nil
	}
	return n, err
}

// Write implements the io.Writer interface. It writes data as a binary
// message to the websocket connection.
func (w *wsConnWrapper) Write(p []byte) (n int, err error) {
	// noVNC expects binary data for VNC.
	err = w.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close implements the io.Closer interface.
func (w *wsConnWrapper) Close() error {
	return w.Conn.Close()
}

// HandleConsole finds the VM's VNC console details and proxies the connection.
func HandleConsole(db *gorm.DB, connector *libvirt.Connector, w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade websocket for console: %v", err)
		return
	}
	defer wsConn.Close()

	// Wrap the websocket connection to make it an io.ReadWriteCloser
	wrappedWsConn := &wsConnWrapper{Conn: wsConn}

	// Get libvirt connection for the host
	lvConn, err := connector.GetConnection(hostID)
	if err != nil {
		log.Printf("Console proxy error: could not get libvirt connection for host %s: %v", hostID, err)
		return
	}

	// Find the domain (VM)
	domain, err := lvConn.DomainLookupByName(vmName)
	if err != nil {
		log.Printf("Console proxy error: could not find VM %s on host %s: %v", vmName, hostID, err)
		return
	}

	// Get the VM's XML definition to find graphics details
	xmlDesc, err := lvConn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		log.Printf("Console proxy error: failed to get XML for %s: %v", vmName, err)
		return
	}

	// Parse the XML to find the VNC port
	type Graphics struct {
		Type string `xml:"type,attr"`
		Port string `xml:"port,attr"`
		Host string `xml:"listen,attr"`
	}
	type DomainDef struct {
		Graphics []Graphics `xml:"devices>graphics"`
	}

	var def DomainDef
	if err := xml.Unmarshal([]byte(xmlDesc), &def); err != nil {
		log.Printf("Console proxy error: failed to parse XML for %s: %v", vmName, err)
		return
	}

	var vncPort, vncHost string
	for _, g := range def.Graphics {
		if strings.ToLower(g.Type) == "vnc" {
			vncPort = g.Port
			vncHost = g.Host
			break
		}
	}

	if vncPort == "" {
		log.Printf("Console proxy error: VNC not configured or enabled for VM %s", vmName)
		return
	}

	// Libvirt reports -1 for autoport, but we can't connect to that.
	if vncPort == "-1" {
		log.Printf("Console proxy error: VNC port is set to autoport (-1), cannot connect for VM %s", vmName)
		return
	}

	// *** FIX: If listen address is local, empty, or unspecified, use the host's actual address from the DB. ***
	if vncHost == "" || vncHost == "127.0.0.1" || vncHost == "0.0.0.0" || vncHost == "::" {
		var host storage.Host
		if result := db.First(&host, "id = ?", hostID); result.Error != nil {
			log.Printf("Console proxy error: could not find host %s in DB to determine address: %v", hostID, result.Error)
			return
		}
		// A simple way to get hostname from a libvirt URI like qemu+ssh://user@hostname/system
		parts := strings.SplitN(host.URI, "@", 2)
		if len(parts) > 1 {
			hostPart := strings.Split(parts[1], "/")[0]
			// Handle potential port in hostname, e.g., user@hostname:port/system
			if strings.Contains(hostPart, ":") {
				vncHost, _, _ = net.SplitHostPort(hostPart)
			} else {
				vncHost = hostPart
			}
		} else {
			log.Printf("Console proxy error: could not determine VNC host address from URI %s", host.URI)
			return
		}
		log.Printf("VNC listen address was local; resolved to hypervisor address: %s", vncHost)
	}

	targetAddr := fmt.Sprintf("%s:%s", vncHost, vncPort)
	log.Printf("Proxying console for %s to VNC target %s", vmName, targetAddr)

	// Dial the actual VNC service on the hypervisor
	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Console proxy error: failed to connect to VNC service at %s: %v", targetAddr, err)
		return
	}
	defer target.Close()

	// Start proxying data in both directions
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(target, wrappedWsConn)
	}()
	go func() {
		defer wg.Done()
		io.Copy(wrappedWsConn, target)
	}()

	wg.Wait()
	log.Printf("VNC console proxy session ended for %s", vmName)
}

// HandleSpiceConsole finds the VM's SPICE console details and proxies the connection.
func HandleSpiceConsole(db *gorm.DB, connector *libvirt.Connector, w http.ResponseWriter, r *http.Request) {
	hostID := chi.URLParam(r, "hostID")
	vmName := chi.URLParam(r, "vmName")

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade websocket for SPICE console: %v", err)
		return
	}
	defer wsConn.Close()

	// Wrap the websocket connection to make it an io.ReadWriteCloser.
	// SPICE-HTML5 client expects binary messages.
	wrappedWsConn := &wsConnWrapper{Conn: wsConn}

	// Get libvirt connection for the host
	lvConn, err := connector.GetConnection(hostID)
	if err != nil {
		log.Printf("SPICE proxy error: could not get libvirt connection for host %s: %v", hostID, err)
		return
	}

	// Find the domain (VM)
	domain, err := lvConn.DomainLookupByName(vmName)
	if err != nil {
		log.Printf("SPICE proxy error: could not find VM %s on host %s: %v", vmName, hostID, err)
		return
	}

	// Get the VM's XML definition to find graphics details
	xmlDesc, err := lvConn.DomainGetXMLDesc(domain, 0)
	if err != nil {
		log.Printf("SPICE proxy error: failed to get XML for %s: %v", vmName, err)
		return
	}

	// Parse the XML to find the SPICE port
	type Graphics struct {
		XMLName xml.Name `xml:"graphics"`
		Type    string   `xml:"type,attr"`
		Port    string   `xml:"port,attr"`
		TlsPort string   `xml:"tlsPort,attr"`
		Listen  string   `xml:"listen,attr"`
	}
	type DomainDef struct {
		XMLName  xml.Name   `xml:"domain"`
		Graphics []Graphics `xml:"devices>graphics"`
	}

	var def DomainDef
	if err := xml.Unmarshal([]byte(xmlDesc), &def); err != nil {
		log.Printf("SPICE proxy error: failed to parse XML for %s: %v", vmName, err)
		return
	}

	var spicePort, spiceHost string
	// Prioritize TLS port if available, otherwise fall back to regular port.
	for _, g := range def.Graphics {
		if strings.ToLower(g.Type) == "spice" {
			if g.TlsPort != "" && g.TlsPort != "-1" {
				spicePort = g.TlsPort
			} else if g.Port != "" && g.Port != "-1" {
				spicePort = g.Port
			}
			spiceHost = g.Listen
			break
		}
	}

	if spicePort == "" {
		log.Printf("SPICE proxy error: SPICE not configured or enabled for VM %s", vmName)
		return
	}

	// If listen address is local, empty, or unspecified, use the host's actual address from the DB.
	if spiceHost == "" || spiceHost == "127.0.0.1" || spiceHost == "0.0.0.0" || spiceHost == "::" {
		var host storage.Host
		if result := db.First(&host, "id = ?", hostID); result.Error != nil {
			log.Printf("SPICE proxy error: could not find host %s in DB to determine address: %v", hostID, result.Error)
			return
		}
		// A simple way to get hostname from a libvirt URI like qemu+ssh://user@hostname/system
		parts := strings.SplitN(host.URI, "@", 2)
		if len(parts) > 1 {
			hostPart := strings.Split(parts[1], "/")[0]
			// Handle potential port in hostname, e.g., user@hostname:port/system
			if strings.Contains(hostPart, ":") {
				spiceHost, _, _ = net.SplitHostPort(hostPart)
			} else {
				spiceHost = hostPart
			}
		} else {
			log.Printf("SPICE proxy error: could not determine VNC host address from URI %s", host.URI)
			return
		}
		log.Printf("SPICE listen address was local; resolved to hypervisor address: %s", spiceHost)
	}

	targetAddr := fmt.Sprintf("%s:%s", spiceHost, spicePort)
	log.Printf("Proxying console for %s to SPICE target %s", vmName, targetAddr)

	// Dial the actual SPICE service on the hypervisor.
	// Note: This simple proxy does not handle TLS between the proxy and the SPICE server.
	// For production, a TLS dialer would be needed if connecting to a TlsPort.
	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("SPICE proxy error: failed to connect to SPICE service at %s: %v", targetAddr, err)
		return
	}
	defer target.Close()

	// Start proxying data in both directions
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(target, wrappedWsConn)
	}()
	go func() {
		defer wg.Done()
		io.Copy(wrappedWsConn, target)
	}()

	wg.Wait()
	log.Printf("SPICE console proxy session ended for %s", vmName)
}


