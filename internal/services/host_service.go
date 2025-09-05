package services

import (
	"fmt"
	"log"

	"github.com/capsali/virtumancer/internal/libvirt"
	"github.com/capsali/virtumancer/internal/storage"
	"github.com/capsali/virtumancer/internal/ws"
	"gorm.io/gorm"
)

type HostService struct {
	db        *gorm.DB
	connector *libvirt.Connector
	hub       *ws.Hub // <-- Add Hub
}

func NewHostService(db *gorm.DB, connector *libvirt.Connector, hub *ws.Hub) *HostService {
	return &HostService{
		db:        db,
		connector: connector,
		hub:       hub, // <-- Initialize Hub
	}
}

// broadcastUpdate sends a message to all WebSocket clients to refresh their state.
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

	s.broadcastUpdate() // Broadcast update after successful action
	return &host, nil
}

func (s *HostService) RemoveHost(hostID string) error {
	if err := s.connector.RemoveHost(hostID); err != nil {
		log.Printf("Warning: failed to disconnect from host %s during removal, continuing with DB deletion: %v", hostID, err)
	}

	if err := s.db.Where("id = ?", hostID).Delete(&storage.Host{}).Error; err != nil {
		return fmt.Errorf("failed to delete host from database: %w", err)
	}
	s.broadcastUpdate() // Broadcast update after successful action
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
		}
	}
}

// --- VM Management ---

func (s *HostService) ListVMs(hostID string) ([]libvirt.VMInfo, error) {
	vms, err := s.connector.ListAllDomains(hostID)
	if err != nil {
		return nil, fmt.Errorf("service failed to list vms for host %s: %w", hostID, err)
	}
	return vms, nil
}

func (s *HostService) StartVM(hostID, vmName string) error {
	if err := s.connector.StartDomain(hostID, vmName); err != nil {
		return err
	}
	s.broadcastUpdate()
	return nil
}

func (s *HostService) ShutdownVM(hostID, vmName string) error {
	if err := s.connector.ShutdownDomain(hostID, vmName); err != nil {
		return err
	}
	s.broadcastUpdate()
	return nil
}

func (s *HostService) RebootVM(hostID, vmName string) error {
	if err := s.connector.RebootDomain(hostID, vmName); err != nil {
		return err
	}
	s.broadcastUpdate()
	return nil
}

func (s *HostService) ForceOffVM(hostID, vmName string) error {
	if err := s.connector.DestroyDomain(hostID, vmName); err != nil {
		return err
	}
	s.broadcastUpdate()
	return nil
}

func (s *HostService) ForceResetVM(hostID, vmName string) error {
	if err := s.connector.ResetDomain(hostID, vmName); err != nil {
		return err
	}
	s.broadcastUpdate()
	return nil
}


