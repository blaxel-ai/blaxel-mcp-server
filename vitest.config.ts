import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    // Load environment variables from .env files
    env: {
      // This will be merged with process.env
    },
    // Setup file to load dotenv
    setupFiles: ['./vitest.setup.ts'],
    // Test timeout
    testTimeout: 120000,
    // Hook timeout
    hookTimeout: 30000,
  },
});
