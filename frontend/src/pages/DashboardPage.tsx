import { useCallback, useState } from 'react'
import { api } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { useThreads } from '../hooks/useThreads'
import Chat from '../components/Chat'
import ThemeToggle from '../components/ThemeToggle'

export default function DashboardPage() {
  const { user, logout } = useAuth()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const {
    threads,
    messages,
    selectedId,
    loading,
    createThread,
    loadThread,
    refreshMessage,
  } = useThreads()

  const handleSend = useCallback(
    async (query: string) => {
      let threadId = selectedId
      if (!threadId) {
        const title = query.length > 48 ? `${query.slice(0, 48)}…` : query
        const thread = await createThread(title)
        threadId = thread.id
      }

      const resp = await api.postMessage(threadId, query)
      await refreshMessage(threadId)
      return { messageId: resp.message.id, threadId }
    },
    [selectedId, createThread, refreshMessage],
  )

  const pickThread = (id: string) => {
    loadThread(id)
    setSidebarOpen(false)
  }

  const newThread = async () => {
    await createThread('New thread')
    setSidebarOpen(false)
  }

  if (loading) return <div className="dashboard muted">Loading…</div>

  return (
    <div className="dashboard">
      {sidebarOpen && <div className="sidebar-overlay" onClick={() => setSidebarOpen(false)} />}
      <aside className={`sidebar ${sidebarOpen ? 'open' : ''}`}>
        <div className="sidebar-header">
          <button className="icon-btn menu-toggle" onClick={() => setSidebarOpen(false)} aria-label="Close menu">
            <CloseIcon />
          </button>
          <span className="brand">
            <span className="brand-gradient">Excavate</span>
          </span>
          <ThemeToggle />
          <button className="icon-btn" onClick={newThread} title="New thread" aria-label="New thread">
            <PlusIcon />
          </button>
        </div>
        <nav className="thread-list">
          {threads.map((t) => (
            <button
              key={t.id}
              className={`thread-item ${t.id === selectedId ? 'active' : ''}`}
              onClick={() => pickThread(t.id)}
              title={t.title}
            >
              {t.title}
            </button>
          ))}
          {threads.length === 0 && <div className="muted" style={{ padding: '12px' }}>No threads yet</div>}
        </nav>
        <div className="sidebar-footer">
          <span className="email" title={user?.email}>{user?.email}</span>
          <button className="link" onClick={logout}>
            Log out
          </button>
        </div>
      </aside>
      <main className="main">
        <div className="chat-topbar">
          <button className="icon-btn menu-toggle" onClick={() => setSidebarOpen(true)} aria-label="Open menu">
            <MenuIcon />
          </button>
          <span className="muted" style={{ fontSize: 13 }}>
            {selectedId ? '' : 'Select a thread or ask a new question'}
          </span>
        </div>
        <Chat
          threadId={selectedId}
          messages={messages}
          onSend={handleSend}
          onThreadRefresh={refreshMessage}
        />
      </main>
    </div>
  )
}

function MenuIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
      <path d="M4 6h16M4 12h16M4 18h16" />
    </svg>
  )
}

function CloseIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
      <path d="M6 6l12 12M18 6L6 18" />
    </svg>
  )
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
      <path d="M12 5v14M5 12h14" />
    </svg>
  )
}