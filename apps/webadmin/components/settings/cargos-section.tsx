"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Award, Pencil, Plus, Trash2 } from "lucide-react";
import { Badge, type Tone } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  listCargos, createCargo, updateCargo, deleteCargo, type Cargo,
} from "@/lib/api";
import { CARGO_KINDS } from "@/lib/constants";

type Form = {
  name: string;
  kind: string;
  requires_term: boolean;
  is_active: boolean;
  sort_order: string;
};

const EMPTY: Form = {
  name: "", kind: "eclesiastico", requires_term: false, is_active: true, sort_order: "0",
};

/**
 * CargosSection e o catalogo de cargos/funcoes da igreja, em tabela.
 *
 * O catalogo vive na tabela `cargos` (migracao 000020) e vale para todo o tenant -
 * nao e uma lista fixa no codigo. A atribuicao a um membro (com mandato) e feita
 * na aba "Cargos" do membro; aqui a igreja cria, agrupa, ordena, desativa e
 * exclui os cargos.
 */
export function CargosSection({ canWrite }: { canWrite: boolean }) {
  const { toast } = useToast();
  const [items, setItems] = useState<Cargo[] | null>(null);
  const [drawer, setDrawer] = useState<{ open: boolean; editing?: Cargo }>({ open: false });
  const [form, setForm] = useState<Form>({ ...EMPTY });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await listCargos();
      setItems(r.cargos);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar cargos", "error");
      setItems([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  const sorted = useMemo(() => {
    const arr = [...(items ?? [])];
    arr.sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
    return arr;
  }, [items]);

  function openCreate() {
    setForm({ ...EMPTY });
    setDrawer({ open: true });
  }

  function openEdit(c: Cargo) {
    setForm({
      name: c.name, kind: c.kind, requires_term: c.requires_term,
      is_active: c.is_active, sort_order: String(c.sort_order),
    });
    setDrawer({ open: true, editing: c });
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) return;
    setSaving(true);
    const payload = {
      name: form.name.trim(),
      kind: form.kind,
      requires_term: form.requires_term,
      is_active: form.is_active,
      sort_order: Number(form.sort_order) || 0,
    };
    try {
      if (drawer.editing) {
        await updateCargo(drawer.editing.id, payload);
        toast("Cargo atualizado.");
      } else {
        await createCargo(payload);
        toast("Cargo criado.");
      }
      setDrawer({ open: false });
      await load();
    } catch (err) {
      // 409 do backend: nome/slug repetido no tenant.
      toast(err instanceof Error ? err.message : "Erro ao salvar cargo", "error");
    } finally {
      setSaving(false);
    }
  }

  async function remove(c: Cargo) {
    if (!confirm(`Excluir o cargo "${c.name}"?`)) return;
    try {
      await deleteCargo(c.id);
      toast("Cargo excluido.");
      await load();
    } catch (err) {
      // 409: existe mandato (mesmo encerrado) apontando para o cargo.
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-xs text-zinc-400">
          Cada igreja define os seus cargos/funcoes e o grupo de cada um. A atribuicao a um
          membro (com mandato) e feita na aba <span className="font-medium">Cargos</span> do membro.
        </p>
        {canWrite && (
          <Button className="h-8 text-sm" onClick={openCreate}><Plus className="h-4 w-4" /> Novo cargo</Button>
        )}
      </div>

      <Card className="overflow-hidden p-0">
        {items === null ? (
          <div className="p-4"><SkeletonRows rows={5} /></div>
        ) : sorted.length === 0 ? (
          <EmptyState
            icon={<Award className="h-10 w-10" />}
            title="Nenhum cargo cadastrado"
            description="Crie os cargos e funcoes da igreja (Pastor, Diacono, Lider de Louvor...)."
          />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Cargo</TH><TH>Grupo</TH><TH>Mandato</TH>
                <TH>Situacao</TH><TH className="text-right">Acoes</TH>
              </TRow>
            </THead>
            <TBody>
              {sorted.map((c) => {
                const kind = CARGO_KINDS[c.kind] ?? CARGO_KINDS.outro;
                return (
                  <TRow key={c.id}>
                    <TD>
                      <div className="flex items-center gap-2">
                        <Badge tone={kind.tone as Tone} className="text-[10px]">{c.name}</Badge>
                        <span className="text-[11px] text-zinc-400">{c.slug}</span>
                      </div>
                    </TD>
                    <TD className="text-sm">{kind.label}</TD>
                    <TD>
                      {c.requires_term
                        ? <Badge tone="zinc" variant="outline" className="text-[10px]">Exige vencimento</Badge>
                        : <span className="text-xs text-zinc-400">-</span>}
                    </TD>
                    <TD>
                      <Badge tone={c.is_active ? "green" : "zinc"} className="text-[10px]">
                        {c.is_active ? "Ativo" : "Inativo"}
                      </Badge>
                    </TD>
                    <TD className="text-right">
                      {canWrite && (
                        <div className="flex justify-end gap-1">
                          <Button variant="ghost" className="h-7 px-2" onClick={() => openEdit(c)}>
                            <Pencil className="h-3.5 w-3.5" />
                          </Button>
                          <Button variant="ghost" className="h-7 px-2 text-red-600" onClick={() => remove(c)}>
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      )}
                    </TD>
                  </TRow>
                );
              })}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer
        open={drawer.open}
        onClose={() => setDrawer({ open: false })}
        title={drawer.editing ? "Editar cargo" : "Novo cargo"}
      >
        <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Nome" required className="sm:col-span-2">
            <Input
              required
              className="h-8 text-sm"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Ex: Lider de Adolescentes"
            />
          </Field>
          <Field label="Grupo">
            <Select className="h-8 text-sm" value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}>
              {Object.entries(CARGO_KINDS).map(([k, v]) => (<option key={k} value={k}>{v.label}</option>))}
            </Select>
          </Field>
          <Field label="Ordem">
            <Input type="number" className="h-8 text-sm" value={form.sort_order} onChange={(e) => setForm({ ...form, sort_order: e.target.value })} />
          </Field>

          <label className="flex items-center gap-2 text-sm sm:col-span-2">
            <input type="checkbox" checked={form.requires_term} onChange={(e) => setForm({ ...form, requires_term: e.target.checked })} />
            Exige mandato com data de vencimento
          </label>
          <label className="flex items-center gap-2 text-sm sm:col-span-2">
            <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
            Cargo ativo (disponivel para atribuicao)
          </label>
          <p className="text-xs text-zinc-400 sm:col-span-2">
            Desative um cargo que saiu de uso em vez de excluir: o historico de quem o exerceu e
            preservado. A exclusao so e permitida quando nunca houve mandato.
          </p>

          <div className="flex justify-end gap-2 sm:col-span-2">
            <Button type="button" variant="outline" className="h-8 text-sm" onClick={() => setDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
