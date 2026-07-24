<script setup lang="ts">
import { ref, onMounted, computed, unref } from 'vue'
import { useBillsStore } from '@/stores/bills'
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

const showModal = ref(false)
const editingBill = ref<string | null>(null)
const filterStatus = ref<BillStatus | 'all'>('all')
const selected = ref<Set<string>>(new Set())
const saving = ref(false)
const viewMode = ref<'list' | 'history'>('list')

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

const filteredBills = computed(() => {
  let all = unref(store.bills)
  if (!Array.isArray(all)) return []
  if (filterStatus.value !== 'all') all = all.filter(b => b.status === filterStatus.value)
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
  clearSelection()
}

const allSelected = computed(() => {
  const list = filteredBills.value
  return list.length > 0 && list.every(b => selected.value.has(b.id))
})

onMounted(async () => {
  await Promise.all([store.fetchBills(), store.fetchUserProviders()])
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
  form.value = { user_provider_id: '', amount: '', due_date: '', issued_date: '', notes: '', payment_code: '', recurring: false }
  scanError.value = ''
  showModal.value = true
}

function openEdit(id: string) {
  const bill = unref(store.bills).find(b => b.id === id)
  if (!bill) return
  editingBill.value = id
  form.value = {
    user_provider_id: bill.user_provider_id,
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
        <button @click="enableNotifications" class="text-white bg-blue-600 hover:bg-blue-700 px-3 py-1 rounded-lg text-xs font-medium transition-colors">
          {{ t('bills.enable_notif') }}
        </button>
        <button @click="dismissNotifBanner" class="text-gray-400 hover:text-gray-300 px-2 py-1 text-xs">✕</button>
      </div>
    </div>

    <div class="flex items-center justify-between mb-5">
      <div class="flex items-center gap-3">
        <h2 class="text-xl md:text-2xl font-bold text-gray-50">{{ t('bills.title') }}</h2>
        <div class="flex rounded-lg border border-gray-800 overflow-hidden">
          <button
            @click="viewMode = 'list'"
            class="text-xs font-medium px-3 py-1.5 transition-colors"
            :class="viewMode === 'list' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-gray-300'"
          >{{ t('bills.list_tab') }}</button>
          <button
            @click="viewMode = 'history'"
            class="text-xs font-medium px-3 py-1.5 transition-colors"
            :class="viewMode === 'history' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-gray-300'"
          >{{ t('bills.history_tab') }}</button>
        </div>
      </div>
      <button
        @click="openCreate"
        class="bg-blue-600 hover:bg-blue-500 text-white text-sm font-semibold px-3 md:px-4 py-2 rounded-xl transition-colors"
      >
        {{ t('bills.new_bill') }}
      </button>
    </div>

    <!-- History view -->
    <div v-if="viewMode === 'history'" class="space-y-4">
      <div v-if="store.loading" class="space-y-2">
        <div v-for="i in 3" :key="i" class="bg-gray-900 rounded-2xl border border-gray-800 h-24 animate-pulse" />
      </div>
      <div v-else-if="billsByMonth.length === 0" class="text-center py-16 bg-gray-900 rounded-2xl border border-gray-800">
        <p class="text-gray-500 text-sm">{{ t('bills.no_history') }}</p>
      </div>
      <div v-else v-for="([month, bills]) in billsByMonth" :key="month" class="bg-gray-900 rounded-2xl border border-gray-800 overflow-hidden">
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-800 bg-gray-800/40">
          <p class="text-xs font-semibold text-gray-400 uppercase tracking-wider">
            {{ monthLabel(month) }}
          </p>
          <p class="text-xs font-bold text-gray-300">
            {{ formatAmount(bills.reduce((s, b) => s + b.amount, 0)) }}
          </p>
        </div>
        <div v-for="b in bills" :key="b.id" class="flex items-center gap-3 px-4 py-3 border-b border-gray-800/60 last:border-0">
          <div class="w-7 h-7 rounded-lg flex items-center justify-center text-white text-xs font-bold shrink-0"
            :style="{ backgroundColor: b.user_provider?.provider?.color || '#6B7280' }">
            {{ (b.user_provider?.nickname || b.user_provider?.provider?.name || '?').charAt(0) }}
          </div>
          <span class="text-sm text-gray-200 flex-1 truncate">
            {{ b.user_provider?.nickname || b.user_provider?.provider?.name }}
          </span>
          <span class="text-xs" :class="b.recurring ? 'text-blue-400' : 'text-transparent'">↻</span>
          <span class="text-xs px-2 py-0.5 rounded-full font-medium" :class="statusClass(b.status)">{{ statusLabel(b.status) }}</span>
          <span class="text-sm font-bold text-gray-200 shrink-0">{{ formatAmount(b.amount) }}</span>
        </div>
      </div>
    </div>

    <!-- Filter tabs (only in list mode) -->
    <div v-if="viewMode === 'list'" class="flex gap-2 mb-3 overflow-x-auto">
      <button
        v-for="f in (['all', 'pending', 'overdue', 'paid'] as const)"
        :key="f"
        @click="filterStatus = f; clearSelection()"
        class="whitespace-nowrap px-3 py-1.5 rounded-lg text-sm font-medium transition-colors shrink-0"
        :class="filterStatus === f ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-400 border border-gray-700 hover:bg-gray-700/60'"
      >
        {{ f === 'all' ? t('bills.all_filter') : statusLabel(f) }}
      </button>
    </div>

    <!-- Date range filter -->
    <div v-if="viewMode === 'list'" class="flex flex-wrap items-center gap-2 mb-5">
      <input
        v-model="dateFrom"
        type="date"
        @change="clearSelection()"
        class="px-3 py-1.5 rounded-lg border border-gray-700 bg-gray-800 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
      <span class="text-gray-500 text-sm">—</span>
      <input
        v-model="dateTo"
        type="date"
        @change="clearSelection()"
        class="px-3 py-1.5 rounded-lg border border-gray-700 bg-gray-800 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
      <button
        v-if="dateFrom || dateTo"
        @click="clearDateFilter"
        class="text-xs text-gray-400 hover:text-gray-300 px-2 py-1.5 rounded-lg border border-gray-700 hover:bg-gray-700/60 transition-colors whitespace-nowrap"
      >
        {{ t('bills.clear_filter') }}
      </button>
    </div>

    <!-- Bulk action bar -->
    <div
      v-if="viewMode === 'list' && selected.size > 0"
      class="flex items-center justify-between bg-gray-900 border border-gray-800 rounded-2xl px-4 py-3 mb-4"
    >
      <span class="text-sm font-medium text-gray-300">
        <span class="text-white font-semibold">{{ selected.size }}</span> {{ t('bills.selected') }}
      </span>
      <div class="flex gap-2">
        <button
          @click="clearSelection"
          class="text-sm text-gray-400 hover:text-gray-300 px-3 py-1.5 rounded-lg border border-gray-700 hover:bg-gray-800 transition-colors"
        >
          {{ t('common.cancel') }}
        </button>
        <button
          @click="bulkDelete"
          class="text-sm font-medium text-white bg-red-700 hover:bg-red-600 px-3 py-1.5 rounded-lg transition-colors"
        >
          {{ t('common.delete') }} {{ selected.size }}
        </button>
      </div>
    </div>

    <!-- Bills list (list mode only) -->
    <div v-if="viewMode === 'list' && store.loading" class="space-y-2">
      <div v-for="i in 4" :key="i" class="bg-gray-900 rounded-2xl border border-gray-800 h-16 animate-pulse" />
    </div>

    <div v-else-if="viewMode === 'list' && filteredBills.length === 0" class="text-center py-16 bg-gray-900 rounded-2xl border border-gray-800">
      <p class="text-gray-500 text-sm font-medium">{{ t('bills.no_bills') }}</p>
      <button @click="openCreate" class="mt-3 text-xs text-blue-400 hover:text-blue-300 transition-colors">
        {{ t('bills.add_first') }}
      </button>
    </div>

    <div v-else-if="viewMode === 'list'" class="bg-gray-900 rounded-2xl border border-gray-800 divide-y divide-gray-800 overflow-hidden">
      <!-- Select all header -->
      <div class="flex items-center gap-3 px-4 py-2.5 bg-gray-800/40">
        <input
          type="checkbox"
          :checked="allSelected"
          @change="toggleSelectAll"
          class="w-4 h-4 rounded border-gray-700 accent-blue-600 cursor-pointer"
        />
        <span class="text-xs text-gray-500">{{ t('bills.select_all') }} ({{ filteredBills.length }})</span>
      </div>

      <div
        v-for="bill in filteredBills"
        :key="bill.id"
        class="flex items-center gap-3 md:gap-4 p-4 hover:bg-gray-800/40 transition-colors relative"
        :class="selected.has(bill.id) ? 'bg-blue-900/10' : ''"
      >
        <!-- Provider color stripe -->
        <div class="absolute left-0 top-3 bottom-3 w-[3px] rounded-full opacity-60"
          :style="{ backgroundColor: bill.user_provider?.provider?.color || '#6B7280' }" />
        <input
          type="checkbox"
          :checked="selected.has(bill.id)"
          @change="toggleSelect(bill.id)"
          class="w-4 h-4 rounded border-gray-700 accent-blue-600 cursor-pointer shrink-0 ml-2"
        />

        <div
          class="w-10 h-10 rounded-xl flex items-center justify-center text-white text-sm font-bold shrink-0"
          :style="{ backgroundColor: bill.user_provider?.provider?.color }"
        >
          {{ (bill.user_provider?.nickname || bill.user_provider?.provider?.name || '?').charAt(0).toUpperCase() }}
        </div>

        <div class="flex-1 min-w-0">
          <p class="text-sm font-semibold text-gray-50">
            {{ bill.user_provider?.nickname || bill.user_provider?.provider?.name }}
          </p>
          <p class="text-xs text-gray-400 hidden sm:block">{{ t('bills.due_date') }}: {{ formatDate(bill.due_date) }}</p>
          <p class="text-xs text-gray-400 sm:hidden">{{ formatDate(bill.due_date) }}</p>
          <button
            v-if="bill.payment_code"
            @click.stop="copyPaymentCode(bill.payment_code, bill.id)"
            class="flex items-center gap-1 mt-0.5 max-w-full group"
            :title="(locale === 'en' ? 'Copy: ' : 'Αντιγραφή: ') + bill.payment_code"
          >
            <span class="text-[11px] font-mono text-blue-400/80 truncate group-hover:text-blue-300 transition-colors">{{ bill.payment_code }}</span>
            <span class="text-[10px] shrink-0 transition-colors" :class="copiedId === bill.id ? 'text-green-400' : 'text-gray-500 group-hover:text-gray-300'">
              {{ copiedId === bill.id ? '✓' : '⎘' }}
            </span>
          </button>
        </div>

        <!-- Status: dot on mobile, badge on sm+ -->
        <span
          class="sm:hidden w-2 h-2 rounded-full shrink-0"
          :class="{
            'bg-blue-400': bill.status === 'pending',
            'bg-red-400': bill.status === 'overdue',
            'bg-green-400': bill.status === 'paid',
          }"
        />
        <span class="hidden sm:inline px-2 py-1 rounded-full text-xs font-medium shrink-0" :class="statusClass(bill.status)">
          {{ statusLabel(bill.status) }}
        </span>

        <p class="text-sm sm:text-base font-bold text-gray-50 w-16 sm:w-24 text-right shrink-0">{{ formatAmount(bill.amount) }}</p>

        <div class="flex items-center gap-1.5 shrink-0">
          <button
            v-if="bill.status !== 'paid'"
            @click="markPaid(bill.id)"
            class="text-xs bg-gray-700 hover:bg-green-900/30 text-gray-300 hover:text-green-400 font-medium py-1.5 rounded-lg border border-gray-600 hover:border-green-700 transition-colors px-1.5 sm:px-2"
            :title="t('bills.mark_paid')"
          >
            <span class="sm:hidden">✓</span>
            <span class="hidden sm:inline">{{ t('bills.mark_paid') }}</span>
          </button>
          <button
            v-else
            @click="markUnpaid(bill.id)"
            class="text-xs bg-gray-700 hover:bg-amber-900/30 text-gray-400 hover:text-amber-400 font-medium py-1.5 rounded-lg border border-gray-600 hover:border-amber-700 transition-colors px-1.5 sm:px-2"
            :title="t('bills.undo')"
          >
            <span class="sm:hidden">↩</span>
            <span class="hidden sm:inline">{{ t('bills.undo') }}</span>
          </button>

          <!-- IRIS payment button — always show if payment_code exists -->
          <button
            v-if="bill.payment_code"
            @click="payWithIRIS(bill)"
            class="text-xs bg-indigo-900/30 hover:bg-indigo-900/50 text-indigo-300 hover:text-indigo-200 font-medium px-1.5 py-1.5 rounded-lg border border-indigo-800/60 hover:border-indigo-600 transition-colors"
            :title="t('bills.pay_iris')"
          >
            <span class="hidden sm:inline">📱 {{ t('bills.pay_iris') }}</span>
            <span class="sm:hidden">📱</span>
          </button>

          <a
            v-if="bill.user_provider?.provider?.payment_url"
            :href="bill.user_provider?.provider?.payment_url"
            target="_blank"
            rel="noopener"
            class="hidden sm:block text-xs bg-gray-700 hover:bg-gray-600 text-gray-300 font-medium px-2.5 py-1.5 rounded-lg transition-colors"
          >
            {{ t('bills.payment_link') }}
          </a>
          <button
            @click="openEdit(bill.id)"
            class="text-xs text-gray-400 hover:text-gray-300 px-1.5 py-1.5 rounded-lg transition-colors"
          >
            ✎
          </button>
          <button
            @click="deleteBill(bill.id)"
            class="text-xs text-gray-400 hover:text-red-400 px-1.5 py-1.5 rounded-lg transition-colors"
          >
            ✕
          </button>
        </div>
      </div>
    </div>

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

    <!-- Hidden file input for scanning -->
    <input
      ref="fileInput"
      type="file"
      accept="image/*,.pdf"
      class="hidden"
      @change="onFileSelected"
    />

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0">
      <div class="bg-gray-900 border border-gray-800 rounded-2xl shadow-2xl w-full max-w-md p-6">
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
          <div v-if="!editingBill" class="flex items-center justify-between py-1 px-3.5 bg-gray-800/40 rounded-xl border border-gray-700/50">
            <div>
              <p class="text-sm font-medium text-gray-200">{{ t('bills.recurring') }}</p>
              <p class="text-xs text-gray-500">{{ t('bills.recurring_desc') }}</p>
            </div>
            <button
              type="button"
              @click="form.recurring = !form.recurring"
              class="relative w-10 h-6 rounded-full transition-colors shrink-0 ml-4"
              :class="form.recurring ? 'bg-blue-600' : 'bg-gray-700'"
            >
              <span
                class="absolute top-1 w-4 h-4 rounded-full bg-white shadow transition-transform"
                :class="form.recurring ? 'translate-x-5' : 'translate-x-1'"
              />
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
