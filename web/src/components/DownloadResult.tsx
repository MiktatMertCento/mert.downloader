import { useEffect, useRef, useState } from 'react'
import type { DownloadResponse, DownloadedFile } from '../lib/api'
import { applyMobileVideoAttributes, formatSize, getMediaTypeLabel, getPreviewMediaClass } from '../lib/utils'
import MediaCarousel from './MediaCarousel'

interface DownloadResultProps {
    data: DownloadResponse | null
    error: string | null
}

function SinglePreview({ file }: { file: DownloadedFile }) {
    const videoRef = useRef<HTMLVideoElement>(null)

    useEffect(() => {
        if (file.type !== 'video' || !videoRef.current) return
        applyMobileVideoAttributes(videoRef.current)
    }, [file.path, file.type])

    if (file.type === 'video') {
        return (
            <video
                key={file.path}
                ref={videoRef}
                src={file.path}
                controls
                preload="metadata"
                playsInline
                className={getPreviewMediaClass(file, 'fullscreen')}
                data-testid="preview-video"
            />
        )
    }

    return (
        <img
            src={file.path}
            alt={file.filename}
            className={getPreviewMediaClass(file, 'fullscreen')}
            data-testid="preview-image"
        />
    )
}

function PreviewModal({
    files,
    initialIndex,
    onClose,
}: {
    files: DownloadedFile[]
    initialIndex: number
    onClose: () => void
}) {
    useEffect(() => {
        const previousOverflow = document.body.style.overflow
        document.body.style.overflow = 'hidden'
        return () => {
            document.body.style.overflow = previousOverflow
        }
    }, [])

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') onClose()
        }

        window.addEventListener('keydown', onKeyDown)
        return () => window.removeEventListener('keydown', onKeyDown)
    }, [onClose])

    const isMulti = files.length > 1

    return (
        <div
            className="fixed inset-0 z-[9999] flex items-center justify-center bg-surface/90 backdrop-blur-md animate-[fadeIn_0.2s_ease-out] p-4 pt-[max(1rem,env(safe-area-inset-top))] pb-[max(1rem,env(safe-area-inset-bottom))]"
            onClick={onClose}
            role="dialog"
            aria-modal="true"
            aria-label="Önizleme"
            style={{ overscrollBehavior: 'contain', touchAction: 'manipulation' }}
        >
            <div
                className={`relative flex items-center justify-center animate-[scaleIn_0.2s_ease-out] shadow-2xl shadow-primary/20 rounded-xl min-h-0 ${
                    isMulti ? 'w-full max-w-[min(95vw,100%)] max-h-[min(90dvh,100%)]' : 'max-w-[90vw] max-h-[90vh]'
                }`}
                onClick={(e) => e.stopPropagation()}
            >
                <button
                    onClick={onClose}
                    className="absolute -top-3 -right-3 sm:-top-4 sm:-right-4 z-20 w-10 h-10 bg-surface border-2 border-surface-lighter rounded-full flex items-center justify-center text-text shadow-lg hover:text-primary-light hover:border-primary-light hover:scale-110 transition-all cursor-pointer"
                    aria-label="Kapat"
                >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                </button>

                {isMulti ? (
                    <MediaCarousel
                        files={files}
                        initialIndex={initialIndex}
                        variant="fullscreen"
                        className="w-full"
                    />
                ) : (
                    <SinglePreview file={files[0]} />
                )}
            </div>
        </div>
    )
}

export default function DownloadResult({ data, error }: DownloadResultProps) {
    const [previewIndex, setPreviewIndex] = useState<number | null>(null)

    if (error) {
        return (
            <div
                id="error-result"
                className="w-full max-w-2xl mx-auto mt-8 p-5 bg-red-500/10 border border-red-500/30 rounded-2xl backdrop-blur-sm"
            >
                <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-red-500/20 flex items-center justify-center shrink-0">
                        <svg className="w-5 h-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </div>
                    <p className="text-red-300 text-sm font-medium" role="alert">{error}</p>
                </div>
            </div>
        )
    }

    if (!data) return null

    return (
        <div id="download-result" className="w-full max-w-2xl mx-auto mt-8 space-y-4">
            {previewIndex !== null && (
                <PreviewModal
                    files={data.files}
                    initialIndex={previewIndex}
                    onClose={() => setPreviewIndex(null)}
                />
            )}

            <div className="p-5 bg-surface-light/80 border border-surface-lighter/40 rounded-2xl backdrop-blur-sm">
                <div className="flex items-start gap-4">
                    <div className="w-12 h-12 rounded-xl bg-linear-to-br from-success/20 to-primary/20 flex items-center justify-center shrink-0">
                        <svg className="w-6 h-6 text-success" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                    </div>
                    <div className="flex-1 min-w-0">
                        <h2 className="text-lg font-bold text-text">İndirme Başarılı</h2>
                        <div className="flex flex-wrap gap-3 mt-2 text-sm text-text-muted">
                            {data.username && (
                                <span className="flex items-center gap-1">
                                    <span className="text-primary-light font-medium">@{data.username}</span>
                                </span>
                            )}
                            <span className="px-2.5 py-0.5 bg-primary/15 text-primary-light rounded-full text-xs font-medium">
                                {getMediaTypeLabel(data.media_type)}
                            </span>
                            <span>{data.files.length} dosya</span>
                        </div>
                        {data.caption && (
                            <p className="mt-3 text-sm text-text-muted leading-relaxed max-h-24 overflow-y-auto">{data.caption}</p>
                        )}
                    </div>
                </div>
            </div>

            <div className="grid gap-3">
                {data.files.map((file, index) => (
                    <div
                        key={index}
                        className="flex items-center gap-4 p-4 bg-surface-light/60 border border-surface-lighter/30 rounded-xl"
                    >
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                            {file.type === 'video' ? (
                                <svg className="w-5 h-5 text-primary-light" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                            ) : (
                                <svg className="w-5 h-5 text-primary-light" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                                </svg>
                            )}
                        </div>
                        <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium text-text truncate">{file.filename}</p>
                            <p className="text-xs text-text-muted mt-0.5">
                                {file.type === 'video' ? 'Video' : 'Fotoğraf'}
                                {file.size > 0 && ` · ${formatSize(file.size)}`}
                                {file.width && file.height && ` · ${file.width}×${file.height}`}
                            </p>
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                            <button
                                onClick={() => setPreviewIndex(index)}
                                className="w-9 h-9 rounded-lg bg-primary/10 hover:bg-primary/20 flex items-center justify-center text-primary-light hover:text-primary transition-colors cursor-pointer"
                                aria-label="Önizle"
                                title="Önizle"
                            >
                                <svg className="w-4.5 h-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                </svg>
                            </button>
                            <a
                                href={file.path}
                                download
                                className="w-9 h-9 rounded-lg bg-primary/10 hover:bg-primary/20 flex items-center justify-center text-primary-light hover:text-primary transition-colors"
                                aria-label="İndir"
                                title="İndir"
                            >
                                <svg className="w-4.5 h-4.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                                </svg>
                            </a>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    )
}
