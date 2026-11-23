<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useBudgetStore } from '@/stores/budget'
import { useMembersStore } from '@/stores/members'
import { useConfirm } from '@/composables/useConfirm'
import { useLocale } from '@/composables/useLocale'
import { formatAmount, formatDate, expenseCategoryLabel, expenseCategoryIcon, expenseStatusLabel, expenseStatusClass, currentMonth, monthLabel } from '@/utils/format'
import type { Expense, ExpenseCategory } from '@/types'

const { confirm } = useConfirm()
const { t, locale } = useLocale()

const store = useBudgetStore()
const { expenses, loading } = storeToRefs(store)
const membersStore = useMembersStore()
const showModal = ref(false)
const editingExpense = ref<Expense | null>(null)
const filterStatus = ref<'all' | 'planned' | 'bought'>('all')
const month = ref(currentMonth())
const dateFrom = ref('')
const dateTo = ref('')

const form = ref({
  name: '',
  amount: '',
  category: 'other' as ExpenseCategory,
  member_id: '',
  store: '',
  notes: '',
  planned_date: '',
})

const categories = computed<{ value: ExpenseCategory; label: string }[]>(() => [
  { value: 'shopping', label: t('expenses.categories.shopping') },
  { value: 'food', label: t('expenses.categories.food') },
  { value: 'transport', label: t('expenses.categories.transport') },
  { value: 'health', label: t('expenses.categories.health') },
  { value: 'entertainment', label: t('expenses.categories.entertainment') },
  { value: 'other', label: t('expenses.categories.other') },
])

const totalAll = computed(() => (Array.isArray(expenses.value) ? expenses.value : []).reduce((s, e) => s + e.amount, 0))
const totalBought = computed(() => (Array.isArray(expenses.value) ? expenses.value : []).filter(e => e.status === 'bought').reduce((s, e) => s + e.amount, 0))
const totalPlanned = computed(() => (Array.isArray(expenses.value) ? expenses.value : []).filter(e => e.status === 'planned').reduce((s, e) => s + e.amount, 0))
const boughtPercent = computed(() => totalAll.value > 0 ? Math.round((totalBought.value / totalAll.value) * 100) : 0)

const filtered = computed(() => {
  let all = Array.isArray(expenses.value) ? expenses.value : []
  if (filterStatus.value !== 'all') all = all.filter(e => e.status === filterStatus.value)
  if (dateFrom.value || dateTo.value) {
    all = all.filter(e => {
      const d = e.planned_date ? e.planned_date.slice(0, 10) : ''
      if (!d) return true
      if (dateFrom.value && d < dateFrom.value) return false
      if (dateTo.value && d > dateTo.value) return false
      return true
    })
  }
  return all
})

const groupedExpenses = computed(() => {
  const groups = new Map<string, { label: string; icon: string; total: number; items: Expense[] }>()
  for (const e of filtered.value) {
    const key = e.category
    if (!groups.has(key)) {
      groups.set(key, { label: expenseCategoryLabel(key), icon: expenseCategoryIcon(key), total: 0, items: [] })
    }
    const g = groups.get(key)!
    g.total += e.amount
    g.items.push(e)
  }
  return [...groups.entries()]
    .sort((a, b) => b[1].total - a[1].total)
    .map(([key, g]) => ({ key, ...g }))
})

onMounted(() => {
  store.fetchExpenses(month.value)
  membersStore.fetchMembers()
})

function prevMonth() {
  const [y, m] = month.value.split('-').map(Number)
  const d = new Date(y!, m! - 2, 1)
  month.value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
  store.fetchExpenses(month.value)
}

function nextMonth() {
  const [y, m] = month.value.split('-').map(Number)
  const d = new Date(y!, m!, 1)
  month.value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
  store.fetchExpenses(month.value)
}

function openCreate() {
  editingExpense.value = null
  form.value = { name: '', amount: '', category: 'other', member_id: '', store: '', notes: '', planned_date: '' }
  showModal.value = true
}

