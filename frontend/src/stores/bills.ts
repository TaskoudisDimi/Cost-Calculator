import { defineStore } from 'pinia'
import { ref } from 'vue'
import { collection, query, where, onSnapshot, type Unsubscribe } from 'firebase/firestore'
import { db, auth } from '@/firebase'
import api from '@/api/client'
import type { Bill, UserProvider, Provider, DashboardSummary, UserProviderTemplate, ScanResult } from '@/types'


export const useBillsStore = defineStore('bills', () => {
  const bills = ref<Bill[]>([])
  const userProviders = ref<UserProvider[]>([])
  const providers = ref<Provider[]>([])
  const providerTemplates = ref<UserProviderTemplate[]>([])
  const dashboard = ref<DashboardSummary | null>(null)
  const loading = ref(false)

  async function fetchDashboard(month?: string) {
    const params = month ? `?month=${month}` : ''
    const { data } = await api.get<DashboardSummary>(`/dashboard${params}`)
    dashboard.value = data
  }

  async function fetchBills() {
    loading.value = true
    try {
      const { data } = await api.get<Bill[]>('/bills')
      bills.value = data
    } finally {
      loading.value = false
    }
  }

  async function createBill(payload: {
    user_provider_id: string
    amount: number
    due_date: string
    issued_date?: string
    notes?: string
  }) {
    const { data } = await api.post<Bill>('/bills', payload)
    bills.value.unshift(data)
    return data
  }

  async function updateBill(id: string, payload: Partial<Pick<Bill, 'amount' | 'due_date' | 'issued_date' | 'notes'>>) {
    const { data } = await api.put<Bill>(`/bills/${id}`, payload)
    const idx = bills.value.findIndex(b => b.id === id)
    if (idx !== -1) bills.value[idx] = data
    return data
  }

  async function markPaid(id: string) {
    const { data } = await api.patch<Bill>(`/bills/${id}/pay`)
    const idx = bills.value.findIndex(b => b.id === id)
    if (idx !== -1) bills.value[idx] = data
    await fetchDashboard()
    return data
  }

  async function deleteBill(id: string) {
    await api.delete(`/bills/${id}`)
    bills.value = bills.value.filter(b => b.id !== id)
  }

  async function fetchProviders() {
    const { data } = await api.get<Provider[]>('/providers')
    providers.value = data
  }

  async function fetchUserProviders() {
    const { data } = await api.get<UserProvider[]>('/user/providers')
    userProviders.value = data
  }

  async function addUserProvider(payload: {
    provider_id?: string
    custom_name?: string
    custom_category?: string
    custom_color?: string
    custom_payment_url?: string
    nickname?: string
    account_num?: string
  }) {
    const { data } = await api.post<UserProvider>('/user/providers', payload)
    userProviders.value.push(data)
    return data
  }

  async function deleteUserProvider(id: string) {
    await api.delete(`/user/providers/${id}`)
    userProviders.value = userProviders.value.filter(p => p.id !== id)
  }

  async function bulkDeleteBills(ids: string[]) {
    await api.delete('/bills', { data: { ids } })
    bills.value = bills.value.filter(b => !ids.includes(b.id))
  }

  async function fetchProviderTemplates() {
    const { data } = await api.get<UserProviderTemplate[]>('/user/provider-templates')
    providerTemplates.value = data
  }

  async function createProviderTemplate(payload: {
    name: string
    category?: string
    color?: string
    payment_url?: string
  }) {
    const { data } = await api.post<UserProviderTemplate>('/user/provider-templates', payload)
    providerTemplates.value.push(data)
    return data
  }

  async function deleteProviderTemplate(id: string) {
    await api.delete(`/user/provider-templates/${id}`)
    providerTemplates.value = providerTemplates.value.filter(t => t.id !== id)
  }

  async function scanBill(file: File): Promise<ScanResult> {
    const form = new FormData()
    form.append('file', file)
    const { data } = await api.post<ScanResult>('/bills/scan', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return data
  }

  // ── Real-time sync via Firestore onSnapshot ─────────────────────────────────
  let billsUnsub: Unsubscribe | null = null
  let providersUnsub: Unsubscribe | null = null

  function startRealtimeSync() {
    const uid = auth.currentUser?.uid
    if (!uid || billsUnsub) return

    let firstBills = true
    billsUnsub = onSnapshot(
      query(collection(db, 'bills'), where('user_id', '==', uid)),
      () => {
        if (firstBills) { firstBills = false; return }
        fetchBills()
      },
      err => console.error('[bills] sync error:', err),
    )

    let firstProviders = true
    providersUnsub = onSnapshot(
      query(collection(db, 'userProviders'), where('user_id', '==', uid)),
      () => {
        if (firstProviders) { firstProviders = false; return }
        fetchUserProviders()
      },
      err => console.error('[userProviders] sync error:', err),
    )
  }

  function stopRealtimeSync() {
    billsUnsub?.(); billsUnsub = null
    providersUnsub?.(); providersUnsub = null
  }

  return {
    bills, userProviders, providers, providerTemplates, dashboard, loading,
    fetchDashboard, fetchBills, createBill, updateBill, markPaid, deleteBill, bulkDeleteBills,
    fetchProviders, fetchUserProviders, addUserProvider, deleteUserProvider,
    fetchProviderTemplates, createProviderTemplate, deleteProviderTemplate,
    scanBill, startRealtimeSync, stopRealtimeSync,
  }
})
