import { useEffect, useRef, useState } from 'react'
import * as api from '../api.js'
import { Cover, Icon, MiniWave, dur, mmss, pct } from '../ui.jsx'
import { go, useAuth, useLibrary, useUI } from '../store.jsx'

/* ---------- hero carousel ---------- */
function Carousel() {
  const { openSong } = useUI()
  const { isSaved, toggleSave } = useLibrary()
  const { user } = useAuth()
  const [songs, setSongs] = useState(null)
  const [i, setI] = useState(0)
  useEffect(() => { api.getSongs({ sort: 'popular', limit: 7 }).then(setSongs).catch(() => setSongs([])) }, [])

  if (songs && songs.length === 0) {
    return <div className="carousel"><div className="slide pos0 empty-card"><Icon name="music" size={34} /><strong>Your catalog is empty</strong><p>Be the first to <a href={user ? '#/upload' : '#/auth?next=/upload'}>add a song</a>.</p></div></div>
  }
  if (!songs) return <div className="carousel" />
  const n = songs.length
  const pos = (k) => { const d = (k - i + n) % n; return d === 0 ? 0 : d === 1 ? 1 : d === n - 1 ? -1 : 9 }
  return (
    <div className="carousel">
      <div className="stage">
        {songs.map((s, k) => {
          const p = pos(k)
          return (
            <article key={s.id} className={`slide pos${p === -1 ? 'm' : p}`} aria-hidden={p !== 0}>
              <div className="art">
                <Cover song={s} size="100%" radius={0} />
                <span className="pill">{k === 0 ? 'Featured Discovery' : 'Just Identified'}</span>
                {p === 0 && <button className="heart" onClick={() => user ? toggleSave(s) : go('/auth?next=/')} aria-label={user ? 'Save' : 'Sign in to save'}><Icon name="heart" size={18} fill={isSaved(s.id)} /></button>}
              </div>
              <div className="meta">
                <h4>{s.title}</h4><p>{s.artist}</p>
                <div className="between">
                  {s.genre ? <span className="chip static">{s.genre}</span> : <span />}
                  <button className="round" onClick={() => openSong(s)} aria-label="Details"><Icon name="right" size={16} /></button>
                </div>
              </div>
            </article>
          )
        })}
      </div>
      <div className="ctrl">
        <button className="round white" onClick={() => setI((i - 1 + n) % n)} aria-label="Previous"><Icon name="left" size={16} /></button>
        <div className="dots">{songs.map((_, k) => <i key={k} className={k === i ? 'on' : ''} onClick={() => setI(k)} />)}</div>
        <button className="round white" onClick={() => setI((i + 1) % n)} aria-label="Next"><Icon name="right" size={16} /></button>
      </div>
    </div>
  )
}

/* ---------- where the clip sits in the song ---------- */
function Timeline({ song, offset, clip }) {
  const N = 48
  const total = Math.max(song.duration_seconds, 1)
  const a = Math.min(N - 1, Math.floor((offset / total) * N))
  const b = Math.max(a, Math.min(N - 1, Math.ceil(((offset + clip) / total) * N) - 1))
  let s = song.id * 7919
  const rnd = () => ((s = (s * 9301 + 49297) % 233280) / 233280)
  return (
    <div className="timeline">
      <small className="muted">Where your clip sits in the song</small>
      <div className="bars">{Array.from({ length: N }, (_, k) => (
        <i key={k} className={k >= a && k <= b ? 'hit' : ''} style={{ height: 8 + Math.round(rnd() * 34) }} />
      ))}</div>
      <small className="teal">Teal section = matched region ({mmss(offset)} – {mmss(Math.min(offset + clip, total))})</small>
    </div>
  )
}

