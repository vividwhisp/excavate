import type { Source } from '../api/types'

export default function Sources({ sources }: { sources: Source[] }) {
  if (sources.length === 0) return null
  return (
    <div className="sources">
      {sources.map((s) => (
        <a key={s.position} className="source-card" href={s.url} target="_blank" rel="noreferrer">
          <span className="source-index">{s.position}</span>
          <div className="source-title">{s.title || s.url}</div>
          <div className="source-snippet">{s.snippet}</div>
          <div className="source-url">{hostname(s.url)}</div>
        </a>
      ))}
    </div>
  )
}

function hostname(url: string): string {
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}