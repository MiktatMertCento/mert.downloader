import { describe, it, expect } from 'vitest'
import { extractSharedUrl, getPreviewMediaClass, isSmallPreviewMedia } from './utils'

describe('extractSharedUrl', () => {
    it('extracts url from share target url param', () => {
        expect(extractSharedUrl(null, null, 'https://www.instagram.com/p/ABC123/')).toBe(
            'https://www.instagram.com/p/ABC123/',
        )
    })

    it('extracts url from shared text', () => {
        expect(
            extractSharedUrl('Instagram', 'Check this https://www.youtube.com/watch?v=dQw4w9WgXcQ', null),
        ).toBe('https://www.youtube.com/watch?v=dQw4w9WgXcQ')
    })

    it('prefers direct url param over text', () => {
        expect(
            extractSharedUrl('ignored', 'https://www.youtube.com/watch?v=other', 'https://www.instagram.com/reel/XYZ/'),
        ).toBe('https://www.instagram.com/reel/XYZ/')
    })

    it('returns null when no url found', () => {
        expect(extractSharedUrl('hello', 'world', null)).toBeNull()
    })

    it('strips trailing punctuation from shared urls', () => {
        expect(extractSharedUrl(null, 'Bak: https://www.instagram.com/p/ABC123/.', null)).toBe(
            'https://www.instagram.com/p/ABC123/',
        )
    })
})

describe('isSmallPreviewMedia', () => {
    it('detects small highlight covers', () => {
        expect(isSmallPreviewMedia({ width: 150, height: 150 })).toBe(true)
    })

    it('ignores large images', () => {
        expect(isSmallPreviewMedia({ width: 1080, height: 1920 })).toBe(false)
    })
})

describe('getPreviewMediaClass', () => {
    it('upscales small images in fullscreen preview', () => {
        const className = getPreviewMediaClass({ type: 'image', width: 150, height: 150 }, 'fullscreen')
        expect(className).toContain('28rem')
        expect(className).toContain('72vw')
    })
})
