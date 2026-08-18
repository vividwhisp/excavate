import { useCallback } from 'react'
import { api } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { useThreads } from '../hooks/useThreads'
import Chat from '../components/Chat'

export default function DashboardPage() {
  const { user, logout } = useAuth()
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

  if (loading) return <div className="dashboard muted">Loading…</div>

  return (
    <div className="dashboard">
      <aside className="sidebar">
        <div className="sidebar-header">
          <span className="brand">Excavate</span>
          <button className="new-thread" onClick={() => createThread('New thread')} title="New thread">
            +
          </button>
        </div>
        <nav className="thread-list">
          {threads.map((t) => (
            <button
              key={t.id}
              className={`thread-item ${t.id === selectedId ? 'active' : ''}`}
              onClick={() => loadThread(t.id)}
              title={t.title}
            >
              {t.title}
            </button>
          ))}
          {threads.length === 0 && <div className="muted">No threads yet</div>}
        </nav>
        <div className="sidebar-footer">
          <span className="email">{user?.email}</span>
          <button className="link" onClick={logout}>
            Log out
          </button>
        </div>
      </aside>
      <main className="main">
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