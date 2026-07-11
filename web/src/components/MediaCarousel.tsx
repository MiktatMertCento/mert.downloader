import { useCallback, useEffect, useRef, useState } from 'react'
import type { DownloadedFile } from '../lib/api'
import { applyMobileVideoAttributes, getPreviewMediaClass } from '../lib/utils'

interface MediaCarouselProps {
    files: DownloadedFile[]
    initialIndex?: number
    className?: string
    onIndexChange?: (index: number) => void
    variant?: 'inline' | 'fullscreen'
}

const SWIPE_THRESHOLD = 48
const CLICK_SUPPRESS_MS = 350

function preloadNeighborImage(file: DownloadedFile, cache: Set<string>) {
    if (cache.has(file.path) || file.type === 'video') return
    cache.add(file.path)
    const image = new Image()
    image.src = file.path
}

function CarouselSlide({
    file,
    isActive,
    priority,
    variant,
}: {
    file: DownloadedFile
    isActive: boolean
    priority: 'high' | 'low'
    variant: 'inline' | 'fullscreen'
}) {
    const videoRef = useRef<HTMLVideoElement>(null)

    useEffect(() => {
        if (file.type !== 'video' || !videoRef.current) return
        applyMobileVideoAttributes(videoRef.current)
    }, [file.path, file.type])

    const mediaClass = getPreviewMediaClass(file, variant)

    if (file.type === 'video') {
        return (
            <video
                key={file.path}
                ref={videoRef}
                src={file.path}
                controls={isActive}
                preload={priority === 'high' ? 'auto' : 'metadata'}
                playsInline
                className={`${mediaClass} bg-black`}
                data-testid={isActive ? 'carousel-active-video' : 'carousel-video'}
            />
        )
    }

    return (
        <img
            src={file.path}
            alt={file.filename}
            loading={priority === 'high' ? 'eager' : 'lazy'}
            decoding="async"
            className={mediaClass}
            draggable={false}
            data-testid={isActive ? 'carousel-active-image' : 'carousel-image'}
        />
    )
}

