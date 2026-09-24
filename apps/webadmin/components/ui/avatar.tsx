"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";
import { initials } from "@/lib/format";

export function Avatar({
  name,
  src,
  size = "md",
  className,
}: {
  name: string;
  /** Foto do membro. Se faltar ou falhar ao carregar, cai nas iniciais. */
  src?: string | null;
  size?: "sm" | "md" | "lg" | "xl";
  className?: string;
}) {
  const [broken, setBroken] = useState(false);
  const sizes = {
    sm: "h-8 w-8 text-xs",
    md: "h-10 w-10 text-sm",
    lg: "h-16 w-16 text-lg",
    xl: "h-24 w-24 text-2xl",
  };
  const showPhoto = Boolean(src) && !broken;

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-sky-600 font-semibold text-white ring-1 ring-black/5 dark:ring-white/10",
        sizes[size],
        className,
      )}
    >
      {showPhoto ? (
        // <img> simples de propósito: next/image exigiria declarar o host da API
        // em remotePatterns, e a URL da foto é montada em runtime.
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={src as string}
          alt={name}
          loading="lazy"
          onError={() => setBroken(true)}
          className="h-full w-full object-cover"
        />
      ) : (
        initials(name)
      )}
    </span>
  );
}
