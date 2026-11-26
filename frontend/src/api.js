const KEY = 'dhvani_token'
let token = null
try { token = localStorage.getItem(KEY) } catch {}
let onUnauthorized = () => {}

export const getToken = () => token
export const setToken = (t) => {
  token = t
  try { t ? localStorage.setItem(KEY, t) : localStorage.removeItem(KEY) } catch {}
}
export const setUnauthorizedHandler = (fn) => { onUnauthorized = fn }

async function call(path, { method = 'GET', json, form } = {}) {
  const headers = {}
  if (token) headers.Authorization = `Bearer ${token}`
  let body
  if (json) { headers['Content-Type'] = 'application/json'; body = JSON.stringify(json) }
  if (form) body = form
  let res
  try {
    res = await fetch('/api/v1' + path, { method, headers, body })
  } catch {
    throw new Error('Cannot reach the server. Is the API running?')
  }
  const data = await res.json().catch(() => ({}))
  if (res.status === 401 && token) onUnauthorized()
  if (!res.ok || data.status === 'error') throw new Error(data.message || 'Something went wrong')
  return data.data
}

const formOf = (fields) => {
  const fd = new FormData()
  for (const [k, v] of Object.entries(fields)) if (v) fd.append(k, v)
  return fd
}
const qs = (o) => {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(o)) if (v) p.set(k, v)
  const s = p.toString()
  return s ? '?' + s : ''
}

export const register = (name, email, password) => call('/auth/register', { method: 'POST', json: { name, email, password } })
export const login = (email, password) => call('/auth/login', { method: 'POST', json: { email, password } })
export const getMe = () => call('/me')
export const updateName = (name) => call('/me', { method: 'PATCH', json: { name } })

export const recognize = (audio) => call('/recognize', { method: 'POST', form: formOf({ audio }) })
export const getHistory = (limit = 200) => call('/history' + qs({ limit }))
export const getSongs = (params = {}) => call('/songs' + qs(params))
export const getGenres = () => call('/genres')
export const addSong = (fields) => call('/songs', { method: 'POST', form: formOf(fields) })

export const getCollections = () => call('/collections')
export const createCollection = (name) => call('/collections', { method: 'POST', json: { name } })
export const getCollectionSongs = (id) => call(`/collections/${id}/songs`)
export const saveSong = (cid, sid) => call(`/collections/${cid}/songs/${sid}`, { method: 'PUT' })
export const unsaveSong = (cid, sid) => call(`/collections/${cid}/songs/${sid}`, { method: 'DELETE' })
export const getUserUploads = () => call('/me/uploads')
export const requestSongRemoval = (id) => call(`/songs/${id}/removal-requests`, { method: 'POST' })
export const getRemovalRequests = () => call('/admin/removal-requests')
export const reviewRemovalRequest = (id, decision) => call(`/admin/removal-requests/${id}`, { method: 'PATCH', json: { decision } })
