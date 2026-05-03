"use client";

import { useState, useEffect } from "react";
import type { Expense, ExpenseCategory, CreateExpenseRequest, UpdateExpenseRequest } from "../../reports/lib/types";

interface ExpenseFormProps {
  categories: ExpenseCategory[];
  expense?: Expense | null;
  onSubmit: (data: CreateExpenseRequest | UpdateExpenseRequest) => Promise<void>;
  onCancel: () => void;
}

export function ExpenseForm({ categories, expense, onSubmit, onCancel }: ExpenseFormProps) {
  const [categoryId, setCategoryId] = useState(expense?.category_id ?? "");
  const [amount, setAmount] = useState(expense ? String(expense.amount) : "");
  const [description, setDescription] = useState(expense?.description ?? "");
  const [expenseDate, setExpenseDate] = useState(expense?.expense_date ?? new Date().toISOString().slice(0, 10));
  const [isRecurring, setIsRecurring] = useState(expense?.is_recurring ?? false);
  const [recurringDay, setRecurringDay] = useState(expense?.recurring_day ?? 1);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (expense) {
      setCategoryId(expense.category_id);
      setAmount(String(expense.amount));
      setDescription(expense.description);
      setExpenseDate(expense.expense_date);
      setIsRecurring(expense.is_recurring);
      setRecurringDay(expense.recurring_day ?? 1);
    }
  }, [expense]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!categoryId || !amount || !expenseDate) return;
    setSaving(true);
    setError(null);
    try {
      await onSubmit({
        category_id: categoryId,
        amount: Number(amount),
        description,
        expense_date: expenseDate,
        is_recurring: isRecurring,
        recurring_day: isRecurring ? recurringDay : undefined,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan");
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="rounded-xl border border-slate-200 bg-white p-5">
      <h3 className="mb-4 text-lg font-semibold text-slate-900">
        {expense ? "Edit Pengeluaran" : "Tambah Pengeluaran"}
      </h3>

      <div className="space-y-4">
        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700">Kategori</label>
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
            required
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            style={{ minHeight: 44 }}
          >
            <option value="">Pilih kategori</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700">Jumlah (Rp)</label>
          <input
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            required
            min={1}
            placeholder="100000"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            style={{ minHeight: 44 }}
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700">Keterangan</label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Opsional"
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            style={{ minHeight: 44 }}
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700">Tanggal</label>
          <input
            type="date"
            value={expenseDate}
            onChange={(e) => setExpenseDate(e.target.value)}
            required
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            style={{ minHeight: 44 }}
          />
        </div>

        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input
              type="checkbox"
              checked={isRecurring}
              onChange={(e) => setIsRecurring(e.target.checked)}
              className="h-4 w-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
            />
            Recurring (berulang)
          </label>
          {isRecurring && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-500">Tanggal:</span>
              <input
                type="number"
                value={recurringDay}
                onChange={(e) => setRecurringDay(Number(e.target.value))}
                min={1}
                max={28}
                className="w-16 rounded-lg border border-slate-300 px-2 py-1 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                style={{ minHeight: 44 }}
              />
            </div>
          )}
        </div>

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
        )}

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={saving}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            style={{ minHeight: 44 }}
          >
            {saving ? "Menyimpan..." : expense ? "Simpan Perubahan" : "Tambah"}
          </button>
          <button
            type="button"
            onClick={onCancel}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
            style={{ minHeight: 44 }}
          >
            Batal
          </button>
        </div>
      </div>
    </form>
  );
}
