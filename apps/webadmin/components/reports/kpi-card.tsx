import type { ElementType } from "react";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import { pct } from "./chart-theme";

type Tone = "sky" | "green" | "red" | "amber" | "zinc" | "indigo";

/**
 * KPI com contexto: valor, variacao (periodo anterior/YoY) e um mini grafico
 * de tendencia. A cor da variacao respeita `positiveIsGood` (saida caindo e bom).
 */
export function KpiCard({
  label,
  value,
  hint,
  delta,
  deltaLabel,
  positiveIsGood = true,
  tone = "sky",
  icon: Icon,
  spark,
  className,
}: {
  label: string;
  value: string;
  hint?: string;
  delta?: number;
  deltaLabel?: string;
  positiveIsGood?: boolean;
  tone?: Tone;
  icon?: ElementType;
  spark?: number[];
  className?: string;
}) {
  const tones: Record<Tone, string> = {
    sky: "text-sky-600 bg-sky-50 dark:bg-sky-950/40",
    green: "text-emerald-600 bg-emerald-50 dark:bg-emerald-950/40",
    red: "text-red-600 bg-red-50 dark:bg-red-950/40",
    amber: "text-amber-600 bg-amber-50 dark:bg-amber-950/40",
    zinc: "text-zinc-600 bg-zinc-100 dark:bg-zinc-800",
    indigo: "text-indigo-600 bg-indigo-50 dark:bg-indigo-950/40",
  };
  const deltaGood = delta != null && (positiveIsGood ? delta >= 0 : delta <= 0);

  return (
    <Card className={cn("p-3", className)}>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-xs text-zinc-500">{label}</p>
          <p className="mt-0.5 truncate text-xl font-semibold tracking-tight">{value}</p>
        </div>
        {Icon && (
          <span className={cn("flex h-8 w-8 shrink-0 items-center justify-center rounded-lg", tones[tone])}>
            <Icon className="h-4 w-4" />
          </span>
        )}
      </div>

      <div className="mt-2 flex items-end justify-between gap-2">
        <div className="min-w-0">
          {delta != null && (
            <p className={cn("text-xs font-medium", deltaGood ? "text-emerald-600" : "text-red-600")}>
              {pct(delta)} {deltaLabel ?? "vs. anterior"}
            </p>
          )}
          {hint && <p className="truncate text-xs text-zinc-400">{hint}</p>}
        </div>
        {spark && spark.length > 1 && <Sparkline values={spark} className={cn(deltaGood ? "text-emerald-500" : "text-red-500")} />}
      </div>
    </Card>
  );
}

function Sparkline({ values, className }: { values: number[]; className?: string }) {
  const w = 64;
  const h = 22;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = max - min || 1;
  const points = values
    .map((v, i) => {
      const x = (i / (values.length - 1)) * w;
      const y = h - ((v - min) / span) * (h - 2) - 1;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");

  return (
    <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`} className={cn("shrink-0", className)} aria-hidden="true">
      <polyline points={points} fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}
