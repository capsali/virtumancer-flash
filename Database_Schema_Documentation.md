# Virtumancer Database Evolution Plan

## 1. Core Philosophy

The database remains the "source of truth" for user-defined configurations and metadata. Real-time state and performance metrics will continue to be fetched directly from libvirt hosts.

As Virtumancer evolves, the database will expand to store configurations for additional components (like storage and networking) and metadata for advanced features (like snapshots, templates, and auditing), providing a rich, persistent layer that complements the live data from libvirt.

## 2. Current Schema (Phase 1 - Complete)

The existing schema is the foundation upon which we will build.

* **`hosts`**: Stores connection information for each managed libvirt host.
    * `id` (TEXT, PK)
    * `uri` (TEXT)
* **`virtual_machines`**: Caches basic VM configuration discovered from libvirt.
    * `id` (INTEGER, PK)
    * `name` (TEXT)
    * `host_id` (TEXT, FK to hosts.id)
    * `config_json` (TEXT)

## 3. Proposed Schema for Future Phases

The following tables and modifications are proposed to support the features outlined in the UI/UX plan.

### Core Entity Redefinition: The Virtumancer VM Model

To enable advanced features, we must evolve the `virtual_machines` table from a simple cache of libvirt's configuration into Virtumancer's own canonical definition of a VM. This decouples our application from the live state, making Virtumancer the authoritative source for a VM's intended configuration.

* **`virtual_machines` (Revised)**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `host_id` | `TEXT` | Foreign Key to `hosts.id`. Defines where the VM runs. |
    | `name` | `TEXT` | User-defined name for the VM. |
    | `uuid` | `TEXT` | The libvirt UUID of the domain. This is the key link to the live resource. |
    | `description` | `TEXT` | User-provided description or notes. |
    | `vcpu_count` | `INTEGER` | The number of virtual CPUs assigned. |
    | `cpu_model` | `TEXT` | CPU model (e.g., 'host-passthrough', 'Cascadelake-Server'). |
    | `cpu_topology_json` | `TEXT` | JSON for sockets, cores, threads. |
    | `memory_bytes` | `INTEGER` | The amount of RAM in bytes assigned to the VM. |
    | `os_type` | `TEXT` | Operating system type hint (e.g., 'linux', 'windows'). |
    | `is_template` | `BOOLEAN` | `true` if this VM is a template for cloning. |
    | `created_at` | `DATETIME` | |
    | `updated_at` | `DATETIME` | |

### Storage Management (Phase 2)

* **`storage_pools`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `host_id` | `TEXT` | Foreign Key to `hosts.id` |
    | `name` | `TEXT` | User-friendly name (e.g., "local-lvm") |
    | `uuid` | `TEXT` | Libvirt pool UUID for stable identification |
    | `type` | `TEXT` | Pool type (e.g., 'dir', 'lvm', 'nfs') |
    | `path` | `TEXT` | Target path for the pool |
    | `capacity_bytes` | `INTEGER` | Total capacity in bytes |
    | `allocation_bytes` | `INTEGER` | Used space in bytes |

* **`volumes`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `storage_pool_id` | `INTEGER` | Foreign Key to `storage_pools.id` |
    | `name` | `TEXT` | Volume name (e.g., "vm-101-disk-0.qcow2", "ubuntu.iso") |
    | `type` | `TEXT` | 'DISK' or 'ISO' |
    | `format` | `TEXT` | Volume format (e.g., 'qcow2', 'raw') |
    | `capacity_bytes` | `INTEGER` | Total capacity in bytes |
    | `allocation_bytes` | `INTEGER` | Used space in bytes |

* **`volume_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `volume_id` | `INTEGER` | Foreign Key to `volumes.id` |
    | `device_name` | `TEXT` | Target device name inside the VM (e.g., "vda", "hdb") |
    | `bus_type` | `TEXT` | The bus type used for attachment (e.g., "virtio", "sata", "ide") |
    | `is_read_only` | `BOOLEAN` | Whether the volume is attached in read-only mode |

### Network Management (Phase 2)

* **`networks`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `host_id` | `TEXT` | Foreign Key to `hosts.id` |
    | `name` | `TEXT` | User-friendly name (e.g., "vmbr0") |
    | `uuid` | `TEXT` | Libvirt network UUID |
    | `bridge_name` | `TEXT` | Name of the bridge device on the host |
    | `mode` | `TEXT` | Network mode (e.g., 'bridged', 'nat', 'isolated') |

* **`ports`** (for vNICs)
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `mac_address` | `TEXT` | Unique MAC address of the virtual NIC |
    | `model` | `TEXT` | NIC model (e.g., 'virtio', 'e1000') |
    | `ip_address` | `TEXT` | IP address assigned to the port (if known) |

* **`port_bindings`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `port_id` | `INTEGER` | Foreign Key to `ports.id` |
    | `network_id` | `INTEGER` | Foreign Key to `networks.id` |

### Virtual Hardware Management (Phase 2/3)

* **`controllers`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `type` | `TEXT` | Controller type (e.g., 'usb', 'sata', 'virtio-serial') |
    | `model` | `TEXT` | Specific model (e.g., 'ich9-ehci1', 'ahci') |
    | `index` | `INTEGER`| Controller index for ordering |

* **`controller_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `controller_id` | `INTEGER`| Foreign Key to `controllers.id` |

