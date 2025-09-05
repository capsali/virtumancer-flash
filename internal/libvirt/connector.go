package libvirt

import (
	"fmt"
	"log"
	"net/url"
	"sync"

	"github.com/capsali/virtumancer/internal/storage"
	"libvirt.org/go/libvirt"
)

// VMInfo holds basic information about a virtual machine.
type VMInfo struct {
	ID         uint32              `json:"id"`
	Name       string              `json:"name"`
	State      libvirt.DomainState `json:"state"`
	MaxMem     uint64              `json:"max_mem"`
	Memory     uint64              `json:"memory"`
	Vcpu       uint                `json:"vcpu"`
	Persistent bool                `json:"persistent"`
	Autostart  bool                `json:"autostart"`
}

// Connector manages active connections to libvirt hosts.
type Connector struct {
	connections map[string]*libvirt.Connect
	mu          sync.RWMutex
}

// NewConnector creates a new libvirt connection manager.
func NewConnector() *Connector {
	return &Connector{
		connections: make(map[string]*libvirt.Connect),
	}
}

// AddHost connects to a given libvirt URI and adds it to the connection pool.
func (c *Connector) AddHost(host storage.Host) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.connections[host.ID]; ok {
		return fmt.Errorf("host '%s' is already connected", host.ID)
	}

	connectURI := host.URI
	parsedURI, err := url.Parse(host.URI)
	if err == nil && parsedURI.Scheme == "qemu+ssh" {
		q := parsedURI.Query()
		if q.Get("no_verify") == "" {
			q.Set("no_verify", "1")
			parsedURI.RawQuery = q.Encode()
			connectURI = parsedURI.String()
			log.Printf("Amended URI for %s to %s for non-interactive connection", host.ID, connectURI)
		}
	}

	conn, err := libvirt.NewConnect(connectURI)
	if err != nil {
		return fmt.Errorf("failed to connect to host '%s' using URI %s: %w", host.ID, connectURI, err)
	}

	c.connections[host.ID] = conn
	log.Printf("Successfully connected to host: %s", host.ID)
	return nil
}

// RemoveHost disconnects from a libvirt host and removes it from the pool.
func (c *Connector) RemoveHost(hostID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, ok := c.connections[hostID]
	if !ok {
		return fmt.Errorf("host '%s' not found", hostID)
	}

	if _, err := conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection to host '%s': %w", hostID, err)
	}

	delete(c.connections, hostID)
	log.Printf("Disconnected from host: %s", hostID)
	return nil
}

// GetConnection returns the active connection for a given host ID.
func (c *Connector) GetConnection(hostID string) (*libvirt.Connect, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conn, ok := c.connections[hostID]
	if !ok {
		return nil, fmt.Errorf("not connected to host '%s'", hostID)
	}
	return conn, nil
}

// ListAllDomains lists all domains (VMs) on a specific host.
func (c *Connector) ListAllDomains(hostID string) ([]VMInfo, error) {
	conn, err := c.GetConnection(hostID)
	if err != nil {
		return nil, err
	}

	domains, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE | libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}

	var vms []VMInfo
	for i := range domains {
		defer domains[i].Free()

		name, err := domains[i].GetName()
		if err != nil {
			log.Printf("Warning: could not get name for a domain on host %s: %v", hostID, err)
			continue
		}
		id, err := domains[i].GetID()
		if err != nil {
			id = 0 // Not running
		}
		state, _, err := domains[i].GetState()
		if err != nil {
			log.Printf("Warning: could not get state for domain %s: %v", name, err)
			continue
		}
		info, err := domains[i].GetInfo()
		if err != nil {
			log.Printf("Warning: could not get info for domain %s: %v", name, err)
			continue
		}
		isPersistent, err := domains[i].IsPersistent()
		if err != nil {
			log.Printf("Warning: could not get persistence for domain %s: %v", name, err)
			continue
		}
		autostart, err := domains[i].GetAutostart()
		if err != nil {
			log.Printf("Warning: could not get autostart for domain %s: %v", name, err)
			continue
		}

		vms = append(vms, VMInfo{
			ID:         uint32(id),
			Name:       name,
			State:      state,
			MaxMem:     info.MaxMem,
			Memory:     info.Memory,
			Vcpu:       uint(info.NrVirtCpu),
			Persistent: isPersistent,
			Autostart:  autostart,
		})
	}

	return vms, nil
}

// --- VM Actions ---

// getDomainByName is a helper function to look up a domain by its name on a host.
func (c *Connector) getDomainByName(hostID, vmName string) (*libvirt.Domain, error) {
	conn, err := c.GetConnection(hostID)
	if err != nil {
		return nil, err
	}
	domain, err := conn.LookupDomainByName(vmName)
	if err != nil {
		return nil, fmt.Errorf("could not find VM '%s' on host '%s': %w", vmName, hostID, err)
	}
	return domain, nil
}

// StartDomain starts a managed domain (VM).
func (c *Connector) StartDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Create()
}

// ShutdownDomain gracefully shuts down a managed domain.
func (c *Connector) ShutdownDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Shutdown()
}

// RebootDomain gracefully reboots a managed domain.
func (c *Connector) RebootDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Reboot(0) // 0 is the default flag
}

// DestroyDomain forcefully stops a managed domain (the equivalent of pulling the plug).
func (c *Connector) DestroyDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Destroy()
}

// ResetDomain forcefully resets a managed domain.
func (c *Connector) ResetDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Reset(0) // 0 is the default flag
}


