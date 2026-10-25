"use client";

import { useCallback, useEffect, useState } from "react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Modal } from "@/components/ui/modal";
import { Pagination } from "@/components/ui/pagination";
import { useToast } from "@/components/ui/toast";
import {
  getMonthlyStatement, getMonthlyStatementTransactions,
  type MonthlyStatement, type StatementLine, type StatementTransaction, type WeekRange,
} from "@/lib/api";
import { currency, datePt } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";

function mesAtual() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

// "2026-03-01" -> "01/03" (dia e mes da semana).
function diaMes(iso: string) {
  return iso?.length === 10 ? `${iso.slice(8, 10)}/${iso.slice(5, 7)}` : iso;
}

const DETAIL_PER_PAGE = 10;

export default function MonthlyStatementPage() {
  const { toast } = useToast();
  const [month, setMonth] = useState(mesAtual());
  const [st, setSt] = useState<MonthlyStatement | null>(null);
  const [loading, setLoading] = useState(true);
  const [detail, setDetail] = useState<{
    line: StatementLine;
    week: WeekRange;
    type: "income" | "expense";
    total: number;
  } | null>(null);
  const [detailTxns, setDetailTxns] = useState<StatementTransaction[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailPage, setDetailPage] = useState(1);

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

  useEffect(() => {
    setDetail(null);
  }, [month]);

  // A semana pode comecar/terminar fora do mes; o detalhe so considera os dias
  // do proprio demonstrativo (o total da celula tambem).
  function weekWithinMonth(week: WeekRange) {
    const from = st && week.from < st.from ? st.from : week.from;
    const to = st && week.to > st.to ? st.to : week.to;
    return { from, to };
  }

  function openWeek(line: StatementLine, week: WeekRange, type: "income" | "expense", total: number) {
    const { from, to } = weekWithinMonth(week);
    setDetail({ line, week, type, total });
    setDetailTxns([]);
    setDetailPage(1);
    setDetailLoading(true);
    getMonthlyStatementTransactions({ category_id: line.category_id, from, to })
      .then((r) => setDetailTxns(r.transactions))
      .catch((e) => toast(e instanceof Error ? e.message : "Erro ao carregar lancamentos", "error"))
      .finally(() => setDetailLoading(false));
  }

  const linha = (l: StatementLine, weeks: WeekRange[], type: "income" | "expense") => (
    <TRow key={l.category_id}>
      <TD className="w-14 text-xs text-zinc-400">{l.code}</TD>
      <TD className="font-medium">{l.name}</TD>
      {l.weeks.map((v, i) => {
        const week = weeks[i];
        return (
          <TD key={i} className={`text-right tabular-nums ${v === 0 ? "text-zinc-300 dark:text-zinc-600" : ""}`}>
            {v === 0 || !week ? currency(v) : (
              <button
                type="button"
                onClick={() => openWeek(l, week, type, v)}
                className="font-medium text-sky-700 underline-offset-2 hover:underline dark:text-sky-400"
                title={`Ver lancamentos de ${week.label}`}
              >
                {currency(v)}
              </button>
            )}
          </TD>
        );
      })}
      <TD className="text-right font-semibold tabular-nums">{currency(l.total)}</TD>
    </TRow>
  );

  return (
    <div className="page">
      <PageHeader
        title="Demonstrativo Mensal"
        description="Regime de caixa - entradas e saidas por semana (domingo a sabado)"
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
                  {st.weeks.map((w) => (
                    <TH key={w.label} className="text-right">
                      <span className="block">{w.label}</span>
                      <span className="block text-[10px] font-normal normal-case tracking-normal text-zinc-400">
                        {diaMes(w.from)} a {diaMes(w.to)}
                      </span>
                    </TH>
                  ))}
                  <TH className="text-right">Total</TH>
                </TRow>
              </THead>
              <TBody>
                {st.income.map((l) => linha(l, st.weeks, "income"))}
                {st.income.length === 0 && <TRow><TD colSpan={st.weeks.length + 3} className="py-6 text-center text-zinc-400">Sem contas de entrada.</TD></TRow>}
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
                  {st.weeks.map((w) => (
                    <TH key={w.label} className="text-right">
                      <span className="block">{w.label}</span>
                      <span className="block text-[10px] font-normal normal-case tracking-normal text-zinc-400">
                        {diaMes(w.from)} a {diaMes(w.to)}
                      </span>
                    </TH>
                  ))}
                  <TH className="text-right">Total</TH>
                </TRow>
              </THead>
              <TBody>
                {st.expense.map((l) => linha(l, st.weeks, "expense"))}
                {st.expense.length === 0 && <TRow><TD colSpan={st.weeks.length + 3} className="py-6 text-center text-zinc-400">Sem contas de saida.</TD></TRow>}
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

          <Modal open={!!detail} onClose={() => setDetail(null)} title="Lancamentos da semana" size="lg">
            {detail && (() => {
              const range = weekWithinMonth(detail.week);
              const pages = Math.max(1, Math.ceil(detailTxns.length / DETAIL_PER_PAGE));
              const safePage = Math.min(detailPage, pages);
              const pageTxns = detailTxns.slice((safePage - 1) * DETAIL_PER_PAGE, safePage * DETAIL_PER_PAGE);
              return (
                <div className="space-y-3 text-sm">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div>
                      <p className="font-semibold">
                        {detail.line.code ? `${detail.line.code} - ` : ""}{detail.line.name}
                      </p>
                      <p className="text-xs text-zinc-500">
                        {detail.week.label} · {diaMes(range.from)} a {diaMes(range.to)} · {detailTxns.length} lancamento(s)
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-zinc-500">Total da semana</p>
                      <p className={`text-lg font-semibold ${detail.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
                        {currency(detail.total)}
                      </p>
                    </div>
                  </div>

                  {detailLoading ? (
                    <p className="text-xs text-zinc-500">Carregando lancamentos...</p>
                  ) : (
                    <>
                      <Table wrapperClassName="max-h-[55vh] overflow-y-auto rounded-lg border border-zinc-100 dark:border-zinc-800">
                        <THead className="sticky top-0 z-10 bg-white shadow-[0_1px_0_rgba(0,0,0,0.06)] dark:bg-zinc-950">
                          <TRow>
                            <TH>Data</TH><TH>Descricao</TH><TH>Tipo</TH><TH className="text-right">Valor</TH>
                          </TRow>
                        </THead>
                        <TBody>
                          {pageTxns.map((t) => (
                            <TRow key={t.id}>
                              <TD className="whitespace-nowrap text-zinc-500">{datePt(t.occurred_at)}</TD>
                              <TD>
                                <span className="font-medium">{t.description || t.donor_name || "-"}</span>
                                {t.description && t.donor_name && <span className="block text-xs text-zinc-400">{t.donor_name}</span>}
                                {t.account_name && <span className="block text-xs text-zinc-400">{t.account_name}</span>}
                              </TD>
                              <TD><Badge tone={t.type === "income" ? "green" : "red"}>{t.type === "income" ? "Entrada" : "Saida"}</Badge></TD>
                              <TD className={`text-right font-medium ${t.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(t.amount)}</TD>
                            </TRow>
                          ))}
                          {detailTxns.length === 0 && (
                            <TRow><TD colSpan={4} className="py-6 text-center text-zinc-400">Nenhum lancamento nesta semana.</TD></TRow>
                          )}
                        </TBody>
                      </Table>
                      <Pagination
                        page={safePage}
                        total={detailTxns.length}
                        perPage={DETAIL_PER_PAGE}
                        onChange={setDetailPage}
                      />
                    </>
                  )}
                </div>
              );
            })()}
          </Modal>
        </>
      )}
    </div>
  );
}