export default function MediaCarousel({
    files,
    initialIndex = 0,
    className = '',
    onIndexChange,
    variant = 'fullscreen',
}: MediaCarouselProps) {
    const [index, setIndex] = useState(initialIndex)
    const touchStartX = useRef(0)
    const touchDeltaX = useRef(0)
    const suppressClickUntil = useRef(0)
    const cacheRef = useRef<Set<string>>(new Set())
    const rootRef = useRef<HTMLDivElement>(null)

    const goTo = useCallback(
        (nextIndex: number) => {
            if (nextIndex < 0 || nextIndex >= files.length || nextIndex === index) return
            setIndex(nextIndex)
            onIndexChange?.(nextIndex)
        },
        [files.length, index, onIndexChange],
    )

    const goPrev = useCallback(() => goTo(index - 1), [goTo, index])
    const goNext = useCallback(() => goTo(index + 1), [goTo, index])

    useEffect(() => {
        setIndex(initialIndex)
    }, [initialIndex, files])

    useEffect(() => {
        const cache = cacheRef.current
        const indices = new Set<number>([index])
        if (index > 0) indices.add(index - 1)
        if (index < files.length - 1) indices.add(index + 1)

        indices.forEach((itemIndex) => preloadNeighborImage(files[itemIndex], cache))
    }, [index, files])

    useEffect(() => {
        if (variant !== 'fullscreen') return

        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
            const active = document.activeElement
            if (active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement) return
            event.preventDefault()
            if (event.key === 'ArrowLeft') goPrev()
            if (event.key === 'ArrowRight') goNext()
        }

        window.addEventListener('keydown', onKeyDown)
        return () => window.removeEventListener('keydown', onKeyDown)
    }, [goNext, goPrev, variant])

    const onTouchStart = (event: React.TouchEvent) => {
        touchStartX.current = event.touches[0].clientX
        touchDeltaX.current = 0
    }

    const onTouchMove = (event: React.TouchEvent) => {
        touchDeltaX.current = event.touches[0].clientX - touchStartX.current
    }

    const onTouchEnd = () => {
        const delta = touchDeltaX.current
        touchDeltaX.current = 0
        if (Math.abs(delta) < SWIPE_THRESHOLD) return
        suppressClickUntil.current = Date.now() + CLICK_SUPPRESS_MS
        if (delta <= -SWIPE_THRESHOLD) goNext()
        else goPrev()
    }

    const onSlideClick = (event: React.MouseEvent<HTMLDivElement>) => {
        if (Date.now() < suppressClickUntil.current) return
        const rect = event.currentTarget.getBoundingClientRect()
        const ratio = (event.clientX - rect.left) / rect.width
        if (ratio < 0.33) goPrev()
        else if (ratio > 0.66) goNext()
    }

    const hasPrev = index > 0
    const hasNext = index < files.length - 1

    return (
        <div ref={rootRef} className={`relative flex flex-col w-full max-w-full min-w-0 min-h-0 ${className}`} data-testid="media-carousel">
            <div
                className="relative overflow-hidden rounded-xl w-full min-w-0 min-h-0 flex-1 flex items-center justify-center"
                onTouchStart={onTouchStart}
                onTouchMove={onTouchMove}
                onTouchEnd={onTouchEnd}
                onClick={onSlideClick}
                style={{ touchAction: 'pan-y' }}
            >
                <div
                    className="flex transition-transform duration-300 ease-out w-full"
                    style={{ transform: `translateX(-${index * 100}%)` }}
                >
                    {files.map((file, slideIndex) => (
                        <div
                            key={`${file.path}-${slideIndex}`}
                            className="w-full min-w-full shrink-0 flex items-center justify-center px-1"
                            aria-hidden={slideIndex !== index}
                        >
                            <CarouselSlide
                                file={file}
                                isActive={slideIndex === index}
                                priority={Math.abs(slideIndex - index) <= 1 ? 'high' : 'low'}
                                variant={variant}
                            />
                        </div>
                    ))}
                </div>

                {hasPrev && (
                    <button
                        type="button"
                        onClick={(event) => {
                            event.stopPropagation()
                            goPrev()
                        }}
                        className="absolute left-2 top-1/2 -translate-y-1/2 z-10 w-10 h-10 rounded-full bg-surface/80 border border-surface-lighter/60 text-text backdrop-blur-sm flex items-center justify-center hover:bg-surface hover:scale-105 transition-all cursor-pointer"
                        aria-label="Önceki"
                        data-testid="carousel-prev"
                    >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
                        </svg>
                    </button>
                )}

                {hasNext && (
                    <button
                        type="button"
                        onClick={(event) => {
                            event.stopPropagation()
                            goNext()
                        }}
                        className="absolute right-2 top-1/2 -translate-y-1/2 z-10 w-10 h-10 rounded-full bg-surface/80 border border-surface-lighter/60 text-text backdrop-blur-sm flex items-center justify-center hover:bg-surface hover:scale-105 transition-all cursor-pointer"
                        aria-label="Sonraki"
                        data-testid="carousel-next"
                    >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                        </svg>
                    </button>
                )}
            </div>

            {files.length > 1 && (
                <div className="mt-3 shrink-0 flex items-center justify-center gap-3 px-2">
                    <span
                        className="text-xs text-text-muted font-medium whitespace-nowrap tabular-nums shrink-0"
                        aria-live="polite"
                        data-testid="carousel-counter"
                    >
                        {index + 1} / {files.length}
                    </span>
                    <div className="flex items-center gap-1.5 overflow-x-auto max-w-[min(70vw,18rem)] py-1 [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden">
                        {files.map((file, dotIndex) => (
                            <button
                                key={`${file.path}-dot-${dotIndex}`}
                                type="button"
                                onClick={() => goTo(dotIndex)}
                                className={`h-1.5 rounded-full transition-all cursor-pointer shrink-0 ${
                                    dotIndex === index ? 'w-5 bg-primary-light' : 'w-1.5 bg-surface-lighter hover:bg-primary/40'
                                }`}
                                aria-label={`${dotIndex + 1}. medyaya git`}
                                data-testid={`carousel-dot-${dotIndex}`}
                            />
                        ))}
                    </div>
                </div>
            )}
        </div>
    )
}
