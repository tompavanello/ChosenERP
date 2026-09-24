"use client";

import { useEffect, useState } from "react";
import { QrCode, Printer, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/toast";
import { Dropdown } from "@/components/ui/dropdown";
import { issueCard, getCard, cardPrintURL } from "@/lib/api";

/**
 * CardCell mostra a carteirinha digital ja no grid: numero em monospace quando
 * existe, botao "Emitir" quando nao existe.
 *
 * A listagem de membros devolve so o card_ref (o qr_token NAO vem na lista de
 * proposito: /public/card/{token} e um endpoint sem autenticacao, entao expor N
 * tokens numa listagem seria abrir N URLs permanentes de uma vez). O token e
 * buscado sob demanda, ao abrir a carteirinha de um membro especifico.
 *
 * A emissao e idempotente no backend, entao clicar duas vezes nao gera uma
 * segunda carteirinha com outro numero.
 */
export function CardCell({
  memberId,
  cardRef,
  onIssued,
}: {
  memberId: string;
  cardRef?: string;
  onIssued?: (ref: string) => void;
}) {
  const { toast } = useToast();
  const [ref, setRef] = useState(cardRef);
  const [token, setToken] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => setRef(cardRef), [cardRef]);

  async function emitir() {
    setBusy(true);
    try {
      const res = await issueCard(memberId);
      setRef(res.card_ref);
      setToken(res.token);
      onIssued?.(res.card_ref);
      toast(`Carteirinha ${res.card_ref} emitida.`);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Falha ao emitir a carteirinha", "error");
    } finally {
      setBusy(false);
    }
  }

  async function comToken(): Promise<string | null> {
    if (token) return token;
    try {
      const res = await getCard(memberId);
      setToken(res.token);
      return res.token;
    } catch (e) {
      toast(e instanceof Error ? e.message : "Carteirinha nao encontrada", "error");
      return null;
    }
  }

  async function abrir() {
    const t = await comToken();
    if (t) window.open(`/member/${t}`, "_blank", "noopener");
  }

  async function imprimir() {
    const t = await comToken();
    if (t) window.open(cardPrintURL(t), "_blank", "noopener");
  }

  if (!ref) {
    return (
      <Button variant="outline" size="sm" className="h-7" onClick={emitir} disabled={busy}>
        {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <QrCode className="h-3.5 w-3.5" />}
        Emitir
      </Button>
    );
  }

  return (
    <div className="flex items-center gap-1.5">
      <span className="tnum font-mono text-xs font-medium tracking-tight text-zinc-700 dark:text-zinc-300">
        {ref}
      </span>
      <Dropdown
        trigger={<QrCode className="h-3.5 w-3.5" />}
        items={[
          { label: "Ver carteirinha", icon: <QrCode className="h-3.5 w-3.5" />, onClick: abrir },
          { label: "Imprimir", icon: <Printer className="h-3.5 w-3.5" />, onClick: imprimir },
        ]}
      />
    </div>
  );
}
