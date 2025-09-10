package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/capsali/virtumancer/internal/libvirt"
	"github.com/capsali/virtumancer/internal/storage"
	"github.com/capsali/virtumancer/internal/ws"
	golibvirt "github.com/digitalocean/go-libvirt"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VMView is a combination of DB data and live libvirt data for the frontend.
type VMView struct {
	// From DB
	ID              uint   `json:"db_id"`
	Name            string `json:"name"`
	UUID            string `json:"uuid"`
	DomainUUID      string `json:"domain_uuid"`
	Description     string `json:"description"`
	VCPUCount       uint   `json:"vcpu_count"`
	MemoryBytes     uint64 `json:"memory_bytes"`
	IsTemplate      bool   `json:"is_template"`
	CPUModel        string `json:"cpu_model"`
	CPUTopologyJSON string `json:"cpu_topology_json"`

	// From Libvirt or DB cache
	State    storage.VMState       `json:"state"` // Use our custom string state
	Graphics libvirt.GraphicsInfo    `json:"graphics"`
	Hardware *libvirt.HardwareInfo `json:"hardware,omitempty"` // Pointer to allow for null

	// From Libvirt (live data, only in some calls)
	MaxMem  uint64 `json:"max_mem"`
	Memory  uint64 `json:"memory"`
	CpuTime uint64 `json:"cpu_time"`
	Uptime  int64  `json:"uptime"`
}

// VmSubscription holds the clients subscribed to a VM's stats and a channel to stop polling.
type VmSubscription struct {
	clients      map[*ws.Client]bool
	stop         chan struct{}
	lastKnownStats *libvirt.VMStats
	mu           sync.RWMutex
}

// MonitoringManager handles real-time VM stat subscriptions.
type MonitoringManager struct {
	mu            sync.Mutex
	subscriptions map[string]*VmSubscription // key is "hostId:vmName"
	service       *HostService               // back-reference
}

// NewMonitoringManager creates a new manager.
func NewMonitoringManager(service *HostService) *MonitoringManager {
	return &MonitoringManager{
		subscriptions: make(map[string]*VmSubscription),
		service:       service,
	}
}

type HostServiceProvider interface {
	ws.InboundMessageHandler
	GetAllHosts() ([]storage.Host, error)
	GetHostInfo(hostID string) (*libvirt.HostInfo, error)
	AddHost(host storage.Host) (*storage.Host, error)
	RemoveHost(hostID string) error
	ConnectToAllHosts()
	GetVMsForHostFromDB(hostID string) ([]VMView, error)
	GetVMStats(hostID, vmName string) (*libvirt.VMStats, error)
	GetVMHardwareAndTriggerSync(hostID, vmName string) (*libvirt.HardwareInfo, error)
	SyncVMsForHost(hostID string)
	StartVM(hostID, vmName string) error
	ShutdownVM(hostID, vmName string) error
	RebootVM(hostID, vmName string) error
	ForceOffVM(hostID, vmName string) error
	ForceResetVM(hostID, vmName string) error
}

type HostService struct {
	db        *gorm.DB
	connector *libvirt.Connector
	hub       *ws.Hub
	monitor   *MonitoringManager
}

func NewHostService(db *gorm.DB, connector *libvirt.Connector, hub *ws.Hub) *HostService {
	s := &HostService{
		db:        db,
		connector: connector,
		hub:       hub,
	}
	s.monitor = NewMonitoringManager(s)
	return s
}

func (s *HostService) broadcastHostsChanged() {
	s.hub.BroadcastMessage(ws.Message{Type: "hosts-changed"})
}

func (s *HostService) broadcastVMsChanged(hostID string) {
	s.hub.BroadcastMessage(ws.Message{
		Type:    "vms-changed",
		Payload: ws.MessagePayload{"hostId": hostID},
	})
}

// --- Host Management ---

func (s *HostService) GetAllHosts() ([]storage.Host, error) {
	var hosts []storage.Host
	if err := s.db.Find(&hosts).Error; err != nil {
		return nil, err
	}
	return hosts, nil
}

func (s *HostService) GetHostInfo(hostID string) (*libvirt.HostInfo, error) {
	return s.connector.GetHostInfo(hostID)
}

