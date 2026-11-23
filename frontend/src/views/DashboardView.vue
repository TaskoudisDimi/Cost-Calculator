<script setup lang="ts">
import { ref, onMounted, onActivated, computed, unref } from 'vue'
import { useBillsStore } from '@/stores/bills'
import { useBudgetStore } from '@/stores/budget'
import { useMembersStore } from '@/stores/members'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { useLocale } from '@/composables/useLocale'
import { formatDate, formatAmount, currentMonth, monthLabel } from '@/utils/format'

const { confirm } = useConfirm()
const { toast } = useToast()
const { t, locale } = useLocale()

const billsStore = useBillsStore()
const budgetStore = useBudgetStore()
const membersStore = useMembersStore()

const month = ref(currentMonth())
const showIncomeModal = ref(false)
const incomeForm = ref({ description: '', amount: '', member_id: '' })
const savingIncome = ref(false)
const incomeExpanded = ref(false)

const d = computed(() => billsStore.dashboard)
const incomes = computed(() => {
  const list = unref(budgetStore.incomes)
  return Array.isArray(list) ? list : []
})
const pendingBills = computed(() => d.value?.upcoming_bills?.filter(b => b.status === 'pending') ?? [])
const overdueBills = computed(() => d.value?.upcoming_bills?.filter(b => b.status === 'overdue') ?? [])
const totalIncome = computed(() => incomes.value.reduce((s, i) => s + i.amount, 0))
const spentPercent = computed(() => {
  if (!d.value || d.value.total_income <= 0) return 0
  return Math.min(((d.value.amount_bills + d.value.amount_expenses_planned) / d.value.total_income) * 100, 100)
})

// Forecast: sum of all recurring bills as next month estimate
const forecast = computed(() => {
  const allBills = unref(billsStore.bills)
  if (!Array.isArray(allBills)) return null
  const recurring = allBills.filter(b => b.recurring)
  if (recurring.length === 0) return null
  const total = recurring.reduce((s, b) => s + b.amount, 0)
  return { total, count: recurring.length }
})

const memberIncomeBreakdown = computed(() => {
  if (membersStore.members.length === 0) return []
  const memberMap = new Map(membersStore.members.map(m => [m.id, { ...m, total: 0 }]))
  let unassigned = 0
  for (const inc of incomes.value) {
    if (inc.member_id && memberMap.has(inc.member_id)) {
      memberMap.get(inc.member_id)!.total += inc.amount
    } else {
      unassigned += inc.amount
    }
  }
  const result = [...memberMap.values()].filter(m => m.total > 0)
  return result
})

onMounted(async () => {
  const hasBills = billsStore.bills.length > 0
  await Promise.all([
    billsStore.fetchDashboard(month.value),
    hasBills ? Promise.resolve() : billsStore.fetchBills(),
    budgetStore.fetchIncomes(month.value),
    budgetStore.fetchExpenses(month.value),
    membersStore.fetchMembers(),
  ])
})

onActivated(() => {
  billsStore.fetchDashboard(month.value)
})

async function addIncome() {
  savingIncome.value = true
  try {
    await budgetStore.createIncome({
      description: incomeForm.value.description,
      amount: parseFloat(incomeForm.value.amount),
      month: month.value,
      member_id: incomeForm.value.member_id || undefined,
    })
    incomeForm.value = { description: '', amount: '', member_id: '' }
    showIncomeModal.value = false
    await billsStore.fetchDashboard(month.value)
    toast(locale.value === 'en' ? 'Income added' : 'Έσοδο προστέθηκε', 'success')
  } finally {
    savingIncome.value = false
  }
}

async function removeIncome(id: string) {
  if (!await confirm({ message: locale.value === 'en' ? 'Income will be permanently deleted.' : 'Το έσοδο θα διαγραφεί οριστικά.' })) return
  await budgetStore.deleteIncome(id)
  await billsStore.fetchDashboard(month.value)
  toast(locale.value === 'en' ? 'Deleted' : 'Διαγράφηκε', 'warning')
}

