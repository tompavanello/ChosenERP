import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export type Tone = "green" | "red" | "amber" | "sky" | "zinc" | "indigo" | "pink" | "orange" | "yellow";
type Variant = "solid" | "outline" | "soft";

const solidTones: Record<Tone, string> = {
  green: "bg-emerald-500 text-white",
  red: "bg-red-500 text-white",
  amber: "bg-amber-500 text-white",
  sky: "bg-sky-500 text-white",
  indigo: "bg-indigo-500 text-white",
  zinc: "bg-zinc-500 text-white",
  pink: "bg-pink-500 text-white",
  orange: "bg-orange-500 text-white",
  yellow: "bg-yellow-500 text-white",
};

const outlineTones: Record<Tone, string> = {
  green: "border border-emerald-200 text-emerald-700",
  red: "border border-red-200 text-red-700",
  amber: "border border-amber-200 text-amber-700",
  sky: "border border-sky-200 text-sky-700",
  indigo: "border border-indigo-200 text-indigo-700",
  zinc: "border border-zinc-200 text-zinc-600",
  pink: "border border-pink-200 text-pink-700",
  orange: "border border-orange-200 text-orange-700",
  yellow: "border border-yellow-200 text-yellow-700",
};

const softTones: Record<Tone, string> = {
  green: "bg-emerald-50 text-emerald-700",
  red: "bg-red-50 text-red-700",
  amber: "bg-amber-50 text-amber-700",
  sky: "bg-sky-50 text-sky-700",
  indigo: "bg-indigo-50 text-indigo-700",
  zinc: "bg-zinc-100 text-zinc-600",
  pink: "bg-pink-50 text-pink-700",
  orange: "bg-orange-50 text-orange-700",
  yellow: "bg-yellow-50 text-yellow-700",
};

const variantTones: Record<Variant, Record<Tone, string>> = {
  solid: solidTones,
  outline: outlineTones,
  soft: softTones,
};

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: Tone;
  variant?: Variant;
}

export function Badge({ className, tone = "zinc", variant = "soft", ...props }: BadgeProps) {
  const toneClasses = variantTones[variant][tone];
  const isOutline = variant === "outline";
  const baseClasses = isOutline 
    ? "inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-medium"
    : "inline-flex items-center gap-1 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-medium";
  
  return (
    <span className={cn(baseClasses, toneClasses, className)} {...props} />
  );
}
