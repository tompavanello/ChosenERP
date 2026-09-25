"use client";

import {
  useState, useRef, useEffect, cloneElement, isValidElement,
  type ReactElement, type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/utils";

export interface DropdownItem {
  label: string;
  icon?: ReactNode;
  danger?: boolean;
  onClick: () => void;
  disabled?: boolean;
}

// Largura do menu (w-44 = 11rem).
const MENU_WIDTH = 176;

/**
 * Menu de acoes.
 *
 * O menu e renderizado em um PORTAL, com posicionamento `fixed` calculado a
 * partir do gatilho. Antes ele era `absolute` dentro do gatilho e, quando usado
 * dentro de tabelas com `overflow` (DataTable), era cortado e fazia o grid
 * ganhar scroll em vez de aparecer. O portal escapa de qualquer ancestral com
 * overflow.
 */
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
  const [pos, setPos] = useState<{ top: number; left: number }>({ top: 0, left: 0 });
  const containerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // Reposiciona o menu a partir do retangulo do gatilho. Abre para baixo ou,
  // se nao couber, para cima.
  function place() {
    const el = containerRef.current;
    if (!el || typeof window === "undefined") return;
    const r = el.getBoundingClientRect();
    let left = r.right - MENU_WIDTH;
    left = Math.max(8, Math.min(left, window.innerWidth - MENU_WIDTH - 8));
    const menuH = menuRef.current?.offsetHeight ?? items.length * 32 + 12;
    let top = r.bottom + 4;
    if (top + menuH > window.innerHeight - 8) top = r.top - menuH - 4;
    setPos({ top, left });
  }

  useEffect(() => {
    if (!open) return;
    place();
    const onScrollResize = () => place();
    // capture=true pega o scroll de QUALQUER ancestral (ex.: o overflow da tabela).
    window.addEventListener("scroll", onScrollResize, true);
    window.addEventListener("resize", onScrollResize);
    return () => {
      window.removeEventListener("scroll", onScrollResize, true);
      window.removeEventListener("resize", onScrollResize);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      const t = e.target as Node;
      if (containerRef.current?.contains(t) || menuRef.current?.contains(t)) return;
      setOpen(false);
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

  const menu =
    open && typeof document !== "undefined"
      ? createPortal(
          <div
            ref={menuRef}
            style={{ position: "fixed", top: pos.top, left: pos.left, width: MENU_WIDTH }}
            className="z-[100] overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-lg dark:border-zinc-800 dark:bg-zinc-900"
          >
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
                    "flex w-full items-center gap-2 px-3 py-1.5 text-left hover:bg-zinc-50 disabled:opacity-50 dark:hover:bg-zinc-800",
                    item.danger && "text-red-600 hover:text-red-700",
                  )}
                >
                  {item.icon}
                  <span>{item.label}</span>
                </button>
              ))}
            </div>
          </div>,
          document.body,
        )
      : null;

  return (
    <div ref={containerRef} className="relative inline-block">
      {triggerNode}
      {menu}
    </div>
  );
}
