import { useEffect, useRef, useState } from 'react'
import { useResearchStream } from '../hooks/useResearchStream'
import type { Message } from '../api/types'
import MessageBubble from './MessageBubble'
import SearchInput from './SearchInput'

const suggestions = [
  'Why is the sky blue?',
  'Explain quantum computing simply',
  'Latest AI trends in 2026',
  'How does a search engine rank pages?',
]

interface ChatProps {
  threadId: string | null
  messages: Message[]
  onSend: (query: string) => Promise<{ messageId: string; threadId: string } | null>
  onThreadRefresh: (threadId: string) => Promise<void>
}

export default function Chat({ threadId, messages, onSend, onThreadRefresh }: ChatProps) {
  const [activeMessageId, setActiveMessageId] = useState<string | null>(null)
  const [sending, setSending] = useState(false)
  const stream = useResearchStream(activeMessageId)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length, stream.answer, stream.sources.length, stream.stage])

  useEffect(() => {
    if (stream.done && activeMessageId) {
      onThreadRefresh(threadId ?? '').catch(() => {})
    }
  }, [stream.done, activeMessageId, threadId, onThreadRefresh])

  const send = async (query: string) => {
    setSending(true)
    try {
      const result = await onSend(query)
      if (result) {
        setActiveMessageId(result.messageId)
      }
    } finally {
      setSending(false)
    }
  }

  const isBusy = sending || (stream !== undefined && !stream.done && activeMessageId !== null)

  return (
    <div className="chat">
      <div className="messages">
        {messages.length === 0 && (
          <div className="empty-state">
            <h1 className="hero-title">Excavate</h1>
            <p>Ask anything. Get a researched answer with cited sources.</p>
            <div className="suggestion-chips">
              {suggestions.map((s) => (
                <button key={s} className="chip" onClick={() => send(s)} disabled={isBusy}>
                  {s}
                </button>
              ))}
            </div>
          </div>
        )}
        {messages.map((m) => {
          const isActive = m.id === activeMessageId
          const mStream = isActive
            ? {
                stage: stream.stage,
                answer: stream.answer,
                error: stream.error,
                sources: stream.sources,
              }
            : undefined
          return <MessageBubble key={m.id} message={m} stream={mStream} />
        })}
        <div ref={bottomRef} />
      </div>
      <SearchInput onSubmit={send} disabled={isBusy} />
    </div>
  )
}