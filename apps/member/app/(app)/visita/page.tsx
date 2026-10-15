"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Bell, Calendar, HandCoins, Heart, Inbox, User } from "lucide-react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import {
  dateTimePt,
  getMeVisitor,
  getVisitorAnnouncements,
  getVisitorEvents,
  type Announcement,
  type ChurchEvent,
  type VisitorProfile,
} from "@/lib/api";

export default function VisitorHomePage() {
  const [visitor, setVisitor] = useState<VisitorProfile | null>(null);
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [anns, setAnns] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const today = new Date().toISOString().slice(0, 10);
    Promise.all([
      getMeVisitor().catch(() => null),
      getVisitorEvents(today).catch(() => [] as ChurchEvent[]),
      getVisitorAnnouncements().catch(() => [] as Announcement[]),
    ])
      .then(([v, e, a]) => {
        setVisitor(v);
        setEvents(e);
        setAnns(a);
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <SkeletonRows rows={4} />;

  const next = events[0];
  const firstName = visitor?.first_name || visitor?.full_name?.split(" ")[0] || "";

  const links = [
    { href: "/visita/agenda", label: "Agenda", icon: Calendar },
    { href: "/visita/avisos", label: "Avisos", icon: Bell },
    { href: "/visita/oracao", label: "Pedido de oração", icon: Heart },
    { href: "/visita/contribuir", label: "Contribuir", icon: HandCoins },
    { href: "/visita/solicitacoes", label: "Quero ser membro", icon: Inbox },
    { href: "/visita/perfil", label: "Meu contato", icon: User },
  ];

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm text-[var(--muted)]">Seja bem-vindo(a),</p>
        <h1 className="text-xl font-bold">{firstName || "visitante"}</h1>
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
      </section>

      {anns.length > 0 && (
        <section>
          <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-[var(--muted)]">
            <Bell className="h-4 w-4" /> Avisos
          </h2>
          <Card>
            <p className="font-medium">{anns[0].title}</p>
            <p className="mt-1 line-clamp-3 text-sm text-[var(--muted)]">{anns[0].body}</p>
          </Card>
        </section>
      )}

      <section className="grid grid-cols-2 gap-3">
        {links.map(({ href, label, icon: Icon }) => (
          <Link key={href} href={href}>
            <Card className="flex h-full flex-col items-center justify-center gap-1 py-5 text-center">
              <Icon className="h-5 w-5 text-[var(--brand)]" />
              <span className="text-sm font-medium">{label}</span>
            </Card>
          </Link>
        ))}
      </section>
    </div>
  );
}
