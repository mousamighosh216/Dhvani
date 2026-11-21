import { useEffect, useState } from 'react'
import * as api from '../api.js'
import { Cover, Crumbs, Empty, Icon, MiniWave, ago, pct } from '../ui.jsx'
import { useLibrary, useUI } from '../store.jsx'

export default function History() {
  const { openSong } = useUI()
  const { isSaved, toggleSave } = useLibrary()
  const [items, setItems] = useState(null)
  const [q, setQ] = useState('')
  useEffect(() => { api.getHistory(200).then(setItems).catch(() => setItems([])) }, [])

  const term = q.trim().toLowerCase()
  const shown = (items || []).filter((h) => !term || `${h.song.title} ${h.song.artist}`.toLowerCase().includes(term))
  return (
    <>
      <Crumbs here="History" />
      <div className="wrap narrow2 section">
        <h1 className="h2s">Your Recognition History</h1>
        <p className="muted">{items ? `${items.length} song${items.length === 1 ? '' : 's'} identified` : 'Loading…'}</p>
        <label className="field wide"><Icon name="search" size={18} />
          <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search history…" aria-label="Search history" /></label>
        {items && shown.length === 0 ? (
          <Empty title={items.length ? 'No matches in your history' : 'Nothing identified yet'}>
            {items.length ? 'Try a different search.' : <a href="#/identify">Identify your first song</a>}</Empty>
        ) : (
          <ul className="hist">{shown.map((h) => (
            <li key={h.id} className="card hrow">
              <Cover song={h.song} size={60} radius={12} />
              <div className="grow"><strong>{h.song.title}</strong><small>{h.song.artist}</small><MiniWave seed={h.song.id} /></div>
              <div className="hmeta"><b className={h.confidence >= 0.95 ? 'gold' : 'teal'}>{pct(h.confidence)}%</b><small>{ago(h.created_at)}</small></div>
              <button className="btn soft sm" onClick={() => openSong(h.song)}>View</button>
              <button className={'icon-btn soft' + (isSaved(h.song.id) ? ' on' : '')} onClick={() => toggleSave(h.song)} aria-label="Save"><Icon name="heart" size={17} fill={isSaved(h.song.id)} /></button>
            </li>))}</ul>
        )}
      </div>
    </>
  )
}
