"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Cake } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { getBirthdays, type BirthdaysResult } from "@/lib/api";

const MONTHS = [
  "janeiro", "fevereiro", "marco", "abril", "maio", "junho",
  "julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
];

/**
 * Card-resumo de aniversariantes: total do mes e quantos fazem no dia.
 * Substitui a lista completa na Visao Geral (a lista vive no relatorio).
 */
export function BirthdaysSummaryCard({ className }: { className?: string }) {
  const [data, setData] = useState<BirthdaysResult | null>(null);

  useEffect(() => {
    getBirthdays().then(setData).catch(() => setData(null));
  }, []);

  if (!data) {
    return <Card className={`p-6 ${className ?? ""}`}><Skeleton className="h-20" /></Card>;
  }

  const today = new Date().getDate();
  const hoje = data.birthdays.filter((b) => b.day === today).length;
  const casamentosHoje = data.marriages.filter((m) => m.day === today).length;

  return (
    <Card className={`p-6 ${className ?? ""}`}>
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sky-50 text-sky-600 dark:bg-sky-950/40">
          <Cake className="h-5 w-5" />
        </div>
        <div>
          <p className="text-sm font-semibold">Aniversariantes de {MONTHS[(data.month ?? 1) - 1]}</p>
          <p className="text-xs text-zinc-500">Nascimento e casamento</p>
        </div>
      </div>
      <div className="mt-4 grid grid-cols-2 gap-4">
        <div>
          <p className="text-3xl font-semibold tabular-nums">{data.birthdays.length}</p>
          <p className="text-xs text-zinc-500">no mes</p>
        </div>
        <div>
          <p className="text-3xl font-semibold tabular-nums text-sky-700 dark:text-sky-400">{hoje}</p>
          <p className="text-xs text-zinc-500">hoje</p>
        </div>
      </div>
      {casamentosHoje > 0 && (
        <p className="mt-3 text-xs text-zinc-500">{casamentosHoje} aniversario(s) de casamento hoje.</p>
      )}
      <Link href="/dashboard/reports/birthdays" className="mt-3 inline-block text-xs text-sky-600 hover:underline">
        Ver relatorio de aniversariantes
      </Link>
    </Card>
  );
}
