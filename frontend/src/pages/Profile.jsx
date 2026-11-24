import { useState } from 'react'
import * as api from '../api.js'
import { Crumbs, Icon } from '../ui.jsx'
import { useAuth } from '../store.jsx'

const LIMIT = 30 // display-only goal; the API does not enforce it

export default function Profile() {
  const { user, stats, setUser, logout } = useAuth()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(user.name)
  const [err, setErr] = useState('')
  const s = stats || { identifications: 0, unique_songs: 0, this_month: 0, saved: 0, collections: 0 }

  async function save(e) {
    e.preventDefault()
    try { setUser(await api.updateName(name.trim())); setEditing(false); setErr('') } catch (x) { setErr(x.message) }
  }

  const ach = [
    ['🎤', 'First Fingerprint', 'Identify your first song', s.identifications >= 1],
    ['🎧', 'Melody Hunter', 'Identify 10 songs', s.identifications >= 10],
    ['🎼', 'Audiophile', 'Save 5 songs', s.saved >= 5],
  ]
  const cards = [['🎵', s.identifications, 'Songs Identified'], ['♥', s.saved, 'Saved Matches'], ['📚', s.collections, 'Collections'], ['💿', s.unique_songs, 'Unique Songs']]
  return (
    <>
      <Crumbs here="My Account" />
      <div className="wrap narrow2 section">
        <div className="card prof">
          <div className="prof-band" />
          <div className="prof-top">
            <div className="avatar xl">{user.name.charAt(0).toUpperCase()}</div>
            <div>
              {editing ? (
                <form className="namebox" onSubmit={save}>
                  <input autoFocus value={name} onChange={(e) => setName(e.target.value)} maxLength={80} aria-label="Name" />
                  <button className="btn primary sm">Save</button>
                  <button type="button" className="btn ghost sm" onClick={() => { setEditing(false); setName(user.name) }}>Cancel</button>
                </form>
              ) : (
                <h2>{user.name} <button className="icon-btn" onClick={() => setEditing(true)} aria-label="Edit name"><Icon name="edit" size={16} /></button></h2>
              )}
              {err && <p className="error small">{err}</p>}
              <p className="muted small">{user.email} · Member since {new Date(user.create_date).toLocaleDateString(undefined, { month: 'short', year: 'numeric' })}</p>
            </div>
          </div>
          <div className="stats4">{cards.map(([e, n, l]) => <div key={l} className="stat sm"><span className="em">{e}</span><strong>{n}</strong><small>{l}</small></div>)}</div>
        </div>

        <div className="plan">
          <div className="between"><div><h4>Free Plan</h4><p>{LIMIT} identifications/month goal · {s.saved} saved matches</p></div></div>
          <div className="between small gold-t"><span>Monthly identifications</span><b>{s.this_month} / {LIMIT}</b></div>
          <div className="meter"><i style={{ width: `${Math.min(100, (s.this_month / LIMIT) * 100)}%` }} /></div>
        </div>

        <div className="card ach">
          <h5>Achievements</h5>
          <div className="ach-grid">{ach.map(([e, t, d, done]) => (
            <div key={t} className={'ach-i' + (done ? ' done' : '')}><span className="em">{e}</span><strong>{t}</strong><small>{d}</small><small className="state">{done ? '✓ Unlocked' : 'Locked'}</small></div>))}</div>
        </div>
        <p className="center"><button className="btn ghost" onClick={logout}>Sign out</button></p>
      </div>
    </>
  )
}
