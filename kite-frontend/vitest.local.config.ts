import { defineConfig } from 'vitest/config';

// Explicitly opt in: this suite creates QA accounts and CRDs in kite-local-qa.
export default defineConfig({
  test: { environment: 'node', include: ['tests/local-runtime.integration.ts'], testTimeout: 60_000, hookTimeout: 60_000 },
});