func (s *HostService) AddHost(host storage.Host) (*storage.Host, error) {
	if err := s.db.Create(&host).Error; err != nil {
		return nil, fmt.Errorf("failed to save host to database: %w", err)
	}

	err := s.connector.AddHost(host)
	if err != nil {
		if delErr := s.db.Delete(&host).Error; delErr != nil {
			log.Printf("CRITICAL: Failed to rollback host creation for %s after connection failure. DB Error: %v", host.ID, delErr)
		}
		return nil, fmt.Errorf("failed to connect to host: %w", err)
	}

	// Initial sync after adding a host
	go s.SyncVMsForHost(host.ID)

	s.broadcastHostsChanged()
	return &host, nil
}

func (s *HostService) RemoveHost(hostID string) error {
	if err := s.connector.RemoveHost(hostID); err != nil {
		log.Printf("Warning: failed to disconnect from host %s during removal, continuing with DB deletion: %v", hostID, err)
	}

	if err := s.db.Where("host_id = ?", hostID).Delete(&storage.VirtualMachine{}).Error; err != nil {
		log.Printf("Warning: failed to delete VMs for host %s from database: %v", hostID, err)
	}

	if err := s.db.Where("id = ?", hostID).Delete(&storage.Host{}).Error; err != nil {
		return fmt.Errorf("failed to delete host from database: %w", err)
	}

	s.broadcastHostsChanged()
	return nil
}

func (s *HostService) ConnectToAllHosts() {
	hosts, err := s.GetAllHosts()
	if err != nil {
		log.Printf("Error retrieving hosts from database on startup: %v", err)
		return
	}

	for _, host := range hosts {
		log.Printf("Attempting to connect to stored host: %s", host.ID)
		if err := s.connector.AddHost(host); err != nil {
			log.Printf("Failed to connect to host %s (%s) on startup: %v", host.ID, host.URI, err)
		} else {
			go s.SyncVMsForHost(host.ID)
		}
	}
}

// --- VM Management ---
func (s *HostService) GetVMsForHostFromDB(hostID string) ([]VMView, error) {
	var dbVMs []storage.VirtualMachine
	if err := s.db.Where("host_id = ?", hostID).Find(&dbVMs).Error; err != nil {
		return nil, fmt.Errorf("could not get DB VM records for host %s: %w", hostID, err)
	}

	var vmViews []VMView
	for _, dbVM := range dbVMs {
		var graphics libvirt.GraphicsInfo // Default to false

		// Only query for graphics devices if the VM is running.
		if dbVM.State == storage.StateActive {
			var graphicsDevice storage.GraphicsDevice
			err := s.db.Joins("join graphics_device_attachments on graphics_device_attachments.graphics_device_id = graphics_devices.id").
				Where("graphics_device_attachments.vm_id = ?", dbVM.ID).First(&graphicsDevice).Error

			if err != nil && err != gorm.ErrRecordNotFound {
				// Log only unexpected errors, not "not found".
				log.Printf("Error querying graphics device for running VM %d: %v", dbVM.ID, err)
			} else if err == nil {
				graphics.VNC = strings.ToLower(graphicsDevice.Type) == "vnc"
				graphics.SPICE = strings.ToLower(graphicsDevice.Type) == "spice"
			}
		}

		vmViews = append(vmViews, VMView{
			ID:              dbVM.ID,
			Name:            dbVM.Name,
			UUID:            dbVM.UUID,
			DomainUUID:      dbVM.DomainUUID,
			Description:     dbVM.Description,
			VCPUCount:       dbVM.VCPUCount,
			MemoryBytes:     dbVM.MemoryBytes,
			IsTemplate:      dbVM.IsTemplate,
			CPUModel:        dbVM.CPUModel,
			CPUTopologyJSON: dbVM.CPUTopologyJSON,
			State:           dbVM.State,
			Graphics:        graphics,
		})
	}
	return vmViews, nil
}

