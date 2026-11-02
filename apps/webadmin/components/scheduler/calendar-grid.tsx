"use client";

import { useMemo, useState } from "react";
import {
  addDays, eachDayOfInterval, endOfMonth, endOfWeek, format, isSameDay, isSameMonth,
  startOfMonth, startOfWeek,
} from "date-fns";
import { ptBR } from "date-fns/locale";
import { AlertTriangle, CalendarDays, Clock, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { CalendarBackgroundItem, CalendarItem, CalendarView } from "./calendar-types";

const pad = (n: number) => String(n).padStart(2, "0");
const dayKey = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
const localISO = (d: Date) => `${dayKey(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
const HOURS = Array.from({ length: 17 }, (_, i) => i + 6); // 06:00 .. 22:00
const DAY_START = 6 * 60;
const DAY_END = 23 * 60;
const ROW_H = 46;

type Dateable = { id: string; starts_at: string; ends_at?: string };

function overlapsDay(item: Dateable, day: Date) {
  const start = new Date(item.starts_at);
  const end = item.ends_at ? new Date(item.ends_at) : start;
  const d = new Date(day.getFullYear(), day.getMonth(), day.getDate());
  const s = new Date(start.getFullYear(), start.getMonth(), start.getDate());
  const e = new Date(end.getFullYear(), end.getMonth(), end.getDate());
  return d >= s && d <= e;
}

function statusClass(status?: string) {
  if (status === "cancelled" || status === "cancelada") return "opacity-50 line-through";
  if (status === "confirmed" || status === "publicada" || status === "confirmado") return "ring-1 ring-emerald-400/60";
  if (status === "done" || status === "concluida") return "opacity-80";
  if (status === "planned" || status === "rascunho") return "border border-dashed border-white/50";
  return "";
}

/**
 * Grade compartilhada de calendario (mes/semana/dia/lista), com arrastar para
 * remarcar, excluir, criar por slot e itens de contexto em segundo plano.
 * A Agenda e as Escalas usam o mesmo componente via adapters tipados.
 */
export function CalendarGrid<T>({
  items,
  backgroundItems = [],
  view,
  anchor,
  canWrite,
  onOpen,
  onMove,
  onDelete,
  onCreateAt,
}: {
  items: CalendarItem<T>[];
  backgroundItems?: CalendarBackgroundItem[];
  view: CalendarView;
  anchor: Date;
  canWrite: boolean;
  onOpen: (item: T) => void;
  onMove: (item: T, startsAt: string, endsAt: string) => void;
  onDelete: (item: T) => void;
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
    const map = new Map<string, CalendarItem<T>[]>();
    for (const item of items) {
      for (const d of days) {
        if (!overlapsDay(item, d)) continue;
        const key = dayKey(d);
        const arr = map.get(key) ?? [];
        arr.push(item);
        map.set(key, arr);
      }
    }
    for (const arr of map.values()) arr.sort((a, b) => a.starts_at.localeCompare(b.starts_at));
    return map;
  }, [items, days]);

  const bgByDay = useMemo(() => {
    const map = new Map<string, CalendarBackgroundItem[]>();
    for (const item of backgroundItems) {
      for (const d of days) {
        if (!overlapsDay(item, d)) continue;
        const key = dayKey(d);
        const arr = map.get(key) ?? [];
        arr.push(item);
        map.set(key, arr);
      }
    }
    for (const arr of map.values()) arr.sort((a, b) => a.starts_at.localeCompare(b.starts_at));
    return map;
  }, [backgroundItems, days]);

  function finishDrag() {
    setDragId(null);
    setOver(null);
  }

  function moveToDay(item: CalendarItem<T>, day: Date) {
    const start = new Date(item.starts_at);
    const end = item.ends_at ? new Date(item.ends_at) : null;
    const dur = end ? end.getTime() - start.getTime() : 0;
    const nextStart = new Date(day.getFullYear(), day.getMonth(), day.getDate(), start.getHours(), start.getMinutes());
    const nextEnd = end ? new Date(nextStart.getTime() + dur) : null;
    onMove(item.raw, localISO(nextStart), nextEnd ? localISO(nextEnd) : "");
  }

  function moveToSlot(item: CalendarItem<T>, day: Date, hour: number) {
    const start = new Date(item.starts_at);
    const end = item.ends_at ? new Date(item.ends_at) : null;
    const dur = end ? end.getTime() - start.getTime() : 60 * 60 * 1000;
    const nextStart = new Date(day.getFullYear(), day.getMonth(), day.getDate(), hour, start.getMinutes());
    const nextEnd = new Date(nextStart.getTime() + dur);
    onMove(item.raw, localISO(nextStart), localISO(nextEnd));
  }

  if (view === "list") {
    return <CalendarList items={items} backgroundItems={backgroundItems} canWrite={canWrite} onOpen={onOpen} onDelete={onDelete} />;
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
            const bg = bgByDay.get(key) ?? [];
            const inMonth = isSameMonth(day, anchor);
            return (
              <div
                key={key}
                onDragOver={(e) => { if (canWrite && dragId) { e.preventDefault(); setOver(key); } }}
                onDragLeave={() => setOver((v) => (v === key ? null : v))}
                onDrop={(e) => {
                  e.preventDefault();
                  const item = items.find((x) => x.id === dragId);
                  if (item) moveToDay(item, day);
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

                {bg.length > 0 && (
                  <div className="mb-1 space-y-0.5">
                    {bg.slice(0, 2).map((b) => (
                      <div
                        key={b.id}
                        className="truncate rounded border border-dashed border-zinc-200 bg-zinc-50 px-1 py-0.5 text-[9px] text-zinc-400 dark:border-zinc-700 dark:bg-zinc-900"
                        title={`Evento da igreja: ${b.title}`}
                      >
                        {format(new Date(b.starts_at), "HH:mm")} {b.title}
                      </div>
                    ))}
                  </div>
                )}

                <div className="space-y-1">
                  {list.slice(0, 4).map((item) => (
                    <div
                      key={item.id}
                      draggable={canWrite && item.draggable !== false}
                      onDragStart={() => setDragId(item.id)}
                      onDragEnd={finishDrag}
                      onClick={() => onOpen(item.raw)}
                      onDoubleClick={(e) => e.stopPropagation()}
                      className={cn(
                        "group flex cursor-pointer items-center gap-1 rounded px-1.5 py-0.5 text-[10px] text-white shadow-sm",
                        statusClass(item.status),
                      )}
                      style={{ backgroundColor: item.color }}
                      title={`${item.title} · ${format(new Date(item.starts_at), "HH:mm")}`}
                    >
                      <span className="tabular-nums opacity-80">{format(new Date(item.starts_at), "HH:mm")}</span>
                      <span className="min-w-0 flex-1 truncate">{item.title}</span>
                      {item.conflict && <AlertTriangle className="h-2.5 w-2.5 shrink-0 text-amber-200" />}
                      {item.meta && <span className="shrink-0 text-[9px] opacity-80">{item.meta}</span>}
                      {canWrite && item.draggable !== false && (
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); onDelete(item.raw); }}
                          className="hidden text-white/80 hover:text-white group-hover:block"
                          title="Excluir"
                        >
                          <Trash2 className="h-2.5 w-2.5" />
                        </button>
                      )}
                    </div>
                  ))}
                  {list.length > 4 && <p className="pl-1 text-[10px] text-zinc-400">+{list.length - 4} item(ns)</p>}
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
            const bg = bgByDay.get(key) ?? [];
            const layout = layoutDay(list, day);
            const bgLayout = layoutDay(bg, day);
            return (
              <div key={key} className="relative border-l border-zinc-100 dark:border-zinc-800">
                {HOURS.map((h) => (
                  <div
                    key={h}
                    style={{ height: ROW_H }}
                    onDragOver={(e) => { if (canWrite && dragId) { e.preventDefault(); setOver(`${key}-${h}`); } }}
                    onDragLeave={() => setOver((v) => (v === `${key}-${h}` ? null : v))}
                    onDrop={(e) => {
                      e.preventDefault();
                      const item = items.find((x) => x.id === dragId);
                      if (item) moveToSlot(item, day, h);
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

                {bgLayout.map(({ item, top, height, left, width }) => (
                  <div
                    key={item.id}
                    className="pointer-events-none absolute rounded-md border border-dashed border-zinc-300 bg-zinc-100/70 px-1.5 py-0.5 text-[9px] text-zinc-500 dark:border-zinc-700 dark:bg-zinc-900/60 dark:text-zinc-400"
                    style={{ top, height, left: `calc(${left}% + 2px)`, width: `calc(${width}% - 4px)` }}
                    title={`Evento da igreja: ${item.title}`}
                  >
                    <span className="line-clamp-2">{item.title}</span>
                  </div>
                ))}

                {layout.map(({ item, top, height, left, width }) => (
                  <div
                    key={item.id}
                    draggable={canWrite && item.draggable !== false}
                    onDragStart={() => setDragId(item.id)}
                    onDragEnd={finishDrag}
                    onClick={() => onOpen(item.raw)}
                    onDoubleClick={(e) => e.stopPropagation()}
                    className={cn(
                      "group absolute cursor-pointer overflow-hidden rounded-md px-1.5 py-0.5 text-[10px] text-white shadow-sm",
                      statusClass(item.status),
                    )}
                    style={{
                      top,
                      height,
                      left: `calc(${left}% + 2px)`,
                      width: `calc(${width}% - 4px)`,
                      backgroundColor: item.color,
                    }}
                    title={`${item.title}${item.subtitle ? ` · ${item.subtitle}` : ""}`}
                  >
                    <div className="flex items-center gap-1">
                      <span className="truncate font-medium">{item.title}</span>
                      {item.conflict && <AlertTriangle className="h-2.5 w-2.5 shrink-0 text-amber-200" />}
                      {canWrite && item.draggable !== false && (
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); onDelete(item.raw); }}
                          className="ml-auto hidden shrink-0 text-white/80 hover:text-white group-hover:block"
                          title="Excluir"
                        >
                          <Trash2 className="h-2.5 w-2.5" />
                        </button>
                      )}
                    </div>
                    <p className="truncate opacity-85">
                      {format(new Date(item.starts_at), "HH:mm")}
                      {item.ends_at ? ` - ${format(new Date(item.ends_at), "HH:mm")}` : ""}
                      {item.meta ? ` · ${item.meta}` : item.subtitle ? ` · ${item.subtitle}` : ""}
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

function layoutDay<T extends Dateable>(items: T[], day: Date) {
  const itemsWithTime = items
    .map((item) => {
      const start = new Date(item.starts_at);
      const end = item.ends_at ? new Date(item.ends_at) : new Date(start.getTime() + 60 * 60 * 1000);
      const dayStart = new Date(day.getFullYear(), day.getMonth(), day.getDate(), 0, 0);
      const s = Math.max(start.getTime(), dayStart.getTime() + DAY_START * 60000);
      const e = Math.min(end.getTime(), dayStart.getTime() + DAY_END * 60000);
      return { item, s, e };
    })
    .filter((x) => x.e > x.s)
    .sort((a, b) => a.s - b.s || b.e - a.e);

  const columns: number[] = [];
  const placed = itemsWithTime.map((it) => {
    let col = columns.findIndex((end) => end <= it.s);
    if (col === -1) { col = columns.length; columns.push(it.e); } else { columns[col] = it.e; }
    return { ...it, col };
  });
  const total = Math.max(columns.length, 1);
  const dayStartMs = new Date(day.getFullYear(), day.getMonth(), day.getDate(), 0, 0).getTime();

  return placed.map(({ item, s, e, col }) => ({
    item,
    top: ((s - dayStartMs) / 60000 - DAY_START) / 60 * ROW_H,
    height: Math.max(((e - s) / 60000) / 60 * ROW_H - 2, 18),
    left: (col / total) * 100,
    width: (1 / total) * 100,
  }));
}

function CalendarList<T>({
  items,
  backgroundItems,
  canWrite,
  onOpen,
  onDelete,
}: {
  items: CalendarItem<T>[];
  backgroundItems: CalendarBackgroundItem[];
  canWrite: boolean;
  onOpen: (item: T) => void;
  onDelete: (item: T) => void;
}) {
  const sorted = [...items].sort((a, b) => a.starts_at.localeCompare(b.starts_at));
  const bg = [...backgroundItems].sort((a, b) => a.starts_at.localeCompare(b.starts_at));

  return (
    <div className="space-y-3">
      {sorted.length === 0 ? (
        <p className="rounded-xl border border-dashed border-zinc-200 py-12 text-center text-sm text-zinc-400 dark:border-zinc-800">Nenhum item no periodo.</p>
      ) : (
        <div className="space-y-2">
          {sorted.map((item) => (
            <div
              key={item.id}
              className="flex cursor-pointer items-center gap-3 rounded-xl border border-zinc-200 bg-white p-3 hover:border-sky-200 dark:border-zinc-800 dark:bg-zinc-950"
              onClick={() => onOpen(item.raw)}
            >
              <span className="h-10 w-1 rounded-full" style={{ backgroundColor: item.color }} />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">
                  {item.title}
                  {item.conflict && <AlertTriangle className="ml-1 inline h-3.5 w-3.5 text-amber-600" />}
                </p>
                <p className="mt-0.5 flex flex-wrap items-center gap-x-3 text-xs text-zinc-400">
                  <span className="inline-flex items-center gap-1"><CalendarDays className="h-3 w-3" />{format(new Date(item.starts_at), "dd/MM/yyyy")}</span>
                  <span className="inline-flex items-center gap-1"><Clock className="h-3 w-3" />{format(new Date(item.starts_at), "HH:mm")}{item.ends_at ? ` - ${format(new Date(item.ends_at), "HH:mm")}` : ""}</span>
                  {item.subtitle && <span>{item.subtitle}</span>}
                  {item.meta && <span>{item.meta}</span>}
                </p>
              </div>
              {canWrite && item.draggable !== false && (
                <button
                  type="button"
                  onClick={(e) => { e.stopPropagation(); onDelete(item.raw); }}
                  className="rounded p-1.5 text-zinc-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/40"
                  title="Excluir"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {bg.length > 0 && (
        <div>
          <p className="mb-1 text-[11px] font-medium uppercase tracking-wide text-zinc-400">Eventos da igreja no periodo</p>
          <div className="space-y-1">
            {bg.map((b) => (
              <div key={b.id} className="flex items-center gap-2 rounded-lg border border-dashed border-zinc-200 px-3 py-1.5 text-xs text-zinc-400 dark:border-zinc-800">
                <CalendarDays className="h-3 w-3" />
                <span className="shrink-0">{format(new Date(b.starts_at), "dd/MM HH:mm")}</span>
                <span className="truncate">{b.title}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
