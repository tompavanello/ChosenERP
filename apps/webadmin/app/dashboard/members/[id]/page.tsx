"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import {
  ArrowLeft, Award, FileText, GitBranch, HeartHandshake, History, Link as LinkIcon, Pencil,
  Phone, Sparkles, User, Users, Activity, ShieldCheck,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Tabs } from "@/components/ui/tabs";
import { Avatar } from "@/components/ui/avatar";
import { Badge, type Tone } from "@/components/ui/badge";
import { Field, Select } from "@/components/ui/input";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { MemberBulkGrid } from "@/components/members/member-bulk-grid";
import { PhotoField } from "@/components/people/photo-field";
import { CardCell } from "@/components/members/card-cell";
import { CargosSection } from "@/components/members/cargos-section";
import { FamilySection } from "@/components/members/family-section";
import { HistorySection } from "@/components/members/history-section";
import { FrequencySection } from "@/components/members/frequency-section";
import { LgpdSection } from "@/components/members/lgpd-section";
import {
  getMember, getMemberTree, addRelationship, listMembers, assetURL,
  type Member, type MemberAddress, type Relationship,
} from "@/lib/api";
import { useBranches } from "@/lib/swr-hooks";
import { MEMBERSHIP_STATUS, GENDER, MARITAL_STATUS, CARGO_KINDS, RELATION_LABELS, EXIT_REASONS } from "@/lib/constants";
import { datePt, age } from "@/lib/format";

const REL_KINDS = ["spouse", "parent", "child", "discipler", "disciple", "relative"];

