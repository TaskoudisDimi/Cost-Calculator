<script setup lang="ts">
import { ref, computed } from 'vue'
import { useLocale } from '@/composables/useLocale'
import { useBillsStore } from '@/stores/bills'
import type { ScanResult } from '@/types'

const emit = defineEmits<{ close: [] }>()
const { locale } = useLocale()
const billsStore = useBillsStore()

type EmailItem = { subject: string; from: string; body: string; date: string }
type ScanRow = { email: EmailItem; result: ScanResult | null; selected: boolean; providerMatchId: string }

const step = ref<'connect' | 'scanning' | 'review' | 'importing' | 'done'>('connect')
const error = ref('')
const rows = ref<ScanRow[]>([])
const importCount = ref(0)

const selectedRows = computed(() => rows.value.filter(r => r.selected && r.result?.amount))
const hasResults = computed(() => rows.value.some(r => r.result?.amount))

// ── Gmail OAuth ─────────────────────────────────────────────────────────────

function connectGmail() {
  error.value = ''
  const clientId = import.meta.env.VITE_GOOGLE_OAUTH_CLIENT_ID as string
  if (!clientId) {
    error.value = locale.value === 'el' ? 'Λείπει το Google Client ID.' : 'Missing Google Client ID.'
    return
  }
  // @ts-expect-error google loaded via script tag in index.html
  const tokenClient = window.google.accounts.oauth2.initTokenClient({
    client_id: clientId,
    scope: 'https://www.googleapis.com/auth/gmail.readonly',
    callback: (tokenResp: { access_token?: string; error?: string }) => {
      if (tokenResp.error || !tokenResp.access_token) {
        error.value = locale.value === 'el' ? 'Αποτυχία σύνδεσης Gmail.' : 'Failed to connect Gmail.'
        return
      }
      scanGmail(tokenResp.access_token)
    },
  })
  tokenClient.requestAccessToken()
}

// ── Gmail scan ───────────────────────────────────────────────────────────────

