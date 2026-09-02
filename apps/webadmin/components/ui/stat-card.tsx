import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { Card } from "./card";

export function StatCard({
  label,
  value,
  icon: Icon,
  hint,
  tone = "violet",
}: {
  label: string;
  value: string;
  icon?: LucideIcon;
  hint?: string;
  tone?: "violet" | "green" | "red" | "sky" | "amber";
}) {
  const tones: Record<string, string> = {
    violet: "text-violet-600 bg-violet-50 dark:bg-violet-500/15 dark:text-violet-300",
    green: "text-emerald-600 bg-emerald-50 dark:bg-emerald-500/15 dark:text-emerald-300",
    red: "text-red-600 bg-red-50 dark:bg-red-500/15 dark:text-red-300",
    sky: "text-sky-600 bg-sky-50 dark:bg-sky-500/15 dark:text-sky-300",
    amber: "text-amber-600 bg-amber-50 dark:bg-amber-500/15 dark:text-amber-300",
  };
  return (
    <Card className="flex items-center justify-between">
      <div>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">{label}</p>
        <p className="mt-1 text-2xl font-semibold tracking-tight">{value}</p>
        {hint && <p className="mt-1 text-xs text-zinc-400">{hint}</p>}
      </div>
      {Icon && (
        <span className={cn("flex h-11 w-11 items-center justify-center rounded-xl", tones[tone])}>
          <Icon className="h-5 w-5" />
        </span>
      )}
    </Card>
  );
}
