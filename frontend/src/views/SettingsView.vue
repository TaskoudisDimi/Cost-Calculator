<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { updateProfile, sendPasswordResetEmail, deleteUser, reauthenticateWithCredential, EmailAuthProvider } from 'firebase/auth'
import { auth } from '@/firebase'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useBillsStore } from '@/stores/bills'
import { useMembersStore } from '@/stores/members'
import { useToast } from '@/composables/useToast'
import { useLocale } from '@/composables/useLocale'
import { detectProviderFromEmail } from '@/utils/emailPatterns'

const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const billsStore = useBillsStore()
const membersStore = useMembersStore()
const { toast } = useToast()
const { t, locale, setLocale } = useLocale()

// Members
const memberColors = ['#6366f1', '#ec4899', '#f59e0b', '#10b981', '#3b82f6', '#8b5cf6', '#ef4444', '#14b8a6']
const newMemberName = ref('')
const newMemberColor = ref('#6366f1')
const addingMember = ref(false)
const editingMemberID = ref<string | null>(null)
const editingMemberName = ref('')
const editingMemberColor = ref('')

async function addMember() {
  if (!newMemberName.value.trim()) return
  addingMember.value = true
  try {
    await membersStore.createMember(newMemberName.value.trim(), newMemberColor.value)
    newMemberName.value = ''
    newMemberColor.value = '#6366f1'
    toast(locale.value === 'en' ? 'Member added' : 'Το μέλος προστέθηκε', 'success')
  } catch {
    toast(locale.value === 'en' ? 'Failed to add member' : 'Αποτυχία προσθήκης μέλους', 'error')
  } finally {
    addingMember.value = false
  }
}

function startEditMember(id: string, name: string, color: string) {
  editingMemberID.value = id
  editingMemberName.value = name
  editingMemberColor.value = color
}

async function saveEditMember() {
  if (!editingMemberID.value || !editingMemberName.value.trim()) return
  try {
    await membersStore.updateMember(editingMemberID.value, {
      name: editingMemberName.value.trim(),
      color: editingMemberColor.value,
    })
    editingMemberID.value = null
    toast(locale.value === 'en' ? 'Member updated' : 'Το μέλος ενημερώθηκε', 'success')
  } catch {
    toast(locale.value === 'en' ? 'Update failed' : 'Αποτυχία ενημέρωσης', 'error')
  }
}

async function deleteMember(id: string) {
  try {
    await membersStore.deleteMember(id)
    toast(locale.value === 'en' ? 'Member removed' : 'Το μέλος διαγράφηκε', 'success')
  } catch {
    toast(locale.value === 'en' ? 'Delete failed' : 'Αποτυχία διαγραφής', 'error')
  }
}

// Profile
const displayName = ref('')
const savingProfile = ref(false)
const resetEmailSent = ref(false)

// Notifications
const savingNotif = ref(false)

// Delete account
const deletePassword = ref('')
const showDeleteModal = ref(false)
const deletingAccount = ref(false)

const currencyOptions = [
  { value: 'EUR', label: '€ Euro' },
  { value: 'USD', label: '$ Dollar' },
  { value: 'GBP', label: '£ Pound' },
]

const notifDaysOptions = [1, 2, 3, 5, 7]

const userInitials = computed(() => {
  const name = authStore.user?.name || authStore.user?.email || '?'
  return name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2)
})

onMounted(async () => {
  displayName.value = authStore.user?.name || ''
  await Promise.all([
    settingsStore.fetchSettings(),
    billsStore.fetchBills(),
    membersStore.fetchMembers(),
  ])
})

async function saveProfile() {
  if (!auth.currentUser) return
  savingProfile.value = true
  try {
    await updateProfile(auth.currentUser, { displayName: displayName.value.trim() })
    toast(locale.value === 'en' ? 'Profile updated' : 'Προφίλ ενημερώθηκε', 'success')
  } catch {
    toast(locale.value === 'en' ? 'Update failed' : 'Αποτυχία ενημέρωσης', 'error')
  } finally {
    savingProfile.value = false
  }
}

async function sendPasswordReset() {
  const email = authStore.user?.email
  if (!email) return
  try {
    await sendPasswordResetEmail(auth, email)
    resetEmailSent.value = true
    toast(locale.value === 'en' ? 'Reset email sent' : 'Email επαναφοράς εστάλη', 'success')
  } catch {
    toast(locale.value === 'en' ? 'Failed to send email' : 'Αποτυχία αποστολής email', 'error')
  }
}

async function saveNotifSettings() {
  savingNotif.value = true
  try {
    await settingsStore.saveSettings()
    toast(locale.value === 'en' ? 'Settings saved' : 'Ρυθμίσεις αποθηκεύτηκαν', 'success')
  } catch {
    toast(locale.value === 'en' ? 'Save failed' : 'Αποτυχία αποθήκευσης', 'error')
  } finally {
    savingNotif.value = false
  }
}

function exportBillsCSV() {
  const bills = Array.isArray(billsStore.bills) ? billsStore.bills : []
  if (bills.length === 0) {
    toast(locale.value === 'en' ? 'No bills to export' : 'Δεν υπάρχουν λογαριασμοί για εξαγωγή', 'warning')
    return
  }
  const headers = ['Πάροχος', 'Ποσό', 'Κατάσταση', 'Ημ. Λήξης', 'Ημ. Έκδοσης', 'Κωδικός RF', 'Σημειώσεις']
  const rows = bills.map(b => [
    b.user_provider?.nickname || b.user_provider?.provider?.name || '',
    b.amount.toFixed(2),
    b.status,
    b.due_date ? b.due_date.split('T')[0] : '',
    b.issued_date ? b.issued_date.split('T')[0] : '',
    b.payment_code || '',
    b.notes || '',
  ])
  const csv = [headers, ...rows].map(r => r.map(v => `"${String(v).replace(/"/g, '""')}"`).join(',')).join('\n')
  const blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `λογαριασμοί-${new Date().toISOString().split('T')[0]}.csv`
  a.click()
  URL.revokeObjectURL(url)
  toast(`${locale.value === 'en' ? 'Exported' : 'Εξήχθησαν'} ${bills.length} ${locale.value === 'en' ? 'bills' : 'λογαριασμοί'}`, 'success')
}

