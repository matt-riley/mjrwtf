// @ts-check

/** @type {import('@playwright/test').PlaywrightTestConfig} */
const config = {
  testDir: './tests',
  // Allow enough time for docker build + container startup on slower runners.
  timeout: 5 * 60 * 1000,
  expect: {
    timeout: 20 * 1000,
  },
  use: {
    headless: true,
  },
  reporter: [['list']],
};

module.exports = config;
