"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import {
  ArrowLeft, QrCode, Pencil, Link as LinkIcon, User, Phone, Sparkles, GitBranch, FileText, HeartHandshake,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Tabs } from "@/components/ui/tabs";
import { Avatar } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Input, Field, Select } from "@/components/ui/input";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getMember, getMemberTree, updateMember, addRelationship, issueCard, listMembers, type Member, type Relationship } from "@/lib/api";
import { MEMBERSHIP_STATUS, GENDER, MARITAL_STATUS, OFFICES, RELATION_LABELS } from "@/lib/constants";
import { currency, datePt } from "@/lib/format";

const REL_KINDS = ["spouse", "parent", "child", "discipler", "disciple", "relative"];

export default function MemberDetailPage() {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const { toast } = useToast();
  const [member, setMember] = useState<Member | null>(null);
  const [rels, setRels] = useState<Relationship[]>([]);
  const [others, setOthers] = useState<Member[]>([]);
  const [tab, setTab] = useState("dados");
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState<Record<string, string>>({});
  const [rel, setRel] = useState({ relate_member_id: "", kind: "spouse" });
  const [loading, setLoading] = useState(true);

  async function load(target?: Member) {
    const [m, t, all] = await Promise.all([
      target ? Promise.resolve(target) : getMember(id),
      getMemberTree(id),
      listMembers(),
    ]);
    setMember(m);
    setRels(t.relationships.filter((r) => r.kind !== "self"));
    setOthers(all.members.filter((x) => x.id !== m.id));
    setLoading(false);
  }

  useEffect(() => {
    load().catch((e) => toast(e.message, "error"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  function startEdit() {
    setForm({
      phone: member?.phone ?? "", whatsapp: member?.whatsapp ?? "", email: member?.email ?? "",
      birth_date: member?.birth_date ?? "", gender: member?.gender ?? "", marital_status: member?.marital_status ?? "",
      profession: member?.profession ?? "", office: member?.office ?? "", nickname: member?.nickname ?? "",
      membership_status: member?.membership_status ?? "member",
    });
    setEditing(true);
  }

  async function saveEdit(e: React.FormEvent) {
    e.preventDefault();
    try {
      const updated = await updateMember(id, form);
      setMember(updated);
      setEditing(false);
      toast("Perfil atualizado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function doLink(e: React.FormEvent) {
    e.preventDefault();
    if (!rel.relate_member_id) return;
    try {
      await addRelationship(id, rel.relate_member_id, rel.kind);
      setRel({ relate_member_id: "", kind: "spouse" });
      await load();
      toast("Vínculo adicionado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function onCard() {
    try {
      const res = await issueCard(id);
      toast(`Carteirinha emitida: ${res.card_ref}.`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  if (loading) return <div className="mx-auto max-w-4xl"><SkeletonRows rows={6} /></div>;
  if (!member) return <EmptyState title="Membro não encontrado" />;

  const st = MEMBERSHIP_STATUS[member.membership_status] ?? { label: member.membership_status, tone: "zinc" as const };

  return (
    <div className="mx-auto max-w-4xl">
      <Link href="/dashboard/members" className="mb-4 inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-200">
        <ArrowLeft className="h-4 w-4" /> Voltar para membros
      </Link>

      <div className="card mb-6 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <Avatar name={member.full_name} size="lg" />
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-xl font-semibold">{member.full_name}</h2>
              <Badge tone={st.tone as "zinc"}>{st.label}</Badge>
            </div>
            <p className="text-sm text-zinc-500">{member.office ? OFFICES[member.office] ?? member.office : "Membro"} {member.profession ? `· ${member.profession}` : ""}</p>
          </div>
        </div>
        <div className="flex gap-2">
          {!editing && <Button variant="outline" onClick={startEdit}><Pencil className="h-4 w-4" /> Editar</Button>}
          <Button variant="ghost" onClick={onCard}><QrCode className="h-4 w-4" /> Carteirinha</Button>
        </div>
      </div>

      <Tabs
        tabs={[
          { key: "dados", label: "Dados Pessoais", icon: <User className="h-4 w-4" /> },
          { key: "contato", label: "Contato", icon: <Phone className="h-4 w-4" /> },
          { key: "vinc", label: "Vínculos & Família", icon: <GitBranch className="h-4 w-4" /> },
          { key: "espiritual", label: "Espiritual", icon: <Sparkles className="h-4 w-4" /> },
          { key: "docs", label: "Documentos", icon: <FileText className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "dados" && (
        editing ? (
          <form onSubmit={saveEdit} className="card grid grid-cols-2 gap-3">
            <Field label="Apelido"><Input value={form.nickname} onChange={(e) => setForm({ ...form, nickname: e.target.value })} /></Field>
            <Field label="Nascimento"><Input type="date" value={form.birth_date} onChange={(e) => setForm({ ...form, birth_date: e.target.value })} /></Field>
            <Field label="Sexo"><Select value={form.gender} onChange={(e) => setForm({ ...form, gender: e.target.value })}><option value="">—</option>{Object.entries(GENDER).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}</Select></Field>
            <Field label="Estado civil"><Select value={form.marital_status} onChange={(e) => setForm({ ...form, marital_status: e.target.value })}><option value="">—</option>{Object.entries(MARITAL_STATUS).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}</Select></Field>
            <Field label="Profissão"><Input value={form.profession} onChange={(e) => setForm({ ...form, profession: e.target.value })} /></Field>
            <Field label="Cargo"><Select value={form.office} onChange={(e) => setForm({ ...form, office: e.target.value })}><option value="">—</option>{Object.keys(OFFICES).map((k) => (<option key={k} value={k}>{OFFICES[k]}</option>))}</Select></Field>
            <Field label="Status"><Select value={form.membership_status} onChange={(e) => setForm({ ...form, membership_status: e.target.value })}>{Object.entries(MEMBERSHIP_STATUS).map(([k, v]) => (<option key={k} value={k}>{v.label}</option>))}</Select></Field>
            <div className="col-span-2 flex justify-end gap-2 pt-2">
              <Button variant="ghost" type="button" onClick={() => setEditing(false)}>Cancelar</Button>
              <Button type="submit">Salvar</Button>
            </div>
          </form>
        ) : (
          <div className="card">
            <dl className="grid grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-3">
              <Info label="Nome completo" value={member.full_name} />
              <Info label="Apelido" value={member.nickname} />
              <Info label="Nascimento" value={datePt(member.birth_date)} />
              <Info label="Sexo" value={member.gender ? GENDER[member.gender] : "—"} />
              <Info label="Estado civil" value={member.marital_status ? MARITAL_STATUS[member.marital_status] : "—"} />
              <Info label="Profissão" value={member.profession} />
              <Info label="Cargo" value={member.office ? OFFICES[member.office] ?? member.office : "—"} />
              <Info label="Status" value={st.label} />
            </dl>
          </div>
        )
      )}

      {tab === "contato" && (
        editing ? (
          <form onSubmit={saveEdit} className="card grid grid-cols-2 gap-3">
            <Field label="E-mail"><Input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
            <Field label="Telefone"><Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
            <Field label="WhatsApp"><Input value={form.whatsapp} onChange={(e) => setForm({ ...form, whatsapp: e.target.value })} /></Field>
            <div className="col-span-2 flex justify-end gap-2 pt-2">
              <Button variant="ghost" type="button" onClick={() => setEditing(false)}>Cancelar</Button>
              <Button type="submit">Salvar</Button>
            </div>
          </form>
        ) : (
          <div className="card">
            <dl className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <Info label="E-mail" value={member.email} />
              <Info label="Telefone" value={member.phone} />
              <Info label="WhatsApp" value={member.whatsapp} />
            </dl>
          </div>
        )
      )}

      {tab === "vinc" && (
        <div className="space-y-4">
          <Card>
            <h3 className="mb-3 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Adicionar vínculo</h3>
            <form onSubmit={doLink} className="flex flex-wrap items-end gap-3">
              <div className="flex-1 min-w-48">
                <label className="label">Pessoa</label>
                <Select value={rel.relate_member_id} onChange={(e) => setRel({ ...rel, relate_member_id: e.target.value })}>
                  <option value="">Selecione...</option>
                  {others.map((m) => (<option key={m.id} value={m.id}>{m.full_name}</option>))}
                </Select>
              </div>
              <div className="w-44">
                <label className="label">Relacionamento</label>
                <Select value={rel.kind} onChange={(e) => setRel({ ...rel, kind: e.target.value })}>
                  {REL_KINDS.map((k) => (<option key={k} value={k}>{RELATION_LABELS[k] ?? k}</option>))}
                </Select>
              </div>
              <Button type="submit"><LinkIcon className="h-4 w-4" /> Vincular</Button>
            </form>
          </Card>
          <Card>
            <h3 className="mb-3 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Árvore de relacionamentos</h3>
            {rels.length === 0 ? (
              <EmptyState icon={<GitBranch className="h-8 w-8" />} title="Sem vínculos" description="Adicione família e discipulado." />
            ) : (
              <ul className="space-y-2">
                {rels.map((r) => (
                  <li key={r.id} className="flex items-center justify-between rounded-md bg-zinc-50 px-3 py-2 text-sm dark:bg-zinc-900">
                    <span className="flex items-center gap-2 font-medium">
                      <Avatar name={r.related_name} size="sm" /> {r.related_name}
                    </span>
                    <Badge tone="violet">{RELATION_LABELS[r.kind] ?? r.relation}</Badge>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      )}

      {tab === "espiritual" && (
        <Card>
          <dl className="grid grid-cols-2 gap-4 sm:grid-cols-3">
            <Info label="Filiação" value={member.branch_id ? "Vinculado" : "—"} />
            <Info label="Batismo" value={datePt(member.birth_date)} />
            <Info label="Membro desde" value={datePt(member.created_at)} />
          </dl>
        </Card>
      )}

      {tab === "docs" && (
        <Card>
          <div className="mb-4 flex items-center gap-2">
            <HeartHandshake className="h-4 w-4 text-violet-600" />
            <h3 className="text-sm font-semibold text-zinc-700 dark:text-zinc-200">Documentos</h3>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-dashed border-zinc-300 p-4 dark:border-zinc-700">
            <div>
              <p className="font-medium">Carteirinha de membro</p>
              <p className="text-xs text-zinc-400">QR Code com vínculo digital à igreja</p>
            </div>
            <Button onClick={onCard}><QrCode className="h-4 w-4" /> Emitir</Button>
          </div>
        </Card>
      )}
    </div>
  );
}

function Info({ label, value }: { label: string; value: string | undefined | null }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-zinc-400">{label}</dt>
      <dd className="mt-0.5 text-sm font-medium">{value || "—"}</dd>
    </div>
  );
}
