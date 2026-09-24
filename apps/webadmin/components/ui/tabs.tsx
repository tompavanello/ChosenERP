"use client";

import { cn } from "@/lib/utils";

export function Tabs({
  tabs,
  active,
  onChange,
}: {
  tabs: { key: string; label: string; icon?: React.ReactNode }[];
  active: string;
  onChange: (key: string) => void;
}) {
  return (
    <div className="mb-6 flex flex-wrap items-center gap-1 border-b border-zinc-200">
      {tabs.map((t) => (
        <button
          key={t.key}
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
