"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { CreditCard, RefreshCw, TrendingDown, Users } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { currency, datePt } from "@/lib/format";
import {
  getBillingOverview,
  listBillingSubscriptions,
  listBillingInvoices,
  type BillingOverview,
  type BillingSubscriptionRow,
  type BillingInvoiceRow,
} from "@/lib/api";

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

export default function PlatformBillingPage() {
  const { toast } = useToast();
  const [overview, setOverview] = useState<BillingOverview | null>(null);
  const [subs, setSubs] = useState<BillingSubscriptionRow[] | null>(null);
  const [invoices, setInvoices] = useState<BillingInvoiceRow[] | null>(null);
  const [status, setStatus] = useState("");

  const load = useCallback(async () => {
    try {
      const [o, s, i] = await Promise.all([
        getBillingOverview(),
        listBillingSubscriptions(),
        listBillingInvoices(status),
      ]);
      setOverview(o);
      setSubs(s.subscriptions);
      setInvoices(i.invoices);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar cobranca", "error");
      setSubs([]);
      setInvoices([]);
    }
  }, [status, toast]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="page space-y-4">
      <PageHeader
        title="Cobranca"
        description="Assinaturas e faturas das igrejas (Asaas). A cobranca e criada a partir do preco do plano."
        actions={
          <Button variant="outline" onClick={load}>
            <RefreshCw className="h-4 w-4" /> Atualizar
          </Button>
        }
      />

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Stat icon={<CreditCard className="h-4 w-4" />} label="Assinaturas ativas" value={overview ? String(overview.active_subscriptions) : "-"} />
        <Stat icon={<Users className="h-4 w-4" />} label="Receita recorrente (MRR)" value={overview ? currency(overview.mrr_cents / 100) : "-"} />
        <Stat icon={<TrendingDown className="h-4 w-4" />} label="Faturas vencidas" value={overview ? String(overview.overdue_invoices) : "-"} tone="text-red-600" />
        <Stat icon={<TrendingDown className="h-4 w-4" />} label="Em atraso (valor)" value={overview ? currency(overview.overdue_cents / 100) : "-"} tone="text-red-600" />
      </div>

      <Card className="overflow-hidden p-0">
        <CardHeader className="px-4 pt-4">
          <CardTitle>Assinaturas por igreja</CardTitle>
        </CardHeader>
        {subs === null ? (
          <div className="p-4"><SkeletonRows rows={4} /></div>
        ) : subs.length === 0 ? (
          <EmptyState icon={<CreditCard className="h-10 w-10" />} title="Nenhuma igreja" description="Cadastre igrejas para gerir a cobranca." />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Igreja</TH><TH>Plano</TH><TH>Status</TH>
                <TH className="text-right">Valor</TH><TH>Ciclo</TH>
                <TH>Forma</TH><TH>Prox. vencimento</TH><TH></TH>
              </TRow>
            </THead>
            <TBody>
              {subs.map((s) => (
                <TRow key={s.tenant_id}>
                  <TD className="font-medium">{s.tenant_name}</TD>
                  <TD className="font-mono text-xs text-zinc-500">{s.plan}</TD>
                  <TD><Badge tone={STATUS_TONE[s.billing_status] ?? "zinc"}>{STATUS_LABEL[s.billing_status] ?? s.billing_status}</Badge></TD>
                  <TD className="text-right tabular-nums">{currency(s.value_cents / 100)}</TD>
                  <TD>{s.cycle === "YEARLY" ? "Anual" : "Mensal"}</TD>
                  <TD>{s.billing_type || "-"}</TD>
                  <TD>{s.next_due_date ? datePt(s.next_due_date) : "-"}</TD>
                  <TD className="text-right">
                    <Link className="btn-base btn-outline h-7 px-2 text-xs" href={`/dashboard/platform/churches/${s.tenant_id}?tab=cobranca`}>
                      Gerenciar
                    </Link>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Card className="overflow-hidden p-0">
        <CardHeader className="flex flex-row items-center justify-between px-4 pt-4">
          <CardTitle>Faturas</CardTitle>
          <select
            className="select h-8 w-40 text-sm"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="">Todas</option>
            <option value="pending">Pendentes</option>
            <option value="paid">Pagas</option>
            <option value="overdue">Vencidas</option>
            <option value="refunded">Estornadas</option>
            <option value="canceled">Canceladas</option>
          </select>
        </CardHeader>
        {invoices === null ? (
          <div className="p-4"><SkeletonRows rows={4} /></div>
        ) : invoices.length === 0 ? (
          <EmptyState icon={<CreditCard className="h-10 w-10" />} title="Nenhuma fatura" description="As faturas aparecem quando a assinatura gera cobrancas." />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Igreja</TH><TH>Status</TH><TH className="text-right">Valor</TH>
                <TH>Forma</TH><TH>Vencimento</TH><TH>Pago em</TH><TH></TH>
              </TRow>
            </THead>
            <TBody>
              {invoices.map((v) => (
                <TRow key={v.id}>
                  <TD className="font-medium">{v.tenant_name}</TD>
                  <TD><Badge tone={STATUS_TONE[v.status] ?? "zinc"}>{STATUS_LABEL[v.status] ?? v.status}</Badge></TD>
                  <TD className="text-right tabular-nums">{currency(v.value_cents / 100)}</TD>
                  <TD>{v.billing_type || "-"}</TD>
                  <TD>{v.due_date ? datePt(v.due_date) : "-"}</TD>
                  <TD>{v.paid_at ? datePt(v.paid_at) : "-"}</TD>
                  <TD className="text-right">
                    {v.invoice_url && (
                      <a className="btn-base btn-outline h-7 px-2 text-xs" href={v.invoice_url} target="_blank" rel="noreferrer">
                        Ver
                      </a>
                    )}
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>
    </div>
  );
}

function Stat({ icon, label, value, tone }: { icon: React.ReactNode; label: string; value: string; tone?: string }) {
  return (
    <Card>
      <CardContent className="flex items-center gap-3">
        <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300">
          {icon}
        </span>
        <div className="min-w-0">
          <p className="truncate text-xs text-zinc-500">{label}</p>
          <p className={`text-lg font-semibold tabular-nums ${tone ?? ""}`}>{value}</p>
        </div>
      </CardContent>
    </Card>
  );
}