func (s *HostService) getVMHardwareFromDB(hostID, vmName string) (*libvirt.HardwareInfo, error) {
	var vm storage.VirtualMachine
	if err := s.db.Where("host_id = ? AND name = ?", hostID, vmName).First(&vm).Error; err != nil {
		return nil, fmt.Errorf("could not find VM %s in database: %w", vmName, err)
	}

	var hardware libvirt.HardwareInfo

	// Retrieve and populate disks
	var diskAttachments []storage.VolumeAttachment
	s.db.Preload("Volume").Where("vm_id = ?", vm.ID).Find(&diskAttachments)
	for _, da := range diskAttachments {
		hardware.Disks = append(hardware.Disks, libvirt.DiskInfo{
			Device: da.DeviceName,
			Path:   da.Volume.Name,
			Target: struct {
				Dev string `xml:"dev,attr" json:"dev"`
				Bus string `xml:"bus,attr" json:"bus"`
			}{
				Dev: da.DeviceName,
				Bus: da.BusType,
			},
			Driver: struct {
				Name string `xml:"name,attr" json:"driver_name"`
				Type string `xml:"type,attr" json:"type"`
			}{
				Type: da.Volume.Format,
			},
		})
	}

	// Retrieve and populate networks
	var ports []storage.Port
	err := s.db.Where("vm_id = ?", vm.ID).Find(&ports).Error
	if err == nil {
		for _, port := range ports {
			var binding storage.PortBinding
			if err := s.db.Preload("Network").Where("port_id = ?", port.ID).First(&binding).Error; err == nil {
				hardware.Networks = append(hardware.Networks, libvirt.NetworkInfo{
					Mac: struct {
						Address string `xml:"address,attr" json:"address"`
					}{
						Address: port.MACAddress,
					},
					Source: struct {
						Bridge string `xml:"bridge,attr" json:"bridge"`
					}{
						Bridge: binding.Network.BridgeName,
					},
					Model: struct {
						Type string `xml:"type,attr" json:"model_type"`
					}{
						Type: port.ModelName,
					},
					Target: struct {
						Dev string `xml:"dev,attr" json:"dev"`
					}{
						Dev: port.DeviceName,
					},
				})
			}
		}
	}

	return &hardware, nil
}
func (s *HostService) GetVMHardwareAndTriggerSync(hostID, vmName string) (*libvirt.HardwareInfo, error) {
	// We will now always sync and then get from DB for consistency,
	// since the data is structured and no longer a simple JSON blob.
	if changed, syncErr := s.syncSingleVM(hostID, vmName); syncErr != nil {
		log.Printf("Error during hardware sync for %s: %v", vmName, syncErr)
		// We can still try to return what's in the DB
	} else if changed {
		s.broadcastVMsChanged(hostID)
	}

	return s.getVMHardwareFromDB(hostID, vmName)
}

func (s *HostService) SyncVMsForHost(hostID string) {
	changed, err := s.syncAndListVMs(hostID)
	if err != nil {
		log.Printf("Error during background VM sync for host %s: %v", hostID, err)
		return
	}
	if changed {
		s.broadcastVMsChanged(hostID)
	}
}

