import type { ReactNode } from "react";

type Tone = "slate" | "blue" | "green" | "amber" | "red" | "violet";

const toneClass: Record<Tone, string> = {
  slate: "bg-slate-100 text-slate-700 ring-slate-200",
  blue: "bg-blue-50 text-blue-700 ring-blue-200",
  green: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  amber: "bg-amber-50 text-amber-800 ring-amber-200",
  red: "bg-red-50 text-red-700 ring-red-200",
  violet: "bg-violet-50 text-violet-700 ring-violet-200",
};

export function statusTone(status: string): Tone {
  const normalized = status.toLowerCase();
  if (["aktif", "online", "lunas", "tersedia", "verified", "normal", "success"].includes(normalized)) return "green";
  if (["pending", "belum_bayar", "terjual", "bayar_sebagian", "trial", "rendah"].includes(normalized)) return "blue";
  if (["isolir", "terlambat", "degraded", "aktif_voucher", "warning", "sedang", "tinggi"].includes(normalized)) return "amber";
  if (["offline", "suspend", "suspended", "gagal", "blocked"].includes(normalized)) return "red";
  if (["prorate", "scale", "growth", "starter"].includes(normalized)) return "violet";
  return "slate";
}

export function StatusBadge({ status }: { status: string }) {
  const label = status.replaceAll("_", " ");
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold capitalize ring-1 ring-inset ${toneClass[statusTone(status)]}`}
    >
      {label}
    </span>
  );
}

export function PageHeader({
  eyebrow,
  title,
  description,
  actions,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-5 border-b border-slate-200 pb-6 md:flex-row md:items-end md:justify-between">
      <div className="min-w-0">
        {eyebrow && (
          <p className="text-xs font-semibold uppercase tracking-[0.18em] text-blue-700">
            {eyebrow}
          </p>
        )}
        <h1 className="mt-2 text-2xl font-semibold tracking-tight text-slate-950 [overflow-wrap:anywhere] sm:text-3xl">
          {title}
        </h1>
        {description && (
          <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-500">
            {description}
          </p>
        )}
      </div>
      {actions && <div className="flex min-w-0 flex-wrap gap-2">{actions}</div>}
    </div>
  );
}

export function Button({
  children,
  variant = "primary",
  href,
}: {
  children: ReactNode;
  variant?: "primary" | "secondary" | "ghost";
  href?: string;
}) {
  const className =
    variant === "primary"
      ? "bg-blue-600 text-white hover:bg-blue-700"
      : variant === "secondary"
        ? "border border-slate-300 bg-white text-slate-700 hover:bg-slate-50"
        : "text-slate-600 hover:bg-slate-100";

  if (href) {
    return (
      <a
        href={href}
        className={`inline-flex min-w-0 items-center justify-center rounded-md px-4 py-2 text-center text-sm font-semibold leading-5 transition active:scale-[0.98] ${className}`}
      >
        {children}
      </a>
    );
  }

  return (
    <button
      type="button"
      className={`inline-flex min-w-0 items-center justify-center rounded-md px-4 py-2 text-center text-sm font-semibold leading-5 transition active:scale-[0.98] ${className}`}
    >
      {children}
    </button>
  );
}

export function StatGrid({
  stats,
}: {
  stats: { label: string; value: string; delta?: string; tone?: Tone }[];
}) {
  return (
    <div className="grid gap-px overflow-hidden rounded-xl border border-slate-200 bg-slate-200 sm:grid-cols-2 xl:grid-cols-4">
      {stats.map((stat) => (
        <div key={stat.label} className="bg-white p-5">
          <p className="text-sm text-slate-500">{stat.label}</p>
          <div className="mt-3 flex items-end justify-between gap-4">
            <p className="font-mono text-2xl font-semibold tracking-tight text-slate-950">
              {stat.value}
            </p>
            {stat.delta && (
              <span className={`rounded-full px-2 py-1 text-xs font-semibold ring-1 ring-inset ${toneClass[stat.tone ?? "green"]}`}>
                {stat.delta}
              </span>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

export function Section({
  title,
  description,
  children,
  action,
}: {
  title: string;
  description?: string;
  children: ReactNode;
  action?: ReactNode;
}) {
  return (
    <section className="rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="flex flex-col gap-3 border-b border-slate-200 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="font-semibold tracking-tight text-slate-950">{title}</h2>
          {description && <p className="mt-1 text-sm text-slate-500">{description}</p>}
        </div>
        {action}
      </div>
      <div className="p-4 sm:p-5">{children}</div>
    </section>
  );
}

export function FilterBar({
  search = "Cari data...",
  filters = ["Status", "Area", "Paket"],
}: {
  search?: string;
  filters?: string[];
}) {
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-3 sm:flex-row">
      <input
        placeholder={search}
        className="h-10 min-w-0 flex-1 rounded-md border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
      />
      <div className="flex flex-wrap gap-2">
        {filters.map((filter) => (
          <select
            key={filter}
            className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-600 outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            defaultValue=""
          >
            <option value="">{filter}</option>
            <option>Aktif</option>
            <option>Pending</option>
            <option>Nonaktif</option>
          </select>
        ))}
        <Button variant="secondary">Reset</Button>
      </div>
    </div>
  );
}

export function DataTable({
  columns,
  rows,
}: {
  columns: string[];
  rows: (string | ReactNode)[][];
}) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <div className="divide-y divide-slate-100 lg:hidden">
        {rows.map((row, rowIndex) => (
          <div key={rowIndex} className="grid gap-3 p-4">
            {row.map((cell, cellIndex) => (
              <div key={cellIndex} className="grid min-w-0 grid-cols-[minmax(6.5rem,38%)_minmax(0,1fr)] gap-3 text-sm">
                <span className="text-xs font-semibold uppercase tracking-[0.1em] text-slate-400">
                  {columns[cellIndex] ?? "Data"}
                </span>
                <span className="min-w-0 text-right text-slate-700 [overflow-wrap:anywhere]">
                  {cell}
                </span>
              </div>
            ))}
          </div>
        ))}
      </div>
      <div className="hidden overflow-x-auto lg:block">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-slate-50">
            <tr>
              {columns.map((column) => (
                <th
                  key={column}
                  className="whitespace-nowrap px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-slate-500"
                >
                  {column}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {rows.map((row, rowIndex) => (
              <tr key={rowIndex} className="hover:bg-slate-50">
                {row.map((cell, cellIndex) => (
                  <td key={cellIndex} className="whitespace-nowrap px-4 py-3 text-slate-700">
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function FormField({
  label,
  helper,
  children,
}: {
  label: string;
  helper?: string;
  children: ReactNode;
}) {
  return (
    <label className="grid gap-2">
      <span className="text-sm font-medium text-slate-800">{label}</span>
      {children}
      {helper && <span className="text-xs leading-5 text-slate-500">{helper}</span>}
    </label>
  );
}

export function TextInput({ placeholder, defaultValue }: { placeholder?: string; defaultValue?: string }) {
  return (
    <input
      defaultValue={defaultValue}
      placeholder={placeholder}
      className="h-10 w-full min-w-0 rounded-md border border-slate-300 px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
    />
  );
}

export function SelectInput({ options }: { options: string[] }) {
  return (
    <select className="h-10 w-full min-w-0 rounded-md border border-slate-300 bg-white px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100">
      {options.map((option) => (
        <option key={option}>{option}</option>
      ))}
    </select>
  );
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-6 py-10 text-center">
      <h3 className="text-base font-semibold text-slate-950">{title}</h3>
      <p className="mx-auto mt-2 max-w-md text-sm leading-6 text-slate-500">{description}</p>
      {action && <div className="mt-5 flex justify-center">{action}</div>}
    </div>
  );
}
