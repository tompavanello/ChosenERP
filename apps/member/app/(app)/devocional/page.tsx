"use client";

import { useEffect, useState } from "react";
import { ArrowLeft, BookOpen, Check, Circle } from "lucide-react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import {
  completeDevotionalDay,
  getMeDevotionalPlan,
  getMeDevotionalPlans,
  uncompleteDevotionalDay,
  type DevotionalDay,
  type DevotionalPlan,
} from "@/lib/api";

export default function DevocionalPage() {
  const [plans, setPlans] = useState<DevotionalPlan[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<string | null>(null);
  const [detail, setDetail] = useState<{ plan: DevotionalPlan; days: DevotionalDay[]; completed: number[] } | null>(null);
  const [openDay, setOpenDay] = useState<number | null>(null);

  useEffect(() => {
    getMeDevotionalPlans()
      .then(setPlans)
      .catch(() => setPlans([]))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!selected) return;
    setDetail(null);
    getMeDevotionalPlan(selected)
      .then((d) => {
        setDetail(d);
        const pending = d.days.find((x) => !d.completed.includes(x.day_number));
        setOpenDay(pending?.day_number ?? d.days[0]?.day_number ?? null);
      })
      .catch(() => setDetail(null));
  }, [selected]);

  async function toggle(day: DevotionalDay) {
    if (!detail) return;
    const done = detail.completed.includes(day.day_number);
    try {
      if (done) await uncompleteDevotionalDay(detail.plan.id, day.day_number);
      else await completeDevotionalDay(detail.plan.id, day.day_number);
      setDetail({
        ...detail,
        completed: done
          ? detail.completed.filter((n) => n !== day.day_number)
          : [...detail.completed, day.day_number],
      });
    } catch {
      /* silencioso */
    }
  }

  if (loading) return <SkeletonRows rows={3} />;

  if (!selected) {
    return (
      <div className="space-y-4">
        <h1 className="text-lg font-bold">Devocional</h1>
        {plans.length === 0 ? (
          <Card>
            <p className="text-sm text-[var(--muted)]">Nenhum plano devocional disponível.</p>
          </Card>
        ) : (
          <div className="space-y-3">
            {plans.map((p) => {
              const pct = p.day_count ? Math.round(((p.completed_count ?? 0) / p.day_count) * 100) : 0;
              return (
                <button key={p.id} className="w-full text-left" onClick={() => setSelected(p.id)}>
                  <Card>
                    <div className="flex items-center gap-2">
                      <BookOpen className="h-5 w-5 text-[var(--brand)]" />
                      <p className="font-medium">{p.title}</p>
                    </div>
                    {p.description && (
                      <p className="mt-1 line-clamp-2 text-sm text-[var(--muted)]">{p.description}</p>
                    )}
                    <div className="mt-3 h-1.5 w-full overflow-hidden rounded-full bg-black/10">
                      <div className="h-full rounded-full bg-[var(--brand)]" style={{ width: `${pct}%` }} />
                    </div>
                    <p className="mt-1 text-xs text-[var(--muted)]">
                      {p.completed_count ?? 0} de {p.day_count} dias
                    </p>
                  </Card>
                </button>
              );
            })}
          </div>
        )}
      </div>
    );
  }

  if (!detail) return <SkeletonRows rows={3} />;

  const doneCount = detail.completed.length;
  const current = detail.days.find((d) => d.day_number === openDay);

  return (
    <div className="space-y-4">
      <button
        onClick={() => {
          setSelected(null);
          setDetail(null);
        }}
        className="inline-flex items-center gap-1 text-sm text-[var(--muted)]"
      >
        <ArrowLeft className="h-4 w-4" /> Planos
      </button>

      <div>
        <h1 className="text-lg font-bold">{detail.plan.title}</h1>
        <p className="text-xs text-[var(--muted)]">
          {doneCount} de {detail.days.length} dias concluídos
        </p>
      </div>

      <div className="flex gap-1 overflow-x-auto pb-1">
        {detail.days.map((d) => {
          const done = detail.completed.includes(d.day_number);
          const active = d.day_number === openDay;
          return (
            <button
              key={d.id}
              onClick={() => setOpenDay(d.day_number)}
              className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm ${
                active
                  ? "bg-[var(--brand)] text-white"
                  : done
                    ? "bg-[var(--brand-soft)] text-[var(--brand-strong)]"
                    : "bg-black/5 text-[var(--muted)]"
              }`}
            >
              {done && !active ? <Check className="h-4 w-4" /> : d.day_number}
            </button>
          );
        })}
      </div>

      {current && (
        <Card>
          <p className="text-xs text-[var(--muted)]">Dia {current.day_number}</p>
          <h2 className="text-base font-semibold">{current.title}</h2>
          {current.bible_reference && (
            <p className="mt-1 text-sm font-medium text-[var(--brand-strong)]">{current.bible_reference}</p>
          )}
          {current.bible_text && (
            <p className="mt-2 whitespace-pre-line rounded-lg bg-[var(--paper)] p-3 text-sm italic text-[var(--muted)]">
              {current.bible_text}
            </p>
          )}
          {current.body && <p className="mt-3 whitespace-pre-line text-sm">{current.body}</p>}
          {current.prayer && (
            <p className="mt-3 rounded-lg bg-[var(--brand-soft)] p-3 text-sm text-[var(--brand-strong)]">
              {current.prayer}
            </p>
          )}
          <button
            onClick={() => toggle(current)}
            className="mt-4 inline-flex items-center gap-2 text-sm font-medium text-[var(--brand)]"
          >
            {detail.completed.includes(current.day_number) ? (
              <>
                <Check className="h-4 w-4" /> Concluído — desmarcar
              </>
            ) : (
              <>
                <Circle className="h-4 w-4" /> Marcar como lido
              </>
            )}
          </button>
        </Card>
      )}
    </div>
  );
}
