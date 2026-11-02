/**
 * Regras visuais compartilhadas dos relatorios.
 *
 * - Comparacao por comprimento (barras) e tendencia por tempo (linha/area).
 * - Pizza/rosca/radial NAO sao usadas: composicao vira barra 100% empilhada.
 * - Cores com significado fixo: entrada=verde, saida=vermelho, resultado=roxo.
 */

export const COLORS = {
  income: "#10b981",
  expense: "#ef4444",
  net: "#8b5cf6",
  sky: "#0ea5e9",
  indigo: "#6366f1",
  amber: "#f59e0b",
  teal: "#14b8a6",
  zinc: "#94a3b8",
} as const;

export const SERIES_COLORS = [
  "#0ea5e9",
  "#10b981",
  "#8b5cf6",
  "#f59e0b",
  "#ef4444",
  "#14b8a6",
  "#ec4899",
  "#6366f1",
  "#84cc16",
  "#f97316",
];

export const GRID_STROKE = "#e4e4e7";
export const AXIS_STROKE = "#9ca3af";
export const AXIS_TICK = { fontSize: 11 } as const;

/** Moeda compacta para eixos/KPIs (R$ 1,2 mi); o tooltip mostra o valor cheio. */
export function currencyCompact(n: number): string {
  const abs = Math.abs(n);
  if (abs > 0 && abs < 1000) {
    return n.toLocaleString("pt-BR", { style: "currency", currency: "BRL", maximumFractionDigits: 0 });
  }
  return n.toLocaleString("pt-BR", {
    style: "currency",
    currency: "BRL",
    notation: "compact",
    maximumFractionDigits: 1,
  });
}

export function numberCompact(n: number): string {
  if (Math.abs(n) < 1000) return n.toLocaleString("pt-BR");
  return n.toLocaleString("pt-BR", { notation: "compact", maximumFractionDigits: 1 });
}

export function pct(n: number, digits = 1): string {
  return `${n >= 0 ? "+" : ""}${n.toFixed(digits)}%`;
}

/** Estilo do tooltip do recharts, alinhado ao tema claro/escuro do app. */
export const TOOLTIP_STYLE = {
  borderRadius: 8,
  border: "1px solid #e4e4e7",
  fontSize: 12,
  boxShadow: "0 4px 16px rgba(0,0,0,0.08)",
} as const;
