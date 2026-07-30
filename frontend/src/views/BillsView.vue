<script setup lang="ts">
import { ref, onMounted, computed, unref } from 'vue'
import { useBillsStore } from '@/stores/bills'
import { useMembersStore } from '@/stores/members'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { useLocale } from '@/composables/useLocale'
import { formatDate, formatAmount, statusLabel, statusClass, monthLabel } from '@/utils/format'
import type { Bill, BillStatus } from '@/types'
import QRCode from 'qrcode'

const { toast } = useToast()
const { confirm } = useConfirm()
const { t, locale } = useLocale()

const store = useBillsStore()
const membersStore = useMembersStore()

const showModal = ref(false)
const editingBill = ref<string | null>(null)
const filterStatus = ref<BillStatus | 'all'>('all')
const filterMember = ref<string>('all')
const selected = ref<Set<string>>(new Set())
const saving = ref(false)
const viewMode = ref<'list' | 'history'>('list')
const filterOpen = ref(false)
const selectionMode = ref(false)

const scanning = ref(false)
const scanError = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const dateFrom = ref('')
const dateTo = ref('')
const copiedId = ref<string | null>(null)

const showEmailInput = ref(false)
const emailContent = ref('')
const parsingEmail = ref(false)
const emailError = ref('')

const qrBill = ref<Bill | null>(null)
const qrDataUrl = ref('')

const notifSupported = ref('Notification' in window)
const notifDismissed = ref(localStorage.getItem('notif_dismissed') === '1')
const notifGranted = ref('Notification' in window && Notification.permission === 'granted')

const form = ref({
  user_provider_id: '',
  member_id: '',
  amount: '',
  due_date: '',
  issued_date: '',
  notes: '',
  payment_code: '',
  recurring: false,
})

const billsByMonth = computed(() => {
  const all = [...(unref(store.bills) ?? [])]
  all.sort((a, b) => new Date(b.due_date).getTime() - new Date(a.due_date).getTime())
  const groups: Record<string, typeof all> = {}
  for (const b of all) {
    const key = b.due_date.slice(0, 7)
    if (!groups[key]) groups[key] = []
    groups[key].push(b)
  }
  return Object.entries(groups).sort((a, b) => b[0].localeCompare(a[0]))
})

// Map: provider_id → previous bill amount (second-latest bill per provider)
const priceAlerts = computed(() => {
  const all = unref(store.bills)
  if (!Array.isArray(all)) return new Map<string, number>()
  // group by user_provider_id, sort by due_date desc
  const byProvider = new Map<string, typeof all>()
  for (const b of all) {
    const pid = b.user_provider_id
    if (!pid) continue
    if (!byProvider.has(pid)) byProvider.set(pid, [])
    byProvider.get(pid)!.push(b)
  }
  const alerts = new Map<string, number>() // bill.id → previous amount
  for (const [, bills] of byProvider) {
    const sorted = [...bills].sort((a, b) => new Date(b.due_date).getTime() - new Date(a.due_date).getTime())
    if (sorted.length < 2) continue
    const latest = sorted[0]!
    const prev = sorted[1]!
    if (latest.amount > prev.amount * 1.1) { // >10% increase
      alerts.set(latest.id, prev.amount)
    }
  }
  return alerts
})

const filteredBills = computed(() => {
  let all = unref(store.bills)
  if (!Array.isArray(all)) return []
  if (filterStatus.value !== 'all') all = all.filter(b => b.status === filterStatus.value)
  if (filterMember.value !== 'all') {
    if (filterMember.value === 'none') {
      all = all.filter(b => !b.member_id)
    } else {
      all = all.filter(b => b.member_id === filterMember.value)
    }
  }
  if (dateFrom.value) {
    const from = new Date(dateFrom.value)
    all = all.filter(b => new Date(b.due_date) >= from)
  }
  if (dateTo.value) {
    const to = new Date(dateTo.value)
    to.setHours(23, 59, 59, 999)
    all = all.filter(b => new Date(b.due_date) <= to)
  }
  return all
})

function clearDateFilter() {
  dateFrom.value = ''
  dateTo.value = ''
  filterMember.value = 'all'
  filterOpen.value = false
  clearSelection()
}

function exitSelectionMode() {
  selectionMode.value = false
  selected.value = new Set()
}

const allSelected = computed(() => {
  const list = filteredBills.value
  return list.length > 0 && list.every(b => selected.value.has(b.id))
})

onMounted(async () => {
  await Promise.all([store.fetchBills(), store.fetchUserProviders(), membersStore.fetchMembers()])
})

function toggleSelect(id: string) {
  const s = new Set(selected.value)
  s.has(id) ? s.delete(id) : s.add(id)
  selected.value = s
}

function toggleSelectAll() {
  if (allSelected.value) {
    selected.value = new Set()
  } else {
    selected.value = new Set(filteredBills.value.map(b => b.id))
  }
}

function clearSelection() {
  selected.value = new Set()
}

