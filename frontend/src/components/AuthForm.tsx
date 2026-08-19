import { useState } from 'react'
import type { FormEvent } from 'react'
import { useAuth } from '../hooks/useAuth'
import ThemeToggle from './ThemeToggle'

export default function AuthForm() {
  const { login, register } = useAuth()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      if (mode === 'login') {
        await login(email, password)
      } else {
        await register(email, password)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-card">
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <ThemeToggle />
      </div>
      <h1 className="brand-mark">Excavate</h1>
      <p className="muted tagline">Research assistant with cited sources.</p>
      <form onSubmit={submit}>
        <input
          type="email"
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoFocus
        />
        <input
          type="password"
          placeholder="Password (min 8 chars)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={8}
        />
        {error && <div className="error-text">{error}</div>}
        <button className="btn-primary" type="submit" disabled={busy}>
          {busy ? '…' : mode === 'login' ? 'Sign in' : 'Create account'}
        </button>
      </form>
      <button className="link" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
        {mode === 'login' ? 'No account? Register' : 'Have an account? Sign in'}
      </button>
    </div>
  )
}