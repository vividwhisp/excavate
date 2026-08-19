import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import type { Source } from '../api/types'

const citeRe = /\[(\d+)\]/g

/**
 * Turns bare [n] citation markers into clickable links that point at the
 * matching source URL, so react-markdown can render them as badges.
 */
function withCitations(content: string, sources: Source[]): string {
  return content.replace(citeRe, (match, n: string) => {
    const src = sources[Number(n) - 1]
    return src ? `[${match}](${src.url})` : match
  })
}

export default function MarkdownViewer({ content, sources }: { content: string; sources: Source[] }) {
  const md = withCitations(content, sources)

  return (
    <div className="markdown-body">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={{
          a: ({ href, children }) => {
            const text = Array.isArray(children) ? children.join('') : String(children ?? '')
            if (/^\[\d+\]$/.test(text) && href) {
              return (
                <a className="citation" href={href} target="_blank" rel="noreferrer">
                  {text}
                </a>
              )
            }
            return (
              <a href={href} target="_blank" rel="noreferrer">
                {children}
              </a>
            )
          },
        }}
      >
        {md}
      </ReactMarkdown>
    </div>
  )
}