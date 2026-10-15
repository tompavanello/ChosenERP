"use client";

import { useCallback, useEffect, useState } from "react";
import { DatabaseBackup, Download, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Field, Input } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  deleteBackup, downloadBackup, getBackupSettings, listAllBackups, listAllTenants,
  updateBackupSettings,
  type BackupSettings, type TenantBackup, type AdminTenant,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

const KIND_LABEL: Record<string, string> = {
  manual: "Manual",
  scheduled: "Automatico",
  pre_restore: "Pre-restauracao",
};

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

export default function PlatformBackupsPage() {
  const { toast } = useToast();
  const [backups, setBackups] = useState<TenantBackup[] | null>(null);
  const [tenants, setTenants] = useState<AdminTenant[]>([]);
  const [settings, setSettings] = useState<BackupSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const [b, t, s] = await Promise.all([listAllBackups(), listAllTenants(), getBackupSettings()]);
      setBackups(b.backups);
      setTenants(t.tenants);
      setSettings(s);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar backups", "error");
      setBackups([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  function tenantName(id: string): string {
    return tenants.find((t) => t.id === id)?.name ?? id.slice(0, 8);
  }

  async function saveSettings(e: React.FormEvent) {
    e.preventDefault();
    if (!settings) return;
    setSaving(true);
    try {
      const s = await updateBackupSettings({
        enabled: settings.enabled,
        interval_hours: settings.interval_hours,
        retention_days: settings.retention_days,
      });
      setSettings(s);
      toast("Rotina atualizada");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar rotina", "error");
    } finally {
      setSaving(false);
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
    if (!window.confirm(`Excluir o backup "${b.filename}"?`)) return;
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

  return (
    <div className="page space-y-4">
      <PageHeader
        title="Backups"
        description="Rotina de backup por igreja e historico dos ultimos pacotes gerados."
      />

      <Card className="max-w-2xl space-y-3">
        <div className="flex items-center gap-2">
          <DatabaseBackup className="h-4 w-4 text-zinc-500" />
          <h3 className="text-sm font-semibold text-zinc-500">Rotina automatica (global)</h3>
        </div>
        {settings === null ? (
          <SkeletonRows rows={2} />
        ) : (
          <form onSubmit={saveSettings} className="space-y-3">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                className="h-4 w-4"
                checked={settings.enabled}
                onChange={(e) => setSettings({ ...settings, enabled: e.target.checked })}
              />
              Gerar backup automatico de todas as igrejas ativas
            </label>
            <div className="grid grid-cols-2 gap-3">
              <Field label="Intervalo (horas)">
                <Input
                  type="number" min={1} className="h-8 text-sm"
                  value={settings.interval_hours}
                  onChange={(e) => setSettings({ ...settings, interval_hours: Number(e.target.value) })}
                />
              </Field>
              <Field label="Retencao (dias)" hint="0 = manter para sempre.">
                <Input
                  type="number" min={0} className="h-8 text-sm"
                  value={settings.retention_days}
                  onChange={(e) => setSettings({ ...settings, retention_days: Number(e.target.value) })}
                />
              </Field>
            </div>
            <div className="flex justify-end">
              <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar rotina"}</Button>
            </div>
          </form>
        )}
      </Card>

      {backups === null ? (
        <SkeletonRows rows={5} />
      ) : backups.length === 0 ? (
        <Card><p className="text-sm text-zinc-400">Nenhum backup gerado ainda.</p></Card>
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <THead>
              <TRow>
                <TH>Igreja</TH><TH>Arquivo</TH><TH>Tipo</TH><TH>Tamanho</TH><TH>Situacao</TH><TH>Criado em</TH><TH></TH>
              </TRow>
            </THead>
            <TBody>
              {backups.map((b) => (
                <TRow key={b.id}>
                  <TD className="font-medium">{tenantName(b.tenant_id)}</TD>
                  <TD className="text-xs text-zinc-500">{b.filename}</TD>
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
    </div>
  );
}
