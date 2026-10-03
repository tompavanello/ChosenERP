"use client";

import { useEffect, useMemo, useState } from "react";
import { BookOpen, Smartphone, Users } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Tabs } from "@/components/ui/tabs";
import { useToast } from "@/components/ui/toast";
import {
  getMemberAppOverview, listMaterials,
  type MemberAppUser, type StudyMaterial,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

export default function MemberAppPage() {
  const { toast } = useToast();
  const [tab, setTab] = useState("acessos");
  const [users, setUsers] = useState<MemberAppUser[] | null>(null);
  const [totals, setTotals] = useState({ total: 0, active: 0, never_login: 0 });
  const [materials, setMaterials] = useState<StudyMaterial[] | null>(null);
  const [q, setQ] = useState("");

  useEffect(() => {
    getMemberAppOverview()
      .then((r) => {
        setUsers(r.users);
        setTotals(r.totals);
      })
      .catch((e) => {
        toast(e instanceof Error ? e.message : "Erro ao carregar", "error");
        setUsers([]);
      });
    listMaterials()
      .then((r) => setMaterials(r.materials))
      .catch(() => setMaterials([]));
  }, [toast]);

  const filtered = useMemo(() => {
    if (!users) return [];
    const s = q.trim().toLowerCase();
    if (!s) return users;
    return users.filter(
      (u) => u.member_name.toLowerCase().includes(s) || (u.email ?? "").toLowerCase().includes(s) || (u.phone ?? "").includes(s),
    );
  }, [users, q]);

  return (
    <div className="page">
      <PageHeader title="Área do Membro" description="Acessos ao app, uso (último login e dispositivos) e materiais" />

      <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
        <Stat label="Acessos de membro" value={totals.total} />
        <Stat label="Ativos" value={totals.active} tone="green" />
        <Stat label="Nunca acessaram" value={totals.never_login} tone="amber" />
      </div>

      <Tabs
        tabs={[
          { key: "acessos", label: "Acessos", icon: <Users className="h-4 w-4" /> },
          { key: "materiais", label: "Materiais", icon: <BookOpen className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "acessos" && (
        <>
          <div className="mb-3 flex justify-end">
            <Input
              className="h-8 w-64 text-sm"
              placeholder="Buscar por nome, e-mail ou telefone..."
              value={q}
              onChange={(e) => setQ(e.target.value)}
            />
          </div>
          <Card className="overflow-hidden p-0">
            {users === null ? (
              <SkeletonRows rows={6} />
            ) : filtered.length === 0 ? (
              <EmptyState icon={<Smartphone className="h-10 w-10" />} title="Nenhum acesso" description="Crie o acesso do membro em Membros → aba Acesso." />
            ) : (
              <Table>
                <THead><TRow><TH>Membro</TH><TH>Identificador</TH><TH>Situação</TH><TH>Último acesso</TH><TH>Dispositivos</TH></TRow></THead>
                <TBody>
                  {filtered.map((u) => (
                    <TRow key={u.user_id}>
                      <TD className="font-medium">{u.member_name}</TD>
                      <TD className="text-zinc-500">{u.email || u.phone || "-"}</TD>
                      <TD>
                        {!u.is_active ? (
                          <Badge tone="zinc">Inativo</Badge>
                        ) : u.must_change_password ? (
                          <Badge tone="amber">Senha provisória</Badge>
                        ) : (
                          <Badge tone="green">Ativo</Badge>
                        )}
                      </TD>
                      <TD className="text-zinc-500">{u.last_login_at ? dateTimePt(u.last_login_at) : "Nunca acessou"}</TD>
                      <TD className="text-zinc-500">
                        {u.devices > 0 ? (
                          <span className="inline-flex items-center gap-1">
                            <Smartphone className="h-3.5 w-3.5" /> {u.devices}
                            {u.last_device_at && <span className="text-xs">· {dateTimePt(u.last_device_at)}</span>}
                          </span>
                        ) : (
                          "-"
                        )}
                      </TD>
                    </TRow>
                  ))}
                </TBody>
              </Table>
            )}
          </Card>
        </>
      )}

      {tab === "materiais" && (
        <Card className="overflow-hidden p-0">
          {materials === null ? (
            <SkeletonRows rows={5} />
          ) : materials.length === 0 ? (
            <EmptyState icon={<BookOpen className="h-10 w-10" />} title="Nenhum material" description="Publique materiais em Materiais." />
          ) : (
            <Table>
              <THead><TRow><TH>Material</TH><TH>Destino</TH><TH>Tipo</TH><TH>Situação</TH></TRow></THead>
              <TBody>
                {materials.map((m) => (
                  <TRow key={m.id}>
                    <TD className="font-medium">{m.title}</TD>
                    <TD className="text-zinc-500">{m.group_name ?? "Igreja (todos)"}</TD>
                    <TD className="text-zinc-500">{m.kind === "link" ? "Link" : "Arquivo"}</TD>
                    <TD><Badge tone={m.is_published ? "green" : "zinc"}>{m.is_published ? "Publicado" : "Rascunho"}</Badge></TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}
    </div>
  );
}

function Stat({ label, value, tone = "sky" }: { label: string; value: number; tone?: "sky" | "green" | "amber" }) {
  const toneClass = tone === "green" ? "text-emerald-600" : tone === "amber" ? "text-amber-600" : "text-sky-600";
  return (
    <Card className="p-4">
      <p className="text-xs uppercase tracking-wide text-zinc-400">{label}</p>
      <p className={`mt-1 text-2xl font-bold ${toneClass}`}>{value}</p>
    </Card>
  );
}
