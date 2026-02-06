import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // Allow any host so port-forwarded domains (e.g. *.xhd2015.xyz) work
    allowedHosts: true
  },
})
