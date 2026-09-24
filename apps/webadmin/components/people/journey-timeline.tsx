"use client";

import { Badge, type Tone } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { JOURNEY_STAGES, JOURNEY_ORDER } from "@/lib/constants";

const STAGE_TONES: Record<string, Tone> = {
  welcome: "zinc",
  coffee_pastor: "sky",
  course: "sky",
  cell: "indigo",
  converted: "green",
};

const DOT_COLORS: Record<Tone, string> = {
  green: "bg-emerald-500",
  red: "bg-red-500",
  amber: "bg-amber-500",
  sky: "bg-sky-500",
  indigo: "bg-indigo-500",
  zinc: "bg-zinc-500",
  pink: "bg-pink-500",
  orange: "bg-orange-500",
  yellow: "bg-yellow-500",
};

const DOT_BG_COLORS: Record<Tone, string> = {
  green: "bg-emerald-100",
  red: "bg-red-100",
  amber: "bg-amber-100",
  sky: "bg-sky-100",
  indigo: "bg-indigo-100",
  zinc: "bg-zinc-100",
  pink: "bg-pink-100",
  orange: "bg-orange-100",
  yellow: "bg-yellow-100",
};

const TEXT_COLORS: Record<Tone, string> = {
  green: "text-emerald-800",
  red: "text-red-800",
  amber: "text-amber-800",
  sky: "text-sky-800",
  indigo: "text-indigo-800",
  zinc: "text-zinc-800",
  pink: "text-pink-800",
  orange: "text-orange-800",
  yellow: "text-yellow-800",
};

export function JourneyTimeline({
  stage,
  onChange,
  compact = false,
}: {
  stage: string;
  onChange?: (stage: string) => void;
  compact?: boolean;
}) {
  const idx = JOURNEY_ORDER.indexOf(stage);
  return (
    <div className={cn("flex items-center", compact ? "gap-0.5" : "gap-2")}>
      {JOURNEY_ORDER.map((s, i) => {
        const st = JOURNEY_STAGES[s] ?? { label: s, tone: "zinc", order: i };
        const tone = (STAGE_TONES[s] ?? "zinc") as Tone;
        const reached = i <= idx;
        const active = i === idx;
        return (
          <button
            key={s}
            type="button"
            onClick={() => onChange?.(s)}
            disabled={!onChange}
            className={cn(
              "flex items-center gap-0.5 rounded-full font-medium transition",
              compact ? "px-1 py-0.5 text-[9px]" : "px-2 py-1 text-xs",
              active && "ring-2 ring-sky-500",
              reached
                ? `${DOT_BG_COLORS[tone]} ${TEXT_COLORS[tone]}`
                : "bg-zinc-100 text-zinc-400",
            )}
            title={st.label}
          >
            <span className={cn("h-1.5 w-1.5 rounded-full", reached ? DOT_COLORS[tone] : "bg-zinc-300")} />
            {!compact && <span>{st.label}</span>}
            {compact && i < JOURNEY_ORDER.length - 1 && <span className="text-zinc-400">›</span>}
          </button>
        );
      })}
    </div>
  );
}

export function JourneyStageBadge({ stage }: { stage: string }) {
  const st = JOURNEY_STAGES[stage] ?? { label: stage, tone: "zinc" };
  return (
    <Badge tone={(st.tone as Tone) ?? "zinc"} className="text-[10px] whitespace-nowrap">
      {st.label}
    </Badge>
  );
}
