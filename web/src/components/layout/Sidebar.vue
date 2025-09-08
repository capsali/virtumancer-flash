<script setup>
import { useUiStore } from '@/stores/uiStore';
import { useMainStore } from '@/stores/mainStore';
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

const uiStore = useUiStore();
const mainStore = useMainStore();
const router = useRouter();

const expandedHosts = ref({});

onMounted(() => {
  mainStore.initializeRealtime();
  mainStore.fetchHosts().then(() => {
    // Automatically expand all hosts on load
    mainStore.hosts.forEach(host => {
      expandedHosts.value[host.id] = true;
    });
  });
});

function selectDatacenter() {
    mainStore.selectHost(null);
    router.push({ name: 'datacenter' });
}

function selectHost(hostId) {
  mainStore.selectHost(hostId);
  router.push({ name: 'host-dashboard', params: { hostId } });
}

function selectVm(vm) {
  for (const host of mainStore.hosts) {
      if (host.vms && host.vms.find(hvm => hvm.name === vm.name)) {
          mainStore.selectHost(host.id);
          break;
      }
  }
  router.push({ name: 'vm-view', params: { vmName: vm.name } });
}

function toggleHost(hostId) {
    expandedHosts.value[hostId] = !expandedHosts.value[hostId];
}

const runningVmsCount = (host) => {
    return host.vms ? host.vms.filter(vm => vm.state === 1).length : 0;
}
</script>

<template>
  <aside 
    class="flex flex-col bg-gray-900 text-gray-300 transition-all duration-300 ease-in-out"
    :class="uiStore.isSidebarOpen ? 'w-72' : 'w-20'"
  >
    <div class="flex items-center h-16 px-6 border-b border-gray-800 flex-shrink-0">
      <h1 class="text-xl font-bold text-white tracking-wider" v-show="uiStore.isSidebarOpen">
        Virtu<span class="text-indigo-400">Mancer</span>
      </h1>
       <svg v-show="!uiStore.isSidebarOpen" class="h-8 w-8 text-indigo-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
      </svg>
    </div>

    <div class="flex-shrink-0 px-4 py-4">
      <button @click="uiStore.openAddHostModal" class="w-full flex items-center justify-center p-2 rounded-md bg-indigo-600 text-white hover:bg-indigo-700 transition-colors">
        <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6" /></svg>
        <span class="ml-2" v-show="uiStore.isSidebarOpen">Add Host</span>
      </button>
    </div>
    
    <nav class="flex-1 overflow-y-auto">
      <ul class="px-4">
        <!-- Datacenter Root -->
        <li class="mb-2">
            <div @click="selectDatacenter" class="flex items-center p-2 rounded-md cursor-pointer hover:bg-gray-700" :class="{ 'bg-gray-700 text-white': !mainStore.selectedHostId }">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" /></svg>
                <span class="ml-3 font-semibold" v-show="uiStore.isSidebarOpen">Datacenter</span>
            </div>
        </li>

        <!-- Hosts -->
        <li v-for="host in mainStore.hosts" :key="host.id" class="mb-1">
          <div 
            class="flex items-center p-2 rounded-md cursor-pointer hover:bg-gray-700"
            :class="{ 'bg-gray-700 text-white': mainStore.selectedHostId === host.id }"
            @click="selectHost(host.id)"
          >
            <button @click.stop="toggleHost(host.id)" v-show="uiStore.isSidebarOpen" class="mr-2 focus:outline-none">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 transition-transform" :class="{'rotate-90': expandedHosts[host.id]}" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" /></svg>
            </button>
            <svg class="h-6 w-6 flex-shrink-0" :class="{'text-indigo-400': mainStore.selectedHostId === host.id}" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"/></svg>
            <span class="ml-3 font-semibold truncate" v-show="uiStore.isSidebarOpen">{{ host.id }}</span>
             <span v-if="uiStore.isSidebarOpen && host.vms" class="ml-auto text-xs font-mono bg-gray-800 px-2 py-0.5 rounded-full">
              {{ runningVmsCount(host) }}/{{ host.vms.length }}
            </span>
          </div>

          <!-- VMs -->
          <ul v-if="uiStore.isSidebarOpen && expandedHosts[host.id] && host.vms && host.vms.length" class="mt-1 space-y-1" :class="uiStore.isSidebarOpen ? 'ml-6 border-l-2 border-gray-700 pl-4' : ''">
            <li v-for="vm in host.vms" :key="vm.name">
              <div @click="selectVm(vm)" class="flex items-center p-1.5 text-sm rounded-md cursor-pointer hover:bg-gray-700" :class="{'bg-gray-700/50': $route.params.vmName === vm.name}">
                <span class="h-2 w-2 rounded-full mr-2 flex-shrink-0" :class="{
                  'bg-green-500': vm.state === 1, 'bg-red-500': vm.state === 5,
                  'bg-yellow-500': vm.state === 3, 'bg-gray-500': ![1,3,5].includes(vm.state)
                }"></span>
                <span class="truncate">{{ vm.name }}</span>
              </div>
            </li>
          </ul>
        </li>
      </ul>
    </nav>
    <div class="flex-shrink-0 p-4 border-t border-gray-800">
        <div v-show="uiStore.isSidebarOpen">
            <h3 class="text-xs font-semibold text-gray-400 uppercase tracking-wider">Status</h3>
            <div class="mt-2 text-sm text-gray-300">
                <span>{{ mainStore.hosts.length }} Host(s)</span>
                <span class="mx-2">|</span>
                <span>{{ mainStore.totalVms }} VM(s)</span>
            </div>
        </div>
        <div v-show="!uiStore.isSidebarOpen" class="text-center">
             <div class="text-xl font-bold">{{ mainStore.hosts.length }}</div>
             <div class="text-xs text-gray-400">Hosts</div>
        </div>
    </div>
  </aside>
</template>


