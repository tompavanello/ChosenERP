"use client";

import { useEffect, useState } from "react";
import { BookOpen, Download, ExternalLink, FileText, Link2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows } from "@/components/ui/skeleton";
import { datePt, downloadMaterial, getMeMaterials, type StudyMaterial } from "@/lib/api";

function fileSize(n?: number | null): string {
  if (!n) return "";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export default function MateriaisPage() {
  const [items, setItems] = useState<StudyMaterial[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getMeMaterials()
      .then(setItems)
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, []);

  async function download(m: StudyMaterial) {
    setBusy(m.id);
    setError(null);
    try {
      await downloadMaterial(m.id, m.file_name ?? "material");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Não foi possível baixar.");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Materiais</h1>

      {error && <p className="text-sm text-red-600">{error}</p>}

      {loading ? (
        <SkeletonRows rows={4} />
      ) : items.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Nenhum material disponível no momento.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {items.map((m) => (
            <Card key={m.id}>
              <div className="flex items-start gap-3">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-[var(--brand-soft)] text-[var(--brand-strong)]">
                  {m.kind === "link" ? <Link2 className="h-5 w-5" /> : <FileText className="h-5 w-5" />}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="font-medium">{m.title}</p>
                  {m.description && <p className="mt-0.5 text-sm text-[var(--muted)]">{m.description}</p>}
                  <div className="mt-1 flex flex-wrap items-center gap-2">
                    <Badge>{m.group_name ?? "Igreja"}</Badge>
                    <span className="text-xs text-[var(--muted)]">{datePt(m.created_at)}</span>
                    {m.kind === "file" && m.file_size ? (
                      <span className="text-xs text-[var(--muted)]">{fileSize(m.file_size)}</span>
                    ) : null}
                  </div>
                </div>
              </div>

              <div className="mt-3">
                {m.kind === "link" ? (
                  <a
                    href={m.url ?? "#"}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-[var(--brand)] px-4 py-2.5 text-sm font-medium text-white"
                  >
                    <ExternalLink className="h-4 w-4" /> Abrir link
                  </a>
                ) : (
                  <Button type="button" className="w-full" disabled={busy === m.id} onClick={() => download(m)}>
                    <Download className="h-4 w-4" /> {busy === m.id ? "Baixando..." : "Baixar material"}
                  </Button>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}

      <p className="flex items-center justify-center gap-1.5 text-center text-xs text-[var(--muted)]">
        <BookOpen className="h-3.5 w-3.5" /> Materiais da igreja e dos seus grupos.
      </p>
    </div>
  );
}
