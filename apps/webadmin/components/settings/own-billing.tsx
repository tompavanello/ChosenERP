"use client";

import { useCallback, useEffect, useState } from "react";
import { CreditCard } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { currency, datePt } from "@/lib/format";
import { getOwnBilling, type TenantBilling } from "@/lib/api";

const STATUS_TONE: Record<string, "green" | "red" | "amber" | "zinc" | "sky"> = {
  active: "green",
  paid: "green",
  pending: "amber",
  overdue: "red",
  canceled: "zinc",
  refunded: "sky",
  suspended: "red",
};

const STATUS_LABEL: Record<string, string> = {
  active: "Ativa",
  paid: "Pago",
  pending: "Pendente",
  overdue: "Vencida",
  canceled: "Cancelada",
  refunded: "Estornada",
  suspended: "Suspensa",
  none: "Sem assinatura",
};

export function OwnBilling() {
  const { toast } = useToast();
  const [data, setData] = useState<TenantBilling | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setData(await getOwnBilling());
      setError(null);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erro ao carregar a assinatura";
      setError(msg);
      toast(msg, "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  if (loading) return <Card className="p-4"><SkeletonRows rows={3} /></Card>;
  if (error || !data) {
    return (
      <Card className="p-4">
        <p className="text-sm text-zinc-500">
          {error ?? "Nao foi possivel carregar a assinatura."} Verifique se a sua igreja ja tem
          uma assinatura configurada pela plataforma.
        </p>
        <button type="button" onClick={load} className="btn-base btn-outline mt-3 h-8 px-3 text-sm">
          Tentar novamente
        </button>
      </Card>
    );
  }

  const sub = data.subscription;

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader><CardTitle>Assinatura</CardTitle></CardHeader>
        <CardContent className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
          <Info label="Situacao" value={<Badge tone={STATUS_TONE[data.billing_status] ?? "zinc"}>{STATUS_LABEL[data.billing_status] ?? data.billing_status}</Badge>} />
          <Info label="Plano" value={data.plan} />
          <Info label="Valor" value={sub ? currency(sub.value_cents / 100) : "-"} />
          <Info label="Forma" value={sub?.billing_type ?? "-"} />
          <Info label="Ciclo" value={sub ? (sub.cycle === "YEARLY" ? "Anual" : "Mensal") : "-"} />
          <Info label="Prox. vencimento" value={sub?.next_due_date ? datePt(sub.next_due_date) : "-"} />
        </CardContent>
      </Card>

      <Card className="overflow-hidden p-0">
        <CardHeader className="px-4 pt-4"><CardTitle>Faturas</CardTitle></CardHeader>
        {data.invoices.length === 0 ? (
          <EmptyState icon={<CreditCard className="h-10 w-10" />} title="Sem faturas" description="As faturas aparecem quando a assinatura gera cobrancas." />
        ) : (
          <Table>
            <THead>
              <TRow><TH>Status</TH><TH className="text-right">Valor</TH><TH>Forma</TH><TH>Vencimento</TH><TH>Pago em</TH><TH></TH></TRow>
            </THead>
            <TBody>
              {data.invoices.map((v) => (
                <TRow key={v.id}>
                  <TD><Badge tone={STATUS_TONE[v.status] ?? "zinc"}>{STATUS_LABEL[v.status] ?? v.status}</Badge></TD>
                  <TD className="text-right tabular-nums">{currency(v.value_cents / 100)}</TD>
                  <TD>{v.billing_type || "-"}</TD>
                  <TD>{v.due_date ? datePt(v.due_date) : "-"}</TD>
                  <TD>{v.paid_at ? datePt(v.paid_at) : "-"}</TD>
                  <TD className="text-right">
                    {v.invoice_url && <a className="btn-base btn-outline h-7 px-2 text-xs" href={v.invoice_url} target="_blank" rel="noreferrer">Pagar / Ver</a>}
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>
      <p className="text-xs text-zinc-400">Para alterar o plano ou a forma de pagamento, fale com o suporte da plataforma.</p>
    </div>
  );
}

function Info({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div>
      <p className="text-xs text-zinc-500">{label}</p>
      <p className="font-medium">{value}</p>
    </div>
  );
}
