import { cn } from "@/lib/utils";
import { COLORS, currencyCompact, numberCompact } from "./chart-theme";

export type RankedItem = { name: string; value: number };

/**
 * Comparacao por comprimento: barras horizontais ranqueadas.
 * Substitui pizza/rosca e mantem Top N + "Outros" para nao poluir.
 */
export function RankedBar({
  data,
  total,
  topN = 8,
  format = currencyCompact,
  color = COLORS.sky,
  showShare = true,
  emptyLabel = "Sem dados no periodo.",
  className,
}: {
  data: RankedItem[];
  total?: number;
  topN?: number;
  format?: (n: number) => string;
  color?: string;
  showShare?: boolean;
  emptyLabel?: string;
  className?: string;
}) {
  const sorted = [...data].filter((d) => Number.isFinite(d.value) && d.value !== 0).sort((a, b) => Math.abs(b.value) - Math.abs(a.value));
  if (sorted.length === 0) {
    return <p className={cn("py-8 text-center text-sm text-zinc-400", className)}>{emptyLabel}</p>;
  }

  const head = sorted.slice(0, topN);
  const rest = sorted.slice(topN);
  const rows = rest.length > 0
    ? [...head, { name: `Outros (${rest.length})`, value: rest.reduce((s, r) => s + r.value, 0) }]
    : head;

  const max = Math.max(...rows.map((r) => Math.abs(r.value)), 1);
  const sum = total ?? rows.reduce((s, r) => s + Math.abs(r.value), 0);

  return (
    <ul className={cn("space-y-2.5", className)}>
      {rows.map((r) => {
        const abs = Math.abs(r.value);
        const width = Math.max(1.5, (abs / max) * 100);
        const share = sum > 0 ? (abs / sum) * 100 : 0;
        const negative = r.value < 0;
        return (
          <li key={r.name} className="group">
            <div className="flex items-baseline justify-between gap-3 text-sm">
              <span className="min-w-0 truncate text-zinc-600 dark:text-zinc-300" title={r.name}>{r.name}</span>
              <span className={cn("shrink-0 tabular-nums font-medium", negative ? "text-red-600" : "text-zinc-700 dark:text-zinc-200")}>
                {format(r.value)}
                {showShare && <span className="ml-1.5 text-xs font-normal text-zinc-400">{share.toFixed(1)}%</span>}
              </span>
            </div>
            <div className="mt-1 h-2 w-full overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
              <div
                className="h-2 rounded-full transition-all group-hover:opacity-80"
                style={{ width: `${width}%`, backgroundColor: negative ? COLORS.expense : color }}
              />
            </div>
          </li>
        );
      })}
    </ul>
  );
}

/** Variante para contagens inteiras (pessoas, eventos). */
export function RankedCount({ data, ...rest }: Omit<Parameters<typeof RankedBar>[0], "format">) {
  return <RankedBar data={data} format={numberCompact} {...rest} />;
}
