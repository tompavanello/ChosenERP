import { forwardRef, type InputHTMLAttributes, type SelectHTMLAttributes, type TextareaHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => <input ref={ref} className={cn("input", className)} {...props} />
);
Input.displayName = "Input";

export const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, ...props }, ref) => <select ref={ref} className={cn("select", className)} {...props} />
);
Select.displayName = "Select";

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => <textarea ref={ref} className={cn("textarea", className)} {...props} />
);
Textarea.displayName = "Textarea";

export function Field({ label, required, children, className, hint }: { 
  label: string; 
  required?: boolean;
  children: React.ReactNode; 
  className?: string;
  hint?: string;
}) {
  return (
    <div className={cn("space-y-1", className)}>
      <label className="label text-xs">
        {label}
        {required && <span className="text-red-500 ml-0.5">*</span>}
      </label>
      {children}
      {hint && <p className="text-xs text-zinc-400">{hint}</p>}
    </div>
  );
}

// ---- Máscaras de input ----

const MASK_PATTERNS: Record<string, { pattern: RegExp; placeholder: string }> = {
  cpf: { pattern: /\d/, placeholder: "000.000.000-00" },
  cnpj: { pattern: /\d/, placeholder: "00.000.000/0001-00" },
  phone: { pattern: /\d/, placeholder: "(00) 00000-0000" },
  rg: { pattern: /\d/, placeholder: "0.000.000-SSP" },
  cep: { pattern: /\d/, placeholder: "00000-000" },
};

function applyMask(value: string, maskType: "cpf" | "cnpj" | "phone" | "rg" | "cep"): string {
  const digits = value.replace(/\D/g, "");
  
  switch (maskType) {
    case "cpf":
      if (digits.length <= 3) return digits;
      if (digits.length <= 6) return `${digits.slice(0, 3)}.${digits.slice(3)}`;
      if (digits.length <= 9) return `${digits.slice(0, 3)}.${digits.slice(3, 6)}.${digits.slice(6)}`;
      return `${digits.slice(0, 3)}.${digits.slice(3, 6)}.${digits.slice(6, 9)}-${digits.slice(9, 11)}`;
    
    case "cnpj":
      if (digits.length <= 2) return digits;
      if (digits.length <= 5) return `${digits.slice(0, 2)}.${digits.slice(2)}`;
      if (digits.length <= 8) return `${digits.slice(0, 2)}.${digits.slice(2, 5)}.${digits.slice(5)}`;
      if (digits.length <= 12) return `${digits.slice(0, 2)}.${digits.slice(2, 5)}.${digits.slice(5, 8)}/${digits.slice(8)}`;
      return `${digits.slice(0, 2)}.${digits.slice(2, 5)}.${digits.slice(5, 8)}/${digits.slice(8, 12)}-${digits.slice(12, 14)}`;
    
    case "phone":
      if (digits.length <= 2) return digits;
      if (digits.length <= 6) return `(${digits.slice(0, 2)}) ${digits.slice(2)}`;
      if (digits.length <= 10) return `(${digits.slice(0, 2)}) ${digits.slice(2, 6)}-${digits.slice(6)}`;
      return `(${digits.slice(0, 2)}) ${digits.slice(2, 7)}-${digits.slice(7, 11)}`;
    
    case "rg":
      if (digits.length <= 1) return digits;
      if (digits.length <= 4) return `${digits.slice(0, 1)}.${digits.slice(1)}`;
      if (digits.length <= 7) return `${digits.slice(0, 1)}.${digits.slice(1, 4)}.${digits.slice(4)}`;
      return `${digits.slice(0, 1)}.${digits.slice(1, 4)}.${digits.slice(4, 7)}-${digits.slice(7, 9)}`;
    
    case "cep":
      if (digits.length <= 5) return digits;
      return `${digits.slice(0, 5)}-${digits.slice(5, 8)}`;
    
    default:
      return value;
  }
}

export interface MaskedInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "onChange"> {
  mask?: "cpf" | "cnpj" | "phone" | "rg" | "cep" | "none";
  variant?: "standard" | "rg";
  value: string;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
}

export const MaskedInput = forwardRef<HTMLInputElement, MaskedInputProps>(
  ({ className, mask = "none", variant = "standard", value, onChange, ...props }, ref) => {
    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      if (mask !== "none") {
        const masked = applyMask(e.target.value, mask);
        e.target.value = masked;
        onChange(e);
      } else {
        onChange(e);
      }
    };

    return (
      <input
        ref={ref}
        className={cn("input", className)}
        value={value}
        onChange={handleInputChange}
        {...props}
      />
    );
  }
);
MaskedInput.displayName = "MaskedInput";
