<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useCredentialsStore } from '@/stores/credentials'
import { useConfirm } from '@/composables/useConfirm'
import { useLocale } from '@/composables/useLocale'
import { useToast } from '@/composables/useToast'
import type { UserCredential } from '@/types'

const store = useCredentialsStore()
const { confirm } = useConfirm()
const { t, locale } = useLocale()
const { toast } = useToast()

const showModal = ref(false)
const editingId = ref<string | null>(null)
const visiblePasswords = ref<Set<string>>(new Set())
const copiedKey = ref<string | null>(null)

const form = ref({
  label: '',
  username: '',
  password: '',
  url: '',
  notes: '',
})

// ── Bitwarden CSV import ──────────────────────────────────────────────
interface ImportRow {
  label: string
  username: string
  password: string
  url: string
  notes: string
}

const showImportModal = ref(false)
const importItems = ref<ImportRow[]>([])
const importSelected = ref<Set<number>>(new Set())
const importLoading = ref(false)
const importError = ref('')
const csvInput = ref<HTMLInputElement | null>(null)

function parseBitwardenCsv(text: string): ImportRow[] {
  // Strip UTF-8 BOM if present
  const raw = text.startsWith('﻿') ? text.slice(1) : text

  const lines = raw.split(/\r?\n/).filter(l => l.trim())
  if (lines.length < 2) return []

  // Parse CSV respecting quoted fields
  function parseRow(line: string): string[] {
    const cols: string[] = []
    let cur = ''
    let inQuote = false
    for (let i = 0; i < line.length; i++) {
      const ch = line[i]
      if (ch === '"') {
        if (inQuote && line[i + 1] === '"') { cur += '"'; i++ }
        else inQuote = !inQuote
      } else if (ch === ',' && !inQuote) {
        cols.push(cur); cur = ''
      } else {
        cur += ch
      }
    }
    cols.push(cur)
    return cols
  }

  const header = parseRow(lines[0]!).map(h => h.toLowerCase().trim())
  const idx = (name: string) => header.indexOf(name)

  const iType     = idx('type')
  const iName     = idx('name')
  const iNotes    = idx('notes')
  const iUri      = idx('login_uri')
  const iUser     = idx('login_username')
  const iPass     = idx('login_password')

  const rows: ImportRow[] = []
  for (let i = 1; i < lines.length; i++) {
    const cols = parseRow(lines[i]!)
    if (iType !== -1 && cols[iType]?.toLowerCase() !== 'login') continue
    const label = (iName !== -1 ? cols[iName] : '') ?? ''
    if (!label) continue
    rows.push({
      label,
      username: (iUser  !== -1 ? cols[iUser]  : '') ?? '',
      password: (iPass  !== -1 ? cols[iPass]  : '') ?? '',
      url:      (iUri   !== -1 ? cols[iUri]   : '') ?? '',
      notes:    (iNotes !== -1 ? cols[iNotes] : '') ?? '',
    })
  }
  return rows
}

function onCsvFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  importError.value = ''
  const reader = new FileReader()
  reader.onload = ev => {
    try {
      const rows = parseBitwardenCsv(ev.target!.result as string)
      if (rows.length === 0) {
        importError.value = locale.value === 'el'
          ? 'Δεν βρέθηκαν login items στο αρχείο.'
          : 'No login items found in file.'
        return
      }
      importItems.value = rows
      importSelected.value = new Set(rows.map((_, i) => i))
      showImportModal.value = true
    } catch {
      importError.value = locale.value === 'el'
        ? 'Σφάλμα ανάλυσης αρχείου CSV.'
        : 'Failed to parse CSV file.'
    }
    if (csvInput.value) csvInput.value.value = ''
  }
  reader.readAsText(file)
}

function toggleImportItem(i: number) {
  const s = new Set(importSelected.value)
  if (s.has(i)) s.delete(i); else s.add(i)
  importSelected.value = s
}

