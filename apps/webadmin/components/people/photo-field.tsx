"use client";

import { useRef, useState } from "react";
import { Camera, Trash2 } from "lucide-react";
import { Avatar } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/toast";
import { uploadMemberPhoto, deleteMemberPhoto, assetURL } from "@/lib/api";

/**
 * PhotoField faz o upload real da foto do membro (nao um campo de texto com URL).
 *
 * So aparece quando o membro ja existe: o endpoint e POST /members/{id}/photo e
 * nao haveria id na criacao. A URL e montada pelo servidor - o cliente nunca
 * escolhe o caminho do arquivo, o que evita apontar a foto para outro host ou
 * para arquivo de outro tenant.
 */
export function PhotoField({
  memberId,
  name,
  photoUrl,
  onChange,
}: {
  memberId: string;
  name: string;
  photoUrl?: string | null;
  /** Recebe a nova URL (ou undefined ao remover) para o pai atualizar o estado. */
  onChange: (url?: string) => void;
}) {
  const { toast } = useToast();
  const inputRef = useRef<HTMLInputElement>(null);
  const [enviando, setEnviando] = useState(false);
  const [previa, setPrevia] = useState<string | null>(null);

  async function enviar(file: File) {
    setEnviando(true);
    try {
      const res = await uploadMemberPhoto(memberId, file);
      onChange(res.photo_url);
      setPrevia(null);
      toast("Foto atualizada.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Falha no upload da foto", "error");
    } finally {
      setEnviando(false);
    }
  }

  async function remover() {
    setEnviando(true);
    try {
      await deleteMemberPhoto(memberId);
      onChange(undefined);
      toast("Foto removida.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Falha ao remover a foto", "error");
    } finally {
      setEnviando(false);
    }
  }

  const atual = previa ?? assetURL(photoUrl);

  return (
    <div className="flex items-center gap-3">
      <Avatar name={name || "?"} src={atual} size="lg" />
      <div className="space-y-1.5">
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={enviando}
            onClick={() => inputRef.current?.click()}
          >
            <Camera className="h-4 w-4" /> {enviando ? "Enviando..." : "Trocar foto"}
          </Button>
          {photoUrl && (
            <Button type="button" variant="ghost" size="sm" disabled={enviando} onClick={remover}>
              <Trash2 className="h-4 w-4" /> Remover
            </Button>
          )}
        </div>
        <p className="text-xs text-zinc-400">JPG, PNG ou WebP. A foto aparece no grid e na carteirinha.</p>
      </div>

      <input
        ref={inputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          if (file) {
            setPrevia(URL.createObjectURL(file));
            enviar(file);
          }
          e.target.value = "";
        }}
      />
    </div>
  );
}
