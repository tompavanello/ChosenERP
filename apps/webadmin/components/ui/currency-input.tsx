"use client";

import { forwardRef, useEffect, useState, type InputHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export interface CurrencyInputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
  /** Valor em formato numerico ("1234.56") ou "" quando vazio. */
  value: string;
  /** Devolve o valor numerico ("1234.56") ou "" quando o campo esta vazio. */
  onChange: (value: string) => void;
}

// Formata um valor numerico ("1234.56") como moeda pt-BR ("1.234,56").
function toBrl(numeric: string): string {
  if (numeric === "" || numeric == null) return "";
  const n = Number(numeric);
  if (Number.isNaN(n)) return "";
  return n.toLocaleString("pt-BR", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

/**
 * Input de moeda (BRL) com mascara centavos-primeiro: o usuario digita apenas
 * digitos e o campo exibe "1.234,56". O onChange entrega o valor numerico puro
 * ("1234.56"), compativel com os formularios que fazem Number(valor).
 */
export const CurrencyInput = forwardRef<HTMLInputElement, CurrencyInputProps>(
  ({ className, value, onChange, placeholder = "0,00", ...props }, ref) => {
    const [display, setDisplay] = useState(() => toBrl(value));

    // Sincroniza quando o valor muda por fora (prefill de edicao, reset, etc.).
    useEffect(() => {
      setDisplay(toBrl(value));
    }, [value]);

    function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
      const digits = e.target.value.replace(/\D/g, "");
      if (digits === "") {
        setDisplay("");
        onChange("");
        return;
      }
      const cents = Number(digits);
      const numeric = (cents / 100).toFixed(2);
      setDisplay(toBrl(numeric));
      onChange(numeric);
    }

    return (
      <input
        ref={ref}
        inputMode="decimal"
        className={cn("input", className)}
        value={display}
        onChange={handleChange}
        placeholder={placeholder}
        {...props}
      />
    );
  }
);
CurrencyInput.displayName = "CurrencyInput";
