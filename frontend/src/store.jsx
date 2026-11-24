import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import * as api from './api.js'

/* ---------- hash router ---------- */
export function useRoute() {
  const read = () => (location.hash.slice(1) || '/').split('?')[0]
  const [path, setPath] = useState(read)
  useEffect(() => {
    const h = () => setPath(read())
    window.addEventListener('hashchange', h)
    return () => window.removeEventListener('hashchange', h)
  }, [])
  return path
}
export const go = (p) => { location.hash = p }

/* ---------- auth ---------- */
const AuthCtx = createContext(null)
export const useAuth = () => useContext(AuthCtx)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [stats, setStats] = useState(null)
  const [ready, setReady] = useState(!api.getToken())

  const logout = useCallback(() => { api.setToken(null); setUser(null); setStats(null); setReady(true); go('/') }, [])
  const refreshMe = useCallback(async () => {
    const d = await api.getMe()
    setUser(d.user); setStats(d.stats)
  }, [])

  useEffect(() => {
    api.setUnauthorizedHandler(logout)
    if (api.getToken()) refreshMe().catch(() => api.setToken(null)).finally(() => setReady(true))
  }, [logout, refreshMe])

  const start = useCallback(async (p) => {
    const d = await p
    api.setToken(d.token)
    setUser(d.user)
    await refreshMe().catch(() => {})
  }, [refreshMe])

  const value = useMemo(() => ({
    user, stats, ready, refreshMe, logout, setUser,
    signIn: (e, p) => start(api.login(e, p)),
    signUp: (n, e, p) => start(api.register(n, e, p)),
  }), [user, stats, ready, refreshMe, logout, start])
  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>
}

/* ---------- library: collections + Favourites hearts ---------- */
const LibCtx = createContext(null)
export const useLibrary = () => useContext(LibCtx)

export function LibraryProvider({ children }) {
  const { refreshMe, user } = useAuth()
  const [collections, setCollections] = useState([])
  const [favIds, setFavIds] = useState(new Set())
  const fav = collections.find((c) => c.name === 'Favourites')

  const reload = useCallback(async () => {
    const cols = await api.getCollections()
    setCollections(cols)
    const f = cols.find((c) => c.name === 'Favourites')
    setFavIds(new Set(f ? (await api.getCollectionSongs(f.id)).map((s) => s.song.id) : []))
  }, [])
  useEffect(() => {
    if (user) reload().catch(() => {})
    else { setCollections([]); setFavIds(new Set()) }
  }, [reload, user])

  const toggleSave = useCallback(async (song) => {
    if (!fav) return
    const saved = favIds.has(song.id)
    setFavIds((s) => { const n = new Set(s); saved ? n.delete(song.id) : n.add(song.id); return n }) // optimistic
    try {
      await (saved ? api.unsaveSong(fav.id, song.id) : api.saveSong(fav.id, song.id))
      await Promise.all([reload(), refreshMe()])
    } catch {
      reload().catch(() => {})
    }
  }, [fav, favIds, reload, refreshMe])

  const value = useMemo(() => ({
    collections, reload, toggleSave, isSaved: (id) => favIds.has(id),
  }), [collections, reload, toggleSave, favIds])
  return <LibCtx.Provider value={value}>{children}</LibCtx.Provider>
}

/* ---------- UI actions (modals) ---------- */
export const UICtx = createContext(null)
export const useUI = () => useContext(UICtx)
