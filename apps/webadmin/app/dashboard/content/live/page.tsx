"use client";

import { useEffect, useState } from "react";
import { Radio } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { getLiveSettings, putLiveSettings, type LiveSettings } from "@/lib/api";

export default function LivePage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("content.write");

  const [settings, setSettings] = useState<LiveSettings | null>(null);
  const [channel, setChannel] = useState("");
  const [url, setUrl] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    getLiveSettings()
      .then((s) => {
        setSettings(s);
        setChannel(s.youtube_channel_id ?? "");
        setUrl(s.live_stream_url ?? "");
      })
      .catch((e) => toast(e instanceof Error ? e.message : "Erro", "error"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const s = await putLiveSettings({ youtube_channel_id: channel, live_stream_url: url });
      setSettings(s);
      toast("Configuração salva.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setSaving(false);
    }
  }

  if (settings === null) return <SkeletonRows rows={4} />;

  return (
    <div className="page">
      <PageHeader title="Culto ao vivo" description="Canal do YouTube exibido no app do membro" />

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <form onSubmit={save} className="space-y-4">
            <Field label="ID do canal do YouTube">
              <Input
                value={channel}
                onChange={(e) => setChannel(e.target.value)}
                placeholder="UCxxxxxxxxxxxxxxxxxxxxxx"
                disabled={!canWrite}
              />
            </Field>
            <p className="-mt-2 text-xs text-zinc-500">
              Com o ID do canal, o app embute automaticamente a transmissão ao vivo quando houver.
            </p>
            <Field label="URL da live (opcional)">
              <Input
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://youtube.com/@igreja/live"
                disabled={!canWrite}
              />
            </Field>
            {canWrite && <Button type="submit" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>}
          </form>
        </Card>

        <Card>
          <p className="mb-2 flex items-center gap-2 text-sm font-medium">
            <Radio className="h-4 w-4 text-[var(--brand,theme(colors.sky.500))]" /> Prévia
          </p>
          {settings.embed_url ? (
            <div className="aspect-video w-full overflow-hidden rounded-lg bg-black">
              <iframe className="h-full w-full" src={settings.embed_url} title="Prévia" allowFullScreen />
            </div>
          ) : (
            <p className="text-sm text-zinc-500">
              Informe o ID do canal para ver a prévia. Sem ele, o app mostrará apenas o botão para abrir no YouTube.
            </p>
          )}
        </Card>
      </div>
    </div>
  );
}
