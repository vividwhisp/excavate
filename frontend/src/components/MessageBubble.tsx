import type { Message, Source, StreamStage } from '../api/types'
import MarkdownViewer from './MarkdownViewer'
import Sources from './Sources'

const stageLabels: Record<StreamStage, string> = {
  pending: 'Queued',
  searching: 'Searching the web',
  extracting: 'Reading sources',
  reasoning: 'Reasoning',
  done: '',
}

interface StreamState {
  stage: StreamStage
  answer: string
  error: string
  sources: Source[]
}

export default function MessageBubble({
  message,
  stream,
}: {
  message: Message
  stream?: StreamState
}) {
  if (message.role === 'user') {
    return <div className="bubble user">{message.content}</div>
  }

  const sources = stream?.sources ?? message.sources ?? []
  const content = stream?.answer || message.content || ''
  const isStreaming = stream !== undefined
  const streaming = isStreaming && stream!.stage !== 'done'

  return (
    <div className="bubble assistant">
      {streaming && (
        <span className="streaming-pill">
          <span className="dot" />
          {stageLabels[stream!.stage] ?? 'Working'}
        </span>
      )}
      <Sources sources={sources} />
      {stream?.error ? (
        <div className="error-text">Research failed: {stream.error}</div>
      ) : content ? (
        <MarkdownViewer content={content} sources={sources} />
      ) : (
        !isStreaming && <div className="muted">Empty response.</div>
      )}
      {streaming && content && (
        <span className="caret" aria-hidden="true" />
      )}
      {message.status === 'error' && !stream && message.error && (
        <div className="error-text">{message.error}</div>
      )}
    </div>
  )
}