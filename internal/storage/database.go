package storage

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- Core Entities ---

// Host represents a libvirt host connection configuration.
type Host struct {
	ID  string `gorm:"primaryKey" json:"id"`
	URI string `json:"uri"`
}

// VirtualMachine is Virtumancer's canonical definition of a VM's intended state.
type VirtualMachine struct {
	gorm.Model
	HostID          string `gorm:"uniqueIndex:idx_vm_host_name"`
	Name            string `gorm:"uniqueIndex:idx_vm_host_name"`
	UUID            string `gorm:"uniqueIndex"`
	Description     string
	State           int    `gorm:"default:-1"` // Caches the last known libvirt.DomainState
	GraphicsJSON    string // Caches the JSON representation of libvirt.GraphicsInfo
	HardwareJSON    string // Caches the JSON representation of libvirt.HardwareInfo
	VCPUCount       uint
	CPUModel        string
	CPUTopologyJSON string
	MemoryBytes     uint64
	OSType          string
	IsTemplate      bool
}

// --- Storage Management ---

// StoragePool represents a libvirt storage pool (e.g., LVM, a directory).
type StoragePool struct {
	gorm.Model
	HostID          string
	Name            string
	UUID            string `gorm:"uniqueIndex"`
	Type            string
	Path            string
	CapacityBytes   uint64
	AllocationBytes uint64
}

// Volume represents a single storage volume, like a virtual disk or an ISO.
type Volume struct {
	gorm.Model
	StoragePoolID   uint
	Name            string
	Type            string // 'DISK' or 'ISO'
	Format          string
	CapacityBytes   uint64
	AllocationBytes uint64
}

// VolumeAttachment links a Volume to a VirtualMachine.
type VolumeAttachment struct {
	gorm.Model
	VMID       uint
	VolumeID   uint
	DeviceName string // e.g., "vda", "hdb"
	BusType    string // e.g., "virtio", "sata", "ide"
	IsReadOnly bool
}

// --- Network Management ---

// Network represents a virtual network or bridge on a host.
type Network struct {
	gorm.Model
	HostID     string
	Name       string
	UUID       string `gorm:"uniqueIndex"`
	BridgeName string
	Mode       string // e.g., 'bridged', 'nat', 'isolated'
}

// Port represents a virtual Network Interface Card (vNIC) belonging to a VM.
type Port struct {
	gorm.Model
	VMID       uint
	MACAddress string `gorm:"uniqueIndex"`
	ModelName  string // e.g., 'virtio', 'e1000'
	IPAddress  string
}

// PortBinding links a Port to a Network.
type PortBinding struct {
	gorm.Model
	PortID    uint
	NetworkID uint
}

// --- Virtual Hardware Management ---

// Controller represents a hardware controller within a VM (e.g., USB, SATA).
type Controller struct {
	gorm.Model
	Type      string // 'usb', 'sata', 'virtio-serial'
	ModelName string
	Index     uint
}

// ControllerAttachment links a Controller to a VirtualMachine.
type ControllerAttachment struct {
	gorm.Model
	VMID         uint
	ControllerID uint
}

// InputDevice represents an input device like a mouse or keyboard.
type InputDevice struct {
	gorm.Model
	Type string // 'mouse', 'tablet', 'keyboard'
	Bus  string // 'usb', 'ps2', 'virtio'
}

// InputDeviceAttachment links an InputDevice to a VirtualMachine.
type InputDeviceAttachment struct {
	gorm.Model
	VMID          uint
	InputDeviceID uint
}

// GraphicsDevice represents a virtual GPU and display protocol configuration.
type GraphicsDevice struct {
	gorm.Model
	Type          string // 'vnc', 'spice'
	ModelName     string // 'qxl', 'vga', 'virtio'
	VRAMKiB       uint
	ListenAddress string
}

// GraphicsDeviceAttachment links a GraphicsDevice to a VirtualMachine.
type GraphicsDeviceAttachment struct {
	gorm.Model
	VMID             uint
	GraphicsDeviceID uint
}