function toggleAllImport() {
  if (importSelected.value.size === importItems.value.length)
    importSelected.value = new Set()
  else
    importSelected.value = new Set(importItems.value.map((_, i) => i))
}

const importCount = computed(() => importSelected.value.size)

async function runImport() {
  if (importCount.value === 0) return
  importLoading.value = true
  let done = 0
  for (const i of importSelected.value) {
    const row = importItems.value[i]!
    await store.create(row)
    done++
  }
  importLoading.value = false
  showImportModal.value = false
  importItems.value = []
  importSelected.value = new Set()
  toast(
    locale.value === 'el'
      ? `Εισήχθησαν ${done} κωδικοί από Bitwarden`
      : `Imported ${done} credentials from Bitwarden`,
    'success',
  )
}

onMounted(() => store.startSync())
onUnmounted(() => store.stopSync())

function openAdd() {
  editingId.value = null
  form.value = { label: '', username: '', password: '', url: '', notes: '' }
  showModal.value = true
}

function openEdit(cred: UserCredential) {
  editingId.value = cred.id
  form.value = {
    label: cred.label,
    username: cred.username,
    password: cred.password,
    url: cred.url,
    notes: cred.notes,
  }
  showModal.value = true
}

async function submit() {
  if (editingId.value) {
    await store.update(editingId.value, form.value)
  } else {
    await store.create(form.value)
  }
  showModal.value = false
}

async function remove(id: string) {
  const msg = locale.value === 'en' ? 'Password will be permanently deleted.' : 'Ο κωδικός θα διαγραφεί οριστικά.'
  if (!await confirm({ message: msg })) return
  await store.remove(id)
}

function togglePassword(id: string) {
  const next = new Set(visiblePasswords.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  visiblePasswords.value = next
}

async function copyText(text: string, key: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = key
    setTimeout(() => { copiedKey.value = null }, 2000)
  } catch {
    // clipboard unavailable (non-https dev)
  }
}
</script>

