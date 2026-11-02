"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { ArrowLeft, ArrowRight, CheckCircle2, Circle, Compass, Sparkles, X } from "lucide-react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useAuth } from "@/components/providers/auth-provider";
import { TOURS, tourKeyForPath } from "@/lib/tours";
import { useHelp } from "./help-provider";

const WELCOME_SLIDES = [
  {
    title: "Bem-vindo ao Chosen ERP",
    body: "Este guia rapido mostra a navegacao principal. Ele nao bloqueia nada: voce pode fechar e voltar depois pela Ajuda.",
  },
  {
    title: "Tudo no contexto da igreja e da filial",
    body: "No topo voce troca de igreja (quando ha mais de uma) e, na Sede, escolhe a filial de trabalho. O menu e os dados respeitam esse contexto.",
  },
  {
    title: "O menu esconde o que voce nao pode acessar",
    body: "Cada perfil tem permissoes e cada igreja tem um plano. Se um modulo nao aparece, pode ser permissao ou plano.",
  },
  {
    title: "Ajuda em toda tela",
    body: "O botao ? no topo abre o guia da tela, a busca do manual, a IA e os chamados. Comece por ali sempre que tiver duvida.",
  },
];

interface Step {
  key: string;
  label: string;
  hint: string;
  href: string;
  perm?: string;
  platform?: boolean;
}

const CHURCH_STEPS: Step[] = [
  { key: "church", label: "Completar os dados da igreja", hint: "Nome, CNPJ, logo, cor e Pix.", href: "/dashboard/settings", perm: "settings.write" },
  { key: "branches", label: "Revisar as filiais", hint: "Matriz, filiais e PAEs.", href: "/dashboard/settings", perm: "settings.write" },
  { key: "members", label: "Cadastrar os primeiros membros", hint: "Ou importar uma planilha.", href: "/dashboard/members", perm: "members.write" },
  { key: "finance", label: "Configurar o financeiro", hint: "Contas, plano de contas e lancamentos.", href: "/dashboard/finance", perm: "finance.write" },
  { key: "users", label: "Criar acessos da equipe", hint: "Cada pessoa com o proprio perfil.", href: "/dashboard/users", perm: "users.write" },
  { key: "announcement", label: "Enviar o primeiro comunicado", hint: "Convide os membros a instalar o app.", href: "/dashboard/announcements", perm: "announcements.write" },
];

const PLATFORM_STEPS: Step[] = [
  { key: "tenant", label: "Criar a primeira igreja", hint: "Tenant + Matriz + papeis.", href: "/dashboard/platform/churches", platform: true },
  { key: "plans", label: "Revisar o catalogo de planos", hint: "Features e limites.", href: "/dashboard/platform/plans", platform: true },
  { key: "billing", label: "Conferir a cobranca", hint: "Assinaturas e faturas.", href: "/dashboard/platform/billing", platform: true },
  { key: "backups", label: "Configurar os backups", hint: "Frequencia e retencao.", href: "/dashboard/platform/backups", platform: true },
  { key: "support", label: "Acompanhar o suporte", hint: "Chamados abertos pela ajuda.", href: "/dashboard/platform/support", platform: true },
];

export function WelcomeGuide() {
  const { user, ready } = useAuth();
  const { openTours } = useHelp();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const [slide, setSlide] = useState(0);
  const key = user ? `chosen_welcome_v1_${user.id}` : "";

  useEffect(() => {
    if (!ready || !user || !key) return;
    try {
      if (!localStorage.getItem(key)) setOpen(true);
    } catch {
      /* storage indisponivel */
    }
  }, [ready, user, key]);

  function finish(startTour = false) {
    try {
      if (key) localStorage.setItem(key, "1");
    } catch {
      /* ignora */
    }
    setOpen(false);
    if (startTour) {
      window.dispatchEvent(new CustomEvent("chosen:start-tour", { detail: { key: tourKeyForPath(pathname) } }));
    }
  }

  useEffect(() => {
    if (!open) setSlide(0);
  }, [open]);

  if (!user) return null;
  const current = WELCOME_SLIDES[slide];

  return (
    <Modal open={open} onClose={() => finish(false)} title="Guia rapido">
      <div className="space-y-4">
        <div className="flex items-center gap-2 text-sky-600">
          <Compass className="h-5 w-5" />
          <span className="text-xs font-semibold uppercase tracking-wide">Passo {slide + 1} de {WELCOME_SLIDES.length}</span>
        </div>
        <div>
          <h4 className="text-lg font-semibold">{current.title}</h4>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-300">{current.body}</p>
        </div>
        <div className="flex items-center justify-between gap-2 border-t border-zinc-200 pt-3 dark:border-zinc-800">
          <button className="text-xs text-zinc-400 hover:text-zinc-600" onClick={() => finish(false)}>Pular por agora</button>
          <div className="flex items-center gap-2">
            {slide > 0 && (
              <Button size="sm" variant="outline" onClick={() => setSlide((s) => s - 1)}>
                <ArrowLeft className="h-3.5 w-3.5" /> Voltar
              </Button>
            )}
            {slide < WELCOME_SLIDES.length - 1 ? (
              <Button size="sm" onClick={() => setSlide((s) => s + 1)}>
                Proximo <ArrowRight className="h-3.5 w-3.5" />
              </Button>
            ) : (
              <>
                <Button size="sm" variant="outline" onClick={() => finish(true)}>
                  <Compass className="h-3.5 w-3.5" /> Tour desta tela
                </Button>
                <Button size="sm" onClick={() => { finish(false); openTours(); }}>
                  <Sparkles className="h-3.5 w-3.5" /> Roteiros guiados
                </Button>
              </>
            )}
          </div>
        </div>
      </div>
    </Modal>
  );
}

