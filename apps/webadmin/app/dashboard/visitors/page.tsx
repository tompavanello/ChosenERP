"use client";

import { useEffect, useMemo, useState } from "react";
import { Search, UserPlus, DoorOpen, ArrowRight, Pencil, UserCheck } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Select } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge, type Tone } from "@/components/ui/badge";
import { StatCard } from "@/components/ui/stat-card";
import { Pagination } from "@/components/ui/pagination";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Drawer } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { listVisitors, createVisitor, updateVisitorStage, convertVisitorToMember, type Visitor } from "@/lib/api";
import { JOURNEY_STAGES } from "@/lib/constants";
import { datePt } from "@/lib/format";
import { MemberForm } from "@/components/members/member-form";
import { Modal } from "@/components/ui/modal";

const PER_PAGE = 12;
const ORDER = ["welcome", "coffee_pastor", "course", "cell", "converted"];
const SOURCES = ["indicado", "evento", "google", "porta", "redes sociais"];

export default function VisitorsPage() {
  const { toast } = useToast();
  const [visitors, setVisitors] = useState<Visitor[] | null>(null);
  const [query, setQuery] = useState("");
  const [stage, setStage] = useState("");
  const [page, setPage] = useState(1);
  const [open, setOpen] = useState(false);
  const [convertOpen, setConvertOpen] = useState(false);
  const [convertVisitor, setConvertVisitor] = useState<Visitor | null>(null);
  const [form, setForm] = useState({ first_name: "", last_name: "", email: "", phone: "", whatsapp: "", source: "" });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    listVisitors().then((r) => setVisitors(r.visitors)).catch((e) => toast(e.message, "error"));
  }, [toast]);

  const stats = useMemo(() => {
    const all = visitors ?? [];
    return {
      total: all.length,
      converted: all.filter((v) => v.journey_stage === "converted").length,
      inJourney: all.filter((v) => v.journey_stage !== "converted").length,
    };
  }, [visitors]);

  const filtered = useMemo(() => {
    if (!visitors) return [];
    const q = query.trim().toLowerCase();
    return visitors.filter((v) =>
      (!q || v.full_name.toLowerCase().includes(q) || (v.email?.toLowerCase().includes(q) ?? false)) &&
      (!stage || v.journey_stage === stage));
  }, [visitors, query, stage]);

  const pageItems = filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE);
  useEffect(() => setPage(1), [query, stage]);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createVisitor(Object.fromEntries(Object.entries(form).filter(([, v]) => v !== "")));
      toast("Visitante registrado.");
      setOpen(false);
      setVisitors(await listVisitors().then((r) => r.visitors));
      setForm({ first_name: "", last_name: "", email: "", phone: "", whatsapp: "", source: "" });
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function advance(v: Visitor) {
    const next = ORDER[Math.min(ORDER.indexOf(v.journey_stage) + 1, ORDER.length - 1)];
    if (next === v.journey_stage) return;
    try {
      await updateVisitorStage(v.id, next);
      setVisitors(await listVisitors().then((r) => r.visitors));
      toast(`Trilha avançada para "${JOURNEY_STAGES[next]?.label}".`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function convertToMember(visitor: Visitor, data: Record<string, unknown>) {
    setSaving(true);
    try {
      await convertVisitorToMember(visitor.id, data);
      toast(`${visitor.full_name} convertido para membro.`);
      setConvertOpen(false);
      setConvertVisitor(null);
      setVisitors(await listVisitors().then((r) => r.visitors));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Visitantes"
        description="Trilha de acolhimento e conversão"
        actions={<Button onClick={() => setOpen(true)}><UserPlus className="h-4 w-4" /> Novo Visitante</Button>}
      />

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <StatCard label="Visitantes" value={visitors ? String(stats.total) : "…"} icon={DoorOpen} />
        <StatCard label="Em trilha" value={visitors ? String(stats.inJourney) : "…"} tone="sky" />
        <StatCard label="Convertidos" value={visitors ? String(stats.converted) : "…"} tone="green" />
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-48">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
          <Input className="pl-9" placeholder="Buscar por nome ou e-mail" value={query} onChange={(e) => setQuery(e.target.value)} />
        </div>
        <Select className="w-52" value={stage} onChange={(e) => setStage(e.target.value)}>
          <option value="">Todas as etapas</option>
          {ORDER.map((k) => (<option key={k} value={k}>{JOURNEY_STAGES[k]?.label}</option>))}
        </Select>
      </div>

      <Card className="overflow-hidden p-0">
        {visitors === null ? (
          <SkeletonRows />
        ) : pageItems.length === 0 ? (
          <EmptyState icon={<DoorOpen className="h-10 w-10" />} title="Nenhum visitante" description="Registre e acompanhe a jornada de acolhimento." />
        ) : (
          <>
            <Table>
              <THead>
                <TRow><TH>Nome</TH><TH>Contato</TH><TH>Origem</TH><TH>Trilha de acolhimento</TH><TH>Entrada</TH><TH className="text-right">Ação</TH></TRow>
              </THead>
              <TBody>
                {pageItems.map((v) => {
                  const idx = ORDER.indexOf(v.journey_stage);
                  const st = JOURNEY_STAGES[v.journey_stage] ?? { label: v.journey_stage, tone: "zinc" as const };
                  return (
                    <TRow key={v.id}>
                      <TD className="font-medium">{v.full_name}</TD>
                      <TD>
                        <p>{v.phone ?? v.whatsapp ?? "—"}</p>
                        <p className="text-xs text-zinc-400">{v.email ?? ""}</p>
                      </TD>
                      <TD className="text-zinc-500">{v.source ?? "—"}</TD>
                      <TD>
                        <div className="flex items-center gap-2">
                          <div className="flex gap-1">
                            {ORDER.map((s, i) => (
                              <span key={s} title={JOURNEY_STAGES[s]?.label}
                                className={`h-1.5 w-6 rounded-full ${i <= idx ? "bg-sky-500" : "bg-zinc-200"}`} />
                            ))}
                          </div>
                           <Badge tone={st.tone as Tone}>{st.label}</Badge>
                        </div>
                      </TD>
                      <TD className="text-zinc-500">{datePt(v.created_at)}</TD>
                      <TD className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          {v.journey_stage === "converted" && (
                            <Button variant="ghost" size="sm" onClick={() => { setConvertVisitor(v); setConvertOpen(true); }} title="Converter para membro">
                              <UserCheck className="h-3.5 w-3.5 text-green-600" />
                            </Button>
                          )}
                          <Button variant="ghost" size="sm" onClick={() => advance(v)}>
                            Avançar <ArrowRight className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </TD>
                    </TRow>
                  );
                })}
              </TBody>
            </Table>
            <div className="border-t border-zinc-100">
              <Pagination page={page} total={filtered.length} perPage={PER_PAGE} onChange={setPage} />
            </div>
          </>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} title="Novo Visitante">
        <form onSubmit={add} className="grid grid-cols-2 gap-3">
          <Field label="Nome *"><Input required value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></Field>
          <Field label="Sobrenome *"><Input required value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></Field>
          <Field label="E-mail" className="col-span-2"><Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
          <Field label="Telefone"><Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
          <Field label="WhatsApp"><Input value={form.whatsapp} onChange={(e) => setForm({ ...form, whatsapp: e.target.value })} /></Field>
          <Field label="Como nos conheceu?" className="col-span-2">
            <Select value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value })}>
              <option value="">—</option>
              {SOURCES.map((s) => (<option key={s} value={s}>{s}</option>))}
            </Select>
          </Field>
          <div className="col-span-2 flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit">Salvar</Button>
          </div>
        </form>
      </Drawer>

      {convertVisitor && (
        <Modal open={convertOpen} onClose={() => { setConvertOpen(false); setConvertVisitor(null); }} title={`Converter ${convertVisitor.full_name} para membro`}>
          <MemberForm
            visitor={convertVisitor}
            onSubmit={async (data) => convertToMember(convertVisitor, data)}
            onCancel={() => { setConvertOpen(false); setConvertVisitor(null); }}
            saving={saving}
            submitLabel="Converter"
          />
          </Modal>
      )}
    </div>
  );
}
