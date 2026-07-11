import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const appOrigin = env.VITE_APP_ORIGIN?.replace(/\/$/, '') ?? ''
  const shareTargetAction = appOrigin ? `${appOrigin}/` : '/'
  const manifestId = appOrigin ? `${appOrigin}/` : '/'

  return {
    plugins: [
      react(),
      tailwindcss(),
      VitePWA({
        registerType: 'autoUpdate',
        includeAssets: ['pwa-192.png', 'pwa-512.png'],
        manifest: {
          id: manifestId,
          name: 'Mert Downloader',
          short_name: 'MertDL',
          description: 'Instagram ve YouTube video/fotoğraf indirme aracı',
          start_url: '/',
          scope: '/',
          theme_color: '#0f172a',
          background_color: '#0f172a',
          display: 'standalone',
          orientation: 'portrait-primary',
          icons: [
            {
              src: '/pwa-192.png',
              sizes: '192x192',
              type: 'image/png',
              purpose: 'any',
            },
            {
              src: '/pwa-512.png',
              sizes: '512x512',
              type: 'image/png',
              purpose: 'any',
            },
            {
              src: '/pwa-512.png',
              sizes: '512x512',
              type: 'image/png',
              purpose: 'maskable',
            },
          ],
          share_target: {
            action: shareTargetAction,
            method: 'GET',
            enctype: 'application/x-www-form-urlencoded',
            params: {
              title: 'title',
              text: 'text',
              url: 'url',
            },
          },
        },
        workbox: {
          // Hashed JS/CSS/icons only — never long-precache index.html
          globPatterns: ['**/*.{js,css,ico,png,svg,woff2,webmanifest}'],
          navigateFallback: '/index.html',
          navigateFallbackDenylist: [/^\/api/, /^\/downloads/],
          cleanupOutdatedCaches: true,
          clientsClaim: true,
          skipWaiting: true,
          runtimeCaching: [
            {
              urlPattern: ({ request }) => request.mode === 'navigate',
              handler: 'NetworkFirst',
              options: {
                cacheName: 'html-navigations',
                networkTimeoutSeconds: 3,
                expiration: {
                  maxEntries: 8,
                  maxAgeSeconds: 60 * 60 * 24,
                },
              },
            },
            {
              urlPattern: ({ url }) => url.pathname === '/index.html',
              handler: 'NetworkFirst',
              options: {
                cacheName: 'html-shell',
                networkTimeoutSeconds: 3,
              },
            },
            {
              urlPattern: ({ url }) => url.pathname.startsWith('/downloads/'),
              handler: 'NetworkOnly',
            },
            {
              urlPattern: ({ url }) => url.pathname.startsWith('/api/'),
              handler: 'NetworkOnly',
            },
          ],
        },
      }),
    ],
    server: {
      proxy: {
        '/api': 'http://localhost:1905',
        '/downloads': {
          target: 'http://localhost:1905',
          changeOrigin: true,
        },
      },
    },
  }
})
