"use client";

import { useCallback, useEffect, useState } from "react";
import { Heart, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Select, Textarea } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import {
  VISIBILITY_LABELS,
  createPrayer,
  datePt,
  getMyPrayers,
  getPrayerWall,
  reactPrayer,
  type PrayerRequest,
} from "@/lib/api";

type Tab = "wall" | "mine";

export default function OracaoPage() {
  const [tab, setTab] = useState<Tab>("wall");
  const [wall, setWall] = useState<PrayerRequest[]>([]);
  const [mine, setMine] = useState<PrayerRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [body, setBody] = useState("");
  const [visibility, setVisibility] = useState("igreja");
  const [anonymous, setAnonymous] = useState(false);
  const [sending, setSending] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const load = useCallback(async () => {
    const [w, m] = await Promise.all([
      getPrayerWall().catch(() => []),
      getMyPrayers().catch(() => []),
    ]);
    setWall(w);
    setMine(m);
  }, []);

  useEffect(() => {
    load().finally(() => setLoading(false));
  }, [load]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!body.trim()) return;
    setSending(true);
    setMsg(null);
    try {
      await createPrayer({ body: body.trim(), visibility, is_anonymous: anonymous });
      setBody("");
      setAnonymous(false);
      setVisibility("igreja");
      setMsg("Pedido enviado. Deus abençoe!");
      await load();
      setTab("mine");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Não foi possível enviar.");
    } finally {
      setSending(false);
    }
  }

  async function react(id: string) {
    try {
      const updated = await reactPrayer(id);
      setWall((prev) => prev.map((p) => (p.id === id ? updated : p)));
      setMine((prev) => prev.map((p) => (p.id === id ? updated : p)));
    } catch {
      /* silencioso */
    }
  }

  const list = tab === "wall" ? wall : mine;

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Pedidos de oração</h1>

      <Card>
        <form onSubmit={submit} className="space-y-3">
          <Field label="Seu pedido">
            <Textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="Compartilhe seu pedido, agradecimento ou motivo de oração..."
              maxLength={2000}
            />
          </Field>
          <Field label="Quem pode ver">
            <Select value={visibility} onChange={(e) => setVisibility(e.target.value)}>
              {Object.entries(VISIBILITY_LABELS).map(([k, v]) => (
                <option key={k} value={k}>
                  {v}
                </option>
              ))}
            </Select>
          </Field>
          <label className="flex items-center gap-2 text-sm text-[var(--muted)]">
            <input
              type="checkbox"
              checked={anonymous}
              onChange={(e) => setAnonymous(e.target.checked)}
              className="h-4 w-4"
            />
            Enviar como anônimo
          </label>
          {msg && <p className="text-sm text-[var(--brand)]">{msg}</p>}
          <Button type="submit" disabled={sending || !body.trim()} className="w-full">
            <Send className="h-4 w-4" />
            {sending ? "Enviando..." : "Enviar pedido"}
          </Button>
        </form>
      </Card>

      <div className="flex gap-2">
        <button
          onClick={() => setTab("wall")}
          className={`flex-1 rounded-lg py-2 text-sm font-medium ${
            tab === "wall" ? "bg-[var(--brand)] text-white" : "bg-black/5 text-[var(--muted)]"
          }`}
        >
          Mural
        </button>
        <button
          onClick={() => setTab("mine")}
          className={`flex-1 rounded-lg py-2 text-sm font-medium ${
            tab === "mine" ? "bg-[var(--brand)] text-white" : "bg-black/5 text-[var(--muted)]"
          }`}
        >
          Meus pedidos
        </button>
      </div>

      {loading ? (
        <SkeletonRows rows={3} />
      ) : list.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">
            {tab === "wall" ? "Nenhum pedido no mural." : "Você ainda não enviou pedidos."}
          </p>
        </Card>
      ) : (
        <div className="space-y-3">
          {list.map((p) => (
            <Card key={p.id}>
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm font-medium">
                  {p.is_anonymous ? "Anônimo" : p.author_name || "Membro"}
                </span>
                <span className="text-xs text-[var(--muted)]">{datePt(p.created_at)}</span>
              </div>
              <p className="mt-2 whitespace-pre-line text-sm">{p.body}</p>
              {p.answered_note && (
                <p className="mt-2 rounded-lg bg-[var(--brand-soft)] p-2 text-sm text-[var(--brand-strong)]">
                  {p.answered_note}
                </p>
              )}
              <div className="mt-3 flex items-center gap-3">
                <button
                  onClick={() => react(p.id)}
                  disabled={p.reacted_by_me}
                  className={`inline-flex items-center gap-1 text-sm ${
                    p.reacted_by_me ? "text-[var(--brand)]" : "text-[var(--muted)]"
                  }`}
                >
                  <Heart className={`h-4 w-4 ${p.reacted_by_me ? "fill-current" : ""}`} />
                  {p.reacted_by_me ? "Você está orando" : "Estou orando"}
                </button>
                <span className="text-xs text-[var(--muted)]">{p.prayer_count} orando</span>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