* **`input_devices`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `type` | `TEXT` | 'mouse', 'tablet', or 'keyboard' |
    | `bus` | `TEXT` | 'usb', 'ps2', or 'virtio' |

* **`input_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `input_device_id`| `INTEGER` | Foreign Key to `input_devices.id` |

* **`graphics_devices`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `protocol_type` | `TEXT` | Display protocol (e.g., 'vnc', 'spice') |
    | `video_model` | `TEXT` | Video card model (e.g., 'qxl', 'vga', 'virtio') |
    | `vram_kib` | `INTEGER` | Video RAM in kibibytes |
    | `listen_address`| `TEXT` | IP address to listen on (e.g., '0.0.0.0' or '127.0.0.1') |

* **`graphics_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `graphics_device_id`| `INTEGER`| Foreign Key to `graphics_devices.id` |

* **`sound_cards`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `model` | `TEXT` | Sound card model (e.g., 'ich6', 'ac97') |

* **`sound_card_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `sound_card_id`| `INTEGER` | Foreign Key to `sound_cards.id` |

* **`host_devices`** (for passthrough)
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `host_id` | `TEXT` | Foreign Key to `hosts.id` |
    | `type` | `TEXT` | 'pci' or 'usb' |
    | `address` | `TEXT` | Physical address on host (e.g., PCI address or USB bus/device) |
    | `description` | `TEXT` | Human-readable description of the device |

* **`host_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `host_device_id`| `INTEGER` | Foreign Key to `host_devices.id` |

* **`tpms`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `model` | `TEXT` | 'tpm-crb', 'tpm-tis' |
    | `backend_type` | `TEXT` | 'passthrough', 'emulator' |
    | `backend_path` | `TEXT` | Path to TPM device on host if passthrough |

* **`tpm_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `tpm_id`| `INTEGER` | Foreign Key to `tpms.id` |

* **`watchdogs`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `model` | `TEXT` | Watchdog model (e.g., 'i6300esb') |
    | `action` | `TEXT` | Action to take (e.g., 'reset', 'shutdown', 'poweroff') |

* **`watchdog_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `watchdog_id`| `INTEGER` | Foreign Key to `watchdogs.id` |

* **`serial_devices`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `type` | `TEXT` | e.g., 'pty', 'tcp', 'stdio' |
    | `target_port` | `INTEGER`| e.g., 0 for COM1 |
    | `config_json` | `TEXT` | JSON for source-specific settings |

* **`serial_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `serial_device_id`| `INTEGER` | Foreign Key to `serial_devices.id` |

* **`channel_devices`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `type` | `TEXT` | e.g., 'unix', 'spicevmc' |
    | `target_name` | `TEXT` | e.g., 'org.qemu.guest_agent.0' |
    | `config_json` | `TEXT` | JSON for source-specific settings |

* **`channel_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `channel_device_id`| `INTEGER` | Foreign Key to `channel_devices.id` |

* **`filesystems`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `driver_type` | `TEXT` | Filesystem driver (e.g., 'path') |
    | `source_path` | `TEXT` | Path on the host |
    | `target_path` | `TEXT` | Mount tag/path inside the guest |

* **`filesystem_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `filesystem_id`| `INTEGER` | Foreign Key to `filesystems.id` |

* **`smartcards`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `type` | `TEXT` | 'passthrough', 'emulated' |
    | `config_json` | `TEXT` | JSON for device-specific settings |

* **`smartcard_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `smartcard_id`| `INTEGER` | Foreign Key to `smartcards.id` |

* **`usb_redirectors`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `type` | `TEXT` | e.g., 'spicevmc' |
    | `filter_rule` | `TEXT` | USB filter rule string |

* **`usb_redirector_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `usb_redirector_id`| `INTEGER` | Foreign Key to `usb_redirectors.id` |

* **`rng_devices`** (Random Number Generator)
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `model` | `TEXT` | 'virtio' |
    | `backend_type` | `TEXT` | e.g., '/dev/random', 'builtin' |

* **`rng_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `rng_device_id`| `INTEGER` | Foreign Key to `rng_devices.id` |

* **`panic_devices`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `model` | `TEXT` | 'isa', 'hyperv' |

* **`panic_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `panic_device_id`| `INTEGER` | Foreign Key to `panic_devices.id` |

* **`vsocks`** (VirtIO Sockets)
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `guest_cid` | `INTEGER` | Guest Context ID (auto-assigned if 0) |

* **`vsock_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `vsock_id` | `INTEGER` | Foreign Key to `vsocks.id` |