// ─── Calendar (.ics) Export ───────────────────────────────────────────────────

function exportCalendarICS() {
  const bills = (Array.isArray(billsStore.bills) ? billsStore.bills : [])
    .filter(b => b.status !== 'paid')
  if (bills.length === 0) {
    toast(locale.value === 'en' ? 'No upcoming bills to export' : 'Δεν υπάρχουν εκκρεμείς λογαριασμοί', 'warning')
    return
  }

  const stamp = new Date().toISOString().replace(/[-:.]/g, '').slice(0, 15) + 'Z'
  const escape = (s: string) => s.replace(/[\\,;]/g, c => '\\' + c).replace(/\n/g, '\\n')

  const events = bills.map(b => {
    const providerName = b.user_provider?.nickname || b.user_provider?.provider?.name || 'Λογαριασμός'
    const dueDate = (b.due_date.split('T')[0] ?? '').replace(/-/g, '')
    // DTEND = day after for all-day events
    const dueDateObj = new Date(b.due_date)
    dueDateObj.setDate(dueDateObj.getDate() + 1)
    const dueDateNext = (dueDateObj.toISOString().split('T')[0] ?? '').replace(/-/g, '')

    const desc = [
      `Ποσό: ${b.amount.toFixed(2)}€`,
      b.payment_code ? `Κωδικός: ${b.payment_code}` : '',
      b.notes ? `Σημειώσεις: ${b.notes}` : '',
      b.status === 'overdue' ? '⚠️ ΛΗΞΙΠΡΟΘΕΣΜΟΣ' : '',
    ].filter(Boolean).join('\\n')

    return [
      'BEGIN:VEVENT',
      `UID:${b.id}@billtracker`,
      `DTSTAMP:${stamp}`,
      `DTSTART;VALUE=DATE:${dueDate}`,
      `DTEND;VALUE=DATE:${dueDateNext}`,
      `SUMMARY:${escape(providerName)} — ${b.amount.toFixed(2)}€`,
      `DESCRIPTION:${escape(desc)}`,
      b.status === 'overdue' ? 'PRIORITY:1' : 'PRIORITY:5',
      'BEGIN:VALARM',
      'TRIGGER:-P1D',
      'ACTION:DISPLAY',
      `DESCRIPTION:${escape('Αύριο λήγει: ' + providerName)}`,
      'END:VALARM',
      'END:VEVENT',
    ].join('\r\n')
  }).join('\r\n')

  const ics = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//BillTracker//BillTracker//EL',
    'CALSCALE:GREGORIAN',
    'METHOD:PUBLISH',
    `X-WR-CALNAME:BillTracker — Λογαριασμοί`,
    'X-WR-TIMEZONE:Europe/Athens',
    events,
    'END:VCALENDAR',
  ].join('\r\n')

  const blob = new Blob([ics], { type: 'text/calendar;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `billtracker-${new Date().toISOString().split('T')[0]}.ics`
  a.click()
  URL.revokeObjectURL(url)
  toast(`${locale.value === 'en' ? 'Exported' : 'Εξήχθησαν'} ${bills.length} ${locale.value === 'en' ? 'events' : 'εγγραφές'}`, 'success')
}

// ─── CSV Import ───────────────────────────────────────────────────────────────

type ImportRow = {
  providerName: string
  amount: number
  status: string
  dueDate: string
  issuedDate: string
  paymentCode: string
  notes: string
  matchedProviderID: string | null
  matchedProviderLabel: string
}

const csvFileInput = ref<HTMLInputElement | null>(null)
const importRows = ref<ImportRow[]>([])
const showImportModal = ref(false)
const importing = ref(false)

function triggerImport() {
  csvFileInput.value?.click()
}

function parseCSV(text: string): string[][] {
  // Remove BOM if present
  const clean = text.replace(/^﻿/, '')
  return clean.split('\n').filter(l => l.trim()).map(line => {
    const cols: string[] = []
    let cur = ''
    let inQ = false
    for (let i = 0; i < line.length; i++) {
      const ch = line[i]
      if (ch === '"') {
        if (inQ && line[i + 1] === '"') { cur += '"'; i++ }
        else inQ = !inQ
      } else if (ch === ',' && !inQ) {
        cols.push(cur); cur = ''
      } else {
        cur += ch
      }
    }
    cols.push(cur)
    return cols
  })
}

function onFileSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (ev) => {
    try {
      const text = ev.target?.result as string
      const rows = parseCSV(text)
      // Skip header row (first row)
      const dataRows = rows.slice(1).filter(r => r.length >= 4)
      if (dataRows.length === 0) {
        toast(t('settings.import_empty'), 'warning')
        return
      }

      const providers = Array.isArray(billsStore.userProviders) ? billsStore.userProviders : []

      importRows.value = dataRows.map(cols => {
        const providerName = cols[0]?.trim() ?? ''
        const amountStr = cols[1]?.trim().replace(',', '.') ?? ''
        const status = cols[2]?.trim() ?? 'pending'
        const dueDate = cols[3]?.trim() ?? ''
        const issuedDate = cols[4]?.trim() ?? ''
        const paymentCode = cols[5]?.trim() ?? ''
        const notes = cols[6]?.trim() ?? ''

        const amount = parseFloat(amountStr)

        // Match provider: try nickname first, then provider name (case-insensitive)
        const needle = providerName.toLowerCase()
        const match = providers.find(p =>
          (p.nickname || '').toLowerCase() === needle ||
          (p.provider?.name || '').toLowerCase() === needle
        )

        return {
          providerName,
          amount: isNaN(amount) ? 0 : amount,
          status,
          dueDate,
          issuedDate,
          paymentCode,
          notes,
          matchedProviderID: match?.id ?? null,
          matchedProviderLabel: match ? (match.nickname || match.provider?.name || '') : '',
        }
      })

      showImportModal.value = true
    } catch {
      toast(t('settings.import_error'), 'error')
    } finally {
      // Reset file input so the same file can be re-selected
      if (csvFileInput.value) csvFileInput.value.value = ''
    }
  }
  reader.readAsText(file, 'utf-8')
}

