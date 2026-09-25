"use client";

import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import {
  Plus, Trash2, Copy, Check, AlertCircle, SlidersHorizontal, ChevronUp, Loader2, MapPin,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input, Select, MaskedInput, Textarea } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  createMember, updateMember, syncMemberCargos, listCargos, lookupCep,
  type Cargo, type Member,
} from "@/lib/api";
import { GENDER, MARITAL_STATUS, MEMBERSHIP_STATUS, EXIT_REASONS, EXIT_STATUSES } from "@/lib/constants";
import { compactPersonForm } from "@/components/people/person-form";

interface MemberRow {
  key: number;
  id?: string;
  /** Alterada desde o ultimo save (modo edicao). */
  dirty?: boolean;
  expanded: boolean;
  // Linha principal
  first_name: string;
  last_name: string;
  phone: string;
  email: string;
  membership_status: string;
  // Complemento
  nickname: string;
  birth_date: string;
  gender: string;
  marital_status: string;
  cpf: string;
  rg: string;
  profession: string;
  nationality: string;
  education: string;
  address_zip_code: string;
  address_street: string;
  address_number: string;
  address_complement: string;
  address_district: string;
  address_city: string;
  address_state: string;
  baptism_date: string;
  baptism_location: string;
  marriage_date: string;
  joined_at: string;
  exit_reason: string;
  exited_at: string;
  notes: string;
  cargoIds: string[];
  cepStatus: "idle" | "loading" | "error" | "ok";
}

let seq = 0;
function emptyRow(): MemberRow {
  seq += 1;
  return {
    key: seq, expanded: false, dirty: false,
    first_name: "", last_name: "", phone: "", email: "", membership_status: "member",
    nickname: "", birth_date: "", gender: "", marital_status: "", cpf: "", rg: "",
    profession: "", nationality: "", education: "",
    address_zip_code: "", address_street: "", address_number: "", address_complement: "",
    address_district: "", address_city: "", address_state: "",
    baptism_date: "", baptism_location: "", marriage_date: "", joined_at: "",
    exit_reason: "", exited_at: "", notes: "", cargoIds: [], cepStatus: "idle",
  };
}

function fromMember(m: Member): MemberRow {
  seq += 1;
  return {
    key: seq, id: m.id, expanded: false, dirty: false,
    first_name: m.first_name, last_name: m.last_name,
    phone: m.phone ?? "", email: m.email ?? "", membership_status: m.membership_status,
    nickname: m.nickname ?? "", birth_date: m.birth_date ?? "", gender: m.gender ?? "",
    marital_status: m.marital_status ?? "", cpf: m.cpf ?? "", rg: m.rg ?? "",
    profession: m.profession ?? "", nationality: m.nationality ?? "", education: m.education ?? "",
    address_zip_code: m.address?.zip_code ?? "", address_street: m.address?.street ?? "",
    address_number: m.address?.number ?? "", address_complement: m.address?.complement ?? "",
    address_district: m.address?.district ?? "", address_city: m.address?.city ?? "",
    address_state: m.address?.state ?? "",
    baptism_date: m.baptism_date ?? "", baptism_location: m.baptism_location ?? "",
    marriage_date: m.marriage_date ?? "", joined_at: m.joined_at ?? "",
    exit_reason: m.exit_reason ?? "", exited_at: m.exited_at ?? "", notes: m.notes ?? "",
    cargoIds: (m.cargos ?? []).map((c) => c.id),
    cepStatus: "idle",
  };
}

function SubField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <span className="block text-[11px] font-medium uppercase tracking-wide text-zinc-500">{label}</span>
      {children}
    </div>
  );
}

