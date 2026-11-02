"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { usePathname } from "next/navigation";
import {
  ArrowLeft, BookOpen, CheckCircle2, CircleHelp, Compass, Loader2, MessageCircleQuestion,
  Search, Send, Sparkles, Ticket, TriangleAlert,
} from "lucide-react";
import { tourKeyForPath } from "@/lib/tours";
import { Drawer } from "@/components/ui/modal";
import { Tabs } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Input, Select, Textarea } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/components/providers/auth-provider";
import { Markdown } from "./markdown";
import { useHelp } from "./help-provider";
import {
  addSupportTicketMessage, askHelp, createSupportTicket, getHelpArticle, getHelpArticles,
  getHelpContext, getSupportTicket, listMyTickets, sendHelpFeedback,
  type HelpArticle, type HelpChatMessage, type HelpContext, type HelpSummary,
  type SupportTicket,
} from "@/lib/api";

const STATUS_TONE: Record<string, "sky" | "amber" | "green" | "zinc" | "red"> = {
  aberto: "sky", em_andamento: "amber", aguardando: "amber", resolvido: "green", fechado: "zinc",
};

const KIND_LABEL: Record<string, string> = {
  duvida: "Duvida", problema: "Problema", sugestao: "Sugestao", acesso: "Acesso", outro: "Outro",
};