// SoundCard represents a virtual sound device.
type SoundCard struct {
	gorm.Model
	ModelName string // 'ich6', 'ac97'
}

// SoundCardAttachment links a SoundCard to a VirtualMachine.
type SoundCardAttachment struct {
	gorm.Model
	VMID        uint
	SoundCardID uint
}

// HostDevice represents a physical device on a host for passthrough.
type HostDevice struct {
	gorm.Model
	HostID      string
	Type        string // 'pci', 'usb'
	Address     string // Physical address on host
	Description string
}

// HostDeviceAttachment links a HostDevice to a VirtualMachine for passthrough.
type HostDeviceAttachment struct {
	gorm.Model
	VMID         uint
	HostDeviceID uint
}

// TPM represents a Trusted Platform Module device.
type TPM struct {
	gorm.Model
	ModelName   string // 'tpm-crb', 'tpm-tis'
	BackendType string // 'passthrough', 'emulator'
	BackendPath string
}

// TPMAttachment links a TPM to a VirtualMachine.
type TPMAttachment struct {
	gorm.Model
	VMID  uint
	TPMID uint
}

// Watchdog represents a virtual watchdog device.
type Watchdog struct {
	gorm.Model
	ModelName string // 'i6300esb'
	Action    string // 'reset', 'shutdown', 'poweroff'
}

// WatchdogAttachment links a Watchdog to a VirtualMachine.
type WatchdogAttachment struct {
	gorm.Model
	VMID       uint
	WatchdogID uint
}

// SerialDevice represents a serial port configuration.
type SerialDevice struct {
	gorm.Model
	Type       string // 'pty', 'tcp', 'stdio'
	TargetPort uint
	ConfigJSON string
}

// SerialDeviceAttachment links a SerialDevice to a VirtualMachine.
type SerialDeviceAttachment struct {
	gorm.Model
	VMID           uint
	SerialDeviceID uint
}

// ChannelDevice represents a communication channel (e.g., for guest agent).
type ChannelDevice struct {
	gorm.Model
	Type       string // 'unix', 'spicevmc'
	TargetName string // e.g., 'org.qemu.guest_agent.0'
	ConfigJSON string
}

// ChannelDeviceAttachment links a ChannelDevice to a VirtualMachine.
type ChannelDeviceAttachment struct {
	gorm.Model
	VMID            uint
	ChannelDeviceID uint
}

// Filesystem represents a shared filesystem for a VM.
type Filesystem struct {
	gorm.Model
	DriverType  string
	SourcePath  string
	TargetPath  string
}

// FilesystemAttachment links a Filesystem to a VM.
type FilesystemAttachment struct {
	gorm.Model
	VMID         uint
	FilesystemID uint
}

// Smartcard represents a smartcard device for a VM.
type Smartcard struct {
	gorm.Model
	Type       string
	ConfigJSON string
}

// SmartcardAttachment links a Smartcard to a VM.
type SmartcardAttachment struct {
	gorm.Model
	VMID        uint
	SmartcardID uint
}

// USBRedirector represents a USB redirection device.
type USBRedirector struct {
	gorm.Model
	Type       string
	FilterRule string
}

// USBRedirectorAttachment links a USBRedirector to a VM.
type USBRedirectorAttachment struct {
	gorm.Model
	VMID            uint
	USBRedirectorID uint
}

// RngDevice represents a Random Number Generator device.
type RngDevice struct {
	gorm.Model
	ModelName   string
	BackendType string
}

// RngDeviceAttachment links an RngDevice to a VM.
type RngDeviceAttachment struct {
	gorm.Model
	VMID        uint
	RngDeviceID uint
}

// PanicDevice represents a panic device for a VM.
type PanicDevice struct {
	gorm.Model
	ModelName string
}

// PanicDeviceAttachment links a PanicDevice to a VM.
type PanicDeviceAttachment struct {
	gorm.Model
	VMID          uint
	PanicDeviceID uint
}

