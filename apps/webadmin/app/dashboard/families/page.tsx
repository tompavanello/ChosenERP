"use client";

import { useEffect, useState } from "react";
import { Plus, Users2, Search, Users } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Select } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { listFamilies, createFamily, addFamilyMember, listMembers, type Family, type Member } from "@/lib/api";
import { RELATION_LABELS } from "@/lib/constants";

export default function FamiliesPage() {
  const { toast } = useToast();
  const [families, setFamilies] = useState<Family[] | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [link, setLink] = useState<Family | null>(null);
  const [linkForm, setLinkForm] = useState({ member_id: "", relate_id: "", relation: "relative" });

  useEffect(() => {
    listFamilies().then((r) => setFamilies(r.families)).catch((e) => toast(e.message, "error"));
    listMembers().then((r) => setMembers(r.members)).catch(() => {});
  }, [toast]);

  const filtered = (families ?? []).filter((f) => !query.trim() || f.name.toLowerCase().includes(query.trim().toLowerCase()));

  async function add(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createFamily(newName);
      toast("Família criada.");
      setOpen(false);
      setNewName("");
      setFamilies(await listFamilies().then((r) => r.families));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function doLink(e: React.FormEvent) {
    e.preventDefault();
    if (!link || !linkForm.member_id) return;
    try {
      await addFamilyMember(link.id, linkForm.member_id, linkForm.relate_id, linkForm.relation);
      toast("Membro vinculado à família.");
      setLinkForm({ member_id: "", relate_id: "", relation: "relative" });
      setFamilies(await listFamilies().then((r) => r.families));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="mx-auto max-w-6xl">
      <PageHeader
        title="Famílias"
        description="Núcleos familiares e vínculos"
        actions={<Button onClick={() => setOpen(true)}><Plus className="h-4 w-4" /> Nova Família</Button>}
      />

      <div className="mb-4 relative max-w-sm">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
        <Input className="pl-9" placeholder="Buscar família" value={query} onChange={(e) => setQuery(e.target.value)} />
      </div>

      <Card className="overflow-hidden p-0">
        {families === null ? (
          <SkeletonRows />
        ) : filtered.length === 0 ? (
          <EmptyState icon={<Users2 className="h-10 w-10" />} title="Nenhuma família" description="Crie núcleos familiares e vincule membros." />
        ) : (
          <Table>
            <THead><TRow><TH>Família</TH><TH>Membros</TH><TH>Responsável</TH><TH className="text-right">Ações</TH></TRow></THead>
            <TBody>
              {filtered.map((f) => (
                <TRow key={f.id}>
                  <TD className="font-medium">{f.name}</TD>
                  <TD><Badge tone="violet"><Users className="h-3 w-3" /> {f.member_count}</Badge></TD>
                  <TD className="text-zinc-500">{members.find((m) => m.id === f.head_id)?.full_name ?? "—"}</TD>
                  <TD className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => setLink(f)}>Vincular membro</Button>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Modal open={open} onClose={() => setOpen(false)} title="Nova Família">
        <form onSubmit={add} className="space-y-3">
          <Field label="Nome da família *"><Input required value={newName} onChange={(e) => setNewName(e.target.value)} /></Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit">Criar</Button>
          </div>
        </form>
      </Modal>

      <Modal open={!!link} onClose={() => setLink(null)} title={`Vincular a ${link?.name ?? ""}`}>
        <form onSubmit={doLink} className="space-y-3">
          <Field label="Membro">
            <Select value={linkForm.member_id} onChange={(e) => setLinkForm({ ...linkForm, member_id: e.target.value })}>
              <option value="">Selecione...</option>
              {members.map((m) => (<option key={m.id} value={m.id}>{m.full_name}</option>))}
            </Select>
          </Field>
          <Field label="Vínculo com (responsável, opcional)">
            <Select value={linkForm.relate_id} onChange={(e) => setLinkForm({ ...linkForm, relate_id: e.target.value })}>
              <option value="">—</option>
              {members.filter((m) => m.id !== linkForm.member_id).map((m) => (<option key={m.id} value={m.id}>{m.full_name}</option>))}
            </Select>
          </Field>
          <Field label="Relação">
            <Select value={linkForm.relation} onChange={(e) => setLinkForm({ ...linkForm, relation: e.target.value })}>
              {Object.entries(RELATION_LABELS).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
            </Select>
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setLink(null)}>Cancelar</Button>
            <Button type="submit">Vincular</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
