"use client";

import { useEffect, useState } from "react";
import { Check, Copy, QrCode } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { getPublicTenant, tenantSlugFromHost, type PublicTenant } from "@/lib/api";

export default function VisitorContribuirPage() {
  const [tenant, setTenant] = useState<PublicTenant | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    const slug = tenantSlugFromHost();
    if (slug) {
      getPublicTenant(slug).then(setTenant).catch(() => {});
    }
  }, []);

  async function copyPix() {
    if (!tenant?.pix_key) return;
    try {
      await navigator.clipboard.writeText(tenant.pix_key);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* area de transferencia indisponivel */
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Contribuir</h1>
      <Card>
        <div className="mb-3 flex items-center gap-2">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-[var(--brand-soft)] text-[var(--brand-strong)]">
            <QrCode className="h-5 w-5" />
          </span>
          <div>
            <p className="font-semibold">Pix da igreja</p>
            <p className="text-xs text-[var(--muted)]">
              {tenant?.pix_name || tenant?.name || "Dízimos e ofertas"}
            </p>
          </div>
        </div>

        {tenant?.pix_key ? (
          <>
            <div className="tnum break-all rounded-lg border border-[var(--line)] bg-[var(--paper)] px-3 py-2 font-mono text-sm">
              {tenant.pix_key}
            </div>
            <Button type="button" className="mt-3 w-full" onClick={copyPix}>
              {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              {copied ? "Chave copiada" : "Copiar chave Pix"}
            </Button>
            <p className="mt-2 text-center text-xs text-[var(--muted)]">
              Abra o app do seu banco, escolha Pix e cole a chave.
            </p>
          </>
        ) : (
          <p className="text-sm text-[var(--muted)]">
            A igreja ainda não cadastrou a chave Pix. Procure a secretaria.
          </p>
        )}
      </Card>
    </div>
  );
}
