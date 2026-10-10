import { defineConfig } from '@playwright/test';

// Requires the stack running (default: http://localhost:8080).
//   make up && cd tests/e2e && npm install && npx playwright install chromium && npm test
export default defineConfig({
  testDir: './specs',
  timeout: 30_000,
  retries: 0,
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:8080',
    locale: 'ru-RU',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  // CHROMIUM_PATH lets NixOS/self-managed systems point at a system browser;
  // without it Playwright uses its own downloaded Chromium.
  projects: [{
    name: 'chromium',
    use: {
      launchOptions: process.env.CHROMIUM_PATH
        ? { executablePath: process.env.CHROMIUM_PATH }
        : {},
    },
  }],
});
