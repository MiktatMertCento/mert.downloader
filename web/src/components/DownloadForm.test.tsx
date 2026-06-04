import { useState, type ComponentProps } from 'react'
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import DownloadForm from './DownloadForm'

function DownloadFormHarness(props: Omit<ComponentProps<typeof DownloadForm>, 'url' | 'setUrl'> & { initialUrl?: string }) {
    const [url, setUrl] = useState(props.initialUrl ?? '')
    const { initialUrl: _, ...rest } = props
    return <DownloadForm url={url} setUrl={setUrl} {...rest} />
}

describe('DownloadForm', () => {
    it('renders input and button', () => {
        render(<DownloadFormHarness onSubmit={() => { }} isLoading={false} />)

        expect(screen.getByLabelText('Video URL')).toBeInTheDocument()
        expect(screen.getByRole('button', { name: 'İndir' })).toBeInTheDocument()
    })

    it('calls onSubmit with trimmed url', async () => {
        const user = userEvent.setup()
        const onSubmit = vi.fn()
        render(<DownloadFormHarness onSubmit={onSubmit} isLoading={false} />)

        const input = screen.getByLabelText('Video URL')
        await user.type(input, '  https://www.instagram.com/p/test123/  ')
        await user.click(screen.getByRole('button', { name: 'İndir' }))

        expect(onSubmit).toHaveBeenCalledWith('https://www.instagram.com/p/test123/')
    })

    it('does not submit empty url', async () => {
        const user = userEvent.setup()
        const onSubmit = vi.fn()
        render(<DownloadFormHarness onSubmit={onSubmit} isLoading={false} />)

        await user.click(screen.getByRole('button', { name: 'İndir' }))
        expect(onSubmit).not.toHaveBeenCalled()
    })

    it('disables input and button while loading', () => {
        render(<DownloadFormHarness onSubmit={() => { }} isLoading={true} />)

        expect(screen.getByLabelText('Video URL')).toBeDisabled()
        expect(screen.getByRole('button', { name: /İndiriliyor/i })).toBeDisabled()
    })

    it('shows loading spinner when loading', () => {
        render(<DownloadFormHarness onSubmit={() => { }} isLoading={true} />)

        expect(screen.getByText('İndiriliyor')).toBeInTheDocument()
    })

    it('disables button when input is empty', () => {
        render(<DownloadFormHarness onSubmit={() => { }} isLoading={false} />)

        expect(screen.getByRole('button', { name: 'İndir' })).toBeDisabled()
    })
})
