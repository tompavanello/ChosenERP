"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, CheckCircle2, Loader2, TriangleAlert } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Markdown } from "@/components/help/markdown";
import { getHelpArticle, sendHelpFeedback, type HelpArticle } from "@/lib/api";
import { useHelp } from "@/components/help/help-provider";

export default function HelpArticlePage() {
  const params = useParams<{ slug: string }>();
  const { openHelp } = useHelp();
  const [article, setArticle] = useState<HelpArticle | null>(null);
  const [loading, setLoading] = useState(true);
  const [feedback, setFeedback] = useState<string | null>(null);

  useEffect(() => {
    const slug = Array.isArray(params.slug) ? params.slug[0] : params.slug;
    if (!slug) return;
    setLoading(true);
    getHelpArticle(slug, "webadmin")
      .then(setArticle)
      .catch(() => setArticle(null))
      .finally(() => setLoading(false));
  }, [params.slug]);

  return (
    <div className="page mx-auto max-w-3xl space-y-4">
      <Link href="/dashboard/help" className="inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-sky-600">
        <ArrowLeft className="h-3.5 w-3.5" /> Central de ajuda
      </Link>

      {loading ? (
        <Card className="flex items-center gap-2 p-8 text-sm text-zinc-500">
          <Loader2 className="h-4 w-4 animate-spin" /> Carregando artigo...
        </Card>
      ) : !article ? (
        <Card className="p-8 text-center">
          <p className="text-sm text-zinc-500">Artigo nao encontrado ou indisponivel para o seu perfil/plano.</p>
          <Button variant="outline" className="mt-3" onClick={() => openHelp("tickets")}>Abrir um chamado</Button>
        </Card>
      ) : (
        <Card className="p-5 sm:p-7">
          <div className="mb-4 flex flex-wrap items-center gap-2">
            <Badge tone="sky">{article.category}</Badge>
            {article.tour && <Badge tone="zinc">Tour disponivel</Badge>}
            {article.feature && <Badge tone="zinc">Modulo: {article.feature}</Badge>}
          </div>
          <Markdown>{article.body}</Markdown>
          <div className="mt-8 rounded-lg border border-zinc-200 p-4 dark:border-zinc-800">
            <p className="text-sm font-medium">Este artigo resolveu sua duvida?</p>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <Button size="sm" variant="outline" onClick={() => sendHelpFeedback({ app: "webadmin", slug: article.slug, helpful: true }).then(() => setFeedback("Obrigado pelo retorno!")).catch(() => {})}>
                <CheckCircle2 className="h-3.5 w-3.5" /> Sim
              </Button>
              <Button size="sm" variant="outline" onClick={() => sendHelpFeedback({ app: "webadmin", slug: article.slug, helpful: false }).then(() => setFeedback("Obrigado! Vamos revisar este conteudo.")).catch(() => {})}>
                <TriangleAlert className="h-3.5 w-3.5" /> Nao
              </Button>
              <Button size="sm" onClick={() => openHelp("tickets")}>Abrir chamado</Button>
              {feedback && <span className="text-xs text-zinc-500">{feedback}</span>}
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
