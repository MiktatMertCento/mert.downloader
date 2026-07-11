import { type FormEvent } from 'react'

interface DownloadFormProps {
    url: string
    setUrl: (url: string) => void
    onSubmit: (url: string) => void
    isLoading: boolean
}

export default function DownloadForm({ url, setUrl, onSubmit, isLoading }: DownloadFormProps) {

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault()
        const trimmed = url.trim()
        if (!trimmed) return
        onSubmit(trimmed)
    }

    return (
        <form onSubmit={handleSubmit} className="w-full max-w-2xl mx-auto min-w-0 px-1 py-4">
            <div className="relative group">
                <div className="absolute -inset-1 rounded-2xl bg-gradient-to-r from-primary via-accent to-primary-light opacity-60 blur-md transition duration-500 group-hover:opacity-100 group-hover:duration-200 pointer-events-none" />
                <div className="relative flex items-center gap-1 bg-surface-light rounded-2xl border border-surface-lighter/50 overflow-hidden min-w-0">
                    <input
                        id="url-input"
                        type="url"
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                        placeholder="Instagram / YouTube linki..."
                        className="flex-1 min-w-0 bg-transparent px-3 py-3 sm:px-5 sm:py-4 text-text placeholder:text-text-muted outline-none text-sm sm:text-lg"
                        disabled={isLoading}
                        aria-label="Video URL"
                    />
                    <button
                        id="download-btn"
                        type="submit"
                        disabled={isLoading || !url.trim()}
                        className="shrink-0 mr-1 sm:mr-2 px-3 py-2 sm:px-6 sm:py-3 text-sm sm:text-base bg-gradient-to-r from-primary to-primary-dark text-white font-semibold rounded-xl transition-all duration-200 hover:shadow-lg hover:shadow-primary/25 hover:scale-105 disabled:opacity-40 disabled:hover:scale-100 disabled:hover:shadow-none cursor-pointer disabled:cursor-not-allowed"
                    >
                        {isLoading ? (
                            <span className="flex items-center gap-1.5 sm:gap-2">
                                <svg className="animate-spin h-4 w-4 sm:h-5 sm:w-5 shrink-0" viewBox="0 0 24 24" fill="none">
                                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
                                </svg>
                                <span className="sr-only sm:not-sr-only">İndiriliyor</span>
                            </span>
                        ) : (
                            'İndir'
                        )}
                    </button>
                </div>
            </div>
        </form>
    )
}
