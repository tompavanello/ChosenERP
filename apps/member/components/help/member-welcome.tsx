"use client";

import { useEffect, useState } from "react";
import { ArrowRight, Church, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { getCachedUser } from "@/lib/api";

const SLIDES = [
  { title: "Bem-vindo ao app da igreja", body: "Aqui voce acompanha a agenda, os avisos, os pedidos de oracao e muito mais." },
  { title: "Comece pela Home", body: "Os atalhos levam para contribuir, devocional, Biblia, escalas, grupos e solicitacoes." },
  { title: "Precisa de ajuda?", body: "Toque no icone ? no topo para o guia desta tela, buscar no manual, perguntar a IA ou abrir um chamado." },
];

export function MemberWelcome() {
  const user = getCachedUser();
  const key = user ? `chosen_member_welcome_v1_${user.id}` : "";
  const [open, setOpen] = useState(false);
  const [slide, setSlide] = useState(0);

  useEffect(() => {
    if (!key) return;
    try {
      if (!localStorage.getItem(key)) setOpen(true);
    } catch {
      /* ignora */
    }
  }, [key]);

  function close() {
    try {
      if (key) localStorage.setItem(key, "1");
    } catch {
      /* ignora */
    }
    setOpen(false);
  }

  if (!open) return null;
  const current = SLIDES[slide];

  return (
    <div className="fixed inset-0 z-40">
      <div className="absolute inset-0 bg-black/50" onClick={close} />
      <div className="absolute inset-x-0 bottom-0 rounded-t-2xl border-t border-[var(--line)] bg-[var(--card)] p-5 pb-7">
        <div className="mb-2 flex items-center justify-between">
          <span className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-[var(--brand)]">
            <Church className="h-4 w-4" /> Guia rapido {slide + 1}/{SLIDES.length}
          </span>
          <button onClick={close} className="text-[var(--muted)]"><X className="h-4 w-4" /></button>
        </div>
        <h2 className="text-lg font-bold">{current.title}</h2>
        <p className="mt-1 text-sm text-[var(--muted)]">{current.body}</p>
        <div className="mt-4 flex items-center justify-between">
          <button onClick={close} className="text-xs text-[var(--muted)]">Pular</button>
          {slide < SLIDES.length - 1 ? (
            <Button size="sm" onClick={() => setSlide((s) => s + 1)}>Proximo <ArrowRight className="h-3.5 w-3.5" /></Button>
          ) : (
            <Button size="sm" onClick={close}>Comecar a usar</Button>
          )}
        </div>
      </div>
    </div>
  );
}
