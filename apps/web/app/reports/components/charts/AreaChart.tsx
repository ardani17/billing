"use client";

import {
  AreaChart as RechartsAreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";

/** Konfigurasi satu area dalam chart*/
export interface AreaConfig {
  dataKey: string;
  name: string;
  color: string;
  /** Opacity fill area (0-1)*/
  fillOpacity?: number;
  stackId?: string;
}

interface AreaChartProps {
  /** Data array untuk chart*/
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: readonly any[];
  /** Key untuk sumbu X*/
  xKey: string;
  /** Konfigurasi area(s)*/
  areas: AreaConfig[];
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
    boxShadow: "0 4px 6px -1px rgb(0 0 0 / 0.1)",
    fontSize: "13px",
  },
};

export function AreaChart({
  data,
  xKey,
  areas,
  height = 320,
  valueFormatter,
  xFormatter,
  showGrid = true,
  showLegend = true,
}: AreaChartProps) {
  return (
    <div className="w-full overflow-x-auto">
      <div style={{ minWidth: Math.max(data.length * 60, 300) }}>
        <ResponsiveContainer width="100%" height={height}>
          <RechartsAreaChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
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
            {areas.map((area) => (
              <Area
                key={area.dataKey}
                type="monotone"
                dataKey={area.dataKey}
                name={area.name}
                stroke={area.color}
                fill={area.color}
                fillOpacity={area.fillOpacity ?? 0.15}
                strokeWidth={2}
                stackId={area.stackId}
              />
            ))}
          </RechartsAreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
