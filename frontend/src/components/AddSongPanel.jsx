import { useCallback, useEffect, useState } from 'react'
import { ingestSong, listSongs } from '../services/apiClient.js'

const EMPTY = { title: '', artist: '', album: '', release_date: '', cover_art_url: '', spotify: '', youtube: '' }

export default function AddSongPanel({ onClose }) {
  const [fields, setFields] = useState(EMPTY)
  const [file, setFile] = useState(null)
  const [status, setStatus] = useState(null) // {type:'ok'|'error'|'busy', text}
  const [songs, setSongs] = useState([])

  const refresh = useCallback(() => listSongs().then(setSongs).catch(() => setSongs([])), [])
  useEffect(() => { refresh() }, [refresh])

  const set = (k) => (e) => setFields((f) => ({ ...f, [k]: e.target.value }))

  const submit = async (e) => {
    e.preventDefault()
    if (!file) return setStatus({ type: 'error', text: 'Choose an audio file first.' })
    setStatus({ type: 'busy', text: 'Fingerprinting… full tracks can take a few seconds.' })
    try {
      const data = await ingestSong(file, fields)
      setStatus({ type: 'ok', text: `Added “${data.song.title}” with ${data.fingerprints.toLocaleString()} fingerprints.` })
      setFields(EMPTY)
      setFile(null)
      e.target.reset()
      refresh()
    } catch (err) {
      setStatus({ type: 'error', text: err.message })
    }
  }

  return (
    <aside className="panel" aria-label="Catalog">
      <header>
        <h2>Catalog</h2>
        <button type="button" className="close" onClick={onClose} aria-label="Close catalog">×</button>
      </header>

      <form onSubmit={submit}>
        <label>Audio file *<input type="file" accept="audio/*,.wav,.mp3" onChange={(e) => setFile(e.target.files[0] || null)} /></label>
        <label>Title *<input required value={fields.title} onChange={set('title')} /></label>
        <label>Artist *<input required value={fields.artist} onChange={set('artist')} /></label>
        <label>Album<input value={fields.album} onChange={set('album')} /></label>
        <div className="row">
          <label>Release date<input type="date" value={fields.release_date} onChange={set('release_date')} /></label>
          <label>Cover art URL<input type="url" value={fields.cover_art_url} onChange={set('cover_art_url')} placeholder="https://" /></label>
        </div>
        <label>Spotify link<input type="url" value={fields.spotify} onChange={set('spotify')} placeholder="https://" /></label>
        <label>YouTube link<input type="url" value={fields.youtube} onChange={set('youtube')} placeholder="https://" /></label>
        <button type="submit" className="primary" disabled={status?.type === 'busy'}>Add to catalog</button>
        {status && <p className={`status ${status.type}`} role="status">{status.text}</p>}
      </form>

      <h3>Songs ({songs.length})</h3>
      <ul className="song-list">
        {songs.length === 0 && <li className="empty">No songs yet. Add one so Dhwani has something to recognize.</li>}
        {songs.map((s) => (
          <li key={s.id}><strong>{s.title}</strong><span>{s.artist?.name} · {s.duration_seconds}s</span></li>
        ))}
      </ul>
    </aside>
  )
}