function toPayload(r: MemberRow): Record<string, unknown> {
  const flat: Record<string, string> = {
    first_name: r.first_name, last_name: r.last_name, phone: r.phone, email: r.email,
    membership_status: r.membership_status, nickname: r.nickname, birth_date: r.birth_date,
    gender: r.gender, marital_status: r.marital_status, cpf: r.cpf, rg: r.rg,
    profession: r.profession, nationality: r.nationality, education: r.education,
    address_zip_code: r.address_zip_code, address_street: r.address_street,
    address_number: r.address_number, address_complement: r.address_complement,
    address_district: r.address_district, address_city: r.address_city, address_state: r.address_state,
    baptism_date: r.baptism_date, baptism_location: r.baptism_location,
    marriage_date: r.marriage_date, joined_at: r.joined_at,
    exit_reason: r.exit_reason, exited_at: r.exited_at, notes: r.notes,
  };
  // compactPersonForm descarta vazios (evita ''::date no backend) e junta o
  // endereco aninhado em `address`; depois removemos as chaves flat.
  const data = compactPersonForm(flat);
  const address: Record<string, string> = {};
  for (const [k, v] of Object.entries(flat)) {
    if (k.startsWith("address_") && v !== "") address[k.slice("address_".length)] = v;
  }
  for (const k of Object.keys(data)) if (k.startsWith("address_")) delete data[k];
  if (Object.keys(address).length) data.address = address;
  if (!EXIT_STATUSES.includes(r.membership_status)) {
    delete data.exit_reason;
    delete data.exited_at;
  }
  return data;
}

/**
 * Grid de membros, no padrao "linha principal + complemento expansivel".
 *
 * - `mode="create"`: linhas em branco para cadastro rapido.
 * - `mode="edit"`: carrega os membros recebidos e salva apenas as linhas
 *   alteradas (dirty), sem abrir modal.
 *
 * O endereco tem CEP consultado no ViaCEP. Os cargos marcados sao sincronizados
 * (mandatos) apos criar/atualizar.
 */
