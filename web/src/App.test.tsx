import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

vi.mock('./lib/api', () => ({
    checkHealth: vi.fn(),
    downloadMedia: vi.fn(),
}))

import { checkHealth, downloadMedia } from './lib/api'
import App from './App'

const mockedCheckHealth = checkHealth as ReturnType<typeof vi.fn>
const mockedDownloadMedia = downloadMedia as ReturnType<typeof vi.fn>

describe('App', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        window.history.replaceState({}, '', '/')
    })

    afterEach(() => {
        window.history.replaceState({}, '', '/')
    })

    it('renders heading and form', async () => {
        mockedCheckHealth.mockResolvedValue({ status: 'ok', user_id: '123' })

        render(<App />)

        expect(screen.getByText('Mert')).toBeInTheDocument()
        expect(screen.getByText('Downloader')).toBeInTheDocument()
        expect(screen.getByLabelText('Video URL')).toBeInTheDocument()
    })

    it('shows online status on successful health check', async () => {
        mockedCheckHealth.mockResolvedValue({ status: 'ok', user_id: '123' })

        render(<App />)

        await waitFor(() => {
            expect(screen.getByTestId('server-status')).toHaveTextContent('Bağlı')
        })
    })

    it('shows offline status on failed health check', async () => {
        mockedCheckHealth.mockRejectedValue(new Error('Network error'))

        render(<App />)

        await waitFor(() => {
            expect(screen.getByTestId('server-status')).toHaveTextContent('Bağlantı yok')
        })
    })

    it('auto-submits when opened via share target', async () => {
        window.history.pushState({}, '', '/?url=https%3A%2F%2Fwww.instagram.com%2Fp%2FABC123%2F')
        mockedCheckHealth.mockResolvedValue({ status: 'ok', user_id: '123' })
        mockedDownloadMedia.mockResolvedValue({
            success: true,
            shortcode: 'ABC123',
            media_type: 'image',
            username: 'testuser',
            files: [],
        })

        render(<App />)

        expect(screen.getByLabelText('Video URL')).toHaveValue('https://www.instagram.com/p/ABC123/')

        await waitFor(() => {
            expect(mockedDownloadMedia).toHaveBeenCalledWith('https://www.instagram.com/p/ABC123/')
        })

        await waitFor(() => {
            expect(screen.getByText('İndirme Başarılı')).toBeInTheDocument()
        })
    })
})
