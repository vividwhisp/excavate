import { useState } from 'react'
import AuthForm from '../components/AuthForm'
import ThemeToggle from '../components/ThemeToggle'

export default function LoginPage() {
  const [showAuth, setShowAuth] = useState(false)

  if (!showAuth) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <ThemeToggle />
          </div>
          <h1 className="brand-mark">Excavate</h1>
          <p className="muted tagline">Research assistant with cited sources.</p>
          <button className="btn-primary" onClick={() => setShowAuth(true)}>
            Get started
          </button>
        </div>
      </div>
    )
  }
  return <AuthForm />
}