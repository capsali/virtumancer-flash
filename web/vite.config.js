import { fileURLToPath, URL } from 'node:url'
import fs from 'node:fs'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    // Enable HTTPS for the Vite development server to provide a secure context.
    // This uses the same certificates as the Go backend.
    // Note the path to the certs is relative to the `web` directory.
    https: {
      key: fs.readFileSync('../localhost.key'),
      cert: fs.readFileSync('../localhost.crt'),
    },
    proxy: {
      // This rule handles both REST API calls and the VNC console WebSocket.
      '/api': {
        target: 'https://localhost:8888',
        changeOrigin: true,
        // Allow proxying to a backend with a self-signed certificate.
        secure: false,
        // Enable WebSocket proxying for this path.
        ws: true,
      },
      // This rule handles the separate WebSocket for general UI updates.
      '/ws': {
        target: 'wss://localhost:8888',
        ws: true,
        changeOrigin: true,
        // Allow proxying to a backend with a self-signed certificate.
        secure: false,
      }
    }
  }
})


