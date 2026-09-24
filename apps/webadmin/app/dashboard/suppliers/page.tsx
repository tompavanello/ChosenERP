"use client";

import { useEffect, useState } from "react";
import { Plus, Building2, Search, Pencil, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Textarea } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { StatCard } from "@/components/ui/stat-card";
import { Drawer } from "@/components/ui/modal";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/toast";
import { listSuppliers, createSupplier, updateSupplier, deleteSupplier, type Supplier } from "@/lib/api";

const EMPTY = { name: "", trade_name: "", cpf: "", cnpj: "", email: "", phone: "", notes: "" };

export default function SuppliersPage() {
  const { toast } = useToast();
  const [items, setItems] = useState<Supplier[] | null>(null);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [form, setForm] = useState({ ...EMPTY });

  async function load(q = "") {
    setItems(await listSuppliers(q).then((r) => r.suppliers));
  }

  useEffect(() => {
    load().catch((e) => toast(e.message, "error"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function openNew() {
    setEditId(null);
    setForm({ ...EMPTY });
    setOpen(true);
  }

  function openEdit(s: Supplier) {
    setEditId(s.id);
    setForm({
      name: s.name,
      trade_name: s.trade_name ?? "",
      cpf: s.cpf ?? "",
      cnpj: s.cnpj ?? "",
      email: s.email ?? "",
      phone: s.phone ?? "",
      notes: s.notes ?? "",
    });
    setOpen(true);
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    const data = Object.fromEntries(Object.entries(form).filter(([, v]) => v !== ""));
    try {
      if (editId) {
        await updateSupplier(editId, data);
        toast("Fornecedor atualizado.");
      } else {
        await createSupplier(data);
        toast("Fornecedor cadastrado.");
      }
      setOpen(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function remove(s: Supplier) {
    if (!confirm(`Excluir "${s.name}"?`)) return;
    try {
      await deleteSupplier(s.id);
      toast("Fornecedor excluido.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  const filtered = (items ?? []).filter((s) => {
    const q = query.trim().toLowerCase();
    return (
      !q ||
      s.name.toLowerCase().includes(q) ||
      (s.trade_name?.toLowerCase().includes(q) ?? false) ||
      (s.cpf?.includes(q) ?? false) ||
      (s.cnpj?.includes(q) ?? false)
    );
  });

  return (
    <div className="page">
      <PageHeader
        title="Fornecedores"
        description="Pessoas fisicas ou juridicas - CPF/CNPJ opcionais"
        actions={<Button onClick={openNew}><Plus className="h-4 w-4" /> Novo Fornecedor</Button>}
      />

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <StatCard label="Fornecedores" value={items ? String(items.length) : "..."} icon={Building2} />
        <StatCard label="Com CNPJ" value={items ? String(items.filter((s) => s.cnpj).length) : "..."} tone="sky" />
        <StatCard label="Com CPF" value={items ? String(items.filter((s) => s.cpf).length) : "..."} tone="green" />
      </div>

      <div className="mb-4 relative max-w-sm">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
        <Input
          className="pl-9"
          placeholder="Buscar por nome, CPF ou CNPJ"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      <Card className="overflow-hidden p-0">
        {items === null ? (
          <SkeletonRows />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={<Building2 className="h-10 w-10" />}
            title="Nenhum fornecedor"
            description="Cadastre fornecedores para vincular a despesas."
          />
        ) : (
          <Table>
            <THead>
              <TRow><TH>Nome</TH><TH>CPF/CNPJ</TH><TH>Contato</TH><TH>Situacao</TH><TH></TH></TRow>
            </THead>
            <TBody>
              {filtered.map((s) => (
                <TRow key={s.id}>
                  <TD>
                    <p className="font-medium">{s.name}</p>
                    {s.trade_name && <p className="text-xs text-zinc-400">{s.trade_name}</p>}
                  </TD>
                  <TD className="tabular-nums">{s.cnpj || s.cpf || "-"}</TD>
                  <TD>
                    <p>{s.email ?? "-"}</p>
                    <p className="text-xs text-zinc-400">{s.phone ?? ""}</p>
                  </TD>
                  <TD>
                    <Badge tone={s.is_active ? "green" : "zinc"}>{s.is_active ? "Ativo" : "Inativo"}</Badge>
                  </TD>
                  <TD>
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" className="h-7 w-7 p-0" onClick={() => openEdit(s)} title="Editar">
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" className="h-7 w-7 p-0" onClick={() => remove(s)} title="Excluir">
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} title={editId ? "Editar Fornecedor" : "Novo Fornecedor"}>
        <form onSubmit={save} className="grid grid-cols-2 gap-3">
          <Field label="Nome / Razao social *" className="col-span-2">
            <Input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </Field>
          <Field label="Nome fantasia" className="col-span-2">
            <Input value={form.trade_name} onChange={(e) => setForm({ ...form, trade_name: e.target.value })} />
          </Field>
          <Field label="CPF">
            <Input value={form.cpf} onChange={(e) => setForm({ ...form, cpf: e.target.value })} placeholder="000.000.000-00" />
          </Field>
          <Field label="CNPJ">
            <Input value={form.cnpj} onChange={(e) => setForm({ ...form, cnpj: e.target.value })} placeholder="00.000.000/0000-00" />
          </Field>
          <Field label="Telefone">
            <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          </Field>
          <Field label="E-mail">
            <Input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
          </Field>
          <Field label="Observacoes" className="col-span-2">
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
