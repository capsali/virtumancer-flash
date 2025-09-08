import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const useMainStore = defineStore('main', () => {
    // State
    const hosts = ref([]);
    const selectedHostId = ref(null);
    const errorMessage = ref('');
    const isLoading = ref({
        hosts: false,
        vms: false,
        addHost: false,
        vmAction: null,
        vmHardware: false,
    });

    const activeVmStats = ref(null);
    const activeVmHardware = ref(null);

    const totalVms = computed(() => {
        return hosts.value.reduce((total, host) => total + (host.vms ? host.vms.length : 0), 0);
    });

    let ws = null;

    // --- WebSocket and Polling Logic ---

    const connectWebSocket = () => {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsURL = `${protocol}//${window.location.host}/ws`;

        ws = new WebSocket(wsURL);
        ws.onopen = () => console.log('WebSocket for UI updates connected');
        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                if (message.type === 'refresh') {
                    console.log('WebSocket received refresh message, refetching hosts.');
                    fetchHosts();
                }
            } catch (e) {
                console.error("Failed to parse websocket message", e);
            }
        };
        ws.onclose = () => {
            console.log('WebSocket disconnected. Reconnecting in 5s...');
            setTimeout(connectWebSocket, 5000);
        };
        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            ws.close();
        };
    };

    const initializeRealtime = () => {
        connectWebSocket();
    };

    // --- Host Actions ---

    const fetchHosts = async () => {
        isLoading.value.hosts = true;
        errorMessage.value = '';
        try {
            const response = await fetch('/api/v1/hosts');
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            const data = await response.json();

            const hostPromises = (data || []).map(async host => {
                host.vms = await fetchVmsForHost(host.id);
                host.info = await fetchHostInfo(host.id);
                return host;
            });

            hosts.value = await Promise.all(hostPromises);

        } catch (error) {
            console.error("Error fetching hosts:", error);
            errorMessage.value = "Failed to fetch hosts.";
        } finally {
            isLoading.value.hosts = false;
        }
    };

    const fetchHostInfo = async (hostId) => {
        if (!hostId) return null;
        try {
            const response = await fetch(`/api/v1/hosts/${hostId}/info`);
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            return await response.json();
        } catch (error) {
            console.error(`Error fetching info for host ${hostId}:`, error);
            return null;
        }
    };

    const addHost = async (hostData) => {
        isLoading.value.addHost = true;
        errorMessage.value = '';
        try {
            const response = await fetch('/api/v1/hosts', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(hostData),
            });
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || `HTTP error! status: ${response.status}`);
            }
            // No need to call fetchHosts here, the websocket will trigger it
        } catch (error) {
            errorMessage.value = `Failed to add host: ${error.message}`;
            console.error(error);
        } finally {
            isLoading.value.addHost = false;
        }
    };

    const deleteHost = async (hostId) => {
        errorMessage.value = '';
        try {
            const response = await fetch(`/api/v1/hosts/${hostId}`, { method: 'DELETE' });
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || `HTTP error! status: ${response.status}`);
            }
            if (selectedHostId.value === hostId) {
                selectedHostId.value = null;
            }
            // No need to call fetchHosts here, the websocket will trigger it
        } catch (error) {
            errorMessage.value = `Failed to delete host: ${error.message}`;
            console.error(error);
        }
    };

    const selectHost = (hostId) => {
        if (selectedHostId.value !== hostId) {
            selectedHostId.value = hostId;
        }
    };

    // --- VM Actions ---
    const fetchVmsForHost = async (hostId) => {
        if (!hostId) return [];
        try {
            const response = await fetch(`/api/v1/hosts/${hostId}/vms`);
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            return await response.json() || [];
        } catch (error) {
            console.error(`Failed to fetch VMs for ${hostId}:`, error);
            return [];
        }
    };

    const fetchVmStats = async (hostId, vmName) => {
        if (!hostId || !vmName) {
            activeVmStats.value = null;
            return;
        }
        try {
            const response = await fetch(`/api/v1/hosts/${hostId}/vms/${vmName}/stats`);
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            activeVmStats.value = await response.json();
        } catch (error) {
            console.error(`Failed to fetch stats for VM ${vmName}:`, error);
            activeVmStats.value = null;
        }
    };

    const fetchVmHardware = async (hostId, vmName) => {
        if (!hostId || !vmName) {
            activeVmHardware.value = null;
            return;
        }
        isLoading.value.vmHardware = true;
        try {
            const response = await fetch(`/api/v1/hosts/${hostId}/vms/${vmName}/hardware`);
            if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
            activeVmHardware.value = await response.json();
        } catch (error) {
            console.error(`Failed to fetch hardware for VM ${vmName}:`, error);
            activeVmHardware.value = null;
        } finally {
            isLoading.value.vmHardware = false;
        }
    };

    const performVmAction = async (hostId, vmName, action) => {
        isLoading.value.vmAction = `${vmName}:${action}`;
        errorMessage.value = '';
        try {
            const response = await fetch(`/api/v1/hosts/${hostId}/vms/${vmName}/${action}`, { method: 'POST' });
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || `HTTP error! status: ${response.status}`);
            }
            // The websocket will handle the UI update
        } catch (error) {
            errorMessage.value = `Action '${action}' on VM '${vmName}' failed: ${error.message}`;
            console.error(error);
        } finally {
            isLoading.value.vmAction = null;
        }
    };

    const startVm = (hostId, vmName) => performVmAction(hostId, vmName, 'start');
    const gracefulShutdownVm = (hostId, vmName) => performVmAction(hostId, vmName, 'shutdown');
    const gracefulRebootVm = (hostId, vmName) => performVmAction(hostId, vmName, 'reboot');
    const forceOffVm = (hostId, vmName) => performVmAction(hostId, vmName, 'forceoff');
    const forceResetVm = (hostId, vmName) => performVmAction(hostId, vmName, 'forcereset');

    return {
        hosts,
        selectedHostId,
        errorMessage,
        isLoading,
        activeVmStats,
        activeVmHardware,
        totalVms,
        initializeRealtime,
        fetchHosts,
        addHost,
        deleteHost,
        selectHost,
        fetchVmStats,
        fetchVmHardware,
        startVm,
        gracefulShutdownVm,
        gracefulRebootVm,
        forceOffVm,
        forceResetVm,
    };
});



