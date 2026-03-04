import { defineConfig, devices } from '@playwright/test'

const BASE_URL = process.env.E2E_BASE_URL ?? 'http://localhost:5173'
const API_URL = process.env.E2E_API_URL ?? 'http://localhost:8080'

export default defineConfig({
  testDir: './tests',
  fullyParallel: false, // sequential — tests share DB state
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? 'github' : 'html',
  timeout: 30_000,
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  // Global setup/teardown could start services; for now, assume running.
  webServer: process.env.CI
    ? undefined
    : {
        command: 'npm run dev',
        url: BASE_URL,
        cwd: '../web',
        reuseExistingServer: true,
        timeout: 30_000,
      },
})

export { API_URL }
