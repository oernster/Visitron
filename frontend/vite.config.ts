import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Node's own web storage flag. On Node 26.13.0 (measured 2026-11-23) a
// localStorage global is on by default and answers undefined unless given a
// file, which hides jsdom's. Turning it off lets jsdom's show through; a Node
// that does not know the flag has no such global to turn off.
const nodeWebStorage = '--experimental-webstorage'
const workerFlags = process.allowedNodeEnvironmentFlags.has(nodeWebStorage)
  ? [nodeWebStorage.replace('--', '--no-')]
  : []

export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    poolOptions: { forks: { execArgv: workerFlags } },
    coverage: {
      provider: 'v8',
      include: ['src/**'],
      reporter: ['text'],
    },
  },
})
