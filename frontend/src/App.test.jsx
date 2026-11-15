import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.jsx'
import ResultCard from './components/ResultCard.jsx'
import { ingestSong, recognizeAudio } from './services/apiClient.js'

const song = {
  id: 7,
  title: 'Midnight City',
  duration_seconds: 243,
  artist: { id: 1, name: 'M83' },
  album: { id: 2, title: "Hurry Up, We're Dreaming", release_date: '2011-10-18' },
  external_links: { spotify: 'https://open.spotify.com/track/x', youtube: 'https://youtu.be/y' },
}

function mockFetch(handler) {
  globalThis.fetch = vi.fn(async (url, opts) => {
    const { status = 200, body } = handler(url, opts)
    return { ok: status < 400, status, json: async () => body }
  })
}

beforeEach(() => mockFetch((url) => (url.endsWith('/songs') ? { body: { status: 'success', data: { songs: [] } } } : { body: {} })))
afterEach(() => vi.restoreAllMocks())

describe('ResultCard', () => {
  it('shows song details, confidence, offset and links', () => {
    render(<ResultCard result={{ song, confidence: 0.42, votes: 31, match_offset_seconds: 65 }} />)
    expect(screen.getByText('Midnight City')).toBeInTheDocument()
    expect(screen.getByText('M83')).toBeInTheDocument()
    expect(screen.getByText(/2011/)).toBeInTheDocument()
    expect(screen.getByText(/42% match · 31 landmarks/)).toBeInTheDocument()
    expect(screen.getByText(/starts at 1:05 of 4:03/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Spotify' })).toHaveAttribute('href', song.external_links.spotify)
    expect(screen.getByRole('link', { name: 'YouTube' })).toBeInTheDocument()
  })

  it('works without album, cover or links', () => {
    render(<ResultCard result={{ song: { title: 'Bare', artist: { name: 'X' }, duration_seconds: 10 }, confidence: 0.1 }} />)
    expect(screen.getByText('Bare')).toBeInTheDocument()
    expect(screen.queryByRole('link')).toBeNull()
  })
})

describe('apiClient', () => {
  it('posts the clip as multipart "audio" and unwraps data', async () => {
    mockFetch(() => ({ body: { status: 'success', data: { matched: false } } }))
    const data = await recognizeAudio(new Blob(['x']), 'a.webm')
    expect(data).toEqual({ matched: false })
    const [url, opts] = fetch.mock.calls[0]
    expect(url).toBe('/api/v1/recognize')
    expect(opts.method).toBe('POST')
    expect(opts.body.get('audio').name).toBe('a.webm')
  })

  it('throws the server message on errors', async () => {
    mockFetch(() => ({ status: 400, body: { status: 'error', message: 'audio clip is too short' } }))
    await expect(recognizeAudio(new Blob(['x']))).rejects.toThrow('audio clip is too short')
  })

  it('sends only non-empty metadata fields when ingesting', async () => {
    mockFetch(() => ({ body: { status: 'success', data: {} } }))
    await ingestSong(new File(['x'], 's.wav'), { title: 'T', artist: 'A', album: '' })
    const form = fetch.mock.calls[0][1].body
    expect(form.get('title')).toBe('T')
    expect(form.has('album')).toBe(false)
  })
})

describe('Chat flow', () => {
  const upload = () => {
    const file = new File(['fake'], 'clip.wav', { type: 'audio/wav' })
    fireEvent.change(screen.getByTestId('file-input'), { target: { files: [file] } })
  }

  it('uploads a file and renders the identified song', async () => {
    mockFetch(() => ({ body: { status: 'success', data: { matched: true, confidence: 0.5, votes: 40, match_offset_seconds: 12.3, song } } }))
    render(<App />)
    upload()
    expect(await screen.findByText('Midnight City')).toBeInTheDocument()
    expect(screen.getByText('clip.wav')).toBeInTheDocument()
    expect(screen.getByText(/50% match/)).toBeInTheDocument()
  })

  it('explains when nothing matched', async () => {
    mockFetch(() => ({ body: { status: 'success', data: { matched: false } } }))
    render(<App />)
    upload()
    expect(await screen.findByText(/couldn.t find a match/i)).toBeInTheDocument()
  })

  it('shows server errors in the chat', async () => {
    mockFetch(() => ({ status: 400, body: { status: 'error', message: 'could not decode audio' } }))
    render(<App />)
    upload()
    expect(await screen.findByText('could not decode audio')).toBeInTheDocument()
  })

  it('disables inputs while identifying', async () => {
    let release
    globalThis.fetch = vi.fn(() => new Promise((r) => { release = r }))
    render(<App />)
    upload()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Upload audio' })).toBeDisabled())
    release({ ok: true, status: 200, json: async () => ({ status: 'success', data: { matched: false } }) })
    await waitFor(() => expect(screen.getByRole('button', { name: 'Upload audio' })).toBeEnabled())
  })
})

describe('Catalog panel', () => {
  it('opens, lists songs and validates the file before ingesting', async () => {
    mockFetch(() => ({ body: { status: 'success', data: { songs: [{ id: 1, title: 'Alpha', artist: { name: 'Band' }, duration_seconds: 30 }] } } }))
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'Catalog' }))
    expect(await screen.findByText('Alpha')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText(/Title/), { target: { value: 'T' } })
    fireEvent.change(screen.getByLabelText(/Artist/), { target: { value: 'A' } })
    fireEvent.click(screen.getByRole('button', { name: 'Add to catalog' }))
    expect(await screen.findByText('Choose an audio file first.')).toBeInTheDocument()
  })
})
