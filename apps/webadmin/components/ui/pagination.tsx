"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

// Janela de páginas ao redor da atual, com elipses nas pontas.
// Antes era `Array.from({length: pages}).slice(0, 7)`: da página 8 em diante
// (ou seja, acima de ~105 registros) nenhuma página era alcançável.
function pageWindow(page: number, pages: number, span = 1): (number | "…")[] {
  const first = Math.max(2, page - span);
  const last = Math.min(pages - 1, page + span);

  const out: (number | "…")[] = [1];
  if (first > 2) out.push("…");
  for (let p = first; p <= last; p++) out.push(p);
  if (last < pages - 1) out.push("…");
  if (pages > 1) out.push(pages);
  return out;
}

export function Pagination({
  page,
  total,
  perPage,
  onChange,
}: {
  page: number;
  total: number;
  perPage: number;
  onChange: (page: number) => void;
}) {
  const pages = Math.max(1, Math.ceil(total / perPage));
  if (pages <= 1) return null;

  const from = (page - 1) * perPage + 1;
  const to = Math.min(page * perPage, total);
  const items = pageWindow(page, pages);

  const navBtn =
    "rounded-md p-1.5 text-zinc-500 transition-colors hover:bg-zinc-100 disabled:opacity-40 dark:hover:bg-zinc-800";

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-t border-zinc-100 px-4 py-2.5 text-[13px] text-zinc-500 dark:border-zinc-800/70">
      <span className="tnum">
        {from}–{to} de {total}
      </span>
      <div className="flex flex-wrap items-center gap-0.5">
        <button onClick={() => onChange(page - 1)} disabled={page <= 1} className={navBtn} aria-label="Página anterior">
          <ChevronLeft className="h-4 w-4" />
        </button>

        {items.map((it, i) =>
          it === "…" ? (
            <span key={`gap-${i}`} className="px-1.5 text-zinc-400">
              …
            </span>
          ) : (
            <button
              key={it}
              onClick={() => onChange(it)}
              className={cn(
                "tnum min-w-7 rounded-md px-2 py-1 transition-colors",
                it === page
                  ? "bg-sky-600 font-medium text-white"
                  : "hover:bg-zinc-100 dark:hover:bg-zinc-800",
              )}
            >
              {it}
            </button>
          ),
        )}

        <button onClick={() => onChange(page + 1)} disabled={page >= pages} className={navBtn} aria-label="Próxima página">
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
