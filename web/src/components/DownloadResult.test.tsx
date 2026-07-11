import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import DownloadResult from './DownloadResult'
import MediaCarousel from './MediaCarousel'
import { formatSize, getMediaTypeLabel } from '../lib/utils'
import type { DownloadResponse } from '../lib/api'

vi.mock('../lib/api', async () => {
    const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
    return {
        ...actual,
        startUpscale: vi.fn(),
        waitForUpscale: vi.fn(),
    }
})

import { startUpscale, waitForUpscale } from '../lib/api'

const mockedStartUpscale = vi.mocked(startUpscale)
const mockedWaitForUpscale = vi.mocked(waitForUpscale)

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
        expect(getMediaTypeLabel('highlight')).toBe('Öne Çıkan')
        expect(getMediaTypeLabel('highlight_covers')).toBe('Öne Çıkan Kapaklar')
    })

    it('returns raw type for unknown', () => {
        expect(getMediaTypeLabel('unknown')).toBe('unknown')
    })
})

describe('DownloadResult', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

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

    it('does not render inline carousel in results list', () => {
        const data: DownloadResponse = {
            success: true,
            shortcode: 'CAR123',
            media_type: 'carousel',
            username: 'carouseluser',
            files: [
                { filename: 'photo1.jpg', path: '/downloads/CAR123/photo1.jpg', type: 'image', size: 102400 },
                { filename: 'photo2.jpg', path: '/downloads/CAR123/photo2.jpg', type: 'image', size: 102400 },
                { filename: 'video1.mp4', path: '/downloads/CAR123/video1.mp4', type: 'video', size: 5242880 },
            ],
        }

        render(<DownloadResult data={data} error={null} />)

        expect(screen.queryByTestId('media-carousel')).not.toBeInTheDocument()
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
        expect(screen.queryByTestId('media-carousel')).not.toBeInTheDocument()
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

    it('opens native video preview with mobile-friendly attributes', async () => {
        const user = userEvent.setup()
        const data: DownloadResponse = {
            success: true,
            shortcode: 'VID',
            media_type: 'video',
            username: 'user',
            files: [
                { filename: 'vid.mp4', path: '/downloads/VID/vid.mp4', type: 'video', size: 1024 },
            ],
        }

        render(<DownloadResult data={data} error={null} />)
        await user.click(screen.getByLabelText('Önizle'))

        const video = screen.getByTestId('preview-video') as HTMLVideoElement
        expect(video).toBeInTheDocument()
        expect(video).toHaveAttribute('controls')
        expect(video).not.toHaveAttribute('autoplay')
        expect(video).toHaveAttribute('preload', 'metadata')
        expect(video.getAttribute('playsinline')).not.toBeNull()
        expect(video.getAttribute('webkit-playsinline')).toBe('true')
        expect(video.getAttribute('x5-playsinline')).toBe('true')
    })

    it('closes image preview via the close button even when enhance controls are visible', async () => {
        const user = userEvent.setup()
        const data: DownloadResponse = {
            success: true,
            shortcode: 'IMG',
            media_type: 'image',
            username: 'user',
            files: [{ filename: 'photo.jpg', path: '/downloads/IMG/photo.jpg', type: 'image', size: 1024, width: 1080, height: 1350 }],
        }

        render(<DownloadResult data={data} error={null} />)
        await user.click(screen.getByLabelText('Önizle'))

        expect(screen.getByRole('dialog')).toBeInTheDocument()
        expect(screen.getByTestId('enhance-button')).toBeInTheDocument()

        await user.click(screen.getByTestId('preview-close'))
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('opens fullscreen carousel preview for multi-file downloads', async () => {
        const user = userEvent.setup()
        const data: DownloadResponse = {
            success: true,
            shortcode: 'VID',
            media_type: 'carousel',
            username: 'user',
            files: [
                { filename: 'vid.mp4', path: '/downloads/VID/vid.mp4', type: 'video', size: 1024 },
                { filename: 'photo.jpg', path: '/downloads/VID/photo.jpg', type: 'image', size: 1024 },
            ],
        }

        render(<DownloadResult data={data} error={null} />)
        await user.click(screen.getAllByLabelText('Önizle')[0])

        expect(screen.getByRole('dialog')).toBeInTheDocument()
        expect(screen.getByTestId('media-carousel')).toBeInTheDocument()
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('1 / 2')

        const video = screen.getByTestId('carousel-active-video') as HTMLVideoElement
        expect(video).toHaveAttribute('controls')
        expect(video).toHaveAttribute('preload', 'auto')
    })

    it('enhances the open image with progress and ETA', async () => {
        const user = userEvent.setup()
        const data: DownloadResponse = {
            success: true,
            shortcode: 'IMG',
            media_type: 'image',
            username: 'user',
            files: [{ filename: 'photo.jpg', path: '/downloads/IMG/photo.jpg', type: 'image', size: 1024, width: 540, height: 540 }],
        }

        mockedStartUpscale.mockResolvedValue({
            id: 'job-1',
            status: 'queued',
            source_path: '/downloads/IMG/photo.jpg',
            percent: 0,
            eta_seconds: 20,
            elapsed_seconds: 0,
        })
        mockedWaitForUpscale.mockImplementation(async (_id, onUpdate) => {
            onUpdate?.({
                id: 'job-1',
                status: 'running',
                source_path: '/downloads/IMG/photo.jpg',
                percent: 55,
                eta_seconds: 9,
                elapsed_seconds: 4,
            })
            await new Promise((resolve) => setTimeout(resolve, 30))
            return {
                id: 'job-1',
                status: 'completed',
                source_path: '/downloads/IMG/photo.jpg',
                result_path: '/downloads/IMG/photo_upscaled_x2.png',
                filename: 'photo_upscaled_x2.png',
                percent: 100,
                eta_seconds: 0,
                elapsed_seconds: 8,
                width: 1080,
                height: 1080,
                size: 4096,
            }
        })

        render(<DownloadResult data={data} error={null} />)
        await user.click(screen.getByLabelText('Önizle'))
        expect(screen.getByTestId('enhance-button')).toBeInTheDocument()
        await user.click(screen.getByTestId('enhance-button'))

        await waitFor(() => {
            expect(screen.getByTestId('upscale-progress')).toBeInTheDocument()
            expect(screen.getByTestId('upscale-eta')).toHaveTextContent('Kalan süre ~9 sn')
        })

        await waitFor(() => {
            expect(screen.getByTestId('preview-image')).toHaveAttribute('src', '/downloads/IMG/photo_upscaled_x2.png')
        })
        expect(mockedStartUpscale).toHaveBeenCalledWith('/downloads/IMG/photo.jpg')

        const compare = await screen.findByTestId('compare-button')
        expect(screen.getByTestId('compare-hint')).toHaveTextContent('basılı tut')
        expect(screen.getByTestId('preview-image')).toHaveAttribute('src', '/downloads/IMG/photo_upscaled_x2.png')

        fireEvent.pointerDown(compare)
        expect(screen.getByTestId('preview-image')).toHaveAttribute('src', '/downloads/IMG/photo.jpg')
        expect(screen.getByTestId('compare-hint')).toHaveTextContent('Eski hali gösteriliyor')
        expect(screen.getByRole('dialog')).toBeInTheDocument()

        fireEvent.pointerUp(compare)
        fireEvent.click(compare)
        expect(screen.getByRole('dialog')).toBeInTheDocument()
        expect(screen.getByTestId('preview-image')).toHaveAttribute('src', '/downloads/IMG/photo_upscaled_x2.png')
    })

    it('keeps the current carousel page after enhance succeeds', async () => {
        const user = userEvent.setup()
        const data: DownloadResponse = {
            success: true,
            shortcode: 'CAR',
            media_type: 'carousel',
            username: 'user',
            files: [
                { filename: 'a.jpg', path: '/downloads/CAR/a.jpg', type: 'image', size: 1024 },
                { filename: 'b.jpg', path: '/downloads/CAR/b.jpg', type: 'image', size: 1024 },
                { filename: 'c.jpg', path: '/downloads/CAR/c.jpg', type: 'image', size: 1024 },
            ],
        }

        mockedStartUpscale.mockResolvedValue({
            id: 'job-2',
            status: 'queued',
            source_path: '/downloads/CAR/c.jpg',
            percent: 0,
            eta_seconds: 10,
            elapsed_seconds: 0,
        })
        mockedWaitForUpscale.mockResolvedValue({
            id: 'job-2',
            status: 'completed',
            source_path: '/downloads/CAR/c.jpg',
            result_path: '/downloads/CAR/c_upscaled_x2.png',
            filename: 'c_upscaled_x2.png',
            percent: 100,
            eta_seconds: 0,
            elapsed_seconds: 5,
            width: 200,
            height: 200,
            size: 2048,
        })

        render(<DownloadResult data={data} error={null} />)
        await user.click(screen.getAllByLabelText('Önizle')[0])
        await user.click(screen.getByTestId('carousel-next'))
        await user.click(screen.getByTestId('carousel-next'))
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('3 / 3')

        await user.click(screen.getByTestId('enhance-button'))

        await waitFor(() => {
            expect(screen.getByTestId('carousel-active-image')).toHaveAttribute(
                'src',
                '/downloads/CAR/c_upscaled_x2.png',
            )
        })
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('3 / 3')
        expect(mockedStartUpscale).toHaveBeenCalledWith('/downloads/CAR/c.jpg')
    })
})



describe('MediaCarousel', () => {
    const files = [
        { filename: 'photo1.jpg', path: '/downloads/CAR/photo1.jpg', type: 'image', size: 1024 },
        { filename: 'photo2.jpg', path: '/downloads/CAR/photo2.jpg', type: 'image', size: 1024 },
        { filename: 'video1.mp4', path: '/downloads/CAR/video1.mp4', type: 'video', size: 2048 },
    ]

    it('navigates with next and previous buttons', async () => {
        const user = userEvent.setup()
        render(<MediaCarousel files={files} variant="inline" />)

        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('1 / 3')
        expect(screen.queryByTestId('carousel-prev')).not.toBeInTheDocument()

        await user.click(screen.getByTestId('carousel-next'))
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 / 3')
        expect(screen.getByTestId('carousel-prev')).toBeInTheDocument()

        await user.click(screen.getByTestId('carousel-next'))
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('3 / 3')
        expect(screen.queryByTestId('carousel-next')).not.toBeInTheDocument()

        await user.click(screen.getByTestId('carousel-prev'))
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 / 3')
    })

    it('navigates when clicking right and left sides', () => {
        render(<MediaCarousel files={files} initialIndex={1} />)

        const viewport = screen.getByTestId('media-carousel').querySelector('.overflow-hidden') as HTMLDivElement
        Object.defineProperty(viewport, 'getBoundingClientRect', {
            value: () => ({
                left: 0,
                top: 0,
                right: 300,
                bottom: 200,
                width: 300,
                height: 200,
            }),
        })

        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 / 3')

        fireEvent.click(viewport, { clientX: 250 })
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('3 / 3')

        fireEvent.click(viewport, { clientX: 20 })
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 / 3')
    })

    it('navigates with dot buttons', async () => {
        const user = userEvent.setup()
        render(<MediaCarousel files={files} variant="inline" />)

        await user.click(screen.getByTestId('carousel-dot-2'))
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('3 / 3')
        expect(screen.getByTestId('carousel-active-video')).toBeInTheDocument()
    })

    it('navigates with swipe gestures', () => {
        render(<MediaCarousel files={files} initialIndex={1} />)

        const viewport = screen.getByTestId('media-carousel').querySelector('.overflow-hidden') as HTMLDivElement

        fireEvent.touchStart(viewport, { touches: [{ clientX: 200 }] })
        fireEvent.touchMove(viewport, { touches: [{ clientX: 120 }] })
        fireEvent.touchEnd(viewport)
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('3 / 3')

        fireEvent.touchStart(viewport, { touches: [{ clientX: 120 }] })
        fireEvent.touchMove(viewport, { touches: [{ clientX: 220 }] })
        fireEvent.touchEnd(viewport)
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 / 3')
    })

    it('ignores synthetic clicks after a swipe', () => {
        render(<MediaCarousel files={files} initialIndex={1} />)

        const viewport = screen.getByTestId('media-carousel').querySelector('.overflow-hidden') as HTMLDivElement
        Object.defineProperty(viewport, 'getBoundingClientRect', {
            value: () => ({
                left: 0,
                top: 0,
                right: 300,
                bottom: 200,
                width: 300,
                height: 200,
            }),
        })

        fireEvent.touchStart(viewport, { touches: [{ clientX: 200 }] })
        fireEvent.touchMove(viewport, { touches: [{ clientX: 120 }] })
        fireEvent.touchEnd(viewport)
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('3 / 3')

        fireEvent.click(viewport, { clientX: 20 })
        expect(screen.getByTestId('carousel-counter')).toHaveTextContent('3 / 3')
    })
})
