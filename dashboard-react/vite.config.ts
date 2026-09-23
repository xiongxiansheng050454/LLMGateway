import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/',
  server: {
    proxy: {
      '/admin': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
      '/v1': 'http://localhost:8080',
    },
  },
})
