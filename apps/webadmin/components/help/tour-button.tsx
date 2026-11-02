"use client";

import { Compass } from "lucide-react";
import { usePathname } from "next/navigation";
import { tourKeyForPath } from "@/lib/tours";

/**
 * O botao do topo abre SEMPRE o tour da tela atual diretamente. A biblioteca
 * completa de roteiros fica na Ajuda > Roteiros guiados.
 */
export function TourButton({ className = "" }: { className?: string }) {
  const pathname = usePathname();
  return (
    <button
      type="button"
      onClick={() => window.dispatchEvent(new CustomEvent("chosen:start-tour", { detail: { key: tourKeyForPath(pathname) } }))}
      title="Tour desta tela"
      aria-label="Tour desta tela"
      className={`inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium text-sky-700 transition-colors hover:bg-sky-50 dark:text-sky-300 dark:hover:bg-sky-500/10 ${className}`}
    >
      <Compass className="h-4 w-4" />
      <span className="hidden sm:inline">Tour</span>
    </button>
  );
}
