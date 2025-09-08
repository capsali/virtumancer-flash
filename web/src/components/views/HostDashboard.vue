<script setup>
import { useMainStore } from '@/stores/mainStore';
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';

const mainStore = useMainStore();
const route = useRoute();
const router = useRouter();

const selectedHost = computed(() => {
  const hostId = route.params.hostId;
  if (!hostId) return null;
  return mainStore.hosts.find(h => h.id === hostId);
});

const vms = computed(() => {
    return selectedHost.value?.vms || [];
});

const totalMemory = computed(() => {
    return selectedHost.value?.info?.memory * 1024 || 0;
});

const usedMemory = computed(() => {
    return vms.value.reduce((total, vm) => total + (vm.state === 1 ? vm.memory * 1024 : 0), 0);
});

const memoryUsagePercent = computed(() => {
    if (!totalMemory.value) return 0;
    return (usedMemory.value / totalMemory.value) * 100;
});

const totalCpu = computed(() => {
    return selectedHost.value?.info?.cpu || 0;
});

const usedCpu = computed(() => {
    return vms.value.reduce((total, vm) => total + (vm.state === 1 ? vm.vcpu : 0), 0);
});

const cpuUsagePercent = computed(() => {
    if (!totalCpu.value) return 0;
    return (usedCpu.value / totalCpu.value) * 100;
});


const selectVm = (vmName) => {
    router.push({ name: 'vm-view', params: { vmName } });
}

// Helper functions for display
const stateText = (state) => {
    const states = {
        0: 'No State', 1: 'Running', 2: 'Blocked', 3: 'Paused',
        4: 'Shutdown', 5: 'Shutoff', 6: 'Crashed', 7: 'PMSuspended',
    };
    return states[state] || 'Unknown';
};

const stateColor = (state) => {
  const colors = {
    1: 'text-green-400 bg-green-900/50', // Running
    3: 'text-yellow-400 bg-yellow-900/50', // Paused
    5: 'text-red-400 bg-red-900/50', // Shutoff
  };
  return colors[state] || 'text-gray-400 bg-gray-700';
};

const formatMemory = (kb) => {
    if (kb === 0) return '0 MB';
    const mb = kb / 1024;
    if (mb < 1024) return `${mb.toFixed(0)} MB`;
    const gb = mb / 1024;
    return `${gb.toFixed(2)} GB`;
};

const formatBytes = (bytes, decimals = 2) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
};

</script>

<template>
  <div v-if="selectedHost">
    <!-- Header -->
    <div class="mb-6">
      <h1 class="text-3xl font-bold text-white">Host: {{ selectedHost.id }}</h1>
      <p class="text-gray-400 font-mono mt-1">{{ selectedHost.uri }}</p>
    </div>
    
    <!-- Summary Section -->
    <div class="mb-6">
        <h2 class="text-xl font-semibold text-white mb-4">Summary</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <!-- CPU Usage -->
            <div class="bg-gray-800 p-4 rounded-lg">
                <h3 class="text-sm font-medium text-gray-400">CPU Usage</h3>
                <p class="text-2xl font-semibold text-white mt-1">{{ usedCpu }} / {{ totalCpu }} Cores</p>
                <div class="w-full bg-gray-700 rounded-full h-2.5 mt-2">
                    <div class="bg-indigo-500 h-2.5 rounded-full" :style="{ width: cpuUsagePercent + '%' }"></div>
                </div>
            </div>
            <!-- Memory Usage -->
            <div class="bg-gray-800 p-4 rounded-lg">
                <h3 class="text-sm font-medium text-gray-400">Memory Usage</h3>
                <p class="text-2xl font-semibold text-white mt-1">{{ formatBytes(usedMemory) }} / {{ formatBytes(totalMemory) }}</p>
                <div class="w-full bg-gray-700 rounded-full h-2.5 mt-2">
                    <div class="bg-teal-500 h-2.5 rounded-full" :style="{ width: memoryUsagePercent + '%' }"></div>
                </div>
            </div>
            <!-- Other Host Info -->
             <div class="bg-gray-800 p-4 rounded-lg">
                <h3 class="text-sm font-medium text-gray-400">Hostname</h3>
                <p class="text-2xl font-semibold text-white mt-1 truncate">{{ selectedHost.info?.hostname || 'Loading...' }}</p>
            </div>
             <div class="bg-gray-800 p-4 rounded-lg">
                <h3 class="text-sm font-medium text-gray-400">Cores / Threads</h3>
                <p class="text-2xl font-semibold text-white mt-1">{{ selectedHost.info?.cores || 'N/A' }} / {{ selectedHost.info?.threads || 'N/A' }}</p>
            </div>
        </div>
    </div>

    <!-- VM List Section -->
    <div class="bg-gray-900 rounded-lg">
      <h2 class="text-xl font-semibold text-white p-4">Virtual Machines</h2>
      
      <div v-if="mainStore.isLoading.vms && vms.length === 0" class="flex items-center justify-center h-48 text-gray-400">
        <svg class="animate-spin mr-3 h-8 w-8 text-indigo-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <span>Loading VMs...</span>
      </div>

      <div v-else-if="vms.length === 0" class="flex items-center justify-center h-48 text-gray-500 bg-gray-800/50 rounded-lg m-4">
        <p>No Virtual Machines found on this host.</p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-700">
          <thead class="bg-gray-800">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Name</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">State</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">vCPUs</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">Memory</th>
            </tr>
          </thead>
          <tbody class="bg-gray-900 divide-y divide-gray-800">
            <tr v-for="vm in vms" :key="vm.name" @click="selectVm(vm.name)" class="hover:bg-gray-800 cursor-pointer transition-colors duration-150">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                  <div class="h-2.5 w-2.5 rounded-full mr-3 flex-shrink-0" :class="{
                    'bg-green-500': vm.state === 1, 'bg-red-500': vm.state === 5,
                    'bg-yellow-500': vm.state === 3, 'bg-gray-500': ![1,3,5].includes(vm.state)
                  }"></div>
                  <div class="text-sm font-medium text-white">{{ vm.name }}</div>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full" :class="stateColor(vm.state)">
                  {{ stateText(vm.state) }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">{{ vm.vcpu }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-300">{{ formatMemory(vm.max_mem) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
  <div v-else class="flex items-center justify-center h-full text-gray-500">
    <p>Select a host from the sidebar to view details.</p>
  </div>
</template>

