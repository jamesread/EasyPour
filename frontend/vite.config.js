import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'
import pkg from './package.json' with { type: 'json' }

const __dirname = dirname(fileURLToPath(import.meta.url))
const backendTarget = `http://localhost:${process.env.PORT || 9654}`
const backendPaths = ['/easypour.v1.EasyPourService', '/login', '/logout', '/upload', '/images', '/orders/events', '/metrics']

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
    proxy: Object.fromEntries(
      backendPaths.map((path) => [path, { target: backendTarget, changeOrigin: true }]),
    ),
  },
})