async function bulkDelete() {
  const ids = [...selected.value]
  const msg = locale.value === 'en'
    ? `${ids.length} bills will be deleted. This cannot be undone.`
    : `Θα διαγραφούν ${ids.length} λογαριασμοί. Η ενέργεια δεν αναιρείται.`
  if (!await confirm({ message: msg })) return
  await store.bulkDeleteBills(ids)
  selected.value = new Set()
}

async function copyPaymentCode(code: string, id: string) {
  await navigator.clipboard.writeText(code)
  copiedId.value = id
  setTimeout(() => { copiedId.value = null }, 2000)
}

async function showQR(bill: Bill) {
  qrBill.value = bill
  const content = bill.payment_code || `${bill.user_provider?.nickname || bill.user_provider?.provider?.name || ''}`
  try {
    qrDataUrl.value = await QRCode.toDataURL(content, { width: 256, margin: 2, color: { dark: '#1f2937', light: '#f9fafb' } })
  } catch {
    qrDataUrl.value = ''
  }
}

async function payWithIRIS(bill: Bill) {
  const code = bill.payment_code
  if (!code) return
  // Try Web Share API on mobile first (most seamless UX)
  if (navigator.share) {
    try {
      await navigator.share({ title: t('bills.pay_iris'), text: code })
      return
    } catch { /* user cancelled or not supported */ }
  }
  // Fallback: show QR modal
  await showQR(bill)
}

async function enableNotifications() {
  if (!notifSupported.value) return
  try {
    const permission = await Notification.requestPermission()
    if (permission === 'granted') {
      notifGranted.value = true
      notifDismissed.value = true
      localStorage.setItem('notif_dismissed', '1')
      toast(locale.value === 'en' ? 'Reminders enabled' : 'Υπενθυμίσεις ενεργοποιήθηκαν', 'success')
      try {
        const { requestNotificationToken } = await import('@/firebase')
        const token = await requestNotificationToken()
        if (token) await store.registerNotificationToken(token)
      } catch { /* silent */ }
    } else {
      toast(locale.value === 'en' ? 'Permission denied' : 'Απορρίφθηκε η άδεια ειδοποιήσεων', 'warning')
      notifDismissed.value = true
      localStorage.setItem('notif_dismissed', '1')
    }
  } catch {
    toast(locale.value === 'en' ? 'Notifications not supported' : 'Οι ειδοποιήσεις δεν υποστηρίζονται', 'error')
  }
}

function dismissNotifBanner() {
  notifDismissed.value = true
  localStorage.setItem('notif_dismissed', '1')
}

async function parseEmail() {
  if (!emailContent.value.trim()) return
  parsingEmail.value = true
  emailError.value = ''
  try {
    const result = await store.parseEmail(emailContent.value)
    if (result.amount != null) form.value.amount = String(result.amount)
    if (result.due_date) form.value.due_date = result.due_date
    if (result.issued_date) form.value.issued_date = result.issued_date
    if (result.notes) form.value.notes = result.notes
    if (result.payment_code) form.value.payment_code = result.payment_code
    if (result.provider_name) {
      const name = result.provider_name.toLowerCase()
      const match = unref(store.userProviders).find(up =>
        (up.nickname || up.provider?.name || '').toLowerCase().includes(name) ||
        name.includes((up.nickname || up.provider?.name || '').toLowerCase())
      )
      if (match) form.value.user_provider_id = match.id
    }
    showEmailInput.value = false
    emailContent.value = ''
  } catch (e: any) {
    emailError.value = e.response?.data?.error || (locale.value === 'en' ? 'Failed to parse email.' : 'Αποτυχία ανάλυσης email.')
  } finally {
    parsingEmail.value = false
  }
}

function openCreate() {
  editingBill.value = null
  form.value = { user_provider_id: '', member_id: '', amount: '', due_date: '', issued_date: '', notes: '', payment_code: '', recurring: false }
  scanError.value = ''
  showModal.value = true
}

function openEdit(id: string) {
  const bill = unref(store.bills).find(b => b.id === id)
  if (!bill) return
  editingBill.value = id
  form.value = {
    user_provider_id: bill.user_provider_id,
    member_id: bill.member_id || '',
    amount: String(bill.amount),
    due_date: bill.due_date.split('T')[0] ?? '',
    issued_date: bill.issued_date ? bill.issued_date.split('T')[0] ?? '' : '',
    notes: bill.notes,
    payment_code: bill.payment_code || '',
    recurring: bill.recurring ?? false,
  }
  scanError.value = ''
  showModal.value = true
}

