# **Virtumancer Database Schema**

Virtumancer uses a SQLite database (virtumancer.db) with a normalized relational schema to store host configurations and cache virtual machine hardware details. The schema is managed by GORM and is automatically migrated on application startup.

## **Table Definitions**

### **hosts**

Stores the connection details for each managed libvirt host.

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | TEXT | PRIMARY KEY | A user-defined, unique ID for the host. |
| uri | TEXT | NOT NULL | The full libvirt connection URI. |
| created\_at | DATETIME |  | Timestamp of creation. |

### **virtual\_machines**

The central table for virtual machines, caching their basic state and configuration.

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INTEGER | PRIMARY KEY | Auto-incrementing primary key. |
| host\_id | TEXT | NOT NULL | Foreign key to the hosts table. |
| uuid | TEXT | UNIQUE, NOT NULL | The libvirt-assigned unique ID of the VM. |
| name | TEXT |  | The name of the VM. |
| description | TEXT |  | A user-defined description. |
| vcpu\_count | INTEGER |  | Number of virtual CPUs. |
| memory\_bytes | INTEGER |  | Maximum memory allocated in bytes. |
| state | INTEGER |  | The last known power state from libvirt. |
| is\_template | BOOLEAN |  | (Future Use) If the VM is a template. |
| cpu\_model | TEXT |  | (Future Use) The configured CPU model. |
| cpu\_topology\_json | TEXT |  | (Future Use) JSON blob for sockets, cores, threads. |

### **volumes**

Represents storage volumes (virtual disks, ISOs).

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INTEGER | PRIMARY KEY | Auto-incrementing primary key. |
| name | TEXT | UNIQUE | The unique name/path of the storage volume. |
| type | TEXT |  | The type of volume, e.g., DISK, ISO. |
| format | TEXT |  | The disk format, e.g., qcow2, raw. |

### **volume\_attachments**

A join table linking virtual\_machines to volumes.

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INTEGER | PRIMARY KEY | Auto-incrementing primary key. |
| vm\_id | INTEGER |  | Foreign key to virtual\_machines. |
| volume\_id | INTEGER |  | Foreign key to volumes. |
| device\_name | TEXT |  | The device name inside the guest, e.g., vda. |
| bus\_type | TEXT |  | The bus type, e.g., virtio, sata. |

### **networks**

Represents virtual networks on a host.

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INTEGER | PRIMARY KEY | Auto-incrementing primary key. |
| host\_id | TEXT | NOT NULL | Foreign key to the hosts table. |
| uuid | TEXT | UNIQUE | The libvirt-assigned UUID (can be empty). |
| name | TEXT |  | The name of the network. |
| bridge\_name | TEXT |  | The name of the host bridge interface. |
| mode | TEXT |  | The network mode, e.g., bridged. |

*Note: A UNIQUE constraint exists on the combination of (host\_id, name).*

### **ports**

Represents a virtual network interface (vNIC).

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INTEGER | PRIMARY KEY | Auto-incrementing primary key. |
| vm\_id | INTEGER |  | Foreign key to virtual\_machines. |
| mac\_address | TEXT | UNIQUE | The unique MAC address of the vNIC. |
| model\_name | TEXT |  | The vNIC model, e.g., virtio. |

### **port\_bindings**

A join table linking a port to a network.

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INTEGER | PRIMARY KEY | Auto-incrementing primary key. |
| port\_id | INTEGER |  | Foreign key to ports. |
| network\_id | INTEGER |  | Foreign key to networks. |

### **graphics\_devices**

Represents a graphical console device type.

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INTEGER | PRIMARY KEY | Auto-incrementing primary key. |
| type | TEXT |  | The type of console, e.g., vnc, spice. |
| model\_name | TEXT |  | The graphics model, e.g., qxl. |
| vram\_kib | INTEGER |  | (Future Use) Video RAM in KiB. |
| listen\_address | TEXT |  | (Future Use) The listen address. |

### **graphics\_device\_attachments**

A join table linking a virtual\_machine to a graphics\_device.

| Column | Type | Constraints | Description |
| :---- | :---- | :---- | :---- |
| id | INTEGER | PRIMARY KEY | Auto-incrementing primary key. |
| vm\_id | INTEGER |  | Foreign key to virtual\_machines. |
| graphics\_device\_id | INTEGER |  | Foreign key to graphics\_devices. |


