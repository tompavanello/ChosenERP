"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import {
  Crown, Home, MapPin, Pencil, Plus, Trash2, UserMinus, UserPlus, Users,
} from "lucide-react";
import { Avatar } from "@/components/ui/avatar";
import { Badge, type Tone } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Combobox } from "@/components/ui/combobox";
import { Field, Input, Select } from "@/components/ui/input";
import { Modal, Section } from "@/components/ui/modal";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  listMemberFamilies, listFamilyMembers, listMembers, createFamily, updateFamily,
  deleteFamily, addFamilyMember, removeFamilyMember, assetURL,
  type Family, type FamilyAddress, type FamilyMemberRow, type Member,
} from "@/lib/api";
import { MEMBERSHIP_STATUS, RELATION_LABELS } from "@/lib/constants";
import { datePt } from "@/lib/format";

const RELACOES = Object.entries(RELATION_LABELS).map(([value, label]) => ({ value, label }));

const ENDERECO_VAZIO: FamilyAddress = {
  street: "", number: "", complement: "", district: "", city: "", state: "", zip: "",
};

/** Endereco em uma linha; devolve null quando nao ha nada preenchido. */
function enderecoLinha(a?: FamilyAddress): string | null {
  if (!a) return null;
  const linha1 = [a.street, a.number, a.complement].filter(Boolean).join(", ");
  const linha2 = [a.district, [a.city, a.state].filter(Boolean).join("/"), a.zip].filter(Boolean).join(" - ");
  const txt = [linha1, linha2].filter(Boolean).join(" - ");
  return txt || null;
}

/**
 * FamilySection e a aba "Familia" do membro.
 *
 * Substituiu a pagina /dashboard/families: o agrupamento familiar so faz sentido
 * a partir da ficha de alguem, e na planilha do cliente e assim que os dados
 * aparecem (a familia com as pessoas listadas abaixo, nao o contrario).
 *
 * O "chefe" da familia e o anchor dos parentes: os vinculos criados aqui sao
 * sempre "esta pessoa em relacao ao chefe", que e o que FamilyMembers projeta.
 */
