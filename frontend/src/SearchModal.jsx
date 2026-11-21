import { useEffect, useState } from 'react'
import * as api from './api.js'
import { Cover, Icon, Modal, dur } from './ui.jsx'
import { go, useLibrary, useUI } from './store.jsx'

export function SearchModal({ onClose }) {
  const { openSong } = useUI()
  const [q, setQ] = useState('')
  const [songs, setSongs] = useState(null)
  const [genres, setGenres] = useState([])
  const term = q.trim()

  useEffect(() => { api.getGenres().then((g) => setGenres(g.slice(0, 3))).catch(() => {}) }, [])
  useEffect(() => {
    let live = true
    const t = setTimeout(() => {
      api.getSongs({ q: term, sort: term ? '' : 'popular', limit: 6 }).then((s) => live && setSongs(s)).catch(() => live && setSongs([]))
    }, term ? 220 : 0)
    return () => { live = false; clearTimeout(t) }
  }, [term])

  const explore = () => { onClose(); go('/explore') }
  return (
    <Modal onClose={onClose} label="Search Dhvani">
      <div className="sm-head">
        <div><h3>Search DHVANI</h3><p>Find songs, artists, and genres</p></div>
        <button className="icon-btn soft" onClick={onClose} aria-label="Close"><Icon name="close" size={18} /></button>
      </div>
      <form className="sm-input" onSubmit={(e) => { e.preventDefault(); explore() }}>
        <Icon name="search" size={18} />
        <input autoFocus value={q} onChange={(e) => setQ(e.target.value)} placeholder="What do you want to hear?" />
        <button className="go" aria-label="Search"><Icon name="right" size={16} /></button>
      </form>
      {genres.length > 0 && (
        <div className="chips">{genres.map((g) => <button key={g.genre} className="chip" onClick={() => setQ(g.genre)}>{g.genre}</button>)}</div>
      )}
      <div className="sm-body">
        <h6>{term ? 'RESULTS' : 'POPULAR RIGHT NOW'}</h6>
        {songs === null ? <p className="muted pad">Loading…</p> : songs.length === 0 ? (
          <p className="muted pad">{term ? `Nothing found for “${term}”.` : 'No songs in the catalog yet.'}</p>
        ) : songs.map((s) => (
          <button key={s.id} className="sm-row" onClick={() => { onClose(); openSong(s) }}>
            <span className="tile"><Icon name="music" size={20} /></span>
            <span className="grow"><strong>{s.title}</strong><small>{s.artist}{s.genre ? ` · ${s.genre}` : ''}</small></span>
            <span className="muted small">{dur(s.duration_seconds)}</span>
            <Icon name="right" size={16} className="muted" />
          </button>
        ))}
      </div>
      <div className="sm-foot"><span>Press Esc to close</span><button onClick={explore}>Explore all</button></div>
    </Modal>
  )
}

export function SongModal({ song, onClose }) {
  const { isSaved, toggleSave } = useLibrary()
  const saved = isSaved(song.id)
  return (
    <Modal onClose={onClose} label={song.title}>
      <div className="song-modal">
        <Cover song={song} size={140} radius={18} />
        <h3>{song.title}</h3>
        <p>{song.artist}</p>
        <p className="muted small">{song.genre ? `${song.genre} · ` : ''}{dur(song.duration_seconds)}</p>
        <div className="row-actions">
          <button className={'btn ghost' + (saved ? ' on' : '')} onClick={() => toggleSave(song)}>
            <Icon name="heart" size={16} fill={saved} /> {saved ? 'Saved' : 'Save'}
          </button>
          <button className="btn ghost" onClick={onClose}>Close</button>
        </div>
      </div>
    </Modal>
  )
}
