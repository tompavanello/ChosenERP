"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Award, Check, Pencil, Plus, Power, Settings2, Trash2, TriangleAlert, X } from "lucide-react";
import { Badge, type Tone } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Combobox } from "@/components/ui/combobox";
import { Field, Input, Select } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { CargosManagerDialog } from "@/components/people/cargos-manager";
import {
  listCargos, listMemberCargos, assignCargo, updateMemberCargo, unassignCargo,
  type Cargo, type MemberCargo,
} from "@/lib/api";
import { CARGO_EXPIRY_WINDOW_DAYS, CARGO_KINDS, CARGO_STATUS } from "@/lib/constants";
import { datePt } from "@/lib/format";
import { cn } from "@/lib/utils";

const hojeISO = () => new Date().toISOString().slice(0, 10);

/**
 * Situacao do mandato frente ao vencimento.
 * Retorna null quando nao ha vencimento (cargo sem mandato) ou quando ele esta
 * longe o bastante para nao valer um alerta na tela.
 */
function situacaoVencimento(mc: MemberCargo): { dias: number; texto: string } | null {
  if (mc.status !== "ativo" || !mc.ends_at) return null;
  const fim = new Date(`${mc.ends_at}T00:00:00`);
  if (Number.isNaN(fim.getTime())) return null;
  const hoje = new Date();
  hoje.setHours(0, 0, 0, 0);
  const dias = Math.round((fim.getTime() - hoje.getTime()) / 86_400_000);
  if (dias < 0) return { dias, texto: `Mandato vencido ha ${Math.abs(dias)} dia(s)` };
  if (dias <= CARGO_EXPIRY_WINDOW_DAYS) {
    return { dias, texto: dias === 0 ? "Vence hoje" : `Vence em ${dias} dia(s)` };
  }
  return null;
}

type Rascunho = {
  cargo_id: string;
  started_at: string;
  ends_at: string;
  notes: string;
  /** So usado na edicao de um mandato existente. */
  status: string;
};

const RASCUNHO_VAZIO: Rascunho = {
  cargo_id: "", started_at: hojeISO(), ends_at: "", notes: "", status: "ativo",
};

/**
 * CargosSection e a aba de cargos do membro: o requisito 1.3 (mais de uma funcao
 * por membro) e o 1.4 (mandato com inicio, vencimento e situacao) juntos.
 */
