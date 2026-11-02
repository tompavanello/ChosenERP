import { cn } from "@/lib/utils";
import { COLORS } from "./chart-theme";

/**
 * Bullet chart: compara o realizado com o periodo anterior e uma meta.
 * Substitui gauges/medidores e responde "estamos melhor ou pior?".
 */
export function BulletChart({
  label,
  actual,
  previous,
  target,
  format = (n) => String(n),
  higherIsBetter = true,
  className,
}: {
  label: string;
  actual: number;
  previous?: number;
  target?: number;
  format?: (n: number) => string;
  higherIsBetter?: boolean;
  className?: string;
}) {
  const max = Math.max(Math.abs(actual), Math.abs(previous ?? 0), Math.abs(target ?? 0), 1);
  const actualPct = (Math.abs(actual) / max) * 100;
  const prevPct = previous != null ? (Math.abs(previous) / max) * 100 : null;
  const targetPct = target != null ? (Math.abs(target) / max) * 100 : null;
  const good = previous == null ? true : higherIsBetter ? actual >= previous : actual <= previous;

  return (
    <div className={cn("space-y-1.5", className)}>
      <div className="flex items-baseline justify-between gap-2 text-xs">
        <span className="text-zinc-500">{label}</span>
        <span className={cn("font-semibold tabular-nums", good ? "text-emerald-600" : "text-red-600")}>{format(actual)}</span>
      </div>
      <div className="relative h-3 w-full rounded-full bg-zinc-100 dark:bg-zinc-800">
        <div
          className="absolute inset-y-0 left-0 rounded-full"
          style={{ width: `${actualPct}%`, backgroundColor: good ? COLORS.income : COLORS.expense }}
        />
        {prevPct != null && (
          <div className="absolute inset-y-[-3px] w-0.5 bg-zinc-400" style={{ left: `${prevPct}%` }} title={`Anterior: ${format(previous!)}`} />
        )}
        {targetPct != null && (
          <div className="absolute inset-y-[-3px] w-0.5 bg-zinc-900 dark:bg-zinc-100" style={{ left: `${targetPct}%` }} title={`Meta: ${format(target!)}`} />
        )}
      </div>
      <div className="flex justify-between text-[10px] text-zinc-400">
        <span>{previous != null ? `anterior ${format(previous)}` : ""}</span>
        <span>{target != null ? `meta ${format(target)}` : ""}</span>
      </div>
    </div>
  );
}
