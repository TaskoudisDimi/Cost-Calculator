<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useBillsStore } from '@/stores/bills'
import { useBudgetStore } from '@/stores/budget'
import { useConfirm } from '@/composables/useConfirm'
import { formatDate, formatAmount, statusLabel, statusClass, categoryLabel, currentMonth, monthLabel } from '@/utils/format'

const { confirm } = useConfirm()

const billsStore = useBillsStore()
const budgetStore = useBudgetStore()

const month = ref(currentMonth())
const showIncomeModal = ref(false)
const incomeForm = ref({ description: '', amount: '' })
const selectedIncomes = ref<Set<string>>(new Set())

const d = computed(() => billsStore.dashboard)
const pendingBills = computed(() => d.value?.upcoming_bills.filter(b => b.status === 'pending') ?? [])
const overdueBills = computed(() => d.value?.upcoming_bills.filter(b => b.status === 'overdue') ?? [])

const totalIncome = computed(() =>
  budgetStore.incomes.reduce((s, i) => s + i.amount, 0)
)

onMounted(async () => {
  await Promise.all([
    billsStore.fetchDashboard(month.value),
    budgetStore.fetchIncomes(month.value),
    budgetStore.fetchExpenses(month.value),
  ])
})

async function addIncome() {
  await budgetStore.createIncome({
    description: incomeForm.value.description,
    amount: parseFloat(incomeForm.value.amount),
    month: month.value,
  })
  incomeForm.value = { description: '', amount: '' }
  showIncomeModal.value = false
  await billsStore.fetchDashboard(month.value)
}

