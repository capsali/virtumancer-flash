package libvirt

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/capsali/virtumancer-flash/internal/storage"
	"libvirt.org/go/libvirt"
)

// GraphicsInfo holds details about available graphics consoles.
type GraphicsInfo struct {
	VNC   bool `json:"vnc"`
	SPICE bool `json:"spice"`
}

// VMInfo holds basic information about a virtual machine.
type VMInfo struct {
	ID         uint32              `json:"id"`
	UUID       string              `json:"uuid"`
	Name       string              `json:"name"`
	State      libvirt.DomainState `json:"state"`
	MaxMem     uint64              `json:"max_mem"`
	Memory     uint64              `json:"memory"`
	Vcpu       uint                `json:"vcpu"`
	CpuTime    uint64              `json:"cpu_time"`
	Uptime     int64               `json:"uptime"`
	Persistent bool                `json:"persistent"`
	Autostart  bool                `json:"autostart"`
	Graphics   GraphicsInfo        `json:"graphics"`
}

// DomainDiskStats holds I/O statistics for a single disk device.
type DomainDiskStats struct {
	Device     string `json:"device"`
	ReadBytes  int64  `json:"read_bytes"`
	WriteBytes int64  `json:"write_bytes"`
}

// DomainNetworkStats holds I/O statistics for a single network interface.
type DomainNetworkStats struct {
	Device     string `json:"device"`
	ReadBytes  int64  `json:"read_bytes"`
	WriteBytes int64  `json:"write_bytes"`
}

// VMStats holds real-time statistics for a single VM.
type VMStats struct {
	State      libvirt.DomainState  `json:"state"`
	Memory     uint64               `json:"memory"`
	MaxMem     uint64               `json:"max_mem"`
	Vcpu       uint                 `json:"vcpu"`
	CpuTime    uint64               `json:"cpu_time"`
	DiskStats  []DomainDiskStats    `json:"disk_stats"`
	NetStats   []DomainNetworkStats `json:"net_stats"`
}

// HardwareInfo holds the hardware configuration of a VM.
type HardwareInfo struct {
	Disks    []DiskInfo    `json:"disks"`
	Networks []NetworkInfo `json:"networks"`
}

// DiskInfo represents a virtual disk.
type DiskInfo struct {
	Type   string `xml:"type,attr" json:"type"`
	Device string `xml:"device,attr" json:"device"`
	Driver struct {
		Name string `xml:"name,attr" json:"driver_name"`
		Type string `xml:"type,attr" json:"type"`
	} `xml:"driver" json:"driver"`
	Source struct {
		File string `xml:"file,attr"`
		Dev  string `xml:"dev,attr"`
	} `xml:"source"`
	Path   string `json:"path"`
	Target struct {
		Dev string `xml:"dev,attr" json:"dev"`
		Bus string `xml:"bus,attr" json:"bus"`
	} `xml:"target" json:"target"`
}

// NetworkInfo represents a virtual network interface.
type NetworkInfo struct {
	Type   string `xml:"type,attr" json:"type"`
	Mac    struct {
		Address string `xml:"address,attr" json:"address"`
	} `xml:"mac" json:"mac"`
	Source struct {
		Bridge string `xml:"bridge,attr" json:"bridge"`
	} `xml:"source" json:"source"`
	Model struct {
		Type string `xml:"type,attr" json:"model_type"`
	} `xml:"model" json:"model"`
	Target struct {
		Dev string `xml:"dev,attr" json:"dev"`
	} `xml:"target" json:"target"`
}

// DomainHardwareXML is used for unmarshalling hardware info from the domain XML.
type DomainHardwareXML struct {
	Devices struct {
		Disks      []DiskInfo    `xml:"disk"`
		Interfaces []NetworkInfo `xml:"interface"`
	} `xml:"devices"`
}

