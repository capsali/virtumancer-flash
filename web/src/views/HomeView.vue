<script setup>
import Sidebar from '@/components/layout/Sidebar.vue';
import Header from '@/components/layout/Header.vue';
import MainContent from '@/components/layout/MainContent.vue';
import AddHostModal from '@/components/modals/AddHostModal.vue';
import { useUiStore } from '@/stores/uiStore';

const uiStore = useUiStore();
</script>

<template>
  <div class="flex h-screen bg-gray-800">
    <Sidebar />
    <div class="flex-1 flex flex-col overflow-hidden">
      <Header />
      <MainContent />
    </div>
    <AddHostModal v-if="uiStore.isAddHostModalOpen" />
  </div>
</template>


