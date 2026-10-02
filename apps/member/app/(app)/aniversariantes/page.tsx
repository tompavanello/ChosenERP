"use client";

import { useEffect, useState } from "react";
import { Cake, ChevronLeft, ChevronRight, HeartHandshake } from "lucide-react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { getMeBirthdays, type Birthday, type MarriageAnniversary } from "@/lib/api";

const MONTHS = [
  "Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
  "Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
];

export default function AniversariantesPage() {
  const [month, setMonth] = useState(new Date().getMonth() + 1);
  const [birthdays, setBirthdays] = useState<Birthday[]>([]);
  const [marriages, setMarriages] = useState<MarriageAnniversary[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    getMeBirthdays(month)
      .then((d) => {
        setBirthdays(d.birthdays ?? []);
        setMarriages(d.marriages ?? []);
      })
      .catch(() => {
        setBirthdays([]);
        setMarriages([]);
      })
      .finally(() => setLoading(false));
  }, [month]);

  function step(delta: number) {
    setMonth((m) => Math.min(12, Math.max(1, m + delta)));
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Aniversariantes</h1>

      <div className="flex items-center justify-between rounded-xl border border-[var(--line)] bg-[var(--card)] px-2 py-1.5">
        <button onClick={() => step(-1)} disabled={month === 1} className="p-1 text-[var(--muted)] disabled:opacity-30" aria-label="Mês anterior">
          <ChevronLeft className="h-5 w-5" />
        </button>
        <span className="text-sm font-semibold">{MONTHS[month - 1]}</span>
        <button onClick={() => step(1)} disabled={month === 12} className="p-1 text-[var(--muted)] disabled:opacity-30" aria-label="Próximo mês">
          <ChevronRight className="h-5 w-5" />
        </button>
      </div>

      {loading ? (
        <SkeletonRows rows={4} />
      ) : birthdays.length === 0 && marriages.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Ninguém faz aniversário neste mês.</p>
        </Card>
      ) : (
        <>
          {birthdays.length > 0 && (
            <section>
              <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-[var(--muted)]">
                <Cake className="h-4 w-4" /> Aniversários
              </h2>
              <div className="space-y-2">
                {birthdays.map((b) => (
                  <Card key={b.id} className="flex items-center gap-3 py-3">
                    <span className="tnum flex h-10 w-10 shrink-0 flex-col items-center justify-center rounded-lg bg-[var(--brand-soft)] text-[var(--brand-strong)]">
                      <span className="text-sm font-bold leading-none">{b.day}</span>
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium">{b.full_name}</p>
                      <p className="text-xs text-[var(--muted)]">{b.age} anos</p>
                    </div>
                  </Card>
                ))}
              </div>
            </section>
          )}

          {marriages.length > 0 && (
            <section>
              <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-[var(--muted)]">
                <HeartHandshake className="h-4 w-4" /> Bodas
              </h2>
              <div className="space-y-2">
                {marriages.map((m) => (
                  <Card key={m.id} className="flex items-center gap-3 py-3">
                    <span className="tnum flex h-10 w-10 shrink-0 flex-col items-center justify-center rounded-lg bg-[var(--brand-soft)] text-[var(--brand-strong)]">
                      <span className="text-sm font-bold leading-none">{m.day}</span>
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium">
                        {m.full_name}
                        {m.spouse_name ? ` & ${m.spouse_name}` : ""}
                      </p>
                      <p className="text-xs text-[var(--muted)]">{m.years} anos de casamento</p>
                    </div>
                  </Card>
                ))}
              </div>
            </section>
          )}
        </>
      )}

      <p className="text-center text-xs text-[var(--muted)]">
        Envie os parabéns pelo WhatsApp na tela do perfil.
      </p>
    </div>
  );
}
