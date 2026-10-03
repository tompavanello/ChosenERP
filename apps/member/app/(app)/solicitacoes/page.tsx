"use client";

import { useCallback, useEffect, useState } from "react";
import { Inbox, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import { createRequest, datePt, getMyRequests, type MemberRequest } from "@/lib/api";

const KIND_LABELS: Record<string, string> = {
  cadastro: "Atualização cadastral",
  carta: "Carta / documento",
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

export default function SolicitacoesPage() {
  const [items, setItems] = useState<MemberRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [kind, setKind] = useState("cadastro");
  const [subject, setSubject] = useState("");
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setItems(await getMyRequests());
    } catch {
      setItems([]);
    }
  }, []);

  useEffect(() => {
    load().finally(() => setLoading(false));
  }, [load]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!subject.trim()) return;
    setSending(true);
    setMsg(null);
    try {
      await createRequest({ kind, subject: subject.trim(), message: message.trim() });
      setSubject("");
      setMessage("");
      setKind("cadastro");
      setMsg("Solicitação enviada. A secretaria responderá em breve.");
      await load();
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Não foi possível enviar.");
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Solicitações</h1>

      <Card>
        <form onSubmit={submit} className="space-y-3">
          <Field label="Tipo">
            <Select value={kind} onChange={(e) => setKind(e.target.value)}>
              {Object.entries(KIND_LABELS).map(([k, v]) => (
                <option key={k} value={k}>{v}</option>
              ))}
            </Select>
          </Field>
          <Field label="Assunto">
            <Input
              required
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Resuma seu pedido"
            />
          </Field>
          <Field label="Detalhes (opcional)">
            <Textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Descreva sua solicitação..."
              maxLength={2000}
            />
          </Field>
          {msg && <p className="text-sm text-[var(--brand)]">{msg}</p>}
          <Button type="submit" className="w-full" disabled={sending || !subject.trim()}>
            <Send className="h-4 w-4" />
            {sending ? "Enviando..." : "Enviar solicitação"}
          </Button>
        </form>
      </Card>

      <section>
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-[var(--muted)]">
          <Inbox className="h-4 w-4" /> Minhas solicitações
        </h2>
        {loading ? (
          <SkeletonRows rows={3} />
        ) : items.length === 0 ? (
          <Card>
            <p className="text-sm text-[var(--muted)]">Você ainda não enviou solicitações.</p>
          </Card>
        ) : (
          <div className="space-y-3">
            {items.map((r) => (
              <Card key={r.id}>
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <p className="font-medium">{r.subject}</p>
                    <p className="text-xs text-[var(--muted)]">
                      {KIND_LABELS[r.kind] ?? r.kind} · {datePt(r.created_at)}
                    </p>
                  </div>
                  <Badge>{STATUS_LABELS[r.status] ?? r.status}</Badge>
                </div>
                {r.message && <p className="mt-2 whitespace-pre-line text-sm text-[var(--muted)]">{r.message}</p>}
                {r.response && (
                  <p className="mt-2 rounded-lg bg-[var(--brand-soft)] p-2 text-sm text-[var(--brand-strong)]">
                    {r.response}
                  </p>
                )}
              </Card>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
