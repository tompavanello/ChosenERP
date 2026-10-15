"use client";

import { useCallback, useEffect, useState } from "react";
import { Send, UserPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input, Textarea } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import { createVisitorRequest, datePt, getMyVisitorRequests, type MemberRequest } from "@/lib/api";

const STATUS_LABELS: Record<string, string> = {
  pendente: "Pendente",
  em_andamento: "Em andamento",
  concluida: "Concluída",
  cancelada: "Cancelada",
};

export default function VisitorSolicitacoesPage() {
  const [items, setItems] = useState<MemberRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [subject, setSubject] = useState("Quero ser membro");
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const load = useCallback(async () => {
    const list = await getMyVisitorRequests().catch(() => []);
    setItems(list);
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
      await createVisitorRequest({
        kind: "quero_ser_membro",
        subject: subject.trim(),
        message: message.trim() || undefined,
      });
      setMessage("");
      setMsg("Solicitação enviada! A secretaria entrará em contato.");
      await load();
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Não foi possível enviar.");
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Quero ser membro</h1>

      <Card>
        <form onSubmit={submit} className="space-y-3">
          <p className="flex items-center gap-2 text-sm text-[var(--muted)]">
            <UserPlus className="h-4 w-4" />
            Demonstre seu interesse e a secretaria dará os próximos passos.
          </p>
          <Field label="Assunto">
            <Input value={subject} onChange={(e) => setSubject(e.target.value)} />
          </Field>
          <Field label="Mensagem (opcional)">
            <Textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Conte um pouco sobre você, se já participou de um encontro..."
            />
          </Field>
          {msg && <p className="text-sm text-[var(--brand)]">{msg}</p>}
          <Button type="submit" disabled={sending || !subject.trim()} className="w-full">
            <Send className="h-4 w-4" />
            {sending ? "Enviando..." : "Enviar solicitação"}
          </Button>
        </form>
      </Card>

      <section>
        <h2 className="mb-2 text-sm font-semibold text-[var(--muted)]">Minhas solicitações</h2>
        {loading ? (
          <SkeletonRows rows={2} />
        ) : items.length === 0 ? (
          <Card>
            <p className="text-sm text-[var(--muted)]">Você ainda não enviou solicitações.</p>
          </Card>
        ) : (
          <div className="space-y-2">
            {items.map((r) => (
              <Card key={r.id}>
                <div className="flex items-center justify-between gap-2">
                  <p className="text-sm font-medium">{r.subject}</p>
                  <span className="text-xs text-[var(--muted)]">
                    {STATUS_LABELS[r.status] ?? r.status}
                  </span>
                </div>
                <p className="mt-1 text-xs text-[var(--muted)]">{datePt(r.created_at)}</p>
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
