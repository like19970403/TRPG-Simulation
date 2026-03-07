import ReactMarkdown from 'react-markdown'
import { cn } from '../../lib/cn'

interface MarkdownProps {
  children: string
  className?: string
}

/**
 * Renders markdown text with styled prose.
 * Used for item descriptions, scene content, NPC fields, GM broadcasts, etc.
 */
export function Markdown({ children, className }: MarkdownProps) {
  return (
    <div className={cn('prose-game', className)}>
    <ReactMarkdown
      components={{
        // Strip wrapper <p> when content is a single paragraph
        p: ({ children: c }) => <p className="mb-2 last:mb-0">{c}</p>,
        strong: ({ children: c }) => (
          <strong className="font-semibold text-gold">{c}</strong>
        ),
        em: ({ children: c }) => <em className="text-text-primary">{c}</em>,
        ul: ({ children: c }) => (
          <ul className="mb-2 list-disc pl-4 last:mb-0">{c}</ul>
        ),
        ol: ({ children: c }) => (
          <ol className="mb-2 list-decimal pl-4 last:mb-0">{c}</ol>
        ),
        li: ({ children: c }) => <li className="mb-0.5">{c}</li>,
        h1: ({ children: c }) => (
          <h1 className="mb-2 text-lg font-bold text-text-primary">{c}</h1>
        ),
        h2: ({ children: c }) => (
          <h2 className="mb-2 text-base font-bold text-text-primary">{c}</h2>
        ),
        h3: ({ children: c }) => (
          <h3 className="mb-1 text-sm font-bold text-text-primary">{c}</h3>
        ),
        hr: () => <hr className="my-3 border-border" />,
        code: ({ children: c }) => (
          <code className="rounded bg-bg-input px-1 py-0.5 font-mono text-xs text-gold">
            {c}
          </code>
        ),
        blockquote: ({ children: c }) => (
          <blockquote className="my-2 border-l-2 border-gold/40 pl-3 italic text-text-tertiary">
            {c}
          </blockquote>
        ),
        table: ({ children: c }) => (
          <div className="my-2 overflow-x-auto">
            <table className="w-full border-collapse text-sm">{c}</table>
          </div>
        ),
        thead: ({ children: c }) => (
          <thead className="border-b border-gold/30 text-left text-xs font-semibold text-gold">{c}</thead>
        ),
        tbody: ({ children: c }) => <tbody>{c}</tbody>,
        tr: ({ children: c }) => (
          <tr className="border-b border-border/50">{c}</tr>
        ),
        th: ({ children: c }) => (
          <th className="px-2 py-1.5 font-semibold">{c}</th>
        ),
        td: ({ children: c }) => (
          <td className="px-2 py-1.5 text-text-secondary">{c}</td>
        ),
        img: ({ src, alt }) => (
          <img
            src={src}
            alt={alt ?? ''}
            className="my-3 max-w-full rounded-lg border border-border"
            loading="lazy"
          />
        ),
      }}
    >
      {children}
    </ReactMarkdown>
    </div>
  )
}