export function OnboardingChecklist() {
  const { user, hasPerm, isPlatformAdmin } = useAuth();
  const [hidden, setHidden] = useState(false);
  const [done, setDone] = useState<Record<string, boolean>>({});
  const storageKey = user ? `chosen_checklist_v1_${user.id}` : "";

  useEffect(() => {
    try {
      const raw = storageKey ? localStorage.getItem(storageKey) : null;
      if (raw) setDone(JSON.parse(raw) as Record<string, boolean>);
      if (storageKey && localStorage.getItem(`${storageKey}_hidden`) === "1") setHidden(true);
    } catch {
      /* ignora */
    }
  }, [storageKey]);

  const steps = useMemo(() => {
    if (isPlatformAdmin) return PLATFORM_STEPS;
    return CHURCH_STEPS.filter((s) => !s.perm || hasPerm(s.perm));
  }, [isPlatformAdmin, hasPerm]);

  if (!user || hidden || steps.length === 0) return null;
  const completed = steps.filter((s) => done[s.key]).length;

  function toggle(key: string) {
    const next = { ...done, [key]: !done[key] };
    setDone(next);
    try {
      if (storageKey) localStorage.setItem(storageKey, JSON.stringify(next));
    } catch {
      /* ignora */
    }
  }

  function dismiss() {
    setHidden(true);
    try {
      if (storageKey) localStorage.setItem(`${storageKey}_hidden`, "1");
    } catch {
      /* ignora */
    }
  }

  return (
    <Card className="p-4">
      <div className="mb-3 flex items-start justify-between gap-2">
        <div>
          <h3 className="flex items-center gap-2 text-sm font-semibold">
            <Sparkles className="h-4 w-4 text-sky-600" /> Primeiros passos
          </h3>
          <p className="text-xs text-zinc-500">
            {completed} de {steps.length} concluidos. Marque conforme avanca.
          </p>
        </div>
        <button onClick={dismiss} title="Ocultar" className="text-zinc-400 hover:text-zinc-600">
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="mb-3 h-1.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
        <div className="h-full rounded-full bg-sky-500 transition-all" style={{ width: `${steps.length ? (completed / steps.length) * 100 : 0}%` }} />
      </div>
      <div className="space-y-1">
        {steps.map((s) => (
          <div key={s.key} className="flex items-center gap-2 rounded-md px-1 py-1.5 hover:bg-zinc-50 dark:hover:bg-zinc-900">
            <button onClick={() => toggle(s.key)} title={done[s.key] ? "Marcar como pendente" : "Marcar como concluido"} className="shrink-0">
              {done[s.key] ? <CheckCircle2 className="h-4 w-4 text-emerald-500" /> : <Circle className="h-4 w-4 text-zinc-300" />}
            </button>
            <Link href={s.href} className="min-w-0 flex-1">
              <span className={`block text-sm ${done[s.key] ? "text-zinc-400 line-through" : ""}`}>{s.label}</span>
              <span className="block text-[11px] text-zinc-400">{s.hint}</span>
            </Link>
            <ArrowRight className="h-3.5 w-3.5 shrink-0 text-zinc-300" />
          </div>
        ))}
      </div>
    </Card>
  );
}

