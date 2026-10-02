"use client";

import { useEffect, useState } from "react";
import { Bell, BellOff, BellRing } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { getPushPublicKey, subscribePush, unsubscribePush } from "@/lib/api";

function urlBase64ToUint8Array(base64String: string): ArrayBuffer {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(base64);
  const buffer = new ArrayBuffer(raw.length);
  const bytes = new Uint8Array(buffer);
  for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i);
  return buffer;
}

/** Ativa/desativa notificacoes Web Push no aparelho do membro. */
export function PushToggle() {
  const [supported, setSupported] = useState(true);
  const [enabled, setEnabled] = useState(false);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  useEffect(() => {
    const ok =
      typeof window !== "undefined" &&
      "serviceWorker" in navigator &&
      "PushManager" in window &&
      "Notification" in window;
    setSupported(ok);
    if (!ok) return;
    navigator.serviceWorker.ready
      .then((reg) => reg.pushManager.getSubscription())
      .then((sub) => setEnabled(!!sub))
      .catch(() => {});
  }, []);

  async function enable() {
    setBusy(true);
    setMsg(null);
    try {
      const perm = await Notification.requestPermission();
      if (perm !== "granted") {
        setMsg("Permissão negada. Ative as notificações nas configurações do navegador.");
        return;
      }
      const { public_key, enabled: available } = await getPushPublicKey();
      if (!available || !public_key) {
        setMsg("Notificações indisponíveis no momento.");
        return;
      }
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(public_key),
      });
      await subscribePush(sub.toJSON() as { endpoint: string; keys: { p256dh: string; auth: string } });
      setEnabled(true);
      setMsg("Notificações ativadas.");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Não foi possível ativar.");
    } finally {
      setBusy(false);
    }
  }

  async function disable() {
    setBusy(true);
    setMsg(null);
    try {
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      if (sub) {
        const endpoint = sub.endpoint;
        await sub.unsubscribe();
        await unsubscribePush(endpoint).catch(() => {});
      }
      setEnabled(false);
      setMsg("Notificações desativadas.");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Não foi possível desativar.");
    } finally {
      setBusy(false);
    }
  }

  if (!supported) return null;

  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-[var(--brand-soft)] text-[var(--brand-strong)]">
            {enabled ? <BellRing className="h-5 w-5" /> : <Bell className="h-5 w-5" />}
          </span>
          <div>
            <p className="font-medium">Notificações</p>
            <p className="text-xs text-[var(--muted)]">
              Receba avisos e lembretes de escalas no seu aparelho.
            </p>
            {msg && <p className="mt-1 text-xs text-[var(--brand)]">{msg}</p>}
          </div>
        </div>
        <Button
          type="button"
          variant={enabled ? "outline" : "primary"}
          className="shrink-0"
          disabled={busy}
          onClick={enabled ? disable : enable}
        >
          {enabled ? <BellOff className="h-4 w-4" /> : <Bell className="h-4 w-4" />}
          {enabled ? "Desativar" : "Ativar"}
        </Button>
      </div>
    </Card>
  );
}
