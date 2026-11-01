import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';

// UI-A #228: the production bundle is served by the existing v2 ALM server
// (web/v2/journey-server.mjs) under the /app feature gate; base stays
// relative and routing is hash-based so no server-side rewrite is required.
export default defineConfig({
  plugins: [react()],
  base: './',
  build: {outDir: 'dist', sourcemap: true, target: 'es2022'},
  server: {port: 5173, strictPort: false},
});
