"use client";

import { useEffect, useState } from "react";
import { ExternalLink, Loader2, Paperclip } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { getTransactionGroup, type TransactionGroup, type TransactionGroupMember } from "@/lib/api";
import { currency, datePt } from "@/lib/format";

type Props = {
  groupId: string;
  onOpenGroup?: (groupId: string) => void;
  onOpenTransaction?: (member: TransactionGroupMember) => void;
  from?: string;
  to?: string;
};

/**
 * Detalhamento inline dos membros de um agrupamento. Busca o grupo completo
 * (inclusive lancamentos fora do periodo exibido) para manter a leitura fiel.
 */
export function GroupMembersInline({ groupId, onOpenGroup, onOpenTransaction, from, to }: Props) {
  const [group, setGroup] = useState<TransactionGroup | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setError(null);
    getTransactionGroup(groupId)
      .then((g) => {
        if (alive) setGroup(g);
      })
      .catch((e) => {
        if (alive) setError(e instanceof Error ? e.message : "Falha ao carregar agrupamento");
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [groupId]);

  if (loading) {
    return (
      <div className="flex items-center gap-2 px-2 py-2 text-xs text-zinc-500">
        <Loader2 className="h-3.5 w-3.5 animate-spin" /> Carregando membros do agrupamento...
      </div>
    );
  }
  if (error || !group) {
    return <p className="px-2 py-2 text-xs text-red-600">{error ?? "Agrupamento nao encontrado."}</p>;
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="text-xs text-zinc-500">
          <span className="font-semibold text-zinc-700 dark:text-zinc-200">
            {group.member_count} lancamento(s)
          </span>{" "}
          | {group.account_name ?? "Conta bancaria"} | {datePt(group.occurred_at)}
          {group.description ? ` | ${group.description}` : ""}
        </div>
        {onOpenGroup && (
          <Button variant="ghost" size="sm" onClick={() => onOpenGroup(group.id)}>
            <ExternalLink className="h-3.5 w-3.5" /> Abrir grupo
          </Button>
        )}
      </div>
      <div className="overflow-x-auto rounded-lg border border-zinc-200 dark:border-zinc-800">
        <Table>
          <THead>
            <TRow>
              <TH>Data</TH>
              <TH>Conta contabil</TH>
              <TH>Doador</TH>
              <TH>Descricao</TH>
              <TH className="text-center">Anexos</TH>
              <TH className="text-right">Valor</TH>
              {onOpenTransaction && <TH></TH>}
            </TRow>
          </THead>
          <TBody>
            {(group.members ?? []).map((m) => (
              <TRow key={m.id}>
                <TD className="whitespace-nowrap text-zinc-500">
                  {datePt(m.occurred_at)}
                  {((from && m.occurred_at < from) || (to && m.occurred_at > to)) && (
                    <span className="ml-1 text-[10px] font-medium text-amber-600">fora do periodo</span>
                  )}
                </TD>
                <TD>{m.category_name ?? "-"}</TD>
                <TD className="text-zinc-500">{m.donor_name ?? "-"}</TD>
                <TD className="text-zinc-500">{m.description ?? "-"}</TD>
                <TD className="text-center">
                  {m.attachment_count > 0 ? (
                    onOpenTransaction ? (
                      <button
                        type="button"
                        onClick={() => onOpenTransaction(m)}
                        className="inline-flex items-center gap-0.5 text-sky-600 hover:text-sky-700"
                        title={`Ver ${m.attachment_count} anexo(s)`}
                        aria-label={`Ver ${m.attachment_count} anexo(s)`}
                      >
                        <Paperclip className="h-3.5 w-3.5" />
                        <span className="text-[10px] font-medium">{m.attachment_count}</span>
                      </button>
                    ) : (
                      <span className="inline-flex items-center gap-0.5 text-zinc-500">
                        <Paperclip className="h-3.5 w-3.5" />
                        <span className="text-[10px]">{m.attachment_count}</span>
                      </span>
                    )
                  ) : (
                    <span className="text-zinc-300">-</span>
                  )}
                </TD>
                <TD className="text-right font-medium">{currency(m.amount)}</TD>
                {onOpenTransaction && (
                  <TD className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onOpenTransaction(m)}
                      title="Abrir lancamento"
                      aria-label="Abrir lancamento"
                    >
                      <ExternalLink className="h-3.5 w-3.5" />
                    </Button>
                  </TD>
                )}
              </TRow>
            ))}
          </TBody>
        </Table>
      </div>
    </div>
  );
}
