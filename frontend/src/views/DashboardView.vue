<script setup lang="ts">
import { ref, onMounted, onActivated, computed, unref } from 'vue'
import { useBillsStore } from '@/stores/bills'
import { useBudgetStore } from '@/stores/budget'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { formatDate, formatAmount, statusLabel, statusClass, categoryLabel, currentMonth, monthLabel } from '@/utils/format'

const { confirm } = useConfirm()
const { toast } = useToast()

const billsStore = useBillsStore()
const budgetStore = useBudgetStore()

const month = ref(currentMonth())
const showIncomeModal = ref(false)
const incomeForm = ref({ description: '', amount: '' })
const selectedIncomes = ref<Set<string>>(new Set())
const savingIncome = ref(false)

const d = computed(() => billsStore.dashboard)
const pendingBills = computed(() => d.value?.upcoming_bills?.filter(b => b.status === 'pending') ?? [])
const overdueBills = computed(() => d.value?.upcoming_bills?.filter(b => b.status === 'overdue') ?? [])
const totalIncome = computed(() => {
  const incomes = unref(budgetStore.incomes)
  return Array.isArray(incomes) ? incomes.reduce((s, i) => s + i.amount, 0) : 0
})
const spentPercent = computed(() => {
  if (!d.value || d.value.total_income <= 0) return 0
  return Math.min(((d.value.amount_bills + d.value.amount_expenses_planned) / d.value.total_income) * 100, 100)
})

onMounted(async () => {
  await Promise.all([
    billsStore.fetchDashboard(month.value),
    budgetStore.fetchIncomes(month.value),
    budgetStore.fetchExpenses(month.value),
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
    })
    incomeForm.value = { description: '', amount: '' }
    showIncomeModal.value = false
    await billsStore.fetchDashboard(month.value)
    toast('Έσοδο προστέθηκε', 'success')
  } finally {
    savingIncome.value = false
  }
}

async function removeIncome(id: string) {
  if (!await confirm({ message: 'Το έσοδο θα διαγραφεί οριστικά.' })) return
  await budgetStore.deleteIncome(id)
  await billsStore.fetchDashboard(month.value)
  toast('Διαγράφηκε', 'warning')
}

function toggleIncomeSelect(id: string) {
  const s = new Set(selectedIncomes.value)
  s.has(id) ? s.delete(id) : s.add(id)
  selectedIncomes.value = s
}

async function bulkDeleteIncomes() {
  const ids = [...selectedIncomes.value]
  if (!await confirm({ message: `Θα διαγραφούν ${ids.length} έσοδα. Η ενέργεια δεν αναιρείται.` })) return
  await budgetStore.bulkDeleteIncome(ids)
  selectedIncomes.value = new Set()
  await billsStore.fetchDashboard(month.value)
  toast(`Διαγράφηκαν ${ids.length} έσοδα`, 'warning')
}