export function HelpDrawer() {
  const { open, tab, article, closeHelp, setTab, setArticle, openTours } = useHelp();
  const { user } = useAuth();
  const pathname = usePathname();
  const isPlatform = !!user?.is_platform_admin;
  const [ctx, setCtx] = useState<HelpContext | null>(null);
  const [aiEnabled, setAiEnabled] = useState(true);
  const [loadingCtx, setLoadingCtx] = useState(false);
  const [ctxError, setCtxError] = useState<string | null>(null);

  const [query, setQuery] = useState("");
  const [results, setResults] = useState<HelpSummary[]>([]);
  const [searching, setSearching] = useState(false);

  const [fullArticle, setFullArticle] = useState<HelpArticle | null>(null);
  const [articleLoading, setArticleLoading] = useState(false);
  const [feedback, setFeedback] = useState<string | null>(null);

  const [messages, setMessages] = useState<HelpChatMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [sending, setSending] = useState(false);
  const [chatError, setChatError] = useState<string | null>(null);
  const [sources, setSources] = useState<HelpSummary[]>([]);
  const [escalate, setEscalate] = useState(false);

  const [tickets, setTickets] = useState<SupportTicket[]>([]);
  const [ticketsLoading, setTicketsLoading] = useState(false);
  const [selected, setSelected] = useState<SupportTicket | null>(null);
  const [ticketForm, setTicketForm] = useState({ kind: "duvida", subject: "", description: "" });
  const [creating, setCreating] = useState(false);
  const [ticketMsg, setTicketMsg] = useState<string | null>(null);
  const [reply, setReply] = useState("");
  const endRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) return;
    let alive = true;
    setLoadingCtx(true);
    setCtxError(null);
    getHelpContext(pathname, "webadmin")
      .then((res) => {
        if (!alive) return;
        setCtx(res.context);
        setAiEnabled(res.ai_enabled);
        setResults(res.context.articles);
      })
      .catch((err) => alive && setCtxError(err instanceof Error ? err.message : "Nao foi possivel carregar a ajuda."))
      .finally(() => alive && setLoadingCtx(false));
    return () => { alive = false; };
  }, [open, pathname]);

  useEffect(() => {
    if (!open || !article) { setFullArticle(null); return; }
    let alive = true;
    setArticleLoading(true);
    setFeedback(null);
    getHelpArticle(article.slug, "webadmin")
      .then((a) => alive && setFullArticle(a))
      .catch(() => alive && setFullArticle(null))
      .finally(() => alive && setArticleLoading(false));
    return () => { alive = false; };
  }, [open, article]);

  useEffect(() => {
    if (!open || tab !== "search") return;
    const q = query.trim();
    let alive = true;
    setSearching(true);
    const timer = setTimeout(() => {
      getHelpArticles({ app: "webadmin", q })
        .then((res) => alive && setResults(res.articles))
        .catch(() => {})
        .finally(() => alive && setSearching(false));
    }, 220);
    return () => { alive = false; clearTimeout(timer); };
  }, [open, tab, query]);

  useEffect(() => {
    if (!open || tab !== "tickets") return;
    let alive = true;
    setTicketsLoading(true);
    listMyTickets("webadmin")
      .then((res) => alive && setTickets(res.tickets ?? []))
      .catch(() => {})
      .finally(() => alive && setTicketsLoading(false));
    return () => { alive = false; };
  }, [open, tab]);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, sending]);

  const title = useMemo(() => (article ? article.title : "Central de ajuda"), [article]);

  async function sendChat() {
    const text = chatInput.trim();
    if (!text || sending) return;
    const next: HelpChatMessage[] = [...messages, { role: "user", content: text }];
    setMessages(next);
    setChatInput("");
    setSending(true);
    setChatError(null);
    try {
      const res = await askHelp({ app: "webadmin", path: pathname, messages: next });
      setMessages([...next, { role: "assistant", content: res.answer }]);
      setSources(res.sources ?? []);
      setEscalate(!!res.escalate);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Falha ao falar com a IA.";
      if (msg.includes("ai_disabled")) {
        setAiEnabled(false);
        setChatError("A IA da ajuda ainda nao esta configurada nesta instalacao. Use a busca do manual ou abra um chamado.");
      } else {
        setChatError(msg);
      }
    } finally {
      setSending(false);
    }
  }

  async function submitTicket(e: React.FormEvent) {
    e.preventDefault();
    if (creating) return;
    setCreating(true);
    setTicketMsg(null);
    try {
      const context = {
        path: pathname,
        role: user?.role,
        tenant_id: user?.tenant_id,
        branch_id: user?.branch_id,
        tenant_name: user?.tenant_id ? undefined : undefined,
        user_agent: typeof navigator !== "undefined" ? navigator.userAgent : "",
        features: user?.features ?? [],
      };
      const lastAnswer = [...messages].reverse().find((m) => m.role === "assistant")?.content ?? "";
      const created = await createSupportTicket({
        kind: ticketForm.kind,
        subject: ticketForm.subject,
        description: ticketForm.description,
        app: "webadmin",
        page_path: pathname,
        priority: "normal",
        context,
        ai_summary: lastAnswer ? lastAnswer.slice(0, 1200) : "",
        ai_transcript: messages.length > 0 ? messages : undefined,
      });
      setTickets((t) => [created, ...t]);
      setTicketForm({ kind: "duvida", subject: "", description: "" });
      setTicketMsg("Chamado aberto. Voce acompanha a resposta nesta aba.");
    } catch (err) {
      setTicketMsg(err instanceof Error ? err.message : "Nao foi possivel abrir o chamado.");
    } finally {
      setCreating(false);
    }
  }

  async function openTicket(id: string) {
    try {
      const t = await getSupportTicket(id);
      setSelected(t);
      setReply("");
    } catch {
      /* ignora */
    }
  }

  async function sendReply(e: React.FormEvent) {
    e.preventDefault();
    if (!selected || !reply.trim()) return;
    try {
      const msg = await addSupportTicketMessage(selected.id, reply.trim());
      setSelected({ ...selected, messages: [...(selected.messages ?? []), msg] });
      setReply("");
    } catch (err) {
      setTicketMsg(err instanceof Error ? err.message : "Nao foi possivel enviar a mensagem.");
    }
  }

  function openArticle(item: { slug: string; title: string }) {
    setArticle(item);
  }

  function goSupport() {
    if (isPlatform) {
      closeHelp();
      window.location.href = "/dashboard/platform/support";
      return;
    }
    setTab("tickets");
  }

  function startTour(key: string) {
    window.dispatchEvent(new CustomEvent("chosen:start-tour", { detail: { key } }));
    closeHelp();
  }

  return (
    <Drawer open={open} onClose={closeHelp} title={title} size="lg">
      {article ? (
        <div>
          <button onClick={() => setArticle(null)} className="mb-3 inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-sky-600">
            <ArrowLeft className="h-3.5 w-3.5" /> Voltar para a ajuda
          </button>
          {articleLoading ? (
            <div className="flex items-center gap-2 py-10 text-sm text-zinc-500"><Loader2 className="h-4 w-4 animate-spin" /> Carregando artigo...</div>
          ) : fullArticle ? (
            <>
              <Markdown>{fullArticle.body}</Markdown>
              <div className="mt-6 rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
                <p className="text-xs font-medium text-zinc-500">Este artigo resolveu sua duvida?</p>
                <div className="mt-2 flex items-center gap-2">
                  <Button size="sm" variant="outline" onClick={() => { sendHelpFeedback({ app: "webadmin", slug: fullArticle.slug, helpful: true }).then(() => setFeedback("Obrigado pelo retorno!")).catch(() => {}); }}>
                    <CheckCircle2 className="h-3.5 w-3.5" /> Sim
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => { sendHelpFeedback({ app: "webadmin", slug: fullArticle.slug, helpful: false }).then(() => setFeedback("Obrigado! Vamos revisar este conteudo.")).catch(() => {}); }}>
                    <TriangleAlert className="h-3.5 w-3.5" /> Nao
                  </Button>
                  {feedback && <span className="text-xs text-zinc-500">{feedback}</span>}
                </div>
              </div>
            </>
          ) : (
            <p className="text-sm text-zinc-500">Nao foi possivel carregar o artigo.</p>
          )}
        </div>
      ) : (
        <>
          <Tabs
            tabs={[
              { key: "guide", label: "Esta tela", icon: <BookOpen className="h-4 w-4" /> },
              { key: "search", label: "Buscar", icon: <Search className="h-4 w-4" /> },
              { key: "ai", label: "Perguntar a IA", icon: <Sparkles className="h-4 w-4" /> },
              ...(isPlatform ? [] : [{ key: "tickets", label: "Meus chamados", icon: <Ticket className="h-4 w-4" /> }]),
            ]}
            active={tab}
            onChange={(k) => { setTab(k as typeof tab); setSelected(null); }}
          />

          {tab === "guide" && (
            <div>
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <Button size="sm" onClick={() => startTour(ctx?.tour || tourKeyForPath(pathname))}>
                  <Compass className="h-3.5 w-3.5" /> Tour desta tela
                </Button>
                <Button size="sm" variant="outline" onClick={openTours}>
                  <Compass className="h-3.5 w-3.5" /> Roteiros guiados
                </Button>
                <Button size="sm" variant="outline" onClick={() => setTab("ai")}>
                  <MessageCircleQuestion className="h-3.5 w-3.5" /> Perguntar a IA
                </Button>
              </div>
              {loadingCtx ? (
                <div className="flex items-center gap-2 py-10 text-sm text-zinc-500"><Loader2 className="h-4 w-4 animate-spin" /> Carregando o guia desta tela...</div>
              ) : ctxError ? (
                <p className="rounded-lg bg-amber-50 p-3 text-sm text-amber-800">{ctxError}</p>
              ) : ctx?.guide ? (
                <>
                  <div className="mb-3">
                    <Badge tone="sky">{ctx.guide.category}</Badge>
                  </div>
                  <Markdown>{ctx.guide.body}</Markdown>
                  {ctx.articles.filter((a) => a.slug !== ctx.guide?.slug).length > 0 && (
                    <div className="mt-6 border-t border-zinc-200 pt-4 dark:border-zinc-800">
                      <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-zinc-400">Relacionados</p>
                      <div className="space-y-1">
                        {ctx.articles.filter((a) => a.slug !== ctx.guide?.slug).map((a) => (
                          <button key={a.slug} onClick={() => openArticle(a)} className="flex w-full items-center justify-between rounded-md px-2 py-2 text-left text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800">
                            <span>{a.title}</span>
                            <span className="text-xs text-zinc-400">{a.category}</span>
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                  <div className="mt-6 rounded-lg bg-sky-50 p-3 text-xs text-sky-800 dark:bg-sky-950/30 dark:text-sky-200">
                    Nao resolveu? Abra um chamado e o contexto desta tela vai junto.
                    <Button size="sm" variant="outline" className="ml-2" onClick={() => goSupport()}>
                      <Ticket className="h-3.5 w-3.5" /> Abrir chamado
                    </Button>
                  </div>
                </>
              ) : (
                <div className="rounded-lg border border-zinc-200 p-6 text-center dark:border-zinc-800">
                  <CircleHelp className="mx-auto h-8 w-8 text-zinc-300" />
                  <p className="mt-2 text-sm text-zinc-500">Ainda nao ha um guia especifico para esta tela.</p>
                  <Button size="sm" variant="outline" className="mt-3" onClick={() => setTab("search")}>Buscar no manual</Button>
                </div>
              )}
            </div>
          )}

          {tab === "search" && (
            <div>
              <div className="relative mb-3">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
                <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Ex.: como lancar um dizimo, conciliar periodo..." className="pl-9" autoFocus />
              </div>
              {searching ? (
                <div className="flex items-center gap-2 py-6 text-sm text-zinc-500"><Loader2 className="h-4 w-4 animate-spin" /> Buscando...</div>
              ) : results.length === 0 ? (
                <p className="py-6 text-center text-sm text-zinc-500">Nenhum artigo encontrado. Tente outra palavra ou abra um chamado.</p>
              ) : (
                <div className="space-y-1">
                  {results.map((a) => (
                    <button key={a.slug} onClick={() => openArticle(a)} className="flex w-full items-start justify-between gap-3 rounded-md px-2 py-2 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800">
                      <span>
                        <span className="block text-sm font-medium">{a.title}</span>
                        <span className="line-clamp-2 text-xs text-zinc-500">{a.summary}</span>
                      </span>
                      <Badge tone="zinc">{a.category}</Badge>
                    </button>
                  ))}
                </div>
              )}
              <div className="mt-4 border-t border-zinc-200 pt-3 text-xs text-zinc-500 dark:border-zinc-800">
                Nao achou? <button className="text-sky-600 underline" onClick={() => goSupport()}>Abrir um chamado</button>.
              </div>
            </div>
          )}

          {tab === "ai" && (
            <div className="flex min-h-[320px] flex-col">
              {!aiEnabled ? (
                <div className="rounded-lg bg-amber-50 p-4 text-sm text-amber-800">
                  A IA da ajuda nao esta configurada nesta instalacao. Use a busca do manual ou abra um chamado.
                  <Button size="sm" variant="outline" className="mt-3" onClick={() => goSupport()}>Abrir chamado</Button>
                </div>
              ) : (
                <>
                  <div className="max-h-[45vh] flex-1 space-y-3 overflow-y-auto pr-1">
                    {messages.length === 0 && (
                      <div className="rounded-lg bg-zinc-50 p-3 text-sm text-zinc-600 dark:bg-zinc-900 dark:text-zinc-300">
                        Ola! Posso explicar esta tela e os fluxos do Chosen ERP com base no manual. Nao executo acoes nem acesso dados financeiros/pessoais.
                      </div>
                    )}
                    {messages.map((m, i) => (
                      <div key={i} className={m.role === "user" ? "ml-6 rounded-lg bg-sky-50 p-3 text-sm dark:bg-sky-950/40" : "mr-2 rounded-lg bg-zinc-100 p-3 text-sm dark:bg-zinc-800"}>
                        <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-zinc-400">{m.role === "user" ? "Voce" : "Assistente"}</p>
                        <Markdown>{m.content}</Markdown>
                      </div>
                    ))}
                    {sending && <div className="flex items-center gap-2 text-xs text-zinc-500"><Loader2 className="h-3.5 w-3.5 animate-spin" /> Pensando...</div>}
                    {chatError && <p className="rounded-md bg-red-50 p-2 text-xs text-red-700">{chatError}</p>}
                    {sources.length > 0 && (
                      <div className="flex flex-wrap gap-1.5">
                        {sources.map((s) => (
                          <button key={s.slug} onClick={() => openArticle(s)} className="rounded-full border border-zinc-200 px-2 py-0.5 text-[11px] text-zinc-600 hover:border-sky-300 hover:text-sky-700 dark:border-zinc-700 dark:text-zinc-300">
                            {s.title}
                          </button>
                        ))}
                      </div>
                    )}
                    {(escalate || messages.length > 0) && !sending && (
                      <div className="rounded-lg bg-amber-50 p-3 text-xs text-amber-800">
                        Ainda precisa de ajuda? Abra um chamado com o contexto desta tela e a conversa.
                        <Button size="sm" variant="outline" className="ml-2" onClick={() => goSupport()}>Abrir chamado</Button>
                      </div>
                    )}
                    <div ref={endRef} />
                  </div>
                  <form
                    onSubmit={(e) => { e.preventDefault(); sendChat(); }}
                    className="mt-3 flex items-end gap-2 border-t border-zinc-200 pt-3 dark:border-zinc-800"
                  >
                    <Textarea
                      value={chatInput}
                      onChange={(e) => setChatInput(e.target.value)}
                      placeholder="Descreva sua duvida..."
                      rows={2}
                      className="min-h-[44px] resize-none"
                      onKeyDown={(e) => {
                        if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); sendChat(); }
                      }}
                    />
                    <Button type="submit" disabled={sending || !chatInput.trim()} className="h-10 shrink-0 px-3">
                      <Send className="h-4 w-4" />
                    </Button>
                  </form>
                </>
              )}
            </div>
          )}

          {tab === "tickets" && (
            <div>
              {selected ? (
                <div>
                  <button onClick={() => setSelected(null)} className="mb-3 inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-sky-600">
                    <ArrowLeft className="h-3.5 w-3.5" /> Voltar
                  </button>
                  <div className="mb-2 flex flex-wrap items-center gap-2">
                    <h3 className="text-base font-semibold">{selected.subject}</h3>
                    <Badge tone={STATUS_TONE[selected.status] ?? "zinc"}>{selected.status.replace("_", " ")}</Badge>
                    <Badge tone="zinc">{KIND_LABEL[selected.kind] ?? selected.kind}</Badge>
                  </div>
                  <p className="mb-3 text-xs text-zinc-500">Aberto em {new Date(selected.created_at).toLocaleString("pt-BR")} · {selected.page_path || "-"}</p>
                  <div className="mb-4 rounded-lg bg-zinc-50 p-3 text-sm whitespace-pre-wrap dark:bg-zinc-900">{selected.description}</div>
                  <div className="space-y-2">
                    {(selected.messages ?? []).map((m) => (
                      <div key={m.id} className={m.author_kind === "user" ? "rounded-lg bg-sky-50 p-2.5 text-sm dark:bg-sky-950/40" : "rounded-lg bg-zinc-100 p-2.5 text-sm dark:bg-zinc-800"}>
                        <p className="mb-1 text-[10px] uppercase tracking-wide text-zinc-400">{m.author_kind === "user" ? "Voce" : m.author_kind === "platform" ? "Suporte" : "Assistente"}</p>
                        <p className="whitespace-pre-wrap">{m.body}</p>
                      </div>
                    ))}
                  </div>
                  <form onSubmit={sendReply} className="mt-4 flex items-end gap-2 border-t border-zinc-200 pt-3 dark:border-zinc-800">
                    <Textarea value={reply} onChange={(e) => setReply(e.target.value)} rows={2} placeholder="Complemente seu chamado..." className="min-h-[44px] resize-none" />
                    <Button type="submit" disabled={!reply.trim()} className="h-10 shrink-0 px-3"><Send className="h-4 w-4" /></Button>
                  </form>
                </div>
              ) : (
                <>
                  {ticketsLoading ? (
                    <div className="flex items-center gap-2 py-6 text-sm text-zinc-500"><Loader2 className="h-4 w-4 animate-spin" /> Carregando chamados...</div>
                  ) : tickets.length > 0 ? (
                    <div className="mb-5 space-y-1">
                      <p className="mb-1 text-xs font-semibold uppercase tracking-wide text-zinc-400">Chamados recentes</p>
                      {tickets.map((t) => (
                        <button key={t.id} onClick={() => openTicket(t.id)} className="flex w-full items-center justify-between gap-2 rounded-md px-2 py-2 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800">
                          <span>
                            <span className="block text-sm font-medium">{t.subject}</span>
                            <span className="text-xs text-zinc-500">{new Date(t.created_at).toLocaleString("pt-BR")}</span>
                          </span>
                          <Badge tone={STATUS_TONE[t.status] ?? "zinc"}>{t.status.replace("_", " ")}</Badge>
                        </button>
                      ))}
                    </div>
                  ) : null}

                  <form onSubmit={submitTicket} className="space-y-3 border-t border-zinc-200 pt-4 dark:border-zinc-800">
                    <p className="text-sm font-semibold">Abrir novo chamado</p>
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Select value={ticketForm.kind} onChange={(e) => setTicketForm({ ...ticketForm, kind: e.target.value })}>
                        {Object.entries(KIND_LABEL).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
                      </Select>
                      <Input value={user?.email ?? ""} disabled placeholder="Seu contato" />
                    </div>
                    <Input required value={ticketForm.subject} onChange={(e) => setTicketForm({ ...ticketForm, subject: e.target.value })} placeholder="Assunto (ex.: erro ao lancar dizimo)" />
                    <Textarea required rows={4} value={ticketForm.description} onChange={(e) => setTicketForm({ ...ticketForm, description: e.target.value })} placeholder="Descreva o que aconteceu, o que voce esperava e o que ja tentou." />
                    <p className="text-[11px] text-zinc-400">Vao junto: tela atual, seu perfil, igreja, navegador e a conversa com a IA (quando houver).</p>
                    {ticketMsg && <p className="text-xs text-sky-700">{ticketMsg}</p>}
                    <Button type="submit" disabled={creating || !ticketForm.subject.trim() || !ticketForm.description.trim()}>
                      {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ticket className="h-4 w-4" />} Abrir chamado
                    </Button>
                  </form>
                </>
              )}
            </div>
          )}
        </>
      )}
    </Drawer>
  );
}

