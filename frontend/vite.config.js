import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'
import pkg from './package.json'

const __dirname = dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  resolve: {
    alias: {
      '../../gen/easypour/v1/easypour_pb.js': resolve(__dirname, 'gen/easypour/v1/easypour_pb.ts'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
  },
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
      '/orders/events': {
        target: 'http://localhost:9654',
        changeOrigin: true,
      },
    },
  },
})
