"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import {
  Search, Users, MessageCircle, Phone, Mail, IdCard, GitBranch, Trash2, ListPlus,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Select } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Badge, type Tone } from "@/components/ui/badge";
import { Avatar } from "@/components/ui/avatar";
import { StatCard } from "@/components/ui/stat-card";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listMembers, deleteMember, assetURL, type Member,
} from "@/lib/api";
import { useBranches } from "@/lib/swr-hooks";
import { MemberBulkGrid } from "@/components/members/member-bulk-grid";
import { CardCell } from "@/components/members/card-cell";
import { RowActions } from "@/components/people/row-actions";
import { MEMBERSHIP_STATUS, CARGO_KINDS } from "@/lib/constants";
import { datePt, age } from "@/lib/format";
import { DataTable, Column } from "@/components/ui/data-table";

const PER_PAGE = 15;

const digits = (s?: string) => (s ?? "").replace(/\D/g, "");

export default function MembersPage() {
  const { hasPerm, user } = useAuth();
  const { toast } = useToast();
  const { data: branchData } = useBranches();
  const [members, setMembers] = useState<Member[] | null>(null);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PER_PAGE);
  const [showQuick, setShowQuick] = useState(false);
  const [sortColumn, setSortColumn] = useState<string>("full_name");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");

  const canWrite = hasPerm("members.write");
  // Exclusao destrutiva: apenas a Sede (super_admin/admin_sede), alinhado ao
  // gate do backend (handleDeleteMember).
  const canDelete = user?.role === "super_admin" || user?.role === "admin_sede";

  // Nome da filial no lugar do UUID cru que aparecia na coluna.
  const branchName = useCallback(
    (id?: string) => {
      if (!id) return "Sede";
      return branchData?.branches.find((b) => b.id === id)?.name ?? "Sede";
    },
    [branchData],
  );

  const reload = useCallback(async () => {
    try {
      const r = await listMembers();
      setMembers(r.members);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar membros", "error");
      setMembers([]);
    }
  }, [toast]);

  useEffect(() => {
    reload();
  }, [reload]);

  const stats = useMemo(() => {
    const all = members ?? [];
    return {
      total: all.length,
      active: all.filter((m) => m.membership_status === "active").length,
      nonprofessed: all.filter((m) => m.membership_status === "member").length,
      inactive: all.filter((m) => m.membership_status === "inactive").length,
    };
  }, [members]);

  const filtered = useMemo(() => {
    if (!members) return [];
    const q = query.trim().toLowerCase();
    return members.filter((m) => {
      const okQ =
        !q ||
        m.full_name.toLowerCase().includes(q) ||
        (m.nickname?.toLowerCase().includes(q) ?? false) ||
        (m.email?.toLowerCase().includes(q) ?? false) ||
        (m.phone?.includes(q) ?? false) ||
        (m.cpf?.includes(q) ?? false) ||
        (m.whatsapp?.includes(q) ?? false) ||
        (m.cargos?.some((c) => c.name.toLowerCase().includes(q)) ?? false);
      const okS = !status || m.membership_status === status;
      return okQ && okS;
    });
  }, [members, query, status]);

  useEffect(() => setPage(1), [query, status, pageSize]);

  async function remove(m: Member) {
    if (!confirm(`Excluir definitivamente o membro "${m.full_name}"? Historico, cargos, presencas e vinculos serao removidos. Esta acao nao pode ser desfeita.`)) return;
    try {
      await deleteMember(m.id);
      toast("Membro excluido.");
      await reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  function handleSort(column: string) {
    if (sortColumn === column) {
      setSortDirection((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortColumn(column);
      setSortDirection("asc");
    }
  }

  const columns: Column<Member>[] = useMemo(
    () => [
      {
        key: "full_name",
        label: "Membro",
        sortable: true,
        sortValue: (m) => m.full_name,
        render: (m) => {
          const anos = age(m.birth_date);
          const secundaria = [
            m.nickname ? `"${m.nickname}"` : null,
            anos !== null ? `${anos} anos` : null,
            m.profession,
            branchName(m.branch_id),
          ].filter(Boolean);
          return (
            <Link href={`/dashboard/members/${m.id}`} className="group flex items-center gap-3">
              <Avatar name={m.full_name} src={assetURL(m.photo_url)} size="md" />
              <div className="min-w-0">
                <p className="truncate font-medium group-hover:text-sky-700 dark:group-hover:text-sky-400">
                  {m.full_name}
                </p>
                <p className="truncate text-xs text-zinc-400">{secundaria.join(" - ")}</p>
              </div>
            </Link>
          );
        },
      },
      {
        key: "cargos",
        label: "Cargos",
        width: "w-52",
        render: (m) => {
          if (!m.cargos || m.cargos.length === 0) {
            return <span className="text-xs text-zinc-300 dark:text-zinc-600">-</span>;
          }
          return (
            <div className="flex flex-wrap gap-1">
              {m.cargos.map((c) => (
                <Badge key={c.id} tone={(CARGO_KINDS[c.kind]?.tone as Tone) ?? "zinc"} className="text-[10px]">
                  {c.name}
                </Badge>
              ))}
            </div>
          );
        },
      },
      {
        key: "contato",
        label: "Contato",
        width: "w-56",
        render: (m) => {
          const zap = digits(m.whatsapp);
          const fone = digits(m.phone);
          const principal = m.whatsapp ?? m.phone ?? m.email;
          if (!principal) return <span className="text-xs text-zinc-300 dark:text-zinc-600">-</span>;
          return (
            <div className="flex items-center gap-2">
              {zap && (
                <a
                  href={`https://wa.me/55${zap}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  title={`WhatsApp ${m.whatsapp}`}
                  className="text-emerald-600 hover:opacity-70"
                >
                  <MessageCircle className="h-3.5 w-3.5" />
                </a>
              )}
              {fone && (
                <a href={`tel:${fone}`} title={`Ligar ${m.phone}`} className="text-zinc-400 hover:text-zinc-700">
                  <Phone className="h-3.5 w-3.5" />
                </a>
              )}
              {m.email && (
                <a href={`mailto:${m.email}`} title={m.email} className="text-zinc-400 hover:text-zinc-700">
                  <Mail className="h-3.5 w-3.5" />
                </a>
              )}
              <span className="truncate text-xs text-zinc-500">{principal}</span>
            </div>
          );
        },
      },
      {
        key: "card_ref",
        label: "Carteirinha",
        width: "w-44",
        render: (m) => (
          <CardCell
            memberId={m.id}
            cardRef={m.card_ref}
            onIssued={(ref) =>
              setMembers((atual) => atual?.map((x) => (x.id === m.id ? { ...x, card_ref: ref } : x)) ?? atual)
            }
          />
        ),
      },
      {
        key: "membership_status",
        label: "Status",
        sortable: true,
        width: "w-28",
        sortValue: (m) => MEMBERSHIP_STATUS[m.membership_status]?.label ?? m.membership_status,
        render: (m) => {
          const st = MEMBERSHIP_STATUS[m.membership_status] ?? { label: m.membership_status, tone: "zinc" };
          return <Badge tone={st.tone as Tone}>{st.label}</Badge>;
        },
      },
      {
        key: "joined_at",
        label: "Desde",
        sortable: true,
        width: "w-28",
        // Ordena pela data ISO, nao pela data ja formatada (dd/mm/yyyy ordenaria
        // por dia). Campo vazio vai para o fim em ordem crescente.
        sortValue: (m) => m.joined_at ?? "9999-12-31",
        render: (m) => (
          <span className="tnum text-xs text-zinc-500">{m.joined_at ? datePt(m.joined_at) : "-"}</span>
        ),
      },
      {
        key: "actions",
        label: "",
        align: "right",
        width: "w-12",
        render: (m) => (
          <div className="flex justify-end">
            <RowActions
              items={[
                {
                  label: "Abrir ficha",
                  icon: <IdCard className="h-3.5 w-3.5" />,
                  onClick: () => window.location.assign(`/dashboard/members/${m.id}`),
                },
                {
                  label: "Familia e vinculos",
                  icon: <GitBranch className="h-3.5 w-3.5" />,
                  onClick: () => window.location.assign(`/dashboard/members/${m.id}?tab=familia`),
                },
                {
                  label: "Excluir",
                  icon: <Trash2 className="h-3.5 w-3.5" />,
                  danger: true,
                  disabled: !canDelete,
                  onClick: () => remove(m),
                },
              ]}
            />
          </div>
        ),
      },
    ],
    [branchName, canDelete],
  );

  /** Card do mobile: as mesmas informacoes essenciais, empilhadas. */
  const renderMobileCard = useCallback(
    (m: Member) => {
      const st = MEMBERSHIP_STATUS[m.membership_status] ?? { label: m.membership_status, tone: "zinc" };
      const anos = age(m.birth_date);
      const zap = digits(m.whatsapp);
      return (
        <div className="p-3">
          <div className="flex items-start gap-3">
            <Avatar name={m.full_name} src={assetURL(m.photo_url)} size="md" />
            <div className="min-w-0 flex-1">
              <Link
                href={`/dashboard/members/${m.id}`}
                className="block truncate font-medium hover:text-sky-700 dark:hover:text-sky-400"
              >
                {m.full_name}
              </Link>
              <p className="truncate text-xs text-zinc-400">
                {[m.nickname ? `"${m.nickname}"` : null, anos !== null ? `${anos} anos` : null, branchName(m.branch_id)]
                  .filter(Boolean)
                  .join(" - ")}
              </p>
              {m.cargos && m.cargos.length > 0 && (
                <div className="mt-1 flex flex-wrap gap-1">
                  {m.cargos.map((c) => (
                    <Badge key={c.id} tone={(CARGO_KINDS[c.kind]?.tone as Tone) ?? "zinc"} className="text-[10px]">
                      {c.name}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
            <Badge tone={st.tone as Tone} className="shrink-0 text-[10px]">{st.label}</Badge>
          </div>

          <div className="mt-3 flex flex-wrap items-center gap-2">
            <CardCell memberId={m.id} cardRef={m.card_ref} />
            {zap && (
              <a
                href={`https://wa.me/55${zap}`}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-xs text-emerald-600"
              >
                <MessageCircle className="h-3.5 w-3.5" /> WhatsApp
              </a>
            )}
            {digits(m.phone) && (
              <a href={`tel:${digits(m.phone)}`} className="inline-flex items-center gap-1 text-xs text-zinc-500">
                <Phone className="h-3.5 w-3.5" /> Ligar
              </a>
            )}
            {m.email && (
              <a href={`mailto:${m.email}`} className="inline-flex items-center gap-1 text-xs text-zinc-500">
                <Mail className="h-3.5 w-3.5" /> E-mail
              </a>
            )}
          </div>
        </div>
      );
    },
    [branchName],
  );

  return (
    <div className="page">
      <PageHeader
        title="Membros"
        description="Cadastro, cargos, familia e carteirinha digital em um so lugar"
        actions={
          canWrite && (
            <Button variant="outline" onClick={() => setShowQuick((v) => !v)}>
              <ListPlus className="h-4 w-4" /> {showQuick ? "Fechar cadastro rapido" : "Cadastro rapido"}
            </Button>
          )
        }
      />

      {showQuick && canWrite && (
        <Card className="mb-4 p-4">
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <Users className="h-4 w-4 text-sky-600" />
            <h3 className="text-sm font-semibold text-zinc-700">Cadastro rapido de membros</h3>
            <span className="text-xs text-zinc-400">
              Cada linha e um membro; use o botao de complemento para documentos, endereco (CEP automatico), cargos e mais.
              A edicao de um membro e feita na ficha dele.
            </span>
          </div>
          <MemberBulkGrid onSaved={reload} />
        </Card>
      )}

      <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Total" value={members ? String(stats.total) : "..."} icon={Users} />
        <StatCard label="Professos" value={members ? String(stats.active) : "..."} tone="green" />
        <StatCard label="Nao professos" value={members ? String(stats.nonprofessed) : "..."} tone="sky" />
        <StatCard label="Inativos" value={members ? String(stats.inactive) : "..."} tone="zinc" />
      </div>

      <Card className="mb-3 p-3">
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative min-w-52 flex-1 sm:max-w-sm">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-400" />
            <Input
              className="h-8 pl-8 text-sm"
              placeholder="Buscar por nome, apelido, cargo, e-mail, telefone ou CPF"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
          <Select className="h-8 w-36 text-sm" value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">Todos os status</option>
            {Object.entries(MEMBERSHIP_STATUS).map(([k, v]) => (
              <option key={k} value={k}>{v.label}</option>
            ))}
          </Select>
          <Select className="h-8 w-28 text-sm" value={String(pageSize)} onChange={(e) => setPageSize(Number(e.target.value))}>
            <option value="10">10 / pag</option>
            <option value="15">15 / pag</option>
            <option value="25">25 / pag</option>
            <option value="50">50 / pag</option>
          </Select>
        </div>
      </Card>

      {members === null ? (
        <Card>
          <SkeletonRows rows={8} />
        </Card>
      ) : filtered.length === 0 ? (
        <Card>
          <EmptyState
            icon={<Users className="h-10 w-10" />}
            title="Nenhum membro encontrado"
            description="Ajuste a busca ou use o Cadastro rapido para incluir membros."
          />
        </Card>
      ) : (
        <DataTable
          columns={columns}
          data={filtered}
          keyExtractor={(m) => m.id}
          compact
          hoverable
          renderMobileCard={renderMobileCard}
          sort={{ column: sortColumn, direction: sortDirection, onSort: handleSort }}
          pagination={{
            page,
            pageSize,
            total: filtered.length,
            onPageChange: setPage,
            onPageSizeChange: setPageSize,
            pageSizeOptions: [10, 15, 25, 50],
          }}
        />
      )}
    </div>
  );
}
