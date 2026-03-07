import axios from 'axios'

export interface DownloadedFile {
    filename: string
    path: string
    type: string
    size: number
    width?: number
    height?: number
}

export interface DownloadResponse {
    success: boolean
    shortcode: string
    media_type: string
    username: string
    caption?: string
    files: DownloadedFile[]
    error?: string
}

export interface HealthResponse {
    status: string
    user_id: string
}

const api = axios.create({
    headers: { 'Content-Type': 'application/json' },
})

export async function checkHealth(): Promise<HealthResponse> {
    const { data } = await api.get<HealthResponse>('/api/health')
    return data
}

export async function downloadMedia(url: string): Promise<DownloadResponse> {
    let response
    try {
        response = await api.post<DownloadResponse>('/api/download', { url })
    } catch (err) {
        if (axios.isAxiosError(err) && err.response?.data?.error) {
            throw new Error(err.response.data.error)
        }
        throw err
    }

    const { data } = response
    if (!data.success) {
        throw new Error(data.error || 'İndirme başarısız')
    }

    return data
}
