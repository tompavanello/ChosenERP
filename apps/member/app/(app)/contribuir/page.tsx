"use client";

import { useEffect, useState } from "react";
import { Check, Copy, QrCode, ReceiptText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import {
  datePt,
  getMeContributions,
  getPublicTenant,
  tenantSlugFromHost,
  type Contribution,
  type PublicTenant,
} from "@/lib/api";
import { currency } from "@/lib/utils";

export default function ContribuirPage() {
  const [tenant, setTenant] = useState<PublicTenant | null>(null);
  const [year, setYear] = useState(new Date().getFullYear());
  const [items, setItems] = useState<Contribution[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    const slug = tenantSlugFromHost();
    if (slug) {
      getPublicTenant(slug).then(setTenant).catch(() => {});
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    getMeContributions(year)
      .then((d) => {
        setItems(d.contributions ?? []);
        setTotal(d.total ?? 0);
      })
      .catch(() => {
        setItems([]);
        setTotal(0);
      })
      .finally(() => setLoading(false));
  }, [year]);

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

      <section>
        <div className="mb-2 flex items-center justify-between">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-[var(--muted)]">
            <ReceiptText className="h-4 w-4" /> Minhas contribuições
          </h2>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setYear((y) => y - 1)}
              className="rounded-md px-2 py-0.5 text-sm text-[var(--muted)] hover:bg-black/5"
            >
              ←
            </button>
            <span className="tnum text-sm font-medium">{year}</span>
            <button
              onClick={() => setYear((y) => Math.min(new Date().getFullYear(), y + 1))}
              disabled={year >= new Date().getFullYear()}
              className="rounded-md px-2 py-0.5 text-sm text-[var(--muted)] hover:bg-black/5 disabled:opacity-30"
            >
              →
            </button>
          </div>
        </div>

        {loading ? (
          <SkeletonRows rows={3} />
        ) : (
          <>
            <Card className="mb-3">
              <p className="text-xs text-[var(--muted)]">Total em {year}</p>
              <p className="tnum text-xl font-bold text-[var(--brand-strong)]">{currency(total)}</p>
            </Card>

            {items.length === 0 ? (
              <Card>
                <p className="text-sm text-[var(--muted)]">Nenhuma contribuição registrada neste ano.</p>
              </Card>
            ) : (
              <div className="space-y-2">
                {items.map((c) => (
                  <Card key={c.id} className="flex items-center justify-between gap-3 py-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">
                        {c.category_name || c.description || "Contribuição"}
                      </p>
                      <p className="text-xs text-[var(--muted)]">
                        {datePt(c.occurred_at)}
                        {c.payment_method ? ` · ${c.payment_method}` : ""}
                      </p>
                    </div>
                    <span className="tnum shrink-0 text-sm font-semibold">{currency(c.amount)}</span>
                  </Card>
                ))}
              </div>
            )}
          </>
        )}
      </section>
    </div>
  );
}
