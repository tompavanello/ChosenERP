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
}: ComboboxProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
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

  // Keyboard navigation
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      if (e.key === " " || e.key === "ArrowDown" || e.key === "Enter") {
        e.preventDefault();
        setOpen(true);
      }
      return;
    }
    if (e.key === "Escape") {
      setOpen(false);
      setSearch("");
      setActiveIndex(-1);
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIndex((i) => Math.min(i + 1, filtered.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (activeIndex >= 0 && activeIndex < filtered.length) {
        selectOption(filtered[activeIndex]);
      }
    }
  };

  const selectOption = (option: ComboboxOption) => {
    onChange(option.value);
    setOpen(false);
    setSearch("");
    setActiveIndex(-1);
  };

  return (
    <div ref={containerRef} className={cn("relative w-full", className)}>
      <div
        className={cn(
          "flex min-h-[34px] w-full cursor-pointer items-center justify-between rounded-lg border border-zinc-200 bg-white px-2.5 text-sm",
          !selectedLabel && "text-zinc-400"
        )}
        onClick={() => {
          setOpen(!open);
          setSearch("");
          setActiveIndex(-1);
          if (!open) {
            setTimeout(() => inputRef.current?.focus(), 100);
          }
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
                  className="h-7 w-full rounded-md border-zinc-300 pl-7 text-xs outline-none"
                  placeholder={searchPlaceholder}
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  onClick={(e) => e.stopPropagation()}
                />
              </div>
            </div>
          )}
          <div className={cn("overflow-y-auto", maxHeight)}>
            {filtered.length === 0 ? (
              <div className="p-3 text-center text-xs text-zinc-400">{emptyMessage}</div>
            ) : (
              filtered.map((option, idx) => (
                <div
                  key={option.value}
                  className={cn(
                    "flex cursor-pointer items-center justify-between px-3 py-1.5 text-xs",
                    activeIndex === idx && "bg-zinc-100",
                    option.value === (value ?? defaultValue) && "bg-sky-50 font-medium"
                  )}
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
