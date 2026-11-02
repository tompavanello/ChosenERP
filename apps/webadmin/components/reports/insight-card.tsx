import type { ReactNode } from "react";
import Link from "next/link";
import { AlertTriangle, CheckCircle2, Info, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";

export type InsightSeverity = "info" | "good" | "warn" | "critical";

export type Insight = {
  id?: string;
  severity: InsightSeverity;
  title: string;
  message: string;
  metric?: string;
  benchmark?: string;
  action?: { label: string; href: string };
};

const SEVERITY_META: Record<InsightSeverity, { icon: typeof Info; className: string; label: string }> = {
  good: { icon: CheckCircle2, className: "text-emerald-600 bg-emerald-50 dark:bg-emerald-950/40", label: "Saudavel" },
  info: { icon: Info, className: "text-sky-600 bg-sky-50 dark:bg-sky-950/40", label: "Acompanhar" },
  warn: { icon: AlertTriangle, className: "text-amber-600 bg-amber-50 dark:bg-amber-950/40", label: "Atencao" },
  critical: { icon: XCircle, className: "text-red-600 bg-red-50 dark:bg-red-950/40", label: "Critico" },
};

export function InsightCard({ insight, className }: { insight: Insight; className?: string }) {
  const meta = SEVERITY_META[insight.severity] ?? SEVERITY_META.info;
  const Icon = meta.icon;
  return (
    <Card className={cn("flex gap-3 p-3", className)}>
      <span className={cn("mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg", meta.className)}>
        <Icon className="h-4 w-4" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-sm font-semibold text-zinc-700 dark:text-zinc-200">{insight.title}</p>
          <span className="text-[10px] uppercase tracking-wide text-zinc-400">{meta.label}</span>
        </div>
        <p className="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">{insight.message}</p>
        {(insight.metric || insight.benchmark) && (
          <p className="mt-1 flex flex-wrap gap-x-3 text-[11px] text-zinc-400">
            {insight.metric && <span><b className="font-medium text-zinc-500">Atual:</b> {insight.metric}</span>}
            {insight.benchmark && <span><b className="font-medium text-zinc-500">Referencia:</b> {insight.benchmark}</span>}
          </p>
        )}
        {insight.action && (
          <Link href={insight.action.href} className="mt-1 inline-block text-xs font-medium text-sky-600 hover:underline">
            {insight.action.label} {"->"}
          </Link>
        )}
      </div>
    </Card>
  );
}

export function InsightList({
  insights,
  empty,
  className,
  columns = 1,
}: {
  insights: Insight[];
  empty?: ReactNode;
  className?: string;
  columns?: 1 | 2;
}) {
  if (insights.length === 0) {
    return <>{empty ?? <p className={cn("text-sm text-zinc-400", className)}>Nenhum ponto de atencao no periodo.</p>}</>;
  }
  return (
    <div className={cn("grid gap-3", columns === 2 && "md:grid-cols-2", className)}>
      {insights.map((it, i) => <InsightCard key={it.id ?? `${it.title}-${i}`} insight={it} />)}
    </div>
  );
}