function openEdit(e: Expense) {
  editingExpense.value = e
  form.value = {
    name: e.name,
    amount: String(e.amount),
    category: e.category,
    member_id: e.member_id ?? '',
    store: e.store ?? '',
    notes: e.notes ?? '',
    planned_date: e.planned_date ? e.planned_date.slice(0, 10) : '',
  }
  showModal.value = true
}

async function submit() {
  const payload = {
    name: form.value.name,
    amount: parseFloat(form.value.amount),
    category: form.value.category,
    month: month.value,
    member_id: form.value.member_id || undefined,
    store: form.value.store,
    notes: form.value.notes,
    planned_date: form.value.planned_date ? new Date(form.value.planned_date).toISOString() : undefined,
  }
  if (editingExpense.value) {
    await store.updateExpense(editingExpense.value.id, payload)
  } else {
    await store.createExpense(payload)
  }
  showModal.value = false
  editingExpense.value = null
  form.value = { name: '', amount: '', category: 'other', member_id: '', store: '', notes: '', planned_date: '' }
}

async function markBought(id: string) {
  await store.markBought(id)
}

async function remove(id: string) {
  const msg = locale.value === 'en' ? 'Expense will be permanently deleted.' : 'Η αγορά θα διαγραφεί οριστικά.'
  if (!await confirm({ message: msg })) return
  await store.deleteExpense(id)
}
</script>

