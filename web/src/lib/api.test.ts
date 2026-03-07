import { describe, it, expect, vi, beforeEach } from 'vitest'
import axios from 'axios'
import { checkHealth, downloadMedia } from './api'

vi.mock('axios', async () => {
    const mockAxios = {
        create: vi.fn(() => mockAxios),
        get: vi.fn(),
        post: vi.fn(),
        isAxiosError: vi.fn(),
    }
    return { default: mockAxios }
})

const mockedAxios = axios as unknown as {
    create: ReturnType<typeof vi.fn>
    get: ReturnType<typeof vi.fn>
    post: ReturnType<typeof vi.fn>
    isAxiosError: ReturnType<typeof vi.fn>
}

describe('checkHealth', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('returns health data on success', async () => {
        const mockData = { status: 'ok', user_id: '12345' }
        mockedAxios.get.mockResolvedValue({ data: mockData })

        const result = await checkHealth()
        expect(result).toEqual(mockData)
        expect(mockedAxios.get).toHaveBeenCalledWith('/api/health')
    })

    it('throws on network error', async () => {
        mockedAxios.get.mockRejectedValue(new Error('Network Error'))

        await expect(checkHealth()).rejects.toThrow('Network Error')
    })
})

describe('downloadMedia', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('sends POST with url and returns data', async () => {
        const mockResponse = {
            success: true,
            shortcode: 'ABC123',
            media_type: 'image',
            username: 'testuser',
            files: [{ filename: 'test.jpg', path: '/downloads/test.jpg', type: 'image', size: 1024 }],
        }

        mockedAxios.post.mockResolvedValue({ data: mockResponse })

        const result = await downloadMedia('https://www.instagram.com/p/ABC123/')
        expect(result).toEqual(mockResponse)
        expect(mockedAxios.post).toHaveBeenCalledWith('/api/download', {
            url: 'https://www.instagram.com/p/ABC123/',
        })
    })

    it('throws when success is false', async () => {
        mockedAxios.post.mockResolvedValue({
            data: { success: false, error: 'URL boş' },
        })

        await expect(downloadMedia('')).rejects.toThrow('URL boş')
    })

    it('throws with API error message from axios error response', async () => {
        const axiosError = {
            response: { data: { error: 'Medya bulunamadı' } },
        }
        mockedAxios.post.mockRejectedValue(axiosError)
        mockedAxios.isAxiosError.mockReturnValue(true)

        await expect(downloadMedia('https://example.com')).rejects.toThrow('Medya bulunamadı')
    })

    it('rethrows non-axios errors', async () => {
        mockedAxios.post.mockRejectedValue(new Error('Unexpected'))
        mockedAxios.isAxiosError.mockReturnValue(false)

        await expect(downloadMedia('https://example.com')).rejects.toThrow('Unexpected')
    })
})
