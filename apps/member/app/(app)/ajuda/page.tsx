"use client";

import { useEffect, useState } from "react";
import { ArrowLeft, BookOpen, Loader2, Search, Sparkles, Ticket } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Markdown } from "@/components/help/markdown";
import { useHelp } from "@/components/help/help-provider";
import { getHelpArticle, getHelpArticles, type HelpArticle, type HelpSummary } from "@/lib/api";

export default function AjudaPage() {
  const { openHelp } = useHelp();
  const [query, setQuery] = useState("");
  const [items, setItems] = useState<HelpSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [article, setArticle] = useState<HelpArticle | null>(null);
  const [articleLoading, setArticleLoading] = useState(false);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    const timer = setTimeout(() => {
      getHelpArticles({ app: "member", q: query.trim() })
        .then((res) => alive && setItems(res.articles))
        .catch(() => {})
        .finally(() => alive && setLoading(false));
    }, 200);
    return () => { alive = false; clearTimeout(timer); };
  }, [query]);

  async function open(slug: string) {
    setArticleLoading(true);
    try {
      setArticle(await getHelpArticle(slug, "member"));
    } catch {
      setArticle(null);
    } finally {
      setArticleLoading(false);
    }
  }

  if (article) {
    return (
      <div className="space-y-4">
        <button onClick={() => setArticle(null)} className="inline-flex items-center gap-1 text-xs text-[var(--muted)]">
          <ArrowLeft className="h-3.5 w-3.5" /> Voltar para a ajuda
        </button>
        {articleLoading ? (
          <Card><p className="text-sm text-[var(--muted)]">Carregando...</p></Card>
        ) : (
          <Card>
            <Markdown>{article.body}</Markdown>
            <div className="mt-4 flex gap-2 border-t border-[var(--line)] pt-3">
              <Button size="sm" onClick={() => openHelp("ai")}><Sparkles className="h-3.5 w-3.5" /> Perguntar a IA</Button>
              <Button size="sm" variant="outline" onClick={() => openHelp("tickets")}><Ticket className="h-3.5 w-3.5" /> Abrir chamado</Button>
            </div>
          </Card>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-lg font-bold">Ajuda</h1>
        <p className="text-sm text-[var(--muted)]">Manual do app, guias e suporte.</p>
      </div>

      <div className="flex gap-2">
        <Button size="sm" onClick={() => openHelp("ai")}><Sparkles className="h-3.5 w-3.5" /> IA</Button>
        <Button size="sm" variant="outline" onClick={() => openHelp("tickets")}><Ticket className="h-3.5 w-3.5" /> Chamados</Button>
      </div>

      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--muted)]" />
        <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Buscar no manual..." className="pl-9" />
      </div>

      {loading ? (
        <Card><p className="flex items-center gap-2 text-sm text-[var(--muted)]"><Loader2 className="h-4 w-4 animate-spin" /> Carregando...</p></Card>
      ) : items.length === 0 ? (
        <Card className="text-center">
          <BookOpen className="mx-auto h-7 w-7 text-[var(--muted)]" />
          <p className="mt-2 text-sm text-[var(--muted)]">Nenhum artigo encontrado.</p>
          <Button size="sm" variant="outline" className="mt-3" onClick={() => openHelp("tickets")}>Abrir chamado</Button>
        </Card>
      ) : (
        <div className="space-y-2">
          {items.map((a) => (
            <button key={a.slug} onClick={() => open(a.slug)} className="block w-full text-left">
              <Card>
                <p className="text-sm font-medium">{a.title}</p>
                <p className="line-clamp-2 text-xs text-[var(--muted)]">{a.summary}</p>
                <p className="mt-1 text-[10px] uppercase tracking-wide text-[var(--muted)]">{a.category}</p>
              </Card>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
