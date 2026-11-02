"use client";

import { cn } from "@/lib/utils";

export function Tabs({
  tabs,
  active,
  onChange,
  tourPrefix,
}: {
  tabs: { key: string; label: string; icon?: React.ReactNode }[];
  active: string;
  onChange: (key: string) => void;
  /** Quando informado, gera data-tour={`${tourPrefix}-${key}`} em cada aba. */
  tourPrefix?: string;
}) {
  return (
    <div className="mb-6 flex flex-wrap items-center gap-1 border-b border-zinc-200">
      {tabs.map((t) => (
        <button
          key={t.key}
          data-tour={tourPrefix ? `${tourPrefix}-${t.key}` : undefined}
          onClick={() => onChange(t.key)}
          className={cn(
            "inline-flex items-center gap-2 border-b-2 px-3 py-2.5 text-sm font-medium transition",
            active === t.key
              ? "border-sky-600 text-sky-700"
              : "border-transparent text-zinc-500 hover:text-zinc-700",
          )}
        >
          {t.icon}
          {t.label}
        </button>
      ))}
    </div>
  );
}
