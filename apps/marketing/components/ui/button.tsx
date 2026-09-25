import type {
  AnchorHTMLAttributes,
  ButtonHTMLAttributes,
  ReactNode,
} from "react";
import { cn } from "@/lib/utils";

type Variant = "primary" | "outline" | "ghost" | "dark" | "white";
type Size = "sm" | "md" | "lg";

const base =
  "inline-flex items-center justify-center gap-2 rounded-xl font-semibold transition-all duration-200 focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-60 disabled:pointer-events-none whitespace-nowrap";

const variants: Record<Variant, string> = {
  primary:
    "bg-sky-600 text-white shadow-lg shadow-sky-600/25 hover:bg-sky-700 hover:shadow-sky-700/30",
  outline:
    "border border-slate-300 bg-white text-slate-800 hover:border-sky-400 hover:text-sky-700 hover:bg-sky-50",
  ghost: "text-slate-700 hover:bg-slate-100 hover:text-slate-900",
  dark: "bg-slate-900 text-white hover:bg-slate-800",
  white: "bg-white text-sky-700 shadow-lg shadow-sky-950/20 hover:bg-sky-50",
};

const sizes: Record<Size, string> = {
  sm: "px-3.5 py-2 text-sm",
  md: "px-5 py-2.5 text-sm",
  lg: "px-6 py-3.5 text-base",
};

type Common = { variant?: Variant; size?: Size; className?: string; children: ReactNode };

type Props =
  | (Common &
      { href: string } &
      Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "className" | "children">)
  | (Common &
      { href?: undefined } &
      Omit<ButtonHTMLAttributes<HTMLButtonElement>, "className" | "children">);

export function Button(props: Props) {
  const { variant = "primary", size = "md", className, children } = props;
  const classes = cn(base, variants[variant], sizes[size], className);

  if (props.href !== undefined) {
    const { variant: _v, size: _s, className: _c, children: _ch, href, ...rest } =
      props;
    const external = href.startsWith("http");
    return (
      <a
        href={href}
        className={classes}
        {...(external ? { target: "_blank", rel: "noopener noreferrer" } : {})}
        {...rest}
      >
        {children}
      </a>
    );
  }

  const { variant: _v, size: _s, className: _c, children: _ch, ...rest } = props;
  return (
    <button className={classes} {...rest}>
      {children}
    </button>
  );
}
