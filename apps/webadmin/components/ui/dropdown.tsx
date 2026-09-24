"use client";

import { useState, useRef, useEffect, cloneElement, isValidElement, type ReactElement, type ReactNode } from "react";
import { cn } from "@/lib/utils";

export interface DropdownItem {
  label: string;
  icon?: ReactNode;
  danger?: boolean;
  onClick: () => void;
  disabled?: boolean;
}

export function Dropdown({
  trigger,
  triggerAsChild = false,
  items,
}: {
  trigger: ReactNode;
  /**
   * Marque quando o `trigger` JA e um botao (ex.: <Button>). Sem isso o Dropdown
   * o envolve no seu proprio <button>, e <button> dentro de <button> e HTML
   * invalido - o React acusa no console e quebra a hidratacao.
   * Nesse modo o clique do trigger passa a ser o do Dropdown (o handler do
   * filho e substituido).
   */
  triggerAsChild?: boolean;
  items: DropdownItem[];
}) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const toggle = () => setOpen((o) => !o);

  const triggerNode =
    triggerAsChild && isValidElement(trigger) ? (
      cloneElement(trigger as ReactElement<{ onClick?: () => void }>, { onClick: toggle })
    ) : (
      <button
        type="button"
        onClick={toggle}
        className="rounded-lg p-1.5 text-zinc-500 hover:bg-zinc-100"
      >
        {trigger}
      </button>
    );

  return (
    <div ref={containerRef} className="relative inline-block">
      {triggerNode}
      {open && (
        <div className="absolute top-full right-0 z-50 mt-1 w-44 overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-lg dark:border-zinc-800 dark:bg-zinc-900">
          <div className="py-1 text-sm">
            {items.map((item, i) => (
              <button
                key={i}
                type="button"
                disabled={item.disabled}
                onClick={() => {
                  setOpen(false);
                  item.onClick();
                }}
                className={cn(
                  "flex w-full items-center gap-2 px-3 py-1.5 text-left hover:bg-zinc-50 disabled:opacity-50",
                  item.danger && "hover:text-red-700",
                )}
              >
                {item.icon}
                <span>{item.label}</span>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