async function removeIncome(id: string) {
  if (!await confirm({ message: 'Το έσοδο θα διαγραφεί οριστικά.' })) return
  await budgetStore.deleteIncome(id)
  await billsStore.fetchDashboard(month.value)
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
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold text-gray-900">Dashboard</h2>
        <p class="text-sm text-gray-400 capitalize mt-0.5">{{ monthLabel(month) }}</p>
      </div>
      <input
        v-model="month"
        type="month"
        class="px-3 py-2 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        @change="onMonthChange"
      />
    </div>

    <div v-if="d" class="space-y-5">

      <!-- Budget summary -->
      <div class="bg-white rounded-xl border border-gray-200 p-5">
        <h3 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-4">Μηνιαίος προϋπολογισμός</h3>
        <div class="grid grid-cols-4 gap-4">
          <div class="text-center">
            <p class="text-xs text-gray-400 mb-1">Έσοδα</p>
            <p class="text-xl font-bold text-green-600">{{ formatAmount(d.total_income) }}</p>
          </div>
          <div class="text-center">
            <p class="text-xs text-gray-400 mb-1">Λογαριασμοί</p>
            <p class="text-xl font-bold text-gray-800">−{{ formatAmount(d.amount_bills) }}</p>
          </div>
          <div class="text-center">
            <p class="text-xs text-gray-400 mb-1">Αγορές</p>
            <p class="text-xl font-bold text-amber-600">−{{ formatAmount(d.amount_expenses_planned) }}</p>
          </div>
          <div class="text-center border-l border-gray-100 pl-4">
            <p class="text-xs text-gray-400 mb-1">Διαθέσιμο</p>
            <p class="text-xl font-bold" :class="d.remaining >= 0 ? 'text-blue-600' : 'text-red-600'">
              {{ formatAmount(d.remaining) }}
            </p>
          </div>
        </div>

        <!-- Progress bar -->
        <div v-if="d.total_income > 0" class="mt-4">
          <div class="h-2 bg-gray-100 rounded-full overflow-hidden flex">
            <div
              class="h-full bg-red-400 transition-all"
              :style="{ width: Math.min((d.amount_bills / d.total_income) * 100, 100) + '%' }"
            />
            <div
              class="h-full bg-amber-400 transition-all"
              :style="{ width: Math.min((d.amount_expenses_planned / d.total_income) * 100, 100) + '%' }"
            />
          </div>
          <div class="flex gap-4 mt-1.5 text-xs text-gray-400">
            <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-red-400 inline-block"></span>Λογαριασμοί</span>
            <span class="flex items-center gap-1"><span class="w-2 h-2 rounded-full bg-amber-400 inline-block"></span>Αγορές</span>
          </div>
        </div>
      </div>

      <!-- Income section -->
      <div class="bg-white rounded-xl border border-gray-200 p-5">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">Έσοδα μήνα</h3>
          <div class="flex items-center gap-3">
            <button
              v-if="selectedIncomes.size > 0"
              @click="bulkDeleteIncomes"
              class="text-xs font-medium text-white bg-red-500 hover:bg-red-600 px-2.5 py-1.5 rounded-lg transition-colors"
            >
              Διαγραφή ({{ selectedIncomes.size }})
            </button>
            <button
              @click="showIncomeModal = true"
              class="text-xs text-blue-600 hover:text-blue-700 font-medium"
            >
              + Προσθήκη
            </button>
          </div>
        </div>

        <div v-if="budgetStore.incomes.length === 0" class="text-sm text-gray-400 py-2">
          Δεν έχεις προσθέσει έσοδα για αυτόν τον μήνα.
        </div>

        <div v-else class="space-y-1">
          <div
            v-for="inc in budgetStore.incomes"
            :key="inc.id"
            class="flex items-center gap-3 py-1.5 px-2 rounded-lg transition-colors cursor-pointer"
            :class="selectedIncomes.has(inc.id) ? 'bg-blue-50' : 'hover:bg-gray-50'"
            @click="toggleIncomeSelect(inc.id)"
          >
            <input
              type="checkbox"
              :checked="selectedIncomes.has(inc.id)"
              @click.stop
              @change="toggleIncomeSelect(inc.id)"
              class="w-4 h-4 rounded border-gray-300 accent-blue-600 cursor-pointer shrink-0"
            />
            <span class="text-sm text-gray-700 flex-1">{{ inc.description }}</span>
            <div class="flex items-center gap-3">
              <span class="text-sm font-semibold text-green-600">+{{ formatAmount(inc.amount) }}</span>
              <button
                @click.stop="removeIncome(inc.id)"
                class="text-gray-300 hover:text-red-400 text-sm transition-colors"
              >✕</button>
            </div>
          </div>
          <div class="flex justify-between pt-2 border-t border-gray-100 mt-1">
            <span class="text-sm font-semibold text-gray-700">Σύνολο</span>
            <span class="text-sm font-bold text-green-600">{{ formatAmount(totalIncome) }}</span>
          </div>
        </div>
      </div>

      <!-- Bill stats + planned expenses -->
      <div class="grid grid-cols-3 gap-4">

        <!-- Pending bills -->
        <div class="bg-white rounded-xl border border-gray-200 p-4 flex flex-col gap-3">
          <div class="flex items-center justify-between">
            <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide">Εκκρεμείς λογαριασμοί</p>
            <span class="text-xs font-bold text-blue-600">{{ formatAmount(d.amount_bills - d.amount_overdue) }}</span>
          </div>
          <p v-if="pendingBills.length === 0" class="text-sm text-gray-400">Κανένας εκκρεμής</p>
          <ul v-else class="space-y-2">
            <li v-for="b in pendingBills" :key="b.id" class="flex items-center justify-between">
              <div class="flex items-center gap-2 min-w-0">
                <div class="w-6 h-6 rounded-lg flex items-center justify-center text-white text-xs font-bold shrink-0"
                  :style="{ backgroundColor: b.user_provider.provider.color }">
                  {{ b.user_provider.provider.name.charAt(0) }}
                </div>
                <span class="text-sm text-gray-700 truncate">{{ b.user_provider.nickname || b.user_provider.provider.name }}</span>
              </div>
              <div class="text-right shrink-0 ml-2">
                <p class="text-sm font-semibold text-gray-900">{{ formatAmount(b.amount) }}</p>
                <p class="text-xs text-gray-400">{{ formatDate(b.due_date) }}</p>
              </div>
            </li>
          </ul>
        </div>

        <!-- Overdue bills -->
        <div class="bg-white rounded-xl border border-red-100 p-4 flex flex-col gap-3">
          <div class="flex items-center justify-between">
            <p class="text-xs font-semibold text-red-400 uppercase tracking-wide">Ληξιπρόθεσμοι</p>
            <span class="text-xs font-bold text-red-500">{{ formatAmount(d.amount_overdue) }}</span>
          </div>
          <p v-if="overdueBills.length === 0" class="text-sm text-gray-400">Κανένας ληξιπρόθεσμος</p>
          <ul v-else class="space-y-2">
            <li v-for="b in overdueBills" :key="b.id" class="flex items-center justify-between">
              <div class="flex items-center gap-2 min-w-0">
                <div class="w-6 h-6 rounded-lg flex items-center justify-center text-white text-xs font-bold shrink-0"
                  :style="{ backgroundColor: b.user_provider.provider.color }">
                  {{ b.user_provider.provider.name.charAt(0) }}
                </div>
                <span class="text-sm text-gray-700 truncate">{{ b.user_provider.nickname || b.user_provider.provider.name }}</span>
              </div>
              <div class="text-right shrink-0 ml-2">
                <p class="text-sm font-semibold text-red-600">{{ formatAmount(b.amount) }}</p>
                <p class="text-xs text-red-400">{{ formatDate(b.due_date) }}</p>
              </div>
            </li>
          </ul>
        </div>

        <!-- Planned expenses -->
        <div class="bg-white rounded-xl border border-amber-100 p-4 flex flex-col gap-3">
          <div class="flex items-center justify-between">
            <p class="text-xs font-semibold text-amber-500 uppercase tracking-wide">Αγορές μήνα</p>
            <span class="text-xs font-bold text-amber-600">{{ formatAmount(d.amount_expenses_planned) }}</span>
          </div>
          <p v-if="d.planned_expenses.length === 0" class="text-sm text-gray-400">Καμία αγορά</p>
          <ul v-else class="space-y-2">
            <li v-for="e in d.planned_expenses" :key="e.id" class="flex items-center justify-between">
              <span class="text-sm text-gray-700 truncate">{{ e.name }}</span>
              <div class="text-right shrink-0 ml-2">
                <p class="text-sm font-semibold text-amber-700">{{ formatAmount(e.amount) }}</p>
                <p v-if="e.store" class="text-xs text-gray-400">{{ e.store }}</p>
              </div>
            </li>
          </ul>
        </div>

      </div>

      <!-- Upcoming bills -->
      <div class="bg-white rounded-xl border border-gray-200 p-5">
        <h3 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-4">Επόμενοι λογαριασμοί</h3>
        <div v-if="d.upcoming_bills.length === 0" class="text-center py-6 text-gray-400 text-sm">
          Δεν υπάρχουν εκκρεμείς λογαριασμοί
        </div>
        <div v-else class="space-y-1">
          <div v-for="bill in d.upcoming_bills" :key="bill.id"
            class="flex items-center justify-between py-2.5 border-b border-gray-100 last:border-0">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-lg flex items-center justify-center text-white text-xs font-bold"
                :style="{ backgroundColor: bill.user_provider.provider.color }">
                {{ bill.user_provider.provider.name.charAt(0) }}
              </div>
              <div>
                <p class="text-sm font-medium text-gray-900">
                  {{ bill.user_provider.nickname || bill.user_provider.provider.name }}
                </p>
                <p class="text-xs text-gray-400">{{ categoryLabel(bill.user_provider.provider.category) }}</p>
              </div>
            </div>
            <div class="flex items-center gap-3">
              <p class="text-xs" :class="bill.status === 'overdue' ? 'text-red-500' : 'text-gray-400'">
                {{ formatDate(bill.due_date) }}
              </p>
              <p class="text-sm font-semibold text-gray-900">{{ formatAmount(bill.amount) }}</p>
              <span class="px-2 py-0.5 rounded-full text-xs font-medium" :class="statusClass(bill.status)">
                {{ statusLabel(bill.status) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="flex items-center justify-center h-64 text-gray-400">Φόρτωση...</div>

    <!-- Income modal -->
    <div v-if="showIncomeModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4">
      <div class="bg-white rounded-2xl shadow-xl w-full max-w-sm p-6">
        <h3 class="text-lg font-semibold text-gray-900 mb-5">Προσθήκη εσόδου</h3>
        <form @submit.prevent="addIncome" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Περιγραφή</label>
            <input v-model="incomeForm.description" type="text" required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="π.χ. Μισθός, Freelance..." />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Ποσό (€)</label>
            <input v-model="incomeForm.amount" type="number" step="0.01" min="0" required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="0.00" />
          </div>
          <div class="flex gap-3 pt-2">
            <button type="button" @click="showIncomeModal = false"
              class="flex-1 px-4 py-2.5 rounded-lg border border-gray-300 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
              Ακύρωση
            </button>
            <button type="submit"
              class="flex-1 px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors">
              Αποθήκευση
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
