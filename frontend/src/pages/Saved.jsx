import { useCallback, useEffect, useMemo, useState } from 'react'
import * as api from '../api.js'
import { Cover, Crumbs, Empty, Icon, Modal, ago, pct } from '../ui.jsx'
import { useLibrary, useUI } from '../store.jsx'

const WEEK = 7 * 864e5

function NewCollection({ onClose, onCreate }) {
  const [name, setName] = useState('')
  const [err, setErr] = useState('')
  async function submit(e) {
    e.preventDefault()
    try { await onCreate(name.trim()); onClose() } catch (x) { setErr(x.message) }
  }
  return (
    <Modal onClose={onClose} label="New collection">
      <form className="song-modal" onSubmit={submit}>
        <h3>New collection</h3>
        <input autoFocus value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Party Vibes" maxLength={60} required />
        {err && <p className="error">{err}</p>}
        <div className="row-actions"><button className="btn primary">Create</button><button type="button" className="btn ghost" onClick={onClose}>Cancel</button></div>
      </form>
    </Modal>
  )
}

export default function Saved() {
  const { openSong } = useUI()
  const { collections, reload, toggleSave, isSaved } = useLibrary()
  const [sel, setSel] = useState('all') // 'all' | 'week' | collection id
  const [sort, setSort] = useState('date')
  const [rows, setRows] = useState(null)
  const [week, setWeek] = useState([])
  const [creating, setCreating] = useState(false)

  const load = useCallback(async () => {
    const hist = await api.getHistory(200).catch(() => [])
    const seen = new Set(), wk = []
    for (const h of hist) if (Date.now() - new Date(h.created_at) < WEEK && !seen.has(h.song.id)) { seen.add(h.song.id); wk.push({ song: h.song, saved_at: h.created_at, confidence: h.confidence }) }
    setWeek(wk)
    setRows(sel === 'week' ? wk : await api.getCollectionSongs(sel).catch(() => []))
  }, [sel])
  useEffect(() => { load() }, [load])

  async function remove(song) {
    if (sel === 'week') return toggleSave(song)
    await api.unsaveSong(sel === 'all' ? 0 : sel, song.id).catch(() => {})
    await Promise.all([load(), reload()])
  }

  const list = useMemo(() => {
    const r = [...(rows || [])]
    if (sort === 'confidence') r.sort((a, b) => b.confidence - a.confidence)
    else if (sort === 'artist') r.sort((a, b) => a.song.artist.localeCompare(b.song.artist) || a.song.title.localeCompare(b.song.title))
    else r.sort((a, b) => new Date(b.saved_at) - new Date(a.saved_at))
    return r
  }, [rows, sort])

  const total = collections.reduce((n, c) => n + c.count, 0)
  return (
    <>
      <Crumbs here="Saved Matches" />
      <div className="wrap narrow2 section">
        <h1 className="h2s">Saved Matches</h1>
        <p className="muted">{sel === 'all' ? `${rows ? rows.length : total} songs saved` : `${list.length} songs`}</p>

        <h5 className="sect gap">Collections</h5>
        <div className="colls">
          <button className={'coll mint' + (sel === 'week' ? ' on' : '')} onClick={() => setSel(sel === 'week' ? 'all' : 'week')}>
            <span className="em">✦</span><strong>Identified This Week</strong><small>{week.length} song{week.length === 1 ? '' : 's'}</small></button>
          {collections.map((c) => (
            <button key={c.id} className={'coll' + (c.name === 'Favourites' ? ' gold' : '') + (sel === c.id ? ' on' : '')} onClick={() => setSel(sel === c.id ? 'all' : c.id)}>
              <span className="em">{c.name === 'Favourites' ? '♥' : '🎵'}</span><strong>{c.name}</strong><small>{c.count} song{c.count === 1 ? '' : 's'}</small></button>))}
          <button className="coll new" onClick={() => setCreating(true)}><Icon name="plus" size={20} /><small>New</small></button>
        </div>

        <div className="sorts"><span className="muted small">Sort by:</span>
          {[['date', 'Date'], ['confidence', 'Confidence'], ['artist', 'Artist']].map(([k, l]) => (
            <button key={k} className={'chip' + (sort === k ? ' on' : '')} onClick={() => setSort(k)}>{l}</button>))}</div>

        {rows && list.length === 0 ? (
          <Empty title="No saved songs here yet">Tap the heart on a match or any song to save it.</Empty>
        ) : (
          <ul className="saved">{list.map(({ song, saved_at, confidence }) => (
            <li key={song.id} className="card srow">
              <button className="li-main" onClick={() => openSong(song)}>
                <Cover song={song} size={68} radius={12} />
                <span className="grow"><strong>{song.title}</strong><small>{song.artist}</small>
                  <span className="tags">{song.genre && <span className="chip static">{song.genre}</span>}<small className="muted">{ago(saved_at)}</small></span></span>
                {confidence > 0 && <span className="pm"><b className={confidence >= 0.95 ? 'gold' : 'teal'}>{pct(confidence)}%</b><small>match</small></span>}
              </button>
              {sel === 'week' && isSaved(song.id)
                ? <span className="icon-btn soft on" title="Saved"><Icon name="heart" size={17} fill /></span>
                : <button className="icon-btn soft" onClick={() => remove(song)} aria-label={sel === 'week' ? 'Save' : 'Remove'}><Icon name={sel === 'week' ? 'heart' : 'trash'} size={17} /></button>}
            </li>))}</ul>
        )}
      </div>
      {creating && <NewCollection onClose={() => setCreating(false)} onCreate={async (n) => { const c = await api.createCollection(n); await reload(); setSel(c.id) }} />}
    </>
  )
}
