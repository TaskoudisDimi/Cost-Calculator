<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useBillsStore } from '@/stores/bills'
import { useConfirm } from '@/composables/useConfirm'
import { formatDate, formatAmount, statusLabel, statusClass } from '@/utils/format'
import type { BillStatus } from '@/types'

const { confirm } = useConfirm()

const store = useBillsStore()

const showModal = ref(false)
const editingBill = ref<string | null>(null)
const filterStatus = ref<BillStatus | 'all'>('all')
const selected = ref<Set<string>>(new Set())

const scanning = ref(false)
const scanError = ref('')
const fileInput = ref<HTMLInputElement | null>(null)

const form = ref({
  user_provider_id: '',
  amount: '',
  due_date: '',
  issued_date: '',
  notes: '',
})

const filteredBills = computed(() => {
  if (filterStatus.value === 'all') return store.bills
  return store.bills.filter(b => b.status === filterStatus.value)
})

const allSelected = computed(
  () => filteredBills.value.length > 0 && filteredBills.value.every(b => selected.value.has(b.id))
)

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
  if (!await confirm({ message: `Θα διαγραφούν ${ids.length} λογαριασμοί. Η ενέργεια δεν αναιρείται.` })) return
  await store.bulkDeleteBills(ids)
  selected.value = new Set()
}

function openCreate() {
  editingBill.value = null
  form.value = { user_provider_id: '', amount: '', due_date: '', issued_date: '', notes: '' }
  scanError.value = ''
  showModal.value = true
}

function openEdit(id: string) {
  const bill = store.bills.find(b => b.id === id)
  if (!bill) return
  editingBill.value = id
  form.value = {
    user_provider_id: bill.user_provider_id,
    amount: String(bill.amount),
    due_date: bill.due_date.split('T')[0] ?? '',
    issued_date: bill.issued_date ? bill.issued_date.split('T')[0] ?? '' : '',
    notes: bill.notes,
  }
  scanError.value = ''
  showModal.value = true
}

async function submitForm() {
  const payload = {
    user_provider_id: form.value.user_provider_id,
    amount: parseFloat(form.value.amount),
    due_date: new Date(form.value.due_date).toISOString(),
    issued_date: form.value.issued_date ? new Date(form.value.issued_date).toISOString() : undefined,
    notes: form.value.notes,
  }
  if (editingBill.value) {
    await store.updateBill(editingBill.value, payload)
  } else {
    await store.createBill(payload)
  }
  showModal.value = false
  await store.fetchBills()
}

async function markPaid(id: string) {
  await store.markPaid(id)
}

