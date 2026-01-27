import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    allowedHosts: ['mindstorm'],
    port: 3000,
    proxy: {
      '/easypour.v1.EasyPourService': {
        target: 'http://localhost:9654',
        changeOrigin: true,
      },
      '/login': {
        target: 'http://localhost:9654',
        changeOrigin: true,
      },
      '/upload': {
        target: 'http://localhost:9654',
        changeOrigin: true,
      },
      '/images': {
        target: 'http://localhost:9654',
        changeOrigin: true,
      },
    },
  },
})
