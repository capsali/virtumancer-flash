<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import RFB from '@novnc/novnc/lib/rfb';

const props = defineProps({
  hostId: String,
  vmName: String,
});

const vncCanvas = ref(null);
const connectionStatus = ref('Connecting...');
const rfb = ref(null);

const connect = () => {
  if (!vncCanvas.value) {
    console.error("VNC canvas ref is not available.");
    connectionStatus.value = 'Error: Canvas not ready.';
    return;
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const url = `${protocol}//${window.location.host}/api/v1/hosts/${props.hostId}/vms/${props.vmName}/console`;
  
  const options = { wsProtocols: ['binary'] };

  const newRfb = new RFB(vncCanvas.value, url, options);

  newRfb.addEventListener('connect', () => {
    connectionStatus.value = 'Connected';
  });

  newRfb.addEventListener('disconnect', () => {
    connectionStatus.value = 'Disconnected';
  });
  
  rfb.value = newRfb;
};

const disconnect = () => {
  if (rfb.value) {
    rfb.value.disconnect();
    rfb.value = null;
  }
};

onMounted(connect);
onUnmounted(disconnect);
</script>

<template>
  <div class="w-full h-full relative bg-black">
    <div ref="vncCanvas" class="w-full h-full"></div>
    <div class="absolute top-2 right-2 text-xs px-2 py-1 rounded" :class="{
        'text-green-400 bg-green-900/50': connectionStatus === 'Connected',
        'text-red-400 bg-red-900/50': connectionStatus === 'Disconnected',
        'text-yellow-400 bg-yellow-900/50': connectionStatus !== 'Connected' && connectionStatus !== 'Disconnected'
    }">
      {{ connectionStatus }}
    </div>
  </div>
</template>

