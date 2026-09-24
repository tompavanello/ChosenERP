"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { QRCodeSVG } from "qrcode.react";
import { Church, BellRing, RefreshCw } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Avatar } from "@/components/ui/avatar";
import { SkeletonRows } from "@/components/ui/skeleton";
import { getPublicCard, cardPhotoURL, type PublicCardData } from "@/lib/api";
import { datePt } from "@/lib/format";

export default function MemberAppPage() {
  const params = useParams<{ token: string }>();
  const token = params.token;
  const [data, setData] = useState<PublicCardData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  // Origem do webadmin, para o QR apontar para esta mesma pagina. Lida no
  // efeito (e nao no render) porque o prerender roda sem `window`.
  const [origem, setOrigem] = useState("");

  useEffect(() => setOrigem(window.location.origin), []);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      setData(await getPublicCard(token));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Carteirinha nao encontrada.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col px-4 py-8">
      <header className="mb-6 flex items-center gap-2">
        <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-sky-600 to-sky-400 text-white">
          <Church className="h-5 w-5" />
        </span>
        <div>
          <p className="text-sm font-semibold">Chosen ERP</p>
          <p className="text-xs text-zinc-400">App do Membro</p>
        </div>
      </header>

      {loading ? (
        <SkeletonRows />
      ) : error ? (
        <Card className="p-6 text-center">
          <p className="text-sm text-zinc-500">{error}</p>
          <button onClick={load} className="mt-4 inline-flex items-center gap-2 text-sm text-sky-600"><RefreshCw className="h-4 w-4" /> Tentar novamente</button>
        </Card>
      ) : data ? (
        <>
          <div className="mb-6 rounded-2xl bg-gradient-to-br from-sky-600 to-sky-400 p-6 text-white shadow-lg">
            <div className="mb-4 text-xs uppercase tracking-widest opacity-80">Carteirinha de Membro</div>

            <div className="flex items-center gap-4">
              <Avatar
                name={data.member.full_name}
                src={cardPhotoURL(data.card.token)}
                size="lg"
                className="ring-2 ring-white/70"
              />
              <div className="min-w-0">
                <div className="truncate text-xl font-bold">{data.member.full_name}</div>
                <div className="tnum mt-0.5 font-mono text-sm opacity-90">{data.card.card_ref}</div>
                {data.card.branch_name && (
                  <div className="text-xs opacity-80">{data.card.branch_name}</div>
                )}
              </div>
            </div>

            {/* QR real apontando para esta pagina: quem escaneia abre a
                carteirinha e ve os avisos publicados pela igreja. */}
            <div className="mt-5 flex items-center gap-4 rounded-xl bg-white p-3">
              <div className="shrink-0 rounded-lg bg-white p-1">
                {origem ? (
                  <QRCodeSVG
                    value={`${origem}/member/${data.card.token}`}
                    size={104}
                    level="M"
                    marginSize={0}
                    title="Carteirinha digital da igreja"
                  />
                ) : (
                  <div className="h-[104px] w-[104px] animate-pulse rounded bg-zinc-100" />
                )}
              </div>
              <div className="text-xs text-zinc-500">
                <p className="font-medium text-zinc-700">Carteirinha digital</p>
                <p className="mt-0.5">
                  Aponte a camera para abrir esta carteirinha e receber os avisos da igreja.
                </p>
              </div>
            </div>
          </div>

          <section>
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold text-zinc-700">
              <BellRing className="h-4 w-4" /> Avisos
            </h2>
            {data.announcements.length === 0 ? (
              <p className="text-sm text-zinc-400">Nenhum aviso no momento.</p>
            ) : (
              <div className="space-y-3">
                {data.announcements.map((a) => (
                  <Card key={a.id} className="p-4">
                    <div className="mb-1 flex items-center justify-between gap-2">
                      <p className="font-medium">{a.title}</p>
                      <Badge tone="sky">{datePt(a.published_at)}</Badge>
                    </div>
                    <p className="text-sm text-zinc-600">{a.body}</p>
                  </Card>
                ))}
              </div>
            )}
          </section>
        </>
      ) : null}
    </div>
  );
}