func (s *HostService) syncSingleVM(hostID, vmName string) (bool, error) {
	vmInfo, err := s.connector.GetDomainInfo(hostID, vmName)
	if err != nil {
		var dbVM storage.VirtualMachine
		if err := s.db.Where("host_id = ? AND name = ?", hostID, vmName).First(&dbVM).Error; err == nil {
			log.Printf("Pruning VM %s from database as it's no longer in libvirt.", vmName)
			if err := s.db.Delete(&dbVM).Error; err != nil {
				log.Printf("Warning: failed to prune old VM %s: %v", dbVM.Name, err)
				return false, err
			}
			return true, nil
		}
		return false, fmt.Errorf("could not fetch info for VM %s on host %s: %w", vmName, hostID, err)
	}

	hardwareInfo, err := s.connector.GetDomainHardware(hostID, vmName)
	if err != nil {
		log.Printf("Warning: could not fetch hardware for VM %s: %v", vmInfo.Name, err)
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var existingVMOnHost storage.VirtualMachine
	var changed bool
	err = tx.Where("host_id = ? AND domain_uuid = ?", hostID, vmInfo.UUID).First(&existingVMOnHost).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return false, err // Database error
	}

	// Case 1: The VM is not in our DB for this host. It's either brand new or has a conflict.
	if err == gorm.ErrRecordNotFound {
		var conflictingVM storage.VirtualMachine
		err := tx.Where("domain_uuid = ? AND host_id != ?", vmInfo.UUID, hostID).First(&conflictingVM).Error

		newVMRecord := storage.VirtualMachine{
			HostID:      hostID,
			Name:        vmInfo.Name,
			DomainUUID:  vmInfo.UUID,
			State:       mapLibvirtStateToVMState(vmInfo.State),
			VCPUCount:   vmInfo.Vcpu,
			MemoryBytes: vmInfo.MaxMem * 1024,
		}

		if err == gorm.ErrRecordNotFound {
			// No conflict found. This is a genuinely new VM to our entire system.
			// Set our internal UUID to be the same as the domain's UUID.
			newVMRecord.UUID = vmInfo.UUID
		} else if err != nil {
			// Some other DB error occurred
			tx.Rollback()
			return false, err
		} else {
			// Conflict found! A VM with this domain UUID exists on another host.
			// Generate a new, unique internal UUID for our system.
			log.Printf("UUID conflict detected for DomainUUID %s. Assigning new internal UUID.", vmInfo.UUID)
			newVMRecord.UUID = uuid.New().String()
		}

		if err := tx.Create(&newVMRecord).Error; err != nil {
			tx.Rollback()
			return false, err
		}
		changed = true
		existingVMOnHost = newVMRecord // Use the newly created record for hardware sync
	} else { // Case 2: The VM already exists in our DB for this host. Just update its state.
		updates := map[string]interface{}{
			"Name":        vmInfo.Name,
			"State":       mapLibvirtStateToVMState(vmInfo.State),
			"VCPUCount":   vmInfo.Vcpu,
			"MemoryBytes": vmInfo.MaxMem * 1024,
		}
		if existingVMOnHost.Name != vmInfo.Name || existingVMOnHost.State != mapLibvirtStateToVMState(vmInfo.State) ||
			existingVMOnHost.VCPUCount != vmInfo.Vcpu || existingVMOnHost.MemoryBytes != (vmInfo.MaxMem*1024) {
			if err := tx.Model(&existingVMOnHost).Updates(updates).Error; err != nil {
				tx.Rollback()
				return false, err
			}
			changed = true
		}
	}

	if hardwareInfo != nil {
		if err := s.syncVMHardware(tx, existingVMOnHost.ID, hostID, hardwareInfo, &vmInfo.Graphics); err != nil {
			tx.Rollback()
			return false, fmt.Errorf("failed to sync hardware: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return false, err
	}

	return changed, nil
}

// syncVMHardware reconciles the live hardware state with the database.
func (s *HostService) syncVMHardware(tx *gorm.DB, vmID uint, hostID string, hardware *libvirt.HardwareInfo, graphics *libvirt.GraphicsInfo) error {
	// Correctly clear existing PortBindings by finding associated ports first
	var portsToDelete []storage.Port
	tx.Where("vm_id = ?", vmID).Find(&portsToDelete)
	if len(portsToDelete) > 0 {
		var portIDs []uint
		for _, p := range portsToDelete {
			portIDs = append(portIDs, p.ID)
		}
		tx.Where("port_id IN ?", portIDs).Delete(&storage.PortBinding{})
	}

	tx.Where("vm_id = ?", vmID).Delete(&storage.VolumeAttachment{})
	tx.Where("vm_id = ?", vmID).Delete(&storage.GraphicsDeviceAttachment{})

	// Sync Disks
	for _, disk := range hardware.Disks {
		var volume storage.Volume
		tx.FirstOrCreate(&volume, storage.Volume{Name: disk.Path}, storage.Volume{
			Name:   disk.Path,
			Format: disk.Driver.Type,
			Type:   "DISK", // Assumption for now
		})

		if volume.ID != 0 {
			attachment := storage.VolumeAttachment{
				VMID:       vmID,
				VolumeID:   volume.ID,
				DeviceName: disk.Target.Dev,
				BusType:    disk.Target.Bus,
			}
			tx.Create(&attachment)
		}
	}

	// Sync Networks
	for _, net := range hardware.Networks {
		var network storage.Network
		networkUUID := uuid.NewSHA1(uuid.Nil, []byte(fmt.Sprintf("%s:%s", hostID, net.Source.Bridge)))

		tx.FirstOrCreate(&network, storage.Network{UUID: networkUUID.String()}, storage.Network{
			HostID:     hostID,
			Name:       net.Source.Bridge,
			BridgeName: net.Source.Bridge,
			Mode:       "bridged",
			UUID:       networkUUID.String(),
		})

		var port storage.Port
		// Use Assign to update fields on existing records or create a new one.
		tx.Where(storage.Port{MACAddress: net.Mac.Address}).
			Assign(storage.Port{
				VMID:       vmID,
				MACAddress: net.Mac.Address,
				DeviceName: net.Target.Dev,
				ModelName:  net.Model.Type,
			}).
			FirstOrCreate(&port)

		if network.ID != 0 && port.ID != 0 {
			binding := storage.PortBinding{
				PortID:    port.ID,
				NetworkID: network.ID,
			}
			tx.FirstOrCreate(&binding, storage.PortBinding{PortID: port.ID, NetworkID: network.ID})
		}
	}

	// Sync Graphics
	var gfxDevice storage.GraphicsDevice
	if graphics.VNC {
		tx.FirstOrCreate(&gfxDevice, storage.GraphicsDevice{Type: "vnc"}, storage.GraphicsDevice{Type: "vnc", ModelName: "vnc"})
	} else if graphics.SPICE {
		tx.FirstOrCreate(&gfxDevice, storage.GraphicsDevice{Type: "spice"}, storage.GraphicsDevice{Type: "spice", ModelName: "qxl"})
	}

	if gfxDevice.ID != 0 {
		attachment := storage.GraphicsDeviceAttachment{
			VMID:             vmID,
			GraphicsDeviceID: gfxDevice.ID,
		}
		tx.Create(&attachment)
	}

	return nil
}

// mapLibvirtStateToVMState translates libvirt's integer state to our string state.
func mapLibvirtStateToVMState(state golibvirt.DomainState) storage.VMState {
	switch state {
	case golibvirt.DomainRunning:
		return storage.StateActive
	case golibvirt.DomainPaused:
		return storage.StatePaused
	case golibvirt.DomainShutdown, golibvirt.DomainShutoff, golibvirt.DomainCrashed:
		return storage.StateStopped
	case golibvirt.DomainPmsuspended:
		return storage.StateSuspended
	default:
		return storage.StateStopped // Default to stopped for unknown/other states
	}
}

// syncAndListVMs is the core function to get VMs from libvirt and sync with the local DB.
// It returns true if any data was changed in the database.
func (s *HostService) syncAndListVMs(hostID string) (bool, error) {
	liveVMs, err := s.connector.ListAllDomains(hostID)
	if err != nil {
		return false, fmt.Errorf("service failed to list vms for host %s: %w", hostID, err)
	}

	var overallChanged bool

	liveVMUUIDs := make(map[string]struct{})
	for _, vmInfo := range liveVMs {
		liveVMUUIDs[vmInfo.UUID] = struct{}{}
		changed, err := s.syncSingleVM(hostID, vmInfo.Name)
		if err != nil {
			log.Printf("Error syncing VM %s: %v", vmInfo.Name, err)
		}
		if changed {
			overallChanged = true
		}
	}

	var dbVMs []storage.VirtualMachine
	if err := s.db.Where("host_id = ?", hostID).Find(&dbVMs).Error; err != nil {
		return false, fmt.Errorf("could not get DB records for pruning check: %w", err)
	}

	for _, dbVM := range dbVMs {
		if _, exists := liveVMUUIDs[dbVM.DomainUUID]; !exists {
			log.Printf("Pruning VM %s (UUID: %s) from database as it's no longer in libvirt.", dbVM.Name, dbVM.UUID)
			if err := s.db.Delete(&dbVM).Error; err != nil {
				log.Printf("Warning: failed to prune old VM %s: %v", dbVM.Name, err)
			} else {
				overallChanged = true
			}
		}
	}

	return overallChanged, nil
}

func (s *HostService) GetVMStats(hostID, vmName string) (*libvirt.VMStats, error) {
	// First, check if there's an active subscription.
	stats := s.monitor.GetLastKnownStats(hostID, vmName)
	if stats != nil {
		return stats, nil
	}

	// If no active subscription, perform a one-time fetch.
	return s.connector.GetDomainStats(hostID, vmName)
}

// --- VM Actions ---

func (s *HostService) StartVM(hostID, vmName string) error {
	if err := s.connector.StartDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastVMsChanged(hostID)
	}
	return nil
}

func (s *HostService) ShutdownVM(hostID, vmName string) error {
	if err := s.connector.ShutdownDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastVMsChanged(hostID)
	}
	return nil
}

