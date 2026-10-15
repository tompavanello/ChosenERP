"use client";

import { useEffect, useState } from "react";
import { ExternalLink, Radio } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { getMeLive, type LiveSettings } from "@/lib/api";

export default function AoVivoPage() {
  const [live, setLive] = useState<LiveSettings | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMeLive()
      .then(setLive)
      .catch(() => setLive(null))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <SkeletonRows rows={3} />;

  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-lg font-bold">
        <Radio className="h-5 w-5 text-[var(--brand)]" /> Culto ao vivo
      </h1>

      {!live?.configured ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">
            A transmissão ainda não foi configurada pela igreja.
          </p>
        </Card>
      ) : (
        <>
          {live.embed_url ? (
            <Card className="overflow-hidden p-0">
              <div className="aspect-video w-full bg-black">
                <iframe
                  className="h-full w-full"
                  src={live.embed_url}
                  title="Culto ao vivo"
                  allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                  allowFullScreen
                />
              </div>
            </Card>
          ) : (
            <Card>
              <p className="text-sm text-[var(--muted)]">
                Toque no botão abaixo para assistir à transmissão no YouTube.
              </p>
            </Card>
          )}
          {live.watch_url && (
            <a href={live.watch_url} target="_blank" rel="noreferrer">
              <Button className="w-full">
                <ExternalLink className="h-4 w-4" /> Abrir no YouTube
              </Button>
            </a>
          )}
        </>
      )}
    </div>
  );
}
