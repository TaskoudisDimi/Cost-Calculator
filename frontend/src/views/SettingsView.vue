<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { updateProfile, sendPasswordResetEmail, deleteUser, reauthenticateWithCredential, EmailAuthProvider } from 'firebase/auth'
import { auth } from '@/firebase'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useBillsStore } from '@/stores/bills'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const billsStore = useBillsStore()
const { toast } = useToast()

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
  ])
})

async function saveProfile() {
  if (!auth.currentUser) return
  savingProfile.value = true
  try {
    await updateProfile(auth.currentUser, { displayName: displayName.value.trim() })
    toast('Προφίλ ενημερώθηκε', 'success')
  } catch {
    toast('Αποτυχία ενημέρωσης', 'error')
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
    toast('Email επαναφοράς εστάλη', 'success')
  } catch {
    toast('Αποτυχία αποστολής email', 'error')
  }
}

async function saveNotifSettings() {
  savingNotif.value = true
  try {
    await settingsStore.saveSettings()
    toast('Ρυθμίσεις αποθηκεύτηκαν', 'success')
  } catch {
    toast('Αποτυχία αποθήκευσης', 'error')
  } finally {
    savingNotif.value = false
  }
}

function exportBillsCSV() {
  const bills = Array.isArray(billsStore.bills) ? billsStore.bills : []
  if (bills.length === 0) {
    toast('Δεν υπάρχουν λογαριασμοί για εξαγωγή', 'warning')
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
  toast(`Εξήχθησαν ${bills.length} λογαριασμοί`, 'success')
}

async function confirmDeleteAccount() {
  if (!auth.currentUser || !auth.currentUser.email) return
  deletingAccount.value = true
  try {
    // Re-authenticate before deletion
    const credential = EmailAuthProvider.credential(auth.currentUser.email, deletePassword.value)
    await reauthenticateWithCredential(auth.currentUser, credential)
    // Delete Firestore data via backend
    await settingsStore.deleteAccount()
    // Delete Firebase Auth user
    await deleteUser(auth.currentUser)
    authStore.logout()
    router.push({ name: 'login' })
  } catch (e: unknown) {
    const code = (e as { code?: string }).code
    const msg = code === 'auth/wrong-password' || code === 'auth/invalid-credential'
      ? 'Λάθος κωδικός πρόσβασης'
      : 'Αποτυχία διαγραφής λογαριασμού'
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
      <h2 class="text-xl md:text-2xl font-bold text-gray-50">Ρυθμίσεις</h2>
    </div>

    <!-- Profile section -->
    <section class="bg-gray-800 rounded-2xl border border-gray-700 p-5">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">Προφίλ</h3>

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
          <label class="block text-sm font-medium text-gray-300 mb-1">Εμφανιζόμενο όνομα</label>
          <div class="flex gap-2">
            <input
              v-model="displayName"
              type="text"
              placeholder="Το όνομά σου"
              class="flex-1 px-3 py-2.5 rounded-xl border border-gray-600 bg-gray-900/50 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              @click="saveProfile"
              :disabled="savingProfile || !displayName.trim()"
              class="px-4 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors disabled:opacity-50"
            >
              {{ savingProfile ? '…' : 'Αποθήκευση' }}
            </button>
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">Email</label>
          <input
            :value="authStore.user?.email"
            type="email"
            disabled
            class="w-full px-3 py-2.5 rounded-xl border border-gray-700 bg-gray-900/30 text-sm text-gray-500 cursor-not-allowed"
          />
        </div>

        <div class="pt-1">
          <button
            @click="sendPasswordReset"
            :disabled="resetEmailSent"
            class="flex items-center gap-2 text-sm text-blue-400 hover:text-blue-300 disabled:text-gray-500 transition-colors font-medium"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z" />
            </svg>
            {{ resetEmailSent ? '✓ Το email εστάλη' : 'Αλλαγή κωδικού μέσω email' }}
          </button>
          <p v-if="resetEmailSent" class="text-xs text-gray-500 mt-1 ml-6">
            Έλεγξε τα εισερχόμενά σου στο {{ authStore.user?.email }}
          </p>
        </div>
      </div>
    </section>

    <!-- Notifications section -->
    <section class="bg-gray-800 rounded-2xl border border-gray-700 p-5">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">Ειδοποιήσεις</h3>

      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm font-medium text-gray-200">Push ειδοποιήσεις</p>
            <p class="text-xs text-gray-500 mt-0.5">Υπενθυμίσεις για επερχόμενους λογαριασμούς</p>
          </div>
          <button
            @click="settingsStore.notifEnabled = !settingsStore.notifEnabled"
            class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
            :class="settingsStore.notifEnabled ? 'bg-blue-600' : 'bg-gray-600'"
          >
            <span
              class="inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform"
              :class="settingsStore.notifEnabled ? 'translate-x-6' : 'translate-x-1'"
            />
          </button>
        </div>

        <div v-if="settingsStore.notifEnabled">
          <label class="block text-sm font-medium text-gray-300 mb-2">
            Ειδοποίηση πριν τη λήξη
          </label>
          <div class="flex gap-2 flex-wrap">
            <button
              v-for="days in notifDaysOptions"
              :key="days"
              @click="settingsStore.notifDays = days"
              class="px-3 py-1.5 rounded-lg text-sm font-medium border transition-colors"
              :class="settingsStore.notifDays === days
                ? 'bg-blue-600 border-blue-600 text-white'
                : 'border-gray-600 text-gray-400 hover:border-gray-500 hover:text-gray-300'"
            >
              {{ days === 1 ? '1 μέρα' : `${days} μέρες` }}
            </button>
          </div>
        </div>

        <button
          @click="saveNotifSettings"
          :disabled="savingNotif"
          class="w-full py-2.5 rounded-xl bg-gray-700 hover:bg-gray-600 text-sm font-medium text-gray-200 transition-colors disabled:opacity-50"
        >
          {{ savingNotif ? 'Αποθήκευση…' : 'Αποθήκευση ρυθμίσεων' }}
        </button>
      </div>
    </section>

    <!-- Currency / Display -->
    <section class="bg-gray-800 rounded-2xl border border-gray-700 p-5">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">Εμφάνιση</h3>
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-2">Νόμισμα</label>
        <div class="flex gap-2">
          <button
            v-for="opt in currencyOptions"
            :key="opt.value"
            @click="settingsStore.currency = opt.value; settingsStore.saveSettings()"
            class="flex-1 py-2 rounded-xl text-sm font-medium border transition-colors"
            :class="settingsStore.currency === opt.value
              ? 'bg-blue-600 border-blue-600 text-white'
              : 'border-gray-600 text-gray-400 hover:border-gray-500 hover:text-gray-200'"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
    </section>

    <!-- Data export -->
    <section class="bg-gray-800 rounded-2xl border border-gray-700 p-5">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-4">Εξαγωγή δεδομένων</h3>
      <p class="text-sm text-gray-400 mb-4">Κατέβασε τα δεδομένα σου σε μορφή CSV (Excel-compatible).</p>
      <button
        @click="exportBillsCSV"
        class="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-gray-600 hover:border-blue-600/60 hover:bg-blue-600/10 text-sm font-medium text-gray-300 hover:text-blue-300 transition-all"
      >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
          <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" />
        </svg>
        Εξαγωγή λογαριασμών (.csv)
      </button>
    </section>

    <!-- Danger zone -->
    <section class="bg-gray-800 rounded-2xl border border-red-900/40 p-5">
      <h3 class="text-sm font-semibold text-red-400 uppercase tracking-wide mb-4">Επικίνδυνη ζώνη</h3>
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="text-sm font-medium text-gray-200">Διαγραφή λογαριασμού</p>
          <p class="text-xs text-gray-500 mt-0.5">Μόνιμη διαγραφή όλων των δεδομένων σου. Δεν αναιρείται.</p>
        </div>
        <button
          @click="showDeleteModal = true"
          class="shrink-0 px-4 py-2 rounded-xl border border-red-700/60 text-red-400 hover:bg-red-900/20 text-sm font-medium transition-colors"
        >
          Διαγραφή
        </button>
      </div>
    </section>

    <!-- Delete account modal -->
    <div v-if="showDeleteModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4" @click.self="showDeleteModal = false">
      <div class="bg-gray-800 rounded-2xl shadow-xl w-full max-w-sm p-6 border border-red-700/40">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 rounded-full bg-red-900/30 flex items-center justify-center shrink-0">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5 text-red-400">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
            </svg>
          </div>
          <div>
            <h3 class="text-base font-semibold text-gray-50">Διαγραφή λογαριασμού</h3>
            <p class="text-xs text-gray-400">Αυτή η ενέργεια δεν αναιρείται</p>
          </div>
        </div>

        <p class="text-sm text-gray-400 mb-4">
          Θα διαγραφούν <span class="text-gray-200 font-medium">όλοι οι λογαριασμοί, αγορές, έσοδα και ρυθμίσεις</span> σου. Εισήγαγε τον κωδικό σου για επιβεβαίωση.
        </p>

        <input
          v-model="deletePassword"
          type="password"
          placeholder="Κωδικός πρόσβασης"
          class="w-full px-3 py-2.5 rounded-xl border border-gray-600 bg-gray-900/50 text-sm text-gray-100 focus:outline-none focus:ring-2 focus:ring-red-500 mb-4"
          @keyup.enter="confirmDeleteAccount"
        />

        <div class="flex gap-3">
          <button
            @click="showDeleteModal = false; deletePassword = ''"
            class="flex-1 px-4 py-2.5 rounded-xl border border-gray-600 text-sm font-medium text-gray-300 hover:bg-gray-700/60 transition-colors"
          >
            Ακύρωση
          </button>
          <button
            @click="confirmDeleteAccount"
            :disabled="!deletePassword || deletingAccount"
            class="flex-1 px-4 py-2.5 rounded-xl bg-red-700 hover:bg-red-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
          >
            {{ deletingAccount ? 'Διαγραφή…' : 'Διαγραφή οριστικά' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>
