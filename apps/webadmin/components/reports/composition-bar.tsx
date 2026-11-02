import { cn } from "@/lib/utils";
import { SERIES_COLORS, currencyCompact } from "./chart-theme";

export type CompositionItem = { name: string; value: number };

/**
 * Composicao (parte do todo) em UMA barra 100% empilhada + legenda.
 * A leitura e por comprimento; nao usa pizza/rosca.
 */
export function CompositionBar({
  data,
  total,
  format = currencyCompact,
  colors = SERIES_COLORS,
  emptyLabel = "Sem dados no periodo.",
  className,
}: {
  data: CompositionItem[];
  total?: number;
  format?: (n: number) => string;
  colors?: string[];
  emptyLabel?: string;
  className?: string;
}) {
  const rows = data.filter((d) => Number.isFinite(d.value) && d.value > 0).sort((a, b) => b.value - a.value);
  if (rows.length === 0) {
    return <p className={cn("py-8 text-center text-sm text-zinc-400", className)}>{emptyLabel}</p>;
  }
  const sum = total ?? rows.reduce((s, r) => s + r.value, 0);

  return (
    <div className={cn("space-y-3", className)}>
      <div className="flex h-4 w-full overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
        {rows.map((r, i) => {
          const pct = sum > 0 ? (r.value / sum) * 100 : 0;
          return (
            <div
              key={r.name}
              className="h-4 first:rounded-l-full last:rounded-r-full"
              style={{ width: `${pct}%`, backgroundColor: colors[i % colors.length] }}
              title={`${r.name}: ${format(r.value)} (${pct.toFixed(1)}%)`}
            />
          );
        })}
      </div>
      <ul className="grid gap-x-4 gap-y-1 text-xs sm:grid-cols-2">
        {rows.map((r, i) => {
          const pct = sum > 0 ? (r.value / sum) * 100 : 0;
          return (
            <li key={r.name} className="flex items-center justify-between gap-2">
              <span className="flex min-w-0 items-center gap-1.5">
                <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: colors[i % colors.length] }} />
                <span className="truncate text-zinc-600 dark:text-zinc-300" title={r.name}>{r.name}</span>
              </span>
              <span className="shrink-0 tabular-nums text-zinc-500">
                {format(r.value)} <span className="text-zinc-400">({pct.toFixed(0)}%)</span>
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
