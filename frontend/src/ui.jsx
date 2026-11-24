import { useEffect } from 'react'

/* ---------- icons ---------- */
const P = {
  menu: <path d="M4 7h16M4 12h11M4 17h16" />,
  search: <><circle cx="11" cy="11" r="6.5" /><path d="M20 20l-4.2-4.2" /></>,
  mic: <><rect x="9" y="3" width="6" height="11" rx="3" /><path d="M5.5 11a6.5 6.5 0 0013 0M12 17.5V21" /></>,
  music: <><path d="M9 18V6.5l9-2V16" /><circle cx="6.8" cy="18" r="2.2" /><circle cx="15.8" cy="16" r="2.2" /></>,
  close: <path d="M6 6l12 12M18 6L6 18" />,
  right: <path d="M9 6l6 6-6 6" />,
  left: <path d="M15 6l-6 6 6 6" />,
  heart: <path d="M12 20s-7.5-4.6-9-9.4A5 5 0 0112 7a5 5 0 019 3.6C19.5 15.4 12 20 12 20z" />,
  trash: <path d="M4 7h16M9 7V4.5h6V7M6.5 7l1 13h9l1-13" />,
  user: <><circle cx="12" cy="8" r="3.8" /><path d="M4.5 20.5a7.5 7.5 0 0115 0" /></>,
  edit: <path d="M4 20h4L19 9l-4-4L4 16v4zM13.5 6.5l4 4" />,
  upload: <path d="M12 16V4M7 9l5-5 5 5M4.5 20h15" />,
  arrow: <path d="M5 12h14M13 6l6 6-6 6" />,
  back: <path d="M19 12H5M11 6l-6 6 6 6" />,
  check: <path d="M5 12.5l4.5 4.5L19 7.5" />,
  plus: <path d="M12 5v14M5 12h14" />,
}

export function Icon({ name, size = 20, fill = false, className = '' }) {
  return (
    <svg className={'icon ' + className} width={size} height={size} viewBox="0 0 24 24" fill={fill ? 'currentColor' : 'none'}
      stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {P[name]}
    </svg>
  )
}

/* ---------- DHVANI wordmark: letters drawn as equaliser bars ---------- */
const LETTERS = {
  D: [[[0, 10]], [[0, 10]], [[1, 3], [7, 9]], [[2.2, 4.4], [5.6, 7.8]], [[3.4, 6.6]]],
  H: [[[0, 10]], [[3.2, 6.4]], [[4, 5.6]], [[3.2, 6.4]], [[0, 10]]],
  V: [[[0, 4]], [[1.5, 5.5]], [[3, 7]], [[6, 10]], [[3, 7]], [[1.5, 5.5]], [[0, 4]]],
  A: [[[6, 10]], [[4, 8]], [[2, 5.5], [7, 8.4]], [[0, 4], [5.4, 6.8]], [[2, 5.5], [7, 8.4]], [[4, 8]], [[6, 10]]],
  N: [[[0, 10]], [[1.4, 4.6]], [[3.4, 6.6]], [[5.4, 8.6]], [[0, 10]]],
  I: [[[0, 1.7], [3, 10]]],
}
const W = 12, GAP = 9, LGAP = 24, U = 15

export function Wordmark({ height = 150, animate = false, className = '' }) {
  let x = 0
  const bars = []
  'DHVANI'.split('').forEach((ch, li) => {
    LETTERS[ch].forEach((segs, ci) => {
      segs.forEach(([a, b], si) =>
        bars.push(<rect key={`${li}-${ci}-${si}`} className={animate ? 'bar-in' : ''} style={{ animationDelay: `${(li * 5 + ci) * 45}ms` }}
          x={x} y={a * U} width={W} height={(b - a) * U} rx={W / 2} fill="url(#wm)" />))
      x += W + GAP
    })
    x += LGAP - GAP
  })
  const width = x - LGAP + GAP
  return (
    <svg className={'wordmark ' + className} viewBox={`0 0 ${width} ${10 * U}`} height={height} role="img" aria-label="Dhvani">
      <defs>
        <linearGradient id="wm" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#a9e9e2" /><stop offset="1" stopColor="#5dcfc8" />
        </linearGradient>
      </defs>
      {bars}
    </svg>
  )
}

/* ---------- small pieces ---------- */
const TINTS = [['#6fd8cd', '#2fa59d'], ['#a58bb5', '#6d5880'], ['#f2c46d', '#d9893b'], ['#7fb2e6', '#4a76b8'], ['#f0a1a1', '#c9606d'], ['#9bd49a', '#4f9b6a']]

export function Cover({ song, size = 64, radius = 12 }) {
  const style = { width: size, height: size, borderRadius: radius }
  if (song.cover_image_url) return <img className="cover" style={style} src={song.cover_image_url} alt="" loading="lazy" />
  const [a, b] = TINTS[song.id % TINTS.length]
  return (
    <div className="cover" style={{ ...style, background: `linear-gradient(145deg, ${a}, ${b})` }}>
      <Icon name="music" size={typeof size === "number" ? Math.round(size * 0.42) : 56} />
    </div>
  )
}

// decorative bars, deterministic per seed
export function MiniWave({ seed = 1, n = 9, className = '' }) {
  let s = seed * 9301 + 49297
  const rnd = () => ((s = (s * 9301 + 49297) % 233280) / 233280)
  return (
    <span className={'miniwave ' + className} aria-hidden="true">
      {Array.from({ length: n }, (_, i) => <i key={i} style={{ height: 6 + Math.round(rnd() * 16), opacity: 0.45 + rnd() * 0.55 }} />)}
    </span>
  )
}

export function Modal({ onClose, children, label }) {
  useEffect(() => {
    const h = (e) => e.key === 'Escape' && onClose()
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [onClose])
  return (
    <div className="scrim" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal" role="dialog" aria-modal="true" aria-label={label}>{children}</div>
    </div>
  )
}

export function Crumbs({ here }) {
  return (
    <div className="crumbs">
      <div className="wrap">
        <a href="#/"><Icon name="left" size={14} /> Home</a>
        <Icon name="right" size={12} className="sep" />
        <span>{here}</span>
      </div>
    </div>
  )
}

export function Empty({ title, children }) {
  return <div className="empty"><strong>{title}</strong><p>{children}</p></div>
}

/* ---------- helpers ---------- */
export const mmss = (s) => `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(Math.floor(s % 60)).padStart(2, '0')}`
export const dur = (s) => `${Math.floor(s / 60)}:${String(Math.round(s % 60)).padStart(2, '0')}`
export const pct = (c) => Math.round(c * 100)

export function ago(iso) {
  const sec = (Date.now() - new Date(iso).getTime()) / 1000
  if (sec < 60) return 'just now'
  if (sec < 3600) return `${Math.floor(sec / 60)} min ago`
  if (sec < 86400) { const h = Math.floor(sec / 3600); return `${h} hour${h > 1 ? 's' : ''} ago` }
  const d = Math.floor(sec / 86400)
  if (d === 1) return 'Yesterday'
  if (d < 7) return `${d} days ago`
  return new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
}

export const compact = (n) => (n >= 1000 ? `${(n / 1000).toFixed(1).replace(/\.0$/, '')}k` : String(n))
