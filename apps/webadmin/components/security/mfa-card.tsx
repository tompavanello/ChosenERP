"use client";

import { useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { ShieldCheck, ShieldOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/toast";
import { mfaStatus, mfaSetup, mfaEnable, mfaDisable } from "@/lib/api";

/**
 * Ativa/desativa o MFA (TOTP) do usuário logado. Usado tanto na tela de
 * Usuários (para admins) quanto na de perfil (qualquer usuário).
 */
export function MfaCard() {
  const { toast } = useToast();
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [setupData, setSetupData] = useState<{ secret: string; otpauth_url: string } | null>(null);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    mfaStatus().then((r) => setEnabled(r.enabled)).catch(() => setEnabled(false));
  }, []);

  async function start() {
    try {
      const r = await mfaSetup();
      setSetupData({ secret: r.secret, otpauth_url: r.otpauth_url });
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    }
  }
  async function confirm(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await mfaEnable(code);
      setEnabled(true);
      setSetupData(null);
      setCode("");
      toast("MFA ativado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Código inválido", "error");
    } finally {
      setBusy(false);
    }
  }
  async function disable() {
    try {
      await mfaDisable();
      setEnabled(false);
      toast("MFA desativado.");
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    }
  }

  if (enabled === null) return null;

  return (
    <Card className="p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className={`flex h-10 w-10 items-center justify-center rounded-xl ${enabled ? "bg-emerald-50 text-emerald-600" : "bg-amber-50 text-amber-600"}`}>
            <ShieldCheck className="h-5 w-5" />
          </div>
          <div>
            <p className="text-sm font-semibold">Autenticação em dois fatores (MFA)</p>
            <p className="text-xs text-zinc-500">
              {enabled ? "Ativada neste usuário. O login pedirá o código do autenticador." : "Recomendada para administradores e tesouraria."}
            </p>
          </div>
        </div>
        {enabled ? (
          <Button variant="outline" onClick={disable}><ShieldOff className="h-4 w-4" /> Desativar</Button>
        ) : (
          !setupData && <Button onClick={start}><ShieldCheck className="h-4 w-4" /> Ativar MFA</Button>
        )}
      </div>

      {setupData && (
        <form onSubmit={confirm} className="mt-4 flex flex-wrap items-start gap-4 border-t border-zinc-100 pt-4 dark:border-zinc-800">
          <div className="rounded-lg bg-white p-2 ring-1 ring-zinc-200">
            <QRCodeSVG value={setupData.otpauth_url} size={148} />
          </div>
          <div className="min-w-56 flex-1 space-y-2">
            <p className="text-xs text-zinc-500">
              Leia o QR no autenticador (Google Authenticator, Authy...) ou use o segredo:
              <code className="ml-1 break-all rounded bg-zinc-100 px-1 text-[11px] dark:bg-zinc-800">{setupData.secret}</code>
            </p>
            <Field label="Código de 6 dígitos">
              <Input
                inputMode="numeric" maxLength={6} className="h-8 w-40 text-sm tracking-widest"
                value={code} onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                required
              />
            </Field>
            <div className="flex gap-2">
              <Button type="submit" className="h-8 text-sm" disabled={busy}>{busy ? "Confirmando..." : "Confirmar e ativar"}</Button>
              <Button type="button" variant="ghost" className="h-8 text-sm" onClick={() => setSetupData(null)}>Cancelar</Button>
            </div>
          </div>
        </form>
      )}
    </Card>
  );
}
