import { defineConfig } from '@playwright/test'

// End-to-end tests drive a REAL browser against the REAL stack:
// Chromium → Vite (5173) → Go API (8080) → Postgres (Docker).
// Precondition: the Postgres dev container is running.
export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  use: {
    baseURL: 'http://localhost:5173',
    // On failure, keep a trace you can open with `npx playwright show-trace`.
    trace: 'retain-on-failure',
  },
  // Playwright starts (or reuses) both dev servers itself.
  webServer: [
    {
      command: 'go run ./cmd/api',
      cwd: '../backend',
      url: 'http://localhost:8080/health',
      reuseExistingServer: true,
      timeout: 60_000,
    },
    {
      command: 'npm run dev',
      url: 'http://localhost:5173',
      reuseExistingServer: true,
      timeout: 60_000,
    },
  ],
})
