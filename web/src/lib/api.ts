import axios from 'axios'

export interface DownloadedFile {
    filename: string
    path: string
    type: string
    size: number
    width?: number
    height?: number
    /** Previous version path after 2x enhance — used for hold-to-compare. */
    originalPath?: string
    originalWidth?: number
    originalHeight?: number
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
    upscale_ready?: boolean
}

export type UpscaleJobStatus = 'queued' | 'running' | 'completed' | 'failed'

export interface UpscaleJob {
    id: string
    status: UpscaleJobStatus
    source_path: string
    result_path?: string
    filename?: string
    width?: number
    height?: number
    size?: number
    percent: number
    eta_seconds: number
    elapsed_seconds: number
    error?: string
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

export async function startUpscale(path: string): Promise<UpscaleJob> {
    try {
        const { data } = await api.post<UpscaleJob>('/api/upscale', { path })
        return data
    } catch (err) {
        if (axios.isAxiosError(err) && err.response?.data?.error) {
            throw new Error(err.response.data.error)
        }
        throw err
    }
}

export async function getUpscaleJob(id: string): Promise<UpscaleJob> {
    try {
        const { data } = await api.get<UpscaleJob>(`/api/upscale/${id}`)
        return data
    } catch (err) {
        if (axios.isAxiosError(err) && err.response?.data?.error) {
            throw new Error(err.response.data.error)
        }
        throw err
    }
}

export async function waitForUpscale(
    id: string,
    onUpdate?: (job: UpscaleJob) => void,
    signal?: AbortSignal,
): Promise<UpscaleJob> {
    for (;;) {
        if (signal?.aborted) {
            throw new Error('Upscale iptal edildi')
        }
        const job = await getUpscaleJob(id)
        onUpdate?.(job)
        if (job.status === 'completed' || job.status === 'failed') {
            return job
        }
        await new Promise((resolve) => setTimeout(resolve, 500))
    }
}
