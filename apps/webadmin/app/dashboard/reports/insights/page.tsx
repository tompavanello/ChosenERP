"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Brain, Building2, Sparkles, TrendingUp, Users, Wallet } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs } from "@/components/ui/tabs";
import { useToast } from "@/components/ui/toast";
import { getInsightNarrative, getInsights, type InsightReport } from "@/lib/api";
import { currency, number } from "@/lib/format";
import { ReportToolbar } from "@/components/reports/report-toolbar";
import { KpiCard } from "@/components/reports/kpi-card";
import { InsightList } from "@/components/reports/insight-card";
import { ReportSection } from "@/components/reports/report-section";

const SCOPES = [
  { key: "all", label: "Todos" },
  { key: "finance", label: "Financeiro" },
  { key: "people", label: "Pessoas" },
  { key: "operations", label: "Operacao" },
];

/**
 * Diagnostico da igreja: regras deterministicas com benchmark e acao.
 * O texto por IA e opcional e usa somente os cartoes agregados.
 */
export default function InsightsPage() {
  const { toast } = useToast();
  const [scope, setScope] = useState("all");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [rep, setRep] = useState<InsightReport | null>(null);
  const [loading, setLoading] = useState(true);

  const [narrative, setNarrative] = useState<{ ai_enabled: boolean; narrative: string } | null>(null);
  const [aiLoading, setAiLoading] = useState(false);

  const load = useCallback(async (f: string, t: string, sc = scope) => {
    setLoading(true);
    setNarrative(null);
    try {
      setRep(await getInsights({ scope: sc, from: f || undefined, to: t || undefined }));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar insights", "error");
      setRep(null);
    } finally {
      setLoading(false);
    }
  }, [scope, toast]);

  useEffect(() => {
    load("", "", scope);
  }, [load, scope]);

  async function generateNarrative() {
    setAiLoading(true);
    try {
      const r = await getInsightNarrative({ scope, from: from || undefined, to: to || undefined });
      setNarrative(r);
      if (!r.ai_enabled) toast("IA nao configurada: usando apenas as regras.", "info");
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao gerar resumo", "error");
    } finally {
      setAiLoading(false);
    }
  }

  const s = rep?.summary;

  return (
    <div className="page">
      <PageHeader
        title="Diagnostico e insights"
        description="Leitura dos numeros com benchmarks de mercado, riscos e proximos passos"
        actions={
          <Button variant="outline" onClick={generateNarrative} disabled={aiLoading}>
            <Sparkles className="h-4 w-4" /> {aiLoading ? "Gerando..." : "Resumo por IA"}
          </Button>
        }
      />

      <Tabs tabs={SCOPES} active={scope} onChange={setScope} />
      <ReportToolbar from={from} to={to} onFrom={setFrom} onTo={setTo} onChange={(f, t) => load(f, t)} />

      {loading ? (
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
          <Skeleton className="h-40" />
        </div>
      ) : !rep || !s ? (
        <Card><p className="py-8 text-center text-sm text-zinc-400">Nao foi possivel carregar o diagnostico.</p></Card>
      ) : (
        <>
          <div className="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <KpiCard label="Entradas no periodo" value={currency(s.income)} icon={Wallet} tone="green" />
            <KpiCard label="Resultado" value={currency(s.net)} icon={TrendingUp} tone="sky" hint={s.expense_ratio > 0 ? `despesas = ${s.expense_ratio.toFixed(0)}% das entradas` : undefined} />
            <KpiCard label="Reserva de caixa" value={s.runway_months > 0 ? `${s.runway_months.toFixed(1)} meses` : "-"} icon={Building2} tone="indigo" hint={`caixa: ${currency(s.cash)}`} />
            <KpiCard label="Doadores ativos" value={number(s.donors)} icon={Users} tone="amber" hint={`${number(s.active_members)} membros ativos`} />
          </div>

          <div className="mb-4 grid gap-4 lg:grid-cols-3">
            <ReportSection title="Concentracao da arrecadacao" subtitle="Participacao dos 10 maiores doadores">
              <p className="text-2xl font-semibold">{s.top10_share.toFixed(0)}%</p>
              <p className="mt-1 text-xs text-zinc-400">Referencia: abaixo de 40% reduz o risco de dependencia.</p>
            </ReportSection>
            <ReportSection title="Visitantes" subtitle="Conversao no periodo">
              <p className="text-2xl font-semibold">{s.visitors > 0 ? `${((s.converted_visitors / s.visitors) * 100).toFixed(0)}%` : "-"}</p>
              <p className="mt-1 text-xs text-zinc-400">{number(s.converted_visitors)} de {number(s.visitors)} visitantes integrados.</p>
            </ReportSection>
            <ReportSection title="Grupos" subtitle="Participacao dos membros ativos">
              <p className="text-2xl font-semibold">{s.active_members > 0 ? `${((s.group_members / s.active_members) * 100).toFixed(0)}%` : "-"}</p>
              <p className="mt-1 text-xs text-zinc-400">{number(s.group_members)} membros em grupos/celulas.</p>
            </ReportSection>
          </div>

          {narrative?.ai_enabled && narrative.narrative && (
            <Card className="mb-4 border-sky-100 bg-sky-50/40 p-4 dark:border-sky-900 dark:bg-sky-950/20">
              <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-sky-700 dark:text-sky-300">
                <Brain className="h-4 w-4" /> Resumo executivo (IA)
              </div>
              <p className="whitespace-pre-line text-sm leading-relaxed text-zinc-600 dark:text-zinc-300">{narrative.narrative}</p>
              <p className="mt-2 text-[11px] text-zinc-400">Gerado apenas a partir de numeros agregados, sem dados pessoais.</p>
            </Card>
          )}

          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-zinc-700 dark:text-zinc-200">O que merece atencao</h3>
            <Link href="/dashboard/reports/balance" className="text-xs text-sky-600 hover:underline">Abrir relatorios {"->"}</Link>
          </div>
          <InsightList insights={rep.insights} columns={2} />
        </>
      )}
    </div>
  );
}