async function onMonthChange() {
  await Promise.all([
    billsStore.fetchDashboard(month.value),
    budgetStore.fetchIncomes(month.value),
    budgetStore.fetchExpenses(month.value),
  ])
}

async function payBill(id: string) {
  await billsStore.markPaid(id)
  toast(locale.value === 'en' ? 'Paid ✓' : 'Πληρώθηκε ✓', 'success')
  await billsStore.fetchDashboard(month.value)
}
</script>

<template>
  <div class="max-w-2xl mx-auto">

    <!-- Header -->
    <div class="flex items-center justify-between mb-5 gap-3">
      <div>
        <h2 class="text-2xl font-black text-white tracking-tight">{{ t('dashboard.title') }}</h2>
        <p class="text-sm text-gray-500 mt-0.5 capitalize">{{ monthLabel(month) }}</p>
      </div>
      <input
        v-model="month"
        type="month"
        class="px-3 py-2 rounded-xl border border-white/[0.08] bg-white/[0.03] text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500/60 focus:border-transparent transition-all"
        @change="onMonthChange"
      />
    </div>

    <!-- Loading -->
    <div v-if="!d" class="space-y-4">
      <div class="bg-[#111119] rounded-2xl border border-white/[0.06] h-52 animate-pulse" />
      <div class="bg-[#111119] rounded-2xl border border-white/[0.06] h-28 animate-pulse" />
      <div class="grid grid-cols-2 gap-3">
        <div class="bg-[#111119] rounded-2xl border border-white/[0.06] h-32 animate-pulse" />
        <div class="bg-[#111119] rounded-2xl border border-white/[0.06] h-32 animate-pulse" />
      </div>
    </div>

    <div v-else class="space-y-4">

      <!-- ─── Overdue banner ───────────────────── -->
      <div
        v-if="overdueBills.length > 0"
        class="flex items-center gap-3 px-4 py-3 rounded-2xl bg-red-500/10 border border-red-500/25"
      >
        <div class="w-8 h-8 rounded-xl bg-red-500/20 flex items-center justify-center shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-4 h-4 text-red-400">
            <path fill-rule="evenodd" d="M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 1.998-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003ZM12 8.25a.75.75 0 0 1 .75.75v3.75a.75.75 0 0 1-1.5 0V9a.75.75 0 0 1 .75-.75Zm0 8.25a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Z" clip-rule="evenodd" />
          </svg>
        </div>
        <div class="flex-1 min-w-0">
          <p class="text-sm font-bold text-red-300">
            {{ overdueBills.length === 1
              ? (locale === 'el' ? '1 εκπρόθεσμος λογαριασμός' : '1 overdue bill')
              : (locale === 'el' ? `${overdueBills.length} εκπρόθεσμοι λογαριασμοί` : `${overdueBills.length} overdue bills`) }}
          </p>
          <p class="text-xs text-red-400/70 mt-0.5">{{ formatAmount(d.amount_overdue) }} {{ locale === 'el' ? 'συνολικά' : 'total' }}</p>
        </div>
        <RouterLink :to="{ name: 'bills' }" class="text-xs font-bold text-red-400 hover:text-red-300 shrink-0 transition-colors">
          {{ locale === 'el' ? 'Δες' : 'View' }} →
        </RouterLink>
      </div>

      <!-- ─── Hero card ─────────────────────────── -->
      <div
        class="relative overflow-hidden rounded-2xl p-5 md:p-6 border"
        :class="d.remaining >= 0
          ? 'bg-gradient-to-br from-blue-950/80 via-indigo-950/40 to-[#111119] border-blue-800/25'
          : 'bg-gradient-to-br from-red-950/80 via-rose-950/40 to-[#111119] border-red-800/25'"
      >
        <div class="absolute -top-16 -right-16 w-56 h-56 rounded-full blur-3xl pointer-events-none"
          :class="d.remaining >= 0 ? 'bg-blue-600/15' : 'bg-red-600/15'" />
        <div class="absolute bottom-0 left-0 right-0 h-px"
          :class="d.remaining >= 0 ? 'bg-gradient-to-r from-transparent via-blue-500/20 to-transparent' : 'bg-gradient-to-r from-transparent via-red-500/20 to-transparent'" />

        <div class="flex items-center justify-between mb-3">
          <p class="text-[11px] font-bold text-gray-500 tracking-widest uppercase">{{ t('dashboard.balance') }}</p>
          <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold"
            :class="d.remaining >= 0
              ? 'bg-emerald-500/12 text-emerald-400 border border-emerald-500/20'
              : 'bg-red-500/12 text-red-400 border border-red-500/20'"
          >
            <span class="w-1.5 h-1.5 rounded-full shrink-0"
              :class="d.remaining >= 0 ? 'bg-emerald-400' : 'bg-red-400'" />
            {{ d.remaining >= 0 ? 'OK' : t('dashboard.overspent') }}
          </div>
        </div>

        <p class="text-5xl font-black tracking-tight mb-1"
          :class="d.remaining >= 0 ? 'text-white' : 'text-red-300'">
          {{ formatAmount(d.remaining) }}
        </p>

        <!-- Progress bar -->
        <div v-if="d.total_income > 0" class="h-1 rounded-full bg-white/[0.06] overflow-hidden mb-5 mt-3">
          <div class="h-full rounded-full transition-all duration-700"
            :class="spentPercent >= 90 ? 'bg-gradient-to-r from-amber-500 to-red-500' : 'bg-gradient-to-r from-blue-500 to-indigo-400'"
            :style="{ width: spentPercent + '%' }" />
        </div>
        <div v-else class="mb-5" />

        <!-- Stats row -->
        <div class="grid grid-cols-3 gap-3 pt-4 border-t border-white/[0.06]">
          <div>
            <p class="text-[10px] text-gray-600 mb-1 font-bold uppercase tracking-wide">{{ t('dashboard.income_label') }}</p>
            <p class="text-sm font-bold text-emerald-400 tabular-nums">{{ formatAmount(d.total_income) }}</p>
          </div>
          <div>
            <p class="text-[10px] text-gray-600 mb-1 font-bold uppercase tracking-wide">{{ t('dashboard.bills_label') }}</p>
            <p class="text-sm font-bold text-blue-300 tabular-nums">{{ formatAmount(d.amount_bills) }}</p>
            <p v-if="d.amount_overdue > 0" class="text-[10px] text-red-400 mt-0.5 font-semibold">{{ formatAmount(d.amount_overdue) }} {{ t('dashboard.overdue_short') }}</p>
          </div>
          <div>
            <p class="text-[10px] text-gray-600 mb-1 font-bold uppercase tracking-wide">{{ t('dashboard.expenses_label') }}</p>
            <p class="text-sm font-bold text-amber-400 tabular-nums">{{ formatAmount(d.amount_expenses_planned) }}</p>
          </div>
        </div>
      </div>

      <!-- ─── Bills: Upcoming + Overdue ─────────── -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">

        <!-- Upcoming -->
        <div class="bg-gradient-to-b from-white/[0.025] to-[#111119] rounded-2xl border border-white/[0.06] p-4">
          <div class="flex items-center justify-between mb-3">
            <p class="text-[11px] font-bold text-gray-500 uppercase tracking-wider flex items-center gap-1.5">
              <span class="w-1.5 h-1.5 rounded-full bg-blue-400 inline-block" />
              {{ t('dashboard.upcoming') }}
            </p>
            <span class="text-xs font-bold text-blue-400 tabular-nums">
              {{ formatAmount(d.amount_bills - d.amount_overdue) }}
            </span>
          </div>
          <p v-if="pendingBills.length === 0" class="text-sm text-gray-600 py-3 text-center">{{ t('dashboard.no_pending') }}</p>
          <ul v-else class="space-y-3">
            <li v-for="b in pendingBills" :key="b.id" class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="w-8 h-8 rounded-xl flex items-center justify-center text-white text-xs font-bold shrink-0"
                  :style="{ backgroundColor: b.user_provider?.provider?.color || '#6B7280' }">
                  {{ (b.user_provider?.nickname || b.user_provider?.provider?.name || '?').charAt(0).toUpperCase() }}
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-1.5 flex-wrap">
                    <p class="text-sm font-medium text-gray-100 truncate">
                      {{ b.user_provider?.nickname || b.user_provider?.provider?.name }}
                    </p>
                    <span
                      v-if="b.member_id && membersStore.members.find(m => m.id === b.member_id)"
                      class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold text-white shrink-0"
                      :style="{ backgroundColor: membersStore.members.find(m => m.id === b.member_id)!.color + 'cc' }"
                    >
                      {{ membersStore.members.find(m => m.id === b.member_id)!.name }}
                    </span>
                  </div>
                  <p class="text-xs text-gray-500">{{ formatDate(b.due_date) }}</p>
                </div>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <p class="text-sm font-semibold text-gray-200">{{ formatAmount(b.amount) }}</p>
                <button
                  @click="payBill(b.id)"
                  class="w-7 h-7 flex items-center justify-center rounded-lg bg-emerald-500/10 border border-emerald-500/25 text-emerald-400 hover:bg-emerald-500/20 transition-colors shrink-0"
                  :title="locale === 'el' ? 'Πληρώθηκε' : 'Mark paid'"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-3.5 h-3.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
                  </svg>
                </button>
              </div>
            </li>
          </ul>
        </div>

        <!-- Overdue -->
        <div class="rounded-2xl border p-4"
          :class="overdueBills.length > 0
            ? 'bg-gradient-to-b from-red-950/25 to-[#111119] border-red-900/50'
            : 'bg-gradient-to-b from-white/[0.025] to-[#111119] border-white/[0.06]'">
          <div class="flex items-center justify-between mb-3">
            <p class="text-[11px] font-bold uppercase tracking-wider flex items-center gap-1.5"
              :class="overdueBills.length > 0 ? 'text-red-400' : 'text-gray-500'">
              <span class="w-1.5 h-1.5 rounded-full inline-block" :class="overdueBills.length > 0 ? 'bg-red-400' : 'bg-gray-600'" />
              {{ t('dashboard.overdue') }}
            </p>
            <span class="text-xs font-bold tabular-nums" :class="overdueBills.length > 0 ? 'text-red-400' : 'text-gray-700'">
              {{ formatAmount(d.amount_overdue) }}
            </span>
          </div>
          <p v-if="overdueBills.length === 0" class="text-sm text-gray-600 py-3 text-center">{{ t('dashboard.no_overdue') }}</p>
          <ul v-else class="space-y-3">
            <li v-for="b in overdueBills" :key="b.id" class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="w-8 h-8 rounded-xl flex items-center justify-center text-white text-xs font-bold shrink-0"
                  :style="{ backgroundColor: b.user_provider?.provider?.color || '#6B7280' }">
                  {{ (b.user_provider?.nickname || b.user_provider?.provider?.name || '?').charAt(0).toUpperCase() }}
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-1.5 flex-wrap">
                    <p class="text-sm font-medium text-gray-100 truncate">
                      {{ b.user_provider?.nickname || b.user_provider?.provider?.name }}
                    </p>
                    <span
                      v-if="b.member_id && membersStore.members.find(m => m.id === b.member_id)"
                      class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold text-white shrink-0"
                      :style="{ backgroundColor: membersStore.members.find(m => m.id === b.member_id)!.color + 'cc' }"
                    >
                      {{ membersStore.members.find(m => m.id === b.member_id)!.name }}
                    </span>
                  </div>
                  <p class="text-xs text-red-400">{{ formatDate(b.due_date) }}</p>
                </div>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <p class="text-sm font-semibold text-red-400">{{ formatAmount(b.amount) }}</p>
                <button
                  @click="payBill(b.id)"
                  class="w-7 h-7 flex items-center justify-center rounded-lg bg-red-500/10 border border-red-500/25 text-red-400 hover:bg-red-500/20 transition-colors shrink-0"
                  :title="locale === 'el' ? 'Πληρώθηκε' : 'Mark paid'"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-3.5 h-3.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
                  </svg>
                </button>
              </div>
            </li>
          </ul>
        </div>

      </div>

      <!-- ─── Έσοδα ──────────────────────────────── -->
      <div>
        <button
          @click="incomeExpanded = !incomeExpanded"
          class="flex items-center justify-between w-full mb-3 group"
        >
          <div class="flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 inline-block" />
            <h3 class="text-xs font-bold text-gray-500 uppercase tracking-wider group-hover:text-gray-300 transition-colors">{{ t('dashboard.income_section') }}</h3>
            <span v-if="totalIncome > 0" class="text-xs font-bold text-emerald-400 tabular-nums">{{ formatAmount(totalIncome) }}</span>
          </div>
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
            class="w-4 h-4 text-gray-600 transition-transform duration-200"
            :class="incomeExpanded ? '' : '-rotate-90'"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
          </svg>
        </button>

        <div v-if="incomeExpanded" class="mb-3">
          <div class="flex justify-end mb-3">
            <button
              @click="showIncomeModal = true"
              class="flex items-center gap-1.5 text-sm font-semibold text-white bg-emerald-700 hover:bg-emerald-600 active:bg-emerald-800 px-3.5 py-1.5 rounded-xl transition-colors"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-3.5 h-3.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
              </svg>
              {{ t('dashboard.new_income') }}
            </button>
          </div>

        <!-- Empty state -->
        <div v-if="incomes.length === 0"
          class="bg-[#111119] border border-dashed border-emerald-900/60 rounded-2xl p-8 text-center"
        >
          <div class="w-12 h-12 bg-emerald-900/25 rounded-2xl flex items-center justify-center mx-auto mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 text-emerald-400">
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0 1 15.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 0 1 3 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 0 0-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 0 1-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 0 0 3 15h-.75M15 10.5a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm3 0h.008v.008H18V10.5Zm-12 0h.008v.008H6V10.5Z" />
            </svg>
          </div>
          <p class="text-sm font-medium text-gray-200 mb-1">{{ t('dashboard.no_income_title') }}</p>
          <p class="text-xs text-gray-500 mb-5 max-w-[220px] mx-auto leading-relaxed">
            {{ t('dashboard.no_income_desc') }}
          </p>
          <button
            @click="showIncomeModal = true"
            class="inline-flex items-center gap-2 bg-emerald-700 hover:bg-emerald-600 text-white text-sm font-semibold px-5 py-2.5 rounded-xl transition-colors shadow-lg shadow-emerald-900/30"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            {{ t('dashboard.add_income') }}
          </button>
        </div>

        <!-- Income list -->
        <div v-else class="bg-[#111119] border border-white/[0.06] rounded-2xl overflow-hidden">
          <div
            v-for="inc in incomes"
            :key="inc.id"
            class="flex items-center gap-3 px-4 py-3.5 border-b border-white/[0.06] last:border-b-0 group"
          >
            <div class="w-8 h-8 bg-emerald-900/25 rounded-xl flex items-center justify-center shrink-0">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4 text-emerald-400">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v12m-3-2.818.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
              </svg>
            </div>
            <div class="flex-1 min-w-0">
              <span class="text-sm text-gray-200 truncate block">{{ inc.description }}</span>
              <span v-if="inc.member_id && membersStore.members.find(m => m.id === inc.member_id)" class="inline-flex items-center gap-1 mt-0.5">
                <span class="w-2 h-2 rounded-full shrink-0" :style="{ backgroundColor: membersStore.members.find(m => m.id === inc.member_id)!.color }" />
                <span class="text-[11px] text-gray-500">{{ membersStore.members.find(m => m.id === inc.member_id)!.name }}</span>
              </span>
            </div>
            <span class="text-sm font-bold text-emerald-400 shrink-0">+{{ formatAmount(inc.amount) }}</span>
            <button
              @click="removeIncome(inc.id)"
              class="text-gray-700 hover:text-red-400 transition-colors shrink-0 ml-1 w-7 h-7 flex items-center justify-center rounded-lg hover:bg-red-900/20"
              :title="t('common.delete')"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-3.5 h-3.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <div class="flex justify-between px-4 py-3 bg-white/[0.03]">
            <span class="text-xs text-gray-500">{{ t('dashboard.income_total') }}</span>
            <span class="text-sm font-bold text-emerald-400">{{ formatAmount(totalIncome) }}</span>
          </div>
          <!-- Per-member income breakdown -->
          <div v-if="memberIncomeBreakdown.length > 0" class="border-t border-white/[0.06] px-4 py-3 flex flex-wrap gap-3">
            <div
              v-for="m in memberIncomeBreakdown"
              :key="m.id"
              class="flex items-center gap-2"
            >
              <div class="w-5 h-5 rounded-full flex items-center justify-center text-white text-[10px] font-bold shrink-0" :style="{ backgroundColor: m.color }">
                {{ m.name[0]?.toUpperCase() }}
              </div>
              <span class="text-xs text-gray-400">{{ m.name }}</span>
              <span class="text-xs font-semibold text-emerald-400">{{ formatAmount(m.total) }}</span>
            </div>
          </div>
        </div>
        </div><!-- /v-if incomeExpanded -->
      </div>

      <!-- ─── Progress bar ─── -->
      <div v-if="d.total_income > 0" class="bg-[#111119] border border-white/[0.06] rounded-2xl p-4">
        <div class="flex justify-between text-xs text-gray-500 mb-2">
          <span>{{ t('dashboard.spending_month') }}</span>
          <span class="font-semibold" :class="spentPercent >= 90 ? 'text-red-400' : 'text-gray-400'">
            {{ spentPercent.toFixed(0) }}%
          </span>
        </div>
        <div class="h-1.5 bg-white/[0.04] rounded-full overflow-hidden flex mb-3">
          <div
            class="h-full transition-all duration-500"
            :class="d.amount_overdue > 0 ? 'bg-red-500' : 'bg-blue-500'"
            :style="{ width: Math.min((d.amount_bills / d.total_income) * 100, 100) + '%' }"
          />
          <div
            class="h-full bg-amber-400 transition-all duration-500"
            :style="{ width: Math.min((d.amount_expenses_planned / d.total_income) * 100, 100) + '%' }"
          />
        </div>
        <div class="flex items-center gap-4 text-xs text-gray-500 flex-wrap">
          <span class="flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-blue-500 shrink-0" />{{ d.total_pending }} {{ t('dashboard.pending_label') }}
          </span>
          <span v-if="d.total_overdue > 0" class="flex items-center gap-1.5 text-red-400 font-medium">
            <span class="w-1.5 h-1.5 rounded-full bg-red-400 shrink-0" />{{ d.total_overdue }} {{ t('dashboard.overdue_label') }}
          </span>
          <span class="flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0" />{{ d.total_paid }} {{ t('dashboard.paid_label') }}
          </span>
        </div>
      </div>

      <!-- ─── Forecast widget ───────────────────── -->
      <div v-if="forecast" class="bg-[#111119] border border-indigo-900/40 rounded-2xl p-4 flex items-center justify-between gap-4">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-xl bg-indigo-900/30 flex items-center justify-center shrink-0">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5 text-indigo-400">
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18 9 11.25l4.306 4.306a11.95 11.95 0 0 1 5.814-5.518l2.74-1.22m0 0-5.94-2.281m5.94 2.28-2.28 5.941" />
            </svg>
          </div>
          <div>
            <p class="text-xs font-semibold text-indigo-400 uppercase tracking-wide">{{ t('dashboard.forecast_title') }}</p>
            <p class="text-[11px] text-gray-500 mt-0.5">{{ t('dashboard.forecast_recurring') }}: {{ forecast.count }}</p>
          </div>
        </div>
        <div class="text-right">
          <p class="text-lg font-bold text-indigo-300">~{{ formatAmount(forecast.total) }}</p>
          <p class="text-[11px] text-gray-500">{{ t('dashboard.forecast_desc') }}</p>
        </div>
      </div>

      <!-- ─── Αγορές (mini) ─────────────────────── -->
      <div v-if="(d.planned_expenses ?? []).length > 0" class="bg-[#111119] rounded-2xl border border-amber-900/40 p-4">
        <div class="flex items-center justify-between mb-3">
          <p class="text-xs font-semibold text-amber-400 uppercase tracking-wider">{{ t('dashboard.expenses_month') }}</p>
          <span class="text-xs font-bold text-amber-400">{{ formatAmount(d.amount_expenses_planned) }}</span>
        </div>
        <ul class="space-y-2.5">
          <li v-for="e in (d.planned_expenses ?? [])" :key="e.id" class="flex items-center justify-between gap-2">
            <span class="text-sm text-gray-300 truncate">{{ e.name }}</span>
            <div class="text-right shrink-0">
              <p class="text-sm font-semibold text-amber-400">{{ formatAmount(e.amount) }}</p>
              <p v-if="e.store" class="text-xs text-gray-500">{{ e.store }}</p>
            </div>
          </li>
        </ul>
      </div>

    </div>

    <!-- ─── Income modal ──────────────────────────── -->
    <div v-if="showIncomeModal"
      class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0"
      @click.self="showIncomeModal = false"
    >
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl shadow-2xl w-full max-w-sm p-6">
        <h3 class="text-base font-semibold text-gray-50 mb-5">{{ t('dashboard.income_modal_title') }}</h3>
        <form @submit.prevent="addIncome" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('dashboard.description') }}</label>
            <input v-model="incomeForm.description" type="text" required autofocus
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent transition-all"
              :placeholder="t('dashboard.income_placeholder')" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('dashboard.amount') }}</label>
            <input v-model="incomeForm.amount" type="number" step="0.01" min="0.01" required
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent transition-all"
              placeholder="0.00" />
          </div>
          <!-- Member picker (only show when members exist) -->
          <div v-if="membersStore.members.length > 0">
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('dashboard.member') }}</label>
            <div class="flex flex-wrap gap-2">
              <button
                type="button"
                @click="incomeForm.member_id = ''"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors"
                :class="!incomeForm.member_id
                  ? 'bg-white/[0.06] border-gray-500 text-gray-200'
                  : 'border-white/[0.08] text-gray-500 hover:border-white/[0.10] hover:text-gray-400'"
              >
                {{ t('dashboard.no_member') }}
              </button>
              <button
                v-for="m in membersStore.members"
                :key="m.id"
                type="button"
                @click="incomeForm.member_id = m.id"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors"
                :class="incomeForm.member_id === m.id
                  ? 'border-transparent text-white'
                  : 'border-white/[0.08] text-gray-400 hover:border-white/[0.10] hover:text-gray-300'"
                :style="incomeForm.member_id === m.id ? { backgroundColor: m.color, borderColor: m.color } : {}"
              >
                <span class="w-4 h-4 rounded-full shrink-0" :style="{ backgroundColor: m.color }" />
                {{ m.name }}
              </button>
            </div>
          </div>
          <div class="flex gap-3 pt-1">
            <button type="button" @click="showIncomeModal = false"
              class="flex-1 px-4 py-2.5 rounded-xl border border-white/[0.08] text-sm font-medium text-gray-300 hover:bg-white/[0.04] transition-colors">
              {{ t('common.cancel') }}
            </button>
            <button type="submit" :disabled="savingIncome"
              class="flex-1 px-4 py-2.5 rounded-xl bg-emerald-700 hover:bg-emerald-600 text-white text-sm font-semibold transition-colors disabled:opacity-55">
              {{ savingIncome ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </form>
      </div>
    </div>

  </div>
</template>
