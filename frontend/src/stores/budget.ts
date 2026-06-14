import { defineStore } from 'pinia'
import { ref } from 'vue'
import { collection, query, where, onSnapshot, type Unsubscribe } from 'firebase/firestore'
import { db, auth } from '@/firebase'
import api from '@/api/client'
import type { Expense, Income } from '@/types'

export const useBudgetStore = defineStore('budget', () => {
  const expenses = ref<Expense[]>([])
  const incomes = ref<Income[]>([])
  const loading = ref(false)

  let _syncMonth = ''
  let expensesUnsub: Unsubscribe | null = null
  let incomeUnsub: Unsubscribe | null = null

  async function fetchExpenses(month?: string) {
    if (month) _syncMonth = month
    loading.value = true
    try {
      const params = month ? `?month=${month}` : ''
      const { data } = await api.get<Expense[]>(`/expenses${params}`)
      expenses.value = data
    } finally {
      loading.value = false
    }
  }

  async function createExpense(payload: {
    name: string; amount: number; category?: string; month?: string
    store?: string; notes?: string; planned_date?: string
  }) {
    const { data } = await api.post<Expense>('/expenses', payload)
    expenses.value.unshift(data)
    return data
  }

  async function markBought(id: string) {
    const { data } = await api.patch<Expense>(`/expenses/${id}/buy`)
    const idx = expenses.value.findIndex(e => e.id === id)
    if (idx !== -1) expenses.value[idx] = data
    return data
  }

  async function deleteExpense(id: string) {
    await api.delete(`/expenses/${id}`)
    expenses.value = expenses.value.filter(e => e.id !== id)
  }

  async function fetchIncomes(month?: string) {
    if (month) _syncMonth = month
    const params = month ? `?month=${month}` : ''
    const { data } = await api.get<Income[]>(`/income${params}`)
    incomes.value = data
  }

  async function createIncome(payload: { description: string; amount: number; month: string }) {
    const { data } = await api.post<Income>('/income', payload)
    incomes.value.unshift(data)
    return data
  }

  async function deleteIncome(id: string) {
    await api.delete(`/income/${id}`)
    incomes.value = incomes.value.filter(i => i.id !== id)
  }

  async function bulkDeleteIncome(ids: string[]) {
    await api.delete('/income', { data: { ids } })
    incomes.value = incomes.value.filter(i => !ids.includes(i.id))
  }

  // ── Real-time sync: detect changes from other devices, refetch current month ─
  function startRealtimeSync() {
    const uid = auth.currentUser?.uid
    if (!uid || expensesUnsub) return

    let firstExp = true
    expensesUnsub = onSnapshot(
      query(collection(db, 'expenses'), where('user_id', '==', uid)),
      () => {
        if (firstExp) { firstExp = false; return }
        if (_syncMonth) fetchExpenses(_syncMonth)
      },
      err => console.error('[expenses] sync error:', err),
    )

    let firstInc = true
    incomeUnsub = onSnapshot(
      query(collection(db, 'income'), where('user_id', '==', uid)),
      () => {
        if (firstInc) { firstInc = false; return }
        if (_syncMonth) fetchIncomes(_syncMonth)
      },
      err => console.error('[income] sync error:', err),
    )
  }

  function stopRealtimeSync() {
    expensesUnsub?.(); expensesUnsub = null
    incomeUnsub?.(); incomeUnsub = null
  }

  return {
    expenses, incomes, loading,
    fetchExpenses, createExpense, markBought, deleteExpense,
    fetchIncomes, createIncome, deleteIncome, bulkDeleteIncome,
    startRealtimeSync, stopRealtimeSync,
  }
})