* **`memory_balloons`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `model` | `TEXT` | 'virtio' is the most common |
    | `config_json`| `TEXT` | JSON for specific tuning options |

* **`memory_balloon_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `memory_balloon_id`| `INTEGER` | Foreign Key to `memory_balloons.id` |

* **`shmem_devices`** (Shared Memory)
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `name` | `TEXT` | Name of the shared memory device |
    | `size_kib` | `INTEGER` | Size of the shared memory region in KiB |
    | `path` | `TEXT` | Path on host for backing, if applicable |

* **`shmem_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `shmem_device_id`| `INTEGER` | Foreign Key to `shmem_devices.id` |

* **`iommu_devices`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `model` | `TEXT` | 'intel' or 'amd' |

* **`iommu_device_attachments`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `iommu_device_id`| `INTEGER` | Foreign Key to `iommu_devices.id` |

### VM Templating & Snapshots (Phase 3)

* **`vm_snapshots`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `vm_id` | `INTEGER` | Foreign Key to `virtual_machines.id` |
    | `name` | `TEXT` | Libvirt snapshot name (unique per VM) |
    | `description` | `TEXT` | User-provided description |
    | `parent_name` | `TEXT` | Name of the parent snapshot, if any |
    | `created_at` | `DATETIME` | Timestamp of snapshot creation |
    | `state` | `TEXT` | 'RUNNING' or 'SHUTDOWN' at time of snapshot |
    | `config_xml` | `TEXT` | A dump of the snapshot's libvirt XML for restoration |

### Users, Roles & Permissions (Phase 3)

* **`users`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `username` | `TEXT` | Unique username for login |
    | `password_hash` | `TEXT` | Hashed password |
    | `role_id` | `INTEGER` | Foreign Key to `roles.id` |

* **`roles`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `name` | `TEXT` | Role name (e.g., "Administrator", "VM User") |

* **`permissions`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `action` | `TEXT` | Granular action name (e.g., "vm.start", "host.create") |
    | `description` | `TEXT` | Human-readable description of the permission |

* **`role_permissions`** (Join Table)
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `role_id` | `INTEGER` | Foreign Key to `roles.id` |
    | `permission_id` | `INTEGER` | Foreign Key to `permissions.id` |

### Tasks & Auditing (Phase 3)

* **`tasks`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `user_id` | `INTEGER` | Foreign Key to `users.id` (who initiated it) |
    | `type` | `TEXT` | Task type (e.g., "vm.clone", "backup.run") |
    | `status` | `TEXT` | 'PENDING', 'RUNNING', 'COMPLETED', 'FAILED' |
    | `progress` | `INTEGER` | Progress percentage (0-100) |
    | `details` | `TEXT` | JSON blob with task-specific info or error messages |
    | `created_at` | `DATETIME` | |
    | `updated_at` | `DATETIME` | |

* **`audit_log`**
    | Column | Type | Description |
    | :--- | :--- | :--- |
    | `id` | `INTEGER` | Primary Key |
    | `user_id` | `INTEGER` | Foreign Key to `users.id` |
    | `timestamp` | `DATETIME` | Time of the event |
    | `action` | `TEXT` | The action performed (e.g., "vm.shutdown") |
    | `target_type` | `TEXT` | The type of object acted upon (e.g., "VM", "Host") |
    | `target_id` | `TEXT` | The ID/Name of the target object |
    | `details` | `TEXT` | JSON blob with relevant event data |

## 4. Entity Relationships

* A `Host` has many `VirtualMachines`, `StoragePools`, `Networks`, and `HostDevices`.
* A `VirtualMachine` has many attachments: `VolumeAttachments`, `ControllerAttachments`, `InputDeviceAttachments`, `GraphicsDeviceAttachments`, `SoundCardAttachments`, `HostDeviceAttachments`, `TpmAttachments`, `WatchdogAttachments`, `SerialDeviceAttachments`, `ChannelDeviceAttachments`, `FilesystemAttachments`, `SmartcardAttachments`, `UsbRedirectorAttachments`, `RngDeviceAttachments`, `PanicDeviceAttachments`, `VsockAttachments`, `MemoryBalloonAttachments`, `ShmemDeviceAttachments`, `IommuDeviceAttachments`.
* A `VirtualMachine` has many `Ports` (vNICs) and `VMSnapshots`.
* Each hardware type (e.g., `Volume`, `Controller`, `Filesystem`) is attached to a VM via its corresponding attachment table.
* A `Port` is bound to a `Network` via a `PortBinding`.
* A `User` has one `Role`, which has many `Permissions`.
* A `User` has many `Tasks` and many `AuditLog` entries.

This schema provides a robust and scalable data model to support Virtumancer's growth into
