import { defineConfig, devices } from '@playwright/test'
import { BASE_URL } from './src/env'

const isCI = !!process.env.CI

export default defineConfig({
  testDir: './tests',
  // Every spec works against the SAME seeded course phase -- there is only one
  // intro-course phase, unlike the per-spec fixture phases the platform repo can
  // afford. Seats, peer groups, tutors, and developer profiles are therefore shared
  // mutable state, so the suite runs serially. Mutating specs still snapshot and
  // restore their slice of the fixture (see tests/intro-course/helpers.ts) so a
  // single test can be run on its own and so retries start from a clean state.
  fullyParallel: false,
  forbidOnly: isCI,
  retries: isCI ? 2 : 0,
  workers: 1,
  timeout: 60_000,
  expect: { timeout: 10_000 },

  // Logs in each seeded role once and writes storageState files.
  globalSetup: require.resolve('./src/global-setup'),

  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report', open: 'never' }],
    ...(isCI ? [['github'] as const] : []),
  ],

  outputDir: 'test-results',

  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    video: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'api',
      testMatch: /.*\.api\.spec\.ts/,
    },
    {
      name: 'chromium',
      testIgnore: /.*\.api\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
