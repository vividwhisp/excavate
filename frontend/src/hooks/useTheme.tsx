import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'

export type Theme = 'light' | 'dark' | 'system'

interface ThemeContextValue {
  theme: Theme
  resolved: 'light' | 'dark'
  setTheme: (t: Theme) => void
  toggle: () => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

function resolve(theme: Theme): 'light' | 'dark' {
  if (theme === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return theme
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => {
    const saved = localStorage.getItem('excavate:theme')
    return saved === 'light' || saved === 'dark' || saved === 'system' ? saved : 'system'
  })

  const [resolved, setResolved] = useState<'light' | 'dark'>(() => resolve(theme))

  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => setResolved(resolve(theme))
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [theme])

  useEffect(() => {
    const r = resolve(theme)
    setResolved(r)
    document.documentElement.classList.toggle('dark', r === 'dark')
  }, [theme])

  const setTheme = useCallback((t: Theme) => {
    setThemeState(t)
    localStorage.setItem('excavate:theme', t)
  }, [])

  const toggle = useCallback(() => {
    setTheme(resolved === 'dark' ? 'light' : 'dark')
  }, [resolved, setTheme])

  return (
    <ThemeContext.Provider value={{ theme, resolved, setTheme, toggle }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}