func (s *HostService) RebootVM(hostID, vmName string) error {
	if err := s.connector.RebootDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastVMsChanged(hostID)
	}
	return nil
}

func (s *HostService) ForceOffVM(hostID, vmName string) error {
	if err := s.connector.DestroyDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastVMsChanged(hostID)
	}
	return nil
}

func (s *HostService) ForceResetVM(hostID, vmName string) error {
	if err := s.connector.ResetDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastVMsChanged(hostID)
	}
	return nil
}

// --- WebSocket Message Handling ---

func (s *HostService) HandleSubscribe(client *ws.Client, payload ws.MessagePayload) {
	hostID, ok1 := payload["hostId"].(string)
	vmName, ok2 := payload["vmName"].(string)
	if !ok1 || !ok2 {
		log.Println("Invalid payload for vm-stats subscription")
		return
	}
	s.monitor.Subscribe(client, hostID, vmName)
}

func (s *HostService) HandleUnsubscribe(client *ws.Client, payload ws.MessagePayload) {
	hostID, ok1 := payload["hostId"].(string)
	vmName, ok2 := payload["vmName"].(string)
	if !ok1 || !ok2 {
		log.Println("Invalid payload for vm-stats unsubscription")
		return
	}
	s.monitor.Unsubscribe(client, hostID, vmName)
}

