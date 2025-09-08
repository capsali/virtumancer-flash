<script setup>
import { ref } from 'vue';
import { useMainStore } from '@/stores/mainStore';
import { useUiStore } from '@/stores/uiStore';

const mainStore = useMainStore();
const uiStore = useUiStore();

const newHostId = ref('');
const newHostUri = ref('qemu+ssh://root@/system');

const submitForm = async () => {
  await mainStore.addHost({ id: newHostId.value, uri: newHostUri.value });
  if (!mainStore.errorMessage) {
    uiStore.closeAddHostModal();
  }
};
</script>

<template>
  <div class="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center z-50" @click.self="uiStore.closeAddHostModal">
    <div class="bg-gray-800 p-8 rounded-lg shadow-2xl w-full max-w-md">
      <h2 class="text-2xl font-bold mb-6 text-white border-b border-gray-700 pb-4">Add New Host</h2>
      <form @submit.prevent="submitForm" class="space-y-6">
        <div>
          <label for="hostId" class="block text-sm font-medium text-gray-300">Host ID (a short name)</label>
          <input 
            id="hostId"
            v-model="newHostId" 
            type="text" 
            placeholder="e.g., proxmox-1"
            required
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
            required
            class="mt-1 block w-full bg-gray-700 border-gray-600 rounded-md shadow-sm text-white placeholder-gray-400 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
        <p v-if="mainStore.errorMessage" class="text-sm text-red-400 bg-red-900/50 p-3 rounded-md">{{ mainStore.errorMessage }}</p>
        <div class="flex justify-end space-x-4 pt-4">
          <button 
            type="button"
            @click="uiStore.closeAddHostModal"
            class="px-4 py-2 text-sm font-medium text-gray-300 bg-gray-600 hover:bg-gray-500 rounded-md transition-colors"
          >
            Cancel
          </button>
          <button 
            type="submit"
            :disabled="mainStore.isLoading.addHost"
            class="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-md transition-colors disabled:bg-indigo-800 disabled:cursor-not-allowed flex items-center"
          >
            <svg v-if="mainStore.isLoading.addHost" class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ mainStore.isLoading.addHost ? 'Connecting...' : 'Add Host' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

