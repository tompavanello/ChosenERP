"use client";

import { useEffect, useState } from "react";
import {
  ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { getDemographics, type Demographics } from "@/lib/api";
import { MEMBERSHIP_STATUS, MARITAL_STATUS, GENDER } from "@/lib/constants";

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
    name: MEMBERSHIP_STATUS[s.key]?.label ?? s.key, Total: s.count,
  }));
  const maritalData = d.by_marital_status.map((s) => ({
    name: rotulo(MARITAL_STATUS, s.key), Total: s.count,
  }));
  const genderData = d.by_gender.map((s) => ({
    name: rotulo(GENDER, s.key), Total: s.count,
  }));

  return (
    <div className="space-y-6">
      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Piramide etaria</h3>
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
        </Card>

        <Card>
          <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Situacao no Rol</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={statusData} layout="vertical" margin={{ left: 8, right: 8 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" horizontal={false} />
                <XAxis type="number" tick={{ fontSize: 12 }} stroke="#9ca3af" allowDecimals={false} />
                <YAxis type="category" dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" width={110} />
                <Tooltip />
                <Bar dataKey="Total" fill="#0ea5e9" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Estado civil</h3>
          <div className="h-56">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={maritalData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                <XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" allowDecimals={false} />
                <Tooltip />
                <Bar dataKey="Total" fill="#8b5cf6" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>

        <Card>
          <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Sexo</h3>
          <div className="h-56">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={genderData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                <XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" allowDecimals={false} />
                <Tooltip />
                <Bar dataKey="Total" fill="#10b981" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Distribuicao titulo="Distribuicao por UF" rows={d.by_state} />
        <Distribuicao titulo="Cidades (top 20)" rows={d.by_city} />
      </div>
    </div>
  );
}

function Distribuicao({ titulo, rows }: { titulo: string; rows: { key: string; count: number }[] }) {
  const max = Math.max(1, ...rows.map((r) => r.count));
  return (
    <Card>
      <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">{titulo}</h3>
      {rows.length === 0 ? (
        <p className="text-sm text-zinc-400">Sem dados de endereco.</p>
      ) : (
        <ul className="space-y-2">
          {rows.map((r) => (
            <li key={r.key} className="text-sm">
              <div className="flex items-center justify-between">
                <span className="truncate">
                  {r.key === "nao_informado" ? "Nao informado" : r.key}
                </span>
                <span className="text-xs text-zinc-400">{r.count}</span>
              </div>
              <div className="mt-1 h-1.5 w-full rounded-full bg-zinc-100 dark:bg-zinc-800">
                <div className="h-1.5 rounded-full bg-sky-500" style={{ width: `${(r.count / max) * 100}%` }} />
              </div>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
