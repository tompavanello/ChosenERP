"use client";

import { useEffect, useState } from "react";
import { Plus, HeartHandshake, Search } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Textarea } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { StatCard } from "@/components/ui/stat-card";
import { Drawer } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { listBenefactors, createBenefactor, type Benefactor } from "@/lib/api";
import { datePt } from "@/lib/format";

export default function BenefactorsPage() {
  const { toast } = useToast();
  const [items, setItems] = useState<Benefactor[] | null>(null);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ name: "", cpf: "", email: "", phone: "", notes: "" });

  useEffect(() => {
    listBenefactors().then((r) => setItems(r.benefactors)).catch((e) => toast(e.message, "error"));
  }, [toast]);

  const filtered = (items ?? []).filter((b) => {
    const q = query.trim().toLowerCase();
    return !q || b.name.toLowerCase().includes(q) || (b.email?.toLowerCase().includes(q) ?? false);
  });

  async function add(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createBenefactor(Object.fromEntries(Object.entries(form).filter(([, v]) => v !== "")));
      toast("Benfeitor cadastrado.");
      setOpen(false);
      setItems(await listBenefactors().then((r) => r.benefactors));
      setForm({ name: "", cpf: "", email: "", phone: "", notes: "" });
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Benfeitores"
        description="Apoiadores sem vínculo de membresia"
        actions={<Button onClick={() => setOpen(true)}><Plus className="h-4 w-4" /> Novo Benfeitor</Button>}
      />

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <StatCard label="Benfeitores" value={items ? String(items.length) : "…"} icon={HeartHandshake} />
        <StatCard label="Com e-mail" value={items ? String(items.filter((b) => b.email).length) : "…"} tone="green" />
        <StatCard label="Com telefone" value={items ? String(items.filter((b) => b.phone).length) : "…"} tone="sky" />
      </div>

      <div className="mb-4 relative max-w-sm">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
        <Input className="pl-9" placeholder="Buscar por nome ou e-mail" value={query} onChange={(e) => setQuery(e.target.value)} />
      </div>

      <Card className="overflow-hidden p-0">
        {items === null ? (
          <SkeletonRows />
        ) : filtered.length === 0 ? (
          <EmptyState icon={<HeartHandshake className="h-10 w-10" />} title="Nenhum benfeitor" description="Cadastre apoiadores e registre contato." />
        ) : (
          <Table>
            <THead>
              <TRow><TH>Nome</TH><TH>Contato</TH><TH>Observações</TH><TH>Cadastrado</TH></TRow>
            </THead>
            <TBody>
              {filtered.map((b) => (
                <TRow key={b.id}>
                  <TD className="font-medium">{b.name}</TD>
                  <TD>
                    <p>{b.email ?? "—"}</p>
                    <p className="text-xs text-zinc-400">{b.phone ?? ""}</p>
                  </TD>
                  <TD className="text-zinc-500">{b.notes ?? "—"}</TD>
                  <TD className="text-zinc-500">{datePt(b.created_at)}</TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} title="Novo Benfeitor">
        <form onSubmit={add} className="grid grid-cols-2 gap-3">
          <Field label="Nome *" className="col-span-2"><Input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Field>
          <Field label="CPF"><Input value={form.cpf} onChange={(e) => setForm({ ...form, cpf: e.target.value })} placeholder="000.000.000-00" /></Field>
          <Field label="Telefone"><Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></Field>
          <Field label="E-mail" className="col-span-2"><Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></Field>
          <Field label="Observações" className="col-span-2">
            <Textarea rows={3} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
          </Field>
          <div className="col-span-2 flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit">Salvar</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
