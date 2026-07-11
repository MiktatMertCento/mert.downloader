import { useEffect, useMemo, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react'
import { createPortal } from 'react-dom'
import type { DownloadResponse, DownloadedFile, UpscaleJob } from '../lib/api'
import { startUpscale, waitForUpscale } from '../lib/api'
import { applyMobileVideoAttributes, formatEta, formatSize, getMediaTypeLabel, getPreviewMediaClass } from '../lib/utils'
import MediaCarousel from './MediaCarousel'

interface DownloadResultProps {
    data: DownloadResponse | null
    error: string | null
}

function fileForCompareView(file: DownloadedFile, showOriginal: boolean): DownloadedFile {
    if (!showOriginal || !file.originalPath) return file
    return {
        ...file,
        path: file.originalPath,
        width: file.originalWidth ?? file.width,
        height: file.originalHeight ?? file.height,
    }
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
            key={file.path}
            src={file.path}
            alt={file.filename}
            className={getPreviewMediaClass(file, 'fullscreen')}
            data-testid="preview-image"
        />
    )
}

function UpscaleProgress({ job }: { job: UpscaleJob }) {
    return (
        <div
            className="w-full max-w-sm rounded-xl border border-surface-lighter/50 bg-surface/95 px-4 py-3 shadow-lg"
            data-testid="upscale-progress"
            onClick={(e) => e.stopPropagation()}
        >
            <div className="flex items-center justify-between gap-3 text-sm">
                <span className="font-medium text-text">2x netleştiriliyor</span>
                <span className="tabular-nums text-text-muted" data-testid="upscale-percent">
                    {Math.round(job.percent)}%
                </span>
            </div>
            <div className="mt-2 h-2 overflow-hidden rounded-full bg-surface-lighter/40">
                <div
                    className="h-full rounded-full bg-primary-light transition-all duration-300"
                    style={{ width: `${Math.min(100, Math.max(0, job.percent))}%` }}
                />
            </div>
            <p className="mt-2 text-xs text-text-muted" data-testid="upscale-eta">
                {job.status === 'queued'
                    ? 'Sıraya alındı…'
                    : job.eta_seconds > 0
                      ? `Kalan süre ${formatEta(job.eta_seconds)}`
                      : 'Neredeyse bitti…'}
            </p>
        </div>
    )
}

