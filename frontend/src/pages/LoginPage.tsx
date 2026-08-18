import { useState } from 'react'
import AuthForm from '../components/AuthForm'

export default function LoginPage() {
  const [showAuth, setShowAuth] = useState(false)

  if (!showAuth) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <h1>Excavate</h1>
          <p className="muted">Research assistant with cited sources.</p>
          <button onClick={() => setShowAuth(true)}>Get started</button>
        </div>
      </div>
    )
  }
  return <AuthForm />
}