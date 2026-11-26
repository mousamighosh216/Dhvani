import { useCallback, useEffect, useRef, useState } from 'react'
import { Icon, Wordmark } from './ui.jsx'
import { AuthProvider, LibraryProvider, UICtx, go, useAuth, useRoute } from './store.jsx'
import { SearchModal, SongModal } from './SearchModal.jsx'
import AuthPage from './pages/AuthPage.jsx'
import Home from './pages/Home.jsx'
import Explore from './pages/Explore.jsx'
import Library from './pages/Library.jsx'
import History from './pages/History.jsx'
import Saved from './pages/Saved.jsx'
import Profile from './pages/Profile.jsx'
import Upload from './pages/Upload.jsx'

function Splash() {
  return (
    <div className="splash">
      <div className="dust" aria-hidden="true">{Array.from({ length: 14 }, (_, i) => <i key={i} style={{ left: `${(i * 37) % 100}%`, top: `${(i * 53) % 100}%`, animationDelay: `${i * 0.3}s` }} />)}</div>
      <Wordmark height={150} animate />
      <p className="tag">Finding the identity behind the sound.</p>
      <p className="ready">ALMOST READY</p>
      <div className="loadbar"><i /></div>
    </div>
  )
}

const MENU = [['Home', '/'], ['Explore', '/explore'], ['History', '/history'], ['Library', '/library'], ['Saved Matches', '/saved']]
const NAV = [['Discover', '/'], ['Explore', '/explore'], ['Library', '/library']]

function Header({ path, onSearch }) {
  const { user, logout } = useAuth()
  const [open, setOpen] = useState(false)
  const box = useRef(null)
  useEffect(() => { setOpen(false) }, [path])
  useEffect(() => {
    const h = (e) => box.current && !box.current.contains(e.target) && setOpen(false)
    document.addEventListener('mousedown', h)
    return () => document.removeEventListener('mousedown', h)
  }, [])
  const active = (p) => (p === '/' ? path === '/' || path === '/identify' : path === p)
  return (
    <header className="top">
      <div className="wrap bar">
        <div className="menu-box" ref={box}>
          <button className="icon-btn menu-btn" onClick={() => setOpen(!open)} aria-label="Menu" aria-expanded={open}><Icon name="menu" size={24} /></button>
          {open && (
            <nav className="menu" aria-label="Navigate">
              <h6>NAVIGATE</h6>
              {MENU.filter(([, p]) => user || p === '/' || p === '/explore').map(([label, p]) => (
                <a key={p} href={'#' + p} className={active(p) ? 'on' : ''}><i />{label}</a>
              ))}
              {user ? <>
                <hr />
                <a href="#/profile" className={path === '/profile' ? 'on' : ''}><Icon name="user" size={16} /> View Profile</a>
                <button onClick={logout}>Sign out</button>
              </> : <a href="#/auth?next=/"><Icon name="user" size={16} /> Sign in / Create account</a>}
            </nav>
          )}
        </div>
        <a href="#/" className="brand" aria-label="Dhvani home"><Wordmark height={22} /></a>
        {/* <nav className="links">{NAV.filter(([, p]) => user || p !== '/library').map(([label, p]) => <a key={p} href={'#' + p} className={active(p) ? 'on' : ''}>{label}</a>)}</nav> */}
        <div className="right">
          <button className="icon-btn" onClick={onSearch} aria-label="Search"><Icon name="search" size={20} /></button>
          {user
            ? <a href="#/profile" className="avatar" aria-label="Profile">{user.name.charAt(0).toUpperCase()}</a>
            : <a href="#/auth?next=/" className="btn sm">Sign in</a>}
        </div>
      </div>
    </header>
  )
}

function Layout() {
  const path = useRoute()
  const { user } = useAuth()
  const [search, setSearch] = useState(false)
  const [song, setSong] = useState(null)
  const ui = { openSearch: useCallback(() => setSearch(true), []), openSong: useCallback((s) => setSong(s), []) }

  useEffect(() => { if (path !== '/identify') window.scrollTo(0, 0) }, [path])
  useEffect(() => {
    const h = (e) => {
      const typing = /input|textarea|select/i.test(e.target.tagName)
      if ((e.key === '/' && !typing) || ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k')) { e.preventDefault(); setSearch(true) }
    }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [])

  const protectedPaths = ['/history', '/library', '/saved', '/profile', '/upload']
  const nextPath = () => {
    const next = new URLSearchParams(location.hash.split('?')[1] || '').get('next')
    return next?.startsWith('/') && next !== '/auth' ? next : '/'
  }
  const page = path === '/auth'
    ? <AuthPage nextPath={nextPath()} />
    : !user && protectedPaths.includes(path)
      ? <AuthPage nextPath={path} />
      : ({ '/': <Home />, '/identify': <Home scrollToIdentify />, '/explore': <Explore />, '/library': <Library />,
        '/history': <History />, '/saved': <Saved />, '/profile': <Profile />, '/upload': <Upload /> }[path] || <Home />)
  return (
    <UICtx.Provider value={ui}>
      <Header path={path} onSearch={ui.openSearch} />
      <main>{page}</main>
      <footer className="foot"><div className="wrap">Dhvani · Audio fingerprinting</div></footer>
      {search && <SearchModal onClose={() => setSearch(false)} />}
      {song && <SongModal song={song} onClose={() => setSong(null)} />}
    </UICtx.Provider>
  )
}

function Shell() {
  const { ready } = useAuth()
  const [splash, setSplash] = useState(() => { try { return !sessionStorage.getItem('dhvani_seen') } catch { return true } })
  useEffect(() => {
    if (!splash) return
    const t = setTimeout(() => { setSplash(false); try { sessionStorage.setItem('dhvani_seen', '1') } catch {} }, 2400)
    return () => clearTimeout(t)
  }, [splash])
  if (splash) return <Splash />
  if (!ready) return <div className="boot" />
  return <LibraryProvider><Layout /></LibraryProvider>
}

export default function App() {
  return <AuthProvider><Shell /></AuthProvider>
}
