/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // The dev server forwards /api/* to the Go backend, so the browser only
    // ever talks to one origin (localhost:5173) — no CORS needed, and the
    // same relative URLs work in production behind Nginx.
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    // jsdom simulates a browser DOM in Node so component tests need no
    // real browser (that's Playwright's job).
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    // e2e/ belongs to Playwright's runner, not Vitest.
    exclude: ['e2e/**', 'node_modules/**'],
  },
})
