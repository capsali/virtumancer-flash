package services

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/capsali/virtumancer-flash/internal/libvirt"
	"github.com/capsali/virtumancer-flash/internal/storage"
	"github.com/capsali/virtumancer-flash/internal/ws"
	"gorm.io/gorm"
	lv "libvirt.org/go/libvirt"
)

// VMView is a combination of DB data and live libvirt data for the frontend.
type VMView struct {
	// From DB
	ID              uint   `json:"db_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	VCPUCount       uint   `json:"vcpu_count"`
	MemoryBytes     uint64 `json:"memory_bytes"`
	IsTemplate      bool   `json:"is_template"`
	CPUModel        string `json:"cpu_model"`
	CPUTopologyJSON string `json:"cpu_topology_json"`

	// From Libvirt or DB cache
	State    lv.DomainState        `json:"state"`
	Graphics libvirt.GraphicsInfo  `json:"graphics"`
	Hardware *libvirt.HardwareInfo `json:"hardware,omitempty"` // Pointer to allow for null

	// From Libvirt (live data, only in some calls)
	MaxMem  uint64 `json:"max_mem"`
	Memory  uint64 `json:"memory"`
	CpuTime uint64 `json:"cpu_time"`
	Uptime  int64  `json:"uptime"`
}

type HostService struct {
	db        *gorm.DB
	connector *libvirt.Connector
	hub       *ws.Hub
}

func NewHostService(db *gorm.DB, connector *libvirt.Connector, hub *ws.Hub) *HostService {
	return &HostService{
		db:        db,
		connector: connector,
		hub:       hub,
	}
}

func (s *HostService) broadcastUpdate() {
	s.hub.BroadcastMessage([]byte(`{"type": "refresh"}`))
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

	s.broadcastUpdate()
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

	s.broadcastUpdate()
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

// GetVMsForHostFromDB retrieves the merged VM list for a host purely from the database,
// for a fast initial UI load.
func (s *HostService) GetVMsForHostFromDB(hostID string) ([]VMView, error) {
	var dbVMs []storage.VirtualMachine
	if err := s.db.Where("host_id = ?", hostID).Find(&dbVMs).Error; err != nil {
		return nil, fmt.Errorf("could not get DB VM records for host %s: %w", hostID, err)
	}

	var vmViews []VMView
	for _, dbVM := range dbVMs {
		var graphics libvirt.GraphicsInfo
		if err := json.Unmarshal([]byte(dbVM.GraphicsJSON), &graphics); err != nil {
			log.Printf("Warning: could not parse cached graphics info for VM %s: %v", dbVM.Name, err)
		}

		vmViews = append(vmViews, VMView{
			ID:              dbVM.ID,
			Name:            dbVM.Name,
			Description:     dbVM.Description,
			VCPUCount:       dbVM.VCPUCount,
			MemoryBytes:     dbVM.MemoryBytes,
			IsTemplate:      dbVM.IsTemplate,
			CPUModel:        dbVM.CPUModel,
			CPUTopologyJSON: dbVM.CPUTopologyJSON,
			State:           lv.DomainState(dbVM.State),
			Graphics:        graphics,
		})
	}
	return vmViews, nil
}

// getVMHardwareFromDB retrieves the hardware info from the JSON cache in the database.
func (s *HostService) getVMHardwareFromDB(hostID, vmName string) (*libvirt.HardwareInfo, error) {
	var dbVM storage.VirtualMachine
	if err := s.db.Where("host_id = ? AND name = ?", hostID, vmName).First(&dbVM).Error; err != nil {
		return nil, fmt.Errorf("could not find VM %s in database: %w", vmName, err)
	}

	if dbVM.HardwareJSON == "" {
		return nil, fmt.Errorf("no cached hardware info for VM %s", vmName)
	}

	var hardware libvirt.HardwareInfo
	if err := json.Unmarshal([]byte(dbVM.HardwareJSON), &hardware); err != nil {
		return nil, fmt.Errorf("could not parse cached hardware info for VM %s: %w", vmName, err)
	}

	return &hardware, nil
}

// GetVMHardwareAndTriggerSync serves cached hardware info from the DB immediately
// and triggers a background sync with libvirt.
func (s *HostService) GetVMHardwareAndTriggerSync(hostID, vmName string) (*libvirt.HardwareInfo, error) {
	hardware, err := s.getVMHardwareFromDB(hostID, vmName)
	if err != nil {
		log.Printf("Could not get cached hardware for %s, will attempt live sync.", vmName)
	}

	go func() {
		if changed, syncErr := s.syncSingleVM(hostID, vmName); syncErr == nil && changed {
			s.broadcastUpdate()
		} else if syncErr != nil {
			log.Printf("Error during background hardware sync for %s: %v", vmName, syncErr)
		}
	}()

	return hardware, err
}

// SyncVMsForHost triggers a background sync with libvirt for a specific host's VMs.
// It sends a websocket message upon completion *only if* there were changes.
func (s *HostService) SyncVMsForHost(hostID string) {
	changed, err := s.syncAndListVMs(hostID)
	if err != nil {
		log.Printf("Error during background VM sync for host %s: %v", hostID, err)
		return
	}
	if changed {
		s.broadcastUpdate()
	}
}

// syncSingleVM syncs the state of a single VM from libvirt to the database.
func (s *HostService) syncSingleVM(hostID, vmName string) (bool, error) {
	vmInfo, err := s.connector.GetDomainInfo(hostID, vmName)
	if err != nil {
		// If the VM is not found, it might have been deleted. Check if it exists in the DB.
		var dbVM storage.VirtualMachine
		if err := s.db.Where("host_id = ? AND name = ?", hostID, vmName).First(&dbVM).Error; err == nil {
			// VM exists in DB but not on host, so delete it.
			if err := s.db.Delete(&dbVM).Error; err != nil {
				log.Printf("Warning: failed to prune old VM %s: %v", dbVM.Name, err)
				return false, err
			}
			return true, nil
		}
		return false, fmt.Errorf("could not fetch info for VM %s on host %s: %w", vmName, hostID, err)
	}

	graphicsBytes, err := json.Marshal(vmInfo.Graphics)
	if err != nil {
		log.Printf("Warning: could not marshal graphics info for VM %s: %v", vmInfo.Name, err)
		graphicsBytes = []byte("{}")
	}

	hardwareInfo, err := s.connector.GetDomainHardware(hostID, vmName)
	if err != nil {
		log.Printf("Warning: could not fetch hardware for VM %s: %v", vmInfo.Name, err)
	}
	hardwareBytes, err := json.Marshal(hardwareInfo)
	if err != nil {
		log.Printf("Warning: could not marshal hardware info for VM %s: %v", vmInfo.Name, err)
		hardwareBytes = []byte("{}")
	}

	vmRecord := storage.VirtualMachine{
		HostID:       hostID,
		Name:         vmInfo.Name,
		UUID:         vmInfo.UUID,
		State:        int(vmInfo.State),
		VCPUCount:    vmInfo.Vcpu,
		MemoryBytes:  vmInfo.MaxMem * 1024,
		GraphicsJSON: string(graphicsBytes),
		HardwareJSON: string(hardwareBytes),
	}

	var existingVM storage.VirtualMachine
	if err := s.db.Where("uuid = ?", vmInfo.UUID).First(&existingVM).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if err := s.db.Create(&vmRecord).Error; err != nil {
				log.Printf("Warning: could not create VM %s in database: %v", vmInfo.Name, err)
				return false, err
			}
			return true, nil
		}
		return false, err
	}

	if existingVM.Name != vmRecord.Name ||
		existingVM.State != vmRecord.State ||
		existingVM.VCPUCount != vmRecord.VCPUCount ||
		existingVM.MemoryBytes != vmRecord.MemoryBytes ||
		existingVM.GraphicsJSON != vmRecord.GraphicsJSON ||
		existingVM.HardwareJSON != vmRecord.HardwareJSON {

		if err := s.db.Model(&existingVM).Updates(vmRecord).Error; err != nil {
			log.Printf("Warning: could not update VM %s in database: %v", vmInfo.Name, err)
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// syncAndListVMs is the core function to get VMs from libvirt and sync with the local DB.
// It returns true if any data was changed in the database.
func (s *HostService) syncAndListVMs(hostID string) (bool, error) {
	liveVMs, err := s.connector.ListAllDomains(hostID)
	if err != nil {
		return false, fmt.Errorf("service failed to list vms for host %s: %w", hostID, err)
	}

	var changed bool

	var dbVMs []storage.VirtualMachine
	if err := s.db.Where("host_id = ?", hostID).Find(&dbVMs).Error; err != nil {
		return false, fmt.Errorf("could not get DB records for comparison: %w", err)
	}
	dbVMMap := make(map[string]storage.VirtualMachine)
	for _, vm := range dbVMs {
		dbVMMap[vm.UUID] = vm
	}

	for _, vmInfo := range liveVMs {
		graphicsBytes, err := json.Marshal(vmInfo.Graphics)
		if err != nil {
			log.Printf("Warning: could not marshal graphics info for VM %s: %v", vmInfo.Name, err)
			graphicsBytes = []byte("{}")
		}

		hardwareInfo, err := s.connector.GetDomainHardware(hostID, vmInfo.Name)
		if err != nil {
			log.Printf("Warning: could not fetch hardware for VM %s: %v", vmInfo.Name, err)
		}
		hardwareBytes, err := json.Marshal(hardwareInfo)
		if err != nil {
			log.Printf("Warning: could not marshal hardware info for VM %s: %v", vmInfo.Name, err)
			hardwareBytes = []byte("{}")
		}

		vmRecord := storage.VirtualMachine{
			HostID:       hostID,
			Name:         vmInfo.Name,
			UUID:         vmInfo.UUID,
			State:        int(vmInfo.State),
			VCPUCount:    vmInfo.Vcpu,
			MemoryBytes:  vmInfo.MaxMem * 1024,
			GraphicsJSON: string(graphicsBytes),
			HardwareJSON: string(hardwareBytes),
		}

		existingVM, exists := dbVMMap[vmInfo.UUID]
		if !exists {
			if err := s.db.Create(&vmRecord).Error; err != nil {
				log.Printf("Warning: could not create VM %s in database: %v", vmInfo.Name, err)
			} else {
				changed = true
			}
		} else {
			if existingVM.Name != vmRecord.Name ||
				existingVM.State != vmRecord.State ||
				existingVM.VCPUCount != vmRecord.VCPUCount ||
				existingVM.MemoryBytes != vmRecord.MemoryBytes ||
				existingVM.GraphicsJSON != vmRecord.GraphicsJSON ||
				existingVM.HardwareJSON != vmRecord.HardwareJSON {

				if err := s.db.Model(&existingVM).Updates(vmRecord).Error; err != nil {
					log.Printf("Warning: could not update VM %s in database: %v", vmInfo.Name, err)
				} else {
					changed = true
				}
			}
		}
	}

	liveVMUUIDs := make(map[string]struct{})
	for _, vm := range liveVMs {
		liveVMUUIDs[vm.UUID] = struct{}{}
	}
	for _, dbVM := range dbVMs {
		if _, ok := liveVMUUIDs[dbVM.UUID]; !ok {
			if err := s.db.Delete(&dbVM).Error; err != nil {
				log.Printf("Warning: failed to prune old VM %s: %v", dbVM.Name, err)
			} else {
				changed = true
			}
		}
	}

	return changed, nil
}

func (s *HostService) GetVMStats(hostID, vmName string) (*libvirt.VMStats, error) {
	stats, err := s.connector.GetDomainStats(hostID, vmName)
	if err != nil {
		return nil, fmt.Errorf("service failed to get stats for vm %s on host %s: %w", vmName, hostID, err)
	}
	return stats, nil
}

// --- VM Actions ---

func (s *HostService) StartVM(hostID, vmName string) error {
	if err := s.connector.StartDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastUpdate()
	}
	return nil
}

func (s *HostService) ShutdownVM(hostID, vmName string) error {
	if err := s.connector.ShutdownDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastUpdate()
	}
	return nil
}

func (s *HostService) RebootVM(hostID, vmName string) error {
	if err := s.connector.RebootDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastUpdate()
	}
	return nil
}

func (s *HostService) ForceOffVM(hostID, vmName string) error {
	if err := s.connector.DestroyDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastUpdate()
	}
	return nil
}

func (s *HostService) ForceResetVM(hostID, vmName string) error {
	if err := s.connector.ResetDomain(hostID, vmName); err != nil {
		return err
	}
	if changed, err := s.syncSingleVM(hostID, vmName); err == nil && changed {
		s.broadcastUpdate()
	}
	return nil
}


