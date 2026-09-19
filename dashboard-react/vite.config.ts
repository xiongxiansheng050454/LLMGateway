import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({ plugins: [react()], base: '/dashboard/', server: { proxy: { '/dashboard/legacy': 'http://localhost:8080' } } })
