import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:3000'
const devServerCommand = process.env.PLAYWRIGHT_DEV_SERVER_COMMAND || 'node ./node_modules/vite/bin/vite.js --host 127.0.0.1 --port 3000'

export default defineConfig({
  testDir: './e2e',
  testMatch: ['*.spec.js', '*.setup.js'],
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: process.env.CI ? [['html', { open: 'never' }], ['list']] : 'list',
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    locale: 'zh-CN',
  },
  webServer: process.env.PLAYWRIGHT_SKIP_WEBSERVER
    ? undefined
    : {
        command: devServerCommand,
        port: 3000,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      },
  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.js/,
      use: {
        ...devices['Desktop Chrome'],
      },
    },
    {
      name: 'chromium',
      dependencies: ['setup'],
      testIgnore: /auth\.setup\.js/,
      use: {
        ...devices['Desktop Chrome'],
        storageState: './e2e/.auth/admin.json',
      },
    },
    {
      name: 'mobile-chromium',
      dependencies: ['setup'],
      testMatch: /mobile-smoke\.spec\.js/,
      use: {
        ...devices['iPhone 13'],
        storageState: './e2e/.auth/admin.json',
      },
    },
  ],
})
