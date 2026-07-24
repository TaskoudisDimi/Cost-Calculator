<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useBudgetStore } from '@/stores/budget'
import { useConfirm } from '@/composables/useConfirm'
import { useLocale } from '@/composables/useLocale'
import { formatAmount, formatDate, expenseCategoryLabel, expenseCategoryIcon, expenseStatusLabel, expenseStatusClass, currentMonth, monthLabel } from '@/utils/format'
import type { ExpenseCategory } from '@/types'

const { confirm } = useConfirm()
const { t, locale } = useLocale()

const store = useBudgetStore()
const { expenses, loading } = storeToRefs(store)
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

const categories = computed<{ value: ExpenseCategory; label: string }[]>(() => [
  { value: 'shopping', label: t('expenses.categories.shopping') },
  { value: 'food', label: t('expenses.categories.food') },
  { value: 'transport', label: t('expenses.categories.transport') },
  { value: 'health', label: t('expenses.categories.health') },
  { value: 'entertainment', label: t('expenses.categories.entertainment') },
  { value: 'other', label: t('expenses.categories.other') },
])

const filtered = computed(() => {
  const all = Array.isArray(expenses.value) ? expenses.value : []
  if (filterStatus.value === 'all') return all
  return all.filter(e => e.status === filterStatus.value)
})

const totalPlanned = computed(() =>
  (Array.isArray(expenses.value) ? expenses.value : []).filter(e => e.status === 'planned').reduce((s, e) => s + e.amount, 0)
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
  const msg = locale.value === 'en' ? 'Expense will be permanently deleted.' : 'Η αγορά θα διαγραφεί οριστικά.'
  if (!await confirm({ message: msg })) return
  await store.deleteExpense(id)
}
</script>

<template>
  <div>
    <div class="flex flex-wrap items-start justify-between gap-3 mb-5">
      <div>
        <h2 class="text-xl md:text-2xl font-bold text-gray-50">{{ t('expenses.title') }}</h2>
        <p class="text-sm text-gray-400 mt-0.5 capitalize">
          {{ monthLabel(month) }} · <span class="font-semibold text-amber-400">{{ formatAmount(totalPlanned) }}</span>
        </p>
      </div>
      <div class="flex items-center gap-2">
        <input
          v-model="month"
          type="month"
          class="px-3 py-2 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          @change="onMonthChange"
        />
        <button
          @click="showModal = true"
          class="bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-3 md:px-4 py-2 rounded-lg transition-colors whitespace-nowrap"
        >
          {{ t('expenses.new_expense') }}
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
        :class="filterStatus === f ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-400 border border-gray-700 hover:bg-gray-700/60'"
      >
        {{ f === 'all' ? t('expenses.all_filter') : expenseStatusLabel(f) }}
      </button>
    </div>

    <div v-if="loading" class="space-y-2">
      <div v-for="i in 4" :key="i" class="bg-gray-900 rounded-2xl border border-gray-800 h-16 animate-pulse" />
    </div>

    <div v-else-if="filtered.length === 0" class="text-center py-16 bg-gray-900 rounded-2xl border border-gray-800">
      <div class="text-3xl mb-3">🛍️</div>
      <p class="text-sm font-medium text-gray-400">{{ t('expenses.no_expenses') }}</p>
      <button @click="showModal = true" class="mt-3 text-xs text-blue-400 hover:text-blue-300 transition-colors">
        {{ t('expenses.add_expense') }}
      </button>
    </div>

    <div v-else class="bg-gray-900 rounded-2xl border border-gray-800 divide-y divide-gray-800 overflow-hidden">
      <div
        v-for="e in filtered"
        :key="e.id"
        class="flex items-center gap-3 md:gap-4 p-4 hover:bg-gray-800/40 transition-colors"
        :class="e.status === 'bought' ? 'opacity-50' : ''"
      >
        <div class="w-9 h-9 rounded-xl bg-gray-800 flex items-center justify-center text-base shrink-0">
          {{ expenseCategoryIcon(e.category) }}
        </div>

        <div class="flex-1 min-w-0">
          <p class="text-sm font-semibold text-gray-100" :class="e.status === 'bought' ? 'line-through' : ''">
            {{ e.name }}
          </p>
          <div class="flex items-center gap-1.5 mt-0.5 flex-wrap">
            <span class="text-xs text-gray-500">{{ expenseCategoryLabel(e.category) }}</span>
            <span v-if="e.store" class="text-xs text-gray-500">· {{ e.store }}</span>
            <span v-if="e.planned_date" class="text-xs text-gray-500">· {{ formatDate(e.planned_date) }}</span>
          </div>
          <p v-if="e.notes" class="text-xs text-gray-500 mt-0.5 italic truncate">{{ e.notes }}</p>
        </div>

        <span class="hidden sm:inline px-2 py-0.5 rounded-full text-xs font-medium shrink-0" :class="expenseStatusClass(e.status)">
          {{ expenseStatusLabel(e.status) }}
        </span>

        <p class="text-sm font-bold text-gray-100 w-20 text-right shrink-0">{{ formatAmount(e.amount) }}</p>

        <div class="flex items-center gap-1.5 shrink-0">
          <button
            v-if="e.status === 'planned'"
            @click="markBought(e.id)"
            class="text-xs bg-emerald-900/20 hover:bg-emerald-900/30 text-emerald-400 font-medium px-2.5 py-1.5 rounded-lg border border-emerald-900/30 transition-colors whitespace-nowrap"
          >
            <span class="hidden sm:inline">{{ t('expenses.bought') }}</span>
            <span class="sm:hidden">✓</span>
          </button>
          <button
            @click="remove(e.id)"
            class="text-gray-500 hover:text-red-400 px-1.5 py-1.5 rounded-lg transition-colors text-sm leading-none"
          >
            ✕
          </button>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0">
      <div class="bg-gray-900 border border-gray-800 rounded-2xl shadow-2xl w-full max-w-md p-6">
        <h3 class="text-base font-semibold text-gray-50 mb-5">{{ t('expenses.modal_title') }}</h3>

        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.what_to_buy') }}</label>
            <input
              v-model="form.name"
              type="text"
              required
              autofocus
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              placeholder="π.χ. Βιβλιοθήκη BILLY"
            />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.amount') }}</label>
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
            <div>
              <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.category') }}</label>
              <select
                v-model="form.category"
                class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              >
                <option v-for="c in categories" :key="c.value" :value="c.value">{{ c.label }}</option>
              </select>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.store_label') }}</label>
              <input
                v-model="form.store"
                type="text"
                class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                placeholder="π.χ. IKEA"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('expenses.date') }}</label>
              <input
                v-model="form.planned_date"
                type="date"
                class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              />
            </div>
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ t('common.notes') }}</label>
            <input
              v-model="form.notes"
              type="text"
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              :placeholder="t('common.optional')"
            />
          </div>

          <div class="flex gap-3 pt-1">
            <button type="button" @click="showModal = false"
              class="flex-1 px-4 py-2.5 rounded-xl border border-gray-700 text-sm font-medium text-gray-300 hover:bg-gray-800 transition-colors">
              {{ t('common.cancel') }}
            </button>
            <button type="submit"
              class="flex-1 px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium transition-colors">
              {{ t('expenses.add') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