async function deleteBill(id: string) {
  if (!await confirm({ message: 'Ο λογαριασμός θα διαγραφεί οριστικά.' })) return
  await store.deleteBill(id)
  selected.value.delete(id)
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
    // Try to match provider name to a user provider
    if (result.provider_name) {
      const name = result.provider_name.toLowerCase()
      const match = store.userProviders.find(up =>
        (up.nickname || up.provider.name).toLowerCase().includes(name) ||
        name.includes((up.nickname || up.provider.name).toLowerCase())
      )
      if (match) form.value.user_provider_id = match.id
    }
  } catch (e: any) {
    scanError.value = e.response?.data?.error || 'Αποτυχία ανάγνωσης λογαριασμού.'
  } finally {
    scanning.value = false
  }
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-5">
      <h2 class="text-xl md:text-2xl font-bold text-gray-50">Λογαριασμοί</h2>
      <button
        @click="openCreate"
        class="bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-3 md:px-4 py-2 rounded-lg transition-colors"
      >
        + Νέος
      </button>
    </div>

    <!-- Filter tabs -->
    <div class="flex gap-2 mb-5">
      <button
        v-for="f in (['all', 'pending', 'overdue', 'paid'] as const)"
        :key="f"
        @click="filterStatus = f; clearSelection()"
        class="px-3 py-1.5 rounded-lg text-sm font-medium transition-colors"
        :class="filterStatus === f ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-400 border border-gray-700 hover:bg-gray-700/60'"
      >
        {{ f === 'all' ? 'Όλοι' : statusLabel(f) }}
      </button>
    </div>

    <!-- Bulk action bar -->
    <div
      v-if="selected.size > 0"
      class="flex items-center justify-between bg-red-900/20 border border-red-200 rounded-xl px-4 py-3 mb-4"
    >
      <span class="text-sm font-medium text-red-400">{{ selected.size }} επιλεγμένοι</span>
      <div class="flex gap-2">
        <button
          @click="clearSelection"
          class="text-sm text-gray-400 hover:text-gray-300 px-3 py-1.5 rounded-lg border border-gray-600 bg-gray-800"
        >
          Ακύρωση
        </button>
        <button
          @click="bulkDelete"
          class="text-sm font-medium text-white bg-red-900/200 hover:bg-red-600 px-3 py-1.5 rounded-lg transition-colors"
        >
          Διαγραφή ({{ selected.size }})
        </button>
      </div>
    </div>

    <!-- Bills list -->
    <div v-if="store.loading" class="text-center py-16 text-gray-400">Φόρτωση...</div>

    <div v-else-if="filteredBills.length === 0" class="text-center py-16 text-gray-400 bg-gray-800 rounded-xl border border-gray-700">
      Δεν βρέθηκαν λογαριασμοί
    </div>

    <div v-else class="bg-gray-800 rounded-xl border border-gray-700 divide-y divide-gray-700">
      <!-- Select all header -->
      <div class="flex items-center gap-3 px-4 py-2 bg-gray-900/50 rounded-t-xl">
        <input
          type="checkbox"
          :checked="allSelected"
          @change="toggleSelectAll"
          class="w-4 h-4 rounded border-gray-600 accent-blue-600 cursor-pointer"
        />
        <span class="text-xs text-gray-400">Επιλογή όλων</span>
      </div>

      <div
        v-for="bill in filteredBills"
        :key="bill.id"
        class="flex items-center gap-4 p-4 hover:bg-gray-700/60 transition-colors"
        :class="selected.has(bill.id) ? 'bg-blue-900/20' : ''"
      >
        <input
          type="checkbox"
          :checked="selected.has(bill.id)"
          @change="toggleSelect(bill.id)"
          class="w-4 h-4 rounded border-gray-600 accent-blue-600 cursor-pointer shrink-0"
        />

        <div
          class="w-10 h-10 rounded-xl flex items-center justify-center text-white text-sm font-bold shrink-0"
          :style="{ backgroundColor: bill.user_provider.provider.color }"
        >
          {{ (bill.user_provider.nickname || bill.user_provider.provider.name).charAt(0).toUpperCase() }}
        </div>

        <div class="flex-1 min-w-0">
          <p class="text-sm font-semibold text-gray-50">
            {{ bill.user_provider.nickname || bill.user_provider.provider.name }}
          </p>
          <p class="text-xs text-gray-400 hidden sm:block">Λήξη: {{ formatDate(bill.due_date) }}</p>
          <p class="text-xs text-gray-400 sm:hidden">{{ formatDate(bill.due_date) }}</p>
        </div>

        <span class="px-2 py-1 rounded-full text-xs font-medium shrink-0" :class="statusClass(bill.status)">
          {{ statusLabel(bill.status) }}
        </span>

        <p class="text-base font-bold text-gray-50 w-24 text-right shrink-0">{{ formatAmount(bill.amount) }}</p>

        <div class="flex items-center gap-2 shrink-0">
          <button
            v-if="bill.status !== 'paid'"
            @click="markPaid(bill.id)"
            class="text-xs bg-gray-700 hover:bg-green-900/30 text-gray-300 hover:text-green-400 font-medium px-2 py-1.5 rounded-lg border border-gray-600 hover:border-green-700 transition-colors hidden sm:block"
          >
            ✓ Πλήρωσα
          </button>
          <a
            v-if="bill.user_provider.provider.payment_url"
            :href="bill.user_provider.provider.payment_url"
            target="_blank"
            rel="noopener"
            class="text-xs bg-gray-700 hover:bg-gray-200 text-gray-300 font-medium px-2.5 py-1.5 rounded-lg transition-colors"
          >
            Πληρωμή →
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

    <!-- Hidden file input for scanning -->
    <input
      ref="fileInput"
      type="file"
      accept="image/*,.pdf"
      class="hidden"
      @change="onFileSelected"
    />

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/40 flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0">
      <div class="bg-gray-800 rounded-2xl shadow-xl w-full max-w-md p-6">
        <div class="flex items-center justify-between mb-5">
          <h3 class="text-lg font-semibold text-gray-50">
            {{ editingBill ? 'Επεξεργασία' : 'Νέος λογαριασμός' }}
          </h3>
          <!-- Scan button (only for new bills) -->
          <button
            v-if="!editingBill"
            type="button"
            @click="triggerScan"
            :disabled="scanning"
            class="flex items-center gap-1.5 text-xs font-medium px-3 py-1.5 rounded-lg border border-gray-600 hover:bg-gray-700/60 text-gray-400 transition-colors disabled:opacity-60"
          >
            <span v-if="scanning" class="animate-spin">⏳</span>
            <span v-else>📷</span>
            {{ scanning ? 'Ανάλυση…' : 'Σάρωση λογαριασμού' }}
          </button>
        </div>

        <p v-if="scanError" class="text-red-400 text-xs mb-3 bg-red-900/20 px-3 py-2 rounded-lg">{{ scanError }}</p>

        <form @submit.prevent="submitForm" class="space-y-4">
          <div v-if="!editingBill">
            <label class="block text-sm font-medium text-gray-300 mb-1">Πάροχος</label>
            <select
              v-model="form.user_provider_id"
              required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="" disabled>Επίλεξε πάροχο</option>
              <option v-for="up in store.userProviders" :key="up.id" :value="up.id">
                {{ up.nickname || up.provider.name }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Ποσό (€)</label>
            <input
              v-model="form.amount"
              type="number"
              step="0.01"
              min="0"
              required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="0.00"
            />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium text-gray-300 mb-1">Ημ. λήξης</label>
              <input
                v-model="form.due_date"
                type="date"
                required
                class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-300 mb-1">Ημ. έκδοσης</label>
              <input
                v-model="form.issued_date"
                type="date"
                class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Σημειώσεις</label>
            <input
              v-model="form.notes"
              type="text"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Προαιρετικό"
            />
          </div>

          <div class="flex gap-3 pt-2">
            <button
              type="button"
              @click="showModal = false"
              class="flex-1 px-4 py-2.5 rounded-lg border border-gray-600 text-sm font-medium text-gray-300 hover:bg-gray-700/60 transition-colors"
            >
              Ακύρωση
            </button>
            <button
              type="submit"
              class="flex-1 px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors"
            >
              Αποθήκευση
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
