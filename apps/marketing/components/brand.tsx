import { cn } from "@/lib/utils";

/** Marca do Chosen ERP: arco de igreja com cruz, em gradiente sky. */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      role="img"
      aria-label="Chosen ERP"
      className={cn("h-9 w-9", className)}
    >
      <defs>
        <linearGradient id="chosen-mark" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#0369a1" />
          <stop offset="55%" stopColor="#0284c7" />
          <stop offset="100%" stopColor="#38bdf8" />
        </linearGradient>
      </defs>
      <rect width="32" height="32" rx="9" fill="url(#chosen-mark)" />
      <path
        d="M15.2 3.4h1.6v2.1h2.1v1.6h-2.1v2.3h-1.6V7.1h-2.1V5.5h2.1z"
        fill="#fff"
      />
      <path d="M16 10.4 25.4 18H6.6z" fill="#fff" />
      <path
        d="M8 18h16v8.4a1.6 1.6 0 0 1-1.6 1.6H9.6A1.6 1.6 0 0 1 8 26.4z"
        fill="#fff"
        opacity="0.94"
      />
      <path
        d="M13.4 28v-5.1a2.6 2.6 0 0 1 5.2 0V28z"
        fill="#0369a1"
      />
    </svg>
  );
}

/** Marca + assinatura "Chosen ERP". */
export function Wordmark({
  className,
  compact = false,
}: {
  className?: string;
  compact?: boolean;
}) {
  return (
    <span className={cn("flex items-center gap-2.5", className)}>
      <LogoMark className={compact ? "h-8 w-8" : "h-9 w-9"} />
      <span className="flex flex-col leading-none">
        <span className="text-[1.05rem] font-bold tracking-tight text-slate-900">
          Chosen
        </span>
        <span className="text-[0.62rem] font-semibold uppercase tracking-[0.28em] text-sky-600">
          ERP
        </span>
      </span>
    </span>
  );
}
