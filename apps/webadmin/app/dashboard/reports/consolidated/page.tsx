"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { Building2, RefreshCw } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { StatCard } from "@/components/ui/stat-card";
import { EmptyState, Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { getConsolidated, type ConsolidatedResult } from "@/lib/api";
import { currency, number } from "@/lib/format";

const KIND_LABEL: Record<string, string> = {
  branch: "Filial",
  congregation: "Congregação",
  sub_congregation: "Sub-congregação",
};

export default function ConsolidatedReportPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canRead = hasPerm("finance.read") || hasPerm("members.read");

  const [data, setData] = useState<ConsolidatedResult | null>(null);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await getConsolidated(from, to));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar consolidado", "error");
      setData({ branches: [], totals: { member_count: 0, visitor_count: 0, income: 0, expense: 0, net: 0 } });
    }
  }, [from, to, toast]);

  useEffect(() => { load(); }, [load]);

  if (!canRead) {
    return (
      <div className="page">
        <PageHeader title="Consolidado Sede > Filiais" />
        <EmptyState icon={<Building2 className="h-10 w-10" />} title="Sem permissão" description="Você não tem acesso a este relatório." />
      </div>
    );
  }

  const chartData = (data?.branches ?? []).map((b) => ({
    name: b.name, Entradas: b.income, Saídas: b.expense,
  }));

  return (
    <div className="page">
      <PageHeader
        title="Consolidado Sede > Filiais"
        description="Membros, visitantes e movimentação por filial (inclui sub-congregações)"
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3">
        <div><label className="label">De</label><Input type="date" className="h-8 text-sm" value={from} onChange={(e) => setFrom(e.target.value)} /></div>
        <div><label className="label">Até</label><Input type="date" className="h-8 text-sm" value={to} onChange={(e) => setTo(e.target.value)} /></div>
        <Button variant="outline" className="h-8 text-sm" onClick={load}><RefreshCw className="h-4 w-4" /> Atualizar</Button>
      </Card>

      {data === null ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : (
        <>
          <div className="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
            <StatCard label="Filiais" value={number(data.branches.length)} />
            <StatCard label="Membros" value={number(data.totals.member_count)} />
            <StatCard label="Visitantes" value={number(data.totals.visitor_count)} />
            <StatCard label="Entradas" value={currency(data.totals.income)} />
            <StatCard label="Saldo" value={currency(data.totals.net)} />
          </div>

          {chartData.length > 0 && (
            <Card className="mb-4 p-4">
              <h3 className="mb-3 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Entradas × Saídas por filial</h3>
              <ResponsiveContainer width="100%" height={280}>
                <BarChart data={chartData} margin={{ top: 4, right: 8, left: 8, bottom: 4 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e4e4e7" />
                  <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip formatter={(v) => currency(Number(v))} />
                  <Legend />
                  <Bar dataKey="Entradas" fill="#0ea5e9" radius={[3, 3, 0, 0]} />
                  <Bar dataKey="Saídas" fill="#f97316" radius={[3, 3, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </Card>
          )}

          <Card className="overflow-hidden p-0">
            {data.branches.length === 0 ? (
              <EmptyState icon={<Building2 className="h-10 w-10" />} title="Nenhuma filial no escopo" description="Não há filiais visíveis para o seu usuário." />
            ) : (
              <Table>
                <THead><TRow><TH>Filial</TH><TH>Tipo</TH><TH className="text-right">Membros</TH><TH className="text-right">Visitantes</TH><TH className="text-right">Entradas</TH><TH className="text-right">Saídas</TH><TH className="text-right">Saldo</TH></TRow></THead>
                <TBody>
                  {data.branches.map((b) => (
                    <TRow key={b.id}>
                      <TD className="font-medium">
                        {b.kind === "sub_congregation" && <span className="mr-1 text-zinc-400">↳</span>}
                        {b.name}
                      </TD>
                      <TD><Badge tone="zinc">{KIND_LABEL[b.kind] ?? b.kind}</Badge></TD>
                      <TD className="text-right tabular-nums">{number(b.member_count)}</TD>
                      <TD className="text-right tabular-nums">{number(b.visitor_count)}</TD>
                      <TD className="text-right tabular-nums text-emerald-600">{currency(b.income)}</TD>
                      <TD className="text-right tabular-nums text-orange-600">{currency(b.expense)}</TD>
                      <TD className={`text-right tabular-nums ${b.net < 0 ? "text-red-600" : ""}`}>{currency(b.net)}</TD>
                    </TRow>
                  ))}
                  <TRow className="border-t-2 border-zinc-200 font-semibold dark:border-zinc-700">
                    <TD colSpan={2}>Total</TD>
                    <TD className="text-right tabular-nums">{number(data.totals.member_count)}</TD>
                    <TD className="text-right tabular-nums">{number(data.totals.visitor_count)}</TD>
                    <TD className="text-right tabular-nums">{currency(data.totals.income)}</TD>
                    <TD className="text-right tabular-nums">{currency(data.totals.expense)}</TD>
                    <TD className="text-right tabular-nums">{currency(data.totals.net)}</TD>
                  </TRow>
                </TBody>
              </Table>
            )}
          </Card>
        </>
      )}
    </div>
  );
}
