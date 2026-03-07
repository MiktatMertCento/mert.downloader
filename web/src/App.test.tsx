import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

vi.mock('./lib/api', () => ({
    checkHealth: vi.fn(),
    downloadMedia: vi.fn(),
}))

import { checkHealth } from './lib/api'
import App from './App'

const mockedCheckHealth = checkHealth as ReturnType<typeof vi.fn>

describe('App', () => {
    beforeEach(() => {
        vi.clearAllMocks()
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
})
