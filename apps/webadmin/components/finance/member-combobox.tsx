"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import { useToast } from "@/components/ui/toast";
import { listMembers, searchMembers } from "@/lib/api";

type MemberComboboxProps = {
  value: string;
  /** Recebe o id e, quando disponivel, o nome do membro selecionado. */
  onChange: (id: string, label?: string) => void;
  /** Nome para exibir quando o id ja esta definido (ex.: edicao). */
  initialLabel?: string;
  className?: string;
  placeholder?: string;
  openOnFocus?: boolean;
};

const toOptions = (members: { id: string; full_name: string }[]): ComboboxOption[] =>
  members.map((m) => ({ value: m.id, label: m.full_name }));

// Varias linhas/telas montam o combobox ao mesmo tempo. Compartilha a lista
// inicial (sem busca) por 60s para nao disparar um GET /members por instancia.
let initialOptions: Promise<ComboboxOption[]> | null = null;
let initialOptionsAt = 0;
let warnedOnce = false;

function loadInitialOptions(): Promise<ComboboxOption[]> {
  if (initialOptions && Date.now() - initialOptionsAt < 60_000) return initialOptions;
  initialOptionsAt = Date.now();
  initialOptions = listMembers()
    .then((r) => toOptions(r.members))
    .catch((err) => {
      initialOptions = null;
      throw err;
    });
  return initialOptions;
}

/**
 * Combobox de membro com busca no servidor. O `GET /members` devolve no maximo
 * 100 registros por padrao, entao filtrar a lista localmente escondia membros
 * em igrejas maiores. Aqui cada termo digitado consulta `?q=` (com debounce) e
 * o resultado e devolvido pelo backend.
 */
export function MemberCombobox({
  value,
  onChange,
  initialLabel,
  className,
  placeholder = "Buscar membro...",
  openOnFocus = false,
}: MemberComboboxProps) {
  const { toast } = useToast();
  const [options, setOptions] = useState<ComboboxOption[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedLabel, setSelectedLabel] = useState(initialLabel ?? "");
  const timer = useRef<number | undefined>(undefined);
  const requestId = useRef(0);

  const load = useCallback(
    async (term: string) => {
      const current = ++requestId.current;
      setLoading(true);
      try {
        const q = term.trim();
        const next = q ? toOptions((await searchMembers(q)).members) : await loadInitialOptions();
        if (current !== requestId.current) return;
        setOptions(next);
      } catch (e) {
        if (current !== requestId.current) return;
        setOptions([]);
        if (!warnedOnce) {
          warnedOnce = true;
          toast(e instanceof Error ? e.message : "Falha ao buscar membros", "error");
        }
      } finally {
        if (current === requestId.current) setLoading(false);
      }
    },
    [toast],
  );

  useEffect(() => {
    load("");
    return () => {
      window.clearTimeout(timer.current);
      // Invalida respostas pendentes apos o unmount.
      requestId.current++;
    };
  }, [load]);

  const onSearch = useCallback(
    (term: string) => {
      window.clearTimeout(timer.current);
      timer.current = window.setTimeout(() => {
        load(term).catch(() => {});
      }, 250);
    },
    [load],
  );

  // Mantem o nome visivel ate a busca trazer a opcao correspondente.
  const merged = useMemo(() => {
    if (!value) return options;
    if (options.some((o) => o.value === value)) return options;
    return [{ value, label: selectedLabel || initialLabel || "Membro selecionado" }, ...options];
  }, [options, value, selectedLabel, initialLabel]);

  return (
    <Combobox
      value={value}
      placeholder={placeholder}
      searchPlaceholder={placeholder}
      emptyMessage="Nenhum membro encontrado"
      options={merged}
      onChange={(id) => {
        const found = merged.find((o) => o.value === id);
        if (found) setSelectedLabel(found.label);
        onChange(id, found?.label);
      }}
      onSearch={onSearch}
      loading={loading}
      className={className}
      openOnFocus={openOnFocus}
    />
  );
}
