import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const appOrigin = env.VITE_APP_ORIGIN?.replace(/\/$/, '') ?? ''
  const shareTargetAction = appOrigin ? `${appOrigin}/` : '/'

  return {
    plugins: [
      react(),
      tailwindcss(),
      VitePWA({
        registerType: 'autoUpdate',
        includeAssets: ['pwa-192.png', 'pwa-512.png'],
        manifest: {
          id: '/',
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
            params: {
              title: 'title',
              text: 'text',
              url: 'url',
            },
          },
        },
        workbox: {
          navigateFallback: '/index.html',
        },
      }),
    ],
    server: {
      proxy: {
        '/api': 'http://localhost:1905',
        '/downloads': 'http://localhost:1905',
      },
    },
  }
})
