<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useBudgetStore } from '@/stores/budget'
import { useConfirm } from '@/composables/useConfirm'
import { formatAmount, formatDate, expenseCategoryLabel, expenseStatusLabel, expenseStatusClass, currentMonth, monthLabel } from '@/utils/format'
import type { ExpenseCategory } from '@/types'

const { confirm } = useConfirm()

const store = useBudgetStore()
const showModal = ref(false)
const filterStatus = ref<'all' | 'planned' | 'bought'>('all')
const month = ref(currentMonth())

const form = ref({
  name: '',
  amount: '',
  category: 'other' as ExpenseCategory,
  store: '',
  notes: '',
  planned_date: '',
})

const categories: { value: ExpenseCategory; label: string }[] = [
  { value: 'shopping', label: 'Ψώνια' },
  { value: 'food', label: 'Φαγητό' },
  { value: 'transport', label: 'Μεταφορά' },
  { value: 'health', label: 'Υγεία' },
  { value: 'entertainment', label: 'Ψυχαγωγία' },
  { value: 'other', label: 'Άλλο' },
]

const filtered = computed(() => {
  if (filterStatus.value === 'all') return store.expenses
  return store.expenses.filter(e => e.status === filterStatus.value)
})

const totalPlanned = computed(() =>
  store.expenses.filter(e => e.status === 'planned').reduce((s, e) => s + e.amount, 0)
)

onMounted(() => store.fetchExpenses(month.value))

async function onMonthChange() {
  await store.fetchExpenses(month.value)
}

async function submit() {
  await store.createExpense({
    name: form.value.name,
    amount: parseFloat(form.value.amount),
    category: form.value.category,
    month: month.value,
    store: form.value.store,
    notes: form.value.notes,
    planned_date: form.value.planned_date ? new Date(form.value.planned_date).toISOString() : undefined,
  })
  showModal.value = false
  form.value = { name: '', amount: '', category: 'other', store: '', notes: '', planned_date: '' }
}

async function markBought(id: string) {
  await store.markBought(id)
}

async function remove(id: string) {
  if (!await confirm({ message: 'Η αγορά θα διαγραφεί οριστικά.' })) return
  await store.deleteExpense(id)
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-2xl font-bold text-gray-900">Αγορές</h2>
        <p class="text-sm text-gray-400 mt-0.5 capitalize">
          {{ monthLabel(month) }} · Προγραμματισμένες: <span class="font-semibold text-amber-600">{{ formatAmount(totalPlanned) }}</span>
        </p>
      </div>
      <div class="flex items-center gap-3">
        <input
          v-model="month"
          type="month"
          class="px-3 py-2 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          @change="onMonthChange"
        />
        <button
          @click="showModal = true"
          class="bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          + Νέα αγορά
        </button>
      </div>
    </div>

    <!-- Filter tabs -->
    <div class="flex gap-2 mb-5">
      <button
        v-for="f in (['all', 'planned', 'bought'] as const)"
        :key="f"
        @click="filterStatus = f"
        class="px-3 py-1.5 rounded-lg text-sm font-medium transition-colors"
        :class="filterStatus === f ? 'bg-blue-600 text-white' : 'bg-white text-gray-600 border border-gray-200 hover:bg-gray-50'"
      >
        {{ f === 'all' ? 'Όλες' : expenseStatusLabel(f) }}
      </button>
    </div>

    <div v-if="store.loading" class="text-center py-16 text-gray-400">Φόρτωση...</div>

    <div v-else-if="filtered.length === 0" class="text-center py-16 text-gray-400 bg-white rounded-xl border border-gray-200">
      Δεν βρέθηκαν αγορές
    </div>

    <div v-else class="bg-white rounded-xl border border-gray-200 divide-y divide-gray-100">
      <div
        v-for="e in filtered"
        :key="e.id"
        class="flex items-center gap-4 p-4 hover:bg-gray-50 transition-colors"
        :class="e.status === 'bought' ? 'opacity-60' : ''"
      >
        <div class="flex-1 min-w-0">
          <p class="text-sm font-semibold text-gray-900" :class="e.status === 'bought' ? 'line-through' : ''">
            {{ e.name }}
          </p>
          <div class="flex items-center gap-2 mt-0.5">
            <span class="text-xs text-gray-400">{{ expenseCategoryLabel(e.category) }}</span>
            <span v-if="e.store" class="text-xs text-gray-400">· {{ e.store }}</span>
            <span v-if="e.planned_date" class="text-xs text-gray-400">· {{ formatDate(e.planned_date) }}</span>
          </div>
          <p v-if="e.notes" class="text-xs text-gray-400 mt-0.5 italic">{{ e.notes }}</p>
        </div>

        <span class="px-2 py-1 rounded-full text-xs font-medium shrink-0" :class="expenseStatusClass(e.status)">
          {{ expenseStatusLabel(e.status) }}
        </span>

        <p class="text-base font-bold text-gray-900 w-24 text-right shrink-0">{{ formatAmount(e.amount) }}</p>

        <div class="flex items-center gap-2 shrink-0">
          <button
            v-if="e.status === 'planned'"
            @click="markBought(e.id)"
            class="text-xs bg-green-50 hover:bg-green-100 text-green-700 font-medium px-2.5 py-1.5 rounded-lg transition-colors"
          >
            Αγοράστηκε
          </button>
          <button
            @click="remove(e.id)"
            class="text-xs text-gray-400 hover:text-red-500 px-1.5 py-1.5 rounded-lg transition-colors"
          >
            ✕
          </button>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4">
      <div class="bg-white rounded-2xl shadow-xl w-full max-w-md p-6">
        <h3 class="text-lg font-semibold text-gray-900 mb-5">Νέα αγορά</h3>

        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Τι θέλεις να αγοράσεις;</label>
            <input
              v-model="form.name"
              type="text"
              required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="π.χ. Βιβλιοθήκη BILLY"
            />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Ποσό (€)</label>
              <input
                v-model="form.amount"
                type="number"
                step="0.01"
                min="0"
                required
                class="w-full px-3 py-2.5 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="0.00"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Κατηγορία</label>
              <select
                v-model="form.category"
                class="w-full px-3 py-2.5 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option v-for="c in categories" :key="c.value" :value="c.value">{{ c.label }}</option>
              </select>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Κατάστημα</label>
              <input
                v-model="form.store"
                type="text"
                class="w-full px-3 py-2.5 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="π.χ. IKEA"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Ημερομηνία</label>
              <input
                v-model="form.planned_date"
                type="date"
                class="w-full px-3 py-2.5 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Σημειώσεις</label>
            <input
              v-model="form.notes"
              type="text"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Προαιρετικό"
            />
          </div>

          <div class="flex gap-3 pt-2">
            <button type="button" @click="showModal = false"
              class="flex-1 px-4 py-2.5 rounded-lg border border-gray-300 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
              Ακύρωση
            </button>
            <button type="submit"
              class="flex-1 px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors">
              Προσθήκη
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
