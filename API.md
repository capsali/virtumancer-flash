# **Virtumancer API Documentation**

Virtumancer exposes a RESTful HTTP API for management operations and a WebSocket API for real-time updates and monitoring.

## **REST API**

Base URL: /api

### **Host Management**

#### **GET /api/hosts**

* **Description**: Retrieves a list of all configured hosts from the database.  
* **Response**: 200 OK  
  \[  
    {  
      "id": "kvmsrv",  
      "uri": "qemu+ssh://user@host/system",  
      "created\_at": "2023-10-27T10:00:00Z"  
    }  
  \]

#### **POST /api/hosts**

* **Description**: Adds a new host, connects to it, and stores it in the database.  
* **Request Body**:  
  {  
    "id": "new-kvm-host",  
    "uri": "qemu+ssh://user@new-host/system"  
  }

* **Response**: 200 OK on success, with the created host object. 500 Internal Server Error if the connection fails.

#### **DELETE /api/hosts/:id**

* **Description**: Disconnects from a host and removes it from the database.  
* **URL Parameters**:  
  * id (string): The ID of the host to remove.  
* **Response**: 204 No Content

#### **GET /api/hosts/:id/info**

* **Description**: Retrieves real-time information and statistics about a specific host (CPU, memory, etc.).  
* **URL Parameters**:  
  * id (string): The ID of the host.  
* **Response**: 200 OK  
  {  
    "hostname": "kvm-host-01",  
    "cpu": 8,  
    "memory": 16777216000,  
    "cores": 4,  
    "threads": 2  
  }

### **Virtual Machine Management**

#### **GET /api/hosts/:id/vms**

* **Description**: Retrieves a list of all virtual machines on a specific host from the local database cache.  
* **URL Parameters**:  
  * id (string): The ID of the host.  
* **Response**: 200 OK  
  \[  
    {  
      "db\_id": 1,  
      "name": "ubuntu-vm-01",  
      "description": "",  
      "vcpu\_count": 2,  
      "memory\_bytes": 2147483648,  
      "state": 1,  
      "graphics": {  
        "vnc": true,  
        "spice": false  
      }  
    }  
  \]

#### **GET /api/hosts/:hostId/vms/:vmName/hardware**

* **Description**: Retrieves the hardware configuration for a specific VM. This triggers a fresh sync from libvirt before returning the cached data.  
* **URL Parameters**:  
  * hostId (string): The ID of the host.  
  * vmName (string): The name of the virtual machine.  
* **Response**: 200 OK  
  {  
    "disks": \[  
      {  
        "type": "file",  
        "device": "disk",  
        "driver": { "driver\_name": "qemu", "type": "qcow2" },  
        "path": "/path/to/disk.qcow2",  
        "target": { "dev": "vda", "bus": "virtio" }  
      }  
    \],  
    "networks": \[  
      {  
        "type": "bridge",  
        "mac": { "address": "52:54:00:11:22:33" },  
        "source": { "bridge": "br0" },  
        "model": { "model\_type": "virtio" }  
      }  
    \]  
  }

#### **POST /api/hosts/:hostId/vms/:vmName/action**

* **Description**: Performs a power action on a specific VM.  
* **URL Parameters**:  
  * hostId (string): The ID of the host.  
  * vmName (string): The name of the virtual machine.  
* **Request Body**:  
  {  
    "action": "start"  
  }

  * **Valid actions**: start, shutdown, reboot, destroy (force off), reset (force reset).  
* **Response**: 204 No Content

## **WebSocket API**

The WebSocket API is used for real-time notifications and statistics monitoring.

* **Connection URL**: /ws

### **Client-to-Server Messages**

Messages are sent as JSON objects with type and payload fields.

#### **subscribe-vm-stats**

* **Description**: Subscribes the client to real-time statistics updates for a specific VM. The server will start polling the VM and broadcasting vm-stats-updated messages.  
* **Payload**:  
  {  
    "type": "subscribe-vm-stats",  
    "payload": {  
      "hostId": "kvmsrv",  
      "vmName": "ubuntu-vm-01"  
    }  
  }

#### **unsubscribe-vm-stats**

* **Description**: Unsubscribes the client from a VM's statistics updates. If no clients are left subscribed, the server will stop polling.  
* **Payload**:  
  {  
    "type": "unsubscribe-vm-stats",  
    "payload": {  
      "hostId": "kvmsrv",  
      "vmName": "ubuntu-vm-01"  
    }  
  }

### **Server-to-Client Messages**

#### **hosts-changed**

* **Description**: Sent whenever a host is added or removed. The client should re-fetch the list of hosts via GET /api/hosts.  
* **Payload**: null

#### **vms-changed**

* **Description**: Sent whenever the list of VMs on a host has changed (e.g., a VM was added, removed, or its state changed after a power operation). The client should re-fetch the VM list for the specified host.  
* **Payload**:  
  {  
    "type": "vms-changed",  
    "payload": {  
      "hostId": "kvmsrv"  
    }  
  }

#### **vm-stats-updated**

* **Description**: Broadcast periodically to all subscribed clients for a specific VM.  
* **Payload**:  
  {  
    "type": "vm-stats-updated",  
    "payload": {  
      "hostId": "kvmsrv",  
      "vmName": "ubuntu-vm-01",  
      "stats": {  
        "state": 1,  
        "memory": 2097152,  
        "max\_mem": 2097152,  
        "vcpu": 2,  
        "cpu\_time": 1234567890,  
        "disk\_stats": \[  
          { "device": "vda", "read\_bytes": 1024, "write\_bytes": 2048 }  
        \],  
        "net\_stats": \[  
          { "device": "vnet0", "read\_bytes": 4096, "write\_bytes": 8192 }  
        \]  
      }  
    }  
  }  

