import { cn } from "@/lib/utils";
import type { ButtonHTMLAttributes } from "react";

type Variant = "primary" | "outline" | "ghost";
type Size = "sm" | "default";

export function Button({
  className,
  variant = "primary",
  size = "default",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant; size?: Size }) {
  const styles: Record<Variant, string> = {
    primary: "bg-[var(--brand)] text-white hover:bg-[var(--brand-strong)]",
    outline: "border border-[var(--line)] text-[var(--ink)] hover:bg-black/5",
    ghost: "text-[var(--muted)] hover:bg-black/5",
  };
  const sizes: Record<Size, string> = {
    default: "px-4 py-2.5 text-sm",
    sm: "px-2.5 py-1.5 text-xs",
  };
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-lg font-medium transition disabled:opacity-50",
        sizes[size],
        styles[variant],
        className,
      )}
      {...props}
    />
  );
}
