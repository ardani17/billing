"use client";

import {
  PieChart as RechartsPieChart,
  Pie,
  Cell,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";

interface PieChartDataItem {
  name: string;
  value: number;
  [key: string]: unknown;
}

interface PieChartProps {
  /** Data array untuk chart*/
  data: PieChartDataItem[];
  /** Tinggi chart dalam pixel*/
  height?: number;
  /** Formatter untuk tooltip value*/
  valueFormatter?: (value: number) => string;
  /** Tampilkan sebagai donut chart*/
  donut?: boolean;
  /** Tampilkan legend*/
  showLegend?: boolean;
  /** Warna kustom (opsional, bawaan menggunakan palet bawaan)*/
  colors?: string[];
}

const DEFAULT_COLORS = [
  "#3b82f6", // blue-500
  "#10b981", // emerald-500
  "#f59e0b", // amber-500
  "#ef4444", // red-500
  "#8b5cf6", // violet-500
  "#06b6d4", // cyan-500
  "#f97316", // orange-500
  "#ec4899", // pink-500
  "#14b8a6", // teal-500
  "#6366f1", // indigo-500
];

const TOOLTIP_STYLE = {
  contentStyle: {
    borderRadius: "8px",
    border: "1px solid #e2e8f0",
    boxShadow: "0 4px 6px -1px rgb(0 0 0 / 0.1)",
    fontSize: "13px",
  },
};

const RADIAN = Math.PI / 180;

function renderLabel(props: {
  cx?: number;
  cy?: number;
  midAngle?: number;
  innerRadius?: number;
  outerRadius?: number;
  percent?: number;
}) {
  const cx = props.cx ?? 0;
  const cy = props.cy ?? 0;
  const midAngle = props.midAngle ?? 0;
  const innerRadius = props.innerRadius ?? 0;
  const outerRadius = props.outerRadius ?? 0;
  const percent = props.percent ?? 0;

  if (percent < 0.05) return null;
  const radius = innerRadius + (outerRadius - innerRadius) * 0.5;
  const x = cx + radius * Math.cos(-midAngle * RADIAN);
  const y = cy + radius * Math.sin(-midAngle * RADIAN);
  return (
    <text
      x={x}
      y={y}
      fill="white"
      textAnchor="middle"
      dominantBaseline="central"
      fontSize={12}
      fontWeight={600}
    >
      {`${(percent * 100).toFixed(0)}%`}
    </text>
  );
}

export function PieChart({
  data,
  height = 320,
  valueFormatter,
  donut = false,
  showLegend = true,
  colors = DEFAULT_COLORS,
}: PieChartProps) {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <RechartsPieChart>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          innerRadius={donut ? "55%" : 0}
          outerRadius="80%"
          dataKey="value"
          nameKey="name"
          label={renderLabel}
          labelLine={false}
          strokeWidth={2}
          stroke="#fff"
        >
          {data.map((_, index) => (
            <Cell
              key={`cell-${index}`}
              fill={colors[index % colors.length]}
            />
          ))}
        </Pie>
        <Tooltip
          {...TOOLTIP_STYLE}
          formatter={
            valueFormatter
              ? (value: unknown) => [valueFormatter(Number(value ?? 0)), ""]
              : undefined
          }
        />
        {showLegend && (
          <Legend
            layout="horizontal"
            verticalAlign="bottom"
            wrapperStyle={{ fontSize: "12px", paddingTop: "8px" }}
          />
        )}
      </RechartsPieChart>
    </ResponsiveContainer>
  );
}
