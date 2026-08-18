export interface User {
  id: string
  email: string
  createdAt: string
}

export interface Thread {
  id: string
  userId: string
  title: string
  createdAt: string
  updatedAt: string
}

export interface Source {
  url: string
  title: string
  snippet: string
  position: number
}

export interface Message {
  id: string
  threadId: string
  role: 'user' | 'assistant'
  content: string
  status: string
  error?: string
  createdAt: string
  sources?: Source[]
}

export type StreamStage = 'pending' | 'searching' | 'extracting' | 'reasoning' | 'done'

export type StreamEvent =
  | { type: 'progress'; payload: { stage: StreamStage } }
  | { type: 'sources'; payload: { sources: Source[] } }
  | { type: 'delta'; payload: { text: string } }
  | { type: 'done'; payload: Record<string, never> }
  | { type: 'error'; payload: { message: string } }