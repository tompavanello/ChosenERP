"use client";

import { useCallback, useEffect, useState } from "react";
import { Inbox, MessageSquare } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Field, Select, Textarea } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { listRequests, updateRequest, type MemberRequest } from "@/lib/api";
import { dateTimePt } from "@/lib/format";

const KIND_LABELS: Record<string, string> = {
  cadastro: "Atualização cadastral",
  carta: "Carta",
  visita: "Visita pastoral",
  batismo: "Batismo",
  profissao_fe: "Profissão de fé",
  transferencia: "Transferência",
  casamento: "Casamento",
  evento: "Inscrição em evento",
  outro: "Outro",
};
const STATUS_LABELS: Record<string, string> = {
  pendente: "Pendente",
  em_andamento: "Em andamento",
  concluida: "Concluída",
  cancelada: "Cancelada",
};
const STATUS_TONE: Record<string, "amber" | "sky" | "green" | "zinc"> = {
  pendente: "amber",
  em_andamento: "sky",
  concluida: "green",
  cancelada: "zinc",
};

export default function RequestsPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("members.write");

  const [items, setItems] = useState<MemberRequest[] | null>(null);
  const [filter, setFilter] = useState("");
  const [editing, setEditing] = useState<MemberRequest | null>(null);
  const [form, setForm] = useState({ status: "em_andamento", response: "" });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      setItems((await listRequests(filter || undefined)).requests);
    } catch {
      setItems([]);
    }
  }, [filter]);

  useEffect(() => {
    load();
  }, [load]);

  function openRespond(r: MemberRequest) {
    setEditing(r);
    setForm({ status: r.status === "pendente" ? "em_andamento" : r.status, response: r.response ?? "" });
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!editing) return;
    setSaving(true);
    try {
      await updateRequest(editing.id, { status: form.status, response: form.response });
      toast("Solicitação atualizada.");
      setEditing(null);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page">
      <PageHeader title="Solicitações" description="Pedidos e formulários enviados pelos membros pelo app" />

      <div className="mb-4 flex items-center gap-2">
        <Select className="h-8 w-48 text-sm" value={filter} onChange={(e) => setFilter(e.target.value)}>
          <option value="">Todos os status</option>
          {Object.entries(STATUS_LABELS).map(([k, v]) => (
            <option key={k} value={k}>{v}</option>
          ))}
        </Select>
      </div>

      <Card className="overflow-hidden p-0">
        {items === null ? (
          <SkeletonRows rows={5} />
        ) : items.length === 0 ? (
          <EmptyState icon={<Inbox className="h-10 w-10" />} title="Nenhuma solicitação" description="Os pedidos enviados pelo app aparecem aqui." />
        ) : (
          <Table>
            <THead><TRow><TH>Solicitante</TH><TH>Tipo</TH><TH>Assunto</TH><TH>Status</TH><TH>Data</TH><TH className="text-right">Ações</TH></TRow></THead>
            <TBody>
              {items.map((r) => (
                <TRow key={r.id}>
                  <TD className="font-medium">{r.member_name ?? "-"}</TD>
                  <TD className="text-zinc-500">{KIND_LABELS[r.kind] ?? r.kind}</TD>
                  <TD>
                    <div>{r.subject}</div>
                    {r.message && <div className="line-clamp-1 text-xs text-zinc-500">{r.message}</div>}
                  </TD>
                  <TD><Badge tone={STATUS_TONE[r.status] ?? "zinc"}>{STATUS_LABELS[r.status] ?? r.status}</Badge></TD>
                  <TD className="text-xs text-zinc-500">{dateTimePt(r.created_at)}</TD>
                  <TD className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => openRespond(r)}><MessageSquare className="h-3.5 w-3.5" /> Responder</Button>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Modal open={!!editing} onClose={() => setEditing(null)} title={`Solicitação - ${editing?.subject ?? ""}`}>
        {editing && (
          <div className="space-y-3">
            <div className="rounded-lg bg-zinc-50 p-3 text-sm dark:bg-zinc-900">
              <div className="font-medium">{KIND_LABELS[editing.kind] ?? editing.kind}</div>
              {editing.message && <p className="mt-1 whitespace-pre-line text-zinc-600 dark:text-zinc-300">{editing.message}</p>}
            </div>
            <form onSubmit={save} className="space-y-3">
              <Field label="Status">
                <Select className="h-8 text-sm" value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>
                  {Object.entries(STATUS_LABELS).map(([k, v]) => (
                    <option key={k} value={k}>{v}</option>
                  ))}
                </Select>
              </Field>
              <Field label="Resposta (visível ao membro)">
                <Textarea rows={3} value={form.response} onChange={(e) => setForm({ ...form, response: e.target.value })} />
              </Field>
              <div className="flex justify-end gap-2">
                <Button variant="ghost" type="button" onClick={() => setEditing(null)}>Cancelar</Button>
                <Button type="submit" disabled={!canWrite || saving}>{saving ? "Salvando..." : "Salvar"}</Button>
              </div>
            </form>
          </div>
        )}
      </Modal>
    </div>
  );
}
