"use client";

import type { ChurchEvent } from "@/lib/api";
import { PROGRAM_KINDS } from "@/lib/constants";
import { CalendarGrid } from "./calendar-grid";
import type { CalendarItem, CalendarView } from "./calendar-types";

export type AgendaView = CalendarView;

export function eventColor(ev: ChurchEvent, kinds: { id: string; color?: string; name: string }[]) {
  const k = kinds.find((x) => x.id === ev.kind_id);
  if (k?.color) return k.color;
  const kind = ev.kind_name?.toLowerCase() ?? "";
  if (kind.includes("culto")) return "#0ea5e9";
  if (kind.includes("ora")) return "#6366f1";
  if (kind.includes("celu") || kind.includes("grupo")) return "#10b981";
  if (kind.includes("ebd") || kind.includes("escola")) return "#f59e0b";
  if (kind.includes("ensaio")) return "#ec4899";
  if (kind.includes("reuni")) return "#64748b";
  return "#94a3b8";
}

export function eventTitle(ev: ChurchEvent) {
  if (ev.title) return ev.title;
  if (ev.kind_name) return ev.kind_name;
  return PROGRAM_KINDS[ev.kind_name?.toLowerCase() ?? ""]?.label ?? "Evento";
}

export function toAgendaItem(ev: ChurchEvent, kinds: { id: string; color?: string; name: string }[]): CalendarItem<ChurchEvent> {
  const subtitle = [ev.kind_name, ev.owner_ministry_name].filter(Boolean).join(" · ");
  return {
    id: ev.id,
    title: eventTitle(ev),
    starts_at: ev.starts_at,
    ends_at: ev.ends_at,
    color: eventColor(ev, kinds),
    subtitle: subtitle || undefined,
    meta: ev.participants_count > 0 ? `${ev.participants_count} part.` : undefined,
    status: ev.status,
    raw: ev,
  };
}

export function AgendaCalendar({
  events,
  view,
  anchor,
  kinds,
  canWrite,
  onOpen,
  onMove,
  onDelete,
  onCreateAt,
}: {
  events: ChurchEvent[];
  view: AgendaView;
  anchor: Date;
  kinds: { id: string; color?: string; name: string }[];
  canWrite: boolean;
  onOpen: (ev: ChurchEvent) => void;
  onMove: (ev: ChurchEvent, startsAt: string, endsAt: string) => void;
  onDelete: (ev: ChurchEvent) => void;
  onCreateAt: (date: Date) => void;
}) {
  return (
    <CalendarGrid
      items={events.map((ev) => toAgendaItem(ev, kinds))}
      view={view}
      anchor={anchor}
      canWrite={canWrite}
      onOpen={onOpen}
      onMove={onMove}
      onDelete={onDelete}
      onCreateAt={onCreateAt}
    />
  );
}
