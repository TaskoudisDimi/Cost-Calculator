<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api/client'
import { useLocale } from '@/composables/useLocale'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const { locale } = useLocale()
const { toast } = useToast()

interface Institution {
  id: string
  name: string
  bic: string
  logo: string
}

interface BankStatus {
  connected: boolean
  status: string
  institution_name: string
  accounts: number
  requisition_id?: string
}

const institutions = ref<Institution[]>([])
const status = ref<BankStatus | null>(null)
const loading = ref(true)
const loadingInstitutions = ref(false)
const syncing = ref(false)
const disconnecting = ref(false)
const connecting = ref('')
const searchQuery = ref('')
const showInstitutions = ref(false)
const syncResult = ref<{ transactions_found: number; bills_matched: number } | null>(null)
const isCallback = computed(() => route.path.includes('/callback'))

async function fetchStatus() {
  try {
    const { data } = await api.get<BankStatus>('/bank/status')
    status.value = data
  } catch {
    status.value = null
  } finally {
    loading.value = false
  }
}

async function fetchInstitutions() {
  loadingInstitutions.value = true
  try {
    const { data } = await api.get<Institution[]>('/bank/institutions')
    institutions.value = data
    showInstitutions.value = true
  } catch (e: any) {
    const msg = locale.value === 'en'
      ? 'Failed to load banks. Check if bank integration is configured.'
      : 'Αποτυχία φόρτωσης τραπεζών. Βεβαιώσου ότι έχεις ορίσει τα API κλειδιά Nordigen.'
    toast(msg, 'error')
  } finally {
    loadingInstitutions.value = false
  }
}

async function connect(inst: Institution) {
  connecting.value = inst.id
  try {
    const { data } = await api.post<{ link: string }>('/bank/connect', {
      institution_id: inst.id,
      institution_name: inst.name,
    })
    // Open the bank OAuth in the same tab — the bank will redirect back to /bank/callback
    window.location.href = data.link
  } catch (e: any) {
    const msg = locale.value === 'en' ? 'Connection failed.' : 'Αποτυχία σύνδεσης.'
    toast(msg, 'error')
    connecting.value = ''
  }
}

async function sync() {
  syncing.value = true
  syncResult.value = null
  try {
    const { data } = await api.post<{ transactions_found: number; bills_matched: number }>('/bank/sync')
    syncResult.value = data
    await fetchStatus()
    const msg = locale.value === 'en'
      ? `Synced: ${data.bills_matched} bill(s) auto-marked as paid`
      : `Συγχρονισμός: ${data.bills_matched} λογαριασμός(οί) πληρώθηκαν αυτόματα`
    toast(msg, data.bills_matched > 0 ? 'success' : 'info')
  } catch (e: any) {
    const msg = locale.value === 'en' ? 'Sync failed.' : 'Αποτυχία συγχρονισμού.'
    toast(msg, 'error')
  } finally {
    syncing.value = false
  }
}

async function disconnect() {
  if (!confirm(locale.value === 'en' ? 'Disconnect your bank?' : 'Αποσύνδεση τράπεζας;')) return
  disconnecting.value = true
  try {
    await api.delete('/bank/disconnect')
    status.value = null
    showInstitutions.value = false
    syncResult.value = null
    const msg = locale.value === 'en' ? 'Bank disconnected' : 'Τράπεζα αποσυνδέθηκε'
    toast(msg, 'info')
  } catch {
    const msg = locale.value === 'en' ? 'Failed to disconnect.' : 'Αποτυχία αποσύνδεσης.'
    toast(msg, 'error')
  } finally {
    disconnecting.value = false
  }
}

const filteredInstitutions = computed(() => {
  if (!searchQuery.value.trim()) return institutions.value
  const q = searchQuery.value.toLowerCase()
  return institutions.value.filter(i => i.name.toLowerCase().includes(q))
})