async function onMonthChange() {
  await Promise.all([
    billsStore.fetchDashboard(month.value),
    budgetStore.fetchIncomes(month.value),
    budgetStore.fetchExpenses(month.value),
  ])
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6 gap-3">
      <div>
        <h2 class="text-xl md:text-2xl font-bold text-gray-50">Αρχική</h2>
        <p class="text-sm text-gray-400 capitalize mt-0.5">{{ monthLabel(month) }}</p>
      </div>
      <input
        v-model="month"
        type="month"
        class="px-3 py-2 rounded-lg border border-gray-700 bg-gray-800 text-sm text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
        @change="onMonthChange"
      />
    </div>

    <div v-if="d" class="space-y-4">

      <!-- Hero budget card -->
      <div class="bg-gray-800 rounded-2xl border border-gray-700 p-5">
        <!-- Remaining balance (hero) -->
        <div class="flex items-start justify-between mb-5">
          <div>
            <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-1">Διαθέσιμο υπόλοιπο</p>
            <p
              class="text-3xl md:text-4xl font-bold tracking-tight"
              :class="d.remaining >= 0 ? 'text-white' : 'text-red-400'"
            >
              {{ formatAmount(d.remaining) }}
            </p>
          </div>
          <div
            class="px-3 py-1 rounded-full text-xs font-semibold"
            :class="d.remaining >= 0 ? 'bg-green-900/30 text-green-400 border border-green-700/40' : 'bg-red-900/30 text-red-400 border border-red-700/40'"
          >
            {{ d.remaining >= 0 ? '✓ OK' : '⚠ Υπέρβαση' }}
          </div>
        </div>

        <!-- Breakdown row -->
        <div class="grid grid-cols-3 gap-3 mb-4">
          <div class="bg-gray-900/50 rounded-xl p-3">
            <p class="text-xs text-gray-400 mb-1">Έσοδα</p>
            <p class="text-base font-bold text-green-400">{{ formatAmount(d.total_income) }}</p>
          </div>
          <div class="bg-gray-900/50 rounded-xl p-3">
            <p class="text-xs text-gray-400 mb-1">Λογαριασμοί</p>
            <p class="text-base font-bold text-gray-200">{{ formatAmount(d.amount_bills) }}</p>
            <p v-if="d.amount_overdue > 0" class="text-xs text-red-400 mt-0.5">{{ formatAmount(d.amount_overdue) }} ληξ.</p>
          </div>
          <div class="bg-gray-900/50 rounded-xl p-3">
            <p class="text-xs text-gray-400 mb-1">Αγορές</p>
            <p class="text-base font-bold text-amber-400">{{ formatAmount(d.amount_expenses_planned) }}</p>
          </div>
        </div>

        <!-- Progress bar -->
        <div v-if="d.total_income > 0">
          <div class="flex justify-between text-xs text-gray-500 mb-1">
            <span>Δαπάνες</span>
            <span>{{ spentPercent.toFixed(0) }}% του εισοδήματος</span>
          </div>
          <div class="h-2.5 bg-gray-700 rounded-full overflow-hidden flex">
            <div
              class="h-full transition-all"
              :class="d.amount_overdue > 0 ? 'bg-red-500' : 'bg-blue-500'"
              :style="{ width: Math.min((d.amount_bills / d.total_income) * 100, 100) + '%' }"
            />
            <div
              class="h-full bg-amber-400 transition-all"
              :style="{ width: Math.min((d.amount_expenses_planned / d.total_income) * 100, 100) + '%' }"
            />
          </div>
          <div class="flex gap-4 mt-1.5 text-xs text-gray-500">
            <span class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-blue-500 inline-block"></span>Λογαριασμοί</span>
            <span class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-amber-400 inline-block"></span>Αγορές</span>
          </div>
        </div>

        <!-- Counters -->
        <div class="flex gap-4 mt-4 pt-4 border-t border-gray-700">
          <div class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full bg-blue-400 shrink-0"></span>
            <span class="text-xs text-gray-400">{{ d.total_pending }} εκκρεμείς</span>
          </div>
          <div v-if="d.total_overdue > 0" class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full bg-red-400 shrink-0"></span>
            <span class="text-xs text-red-400 font-medium">{{ d.total_overdue }} ληξιπρόθεσμοι</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="w-2 h-2 rounded-full bg-green-400 shrink-0"></span>
            <span class="text-xs text-gray-400">{{ d.total_paid }} πληρωμένοι</span>
          </div>
        </div>
      </div>

      <!-- Two columns: Upcoming + Overdue -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">

        <!-- Upcoming bills -->
        <div class="bg-gray-800 rounded-2xl border border-gray-700 p-4">
          <div class="flex items-center justify-between mb-3">
            <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Επερχόμενοι</p>
            <span class="text-xs font-bold text-blue-400">{{ formatAmount(d.amount_bills - d.amount_overdue) }}</span>
          </div>
          <p v-if="pendingBills.length === 0" class="text-sm text-gray-500 py-2 text-center">Κανένας εκκρεμής 🎉</p>
          <ul v-else class="space-y-2.5">
            <li v-for="b in pendingBills" :key="b.id" class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="w-8 h-8 rounded-lg flex items-center justify-center text-white text-xs font-bold shrink-0"
                  :style="{ backgroundColor: b.user_provider?.provider?.color || '#6B7280' }">
                  {{ (b.user_provider?.nickname || b.user_provider?.provider?.name || '?').charAt(0).toUpperCase() }}
                </div>
                <div class="min-w-0">
                  <p class="text-sm font-medium text-gray-100 truncate">
                    {{ b.user_provider?.nickname || b.user_provider?.provider?.name }}
                  </p>
                  <p class="text-xs text-gray-500">{{ formatDate(b.due_date) }}</p>
                </div>
              </div>
              <p class="text-sm font-semibold text-gray-100 shrink-0">{{ formatAmount(b.amount) }}</p>
            </li>
          </ul>
        </div>

        <!-- Overdue bills -->
        <div class="bg-gray-800 rounded-2xl border p-4" :class="overdueBills.length > 0 ? 'border-red-700/40' : 'border-gray-700'">
          <div class="flex items-center justify-between mb-3">
            <p class="text-xs font-semibold uppercase tracking-wide" :class="overdueBills.length > 0 ? 'text-red-400' : 'text-gray-400'">
              Ληξιπρόθεσμοι
            </p>
            <span class="text-xs font-bold" :class="overdueBills.length > 0 ? 'text-red-400' : 'text-gray-500'">
              {{ formatAmount(d.amount_overdue) }}
            </span>
          </div>
          <p v-if="overdueBills.length === 0" class="text-sm text-gray-500 py-2 text-center">Κανένας ληξιπρόθεσμος ✓</p>
          <ul v-else class="space-y-2.5">
            <li v-for="b in overdueBills" :key="b.id" class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2.5 min-w-0">
                <div class="w-8 h-8 rounded-lg flex items-center justify-center text-white text-xs font-bold shrink-0"
                  :style="{ backgroundColor: b.user_provider?.provider?.color || '#6B7280' }">
                  {{ (b.user_provider?.nickname || b.user_provider?.provider?.name || '?').charAt(0).toUpperCase() }}
                </div>
                <div class="min-w-0">
                  <p class="text-sm font-medium text-gray-100 truncate">
                    {{ b.user_provider?.nickname || b.user_provider?.provider?.name }}
                  </p>
                  <p class="text-xs text-red-400">{{ formatDate(b.due_date) }}</p>
                </div>
              </div>
              <p class="text-sm font-semibold text-red-400 shrink-0">{{ formatAmount(b.amount) }}</p>
            </li>
          </ul>
        </div>

      </div>

      <!-- Income section -->
      <div class="bg-gray-800 rounded-2xl border border-gray-700 p-4 md:p-5">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-xs font-semibold text-gray-400 uppercase tracking-wide">Έσοδα μήνα</h3>
          <div class="flex items-center gap-3">
            <button
              v-if="selectedIncomes.size > 0"
              @click="bulkDeleteIncomes"
              class="text-xs font-medium text-red-400 hover:text-red-300 px-2.5 py-1 rounded-lg border border-red-700/40 hover:bg-red-900/20 transition-colors"
            >
              Διαγραφή ({{ selectedIncomes.size }})
            </button>
            <button @click="showIncomeModal = true" class="text-xs text-blue-400 hover:text-blue-300 font-medium transition-colors">
              + Προσθήκη
            </button>
          </div>
        </div>

        <div v-if="(budgetStore.incomes as any[]).length === 0" class="text-center py-4">
          <p class="text-sm text-gray-500">Δεν έχεις προσθέσει έσοδα</p>
          <button @click="showIncomeModal = true" class="mt-2 text-xs text-blue-400 hover:text-blue-300">
            Πρόσθεσε το πρώτο σου έσοδο →
          </button>
        </div>
        <div v-else class="space-y-1">
          <div
            v-for="inc in (budgetStore.incomes as any[])"
            :key="inc.id"
            class="flex items-center gap-3 py-2 px-2 rounded-xl transition-colors cursor-pointer"
            :class="selectedIncomes.has(inc.id) ? 'bg-blue-900/20' : 'hover:bg-gray-700/50'"
            @click="toggleIncomeSelect(inc.id)"
          >
            <input type="checkbox" :checked="selectedIncomes.has(inc.id)"
              @click.stop @change="toggleIncomeSelect(inc.id)"
              class="w-4 h-4 rounded border-gray-600 accent-blue-600 cursor-pointer shrink-0" />
            <span class="text-sm text-gray-300 flex-1 truncate">{{ inc.description }}</span>
            <div class="flex items-center gap-3 shrink-0">
              <span class="text-sm font-semibold text-green-400">+{{ formatAmount(inc.amount) }}</span>
              <button @click.stop="removeIncome(inc.id)" class="text-gray-600 hover:text-red-400 text-sm transition-colors w-5 h-5 flex items-center justify-center">✕</button>
            </div>
          </div>
          <div class="flex justify-between pt-2 border-t border-gray-700 mt-1">
            <span class="text-sm font-semibold text-gray-400">Σύνολο</span>
            <span class="text-sm font-bold text-green-400">{{ formatAmount(totalIncome) }}</span>
          </div>
        </div>
      </div>

      <!-- Planned expenses mini -->
      <div v-if="(d.planned_expenses ?? []).length > 0" class="bg-gray-800 rounded-2xl border border-amber-700/30 p-4">
        <div class="flex items-center justify-between mb-3">
          <p class="text-xs font-semibold text-amber-400 uppercase tracking-wide">Αγορές μήνα</p>
          <span class="text-xs font-bold text-amber-400">{{ formatAmount(d.amount_expenses_planned) }}</span>
        </div>
        <ul class="space-y-2">
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

    <!-- Loading state -->
    <div v-else class="space-y-4">
      <div class="bg-gray-800 rounded-2xl border border-gray-700 h-48 animate-pulse" />
      <div class="grid grid-cols-2 gap-4">
        <div class="bg-gray-800 rounded-2xl border border-gray-700 h-32 animate-pulse" />
        <div class="bg-gray-800 rounded-2xl border border-gray-700 h-32 animate-pulse" />
      </div>
    </div>

    <!-- Income modal -->
    <div v-if="showIncomeModal" class="fixed inset-0 bg-black/50 flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0" @click.self="showIncomeModal = false">
      <div class="bg-gray-800 rounded-2xl shadow-xl w-full max-w-sm p-6">
        <h3 class="text-lg font-semibold text-gray-50 mb-5">Προσθήκη εσόδου</h3>
        <form @submit.prevent="addIncome" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Περιγραφή</label>
            <input v-model="incomeForm.description" type="text" required autofocus
              class="w-full px-3 py-2.5 rounded-xl border border-gray-600 bg-gray-900/50 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="π.χ. Μισθός, Freelance..." />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Ποσό (€)</label>
            <input v-model="incomeForm.amount" type="number" step="0.01" min="0.01" required
              class="w-full px-3 py-2.5 rounded-xl border border-gray-600 bg-gray-900/50 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="0.00" />
          </div>
          <div class="flex gap-3 pt-2">
            <button type="button" @click="showIncomeModal = false"
              class="flex-1 px-4 py-2.5 rounded-xl border border-gray-600 text-sm font-medium text-gray-300 hover:bg-gray-700/60 transition-colors">
              Ακύρωση
            </button>
            <button type="submit" :disabled="savingIncome"
              class="flex-1 px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors disabled:opacity-60">
              {{ savingIncome ? 'Αποθήκευση…' : 'Αποθήκευση' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