func (s *HostService) HandleClientDisconnect(client *ws.Client) {
	s.monitor.UnsubscribeClient(client)
}

// --- Monitoring Goroutine Logic ---

func (m *MonitoringManager) Subscribe(client *ws.Client, hostID, vmName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", hostID, vmName)
	sub, exists := m.subscriptions[key]
	if !exists {
		log.Printf("Starting monitoring for %s", key)
		sub = &VmSubscription{
			clients: make(map[*ws.Client]bool),
			stop:    make(chan struct{}),
		}
		m.subscriptions[key] = sub
		go m.pollVmStats(hostID, vmName, sub)
	}
	sub.clients[client] = true
}

func (m *MonitoringManager) Unsubscribe(client *ws.Client, hostID, vmName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", hostID, vmName)
	if sub, exists := m.subscriptions[key]; exists {
		delete(sub.clients, client)
		if len(sub.clients) == 0 {
			log.Printf("Stopping monitoring for %s", key)
			close(sub.stop)
			delete(m.subscriptions, key)
		}
	}
}

func (m *MonitoringManager) UnsubscribeClient(client *ws.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, sub := range m.subscriptions {
		if _, ok := sub.clients[client]; ok {
			delete(sub.clients, client)
			if len(sub.clients) == 0 {
				log.Printf("Stopping monitoring for %s due to client disconnect", key)
				close(sub.stop)
				delete(m.subscriptions, key)
			}
		}
	}
}

func (m *MonitoringManager) GetLastKnownStats(hostID, vmName string) *libvirt.VMStats {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", hostID, vmName)
	if sub, exists := m.subscriptions[key]; exists {
		sub.mu.RLock()
		defer sub.mu.RUnlock()
		return sub.lastKnownStats
	}
	return nil
}

func (m *MonitoringManager) pollVmStats(hostID, vmName string, sub *VmSubscription) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats, err := m.service.connector.GetDomainStats(hostID, vmName)
			if err != nil {
				stats = &libvirt.VMStats{State: golibvirt.DomainShutoff}
			}

			// Update last known stats
			sub.mu.Lock()
			sub.lastKnownStats = stats
			sub.mu.Unlock()

			// Broadcast the stats update.
			m.service.hub.BroadcastMessage(ws.Message{
				Type: "vm-stats-updated",
				Payload: ws.MessagePayload{
					"hostId": hostID,
					"vmName": vmName,
					"stats":  stats,
				},
			})

			// If the VM is no longer running, stop polling it.
			if stats.State != golibvirt.DomainRunning {
				log.Printf("VM %s is not running, stopping stats polling.", vmName)
				// Unsubscribe all clients for this VM
				m.mu.Lock()
				delete(m.subscriptions, fmt.Sprintf("%s:%s", hostID, vmName))
				m.mu.Unlock()
				return
			}
		case <-sub.stop:
			return
		}
	}
}