<template>
  <div class="max-w-2xl mx-auto space-y-4">

    <!-- ── Header ─────────────────────────────────────────────── -->
    <div class="flex items-center justify-between">
      <h2 class="text-2xl font-black text-white tracking-tight">{{ t('expenses.title') }}</h2>
      <button
        @click="openCreate"
        class="flex items-center gap-2 bg-gradient-to-r from-violet-600 to-purple-600 hover:from-violet-500 hover:to-purple-500 text-white text-sm font-bold px-4 py-2.5 rounded-xl transition-all shadow-sm shadow-violet-900/40 active:scale-[0.97]"
      >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-4 h-4">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        {{ locale === 'el' ? 'Νέα' : 'New' }}
      </button>
    </div>

    <!-- ── Summary card ─────────────────────────────────────────── -->
    <div class="relative overflow-hidden rounded-2xl p-5 border bg-gradient-to-br from-violet-950/80 via-purple-950/40 to-[#111119] border-violet-800/25">
      <div class="absolute -top-14 -right-14 w-48 h-48 rounded-full blur-3xl pointer-events-none bg-violet-600/15" />
      <div class="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-violet-500/20 to-transparent" />

      <div class="flex items-start justify-between mb-4">
        <div>
          <p class="text-[11px] font-bold text-gray-500 tracking-widest uppercase mb-1.5">
            {{ locale === 'el' ? 'Συνολικές Αγορές' : 'Total Expenses' }}
          </p>
          <p class="text-4xl font-black text-white tracking-tight">{{ formatAmount(totalAll) }}</p>
        </div>
        <div class="flex flex-col items-end gap-2 mt-1">
          <div class="flex items-center gap-2 text-xs">
            <span class="w-2 h-2 rounded-full bg-amber-400 shrink-0" />
            <span class="text-gray-500">{{ locale === 'el' ? 'Προγρ.' : 'Planned' }}</span>
            <span class="font-bold text-amber-400 tabular-nums">{{ formatAmount(totalPlanned) }}</span>
          </div>
          <div class="flex items-center gap-2 text-xs">
            <span class="w-2 h-2 rounded-full bg-emerald-400 shrink-0" />
            <span class="text-gray-500">{{ locale === 'el' ? 'Αγοράστηκε' : 'Bought' }}</span>
            <span class="font-bold text-emerald-400 tabular-nums">{{ formatAmount(totalBought) }}</span>
          </div>
        </div>
      </div>

      <!-- Progress bar: bought vs planned -->
      <div class="h-2 rounded-full bg-white/[0.06] overflow-hidden flex">
        <div class="h-full bg-emerald-500 rounded-l-full transition-all duration-700" :style="{ width: boughtPercent + '%' }" />
        <div class="h-full bg-amber-500/50 rounded-r-full transition-all duration-700" :style="{ width: (100 - boughtPercent) + '%' }" />
      </div>
      <p class="text-[11px] text-gray-600 mt-1.5">
        {{ boughtPercent }}% {{ locale === 'el' ? 'αγοράστηκε' : 'bought' }}
        · {{ filtered.length }} {{ locale === 'el' ? 'αντικείμενα' : 'items' }}
      </p>
    </div>

    <!-- ── Period & date range ─────────────────────────────────── -->
    <div class="space-y-2">
      <!-- Month navigation -->
      <div class="flex items-center bg-[#111119] border border-white/[0.06] rounded-xl px-1 py-1">
        <button @click="prevMonth" class="w-9 h-9 flex items-center justify-center rounded-lg hover:bg-white/[0.06] text-gray-400 hover:text-white transition-colors shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
          </svg>
        </button>
        <span class="flex-1 text-center text-sm font-bold text-white capitalize">{{ monthLabel(month) }}</span>
        <button @click="nextMonth" class="w-9 h-9 flex items-center justify-center rounded-lg hover:bg-white/[0.06] text-gray-400 hover:text-white transition-colors shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
          </svg>
        </button>
      </div>

      <!-- Date range filter -->
      <div class="flex items-center gap-2">
        <input
          v-model="dateFrom"
          type="date"
          class="flex-1 px-3 py-2 rounded-xl border border-white/[0.06] bg-white/[0.03] text-sm text-gray-400 focus:outline-none focus:ring-2 focus:ring-violet-500/60 focus:border-transparent transition-all [color-scheme:dark]"
          :class="dateFrom ? 'border-violet-500/40 text-gray-200' : ''"
        />
        <span class="text-gray-600 text-sm font-bold shrink-0">—</span>
        <input
          v-model="dateTo"
          type="date"
          class="flex-1 px-3 py-2 rounded-xl border border-white/[0.06] bg-white/[0.03] text-sm text-gray-400 focus:outline-none focus:ring-2 focus:ring-violet-500/60 focus:border-transparent transition-all [color-scheme:dark]"
          :class="dateTo ? 'border-violet-500/40 text-gray-200' : ''"
        />
        <button
          v-if="dateFrom || dateTo"
          @click="dateFrom = ''; dateTo = ''"
          class="shrink-0 w-9 h-9 flex items-center justify-center rounded-xl border border-white/[0.06] text-gray-500 hover:text-white hover:border-white/[0.12] transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- ── Status filter chips ─────────────────────────────────── -->
    <div class="flex gap-2">
      <button
        v-for="f in (['all', 'planned', 'bought'] as const)"
        :key="f"
        @click="filterStatus = f"
        class="px-4 py-2 rounded-xl text-sm font-semibold transition-all"
        :class="filterStatus === f
          ? f === 'all'
            ? 'bg-violet-600 text-white shadow-sm shadow-violet-900/40'
            : f === 'planned'
            ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
            : 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
          : 'bg-[#111119] text-gray-500 border border-white/[0.06] hover:border-white/[0.12] hover:text-gray-300'"
      >
        {{ f === 'all' ? t('expenses.all_filter') : expenseStatusLabel(f) }}
      </button>
    </div>

    <!-- ── Loading ─────────────────────────────────────────────── -->
    <div v-if="loading" class="space-y-3">
      <div v-for="i in 3" :key="i" class="bg-[#111119] rounded-2xl border border-white/[0.06] h-20 animate-pulse" />
    </div>

    <!-- ── Empty state ─────────────────────────────────────────── -->
    <div v-else-if="filtered.length === 0" class="text-center py-16 bg-[#111119] rounded-2xl border border-dashed border-white/[0.08]">
      <div class="w-14 h-14 rounded-2xl bg-violet-900/20 flex items-center justify-center mx-auto mb-4">
        <span class="text-2xl">🛍️</span>
      </div>
      <p class="text-sm font-semibold text-gray-300 mb-1">{{ t('expenses.no_expenses') }}</p>
      <p class="text-xs text-gray-600 mb-5">{{ locale === 'el' ? 'Πρόσθεσε αγορές για τον μήνα' : 'Add your planned expenses' }}</p>
      <button @click="openCreate"
        class="inline-flex items-center gap-2 bg-gradient-to-r from-violet-600 to-purple-600 hover:from-violet-500 hover:to-purple-500 text-white text-sm font-bold px-5 py-2.5 rounded-xl transition-all shadow-lg shadow-violet-900/30">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        {{ t('expenses.add_expense') }}
      </button>
    </div>

    <!-- ── Grouped list ────────────────────────────────────────── -->
    <div v-else class="space-y-4">
      <div v-for="group in groupedExpenses" :key="group.key">

        <!-- Category header -->
        <div class="flex items-center justify-between px-1 mb-2">
          <div class="flex items-center gap-2">
            <span class="text-lg leading-none">{{ group.icon }}</span>
            <span class="text-xs font-black text-gray-500 uppercase tracking-widest">{{ group.label }}</span>
          </div>
          <span class="text-xs font-bold text-gray-500 tabular-nums">{{ formatAmount(group.total) }}</span>
        </div>

        <!-- Items -->
        <div class="rounded-2xl border border-white/[0.06] bg-[#111119] divide-y divide-white/[0.04] overflow-hidden">
          <div
            v-for="e in group.items"
            :key="e.id"
            class="flex items-center gap-3 p-4 hover:bg-white/[0.025] transition-colors group/item"
          >
            <!-- Icon -->
            <div
              class="w-10 h-10 rounded-xl flex items-center justify-center text-xl shrink-0 transition-colors"
              :class="e.status === 'bought' ? 'bg-emerald-900/20' : 'bg-violet-900/20'"
            >
              {{ group.icon }}
            </div>

            <!-- Details -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 flex-wrap">
                <p class="text-sm font-semibold text-gray-100 truncate" :class="e.status === 'bought' ? 'line-through opacity-50' : ''">
                  {{ e.name }}
                </p>
                <span
                  v-if="e.member_id && membersStore.members.find(m => m.id === e.member_id)"
                  class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-bold text-white shrink-0"
                  :style="{ backgroundColor: membersStore.members.find(m => m.id === e.member_id)!.color + 'cc' }"
                >
                  {{ membersStore.members.find(m => m.id === e.member_id)!.name }}
                </span>
              </div>
              <div class="flex items-center gap-1.5 mt-0.5 flex-wrap">
                <span v-if="e.store" class="text-xs text-gray-500 font-medium">{{ e.store }}</span>
                <span v-if="e.planned_date" class="text-xs text-gray-600">{{ e.store ? '·' : '' }} {{ formatDate(e.planned_date) }}</span>
                <span v-if="e.notes" class="text-xs text-gray-600 italic truncate max-w-[120px]">· {{ e.notes }}</span>
              </div>
            </div>

            <!-- Status chip (desktop) -->
            <span
              class="hidden sm:inline px-2.5 py-1 rounded-full text-[11px] font-bold shrink-0"
              :class="expenseStatusClass(e.status)"
            >
              {{ expenseStatusLabel(e.status) }}
            </span>

            <!-- Amount -->
            <p class="text-sm font-black shrink-0 tabular-nums" :class="e.status === 'bought' ? 'text-emerald-400 opacity-70' : 'text-white'">
              {{ formatAmount(e.amount) }}
            </p>

            <!-- Actions -->
            <div class="flex items-center gap-1 shrink-0">
              <button
                v-if="e.status === 'planned'"
                @click="markBought(e.id)"
                class="w-8 h-8 flex items-center justify-center rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 hover:bg-emerald-500/20 transition-all"
                :title="locale === 'el' ? 'Αγοράστηκε' : 'Mark bought'"
              >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-3.5 h-3.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
                </svg>
              </button>
              <button
                @click="openEdit(e)"
                class="w-8 h-8 flex items-center justify-center rounded-xl text-gray-600 hover:text-violet-400 hover:bg-violet-500/10 transition-all"
                title="Επεξεργασία"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Zm0 0L19.5 7.125" />
                </svg>
              </button>
              <button
                @click="remove(e.id)"
                class="w-8 h-8 flex items-center justify-center rounded-xl text-gray-600 hover:text-red-400 hover:bg-red-500/10 transition-all"
                :title="locale === 'el' ? 'Διαγραφή' : 'Delete'"
              >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Modal ──────────────────────────────────────────────── -->
    <div v-if="showModal"
      class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0"
      @click.self="showModal = false; editingExpense = null"
    >
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl shadow-2xl w-full max-w-md p-6">
        <h3 class="text-base font-semibold text-gray-50 mb-5">
          {{ editingExpense ? (locale === 'el' ? 'Επεξεργασία αγοράς' : 'Edit expense') : t('expenses.modal_title') }}
        </h3>

        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.what_to_buy') }}</label>
            <input v-model="form.name" type="text" required autofocus
              class="w-full px-3.5 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.02] text-sm text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all"
              placeholder="π.χ. Βιβλιοθήκη BILLY" />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.amount') }}</label>
              <input v-model="form.amount" type="number" step="0.01" min="0" required
                class="w-full px-3.5 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.02] text-sm text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all"
                placeholder="0.00" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.category') }}</label>
              <select v-model="form.category"
                class="w-full px-3.5 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.02] text-sm text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all">
                <option v-for="c in categories" :key="c.value" :value="c.value">{{ c.label }}</option>
              </select>
            </div>
          </div>

          <!-- Member picker -->
          <div v-if="membersStore.members.length > 0">
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ locale === 'el' ? 'Μέλος' : 'Member' }}</label>
            <div class="flex flex-wrap gap-2">
              <button
                type="button"
                @click="form.member_id = ''"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold border transition-colors"
                :class="!form.member_id
                  ? 'bg-white/[0.06] border-gray-500 text-gray-200'
                  : 'border-white/[0.08] text-gray-500 hover:border-white/[0.12] hover:text-gray-400'"
              >
                {{ locale === 'el' ? 'Χωρίς' : 'None' }}
              </button>
              <button
                v-for="m in membersStore.members"
                :key="m.id"
                type="button"
                @click="form.member_id = m.id"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold border transition-colors"
                :class="form.member_id === m.id
                  ? 'border-transparent text-white'
                  : 'border-white/[0.08] text-gray-400 hover:border-white/[0.12] hover:text-gray-300'"
                :style="form.member_id === m.id ? { backgroundColor: m.color, borderColor: m.color } : {}"
              >
                <span class="w-3 h-3 rounded-full shrink-0" :style="{ backgroundColor: m.color }" />
                {{ m.name }}
              </button>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.store_label') }}</label>
              <input v-model="form.store" type="text"
                class="w-full px-3.5 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.02] text-sm text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all"
                placeholder="π.χ. IKEA" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.date') }}</label>
              <input v-model="form.planned_date" type="date"
                class="w-full px-3.5 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.02] text-sm text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all [color-scheme:dark]" />
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('common.notes') }}</label>
            <input v-model="form.notes" type="text"
              class="w-full px-3.5 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.02] text-sm text-white focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent transition-all"
              :placeholder="t('common.optional')" />
          </div>

          <div class="flex gap-3 pt-1">
            <button type="button" @click="showModal = false; editingExpense = null"
              class="flex-1 px-4 py-2.5 rounded-xl border border-white/[0.08] text-sm font-medium text-gray-300 hover:bg-white/[0.04] transition-colors">
              {{ t('common.cancel') }}
            </button>
            <button type="submit"
              class="flex-1 px-4 py-2.5 rounded-xl bg-gradient-to-r from-violet-600 to-purple-600 hover:from-violet-500 hover:to-purple-500 text-white text-sm font-bold transition-all shadow-sm shadow-violet-900/30">
              {{ editingExpense ? (locale === 'el' ? 'Αποθήκευση' : 'Save') : t('expenses.add') }}
            </button>
          </div>
        </form>
      </div>
    </div>

  </div>
</template>
