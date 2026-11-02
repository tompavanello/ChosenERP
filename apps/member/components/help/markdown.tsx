"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

export function Markdown({ children }: { children: string }) {
  return (
    <div className="help-markdown text-sm leading-relaxed text-[var(--ink,#18181b)]">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children }) => <h1 className="mb-3 mt-1 text-lg font-bold">{children}</h1>,
          h2: ({ children }) => <h2 className="mb-2 mt-4 text-base font-semibold">{children}</h2>,
          h3: ({ children }) => <h3 className="mb-1.5 mt-3 text-sm font-semibold">{children}</h3>,
          p: ({ children }) => <p className="mb-3">{children}</p>,
          ul: ({ children }) => <ul className="mb-3 list-disc space-y-1 pl-5">{children}</ul>,
          ol: ({ children }) => <ol className="mb-3 list-decimal space-y-1 pl-5">{children}</ol>,
          li: ({ children }) => <li>{children}</li>,
          strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
          code: ({ children }) => <code className="rounded bg-black/5 px-1 py-0.5 text-[12px]">{children}</code>,
          a: ({ href, children }) => <a href={href} target="_blank" rel="noreferrer" className="text-[var(--brand)] underline">{children}</a>,
          blockquote: ({ children }) => <blockquote className="mb-3 border-l-2 border-[var(--brand)] bg-black/5 px-3 py-2 text-xs">{children}</blockquote>,
          table: ({ children }) => <div className="mb-3 overflow-x-auto"><table className="w-full text-left text-xs">{children}</table></div>,
          th: ({ children }) => <th className="border-b border-black/10 px-2 py-1 font-semibold">{children}</th>,
          td: ({ children }) => <td className="border-b border-black/5 px-2 py-1">{children}</td>,
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
