"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { AlertTriangle, DatabaseBackup, Download, RotateCcw, Trash2, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  createTenantBackup, createTenantBackupDownload, deleteBackup, downloadBackup,
  listTenantBackups, restoreTenantBackup,
  type TenantBackup,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

function formatBytes(n: number): string {
  if (!n) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

const KIND_LABEL: Record<string, string> = {
  manual: "Manual",
  scheduled: "Automatico",
  pre_restore: "Pre-restauracao",
};

/**
 * Backup e restauracao dos dados completos de uma igreja (console da
 * plataforma). O restore e destrutivo e exige confirmacao pelo slug.
 */
export function TenantBackups({ tenantId, tenantSlug }: { tenantId: string; tenantSlug: string }) {
  const { toast } = useToast();
  const [backups, setBackups] = useState<TenantBackup[] | null>(null);
  const [creating, setCreating] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);

  const [restore, setRestore] = useState<{ open: boolean; source: "existing" | "upload"; backupId: string; file: File | null; confirm: string }>({
    open: false, source: "existing", backupId: "", file: null, confirm: "",
  });
  const [restoring, setRestoring] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    try {
      const r = await listTenantBackups(tenantId);
      setBackups(r.backups);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar backups", "error");
      setBackups([]);
    }
  }, [tenantId, toast]);

  useEffect(() => { load(); }, [load]);

  async function backupNow(download: boolean) {
    setCreating(true);
    try {
      if (download) {
        const ts = new Date().toISOString().slice(0, 19).replace(/[:T]/g, "-");
        await createTenantBackupDownload(tenantId, `${tenantSlug}_${ts}.zip`);
        toast("Backup gerado e baixado");
      } else {
        await createTenantBackup(tenantId);
        toast("Backup gerado");
      }
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao gerar backup", "error");
    } finally {
      setCreating(false);
    }
  }

  async function download(b: TenantBackup) {
    setBusyId(b.id);
    try {
      await downloadBackup(b.id, b.filename);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao baixar backup", "error");
    } finally {
      setBusyId(null);
    }
  }

  async function remove(b: TenantBackup) {
    if (!window.confirm(`Excluir o backup "${b.filename}"? Esta acao nao pode ser desfeita.`)) return;
    setBusyId(b.id);
    try {
      await deleteBackup(b.id);
      toast("Backup excluido");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir backup", "error");
    } finally {
      setBusyId(null);
    }
  }

  function openRestore() {
    const ready = (backups ?? []).filter((b) => b.status === "ready");
    setRestore({ open: true, source: ready.length ? "existing" : "upload", backupId: ready[0]?.id ?? "", file: null, confirm: "" });
  }

  async function confirmRestore() {
    if (restore.confirm.trim().toLowerCase() !== tenantSlug.toLowerCase()) {
      toast("Informe o slug da igreja para confirmar", "error");
      return;
    }
    if (restore.source === "existing" && !restore.backupId) {
      toast("Selecione um backup", "error");
      return;
    }
    if (restore.source === "upload" && !restore.file) {
      toast("Selecione o arquivo .zip", "error");
      return;
    }
    setRestoring(true);
    try {
      const res = await restoreTenantBackup(tenantId, {
        confirm: restore.confirm.trim(),
        backupId: restore.source === "existing" ? restore.backupId : undefined,
        file: restore.source === "upload" ? restore.file ?? undefined : undefined,
      });
      toast(`Restauracao concluida (${res.restore.files} arquivo(s))`);
      setRestore((s) => ({ ...s, open: false }));
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao restaurar", "error");
    } finally {
      setRestoring(false);
    }
  }

  const ready = (backups ?? []).filter((b) => b.status === "ready");

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-zinc-500">Backups da igreja</h3>
          <p className="text-xs text-zinc-400">Pacote completo (dados + arquivos). A restauracao substitui todo o conteudo da igreja.</p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={creating} onClick={() => backupNow(true)}>
            <Download className="h-3.5 w-3.5" /> {creating ? "Gerando..." : "Backup + baixar"}
          </Button>
          <Button size="sm" disabled={creating} onClick={() => backupNow(false)}>
            <DatabaseBackup className="h-3.5 w-3.5" /> Backup agora
          </Button>
        </div>
      </div>

      {backups === null ? (
        <SkeletonRows rows={3} />
      ) : backups.length === 0 ? (
        <Card><p className="text-sm text-zinc-400">Nenhum backup gerado ainda.</p></Card>
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <THead>
              <TRow>
                <TH>Arquivo</TH><TH>Tipo</TH><TH>Tamanho</TH><TH>Situacao</TH><TH>Criado em</TH><TH></TH>
              </TRow>
            </THead>
            <TBody>
              {backups.map((b) => (
                <TRow key={b.id}>
                  <TD className="font-medium">{b.filename}</TD>
                  <TD><Badge tone={b.kind === "pre_restore" ? "amber" : b.kind === "scheduled" ? "sky" : "zinc"}>{KIND_LABEL[b.kind] ?? b.kind}</Badge></TD>
                  <TD className="text-xs text-zinc-500">{formatBytes(b.size_bytes)}</TD>
                  <TD>
                    <Badge tone={b.status === "ready" ? "green" : b.status === "failed" ? "red" : "amber"}>
                      {b.status === "ready" ? "Pronto" : b.status === "failed" ? "Falhou" : "Gerando"}
                    </Badge>
                  </TD>
                  <TD className="text-xs text-zinc-400">{dateTimePt(b.created_at)}</TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button variant="outline" size="sm" title="Baixar" disabled={busyId === b.id || b.status !== "ready"} onClick={() => download(b)}>
                        <Download className="h-3.5 w-3.5" />
                      </Button>
                      <Button variant="outline" size="sm" title="Excluir" disabled={busyId === b.id} onClick={() => remove(b)}>
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </Card>
      )}

      <div className="flex justify-end">
        <Button variant="outline" size="sm" onClick={openRestore} disabled={ready.length === 0 && !backups}>
          <RotateCcw className="h-3.5 w-3.5" /> Restaurar backup
        </Button>
      </div>

      <Drawer open={restore.open} onClose={() => setRestore((s) => ({ ...s, open: false }))} title="Restaurar backup">
        <div className="space-y-3">
          <div className="flex items-start gap-2 rounded border border-amber-300 bg-amber-50 p-2 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>A restauracao SUBSTITUI todos os dados e arquivos atuais desta igreja. Um backup de seguranca e gerado automaticamente antes.</span>
          </div>

          <Field label="Origem">
            <Select className="h-8 text-sm" value={restore.source} onChange={(e) => setRestore((s) => ({ ...s, source: e.target.value as "existing" | "upload" }))}>
              <option value="existing" disabled={ready.length === 0}>Backup existente</option>
              <option value="upload">Enviar arquivo .zip</option>
            </Select>
          </Field>

          {restore.source === "existing" ? (
            <Field label="Backup *">
              <Select className="h-8 text-sm" value={restore.backupId} onChange={(e) => setRestore((s) => ({ ...s, backupId: e.target.value }))}>
                {ready.map((b) => (
                  <option key={b.id} value={b.id}>{b.filename} - {dateTimePt(b.created_at)}</option>
                ))}
              </Select>
            </Field>
          ) : (
            <Field label="Arquivo .zip *">
              <label className="flex cursor-pointer items-center gap-2 rounded border border-dashed border-zinc-300 px-3 py-2 text-sm dark:border-zinc-700">
                <Upload className="h-4 w-4" />
                <span className="truncate">{restore.file ? restore.file.name : "Escolher arquivo"}</span>
                <input
                  ref={fileRef}
                  type="file"
                  accept=".zip,application/zip"
                  className="hidden"
                  onChange={(e) => setRestore((s) => ({ ...s, file: e.target.files?.[0] ?? null }))}
                />
              </label>
            </Field>
          )}

          <Field label={`Confirme digitando "${tenantSlug}"`} hint="Evita restauracao acidental.">
            <Input className="h-8 text-sm" value={restore.confirm} onChange={(e) => setRestore((s) => ({ ...s, confirm: e.target.value }))} />
          </Field>

          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setRestore((s) => ({ ...s, open: false }))}>Cancelar</Button>
            <Button type="button" className="h-8 text-sm" disabled={restoring} onClick={confirmRestore}>
              <RotateCcw className="h-3.5 w-3.5" /> {restoring ? "Restaurando..." : "Restaurar"}
            </Button>
          </div>
        </div>
      </Drawer>
    </div>
  );
}
