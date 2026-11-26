import { useCallback, useEffect, useMemo, useState } from 'react'
import * as api from '../api.js'
import { Crumbs, Empty, dur } from '../ui.jsx'
import { Icon } from '../ui.jsx'
import { useAuth, useLibrary } from '../store.jsx'
import { SongRow } from './Explore.jsx'

export function AddSong({ onAdded }) {
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
      await onAdded?.()
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
        <small className="muted">The file is fingerprinted and discarded; catalog details and fingerprints are public.</small>
        <button className="btn primary" disabled={busy}>{busy ? 'Fingerprinting…' : 'Add song'}</button>
      </div>
      {msg && <p className={msg.ok ? 'teal small' : 'error'} role="status">{msg.text}</p>}
    </form>
  )
}

export default function Library() {
  const { user, stats, refreshMe } = useAuth()
  const { reload } = useLibrary()
  const [hist, setHist] = useState(null)
  const [uploads, setUploads] = useState(null)
  const [removalRequests, setRemovalRequests] = useState(null)
  const [uploadErr, setUploadErr] = useState('')
  const [adminErr, setAdminErr] = useState('')
  const [busyUpload, setBusyUpload] = useState(null)
  const [busyReview, setBusyReview] = useState(null)

  const loadUploads = useCallback(async () => {
    try { setUploads(await api.getUserUploads()); setUploadErr('') }
    catch (e) { setUploadErr(e.message) }
  }, [])
  const loadRemovalRequests = useCallback(async () => {
    if (!user.is_admin) return
    try { setRemovalRequests(await api.getRemovalRequests()); setAdminErr('') }
    catch (e) { setAdminErr(e.message) }
  }, [user.is_admin])
  const loadHistory = useCallback(async () => {
    try { setHist(await api.getHistory(200)) }
    catch { setHist([]) }
  }, [])
  useEffect(() => {
    loadHistory()
    reload().catch(() => {})
    loadUploads()
    loadRemovalRequests()
  }, [loadHistory, reload, loadUploads, loadRemovalRequests])

  const recent = useMemo(() => {
    const m = new Map()
    for (const h of hist || []) {
      const e = m.get(h.song.id) || { song: h.song, count: 0 }
      e.count++; m.set(h.song.id, e)
    }
    return [...m.values()].slice(0, 6)
  }, [hist])

  const cards = [['❤️', stats?.saved ?? 0, 'Songs saved'], ['🎵', stats?.identifications ?? 0, 'Tracks identified'], ['💿', stats?.unique_songs ?? 0, 'Unique songs']]

  async function requestRemoval(songId) {
    setBusyUpload(songId); setUploadErr('')
    try {
      await api.requestSongRemoval(songId)
      await loadUploads()
    } catch (e) { setUploadErr(e.message) }
    setBusyUpload(null)
  }

  async function reviewRemoval(id, decision) {
    if (decision === 'approve' && !window.confirm('Delete this song from the shared catalog for everyone? This removes its fingerprints, saved copies, and recognition history.')) return
    setBusyReview(id); setAdminErr('')
    try {
      await api.reviewRemovalRequest(id, decision)
      await Promise.all([loadUploads(), loadRemovalRequests(), reload(), refreshMe(), loadHistory()])
    } catch (e) { setAdminErr(e.message) }
    setBusyReview(null)
  }

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
        <AddSong onAdded={loadUploads} />

        <h5 className="sect gap">Your uploads</h5>
        {uploadErr && <p className="error" role="alert">{uploadErr}</p>}
        {uploads === null ? <p className="muted">Loading uploads…</p> : uploads.length === 0
          ? <Empty title="No uploads yet">Songs you add to the shared catalog will appear here.</Empty>
          : <ul className="card list">{uploads.map(({ song, removal_request_status }) => (
            <li key={song.id} className="li static">
              <span className="grow"><strong>{song.title}</strong><small>{song.artist}</small></span>
              {removal_request_status === 'pending'
                ? <span className="muted small">Removal pending</span>
                : <button className="btn ghost sm" disabled={busyUpload === song.id} onClick={() => requestRemoval(song.id)}>
                  {removal_request_status === 'rejected' ? 'Request removal again' : 'Request removal'}
                </button>}
            </li>))}</ul>}

        {user.is_admin && <>
          <h5 className="sect gap">Catalog removal requests</h5>
          <p className="muted small">Approval removes the song and its fingerprints, saved copies, and history for all users.</p>
          {adminErr && <p className="error" role="alert">{adminErr}</p>}
          {removalRequests === null ? <p className="muted">Loading requests…</p> : removalRequests.length === 0
            ? <p className="muted">No pending removal requests.</p>
            : <ul className="card list">{removalRequests.map(({ id, song, requested_by }) => (
              <li key={id} className="li static">
                <span className="grow"><strong>{song.title}</strong><small>{song.artist} · requested by {requested_by}</small></span>
                <button className="btn ghost sm" disabled={busyReview === id} onClick={() => reviewRemoval(id, 'reject')}>Reject</button>
                <button className="btn primary sm" disabled={busyReview === id} onClick={() => reviewRemoval(id, 'approve')}>Approve &amp; delete</button>
              </li>))}</ul>}
        </>}
      </div>
    </>
  )
}
