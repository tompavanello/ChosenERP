"use client";

import { useCallback, useEffect, useState } from "react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getDRE, type DRE } from "@/lib/api";
import { currency } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";

export default function DreReportPage() {
  const { toast } = useToast();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [dre, setDre] = useState<DRE | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (f: string, t: string) => {
    setLoading(true);
    try {
      setDre(await getDRE(f, t));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    load("", "");
  }, [load]);

  const params: Record<string, string> = {};
  if (from) params.from = from;
  if (to) params.to = to;
  const delta = dre?.comparison?.delta_pct ?? 0;

  return (
    <div className="page">
      <PageHeader
        title="DRE - Demonstrativo do Resultado"
        description="Resultado por categoria e comparativo de periodo"
        actions={
          <ExportButtons
            path="/api/v1/reports/dre/export"
            params={params}
            filenameBase={`dre-${from || "inicio"}-${to || "atual"}`}
          />
        }
      />

      <Card className="mb-6 flex flex-wrap items-end gap-3">
        <div><label className="label">De</label><Input type="date" value={from} onChange={(e) => setFrom(e.target.value)} /></div>
        <div><label className="label">Ate</label><Input type="date" value={to} onChange={(e) => setTo(e.target.value)} /></div>
        <Button onClick={() => load(from, to)}>Filtrar</Button>
        <Button variant="ghost" onClick={() => { setFrom(""); setTo(""); load("", ""); }}>Mes atual</Button>
      </Card>

      {loading ? (
        <div className="grid gap-4 md:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : dre ? (
        <>
          <div className="mb-6 grid gap-4 md:grid-cols-4">
            <Card><p className="text-sm text-zinc-500">Entradas</p><p className="mt-1 text-2xl font-semibold text-emerald-600">{currency(dre.income)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Saidas</p><p className="mt-1 text-2xl font-semibold text-red-600">{currency(dre.expense)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Resultado</p><p className={`mt-1 text-2xl font-semibold ${dre.net >= 0 ? "text-sky-700" : "text-red-600"}`}>{currency(dre.net)}</p></Card>
            <Card>
              <p className="text-sm text-zinc-500">vs. periodo anterior</p>
              <p className={`mt-1 text-2xl font-semibold ${delta >= 0 ? "text-emerald-600" : "text-red-600"}`}>{delta >= 0 ? "+" : ""}{delta.toFixed(1)}%</p>
              <p className="text-xs text-zinc-400">anterior: {currency(dre.comparison?.prev_net ?? 0)}</p>
            </Card>
          </div>

          <Card className="overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700">DRE por categoria</h3>
            <Table>
              <THead><TRow><TH>Conta contabil</TH><TH>Tipo</TH><TH className="text-right">Total</TH></TRow></THead>
              <TBody>
                {dre.lines.map((l) => (
                  <TRow key={`${l.type}-${l.category_id}`}>
                    <TD className="font-medium">{l.category}</TD>
                    <TD><Badge tone={l.type === "income" ? "green" : "red"}>{l.type === "income" ? "Entrada" : "Saida"}</Badge></TD>
                    <TD className={`text-right font-semibold ${l.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(l.total)}</TD>
                  </TRow>
                ))}
                {dre.lines.length === 0 && <TRow><TD colSpan={3} className="py-8 text-center text-zinc-400">Sem lancamentos no periodo.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </>
      ) : null}
    </div>
  );
}
