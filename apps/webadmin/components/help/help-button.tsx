"use client";

import { CircleHelp } from "lucide-react";
import { useHelp } from "./help-provider";

export function HelpButton({ className = "" }: { className?: string }) {
  const { openHelp } = useHelp();
  return (
    <button
      type="button"
      onClick={() => openHelp("guide")}
      title="Ajuda"
      aria-label="Ajuda"
      data-tour="header-help"
      className={`rounded-md p-2 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-sky-600 dark:hover:bg-zinc-800 ${className}`}
    >
      <CircleHelp className="h-5 w-5" />
    </button>
  );
}