export function MemberBulkGrid({
  mode = "create",
  members,
  focusId,
  onSaved,
}: {
  mode?: "create" | "edit";
  members?: Member[];
  focusId?: string;
  onSaved: () => void;
}) {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const [cargos, setCargos] = useState<Cargo[]>([]);
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState<{ line: number; error: string }[]>([]);
  const [done, setDone] = useState<number | null>(null);
  const isEdit = mode === "edit";
  const initialRef = useRef<Member[]>(members ?? []);
  initialRef.current = members ?? [];

  const [rows, setRows] = useState<MemberRow[]>(() =>
    isEdit ? (members ?? []).map(fromMember) : [emptyRow(), emptyRow(), emptyRow()],
  );

  useEffect(() => {
    listCargos().then((r) => setCargos(r.cargos.filter((c) => c.is_active))).catch(() => setCargos([]));
  }, []);

  // Abre e rola ate o membro focado (usado pelo "Editar" da listagem).
  useEffect(() => {
    if (!focusId) return;
    setRows((rs) => rs.map((r) => (r.id === focusId ? { ...r, expanded: true } : r)));
    const t = setTimeout(() => {
      document.getElementById(`member-row-${focusId}`)?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 120);
    return () => clearTimeout(t);
  }, [focusId]);

  const update = (key: number, patch: Partial<MemberRow>, markDirty = true) =>
    setRows((rs) => rs.map((r) => (r.key === key ? { ...r, ...patch, ...(markDirty ? { dirty: true } : {}) } : r)));

  function addRow(copyOf?: MemberRow) {
    setRows((rs) => [...rs, copyOf ? { ...copyOf, key: ++seq, id: undefined, dirty: true, expanded: false } : emptyRow()]);
  }

  function setFullName(r: MemberRow, v: string) {
    const parts = v.trim().split(/\s+/).filter(Boolean);
    const first = parts.shift() ?? "";
    update(r.key, { first_name: first, last_name: parts.join(" ") });
  }

  const touched = useMemo(
    () => (isEdit ? rows.filter((r) => r.dirty) : rows.filter((r) => r.first_name.trim() !== "" || r.last_name.trim() !== "")),
    [rows, isEdit],
  );
  const ready = touched.filter((r) => r.first_name.trim() !== "");

  async function fillCep(r: MemberRow, value: string) {
    update(r.key, { address_zip_code: value, cepStatus: "idle" });
    const digits = value.replace(/\D/g, "");
    if (digits.length !== 8) return;
    update(r.key, { cepStatus: "loading" }, false);
    try {
      const res = await lookupCep(digits);
      if (!res) {
        update(r.key, { cepStatus: "error" }, false);
        return;
      }
      update(r.key, {
        cepStatus: "ok",
        address_street: res.street ?? r.address_street,
        address_district: res.district ?? r.address_district,
        address_city: res.city ?? r.address_city,
        address_state: res.state ?? r.address_state,
      });
    } catch {
      update(r.key, { cepStatus: "error" }, false);
    }
  }

  function complementCount(r: MemberRow): number {
    return [
      r.nickname, r.birth_date, r.gender, r.marital_status, r.cpf, r.rg, r.profession,
      r.nationality, r.education, r.address_zip_code, r.address_street, r.baptism_date,
      r.joined_at, r.notes,
    ].filter(Boolean).length + (r.cargoIds.length > 0 ? 1 : 0);
  }

  function resetRows() {
    setRows(isEdit ? initialRef.current.map(fromMember) : [emptyRow(), emptyRow(), emptyRow()]);
    setErrors([]);
    setDone(null);
  }

  async function submit() {
    if (!hasPerm("members.write")) return;
    if (touched.length === 0) {
      toast(isEdit ? "Nenhuma alteracao para salvar." : "Preencha ao menos um nome.", "error");
      return;
    }
    setSaving(true);
    setErrors([]);
    setDone(null);
    const errs: { line: number; error: string }[] = [];
    const failed: MemberRow[] = [];
    const successKeys = new Set<number>();
    let saved = 0;

    for (let i = 0; i < touched.length; i++) {
      const r = touched[i];
      if (!r.first_name.trim()) {
        errs.push({ line: i + 1, error: "nome e obrigatorio" });
        failed.push(r);
        continue;
      }
      try {
        if (r.id) {
          await updateMember(r.id, toPayload(r));
          await syncMemberCargos(r.id, r.cargoIds);
        } else {
          const novo = await createMember(toPayload(r));
          await syncMemberCargos(novo.id, r.cargoIds);
        }
        successKeys.add(r.key);
        saved++;
      } catch (e) {
        errs.push({ line: i + 1, error: e instanceof Error ? e.message : "erro" });
        failed.push(r);
      }
    }

    setDone(saved);
    setErrors(errs);
    if (saved > 0) {
      toast(`${saved} membro(s) ${isEdit ? "atualizado(s)" : "cadastrado(s)"}.`);
      onSaved();
    } else {
      toast("Nenhum membro foi salvo.", "error");
    }

    if (isEdit) {
      // Mantem as linhas; tira o "dirty" das que salvaram.
      setRows((rs) => rs.map((r) => (successKeys.has(r.key) ? { ...r, dirty: false } : r)));
    } else {
      setRows(failed.length > 0 ? failed : [emptyRow(), emptyRow(), emptyRow()]);
    }
    setSaving(false);
  }

  return (
    <div className="space-y-3">
      <div className="overflow-x-auto rounded-lg border border-zinc-200 dark:border-zinc-800">
        <table className="w-full min-w-[940px] text-[13px]">
          <thead className="border-b border-zinc-200 bg-zinc-50 text-[10px] uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900">
            <tr>
              <th className="w-8 px-2 py-2 text-left">#</th>
              <th className="px-2 py-2 text-left">Nome completo</th>
              <th className="w-40 px-2 py-2 text-left">Telefone</th>
              <th className="w-56 px-2 py-2 text-left">E-mail</th>
              <th className="w-36 px-2 py-2 text-left">Situacao</th>
              <th className="w-14 px-2 py-2"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/70">
            {rows.map((r, i) => (
              <Fragment key={r.key}>
                <tr
                  id={r.id ? `member-row-${r.id}` : undefined}
                  className={r.expanded ? "bg-sky-50/40 dark:bg-sky-950/10" : r.dirty ? "bg-amber-50/40 dark:bg-amber-950/10" : undefined}
                >
                  <td className="px-2 py-1 text-right text-zinc-400">
                    {i + 1}
                    {r.dirty && <span title="Alterado" className="ml-1 inline-block h-1.5 w-1.5 rounded-full bg-amber-500 align-middle" />}
                  </td>
                  <td className="px-2 py-1">
                    <Input
                      className="h-8 text-xs"
                      placeholder="Nome e sobrenome"
                      value={`${r.first_name} ${r.last_name}`.trim()}
                      onChange={(e) => setFullName(r, e.target.value)}
                    />
                  </td>
                  <td className="px-2 py-1">
                    <MaskedInput mask="phone" className="h-8 text-xs" value={r.phone} onChange={(e) => update(r.key, { phone: e.target.value })} />
                  </td>
                  <td className="px-2 py-1">
                    <Input type="email" className="h-8 text-xs" placeholder="email@exemplo.com" value={r.email} onChange={(e) => update(r.key, { email: e.target.value })} />
                  </td>
                  <td className="px-2 py-1">
                    <Select className="h-8 text-xs" value={r.membership_status} onChange={(e) => update(r.key, { membership_status: e.target.value })}>
                      {Object.entries(MEMBERSHIP_STATUS).map(([k, v]) => (<option key={k} value={k}>{v.label}</option>))}
                    </Select>
                  </td>
                  <td className="px-2 py-1">
                    <div className="flex items-center justify-center gap-0.5">
                      <button
                        type="button"
                        title="Complemento do cadastro"
                        onClick={() => update(r.key, { expanded: !r.expanded }, false)}
                        className={`relative rounded p-1 hover:bg-zinc-100 dark:hover:bg-zinc-800 ${r.expanded ? "text-sky-600" : "text-zinc-400"}`}
                      >
                        <SlidersHorizontal className="h-4 w-4" />
                        {complementCount(r) > 0 && (
                          <span className="absolute -right-0.5 -top-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-sky-600 text-[9px] font-bold text-white">
                            {complementCount(r)}
                          </span>
                        )}
                      </button>
                      {!isEdit && (
                        <>
                          <button type="button" title="Duplicar linha" onClick={() => addRow(r)} className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-sky-600 dark:hover:bg-zinc-800">
                            <Copy className="h-4 w-4" />
                          </button>
                          <button type="button" title="Remover linha" onClick={() => setRows((rs) => (rs.length > 1 ? rs.filter((x) => x.key !== r.key) : rs))} className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-red-600 dark:hover:bg-zinc-800">
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>

                {r.expanded && (
                  <tr className="bg-zinc-50/70 dark:bg-zinc-900/40">
                    <td colSpan={6} className="px-3 py-3">
                      <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Documentos</p>
                      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                        <SubField label="Apelido"><Input className="h-8 text-xs" value={r.nickname} onChange={(e) => update(r.key, { nickname: e.target.value })} /></SubField>
                        <SubField label="Nascimento"><Input type="date" className="h-8 text-xs" value={r.birth_date} onChange={(e) => update(r.key, { birth_date: e.target.value })} /></SubField>
                        <SubField label="Sexo">
                          <Select className="h-8 text-xs" value={r.gender} onChange={(e) => update(r.key, { gender: e.target.value })}>
                            <option value="">-</option>
                            {Object.entries(GENDER).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
                          </Select>
                        </SubField>
                        <SubField label="Estado civil">
                          <Select className="h-8 text-xs" value={r.marital_status} onChange={(e) => update(r.key, { marital_status: e.target.value })}>
                            <option value="">-</option>
                            {Object.entries(MARITAL_STATUS).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
                          </Select>
                        </SubField>
                        <SubField label="CPF"><MaskedInput mask="cpf" className="h-8 text-xs" value={r.cpf} onChange={(e) => update(r.key, { cpf: e.target.value })} /></SubField>
                        <SubField label="RG"><MaskedInput mask="rg" className="h-8 text-xs" value={r.rg} onChange={(e) => update(r.key, { rg: e.target.value })} /></SubField>
                        <SubField label="Profissao"><Input className="h-8 text-xs" value={r.profession} onChange={(e) => update(r.key, { profession: e.target.value })} /></SubField>
                        <SubField label="Nacionalidade"><Input className="h-8 text-xs" value={r.nationality} onChange={(e) => update(r.key, { nationality: e.target.value })} /></SubField>
                        <SubField label="Escolaridade"><Input className="h-8 text-xs" value={r.education} onChange={(e) => update(r.key, { education: e.target.value })} /></SubField>
                      </div>

                      <p className="mb-2 mt-3 text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Endereco</p>
                      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                        <SubField label="CEP">
                          <div className="relative">
                            <MaskedInput mask="cep" className="h-8 text-xs" value={r.address_zip_code} onChange={(e) => fillCep(r, e.target.value)} />
                            <span className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2">
                              {r.cepStatus === "loading" && <Loader2 className="h-3.5 w-3.5 animate-spin text-sky-500" />}
                              {r.cepStatus === "ok" && <MapPin className="h-3.5 w-3.5 text-emerald-600" />}
                              {r.cepStatus === "error" && <AlertCircle className="h-3.5 w-3.5 text-amber-500" />}
                            </span>
                          </div>
                        </SubField>
                        <SubField label="Logradouro"><Input className="h-8 text-xs" value={r.address_street} onChange={(e) => update(r.key, { address_street: e.target.value })} /></SubField>
                        <SubField label="Numero"><Input className="h-8 text-xs" value={r.address_number} onChange={(e) => update(r.key, { address_number: e.target.value })} /></SubField>
                        <SubField label="Complemento"><Input className="h-8 text-xs" value={r.address_complement} onChange={(e) => update(r.key, { address_complement: e.target.value })} /></SubField>
                        <SubField label="Bairro"><Input className="h-8 text-xs" value={r.address_district} onChange={(e) => update(r.key, { address_district: e.target.value })} /></SubField>
                        <SubField label="Cidade"><Input className="h-8 text-xs" value={r.address_city} onChange={(e) => update(r.key, { address_city: e.target.value })} /></SubField>
                        <SubField label="UF"><Input className="h-8 text-xs uppercase" maxLength={2} value={r.address_state} onChange={(e) => update(r.key, { address_state: e.target.value.toUpperCase() })} /></SubField>
                      </div>

                      <p className="mb-2 mt-3 text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Vida eclesiastica</p>
                      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                        <SubField label="Data do batismo"><Input type="date" className="h-8 text-xs" value={r.baptism_date} onChange={(e) => update(r.key, { baptism_date: e.target.value })} /></SubField>
                        <SubField label="Local do batismo"><Input className="h-8 text-xs" value={r.baptism_location} onChange={(e) => update(r.key, { baptism_location: e.target.value })} /></SubField>
                        <SubField label="Data de casamento"><Input type="date" className="h-8 text-xs" value={r.marriage_date} onChange={(e) => update(r.key, { marriage_date: e.target.value })} /></SubField>
                        <SubField label="Membro desde"><Input type="date" className="h-8 text-xs" value={r.joined_at} onChange={(e) => update(r.key, { joined_at: e.target.value })} /></SubField>
                        {EXIT_STATUSES.includes(r.membership_status) && (
                          <>
                            <SubField label="Motivo da baixa">
                              <Select className="h-8 text-xs" value={r.exit_reason} onChange={(e) => update(r.key, { exit_reason: e.target.value })}>
                                <option value="">-</option>
                                {Object.entries(EXIT_REASONS).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
                              </Select>
                            </SubField>
                            <SubField label="Data de saida"><Input type="date" className="h-8 text-xs" value={r.exited_at} onChange={(e) => update(r.key, { exited_at: e.target.value })} /></SubField>
                          </>
                        )}
                      </div>

                      {cargos.length > 0 && (
                        <div className="mt-3">
                          <SubField label="Cargos">
                            <div className="flex flex-wrap gap-x-4 gap-y-1 rounded border border-zinc-200 p-2 dark:border-zinc-800">
                              {cargos.map((c) => {
                                const checked = r.cargoIds.includes(c.id);
                                return (
                                  <label key={c.id} className="flex items-center gap-1.5 whitespace-nowrap text-xs text-zinc-600 dark:text-zinc-300">
                                    <input
                                      type="checkbox"
                                      checked={checked}
                                      onChange={(e) => update(r.key, {
                                        cargoIds: e.target.checked ? [...r.cargoIds, c.id] : r.cargoIds.filter((id) => id !== c.id),
                                      })}
                                      className="h-3.5 w-3.5 rounded border-zinc-300 text-sky-600"
                                    />
                                    {c.name}
                                  </label>
                                );
                              })}
                            </div>
                          </SubField>
                        </div>
                      )}

                      <div className="mt-3 grid gap-3">
                        <SubField label="Observacoes"><Textarea rows={2} className="text-xs" value={r.notes} onChange={(e) => update(r.key, { notes: e.target.value })} /></SubField>
                      </div>

                      <div className="mt-2 flex justify-end">
                        <Button type="button" variant="ghost" size="sm" className="h-7 text-xs" onClick={() => update(r.key, { expanded: false }, false)}>
                          <ChevronUp className="h-3.5 w-3.5" /> Recolher
                        </Button>
                      </div>
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        {!isEdit && (
          <>
            <Button type="button" variant="outline" size="sm" onClick={() => addRow()}>
              <Plus className="h-4 w-4" /> Adicionar linha
            </Button>
            <button
              type="button"
              onClick={() => setRows((rs) => [...rs, ...Array.from({ length: 5 }, () => emptyRow())])}
              className="text-xs text-sky-600 hover:underline"
            >
              +5 linhas
            </button>
          </>
        )}
        <span className="ml-auto text-xs text-zinc-500">
          {isEdit ? `${ready.length} alteracao(oes)` : `${ready.length} membro(s) pronto(s)`}
        </span>
      </div>

      {done !== null && (
        <div className="rounded border border-zinc-200 p-3 text-sm dark:border-zinc-700">
          <p className="flex items-center gap-2 font-medium">
            {done > 0 ? <Check className="h-4 w-4 text-emerald-600" /> : <AlertCircle className="h-4 w-4 text-amber-500" />}
            <Badge tone="green">{done} {isEdit ? "atualizado(s)" : "cadastrado(s)"}</Badge>
            {errors.length > 0 && <Badge tone="amber">{errors.length} com erro</Badge>}
          </p>
          {errors.length > 0 && (
            <ul className="mt-2 max-h-32 space-y-1 overflow-y-auto text-xs text-red-600">
              {errors.map((e, i) => (<li key={i}>Linha {e.line}: {e.error}</li>))}
            </ul>
          )}
        </div>
      )}

      <div className="flex flex-wrap justify-end gap-2 border-t border-zinc-100 pt-3">
        <Button type="button" variant="ghost" onClick={resetRows} disabled={saving}>
          {isEdit ? "Descartar alteracoes" : "Limpar"}
        </Button>
        <Button type="button" onClick={submit} disabled={saving || !hasPerm("members.write")}>
          {saving ? "Salvando..." : isEdit ? `Salvar ${ready.length || ""}`.trim() : `Cadastrar ${ready.length || ""}`.trim()}
        </Button>
      </div>
    </div>
  );
}