// HostInfo holds basic information and statistics about a hypervisor host.
type HostInfo struct {
	Hostname string `json:"hostname"`
	CPU      uint   `json:"cpu"`
	Memory   uint64 `json:"memory"`
	Cores    uint   `json:"cores"`
	Threads  uint   `json:"threads"`
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

// GetHostInfo retrieves statistics about the host itself.
func (c *Connector) GetHostInfo(hostID string) (*HostInfo, error) {
	conn, err := c.GetConnection(hostID)
	if err != nil {
		return nil, err
	}

	nodeInfo, err := conn.GetNodeInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get node info for host %s: %w", hostID, err)
	}

	hostname, err := conn.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname for host %s: %w", hostID, err)
	}

	return &HostInfo{
		Hostname: hostname,
		CPU:      nodeInfo.Cpus,
		Memory:   nodeInfo.Memory,
		Cores:    uint(nodeInfo.Cores),
		Threads:  uint(nodeInfo.Threads),
	}, nil
}

// parseGraphicsFromXML extracts VNC and SPICE availability from a domain's XML definition.
func parseGraphicsFromXML(xmlDesc string) (GraphicsInfo, error) {
	type GraphicsXML struct {
		Type string `xml:"type,attr"`
		Port string `xml:"port,attr"`
	}
	type DomainDef struct {
		Graphics []GraphicsXML `xml:"devices>graphics"`
	}

	var def DomainDef
	var graphics GraphicsInfo

	if err := xml.Unmarshal([]byte(xmlDesc), &def); err != nil {
		return graphics, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	for _, g := range def.Graphics {
		if g.Port != "" && g.Port != "-1" {
			switch strings.ToLower(g.Type) {
			case "vnc":
				graphics.VNC = true
			case "spice":
				graphics.SPICE = true
			}
		}
	}

	return graphics, nil
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
		domain := &domains[i]

		vmInfo, err := c.domainToVMInfo(domain)
		if err != nil {
			name, _ := domain.GetName()
			log.Printf("Warning: could not get info for domain %s on host %s: %v", name, hostID, err)
			domain.Free()
			continue
		}

		vms = append(vms, *vmInfo)
		domain.Free()
	}

	return vms, nil
}

// GetDomainInfo retrieves information for a single domain.
func (c *Connector) GetDomainInfo(hostID, vmName string) (*VMInfo, error) {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return nil, err
	}
	defer domain.Free()

	return c.domainToVMInfo(domain)
}

// domainToVMInfo is a helper to convert a libvirt.Domain object to our VMInfo struct.
func (c *Connector) domainToVMInfo(domain *libvirt.Domain) (*VMInfo, error) {
	name, err := domain.GetName()
	if err != nil {
		return nil, err
	}
	uuid, err := domain.GetUUIDString()
	if err != nil {
		return nil, err
	}
	id, err := domain.GetID()
	if err != nil {
		id = 0 // Not running
	}
	state, _, err := domain.GetState()
	if err != nil {
		return nil, err
	}
	info, err := domain.GetInfo()
	if err != nil {
		return nil, err
	}
	var uptime int64 = -1
	if state == libvirt.DOMAIN_RUNNING {
		timeVal, _, err := domain.GetTime(0)
		if err == nil {
			uptime = timeVal
		}
	}
	isPersistent, err := domain.IsPersistent()
	if err != nil {
		isPersistent = false
	}
	autostart, err := domain.GetAutostart()
	if err != nil {
		autostart = false
	}
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}
	graphics, err := parseGraphicsFromXML(xmlDesc)
	if err != nil {
		return nil, err
	}

	return &VMInfo{
		ID:         uint32(id),
		UUID:       uuid,
		Name:       name,
		State:      state,
		MaxMem:     info.MaxMem,
		Memory:     info.Memory,
		Vcpu:       uint(info.NrVirtCpu),
		CpuTime:    info.CpuTime,
		Uptime:     uptime,
		Persistent: isPersistent,
		Autostart:  autostart,
		Graphics:   graphics,
	}, nil
}

