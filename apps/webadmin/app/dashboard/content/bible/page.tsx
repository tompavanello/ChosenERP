"use client";

import { useEffect, useState } from "react";
import { BookText } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Select, Field } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { getBibleSettings, listBibleVersions, putBibleSettings, type BibleVersion } from "@/lib/api";

export default function BiblePage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("content.write");

  const [loading, setLoading] = useState(true);
  const [configured, setConfigured] = useState(true);
  const [versions, setVersions] = useState<BibleVersion[]>([]);
  const [selected, setSelected] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    Promise.all([listBibleVersions().catch(() => ({ configured: false, versions: [] })), getBibleSettings().catch(() => ({ bible_version_id: "" }))])
      .then(([v, s]) => {
        setConfigured(v.configured);
        setVersions(v.versions ?? []);
        setSelected(s.bible_version_id ?? v.versions?.[0]?.id ?? "");
      })
      .finally(() => setLoading(false));
  }, []);

  async function save() {
    setSaving(true);
    try {
      await putBibleSettings(selected);
      toast("Versão padrão salva.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <SkeletonRows rows={3} />;

  return (
    <div className="page">
      <PageHeader title="Bíblia" description="Leitor da Bíblia no app do membro (api.bible)" />

      <Card className="max-w-xl">
        {!configured ? (
          <div className="space-y-2">
            <p className="flex items-center gap-2 font-medium"><BookText className="h-4 w-4" /> Integração não configurada</p>
            <p className="text-sm text-zinc-500">
              Defina <code>API_BIBLE_KEY</code> no ambiente da API para habilitar o leitor da Bíblia. Sem a chave, a
              tela do app mostra que o recurso não está disponível.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            <Field label="Versão padrão (idioma português)">
              <Select value={selected} onChange={(e) => setSelected(e.target.value)} disabled={!canWrite}>
                {versions.map((v) => (
                  <option key={v.id} value={v.id}>{v.name} ({v.abbreviation})</option>
                ))}
              </Select>
            </Field>
            {canWrite && <Button onClick={save} disabled={saving || !selected}>{saving ? "Salvando..." : "Salvar"}</Button>}
          </div>
        )}
      </Card>
    </div>
  );
}
