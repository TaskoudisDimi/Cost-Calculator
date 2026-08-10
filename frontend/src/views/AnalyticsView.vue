<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useBillsStore } from '@/stores/bills'
import { useLocale } from '@/composables/useLocale'
import { formatAmount, categoryLabel } from '@/utils/format'
import type { AnalyticsSummary } from '@/types'

const store = useBillsStore()
const { t } = useLocale()
const year = ref(String(new Date().getFullYear()))
const data = ref<AnalyticsSummary | null>(null)
const loading = ref(false)
const error = ref('')

const MONTHS_SHORT_EL = ['Ιαν', 'Φεβ', 'Μαρ', 'Απρ', 'Μαϊ', 'Ιουν', 'Ιουλ', 'Αυγ', 'Σεπ', 'Οκτ', 'Νοε', 'Δεκ']
const MONTHS_SHORT_EN = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

const monthsShort = computed(() => {
  const { locale } = useLocale()
  return locale.value === 'en' ? MONTHS_SHORT_EN : MONTHS_SHORT_EL
})

async function fetchData() {
  loading.value = true
  error.value = ''
  try {
    data.value = await store.fetchAnalytics(year.value)
  } catch {
    error.value = t('analytics.no_data') + ' ' + year.value + '.'
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)

// ── Chart calculations ──────────────────────────────────────────────────────

const CHART_W = 470
const CHART_H = 130
const SLOT_W = CHART_W / 12
const BAR_W = 14

const maxMonthValue = computed(() => {
  if (!data.value) return 1
  return Math.max(
    ...data.value.monthly.map(m => Math.max(m.income, m.bills + m.expenses)),
    1,
  )
})

function barH(val: number): number {
  return (val / maxMonthValue.value) * CHART_H
}

const chartMonths = computed(() => {
  if (!data.value) return []
  return data.value.monthly.map((m, i) => {
    const cx = i * SLOT_W + SLOT_W / 2
    const spending = m.bills + m.expenses
    return {
      ...m,
      spending,
      label: monthsShort.value[i],
      cx,
      incomeH: barH(m.income),
      spendingH: barH(spending),
      incomeX: cx - BAR_W - 2,
      spendingX: cx + 2,
    }
  })
})

const yGridValues = computed(() => {
  const max = maxMonthValue.value
  return [Math.round(max / 100) * 100, Math.round(max / 200) * 100, 0]
})

const sortedCategories = computed(() => {
  if (!data.value) return []
  const entries = Object.entries(data.value.by_category).sort((a, b) => b[1] - a[1])
  const maxVal = entries[0]?.[1] ?? 1
  return entries.map(([cat, amount]) => ({
    cat,
    label: categoryLabel(cat),
    amount,
    pct: (amount / maxVal) * 100,
  }))
})

const categoryColors: Record<string, string> = {
  energy: '#f59e0b',
  water: '#3b82f6',
  telecom: '#8b5cf6',
  streaming: '#ec4899',
  subscription: '#06b6d4',
  housing: '#10b981',
  finance: '#6366f1',
  car: '#f97316',
  other: '#6b7280',
}

function catColor(cat: string): string {
  return categoryColors[cat] ?? '#6b7280'
}

function prevYear() {
  year.value = String(Number(year.value) - 1)
  fetchData()
}

function nextYear() {
  const next = Number(year.value) + 1
  if (next <= new Date().getFullYear()) {
    year.value = String(next)
    fetchData()
  }
}
</script>

<template>
  <div class="max-w-2xl mx-auto">

    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-xl font-bold text-white">{{ t('analytics.title') }}</h2>
      <div class="flex items-center gap-2 bg-[#111119] border border-white/[0.06] rounded-xl px-1 py-1">
        <button @click="prevYear" class="p-1.5 text-gray-400 hover:text-white transition-colors rounded-lg hover:bg-white/[0.04]">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5 8.25 12l7.5-7.5" />
          </svg>
        </button>
        <span class="text-sm font-semibold text-white w-10 text-center">{{ year }}</span>
        <button @click="nextYear" :disabled="Number(year) >= new Date().getFullYear()"
          class="p-1.5 text-gray-400 hover:text-white transition-colors rounded-lg hover:bg-white/[0.04] disabled:opacity-30">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="space-y-4">
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div v-for="i in 4" :key="i" class="bg-[#111119] border border-white/[0.06] rounded-2xl h-20 animate-pulse" />
      </div>
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl h-52 animate-pulse" />
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl h-36 animate-pulse" />
    </div>

    <div v-else-if="error" class="text-center py-12 text-red-400 text-sm">{{ error }}</div>

    <div v-else-if="data" class="space-y-4">

      <!-- KPI cards -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div class="bg-[#111119] border border-white/[0.06] rounded-2xl p-4">
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">{{ t('analytics.income') }}</p>
          <p class="text-lg font-bold text-emerald-400">{{ formatAmount(data.total_income) }}</p>
        </div>
        <div class="bg-[#111119] border border-white/[0.06] rounded-2xl p-4">
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">{{ t('analytics.bills') }}</p>
          <p class="text-lg font-bold text-blue-400">{{ formatAmount(data.total_bills) }}</p>
        </div>
        <div class="bg-[#111119] border border-white/[0.06] rounded-2xl p-4">
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">{{ t('analytics.expenses') }}</p>
          <p class="text-lg font-bold text-amber-400">{{ formatAmount(data.total_expenses) }}</p>
        </div>
        <div class="bg-[#111119] border border-white/[0.06] rounded-2xl p-4"
          :class="data.total_saved >= 0 ? 'border-emerald-900/40' : 'border-red-900/40'">
          <p class="text-[11px] text-gray-500 mb-1 uppercase tracking-wider">{{ t('analytics.savings') }}</p>
          <p class="text-lg font-bold" :class="data.total_saved >= 0 ? 'text-white' : 'text-red-400'">
            {{ formatAmount(data.total_saved) }}
          </p>
        </div>
      </div>

      <!-- Bar chart -->
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl p-5">
        <div class="flex items-center justify-between mb-4">
          <p class="text-sm font-semibold text-gray-200">{{ t('analytics.monthly_overview') }}</p>
          <div class="flex items-center gap-3 text-xs text-gray-500">
            <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-sm bg-emerald-500 shrink-0" />{{ t('analytics.income_legend') }}</span>
            <span class="flex items-center gap-1.5"><span class="w-2.5 h-2.5 rounded-sm bg-blue-500 shrink-0" />{{ t('analytics.spending_legend') }}</span>
          </div>
        </div>

        <div v-if="chartMonths.every(m => m.income === 0 && m.spending === 0)" class="text-center py-8 text-gray-600 text-sm">
          {{ t('analytics.no_data') }} {{ year }}
        </div>

        <svg v-else viewBox="0 0 520 175" class="w-full overflow-visible">
          <!-- Grid lines -->
          <line x1="44" :y1="10 + CHART_H - (CHART_H * 1)" x2="510" :y2="10 + CHART_H - (CHART_H * 1)" stroke="#1f2937" stroke-width="1"/>
          <line x1="44" :y1="10 + CHART_H - (CHART_H * 0.5)" x2="510" :y2="10 + CHART_H - (CHART_H * 0.5)" stroke="#1f2937" stroke-width="1"/>
          <line x1="44" :y1="10 + CHART_H" x2="510" :y2="10 + CHART_H" stroke="#374151" stroke-width="1"/>

          <!-- Y axis labels -->
          <text x="40" :y="10 + CHART_H - (CHART_H * 1) + 4" text-anchor="end" fill="#4b5563" font-size="9">{{ yGridValues[0] }}</text>
          <text x="40" :y="10 + CHART_H - (CHART_H * 0.5) + 4" text-anchor="end" fill="#4b5563" font-size="9">{{ yGridValues[1] }}</text>
          <text x="40" :y="10 + CHART_H + 4" text-anchor="end" fill="#4b5563" font-size="9">0</text>

          <!-- Bars group -->
          <g transform="translate(44, 10)">
            <g v-for="m in chartMonths" :key="m.month">
              <rect
                :x="m.incomeX"
                :y="CHART_H - m.incomeH"
                :width="BAR_W"
                :height="m.incomeH || 1"
                fill="#10b981"
                rx="2"
                opacity="0.75"
              >
                <title>{{ t('analytics.income_legend') }} {{ m.label }}: {{ formatAmount(m.income) }}</title>
              </rect>
              <rect
                :x="m.spendingX"
                :y="CHART_H - m.spendingH"
                :width="BAR_W"
                :height="m.spendingH || 1"
                fill="#3b82f6"
                rx="2"
                opacity="0.75"
              >
                <title>{{ t('analytics.spending_legend') }} {{ m.label }}: {{ formatAmount(m.spending) }}</title>
              </rect>
              <text :x="m.cx" :y="CHART_H + 18" text-anchor="middle" fill="#4b5563" font-size="9">{{ m.label }}</text>
            </g>
          </g>
        </svg>
      </div>

      <!-- Category breakdown -->
      <div v-if="sortedCategories.length > 0" class="bg-[#111119] border border-white/[0.06] rounded-2xl p-5">
        <p class="text-sm font-semibold text-gray-200 mb-4">{{ t('analytics.categories_title') }}</p>
        <div class="space-y-3">
          <div v-for="item in sortedCategories" :key="item.cat" class="flex items-center gap-3">
            <span class="text-xs text-gray-400 w-28 shrink-0 truncate">{{ item.label }}</span>
            <div class="flex-1 bg-white/[0.04] rounded-full h-1.5 overflow-hidden">
              <div
                class="h-1.5 rounded-full transition-all duration-500"
                :style="{ width: item.pct + '%', backgroundColor: catColor(item.cat) }"
              />
            </div>
            <span class="text-xs font-semibold text-gray-200 w-20 text-right shrink-0">{{ formatAmount(item.amount) }}</span>
          </div>
        </div>
      </div>

      <!-- Monthly table -->
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl overflow-hidden">
        <div class="grid grid-cols-4 px-4 py-2.5 text-[11px] font-semibold text-gray-500 uppercase tracking-wider bg-white/[0.04]">
          <span>{{ t('analytics.month_col') }}</span>
          <span class="text-right">{{ t('analytics.income') }}</span>
          <span class="text-right">{{ t('analytics.bills_short') }}</span>
          <span class="text-right">{{ t('analytics.expenses') }}</span>
        </div>
        <div
          v-for="m in [...data.monthly].reverse()"
          :key="m.month"
          class="grid grid-cols-4 px-4 py-2.5 border-t border-white/[0.05] text-sm"
          :class="(m.bills + m.expenses > 0 || m.income > 0) ? '' : 'opacity-30'"
        >
          <span class="text-gray-400 text-xs">{{ monthsShort[Number(m.month.slice(5, 7)) - 1] }}</span>
          <span class="text-right text-emerald-400 text-xs font-medium">
            {{ m.income > 0 ? formatAmount(m.income) : '—' }}
          </span>
          <span class="text-right text-gray-200 text-xs font-medium">
            {{ m.bills > 0 ? formatAmount(m.bills) : '—' }}
          </span>
          <span class="text-right text-amber-400 text-xs font-medium">
            {{ m.expenses > 0 ? formatAmount(m.expenses) : '—' }}
          </span>
        </div>
      </div>

    </div>

  </div>
</template>
