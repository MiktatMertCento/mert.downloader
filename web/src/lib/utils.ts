export function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function formatEta(seconds: number): string {
    const safe = Math.max(0, Math.round(seconds))
    if (safe < 60) return `~${safe} sn`
    const minutes = Math.floor(safe / 60)
    const rem = safe % 60
    if (minutes < 60) {
        return rem > 0 ? `~${minutes} dk ${rem} sn` : `~${minutes} dk`
    }
    const hours = Math.floor(minutes / 60)
    const mins = minutes % 60
    return mins > 0 ? `~${hours} sa ${mins} dk` : `~${hours} sa`
}

export function isSmallPreviewMedia(file: { width?: number; height?: number }): boolean {
    if (!file.width || !file.height) return false
    return Math.max(file.width, file.height) <= 360
}

export function getPreviewMediaClass(
    file: { type: string; width?: number; height?: number },
    variant: 'fullscreen' | 'inline' = 'fullscreen',
): string {
    const small = file.type === 'image' && isSmallPreviewMedia(file)
    const shared = 'rounded-xl object-contain select-none'

    if (variant === 'inline') {
        return `${shared} w-full max-h-[min(70dvh,520px)] bg-surface-light/50`
    }

    if (small) {
        return `${shared} w-[min(72vw,28rem)] h-[min(72vw,28rem)] max-w-[90vw] max-h-[min(85dvh,100%)] ring-1 ring-surface-lighter bg-surface-light/50`
    }

    if (file.type === 'video') {
        return `${shared} max-w-full max-h-[min(85dvh,100%)] w-full ring-1 ring-surface-lighter bg-black`
    }

    return `${shared} max-w-full max-h-[min(85dvh,100%)] w-full ring-1 ring-surface-lighter bg-surface-light/50`
}

export function applyMobileVideoAttributes(video: HTMLVideoElement) {
    video.setAttribute('playsinline', '')
    video.setAttribute('webkit-playsinline', 'true')
    video.setAttribute('x5-playsinline', 'true')
    video.setAttribute('x5-video-player-type', 'h5')
}

export function getMediaTypeLabel(type: string): string {
    const labels: Record<string, string> = {
        image: 'Fotoğraf',
        video: 'Video',
        carousel: 'Çoklu Paylaşım',
        reel: 'Reel',
        story: 'Hikaye',
        highlight: 'Öne Çıkan',
        highlight_covers: 'Öne Çıkan Kapaklar',
    }
    return labels[type] || type
}

function cleanMatchedUrl(raw: string): string {
    return raw.replace(/[.,;:!?)\]}>]+$/g, '')
}

export function extractSharedUrl(title: string | null, text: string | null, urlParam: string | null): string | null {
    if (urlParam) {
        const direct = urlParam.match(/https?:\/\/[^\s]+/i)
        if (direct) {
            return cleanMatchedUrl(direct[0])
        }
    }

    const combined = [title, text, urlParam].filter(Boolean).join(' ')
    const match = combined.match(/https?:\/\/[^\s]+/i)
    return match ? cleanMatchedUrl(match[0]) : null
}