export default function MemberDetailPage() {
  const { id } = useParams() as { id: string };
  const { toast } = useToast();
  const { hasPerm, user } = useAuth();
  const { data: branchData } = useBranches();
  const [member, setMember] = useState<Member | null>(null);
  const [rels, setRels] = useState<Relationship[]>([]);
  const [others, setOthers] = useState<Member[]>([]);
  const [tab, setTab] = useState("dados");
  const [editando, setEditando] = useState(false);
  const [rel, setRel] = useState({ relate_member_id: "", kind: "spouse" });
  const [loading, setLoading] = useState(true);

  const canWrite = hasPerm("members.write");
  const isAdmin = user?.role === "super_admin" || user?.role === "admin_sede";

  const load = useCallback(async () => {
    const [m, t] = await Promise.all([getMember(id), getMemberTree(id)]);
    setMember(m);
    setRels(t.relationships.filter((r) => r.kind !== "self"));
    // A arvore ja traz os nomes de quem esta vinculado; a lista completa so
    // serve ao seletor de "adicionar vinculo".
    const todos = await listMembers();
    setOthers(todos.members.filter((x) => x.id !== m.id));
    setLoading(false);
  }, [id]);

  useEffect(() => {
    load().catch((e) => {
      toast(e instanceof Error ? e.message : "Erro ao carregar membro", "error");
      setLoading(false);
    });
  }, [load, toast]);

  // Deep link vindo do grid ("Familia e vinculos" => ?tab=familia). Lido do
  // window em vez de useSearchParams() para nao exigir Suspense no prerender.
  useEffect(() => {
    const alvo = new URLSearchParams(window.location.search).get("tab");
    if (alvo) setTab(alvo);
  }, []);

  async function doLink(e: React.FormEvent) {
    e.preventDefault();
    if (!rel.relate_member_id) return;
    try {
      await addRelationship(id, rel.relate_member_id, rel.kind);
      setRel({ relate_member_id: "", kind: "spouse" });
      await load();
      toast("Vinculo adicionado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao vincular", "error");
    }
  }

  const branchName = useCallback(
    (bid?: string) => {
      if (!bid) return "Sede";
      return branchData?.branches.find((b) => b.id === bid)?.name ?? "Sede";
    },
    [branchData],
  );

  if (loading) return <div className="page"><SkeletonRows rows={6} /></div>;
  if (!member) {
    return (
      <div className="page">
        <EmptyState
          icon={<Users className="h-10 w-10" />}
          title="Membro nao encontrado"
          description="O cadastro pode ter sido removido ou movido para outra filial."
        />
      </div>
    );
  }

  const st = MEMBERSHIP_STATUS[member.membership_status] ?? {
    label: member.membership_status,
    tone: "zinc",
  };

  return (
    <div className="page">
      <Link
        href="/dashboard/members"
        className="mb-3 inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-zinc-700"
      >
        <ArrowLeft className="h-3.5 w-3.5" /> Voltar para membros
      </Link>

      <PageHeader
        title={member.full_name}
        description={[
          member.nickname ? `"${member.nickname}"` : null,
          member.profession,
          branchName(member.branch_id),
        ]
          .filter(Boolean)
          .join(" - ")}
        actions={
          <>
            <CardCell
              memberId={member.id}
              cardRef={member.card_ref}
              onIssued={(ref) => setMember((m) => (m ? { ...m, card_ref: ref } : m))}
            />
            {canWrite && (
              <Button variant="outline" onClick={() => setEditando((v) => !v)}>
                <Pencil className="h-4 w-4" /> {editando ? "Fechar edicao" : "Editar"}
              </Button>
            )}
          </>
        }
      />

      <Card className="mb-4 p-3">
        <div className="flex flex-wrap items-center gap-4">
          <Avatar name={member.full_name} src={assetURL(member.photo_url)} size="lg" />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-base font-semibold tracking-tight">{member.full_name}</h2>
              <Badge tone={st.tone as Tone}>{st.label}</Badge>
              {(member.cargos ?? []).map((c) => (
                <Badge key={c.id} tone={(CARGO_KINDS[c.kind]?.tone as Tone) ?? "zinc"} className="text-[10px]">
                  {c.name}
                </Badge>
              ))}
            </div>
            <p className="mt-1 text-xs text-zinc-400">
              {[
                member.birth_date && age(member.birth_date) !== null ? `${age(member.birth_date)} anos` : null,
                member.email,
                member.whatsapp ?? member.phone,
                member.joined_at ? `membro desde ${datePt(member.joined_at)}` : null,
              ]
                .filter(Boolean)
                .join(" - ") || "Sem dados de contato"}
            </p>
          </div>
        </div>
      </Card>

      {editando && canWrite && (
        <Card className="mb-4 p-4">
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <Pencil className="h-4 w-4 text-sky-600" />
            <h3 className="text-sm font-semibold text-zinc-700">Editar cadastro</h3>
            <span className="text-xs text-zinc-400">
              Altere os campos e salve; o complemento traz documentos, endereco (CEP automatico), cargos e mais
            </span>
          </div>
          <div className="mb-4">
            <PhotoField
              memberId={member.id}
              name={member.full_name}
              photoUrl={member.photo_url}
              onChange={(url) => setMember((m) => (m ? { ...m, photo_url: url } : m))}
            />
          </div>
          <MemberBulkGrid
            mode="edit"
            members={[member]}
            focusId={member.id}
            onSaved={async () => { setEditando(false); await load(); }}
          />
        </Card>
      )}

      <Tabs
        tabs={[
          { key: "dados", label: "Dados Pessoais", icon: <User className="h-4 w-4" /> },
          { key: "contato", label: "Contato", icon: <Phone className="h-4 w-4" /> },
          { key: "cargos", label: "Cargos", icon: <Award className="h-4 w-4" /> },
          { key: "familia", label: "Familia", icon: <Users className="h-4 w-4" /> },
          { key: "vinc", label: "Vinculos", icon: <GitBranch className="h-4 w-4" /> },
          { key: "espiritual", label: "Espiritual", icon: <Sparkles className="h-4 w-4" /> },
          { key: "freq", label: "Frequencia", icon: <Activity className="h-4 w-4" /> },
          { key: "hist", label: "Historico", icon: <History className="h-4 w-4" /> },
          { key: "docs", label: "Documentos", icon: <FileText className="h-4 w-4" /> },
          { key: "lgpd", label: "LGPD", icon: <ShieldCheck className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "dados" && (
        <Card>
          <dl className="grid grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-3">
            <Info label="Nome completo" value={member.full_name} />
            <Info label="Apelido" value={member.nickname} />
            <Info label="Nascimento" value={datePt(member.birth_date)} />
            <Info label="Sexo" value={member.gender ? GENDER[member.gender] : "-"} />
            <Info label="Estado civil" value={member.marital_status ? MARITAL_STATUS[member.marital_status] : "-"} />
            <Info label="Profissao" value={member.profession} />
            <Info
              label="Cargos"
              value={member.cargos?.length ? member.cargos.map((c) => c.name).join(", ") : "-"}
            />
            <Info label="CPF" value={member.cpf} />
            <Info label="RG" value={member.rg} />
            <Info label="Status" value={st.label} />
            <Info label="Classificacao no Rol" value={member.roll_class === "professo" ? "Professo" : "Nao professo"} />
            <Info label="Filial" value={branchName(member.branch_id)} />
            {member.exit_reason && (
              <Info label="Motivo da baixa" value={EXIT_REASONS[member.exit_reason] ?? member.exit_reason} />
            )}
            {member.exited_at && <Info label="Data de saida" value={datePt(member.exited_at)} />}
          </dl>
          <p className="mt-4 border-t border-zinc-100 pt-2 text-xs text-zinc-400 dark:border-zinc-800">
            Para alterar estes dados use <span className="font-medium">Editar</span> - o cadastro
            em grid, com foto, documentos, endereco (CEP automatico) e cargos.
          </p>
        </Card>
      )}

      {tab === "contato" && (
        <Card>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Info label="E-mail" value={member.email} />
            <Info label="Telefone" value={member.phone} />
            <Info label="WhatsApp" value={member.whatsapp} />
            <Info label="Endereco" value={addressLine(member.address)} />
          </dl>
        </Card>
      )}

      {tab === "cargos" && (
        <CargosSection memberId={member.id} canWrite={canWrite} onChanged={load} />
      )}

      {tab === "familia" && (
        <FamilySection
          memberId={member.id}
          memberName={member.full_name}
          canWrite={canWrite}
          onChanged={load}
        />
      )}

      {tab === "vinc" && (
        <div className="space-y-4">
          {canWrite && (
            <Card className="p-3">
              <h3 className="mb-3 text-sm font-semibold">Adicionar vinculo</h3>
              <form onSubmit={doLink} className="flex flex-wrap items-end gap-3">
                <Field label="Pessoa" className="min-w-48 flex-1">
                  <Select
                    className="h-8 text-sm"
                    value={rel.relate_member_id}
                    onChange={(e) => setRel({ ...rel, relate_member_id: e.target.value })}
                  >
                    <option value="">Selecione...</option>
                    {others.map((m) => (
                      <option key={m.id} value={m.id}>{m.full_name}</option>
                    ))}
                  </Select>
                </Field>
                <Field label="Relacionamento" className="w-44">
                  <Select
                    className="h-8 text-sm"
                    value={rel.kind}
                    onChange={(e) => setRel({ ...rel, kind: e.target.value })}
                  >
                    {REL_KINDS.map((k) => (
                      <option key={k} value={k}>{RELATION_LABELS[k] ?? k}</option>
                    ))}
                  </Select>
                </Field>
                <Button className="h-8 text-sm" type="submit">
                  <LinkIcon className="h-4 w-4" /> Vincular
                </Button>
              </form>
              <p className="mt-2 text-xs text-zinc-400">
                Para agrupar por residencia (com endereco e chefe da familia), use a aba{" "}
                <span className="font-medium">Familia</span>.
              </p>
            </Card>
          )}

          <Card className="p-3">
            <h3 className="mb-3 text-sm font-semibold">Arvore de relacionamentos</h3>
            {rels.length === 0 ? (
              <EmptyState
                icon={<GitBranch className="h-8 w-8" />}
                title="Sem vinculos"
                description="Adicione conjuge, filhos, discipulado e outros relacionamentos."
              />
            ) : (
              <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
                {rels.map((r) => (
                  <li key={r.id} className="flex items-center justify-between gap-2 py-2 text-sm">
                    <span className="flex min-w-0 items-center gap-2 font-medium">
                      <Avatar name={r.related_name} size="sm" />
                      <Link
                        href={`/dashboard/members/${r.related_id}`}
                        className="truncate hover:text-sky-700 dark:hover:text-sky-400"
                      >
                        {r.related_name}
                      </Link>
                    </span>
                    <Badge tone="sky" className="text-[10px]">
                      {RELATION_LABELS[r.kind] ?? r.relation}
                    </Badge>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      )}

      {tab === "espiritual" && (
        <Card>
          <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            <Info label="Filial" value={branchName(member.branch_id)} />
            <Info label="Batismo" value={datePt(member.baptism_date)} />
            <Info label="Local do batismo" value={member.baptism_location} />
            <Info label="Casamento" value={datePt(member.marriage_date)} />
            <Info label="Membro desde" value={datePt(member.joined_at)} />
            <Info
              label="Idade"
              value={
                member.birth_date && age(member.birth_date) !== null
                  ? `${age(member.birth_date)} anos`
                  : "-"
              }
            />
            <Info label="Cadastrado em" value={datePt(member.created_at)} />
          </dl>
        </Card>
      )}

      {tab === "freq" && <FrequencySection memberId={member.id} canWrite={canWrite} />}

      {tab === "hist" && <HistorySection memberId={member.id} canWrite={canWrite} />}

      {tab === "docs" && (
        <Card className="p-3">
          <div className="mb-3 flex items-center gap-2">
            <HeartHandshake className="h-4 w-4 text-sky-600" />
            <h3 className="text-sm font-semibold">Documentos</h3>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-dashed border-zinc-300 p-4 dark:border-zinc-700">
            <div>
              <p className="font-medium">Carteirinha de membro</p>
              <p className="text-xs text-zinc-400">
                QR Code com vinculo digital a igreja. O numero e unico e estavel: emitir de novo
                devolve a mesma carteirinha.
              </p>
            </div>
            <CardCell
              memberId={member.id}
              cardRef={member.card_ref}
              onIssued={(ref) => setMember((m) => (m ? { ...m, card_ref: ref } : m))}
            />
          </div>
        </Card>
      )}

      {tab === "lgpd" && <LgpdSection memberId={member.id} canWrite={canWrite} isAdmin={isAdmin} />}
    </div>
  );
}

function Info({ label, value }: { label: string; value: string | undefined | null }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-zinc-400">{label}</dt>
      <dd className="mt-0.5 text-sm font-medium">{value || "-"}</dd>
    </div>
  );
}

/** Monta uma linha de endereco para exibicao. */
function addressLine(a?: MemberAddress): string {
  if (!a) return "-";
  const line1 = [a.street, a.number, a.complement].filter(Boolean).join(", ");
  const line2 = [a.district, a.city, a.state].filter(Boolean).join(" - ");
  const cep = a.zip_code ? `CEP ${a.zip_code}` : "";
  return [line1, line2, cep].filter(Boolean).join(" - ") || "-";
}
