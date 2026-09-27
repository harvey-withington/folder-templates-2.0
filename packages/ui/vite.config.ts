import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { svelteTesting } from '@testing-library/svelte/vite'
import { lucidePerIcon } from './lucide-per-icon'

// Vite here only serves the tests: the package ships source that each host
// compiles with its own Vite build.
export default defineConfig({
  plugins: [lucidePerIcon(), svelte(), svelteTesting()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
  },
})
