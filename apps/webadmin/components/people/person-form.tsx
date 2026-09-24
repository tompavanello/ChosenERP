"use client";

import { useEffect, useState } from "react";
import { Field, Input, Select, MaskedInput, Textarea } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Section } from "@/components/ui/modal";
import { Combobox } from "@/components/ui/combobox";
import { CargoPicker } from "@/components/people/cargo-picker";
import { PhotoField } from "@/components/people/photo-field";
import { GENDER, MARITAL_STATUS, MEMBERSHIP_STATUS, VISITOR_SOURCES, EXIT_REASONS, EXIT_STATUSES } from "@/lib/constants";
import { listMemberCargos, type Person } from "@/lib/api";

export type PersonFormState = Record<string, string>;

export type PersonInput = Partial<Person> & { nickname?: string; photo_url?: string };

/** O que o formulário devolve além do payload JSON. */
export interface PersonSubmitContext {
  /** Cargos marcados. Quem grava é o pai (na criação o id do membro ainda não existe). */
  cargoIds: string[];
}

const EMPTY_STATE: PersonFormState = {
  first_name: "", last_name: "", nickname: "", birth_date: "", gender: "",
  marital_status: "", profession: "", nationality: "", education: "", cpf: "", rg: "", email: "", phone: "",
  whatsapp: "", membership_status: "member", baptism_date: "",
  baptism_location: "", joined_at: "", source: "", notes: "",
  address_zip_code: "", address_street: "", address_number: "", address_complement: "",
  address_district: "", address_city: "", address_state: "",
  exit_reason: "", exited_at: "", marriage_date: "",
};

export function toPersonForm(p?: PersonInput | null): PersonFormState {
  if (!p) return { ...EMPTY_STATE };
  return {
    first_name: p.first_name ?? "",
    last_name: p.last_name ?? "",
    nickname: p.nickname ?? "",
    birth_date: p.birth_date ?? "",
    gender: p.gender ?? "",
    marital_status: p.marital_status ?? "",
    profession: p.profession ?? "",
    nationality: p.nationality ?? "",
    education: p.education ?? "",
    cpf: p.cpf ?? "",
    rg: p.rg ?? "",
    email: p.email ?? "",
    phone: p.phone ?? "",
    whatsapp: p.whatsapp ?? "",
    membership_status: p.membership_status ?? "member",
    baptism_date: p.baptism_date ?? "",
    baptism_location: p.baptism_location ?? "",
    joined_at: p.joined_at ?? "",
    source: p.source ?? "",
    notes: p.notes ?? "",
    address_zip_code: p.address?.zip_code ?? "",
    address_street: p.address?.street ?? "",
    address_number: p.address?.number ?? "",
    address_complement: p.address?.complement ?? "",
    address_district: p.address?.district ?? "",
    address_city: p.address?.city ?? "",
    address_state: p.address?.state ?? "",
    exit_reason: p.exit_reason ?? "",
    exited_at: p.exited_at ?? "",
    marriage_date: p.marriage_date ?? "",
  };
}

export const toMemberForm = toPersonForm;

/**
 * Descarta os campos vazios do payload.
 *
 * Não é cosmético: o backend converte as datas com `$n::date`, e string vazia
 * vira `''::date` — erro 500 no Postgres. Era essa a causa do "quebra ao salvar"
 * em cadastros sem data de nascimento. Campo ausente mantém o valor atual (PATCH).
 */
export function compactPersonForm(f: PersonFormState): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(f)) {
    if (v !== "") out[k] = v;
  }
  return out;
}

export const compactMemberForm = compactPersonForm;

export interface PersonFormProps {
  entityType: "member" | "visitor" | "benefactor";
  /**
   * Id do membro já salvo. Habilita foto e carregamento dos cargos atuais —
   * ambos dependem de um id que ainda não existe na inclusão.
   */
  memberId?: string;
  initial?: PersonInput | null;
  visitor?: { full_name?: string; email?: string; phone?: string; whatsapp?: string } | null;
  submitLabel?: string;
  saving?: boolean;
  loading?: boolean;
  onSubmit: (data: Record<string, unknown>, ctx: PersonSubmitContext) => Promise<void>;
  onCancel: () => void;
  /** Notifica o pai quando a foto muda, para o avatar fora do formulário acompanhar. */
  onPhotoChange?: (url?: string) => void;
}

