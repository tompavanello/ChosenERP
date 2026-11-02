"use client";

import { useEffect, useState } from "react";
import {
  ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { CircleUser, MapPin, Users } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { getDemographics, type Demographics } from "@/lib/api";
import { MEMBERSHIP_STATUS, MARITAL_STATUS, GENDER } from "@/lib/constants";
import { ReportSection } from "./report-section";
import { KpiCard } from "./kpi-card";
import { RankedBar } from "./ranked-bar";
import { COLORS, numberCompact } from "./chart-theme";
import { number } from "@/lib/format";

const rotulo = (map: Record<string, string>, key: string) =>
  key === "nao_informado" ? "Nao informado" : map[key] ?? key;

/** Painel demografico: piramide etaria, situacao, estado civil e geografia. */
export function DemographicsSection() {
  const [d, setD] = useState<Demographics | null>(null);

  useEffect(() => {
    getDemographics().then(setD).catch(() => setD(null));
  }, []);

  if (!d) {
    return (
      <div className="grid gap-4 lg:grid-cols-3">
        {[0, 1, 2].map((i) => <Skeleton key={i} className="h-64" />)}
      </div>
    );
  }

  const pyramid = d.age_pyramid.map((p) => ({
    name: p.bucket, Masculino: p.male, Feminino: p.female,
  }));
  const statusData = d.by_status.map((s) => ({
    name: MEMBERSHIP_STATUS[s.key]?.label ?? s.key, value: s.count,
  }));
  const maritalData = d.by_marital_status.map((s) => ({
    name: rotulo(MARITAL_STATUS, s.key), value: s.count,
  }));
  const genderData = d.by_gender.map((s) => ({
    name: rotulo(GENDER, s.key), value: s.count,
  }));
  const male = d.by_gender.find((g) => g.key === "male")?.count ?? 0;
  const female = d.by_gender.find((g) => g.key === "female")?.count ?? 0;
  const active = d.by_status.filter((s) => s.key !== "inactive").reduce((sum, s) => sum + s.count, 0);
  const activePct = d.total > 0 ? (active / d.total) * 100 : 0;

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard label="Total de pessoas" value={number(d.total)} icon={Users} tone="sky" />
        <KpiCard label="Ativos no Rol" value={number(active)} hint={`${activePct.toFixed(0)}% do total`} tone="green" />
        <KpiCard label="Masculino" value={number(male)} icon={CircleUser} tone="indigo" hint={d.total > 0 ? `${((male / d.total) * 100).toFixed(0)}%` : undefined} />
        <KpiCard label="Feminino" value={number(female)} icon={CircleUser} tone="amber" hint={d.total > 0 ? `${((female / d.total) * 100).toFixed(0)}%` : undefined} />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <ReportSection title="Piramide etaria" subtitle="Distribuicao por faixa e sexo">
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={pyramid} layout="vertical" margin={{ left: 8, right: 8 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" horizontal={false} />
                <XAxis type="number" tick={{ fontSize: 12 }} stroke="#9ca3af" allowDecimals={false} />
                <YAxis type="category" dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" width={48} />
                <Tooltip />
                <Legend />
                <Bar dataKey="Masculino" fill="#38bdf8" stackId="a" />
                <Bar dataKey="Feminino" fill="#f472b6" stackId="a" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </ReportSection>

        <ReportSection title="Situacao no Rol" subtitle="Quem esta ativo e quem precisa de follow-up">
          <RankedBar data={statusData} format={numberCompact} color={COLORS.sky} showShare />
        </ReportSection>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <ReportSection title="Estado civil" subtitle="Composicao por situacao civil">
          <RankedBar data={maritalData} format={numberCompact} color={COLORS.indigo} showShare />
        </ReportSection>
        <ReportSection title="Sexo" subtitle="Comparacao direta">
          <RankedBar data={genderData} format={numberCompact} color={COLORS.teal} showShare />
        </ReportSection>
        <ReportSection title="Cobertura geografica" subtitle="Estados e cidades com cadastro">
          <div className="flex items-center gap-4 py-6 text-sm text-zinc-500">
            <MapPin className="h-8 w-8 text-zinc-300" />
            <div>
              <p><b className="text-zinc-700 dark:text-zinc-200">{d.by_state.length}</b> UF(s)</p>
              <p><b className="text-zinc-700 dark:text-zinc-200">{d.by_city.length}</b> cidade(s)</p>
            </div>
          </div>
        </ReportSection>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <ReportSection title="Distribuicao por UF" subtitle="Ordenada pelo volume">
          <Distribuicao rows={d.by_state} />
        </ReportSection>
        <ReportSection title="Cidades (top 20)" subtitle="Onde a igreja esta presente">
          <Distribuicao rows={d.by_city} />
        </ReportSection>
      </div>
    </div>
  );
}

function Distribuicao({ rows }: { rows: { key: string; count: number }[] }) {
  return (
    <RankedBar
      data={rows.map((r) => ({ name: r.key === "nao_informado" ? "Nao informado" : r.key, value: r.count }))}
      format={numberCompact}
      color={COLORS.sky}
      emptyLabel="Sem dados de endereco."
    />
  );
}
