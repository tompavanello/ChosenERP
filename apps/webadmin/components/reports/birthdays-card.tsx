"use client";

import { useEffect, useState } from "react";
import { Cake, Heart } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { getBirthdays, type BirthdaysResult } from "@/lib/api";

const MONTHS = [
  "janeiro", "fevereiro", "marco", "abril", "maio", "junho",
  "julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
];

/**
 * Widget de aniversariantes do mes (membros e casamentos). Carrega sozinho para
 * poder ser pendurado no dashboard sem acoplar a Visao Geral.
 */
export function BirthdaysCard({ className }: { className?: string }) {
  const [data, setData] = useState<BirthdaysResult | null>(null);

  useEffect(() => {
    getBirthdays().then(setData).catch(() => setData(null));
  }, []);

  if (!data) {
    return (
      <Card className={`p-6 ${className ?? ""}`}>
        <Skeleton className="h-40" />
      </Card>
    );
  }

  const mes = MONTHS[(data.month ?? 1) - 1] ?? "";
  const vazio = data.birthdays.length === 0 && data.marriages.length === 0;

  return (
    <Card className={`p-6 ${className ?? ""}`}>
      <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-zinc-700 dark:text-zinc-200">
        <Cake className="h-4 w-4 text-sky-600" /> Aniversariantes de {mes}
      </h3>

      {vazio ? (
        <p className="text-sm text-zinc-500">Nenhum aniversariante neste mes.</p>
      ) : (
        <div className="grid gap-6 sm:grid-cols-2">
          <div>
            <p className="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-zinc-400">
              <Cake className="h-3.5 w-3.5" /> Nascimento
            </p>
            {data.birthdays.length === 0 ? (
              <p className="text-sm text-zinc-400">-</p>
            ) : (
              <ul className="max-h-64 space-y-1.5 overflow-y-auto pr-1">
                {data.birthdays.map((b) => (
                  <li key={b.id} className="flex items-center justify-between gap-2 text-sm">
                    <span className="truncate">{b.full_name}</span>
                    <span className="shrink-0 text-xs text-zinc-400">
                      dia {b.day} - {b.age} anos
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div>
            <p className="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-zinc-400">
              <Heart className="h-3.5 w-3.5" /> Casamento
            </p>
            {data.marriages.length === 0 ? (
              <p className="text-sm text-zinc-400">-</p>
            ) : (
              <ul className="max-h-64 space-y-1.5 overflow-y-auto pr-1">
                {data.marriages.map((m) => (
                  <li key={m.id} className="flex items-center justify-between gap-2 text-sm">
                    <span className="truncate">
                      {m.full_name}
                      {m.spouse_name ? ` & ${m.spouse_name}` : ""}
                    </span>
                    <span className="shrink-0 text-xs text-zinc-400">
                      dia {m.day} - {m.years} anos
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </Card>
  );
}
