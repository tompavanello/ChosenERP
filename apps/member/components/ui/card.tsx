import { cn } from "@/lib/utils";
import type { HTMLAttributes } from "react";

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("rounded-xl border border-[var(--line)] bg-[var(--card)] p-4 shadow-sm", className)}
      {...props}
    />
  );
}
