export type ProviderCategory = 'energy' | 'water' | 'telecom' | 'streaming' | 'subscription' | 'housing' | 'finance' | 'car' | 'other'
export type BillStatus = 'pending' | 'paid' | 'overdue'

export interface User {
  id: string
  name: string
  email: string
}

export interface Provider {
  id: string
  name: string
  category: ProviderCategory
  payment_url: string
  logo_url: string
  color: string
  description: string
}

export interface UserProvider {
  id: string
  user_id: string
  provider_id: string
  provider: Provider
  nickname: string
  account_num: string
}

export interface Bill {
  id: string
  user_provider_id: string
  user_provider: UserProvider
  amount: number
  due_date: string
  issued_date: string | null
  status: BillStatus
  paid_at: string | null
  notes: string
  payment_code: string
  recurring: boolean
  created_at: string
  updated_at: string
}

export interface MonthStat {
  month: string
  bills: number
  expenses: number
  income: number
}

export interface AnalyticsSummary {
  year: string
  monthly: MonthStat[]
  by_category: Record<string, number>
  total_income: number
  total_bills: number
  total_expenses: number
  total_saved: number
}

export type ExpenseStatus = 'planned' | 'bought'
export type ExpenseCategory = 'shopping' | 'food' | 'transport' | 'health' | 'entertainment' | 'other'

export interface Expense {
  id: string
  user_id: string
  month: string
  name: string
  amount: number
  category: ExpenseCategory
  status: ExpenseStatus
  store: string
  notes: string
  planned_date: string | null
  bought_at: string | null
  created_at: string
  updated_at: string
}

export interface Income {
  id: string
  user_id: string
  description: string
  amount: number
  month: string
  created_at: string
  updated_at: string
}

export interface DashboardSummary {
  month: string
  total_income: number
  total_pending: number
  total_overdue: number
  total_paid: number
  amount_bills: number
  amount_overdue: number
  amount_expenses_planned: number
  total_expenses_planned: number
  remaining: number
  upcoming_bills: Bill[]
  planned_expenses: Expense[]
}

export interface UserProviderTemplate {
  id: string
  user_id: string
  name: string
  category: string
  color: string
  payment_url: string
}

export interface ScanResult {
  provider_name: string | null
  amount: number | null
  due_date: string | null
  issued_date: string | null
  notes: string | null
  payment_code: string | null
}

export interface UserCredential {
  id: string
  user_id: string
  label: string
  username: string
  password: string
  url: string
  notes: string
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface Settings {
  user: User
  
}