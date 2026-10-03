"use client";

import { useEffect, useState } from "react";
import { CalendarCheck, Check, HelpCircle, MapPin, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { checkinEvent, dateTimePt, getMeEvents, setEventRSVP, type ChurchEvent } from "@/lib/api";

export default function AgendaPage() {
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const [done, setDone] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const today = new Date();
    const from = today.toISOString().slice(0, 10);
    const to = new Date(today.getTime() + 90 * 86400000).toISOString().slice(0, 10);
    getMeEvents(from, to)
      .then(setEvents)
      .catch(() => setEvents([]))
      .finally(() => setLoading(false));
  }, []);

  async function rsvp(e: ChurchEvent, status: "going" | "maybe" | "declined") {
    setBusy(e.id);
    setError(null);
    try {
      await setEventRSVP(e.id, status);
      setEvents((prev) => prev.map((x) => (x.id === e.id ? { ...x, my_rsvp: status } : x)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Não foi possível confirmar.");
    } finally {
      setBusy(null);
    }
  }

  async function checkin(e: ChurchEvent) {
    setBusy(e.id);
    setError(null);
    try {
      await checkinEvent(e.id);
      setDone(e.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Não foi possível fazer o check-in.");
    } finally {
      setBusy(null);
    }
  }

  if (loading) return <SkeletonRows rows={5} />;

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Agenda da igreja</h1>

      {error && <p className="text-sm text-red-600">{error}</p>}

      {events.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Nenhum evento agendado.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {events.map((e) => {
            const going = e.my_rsvp === "going";
            return (
              <Card key={e.id}>
                <div className="flex items-center justify-between gap-2">
                  <p className="font-medium">{e.title ?? "Evento"}</p>
                  {e.kind_name && (
                    <span className="rounded-full bg-[var(--brand-soft)] px-2 py-0.5 text-xs text-[var(--brand-strong)]">
                      {e.kind_name}
                    </span>
                  )}
                </div>
                <p className="mt-1 text-sm text-[var(--muted)]">{dateTimePt(e.starts_at)}</p>
                {e.location && (
                  <p className="mt-1 flex items-center gap-1 text-sm text-[var(--muted)]">
                    <MapPin className="h-3.5 w-3.5" /> {e.location}
                  </p>
                )}

                <p className="mt-3 text-xs font-medium uppercase tracking-wide text-[var(--muted)]">
                  Você vai?
                </p>
                <div className="mt-1 grid grid-cols-3 gap-2">
                  <RsvpButton
                    active={going}
                    disabled={busy === e.id}
                    onClick={() => rsvp(e, "going")}
                    icon={<Check className="h-4 w-4" />}
                    label="Eu vou"
                  />
                  <RsvpButton
                    active={e.my_rsvp === "maybe"}
                    disabled={busy === e.id}
                    onClick={() => rsvp(e, "maybe")}
                    icon={<HelpCircle className="h-4 w-4" />}
                    label="Talvez"
                  />
                  <RsvpButton
                    active={e.my_rsvp === "declined"}
                    disabled={busy === e.id}
                    onClick={() => rsvp(e, "declined")}
                    icon={<X className="h-4 w-4" />}
                    label="Não vou"
                  />
                </div>

                <Button
                  type="button"
                  variant={done === e.id ? "outline" : "primary"}
                  className="mt-3 w-full"
                  disabled={busy === e.id || done === e.id}
                  onClick={() => checkin(e)}
                >
                  <CalendarCheck className="h-4 w-4" />
                  {done === e.id ? "Check-in registrado" : "Fazer check-in"}
                </Button>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}

function RsvpButton({
  active,
  disabled,
  onClick,
  icon,
  label,
}: {
  active: boolean;
  disabled?: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`flex flex-col items-center gap-0.5 rounded-lg border py-2 text-xs font-medium transition disabled:opacity-50 ${
        active
          ? "border-[var(--brand)] bg-[var(--brand-soft)] text-[var(--brand-strong)]"
          : "border-[var(--line)] text-[var(--muted)]"
      }`}
    >
      {icon}
      {label}
    </button>
  );
}
