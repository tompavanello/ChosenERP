"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { BookOpen, Compass, Loader2, Search, Sparkles } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useHelp } from "@/components/help/help-provider";
import { getHelpArticles, type HelpSummary } from "@/lib/api";

export default function HelpCenterPage() {
  const { openHelp, openTours } = useHelp();
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("");
  const [data, setData] = useState<{ articles: HelpSummary[]; categories: string[]; ai_enabled: boolean } | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    const timer = setTimeout(() => {
      getHelpArticles({ app: "webadmin", q: query.trim(), category })
        .then((res) => alive && setData(res))
        .catch(() => {})
        .finally(() => alive && setLoading(false));
    }, 200);
    return () => { alive = false; clearTimeout(timer); };
  }, [query, category]);

  const grouped = (data?.articles ?? []).reduce<Record<string, HelpSummary[]>>((acc, a) => {
    (acc[a.category] ??= []).push(a);
    return acc;
  }, {});

  return (
    <div className="page space-y-6">
      <PageHeader
        title="Central de ajuda"
        description="Manual do usuario, guias por tela e suporte. Busque pelo que precisa ou pergunte a IA."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" onClick={openTours}>
              <Compass className="h-4 w-4" /> Roteiros guiados
            </Button>
            <Button variant="outline" onClick={() => openHelp("ai")}>
              <Sparkles className="h-4 w-4" /> Perguntar a IA
            </Button>
          </div>
        }
      />

      <Card className="p-4">
        <div className="flex flex-col gap-3 sm:flex-row">
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
            <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Buscar no manual (ex.: conciliacao, membros, escalas...)" className="pl-9" />
          </div>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="select h-9 min-w-48"
          >
            <option value="">Todas as categorias</option>
            {(data?.categories ?? []).map((c) => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
        <p className="mt-2 text-xs text-zinc-400">
          {data?.articles.length ?? 0} artigo(s) visiveis para o seu perfil e plano.
          {data && !data.ai_enabled && " A IA da ajuda nao esta configurada."}
        </p>
      </Card>

      {loading ? (
        <Card className="p-4"><SkeletonRows rows={6} /></Card>
      ) : Object.keys(grouped).length === 0 ? (
        <Card className="p-8 text-center">
          <BookOpen className="mx-auto h-8 w-8 text-zinc-300" />
          <p className="mt-2 text-sm text-zinc-500">Nenhum artigo encontrado.</p>
          <Button variant="outline" className="mt-3" onClick={() => openHelp("tickets")}>Abrir um chamado</Button>
        </Card>
      ) : (
        <div className="space-y-5">
          {Object.entries(grouped).map(([cat, items]) => (
            <section key={cat}>
              <h3 className="mb-2 text-sm font-semibold text-zinc-700 dark:text-zinc-200">{cat}</h3>
              <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                {items.map((a) => (
                  <Link key={a.slug} href={`/dashboard/help/${a.slug}`}>
                    <Card className="h-full p-4 transition hover:border-sky-200 hover:shadow-md">
                      <div className="mb-2 flex items-start justify-between gap-2">
                        <h4 className="text-sm font-semibold">{a.title}</h4>
                        <Badge tone="zinc">{a.category}</Badge>
                      </div>
                      <p className="line-clamp-3 text-xs text-zinc-500">{a.summary}</p>
                    </Card>
                  </Link>
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  );
}
