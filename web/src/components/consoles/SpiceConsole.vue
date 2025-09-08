<script setup>
import { ref, computed } from 'vue';

const props = defineProps({
  hostId: String,
  vmName: String,
});

const spiceIframeUrl = computed(() => {
  const scheme = 'https';
  const port = window.location.port || (scheme === 'https' ? 443 : 80);
  const host = window.location.hostname;
  
  const params = new URLSearchParams({
    host,
    port,
    path: `/api/v1/hosts/${props.hostId}/vms/${props.vmName}/spice`,
    token: `${props.hostId}-${props.vmName}`, // A simple token for identification
    encrypt: scheme === 'https' ? '1' : '0',
  });

  return `/spice/spice_auto.html?${params.toString()}`;
});
</script>

<template>
  <div class="w-full h-full bg-black">
    <iframe :src="spiceIframeUrl" class="w-full h-full border-0"></iframe>
  </div>
</template>

