"use client";

import { useCallback, useEffect, useState } from "react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getMonthlyStatement, type MonthlyStatement, type StatementLine } from "@/lib/api";
import { currency } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";

function mesAtual() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

export default function MonthlyStatementPage() {
  const { toast } = useToast();
  const [month, setMonth] = useState(mesAtual());
  const [st, setSt] = useState<MonthlyStatement | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (m: string) => {
    setLoading(true);
    try {
      setSt(await getMonthlyStatement(m));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    load(month);
  }, [load, month]);

  const linha = (l: StatementLine) => (
    <TRow key={l.category_id}>
      <TD className="w-14 text-xs text-zinc-400">{l.code}</TD>
      <TD className="font-medium">{l.name}</TD>
      {l.weeks.map((v, i) => (
        <TD key={i} className={`text-right tabular-nums ${v === 0 ? "text-zinc-300 dark:text-zinc-600" : ""}`}>
          {currency(v)}
        </TD>
      ))}
      <TD className="text-right font-semibold tabular-nums">{currency(l.total)}</TD>
    </TRow>
  );

  return (
    <div className="page">
      <PageHeader
        title="Demonstrativo Mensal"
        description="Regime de caixa - entradas e saidas por semana"
        actions={
          <ExportButtons
            path="/api/v1/reports/monthly-statement/export"
            params={{ month }}
            filenameBase={`demonstrativo-${month}`}
          />
        }
      />

      <Card className="mb-6 flex flex-wrap items-end gap-3">
        <div>
          <label className="label">Mes</label>
          <Input type="month" className="h-8 text-sm" value={month} onChange={(e) => setMonth(e.target.value)} />
        </div>
      </Card>

      {loading || !st ? (
        <div className="grid gap-4 md:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : (
        <>
          <div className="mb-6 grid gap-4 md:grid-cols-4">
            <Card><p className="text-sm text-zinc-500">Saldo inicial</p><p className="mt-1 text-2xl font-semibold">{currency(st.opening_balance)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Entradas</p><p className="mt-1 text-2xl font-semibold text-emerald-600">{currency(st.total_income)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Saidas</p><p className="mt-1 text-2xl font-semibold text-red-600">{currency(st.total_expense)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Saldo final</p><p className={`mt-1 text-2xl font-semibold ${st.closing_balance >= 0 ? "text-sky-700" : "text-red-600"}`}>{currency(st.closing_balance)}</p></Card>
          </div>

          <Card className="mb-6 overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-emerald-700 dark:border-zinc-800 dark:text-emerald-400">Entradas</h3>
            <Table>
              <THead>
                <TRow>
                  <TH>Cod</TH><TH>Descricao</TH>
                  {st.weeks.map((w) => <TH key={w.label} className="text-right">{w.label}</TH>)}
                  <TH className="text-right">Total</TH>
                </TRow>
              </THead>
              <TBody>
                {st.income.map(linha)}
                {st.income.length === 0 && <TRow><TD colSpan={8} className="py-6 text-center text-zinc-400">Sem contas de entrada.</TD></TRow>}
                <TRow>
                  <TD /><TD className="font-semibold">Total das entradas</TD>
                  {st.weeks.map((_, i) => <TD key={i} />)}
                  <TD className="text-right font-semibold text-emerald-600">{currency(st.total_income)}</TD>
                </TRow>
              </TBody>
            </Table>
          </Card>

          <Card className="overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-red-700 dark:border-zinc-800 dark:text-red-400">Saidas</h3>
            <Table>
              <THead>
                <TRow>
                  <TH>Cod</TH><TH>Descricao</TH>
                  {st.weeks.map((w) => <TH key={w.label} className="text-right">{w.label}</TH>)}
                  <TH className="text-right">Total</TH>
                </TRow>
              </THead>
              <TBody>
                {st.expense.map(linha)}
                {st.expense.length === 0 && <TRow><TD colSpan={8} className="py-6 text-center text-zinc-400">Sem contas de saida.</TD></TRow>}
                <TRow>
                  <TD /><TD className="font-semibold">Total das saidas</TD>
                  {st.weeks.map((_, i) => <TD key={i} />)}
                  <TD className="text-right font-semibold text-red-600">{currency(st.total_expense)}</TD>
                </TRow>
              </TBody>
            </Table>
          </Card>

          <p className="mt-4 text-xs text-zinc-400">
            Saldo inicial: liquido acumulado ate o mes anterior. Saldo final = saldo inicial + entradas  saidas.
          </p>
        </>
      )}
    </div>
  );
}
