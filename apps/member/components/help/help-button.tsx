"use client";

import { CircleHelp } from "lucide-react";
import { useHelp } from "./help-provider";

export function HelpButton() {
  const { openHelp } = useHelp();
  return (
    <button onClick={() => openHelp("guide")} aria-label="Ajuda" title="Ajuda" className="text-[var(--muted)] hover:text-[var(--brand)]">
      <CircleHelp className="h-5 w-5" />
    </button>
  );
}