const statusLabel = computed(() => {
  if (!status.value) return ''
  const s = status.value.status
  if (s === 'LN') return locale.value === 'en' ? 'Connected' : 'Συνδεδεμένος'
  if (s === 'CR') return locale.value === 'en' ? 'Awaiting authorization' : 'Αναμονή εξουσιοδότησης'
  if (s === 'RJ') return locale.value === 'en' ? 'Rejected' : 'Απορρίφθηκε'
  if (s === 'EX') return locale.value === 'en' ? 'Expired' : 'Έληξε'
  return s
})

onMounted(async () => {
  await fetchStatus()
  // If we're on the callback page, refresh status (bank completed OAuth)
  if (isCallback.value) {
    // Give Nordigen a moment to update the requisition status
    setTimeout(fetchStatus, 2000)
  }
})
</script>

<template>
  <div class="max-w-2xl mx-auto">

    <!-- Header -->
    <div class="mb-6">
      <h2 class="text-xl font-bold text-white">
        {{ locale === 'en' ? 'Bank Connection' : 'Σύνδεση Τράπεζας' }}
      </h2>
      <p class="text-sm text-gray-400 mt-0.5">
        {{ locale === 'en'
          ? 'Connect your Greek bank account to auto-detect paid bills'
          : 'Σύνδεσε τον τραπεζικό σου λογαριασμό για αυτόματη αναγνώριση πληρωμών' }}
      </p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="space-y-3">
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl h-32 animate-pulse" />
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl h-16 animate-pulse" />
    </div>

    <div v-else class="space-y-4">

      <!-- Callback success banner -->
      <div v-if="isCallback"
        class="flex items-center gap-3 bg-blue-900/30 border border-blue-700/50 rounded-2xl px-5 py-4">
        <span class="text-2xl">🏦</span>
        <div>
          <p class="text-sm font-semibold text-blue-200">
            {{ locale === 'en' ? 'Authorization completed!' : 'Εξουσιοδότηση ολοκληρώθηκε!' }}
          </p>
          <p class="text-xs text-blue-400 mt-0.5">
            {{ locale === 'en'
              ? 'Your bank is being linked. This may take a moment.'
              : 'Η τράπεζά σου συνδέεται. Μπορεί να χρειαστεί λίγη ώρα.' }}
          </p>
        </div>
      </div>

      <!-- Connected card -->
      <div v-if="status?.connected"
        class="bg-[#111119] border border-emerald-800/50 rounded-2xl p-5">
        <div class="flex items-start justify-between gap-4">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl bg-emerald-900/40 border border-emerald-700/50 flex items-center justify-center text-xl">
              🏦
            </div>
            <div>
              <p class="text-sm font-semibold text-gray-50">{{ status.institution_name }}</p>
              <div class="flex items-center gap-1.5 mt-0.5">
                <span class="w-2 h-2 rounded-full bg-emerald-400" />
                <span class="text-xs text-emerald-400 font-medium">{{ statusLabel }}</span>
              </div>
              <p class="text-xs text-gray-500 mt-0.5">
                {{ status.accounts }}
                {{ locale === 'en' ? 'account(s) linked' : 'λογαριασμός(οί) συνδεδεμένοι' }}
              </p>
            </div>
          </div>
          <button
            @click="disconnect"
            :disabled="disconnecting"
            class="text-xs text-red-400 hover:text-red-300 border border-red-900/50 hover:border-red-700/60 px-3 py-1.5 rounded-lg transition-colors disabled:opacity-50 shrink-0"
          >
            {{ disconnecting
              ? (locale === 'en' ? 'Disconnecting…' : 'Αποσύνδεση…')
              : (locale === 'en' ? 'Disconnect' : 'Αποσύνδεση') }}
          </button>
        </div>

        <!-- Sync section -->
        <div class="mt-4 pt-4 border-t border-white/[0.06]">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-sm font-medium text-gray-200">
                {{ locale === 'en' ? 'Sync transactions' : 'Συγχρονισμός συναλλαγών' }}
              </p>
              <p class="text-xs text-gray-500 mt-0.5">
                {{ locale === 'en'
                  ? 'Fetch the last 90 days and auto-mark matching bills as paid'
                  : 'Φέρνει τις τελευταίες 90 μέρες και σημειώνει αυτόματα τους αντίστοιχους λογαριασμούς ως πληρωμένους' }}
              </p>
            </div>
            <button
              @click="sync"
              :disabled="syncing"
              class="shrink-0 flex items-center gap-1.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-55 text-white text-sm font-medium px-4 py-2 rounded-xl transition-colors"
            >
              <svg v-if="syncing" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
              <span v-else>↻</span>
              {{ syncing
                ? (locale === 'en' ? 'Syncing…' : 'Συγχρονισμός…')
                : (locale === 'en' ? 'Sync now' : 'Συγχρονισμός') }}
            </button>
          </div>

          <!-- Sync result -->
          <div v-if="syncResult" class="mt-3 bg-white/[0.04]/60 rounded-xl px-4 py-3 flex items-center gap-4 text-sm">
            <div class="text-center">
              <p class="text-lg font-bold text-gray-100">{{ syncResult.transactions_found }}</p>
              <p class="text-xs text-gray-500">{{ locale === 'en' ? 'transactions' : 'συναλλαγές' }}</p>
            </div>
            <div class="w-px h-8 bg-white/[0.06]" />
            <div class="text-center">
              <p class="text-lg font-bold" :class="syncResult.bills_matched > 0 ? 'text-emerald-400' : 'text-gray-400'">
                {{ syncResult.bills_matched }}
              </p>
              <p class="text-xs text-gray-500">{{ locale === 'en' ? 'bills matched' : 'λ/σμοί' }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Not connected — status pending -->
      <div v-else-if="status && !status.connected && status.status"
        class="bg-[#111119] border border-amber-800/40 rounded-2xl p-5">
        <div class="flex items-center gap-3">
          <span class="text-2xl">⏳</span>
          <div>
            <p class="text-sm font-semibold text-amber-300">{{ statusLabel }}</p>
            <p class="text-xs text-gray-400 mt-0.5">{{ status.institution_name }}</p>
            <p class="text-xs text-gray-500 mt-1">
              {{ locale === 'en'
                ? 'Complete the authorization in your bank\'s page, then come back here.'
                : 'Ολοκλήρωσε την εξουσιοδότηση στη σελίδα της τράπεζάς σου και επέστρεψε εδώ.' }}
            </p>
          </div>
        </div>
        <div class="flex gap-2 mt-4">
          <button @click="fetchStatus"
            class="flex-1 text-sm text-gray-300 border border-white/[0.08] hover:bg-white/[0.04] rounded-xl py-2 transition-colors">
            {{ locale === 'en' ? 'Refresh status' : 'Ανανέωση κατάστασης' }}
          </button>
          <button @click="disconnect" :disabled="disconnecting"
            class="text-sm text-red-400 border border-red-900/50 hover:border-red-700 px-4 py-2 rounded-xl transition-colors disabled:opacity-50">
            {{ locale === 'en' ? 'Cancel' : 'Ακύρωση' }}
          </button>
        </div>
      </div>

      <!-- No connection — pick a bank -->
      <div v-else class="bg-[#111119] border border-white/[0.06] rounded-2xl p-5">
        <div class="text-center py-4">
          <div class="text-4xl mb-3">🏦</div>
          <p class="text-sm font-semibold text-gray-200 mb-1">
            {{ locale === 'en' ? 'No bank connected' : 'Δεν υπάρχει συνδεδεμένη τράπεζα' }}
          </p>
          <p class="text-xs text-gray-400 mb-5 max-w-xs mx-auto">
            {{ locale === 'en'
              ? 'Connect your Greek bank to automatically sync transactions and mark bills as paid.'
              : 'Σύνδεσε την ελληνική τράπεζά σου για να συγχρονίζονται αυτόματα οι συναλλαγές.' }}
          </p>
          <button
            @click="fetchInstitutions"
            :disabled="loadingInstitutions"
            class="inline-flex items-center gap-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-55 text-white text-sm font-medium px-5 py-2.5 rounded-xl transition-colors"
          >
            <svg v-if="loadingInstitutions" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            {{ loadingInstitutions
              ? (locale === 'en' ? 'Loading…' : 'Φόρτωση…')
              : (locale === 'en' ? 'Choose your bank' : 'Επίλεξε τράπεζα') }}
          </button>
        </div>
      </div>

      <!-- Bank picker -->
      <div v-if="showInstitutions && !status?.connected" class="bg-[#111119] border border-white/[0.06] rounded-2xl p-4">
        <p class="text-sm font-semibold text-gray-200 mb-3">
          {{ locale === 'en' ? 'Greek banks' : 'Ελληνικές τράπεζες' }}
        </p>
        <input
          v-model="searchQuery"
          type="search"
          :placeholder="locale === 'en' ? 'Search…' : 'Αναζήτηση…'"
          class="w-full px-3.5 py-2 rounded-xl border border-white/[0.08] bg-white/[0.04] text-sm text-gray-200 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 mb-3"
        />
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-72 overflow-y-auto">
          <button
            v-for="inst in filteredInstitutions"
            :key="inst.id"
            @click="connect(inst)"
            :disabled="connecting !== ''"
            class="flex items-center gap-3 p-3 rounded-xl border border-white/[0.08] hover:border-blue-500 hover:bg-blue-900/10 text-left transition-colors disabled:opacity-50"
          >
            <img v-if="inst.logo" :src="inst.logo" :alt="inst.name" class="w-8 h-8 rounded-lg object-contain bg-white p-0.5 shrink-0" />
            <div v-else class="w-8 h-8 rounded-lg bg-blue-900/40 flex items-center justify-center text-blue-400 text-sm font-bold shrink-0">
              {{ inst.name.charAt(0) }}
            </div>
            <div class="min-w-0">
              <p class="text-sm font-medium text-gray-100 truncate">{{ inst.name }}</p>
              <p class="text-xs text-gray-500">{{ inst.bic }}</p>
            </div>
            <div v-if="connecting === inst.id" class="ml-auto shrink-0">
              <svg class="w-4 h-4 animate-spin text-blue-400" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
              </svg>
            </div>
          </button>
          <p v-if="filteredInstitutions.length === 0" class="col-span-2 text-center text-sm text-gray-500 py-4">
            {{ locale === 'en' ? 'No banks found' : 'Δεν βρέθηκαν τράπεζες' }}
          </p>
        </div>
      </div>

      <!-- Info box -->
      <div class="bg-[#111119]/50 border border-white/[0.06] rounded-2xl p-4">
        <p class="text-xs font-semibold text-gray-400 mb-2 uppercase tracking-wider">
          {{ locale === 'en' ? 'How it works' : 'Πώς λειτουργεί' }}
        </p>
        <ul class="space-y-1.5 text-xs text-gray-500">
          <li class="flex items-start gap-2">
            <span class="text-blue-400 shrink-0 mt-0.5">1.</span>
            {{ locale === 'en'
              ? 'You choose your bank and complete a one-time authorization.'
              : 'Επιλέγεις την τράπεζά σου και κάνεις μια εφάπαξ εξουσιοδότηση.' }}
          </li>
          <li class="flex items-start gap-2">
            <span class="text-blue-400 shrink-0 mt-0.5">2.</span>
            {{ locale === 'en'
              ? 'We get read-only access to your transactions (via GoCardless Open Banking / PSD2).'
              : 'Αποκτούμε read-only πρόσβαση στις συναλλαγές σου (μέσω GoCardless / PSD2).' }}
          </li>
          <li class="flex items-start gap-2">
            <span class="text-blue-400 shrink-0 mt-0.5">3.</span>
            {{ locale === 'en'
              ? '"Sync now" matches transactions to your pending bills and marks them as paid.'
              : 'Ο "Συγχρονισμός" αντιστοιχεί τις συναλλαγές με εκκρεμείς λογαριασμούς και τους σημειώνει ως πληρωμένους.' }}
          </li>
        </ul>
      </div>

    </div>
  </div>
</template>
