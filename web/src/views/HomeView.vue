<script setup>
import AddHostForm from '@/components/AddHostForm.vue';
import HostList from '@/components/HostList.vue';
import VmList from '@/components/VmList.vue';
import { onMounted } from 'vue';
import { useMainStore } from '@/stores/mainStore';

const store = useMainStore();

// When the main view is mounted, fetch the initial host list and start real-time updates.
onMounted(() => {
  store.fetchHosts();
  store.initializeRealtime();
});
</script>

<template>
  <div class="bg-gray-900 text-gray-100 min-h-screen font-sans">
    <div class="container mx-auto p-4 md:p-8">

      <header class="mb-10">
        <h1 class="text-4xl md:text-5xl font-bold text-white tracking-wider">Virtu<span class="text-indigo-400">Mancer</span></h1>
        <p class="text-gray-400 mt-2">The Modern Web UI for Libvirt</p>
      </header>

      <main class="grid grid-cols-1 lg:grid-cols-3 gap-8">
        
        <div class="lg:col-span-1 flex flex-col gap-8">
          <AddHostForm />
          <HostList />
        </div>

        <div class="lg:col-span-2">
          <VmList />
        </div>
      </main>

    </div>
  </div>
</template>


