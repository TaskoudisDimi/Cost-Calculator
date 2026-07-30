import { ref } from 'vue'
import { locale } from '@/composables/useLocale'
import type { BillStatus, ProviderCategory, ExpenseCategory, ExpenseStatus } from '@/types'

export const activeCurrency = ref('EUR')

export function formatAmount(amount: number): string {
  return new Intl.NumberFormat('el-GR', {
    style: 'currency',
    currency: activeCurrency.value,
  }).format(amount)
}

export function formatDate(date: string | null | undefined): string {
  if (!date) return '—'
  const d = new Date(date)
  if (isNaN(d.getTime())) return '—'
  try {
    return new Intl.DateTimeFormat('el-GR', { day: '2-digit', month: 'short', year: 'numeric' }).format(d)
  } catch {
    return date.slice(0, 10)
  }
}

export function statusLabel(status: BillStatus): string {
  const el = { pending: 'Εκκρεμής', paid: 'Πληρωμένος', overdue: 'Ληξιπρόθεσμος' }
  const en = { pending: 'Pending', paid: 'Paid', overdue: 'Overdue' }
  return (locale.value === 'en' ? en : el)[status]
}

export function statusClass(status: BillStatus): string {
  return {
    pending: 'bg-amber-400/10 text-amber-300 border border-amber-400/20',
    paid: 'bg-emerald-400/10 text-emerald-300 border border-emerald-400/20',
    overdue: 'bg-red-400/10 text-red-300 border border-red-400/20',
  }[status]
}

export function categoryLabel(cat: string): string {
  const el: Record<string, string> = { energy: 'Ρεύμα', water: 'Νερό', telecom: 'Τηλεπικοινωνίες', streaming: 'Streaming', subscription: 'Συνδρομές', housing: 'Κατοικία', finance: 'Οικονομικά', car: 'Αυτοκίνητο', other: 'Άλλο' }
  const en: Record<string, string> = { energy: 'Energy', water: 'Water', telecom: 'Telecom', streaming: 'Streaming', subscription: 'Subscriptions', housing: 'Housing', finance: 'Finance', car: 'Car', other: 'Other' }
  const map = locale.value === 'en' ? en : el
  return map[cat] ?? cat
}

export function expenseCategoryLabel(cat: ExpenseCategory): string {
  const el = { shopping: 'Ψώνια', food: 'Φαγητό', transport: 'Μεταφορά', health: 'Υγεία', entertainment: 'Ψυχαγωγία', other: 'Άλλο' }
  const en = { shopping: 'Shopping', food: 'Food', transport: 'Transport', health: 'Health', entertainment: 'Entertainment', other: 'Other' }
  return (locale.value === 'en' ? en : el)[cat]
}

export function expenseCategoryIcon(cat: ExpenseCategory): string {
  return {
    shopping: '🛍️', food: '🍽️', transport: '🚗',
    health: '💊', entertainment: '🎬', other: '📦',
  }[cat]
}

export function expenseStatusLabel(s: ExpenseStatus): string {
  const el = { planned: 'Προγραμματισμένο', bought: 'Αγοράστηκε' }
  const en = { planned: 'Planned', bought: 'Purchased' }
  return (locale.value === 'en' ? en : el)[s]
}

export function expenseStatusClass(s: ExpenseStatus): string {
  return { planned: 'bg-amber-900/30 text-amber-400', bought: 'bg-green-900/30 text-green-400' }[s]
}

export function currentMonth(): string {
  return new Date().toISOString().slice(0, 7)
}

export function monthLabel(month: string): string {
  const [year, m] = month.split('-')
  const loc = locale.value === 'en' ? 'en-GB' : 'el-GR'
  return new Intl.DateTimeFormat(loc, { month: 'long', year: 'numeric' })
    .format(new Date(Number(year), Number(m) - 1))
}
