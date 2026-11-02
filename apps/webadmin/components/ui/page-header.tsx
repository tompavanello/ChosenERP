"use client";

import type { ReactNode } from "react";
import { HelpButton } from "@/components/help/help-button";

export function PageHeader({ title, description, actions }: { title: string; description?: string; actions?: ReactNode }) {
  return (
    <div data-tour="page-header" className="mb-6 flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">{title}</h2>
        {description && <p className="mt-1 text-sm text-zinc-500">{description}</p>}
      </div>
      <div className="flex items-center gap-2">
        {actions}
        <HelpButton />
      </div>
    </div>
  );
}
