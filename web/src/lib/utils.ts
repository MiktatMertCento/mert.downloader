export function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function getMediaTypeLabel(type: string): string {
    const labels: Record<string, string> = {
        image: 'Fotoğraf',
        video: 'Video',
        carousel: 'Çoklu Paylaşım',
        reel: 'Reel',
    }
    return labels[type] || type
}