function PreviewModal({
    files,
    initialIndex,
    onClose,
    onFileEnhanced,
}: {
    files: DownloadedFile[]
    initialIndex: number
    onClose: () => void
    onFileEnhanced: (index: number, file: DownloadedFile) => void
}) {
    const [currentIndex, setCurrentIndex] = useState(initialIndex)
    const [upscaleJob, setUpscaleJob] = useState<UpscaleJob | null>(null)
    const [upscaleError, setUpscaleError] = useState<string | null>(null)
    const [comparing, setComparing] = useState(false)
    const abortRef = useRef<AbortController | null>(null)

    const current = files[currentIndex]
    const canEnhance = current?.type === 'image' && !upscaleJob
    const isUpscaling = upscaleJob?.status === 'queued' || upscaleJob?.status === 'running'
    const canCompare = Boolean(current?.type === 'image' && current.originalPath && !isUpscaling)

    const displayFiles = useMemo(
        () => files.map((file, index) => (index === currentIndex ? fileForCompareView(file, comparing) : file)),
        [files, currentIndex, comparing],
    )

    useEffect(() => {
        setComparing(false)
    }, [currentIndex])

    useEffect(() => {
        const previousOverflow = document.body.style.overflow
        document.body.style.overflow = 'hidden'
        return () => {
            document.body.style.overflow = previousOverflow
            abortRef.current?.abort()
        }
    }, [])

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') onClose()
        }

        window.addEventListener('keydown', onKeyDown)
        return () => window.removeEventListener('keydown', onKeyDown)
    }, [onClose])

    const stopComparing = (event?: ReactPointerEvent<HTMLButtonElement>) => {
        event?.preventDefault()
        event?.stopPropagation()
        setComparing(false)
    }

    const startComparing = (event: ReactPointerEvent<HTMLButtonElement>) => {
        event.preventDefault()
        event.stopPropagation()
        if (typeof event.currentTarget.setPointerCapture === 'function') {
            event.currentTarget.setPointerCapture(event.pointerId)
        }
        setComparing(true)
    }

    const handleEnhance = async () => {
        if (!current || current.type !== 'image' || isUpscaling) return
        setUpscaleError(null)
        setComparing(false)
        abortRef.current?.abort()
        const controller = new AbortController()
        abortRef.current = controller

        try {
            const started = await startUpscale(current.path)
            setUpscaleJob(started)
            const finished = await waitForUpscale(started.id, setUpscaleJob, controller.signal)
            if (finished.status === 'failed') {
                setUpscaleError(finished.error || 'Netleştirme başarısız')
                setUpscaleJob(null)
                return
            }
            if (finished.result_path && finished.filename) {
                onFileEnhanced(currentIndex, {
                    filename: finished.filename,
                    path: finished.result_path,
                    type: 'image',
                    size: finished.size || 0,
                    width: finished.width,
                    height: finished.height,
                    originalPath: current.path,
                    originalWidth: current.width,
                    originalHeight: current.height,
                })
            }
            setUpscaleJob(null)
        } catch (err) {
            if (controller.signal.aborted) return
            setUpscaleError(err instanceof Error ? err.message : 'Netleştirme başarısız')
            setUpscaleJob(null)
        }
    }

    const isMulti = files.length > 1

    return createPortal(
        <div
            className="fixed inset-0 z-[9999] flex h-dvh min-h-dvh w-screen max-w-none flex-col bg-surface/90 backdrop-blur-md animate-[fadeIn_0.2s_ease-out]"
            onClick={onClose}
            role="dialog"
            aria-modal="true"
            aria-label="Önizleme"
            style={{ overscrollBehavior: 'contain' }}
        >
            <header
                className="relative z-50 flex shrink-0 items-start gap-2 px-3 pt-[max(0.75rem,env(safe-area-inset-top))] pr-[max(0.75rem,env(safe-area-inset-right))] pl-[max(0.75rem,env(safe-area-inset-left))]"
                onClick={(e) => e.stopPropagation()}
                onPointerDown={(e) => e.stopPropagation()}
            >
                <div className="min-w-0 flex-1 flex flex-col items-center gap-2">
                    {current?.type === 'image' && (
                        <>
                            {isUpscaling && upscaleJob ? (
                                <UpscaleProgress job={upscaleJob} />
                            ) : (
                                <div className="flex flex-wrap items-center justify-center gap-2">
                                    <button
                                        type="button"
                                        onClick={(e) => {
                                            e.stopPropagation()
                                            void handleEnhance()
                                        }}
                                        disabled={!canEnhance}
                                        className="rounded-full border border-primary/40 bg-surface/95 px-4 py-2 text-sm font-semibold text-primary-light shadow-lg transition hover:border-primary-light hover:bg-surface disabled:cursor-not-allowed disabled:opacity-50 cursor-pointer"
                                        data-testid="enhance-button"
                                    >
                                        2x Netleştir
                                    </button>
                                    {canCompare && (
                                        <button
                                            type="button"
                                            onPointerDown={startComparing}
                                            onPointerUp={stopComparing}
                                            onPointerCancel={stopComparing}
                                            onLostPointerCapture={() => setComparing(false)}
                                            onClick={(e) => e.stopPropagation()}
                                            onContextMenu={(e) => e.preventDefault()}
                                            className={`select-none rounded-full border px-4 py-2 text-sm font-semibold shadow-lg transition cursor-pointer touch-none ${
                                                comparing
                                                    ? 'border-accent/50 bg-accent/20 text-accent-light'
                                                    : 'border-surface-lighter/60 bg-surface/95 text-text hover:border-primary-light'
                                            }`}
                                            data-testid="compare-button"
                                            aria-pressed={comparing}
                                        >
                                            {comparing ? 'Önceki' : 'Farkı gör'}
                                        </button>
                                    )}
                                </div>
                            )}
                            {canCompare && !isUpscaling && (
                                <p className="text-[11px] text-text-muted" data-testid="compare-hint">
                                    {comparing ? 'Eski hali gösteriliyor' : 'Farkı gör: basılı tut'}
                                </p>
                            )}
                            {upscaleError && (
                                <p className="rounded-lg bg-red-500/15 px-3 py-1.5 text-xs text-red-300" data-testid="upscale-error">
                                    {upscaleError}
                                </p>
                            )}
                        </>
                    )}
                </div>
                <button
                    type="button"
                    onClick={(e) => {
                        e.stopPropagation()
                        onClose()
                    }}
                    className="relative z-50 shrink-0 w-10 h-10 bg-surface border-2 border-surface-lighter rounded-full flex items-center justify-center text-text shadow-lg hover:text-primary-light hover:border-primary-light hover:scale-110 transition-all cursor-pointer"
                    aria-label="Kapat"
                    data-testid="preview-close"
                >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                </button>
            </header>

            <div
                className="flex flex-1 min-h-0 w-full items-center justify-center px-3 pb-[max(1rem,env(safe-area-inset-bottom))]"
                onClick={(e) => e.stopPropagation()}
                style={{ touchAction: 'pan-x pan-y pinch-zoom' }}
            >
                {isMulti ? (
                    <MediaCarousel
                        files={displayFiles}
                        initialIndex={currentIndex}
                        variant="fullscreen"
                        className="w-full max-w-3xl h-full max-h-full"
                        onIndexChange={setCurrentIndex}
                    />
                ) : (
                    <SinglePreview file={displayFiles[0]} />
                )}
            </div>
        </div>,
        document.body,
    )
}

