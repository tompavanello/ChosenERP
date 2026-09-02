"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { Search, UserPlus, QrCode, Users } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Select } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Avatar } from "@/components/ui/avatar";
import { Pagination } from "@/components/ui/pagination";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { listMembers, createMember, issueCard, type Member } from "@/lib/api";
import { MEMBERSHIP_STATUS } from "@/lib/constants";
import { datePt } from "@/lib/format";

const PER_PAGE = 10;

export default function MembersPage() {
  const { hasPerm } = useAuth();
  const { toast } = useToast();
  const [members, setMembers] = useState<Member[] | null>(null);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ first_name: "", last_name: "", email: "", phone: "", gender: "", marital_status: "", membership_status: "member" });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    listMembers().then((r) => setMembers(r.members)).catch((e) => toast(e.message, "error"));
  }, [toast]);

  const filtered = useMemo(() => {
    if (!members) return [];
    const q = query.trim().toLowerCase();
    return members.filter((m) => {
      const okQ = !q || m.full_name.toLowerCase().includes(q) || (m.email?.toLowerCase().includes(q) ?? false);
      const okS = !status || m.membership_status === status;
      return okQ && okS;
    });
  }, [members, query, status]);

  const total = filtered.length;
  const pageItems = filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE);

  useEffect(() => setPage(1), [query, status]);

  async function addMember(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await createMember(form as Partial<Member>);
      toast("Membro adicionado.");
      setOpen(false);
      setMembers(await listMembers().then((r) => r.members));
      setForm({ first_name: "", last_name: "", email: "", phone: "", gender: "", marital_status: "", membership_status: "member" });
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setSaving(false);
    }
  }

  async function onCard(m: Member) {
    try {
      const res = await issueCard(m.id);
      toast(`Carteirinha emitida para ${res.member}.`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="mx-auto max-w-6xl">
      <PageHeader
        title="Membros"
        description={`${total} membro(s) no escopo`}
        actions={hasPerm("members.write") ? (
          <Button onClick={() => setOpen(true)}><UserPlus className="h-4 w-4" /> Novo Membro</Button>
        ) : undefined}
      />

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-48">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
          <Input className="pl-9" placeholder="Buscar por nome ou e-mail" value={query} onChange={(e) => setQuery(e.target.value)} />
        </div>
        <Select className="w-44" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">Todos os status</option>
          {Object.entries(MEMBERSHIP_STATUS).map(([k, v]) => (<option key={k} value={k}>{v.label}</option>))}
        </Select>
      </div>

      <Card className="overflow-hidden p-0">
        {members === null ? (
          <SkeletonRows />
        ) : pageItems.length === 0 ? (
          <EmptyState icon={<Users className="h-10 w-10" />} title="Nenhum membro encontrado" description="Ajuste a busca ou adicione um novo membro." />
        ) : (
          <>
            <Table>
              <THead>
                <TRow>
                  <TH>Membro</TH>
                  <TH>Contato</TH>
                  <TH>Nascimento</TH>
                  <TH>Status</TH>
                  <TH className="text-right">Ações</TH>
                </TRow>
              </THead>
              <TBody>
                {pageItems.map((m) => {
                  const st = MEMBERSHIP_STATUS[m.membership_status] ?? { label: m.membership_status, tone: "zinc" as const };
                  return (
                    <TRow key={m.id}>
                      <TD>
                        <Link href={`/dashboard/members/${m.id}`} className="flex items-center gap-3">
                          <Avatar name={m.full_name} size="sm" />
                          <div>
                            <p className="font-medium hover:text-violet-700">{m.full_name}</p>
                            {m.office || m.profession ? <p className="text-xs text-zinc-400">{m.office || m.profession}</p> : null}
                          </div>
                        </Link>
                      </TD>
                      <TD>
                        <p>{m.phone ?? m.whatsapp ?? "—"}</p>
                        <p className="text-xs text-zinc-400">{m.email ?? ""}</p>
                      </TD>
                      <TD className="text-zinc-500">{datePt(m.birth_date)}</TD>
                      <TD><Badge tone={st.tone as "zinc"}>{st.label}</Badge></TD>
                      <TD className="text-right">
                        <Button variant="ghost" size="sm" onClick={() => onCard(m)}><QrCode className="h-3.5 w-3.5" /> Carteirinha</Button>
                      </TD>
                    </TRow>
                  );
                })}
              </TBody>
            </Table>
            <div className="border-t border-zinc-100 dark:border-zinc-800">
              <Pagination page={page} total={total} perPage={PER_PAGE} onChange={setPage} />
            </div>
          </>
        )}
      </Card>

      <Modal open={open} onClose={() => setOpen(false)} title="Novo Membro">
        <form onSubmit={addMember} className="grid grid-cols-2 gap-3">
          <Field label="Nome *"><Input required value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></Field>
          <Field label="Sobrenome *"><Input required value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></Field>
          <Field label="E-mail"><Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
          <Field label="Telefone"><Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
          <Field label="Sexo"><Select value={form.gender} onChange={(e) => setForm({ ...form, gender: e.target.value })}><option value="">—</option><option value="male">Masculino</option><option value="female">Feminino</option></Select></Field>
          <Field label="Estado civil"><Select value={form.marital_status} onChange={(e) => setForm({ ...form, marital_status: e.target.value })}><option value="">—</option><option value="single">Solteiro(a)</option><option value="married">Casado(a)</option><option value="divorced">Divorciado(a)</option><option value="widowed">Viúvo(a)</option></Select></Field>
          <Field label="Status"><Select value={form.membership_status} onChange={(e) => setForm({ ...form, membership_status: e.target.value })}>{Object.entries(MEMBERSHIP_STATUS).map(([k, v]) => (<option key={k} value={k}>{v.label}</option>))}</Select></Field>
          <div className="col-span-2 flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
