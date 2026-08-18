import type { Message, Thread, User } from './types'

export class UnauthorizedError extends Error {
  constructor() {
    super('unauthorized')
    this.name = 'UnauthorizedError'
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const res = await fetch(path, {
    method: opts.method ?? 'GET',
    credentials: 'include',
    headers: opts.body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  })

  if (res.status === 401) {
    throw new UnauthorizedError()
  }

  const data = (await res.json().catch(() => null)) as T & { error?: string } | null
  if (!res.ok) {
    throw new Error(data?.error ?? `Request failed (${res.status})`)
  }
  return data as T
}

export const api = {
  me: () => request<{ user: User }>('/api/me'),

  login: (email: string, password: string) =>
    request<{ user: User }>('/api/auth/login', { method: 'POST', body: { email, password } }),

  register: (email: string, password: string) =>
    request<{ user: User }>('/api/auth/register', { method: 'POST', body: { email, password } }),

  logout: () => request<{ status: string }>('/api/auth/logout', { method: 'POST' }),

  listThreads: () => request<{ threads: Thread[] }>('/api/threads'),

  createThread: (title: string) =>
    request<{ thread: Thread }>('/api/threads', { method: 'POST', body: { title } }),

  getThread: (id: string) => request<{ thread: Thread; messages: Message[] }>(`/api/threads/${id}`),

  postMessage: (threadId: string, content: string) =>
    request<{ message: Message }>('/api/messages', { method: 'POST', body: { threadId, content } }),
}

export const streamUrl = (messageId: string) => `/api/research/stream?messageID=${messageId}`