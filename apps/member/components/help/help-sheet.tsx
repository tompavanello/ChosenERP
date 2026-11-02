"use client";

import { useEffect, useRef, useState } from "react";
import { usePathname } from "next/navigation";
import { ArrowLeft, BookOpen, CheckCircle2, CircleHelp, Loader2, Search, Send, Sparkles, Ticket, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input, Textarea } from "@/components/ui/input";
import { Markdown } from "./markdown";
import { useHelp } from "./help-provider";
import {
  addSupportTicketMessage, askHelp, createSupportTicket, getCachedUser, getHelpArticle,
  getHelpArticles, getHelpContext, getSupportTicket, listMyTickets, sendHelpFeedback,
  type HelpArticle, type HelpChatMessage, type HelpContext, type HelpSummary, type SupportTicket,
} from "@/lib/api";

const STATUS_LABEL: Record<string, string> = {
  aberto: "Aberto", em_andamento: "Em andamento", aguardando: "Aguardando", resolvido: "Resolvido", fechado: "Fechado",
};

export function HelpSheet() {
  const { open, tab, closeHelp, setTab } = useHelp();
  const pathname = usePathname();
  const user = getCachedUser();

  const [ctx, setCtx] = useState<HelpContext | null>(null);
  const [aiEnabled, setAiEnabled] = useState(true);
  const [loadingCtx, setLoadingCtx] = useState(false);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<HelpSummary[]>([]);
  const [searching, setSearching] = useState(false);
  const [article, setArticle] = useState<{ slug: string; title: string } | null>(null);
  const [fullArticle, setFullArticle] = useState<HelpArticle | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);

  const [messages, setMessages] = useState<HelpChatMessage[]>([]);
  const [chatInput, setChatInput] = useState("");
  const [sending, setSending] = useState(false);
  const [chatError, setChatError] = useState<string | null>(null);
  const [sources, setSources] = useState<HelpSummary[]>([]);
  const endRef = useRef<HTMLDivElement | null>(null);

  const [tickets, setTickets] = useState<SupportTicket[]>([]);
  const [selected, setSelected] = useState<SupportTicket | null>(null);
  const [form, setForm] = useState({ kind: "duvida", subject: "", description: "" });
  const [creating, setCreating] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [reply, setReply] = useState("");

  useEffect(() => {
    if (!open) return;
    let alive = true;
    setLoadingCtx(true);
    getHelpContext(pathname, "member")
      .then((res) => {
        if (!alive) return;
        setCtx(res.context);
        setAiEnabled(res.ai_enabled);
        setResults(res.context.articles);
      })
      .catch(() => {})
      .finally(() => alive && setLoadingCtx(false));
    return () => { alive = false; };
  }, [open, pathname]);

  useEffect(() => {
    if (!open || !article) { setFullArticle(null); return; }
    let alive = true;
    getHelpArticle(article.slug, "member")
      .then((a) => alive && setFullArticle(a))
      .catch(() => alive && setFullArticle(null));
    return () => { alive = false; };
  }, [open, article]);

  useEffect(() => {
    if (!open || tab !== "search") return;
    let alive = true;
    setSearching(true);
    const timer = setTimeout(() => {
      getHelpArticles({ app: "member", q: query.trim() })
        .then((res) => alive && setResults(res.articles))
        .catch(() => {})
        .finally(() => alive && setSearching(false));
    }, 220);
    return () => { alive = false; clearTimeout(timer); };
  }, [open, tab, query]);

  useEffect(() => {
    if (!open || tab !== "tickets") return;
    listMyTickets("member").then((res) => setTickets(res.tickets ?? [])).catch(() => {});
  }, [open, tab]);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, sending]);

  async function sendChat() {
    const text = chatInput.trim();
    if (!text || sending) return;
    const next: HelpChatMessage[] = [...messages, { role: "user", content: text }];
    setMessages(next);
    setChatInput("");
    setSending(true);
    setChatError(null);
    try {
      const res = await askHelp({ app: "member", path: pathname, messages: next });
      setMessages([...next, { role: "assistant", content: res.answer }]);
      setSources(res.sources ?? []);
    } catch (err) {
      const m = err instanceof Error ? err.message : "Falha ao falar com a IA.";
      if (m.includes("ai_disabled")) {
        setAiEnabled(false);
        setChatError("A IA da ajuda ainda nao esta configurada. Use a busca ou abra um chamado.");
      } else {
        setChatError(m);
      }
    } finally {
      setSending(false);
    }
  }

  async function submitTicket(e: React.FormEvent) {
    e.preventDefault();
    if (creating) return;
    setCreating(true);
    setMsg(null);
    try {
      const last = [...messages].reverse().find((m) => m.role === "assistant")?.content ?? "";
      const created = await createSupportTicket({
        kind: form.kind,
        subject: form.subject,
        description: form.description,
        app: "member",
        page_path: pathname,
        priority: "normal",
        context: {
          path: pathname,
          role: user?.role,
          tenant_id: user?.tenant_id,
          branch_id: user?.branch_id,
          user_agent: typeof navigator !== "undefined" ? navigator.userAgent : "",
        },
        ai_summary: last ? last.slice(0, 1200) : "",
        ai_transcript: messages.length > 0 ? messages : undefined,
      });
      setTickets((list) => [created, ...list]);
      setForm({ kind: "duvida", subject: "", description: "" });
      setMsg("Chamado aberto! A igreja/suporte respondera por aqui.");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Nao foi possivel abrir o chamado.");
    } finally {
      setCreating(false);
    }
  }

  async function openTicket(id: string) {
    try {
      setSelected(await getSupportTicket(id));
    } catch {
      /* ignora */
    }
  }

  async function sendReply(e: React.FormEvent) {
    e.preventDefault();
    if (!selected || !reply.trim()) return;
    try {
      const m = await addSupportTicketMessage(selected.id, reply.trim());
      setSelected({ ...selected, messages: [...(selected.messages ?? []), m] });
      setReply("");
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Falha ao enviar.");
    }
  }

  if (!open) return null;

  const tabButton = (key: typeof tab, label: string, Icon: typeof BookOpen) => (
    <button
      key={key}
      onClick={() => { setTab(key); setArticle(null); setSelected(null); }}
      className={`flex flex-1 flex-col items-center gap-0.5 rounded-md py-1.5 text-[10px] ${tab === key ? "bg-[var(--brand-soft)] text-[var(--brand-strong)]" : "text-[var(--muted)]"}`}
    >
      <Icon className="h-4 w-4" />
      {label}
    </button>
  );

  return (
    <div className="fixed inset-0 z-50">
      <div className="absolute inset-0 bg-black/50" onClick={closeHelp} />
      <div className="absolute inset-x-0 bottom-0 max-h-[90vh] overflow-y-auto rounded-t-2xl border-t border-[var(--line)] bg-[var(--card)] p-4 pb-6">
        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <CircleHelp className="h-5 w-5 text-[var(--brand)]" />
            <span className="text-sm font-semibold">{article ? article.title : "Ajuda"}</span>
          </div>
          <button onClick={closeHelp} className="text-[var(--muted)]"><X className="h-5 w-5" /></button>
        </div>

        {article ? (
          <div>
            <button onClick={() => setArticle(null)} className="mb-3 inline-flex items-center gap-1 text-xs text-[var(--muted)]">
              <ArrowLeft className="h-3.5 w-3.5" /> Voltar
            </button>
            {fullArticle ? (
              <>
                <Markdown>{fullArticle.body}</Markdown>
                <div className="mt-4 rounded-lg border border-[var(--line)] p-3">
                  <p className="text-xs text-[var(--muted)]">Isso resolveu?</p>
                  <div className="mt-2 flex gap-2">
                    <Button size="sm" variant="outline" onClick={() => sendHelpFeedback({ app: "member", slug: fullArticle.slug, helpful: true }).then(() => setFeedback("Obrigado!")).catch(() => {})}><CheckCircle2 className="h-3.5 w-3.5" /> Sim</Button>
                    <Button size="sm" variant="outline" onClick={() => sendHelpFeedback({ app: "member", slug: fullArticle.slug, helpful: false }).then(() => setFeedback("Obrigado! Vamos revisar.")).catch(() => {})}>Nao</Button>
                    {feedback && <span className="self-center text-xs text-[var(--muted)]">{feedback}</span>}
                  </div>
                </div>
              </>
            ) : (
              <div className="flex items-center gap-2 py-6 text-sm text-[var(--muted)]"><Loader2 className="h-4 w-4 animate-spin" /> Carregando...</div>
            )}
          </div>
        ) : (
          <>
            <div className="mb-3 flex gap-1 rounded-lg bg-black/5 p-1">
              {tabButton("guide", "Esta tela", BookOpen)}
              {tabButton("search", "Buscar", Search)}
              {tabButton("ai", "IA", Sparkles)}
              {tabButton("tickets", "Chamados", Ticket)}
            </div>

            {tab === "guide" && (
              loadingCtx ? (
                <div className="flex items-center gap-2 py-6 text-sm text-[var(--muted)]"><Loader2 className="h-4 w-4 animate-spin" /> Carregando...</div>
              ) : ctx?.guide ? (
                <div>
                  <Markdown>{ctx.guide.body}</Markdown>
                  {ctx.articles.filter((a) => a.slug !== ctx.guide?.slug).length > 0 && (
                    <div className="mt-4 border-t border-[var(--line)] pt-3">
                      <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-[var(--muted)]">Relacionados</p>
                      {ctx.articles.filter((a) => a.slug !== ctx.guide?.slug).map((a) => (
                        <button key={a.slug} onClick={() => setArticle({ slug: a.slug, title: a.title })} className="block w-full py-1.5 text-left text-sm">
                          {a.title}
                        </button>
                      ))}
                    </div>
                  )}
                  <Button size="sm" variant="outline" className="mt-3" onClick={() => setTab("tickets")}><Ticket className="h-3.5 w-3.5" /> Abrir chamado</Button>
                </div>
              ) : (
                <div className="py-6 text-center">
                  <CircleHelp className="mx-auto h-7 w-7 text-[var(--muted)]" />
                  <p className="mt-2 text-sm text-[var(--muted)]">Sem guia especifico para esta tela.</p>
                  <Button size="sm" variant="outline" className="mt-3" onClick={() => setTab("search")}>Buscar no manual</Button>
                </div>
              )
            )}

            {tab === "search" && (
              <div>
                <div className="relative mb-3">
                  <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--muted)]" />
                  <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Buscar no manual..." className="pl-9" />
                </div>
                {searching ? (
                  <div className="flex items-center gap-2 py-4 text-sm text-[var(--muted)]"><Loader2 className="h-4 w-4 animate-spin" /> Buscando...</div>
                ) : results.length === 0 ? (
                  <p className="py-4 text-center text-sm text-[var(--muted)]">Nenhum artigo encontrado.</p>
                ) : (
                  <div className="space-y-1">
                    {results.map((a) => (
                      <button key={a.slug} onClick={() => setArticle({ slug: a.slug, title: a.title })} className="block w-full rounded-md px-2 py-2 text-left hover:bg-black/5">
                        <span className="block text-sm font-medium">{a.title}</span>
                        <span className="line-clamp-2 text-xs text-[var(--muted)]">{a.summary}</span>
                      </button>
                    ))}
                  </div>
                )}
                <Button size="sm" variant="outline" className="mt-3" onClick={() => setTab("tickets")}>Nao achei — abrir chamado</Button>
              </div>
            )}

            {tab === "ai" && (
              <div>
                {!aiEnabled ? (
                  <div className="rounded-lg bg-amber-50 p-3 text-sm text-amber-800">
                    A IA da ajuda nao esta configurada. Use a busca ou abra um chamado.
                    <Button size="sm" variant="outline" className="mt-2" onClick={() => setTab("tickets")}>Abrir chamado</Button>
                  </div>
                ) : (
                  <>
                    <div className="max-h-[42vh] space-y-2 overflow-y-auto">
                      {messages.length === 0 && <p className="rounded-lg bg-black/5 p-3 text-sm text-[var(--muted)]">Ola! Posso explicar as telas e fluxos do app com base no manual. Nao executo acoes nem acesso dados sensiveis.</p>}
                      {messages.map((m, i) => (
                        <div key={i} className={m.role === "user" ? "ml-5 rounded-lg bg-[var(--brand-soft)] p-2.5 text-sm" : "rounded-lg bg-black/5 p-2.5 text-sm"}>
                          <Markdown>{m.content}</Markdown>
                        </div>
                      ))}
                      {sending && <div className="flex items-center gap-2 text-xs text-[var(--muted)]"><Loader2 className="h-3.5 w-3.5 animate-spin" /> Pensando...</div>}
                      {chatError && <p className="rounded bg-red-50 p-2 text-xs text-red-700">{chatError}</p>}
                      {sources.length > 0 && (
                        <div className="flex flex-wrap gap-1">
                          {sources.map((s) => (
                            <button key={s.slug} onClick={() => setArticle({ slug: s.slug, title: s.title })} className="rounded-full border border-[var(--line)] px-2 py-0.5 text-[10px]">{s.title}</button>
                          ))}
                        </div>
                      )}
                      <div ref={endRef} />
                    </div>
                    <form onSubmit={(e) => { e.preventDefault(); sendChat(); }} className="mt-3 flex items-end gap-2">
                      <Textarea value={chatInput} onChange={(e) => setChatInput(e.target.value)} rows={2} placeholder="Sua duvida..." className="min-h-[44px] resize-none" />
                      <Button type="submit" disabled={sending || !chatInput.trim()} className="h-10 px-3"><Send className="h-4 w-4" /></Button>
                    </form>
                    <Button size="sm" variant="outline" className="mt-2" onClick={() => setTab("tickets")}>Nao resolveu — abrir chamado</Button>
                  </>
                )}
              </div>
            )}

            {tab === "tickets" && (
              <div>
                {selected ? (
                  <div>
                    <button onClick={() => setSelected(null)} className="mb-2 inline-flex items-center gap-1 text-xs text-[var(--muted)]"><ArrowLeft className="h-3.5 w-3.5" /> Voltar</button>
                    <p className="text-sm font-semibold">{selected.subject}</p>
                    <p className="mb-2 text-xs text-[var(--muted)]">{STATUS_LABEL[selected.status] ?? selected.status} · {new Date(selected.created_at).toLocaleString("pt-BR")}</p>
                    <p className="mb-3 whitespace-pre-wrap rounded-lg bg-black/5 p-2.5 text-sm">{selected.description}</p>
                    <div className="space-y-2">
                      {(selected.messages ?? []).map((m) => (
                        <div key={m.id} className={m.author_kind === "user" ? "rounded-lg bg-[var(--brand-soft)] p-2.5 text-sm" : "rounded-lg bg-black/5 p-2.5 text-sm"}>
                          <p className="mb-1 text-[10px] uppercase text-[var(--muted)]">{m.author_kind === "platform" ? "Suporte" : m.author_kind === "ai" ? "Assistente" : "Voce"}</p>
                          <p className="whitespace-pre-wrap">{m.body}</p>
                        </div>
                      ))}
                    </div>
                    <form onSubmit={sendReply} className="mt-3 flex items-end gap-2">
                      <Textarea value={reply} onChange={(e) => setReply(e.target.value)} rows={2} placeholder="Complementar..." className="min-h-[44px] resize-none" />
                      <Button type="submit" disabled={!reply.trim()} className="h-10 px-3"><Send className="h-4 w-4" /></Button>
                    </form>
                  </div>
                ) : (
                  <>
                    {tickets.length > 0 && (
                      <div className="mb-4 space-y-1">
                        {tickets.map((t) => (
                          <button key={t.id} onClick={() => openTicket(t.id)} className="flex w-full items-center justify-between gap-2 rounded-md px-2 py-2 text-left hover:bg-black/5">
                            <span className="min-w-0">
                              <span className="block truncate text-sm font-medium">{t.subject}</span>
                              <span className="text-[10px] text-[var(--muted)]">{new Date(t.created_at).toLocaleString("pt-BR")}</span>
                            </span>
                            <span className="shrink-0 rounded-full bg-black/5 px-2 py-0.5 text-[10px]">{STATUS_LABEL[t.status] ?? t.status}</span>
                          </button>
                        ))}
                      </div>
                    )}
                    <form onSubmit={submitTicket} className="space-y-2 border-t border-[var(--line)] pt-3">
                      <p className="text-sm font-semibold">Abrir chamado</p>
                      <select value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })} className="select">
                        <option value="duvida">Duvida</option>
                        <option value="problema">Problema</option>
                        <option value="sugestao">Sugestao</option>
                        <option value="acesso">Acesso</option>
                        <option value="outro">Outro</option>
                      </select>
                      <Input required value={form.subject} onChange={(e) => setForm({ ...form, subject: e.target.value })} placeholder="Assunto" />
                      <Textarea required rows={4} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="Descreva o que aconteceu..." />
                      <p className="text-[10px] text-[var(--muted)]">Vao junto: tela atual, seu perfil, igreja e a conversa com a IA.</p>
                      {msg && <p className="text-xs text-[var(--brand)]">{msg}</p>}
                      <Button type="submit" disabled={creating || !form.subject.trim() || !form.description.trim()}>
                        {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ticket className="h-4 w-4" />} Abrir chamado
                      </Button>
                    </form>
                  </>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