<template>
  <div>
    <!-- Hidden CSV input -->
    <input ref="csvInput" type="file" accept=".csv,text/csv" class="hidden" @change="onCsvFile" />

    <div class="flex items-center justify-between mb-5 gap-3">
      <div class="min-w-0">
        <h2 class="text-xl md:text-2xl font-bold text-gray-50">{{ t('credentials.title') }}</h2>
        <p class="text-sm text-gray-400 mt-0.5">{{ t('credentials.subtitle') }}</p>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <!-- Import from Bitwarden -->
        <button
          @click="csvInput?.click()"
          class="flex items-center gap-1.5 px-3 py-2 rounded-lg border border-white/[0.08] bg-white/[0.03] text-sm font-medium text-gray-400 hover:text-gray-200 hover:border-white/[0.14] hover:bg-white/[0.06] transition-all"
          :title="locale === 'el' ? 'Εισαγωγή από Bitwarden CSV' : 'Import from Bitwarden CSV'"
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-4 h-4 text-blue-400">
            <path fill-rule="evenodd" d="M12 1.5a5.25 5.25 0 0 0-5.25 5.25v3a3 3 0 0 0-3 3v6.75a3 3 0 0 0 3 3h10.5a3 3 0 0 0 3-3v-6.75a3 3 0 0 0-3-3v-3c0-2.9-2.35-5.25-5.25-5.25Zm3.75 8.25v-3a3.75 3.75 0 1 0-7.5 0v3h7.5Z" clip-rule="evenodd" />
          </svg>
          {{ locale === 'el' ? 'Import' : 'Import' }}
        </button>
        <button
          @click="openAdd"
          class="bg-violet-600 hover:bg-violet-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors whitespace-nowrap"
        >
          {{ t('credentials.new_btn') }}
        </button>
      </div>
    </div>

    <!-- Loading skeleton -->
    <div v-if="store.loading" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-for="i in 3" :key="i" class="bg-white/[0.04] rounded-xl border border-white/[0.08] p-4 h-36 animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="store.credentials.length === 0"
      class="text-center py-20 bg-white/[0.04] rounded-xl border border-white/[0.08]">
      <div class="text-5xl mb-3">🔑</div>
      <p class="text-gray-400 font-medium">{{ t('credentials.no_creds') }}</p>
      <p class="text-sm text-gray-400 mt-1">{{ t('credentials.no_creds_desc') }}</p>
      <button @click="openAdd"
        class="mt-4 bg-violet-600 hover:bg-violet-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors">
        {{ t('credentials.new_btn') }}
      </button>
    </div>

    <!-- Cards grid -->
    <div v-else-if="!store.loading" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
      <div
        v-for="cred in store.credentials"
        :key="cred.id"
        class="bg-white/[0.04] rounded-xl border border-white/[0.08] p-4 shadow-sm flex flex-col gap-3"
      >
        <!-- Header -->
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <p class="font-semibold text-gray-50 truncate">{{ cred.label }}</p>
            <p v-if="cred.notes" class="text-xs text-gray-400 mt-0.5 truncate">{{ cred.notes }}</p>
          </div>
          <div class="flex items-center gap-1 shrink-0">
            <button @click="openEdit(cred)"
              class="p-1.5 rounded-lg text-gray-400 hover:text-violet-500 hover:bg-violet-900/20 transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
                <path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Z"/>
              </svg>
            </button>
            <button @click="remove(cred.id)"
              class="p-1.5 rounded-lg text-gray-400 hover:text-red-400 hover:bg-red-900/20 transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
        </div>

        <!-- Username row -->
        <div v-if="cred.username" class="flex items-center gap-2 bg-[#111119]/50 rounded-lg px-3 py-2">
          <span class="text-xs text-gray-400 w-16 shrink-0">{{ t('credentials.user_label') }}</span>
          <span class="flex-1 text-sm text-gray-100 font-mono truncate">{{ cred.username }}</span>
          <button
            @click="copyText(cred.username, cred.id + '-u')"
            class="shrink-0 text-xs px-2 py-1 rounded-md font-medium transition-colors"
            :class="copiedKey === cred.id + '-u'
              ? 'bg-green-900/30 text-green-400'
              : 'bg-white/[0.04] border border-white/[0.08] text-gray-400 hover:border-violet-300 hover:text-violet-400'"
          >
            {{ copiedKey === cred.id + '-u' ? t('common.copied') : t('common.copy') }}
          </button>
        </div>

        <!-- Password row -->
        <div v-if="cred.password" class="flex items-center gap-2 bg-[#111119]/50 rounded-lg px-3 py-2">
          <span class="text-xs text-gray-400 w-16 shrink-0">{{ t('credentials.password_label') }}</span>
          <span class="flex-1 text-sm font-mono text-gray-100 truncate select-none">
            {{ visiblePasswords.has(cred.id) ? cred.password : '••••••••' }}
          </span>
          <button
            @click="togglePassword(cred.id)"
            class="shrink-0 p-1 rounded-md text-gray-400 hover:text-gray-300 transition-colors"
            :title="visiblePasswords.has(cred.id) ? t('credentials.hide') : t('credentials.show')"
          >
            <svg v-if="visiblePasswords.has(cred.id)" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 0 0 1.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.451 10.451 0 0 1 12 4.5c4.756 0 8.773 3.162 10.065 7.498a10.522 10.522 0 0 1-4.293 5.774M6.228 6.228 3 3m3.228 3.228 3.65 3.65m7.894 7.894L21 21m-3.228-3.228-3.65-3.65m0 0a3 3 0 1 0-4.243-4.243m4.242 4.242L9.88 9.88"/>
            </svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 0 1 0-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178Z"/>
              <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"/>
            </svg>
          </button>
          <button
            @click="copyText(cred.password, cred.id + '-p')"
            class="shrink-0 text-xs px-2 py-1 rounded-md font-medium transition-colors"
            :class="copiedKey === cred.id + '-p'
              ? 'bg-green-900/30 text-green-400'
              : 'bg-white/[0.04] border border-white/[0.08] text-gray-400 hover:border-violet-300 hover:text-violet-400'"
          >
            {{ copiedKey === cred.id + '-p' ? t('common.copied') : t('common.copy') }}
          </button>
        </div>

        <!-- Open URL -->
        <a v-if="cred.url" :href="cred.url" target="_blank" rel="noopener"
          class="flex items-center gap-1.5 text-xs text-violet-400 hover:text-violet-400 hover:underline mt-1">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-3.5 h-3.5 shrink-0">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v10.5A2.25 2.25 0 0 0 5.25 21h10.5A2.25 2.25 0 0 0 18 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25"/>
          </svg>
          {{ t('credentials.open_url') }}
        </a>
      </div>
    </div>

    <!-- ── Bitwarden Import Modal ──────────────────────────── -->
    <div v-if="showImportModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0">
      <div class="bg-[#111119] border border-white/[0.08] rounded-2xl shadow-2xl w-full max-w-lg flex flex-col max-h-[85vh]">

        <!-- Header -->
        <div class="flex items-center justify-between px-5 pt-5 pb-4 border-b border-white/[0.06] shrink-0">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-xl bg-blue-500/15 border border-blue-500/25 flex items-center justify-center shrink-0">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-5 h-5 text-blue-400">
                <path fill-rule="evenodd" d="M12 1.5a5.25 5.25 0 0 0-5.25 5.25v3a3 3 0 0 0-3 3v6.75a3 3 0 0 0 3 3h10.5a3 3 0 0 0 3-3v-6.75a3 3 0 0 0-3-3v-3c0-2.9-2.35-5.25-5.25-5.25Zm3.75 8.25v-3a3.75 3.75 0 1 0-7.5 0v3h7.5Z" clip-rule="evenodd" />
              </svg>
            </div>
            <div>
              <h3 class="text-base font-bold text-gray-50">
                {{ locale === 'el' ? 'Εισαγωγή από Bitwarden' : 'Import from Bitwarden' }}
              </h3>
              <p class="text-xs text-gray-500">
                {{ importItems.length }} {{ locale === 'el' ? 'login items βρέθηκαν' : 'login items found' }}
              </p>
            </div>
          </div>
          <button @click="showImportModal = false" class="p-2 text-gray-500 hover:text-gray-300 hover:bg-white/[0.06] rounded-xl transition-all">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <!-- Select all / deselect all -->
        <div class="flex items-center justify-between px-5 py-2.5 border-b border-white/[0.04] shrink-0">
          <button @click="toggleAllImport" class="text-xs font-semibold text-violet-400 hover:text-violet-300 transition-colors">
            {{ importSelected.size === importItems.length
              ? (locale === 'el' ? 'Αποεπιλογή όλων' : 'Deselect all')
              : (locale === 'el' ? 'Επιλογή όλων' : 'Select all') }}
          </button>
          <span class="text-xs text-gray-500">
            {{ importSelected.size }}/{{ importItems.length }} {{ locale === 'el' ? 'επιλεγμένα' : 'selected' }}
          </span>
        </div>

        <!-- Item list -->
        <div class="overflow-y-auto flex-1 py-1">
          <label
            v-for="(item, i) in importItems"
            :key="i"
            class="flex items-start gap-3 px-5 py-3 hover:bg-white/[0.03] active:bg-white/[0.05] cursor-pointer transition-colors"
          >
            <input
              type="checkbox"
              :checked="importSelected.has(i)"
              @change="toggleImportItem(i)"
              class="mt-0.5 w-4 h-4 rounded border-gray-600 accent-violet-600 cursor-pointer shrink-0"
            />
            <div class="flex-1 min-w-0">
              <p class="text-sm font-semibold text-gray-100 truncate">{{ item.label }}</p>
              <p v-if="item.username" class="text-xs text-gray-500 truncate font-mono mt-0.5">{{ item.username }}</p>
              <p v-if="item.url" class="text-[11px] text-gray-600 truncate mt-0.5">{{ item.url }}</p>
            </div>
            <span v-if="item.password" class="text-[10px] font-bold px-1.5 py-0.5 rounded-md bg-emerald-500/10 border border-emerald-500/20 text-emerald-500 shrink-0 mt-0.5">pw</span>
          </label>
        </div>

        <!-- Footer -->
        <div class="px-5 py-4 border-t border-white/[0.06] flex gap-3 shrink-0">
          <button
            @click="showImportModal = false"
            class="flex-1 px-4 py-2.5 rounded-xl border border-white/[0.08] text-sm font-medium text-gray-300 hover:bg-white/[0.04] transition-colors"
          >{{ locale === 'el' ? 'Ακύρωση' : 'Cancel' }}</button>
          <button
            @click="runImport"
            :disabled="importCount === 0 || importLoading"
            class="flex-1 px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-bold transition-colors flex items-center justify-center gap-2"
          >
            <svg v-if="importLoading" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            {{ importLoading
              ? (locale === 'el' ? 'Εισαγωγή...' : 'Importing...')
              : (locale === 'el' ? `Εισαγωγή ${importCount}` : `Import ${importCount}`) }}
          </button>
        </div>
      </div>
    </div>

    <!-- Add / Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/40 flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0">
      <div class="bg-white/[0.04] rounded-2xl shadow-xl w-full max-w-md p-6">
        <h3 class="text-lg font-semibold text-gray-50 mb-5">
          {{ editingId ? t('credentials.modal_edit') : t('credentials.modal_new') }}
        </h3>

        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.label') }}</label>
            <input v-model="form.label" type="text" required
              class="w-full px-3 py-2.5 rounded-lg border border-white/[0.10] text-sm focus:outline-none focus:ring-2 focus:ring-violet-500"
              :placeholder="t('credentials.label_placeholder')" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.username') }}</label>
            <input v-model="form.username" type="text" autocomplete="off"
              class="w-full px-3 py-2.5 rounded-lg border border-white/[0.10] text-sm focus:outline-none focus:ring-2 focus:ring-violet-500"
              placeholder="username or email" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.password') }}</label>
            <input v-model="form.password" type="text" autocomplete="new-password"
              class="w-full px-3 py-2.5 rounded-lg border border-white/[0.10] text-sm font-mono focus:outline-none focus:ring-2 focus:ring-violet-500"
              :placeholder="t('credentials.password').toLowerCase()" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.url') }}</label>
            <input v-model="form.url" type="url"
              class="w-full px-3 py-2.5 rounded-lg border border-white/[0.10] text-sm focus:outline-none focus:ring-2 focus:ring-violet-500"
              placeholder="https://..." />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.notes') }}</label>
            <input v-model="form.notes" type="text"
              class="w-full px-3 py-2.5 rounded-lg border border-white/[0.10] text-sm focus:outline-none focus:ring-2 focus:ring-violet-500"
              :placeholder="t('common.optional')" />
          </div>

          <div class="flex gap-3 pt-2">
            <button type="button" @click="showModal = false"
              class="flex-1 px-4 py-2.5 rounded-lg border border-white/[0.10] text-sm font-medium text-gray-300 hover:bg-white/[0.06] transition-colors">
              {{ t('common.cancel') }}
            </button>
            <button type="submit"
              class="flex-1 px-4 py-2.5 rounded-lg bg-violet-600 hover:bg-violet-700 text-white text-sm font-medium transition-colors">
              {{ editingId ? t('common.save') : t('common.add') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
