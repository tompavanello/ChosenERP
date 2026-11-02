"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, TrendingDown, UserMinus, Wallet } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Field, Select } from "@/components/ui/input";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { getContributionDrops, type ContributionDrop } from "@/lib/api";
import { currency, datePt, number } from "@/lib/format";
import { ReportSection } from "@/components/reports/report-section";
import { KpiCard } from "@/components/reports/kpi-card";
import { RankedBar } from "@/components/reports/ranked-bar";
import { COLORS } from "@/components/reports/chart-theme";

export default function ContributionDropPage() {
  const [items, setItems] = useState<ContributionDrop[] | null>(null);
  const [months, setMonths] = useState(6);
  const [recent, setRecent] = useState(1);

  const load = useCallback(async () => {
    setItems(null);
    try {
      setItems((await getContributionDrops(months, recent)).members);
    } catch {
      setItems([]);
    }
  }, [months, recent]);

  useEffect(() => {
    load();
  }, [load]);

  const total = (items ?? []).reduce((s, m) => s + m.total, 0);
  const avg = items && items.length > 0 ? total / items.length : 0;
  const top = useMemo(() => (items ?? []).slice(0, 8).map((m) => ({ name: m.member_name, value: m.total })), [items]);
  const buckets = useMemo(() => {
    const b = [
      { name: `Ha ${recent} a ${recent + 1} mes(es)`, value: 0 },
      { name: "Ha 2 a 3 meses", value: 0 },
      { name: "Ha 4 a 5 meses", value: 0 },
      { name: "Ha 6 meses ou mais", value: 0 },
    ];
    for (const m of items ?? []) {
      const days = Math.max(0, Math.floor((Date.now() - new Date(m.last_at).getTime()) / 86400000));
      const mo = Math.floor(days / 30);
      const idx = mo <= recent + 1 ? 0 : mo <= 3 ? 1 : mo <= 5 ? 2 : 3;
      b[idx].value += 1;
    }
    return b;
  }, [items, recent]);

  return (
    <div className="page">
      <PageHeader title="Queda de contribuição" description="Membros que contribuíram no período e pararam (sigilo financeiro)" />

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-3">
        <Field label="Janela (meses)">
          <Select className="h-8 w-32 text-sm" value={months} onChange={(e) => setMonths(Number(e.target.value))}>
            {[3, 6, 12].map((m) => <option key={m} value={m}>{m}</option>)}
          </Select>
        </Field>
        <Field label="Sem contribuir há (meses)">
          <Select className="h-8 w-32 text-sm" value={recent} onChange={(e) => setRecent(Number(e.target.value))}>
            {[1, 2, 3].map((m) => <option key={m} value={m}>{m}</option>)}
          </Select>
        </Field>
        <p className="ml-auto text-xs text-zinc-400">Os valores abaixo sao sigilosos e visiveis apenas por quem tem permissao financeira.</p>
      </Card>

      <div className="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard label="Pessoas em queda" value={items ? number(items.length) : "..."} icon={UserMinus} tone="amber" />
        <KpiCard label="Total que parou" value={currency(total)} icon={Wallet} tone="red" hint="soma das ultimas contribuicoes" />
        <KpiCard label="Ticket medio" value={currency(avg)} icon={TrendingDown} tone="sky" />
        <KpiCard label="Risco principal" value={top[0]?.name ?? "-"} icon={AlertTriangle} tone="indigo" hint={top[0] ? currency(top[0].value) : "sem doadores em queda"} />
      </div>

      {items && items.length > 0 && (
        <div className="mb-4 grid gap-4 lg:grid-cols-2">
          <ReportSection title="Ha quanto tempo pararam?" subtitle="Faixas de tempo desde a ultima contribuicao">
            <RankedBar data={buckets} format={(n) => `${n} pessoa(s)`} color={COLORS.amber} showShare />
          </ReportSection>
          <ReportSection title="Maiores valores em risco" subtitle="Top 8 por total contribuido no periodo" insight="Comece o acolhimento pelos maiores valores e pelos mais antigos.">
            <RankedBar data={top} total={total} color={COLORS.expense} emptyLabel="Sem doadores em queda." />
          </ReportSection>
        </div>
      )}

      <Card className="overflow-hidden p-0">
        {items === null ? (
          <SkeletonRows rows={5} />
        ) : items.length === 0 ? (
          <EmptyState icon={<TrendingDown className="h-10 w-10" />} title="Nenhuma queda no período" description="Todos os doadores do período contribuíram recentemente." />
        ) : (
          <Table>
            <THead><TRow><TH>Membro</TH><TH>Última contribuição</TH><TH className="text-right">Total no período</TH></TRow></THead>
            <TBody>
              {items.map((m) => (
                <TRow key={m.member_id}>
                  <TD className="font-medium">{m.member_name}</TD>
                  <TD className="text-zinc-500">{datePt(m.last_at)}</TD>
                  <TD className="tnum text-right">{currency(m.total)}</TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      {items && items.length > 0 && (
        <p className="mt-3 text-sm text-zinc-500">{items.length} membro(s) · total {currency(total)} no período.</p>
      )}
    </div>
  );
}
