// @ts-check
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './e2e',
  timeout: 30000,
  retries: 0,
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
    { name: 'firefox', use: { browserName: 'firefox' } },
    // Uncomment to test Safari (requires macOS)
    // { name: 'webkit', use: { browserName: 'webkit' } },
  ],
  // Don't start server automatically - user runs it separately
  webServer: null,
});