export function CargosSection({
  memberId,
  canWrite,
  onChanged,
}: {
  memberId: string;
  canWrite: boolean;
  onChanged?: () => void;
}) {
  const { toast } = useToast();
  const [vinculos, setVinculos] = useState<MemberCargo[] | null>(null);
  const [catalogo, setCatalogo] = useState<Cargo[]>([]);
  const [rascunho, setRascunho] = useState<Rascunho>(RASCUNHO_VAZIO);
  const [criando, setCriando] = useState(false);
  const [editando, setEditando] = useState<string | null>(null);
  const [editRascunho, setEditRascunho] = useState<Rascunho>(RASCUNHO_VAZIO);
  const [gerenciar, setGerenciar] = useState(false);
  const [salvando, setSalvando] = useState(false);

  const load = useCallback(async () => {
    try {
      const [v, c] = await Promise.all([listMemberCargos(memberId), listCargos()]);
      setVinculos(v.cargos);
      setCatalogo(c.cargos);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar cargos", "error");
      setVinculos([]);
    }
  }, [memberId, toast]);

  useEffect(() => {
    setVinculos(null);
    load();
  }, [load]);

  function reload() {
    load();
    onChanged?.();
  }

  // So cargos ativos entram como opcao de novo mandato; os ja vigentes para este
  // membro saem da lista para nao sugerir duplicata (o backend responderia 409).
  const disponiveis = useMemo(() => {
    const ativos = new Set((vinculos ?? []).filter((v) => v.status === "ativo").map((v) => v.cargo_id));
    return catalogo.filter((c) => c.is_active && !ativos.has(c.id));
  }, [catalogo, vinculos]);

  async function adicionar(e: React.FormEvent) {
    e.preventDefault();
    if (!rascunho.cargo_id) return;
    setSalvando(true);
    try {
      await assignCargo(memberId, {
        cargo_id: rascunho.cargo_id,
        started_at: rascunho.started_at,
        ends_at: rascunho.ends_at,
        status: "ativo",
        notes: rascunho.notes,
      });
      setRascunho({ ...RASCUNHO_VAZIO, started_at: hojeISO() });
      setCriando(false);
      toast("Cargo atribuido.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao atribuir cargo", "error");
    } finally {
      setSalvando(false);
    }
  }

  async function salvarEdicao(mc: MemberCargo) {
    setSalvando(true);
    try {
      // String vazia LIMPA a data (o backend trata '' como NULL); data ausente
      // manteria o valor antigo, entao mandamos sempre os dois campos.
      await updateMemberCargo(memberId, mc.id, {
        started_at: editRascunho.started_at,
        ends_at: editRascunho.ends_at,
        status: editRascunho.status || mc.status,
        notes: editRascunho.notes,
      });
      setEditando(null);
      toast("Mandato atualizado.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar mandato", "error");
    } finally {
      setSalvando(false);
    }
  }

  async function encerrar(mc: MemberCargo) {
    try {
      await updateMemberCargo(memberId, mc.id, { status: "encerrado", ends_at: hojeISO() });
      toast("Mandato encerrado.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao encerrar", "error");
    }
  }

  async function reabrir(mc: MemberCargo) {
    try {
      await updateMemberCargo(memberId, mc.id, { status: "ativo", ends_at: "" });
      toast("Mandato reaberto.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao reabrir", "error");
    }
  }

  async function remover(mc: MemberCargo) {
    try {
      await unassignCargo(memberId, mc.id);
      toast("Mandato removido do historico.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao remover", "error");
    }
  }

  if (vinculos === null) return <Card><SkeletonRows rows={3} /></Card>;

  const ativos = vinculos.filter((v) => v.status === "ativo");
  const encerrados = vinculos.filter((v) => v.status !== "ativo");

  return (
    <div className="space-y-4">
      <Card>
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="flex items-center gap-2 text-sm font-semibold">
              <Award className="h-4 w-4 text-sky-600" /> Cargos e mandatos
            </h3>
            <p className="mt-0.5 text-xs text-zinc-400">
              Um membro pode exercer mais de um cargo, cada um com o seu mandato.
            </p>
          </div>
          {canWrite && (
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => setGerenciar(true)}>
                <Settings2 className="h-4 w-4" /> Gerenciar cargos
              </Button>
              <Button size="sm" onClick={() => setCriando((v) => !v)} disabled={disponiveis.length === 0}>
                <Plus className="h-4 w-4" /> Atribuir cargo
              </Button>
            </div>
          )}
        </div>

        {criando && (
          <form onSubmit={adicionar} className="mb-4 rounded-lg border border-sky-200 bg-sky-50/50 p-3 dark:border-sky-900 dark:bg-sky-950/30">
            <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-4">
              <Field label="Cargo *" className="sm:col-span-2">
                <Combobox
                  value={rascunho.cargo_id}
                  placeholder="Selecione o cargo..."
                  searchPlaceholder="Buscar cargo..."
                  options={disponiveis.map((c) => ({
                    value: c.id,
                    label: `${c.name} - ${(CARGO_KINDS[c.kind] ?? CARGO_KINDS.outro).label}`,
                  }))}
                  onChange={(v) => setRascunho({ ...rascunho, cargo_id: v })}
                />
              </Field>
              <Field label="Inicio">
                <Input
                  type="date"
                  className="h-8 text-sm"
                  value={rascunho.started_at}
                  onChange={(e) => setRascunho({ ...rascunho, started_at: e.target.value })}
                />
              </Field>
              <Field label="Vencimento">
                <Input
                  type="date"
                  className="h-8 text-sm"
                  value={rascunho.ends_at}
                  onChange={(e) => setRascunho({ ...rascunho, ends_at: e.target.value })}
                />
              </Field>
            </div>
            <Field label="Observacao" className="mt-2.5">
              <Input
                className="h-8 text-sm"
                placeholder="Ex: eleito em assembleia de 12/2025"
                value={rascunho.notes}
                onChange={(e) => setRascunho({ ...rascunho, notes: e.target.value })}
              />
            </Field>
            <div className="mt-3 flex justify-end gap-2">
              <Button variant="ghost" size="sm" type="button" onClick={() => setCriando(false)}>Cancelar</Button>
              <Button size="sm" type="submit" disabled={salvando || !rascunho.cargo_id}>
                {salvando ? "Atribuindo..." : "Atribuir"}
              </Button>
            </div>
          </form>
        )}

        {vinculos.length === 0 ? (
          <EmptyState
            icon={<Award className="h-10 w-10" />}
            title="Nenhum cargo atribuido"
            description="Atribua os cargos que este membro exerce na igreja."
          />
        ) : (
          <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {[...ativos, ...encerrados].map((mc) => {
              const kind = CARGO_KINDS[mc.cargo_kind] ?? CARGO_KINDS.outro;
              const st = CARGO_STATUS[mc.status] ?? CARGO_STATUS.encerrado;
              const alerta = situacaoVencimento(mc);
              const emEdicao = editando === mc.id;
              return (
                <li key={mc.id} className={cn("py-2.5", mc.status !== "ativo" && "opacity-70")}>
                  {emEdicao ? (
                    <div className="space-y-2">
                      <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-4">
                        <Field label="Inicio">
                          <Input
                            type="date"
                            className="h-8 text-sm"
                            value={editRascunho.started_at}
                            onChange={(e) => setEditRascunho({ ...editRascunho, started_at: e.target.value })}
                          />
                        </Field>
                        <Field label="Vencimento">
                          <Input
                            type="date"
                            className="h-8 text-sm"
                            value={editRascunho.ends_at}
                            onChange={(e) => setEditRascunho({ ...editRascunho, ends_at: e.target.value })}
                          />
                        </Field>
                        <Field label="Situacao">
                          <Select
                            className="h-8 text-sm"
                            value={editRascunho.status}
                            onChange={(e) => setEditRascunho({ ...editRascunho, status: e.target.value })}
                          >
                            <option value="ativo">Ativo</option>
                            <option value="encerrado">Encerrado</option>
                          </Select>
                        </Field>
                        <Field label="Observacao">
                          <Input
                            className="h-8 text-sm"
                            value={editRascunho.notes}
                            onChange={(e) => setEditRascunho({ ...editRascunho, notes: e.target.value })}
                          />
                        </Field>
                      </div>
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="sm" disabled={salvando} onClick={() => salvarEdicao(mc)}>
                          <Check className="h-4 w-4 text-emerald-600" /> Salvar
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => setEditando(null)}>
                          <X className="h-4 w-4" /> Cancelar
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium">{mc.cargo_name}</span>
                      <Badge tone={kind.tone as Tone} className="text-[10px]">{kind.label}</Badge>
                      <Badge tone={st.tone as Tone} className="text-[10px]">{st.label}</Badge>
                      <span className="tnum text-xs text-zinc-500">
                        {mc.started_at ? datePt(mc.started_at) : "sem inicio definido"}
                        {" -> "}
                        {mc.ends_at ? datePt(mc.ends_at) : "sem vencimento"}
                      </span>
                      {alerta && (
                        <span className={cn("inline-flex items-center gap-1 text-xs", alerta.dias < 0 ? "text-red-600" : "text-amber-600")}>
                          <TriangleAlert className="h-3.5 w-3.5" /> {alerta.texto}
                        </span>
                      )}
                      {mc.notes && <span className="text-xs text-zinc-400">- {mc.notes}</span>}

                      {canWrite && (
                        <div className="ml-auto flex gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            title="Editar mandato"
                            onClick={() => {
                              setEditando(mc.id);
                              setEditRascunho({
                                cargo_id: mc.cargo_id,
                                started_at: mc.started_at ?? "",
                                ends_at: mc.ends_at ?? "",
                                notes: mc.notes ?? "",
                                status: mc.status,
                              });
                            }}
                          >
                            <Pencil className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            title={mc.status === "ativo" ? "Encerrar mandato" : "Reabrir mandato"}
                            onClick={() => (mc.status === "ativo" ? encerrar(mc) : reabrir(mc))}
                          >
                            <Power className={cn("h-3.5 w-3.5", mc.status === "ativo" ? "text-emerald-600" : "text-zinc-400")} />
                          </Button>
                          <Button variant="ghost" size="sm" title="Remover do historico" onClick={() => remover(mc)}>
                            <Trash2 className="h-3.5 w-3.5 text-red-500" />
                          </Button>
                        </div>
                      )}
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </Card>

      <CargosManagerDialog
        open={gerenciar}
        onClose={() => setGerenciar(false)}
        onChanged={load}
      />
    </div>
  );
}
