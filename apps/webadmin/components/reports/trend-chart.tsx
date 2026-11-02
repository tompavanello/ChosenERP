"use client";

import { useId } from "react";
import {
  ResponsiveContainer, ComposedChart, Area, Bar, Line, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { AXIS_STROKE, AXIS_TICK, COLORS, GRID_STROKE, TOOLTIP_STYLE, currencyCompact } from "./chart-theme";

export type TrendSeries = {
  key: string;
  label: string;
  color?: string;
  type?: "line" | "area" | "bar";
};

/**
 * Tendencia no tempo: area/linha para evolucao, barras para comparar valores
 * lado a lado (ex.: entradas x saidas). Aceita series mistas.
 */
export function TrendChart({
  data,
  xKey = "name",
  series,
  height = 260,
  format = currencyCompact,
  yFormat,
}: {
  data: Array<Record<string, string | number>>;
  xKey?: string;
  series: TrendSeries[];
  height?: number;
  format?: (n: number) => string;
  yFormat?: (n: number) => string;
}) {
  const uid = useId().replace(/[:]/g, "");
  const tick = yFormat ?? format;

  return (
    <div style={{ height }}>
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <defs>
            {series.map((s, i) =>
              s.type === "area" ? (
                <linearGradient key={s.key} id={`grad-${uid}-${i}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={s.color ?? COLORS.sky} stopOpacity={0.35} />
                  <stop offset="95%" stopColor={s.color ?? COLORS.sky} stopOpacity={0} />
                </linearGradient>
              ) : null,
            )}
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke={GRID_STROKE} vertical={false} />
          <XAxis dataKey={xKey} tick={AXIS_TICK} stroke={AXIS_STROKE} />
          <YAxis tick={AXIS_TICK} stroke={AXIS_STROKE} width={64} tickFormatter={(v) => tick(Number(v))} />
          <Tooltip formatter={(v: unknown) => format(Number(v))} contentStyle={TOOLTIP_STYLE} />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          {series.map((s, i) => {
            if (s.type === "bar") return <Bar key={s.key} dataKey={s.key} name={s.label} fill={s.color ?? COLORS.sky} radius={[3, 3, 0, 0]} />;
            if (s.type === "area") {
              return (
                <Area
                  key={s.key}
                  type="monotone"
                  dataKey={s.key}
                  name={s.label}
                  stroke={s.color ?? COLORS.sky}
                  strokeWidth={2.5}
                  fill={`url(#grad-${uid}-${i})`}
                />
              );
            }
            return <Line key={s.key} type="monotone" dataKey={s.key} name={s.label} stroke={s.color ?? COLORS.sky} strokeWidth={2.5} dot={false} />;
          })}
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}
