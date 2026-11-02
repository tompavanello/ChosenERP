import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";

/**
 * Bloco de relatorio: titulo-pergunta, subtitulo, insight e o corpo do grafico.
 * O rodape e reservado para fonte/observacao (ex.: "valores em regime de caixa").
 */
export function ReportSection({
  title,
  subtitle,
  insight,
  actions,
  footer,
  className,
  bodyClassName,
  children,
}: {
  title: string;
  subtitle?: string;
  insight?: ReactNode;
  actions?: ReactNode;
  footer?: ReactNode;
  className?: string;
  bodyClassName?: string;
  children: ReactNode;
}) {
  return (
    <Card className={cn("p-4", className)}>
      <div className="mb-3 flex flex-wrap items-start justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-zinc-700 dark:text-zinc-200">{title}</h3>
          {subtitle && <p className="mt-0.5 text-xs text-zinc-400">{subtitle}</p>}
        </div>
        {actions}
      </div>
      {insight && (
        <div className="mb-3 rounded-lg border border-zinc-100 bg-zinc-50/70 px-3 py-2 text-xs text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900/40 dark:text-zinc-300">
          {insight}
        </div>
      )}
      <div className={bodyClassName}>{children}</div>
      {footer && <div className="mt-3 text-[11px] text-zinc-400">{footer}</div>}
    </Card>
  );
}
