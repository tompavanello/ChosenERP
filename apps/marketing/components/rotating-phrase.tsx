"use client";

import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";

/**
 * Trecho de texto que troca de valor periodicamente com um fade suave.
 *
 * Usado no H1 do hero: "Cuide <frase> em uma só plataforma". A frase some
 * (opacity 0) antes de trocar, para o reflow da linha acontecer fora da vista.
 * Respeita prefers-reduced-motion (mostra a primeira frase e nao gira).
 */
export function RotatingPhrase({
  phrases,
  interval = 2800,
  className,
}: {
  phrases: readonly string[];
  interval?: number;
  className?: string;
}) {
  const [index, setIndex] = useState(0);
  const [shown, setShown] = useState(true);
  const [reduceMotion, setReduceMotion] = useState(false);

  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduceMotion(mq.matches);
    update();
    mq.addEventListener("change", update);
    return () => mq.removeEventListener("change", update);
  }, []);

  useEffect(() => {
    if (reduceMotion || phrases.length <= 1) return;
    const id = window.setInterval(() => {
      setShown(false);
      window.setTimeout(() => {
        setIndex((i) => (i + 1) % phrases.length);
        setShown(true);
      }, 300);
    }, interval);
    return () => window.clearInterval(id);
  }, [reduceMotion, phrases.length, interval]);

  return (
    <span
      className={cn(
        "gradient-text inline-block transition-[opacity,transform] duration-300 ease-out will-change-transform",
        shown ? "translate-y-0 opacity-100" : "-translate-y-2 opacity-0",
        className,
      )}
    >
      {phrases[index]}
    </span>
  );
}
