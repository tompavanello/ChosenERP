import { cn } from "@/lib/utils";

/**
 * Heatmap para sazonalidade (dia da semana x horario, mes x filial, etc.).
 * A intensidade comunica volume; o numero exato fica no title (hover).
 */
export function Heatmap({
  rows,
  cols,
  values,
  format = (n) => String(n),
  className,
  emptyLabel = "Sem dados no periodo.",
}: {
  rows: string[];
  cols: string[];
  values: number[][];
  format?: (n: number) => string;
  className?: string;
  emptyLabel?: string;
}) {
  const flat = values.flat().filter((v) => Number.isFinite(v));
  if (flat.length === 0 || Math.max(...flat) === 0) {
    return <p className={cn("py-8 text-center text-sm text-zinc-400", className)}>{emptyLabel}</p>;
  }
  const max = Math.max(...flat, 1);

  return (
    <div className={cn("overflow-x-auto", className)}>
      <table className="w-full border-separate border-spacing-1 text-xs">
        <thead>
          <tr>
            <th className="w-16" />
            {cols.map((c) => (
              <th key={c} className="px-1 pb-1 text-center text-[10px] font-medium text-zinc-400">{c}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, ri) => (
            <tr key={r}>
              <td className="pr-2 text-right text-[10px] font-medium text-zinc-500">{r}</td>
              {cols.map((c, ci) => {
                const v = values[ri]?.[ci] ?? 0;
                const intensity = v > 0 ? 0.12 + (v / max) * 0.88 : 0;
                return (
                  <td key={c} className="p-0">
                    <div
                      className="flex h-8 min-w-8 items-center justify-center rounded text-[10px] tabular-nums"
                      style={{
                        backgroundColor: v > 0 ? `rgba(14, 165, 233, ${intensity.toFixed(2)})` : "rgba(148,163,184,0.08)",
                        color: intensity > 0.55 ? "#fff" : "#52525b",
                      }}
                      title={`${r} · ${c}: ${format(v)}`}
                    >
                      {v > 0 ? format(v) : ""}
                    </div>
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
