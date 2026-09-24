"use client";

import { useCallback, useEffect, useState } from "react";
import { Plus, Pencil, Trash2, Power, BadgeCheck, Check, X } from "lucide-react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Field, Input, Select } from "@/components/ui/input";
import { Badge, type Tone } from "@/components/ui/badge";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  listCargos, createCargo, updateCargo, deleteCargo, type Cargo,
} from "@/lib/api";
import { CARGO_KINDS } from "@/lib/constants";
import { cn } from "@/lib/utils";

/** Formulario em branco do catalogo. */
type CargoDraft = {
  name: string;
  kind: string;
  requires_term: boolean;
  sort_order: string;
};

const EMPTY_DRAFT: CargoDraft = { name: "", kind: "eclesiastico", requires_term: false, sort_order: "0" };

/**
 * CargosManagerDialog e o "lugar para criar os cargos" pedido pelo cliente: o
 * catalogo e da igreja, nao uma lista fixa no codigo. Aqui ela cria, renomeia,
 * agrupa, ordena, desativa e - so quando nunca houve mandato - exclui.
 */
export function CargosManagerDialog({
  open,
  onClose,
  onChanged,
}: {
  open: boolean;
  onClose: () => void;
  /** Avisa o pai para recarregar as opcoes (o seletor de cargos usa a mesma lista). */
  onChanged?: () => void;
}) {
  const { toast } = useToast();
  const [cargos, setCargos] = useState<Cargo[] | null>(null);
  const [draft, setDraft] = useState<CargoDraft>(EMPTY_DRAFT);
  const [criando, setCriando] = useState(false);
  const [editando, setEditando] = useState<string | null>(null);
  const [editDraft, setEditDraft] = useState<CargoDraft>(EMPTY_DRAFT);
  const [salvando, setSalvando] = useState(false);

  const load = useCallback(async () => {
    try {
      const { cargos: lista } = await listCargos();
      setCargos(lista);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar cargos", "error");
      setCargos([]);
    }
  }, [toast]);

  useEffect(() => {
    if (open) {
      setDraft(EMPTY_DRAFT);
      setEditando(null);
      load();
    }
  }, [open, load]);

  function reload() {
    load();
    onChanged?.();
  }

  async function criar(e: React.FormEvent) {
    e.preventDefault();
    if (!draft.name.trim()) return;
    setSalvando(true);
    try {
      await createCargo({
        name: draft.name.trim(),
        kind: draft.kind,
        requires_term: draft.requires_term,
        sort_order: Number(draft.sort_order) || 0,
      });
      setDraft(EMPTY_DRAFT);
      setCriando(false);
      toast("Cargo criado.");
      reload();
    } catch (err) {
      // O backend recusa nome/slug repetido no mesmo tenant (indice unico).
      toast(err instanceof Error ? err.message : "Erro ao criar cargo", "error");
    } finally {
      setSalvando(false);
    }
  }

  async function salvarEdicao(cargo: Cargo) {
    setSalvando(true);
    try {
      await updateCargo(cargo.id, {
        name: editDraft.name.trim(),
        kind: editDraft.kind,
        requires_term: editDraft.requires_term,
        sort_order: Number(editDraft.sort_order) || 0,
      });
      setEditando(null);
      toast("Cargo atualizado.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSalvando(false);
    }
  }

  async function alternarAtivo(cargo: Cargo) {
    try {
      await updateCargo(cargo.id, { is_active: !cargo.is_active });
      toast(cargo.is_active ? "Cargo desativado." : "Cargo reativado.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao alterar situacao", "error");
    }
  }

  async function excluir(cargo: Cargo) {
    try {
      await deleteCargo(cargo.id);
      toast("Cargo excluido.");
      reload();
    } catch (err) {
      // 409 do backend: existe mandato (mesmo encerrado) apontando para o cargo.
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Cargos da igreja">
      <p className="-mt-2 mb-4 text-xs text-zinc-500">
        O catalogo vale para toda a igreja. Desative um cargo que saiu de uso em vez de
        exclui-lo: o historico de quem o exerceu e preservado.
      </p>

      {criando ? (
        <form onSubmit={criar} className="mb-4 rounded-lg border border-sky-200 bg-sky-50/50 p-3">
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-4">
            <Field label="Nome *" className="sm:col-span-2">
              <Input
                autoFocus
                required
                className="h-8 text-sm"
                placeholder="Ex: Lider de Adolescentes"
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              />
            </Field>
            <Field label="Grupo">
              <Select className="h-8 text-sm" value={draft.kind} onChange={(e) => setDraft({ ...draft, kind: e.target.value })}>
                {Object.entries(CARGO_KINDS).map(([k, v]) => (
                  <option key={k} value={k}>{v.label}</option>
                ))}
              </Select>
            </Field>
            <Field label="Ordem">
              <Input
                type="number"
                className="h-8 text-sm"
                value={draft.sort_order}
                onChange={(e) => setDraft({ ...draft, sort_order: e.target.value })}
              />
            </Field>
          </div>
          <label className="mt-2 flex items-center gap-2 text-xs text-zinc-600">
            <input
              type="checkbox"
              checked={draft.requires_term}
              onChange={(e) => setDraft({ ...draft, requires_term: e.target.checked })}
              className="rounded border-zinc-300 text-sky-600 focus:ring-sky-500"
            />
            Exige mandato com data de vencimento
          </label>
          <div className="mt-3 flex justify-end gap-2">
            <Button variant="ghost" size="sm" type="button" onClick={() => setCriando(false)}>Cancelar</Button>
            <Button size="sm" type="submit" disabled={salvando || !draft.name.trim()}>
              {salvando ? "Criando..." : "Criar cargo"}
            </Button>
          </div>
        </form>
      ) : (
        <Button size="sm" className="mb-4" onClick={() => setCriando(true)}>
          <Plus className="h-4 w-4" /> Novo cargo
        </Button>
      )}

      {cargos === null ? (
        <SkeletonRows rows={4} />
      ) : cargos.length === 0 ? (
        <EmptyState
          icon={<BadgeCheck className="h-10 w-10" />}
          title="Nenhum cargo cadastrado"
          description="Crie os cargos e funcoes que a igreja usa (Pastor, Diacono, Lider de Louvor...)."
        />
      ) : (
        <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
          {cargos.map((c) => {
            const kind = CARGO_KINDS[c.kind] ?? CARGO_KINDS.outro;
            const emEdicao = editando === c.id;
            return (
              <li key={c.id} className={cn("flex flex-wrap items-center gap-2 py-2", !c.is_active && "opacity-60")}>
                {emEdicao ? (
                  <>
                    <Input
                      className="h-8 flex-1 text-sm"
                      value={editDraft.name}
                      onChange={(e) => setEditDraft({ ...editDraft, name: e.target.value })}
                    />
                    <Select
                      className="h-8 w-36 text-sm"
                      value={editDraft.kind}
                      onChange={(e) => setEditDraft({ ...editDraft, kind: e.target.value })}
                    >
                      {Object.entries(CARGO_KINDS).map(([k, v]) => (
                        <option key={k} value={k}>{v.label}</option>
                      ))}
                    </Select>
                    <Input
                      type="number"
                      title="Ordem de exibicao"
                      className="h-8 w-20 text-sm"
                      value={editDraft.sort_order}
                      onChange={(e) => setEditDraft({ ...editDraft, sort_order: e.target.value })}
                    />
                    <label className="flex items-center gap-1.5 text-xs text-zinc-600">
                      <input
                        type="checkbox"
                        checked={editDraft.requires_term}
                        onChange={(e) => setEditDraft({ ...editDraft, requires_term: e.target.checked })}
                        className="rounded border-zinc-300 text-sky-600 focus:ring-sky-500"
                      />
                      Exige mandato
                    </label>
                    <div className="ml-auto flex gap-1">
                      <Button variant="ghost" size="sm" onClick={() => salvarEdicao(c)} disabled={salvando} title="Salvar">
                        <Check className="h-4 w-4 text-emerald-600" />
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => setEditando(null)} title="Cancelar">
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  </>
                ) : (
                  <>
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">{c.name}</span>
                    <Badge tone={kind.tone as Tone} className="text-[10px]">{kind.label}</Badge>
                    {c.requires_term && (
                      <Badge tone="zinc" variant="outline" className="text-[10px]">Mandato</Badge>
                    )}
                    {!c.is_active && <Badge tone="zinc" className="text-[10px]">Inativo</Badge>}
                    <div className="ml-auto flex gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        title="Editar"
                        onClick={() => {
                          setEditando(c.id);
                          setEditDraft({
                            name: c.name,
                            kind: c.kind,
                            requires_term: c.requires_term,
                            sort_order: String(c.sort_order),
                          });
                        }}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        title={c.is_active ? "Desativar" : "Reativar"}
                        onClick={() => alternarAtivo(c)}
                      >
                        <Power className={cn("h-4 w-4", c.is_active ? "text-emerald-600" : "text-zinc-400")} />
                      </Button>
                      <Button variant="ghost" size="sm" title="Excluir" onClick={() => excluir(c)}>
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  </>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </Modal>
  );
}
