import type { ElementType } from "react";
import { cn } from "@/lib/utils";
import { Card } from "./card";

export function StatCard({
  label,
  value,
  icon: Icon,
  hint,
  tone = "sky",
}: {
  label: string;
  value: string;
  icon?: ElementType;
  hint?: string;
  tone?: "sky" | "green" | "red" | "amber" | "zinc" | "pink" | "indigo";
}) {
  const tones: Record<string, string> = {
    sky: "text-sky-600 bg-sky-50",
    green: "text-emerald-600 bg-emerald-50",
    red: "text-red-600 bg-red-50",
    amber: "text-amber-600 bg-amber-50",
    zinc: "text-zinc-600 bg-zinc-100",
    pink: "text-pink-600 bg-pink-50",
    indigo: "text-indigo-600 bg-indigo-50",
  };
  return (
    <Card className="flex items-center justify-between p-3">
      <div>
        <p className="text-xs text-zinc-500">{label}</p>
        <p className="mt-0.5 text-xl font-semibold tracking-tight">{value}</p>
        {hint && <p className="mt-0.5 text-xs text-zinc-400">{hint}</p>}
      </div>
      {Icon && (
        <span className={cn("flex h-8 w-8 items-center justify-center rounded-lg", tones[tone])}>
          <Icon className="h-4 w-4" />
        </span>
      )}
    </Card>
  );
}
