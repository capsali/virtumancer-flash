import { defineStore } from 'pinia';
import { ref } from 'vue';

export const useUiStore = defineStore('ui', () => {
    const isSidebarOpen = ref(true);
    const isAddHostModalOpen = ref(false);

    function toggleSidebar() {
        isSidebarOpen.value = !isSidebarOpen.value;
    }

    function openAddHostModal() {
        isAddHostModalOpen.value = true;
    }

    function closeAddHostModal() {
        isAddHostModalOpen.value = false;
    }

    return {
        isSidebarOpen,
        isAddHostModalOpen,
        toggleSidebar,
        openAddHostModal,
        closeAddHostModal,
    };
});


