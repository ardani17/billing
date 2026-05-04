"use client";

import type { FormEvent } from "react";
import { FormField, Section, TextInput } from "../../../components/ui";
import type { RouterEditForm, RouterRecord } from "../../lib/types";

export function RouterEditPanel({
  router,
  editForm,
  saving,
  onCancel,
  onSubmit,
  onUpdate,
}: {
  router: RouterRecord;
  editForm: RouterEditForm;
  saving: boolean;
  onCancel: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onUpdate: (field: keyof RouterEditForm, value: string | boolean) => void;
}) {
  return (
    <Section
      title="Edit router"
      description="Mengubah data koneksi hanya menyimpan konfigurasi aplikasi. RouterOS tidak di-login sampai ada aksi manual."
    >
      <form onSubmit={onSubmit} className="grid gap-5">
        <div className="grid gap-4 lg:grid-cols-2">
          <FormField label="Nama router">
            <TextInput value={editForm.name} onChange={(event) => onUpdate("name", event.target.value)} required />
          </FormField>
          <FormField label="Host / IP">
            <TextInput value={editForm.host} onChange={(event) => onUpdate("host", event.target.value)} required />
          </FormField>
          <FormField label="Username">
            <TextInput value={editForm.username} onChange={(event) => onUpdate("username", event.target.value)} required />
          </FormField>
          <FormField label="Password baru" helper="Kosongkan jika password tidak berubah.">
            <TextInput
              type="password"
              value={editForm.password}
              onChange={(event) => onUpdate("password", event.target.value)}
              autoComplete="new-password"
              placeholder="Tidak diubah"
            />
          </FormField>
          <FormField label="Port API">
            <TextInput
              type="number"
              min={1}
              max={65535}
              value={editForm.port}
              onChange={(event) => onUpdate("port", event.target.value)}
              required
            />
          </FormField>
          <FormField label="Interval health check">
            <TextInput
              type="number"
              min={10}
              max={3600}
              value={editForm.healthCheckIntervalSec}
              onChange={(event) => onUpdate("healthCheckIntervalSec", event.target.value)}
              required
            />
          </FormField>
          <FormField label="Status">
            <select
              value={editForm.status}
              onChange={(event) => onUpdate("status", event.target.value)}
              className="h-10 w-full min-w-0 rounded-md border border-slate-300 bg-white px-3 text-sm outline-none transition focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            >
              <option value={router.status}>{router.status}</option>
              <option value="online">online</option>
              <option value="offline">offline</option>
              <option value="maintenance">maintenance</option>
            </select>
          </FormField>
          <label className="flex min-h-10 items-center gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 text-sm font-medium text-slate-700">
            <input
              type="checkbox"
              checked={editForm.useSsl}
              onChange={(event) => onUpdate("useSsl", event.target.checked)}
              className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
            />
            Gunakan API-SSL
          </label>
          <div className="lg:col-span-2">
            <FormField label="Catatan">
              <TextInput
                value={editForm.notes}
                onChange={(event) => onUpdate("notes", event.target.value)}
                placeholder="Lokasi, upstream, atau akses VPN"
              />
            </FormField>
          </div>
        </div>
        <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={onCancel}
            className="inline-flex min-w-0 items-center justify-center rounded-md border border-slate-300 bg-white px-4 py-2 text-center text-sm font-semibold leading-5 text-slate-700 transition hover:bg-slate-50 active:scale-[0.98]"
          >
            Batal
          </button>
          <button
            type="submit"
            disabled={saving}
            className="inline-flex min-w-0 items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-center text-sm font-semibold leading-5 text-white transition hover:bg-blue-700 active:scale-[0.98] disabled:cursor-wait disabled:opacity-60"
          >
            {saving ? "Menyimpan..." : "Simpan Perubahan"}
          </button>
        </div>
      </form>
    </Section>
  );
}
