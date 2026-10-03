"use client";

import { useCallback, useEffect, useState } from "react";
import { TrendingDown } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Field, Select } from "@/components/ui/input";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { getContributionDrops, type ContributionDrop } from "@/lib/api";
import { currency, datePt } from "@/lib/format";

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

  return (
    <div className="page">
      <PageHeader title="Queda de contribuição" description="Membros que contribuíram no período e pararam (sigilo financeiro)" />

      <div className="mb-4 flex items-end gap-3">
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
      </div>

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
