"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { CalendarDays, Trophy, TrendingUp, Users, UserCheck } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input, Select } from "@/components/ui/input";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { StatCard } from "@/components/ui/stat-card";
import { useToast } from "@/components/ui/toast";
import { ExportButtons } from "@/components/reports/export-buttons";
import {
  getAttendanceReport, listEventKinds,
  type AttendanceEventRow, type AttendanceReport, type EventKind,
} from "@/lib/api";
import { dateTimePt, monthLabel, number, todayISO } from "@/lib/format";

const ANO = new Date().getFullYear();

export default function AttendanceReportPage() {
  const { toast } = useToast();
  const [from, setFrom] = useState(`${ANO}-01-01`);
  const [to, setTo] = useState(todayISO());
  const [kind, setKind] = useState("");
  const [kinds, setKinds] = useState<EventKind[]>([]);
  const [rep, setRep] = useState<AttendanceReport | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listEventKinds().then((r) => setKinds(r.kinds)).catch(() => setKinds([]));
  }, []);

  const load = useCallback(async (f: string, t: string, k: string) => {
    setLoading(true);
    try {
      const data = await getAttendanceReport({ from: f, to: t, kind: k });
      setRep(data);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
      setRep(null);
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    load(from, to, kind);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const params = useMemo(() => {
    const p: Record<string, string> = {};
    if (from) p.from = from;
    if (to) p.to = to;
    if (kind) p.kind = kind;
    return p;
  }, [from, to, kind]);

  const s = rep?.summary;

  return (
    <div className="page">
      <PageHeader
        title="Participantes por evento"
        description="Publico por evento e por tipo, comparado com o mesmo periodo do ano anterior"
        actions={
          <ExportButtons
            path="/api/v1/reports/attendance/export"
            params={params}
            filenameBase={`participantes-${from || "inicio"}-${to || "hoje"}`}
          />
        }
      />

      <Card className="mb-6 flex flex-wrap items-end gap-3">
        <div><label className="label">De</label><Input type="date" value={from} onChange={(e) => setFrom(e.target.value)} /></div>
        <div><label className="label">Ate</label><Input type="date" value={to} onChange={(e) => setTo(e.target.value)} /></div>
        <div>
          <label className="label">Tipo de evento</label>
          <Select value={kind} onChange={(e) => setKind(e.target.value)}>
            <option value="">Todos</option>
            {kinds.map((k) => <option key={k.id} value={k.id}>{k.name}</option>)}
          </Select>
        </div>
        <Button onClick={() => load(from, to, kind)}>Filtrar</Button>
        <Button variant="ghost" onClick={() => { setFrom(`${ANO}-01-01`); setTo(todayISO()); setKind(""); load(`${ANO}-01-01`, todayISO(), ""); }}>
          Ano atual
        </Button>
      </Card>

      {loading ? (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-20" />)}</div>
          <div className="grid gap-6 lg:grid-cols-2">{[0, 1].map((i) => <Skeleton key={i} className="h-72" />)}</div>
          <Skeleton className="h-64" />
        </div>
      ) : !rep ? (
        <Card><p className="py-8 text-center text-sm text-zinc-400">Nao foi possivel carregar o relatorio.</p></Card>
      ) : (
        <>
          {s && (
            <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <StatCard
                label="Participantes"
                value={number(s.participants)}
                icon={Users}
                tone="sky"
                hint={`${s.delta_participants_pct >= 0 ? "+" : ""}${s.delta_participants_pct.toFixed(1)}% vs. ano anterior`}
              />
              <StatCard
                label="Eventos realizados"
                value={number(s.events)}
                icon={CalendarDays}
                tone="indigo"
                hint={`${s.delta_events_pct >= 0 ? "+" : ""}${s.delta_events_pct.toFixed(1)}% vs. ano anterior`}
              />
              <StatCard
                label="Media por evento"
                value={s.avg_per_event.toFixed(1)}
                icon={TrendingUp}
                tone="green"
                hint={`${s.delta_avg_pct >= 0 ? "+" : ""}${s.delta_avg_pct.toFixed(1)}% vs. ano anterior`}
              />
              <StatCard
                label="Maior publico"
                value={rep.top_events[0] ? number(rep.top_events[0].participants) : "0"}
                icon={Trophy}
                tone="amber"
                hint={rep.top_events[0]?.title ?? "Sem eventos no periodo"}
              />
            </div>
          )}

          <div className="mb-6 grid gap-6 lg:grid-cols-2">
            <Card>
              <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Evolucao mensal de participantes</h3>
              <div className="h-72">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={rep.monthly.map((m) => ({ name: monthLabel(m.month), Atual: m.participants, "Ano anterior": m.prev_participants }))}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                    <XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                    <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" allowDecimals={false} />
                    <Tooltip />
                    <Legend />
                    <Bar dataKey="Atual" fill="#0ea5e9" />
                    <Bar dataKey="Ano anterior" fill="#cbd5e1" />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>

            <Card>
              <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Participantes por tipo de evento</h3>
              <div className="h-72">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={rep.by_kind.map((k) => ({ name: k.kind_name, Atual: k.participants, "Ano anterior": k.prev_participants }))} layout="vertical" margin={{ left: 8, right: 8 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" horizontal={false} />
                    <XAxis type="number" tick={{ fontSize: 12 }} stroke="#9ca3af" allowDecimals={false} />
                    <YAxis type="category" dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" width={90} />
                    <Tooltip />
                    <Legend />
                    <Bar dataKey="Atual" fill="#8b5cf6" />
                    <Bar dataKey="Ano anterior" fill="#ddd6fe" />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          </div>

          <Card className="mb-6 overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">
              Eventos com maior numero de participantes
            </h3>
            <Table>
              <THead>
                <TRow>
                  <TH className="w-10 text-right">#</TH>
                  <TH>Data</TH>
                  <TH>Evento</TH>
                  <TH>Tipo</TH>
                  <TH>Filial</TH>
                  <TH className="text-right">Participantes</TH>
                  <TH className="text-right">Convidados</TH>
                  <TH>Modo</TH>
                </TRow>
              </THead>
              <TBody>
                {rep.top_events.map((e, i) => (
                  <TRow key={e.id}>
                    <TD className="text-right text-zinc-400">{i + 1}</TD>
                    <TD className="whitespace-nowrap">{dateTimePt(e.starts_at)}</TD>
                    <TD className="font-medium">{e.title}</TD>
                    <TD>{e.kind_name}</TD>
                    <TD>{e.branch_name}</TD>
                    <TD className="text-right font-semibold">{number(e.participants)}</TD>
                    <TD className="text-right text-zinc-500">{number(e.invited_count)}</TD>
                    <TD><ModeBadge mode={e.attendance_mode} /></TD>
                  </TRow>
                ))}
                {rep.top_events.length === 0 && <TRow><TD colSpan={8} className="py-8 text-center text-zinc-400">Sem eventos no periodo.</TD></TRow>}
              </TBody>
            </Table>
          </Card>

          <Card className="mb-6 overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">
              Comparativo por tipo de evento
            </h3>
            <Table>
              <THead>
                <TRow>
                  <TH>Tipo</TH>
                  <TH className="text-right">Eventos</TH>
                  <TH className="text-right">Participantes</TH>
                  <TH className="text-right">Media</TH>
                  <TH className="text-right">Participantes (ano anterior)</TH>
                  <TH className="text-right">Variacao</TH>
                </TRow>
              </THead>
              <TBody>
                {rep.by_kind.map((k) => (
                  <TRow key={k.kind_id || k.kind_name}>
                    <TD className="font-medium">{k.kind_name}</TD>
                    <TD className="text-right">{number(k.events)}</TD>
                    <TD className="text-right font-semibold">{number(k.participants)}</TD>
                    <TD className="text-right">{k.avg_per_event.toFixed(1)}</TD>
                    <TD className="text-right text-zinc-500">{number(k.prev_participants)}</TD>
                    <TD className="text-right"><Delta value={k.delta_pct} /></TD>
                  </TRow>
                ))}
                {rep.by_kind.length === 0 && <TRow><TD colSpan={6} className="py-8 text-center text-zinc-400">Sem eventos no periodo.</TD></TRow>}
              </TBody>
            </Table>
          </Card>

          <Card className="overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">
              Todos os eventos do periodo
            </h3>
            <Table>
              <THead>
                <TRow>
                  <TH>Data</TH>
                  <TH>Evento</TH>
                  <TH>Tipo</TH>
                  <TH>Filial</TH>
                  <TH className="text-right">Participantes</TH>
                  <TH className="text-right">Chamada</TH>
                  <TH className="text-right">Convidados</TH>
                  <TH>Modo</TH>
                </TRow>
              </THead>
              <TBody>
                {rep.events.map((e) => (
                  <TRow key={e.id}>
                    <TD className="whitespace-nowrap">{dateTimePt(e.starts_at)}</TD>
                    <TD className="font-medium">{e.title}</TD>
                    <TD>{e.kind_name}</TD>
                    <TD>{e.branch_name}</TD>
                    <TD className="text-right font-semibold">{number(e.participants)}</TD>
                    <TD className="text-right text-zinc-500">{e.attendance_mode === "nominal" ? number(e.attendance_count) : "-"}</TD>
                    <TD className="text-right text-zinc-500">{number(e.invited_count)}</TD>
                    <TD><ModeBadge mode={e.attendance_mode} /></TD>
                  </TRow>
                ))}
                {rep.events.length === 0 && <TRow><TD colSpan={8} className="py-8 text-center text-zinc-400">Sem eventos no periodo.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </>
      )}
    </div>
  );
}

function ModeBadge({ mode }: { mode: AttendanceEventRow["attendance_mode"] }) {
  return (
    <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${mode === "nominal" ? "bg-sky-50 text-sky-700" : "bg-zinc-100 text-zinc-600"}`}>
      <UserCheck className="h-3 w-3" />
      {mode === "nominal" ? "Chamada" : "Contagem"}
    </span>
  );
}

function Delta({ value }: { value: number }) {
  const pos = value >= 0;
  return (
    <span className={pos ? "text-emerald-600" : "text-red-600"}>
      {pos ? "+" : ""}{value.toFixed(1)}%
    </span>
  );
}
