"use client";

import { useCallback, useEffect, useState } from "react";
import { Plus, Settings2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import { Badge, type Tone } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { CargosManagerDialog } from "@/components/people/cargos-manager";
import { listCargos, type Cargo } from "@/lib/api";
import { CARGO_KINDS } from "@/lib/constants";

/**
 * CargoPicker é o seletor múltiplo de cargos do membro (requisito 1.3: "um mesmo
 * membro pode ter MAIS DE UMA função"). Substituiu o Combobox de valor único que
 * lia o dicionário fixo OFFICES do frontend — agora o catálogo vem do banco, e a
 * própria igreja cria os cargos pelo diálogo "Gerenciar".
 *
 * O componente só cuida da SELEÇÃO: quem grava é o pai, chamando
 * syncMemberCargos() depois de salvar o membro (na criação o id ainda não existe).
 */
export function CargoPicker({
  value,
  onChange,
}: {
  value: string[];
  onChange: (ids: string[]) => void;
}) {
  const [cargos, setCargos] = useState<Cargo[] | null>(null);
  const [gerenciar, setGerenciar] = useState(false);

  const load = useCallback(() => {
    listCargos()
      .then((r) => setCargos(r.cargos))
      .catch(() => setCargos([]));
  }, []);

  useEffect(load, [load]);

  if (cargos === null) return <Skeleton className="h-9 w-full" />;

  const ativos = cargos.filter((c) => c.is_active);
  const selecionados = value
    .map((id) => cargos.find((c) => c.id === id))
    .filter((c): c is Cargo => Boolean(c));

  const disponiveis = ativos.filter((c) => !value.includes(c.id));

  function adicionar(id: string) {
    if (!id || value.includes(id)) return;
    onChange([...value, id]);
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1">
          <Combobox
            value=""
            placeholder={disponiveis.length === 0 ? "Todos os cargos já atribuídos" : "Adicionar cargo..."}
            searchPlaceholder="Buscar cargo..."
            options={[
              { value: "", label: "Adicionar cargo..." },
              ...disponiveis.map((c) => ({ value: c.id, label: c.name })),
            ]}
            onChange={adicionar}
          />
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setGerenciar(true)}
          title="Criar, renomear ou desativar cargos"
        >
          <Settings2 className="h-4 w-4" /> Gerenciar
        </Button>
      </div>

      {selecionados.length === 0 ? (
        <p className="text-xs text-zinc-400">
          Nenhum cargo atribuído. Use o campo acima para adicionar quantos quiser.
        </p>
      ) : (
        <ul className="flex flex-wrap gap-1.5">
          {selecionados.map((c) => {
            const kind = CARGO_KINDS[c.kind] ?? CARGO_KINDS.outro;
            return (
              <li key={c.id}>
                <Badge tone={kind.tone as Tone} className="gap-1.5 pr-1">
                  {c.name}
                  <button
                    type="button"
                    title={`Remover ${c.name}`}
                    onClick={() => onChange(value.filter((id) => id !== c.id))}
                    className="rounded-full p-0.5 hover:bg-black/10"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              </li>
            );
          })}
        </ul>
      )}

      {cargos.length === 0 && (
        <p className="flex items-center gap-1.5 text-xs text-amber-600">
          <Plus className="h-3.5 w-3.5" />
          O catálogo está vazio — crie o primeiro cargo em <strong>Gerenciar</strong>.
        </p>
      )}

      <CargosManagerDialog
        open={gerenciar}
        onClose={() => setGerenciar(false)}
        onChanged={load}
      />
    </div>
  );
}
