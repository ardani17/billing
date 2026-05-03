"use client";

import { useState, useEffect, useCallback } from "react";
import type { Expense, ExpenseCategory, CreateExpenseRequest, UpdateExpenseRequest } from "../reports/lib/types";
import { fetchExpenses, fetchCategories, createExpense, updateExpense, deleteExpense } from "../reports/lib/api";
import { ExpenseTable } from "./components/ExpenseTable";
import { ExpenseForm } from "./components/ExpenseForm";
import { CategoryManager } from "./components/CategoryManager";

export default function ExpensePage() {
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [categories, setCategories] = useState<ExpenseCategory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingExpense, setEditingExpense] = useState<Expense | null>(null);
  const [showCategories, setShowCategories] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [expData, catData] = await Promise.all([fetchExpenses(), fetchCategories()]);
      setExpenses(expData);
      setCategories(catData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal memuat data");
    } finally {
      setLoading(false);
    }
  }, []);

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

  return (
    <div className="mx-auto max-w-7xl space-y-6 px-4 py-6 md:px-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Pengeluaran</h1>
          <p className="mt-1 text-sm text-slate-500">Kelola pengeluaran operasional ISP Anda</p>
        </div>
        <div className="flex gap-2">
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
        </div>
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-5 py-4 text-sm text-red-700">{error}</div>
      )}

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
        <ExpenseTable expenses={expenses} onEdit={handleEdit} onDelete={handleDelete} />
      )}
    </div>
  );
}
