<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useBillsStore } from '@/stores/bills'
import { useBudgetStore } from '@/stores/budget'
import { formatAmount } from '@/utils/format'

const emit = defineEmits<{ (e: 'close'): void }>()
const router = useRouter()

const billsStore = useBillsStore()
const budgetStore = useBudgetStore()
const { bills } = storeToRefs(billsStore)
const { expenses } = storeToRefs(budgetStore)

const query = ref('')
const inputRef = ref<HTMLInputElement | null>(null)
const selectedIndex = ref(0)

interface SearchResult {
  type: 'bill' | 'expense'
  id: string
  label: string
  sub: string
  amount: number
  route: string
}

const results = computed<SearchResult[]>(() => {
  const q = query.value.trim().toLowerCase()
  if (q.length < 1) return []

  const out: SearchResult[] = []

  const allBills = Array.isArray(bills.value) ? bills.value : []
  for (const b of allBills) {
    const name = b.user_provider?.provider?.name ?? b.user_provider?.nickname ?? ''
    const category = b.user_provider?.provider?.category ?? ''
    const notes = b.notes ?? ''
    if (name.toLowerCase().includes(q) || category.toLowerCase().includes(q) || notes.toLowerCase().includes(q)) {
      out.push({
        type: 'bill',
        id: b.id,
        label: name || 'Λογαριασμός',
        sub: b.due_date ? b.due_date.slice(0, 10) : '',
        amount: b.amount,
        route: '/bills',
      })
    }
  }

  const allExpenses = Array.isArray(expenses.value) ? expenses.value : []
  for (const e of allExpenses) {
    const name = e.name ?? ''
    const store = e.store ?? ''
    if (name.toLowerCase().includes(q) || store.toLowerCase().includes(q)) {
      out.push({
        type: 'expense',
        id: e.id,
        label: name,
        sub: store || e.category,
        amount: e.amount,
        route: '/expenses',
      })
    }
  }

  return out.slice(0, 8)
})

watch(query, () => { selectedIndex.value = 0 })

function select(r: SearchResult) {
  router.push(r.route)
  emit('close')
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') { e.preventDefault(); selectedIndex.value = Math.min(selectedIndex.value + 1, results.value.length - 1) }
  if (e.key === 'ArrowUp') { e.preventDefault(); selectedIndex.value = Math.max(selectedIndex.value - 1, 0) }
  if (e.key === 'Enter') { const r = results.value[selectedIndex.value]; if (r) select(r) }
  if (e.key === 'Escape') emit('close')
}

onMounted(() => nextTick(() => inputRef.value?.focus()))
</script>

<template>
  <div class="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-start justify-center pt-[10vh] px-4" @click.self="$emit('close')">
    <div class="w-full max-w-lg bg-[#111119] border border-white/[0.08] rounded-2xl shadow-2xl overflow-hidden">

      <!-- Input -->
      <div class="flex items-center gap-3 px-4 py-3.5 border-b border-white/[0.06]">
        <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-gray-500 shrink-0" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 15.803a7.5 7.5 0 0 0 10.607 0Z" />
        </svg>
        <input
          ref="inputRef"
          v-model="query"
          type="text"
          placeholder="Αναζήτηση λογαριασμών, αγορών…"
          class="flex-1 bg-transparent text-sm text-gray-100 placeholder-gray-500 outline-none"
          @keydown="onKey"
        />
        <kbd class="text-[10px] text-gray-600 border border-white/[0.08] rounded px-1.5 py-0.5 font-mono">Esc</kbd>
      </div>

      <!-- Results -->
      <div v-if="results.length > 0" class="divide-y divide-white/[0.04] max-h-80 overflow-y-auto">
        <button
          v-for="(r, i) in results"
          :key="r.id"
          class="w-full flex items-center gap-3 px-4 py-3 text-left transition-colors"
          :class="i === selectedIndex ? 'bg-violet-600/10' : 'hover:bg-white/[0.03]'"
          @click="select(r)"
          @mouseenter="selectedIndex = i"
        >
          <div class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0 text-sm"
            :class="r.type === 'bill' ? 'bg-violet-900/30 text-violet-400' : 'bg-amber-900/30 text-amber-400'">
            {{ r.type === 'bill' ? '📄' : '🛍️' }}
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-gray-100 truncate">{{ r.label }}</p>
            <p class="text-xs text-gray-500 truncate">{{ r.sub }}</p>
          </div>
          <span class="text-sm font-semibold text-gray-300 shrink-0">{{ formatAmount(r.amount) }}</span>
        </button>
      </div>

      <div v-else-if="query.length > 0" class="px-4 py-8 text-center">
        <p class="text-sm text-gray-500">Δεν βρέθηκαν αποτελέσματα για "<span class="text-gray-300">{{ query }}</span>"</p>
      </div>

      <div v-else class="px-4 py-4 flex gap-4">
        <span class="text-xs text-gray-600">↑↓ πλοήγηση</span>
        <span class="text-xs text-gray-600">↵ μετάβαση</span>
        <span class="text-xs text-gray-600">Esc κλείσιμο</span>
      </div>
    </div>
  </div>
</template>
