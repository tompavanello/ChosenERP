"use client";

import type { ChurchEvent, Roster } from "@/lib/api";
import { CalendarGrid } from "./calendar-grid";
import type { CalendarBackgroundItem, CalendarItem, CalendarView } from "./calendar-types";

const ROSTER_COLORS = [
  "#0ea5e9", "#8b5cf6", "#10b981", "#f59e0b", "#ec4899",
  "#14b8a6", "#6366f1", "#f97316", "#84cc16", "#64748b",
];

function colorFor(id?: string) {
  if (!id) return "#94a3b8";
  let h = 0;
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) % 997;
  return ROSTER_COLORS[h % ROSTER_COLORS.length];
}

export function toRosterItem(r: Roster, hasConflict: boolean): CalendarItem<Roster> {
  const subtitle = [r.ministry_name, r.event_name ?? r.event_kind_name].filter(Boolean).join(" · ");
  return {
    id: r.id,
    title: r.title,
    starts_at: r.starts_at,
    ends_at: r.ends_at,
    color: colorFor(r.ministry_id),
    subtitle: subtitle || undefined,
    meta: `${r.confirmed_count}/${r.assignment_count}`,
    status: r.status,
    conflict: hasConflict,
    raw: r,
  };
}

/**
 * Calendario de escalas. Usa a mesma grade da Agenda e mostra os eventos da
 * igreja em segundo plano (contexto de cultos/eventos), sem permitir arrasta-los.
 */
export function RosterCalendar({
  rosters,
  events = [],
  view,
  anchor,
  canWrite,
  conflictRosterIds,
  onOpen,
  onMove,
  onDelete,
  onCreateAt,
}: {
  rosters: Roster[];
  events?: ChurchEvent[];
  view: CalendarView;
  anchor: Date;
  canWrite: boolean;
  conflictRosterIds?: Set<string>;
  onOpen: (roster: Roster) => void;
  onMove: (roster: Roster, startsAt: string, endsAt: string) => void;
  onDelete: (roster: Roster) => void;
  onCreateAt: (date: Date) => void;
}) {
  const background: CalendarBackgroundItem[] = events.map((ev) => ({
    id: ev.id,
    title: ev.title || ev.kind_name || "Evento",
    starts_at: ev.starts_at,
    ends_at: ev.ends_at,
    color: "#94a3b8",
  }));

  return (
    <CalendarGrid
      items={rosters.map((r) => toRosterItem(r, conflictRosterIds?.has(r.id) ?? false))}
      backgroundItems={background}
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
