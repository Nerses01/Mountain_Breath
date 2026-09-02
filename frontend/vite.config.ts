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
      '/uploads': 'http://localhost:8080',
      '/sitemap.xml': 'http://localhost:8080',
    },
  },
  // vite preview (the built app, used by Lighthouse CI) does NOT inherit
  // server.proxy — without this mirror the audited pages would render
  // against a dead API and measure an error state.
  preview: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/uploads': 'http://localhost:8080',
      '/sitemap.xml': 'http://localhost:8080',
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
