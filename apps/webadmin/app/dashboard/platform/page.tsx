"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Building2, CheckCircle2, Users, UserCog, GitBranch, HardDrive, ArrowRight } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getAdminStats, type PlatformStats } from "@/lib/api";

const fmtBytes = (n: number) => {
  if (!n) return "0 B";
  const u = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  let v = n;
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return `${v.toFixed(v >= 10 || i === 0 ? 0 : 1)} ${u[i]}`;
};

export default function PlatformOverviewPage() {
  const { toast } = useToast();
  const [stats, setStats] = useState<PlatformStats | null>(null);

  useEffect(() => {
    getAdminStats()
      .then(setStats)
      .catch((err) => toast(err instanceof Error ? err.message : "Erro ao carregar estatisticas", "error"));
  }, [toast]);

  return (
    <div className="page space-y-4">
      <PageHeader
        title="Plataforma"
        description="Administracao do ambiente e estatisticas gerais das igrejas (sem dados operacionais)."
        actions={
          <Link href="/dashboard/platform/churches" className="btn-base btn-primary px-4 py-2 text-sm rounded-lg">
            Gerenciar igrejas <ArrowRight className="h-4 w-4" />
          </Link>
        }
      />

      {!stats ? (
        <SkeletonRows rows={3} />
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
            <Stat icon={<Building2 className="h-4 w-4" />} label="Igrejas" value={stats.tenants} />
            <Stat icon={<CheckCircle2 className="h-4 w-4" />} label="Ativas" value={stats.tenants_active} />
            <Stat icon={<Users className="h-4 w-4" />} label="Membros" value={stats.members} />
            <Stat icon={<UserCog className="h-4 w-4" />} label="Usuarios" value={stats.users} />
            <Stat icon={<GitBranch className="h-4 w-4" />} label="Filiais" value={stats.branches} />
            <Stat icon={<HardDrive className="h-4 w-4" />} label="Armazenamento" value={fmtBytes(stats.storage_bytes)} />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <h3 className="mb-3 text-sm font-semibold text-zinc-500">Igrejas por plano</h3>
              {stats.by_plan.length === 0 ? (
                <p className="text-sm text-zinc-400">Sem dados.</p>
              ) : (
                <Table>
                  <THead><TRow><TH>Plano</TH><TH className="text-right">Igrejas</TH></TRow></THead>
                  <TBody>
                    {stats.by_plan.map((p) => (
                      <TRow key={p.plan}>
                        <TD><Badge tone="zinc">{p.plan}</Badge></TD>
                        <TD className="text-right tabular-nums">{p.count}</TD>
                      </TRow>
                    ))}
                  </TBody>
                </Table>
              )}
            </Card>

            <Card>
              <h3 className="mb-3 text-sm font-semibold text-zinc-500">Igrejas recentes</h3>
              {stats.recent_tenants.length === 0 ? (
                <p className="text-sm text-zinc-400">Nenhuma igreja.</p>
              ) : (
                <Table>
                  <THead><TRow><TH>Igreja</TH><TH>Plano</TH><TH className="text-right">Membros</TH><TH>Situacao</TH></TRow></THead>
                  <TBody>
                    {stats.recent_tenants.map((t) => (
                      <TRow key={t.id}>
                        <TD className="font-medium">{t.name}</TD>
                        <TD className="text-xs text-zinc-500">{t.plan}</TD>
                        <TD className="text-right tabular-nums">{t.member_count}</TD>
                        <TD><Badge tone={t.is_active ? "green" : "zinc"}>{t.is_active ? "Ativa" : "Inativa"}</Badge></TD>
                      </TRow>
                    ))}
                  </TBody>
                </Table>
              )}
            </Card>
          </div>
        </>
      )}
    </div>
  );
}

function Stat({ icon, label, value }: { icon: React.ReactNode; label: string; value: number | string }) {
  return (
    <Card className="p-3">
      <div className="flex items-center gap-1.5 text-zinc-400">{icon}<span className="text-[11px] uppercase tracking-wide">{label}</span></div>
      <div className="mt-1 text-xl font-semibold tabular-nums">{value}</div>
    </Card>
  );
}
