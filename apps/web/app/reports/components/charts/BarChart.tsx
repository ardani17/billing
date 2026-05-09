"use client";

import {
  BarChart as RechartsBarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";

/** Konfigurasi satu bar dalam chart*/
export interface BarConfig {
  dataKey: string;
  name: string;
  color: string;
  stackId?: string;
}

interface BarChartProps {
  /** Data array untuk chart*/
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: readonly any[];
  /** Key untuk sumbu X*/
  xKey: string;
  /** Konfigurasi bar(s)*/
  bars: BarConfig[];
  /** Tinggi chart dalam pixel*/
  height?: number;
  /** Formatter untuk tooltip value*/
  valueFormatter?: (value: number) => string;
  /** Formatter untuk label sumbu X*/
  xFormatter?: (value: string) => string;
  /** Tampilkan grid*/
  showGrid?: boolean;
  /** Tampilkan legend*/
  showLegend?: boolean;
}

const TOOLTIP_STYLE = {
  contentStyle: {
    borderRadius: "8px",
    border: "1px solid #e2e8f0",
    boxShadow: "0 18px 40px -24px rgb(15 23 42 / 0.45)",
    fontSize: "13px",
    color: "#0f172a",
  },
};

export function BarChart({
  data,
  xKey,
  bars,
  height = 320,
  valueFormatter,
  xFormatter,
  showGrid = true,
  showLegend = true,
}: BarChartProps) {
  return (
    <div className="w-full overflow-x-auto rounded-md">
      <div style={{ minWidth: Math.max(data.length * 50, 300) }}>
        <ResponsiveContainer width="100%" height={height}>
          <RechartsBarChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
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
            {bars.map((bar) => (
              <Bar
                key={bar.dataKey}
                dataKey={bar.dataKey}
                name={bar.name}
                fill={bar.color}
                maxBarSize={42}
                stackId={bar.stackId}
                radius={bar.stackId ? undefined : [4, 4, 0, 0]}
              />
            ))}
          </RechartsBarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