async function submitForm() {
  saving.value = true
  const payload = {
    user_provider_id: form.value.user_provider_id,
    member_id: form.value.member_id || undefined,
    amount: parseFloat(form.value.amount),
    due_date: new Date(form.value.due_date).toISOString(),
    issued_date: form.value.issued_date ? new Date(form.value.issued_date).toISOString() : undefined,
    notes: form.value.notes,
    payment_code: form.value.payment_code,
  }
  try {
    if (editingBill.value) {
      await store.updateBill(editingBill.value, payload)
      toast(locale.value === 'en' ? 'Saved' : 'Αποθηκεύτηκε', 'success')
    } else {
      await store.createBill({ ...payload, recurring: form.value.recurring })
      toast(locale.value === 'en' ? 'Bill added' : 'Λογαριασμός προστέθηκε', 'success')
    }
    showModal.value = false
    await store.fetchBills()
  } finally {
    saving.value = false
  }
}

async function markPaid(id: string) {
  await store.markPaid(id)
  toast(locale.value === 'en' ? 'Paid ✓' : 'Πληρώθηκε ✓', 'success')
}

async function markUnpaid(id: string) {
  await store.markUnpaid(id)
  toast(locale.value === 'en' ? 'Undone' : 'Αναιρέθηκε', 'info')
}

async function deleteBill(id: string) {
  const msg = locale.value === 'en' ? 'Bill will be permanently deleted.' : 'Ο λογαριασμός θα διαγραφεί οριστικά.'
  if (!await confirm({ message: msg })) return
  await store.deleteBill(id)
  selected.value.delete(id)
  toast(locale.value === 'en' ? 'Deleted' : 'Διαγράφηκε', 'warning')
}

function triggerScan() {
  fileInput.value?.click()
}

async function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  input.value = ''

  scanning.value = true
  scanError.value = ''
  try {
    const result = await store.scanBill(file)
    if (result.amount != null) form.value.amount = String(result.amount)
    if (result.due_date) form.value.due_date = result.due_date
    if (result.issued_date) form.value.issued_date = result.issued_date
    if (result.notes) form.value.notes = result.notes
    if (result.payment_code) form.value.payment_code = result.payment_code
    if (result.provider_name) {
      const name = result.provider_name.toLowerCase()
      const match = unref(store.userProviders).find(up =>
        (up.nickname || up.provider?.name || '').toLowerCase().includes(name) ||
        name.includes((up.nickname || up.provider?.name || '').toLowerCase())
      )
      if (match) form.value.user_provider_id = match.id
    }
  } catch (e: any) {
    scanError.value = e.response?.data?.error || (locale.value === 'en' ? 'Failed to read bill.' : 'Αποτυχία ανάγνωσης λογαριασμού.')
  } finally {
    scanning.value = false
  }
}
</script>

