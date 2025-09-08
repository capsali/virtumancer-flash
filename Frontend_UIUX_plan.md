# Virtumancer Frontend UI/UX Plan

## 1. Vision & Philosophy

Our goal is to create a virtualization management experience that is **superior to Proxmox and vSphere**. We will achieve this by focusing on three core pillars:

* **Clarity and Intuitiveness**: Eliminate the cluttered, expert-focused interfaces of our competitors. Virtumancer will be immediately understandable to new users while still providing the depth required by experts. We will achieve this through logical information architecture, progressive disclosure, and clear visual language.
* **Performance and Fluidity**: The interface must feel instantaneous. Leveraging a real-time WebSocket backend, every user action—from opening a console to migrating a VM—will provide immediate feedback. No more page reloads for simple state changes.
* **Modern Aesthetics & Ergonomics**: We will use a clean, dark-themed UI with generous spacing, beautiful typography, and purpose-driven animations. The design will be fully responsive, providing a first-class experience on any device, from a desktop monitor to a tablet.

## 2. Overall Layout

The UI will use a proven, three-pane layout optimized for information density and ease of navigation.

* **Left Pane (Collapsible Sidebar)**:
    * **Purpose**: The primary navigation hub for the entire virtual datacenter.
    * **Structure**: A hierarchical tree view:
        * `Datacenter` (Root)
            * `Host 1` (`4/10 VMs running`)
                * `VM 101` (Running)
                * `VM 102` (Stopped)
            * `Host 2` (`8/8 VMs running`)
    * **Interactivity**: The sidebar will be collapsible to maximize content space. It will feature clear visual indicators for resource status (e.g., green for running, red for stopped).

* **Top Pane (Header)**:
    * **Purpose**: Global actions, notifications, and context.
    * **Content**: Sidebar toggle, breadcrumb navigation, a powerful global search bar (future), and a notification center for tasks and alerts.

* **Center Pane (Main Content Area)**:
    * **Purpose**: A dynamic workspace that displays detailed information and management options for the resource selected in the sidebar.
    * **Structure**: The content area will be tab-based to cleanly organize vast amounts of information without overwhelming the user.

## 3. Phased Implementation Plan

### Phase 1: Foundation & Core Management (View-Only & Basic Actions)

* **Objective**: Solidify the layout and provide a best-in-class experience for *viewing and managing existing resources*. This phase is about building a rock-solid foundation.

* **Tasks**:
    1.  **Refine Sidebar Tree**:
        * Implement the full tree structure: `Datacenter` -> `Host` -> `VMs`.
        * Add running/total VM count next to each host.
        * Ensure clear highlighting of the selected resource.
    2.  **Build Datacenter Dashboard**:
        * Display a grid of "Host Cards."
        * Each card will show:
            * Host Name & Connection Status.
            * **Live** sparkline graphs for CPU and Memory usage.
            * Storage pool usage summary (e.g., `SSD_POOL: 1.2TB / 2TB used`).
            * VM counts.
    3.  **Enhance Host Dashboard**:
        * A "Summary" tab with detailed, real-time graphs for host-wide CPU, Memory, Network, and Disk I/O.
        * A filterable, sortable table of all VMs on that host.
    4.  **Complete VM View**:
        * **Summary Tab**:
            * Display key metadata (OS type, description).
            * Show **live performance graphs** for CPU, Memory, Network I/O, and Disk I/O.
        * **Console Tab**: Embed the VNC/SPICE console.
        * **Hardware Tab**: A detailed, read-only view of all virtual hardware (disks, NICs, memory, CPUs).
        * **Snapshots Tab**: A read-only list of existing snapshots.
        * **Lifecycle Buttons**: Ensure all action buttons (Start, Stop, Reboot) are context-aware and provide visual feedback (loading/disabled states).

### Phase 2: Resource Creation & Editing (The "Write" Phase)

* **Objective**: Enable users to create and modify resources directly from the UI, moving beyond view-only functionality.

* **Tasks**:
    1.  **Create VM Wizard**: A beautiful, intuitive multi-step modal for new VM creation.
    2.  **Edit VM Hardware**: An interface within the VM's "Hardware" tab to add, remove, or modify virtual disks, network interfaces, etc., with live validation.
    3.  **Storage Management**: A dedicated view for managing storage pools (creating LVs, uploading ISOs).
    4.  **Network Management**: A dedicated view for managing virtual networks and bridges.

### Phase 3: Advanced Features & Polish

* **Objective**: Introduce power-user features that set Virtumancer apart.

* **Tasks**:
    1.  **Full Snapshot Management**: Create, delete, and revert to snapshots.
    2.  **User Roles & Permissions**: Integrate a robust authentication system.
    3.  **High Availability (HA) Management**: A simplified UI for managing HA groups and policies.
    4.  **Backup & Restore**: A dedicated interface for scheduling and running backups.
