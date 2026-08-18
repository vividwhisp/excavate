import { useEffect, useState } from 'react'
import type { Source, StreamEvent, StreamStage } from '../api/types'
import { streamUrl } from '../api/client'

export interface ResearchStreamState {
  stage: StreamStage
  sources: Source[]
  answer: string
  error: string
  done: boolean
}

const initial: ResearchStreamState = {
  stage: 'pending',
  sources: [],
  answer: '',
  error: '',
  done: false,
}

/**
 * Opens an EventSource for an assistant message and folds each SSE frame into
 * local state: progress stage, discovered sources, streamed answer deltas and
 * the terminal done/error events.
 */
export function useResearchStream(messageId: string | null): ResearchStreamState {
  const [state, setState] = useState<ResearchStreamState>(initial)

  useEffect(() => {
    if (!messageId) return
    setState(initial)

    const es = new EventSource(streamUrl(messageId))
    es.onmessage = (e) => {
      let ev: StreamEvent
      try {
        ev = JSON.parse(e.data)
      } catch {
        return
      }
      switch (ev.type) {
        case 'progress':
          setState((s) => ({ ...s, stage: ev.payload.stage }))
          break
        case 'sources':
          setState((s) => ({ ...s, sources: ev.payload.sources }))
          break
        case 'delta':
          setState((s) => ({ ...s, answer: s.answer + ev.payload.text }))
          break
        case 'done':
          setState((s) => ({ ...s, stage: 'done', done: true }))
          es.close()
          break
        case 'error':
          setState((s) => ({ ...s, stage: 'done', done: true, error: ev.payload.message }))
          es.close()
          break
      }
    }
    // onerror: the browser auto-reconnects; we only close on done/error above.

    return () => es.close()
  }, [messageId])

  return state
}