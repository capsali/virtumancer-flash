<script setup>
import { useUiStore } from '@/stores/uiStore';
const uiStore = useUiStore();
</script>

<template>
  <header class="flex items-center justify-between h-16 px-6 bg-gray-900 border-b border-gray-800">
    <button @click="uiStore.toggleSidebar" class="p-2 rounded-md hover:bg-gray-700">
      <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16m-7 6h7" />
      </svg>
    </button>
    <div>
      <h2 class="text-lg font-semibold">Dashboard</h2>
    </div>
    <div>
      <!-- Placeholder for user menu or actions -->
      <button class="p-2 rounded-full hover:bg-gray-700">
        <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6 6 0 10-12 0v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" /></svg>
      </button>
    </div>
  </header>
</template>


