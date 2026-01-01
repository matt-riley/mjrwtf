// @ts-check

/** @type {import('@playwright/test').PlaywrightTestConfig} */
const config = {
  testDir: './tests',
  // Fail fast in CI; we're running the Go server directly (no docker compose build).
  timeout: process.env.CI ? 3 * 60 * 1000 : 5 * 60 * 1000,
  expect: {
    timeout: 20 * 1000,
  },
  // Prefer stability in CI.
  workers: process.env.CI ? 1 : undefined,
  use: {
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
  },
  reporter: [['list'], ['html', { open: 'never' }]],
};

module.exports = config;
