"use client";

import { DataTable, EmptyState, Section, StatusBadge } from "../../../components/ui";
import type { PPPoEUser } from "../../lib/types";

export function PppoeUsersPanel({
  users,
  actionBusy,
  onToggle,
  onDisconnect,
  onDelete,
}: {
  users: PPPoEUser[];
  actionBusy: string | null;
  onToggle: (user: PPPoEUser, disabled: boolean) => void;
  onDisconnect: (userId: string) => void;
  onDelete: (user: PPPoEUser) => void;
}) {
  return (
    <Section title="PPPoE users terkelola" description="User yang dibuat atau disinkronkan ISPBoss ke router.">
      {users.length === 0 ? (
        <EmptyState
          title="Belum ada user terkelola"
          description="Saat pelanggan diaktivasi, ISPBoss akan membuat PPPoE secret dan mencatat status sync di sini."
        />
      ) : (
        <DataTable
          columns={["Username", "Profile", "Remote IP", "Sync", "Status", "Aksi"]}
          rows={users.map((user) => [
            user.username,
            user.profile_name,
            user.remote_address || "-",
            user.sync_status,
            <StatusBadge key={user.id} status={user.disabled ? "disabled" : user.status} />,
            <div key={`${user.id}-actions`} className="flex flex-wrap justify-end gap-2 lg:justify-start">
              <button
                type="button"
                disabled={actionBusy === `user-toggle:${user.id}`}
                onClick={() => onToggle(user, !user.disabled)}
                className="rounded-md px-3 py-2 text-sm font-semibold text-blue-700 hover:bg-blue-50 disabled:cursor-wait disabled:opacity-60"
              >
                {actionBusy === `user-toggle:${user.id}` ? "Menyimpan..." : user.disabled ? "Enable" : "Disable"}
              </button>
              <button
                type="button"
                disabled={actionBusy === `user-disconnect:${user.id}`}
                onClick={() => onDisconnect(user.id)}
                className="rounded-md px-3 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-100 disabled:cursor-wait disabled:opacity-60"
              >
                {actionBusy === `user-disconnect:${user.id}` ? "Memutus..." : "Disconnect"}
              </button>
              <button
                type="button"
                disabled={actionBusy === `user-delete:${user.id}`}
                onClick={() => onDelete(user)}
                className="rounded-md px-3 py-2 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60"
              >
                {actionBusy === `user-delete:${user.id}` ? "Menghapus..." : "Hapus"}
              </button>
            </div>,
          ])}
        />
      )}
    </Section>
  );
}
