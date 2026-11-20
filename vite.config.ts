import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
  plugins: [svelte(), svelteTesting()],
  clearScreen: false,
  server: { port: 1430, strictPort: true, watch: { ignored: ['**/build/**', '**/wailsjs/runtime/**'] } },
  optimizeDeps: {
    entries: ['index.html'],
    // A first visit to a lazy editor must not restart the dev server's dependency optimizer.
    include: ['@codemirror/commands', '@codemirror/language', '@codemirror/language-data', '@codemirror/merge', '@codemirror/search', '@codemirror/state', '@codemirror/view', '@lezer/highlight'],
  },
  test: {
    // jsdom everywhere: the stores touch window at import time.
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    setupFiles: ['src/test/setup.ts'],
    passWithNoTests: false,
  },
});
