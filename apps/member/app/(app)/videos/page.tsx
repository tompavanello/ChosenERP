"use client";

import { useEffect, useState } from "react";
import { Play } from "lucide-react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { datePt, getMeVideos, type Video } from "@/lib/api";

export default function VideosPage() {
  const [videos, setVideos] = useState<Video[]>([]);
  const [loading, setLoading] = useState(true);
  const [playing, setPlaying] = useState<Video | null>(null);

  useEffect(() => {
    getMeVideos()
      .then(setVideos)
      .catch(() => setVideos([]))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <SkeletonRows rows={4} />;

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Vídeos</h1>

      {playing && (
        <Card className="overflow-hidden p-0">
          <div className="aspect-video w-full bg-black">
            <iframe
              className="h-full w-full"
              src={`https://www.youtube.com/embed/${playing.youtube_id}?autoplay=1`}
              title={playing.title}
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
              allowFullScreen
            />
          </div>
          <div className="p-3">
            <p className="font-medium">{playing.title}</p>
            {playing.speaker && <p className="text-sm text-[var(--muted)]">{playing.speaker}</p>}
          </div>
        </Card>
      )}

      {videos.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Nenhum vídeo publicado.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {videos.map((v) => (
            <button key={v.id} className="w-full text-left" onClick={() => setPlaying(v)}>
              <Card className="flex items-center gap-3 p-2">
                <span className="relative h-16 w-28 shrink-0 overflow-hidden rounded-lg bg-black/5">
                  {v.thumbnail_url && (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={v.thumbnail_url} alt="" className="h-full w-full object-cover" />
                  )}
                  <span className="absolute inset-0 flex items-center justify-center">
                    <Play className="h-6 w-6 text-white drop-shadow" />
                  </span>
                </span>
                <span className="min-w-0">
                  <span className="block truncate font-medium">{v.title}</span>
                  <span className="block text-xs text-[var(--muted)]">
                    {v.speaker ? `${v.speaker} · ` : ""}
                    {datePt(v.recorded_at)}
                  </span>
                </span>
              </Card>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
