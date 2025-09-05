<!-- VIRTUMANCER: Force Reload v1.2 -->
<script setup>
import { useMainStore } from '@/stores/mainStore';
import { ref, onMounted, onUnmounted } from 'vue';

const store = useMainStore();
const openDropdown = ref(null); // Tracks which VM's dropdown is open
const dropdownContainerRefs = ref({}); // Holds refs to the dropdown containers

const toggleDropdown = (vmName) => {
  if (openDropdown.value === vmName) {
    openDropdown.value = null;
  } else {
    openDropdown.value = vmName;
  }
};

const handleClickOutside = (event) => {
  // If no dropdown is open, do nothing
  if (!openDropdown.value) return;

  const container = dropdownContainerRefs.value[openDropdown.value];
  
  // If the click is outside the open dropdown's container, close it
  if (container && !container.contains(event.target)) {
    openDropdown.value = null;
  }
};

onMounted(() => {
  window.addEventListener('click', handleClickOutside);
});

onUnmounted(() => {
  window.removeEventListener('click', handleClickOutside);
});


const stateColor = (state) => {
  const colors = {
    1: 'text-green-400', // Running
    3: 'text-yellow-400', // Paused
    5: 'text-red-400', // Shutoff
  };
  return colors[state] || 'text-gray-400';
};

const stateText = (state) => {
    const states = {
        0: 'No State', 1: 'Running', 2: 'Blocked', 3: 'Paused',
        4: 'Shutdown', 5: 'Shutoff', 6: 'Crashed', 7: 'PMSuspended',
    };
    return states[state] || 'Unknown';
}

const formatMemory = (kb) => {
    if (kb === 0) return '0 MB';
    const mb = kb / 1024;
    if (mb < 1024) return `${mb.toFixed(0)} MB`;
    const gb = mb / 1024;
    return `${gb.toFixed(2)} GB`;
}
</script>

<template>
  <div class="bg-gray-800 p-6 rounded-lg shadow-lg h-full">
    <h2 class="text-2xl font-semibold mb-4 border-b border-gray-700 pb-3">Virtual Machines on <span class="text-indigo-400">{{ store.selectedHostId || '...' }}</span></h2>
    
    <div v-if="!store.selectedHostId" class="flex items-center justify-center h-full text-gray-500">
      <p>Select a host to view its VMs.</p>
    </div>
    
    <div v-else-if="store.isLoading.vms && store.vms.length === 0" class="flex items-center justify-center h-full text-gray-400">
      <svg class="animate-spin mr-3 h-8 w-8 text-indigo-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
      </svg>
      <span>Loading VMs for {{ store.selectedHostId }}...</span>
    </div>

    <div v-else-if="store.vms.length === 0" class="flex items-center justify-center h-full text-gray-500">
      <p>No Virtual Machines found on this host.</p>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-for="vm in store.vms" :key="vm.name" 
           class="bg-gray-700 rounded-lg flex flex-col justify-between shadow-md hover:shadow-xl hover:bg-gray-600/50 transition-all duration-200 relative">
        
        <div v-if="store.isLoading.vmAction && store.isLoading.vmAction.startsWith(vm.name)" class="absolute inset-0 bg-gray-900/70 flex items-center justify-center z-20">
          <svg class="animate-spin h-8 w-8 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>

        <div class="p-4 flex-grow">
          <div class="flex items-center justify-between">
            <h3 class="font-bold text-lg truncate text-white" :title="vm.name">{{ vm.name }}</h3>
            <div class="flex items-center space-x-2">
                <span class="text-xs font-semibold px-2 py-1 rounded-full" :class="stateColor(vm.state)">●</span>
                <span class="text-sm font-medium" :class="stateColor(vm.state)">{{ stateText(vm.state) }}</span>
            </div>
          </div>
          <div class="mt-4 space-y-2 text-sm text-gray-300">
              <div class="flex items-center">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M12 6V4m0 16v-2M8 12a4 4 0 118 0 4 4 0 01-8 0z" /></svg>
                  <span>{{ vm.vcpu }} vCPU</span>
              </div>
              <div class="flex items-center">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-2 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5v4m0 0h-4m4 0l-5-5" /></svg>
                  <span>{{ formatMemory(vm.max_mem) }} Memory</span>
              </div>
          </div>
        </div>
        
        <div class="bg-gray-700/50 p-2 flex justify-end items-center space-x-2">
            <button v-if="vm.state === 5" @click="store.startVm(store.selectedHostId, vm.name)"
                    class="px-3 py-1 text-xs font-medium text-white bg-green-600 hover:bg-green-700 rounded transition-colors w-full">Start</button>
            
            <div v-if="vm.state === 1" class="relative w-full" :ref="(el) => { if (el) dropdownContainerRefs[vm.name] = el }">
                <button @click.prevent.stop="toggleDropdown(vm.name)" class="w-full px-3 py-1 text-xs font-medium text-white bg-gray-600 hover:bg-gray-500 rounded transition-colors flex items-center justify-center">
                    <span>Actions</span>
                    <svg class="w-4 h-4 ml-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
                </button>
                <div v-if="openDropdown === vm.name" class="absolute top-full right-0 mt-2 w-48 bg-gray-900 border border-gray-700 rounded-md shadow-lg z-10 text-sm">
                    <a @click.prevent="store.gracefulShutdownVm(store.selectedHostId, vm.name); openDropdown = null" class="block px-4 py-2 text-gray-300 hover:bg-orange-600 hover:text-white cursor-pointer">Graceful Shutdown</a>
                    <a @click.prevent="store.gracefulRebootVm(store.selectedHostId, vm.name); openDropdown = null" class="block px-4 py-2 text-gray-300 hover:bg-blue-600 hover:text-white cursor-pointer">Graceful Reboot</a>
                    <div class="border-t border-gray-700 my-1"></div>
                    <a @click.prevent="store.forceOffVm(store.selectedHostId, vm.name); openDropdown = null" class="block px-4 py-2 text-red-400 hover:bg-red-600 hover:text-white cursor-pointer">Force Off</a>
                    <a @click.prevent="store.forceResetVm(store.selectedHostId, vm.name); openDropdown = null" class="block px-4 py-2 text-red-400 hover:bg-red-600 hover:text-white cursor-pointer">Force Reset</a>
                </div>
            </div>
        </div>
      </div>
    </div>
  </div>
</template>


