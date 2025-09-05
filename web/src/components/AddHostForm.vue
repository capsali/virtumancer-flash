<script setup>
import { ref } from 'vue';
import { useMainStore } from '@/stores/mainStore';

const store = useMainStore();
const newHostId = ref('');
const newHostUri = ref('');

const submitForm = () => {
  store.addHost({ id: newHostId.value, uri: newHostUri.value });
  newHostId.value = '';
  newHostUri.value = '';
};
</script>

<template>
  <div class="bg-gray-800 p-6 rounded-lg shadow-lg">
    <h2 class="text-2xl font-semibold mb-4 border-b border-gray-700 pb-3">Add New Host</h2>
    <form @submit.prevent="submitForm" class="space-y-4">
      <div>
        <label for="hostId" class="block text-sm font-medium text-gray-300">Host ID (a short name)</label>
        <input 
          id="hostId"
          v-model="newHostId" 
          type="text" 
          placeholder="e.g., proxmox-1"
          class="mt-1 block w-full bg-gray-700 border-gray-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:ring-indigo-500 focus:border-indigo-500"
        />
      </div>
      <div>
        <label for="hostUri" class="block text-sm font-medium text-gray-300">Connection URI</label>
        <input 
          id="hostUri"
          v-model="newHostUri" 
          type="text" 
          placeholder="qemu+ssh://user@hostname/system"
          class="mt-1 block w-full bg-gray-700 border-gray-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:ring-indigo-500 focus:border-indigo-500"
        />
      </div>
      <button 
        type="submit"
        :disabled="store.isLoading.addHost"
        class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 focus:ring-offset-gray-800 disabled:bg-indigo-800 disabled:cursor-not-allowed"
      >
        <svg v-if="store.isLoading.addHost" class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        {{ store.isLoading.addHost ? 'Connecting...' : 'Add Host' }}
      </button>
    </form>
    <p v-if="store.errorMessage" class="mt-4 text-sm text-red-400 bg-red-900/50 p-3 rounded-md">{{ store.errorMessage }}</p>
  </div>
</template>

