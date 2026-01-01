// @ts-check

/** @type {import('@playwright/test').PlaywrightTestConfig} */
const config = {
  testDir: './tests',
  // Allow enough time for docker build + container startup on slower runners.
  timeout: 10 * 60 * 1000,
  expect: {
    timeout: 20 * 1000,
  },
  // Prefer stability in CI (docker build + shared runners).
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