<template>
  <div>
    <!-- Notification permission banner -->
    <div
      v-if="notifSupported && !notifDismissed && !notifGranted"
      class="flex items-center justify-between gap-3 bg-blue-900/30 border border-blue-700/50 rounded-xl px-4 py-3 mb-4 text-sm"
    >
      <span class="text-blue-300">{{ t('bills.notif_banner') }}</span>
      <div class="flex gap-2 shrink-0">
        <button @click="enableNotifications" class="text-white bg-blue-600 hover:bg-blue-700 px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors">
          {{ t('bills.enable_notif') }}
        </button>
        <button @click="dismissNotifBanner" class="text-gray-400 hover:text-gray-300 px-2 py-1 text-sm">✕</button>
      </div>
    </div>

    <!-- ── Header ─────────────────────────────────────────────── -->
    <div class="flex items-center justify-between mb-5 gap-3">
      <div class="flex items-center gap-3 min-w-0">
        <h2 class="text-xl font-bold text-gray-50 shrink-0">{{ t('bills.title') }}</h2>
        <div class="flex rounded-xl border border-gray-700/80 overflow-hidden shrink-0">
          <button
            @click="viewMode = 'list'; exitSelectionMode()"
            class="text-xs font-semibold px-3.5 py-2 transition-colors"
            :class="viewMode === 'list' ? 'bg-gray-700 text-white' : 'text-gray-500 hover:text-gray-300'"
          >{{ t('bills.list_tab') }}</button>
          <button
            @click="viewMode = 'history'; exitSelectionMode()"
            class="text-xs font-semibold px-3.5 py-2 transition-colors"
            :class="viewMode === 'history' ? 'bg-gray-700 text-white' : 'text-gray-500 hover:text-gray-300'"
          >{{ t('bills.history_tab') }}</button>
        </div>
      </div>
      <button
        @click="openCreate"
        class="flex items-center gap-2 bg-blue-600 hover:bg-blue-500 text-white text-sm font-bold px-4 py-2.5 rounded-xl transition-colors shrink-0"
      >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-4 h-4">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        {{ locale === 'el' ? 'Νέος' : 'New' }}
      </button>
    </div>

    <!-- ── History view ──────────────────────────────────────── -->
    <div v-if="viewMode === 'history'" class="space-y-3">
      <div v-if="store.loading" class="space-y-2">
        <div v-for="i in 3" :key="i" class="bg-gray-900 rounded-2xl border border-gray-800 h-24 animate-pulse" />
      </div>
      <div v-else-if="billsByMonth.length === 0" class="text-center py-16 bg-gray-900 rounded-2xl border border-gray-800">
        <p class="text-gray-500 text-sm">{{ t('bills.no_history') }}</p>
      </div>
      <div v-else v-for="([month, bills]) in billsByMonth" :key="month" class="bg-gray-900 rounded-2xl border border-gray-800 overflow-hidden">
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-800 bg-gray-800/40">
          <p class="text-xs font-bold text-gray-400 uppercase tracking-wider">{{ monthLabel(month) }}</p>
          <p class="text-sm font-bold text-gray-200">{{ formatAmount(bills.reduce((s, b) => s + b.amount, 0)) }}</p>
        </div>
        <div v-for="b in bills" :key="b.id" class="flex items-center gap-3 px-4 py-3.5 border-b border-gray-800/50 last:border-0">
          <div class="w-8 h-8 rounded-lg flex items-center justify-center text-white text-xs font-bold shrink-0"
            :style="{ backgroundColor: b.user_provider?.provider?.color || '#6B7280' }">
            {{ (b.user_provider?.nickname || b.user_provider?.provider?.name || '?').charAt(0) }}
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm text-gray-200 truncate font-medium">{{ b.user_provider?.nickname || b.user_provider?.provider?.name }}</p>
            <div class="flex items-center gap-2 mt-0.5">
              <span class="text-xs" :class="statusClass(b.status)">{{ statusLabel(b.status) }}</span>
              <span v-if="b.member_id && membersStore.members.find(m => m.id === b.member_id)"
                class="text-[11px] font-medium px-1.5 py-0.5 rounded-full text-white"
                :style="{ backgroundColor: membersStore.members.find(m => m.id === b.member_id)!.color + 'cc' }">
                {{ membersStore.members.find(m => m.id === b.member_id)!.name }}
              </span>
            </div>
          </div>
          <span class="text-sm font-bold text-gray-100 shrink-0">{{ formatAmount(b.amount) }}</span>
        </div>
      </div>
    </div>

    <!-- ── List view filters ─────────────────────────────────── -->
    <template v-if="viewMode === 'list'">

      <!-- Status chips + filter toggle + select button -->
      <div class="flex items-center gap-2 mb-3">
        <div class="flex gap-1.5 overflow-x-auto flex-1 pb-0.5 scrollbar-none">
          <button
            v-for="f in (['all', 'pending', 'overdue', 'paid'] as const)"
            :key="f"
            @click="filterStatus = f; clearSelection()"
            class="whitespace-nowrap px-3.5 py-2 rounded-xl text-sm font-semibold transition-all shrink-0"
            :class="filterStatus === f ? 'bg-blue-600 text-white' : 'bg-gray-800/80 text-gray-400 border border-gray-700/60 hover:border-gray-600'"
          >
            {{ f === 'all' ? t('bills.all_filter') : statusLabel(f) }}
          </button>
        </div>

        <!-- Filter panel toggle -->
        <button
          @click="filterOpen = !filterOpen"
          class="shrink-0 flex items-center justify-center w-9 h-9 rounded-xl border transition-all relative"
          :class="filterOpen || dateFrom || dateTo || filterMember !== 'all'
            ? 'border-blue-600/60 bg-blue-600/15 text-blue-400'
            : 'border-gray-700/60 bg-gray-800/80 text-gray-500 hover:text-gray-300 hover:border-gray-600'"
          :title="locale === 'el' ? 'Φίλτρα' : 'Filters'"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6h9.75M10.5 6a1.5 1.5 0 1 1-3 0m3 0a1.5 1.5 0 1 0-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m-9.75 0h9.75" />
          </svg>
          <!-- Active dot -->
          <span v-if="dateFrom || dateTo || filterMember !== 'all'" class="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-blue-500 border border-gray-950" />
        </button>

        <!-- Selection mode toggle -->
        <button
          @click="selectionMode ? exitSelectionMode() : (selectionMode = true)"
          class="shrink-0 px-3 py-2 rounded-xl border text-sm font-medium transition-all"
          :class="selectionMode
            ? 'border-blue-600/60 bg-blue-600/15 text-blue-400'
            : 'border-gray-700/60 bg-gray-800/80 text-gray-500 hover:text-gray-300 hover:border-gray-600'"
        >
          {{ locale === 'el' ? 'Επιλογή' : 'Select' }}
        </button>
      </div>

      <!-- Expandable filter panel -->
      <div v-if="filterOpen" class="mb-3 space-y-3 p-3.5 bg-gray-900 rounded-2xl border border-gray-800">
        <!-- Member chips -->
        <div v-if="membersStore.members.length > 0" class="flex gap-2 overflow-x-auto pb-0.5 scrollbar-none">
          <button
            @click="filterMember = 'all'"
            class="whitespace-nowrap flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium transition-all shrink-0"
            :class="filterMember === 'all' ? 'bg-gray-600 text-white' : 'bg-gray-800 text-gray-400 border border-gray-700 hover:border-gray-600'"
          >{{ t('bills.all_filter') }}</button>
          <button
            v-for="m in membersStore.members"
            :key="m.id"
            @click="filterMember = m.id"
            class="whitespace-nowrap flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium transition-all shrink-0 border"
            :class="filterMember === m.id ? 'text-white border-transparent' : 'bg-gray-800 text-gray-400 border-gray-700 hover:border-gray-600'"
            :style="filterMember === m.id ? { backgroundColor: m.color } : {}"
          >
            <span class="w-2 h-2 rounded-full shrink-0" :style="{ backgroundColor: filterMember === m.id ? 'rgba(255,255,255,0.7)' : m.color }" />
            {{ m.name }}
          </button>
        </div>

        <!-- Date range -->
        <div class="flex items-center gap-2">
          <input v-model="dateFrom" type="date" @change="clearSelection()"
            class="flex-1 px-3 py-2 rounded-xl border border-gray-700 bg-gray-800 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <span class="text-gray-600 text-sm shrink-0">—</span>
          <input v-model="dateTo" type="date" @change="clearSelection()"
            class="flex-1 px-3 py-2 rounded-xl border border-gray-700 bg-gray-800 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <button v-if="dateFrom || dateTo || filterMember !== 'all'" @click="clearDateFilter"
          class="w-full py-2 rounded-xl border border-gray-700 text-sm text-gray-400 hover:text-gray-200 hover:border-gray-600 transition-colors">
          {{ t('bills.clear_filter') }}
        </button>
      </div>

      <!-- ── Loading / empty ──────────────────────────────── -->
      <div v-if="store.loading" class="space-y-3">
        <div v-for="i in 4" :key="i" class="bg-gray-900 rounded-2xl border border-gray-800 h-28 animate-pulse" />
      </div>

      <div v-else-if="filteredBills.length === 0" class="text-center py-16 bg-gray-900 rounded-2xl border border-gray-800">
        <p class="text-gray-500 text-sm font-medium">{{ t('bills.no_bills') }}</p>
        <button @click="openCreate" class="mt-3 text-sm text-blue-400 hover:text-blue-300 transition-colors">
          {{ t('bills.add_first') }}
        </button>
      </div>

      <!-- ── Bill cards ────────────────────────────────────── -->
      <div v-else class="space-y-3">
        <div
          v-for="bill in filteredBills"
          :key="bill.id"
          class="bg-gray-900 rounded-2xl border overflow-hidden transition-all"
          :class="selected.has(bill.id) ? 'border-blue-600/40' : 'border-gray-800'"
        >
          <!-- Card top: icon + info + amount -->
          <div class="flex items-center gap-3 px-4 pt-4 pb-3 relative">
            <!-- Color stripe -->
            <div class="absolute left-0 top-0 bottom-0 w-1 rounded-l-2xl opacity-80"
              :style="{ backgroundColor: bill.user_provider?.provider?.color || '#6B7280' }" />

            <!-- Selection checkbox (only in selection mode) -->
            <input
              v-if="selectionMode"
              type="checkbox"
              :checked="selected.has(bill.id)"
              @change="toggleSelect(bill.id)"
              class="w-5 h-5 rounded border-gray-600 accent-blue-600 cursor-pointer shrink-0 ml-2"
            />

            <!-- Provider icon -->
            <div
              class="w-11 h-11 rounded-xl flex items-center justify-center text-white text-base font-bold shrink-0"
              :class="!selectionMode && 'ml-2'"
              :style="{ backgroundColor: bill.user_provider?.provider?.color || '#6B7280' }"
            >
              {{ (bill.user_provider?.nickname || bill.user_provider?.provider?.name || '?').charAt(0).toUpperCase() }}
            </div>

            <!-- Name + meta -->
            <div class="flex-1 min-w-0">
              <p class="text-sm font-bold text-gray-50 truncate">
                {{ bill.user_provider?.nickname || bill.user_provider?.provider?.name }}
              </p>
              <div class="flex items-center gap-2 mt-1 flex-wrap">
                <span class="px-2 py-0.5 rounded-full text-[11px] font-bold shrink-0" :class="statusClass(bill.status)">
                  {{ statusLabel(bill.status) }}
                </span>
                <span class="text-xs text-gray-400 shrink-0">{{ formatDate(bill.due_date) }}</span>
                <span
                  v-if="bill.member_id && membersStore.members.find(m => m.id === bill.member_id)"
                  class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-semibold text-white shrink-0"
                  :style="{ backgroundColor: membersStore.members.find(m => m.id === bill.member_id)!.color }"
                >
                  {{ membersStore.members.find(m => m.id === bill.member_id)!.name }}
                </span>
                <span v-if="bill.recurring" class="text-[11px] text-blue-400 font-semibold shrink-0">↻</span>
              </div>
            </div>

            <!-- Amount -->
            <div class="text-right shrink-0">
              <p class="text-lg font-bold text-gray-50 tabular-nums leading-tight">{{ formatAmount(bill.amount) }}</p>
              <p v-if="priceAlerts.has(bill.id)"
                class="text-[11px] text-amber-400 font-bold mt-0.5"
                :title="`${locale === 'el' ? 'Προηγ.' : 'Prev'}: ${formatAmount(priceAlerts.get(bill.id)!)}`"
              >↑ {{ (((bill.amount - priceAlerts.get(bill.id)!) / priceAlerts.get(bill.id)!) * 100).toFixed(0) }}%</p>
            </div>
          </div>

          <!-- RF / Payment code — prominent copy area -->
          <div v-if="bill.payment_code" class="px-4 pb-3">
            <button
              @click.stop="copyPaymentCode(bill.payment_code, bill.id)"
              class="w-full flex items-center gap-3 px-4 py-3 rounded-xl border transition-all"
              :class="copiedId === bill.id
                ? 'border-green-700/50 bg-green-900/15 text-green-300'
                : 'border-gray-700/60 bg-gray-800/50 text-blue-300 hover:border-blue-600/50 hover:bg-blue-900/10'"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4 shrink-0">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184" />
              </svg>
              <span class="font-mono text-sm flex-1 text-left truncate">{{ bill.payment_code }}</span>
              <span class="text-xs font-bold px-2.5 py-1 rounded-lg shrink-0 transition-colors"
                :class="copiedId === bill.id ? 'bg-green-900/40 text-green-300' : 'bg-gray-700 text-gray-400'">
                {{ copiedId === bill.id ? t('bills.copied') : t('bills.copy') }}
              </span>
            </button>
          </div>

          <!-- Actions: primary + icon buttons -->
          <div class="flex gap-2 px-4 pb-4">
            <!-- Mark paid / undo — takes most of the width -->
            <button
              v-if="bill.status !== 'paid'"
              @click="markPaid(bill.id)"
              class="flex-1 flex items-center justify-center gap-2 py-3 rounded-xl font-bold text-sm transition-all"
              :class="bill.status === 'overdue'
                ? 'bg-red-500/15 border border-red-500/40 text-red-300 hover:bg-red-500/25'
                : 'bg-green-500/15 border border-green-500/40 text-green-300 hover:bg-green-500/25'"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-4 h-4">
                <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
              </svg>
              {{ t('bills.mark_paid') }}
            </button>
            <button
              v-else
              @click="markUnpaid(bill.id)"
              class="flex-1 flex items-center justify-center gap-2 py-3 rounded-xl border border-gray-700 bg-gray-800/30 text-gray-400 hover:text-amber-300 hover:border-amber-700/50 font-semibold text-sm transition-all"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 15 3 9m0 0 6-6M3 9h12a6 6 0 0 1 0 12h-3" />
              </svg>
              {{ t('bills.undo') }}
            </button>

            <!-- IRIS (only if payment code) -->
            <button
              v-if="bill.payment_code"
              @click="payWithIRIS(bill)"
              class="w-12 h-12 flex items-center justify-center rounded-xl border border-indigo-800/40 bg-indigo-900/20 text-indigo-400 hover:bg-indigo-900/40 transition-all shrink-0"
              :title="t('bills.pay_iris')"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 1.5H8.25A2.25 2.25 0 0 0 6 3.75v16.5a2.25 2.25 0 0 0 2.25 2.25h7.5A2.25 2.25 0 0 0 18 20.25V3.75a2.25 2.25 0 0 0-2.25-2.25H13.5m-3 0V3h3V1.5m-3 0h3m-3 8.25h3" />
              </svg>
            </button>

            <!-- Edit -->
            <button
              @click="openEdit(bill.id)"
              class="w-12 h-12 flex items-center justify-center rounded-xl border border-gray-700 bg-gray-800/30 text-gray-400 hover:text-gray-100 hover:border-gray-500 transition-all shrink-0"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Zm0 0L19.5 7.125" />
              </svg>
            </button>

            <!-- Delete -->
            <button
              @click="deleteBill(bill.id)"
              class="w-12 h-12 flex items-center justify-center rounded-xl border border-gray-700 bg-gray-800/30 text-gray-400 hover:text-red-400 hover:border-red-800/50 hover:bg-red-900/10 transition-all shrink-0"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
              </svg>
            </button>
          </div>
        </div>
      </div>

    </template>

    <!-- QR / IRIS modal -->
    <div v-if="qrBill" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 px-4" @click.self="qrBill = null">
      <div class="bg-gray-900 border border-gray-800 rounded-2xl shadow-2xl p-6 w-full max-w-xs text-center">
        <div class="flex items-center justify-center gap-2 mb-1">
          <span class="text-lg">📱</span>
          <h3 class="text-base font-semibold text-gray-50">{{ t('bills.pay_iris') }}</h3>
        </div>
        <p class="text-sm font-medium text-gray-300 mb-1">
          {{ qrBill.user_provider?.nickname || qrBill.user_provider?.provider?.name }}
        </p>
        <p class="text-xs text-gray-400 mb-4">{{ t('bills.iris_scan_desc') }}</p>
        <div class="flex justify-center mb-4">
          <img v-if="qrDataUrl" :src="qrDataUrl" alt="QR Code" class="rounded-xl w-52 h-52" />
          <div v-else class="w-52 h-52 flex items-center justify-center text-gray-500 text-sm">{{ t('bills.no_code') }}</div>
        </div>
        <p class="text-xs font-mono text-blue-400 bg-gray-900/60 rounded-lg px-3 py-2 mb-4 break-all select-all">
          {{ qrBill.payment_code }}
        </p>
        <div class="flex gap-2">
          <button
            @click="copyPaymentCode(qrBill!.payment_code, qrBill!.id)"
            class="flex-1 py-2 rounded-xl border border-gray-700 text-xs font-medium text-gray-300 hover:bg-gray-800 transition-colors"
          >
            {{ copiedId === qrBill.id ? t('bills.copied') : t('bills.copy') }}
          </button>
          <button
            @click="qrBill = null"
            class="flex-1 py-2 rounded-xl bg-gray-800 hover:bg-gray-700 text-xs font-medium text-gray-300 transition-colors"
          >
            {{ t('common.close') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Floating bulk action bar -->
    <Teleport to="body">
      <Transition name="slide-up">
        <div
          v-if="selectionMode && selected.size > 0"
          class="fixed bottom-20 inset-x-4 z-40 flex items-center gap-3 bg-gray-900 border border-gray-700 rounded-2xl px-4 py-3 shadow-2xl shadow-black/50"
        >
          <span class="text-sm text-gray-300 flex-1">
            <span class="font-bold text-white">{{ selected.size }}</span>
            {{ locale === 'el' ? ' επιλεγμένα' : ' selected' }}
          </span>
          <button
            @click="exitSelectionMode()"
            class="px-3 py-2 rounded-xl border border-gray-700 text-sm text-gray-400 hover:text-gray-200 transition-colors"
          >{{ locale === 'el' ? 'Ακύρωση' : 'Cancel' }}</button>
          <button
            @click="bulkDelete"
            class="px-4 py-2 rounded-xl bg-red-700 hover:bg-red-600 text-sm font-bold text-white transition-colors"
          >{{ locale === 'el' ? 'Διαγραφή' : 'Delete' }}</button>
        </div>
      </Transition>
    </Teleport>

    <!-- Hidden file input for scanning -->
    <input
      ref="fileInput"
      type="file"
      accept="image/*,.pdf"
      class="hidden"
      @change="onFileSelected"
    />

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-3 pb-3 sm:px-4 sm:pb-0">
      <div class="bg-gray-900 border border-gray-800 rounded-2xl shadow-2xl w-full max-w-md p-5 sm:p-6 max-h-[92dvh] overflow-y-auto">
        <div class="flex items-center justify-between mb-5">
          <h3 class="text-base font-semibold text-gray-50">
            {{ editingBill ? t('bills.modal_edit') : t('bills.modal_new') }}
          </h3>
          <!-- Scan + Email buttons (only for new bills) -->
          <div v-if="!editingBill" class="flex gap-1.5">
            <button
              type="button"
              @click="triggerScan"
              :disabled="scanning || parsingEmail"
              class="flex items-center gap-1 text-xs font-medium px-2.5 py-1.5 rounded-lg border border-gray-600 hover:bg-gray-700/60 text-gray-400 transition-colors disabled:opacity-60"
              :title="t('bills.scan')"
            >
              <span v-if="scanning" class="animate-spin">⏳</span>
              <span v-else>📷</span>
              <span class="hidden sm:inline">{{ scanning ? t('bills.scanning') : t('bills.scan') }}</span>
            </button>
            <button
              type="button"
              @click="showEmailInput = !showEmailInput; emailError = ''"
              :disabled="scanning || parsingEmail"
              class="flex items-center gap-1 text-xs font-medium px-2.5 py-1.5 rounded-lg border border-gray-600 hover:bg-gray-700/60 text-gray-400 transition-colors disabled:opacity-60"
              :class="showEmailInput ? 'border-blue-600 text-blue-400' : ''"
              :title="t('bills.email_btn')"
            >
              <span v-if="parsingEmail" class="animate-spin">⏳</span>
              <span v-else>📧</span>
              <span class="hidden sm:inline">{{ parsingEmail ? t('bills.parsing') : t('bills.email_btn') }}</span>
            </button>
          </div>
        </div>

        <p v-if="scanError" class="text-red-400 text-xs mb-3 bg-red-900/20 px-3 py-2 rounded-lg">{{ scanError }}</p>

        <!-- Email paste area -->
        <div v-if="showEmailInput && !editingBill" class="mb-4 bg-gray-900/60 rounded-xl p-3 border border-blue-700/40">
          <label class="block text-xs font-medium text-blue-300 mb-1.5">{{ t('bills.email_paste_label') }}</label>
          <textarea
            v-model="emailContent"
            rows="5"
            :placeholder="t('bills.email_paste_placeholder')"
            class="w-full px-3 py-2 rounded-lg border border-gray-600 bg-gray-800 text-xs text-gray-300 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
          />
          <p v-if="emailError" class="text-red-400 text-xs mt-1">{{ emailError }}</p>
          <button
            type="button"
            @click="parseEmail"
            :disabled="parsingEmail || !emailContent.trim()"
            class="mt-2 w-full py-2 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-xs font-medium transition-colors"
          >
            {{ parsingEmail ? t('bills.parsing') : t('bills.email_extract') }}
          </button>
        </div>

        <form @submit.prevent="submitForm" class="space-y-4">
          <div v-if="!editingBill">
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('bills.provider') }}</label>
            <select
              v-model="form.user_provider_id"
              required
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
            >
              <option value="" disabled>{{ t('bills.select_provider') }}</option>
              <option v-for="up in store.userProviders" :key="up.id" :value="up.id">
                {{ up.nickname || up.provider?.name }}
              </option>
            </select>
          </div>

          <!-- Member picker (only show when members exist) -->
          <div v-if="membersStore.members.length > 0">
            <label class="block text-sm font-medium text-gray-300 mb-1.5">{{ t('bills.member') }}</label>
            <div class="flex flex-wrap gap-2">
              <button
                type="button"
                @click="form.member_id = ''"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors"
                :class="!form.member_id
                  ? 'bg-gray-700 border-gray-500 text-gray-200'
                  : 'border-gray-700 text-gray-500 hover:border-gray-600 hover:text-gray-400'"
              >
                {{ t('bills.no_member') }}
              </button>
              <button
                v-for="m in membersStore.members"
                :key="m.id"
                type="button"
                @click="form.member_id = m.id"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors"
                :class="form.member_id === m.id
                  ? 'border-transparent text-white'
                  : 'border-gray-700 text-gray-400 hover:border-gray-600 hover:text-gray-300'"
                :style="form.member_id === m.id ? { backgroundColor: m.color, borderColor: m.color } : {}"
              >
                <span class="w-4 h-4 rounded-full shrink-0" :style="{ backgroundColor: m.color }" />
                {{ m.name }}
              </button>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('bills.amount') }}</label>
            <input
              v-model="form.amount"
              type="number"
              step="0.01"
              min="0"
              required
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              placeholder="0.00"
            />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('bills.due_date') }}</label>
              <input
                v-model="form.due_date"
                type="date"
                required
                class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('bills.issued_date') }}</label>
              <input
                v-model="form.issued_date"
                type="date"
                class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('common.notes') }}</label>
            <input
              v-model="form.notes"
              type="text"
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              :placeholder="t('common.optional')"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('bills.payment_code') }}</label>
            <input
              v-model="form.payment_code"
              type="text"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="π.χ. RF47 1234 5678 9012"
            />
            <p class="text-xs text-gray-500 mt-1">{{ t('bills.payment_code_hint') }}</p>
          </div>

          <!-- Recurring toggle (create only) -->
          <div v-if="!editingBill">
            <button
              type="button"
              @click="form.recurring = !form.recurring"
              class="w-full flex items-center justify-between gap-4 px-4 py-3.5 rounded-xl border transition-colors"
              :class="form.recurring
                ? 'bg-blue-950/40 border-blue-700/60'
                : 'bg-gray-800/40 border-gray-700/50 hover:border-gray-600'"
            >
              <div class="text-left">
                <p class="text-sm font-medium" :class="form.recurring ? 'text-blue-200' : 'text-gray-200'">
                  {{ t('bills.recurring') }}
                </p>
                <p class="text-xs text-gray-500 mt-0.5">{{ t('bills.recurring_desc') }}</p>
              </div>
              <div
                class="relative w-11 h-6 rounded-full transition-colors shrink-0"
                :class="form.recurring ? 'bg-blue-600' : 'bg-gray-600'"
              >
                <span
                  class="absolute top-1 left-1 w-4 h-4 rounded-full bg-white shadow-sm transition-transform duration-200"
                  :class="form.recurring ? 'translate-x-5' : 'translate-x-0'"
                />
              </div>
            </button>
          </div>

          <div class="flex gap-3 pt-1">
            <button
              type="button"
              @click="showModal = false"
              class="flex-1 px-4 py-2.5 rounded-xl border border-gray-700 text-sm font-medium text-gray-300 hover:bg-gray-800 transition-colors"
            >
              {{ t('common.cancel') }}
            </button>
            <button
              type="submit"
              :disabled="saving"
              class="flex-1 px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-55 text-white text-sm font-semibold transition-colors"
            >
              {{ saving ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(100%);
  opacity: 0;
}
</style>
