import { cn } from "@/lib/utils";
import { COLORS, currencyCompact } from "./chart-theme";

export type WaterfallItem = {
  name: string;
  value: number;
  type?: "positive" | "negative" | "total";
};

/**
 * Fluxo de caixa (waterfall): mostra como o saldo sai de um ponto e chega a
 * outro. Muito mais legivel que pizza para "de onde veio / para onde foi".
 */
export function WaterfallChart({
  items,
  format = currencyCompact,
  height = 220,
  className,
}: {
  items: WaterfallItem[];
  format?: (n: number) => string;
  height?: number;
  className?: string;
}) {
  if (items.length === 0) {
    return <p className={cn("py-8 text-center text-sm text-zinc-400", className)}>Sem dados no periodo.</p>;
  }

  let running = 0;
  const bars = items.map((it) => {
    const type = it.type ?? (it.value >= 0 ? "positive" : "negative");
    if (type === "total") return { ...it, type, from: 0, to: running };
    const from = running;
    const to = running + it.value;
    running = to;
    return { ...it, type, from, to };
  });

  const all = bars.flatMap((b) => [b.from, b.to, 0]);
  const min = Math.min(...all);
  const max = Math.max(...all);
  const span = max - min || 1;

  return (
    <div className={className}>
      <div className="flex items-end gap-2" style={{ height }}>
        {bars.map((b) => {
          const bottom = ((Math.min(b.from, b.to) - min) / span) * 100;
          const size = (Math.abs(b.to - b.from) / span) * 100;
          const color =
            b.type === "total" ? COLORS.net : b.type === "negative" ? COLORS.expense : COLORS.income;
          return (
            <div key={b.name} className="group relative flex h-full min-w-0 flex-1 flex-col justify-end">
              <span className="mb-1 truncate text-center text-[10px] tabular-nums text-zinc-500" title={format(b.to)}>
                {format(b.to)}
              </span>
              <div className="relative flex-1">
                <div
                  className="absolute left-0 right-0 rounded-t-sm transition-opacity group-hover:opacity-80"
                  style={{ bottom: `${bottom}%`, height: `${Math.max(size, 0.8)}%`, backgroundColor: color }}
                  title={`${b.name}: ${format(b.value)}`}
                />
              </div>
              <span className="mt-1 h-8 truncate text-center text-[10px] text-zinc-400" title={b.name}>{b.name}</span>
            </div>
          );
        })}
      </div>
      <div className="mt-1 flex justify-center gap-4 text-[10px] text-zinc-400">
        <span className="flex items-center gap-1"><i className="h-2 w-2 rounded-full" style={{ backgroundColor: COLORS.income }} /> ganho</span>
        <span className="flex items-center gap-1"><i className="h-2 w-2 rounded-full" style={{ backgroundColor: COLORS.expense }} /> perda</span>
        <span className="flex items-center gap-1"><i className="h-2 w-2 rounded-full" style={{ backgroundColor: COLORS.net }} /> resultado</span>
      </div>
    </div>
  );
}
