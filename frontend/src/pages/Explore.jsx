import { useEffect, useState } from 'react'
import * as api from '../api.js'
import { Cover, Crumbs, Empty, Icon, compact, dur } from '../ui.jsx'
import { go, useAuth, useLibrary, useUI } from '../store.jsx'

const EMOJI = [['hip', '🎧'], ['indie', '🌿'], ['pop', '🎤'], ['electr', '🎛️'], ['r&b', '🎸'], ['soul', '🎸'], ['jazz', '🎷'], ['classic', '🎻'], ['lo-fi', '☁️'], ['lofi', '☁️'], ['rock', '🤘'], ['synth', '🎹'], ['dance', '🪩']]
const emoji = (g) => (EMOJI.find(([k]) => g.toLowerCase().includes(k)) || [0, '🎵'])[1]

export function SongRow({ song, right }) {
  const { openSong } = useUI()
  const { isSaved, toggleSave } = useLibrary()
  return (
    <li className="li static">
      <button className="li-main" onClick={() => openSong(song)}>
        <Cover song={song} size={52} radius={12} />
        <span className="grow"><strong>{song.title}</strong><small>{song.artist}{song.genre ? ` · ${song.genre}` : ''}</small></span>
        <span className="muted small">{dur(song.duration_seconds)}</span>
      </button>
      {right ?? <button className={'icon-btn soft' + (isSaved(song.id) ? ' on' : '')} onClick={() => toggleSave(song)} aria-label="Save"><Icon name="heart" size={17} fill={isSaved(song.id)} /></button>}
    </li>
  )
}

export default function Explore() {
  const { user } = useAuth()
  const [q, setQ] = useState('')
  const [genre, setGenre] = useState('')
  const [genres, setGenres] = useState([])
  const [songs, setSongs] = useState(null)
  const term = q.trim()

  useEffect(() => { api.getGenres().then(setGenres).catch(() => {}) }, [])
  useEffect(() => {
    let live = true
    const t = setTimeout(() => {
      api.getSongs({ q: term, genre, limit: 100, sort: term || genre ? 'artist' : '' })
        .then((s) => live && setSongs(s)).catch(() => live && setSongs([]))
    }, term ? 220 : 0)
    return () => { live = false; clearTimeout(t) }
  }, [term, genre])

  const title = term ? `Results for “${term}”` : genre || 'All songs'
  return (
    <>
      <Crumbs here="Explore" />
      <section className="band">
        <div className="wrap">
          <span className="eyebrow">MUSIC DISCOVERY</span>
          <h1 className="h1s">Explore the world<br />of sound.</h1>
          <p className="muted lead2">Browse by genre, discover new releases, and find music that matches your taste.</p>
          <div className="find">
            <label className="field"><Icon name="search" size={18} />
              <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search songs, artists, genres…" aria-label="Search" /></label>
            <button className="btn primary" onClick={() => go('/identify')}><Icon name="mic" size={18} /> Identify a Song</button>
          </div>
        </div>
      </section>
      <div className="wrap section">
        <div className="between"><h5 className="sect">Browse by Genre</h5>
          {genre && <button className="link" onClick={() => setGenre('')}>See all →</button>}</div>
        {genres.length === 0 ? <p className="muted">Genres appear once songs with a genre are added.</p> : (
          <div className="genres">{genres.map((g, i) => (
            <button key={g.genre} className={`gtile t${i % 8}` + (genre.toLowerCase() === g.genre.toLowerCase() ? ' on' : '')}
              onClick={() => setGenre(genre === g.genre ? '' : g.genre)}>
              <span className="em">{emoji(g.genre)}</span><strong>{g.genre}</strong><small>{compact(g.count)} song{g.count === 1 ? '' : 's'}</small>
            </button>))}</div>
        )}
        <h5 className="sect gap">{title}</h5>
        {songs === null ? <p className="muted">Loading…</p> : songs.length === 0
          ? <Empty title="No songs found">Try a different search, or <a href={user ? '#/upload' : '#/auth?next=/upload'}>upload a song</a>.</Empty>
          : <ul className="card list">{songs.map((s) => <SongRow key={s.id} song={s} />)}</ul>}
      </div>
    </>
  )
}
