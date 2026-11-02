"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

// Renderizacao dos artigos do manual. O conteudo e versionado no repositorio
// (nao vem do usuario), mas mantemos os links externos em nova aba.
export function Markdown({ children, className = "" }: { children: string; className?: string }) {
  return (
    <div className={`help-markdown text-sm leading-relaxed text-zinc-700 dark:text-zinc-300 ${className}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children }) => <h1 className="mb-3 mt-1 text-xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">{children}</h1>,
          h2: ({ children }) => <h2 className="mb-2 mt-5 text-base font-semibold text-zinc-900 dark:text-zinc-100">{children}</h2>,
          h3: ({ children }) => <h3 className="mb-1.5 mt-4 text-sm font-semibold text-zinc-800 dark:text-zinc-200">{children}</h3>,
          p: ({ children }) => <p className="mb-3">{children}</p>,
          ul: ({ children }) => <ul className="mb-3 list-disc space-y-1 pl-5">{children}</ul>,
          ol: ({ children }) => <ol className="mb-3 list-decimal space-y-1 pl-5">{children}</ol>,
          li: ({ children }) => <li>{children}</li>,
          strong: ({ children }) => <strong className="font-semibold text-zinc-900 dark:text-zinc-100">{children}</strong>,
          code: ({ children }) => <code className="rounded bg-zinc-100 px-1 py-0.5 text-[12px] dark:bg-zinc-800">{children}</code>,
          pre: ({ children }) => <pre className="mb-3 overflow-x-auto rounded-lg bg-zinc-100 p-3 text-xs dark:bg-zinc-800">{children}</pre>,
          table: ({ children }) => <div className="mb-3 overflow-x-auto"><table className="w-full text-left text-xs">{children}</table></div>,
          th: ({ children }) => <th className="border-b border-zinc-200 px-2 py-1 font-semibold dark:border-zinc-700">{children}</th>,
          td: ({ children }) => <td className="border-b border-zinc-100 px-2 py-1 dark:border-zinc-800">{children}</td>,
          a: ({ href, children }) => (
            <a href={href} target="_blank" rel="noreferrer" className="text-sky-600 underline underline-offset-2 dark:text-sky-400">
              {children}
            </a>
          ),
          blockquote: ({ children }) => (
            <blockquote className="mb-3 border-l-2 border-sky-300 bg-sky-50/60 px-3 py-2 text-xs text-zinc-600 dark:border-sky-700 dark:bg-sky-950/30 dark:text-zinc-300">
              {children}
            </blockquote>
          ),
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
