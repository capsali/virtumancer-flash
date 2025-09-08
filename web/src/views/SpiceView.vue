<script setup>
import { ref, computed } from 'vue';
import { useRoute } from 'vue-router';

const route = useRoute();
const connectionStatus = ref('Loading...'); // Initial status

// Extract route parameters
const hostId = computed(() => route.params.hostId);
const vmName = computed(() => route.params.vmName);

// Dynamically construct the source URL for the iframe
const spiceIframeSrc = computed(() => {
  if (!hostId.value || !vmName.value) {
    return '';
  }

  // Use `hostname` to avoid including the port in the host parameter.
  const host = window.location.hostname;
  // The port is needed separately by the spice-html5 client.
  const port = window.location.port || (window.location.protocol === 'https:' ? '443' : '80');
  
  // The backend proxy path for the SPICE connection.
  const path = `api/v1/hosts/${hostId.value}/vms/${vmName.value}/spice`;

  // Assemble the query parameters for spice_auto.html
  const params = new URLSearchParams({
    host: host,
    port: port,
    password: '', // Assuming no password for now
    path: path,
    encrypt: window.location.protocol === 'https:' ? '1' : '0' // Use encryption for HTTPS connections
  });

  return `/spice/spice_auto.html?${params.toString()}`;
});

// Update status when the iframe has loaded the page.
// Note: We can't directly know if the WebSocket inside connected due to cross-origin policies,
// but we can infer a basic "loaded" state. A more advanced implementation could use postMessage.
const onIframeLoad = () => {
  connectionStatus.value = 'Client Loaded'; // We can't see inside the iframe, but the page itself is ready.
};

</script>

<template>
  <div class="bg-black w-screen h-screen flex flex-col text-white font-sans">
    <header class="bg-gray-800 p-2 flex items-center justify-between shadow-md z-10 flex-shrink-0">
      <div class="flex items-center">
        <router-link to="/" class="text-indigo-400 hover:text-indigo-300 mr-4">
          &larr; Back
        </router-link>
        <div>
          <h1 class="font-bold text-lg">SPICE Console: {{ vmName }}</h1>
          <p class="text-xs text-gray-400">Host: {{ hostId }}</p>
        </div>
      </div>
      <div class="text-right">
        <!-- The connection status is managed by the client inside the iframe, so we show a generic status here -->
        <p class="font-semibold text-sm text-yellow-400">
          {{ connectionStatus }}
        </p>
      </div>
    </header>
    <main class="flex-grow w-full h-full relative bg-black">
      <iframe
        v-if="spiceIframeSrc"
        :src="spiceIframeSrc"
        @load="onIframeLoad"
        class="w-full h-full border-0"
        title="SPICE Console"
      ></iframe>
      <div v-else class="flex items-center justify-center h-full">
        <p>Generating connection URL...</p>
      </div>
    </main>
  </div>
</template>

<style scoped>
/* Scoped styles for this component */
iframe {
  /* Ensures the iframe takes up all available space in the main container */
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}
</style>


