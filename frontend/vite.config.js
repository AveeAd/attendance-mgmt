import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Builds straight into the Go backend's embedded frontend directory so
// `go build` in backend/ picks up the latest build with no manual copy step.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../backend/internal/webui/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
