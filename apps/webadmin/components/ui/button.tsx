import { forwardRef, type ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

type Variant = "primary" | "ghost" | "outline";
type Size = "sm" | "default" | "lg";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
}

const sizeMap: Record<Size, string> = {
  sm: "px-2.5 py-1.5 text-xs rounded-md",
  default: "px-4 py-2 text-sm rounded-lg",
  lg: "px-5 py-2.5 text-base rounded-lg",
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "primary", size = "default", ...props }, ref) => {
    const v = variant === "ghost" ? "btn-ghost" : variant === "outline" ? "btn-outline" : "btn-primary";
    return (
      <button ref={ref} className={cn("btn-base", v, sizeMap[size], className)} {...props} />
    );
  }
);
Button.displayName = "Button";
