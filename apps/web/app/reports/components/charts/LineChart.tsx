"use client";

import {
  LineChart as RechartsLineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from "recharts";

/** Konfigurasi satu line dalam chart */
export interface LineConfig {
  dataKey: string;
  name: string;
  color: string;
  /** Garis putus-putus (untuk proyeksi/forecast) */
  dashed?: boolean;
}

interface LineChartProps {
  /** Data array untuk chart */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: readonly any[];
  /** Key untuk sumbu X */
  xKey: string;
  /** Konfigurasi line(s) */
  lines: LineConfig[];
  /** Tinggi chart dalam pixel */
  height?: number;
  /** Formatter untuk tooltip value */
  valueFormatter?: (value: number) => string;
  /** Formatter untuk label sumbu X */
  xFormatter?: (value: string) => string;
  /** Garis referensi horizontal (contoh: target KPI) */
  referenceLine?: { value: number; label: string; color?: string };
  /** Tampilkan grid */
  showGrid?: boolean;
  /** Tampilkan legend */
  showLegend?: boolean;
}

const TOOLTIP_STYLE = {
  contentStyle: {
    borderRadius: "8px",
    border: "1px solid #e2e8f0",
    boxShadow: "0 4px 6px -1px rgb(0 0 0 / 0.1)",
    fontSize: "13px",
  },
};

export function LineChart({
  data,
  xKey,
  lines,
  height = 320,
  valueFormatter,
  xFormatter,
  referenceLine,
  showGrid = true,
  showLegend = true,
}: LineChartProps) {
  return (
    <div className="w-full overflow-x-auto">
      <div style={{ minWidth: Math.max(data.length * 60, 300) }}>
        <ResponsiveContainer width="100%" height={height}>
          <RechartsLineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
            {showGrid && <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />}
            <XAxis
              dataKey={xKey}
              tick={{ fontSize: 12, fill: "#64748b" }}
              tickFormatter={xFormatter}
              axisLine={{ stroke: "#e2e8f0" }}
              tickLine={false}
            />
            <YAxis
              tick={{ fontSize: 12, fill: "#64748b" }}
              tickFormatter={valueFormatter ? (v) => valueFormatter(v as number) : undefined}
              axisLine={false}
              tickLine={false}
              width={60}
            />
            <Tooltip
              {...TOOLTIP_STYLE}
              formatter={
                valueFormatter
                  ? (value: unknown) => [valueFormatter(Number(value ?? 0)), ""]
                  : undefined
              }
              labelFormatter={xFormatter ? (label: unknown) => xFormatter(String(label ?? "")) : undefined}
            />
            {showLegend && (
              <Legend
                wrapperStyle={{ fontSize: "12px", paddingTop: "8px" }}
              />
            )}
            {referenceLine && (
              <ReferenceLine
                y={referenceLine.value}
                label={{
                  value: referenceLine.label,
                  position: "right",
                  fontSize: 11,
                  fill: referenceLine.color ?? "#ef4444",
                }}
                stroke={referenceLine.color ?? "#ef4444"}
                strokeDasharray="6 4"
              />
            )}
            {lines.map((line) => (
              <Line
                key={line.dataKey}
                type="monotone"
                dataKey={line.dataKey}
                name={line.name}
                stroke={line.color}
                strokeWidth={2}
                strokeDasharray={line.dashed ? "6 4" : undefined}
                dot={{ r: 3, fill: line.color }}
                activeDot={{ r: 5 }}
              />
            ))}
          </RechartsLineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
