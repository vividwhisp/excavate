import { useEffect, useRef, useState } from 'react'
import { useResearchStream } from '../hooks/useResearchStream'
import type { Message } from '../api/types'
import MessageBubble from './MessageBubble'
import SearchInput from './SearchInput'

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
            <h2>Excavate</h2>
            <p className="muted">Ask a question and get a cited, researched answer.</p>
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