function Results({ r, onReset }) {
  const { isSaved, toggleSave } = useLibrary()
  const { openSong } = useUI()
  const { user } = useAuth()
  if (!r.matched) {
    return (
      <div className="results">
        <div className="res-head"><div><h2>No match found</h2><p className="muted">We couldn’t find this in the catalog.</p></div>
          <button className="btn ghost" onClick={onReset}><Icon name="back" size={16} /> New Search</button></div>
        <div className="card pad-lg">
          <p>Try 8–10 seconds of clear audio with less background noise. If it’s not in the catalog, help others find it by adding the song.</p>
          <div className="cta">
            <a className="btn primary" href={user ? '#/upload' : '#/auth?next=/upload'}>Add this song</a>
            {!user && <span className="muted small">Sign in or create an account to upload.</span>}
          </div>
        </div>
      </div>
    )
  }
  const s = r.song, saved = isSaved(s.id)
  return (
    <div className="results">
      <div className="res-head"><div><h2>We found a strong match</h2><p className="muted">Based on frequency peaks and audio fingerprint similarity.</p></div>
        <button className="btn ghost" onClick={onReset}><Icon name="back" size={16} /> New Search</button></div>
      <div className="card top-match">
        <div className="tm-label"><i /> TOP MATCH</div>
        <div className="tm-body">
          <Cover song={s} size={116} radius={16} />
          <div className="grow">
            <h3>{s.title}</h3>
            <p className="artist">{s.artist}</p>
            <p className="muted small">{s.genre ? `${s.genre} · ` : ''}{dur(s.duration_seconds)}</p>
            <p className="teal small strong">Matched at {mmss(r.match_offset_seconds)}</p>
          </div>
          <span className="badge gold">{pct(r.confidence)}% Match</span>
        </div>
        <Timeline song={s} offset={r.match_offset_seconds} clip={r.clip_seconds} />
        <div className="tm-actions">
          <button className={'btn soft' + (saved ? ' on' : '')} onClick={() => user ? toggleSave(s) : go('/auth?next=/')}><Icon name="heart" size={16} fill={saved} /> {user ? (saved ? 'Saved' : 'Save') : 'Sign in to save'}</button>
          <button className="btn soft" onClick={() => openSong(s)}>View song</button>
        </div>
      </div>
      {r.others?.length > 0 && (
        <>
          <h5 className="sect gap">Other likely matches</h5>
          <div className="card list">
            {r.others.map((o, k) => (
              <button key={o.song.id} className="li" onClick={() => openSong(o.song)}>
                <span className="rank">{k + 2}</span><Cover song={o.song} size={44} radius={10} />
                <span className="grow"><strong>{o.song.title}</strong><small>{o.song.artist}{o.song.genre ? ` · ${o.song.genre}` : ''}</small></span>
                <span className="pctv">{pct(o.confidence)}%</span>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  )
}

/* ---------- identify panel ---------- */
function Identify() {
  const { refreshMe } = useAuth()
  const [tab, setTab] = useState('mic')
  const [phase, setPhase] = useState('idle') // idle | recording | working
  const [secs, setSecs] = useState(0)
  const [result, setResult] = useState(null)
  const [err, setErr] = useState('')
  const rec = useRef(null)
  const resRef = useRef(null)
  const [drag, setDrag] = useState(false)

  async function identify(file) {
    setPhase('working'); setErr(''); setResult(null)
    try {
      setResult(await api.recognize(file))
      refreshMe().catch(() => {})
    } catch (e) { setErr(e.message) }
    setPhase('idle')
  }

  async function mic() {
    if (phase === 'recording') return rec.current.stop()
    setErr('')
    let stream
    try { stream = await navigator.mediaDevices.getUserMedia({ audio: true }) }
    catch { return setErr('Microphone unavailable. Allow access, and use localhost or HTTPS.') }
    const chunks = []
    const mr = new MediaRecorder(stream)
    mr.ondataavailable = (e) => e.data.size && chunks.push(e.data)
    const tick = setInterval(() => setSecs((s) => s + 1), 1000)
    mr.onstop = () => {
      clearInterval(tick); clearTimeout(cap)
      stream.getTracks().forEach((t) => t.stop())
      identify(new File(chunks, 'clip.webm', { type: mr.mimeType }))
    }
    const cap = setTimeout(() => mr.state === 'recording' && mr.stop(), 12000)
    rec.current = mr; setSecs(0); setPhase('recording'); mr.start()
  }

  useEffect(() => { if (result) resRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' }) }, [result])
  const reset = () => { setResult(null); setErr('') }
  if (result) return <div ref={resRef} className="scroll-m"><Results r={result} onReset={reset} /></div>
  const busy = phase === 'working'
  return (
    <>
      <div className="id-head">
        <span className="eyebrow line">NOW, LET’S IDENTIFY YOUR SOUND</span>
        <h2>What are you listening to?</h2>
        <p className="muted">Upload or record a sound — DHVANI will search for its musical fingerprint.</p>
      </div>
      <div className="card id-card">
        <div className="tabs" role="tablist">
          <button role="tab" aria-selected={tab === 'upload'} className={tab === 'upload' ? 'on' : ''} onClick={() => setTab('upload')} disabled={phase === 'recording'}>Upload</button>
          <button role="tab" aria-selected={tab === 'mic'} className={tab === 'mic' ? 'on' : ''} onClick={() => setTab('mic')} disabled={phase === 'recording'}>Microphone</button>
        </div>
        <div className="id-body">
          {busy ? (
            <><div className="eq"><i /><i /><i /><i /><i /></div><p className="strong">Listening for a fingerprint…</p></>
          ) : tab === 'mic' ? (
            <>
              <p className="muted">Play a song nearby or let DHVANI listen to a short segment.</p>
              <button className={'mic' + (phase === 'recording' ? ' live' : '')} onClick={mic} aria-label={phase === 'recording' ? 'Stop and identify' : 'Start listening'}><Icon name="mic" size={40} /></button>
              <p className="strong lbl">{phase === 'recording' ? `Listening… ${secs}s — tap to identify` : 'Tap to identify from microphone'}</p>
            </>
          ) : (
            <label className={'drop' + (drag ? ' over' : '')}
              onDragOver={(e) => { e.preventDefault(); setDrag(true) }} onDragLeave={() => setDrag(false)}
              onDrop={(e) => { e.preventDefault(); setDrag(false); e.dataTransfer.files[0] && identify(e.dataTransfer.files[0]) }}>
              <Icon name="upload" size={34} />
              <strong>Drop an audio or video file</strong>
              <span className="muted small">or click to browse · MP3, WAV, M4A, MP4… up to 25 MB</span>
              <input type="file" accept="audio/*,video/*" hidden onChange={(e) => e.target.files[0] && identify(e.target.files[0])} />
            </label>
          )}
        </div>
      </div>
      {err && <p className="error center" role="alert">{err}</p>}
    </>
  )
}

export default function Home({ scrollToIdentify }) {
  const { user } = useAuth()
  const ref = useRef(null)
  useEffect(() => { if (scrollToIdentify) ref.current?.scrollIntoView({ behavior: 'smooth', block: 'start' }) }, [scrollToIdentify])
  return (
    <>
      <section className="hero">
        <div className="wrap hero-grid">
          <div>
            <span className="eyebrow">DISCOVER • IDENTIFY • LISTEN</span>
            <h1>Find the sound.<br /><span className="grad">Discover the song.</span></h1>
            <p className="lead">From a few seconds of audio to the song behind it — DHVANI listens, recognizes and helps you discover more.</p>
            <p className="muted">Identify music from audio, video, screen recordings or even a recording from your microphone.</p>
            <div className="cta">
              <button className="btn primary" onClick={() => ref.current?.scrollIntoView({ behavior: 'smooth' })}>Identify a Song</button>
              <button className="btn ghost" onClick={() => go('/explore')}>Explore Music</button>
              <a className="btn ghost" href={user ? '#/upload' : '#/auth?next=/upload'}>Upload a Song</a>
            </div>
            <div className="tech"><MiniWave seed={4} n={12} className="big" /><span className="muted small">Audio fingerprinting technology</span></div>
          </div>
          <Carousel />
        </div>
      </section>
      <section className="identify" ref={ref}><div className="wrap narrow"><Identify /></div></section>
    </>
  )
}
