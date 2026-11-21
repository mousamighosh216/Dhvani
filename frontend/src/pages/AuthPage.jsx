import { useState } from 'react'
import { Wordmark } from '../ui.jsx'
import { useAuth } from '../store.jsx'

export default function AuthPage() {
  const { signIn, signUp } = useAuth()
  const [mode, setMode] = useState('in')
  const [f, setF] = useState({ name: '', email: '', password: '' })
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const set = (k) => (e) => setF({ ...f, [k]: e.target.value })

  async function submit(e) {
    e.preventDefault()
    setBusy(true); setErr('')
    try {
      await (mode === 'in' ? signIn(f.email, f.password) : signUp(f.name, f.email, f.password))
    } catch (x) {
      setErr(x.message); setBusy(false)
    }
  }

  return (
    <div className="auth">
      <Wordmark height={64} />
      <p className="tag">Find the sound. Discover the song.</p>
      <div className="card auth-card">
        <div className="tabs">
          <button className={mode === 'in' ? 'on' : ''} onClick={() => { setMode('in'); setErr('') }}>Sign in</button>
          <button className={mode === 'up' ? 'on' : ''} onClick={() => { setMode('up'); setErr('') }}>Create account</button>
        </div>
        <form onSubmit={submit}>
          {mode === 'up' && <label>Name<input value={f.name} onChange={set('name')} autoComplete="name" required maxLength={80} /></label>}
          <label>Email<input type="email" value={f.email} onChange={set('email')} autoComplete="email" required /></label>
          <label>Password<input type="password" value={f.password} onChange={set('password')} minLength={mode === 'up' ? 8 : undefined}
            autoComplete={mode === 'in' ? 'current-password' : 'new-password'} required />
            {mode === 'up' && <small className="muted">At least 8 characters.</small>}
          </label>
          {err && <p className="error" role="alert">{err}</p>}
          <button className="btn primary wide" disabled={busy}>{busy ? 'Please wait…' : mode === 'in' ? 'Sign in' : 'Create account'}</button>
        </form>
      </div>
    </div>
  )
}