const matchedRows = computed(() => importRows.value.filter(r => r.matchedProviderID && r.amount > 0 && r.dueDate))
const skippedRows = computed(() => importRows.value.filter(r => !r.matchedProviderID || r.amount <= 0 || !r.dueDate))

async function confirmImport() {
  if (matchedRows.value.length === 0) return
  importing.value = true
  let created = 0
  for (const row of matchedRows.value) {
    try {
      await billsStore.createBill({
        user_provider_id: row.matchedProviderID!,
        amount: row.amount,
        due_date: new Date(row.dueDate).toISOString(),
        issued_date: row.issuedDate ? new Date(row.issuedDate).toISOString() : undefined,
        notes: row.notes || undefined,
        payment_code: row.paymentCode || undefined,
      })
      created++
    } catch { /* skip individual failures */ }
  }
  importing.value = false
  showImportModal.value = false
  importRows.value = []
  await billsStore.fetchBills()
  const msg = t('settings.import_done').replace('{n}', String(created))
  toast(msg, 'success')
}

// ─── Gmail auto-import ───────────────────────────────────────────────────────


const gmailClientID = import.meta.env.VITE_GOOGLE_OAUTH_CLIENT_ID || ''
const gmailScanning = ref(false)
const gmailFoundBills = ref<Array<{ result: { provider_name: string | null; amount: number | null; due_date: string | null; issued_date: string | null; notes: string | null; payment_code: string | null }; subject: string; userProviderID: string | null }>>([])
const gmailImporting = ref(false)
const gmailImported = ref<number | null>(null)
const gmailError = ref<string | null>(null)
const showGmailModal = ref(false)
const gmailDays = ref(30)

function loadGIS(): Promise<void> {
  return new Promise((resolve, reject) => {
    if ((window as unknown as Record<string, unknown>).google) { resolve(); return }
    const script = document.createElement('script')
    script.src = 'https://accounts.google.com/gsi/client'
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('GIS load failed'))
    document.head.appendChild(script)
  })
}

