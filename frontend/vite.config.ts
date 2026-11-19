import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('firebase/auth') || id.includes('firebase/app')) return 'firebase-core'
          if (id.includes('firebase/firestore')) return 'firebase-firestore'
          if (id.includes('firebase/storage') || id.includes('firebase/messaging')) return 'firebase-optional'
          // Core Vue ecosystem only — keeps jsQR, QRCode, etc. as lazy chunks
          if (id.includes('/node_modules/vue/') || id.includes('/node_modules/vue-router/') ||
              id.includes('/node_modules/pinia/') || id.includes('/node_modules/axios/') ||
              id.includes('/node_modules/@vue/')) return 'vendor'
        }
      }
    }
  }
})