async function scanGmail(token: string) {
  step.value = 'scanning'
  error.value = ''

  const keywords = 'λογαριασμός OR οφειλή OR πληρωμή OR invoice OR bill OR RF OR ΔΕΗ OR COSMOTE OR VODAFONE OR ΕΥΔΑΠ OR ΔΕΔΔΗΕ'
  const query = encodeURIComponent(`${keywords} newer_than:60d`)

  try {
    // 1. List matching message IDs
    const listRes = await fetch(
      `https://gmail.googleapis.com/gmail/v1/users/me/messages?q=${query}&maxResults=25`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    if (!listRes.ok) throw new Error('Gmail API error')
    const listData = await listRes.json()
    const messages: { id: string }[] = listData.messages ?? []

    if (messages.length === 0) {
      rows.value = []
      step.value = 'review'
      return
    }

    // 2. Fetch each message and extract text
    const emails: EmailItem[] = []
    for (const { id } of messages.slice(0, 20)) {
      try {
        const msgRes = await fetch(
          `https://gmail.googleapis.com/gmail/v1/users/me/messages/${id}?format=full`,
          { headers: { Authorization: `Bearer ${token}` } },
        )
        if (!msgRes.ok) continue
        const msg = await msgRes.json()
        const email = extractEmailText(msg)
        if (email) emails.push(email)
      } catch {
        // skip individual message failures
      }
    }

    if (emails.length === 0) {
      rows.value = []
      step.value = 'review'
      return
    }

    // 3. Send batch to backend for Claude parsing
    const scanResults = await billsStore.scanEmails(emails)
    rows.value = emails.map((e, i) => ({
      email: e,
      result: scanResults[i] ?? null,
      selected: !!(scanResults[i]?.amount),
      providerMatchId: '',
    }))

    step.value = 'review'
  } catch (e: unknown) {
    error.value = locale.value === 'el'
      ? 'Αποτυχία ανάλυσης Gmail: ' + (e instanceof Error ? e.message : String(e))
      : 'Gmail scan failed: ' + (e instanceof Error ? e.message : String(e))
    step.value = 'connect'
  }
}

function extractEmailText(msg: Record<string, unknown>): EmailItem | null {
  const headers: { name: string; value: string }[] =
    (msg.payload as Record<string, unknown>)?.headers as { name: string; value: string }[] ?? []
  const subject = headers.find(h => h.name === 'Subject')?.value ?? ''
  const from = headers.find(h => h.name === 'From')?.value ?? ''
  const date = headers.find(h => h.name === 'Date')?.value ?? ''
  const body = extractBody(msg.payload as Record<string, unknown>)
  if (!body && !subject) return null
  return { subject, from, body: body.slice(0, 3000), date }
}

function extractBody(payload: Record<string, unknown>): string {
  if (!payload) return ''
  const mimeType = (payload.mimeType as string) ?? ''

  if (mimeType === 'text/plain' || mimeType === 'text/html') {
    const data = (payload.body as Record<string, unknown>)?.data as string
    if (data) {
      try {
        // Gmail uses base64url (no padding) — fix before decoding
        const b64 = data.replace(/-/g, '+').replace(/_/g, '/')
        const padded = b64 + '='.repeat((4 - (b64.length % 4)) % 4)
        const decoded = atob(padded)
        if (mimeType === 'text/html') {
          // Strip tags, collapse whitespace
          return decoded.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
        }
        return decoded
      } catch {
        return ''
      }
    }
  }

  // Prefer text/plain parts, fall back to text/html
  const parts = (payload.parts as Record<string, unknown>[]) ?? []
  const plainPart = parts.find(p => (p.mimeType as string) === 'text/plain')
  if (plainPart) {
    const result = extractBody(plainPart)
    if (result) return result
  }
  for (const part of parts) {
    const text = extractBody(part)
    if (text) return text
  }
  return ''
}

// ── Import confirmed bills ───────────────────────────────────────────────────

async function importSelected() {
  if (selectedRows.value.length === 0) return
  step.value = 'importing'

  const userProviders = billsStore.userProviders

  let count = 0
  for (const row of selectedRows.value) {
    const r = row.result
    if (!r?.amount) continue

    // Try to match provider name to an existing userProvider
    const providerName = (r.provider_name ?? '').toLowerCase()
    const matched = userProviders.find(up =>
      providerName.includes(up.provider?.name?.toLowerCase() ?? '') ||
      (up.provider?.name ?? '').toLowerCase().includes(providerName.split(/\s/)[0] ?? '')
    )
    if (!matched) continue

    try {
      await billsStore.createBill({
        user_provider_id: matched.id,
        amount: r.amount!,
        due_date: r.due_date ? new Date(r.due_date).toISOString() : new Date().toISOString(),
        issued_date: r.issued_date ? new Date(r.issued_date).toISOString() : undefined,
        notes: r.notes ?? '',
        payment_code: r.payment_code ?? '',
        recurring: false,
      })
      count++
    } catch {
      // skip failed imports
    }
  }

  importCount.value = count
  step.value = 'done'
}
</script>

<template>
  <!-- Backdrop -->
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="emit('close')">
    <div class="absolute inset-0 bg-black/70 backdrop-blur-sm" @click="emit('close')" />

    <div class="relative z-10 w-full max-w-lg bg-[#13131e] rounded-2xl border border-white/[0.08] shadow-2xl overflow-hidden">

      <!-- Header -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-white/[0.06]">
        <div class="flex items-center gap-3">
          <!-- Gmail G icon -->
          <div class="w-8 h-8 rounded-lg bg-white/[0.06] flex items-center justify-center">
            <svg viewBox="0 0 24 24" class="w-4 h-4" fill="none">
              <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09Z" fill="#4285F4"/>
              <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23Z" fill="#34A853"/>
              <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84Z" fill="#FBBC05"/>
              <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53Z" fill="#EA4335"/>
            </svg>
          </div>
          <div>
            <h3 class="text-sm font-semibold text-gray-50">
              {{ locale === 'el' ? 'Σάρωση Gmail' : 'Scan Gmail' }}
            </h3>
            <p class="text-xs text-gray-500">
              {{ locale === 'el' ? 'Εντοπισμός λογαριασμών από email' : 'Detect bills from email' }}
            </p>
          </div>
        </div>
        <button @click="emit('close')" class="w-8 h-8 flex items-center justify-center rounded-lg text-gray-500 hover:text-gray-300 hover:bg-white/[0.06] transition-colors">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Body -->
      <div class="p-5">

        <!-- Step: connect -->
        <div v-if="step === 'connect'" class="text-center py-4">
          <div class="w-14 h-14 rounded-2xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center mx-auto mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-7 h-7 text-blue-400">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21.75 6.75v10.5a2.25 2.25 0 0 1-2.25 2.25h-15a2.25 2.25 0 0 1-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0 0 19.5 4.5h-15a2.25 2.25 0 0 0-2.25 2.25m19.5 0v.243a2.25 2.25 0 0 1-1.07 1.916l-7.5 4.615a2.25 2.25 0 0 1-2.36 0L3.32 8.91a2.25 2.25 0 0 1-1.07-1.916V6.75" />
            </svg>
          </div>
          <p class="text-sm text-gray-300 mb-1">
            {{ locale === 'el' ? 'Σύνδεσε το Gmail σου για αυτόματο εντοπισμό λογαριασμών.' : 'Connect your Gmail to auto-detect bills.' }}
          </p>
          <p class="text-xs text-gray-500 mb-6">
            {{ locale === 'el' ? 'Διαβάζουμε μόνο email που μοιάζουν με λογαριασμούς (60 μέρες).' : 'We only read emails that look like bills (last 60 days).' }}
          </p>
          <p v-if="error" class="text-xs text-red-400 mb-3">{{ error }}</p>
          <button
            @click="connectGmail"
            class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-sm font-semibold transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244" />
            </svg>
            {{ locale === 'el' ? 'Σύνδεση Gmail' : 'Connect Gmail' }}
          </button>
        </div>

        <!-- Step: scanning -->
        <div v-else-if="step === 'scanning'" class="text-center py-8">
          <div class="w-10 h-10 border-2 border-blue-500/30 border-t-blue-500 rounded-full animate-spin mx-auto mb-4" />
          <p class="text-sm text-gray-300">
            {{ locale === 'el' ? 'Σάρωση email...' : 'Scanning emails...' }}
          </p>
          <p class="text-xs text-gray-500 mt-1">
            {{ locale === 'el' ? 'Χρησιμοποιούμε AI για εντοπισμό λογαριασμών' : 'Using AI to detect bills' }}
          </p>
        </div>

        <!-- Step: review -->
        <div v-else-if="step === 'review'">
          <div v-if="!hasResults" class="text-center py-6">
            <p class="text-sm text-gray-400">
              {{ locale === 'el' ? 'Δεν βρέθηκαν λογαριασμοί στα τελευταία 60 μέρες.' : 'No bills found in the last 60 days.' }}
            </p>
          </div>
          <div v-else>
            <p class="text-xs text-gray-500 mb-3">
              {{ locale === 'el'
                ? `Βρέθηκαν ${rows.filter(r => r.result?.amount).length} πιθανοί λογαριασμοί. Επίλεξε ποιους να εισαγάγεις.`
                : `Found ${rows.filter(r => r.result?.amount).length} possible bills. Select which to import.` }}
            </p>
            <div class="space-y-2 max-h-72 overflow-y-auto pr-1">
              <label
                v-for="(row, i) in rows.filter(r => r.result?.amount)"
                :key="i"
                class="flex items-start gap-3 p-3 rounded-xl border cursor-pointer transition-colors"
                :class="row.selected
                  ? 'border-blue-500/40 bg-blue-500/8'
                  : 'border-white/[0.06] bg-white/[0.02] hover:border-white/[0.1]'"
              >
                <input v-model="row.selected" type="checkbox" class="mt-0.5 accent-blue-500 shrink-0" />
                <div class="flex-1 min-w-0">
                  <div class="flex items-center justify-between gap-2">
                    <span class="text-sm font-medium text-gray-200 truncate">{{ row.result?.provider_name || row.email.from }}</span>
                    <span class="text-sm font-semibold text-emerald-400 shrink-0">{{ row.result?.amount?.toFixed(2) }} €</span>
                  </div>
                  <div class="flex items-center gap-3 mt-0.5">
                    <span v-if="row.result?.due_date" class="text-xs text-gray-500">
                      {{ locale === 'el' ? 'Λήξη:' : 'Due:' }} {{ row.result.due_date }}
                    </span>
                    <span v-if="row.result?.payment_code" class="text-xs font-mono text-violet-400 truncate">{{ row.result.payment_code }}</span>
                  </div>
                  <p class="text-xs text-gray-600 truncate mt-0.5">{{ row.email.subject }}</p>
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- Step: importing -->
        <div v-else-if="step === 'importing'" class="text-center py-8">
          <div class="w-10 h-10 border-2 border-emerald-500/30 border-t-emerald-500 rounded-full animate-spin mx-auto mb-4" />
          <p class="text-sm text-gray-300">{{ locale === 'el' ? 'Εισαγωγή λογαριασμών...' : 'Importing bills...' }}</p>
        </div>

        <!-- Step: done -->
        <div v-else-if="step === 'done'" class="text-center py-6">
          <div class="w-14 h-14 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center mx-auto mb-4">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-7 h-7 text-emerald-400">
              <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
            </svg>
          </div>
          <p class="text-sm font-semibold text-gray-200 mb-1">
            {{ locale === 'el' ? `${importCount} λογαριασμοί εισήχθησαν!` : `${importCount} bills imported!` }}
          </p>
          <p v-if="importCount < selectedRows.length" class="text-xs text-amber-400">
            {{ locale === 'el'
              ? `${selectedRows.length - importCount} δεν εισήχθησαν (ο πάροχος δεν βρέθηκε στη λίστα σου).`
              : `${selectedRows.length - importCount} skipped — provider not found in your list.` }}
          </p>
        </div>

      </div>

      <!-- Footer -->
      <div v-if="step === 'review' && hasResults" class="flex items-center justify-between px-5 py-4 border-t border-white/[0.06]">
        <span class="text-xs text-gray-500">
          {{ selectedRows.length }} {{ locale === 'el' ? 'επιλεγμένοι' : 'selected' }}
        </span>
        <button
          @click="importSelected"
          :disabled="selectedRows.length === 0"
          class="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-semibold transition-colors"
        >
          {{ locale === 'el' ? 'Εισαγωγή' : 'Import' }}
        </button>
      </div>
      <div v-else-if="step === 'done'" class="px-5 py-4 border-t border-white/[0.06]">
        <button @click="emit('close')" class="w-full px-4 py-2.5 rounded-xl bg-white/[0.06] hover:bg-white/[0.1] text-gray-200 text-sm font-semibold transition-colors">
          {{ locale === 'el' ? 'Κλείσιμο' : 'Close' }}
        </button>
      </div>

    </div>
  </div>
</template>