export function PersonForm({
  entityType,
  memberId,
  initial,
  visitor,
  submitLabel = "Salvar",
  saving,
  loading,
  onSubmit,
  onCancel,
  onPhotoChange,
}: PersonFormProps) {
  const [form, setForm] = useState<PersonFormState>(() => {
    let base: PersonFormState = initial ? toPersonForm(initial) : { ...EMPTY_STATE };
    if (visitor) {
      const parts = (visitor.full_name ?? "").trim().split(/\s+/);
      const fn = parts.length > 1 ? parts.slice(0, -1).join(" ") : parts[0] ?? "";
      const ln = parts.length > 1 ? parts[parts.length - 1] : "";
      base = {
        ...base,
        first_name: fn || base.first_name,
        last_name: ln || base.last_name,
        email: visitor.email ?? base.email,
        phone: visitor.phone ?? base.phone,
        whatsapp: visitor.whatsapp ?? base.whatsapp,
      };
    }
    return base;
  });
  const [cargoIds, setCargoIds] = useState<string[]>([]);
  const [photoUrl, setPhotoUrl] = useState<string | undefined>(
    () => (initial as PersonInput | null | undefined)?.photo_url ?? undefined,
  );
  const [tab, setTab] = useState<"dados" | "igreja">("dados");
  const isSaving = saving ?? loading ?? false;
  const set = (k: string, v: string) => setForm((f) => ({ ...f, [k]: v }));

  // UI de nome único: o backend ainda guarda first_name/last_name (NOT NULL),
  // então a digitação é dividida aqui — primeiro token = nome, resto = sobrenome.
  const fullName = `${form.first_name} ${form.last_name}`.trim();
  function setFullName(v: string) {
    const parts = v.trim().split(/\s+/).filter(Boolean);
    const first = parts.shift() ?? "";
    setForm((f) => ({ ...f, first_name: first, last_name: parts.join(" ") }));
  }

  // Cargos atuais do membro: só os ATIVOS entram marcados. Os encerrados vivem
  // na aba Cargos do detalhe, que é onde o histórico completo faz sentido.
  useEffect(() => {
    if (entityType !== "member" || !memberId) return;
    listMemberCargos(memberId)
      .then((r) => setCargoIds(r.cargos.filter((c) => c.status === "ativo").map((c) => c.cargo_id)))
      .catch(() => setCargoIds([]));
  }, [entityType, memberId]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const data = compactPersonForm(form);
    // Endereço é aninhado na API (jsonb): junta os campos flat address_* num
    // objeto `address` e remove as chaves flat, que o backend recusaria.
    const address: Record<string, string> = {};
    for (const [k, v] of Object.entries(form)) {
      if (k.startsWith("address_") && v !== "") address[k.slice("address_".length)] = v;
    }
    for (const k of Object.keys(data)) {
      if (k.startsWith("address_")) delete data[k];
    }
    if (Object.keys(address).length) data.address = address;
    // Motivo/data de saída só valem para situações de baixa.
    if (!EXIT_STATUSES.includes(form.membership_status)) {
      delete data.exit_reason;
      delete data.exited_at;
    }
    if (entityType === "benefactor") {
      data.name = `${form.first_name} ${form.last_name}`.trim() || form.nickname;
    }
    await onSubmit(data, { cargoIds });
  }

  const isMember = entityType === "member";
  const isVisitor = entityType === "visitor";
  const isBenefactor = entityType === "benefactor";
  const isExit = isMember && EXIT_STATUSES.includes(form.membership_status);

  return (
    <form onSubmit={submit} className="space-y-3">
      {isMember && (
        <div className="mb-1 flex gap-1 border-b border-zinc-200 dark:border-zinc-800">
          <button
            type="button"
            onClick={() => setTab("dados")}
            className={`border-b-2 px-3 py-1.5 text-sm font-medium transition ${tab === "dados" ? "border-sky-600 text-sky-700 dark:text-sky-400" : "border-transparent text-zinc-500 hover:text-zinc-700"}`}
          >
            Dados e endereço
          </button>
          <button
            type="button"
            onClick={() => setTab("igreja")}
            className={`border-b-2 px-3 py-1.5 text-sm font-medium transition ${tab === "igreja" ? "border-sky-600 text-sky-700 dark:text-sky-400" : "border-transparent text-zinc-500 hover:text-zinc-700"}`}
          >
            Vida eclesiástica
          </button>
        </div>
      )}

      {/* Aba "Dados e endereço". Para visitante/benfeitor é a única tela.
          Fica montada (só oculta) para não perder a validação nativa. */}
      <div className={isMember && tab !== "dados" ? "hidden" : undefined}>
      {isMember && (
        <Section title="Foto">
          <div className="sm:col-span-2">
            {memberId ? (
              <PhotoField
                memberId={memberId}
                name={`${form.first_name} ${form.last_name}`.trim()}
                photoUrl={photoUrl}
                onChange={(url) => {
                  setPhotoUrl(url);
                  onPhotoChange?.(url);
                }}
              />
            ) : (
              <p className="text-xs text-zinc-400">
                A foto pode ser enviada logo depois de salvar — o cadastro precisa existir
                antes do upload.
              </p>
            )}
          </div>
        </Section>
      )}

      <Section title="Identificação">
        <Field label="Nome completo *" required className="sm:col-span-2">
          <Input required className="h-8 text-sm" value={fullName} onChange={(e) => setFullName(e.target.value)} />
        </Field>
        <Field label="Apelido">
          <Input className="h-8 text-sm" value={form.nickname} onChange={(e) => set("nickname", e.target.value)} />
        </Field>
        <Field label="Nascimento">
          <Input type="date" className="h-8 text-sm" value={form.birth_date} onChange={(e) => set("birth_date", e.target.value)} />
        </Field>
        <Field label="Sexo">
          <Select className="h-8 text-sm" value={form.gender} onChange={(e) => set("gender", e.target.value)}>
            <option value="">—</option>
            {Object.entries(GENDER).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
          </Select>
        </Field>
        <Field label="Estado civil">
          <Select className="h-8 text-sm" value={form.marital_status} onChange={(e) => set("marital_status", e.target.value)}>
            <option value="">—</option>
            {Object.entries(MARITAL_STATUS).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
          </Select>
        </Field>
      </Section>

      <Section title="Documentos" hint="Opcional — usado em recibos e cartas.">
        <Field label="CPF">
          <MaskedInput
            mask="cpf"
            className="h-8 text-sm"
            value={form.cpf}
            onChange={(e) => set("cpf", e.target.value)}
            placeholder="000.000.000-00"
          />
        </Field>
        <Field label="RG">
          <MaskedInput
            mask="rg"
            className="h-8 text-sm"
            value={form.rg}
            onChange={(e) => set("rg", e.target.value)}
            placeholder="0.000.000-SSP"
          />
        </Field>
        {isMember && (
          <Field label="Profissão">
            <Input className="h-8 text-sm" value={form.profession} onChange={(e) => set("profession", e.target.value)} />
          </Field>
        )}
        {isMember && (
          <Field label="Nacionalidade">
            <Input className="h-8 text-sm" value={form.nationality} onChange={(e) => set("nationality", e.target.value)} />
          </Field>
        )}
        {isMember && (
          <Field label="Escolaridade">
            <Input className="h-8 text-sm" value={form.education} onChange={(e) => set("education", e.target.value)} />
          </Field>
        )}
      </Section>

      <Section title="Contato">
        <Field label="E-mail" className="sm:col-span-2">
          <Input type="email" className="h-8 text-sm" value={form.email} onChange={(e) => set("email", e.target.value)} />
        </Field>
        <Field label="Telefone">
          <MaskedInput mask="phone" className="h-8 text-sm" value={form.phone} onChange={(e) => set("phone", e.target.value)} />
        </Field>
        <Field label="WhatsApp">
          <MaskedInput mask="phone" className="h-8 text-sm" value={form.whatsapp} onChange={(e) => set("whatsapp", e.target.value)} />
        </Field>
        {isVisitor && (
          <Field label="Como nos conheceu?">
            <Select className="h-8 text-sm" value={form.source} onChange={(e) => set("source", e.target.value)}>
              <option value="">—</option>
              {VISITOR_SOURCES.map((s) => (<option key={s.value} value={s.value}>{s.label}</option>))}
            </Select>
          </Field>
        )}
      </Section>

      {isMember && (
        <Section title="Endereço" hint="Endereço do membro (requisito 1.1).">
          <Field label="CEP">
            <MaskedInput mask="cep" className="h-8 text-sm" value={form.address_zip_code} onChange={(e) => set("address_zip_code", e.target.value)} placeholder="00000-000" />
          </Field>
          <Field label="Logradouro" className="sm:col-span-2">
            <Input className="h-8 text-sm" value={form.address_street} onChange={(e) => set("address_street", e.target.value)} />
          </Field>
          <Field label="Número">
            <Input className="h-8 text-sm" value={form.address_number} onChange={(e) => set("address_number", e.target.value)} />
          </Field>
          <Field label="Complemento">
            <Input className="h-8 text-sm" value={form.address_complement} onChange={(e) => set("address_complement", e.target.value)} />
          </Field>
          <Field label="Bairro">
            <Input className="h-8 text-sm" value={form.address_district} onChange={(e) => set("address_district", e.target.value)} />
          </Field>
          <Field label="Cidade">
            <Input className="h-8 text-sm" value={form.address_city} onChange={(e) => set("address_city", e.target.value)} />
          </Field>
          <Field label="UF">
            <Input className="h-8 text-sm uppercase" maxLength={2} value={form.address_state} onChange={(e) => set("address_state", e.target.value.toUpperCase())} />
          </Field>
        </Section>
      )}
      </div>

      {isMember && (
        <div className={tab === "igreja" ? undefined : "hidden"}>
        <Section title="Vida eclesiástica">
          <Field label="Status">
            <Combobox
              value={form.membership_status}
              placeholder="Selecione..."
              options={Object.entries(MEMBERSHIP_STATUS).map(([k, v]) => ({ value: k, label: v.label }))}
              onChange={(val) => set("membership_status", val)}
            />
          </Field>
          <Field label="Cargos" className="sm:col-span-2" hint="Um membro pode exercer mais de um cargo.">
            <CargoPicker value={cargoIds} onChange={setCargoIds} />
          </Field>
          {isExit && (
            <>
              <Field label="Motivo da baixa" hint="Requisito 1.7.">
                <Select className="h-8 text-sm" value={form.exit_reason} onChange={(e) => set("exit_reason", e.target.value)}>
                  <option value="">—</option>
                  {Object.entries(EXIT_REASONS).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
                </Select>
              </Field>
              <Field label="Data de saída">
                <Input type="date" className="h-8 text-sm" value={form.exited_at} onChange={(e) => set("exited_at", e.target.value)} />
              </Field>
            </>
          )}
          <Field label="Data do batismo">
            <Input type="date" className="h-8 text-sm" value={form.baptism_date} onChange={(e) => set("baptism_date", e.target.value)} />
          </Field>
          <Field label="Local do batismo">
            <Input className="h-8 text-sm" value={form.baptism_location} onChange={(e) => set("baptism_location", e.target.value)} placeholder="Ex: Igreja Sede Matriz" />
          </Field>
          <Field label="Data de casamento" hint="Usada nos aniversários de casamento.">
            <Input type="date" className="h-8 text-sm" value={form.marriage_date} onChange={(e) => set("marriage_date", e.target.value)} />
          </Field>
          <Field label="Membro desde">
            <Input type="date" className="h-8 text-sm" value={form.joined_at} onChange={(e) => set("joined_at", e.target.value)} />
          </Field>
        </Section>

        <Section title="Observações">
          <Field label="Notas" className="sm:col-span-2">
            <Textarea rows={3} value={form.notes} onChange={(e) => set("notes", e.target.value)} />
          </Field>
        </Section>
        </div>
      )}

      {isBenefactor && (
        <Section title="Observações">
          <Field label="Notas" className="sm:col-span-2">
            <Textarea rows={3} value={form.notes} onChange={(e) => set("notes", e.target.value)} />
          </Field>
        </Section>
      )}

      <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
        <Button variant="ghost" type="button" className="h-8 text-sm" onClick={onCancel}>Cancelar</Button>
        <Button type="submit" className="h-8 text-sm" disabled={isSaving}>{isSaving ? "Salvando..." : submitLabel}</Button>
      </div>
    </form>
  );
}
