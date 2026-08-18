import { AuthProvider, useAuth } from './hooks/useAuth'
import DashboardPage from './pages/DashboardPage'
import LoginPage from './pages/LoginPage'

function Shell() {
  const { user, loading } = useAuth()
  if (loading) return <div className="auth-page muted">Loading…</div>
  return user ? <DashboardPage /> : <LoginPage />
}

export default function App() {
  return (
    <AuthProvider>
      <Shell />
    </AuthProvider>
  )
}