<script setup>
import { useMainStore } from '@/stores/mainStore';
const store = useMainStore();
</script>

<template>
  <div class="bg-gray-800 p-6 rounded-lg shadow-lg">
    <h2 class="text-2xl font-semibold mb-4 border-b border-gray-700 pb-3">Managed Hosts</h2>
    <div v-if="store.isLoading.hosts" class="text-center text-gray-400">Loading hosts...</div>
    <div v-else-if="store.hosts.length === 0" class="text-gray-400">
      No hosts configured yet. Add one to get started.
    </div>
    <ul v-else class="space-y-3">
      <li v-for="host in store.hosts" 
          :key="host.id" 
          @click="store.selectHost(host.id)"
          class="flex items-center justify-between p-4 rounded-md transition-all duration-200 cursor-pointer"
          :class="store.selectedHostId === host.id ? 'bg-indigo-600 shadow-lg' : 'bg-gray-700 hover:bg-gray-600'"
      >
        <div>
          <p class="font-semibold text-white">{{ host.id }}</p>
          <p class="text-sm font-mono" :class="store.selectedHostId === host.id ? 'text-indigo-200' : 'text-gray-400'">{{ host.uri }}</p>
        </div>
        <button 
          @click.stop="store.deleteHost(host.id)"
          class="text-gray-400 hover:text-red-400 focus:outline-none focus:text-red-400 transition-colors"
          aria-label="Delete host"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </button>
      </li>
    </ul>
  </div>
</template>

