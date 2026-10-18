import { defineConfig, loadEnv, type Plugin } from 'vite';
import react from '@vitejs/plugin-react';
import { fileURLToPath, URL } from 'node:url';

/** Replaces %SITE_URL% in index.html with VITE_SITE_URL (empty if not set). */
function siteUrl(url: string): Plugin {
  return {
    name: 'site-url',
    transformIndexHtml: (html) => html.replace(/%SITE_URL%/g, url),
  };
}

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_');
  const site = (env.VITE_SITE_URL ?? '').replace(/\/$/, '');

  return {
    plugins: [react(), siteUrl(site)],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    optimizeDeps: {
      exclude: ['lucide-react'],
    },
    // In development the browser talks to this server and /api is forwarded to the backend,
    // so there are no CORS problems. Override the target with VITE_PROXY_TARGET.
    server: {
      port: 5174,
      proxy: {
        '/api': { target: env.VITE_PROXY_TARGET || 'http://localhost:8080', changeOrigin: true },
      },
    },
  };
});
