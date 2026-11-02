"use client";

import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { monthStartISO, todayISO } from "@/lib/format";

function addDays(days: number) {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return todayISO(d);
}

function addMonths(months: number) {
  const d = new Date();
  d.setMonth(d.getMonth() + months);
  return todayISO(d);
}

/**
 * Barra de periodo padrao dos relatorios: atalhos (mes/ano/12 meses),
 * intervalo livre e filtros extras (filial, conta, tipo...).
 */
export function ReportToolbar({
  from,
  to,
  onFrom,
  onTo,
  onChange,
  extra,
  actions,
  presets = true,
}: {
  from: string;
  to: string;
  onFrom: (v: string) => void;
  onTo: (v: string) => void;
  onChange?: (from: string, to: string) => void;
  extra?: ReactNode;
  actions?: ReactNode;
  presets?: boolean;
}) {
  const apply = (f: string, t: string) => {
    onFrom(f);
    onTo(t);
    onChange?.(f, t);
  };

  return (
    <Card className="mb-4 flex flex-wrap items-end gap-3 p-3">
      <div>
        <label className="label text-xs">De</label>
        <Input type="date" className="h-8 text-sm" value={from} onChange={(e) => onFrom(e.target.value)} />
      </div>
      <div>
        <label className="label text-xs">Ate</label>
        <Input type="date" className="h-8 text-sm" value={to} onChange={(e) => onTo(e.target.value)} />
      </div>
      {presets && (
        <div className="flex flex-wrap gap-1">
          <Button variant="ghost" className="h-8 px-2 text-xs" onClick={() => apply(monthStartISO(), todayISO())}>
            Mes atual
          </Button>
          <Button variant="ghost" className="h-8 px-2 text-xs" onClick={() => apply(`${new Date().getFullYear()}-01-01`, todayISO())}>
            Ano atual
          </Button>
          <Button variant="ghost" className="h-8 px-2 text-xs" onClick={() => apply(addMonths(-12), todayISO())}>
            12 meses
          </Button>
          <Button variant="ghost" className="h-8 px-2 text-xs" onClick={() => apply(addDays(-90), todayISO())}>
            90 dias
          </Button>
        </div>
      )}
      {extra}
      <div className="ml-auto flex items-end gap-2">
        {actions}
        {onChange && (
          <Button className="h-8 text-sm" onClick={() => onChange(from, to)}>
            Aplicar
          </Button>
        )}
      </div>
    </Card>
  );
}
