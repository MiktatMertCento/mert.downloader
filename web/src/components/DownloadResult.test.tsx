import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import DownloadResult from './DownloadResult'
import { formatSize, getMediaTypeLabel } from '../lib/utils'
import type { DownloadResponse } from '../lib/api'

describe('formatSize', () => {
    it('formats bytes', () => {
        expect(formatSize(500)).toBe('500 B')
    })

    it('formats kilobytes', () => {
        expect(formatSize(2048)).toBe('2.0 KB')
    })

    it('formats megabytes', () => {
        expect(formatSize(5 * 1024 * 1024)).toBe('5.0 MB')
    })
})

describe('getMediaTypeLabel', () => {
    it('returns Turkish labels', () => {
        expect(getMediaTypeLabel('image')).toBe('Fotoğraf')
        expect(getMediaTypeLabel('video')).toBe('Video')
        expect(getMediaTypeLabel('carousel')).toBe('Çoklu Paylaşım')
        expect(getMediaTypeLabel('reel')).toBe('Reel')
    })

    it('returns raw type for unknown', () => {
        expect(getMediaTypeLabel('unknown')).toBe('unknown')
    })
})

describe('DownloadResult', () => {
    it('renders nothing when no data and no error', () => {
        const { container } = render(<DownloadResult data={null} error={null} />)
        expect(container.firstChild).toBeNull()
    })

    it('renders error message', () => {
        render(<DownloadResult data={null} error="URL boş" />)

        expect(screen.getByRole('alert')).toHaveTextContent('URL boş')
        expect(document.getElementById('error-result')).toBeInTheDocument()
    })

    it('renders success with files', () => {
        const data: DownloadResponse = {
            success: true,
            shortcode: 'ABC123',
            media_type: 'image',
            username: 'testuser',
            caption: 'Test caption',
            files: [
                { filename: 'photo1.jpg', path: '/downloads/ABC123/photo1.jpg', type: 'image', size: 102400 },
                { filename: 'video1.mp4', path: '/downloads/ABC123/video1.mp4', type: 'video', size: 5242880 },
            ],
        }

        render(<DownloadResult data={data} error={null} />)

        expect(screen.getByText('İndirme Başarılı')).toBeInTheDocument()
        expect(screen.getByText('@testuser')).toBeInTheDocument()
        expect(screen.getByText('Fotoğraf')).toBeInTheDocument()
        expect(screen.getByText('2 dosya')).toBeInTheDocument()
        expect(screen.getByText('Test caption')).toBeInTheDocument()
        expect(screen.getByText('photo1.jpg')).toBeInTheDocument()
        expect(screen.getByText('video1.mp4')).toBeInTheDocument()
    })

    it('renders without username and caption', () => {
        const data: DownloadResponse = {
            success: true,
            shortcode: 'XYZ',
            media_type: 'video',
            username: '',
            files: [
                { filename: 'vid.mp4', path: '/downloads/XYZ/vid.mp4', type: 'video', size: 1024 },
            ],
        }

        render(<DownloadResult data={data} error={null} />)

        expect(screen.getByText('İndirme Başarılı')).toBeInTheDocument()
        expect(screen.queryByText('@')).not.toBeInTheDocument()
        expect(screen.getByText('vid.mp4')).toBeInTheDocument()
    })

    it('shows file dimensions when available', () => {
        const data: DownloadResponse = {
            success: true,
            shortcode: 'DIM',
            media_type: 'image',
            username: 'user',
            files: [
                { filename: 'img.jpg', path: '/downloads/DIM/img.jpg', type: 'image', size: 2048, width: 1080, height: 1920 },
            ],
        }

        render(<DownloadResult data={data} error={null} />)

        expect(screen.getByText(/1080×1920/)).toBeInTheDocument()
    })
})
