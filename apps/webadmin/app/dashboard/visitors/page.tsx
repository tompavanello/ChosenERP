"use client";

import { useEffect, useMemo, useState } from "react";
import { Search, UserPlus, DoorOpen, ArrowRight } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Pagination } from "@/components/ui/pagination";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { listVisitors, createVisitor, updateVisitorStage, type Visitor } from "@/lib/api";
import { JOURNEY_STAGES } from "@/lib/constants";
import { datePt } from "@/lib/format";

const PER_PAGE = 12;

export default function VisitorsPage() {
  const { toast } = useToast();
  const [visitors, setVisitors] = useState<Visitor[] | null>(null);
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ first_name: "", last_name: "", phone: "", source: "" });

  useEffect(() => {
    listVisitors().then((r) => setVisitors(r.visitors)).catch((e) => toast(e.message, "error"));
  }, [toast]);

  const filtered = useMemo(() => {
    if (!visitors) return [];
    const q = query.trim().toLowerCase();
    return visitors.filter((v) => !q || v.full_name.toLowerCase().includes(q));
  }, [visitors, query]);

  const pageItems = filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE);
  useEffect(() => setPage(1), [query]);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createVisitor(form);
      toast("Visitante registrado.");
      setOpen(false);
      setVisitors(await listVisitors().then((r) => r.visitors));
      setForm({ first_name: "", last_name: "", phone: "", source: "" });
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function advance(v: Visitor) {
    const order = ["welcome", "coffee_pastor", "course", "cell", "converted"];
    const idx = order.indexOf(v.journey_stage);
    const next = order[Math.min(idx + 1, order.length - 1)];
    if (next === v.journey_stage) return;
    try {
      await updateVisitorStage(v.id, next);
      setVisitors(await listVisitors().then((r) => r.visitors));
      toast(`Trilha avançada para "${JOURNEY_STAGES[next]?.label}".`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="mx-auto max-w-6xl">
      <PageHeader
        title="Visitantes"
        description="Trilha de acolhimento e onboarding"
        actions={<Button onClick={() => setOpen(true)}><UserPlus className="h-4 w-4" /> Novo Visitante</Button>}
      />

      <div className="mb-4 relative max-w-sm">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
        <Input className="pl-9" placeholder="Buscar por nome" value={query} onChange={(e) => setQuery(e.target.value)} />
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
                <TRow><TH>Nome</TH><TH>Contato</TH><TH>Origem</TH><TH>Etapa</TH><TH>Entrada</TH><TH className="text-right">Ação</TH></TRow>
              </THead>
              <TBody>
                {pageItems.map((v) => {
                  const st = JOURNEY_STAGES[v.journey_stage] ?? { label: v.journey_stage, tone: "zinc" as const };
                  return (
                    <TRow key={v.id}>
                      <TD className="font-medium">{v.full_name}</TD>
                      <TD>
                        <p>{v.phone ?? "—"}</p>
                        <p className="text-xs text-zinc-400">{v.email ?? ""}</p>
                      </TD>
                      <TD className="text-zinc-500">{v.source ?? "—"}</TD>
                      <TD><Badge tone={st.tone as "zinc"}>{st.label}</Badge></TD>
                      <TD className="text-zinc-500">{datePt(v.created_at)}</TD>
                      <TD className="text-right">
                        <Button variant="ghost" size="sm" onClick={() => advance(v)}>
                          Avançar <ArrowRight className="h-3.5 w-3.5" />
                        </Button>
                      </TD>
                    </TRow>
                  );
                })}
              </TBody>
            </Table>
            <div className="border-t border-zinc-100 dark:border-zinc-800">
              <Pagination page={page} total={filtered.length} perPage={PER_PAGE} onChange={setPage} />
            </div>
          </>
        )}
      </Card>

      <Modal open={open} onClose={() => setOpen(false)} title="Novo Visitante">
        <form onSubmit={add} className="grid grid-cols-2 gap-3">
          <Field label="Nome *"><Input required value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></Field>
          <Field label="Sobrenome *"><Input required value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></Field>
          <Field label="Telefone"><Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
          <Field label="Origem"><Input placeholder="evento, indicação..." value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value })} /></Field>
          <div className="col-span-2 flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit">Salvar</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