// GetDomainStats retrieves real-time statistics for a single domain (VM).
func (c *Connector) GetDomainStats(hostID, vmName string) (*VMStats, error) {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return nil, err
	}
	defer domain.Free()

	state, _, err := domain.GetState()
	if err != nil {
		return nil, fmt.Errorf("could not get state for domain %s: %w", vmName, err)
	}

	info, err := domain.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("could not get info for domain %s: %w", vmName, err)
	}

	// If not running, return basic info without I/O stats
	if state != libvirt.DOMAIN_RUNNING {
		return &VMStats{
			State:     state,
			Memory:    0,
			MaxMem:    info.MaxMem,
			Vcpu:      uint(info.NrVirtCpu),
			CpuTime:   0,
			DiskStats: []DomainDiskStats{},
			NetStats:  []DomainNetworkStats{},
		}, nil
	}

	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get XML for %s to find devices: %w", vmName, err)
	}

	var def DomainHardwareXML
	if err := xml.Unmarshal([]byte(xmlDesc), &def); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML for devices: %w", err)
	}

	var diskStats []DomainDiskStats
	for _, disk := range def.Devices.Disks {
		if disk.Target.Dev == "" {
			continue
		}
		stats, err := domain.BlockStats(disk.Target.Dev)
		if err != nil {
			log.Printf("Warning: could not get block stats for device %s on VM %s: %v", disk.Target.Dev, vmName, err)
			continue
		}
		diskStats = append(diskStats, DomainDiskStats{
			Device:     disk.Target.Dev,
			ReadBytes:  stats.RdBytes,
			WriteBytes: stats.WrBytes,
		})
	}

	var netStats []DomainNetworkStats
	for _, iface := range def.Devices.Interfaces {
		if iface.Target.Dev == "" {
			continue
		}
		stats, err := domain.InterfaceStats(iface.Target.Dev)
		if err != nil {
			log.Printf("Warning: could not get interface stats for device %s on VM %s: %v", iface.Target.Dev, vmName, err)
			continue
		}
		netStats = append(netStats, DomainNetworkStats{
			Device:     iface.Target.Dev,
			ReadBytes:  stats.RxBytes,
			WriteBytes: stats.TxBytes,
		})
	}

	stats := &VMStats{
		State:      state,
		Memory:     info.Memory,
		MaxMem:     info.MaxMem,
		Vcpu:       uint(info.NrVirtCpu),
		CpuTime:    info.CpuTime,
		DiskStats:  diskStats,
		NetStats:   netStats,
	}

	return stats, nil
}

// GetDomainHardware retrieves the hardware configuration for a single domain (VM).
func (c *Connector) GetDomainHardware(hostID, vmName string) (*HardwareInfo, error) {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return nil, err
	}
	defer domain.Free()

	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get XML for %s to read hardware: %w", vmName, err)
	}

	var def DomainHardwareXML
	if err := xml.Unmarshal([]byte(xmlDesc), &def); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML for hardware: %w", err)
	}

	hardware := &HardwareInfo{
		Disks:    def.Devices.Disks,
		Networks: def.Devices.Interfaces,
	}

	// Post-process disks to populate the unified 'Path' field.
	for i := range hardware.Disks {
		if hardware.Disks[i].Source.File != "" {
			hardware.Disks[i].Path = hardware.Disks[i].Source.File
		} else if hardware.Disks[i].Source.Dev != "" {
			hardware.Disks[i].Path = hardware.Disks[i].Source.Dev
		}
	}

	return hardware, nil
}

// --- VM Actions ---

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

func (c *Connector) StartDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Create()
}

func (c *Connector) ShutdownDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Shutdown()
}

func (c *Connector) RebootDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Reboot(0)
}

func (c *Connector) DestroyDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Destroy()
}

func (c *Connector) ResetDomain(hostID, vmName string) error {
	domain, err := c.getDomainByName(hostID, vmName)
	if err != nil {
		return err
	}
	defer domain.Free()
	return domain.Reset(0)
}


