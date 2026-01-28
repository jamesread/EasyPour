import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import pkg from './package.json'

export default defineConfig({
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  plugins: [vue()],
  server: {
    allowedHosts: ['localhost', 'mindstorm'],
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
      '/logout': {
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
