// @ts-check

/** @type {import('@playwright/test').PlaywrightTestConfig} */
const config = {
  testDir: './tests',
  timeout: 10 * 60 * 1000,
  expect: {
    timeout: 20 * 1000,
  },
  use: {
    headless: true,
  },
  reporter: [['list']],
};

module.exports = config;