function stripHtml(html: string): string {
  return html
    .replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
    .replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/?(p|div|tr|td|th|li)[^>]*>/gi, '\n')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<').replace(/&gt;/g, '>')
    .replace(/&euro;/g, '€').replace(/&#8364;/g, '€')
    .replace(/&#[0-9]+;/g, ' ')
    .replace(/[ \t]+/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

function extractTextFromPart(payload: unknown): { plain: string; html: string } {
  const p = payload as { mimeType?: string; body?: { data?: string }; parts?: unknown[] }
  if (!p) return { plain: '', html: '' }
  const decode = (d: string) => atob(d.replace(/-/g, '+').replace(/_/g, '/'))
  if (p.mimeType === 'text/plain' && p.body?.data) return { plain: decode(p.body.data), html: '' }
  if (p.mimeType === 'text/html' && p.body?.data) return { plain: '', html: decode(p.body.data) }
  if (p.parts) {
    let plain = '', html = ''
    for (const part of p.parts) {
      const r = extractTextFromPart(part)
      if (r.plain) plain = r.plain
      if (r.html) html = r.html
    }
    return { plain, html }
  }
  return { plain: '', html: '' }
}

function extractEmailBody(payload: unknown): string {
  const { plain, html } = extractTextFromPart(payload)
  if (plain && plain.length > 30) return plain
  if (html) return stripHtml(html)
  return plain
}

async function scanGmailInbox(token: string) {
  gmailScanning.value = true
  gmailFoundBills.value = []
  gmailImported.value = null
  gmailError.value = null
  try {
    const after = new Date()
    after.setDate(after.getDate() - gmailDays.value)
    const afterStr = `${after.getFullYear()}/${String(after.getMonth() + 1).padStart(2, '0')}/${String(after.getDate()).padStart(2, '0')}`
    // Expanded query: more Greek/English billing keywords + known sender domains
    const q = [
      'subject:(λογαριασμός OR τιμολόγιο OR invoice OR "e-bill" OR ebill OR bill OR payment OR χρέωση OR οφειλή OR "due date" OR "ημ/νια λήξης")',
      `after:${afterStr}`,
    ].join(' ')

    const listResp = await fetch(
      `https://gmail.googleapis.com/gmail/v1/users/me/messages?q=${encodeURIComponent(q)}&maxResults=30`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    if (!listResp.ok) { gmailError.value = t('settings.gmail_error'); return }
    const listData = await listResp.json()
    const messages: { id: string }[] = listData.messages || []

    const providers = Array.isArray(billsStore.userProviders) ? billsStore.userProviders : []
    const results: typeof gmailFoundBills.value = []

    for (const msg of messages.slice(0, 15)) {
      const msgResp = await fetch(
        `https://gmail.googleapis.com/gmail/v1/users/me/messages/${msg.id}?format=full`,
        { headers: { Authorization: `Bearer ${token}` } },
      )
      if (!msgResp.ok) continue
      const msgData = await msgResp.json()

      const headers: { name: string; value: string }[] = msgData.payload?.headers || []
      const subject = headers.find(h => h.name === 'Subject')?.value || ''
      const from = headers.find(h => h.name === 'From')?.value || ''

      const body = extractEmailBody(msgData.payload)
      // Prepend From + Subject so AI and pattern matching can use them
      const fullText = `From: ${from}\nSubject: ${subject}\n\n${body}`.trim()
      if (fullText.length < 50) continue

      // Local provider detection from From/Subject before hitting the API
      const localProvider = detectProviderFromEmail(from + ' ' + subject)

      try {
        const parsed = await billsStore.parseEmail(fullText)
        if (!parsed.amount || !parsed.due_date) continue

        const providerName = parsed.provider_name || localProvider
        const needle = (providerName || '').toLowerCase()
        const match = needle ? providers.find(p =>
          (p.nickname || '').toLowerCase().includes(needle) ||
          needle.includes((p.nickname || '').toLowerCase()) ||
          (p.provider?.name || '').toLowerCase().includes(needle) ||
          needle.includes((p.provider?.name || '').toLowerCase())
        ) : null

        results.push({
          result: { ...parsed, provider_name: providerName },
          subject,
          userProviderID: match?.id ?? null,
        })
      } catch { /* skip unparseable */ }
    }

    gmailFoundBills.value = results
    if (results.length > 0) {
      showGmailModal.value = true
    } else {
      gmailError.value = t('settings.gmail_none')
    }
  } catch {
    gmailError.value = t('settings.gmail_error')
  } finally {
    gmailScanning.value = false
  }
}

async function connectGmail() {
  if (!gmailClientID) { gmailError.value = t('settings.gmail_setup'); return }
  gmailError.value = null
  try {
    await loadGIS()
    const g = (window as unknown as Record<string, unknown>).google as {
      accounts: { oauth2: { initTokenClient: (cfg: unknown) => { requestAccessToken: () => void } } }
    }
    const tokenClient = g.accounts.oauth2.initTokenClient({
      client_id: gmailClientID,
      scope: 'https://www.googleapis.com/auth/gmail.readonly',
      callback: (resp: { access_token?: string; error?: string }) => {
        if (resp.error || !resp.access_token) { gmailError.value = t('settings.gmail_error'); return }
        scanGmailInbox(resp.access_token)
      },
    })
    tokenClient.requestAccessToken()
  } catch {
    gmailError.value = t('settings.gmail_error')
  }
}

async function importGmailBills() {
  const toImport = gmailFoundBills.value.filter(b => b.userProviderID && b.result.amount && b.result.due_date)
  if (toImport.length === 0) return
  gmailImporting.value = true
  let count = 0
  for (const item of toImport) {
    try {
      await billsStore.createBill({
        user_provider_id: item.userProviderID!,
        amount: item.result.amount!,
        due_date: item.result.due_date!,
        issued_date: item.result.issued_date || undefined,
        notes: item.result.notes || undefined,
        payment_code: item.result.payment_code || undefined,
      })
      count++
    } catch { /* skip */ }
  }
  gmailImporting.value = false
  showGmailModal.value = false
  gmailFoundBills.value = []
  gmailImported.value = count
  await billsStore.fetchBills()
}

async function confirmDeleteAccount() {
  if (!auth.currentUser || !auth.currentUser.email) return
  deletingAccount.value = true
  try {
    const credential = EmailAuthProvider.credential(auth.currentUser.email, deletePassword.value)
    await reauthenticateWithCredential(auth.currentUser, credential)
    await settingsStore.deleteAccount()
    await deleteUser(auth.currentUser)
    authStore.logout()
    router.push({ name: 'login' })
  } catch (e: unknown) {
    const code = (e as { code?: string }).code
    const msg = code === 'auth/wrong-password' || code === 'auth/invalid-credential'
      ? (locale.value === 'en' ? 'Wrong password' : 'Λάθος κωδικός πρόσβασης')
      : (locale.value === 'en' ? 'Account deletion failed' : 'Αποτυχία διαγραφής λογαριασμού')
    toast(msg, 'error')
  } finally {
    deletingAccount.value = false
    showDeleteModal.value = false
    deletePassword.value = ''
  }
}
</script>

<template>
  <div class="max-w-2xl mx-auto space-y-5">

    <div class="flex items-center gap-3 mb-6">
      <h2 class="text-xl md:text-2xl font-bold text-gray-50">{{ t('settings.title') }}</h2>
    </div>

    <!-- Profile section -->
    <section class="bg-[#111119] rounded-2xl border border-white/[0.06] p-5">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">{{ t('settings.profile') }}</h3>

      <div class="flex items-center gap-4 mb-5">
        <div class="w-14 h-14 rounded-2xl bg-blue-600 flex items-center justify-center text-white text-xl font-bold shrink-0">
          {{ userInitials }}
        </div>
        <div>
          <p class="text-base font-semibold text-gray-50">{{ authStore.user?.name || '—' }}</p>
          <p class="text-sm text-gray-400">{{ authStore.user?.email }}</p>
        </div>
      </div>

      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('settings.display_name') }}</label>
          <div class="flex gap-2">
            <input
              v-model="displayName"
              type="text"
              :placeholder="t('settings.name_placeholder')"
              class="flex-1 px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
            />
            <button
              @click="saveProfile"
              :disabled="savingProfile || !displayName.trim()"
              class="px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors disabled:opacity-50"
            >
              {{ savingProfile ? '…' : t('settings.save_profile') }}
            </button>
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('settings.email') }}</label>
          <input
            :value="authStore.user?.email"
            type="email"
            disabled
            class="w-full px-3.5 py-2.5 rounded-xl border border-white/[0.06] text-sm text-gray-600 cursor-not-allowed opacity-60"
          />
        </div>

        <div v-if="!authStore.isGoogleUser" class="pt-1">
          <button
            @click="sendPasswordReset"
            :disabled="resetEmailSent"
            class="flex items-center gap-2 text-sm text-blue-400 hover:text-blue-300 disabled:text-gray-500 transition-colors font-medium"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z" />
            </svg>
            {{ resetEmailSent ? t('settings.password_sent') : t('settings.change_password') }}
          </button>
          <p v-if="resetEmailSent" class="text-xs text-gray-500 mt-1 ml-6">
            {{ t('settings.password_check') }} {{ authStore.user?.email }}
          </p>
        </div>
        <div v-else class="pt-1 flex items-center gap-2 text-xs text-gray-500">
          <svg viewBox="0 0 24 24" class="w-4 h-4 shrink-0">
            <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
            <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
            <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
            <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
          </svg>
          {{ t('settings.google_linked') }}
        </div>
      </div>
    </section>

    <!-- Notifications section -->
    <section class="bg-[#111119] rounded-2xl border border-white/[0.06] p-5">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">{{ t('settings.notifications') }}</h3>

      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-200">{{ t('settings.push_notif') }}</p>
            <p class="text-xs text-gray-500 mt-0.5">{{ t('settings.push_desc') }}</p>
          </div>
          <button
            @click="settingsStore.notifEnabled = !settingsStore.notifEnabled"
            class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
            :class="settingsStore.notifEnabled ? 'bg-blue-600' : 'bg-white/[0.08]'"
          >
            <span
              class="inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform"
              :class="settingsStore.notifEnabled ? 'translate-x-6' : 'translate-x-1'"
            />
          </button>
        </div>

        <div v-if="settingsStore.notifEnabled">
          <label class="block text-sm font-medium text-gray-300 mb-2">
            {{ t('settings.notif_before') }}
          </label>
          <div class="flex gap-2 flex-wrap">
            <button
              v-for="days in notifDaysOptions"
              :key="days"
              @click="settingsStore.notifDays = days"
              class="px-3 py-1.5 rounded-lg text-sm font-medium border transition-colors"
              :class="settingsStore.notifDays === days
                ? 'bg-blue-600 border-blue-600 text-white'
                : 'border-white/[0.10] text-gray-400 hover:border-gray-500 hover:text-gray-300'"
            >
              {{ days === 1 ? `1 ${t('settings.day')}` : `${days} ${t('settings.days')}` }}
            </button>
          </div>
        </div>

        <button
          @click="saveNotifSettings"
          :disabled="savingNotif"
          class="w-full py-2.5 rounded-xl bg-white/[0.06] hover:bg-white/[0.08] text-sm font-medium text-gray-200 transition-colors disabled:opacity-50"
        >
          {{ savingNotif ? t('common.saving') : t('settings.save_settings') }}
        </button>
      </div>
    </section>

    <!-- Currency / Display -->
    <section class="bg-[#111119] rounded-2xl border border-white/[0.06] p-5">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">{{ t('settings.appearance') }}</h3>
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2">{{ t('settings.currency') }}</label>
          <div class="flex gap-2">
            <button
              v-for="opt in currencyOptions"
              :key="opt.value"
              @click="settingsStore.currency = opt.value; settingsStore.saveSettings()"
              class="flex-1 py-2 rounded-xl text-sm font-medium border transition-colors"
              :class="settingsStore.currency === opt.value
                ? 'bg-blue-600 border-blue-600 text-white'
                : 'border-white/[0.10] text-gray-400 hover:border-gray-500 hover:text-gray-200'"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-2">{{ t('settings.language') }}</label>
          <div class="flex gap-2">
            <button
              @click="setLocale('el')"
              class="flex items-center gap-2 flex-1 py-2 rounded-xl text-sm font-medium border transition-colors"
              :class="locale === 'el'
                ? 'bg-blue-600 border-blue-600 text-white'
                : 'border-white/[0.10] text-gray-400 hover:border-gray-500 hover:text-gray-200'"
            >
              <span class="text-base leading-none">🇬🇷</span> Ελληνικά
            </button>
            <button
              @click="setLocale('en')"
              class="flex items-center gap-2 flex-1 py-2 rounded-xl text-sm font-medium border transition-colors justify-center"
              :class="locale === 'en'
                ? 'bg-blue-600 border-blue-600 text-white'
                : 'border-white/[0.10] text-gray-400 hover:border-gray-500 hover:text-gray-200'"
            >
              <span class="text-base leading-none">🇬🇧</span> English
            </button>
          </div>
        </div>
      </div>
    </section>

    <!-- Members section -->
    <section class="bg-[#111119] rounded-2xl border border-white/[0.06] p-5">
      <div class="flex items-start justify-between mb-4">
        <div>
          <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide">{{ t('settings.members') }}</h3>
          <p class="text-xs text-gray-500 mt-0.5">{{ t('settings.members_desc') }}</p>
        </div>
      </div>

      <!-- Existing members list -->
      <div v-if="membersStore.members.length > 0" class="space-y-2 mb-4">
        <div
          v-for="m in membersStore.members"
          :key="m.id"
          class="flex items-center gap-3 p-2.5 rounded-xl border border-white/[0.06] bg-white/[0.03]"
        >
          <template v-if="editingMemberID === m.id">
            <!-- Color swatches in edit mode -->
            <div class="flex gap-1 shrink-0">
              <button
                v-for="c in memberColors"
                :key="c"
                @click="editingMemberColor = c"
                class="w-5 h-5 rounded-full border-2 transition-all"
                :style="{ backgroundColor: c }"
                :class="editingMemberColor === c ? 'border-white scale-110' : 'border-transparent'"
              />
            </div>
            <input
              v-model="editingMemberName"
              type="text"
              class="flex-1 bg-white/[0.06] rounded-lg px-2.5 py-1 text-sm text-gray-100 border border-white/[0.10] focus:outline-none focus:ring-1 focus:ring-blue-500"
              @keyup.enter="saveEditMember"
            />
            <button @click="saveEditMember" class="text-xs text-blue-400 hover:text-blue-300 font-medium px-2">{{ t('common.save') }}</button>
            <button @click="editingMemberID = null" class="text-xs text-gray-500 hover:text-gray-300 px-1">{{ t('common.cancel') }}</button>
          </template>
          <template v-else>
            <div class="w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-bold shrink-0" :style="{ backgroundColor: m.color }">
              {{ m.name[0]?.toUpperCase() }}
            </div>
            <span class="flex-1 text-sm font-medium text-gray-200">{{ m.name }}</span>
            <button @click="startEditMember(m.id, m.name, m.color)" class="text-gray-500 hover:text-gray-300 transition-colors p-1">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
                <path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Zm0 0L19.5 7.125" />
              </svg>
            </button>
            <button @click="deleteMember(m.id)" class="text-gray-500 hover:text-red-400 transition-colors p-1">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
                <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
              </svg>
            </button>
          </template>
        </div>
      </div>

      <p v-else class="text-sm text-gray-500 mb-4 text-center py-3">{{ t('settings.no_members') }}</p>

      <!-- Add new member -->
      <div class="border-t border-white/[0.06] pt-4">
        <div class="flex gap-1 mb-2.5">
          <button
            v-for="c in memberColors"
            :key="c"
            @click="newMemberColor = c"
            class="w-6 h-6 rounded-full border-2 transition-all"
            :style="{ backgroundColor: c }"
            :class="newMemberColor === c ? 'border-white scale-110' : 'border-transparent'"
          />
        </div>
        <div class="flex gap-2">
          <input
            v-model="newMemberName"
            type="text"
            :placeholder="t('settings.member_name_placeholder')"
            class="flex-1 px-3 py-2 rounded-xl border border-white/[0.08] bg-white/[0.04] text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all placeholder-gray-500"
            @keyup.enter="addMember"
          />
          <button
            @click="addMember"
            :disabled="addingMember || !newMemberName.trim()"
            class="px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors disabled:opacity-50 shrink-0"
          >
            {{ addingMember ? '…' : t('settings.add_member') }}
          </button>
        </div>
      </div>
    </section>

    <!-- Bank connection -->
    <section class="bg-[#111119] rounded-2xl border border-white/[0.06] p-5">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-xl bg-blue-900/30 flex items-center justify-center shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5 text-blue-400">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 21v-8.25M15.75 21v-8.25M8.25 21v-8.25M3 9l9-6 9 6m-1.5 12V10.332A48.36 48.36 0 0 0 12 9.75c-2.551 0-5.056.2-7.5.582V21M3 21h18M12 6.75h.008v.008H12V6.75Z" />
          </svg>
        </div>
        <div class="flex-1 min-w-0">
          <h3 class="text-sm font-semibold text-gray-200">{{ locale === 'el' ? 'Σύνδεση Τράπεζας' : 'Bank Connection' }}</h3>
          <p class="text-xs text-gray-500 mt-0.5">{{ locale === 'el' ? 'Σύνδεσε τον τραπεζικό σου λογαριασμό για αυτόματο συγχρονισμό (Nordigen/GoCardless)' : 'Connect your bank for automatic transaction sync (Nordigen/GoCardless)' }}</p>
        </div>
        <RouterLink
          to="/bank"
          class="shrink-0 flex items-center gap-1.5 px-4 py-2 rounded-xl border border-white/[0.08] bg-white/[0.04] text-sm font-medium text-gray-300 hover:text-white hover:border-blue-600/60 hover:bg-blue-600/10 transition-all"
        >
          {{ locale === 'el' ? 'Διαχείριση' : 'Manage' }}
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-3.5 h-3.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
          </svg>
        </RouterLink>
      </div>
    </section>

    <!-- Data export / import -->
    <section class="bg-[#111119] rounded-2xl border border-white/[0.06] p-5">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">{{ t('settings.export') }}</h3>
      <p class="text-sm text-gray-400 mb-4">{{ t('settings.export_desc') }}</p>
      <div class="flex flex-wrap gap-3">
        <button
          @click="exportBillsCSV"
          class="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-white/[0.10] hover:border-blue-600/60 hover:bg-blue-600/10 text-sm font-medium text-gray-300 hover:text-blue-300 transition-all"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" />
          </svg>
          {{ t('settings.export_csv') }}
        </button>
        <button
          @click="triggerImport"
          class="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-white/[0.10] hover:border-emerald-600/60 hover:bg-emerald-600/10 text-sm font-medium text-gray-300 hover:text-emerald-300 transition-all"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 7.5m0 0-4.5 4.5M12 7.5V3" />
          </svg>
          {{ t('settings.import_csv') }}
        </button>
        <button
          @click="exportCalendarICS"
          class="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-white/[0.10] hover:border-purple-600/60 hover:bg-purple-600/10 text-sm font-medium text-gray-300 hover:text-purple-300 transition-all"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 0 1 2.25-2.25h13.5A2.25 2.25 0 0 1 21 7.5v11.25m-18 0A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75m-18 0v-7.5A2.25 2.25 0 0 1 5.25 9h13.5A2.25 2.25 0 0 1 21 11.25v7.5" />
          </svg>
          {{ t('settings.export_ics') }}
        </button>
      </div>
      <!-- Hidden file input -->
      <input ref="csvFileInput" type="file" accept=".csv,text/csv" class="hidden" @change="onFileSelected" />
    </section>

    <!-- Import preview modal -->
    <div v-if="showImportModal" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0" @click.self="showImportModal = false">
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl shadow-2xl w-full max-w-lg flex flex-col max-h-[90vh]">

        <!-- Header -->
        <div class="flex items-center justify-between px-5 pt-5 pb-4 border-b border-white/[0.06] shrink-0">
          <h3 class="text-base font-semibold text-gray-50">{{ t('settings.import_preview_title') }}</h3>
          <button @click="showImportModal = false" class="text-gray-500 hover:text-gray-300 transition-colors">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <!-- Summary badges -->
        <div class="flex gap-3 px-5 py-3 border-b border-white/[0.06] shrink-0">
          <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-900/30 text-emerald-400 border border-emerald-700/40">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400" />
            {{ t('settings.import_matched') }}: {{ matchedRows.length }}
          </span>
          <span v-if="skippedRows.length > 0" class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-red-900/20 text-red-400 border border-red-700/30">
            <span class="w-1.5 h-1.5 rounded-full bg-red-400" />
            {{ t('settings.import_skipped') }}: {{ skippedRows.length }}
          </span>
        </div>

        <!-- Rows table -->
        <div class="overflow-y-auto flex-1 px-5 py-3">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-xs text-gray-500 border-b border-white/[0.06]">
                <th class="pb-2 pr-3 font-medium">{{ t('settings.import_col_provider') }}</th>
                <th class="pb-2 pr-3 font-medium text-right">{{ t('settings.import_col_amount') }}</th>
                <th class="pb-2 pr-3 font-medium">{{ t('settings.import_col_due') }}</th>
                <th class="pb-2 font-medium">{{ t('settings.import_col_status') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, i) in importRows"
                :key="i"
                class="border-b border-white/[0.05] last:border-b-0"
                :class="row.matchedProviderID && row.amount > 0 && row.dueDate ? 'opacity-100' : 'opacity-40'"
              >
                <td class="py-2.5 pr-3">
                  <div class="flex items-center gap-1.5">
                    <span
                      class="w-1.5 h-1.5 rounded-full shrink-0"
                      :class="row.matchedProviderID ? 'bg-emerald-400' : 'bg-red-400'"
                    />
                    <span class="text-gray-200 truncate max-w-[120px]" :title="row.providerName">
                      {{ row.matchedProviderLabel || row.providerName }}
                    </span>
                  </div>
                  <p v-if="!row.matchedProviderID" class="text-[11px] text-red-400 ml-3 mt-0.5">{{ t('settings.import_no_match') }}</p>
                </td>
                <td class="py-2.5 pr-3 text-right font-medium text-gray-200 whitespace-nowrap">
                  {{ row.amount > 0 ? row.amount.toFixed(2) + ' €' : '—' }}
                </td>
                <td class="py-2.5 pr-3 text-gray-400 text-xs whitespace-nowrap">{{ row.dueDate || '—' }}</td>
                <td class="py-2.5 text-xs">
                  <span
                    class="px-1.5 py-0.5 rounded text-[10px] font-semibold"
                    :class="row.status === 'paid' ? 'bg-emerald-900/30 text-emerald-400' : row.status === 'overdue' ? 'bg-red-900/30 text-red-400' : 'bg-white/[0.06] text-gray-400'"
                  >
                    {{ row.status }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Footer -->
        <div class="flex gap-3 px-5 py-4 border-t border-white/[0.06] shrink-0">
          <button
            @click="showImportModal = false"
            class="flex-1 px-4 py-2.5 rounded-xl border border-white/[0.08] text-sm font-medium text-gray-300 hover:bg-white/[0.04] transition-colors"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            @click="confirmImport"
            :disabled="importing || matchedRows.length === 0"
            class="flex-1 px-4 py-2.5 rounded-xl bg-emerald-700 hover:bg-emerald-600 text-white text-sm font-semibold transition-colors disabled:opacity-50"
          >
            {{ importing ? t('settings.import_importing') : `${t('settings.import_confirm')} (${matchedRows.length})` }}
          </button>
        </div>
      </div>
    </div>

    <!-- Gmail auto-import -->
    <section class="bg-[#111119] rounded-2xl border border-white/[0.06] p-5">
      <div class="flex items-start gap-3 mb-4">
        <svg viewBox="0 0 24 24" class="w-5 h-5 mt-0.5 shrink-0" fill="none">
          <path d="M3 5.5h18v13H3v-13Z" stroke="#4285F4" stroke-width="1.5" stroke-linejoin="round"/>
          <path d="m3 5.5 9 8 9-8" stroke="#EA4335" stroke-width="1.5" stroke-linejoin="round"/>
        </svg>
        <div>
          <h3 class="text-sm font-semibold text-gray-300">{{ t('settings.gmail_title') }}</h3>
          <p class="text-xs text-gray-500 mt-0.5">{{ t('settings.gmail_desc') }}</p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <button
          @click="connectGmail"
          :disabled="gmailScanning"
          class="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-white/[0.10] hover:border-blue-600/60 hover:bg-blue-600/10 text-sm font-medium text-gray-300 hover:text-blue-300 transition-all disabled:opacity-50"
        >
          <svg v-if="gmailScanning" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
          <svg v-else xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21.75 6.75v10.5a2.25 2.25 0 0 1-2.25 2.25h-15a2.25 2.25 0 0 1-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0 0 19.5 4.5h-15a2.25 2.25 0 0 0-2.25 2.25m19.5 0v.243a2.25 2.25 0 0 1-1.07 1.916l-7.5 4.615a2.25 2.25 0 0 1-2.36 0L3.32 8.91a2.25 2.25 0 0 1-1.07-1.916V6.75" />
          </svg>
          {{ gmailScanning ? t('settings.gmail_scanning') : t('settings.gmail_connect') }}
        </button>

        <div class="flex items-center gap-1.5 text-xs text-gray-500">
          <span>{{ t('settings.gmail_days').replace('{n}', String(gmailDays)) }}</span>
          <button v-for="d in [7, 30, 60]" :key="d" @click="gmailDays = d"
            class="px-2 py-0.5 rounded-md border text-xs transition-colors"
            :class="gmailDays === d ? 'border-blue-600 text-blue-400 bg-blue-600/10' : 'border-white/[0.08] text-gray-500 hover:border-white/[0.10]'">
            {{ d }}d
          </button>
        </div>
      </div>

      <div v-if="gmailImported !== null" class="mt-3 text-sm text-emerald-400 flex items-center gap-1.5">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4"><path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5"/></svg>
        {{ t('settings.gmail_imported').replace('{n}', String(gmailImported)) }}
      </div>
      <p v-if="gmailError" class="mt-3 text-xs text-amber-400">{{ gmailError }}</p>
    </section>

    <!-- Gmail preview modal -->
    <div v-if="showGmailModal" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0" @click.self="showGmailModal = false">
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl shadow-2xl w-full max-w-lg flex flex-col max-h-[90vh]">
        <div class="flex items-center justify-between px-5 pt-5 pb-4 border-b border-white/[0.06] shrink-0">
          <div>
            <h3 class="text-base font-semibold text-gray-50">Gmail — {{ t('settings.gmail_found').replace('{n}', String(gmailFoundBills.length)) }}</h3>
            <p class="text-xs text-gray-500 mt-0.5">{{ gmailFoundBills.filter(b => b.userProviderID).length }} {{ locale === 'el' ? 'αντιστοιχίστηκαν' : 'matched to a provider' }}</p>
          </div>
          <button @click="showGmailModal = false" class="text-gray-500 hover:text-gray-300">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12"/></svg>
          </button>
        </div>
        <div class="overflow-y-auto flex-1 px-5 py-3 space-y-2">
          <div v-for="(item, i) in gmailFoundBills" :key="i"
            class="p-3 rounded-xl border border-white/[0.06]"
            :class="item.userProviderID ? 'bg-white/[0.04]' : 'bg-white/[0.02] opacity-50'"
          >
            <div class="flex items-center justify-between gap-2 mb-1">
              <span class="text-xs text-gray-500 truncate">{{ item.subject }}</span>
              <span v-if="!item.userProviderID" class="text-[10px] text-red-400 shrink-0">{{ locale === 'el' ? 'χωρίς αντιστοίχιση' : 'no match' }}</span>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-sm font-semibold text-gray-100">{{ item.result.provider_name || '—' }}</span>
              <span class="text-sm font-semibold text-emerald-400">{{ item.result.amount?.toFixed(2) }}€</span>
              <span class="text-xs text-gray-500">{{ item.result.due_date?.split('T')[0] }}</span>
            </div>
          </div>
        </div>
        <div class="flex gap-3 px-5 py-4 border-t border-white/[0.06] shrink-0">
          <button @click="showGmailModal = false" class="flex-1 px-4 py-2.5 rounded-xl border border-white/[0.08] text-sm font-medium text-gray-300 hover:bg-white/[0.04] transition-colors">
            {{ t('common.cancel') }}
          </button>
          <button
            @click="importGmailBills"
            :disabled="gmailImporting || gmailFoundBills.filter(b => b.userProviderID).length === 0"
            class="flex-1 px-4 py-2.5 rounded-xl bg-blue-700 hover:bg-blue-600 text-white text-sm font-semibold transition-colors disabled:opacity-50"
          >
            {{ gmailImporting ? t('settings.gmail_importing') : `${t('settings.gmail_import_all')} (${gmailFoundBills.filter(b => b.userProviderID).length})` }}
          </button>
        </div>
      </div>
    </div>

    <!-- Email notifications info -->
    <section class="bg-[#111119] rounded-2xl border border-white/[0.06] p-5">
      <div class="flex items-start gap-3">
        <div class="w-8 h-8 rounded-lg bg-indigo-900/30 flex items-center justify-center shrink-0 mt-0.5">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4 text-indigo-400">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21.75 6.75v10.5a2.25 2.25 0 0 1-2.25 2.25h-15a2.25 2.25 0 0 1-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0 0 19.5 4.5h-15a2.25 2.25 0 0 0-2.25 2.25m19.5 0v.243a2.25 2.25 0 0 1-1.07 1.916l-7.5 4.615a2.25 2.25 0 0 1-2.36 0L3.32 8.91a2.25 2.25 0 0 1-1.07-1.916V6.75" />
          </svg>
        </div>
        <div class="flex-1 min-w-0">
          <h3 class="text-sm font-semibold text-gray-300">{{ t('settings.email_notif_title') }}</h3>
          <p class="text-xs text-gray-500 mt-0.5">{{ t('settings.email_notif_desc') }}</p>
          <div class="mt-2 flex items-center gap-2 text-xs text-gray-400">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-3.5 h-3.5 text-indigo-400 shrink-0">
              <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5"/>
            </svg>
            <span class="truncate">{{ authStore.user?.email }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Danger zone -->
    <section class="bg-[#111119] rounded-2xl border border-red-950 p-5">
      <h3 class="text-sm font-semibold text-red-400 uppercase tracking-wide mb-4">{{ t('settings.danger') }}</h3>
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="text-sm font-medium text-gray-200">{{ t('settings.delete_account') }}</p>
          <p class="text-xs text-gray-500 mt-0.5">{{ t('settings.delete_account_desc') }}</p>
        </div>
        <button
          @click="showDeleteModal = true"
          class="shrink-0 px-4 py-2 rounded-xl border border-red-700/60 text-red-400 hover:bg-red-900/20 text-sm font-medium transition-colors"
        >
          {{ t('common.delete') }}
        </button>
      </div>
    </section>

    <!-- Delete account modal -->
    <div v-if="showDeleteModal" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 px-4" @click.self="showDeleteModal = false">
      <div class="bg-[#111119] rounded-2xl shadow-2xl w-full max-w-sm p-6 border border-red-950">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 rounded-full bg-red-900/30 flex items-center justify-center shrink-0">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5 text-red-400">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
            </svg>
          </div>
          <div>
            <h3 class="text-base font-semibold text-gray-50">{{ t('settings.delete_account') }}</h3>
            <p class="text-xs text-gray-400">{{ t('settings.delete_modal_subtitle') }}</p>
          </div>
        </div>

        <p class="text-sm text-gray-400 mb-4">
          {{ t('settings.delete_modal_body') }} <span class="text-gray-200 font-medium">{{ t('settings.delete_modal_bold') }}</span> {{ t('settings.delete_modal_body2') }}
        </p>

        <input
          v-model="deletePassword"
          type="password"
          :placeholder="t('settings.delete_password')"
          class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent transition-all mb-4"
          @keyup.enter="confirmDeleteAccount"
        />

        <div class="flex gap-3">
          <button
            @click="showDeleteModal = false; deletePassword = ''"
            class="flex-1 px-4 py-2.5 rounded-xl border border-white/[0.08] text-sm font-medium text-gray-300 hover:bg-white/[0.04] transition-colors"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            @click="confirmDeleteAccount"
            :disabled="!deletePassword || deletingAccount"
            class="flex-1 px-4 py-2.5 rounded-xl bg-red-700 hover:bg-red-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
          >
            {{ deletingAccount ? t('settings.deleting') : t('settings.delete_final') }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>