export default function DownloadResult({ data, error }: DownloadResultProps) {
    const [previewIndex, setPreviewIndex] = useState<number | null>(null)
    const [files, setFiles] = useState<DownloadedFile[]>(data?.files ?? [])

    useEffect(() => {
        setFiles(data?.files ?? [])
    }, [data])

    if (error) {
        return (
            <div
                id="error-result"
                className="w-full max-w-2xl mx-auto mt-8 p-4 sm:p-5 bg-red-500/10 border border-red-500/30 rounded-2xl backdrop-blur-sm min-w-0 overflow-hidden"
            >
                <div className="flex items-start gap-3 min-w-0">
                    <div className="w-10 h-10 rounded-full bg-red-500/20 flex items-center justify-center shrink-0">
                        <svg className="w-5 h-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01M12 5a7 7 0 100 14 7 7 0 000-14z" />
                        </svg>
                    </div>
                    <div className="min-w-0">
                        <h2 className="font-semibold text-red-300">Hata</h2>
                        <p className="mt-1 text-sm text-red-200/90 break-words" role="alert">
                            {error}
                        </p>
                    </div>
                </div>
            </div>
        )
    }

    if (!data) return null

    return (
        <div id="download-result" className="w-full max-w-2xl mx-auto mt-8 space-y-4 min-w-0 overflow-x-clip">
            {previewIndex !== null && (
                <PreviewModal
                    files={files}
                    initialIndex={previewIndex}
                    onClose={() => setPreviewIndex(null)}
                    onFileEnhanced={(index, file) => {
                        setFiles((prev) => prev.map((item, i) => (i === index ? file : item)))
                    }}
                />
            )}

            <div className="p-4 sm:p-5 bg-surface-light/80 border border-surface-lighter/40 rounded-2xl backdrop-blur-sm min-w-0 overflow-hidden">
                <div className="flex items-start gap-3 sm:gap-4 min-w-0">
                    <div className="w-11 h-11 sm:w-12 sm:h-12 rounded-xl bg-linear-to-br from-success/20 to-primary/20 flex items-center justify-center shrink-0">
                        <svg className="w-6 h-6 text-success" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                        </svg>
                    </div>
                    <div className="flex-1 min-w-0 overflow-hidden">
                        <h2 className="text-base sm:text-lg font-bold text-text">İndirme Başarılı</h2>
                        <div className="flex flex-wrap items-center gap-2 sm:gap-3 mt-2 text-sm text-text-muted min-w-0">
                            {data.username && (
                                <span className="text-primary-light font-medium truncate max-w-full">@{data.username}</span>
                            )}
                            <span className="px-2.5 py-0.5 bg-primary/15 text-primary-light rounded-full text-xs font-medium shrink-0">
                                {getMediaTypeLabel(data.media_type)}
                            </span>
                            <span className="shrink-0">{files.length} dosya</span>
                        </div>
                        {data.caption && (
                            <p className="mt-3 text-sm text-text-muted leading-relaxed max-h-24 overflow-y-auto break-words whitespace-pre-wrap">{data.caption}</p>
                        )}
                    </div>
                </div>
            </div>

            <div className="grid grid-cols-1 gap-3 w-full min-w-0">
                {files.map((file, index) => (
                    <div
                        key={`${file.path}-${index}`}
                        className="flex items-center gap-2.5 sm:gap-4 p-3 sm:p-4 bg-surface-light/60 border border-surface-lighter/30 rounded-xl min-w-0 overflow-hidden"
                    >
                        <div className="w-9 h-9 sm:w-10 sm:h-10 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                            {file.type === 'video' ? (
                                <svg className="w-5 h-5 text-primary-light" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                            ) : (
                                <svg className="w-5 h-5 text-primary-light" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                                </svg>
                            )}
                        </div>
                        <div className="flex-1 min-w-0 overflow-hidden">
                            <p className="text-sm font-medium text-text truncate" title={file.filename}>{file.filename}</p>
                            <p className="text-xs text-text-muted mt-0.5 truncate">
                                {file.type === 'video' ? 'Video' : 'Fotoğraf'}
                                {file.originalPath ? ' · 2x net' : ''}
                                {file.size > 0 && ` · ${formatSize(file.size)}`}
                                {file.width && file.height && ` · ${file.width}×${file.height}`}
                            </p>
                        </div>
                        <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
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
