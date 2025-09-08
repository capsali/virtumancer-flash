<script setup>
import { useMainStore } from '@/stores/mainStore';
import { computed, ref, watch, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import VncConsole from '@/components/consoles/VncConsole.vue';
import SpiceConsole from '@/components/consoles/SpiceConsole.vue';

const mainStore = useMainStore();
const route = useRoute();
const activeTab = ref('summary');

let pollInterval = null;

const vm = computed(() => {
    if (!route.params.vmName) return null;
    for (const host of mainStore.hosts) {
        const foundVm = (host.vms || []).find(v => v.name === route.params.vmName);
        if (foundVm) return foundVm;
    }
    return null;
});

const host = computed(() => {
    if (!vm.value) return null;
    return mainStore.hosts.find(h => h.vms && h.vms.some(v => v.name === vm.value.name));
});

const stats = computed(() => mainStore.activeVmStats);
const hardware = computed(() => mainStore.activeVmHardware);

// --- Real-time Stat Calculation ---
const lastCpuTime = ref(0);
const lastCpuTimeTimestamp = ref(0);
const cpuUsagePercent = ref(0);
const lastIoStats = ref(null);
const lastIoStatsTimestamp = ref(0);
const diskRates = ref({});
const netRates = ref({});

watch(stats, (newStats) => {
    if (!newStats || newStats.state !== 1) {
        cpuUsagePercent.value = 0;
        diskRates.value = {};
        netRates.value = {};
        return;
    }
    
    const now = Date.now();

    // CPU Usage
    if (lastCpuTime.value > 0 && lastCpuTimeTimestamp.value > 0) {
        const timeDelta = now - lastCpuTimeTimestamp.value; // ms
        const cpuTimeDelta = newStats.cpu_time - lastCpuTime.value; // ns
        if (timeDelta > 0) {
            const timeDeltaNs = timeDelta * 1_000_000;
            const usage = (cpuTimeDelta / (timeDeltaNs * newStats.vcpu)) * 100;
            cpuUsagePercent.value = Math.min(Math.max(usage, 0), 100);
        }
    }
    lastCpuTime.value = newStats.cpu_time;
    lastCpuTimeTimestamp.value = now;

    // I/O Usage
    if (lastIoStats.value && lastIoStatsTimestamp.value > 0) {
        const timeDeltaSeconds = (now - lastIoStatsTimestamp.value) / 1000;
        if (timeDeltaSeconds > 0) {
            // Disk Rates
            const newDiskRates = {};
            (newStats.disk_stats || []).forEach(current => {
                const previous = (lastIoStats.value.disk_stats || []).find(p => p.device === current.device);
                if (previous) {
                    newDiskRates[current.device] = {
                        read: (current.read_bytes - previous.read_bytes) / timeDeltaSeconds,
                        write: (current.write_bytes - previous.write_bytes) / timeDeltaSeconds,
                    };
                }
            });
            diskRates.value = newDiskRates;

            // Network Rates
            const newNetRates = {};
             (newStats.net_stats || []).forEach(current => {
                const previous = (lastIoStats.value.net_stats || []).find(p => p.device === current.device);
                if (previous) {
                    newNetRates[current.device] = {
                        read: (current.read_bytes - previous.read_bytes) / timeDeltaSeconds,
                        write: (current.write_bytes - previous.write_bytes) / timeDeltaSeconds,
                    };
                }
            });
            netRates.value = newNetRates;
        }
    }
    lastIoStats.value = JSON.parse(JSON.stringify(newStats)); // Deep copy for next calculation
    lastIoStatsTimestamp.value = now;
});


const memoryUsagePercent = computed(() => {
    if (!stats.value || !stats.value.max_mem || stats.value.state !== 1) return 0;
    return (stats.value.memory / stats.value.max_mem) * 100;
});


// --- Helper functions ---
const stateText = (state) => {
    const states = { 0: 'No State', 1: 'Running', 2: 'Blocked', 3: 'Paused', 4: 'Shutdown', 5: 'Shutoff', 6: 'Crashed', 7: 'PMSuspended' };
    return states[state] || 'Unknown';
};

const stateColor = (state) => {
  const colors = { 1: 'text-green-400 bg-green-900/50', 3: 'text-yellow-400 bg-yellow-900/50', 5: 'text-red-400 bg-red-900/50' };
  return colors[state] || 'text-gray-400 bg-gray-700';
};

const formatMemory = (kb) => {
    if (!kb || kb === 0) return '0 MB';
    const mb = kb / 1024;
    if (mb < 1024) return `${mb.toFixed(0)} MB`;
    const gb = mb / 1024;
    return `${gb.toFixed(2)} GB`;
};

const formatUptime = (seconds) => {
    if (seconds <= 0) return 'N/A';
    let d = Math.floor(seconds / (3600*24));
    let h = Math.floor(seconds % (3600*24) / 3600);
    let m = Math.floor(seconds % 3600 / 60);
    let s = Math.floor(seconds % 60);
    
    let dDisplay = d > 0 ? d + (d == 1 ? " day, " : " days, ") : "";
    let hDisplay = h > 0 ? h + (h == 1 ? " hr, " : " hrs, ") : "";
    let mDisplay = m > 0 ? m + (m == 1 ? " min, " : " mins, ") : "";
    let sDisplay = s + (s == 1 ? " sec" : " secs");
    if (d > 0) return dDisplay + hDisplay + mDisplay;
    if (h > 0) return hDisplay + mDisplay;
    if (m > 0) return mDisplay + sDisplay;
    return sDisplay;
}

const formatBps = (bytes) => {
    if (!bytes || bytes < 0) bytes = 0;
    if (bytes === 0) return '0 B/s';
    const k = 1024;
    const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s', 'TB/s'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

// --- Lifecycle & Data Fetching ---
watch(activeTab, (newTab) => {
    if (newTab === 'hardware' && vm.value && host.value) {
        mainStore.fetchVmHardware(host.value.id, vm.value.name);
    }
});

watch(vm, (newVm) => {
    clearInterval(pollInterval);
    pollInterval = null;

    activeTab.value = 'summary';
    lastCpuTime.value = 0;
    lastCpuTimeTimestamp.value = 0;
    cpuUsagePercent.value = 0;
    lastIoStats.value = null;
    lastIoStatsTimestamp.value = 0;
    diskRates.value = {};
    netRates.value = {};
    mainStore.activeVmStats = null;
    mainStore.activeVmHardware = null;

    if (newVm && host.value) {
        mainStore.fetchVmStats(host.value.id, newVm.name);
        pollInterval = setInterval(() => {
            mainStore.fetchVmStats(host.value.id, newVm.name);
        }, 2000);

        if (activeTab.value === 'hardware') {
             mainStore.fetchVmHardware(host.value.id, newVm.name);
        }
    }
}, { immediate: true });

onUnmounted(() => {
    clearInterval(pollInterval);
    mainStore.activeVmStats = null;
    mainStore.activeVmHardware = null;
});

</script>

<template>
  <div v-if="vm && host" class="flex flex-col h-full">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-4">
        <h1 class="text-3xl font-bold text-white">{{ vm.name }}</h1>
        <span 
          class="text-sm font-semibold px-3 py-1 rounded-full"
          :class="stateColor(vm.state)"
        >
          {{ stateText(vm.state) }}
        </span>
      </div>
      <div class="flex items-center space-x-2">
         <button v-if="vm.state === 5" @click="mainStore.startVm(host.id, vm.name)" class="px-4 py-2 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md transition-colors">Start</button>
         <template v-if="vm.state === 1">
            <button @click="mainStore.gracefulShutdownVm(host.id, vm.name)" class="px-4 py-2 text-sm font-medium text-white bg-yellow-600 hover:bg-yellow-700 rounded-md transition-colors">Shutdown</button>
            <button @click="mainStore.gracefulRebootVm(host.id, vm.name)" class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors">Reboot</button>
            <button @click="mainStore.forceOffVm(host.id, vm.name)" class="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors">Force Off</button>
         </template>
      </div>
    </div>
    
    <!-- Tab Navigation -->
    <div class="border-b border-gray-700">
      <nav class="-mb-px flex space-x-8" aria-label="Tabs">
        <button @click="activeTab = 'summary'" :class="[activeTab === 'summary' ? 'border-indigo-500 text-indigo-400' : 'border-transparent text-gray-400 hover:text-gray-200 hover:border-gray-500', 'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm']">Summary</button>
        <button @click="activeTab = 'console'" :class="[activeTab === 'console' ? 'border-indigo-500 text-indigo-400' : 'border-transparent text-gray-400 hover:text-gray-200 hover:border-gray-500', 'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm']">Console</button>
        <button @click="activeTab = 'hardware'" :class="[activeTab === 'hardware' ? 'border-indigo-500 text-indigo-400' : 'border-transparent text-gray-400 hover:text-gray-200 hover:border-gray-500', 'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm']">Hardware</button>
        <button @click="activeTab = 'snapshots'" :class="[activeTab === 'snapshots' ? 'border-indigo-500 text-indigo-400' : 'border-transparent text-gray-400 hover:text-gray-200 hover:border-gray-500', 'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm']">Snapshots</button>
      </nav>
    </div>

    <!-- Tab Content -->
    <div class="flex-grow pt-6 overflow-y-auto">
      <!-- Summary Tab -->
      <div v-if="activeTab === 'summary'" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <div class="bg-gray-900 p-6 rounded-lg shadow-lg">
            <h3 class="text-xl font-semibold mb-4 text-white">Core Performance</h3>
            <div class="space-y-6">
                <div>
                    <div class="flex justify-between items-baseline">
                        <label class="text-sm font-medium text-gray-400">CPU Usage</label>
                        <span class="text-sm font-semibold text-white">{{ cpuUsagePercent.toFixed(2) }}%</span>
                    </div>
                    <div class="w-full bg-gray-700 rounded-full h-2.5 mt-2">
                        <div class="bg-indigo-500 h-2.5 rounded-full" :style="{ width: cpuUsagePercent + '%' }"></div>
                    </div>
                </div>
                <div>
                     <div class="flex justify-between items-baseline">
                        <label class="text-sm font-medium text-gray-400">Memory Usage</label>
                         <span class="text-sm font-semibold text-white">{{ formatMemory(stats?.memory) }} / {{ formatMemory(stats?.max_mem) }}</span>
                    </div>
                    <div class="w-full bg-gray-700 rounded-full h-2.5 mt-2">
                        <div class="bg-teal-500 h-2.5 rounded-full" :style="{ width: memoryUsagePercent + '%' }"></div>
                    </div>
                </div>
            </div>
        </div>
        <div class="bg-gray-900 p-6 rounded-lg shadow-lg">
            <h3 class="text-xl font-semibold mb-4 text-white">I/O Performance</h3>
            <div class="space-y-4">
                <div>
                    <h4 class="text-sm font-medium text-gray-400 mb-2">Disks</h4>
                    <div v-if="stats?.disk_stats?.length" class="space-y-2">
                        <div v-for="disk in stats.disk_stats" :key="disk.device">
                            <p class="text-xs font-mono text-gray-300">{{ disk.device }}</p>
                            <div class="text-sm flex justify-between">
                                <span>Read: {{ formatBps(diskRates[disk.device]?.read) }}</span>
                                <span>Write: {{ formatBps(diskRates[disk.device]?.write) }}</span>
                            </div>
                        </div>
                    </div>
                    <p v-else class="text-sm text-gray-500">No disk devices found.</p>
                </div>
                 <div>
                    <h4 class="text-sm font-medium text-gray-400 mb-2">Network</h4>
                     <div v-if="stats?.net_stats?.length" class="space-y-2">
                        <div v-for="net in stats.net_stats" :key="net.device">
                            <p class="text-xs font-mono text-gray-300">{{ net.device }}</p>
                            <div class="text-sm flex justify-between">
                                <span>Rx: {{ formatBps(netRates[net.device]?.read) }}</span>
                                <span>Tx: {{ formatBps(netRates[net.device]?.write) }}</span>
                            </div>
                        </div>
                    </div>
                    <p v-else class="text-sm text-gray-500">No network interfaces found.</p>
                </div>
            </div>
        </div>
        <div class="bg-gray-900 p-6 rounded-lg shadow-lg">
          <h3 class="text-xl font-semibold mb-4 text-white">Details</h3>
          <dl class="space-y-4">
            <div> <dt class="text-sm font-medium text-gray-400">Host</dt> <dd class="mt-1 text-lg text-gray-200">{{ host.id }}</dd> </div>
            <div> <dt class="text-sm font-medium text-gray-400">Uptime</dt> <dd class="mt-1 text-lg text-gray-200">{{ formatUptime(vm.uptime) }}</dd> </div>
             <div> <dt class="text-sm font-medium text-gray-400">vCPUs</dt> <dd class="mt-1 text-lg text-gray-200">{{ vm.vcpu }}</dd> </div>
            <div> <dt class="text-sm font-medium text-gray-400">Memory</dt> <dd class="mt-1 text-lg text-gray-200">{{ formatMemory(vm.max_mem) }}</dd> </div>
          </dl>
        </div>
      </div>

      <!-- Console Tab -->
      <div v-if="activeTab === 'console'" class="h-full w-full">
         <div v-if="vm.state !== 1" class="flex items-center justify-center h-full text-gray-500 bg-gray-900 rounded-lg">
            <p>Console is only available when the VM is running.</p>
         </div>
         <div v-else class="h-full w-full bg-black rounded-lg overflow-hidden">
            <VncConsole v-if="vm.graphics.vnc" :host-id="host.id" :vm-name="vm.name" />
            <SpiceConsole v-else-if="vm.graphics.spice" :host-id="host.id" :vm-name="vm.name" />
            <div v-else class="flex items-center justify-center h-full text-gray-500">
                <p>No supported console type (VNC or SPICE) is configured for this VM.</p>
            </div>
         </div>
      </div>
      
      <!-- Hardware Tab -->
       <div v-if="activeTab === 'hardware'" class="space-y-8">
            <div v-if="mainStore.isLoading.vmHardware" class="flex items-center justify-center h-48 text-gray-400">
                <svg class="animate-spin mr-3 h-8 w-8 text-indigo-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <span>Loading Hardware...</span>
            </div>
            <div v-else-if="hardware">
                <!-- Storage Devices -->
                <div class="bg-gray-900 rounded-lg shadow-lg">
                    <h3 class="text-xl font-semibold text-white p-4">Storage</h3>
                    <div class="overflow-x-auto">
                        <table class="min-w-full divide-y divide-gray-700">
                            <thead class="bg-gray-800">
                                <tr>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Device</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Bus</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Source</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Format</th>
                                </tr>
                            </thead>
                            <tbody class="bg-gray-900 divide-y divide-gray-800">
                                <tr v-for="disk in hardware.disks" :key="disk.target.dev">
                                    <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-white">{{ disk.target.dev }}</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">{{ disk.target.bus }}</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300 font-mono break-all">{{ disk.path }}</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">{{ disk.driver.type }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Network Adapters -->
                <div class="bg-gray-900 rounded-lg shadow-lg">
                    <h3 class="text-xl font-semibold text-white p-4">Network Adapters</h3>
                     <div class="overflow-x-auto">
                        <table class="min-w-full divide-y divide-gray-700">
                            <thead class="bg-gray-800">
                                <tr>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Device</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">MAC Address</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Source Bridge</th>
                                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Model</th>
                                </tr>
                            </thead>
                            <tbody class="bg-gray-900 divide-y divide-gray-800">
                                <tr v-for="net in hardware.networks" :key="net.mac.address">
                                    <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-white">{{ net.target.dev }}</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300 font-mono">{{ net.mac.address }}</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">{{ net.source.bridge }}</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">{{ net.model.type }}</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
             <div v-else class="flex items-center justify-center h-48 text-gray-500 bg-gray-900 rounded-lg">
                <p>Could not load hardware information.</p>
            </div>
       </div>
       
       <!-- Snapshots Tab Placeholder -->
        <div v-if="activeTab === 'snapshots'" class="flex items-center justify-center h-full text-gray-500 bg-gray-900 rounded-lg">
            <p>Snapshot management will be implemented here.</p>
       </div>

    </div>
  </div>

  <div v-else class="flex items-center justify-center h-full text-gray-500">
    <p>Select a VM from the sidebar to view details, or the VM is still loading.</p>
  </div>
</template>

