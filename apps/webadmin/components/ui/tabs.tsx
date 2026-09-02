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
    <div className="mb-6 flex flex-wrap items-center gap-1 border-b border-zinc-200 dark:border-zinc-800">
      {tabs.map((t) => (
        <button
          key={t.key}
          onClick={() => onChange(t.key)}
          className={cn(
            "inline-flex items-center gap-2 border-b-2 px-3 py-2.5 text-sm font-medium transition",
            active === t.key
              ? "border-violet-600 text-violet-700 dark:text-violet-300"
              : "border-transparent text-zinc-500 hover:text-zinc-700 dark:text-zinc-400",
          )}
        >
          {t.icon}
          {t.label}
        </button>
      ))}
    </div>
  );
}
