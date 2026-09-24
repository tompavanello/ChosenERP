"use client";

import { useCallback, useEffect, useState } from "react";
import { Download, FileText, Plus, ShieldCheck, ShieldOff, Trash2 } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Field, Input, Textarea } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  listConsentTerms, createConsentTerm, listMemberConsents, recordConsent,
  anonymizeMember, exportMemberData,
  type ConsentTerm, type ConsentRecord,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

export function LgpdSection({
  memberId,
  canWrite,
  isAdmin,
}: {
  memberId: string;
  canWrite: boolean;
  isAdmin: boolean;
}) {
  const { toast } = useToast();
  const [terms, setTerms] = useState<ConsentTerm[]>([]);
  const [consents, setConsents] = useState<ConsentRecord[] | null>(null);
  const [showTermForm, setShowTermForm] = useState(false);
  const [termForm, setTermForm] = useState({ title: "", body: "" });

  const load = useCallback(async () => {
    const [t, c] = await Promise.all([listConsentTerms(), listMemberConsents(memberId)]);
    setTerms(t.terms);
    setConsents(c.consents);
  }, [memberId]);

  useEffect(() => {
    load().catch(() => toast("Erro ao carregar dados de LGPD", "error"));
  }, [load, toast]);

  // Ultimo consentimento por termo (o mais recente manda).
  const latest = new Map<string, ConsentRecord>();
  for (const c of consents ?? []) if (!latest.has(c.term_id)) latest.set(c.term_id, c);

  async function toggleConsent(term: ConsentTerm, consented: boolean) {
    try {
      await recordConsent(memberId, { term_id: term.id, consented });
      await load();
      toast(consented ? "Consentimento registrado." : "Consentimento revogado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function submitTerm(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createConsentTerm(termForm);
      setTermForm({ title: "", body: "" });
      setShowTermForm(false);
      await load();
      toast("Termo criado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function doExport() {
    try {
      await exportMemberData(memberId);
      toast("Exportacao gerada.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao exportar", "error");
    }
  }

  async function doAnonymize() {
    if (!confirm("Anonimizar os dados pessoais deste membro? Esta acao nao pode ser desfeita (LGPD).")) return;
    try {
      await anonymizeMember(memberId);
      toast("Membro anonimizado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="space-y-4">
      <Card className="flex flex-wrap items-center justify-between gap-3 p-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sky-50 text-sky-600">
            <FileText className="h-5 w-5" />
          </div>
          <div>
            <p className="text-sm font-semibold">Portabilidade dos dados (LGPD art. 18)</p>
            <p className="text-xs text-zinc-500">Exporta em JSON tudo que o sistema guarda sobre este titular.</p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={doExport}><Download className="h-4 w-4" /> Exportar dados</Button>
          {isAdmin && (
            <Button variant="outline" onClick={doAnonymize}><Trash2 className="h-4 w-4" /> Anonimizar</Button>
          )}
        </div>
      </Card>

      <Card className="p-4">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="flex items-center gap-2 text-sm font-semibold"><ShieldCheck className="h-4 w-4 text-sky-600" /> Termos de consentimento</h3>
          {isAdmin && <Button size="sm" variant="outline" onClick={() => setShowTermForm((v) => !v)}><Plus className="h-4 w-4" /> Novo termo</Button>}
        </div>

        {showTermForm && (
          <form onSubmit={submitTerm} className="mb-4 grid grid-cols-1 gap-2 rounded border border-zinc-200 p-3 sm:grid-cols-2 dark:border-zinc-700">
            <Field label="Titulo *" className="sm:col-span-2">
              <Input required className="h-8 text-sm" value={termForm.title} onChange={(e) => setTermForm({ ...termForm, title: e.target.value })} />
            </Field>
            <Field label="Texto do termo *" className="sm:col-span-2">
              <Textarea required rows={3} className="text-sm" value={termForm.body} onChange={(e) => setTermForm({ ...termForm, body: e.target.value })} />
            </Field>
            <div className="sm:col-span-2">
              <Button type="submit" className="h-8 text-sm">Publicar termo</Button>
            </div>
          </form>
        )}

        {consents === null ? (
          <SkeletonRows rows={3} />
        ) : terms.length === 0 ? (
          <EmptyState icon={<ShieldCheck className="h-8 w-8" />} title="Nenhum termo" description="Cadastre um termo de consentimento." />
        ) : (
          <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {terms.map((t) => {
              const c = latest.get(t.id);
              const accepted = c?.consented ?? false;
              return (
                <li key={t.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                  <div className="min-w-0">
                    <p className="text-sm font-medium">{t.title} <span className="text-xs text-zinc-400">v{t.version}</span></p>
                    <p className="truncate text-xs text-zinc-400">{t.body}</p>
                    {c && <p className="text-xs text-zinc-400">{accepted ? "aceito" : "recusado"} em {dateTimePt(c.consented_at ?? c.created_at)}</p>}
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge tone={accepted ? "green" : "zinc"}>{accepted ? "Consentido" : "Pendente"}</Badge>
                    {canWrite && (accepted ? (
                      <Button size="sm" variant="ghost" onClick={() => toggleConsent(t, false)}><ShieldOff className="h-4 w-4" /> Revogar</Button>
                    ) : (
                      <Button size="sm" onClick={() => toggleConsent(t, true)}><ShieldCheck className="h-4 w-4" /> Registrar</Button>
                    ))}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </Card>
    </div>
  );
}
