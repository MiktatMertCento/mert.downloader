import { useState, type FormEvent } from 'react'

interface DownloadFormProps {
    onSubmit: (url: string) => void
    isLoading: boolean
}

export default function DownloadForm({ onSubmit, isLoading }: DownloadFormProps) {
    const [url, setUrl] = useState('')

    const handleSubmit = (e: FormEvent) => {
        e.preventDefault()
        const trimmed = url.trim()
        if (!trimmed) return
        onSubmit(trimmed)
    }

    return (
        <form onSubmit={handleSubmit} className="w-full max-w-2xl mx-auto">
            <div className="relative group">
                <div className="absolute -inset-0.5 bg-gradient-to-r from-primary via-accent to-primary-light rounded-2xl blur opacity-60 group-hover:opacity-100 transition duration-500 group-hover:duration-200 animate-gradient" />
                <div className="relative flex items-center bg-surface-light rounded-2xl border border-surface-lighter/50">
                    <input
                        id="url-input"
                        type="url"
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                        placeholder="Instagram / YouTube linki..."
                        className="flex-1 min-w-0 bg-transparent px-4 py-3 sm:px-5 sm:py-4 text-text placeholder:text-text-muted outline-none text-base sm:text-lg"
                        disabled={isLoading}
                        aria-label="Video URL"
                    />
                    <button
                        id="download-btn"
                        type="submit"
                        disabled={isLoading || !url.trim()}
                        className="shrink-0 mr-1 sm:mr-2 px-4 py-2 sm:px-6 sm:py-3 text-sm sm:text-base bg-gradient-to-r from-primary to-primary-dark text-white font-semibold rounded-xl transition-all duration-200 hover:shadow-lg hover:shadow-primary/25 hover:scale-105 disabled:opacity-40 disabled:hover:scale-100 disabled:hover:shadow-none cursor-pointer disabled:cursor-not-allowed"
                    >
                        {isLoading ? (
                            <span className="flex items-center gap-2">
                                <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24" fill="none">
                                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
                                </svg>
                                İndiriliyor
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
