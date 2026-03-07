import { useState, useEffect } from 'react'
import DownloadForm from './components/DownloadForm'
import DownloadResult from './components/DownloadResult'
import { downloadMedia, checkHealth, type DownloadResponse } from './lib/api'

export default function App() {
  const [isLoading, setIsLoading] = useState(false)
  const [result, setResult] = useState<DownloadResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [serverStatus, setServerStatus] = useState<'checking' | 'online' | 'offline'>('checking')

  useEffect(() => {
    checkHealth()
      .then(() => setServerStatus('online'))
      .catch(() => setServerStatus('offline'))
  }, [])

  const handleSubmit = async (url: string) => {
    setIsLoading(true)
    setResult(null)
    setError(null)

    try {
      const data = await downloadMedia(url)
      setResult(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Bilinmeyen hata')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-dvh flex flex-col">
      {/* Status indicator */}
      <div className="fixed top-4 right-4 z-50">
        <div className="flex items-center gap-2 px-3 py-1.5 bg-surface-light/80 backdrop-blur-sm rounded-full border border-surface-lighter/40 text-xs">
          <div
            className={`w-2 h-2 rounded-full ${serverStatus === 'online'
                ? 'bg-success animate-pulse'
                : serverStatus === 'offline'
                  ? 'bg-accent'
                  : 'bg-text-muted animate-pulse'
              }`}
          />
          <span className="text-text-muted" data-testid="server-status">
            {serverStatus === 'online' ? 'Bağlı' : serverStatus === 'offline' ? 'Bağlantı yok' : 'Kontrol ediliyor...'}
          </span>
        </div>
      </div>

      {/* Main content */}
      <main className="flex-1 flex flex-col items-center justify-center px-4 py-20">
        {/* Hero */}
        <div className="text-center mb-12 space-y-4">
          <h1 className="text-5xl sm:text-6xl font-extrabold tracking-tight">
            <span className="bg-gradient-to-r from-primary via-accent to-primary-light bg-clip-text text-transparent">
              Mert
            </span>
            <span className="text-text"> Downloader</span>
          </h1>
          <p className="text-text-muted text-lg max-w-md mx-auto">
            Instagram ve YouTube içeriklerini hızlıca indir
          </p>
        </div>

        {/* Form */}
        <DownloadForm onSubmit={handleSubmit} isLoading={isLoading} />

        {/* Results */}
        <DownloadResult data={result} error={error} />

        {/* Supported platforms */}
        <div className="mt-16 flex flex-wrap justify-center gap-6 text-text-muted/50 text-sm">
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2.163c3.204 0 3.584.012 4.85.07 3.252.148 4.771 1.691 4.919 4.919.058 1.265.069 1.645.069 4.849 0 3.205-.012 3.584-.069 4.849-.149 3.225-1.664 4.771-4.919 4.919-1.266.058-1.644.07-4.85.07-3.204 0-3.584-.012-4.849-.07-3.26-.149-4.771-1.699-4.919-4.92-.058-1.265-.07-1.644-.07-4.849 0-3.204.013-3.583.07-4.849.149-3.227 1.664-4.771 4.919-4.919 1.266-.057 1.645-.069 4.849-.069zM12 0C8.741 0 8.333.014 7.053.072 2.695.272.273 2.69.073 7.052.014 8.333 0 8.741 0 12c0 3.259.014 3.668.072 4.948.2 4.358 2.618 6.78 6.98 6.98C8.333 23.986 8.741 24 12 24c3.259 0 3.668-.014 4.948-.072 4.354-.2 6.782-2.618 6.979-6.98.059-1.28.073-1.689.073-4.948 0-3.259-.014-3.667-.072-4.947-.196-4.354-2.617-6.78-6.979-6.98C15.668.014 15.259 0 12 0zm0 5.838a6.162 6.162 0 100 12.324 6.162 6.162 0 000-12.324zM12 16a4 4 0 110-8 4 4 0 010 8zm6.406-11.845a1.44 1.44 0 100 2.881 1.44 1.44 0 000-2.881z" />
            </svg>
            Instagram
          </div>
          <div className="flex items-center gap-2">
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
              <path d="M23.498 6.186a3.016 3.016 0 00-2.122-2.136C19.505 3.546 12 3.546 12 3.546s-7.505 0-9.377.504A3.017 3.017 0 00.502 6.186C0 8.07 0 12 0 12s0 3.93.502 5.814a3.016 3.016 0 002.122 2.136c1.871.504 9.376.504 9.376.504s7.505 0 9.377-.504a3.015 3.015 0 002.122-2.136C24 15.93 24 12 24 12s0-3.93-.502-5.814zM9.545 15.568V8.432L15.818 12l-6.273 3.568z" />
            </svg>
            YouTube
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="py-6 text-center text-text-muted/40 text-xs">
        Mert Downloader &middot; Sadece kişisel kullanım için
      </footer>
    </div>
  )
}
