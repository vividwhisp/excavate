import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Message, Thread } from '../api/types'

export function useThreads() {
  const [threads, setThreads] = useState<Thread[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [loading, setLoading] = useState(true)

  const refreshList = useCallback(async () => {
    const r = await api.listThreads()
    setThreads(r.threads)
  }, [])

  const loadThread = useCallback(async (id: string) => {
    const r = await api.getThread(id)
    setMessages(r.messages)
    setSelectedId(id)
  }, [])

  const createThread = useCallback(async (title: string) => {
    const r = await api.createThread(title)
    await refreshList()
    await loadThread(r.thread.id)
    return r.thread
  }, [refreshList, loadThread])

  useEffect(() => {
    refreshList()
      .catch((err) => console.error('list threads failed', err))
      .finally(() => setLoading(false))
  }, [refreshList])

  const appendMessage = useCallback((m: Message) => {
    setMessages((prev) => (prev.some((x) => x.id === m.id) ? prev : [...prev, m]))
  }, [])

  const updateMessage = useCallback((id: string, patch: Partial<Message>) => {
    setMessages((prev) => prev.map((m) => (m.id === id ? { ...m, ...patch } : m)))
  }, [])

  const refreshMessage = useCallback(async (threadId: string) => {
    const r = await api.getThread(threadId)
    setMessages(r.messages)
  }, [])

  return {
    threads,
    messages,
    selectedId,
    loading,
    refreshList,
    loadThread,
    createThread,
    appendMessage,
    updateMessage,
    refreshMessage,
  }
}