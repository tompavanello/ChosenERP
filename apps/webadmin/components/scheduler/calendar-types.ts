export type CalendarView = "month" | "week" | "day" | "list";

/** Item interativo do calendario (evento, escala, etc.). */
export type CalendarItem<T = unknown> = {
  id: string;
  title: string;
  starts_at: string;
  ends_at?: string;
  color: string;
  /** Linha secundaria (tipo, ministerio...). */
  subtitle?: string;
  /** Linha terciaria curta (ex.: "8/10 confirmados"). */
  meta?: string;
  /** Situacao usada para o estilo (planejado/publicado/cancelado...). */
  status?: string;
  /** Exibe o selo de conflito. */
  conflict?: boolean;
  draggable?: boolean;
  raw: T;
};

/** Item de contexto (evento da igreja) exibido em segundo plano. */
export type CalendarBackgroundItem = {
  id: string;
  title: string;
  starts_at: string;
  ends_at?: string;
  color?: string;
};
