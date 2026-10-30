"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Download, FileText, Loader2, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import {
  downloadFinancialAttachment,
  listAttachments,
  listGroupAttachments,
  uploadGroupAttachment,
  type FinancialAttachment,
} from "@/lib/api";

export type AttachmentOwner = {
  transactionId?: string;
  groupId?: string;
};

type AttachmentListProps = AttachmentOwner & {
  canUpload?: boolean;
  onUploaded?: () => void;
  showSource?: boolean;
};

function formatSize(size: number) {
  if (!size) return "";
  if (size < 1024) return `${size} B`;
  return `${(size / 1024).toFixed(1)} KB`;
}

export function AttachmentList({ transactionId, groupId, canUpload = false, onUploaded, showSource = false }: AttachmentListProps) {
  const { toast } = useToast();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [items, setItems] = useState<FinancialAttachment[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);

  const load = useCallback(async () => {
    if (!transactionId && !groupId) {
      setItems([]);
      return;
    }
    setLoading(true);
    try {
      const res = groupId ? await listGroupAttachments(groupId) : await listAttachments(transactionId!);
      setItems(Array.isArray(res.attachments) ? res.attachments : []);
    } catch (e) {
      setItems([]);
      toast(e instanceof Error ? e.message : "Falha ao carregar anexos", "error");
    } finally {
      setLoading(false);
    }
  }, [transactionId, groupId, toast]);

  useEffect(() => { load(); }, [load]);

  async function upload(file: File) {
    if (!groupId) return;
    setUploading(true);
    try {
      await uploadGroupAttachment(groupId, file);
      toast("Anexo enviado.", "success");
      await load();
      onUploaded?.();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Falha ao enviar anexo", "error");
    } finally {
      setUploading(false);
    }
  }

  async function download(att: FinancialAttachment) {
    try {
      await downloadFinancialAttachment(att.file_url, att.file_name);
    } catch {
      toast("Nao foi possivel abrir o anexo", "error");
    }
  }

  return (
    <div className="space-y-2">
      {loading ? (
        <p className="flex items-center gap-2 text-xs text-zinc-400">
          <Loader2 className="h-3.5 w-3.5 animate-spin" /> Carregando anexos...
        </p>
      ) : items.length === 0 ? (
        <p className="text-xs text-zinc-400">Nenhum anexo.</p>
      ) : (
        <div className="space-y-2">
          {items.map((att) => (
            <div key={att.id} className="flex items-center justify-between gap-2 rounded border border-zinc-200 p-2 dark:border-zinc-700">
              <div className="flex min-w-0 items-center gap-2">
                <FileText className="h-4 w-4 shrink-0 text-zinc-500" />
                <div className="min-w-0">
                  <p className="truncate text-sm">{att.file_name}</p>
                  <p className="text-xs text-zinc-400">
                    {[att.content_type ?? "arquivo", formatSize(att.file_size)].filter(Boolean).join(" - ")}
                  </p>
                </div>
                {showSource && (
                  <Badge tone={att.group_id ? "indigo" : "zinc"} className="shrink-0 text-[10px]">
                    {att.group_id ? "Agrupamento" : "Lancamento"}
                  </Badge>
                )}
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => download(att)}
                title="Baixar anexo"
                aria-label={`Baixar anexo ${att.file_name}`}
              >
                <Download className="h-3.5 w-3.5" />
              </Button>
            </div>
          ))}
        </div>
      )}

      {groupId && canUpload && (
        <div>
          <input
            ref={inputRef}
            type="file"
            accept="image/*,.pdf,.doc,.docx"
            className="hidden"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) upload(file);
              e.target.value = "";
            }}
          />
          <Button
            variant="outline"
            size="sm"
            onClick={() => inputRef.current?.click()}
            disabled={uploading}
          >
            <Upload className="h-4 w-4" /> {uploading ? "Enviando..." : "Anexar ao agrupamento"}
          </Button>
        </div>
      )}
    </div>
  );
}

type AttachmentDrawerProps = {
  open: boolean;
  onClose: () => void;
  title?: string;
  owner: AttachmentOwner;
  canUpload?: boolean;
  onUploaded?: () => void;
};

export function AttachmentDrawer({ open, onClose, title = "Anexos", owner, canUpload, onUploaded }: AttachmentDrawerProps) {
  return (
    <Drawer open={open} onClose={onClose} title={title}>
      <AttachmentList
        transactionId={owner.transactionId}
        groupId={owner.groupId}
        canUpload={canUpload}
        onUploaded={onUploaded}
        showSource={!!owner.groupId}
      />
    </Drawer>
  );
}
