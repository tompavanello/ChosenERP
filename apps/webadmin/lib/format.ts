import { format, formatDistanceToNow, parseISO } from "date-fns";
import { ptBR } from "date-fns/locale";

export function currency(n: number, opts: Intl.NumberFormatOptions = {}) {
  return n.toLocaleString("pt-BR", { style: "currency", currency: "BRL", minimumFractionDigits: 2, ...opts });
}

export function number(n: number) {
  return n.toLocaleString("pt-BR");
}

export function datePt(s?: string | null) {
  if (!s) return "—";
  try {
    return format(parseISO(s), "dd/MM/yyyy", { locale: ptBR });
  } catch {
    return s;
  }
}

export function dateTimePt(s?: string | null) {
  if (!s) return "—";
  try {
    return format(parseISO(s), "dd/MM/yyyy 'às' HH:mm", { locale: ptBR });
  } catch {
    return s;
  }
}

export function relativePt(s?: string | null) {
  if (!s) return "—";
  try {
    return formatDistanceToNow(parseISO(s), { addSuffix: true, locale: ptBR });
  } catch {
    return s;
  }
}

export function initials(name: string) {
  const p = name.trim().split(/\s+/).filter(Boolean);
  if (p.length === 0) return "?";
  return (p[0][0] + (p.length > 1 ? p[p.length - 1][0] : "")).toUpperCase();
}

export function monthLabel(m: string) {
  const [y, mo] = m.split("-");
  const nomes = ["jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"];
  return `${nomes[Number(mo) - 1]}/${y}`;
}