// Tour guiado simples (spotlight + cartao). Sem dependencia externa.
export function TourOverlay() {
  const router = useRouter();
  const pathname = usePathname();
  const { openTours } = useHelp();
  const [key, setKey] = useState<string | null>(null);
  const [index, setIndex] = useState(0);
  const [rect, setRect] = useState<DOMRect | null>(null);

  useEffect(() => {
    function onStart(e: Event) {
      const detail = (e as CustomEvent<{ key?: string }>).detail;
      const next = detail?.key ?? "overview";
      if (!TOURS[next]) return;
      setKey(next);
      setIndex(0);
    }
    function onEnd() { setKey(null); }
    window.addEventListener("chosen:start-tour", onStart);
    window.addEventListener("chosen:end-tour", onEnd);
    return () => {
      window.removeEventListener("chosen:start-tour", onStart);
      window.removeEventListener("chosen:end-tour", onEnd);
    };
  }, []);

  const steps = key ? TOURS[key] : [];
  const step = steps[index];

  function next() {
    setIndex((i) => Math.min(i + 1, steps.length - 1));
  }

  useEffect(() => {
    if (!step) { setRect(null); return; }
    let cancelled = false;

    // Antes de destacar, garante que estamos na rota do passo (ex.: abrir
    // Membros, Financeiro, Eventos). A tela continua interativa.
    const place = (attempt = 0) => {
      if (cancelled) return;
      const el = document.querySelector(step.target) as HTMLElement | null;
      if (!el) {
        // A tela pode estar carregando/re-renderizando (modal aberto, lista
        // recarregada). Tenta por alguns segundos antes de desistir do destaque.
        if (attempt < 10) window.setTimeout(() => place(attempt + 1), 250);
        else setRect(null);
        return;
      }
      el.scrollIntoView({ behavior: "smooth", block: "center" });
      if (step.action === "focus") {
        window.setTimeout(() => { if (!cancelled) el.focus({ preventScroll: true }); }, 250);
      }
      window.setTimeout(() => { if (!cancelled) setRect(el.getBoundingClientRect()); }, 320);
    };

    let timer: number;
    if (step.route && pathname !== step.route) {
      router.push(step.route);
      timer = window.setTimeout(place, 650);
    } else {
      timer = window.setTimeout(place, 200);
    }
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [step, pathname, router]);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (!key) return;
      const target = e.target as HTMLElement | null;
      const tag = target?.tagName ?? "";
      const editing = tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || !!target?.isContentEditable;
      if (editing) return; // nao interfere na digitacao nos campos do tour
      if (e.key === "Escape") setKey(null);
      if (e.key === "ArrowRight") setIndex((i) => Math.min(i + 1, steps.length - 1));
      if (e.key === "ArrowLeft") setIndex((i) => Math.max(0, i - 1));
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [key, steps.length]);

  if (!key || !step) return null;

  function runAction() {
    const el = document.querySelector(step.target) as HTMLElement | null;
    if (el) el.click();
    window.setTimeout(() => next(), 500);
  }

  return (
    <div className="pointer-events-none fixed inset-0 z-[90]">
      <div className="pointer-events-none absolute inset-0 bg-black/15" />
      {rect && (
        <div
          className="pointer-events-none absolute rounded-lg border-2 border-sky-400 shadow-[0_0_0_9999px_rgba(0,0,0,0.05)]"
          style={{ top: rect.top - 4, left: rect.left - 4, width: rect.width + 8, height: rect.height + 8 }}
        />
      )}
      <div className="pointer-events-auto absolute bottom-6 left-1/2 w-[min(92vw,28rem)] -translate-x-1/2 rounded-xl border border-zinc-200 bg-white p-4 shadow-xl dark:border-zinc-700 dark:bg-zinc-900">
        <div className="mb-1 flex items-center justify-between">
          <span className="text-[10px] font-semibold uppercase tracking-wide text-sky-600">Tour {index + 1}/{steps.length}</span>
          <button onClick={() => setKey(null)} className="text-zinc-400 hover:text-zinc-600" aria-label="Fechar tour"><X className="h-4 w-4" /></button>
        </div>
        <h4 className="text-sm font-semibold">{step.title}</h4>
        <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-300">{step.body}</p>
        {step.action === "click" && (
          <div className="mt-3 rounded-lg bg-sky-50 p-3 dark:bg-sky-950/30">
            <p className="text-xs text-sky-800 dark:text-sky-200">
              Esta etapa tem uma acao. Voce pode executa-la agora e avancar automaticamente.
            </p>
            <Button size="sm" className="mt-2" disabled={!rect} onClick={runAction}>
              <Compass className="h-3.5 w-3.5" /> {rect ? "Executar e avancar" : "Abrindo a tela..."}
            </Button>
          </div>
        )}
        {step.action === "focus" && (
          <p className="mt-2 text-[11px] text-sky-700 dark:text-sky-300">O campo em destaque ja esta em foco para voce digitar.</p>
        )}
        <div className="mt-3 flex flex-wrap items-center justify-between gap-2">
          <button className="text-xs text-zinc-400 hover:text-zinc-600" onClick={() => setKey(null)}>Sair</button>
          <div className="flex flex-wrap items-center justify-end gap-2">
            <Button size="sm" variant="ghost" onClick={() => { setKey(null); openTours(); }}>Outros roteiros</Button>
            <Button size="sm" variant="outline" disabled={index === 0} onClick={() => setIndex((i) => Math.max(0, i - 1))}>Anterior</Button>
            {index < steps.length - 1 ? (
              <Button size="sm" onClick={next}>Proximo</Button>
            ) : (
              <Button size="sm" onClick={() => setKey(null)}>Concluir</Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

