import type { BillStatus, ProviderCategory, ExpenseCategory, ExpenseStatus } from '@/types'

export function formatAmount(amount: number): string {
  return new Intl.NumberFormat('el-GR', { style: 'currency', currency: 'EUR' }).format(amount)
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
  return { pending: 'Εκκρεμής', paid: 'Πληρωμένος', overdue: 'Ληξιπρόθεσμος' }[status]
}

export function statusClass(status: BillStatus): string {
  return {
    pending: 'bg-blue-900/30 text-blue-400',
    paid: 'bg-green-900/30 text-green-400',
    overdue: 'bg-red-900/30 text-red-400',
  }[status]
}

export function categoryLabel(cat: string): string {
  const labels: Record<string, string> = {
    energy: 'Ρεύμα',
    water: 'Νερό',
    telecom: 'Τηλεπικοινωνίες',
    streaming: 'Streaming',
    subscription: 'Συνδρομές',
    housing: 'Κατοικία',
    finance: 'Οικονομικά',
    car: 'Αυτοκίνητο',
    other: 'Άλλο',
  }
  return labels[cat] ?? cat
}

export function expenseCategoryLabel(cat: ExpenseCategory): string {
  return {
    shopping: 'Ψώνια', food: 'Φαγητό', transport: 'Μεταφορά',
    health: 'Υγεία', entertainment: 'Ψυχαγωγία', other: 'Άλλο',
  }[cat]
}

export function expenseStatusLabel(s: ExpenseStatus): string {
  return { planned: 'Προγραμματισμένο', bought: 'Αγοράστηκε' }[s]
}

export function expenseStatusClass(s: ExpenseStatus): string {
  return { planned: 'bg-amber-900/30 text-amber-400', bought: 'bg-green-900/30 text-green-400' }[s]
}

export function currentMonth(): string {
  return new Date().toISOString().slice(0, 7)
}

export function monthLabel(month: string): string {
  const [year, m] = month.split('-')
  return new Intl.DateTimeFormat('el-GR', { month: 'long', year: 'numeric' })
    .format(new Date(Number(year), Number(m) - 1))
}
