"use client";

import { useMemo, useState } from "react";
import {
  addDays, eachDayOfInterval, endOfMonth, endOfWeek, format, isSameDay, isSameMonth,
  startOfMonth, startOfWeek,
} from "date-fns";
import { ptBR } from "date-fns/locale";
import { CalendarDays, Clock, MapPin, Trash2, Users } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ChurchEvent } from "@/lib/api";
import { EVENT_ORIGIN, PROGRAM_KINDS } from "@/lib/constants";

const pad = (n: number) => String(n).padStart(2, "0");
const dayKey = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const localISO = (d: Date) => `${dayKey(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
const HOURS = Array.from({ length: 17 }, (_, i) => i + 6); // 06:00 .. 22:00
const DAY_START = 6 * 60;
const DAY_END = 23 * 60;
const ROW_H = 46;

export type AgendaView = "month" | "week" | "day" | "list";

function overlapsDay(ev: ChurchEvent, day: Date) {
  const start = new Date(ev.starts_at);
  const end = ev.ends_at ? new Date(ev.ends_at) : start;
  const d = new Date(day.getFullYear(), day.getMonth(), day.getDate());
  const s = new Date(start.getFullYear(), start.getMonth(), start.getDate());
  const e = new Date(end.getFullYear(), end.getMonth(), end.getDate());
  return d >= s && d <= e;
}

function eventColor(ev: ChurchEvent, kinds: { id: string; color?: string; name: string }[]) {
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

function eventTitle(ev: ChurchEvent) {
  if (ev.title) return ev.title;
  if (ev.kind_name) return ev.kind_name;
  return PROGRAM_KINDS[ev.kind_name?.toLowerCase() ?? ""]?.label ?? "Evento";
}

function statusTone(status: string) {
  if (status === "cancelled") return "opacity-50 line-through";
  if (status === "confirmed") return "ring-1 ring-emerald-400/60";
  if (status === "done") return "opacity-80";
  return "";
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
  const [dragId, setDragId] = useState<string | null>(null);
  const [over, setOver] = useState<string | null>(null);

  const days = useMemo(() => {
    if (view === "week") {
      const s = startOfWeek(anchor, { weekStartsOn: 0 });
      return eachDayOfInterval({ start: s, end: addDays(s, 6) });
    }
    if (view === "day") return [anchor];
    const s = startOfWeek(startOfMonth(anchor), { weekStartsOn: 0 });
    const e = endOfWeek(endOfMonth(anchor), { weekStartsOn: 0 });
    return eachDayOfInterval({ start: s, end: e });
  }, [anchor, view]);

  const byDay = useMemo(() => {
    const map = new Map<string, ChurchEvent[]>();
    for (const ev of events) {
      for (const d of days) {
        if (!overlapsDay(ev, d)) continue;
        const key = dayKey(d);
        const arr = map.get(key) ?? [];
        arr.push(ev);
        map.set(key, arr);
      }
    }
    for (const arr of map.values()) arr.sort((a, b) => a.starts_at.localeCompare(b.starts_at));
    return map;
  }, [events, days]);

  function finishDrag() {
    setDragId(null);
    setOver(null);
  }

  function moveToDay(ev: ChurchEvent, day: Date) {
    const start = new Date(ev.starts_at);
    const end = ev.ends_at ? new Date(ev.ends_at) : null;
    const dur = end ? end.getTime() - start.getTime() : 0;
    const nextStart = new Date(day.getFullYear(), day.getMonth(), day.getDate(), start.getHours(), start.getMinutes());
    const nextEnd = end ? new Date(nextStart.getTime() + dur) : null;
    onMove(ev, localISO(nextStart), nextEnd ? localISO(nextEnd) : "");
  }

  function moveToSlot(ev: ChurchEvent, day: Date, hour: number) {
    const start = new Date(ev.starts_at);
    const end = ev.ends_at ? new Date(ev.ends_at) : null;
    const dur = end ? end.getTime() - start.getTime() : 60 * 60 * 1000;
    const nextStart = new Date(day.getFullYear(), day.getMonth(), day.getDate(), hour, start.getMinutes());
    const nextEnd = new Date(nextStart.getTime() + dur);
    onMove(ev, localISO(nextStart), localISO(nextEnd));
  }

  if (view === "list") {
    return <AgendaList events={events} kinds={kinds} canWrite={canWrite} onOpen={onOpen} onDelete={onDelete} />;
  }

  if (view === "month") {
    return (
      <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
        <div className="grid grid-cols-7 border-b border-zinc-100 bg-zinc-50/70 text-[11px] font-medium text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/40">
          {["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sab"].map((d) => (
            <div key={d} className="px-2 py-2 text-center">{d}</div>
          ))}
        </div>
        <div className="grid grid-cols-7">
          {days.map((day) => {
            const key = dayKey(day);
            const list = byDay.get(key) ?? [];
            const inMonth = isSameMonth(day, anchor);
            return (
              <div
                key={key}
                onDragOver={(e) => { if (canWrite) { e.preventDefault(); setOver(key); } }}
                onDragLeave={() => setOver((v) => (v === key ? null : v))}
                onDrop={(e) => {
                  e.preventDefault();
                  const ev = events.find((x) => x.id === dragId);
                  if (ev) moveToDay(ev, day);
                  finishDrag();
                }}
                onDoubleClick={() => canWrite && onCreateAt(new Date(day.getFullYear(), day.getMonth(), day.getDate(), 19, 0))}
                className={cn(
                  "group min-h-28 border-b border-r border-zinc-100 p-1.5 dark:border-zinc-800",
                  !inMonth && "bg-zinc-50/60 text-zinc-300 dark:bg-zinc-900/30",
                  over === key && "bg-sky-50 ring-2 ring-inset ring-sky-300 dark:bg-sky-950/30",
                  isSameDay(day, new Date()) && "bg-sky-50/40 dark:bg-sky-950/20",
                )}
              >
                <div className="mb-1 flex items-center justify-between">
                  <span className={cn("text-xs font-semibold", isSameDay(day, new Date()) ? "text-sky-600" : "text-zinc-500")}>
                    {format(day, "d")}
                  </span>
                  {canWrite && (
                    <button
                      type="button"
                      onClick={() => onCreateAt(new Date(day.getFullYear(), day.getMonth(), day.getDate(), 19, 0))}
                      className="hidden rounded px-1 text-xs text-zinc-400 hover:bg-zinc-100 hover:text-sky-600 group-hover:block dark:hover:bg-zinc-800"
                      title="Nova ocorrencia neste dia"
                    >
                      +
                    </button>
                  )}
                </div>
                <div className="space-y-1">
                  {list.slice(0, 4).map((ev) => (
                    <div
                      key={ev.id}
                      draggable={canWrite}
                      onDragStart={() => setDragId(ev.id)}
                      onDragEnd={finishDrag}
                      onClick={() => onOpen(ev)}
                      className={cn(
                        "group flex cursor-pointer items-center gap-1 rounded px-1.5 py-0.5 text-[10px] text-white shadow-sm",
                        statusTone(ev.status),
                      )}
                      style={{ backgroundColor: eventColor(ev, kinds) }}
                      title={`${eventTitle(ev)} · ${format(new Date(ev.starts_at), "HH:mm")}`}
                    >
                      <span className="tabular-nums opacity-80">{format(new Date(ev.starts_at), "HH:mm")}</span>
                      <span className="min-w-0 flex-1 truncate">{eventTitle(ev)}</span>
                      {canWrite && (
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); onDelete(ev); }}
                          className="hidden text-white/80 hover:text-white group-hover:block"
                          title="Excluir"
                        >
                          <Trash2 className="h-2.5 w-2.5" />
                        </button>
                      )}
                    </div>
                  ))}
                  {list.length > 4 && <p className="pl-1 text-[10px] text-zinc-400">+{list.length - 4} evento(s)</p>}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    );
  }

  // Semana / dia: grade de horarios
  return (
    <div className="overflow-hidden rounded-xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
      <div className={cn("grid border-b border-zinc-100 bg-zinc-50/70 dark:border-zinc-800 dark:bg-zinc-900/40", view === "day" ? "grid-cols-[56px_1fr]" : "grid-cols-[56px_repeat(7,1fr)]")}>
        <div />
        {days.map((d) => (
          <div key={dayKey(d)} className={cn("px-2 py-2 text-center", isSameDay(d, new Date()) && "bg-sky-50/60 dark:bg-sky-950/20")}>
            <p className="text-[10px] uppercase text-zinc-400">{format(d, "EEE", { locale: ptBR })}</p>
            <p className={cn("text-sm font-semibold", isSameDay(d, new Date()) ? "text-sky-600" : "text-zinc-700 dark:text-zinc-200")}>{format(d, "d")}</p>
          </div>
        ))}
      </div>
      <div className="overflow-y-auto" style={{ maxHeight: 560 }}>
        <div className={cn("relative grid", view === "day" ? "grid-cols-[56px_1fr]" : "grid-cols-[56px_repeat(7,1fr)]")}>
          <div>
            {HOURS.map((h) => (
              <div key={h} style={{ height: ROW_H }} className="border-b border-zinc-100 pr-2 text-right text-[10px] text-zinc-400 dark:border-zinc-800">
                {pad(h)}:00
              </div>
            ))}
          </div>
          {days.map((day) => {
            const key = dayKey(day);
            const list = byDay.get(key) ?? [];
            const layout = layoutDay(list, day);
            return (
              <div key={key} className="relative border-l border-zinc-100 dark:border-zinc-800">
                {HOURS.map((h) => (
                  <div
                    key={h}
                    style={{ height: ROW_H }}
                    onDragOver={(e) => { if (canWrite) { e.preventDefault(); setOver(`${key}-${h}`); } }}
                    onDragLeave={() => setOver((v) => (v === `${key}-${h}` ? null : v))}
                    onDrop={(e) => {
                      e.preventDefault();
                      const ev = events.find((x) => x.id === dragId);
                      if (ev) moveToSlot(ev, day, h);
                      finishDrag();
                    }}
                    onDoubleClick={() => canWrite && onCreateAt(new Date(day.getFullYear(), day.getMonth(), day.getDate(), h, 0))}
                    className={cn(
                      "border-b border-zinc-50 dark:border-zinc-900",
                      over === `${key}-${h}` && "bg-sky-50 dark:bg-sky-950/30",
                      isSameDay(day, new Date()) && "bg-sky-50/20 dark:bg-sky-950/10",
                    )}
                  />
                ))}
                {layout.map(({ ev, top, height, left, width }) => (
                  <div
                    key={ev.id}
                    draggable={canWrite}
                    onDragStart={() => setDragId(ev.id)}
                    onDragEnd={finishDrag}
                    onClick={() => onOpen(ev)}
                    className={cn(
                      "group absolute cursor-pointer overflow-hidden rounded-md px-1.5 py-0.5 text-[10px] text-white shadow-sm",
                      statusTone(ev.status),
                    )}
                    style={{
                      top,
                      height,
                      left: `calc(${left}% + 2px)`,
                      width: `calc(${width}% - 4px)`,
                      backgroundColor: eventColor(ev, kinds),
                    }}
                    title={`${eventTitle(ev)} · ${format(new Date(ev.starts_at), "HH:mm")}`}
                  >
                    <div className="flex items-center gap-1">
                      <span className="truncate font-medium">{eventTitle(ev)}</span>
                      {canWrite && (
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); onDelete(ev); }}
                          className="ml-auto hidden shrink-0 text-white/80 hover:text-white group-hover:block"
                          title="Excluir"
                        >
                          <Trash2 className="h-2.5 w-2.5" />
                        </button>
                      )}
                    </div>
                    <p className="truncate opacity-85">
                      {format(new Date(ev.starts_at), "HH:mm")}
                      {ev.ends_at ? ` - ${format(new Date(ev.ends_at), "HH:mm")}` : ""}
                      {ev.owner_ministry_name ? ` · ${ev.owner_ministry_name}` : ""}
                    </p>
                  </div>
                ))}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function layoutDay(events: ChurchEvent[], day: Date) {
  const items = events
    .map((ev) => {
      const start = new Date(ev.starts_at);
      const end = ev.ends_at ? new Date(ev.ends_at) : new Date(start.getTime() + 60 * 60 * 1000);
      const dayStart = new Date(day.getFullYear(), day.getMonth(), day.getDate(), 0, 0);
      const s = Math.max(start.getTime(), dayStart.getTime() + DAY_START * 60000);
      const e = Math.min(end.getTime(), dayStart.getTime() + DAY_END * 60000);
      return { ev, s, e };
    })
    .filter((x) => x.e > x.s)
    .sort((a, b) => a.s - b.s || b.e - a.e);

  const columns: number[] = [];
  const placed = items.map((it) => {
    let col = columns.findIndex((end) => end <= it.s);
    if (col === -1) { col = columns.length; columns.push(it.e); } else { columns[col] = it.e; }
    return { ...it, col };
  });
  const total = Math.max(columns.length, 1);
  const dayStartMs = new Date(day.getFullYear(), day.getMonth(), day.getDate(), 0, 0).getTime();

  return placed.map(({ ev, s, e, col }) => ({
    ev,
    top: ((s - dayStartMs) / 60000 - DAY_START) / 60 * ROW_H,
    height: Math.max(((e - s) / 60000) / 60 * ROW_H - 2, 18),
    left: (col / total) * 100,
    width: (1 / total) * 100,
  }));
}

function AgendaList({
  events,
  kinds,
  canWrite,
  onOpen,
  onDelete,
}: {
  events: ChurchEvent[];
  kinds: { id: string; color?: string; name: string }[];
  canWrite: boolean;
  onOpen: (ev: ChurchEvent) => void;
  onDelete: (ev: ChurchEvent) => void;
}) {
  if (events.length === 0) {
    return <p className="rounded-xl border border-dashed border-zinc-200 py-12 text-center text-sm text-zinc-400 dark:border-zinc-800">Nenhuma ocorrencia no periodo.</p>;
  }
  const sorted = [...events].sort((a, b) => a.starts_at.localeCompare(b.starts_at));
  return (
    <div className="space-y-2">
      {sorted.map((ev) => (
        <div
          key={ev.id}
          className="flex cursor-pointer items-center gap-3 rounded-xl border border-zinc-200 bg-white p-3 hover:border-sky-200 dark:border-zinc-800 dark:bg-zinc-950"
          onClick={() => onOpen(ev)}
        >
          <span className="h-10 w-1 rounded-full" style={{ backgroundColor: eventColor(ev, kinds) }} />
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{eventTitle(ev)}</p>
            <p className="mt-0.5 flex flex-wrap items-center gap-x-3 text-xs text-zinc-400">
              <span className="inline-flex items-center gap-1"><CalendarDays className="h-3 w-3" />{format(new Date(ev.starts_at), "dd/MM/yyyy")}</span>
              <span className="inline-flex items-center gap-1"><Clock className="h-3 w-3" />{format(new Date(ev.starts_at), "HH:mm")}{ev.ends_at ? ` - ${format(new Date(ev.ends_at), "HH:mm")}` : ""}</span>
              {ev.owner_ministry_name && <span className="inline-flex items-center gap-1"><Users className="h-3 w-3" />{ev.owner_ministry_name}</span>}
              <span className="inline-flex items-center gap-1"><MapPin className="h-3 w-3" />{EVENT_ORIGIN[ev.origin]?.label ?? ev.origin}</span>
            </p>
          </div>
          {canWrite && (
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); onDelete(ev); }}
              className="rounded p-1.5 text-zinc-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/40"
              title="Excluir"
            >
              <Trash2 className="h-4 w-4" />
            </button>
          )}
        </div>
      ))}
    </div>
  );
}
