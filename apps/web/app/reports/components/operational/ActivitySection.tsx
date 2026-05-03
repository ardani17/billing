"use client";

import type { ReportFilter } from "../../lib/types";
import { fetchActivityReport } from "../../lib/api";
import { formatNumber, formatPercentage } from "../../lib/formatters";
import { useReportData } from "../../hooks/useReportData";
import { EmptyState } from "../shared/EmptyState";

interface Props {
  filter: Partial<ReportFilter>;
}

export function ActivitySection({ filter }: Props) {
  const { data, loading, error } = useReportData({
    fetcher: () => fetchActivityReport(filter),
  });

  if (loading) return <Skeleton />;
  if (error) return <ErrorMsg message={error} />;
  if (!data) return <EmptyState />;

  const { per_user, top_actions } = data;

  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Aktivitas Admin</h2>

      {per_user.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Aktivitas per User</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 text-xs text-slate-500">
                  <th className="pb-2 pr-4">User</th>
                  <th className="pb-2 pr-4">Role</th>
                  <th className="pb-2 pr-4 text-right">Hari Login</th>
                  <th className="pb-2 pr-4 text-right">Total Aksi</th>
                  <th className="pb-2">Terakhir Aktif</th>
                </tr>
              </thead>
              <tbody>
                {per_user.map((u) => (
                  <tr key={u.user_id} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700">{u.user_name}</td>
                    <td className="py-2 pr-4 text-slate-600">{u.role}</td>
                    <td className="py-2 pr-4 text-right text-slate-600">{u.login_days}</td>
                    <td className="py-2 pr-4 text-right font-mono text-slate-900">{formatNumber(u.action_count)}</td>
                    <td className="py-2 text-slate-600">{new Date(u.last_active_at).toLocaleDateString("id-ID")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {top_actions.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="mb-3 text-sm font-medium text-slate-700">Aksi Terbanyak</h3>
          <div className="space-y-2">
            {top_actions.map((a) => (
              <div key={a.action_type} className="flex items-center justify-between text-sm">
                <span className="text-slate-700">{a.action_type}</span>
                <div className="flex items-center gap-3">
                  <span className="text-slate-500">{formatPercentage(a.percentage)}</span>
                  <span className="font-mono font-medium text-slate-900">{formatNumber(a.count)}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

function Skeleton() {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Aktivitas Admin</h2>
      <div className="h-48 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
    </section>
  );
}

function ErrorMsg({ message }: { message: string }) {
  return (
    <section className="space-y-4">
      <h2 className="text-lg font-semibold text-slate-900">Aktivitas Admin</h2>
      <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">
        Gagal memuat data: {message}
      </div>
    </section>
  );
}
