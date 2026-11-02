"use client";

import { useMemo } from "react";
import { usePathname } from "next/navigation";
import {
  Bell, CalendarClock, CalendarDays, Compass, LayoutDashboard, Settings, Users, Wallet,
} from "lucide-react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/components/providers/auth-provider";
import { useHelp } from "./help-provider";
import { TOUR_LIBRARY, tourKeyForPath, type TourDefinition } from "@/lib/tours";

const CATEGORY_ICON: Record<string, typeof LayoutDashboard> = {
  Pessoas: Users,
  Financeiro: Wallet,
  Organizacao: CalendarDays,
  Comunicacao: Bell,
  Administracao: Settings,
  Interface: Compass,
};

const CATEGORY_ORDER = ["Pessoas", "Financeiro", "Organizacao", "Comunicacao", "Administracao", "Interface"];

export function TourLibrary() {
  const { tourOpen, closeTours } = useHelp();
  const { hasPerm, hasFeature, isPlatformAdmin } = useAuth();
  const pathname = usePathname();

  const tours = useMemo(
    () => TOUR_LIBRARY.filter((t) => {
      if (t.platformOnly && !isPlatformAdmin) return false;
      // O admin de plataforma nao tem o menu operacional; mostra apenas o
      // roteiro da plataforma e o tour generico da tela.
      if (isPlatformAdmin && !t.platformOnly && t.key !== "page") return false;
      return (!t.permission || hasPerm(t.permission)) && (!t.feature || hasFeature(t.feature));
    }),
    [hasPerm, hasFeature, isPlatformAdmin],
  );
  const currentKey = tourKeyForPath(pathname);
  const current = TOUR_LIBRARY.find((t) => t.key === currentKey);

  const grouped = useMemo(() => {
    const map: Record<string, TourDefinition[]> = {};
    for (const t of tours) (map[t.category] ??= []).push(t);
    return map;
  }, [tours]);

  function start(key: string) {
    closeTours();
    window.setTimeout(() => {
      window.dispatchEvent(new CustomEvent("chosen:start-tour", { detail: { key } }));
    }, 80);
  }

  return (
    <Modal open={tourOpen} onClose={closeTours} title="Roteiros guiados" size="lg">
      <div className="space-y-5">
        <div className="rounded-lg bg-sky-50 p-4 dark:bg-sky-950/30">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-sm font-semibold">{current?.title ?? "Tour desta tela"}</p>
              <p className="mt-0.5 text-xs text-zinc-600 dark:text-zinc-300">
                Destaque do contexto atual: topo, menu ativo, cabecalho e ajuda.
              </p>
            </div>
            <Button size="sm" onClick={() => start(currentKey)}>
              <Compass className="h-3.5 w-3.5" /> Tour desta tela
            </Button>
          </div>
        </div>

        <p className="text-xs text-zinc-500">
          Os roteiros abaixo ensinam tarefas reais. Voce continua no controle: o cartao do tour
          destaca o botao e oferece <b>Executar e avancar</b> quando houver uma acao. A tela
          permanece interativa durante o tour.
        </p>

        {CATEGORY_ORDER.filter((c) => grouped[c]?.length).map((category) => {
          const Icon = CATEGORY_ICON[category] ?? Compass;
          return (
            <section key={category}>
              <h3 className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-zinc-400">
                <Icon className="h-3.5 w-3.5" /> {category}
              </h3>
              <div className="grid gap-2 sm:grid-cols-2">
                {grouped[category].map((t) => (
                  <div key={t.key} className="flex flex-col justify-between rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
                    <div>
                      <div className="flex items-start justify-between gap-2">
                        <p className="text-sm font-semibold">{t.title}</p>
                        <Badge tone="zinc">{t.steps.length} passos</Badge>
                      </div>
                      <p className="mt-1 text-xs text-zinc-500">{t.description}</p>
                    </div>
                    <div className="mt-3 flex justify-end">
                      <Button size="sm" variant="outline" onClick={() => start(t.key)}>
                        <Compass className="h-3.5 w-3.5" /> Iniciar
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          );
        })}

        <div className="flex items-center justify-between border-t border-zinc-200 pt-3 text-xs text-zinc-400 dark:border-zinc-800">
          <span className="flex items-center gap-1"><CalendarClock className="h-3.5 w-3.5" /> Use Esc para sair e as setas para navegar.</span>
          <Button size="sm" variant="ghost" onClick={closeTours}>Fechar</Button>
        </div>
      </div>
    </Modal>
  );
}
