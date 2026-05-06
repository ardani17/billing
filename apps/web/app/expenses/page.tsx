"use client";

import { useState, useEffect, useCallback } from "react";
import type { Expense, ExpenseCategory, CreateExpenseRequest, UpdateExpenseRequest } from "../reports/lib/types";
import { fetchExpenses, fetchCategories, createExpense, updateExpense, deleteExpense } from "../reports/lib/api";
import { ExpenseTable } from "./components/ExpenseTable";
import { ExpenseForm } from "./components/ExpenseForm";
import { CategoryManager } from "./components/CategoryManager";
import AppShell from "../components/app-shell";
import { PageHeader, Section, StatGrid } from "../components/ui";

function monthStart() {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), 1).toISOString().slice(0, 10);
}

function today() {
  return new Date().toISOString().slice(0, 10);
}

export default function ExpensePage() {
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [categories, setCategories] = useState<ExpenseCategory[]>([]);
  const [periodStart, setPeriodStart] = useState(monthStart());
  const [periodEnd, setPeriodEnd] = useState(today());
  const [categoryFilter, setCategoryFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingExpense, setEditingExpense] = useState<Expense | null>(null);
  const [showCategories, setShowCategories] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [expData, catData] = await Promise.all([fetchExpenses(periodStart, periodEnd, categoryFilter), fetchCategories()]);
      setExpenses(expData);
      setCategories(catData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memuat data");
    } finally {
      setLoading(false);
    }
  }, [categoryFilter, periodEnd, periodStart]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleCreate = async (data: CreateExpenseRequest | UpdateExpenseRequest) => {
    await createExpense(data as CreateExpenseRequest);
    setShowForm(false);
    loadData();
  };

  const handleUpdate = async (data: CreateExpenseRequest | UpdateExpenseRequest) => {
    if (!editingExpense) return;
    await updateExpense(editingExpense.id, data as UpdateExpenseRequest);
    setEditingExpense(null);
    setShowForm(false);
    loadData();
  };

  const handleDelete = async (id: string) => {
    await deleteExpense(id);
    loadData();
  };

  const handleEdit = (expense: Expense) => {
    setEditingExpense(expense);
    setShowForm(true);
  };

  const total = expenses.reduce((sum, expense) => sum + expense.amount, 0);
  const recurring = expenses.filter((expense) => expense.is_recurring).length;

  return (
    <AppShell>
      <div className="space-y-6">
        <PageHeader
          eyebrow="Keuangan"
          title="Pengeluaran"
          description="Catat biaya operasional untuk laporan laba rugi dan arus kas tenant."
          actions={
            <>
          <button
            type="button"
            onClick={() => setShowCategories(!showCategories)}
            className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
            style={{ minHeight: 44 }}
          >
            Kelola Kategori
          </button>
          <button
            type="button"
            onClick={() => { setEditingExpense(null); setShowForm(true); }}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            style={{ minHeight: 44 }}
          >
            Tambah Pengeluaran
          </button>
            </>
          }
        />

        <StatGrid
          stats={[
            { label: "Total pengeluaran", value: new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(total), tone: "red" },
            { label: "Jumlah transaksi", value: String(expenses.length), tone: "blue" },
            { label: "Recurring", value: String(recurring), tone: "amber" },
            { label: "Kategori", value: String(categories.length), tone: "slate" },
          ]}
        />

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">{error}</div>
      )}

      <Section title="Filter" description="Gunakan periode dan kategori untuk rekonsiliasi biaya.">
        <div className="grid gap-3 md:grid-cols-4">
          <label className="grid gap-2 text-sm font-medium text-slate-700">
            Dari tanggal
            <input type="date" value={periodStart} onChange={(event) => setPeriodStart(event.target.value)} className="h-10 rounded-md border border-slate-300 px-3 text-sm" />
          </label>
          <label className="grid gap-2 text-sm font-medium text-slate-700">
            Sampai tanggal
            <input type="date" value={periodEnd} onChange={(event) => setPeriodEnd(event.target.value)} className="h-10 rounded-md border border-slate-300 px-3 text-sm" />
          </label>
          <label className="grid gap-2 text-sm font-medium text-slate-700 md:col-span-2">
            Kategori
            <select value={categoryFilter} onChange={(event) => setCategoryFilter(event.target.value)} className="h-10 rounded-md border border-slate-300 bg-white px-3 text-sm">
              <option value="">Semua kategori</option>
              {categories.map((category) => (
                <option key={category.id} value={category.id}>{category.name}</option>
              ))}
            </select>
          </label>
        </div>
      </Section>

      {showCategories && (
        <CategoryManager categories={categories} onUpdate={loadData} />
      )}

      {showForm && (
        <ExpenseForm
          categories={categories}
          expense={editingExpense}
          onSubmit={editingExpense ? handleUpdate : handleCreate}
          onCancel={() => { setShowForm(false); setEditingExpense(null); }}
        />
      )}

      {loading ? (
        <div className="h-48 animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
      ) : (
        <Section title="Daftar pengeluaran">
          <ExpenseTable expenses={expenses} onEdit={handleEdit} onDelete={handleDelete} />
        </Section>
      )}
      </div>
    </AppShell>
  );
}
