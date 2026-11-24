import { useEffect, useMemo, useState } from 'react'
import * as api from '../api.js'
import { Crumbs, Empty, dur } from '../ui.jsx'
import { Icon } from '../ui.jsx'
import { useAuth, useLibrary } from '../store.jsx'
import { SongRow } from './Explore.jsx'

export function AddSong() {
  const [msg, setMsg] = useState(null)
  const [busy, setBusy] = useState(false)
  async function submit(e) {
    e.preventDefault()
    const form = e.target
    setBusy(true); setMsg(null)
    try {
      const out = await api.addSong(Object.fromEntries(new FormData(form).entries()))
      setMsg({ ok: true, text: `Added “${out.song.title}” — ${out.fingerprints} fingerprints stored.` })
      form.reset()
    } catch (x) { setMsg({ ok: false, text: x.message }) }
    setBusy(false)
  }
  return (
    <form className="card form" onSubmit={submit}>
      <div className="grid2">
        <input name="title" placeholder="Title *" required />
        <input name="artist" placeholder="Artist *" required />
        <input name="genre" placeholder="Genre" />
        <input name="cover_image_url" placeholder="Cover image URL (optional)" type="url" />
      </div>
      <input name="audio" type="file" accept="audio/*,video/*" required />
      <div className="between">
        <small className="muted">The full track is fingerprinted so clips of it can be recognised.</small>
        <button className="btn primary" disabled={busy}>{busy ? 'Fingerprinting…' : 'Add song'}</button>
      </div>
      {msg && <p className={msg.ok ? 'teal small' : 'error'} role="status">{msg.text}</p>}
    </form>
  )
}

export default function Library() {
  const { stats } = useAuth()
  const { reload } = useLibrary()
  const [hist, setHist] = useState(null)
  useEffect(() => { api.getHistory(200).then(setHist).catch(() => setHist([])); reload().catch(() => {}) }, [reload])

  const recent = useMemo(() => {
    const m = new Map()
    for (const h of hist || []) {
      const e = m.get(h.song.id) || { song: h.song, count: 0 }
      e.count++; m.set(h.song.id, e)
    }
    return [...m.values()].slice(0, 6)
  }, [hist])

  const cards = [['❤️', stats?.saved ?? 0, 'Songs saved'], ['🎵', stats?.identifications ?? 0, 'Tracks identified'], ['💿', stats?.unique_songs ?? 0, 'Unique songs']]
  return (
    <>
      <Crumbs here="Library" />
      <div className="wrap narrow2 section">
        <h1 className="h2s">Your Library</h1>
        <p className="muted">{stats?.collections ?? 0} collections · {recent.length} recently identified</p>
        <div className="stats3">{cards.map(([e, n, l]) => <div key={l} className="card stat"><span className="em">{e}</span><strong>{n}</strong><small>{l}</small></div>)}</div>

        <h5 className="sect gap">Recently Identified</h5>
        {hist === null ? <p className="muted">Loading…</p> : recent.length === 0
          ? <Empty title="Nothing identified yet">Songs you identify show up here. <a href="#/identify">Identify a song</a></Empty>
          : <ul className="card list">{recent.map((r, k) => (
            <SongRow key={r.song.id} song={r.song} right={<span className="muted small times">♫ {r.count}×</span>} />))}</ul>}

        <h5 className="sect gap">Grow the catalog</h5>
        <AddSong />
      </div>
    </>
  )
}