export function FamilySection({
  memberId,
  memberName,
  canWrite,
  onChanged,
}: {
  memberId: string;
  memberName: string;
  canWrite: boolean;
  onChanged?: () => void;
}) {
  const { toast } = useToast();
  const [familias, setFamilias] = useState<Family[] | null>(null);
  const [porFamilia, setPorFamilia] = useState<Record<string, FamilyMemberRow[]>>({});
  const [pessoas, setPessoas] = useState<Member[]>([]);

  const [criando, setCriando] = useState(false);
  const [nomeNova, setNomeNova] = useState("");
  const [renomeando, setRenomeando] = useState<Family | null>(null);
  const [nomeEdit, setNomeEdit] = useState("");
  const [editandoEndereco, setEditandoEndereco] = useState<Family | null>(null);
  const [endereco, setEndereco] = useState<FamilyAddress>(ENDERECO_VAZIO);
  const [vinculando, setVinculando] = useState<Family | null>(null);
  const [alvo, setAlvo] = useState({ member_id: "", relation: "relative" });
  const [salvando, setSalvando] = useState(false);

  const load = useCallback(async () => {
    try {
      const { families } = await listMemberFamilies(memberId);
      setFamilias(families);
      const entradas = await Promise.all(
        families.map(async (f) => [f.id, (await listFamilyMembers(f.id)).members] as const),
      );
      setPorFamilia(Object.fromEntries(entradas));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar familias", "error");
      setFamilias([]);
    }
  }, [memberId, toast]);

  useEffect(() => {
    setFamilias(null);
    load();
  }, [load]);

  // Lista para os seletores de "vincular pessoa": o backend devolve ate 100
  // membros numa chamada so, o mesmo limite que a listagem de membros usa.
  useEffect(() => {
    listMembers()
      .then((r) => setPessoas(r.members))
      .catch(() => setPessoas([]));
  }, []);

  function reload() {
    load();
    onChanged?.();
  }

  async function criar(e: React.FormEvent) {
    e.preventDefault();
    setSalvando(true);
    try {
      const fam = await createFamily(nomeNova.trim() || `Familia ${memberName}`);
      // Sem anchor e sem chefe, quem entra primeiro VIRA o chefe (LinkMember).
      await addFamilyMember(fam.id, memberId, "", "relative");
      setCriando(false);
      setNomeNova("");
      toast(`Familia ${fam.code ? `#${fam.code} ` : ""}criada.`);
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao criar familia", "error");
    } finally {
      setSalvando(false);
    }
  }

  async function renomear(e: React.FormEvent) {
    e.preventDefault();
    if (!renomeando) return;
    setSalvando(true);
    try {
      await updateFamily(renomeando.id, { name: nomeEdit.trim() });
      setRenomeando(null);
      toast("Familia renomeada.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao renomear", "error");
    } finally {
      setSalvando(false);
    }
  }

  async function salvarEndereco(e: React.FormEvent) {
    e.preventDefault();
    if (!editandoEndereco) return;
    setSalvando(true);
    try {
      await updateFamily(editandoEndereco.id, { address: endereco });
      setEditandoEndereco(null);
      toast("Endereco atualizado.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar endereco", "error");
    } finally {
      setSalvando(false);
    }
  }

  async function definirChefe(fam: Family) {
    try {
      await updateFamily(fam.id, { head_id: memberId });
      toast("Chefe da familia atualizado.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao definir chefe", "error");
    }
  }

  async function vincular(e: React.FormEvent) {
    e.preventDefault();
    if (!vinculando || !alvo.member_id) return;
    setSalvando(true);
    try {
      // relate_id vazio => o backend ancora no chefe atual da familia.
      await addFamilyMember(vinculando.id, alvo.member_id, "", alvo.relation);
      setVinculando(null);
      setAlvo({ member_id: "", relation: "relative" });
      toast("Pessoa vinculada a familia.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao vincular", "error");
    } finally {
      setSalvando(false);
    }
  }

  async function desvincular(fam: Family, pessoa: FamilyMemberRow | { id: string; full_name: string }) {
    try {
      await removeFamilyMember(fam.id, pessoa.id);
      toast(`${pessoa.full_name} saiu da familia.`);
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao desvincular", "error");
    }
  }

  async function excluir(fam: Family) {
    try {
      await deleteFamily(fam.id);
      toast("Familia excluida.");
      reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir familia", "error");
    }
  }

  const nomeSugerido = useMemo(() => {
    const partes = memberName.trim().split(/\s+/);
    const sobrenome = partes.length > 1 ? partes[partes.length - 1] : memberName;
    return `Familia ${sobrenome}`.trim();
  }, [memberName]);

  if (familias === null) return <Card><SkeletonRows rows={3} /></Card>;

  return (
    <div className="space-y-4">
      <Card className="p-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="flex items-center gap-2 text-sm font-semibold">
              <Users className="h-4 w-4 text-sky-600" /> Familias
            </h3>
            <p className="mt-0.5 text-xs text-zinc-400">
              A familia agrupa as pessoas por residencia. O parentesco e sempre registrado em
              relacao ao chefe da familia.
            </p>
          </div>
          {canWrite && (
            <Button
              size="sm"
              onClick={() => {
                setNomeNova(nomeSugerido);
                setCriando(true);
              }}
            >
              <Plus className="h-4 w-4" /> Criar familia
            </Button>
          )}
        </div>
      </Card>

      {familias.length === 0 ? (
        <Card>
          <EmptyState
            icon={<Home className="h-10 w-10" />}
            title="Este membro ainda nao esta em uma familia"
            description="Crie uma familia para agrupar quem mora junto e registrar o parentesco."
          />
        </Card>
      ) : (
        familias.map((fam) => {
          const linhas = porFamilia[fam.id] ?? [];
          const souChefe = fam.head_id === memberId;
          const enderecoTxt = enderecoLinha(fam.address);
          const jaNaFamilia = new Set(linhas.map((l) => l.id));
          jaNaFamilia.add(fam.head_id ?? "");
          const candidatos = pessoas.filter((p) => !jaNaFamilia.has(p.id));

          return (
            <Card key={fam.id} className="p-3">
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="text-sm font-semibold">{fam.name}</h3>
                    {fam.code && (
                      <Badge tone="zinc" className="font-mono text-[10px]">#{fam.code}</Badge>
                    )}
                    {souChefe && (
                      <Badge tone="amber" className="text-[10px]">
                        <Crown className="mr-1 h-3 w-3" /> Chefe
                      </Badge>
                    )}
                  </div>
                  <p className="mt-0.5 flex items-center gap-1 text-xs text-zinc-400">
                    <Crown className="h-3 w-3" />
                    {fam.head_name ? `Chefe: ${fam.head_name}` : "Sem chefe definido"}
                    <span className="mx-1">-</span>
                    {fam.member_count} pessoa(s)
                  </p>
                  <p className="mt-0.5 flex items-start gap-1 text-xs text-zinc-400">
                    <MapPin className="mt-0.5 h-3 w-3 shrink-0" />
                    {enderecoTxt ?? "Endereco nao informado"}
                  </p>
                </div>

                {canWrite && (
                  <div className="flex flex-wrap gap-1.5">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setEditandoEndereco(fam);
                        setEndereco({ ...ENDERECO_VAZIO, ...(fam.address ?? {}) });
                      }}
                    >
                      <MapPin className="h-3.5 w-3.5" /> Endereco
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setRenomeando(fam);
                        setNomeEdit(fam.name);
                      }}
                    >
                      <Pencil className="h-3.5 w-3.5" /> Renomear
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={candidatos.length === 0}
                      onClick={() => {
                        setVinculando(fam);
                        setAlvo({ member_id: "", relation: "relative" });
                      }}
                    >
                      <UserPlus className="h-3.5 w-3.5" /> Vincular
                    </Button>
                    {!souChefe && (
                      <Button variant="ghost" size="sm" onClick={() => definirChefe(fam)}>
                        <Crown className="h-3.5 w-3.5" /> Tornar chefe
                      </Button>
                    )}
                    <Button variant="ghost" size="sm" onClick={() => desvincular(fam, { id: memberId, full_name: memberName })}>
                      <UserMinus className="h-3.5 w-3.5" /> Sair
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => excluir(fam)}>
                      <Trash2 className="h-3.5 w-3.5 text-red-500" />
                    </Button>
                  </div>
                )}
              </div>

              {linhas.length === 0 ? (
                <p className="mt-3 text-xs text-zinc-400">Nenhuma pessoa vinculada alem do chefe.</p>
              ) : (
                <ul className="mt-3 divide-y divide-zinc-100 dark:divide-zinc-800">
                  {linhas.map((p) => {
                    const st = MEMBERSHIP_STATUS[p.membership_status] ?? {
                      label: p.membership_status,
                      tone: "zinc",
                    };
                    return (
                      <li key={p.id} className="flex flex-wrap items-center gap-2 py-2">
                        <Avatar name={p.full_name} src={assetURL(p.photo_url)} size="sm" />
                        <div className="min-w-0">
                          <Link
                            href={`/dashboard/members/${p.id}`}
                            className="text-sm font-medium hover:text-sky-700 dark:hover:text-sky-400"
                          >
                            {p.full_name}
                          </Link>
                          <p className="text-xs text-zinc-400">
                            {[
                              p.is_head ? "Chefe da familia" : (RELATION_LABELS[p.kind ?? ""] ?? p.relation),
                              p.birth_date ? datePt(p.birth_date) : null,
                              p.whatsapp ?? p.phone ?? p.email,
                            ]
                              .filter(Boolean)
                              .join(" - ")}
                          </p>
                        </div>
                        <div className="ml-auto flex items-center gap-1.5">
                          <Badge tone={p.is_head ? "amber" : "sky"} className="text-[10px]">
                            {p.is_head ? "Chefe" : (RELATION_LABELS[p.kind ?? ""] ?? p.relation)}
                          </Badge>
                          <Badge tone={st.tone as Tone} className="text-[10px]">{st.label}</Badge>
                          {canWrite && p.id !== memberId && (
                            <Button
                              variant="ghost"
                              size="sm"
                              title="Remover da familia"
                              onClick={() => desvincular(fam, p)}
                            >
                              <UserMinus className="h-3.5 w-3.5 text-red-500" />
                            </Button>
                          )}
                        </div>
                      </li>
                    );
                  })}
                </ul>
              )}
            </Card>
          );
        })
      )}

      {/* Criar familia */}
      <Modal open={criando} onClose={() => setCriando(false)} title="Nova familia" size="sm">
        <form onSubmit={criar} className="space-y-3">
          <Field label="Nome da familia *" hint="O codigo (#NNN) e gerado automaticamente.">
            <Input
              required
              className="h-8 text-sm"
              value={nomeNova}
              onChange={(e) => setNomeNova(e.target.value)}
              placeholder="Ex: Familia Amaral"
            />
          </Field>
          <p className="text-xs text-zinc-400">
            {memberName} entra como chefe desta familia; os proximos vinculos serao registrados
            em relacao a ele(a).
          </p>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" size="sm" type="button" onClick={() => setCriando(false)}>Cancelar</Button>
            <Button size="sm" type="submit" disabled={salvando}>{salvando ? "Criando..." : "Criar"}</Button>
          </div>
        </form>
      </Modal>

      {/* Renomear */}
      <Modal open={renomeando !== null} onClose={() => setRenomeando(null)} title="Renomear familia" size="sm">
        <form onSubmit={renomear} className="space-y-3">
          <Field label="Nome *">
            <Input
              required
              className="h-8 text-sm"
              value={nomeEdit}
              onChange={(e) => setNomeEdit(e.target.value)}
            />
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" size="sm" type="button" onClick={() => setRenomeando(null)}>Cancelar</Button>
            <Button size="sm" type="submit" disabled={salvando}>{salvando ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Modal>

      {/* Endereco */}
      <Modal
        open={editandoEndereco !== null}
        onClose={() => setEditandoEndereco(null)}
        title={`Endereco - ${editandoEndereco?.name ?? ""}`}
      >
        <form onSubmit={salvarEndereco} className="space-y-3">
          <Section title="Endereco da familia">
            <Field label="Logradouro" className="sm:col-span-2">
              <Input className="h-8 text-sm" value={endereco.street ?? ""} onChange={(e) => setEndereco({ ...endereco, street: e.target.value })} />
            </Field>
            <Field label="Numero">
              <Input className="h-8 text-sm" value={endereco.number ?? ""} onChange={(e) => setEndereco({ ...endereco, number: e.target.value })} />
            </Field>
            <Field label="Complemento">
              <Input className="h-8 text-sm" value={endereco.complement ?? ""} onChange={(e) => setEndereco({ ...endereco, complement: e.target.value })} />
            </Field>
            <Field label="Bairro">
              <Input className="h-8 text-sm" value={endereco.district ?? ""} onChange={(e) => setEndereco({ ...endereco, district: e.target.value })} />
            </Field>
            <Field label="Cidade">
              <Input className="h-8 text-sm" value={endereco.city ?? ""} onChange={(e) => setEndereco({ ...endereco, city: e.target.value })} />
            </Field>
            <Field label="UF">
              <Input className="h-8 w-20 text-sm" maxLength={2} value={endereco.state ?? ""} onChange={(e) => setEndereco({ ...endereco, state: e.target.value.toUpperCase() })} />
            </Field>
            <Field label="CEP">
              <Input className="h-8 text-sm" value={endereco.zip ?? ""} onChange={(e) => setEndereco({ ...endereco, zip: e.target.value })} placeholder="00000-000" />
            </Field>
          </Section>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" size="sm" type="button" onClick={() => setEditandoEndereco(null)}>Cancelar</Button>
            <Button size="sm" type="submit" disabled={salvando}>{salvando ? "Salvando..." : "Salvar endereco"}</Button>
          </div>
        </form>
      </Modal>

      {/* Vincular pessoa */}
      <Modal
        open={vinculando !== null}
        onClose={() => setVinculando(null)}
        title={`Vincular pessoa - ${vinculando?.name ?? ""}`}
        size="sm"
      >
        <form onSubmit={vincular} className="space-y-3">
          <Field label="Pessoa *">
            <Combobox
              value={alvo.member_id}
              placeholder="Selecione a pessoa..."
              searchPlaceholder="Buscar por nome..."
              options={pessoas
                .filter((p) => {
                  const linhas = vinculando ? (porFamilia[vinculando.id] ?? []) : [];
                  return p.id !== memberId && !linhas.some((l) => l.id === p.id);
                })
                .map((p) => ({ value: p.id, label: p.full_name }))}
              onChange={(v) => setAlvo({ ...alvo, member_id: v })}
            />
          </Field>
          <Field label="Parentesco com o chefe">
            <Select
              className="h-8 text-sm"
              value={alvo.relation}
              onChange={(e) => setAlvo({ ...alvo, relation: e.target.value })}
            >
              {RELACOES.map((r) => (
                <option key={r.value} value={r.value}>{r.label}</option>
              ))}
            </Select>
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" size="sm" type="button" onClick={() => setVinculando(null)}>Cancelar</Button>
            <Button size="sm" type="submit" disabled={salvando || !alvo.member_id}>
              {salvando ? "Vinculando..." : "Vincular"}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
