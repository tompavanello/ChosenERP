"use client";

import { useEffect, useState } from "react";
import { ArrowLeft, ChevronRight, Loader2, Send } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import {
  adminAddTicketMessage, adminGetTicket, adminListTickets, adminUpdateTicket,
  type SupportTicket,
} from "@/lib/api";

const STATUSES = ["", "aberto", "em_andamento", "aguardando", "resolvido", "fechado"];
const STATUS_LABEL: Record<string, string> = {
  aberto: "Aberto", em_andamento: "Em andamento", aguardando: "Aguardando", resolvido: "Resolvido", fechado: "Fechado",
};
const STATUS_TONE: Record<string, "sky" | "amber" | "green" | "zinc"> = {
  aberto: "sky", em_andamento: "amber", aguardando: "amber", resolvido: "green", fechado: "zinc",
};
const PRIORITIES = ["baixa", "normal", "alta", "urgente"];

export default function PlatformSupportPage() {
  const [status, setStatus] = useState("");
  const [tickets, setTickets] = useState<SupportTicket[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<SupportTicket | null>(null);
  const [reply, setReply] = useState("");
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  function load() {
    setLoading(true);
    adminListTickets(status || undefined)
      .then((res) => setTickets(res.tickets ?? []))
      .catch(() => setTickets([]))
      .finally(() => setLoading(false));
  }
  useEffect(load, [status]);

  async function open(id: string) {
    setMsg(null);
    try {
      setSelected(await adminGetTicket(id));
    } catch {
      setMsg("Nao foi possivel abrir o chamado.");
    }
  }

  async function patch(body: Record<string, unknown>) {
    if (!selected) return;
    setSaving(true);
    try {
      const updated = await adminUpdateTicket(selected.id, body);
      setSelected(updated);
      setTickets((list) => list.map((t) => (t.id === updated.id ? updated : t)));
      setMsg("Chamado atualizado.");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Falha ao atualizar.");
    } finally {
      setSaving(false);
    }
  }

  async function send(e: React.FormEvent) {
    e.preventDefault();
    if (!selected || !reply.trim()) return;
    setSaving(true);
    try {
      const message = await adminAddTicketMessage(selected.id, reply.trim());
      setSelected({ ...selected, messages: [...(selected.messages ?? []), message] });
      setReply("");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Falha ao enviar.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page space-y-4">
      <PageHeader title="Suporte" description="Chamados abertos pela Central de Ajuda do webadmin e do app do membro." />

      {selected ? (
        <div className="space-y-4">
          <button onClick={() => setSelected(null)} className="inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-sky-600">
            <ArrowLeft className="h-3.5 w-3.5" /> Voltar para a lista
          </button>
          <Card className="p-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 className="text-lg font-semibold">{selected.subject}</h3>
                <p className="text-xs text-zinc-500">
                  {selected.user_name ?? selected.user_id} · {selected.user_email ?? "-"} · {selected.app} · {selected.page_path || "-"}
                </p>
                <p className="text-xs text-zinc-500">Aberto em {new Date(selected.created_at).toLocaleString("pt-BR")}</p>
              </div>
              <div className="flex items-center gap-2">
                <Badge tone={STATUS_TONE[selected.status] ?? "zinc"}>{STATUS_LABEL[selected.status] ?? selected.status}</Badge>
                <Badge tone={selected.priority === "urgente" ? "red" : selected.priority === "alta" ? "amber" : "zinc"}>{selected.priority}</Badge>
              </div>
            </div>
            <p className="mt-3 whitespace-pre-wrap text-sm">{selected.description}</p>
            {selected.ai_summary && (
              <div className="mt-3 rounded-lg bg-sky-50 p-3 text-xs text-sky-900 dark:bg-sky-950/30 dark:text-sky-100">
                <p className="mb-1 font-semibold">Resumo da conversa com a IA</p>
                <p className="whitespace-pre-wrap">{selected.ai_summary}</p>
              </div>
            )}
            {selected.context && Object.keys(selected.context).length > 0 && (
              <details className="mt-3">
                <summary className="cursor-pointer text-xs text-zinc-500">Contexto tecnico</summary>
                <pre className="mt-2 overflow-x-auto rounded bg-zinc-100 p-2 text-[11px] dark:bg-zinc-800">{JSON.stringify(selected.context, null, 2)}</pre>
              </details>
            )}
          </Card>

          <Card className="p-4">
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-400">Conversa</p>
            <div className="space-y-2">
              {(selected.messages ?? []).length === 0 && <p className="text-sm text-zinc-500">Sem mensagens ainda.</p>}
              {(selected.messages ?? []).map((m) => (
                <div key={m.id} className={m.author_kind === "platform" ? "rounded-lg bg-emerald-50 p-2.5 text-sm dark:bg-emerald-950/30" : "rounded-lg bg-zinc-100 p-2.5 text-sm dark:bg-zinc-800"}>
                  <p className="mb-1 text-[10px] uppercase tracking-wide text-zinc-400">
                    {m.author_kind === "platform" ? "Suporte" : m.author_kind === "ai" ? "Assistente" : (m.author_name ?? "Usuario")} · {new Date(m.created_at).toLocaleString("pt-BR")}
                  </p>
                  <p className="whitespace-pre-wrap">{m.body}</p>
                </div>
              ))}
            </div>
            <form onSubmit={send} className="mt-3 flex items-end gap-2 border-t border-zinc-200 pt-3 dark:border-zinc-800">
              <Textarea value={reply} onChange={(e) => setReply(e.target.value)} rows={2} placeholder="Responder ao usuario..." className="min-h-[44px] resize-none" />
              <Button type="submit" disabled={saving || !reply.trim()} className="h-10 shrink-0 px-3"><Send className="h-4 w-4" /></Button>
            </form>
          </Card>

          <Card className="flex flex-wrap items-center gap-2 p-3">
            <span className="text-xs font-medium text-zinc-500">Status:</span>
            {["em_andamento", "aguardando", "resolvido", "fechado"].map((s) => (
              <Button key={s} size="sm" variant={selected.status === s ? "primary" : "outline"} disabled={saving} onClick={() => patch({ status: s })}>
                {STATUS_LABEL[s]}
              </Button>
            ))}
            <span className="ml-3 text-xs font-medium text-zinc-500">Prioridade:</span>
            {PRIORITIES.map((p) => (
              <Button key={p} size="sm" variant={selected.priority === p ? "primary" : "outline"} disabled={saving} onClick={() => patch({ priority: p })}>
                {p}
              </Button>
            ))}
            {msg && <span className="text-xs text-zinc-500">{msg}</span>}
          </Card>
        </div>
      ) : (
        <>
          <Card className="flex flex-wrap items-center gap-2 p-3">
            <span className="text-xs font-medium text-zinc-500">Filtrar:</span>
            {STATUSES.map((s) => (
              <Button key={s || "todos"} size="sm" variant={status === s ? "primary" : "outline"} onClick={() => setStatus(s)}>
                {s ? STATUS_LABEL[s] : "Todos"}
              </Button>
            ))}
          </Card>

          {loading ? (
            <Card className="p-4"><SkeletonRows rows={6} /></Card>
          ) : tickets.length === 0 ? (
            <Card className="p-8 text-center text-sm text-zinc-500">Nenhum chamado encontrado.</Card>
          ) : (
            <Card className="divide-y divide-zinc-100 p-0 dark:divide-zinc-800">
              {tickets.map((t) => (
                <button key={t.id} onClick={() => open(t.id)} className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left hover:bg-zinc-50 dark:hover:bg-zinc-900">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{t.subject}</p>
                    <p className="truncate text-xs text-zinc-500">
                      {t.user_name ?? t.user_id} · {t.app} · {t.page_path || "-"} · {new Date(t.created_at).toLocaleString("pt-BR")}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <Badge tone={STATUS_TONE[t.status] ?? "zinc"}>{STATUS_LABEL[t.status] ?? t.status}</Badge>
                    <ChevronRight className="h-4 w-4 text-zinc-300" />
                  </div>
                </button>
              ))}
            </Card>
          )}
        </>
      )}
    </div>
  );
}
