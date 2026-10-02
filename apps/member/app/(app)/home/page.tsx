"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Bell, BookOpen, Cake, Calendar, CalendarCheck, HandCoins, Heart, User, Users, UsersRound } from "lucide-react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import {
  datePt,
  dateTimePt,
  getMeAnnouncements,
  getMeEvents,
  getMeMember,
  type Announcement,
  type ChurchEvent,
  type Member,
} from "@/lib/api";

export default function HomePage() {
  const [member, setMember] = useState<Member | null>(null);
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [anns, setAnns] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const today = new Date().toISOString().slice(0, 10);
    Promise.all([
      getMeMember().then((m) => {
        setMember(m);
        return m;
      }),
      getMeEvents(today).catch(() => [] as ChurchEvent[]),
      getMeAnnouncements().catch(() => [] as Announcement[]),
    ])
      .then(([, e, a]) => {
        setEvents(e);
        setAnns(a);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Erro ao carregar."))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <SkeletonRows rows={4} />;

  if (error) {
    return (
      <Card>
        <p className="text-sm text-[var(--muted)]">{error}</p>
      </Card>
    );
  }

  const next = events[0];
  const firstName = member?.nickname || member?.full_name?.split(" ")[0] || "";

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm text-[var(--muted)]">Paz do Senhor,</p>
        <h1 className="text-xl font-bold">{firstName || "membro"}</h1>
      </div>

      <section>
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-[var(--muted)]">
          <Calendar className="h-4 w-4" /> Próximo encontro
        </h2>
        {next ? (
          <Card>
            <p className="font-medium">{next.title}</p>
            <p className="mt-1 text-sm text-[var(--muted)]">
              {dateTimePt(next.starts_at)}
              {next.location ? ` · ${next.location}` : ""}
            </p>
          </Card>
        ) : (
          <Card>
            <p className="text-sm text-[var(--muted)]">Nenhum evento próximo.</p>
          </Card>
        )}
        <Link href="/agenda" className="mt-2 inline-block text-sm text-[var(--brand)]">
          Ver agenda completa
        </Link>
      </section>

      <section>
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-[var(--muted)]">
          <Bell className="h-4 w-4" /> Últimos avisos
        </h2>
        {anns.length === 0 ? (
          <Card>
            <p className="text-sm text-[var(--muted)]">Nenhum aviso no momento.</p>
          </Card>
        ) : (
          <div className="space-y-3">
            {anns.slice(0, 3).map((a) => (
              <Card key={a.id}>
                <div className="flex items-center justify-between gap-2">
                  <p className="font-medium">{a.title}</p>
                  <span className="text-xs text-[var(--muted)]">{datePt(a.published_at)}</span>
                </div>
                <p className="mt-1 line-clamp-3 text-sm text-[var(--muted)]">{a.body}</p>
              </Card>
            ))}
          </div>
        )}
      </section>

      <section className="grid grid-cols-2 gap-3">
        <Link href="/oracao">
          <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
            <Heart className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-medium">Pedido de oração</span>
          </Card>
        </Link>
        <Link href="/aniversariantes">
          <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
            <Cake className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-medium">Aniversariantes</span>
          </Card>
        </Link>
        <Link href="/ministerios">
          <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
            <Users className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-medium">Ministérios</span>
          </Card>
        </Link>
        <Link href="/gd">
          <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
            <UsersRound className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-medium">Meu GD</span>
          </Card>
        </Link>
        <Link href="/contribuir">
          <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
            <HandCoins className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-medium">Contribuir</span>
          </Card>
        </Link>
        <Link href="/materiais">
          <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
            <BookOpen className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-medium">Materiais</span>
          </Card>
        </Link>
        <Link href="/escalas">
          <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
            <CalendarCheck className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-medium">Minhas escalas</span>
          </Card>
        </Link>
        <Link href="/perfil" className="col-span-2">
          <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
            <User className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-medium">Meu perfil</span>
          </Card>
        </Link>
      </section>
    </div>
  );
}
