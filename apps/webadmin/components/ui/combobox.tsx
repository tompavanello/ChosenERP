"use client";

import { useState, useRef, useEffect } from "react";
import { Check, ChevronDown, Search as SearchIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export interface ComboboxOption {
  value: string;
  label: string;
  tone?: string;
}

export interface ComboboxProps {
  value?: string;
  defaultValue?: string;
  placeholder?: string;
  options: ComboboxOption[];
  onChange: (value: string) => void;
  className?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  maxHeight?: string;
  showSearch?: boolean;
  /** Ao receber foco (Tab), abre a lista e joga o foco no campo de busca. */
  openOnFocus?: boolean;
}

export function Combobox({
  value,
  defaultValue,
  placeholder = "Selecione...",
  options,
  onChange,
  className,
  searchPlaceholder = "Buscar...",
  emptyMessage = "Nenhuma opcao encontrada",
  maxHeight = "max-h-60",
  showSearch = true,
  openOnFocus = false,
}: ComboboxProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [activeIndex, setActiveIndex] = useState(-1);

  const selectedLabel = options.find((o) => o.value === (value ?? defaultValue))?.label ?? "";

  const filtered = search
    ? options.filter(
        (o) =>
          o.label.toLowerCase().includes(search.toLowerCase()) ||
          o.value.toLowerCase().includes(search.toLowerCase())
      )
    : options;

  function close() {
    setOpen(false);
    setSearch("");
    setActiveIndex(-1);
  }

  function openList(seed = "") {
    setOpen(true);
    setSearch(seed);
    setActiveIndex(-1);
  }

  // Handle click outside
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setSearch("");
        setActiveIndex(-1);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  // Ao abrir, leva o foco para a busca (resolve o fluxo por teclado: Tab e ja
  // digitar, sem precisar clicar no campo).
  useEffect(() => {
    if (!open || !showSearch) return;
    const id = window.setTimeout(() => inputRef.current?.focus(), 0);
    return () => window.clearTimeout(id);
  }, [open, showSearch]);

  // Navegacao por teclado (vale para o gatilho e para o input de busca).
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        openList();
        return;
      }
      // Digitar com a lista fechada abre e ja filtra pela tecla.
      if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
        e.preventDefault();
        openList(e.key);
      }
      return;
    }
    if (e.key === "Escape") {
      e.preventDefault();
      close();
      triggerRef.current?.focus();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex((i) =>
        filtered.length === 0 ? -1 : Math.min(i + 1, filtered.length - 1)
      );
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex((i) => (filtered.length === 0 ? -1 : Math.max(i - 1, 0)));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const pick =
        activeIndex >= 0 && activeIndex < filtered.length ? filtered[activeIndex] : filtered[0];
      if (pick) selectOption(pick);
    }
  };

  const selectOption = (option: ComboboxOption) => {
    onChange(option.value);
    close();
    // Mantem o foco no gatilho para o Tab continuar dali (fluxo por teclado).
    triggerRef.current?.focus();
  };

  return (
    <div
      ref={containerRef}
      className={cn("relative w-full", className)}
      onBlur={(e) => {
        // relatedTarget nulo = o foco saiu para algo nao focavel (ex.: rolar a
        // lista/clicar numa opcao). Nao fecha: o mousedown global fecha apenas
        // quando o clique e realmente fora do combobox. Sem isso a lista
        // sumia antes do clique e nao dava para selecionar com o mouse.
        const next = e.relatedTarget as Node | null;
        if (next && !e.currentTarget.contains(next)) close();
      }}
    >
      <div
        ref={triggerRef}
        className={cn(
          "flex min-h-[34px] w-full cursor-pointer items-center justify-between rounded-lg border border-zinc-200 bg-white px-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-sky-500/30",
          !selectedLabel && "text-zinc-400"
        )}
        role="combobox"
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-autocomplete={showSearch ? "list" : "none"}
        onClick={() => {
          if (open) {
            close();
            triggerRef.current?.focus();
          } else {
            openList();
          }
        }}
        onFocus={() => {
          if (openOnFocus && !open) openList();
        }}
        onMouseDown={(e) => {
          // Com openOnFocus, o foco abriria a lista no mousedown e o clique
          // seguinte a fecharia na mesma hora. Evita o foco automatico no
          // gatilho (o clique/teclado controlam a abertura); o input continua
          // focavel normalmente.
          if (!openOnFocus) return;
          if ((e.target as HTMLElement).closest("input")) return;
          e.preventDefault();
        }}
        onKeyDown={onKeyDown}
        tabIndex={0}
      >
        <span className="truncate">{selectedLabel || placeholder}</span>
        <ChevronDown className={cn("h-4 w-4 transition-transform", open && "rotate-180")} />
      </div>

      {open && (
        <div className="absolute top-full z-50 mt-1 w-full overflow-hidden rounded-lg border border-zinc-200 bg-white shadow-lg">
          {showSearch && (
            <div className="border-b border-zinc-200 p-2">
              <div className="relative">
                <SearchIcon className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-400" />
                <input
                  ref={inputRef}
                  className="h-7 w-full rounded-md border border-zinc-300 pl-7 text-xs outline-none focus:ring-2 focus:ring-sky-500/30"
                  placeholder={searchPlaceholder}
                  aria-label={searchPlaceholder}
                  autoComplete="off"
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setActiveIndex(-1);
                  }}
                  onClick={(e) => e.stopPropagation()}
                />
              </div>
            </div>
          )}
          <div className={cn("overflow-y-auto", maxHeight)} role="listbox">
            {filtered.length === 0 ? (
              <div className="p-3 text-center text-xs text-zinc-400">{emptyMessage}</div>
            ) : (
              filtered.map((option, idx) => (
                <div
                  key={option.value}
                  role="option"
                  aria-selected={option.value === (value ?? defaultValue)}
                  className={cn(
                    "flex cursor-pointer items-center justify-between px-3 py-1.5 text-xs",
                    activeIndex === idx && "bg-zinc-100",
                    option.value === (value ?? defaultValue) && "bg-sky-50 font-medium"
                  )}
                  onMouseDown={(e) => e.preventDefault()}
                  onMouseEnter={() => setActiveIndex(idx)}
                  onClick={(e) => {
                    e.stopPropagation();
                    selectOption(option);
                  }}
                >
                  <span>{option.label}</span>
                  {option.value === (value ?? defaultValue) && <Check className="h-3.5 w-3.5 text-sky-600" />}
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