// Vsock represents a VirtIO socket device.
type Vsock struct {
	gorm.Model
	GuestCID uint
}

// VsockAttachment links a Vsock to a VM.
type VsockAttachment struct {
	gorm.Model
	VMID    uint
	VsockID uint
}

// MemoryBalloon represents a memory balloon device.
type MemoryBalloon struct {
	gorm.Model
	ModelName  string
	ConfigJSON string
}

// MemoryBalloonAttachment links a MemoryBalloon to a VM.
type MemoryBalloonAttachment struct {
	gorm.Model
	VMID            uint
	MemoryBalloonID uint
}

// ShmemDevice represents a shared memory device.
type ShmemDevice struct {
	gorm.Model
	Name    string
	SizeKiB uint
	Path    string
}

// ShmemDeviceAttachment links a ShmemDevice to a VM.
type ShmemDeviceAttachment struct {
	gorm.Model
	VMID          uint
	ShmemDeviceID uint
}

// IOMMUDevice represents an IOMMU device.
type IOMMUDevice struct {
	gorm.Model
	ModelName string
}

// IOMMUDeviceAttachment links an IOMMUDevice to a VM.
type IOMMUDeviceAttachment struct {
	gorm.Model
	VMID          uint
	IOMMUDeviceID uint
}

// --- Advanced Features ---

// VMSnapshot stores metadata about a VM snapshot.
type VMSnapshot struct {
	gorm.Model
	VMID        uint
	Name        string
	Description string
	ParentName  string
	State       string
	ConfigXML   string
}

// User represents a Virtumancer user account.
type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex"`
	PasswordHash string
	RoleID       uint
}

// Role defines a set of permissions.
type Role struct {
	gorm.Model
	Name        string       `gorm:"uniqueIndex"`
	Permissions []Permission `gorm:"many2many:role_permissions;"`
}

// Permission is a granular action that can be performed.
type Permission struct {
	gorm.Model
	Action      string `gorm:"uniqueIndex"`
	Description string
}

// Task tracks a long-running, asynchronous operation.
type Task struct {
	gorm.Model
	UserID   uint
	Type     string
	Status   string
	Progress int
	Details  string
}

// AuditLog records an event that occurred in the system.
type AuditLog struct {
	gorm.Model
	UserID     uint
	Action     string
	TargetType string
	TargetID   string
	Details    string
}

// InitDB initializes and returns a GORM database instance.
func InitDB(dataSourceName string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dataSourceName), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the full schema
	err = db.AutoMigrate(
		&Host{},
		&VirtualMachine{},
		&StoragePool{},
		&Volume{},
		&VolumeAttachment{},
		&Network{},
		&Port{},
		&PortBinding{},
		&Controller{},
		&ControllerAttachment{},
		&InputDevice{},
		&InputDeviceAttachment{},
		&GraphicsDevice{},
		&GraphicsDeviceAttachment{},
		&SoundCard{},
		&SoundCardAttachment{},
		&HostDevice{},
		&HostDeviceAttachment{},
		&TPM{},
		&TPMAttachment{},
		&Watchdog{},
		&WatchdogAttachment{},
		&SerialDevice{},
		&SerialDeviceAttachment{},
		&ChannelDevice{},
		&ChannelDeviceAttachment{},
		&Filesystem{},
		&FilesystemAttachment{},
		&Smartcard{},
		&SmartcardAttachment{},
		&USBRedirector{},
		&USBRedirectorAttachment{},
		&RngDevice{},
		&RngDeviceAttachment{},
		&PanicDevice{},
		&PanicDeviceAttachment{},
		&Vsock{},
		&VsockAttachment{},
		&MemoryBalloon{},
		&MemoryBalloonAttachment{},
		&ShmemDevice{},
		&ShmemDeviceAttachment{},
		&IOMMUDevice{},
		&IOMMUDeviceAttachment{},
		&VMSnapshot{},
		&User{},
		&Role{},
		&Permission{},
		&Task{},
		&AuditLog{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}




