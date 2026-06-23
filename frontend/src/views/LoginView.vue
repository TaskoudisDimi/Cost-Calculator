<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { sendPasswordResetEmail } from 'firebase/auth'
import { auth as firebaseAuth } from '@/firebase'

const router = useRouter()
const auth = useAuthStore()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const showReset = ref(false)
const resetEmail = ref('')
const resetLoading = ref(false)
const resetError = ref('')
const resetSent = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(email.value, password.value)
    router.push({ name: 'dashboard' })
  } catch {
    error.value = 'Λάθος email ή κωδικός.'
  } finally {
    loading.value = false
  }
}

function openReset() {
  resetEmail.value = email.value
  resetError.value = ''
  resetSent.value = false
  showReset.value = true
}

async function sendReset() {
  if (!resetEmail.value) return
  resetLoading.value = true
  resetError.value = ''
  try {
    await sendPasswordResetEmail(firebaseAuth, resetEmail.value)
    resetSent.value = true
  } catch {
    resetError.value = 'Δεν βρέθηκε λογαριασμός με αυτό το email.'
  } finally {
    resetLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-gray-900/50 flex items-center justify-center px-4">
    <div class="w-full max-w-md">
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-gray-50">BillTracker</h1>
        <p class="text-gray-400 mt-1">Διαχείριση λογαριασμών</p>
      </div>

      <div class="bg-gray-800 rounded-2xl shadow-sm border border-gray-700 p-8">
        <h2 class="text-xl font-semibold text-gray-100 mb-6">Σύνδεση</h2>

        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Email</label>
            <input
              v-model="email"
              type="email"
              required
              class="w-full px-4 py-2.5 rounded-lg border border-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="block text-sm font-medium text-gray-300">Κωδικός</label>
              <button
                type="button"
                @click="openReset"
                class="text-xs text-blue-400 hover:underline"
              >
                Ξέχασα τον κωδικό μου
              </button>
            </div>
            <input
              v-model="password"
              type="password"
              required
              class="w-full px-4 py-2.5 rounded-lg border border-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              placeholder="••••••••"
            />
          </div>

          <p v-if="error" class="text-red-400 text-sm">{{ error }}</p>

          <button
            type="submit"
            :disabled="loading"
            class="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white font-medium py-2.5 rounded-lg transition-colors text-sm"
          >
            {{ loading ? 'Σύνδεση...' : 'Σύνδεση' }}
          </button>
        </form>

        <p class="text-center text-sm text-gray-400 mt-6">
          Δεν έχεις λογαριασμό;
          <RouterLink to="/register" class="text-blue-400 hover:underline font-medium">Εγγραφή</RouterLink>
        </p>
      </div>
    </div>

    <!-- Password reset modal -->
    <div v-if="showReset" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50 px-4">
      <div class="bg-gray-800 rounded-2xl shadow-xl w-full max-w-sm p-6">
        <h3 class="text-lg font-semibold text-gray-50 mb-1">Επαναφορά κωδικού</h3>

        <div v-if="resetSent" class="text-center py-4">
          <div class="text-4xl mb-3">📧</div>
          <p class="text-sm text-gray-300 font-medium">Στάλθηκε email επαναφοράς!</p>
          <p class="text-sm text-gray-400 mt-1">Ελέγξτε το inbox σας (και το spam).</p>
          <button
            @click="showReset = false"
            class="mt-5 w-full bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium py-2.5 rounded-lg transition-colors"
          >
            Κλείσιμο
          </button>
        </div>

        <template v-else>
          <p class="text-sm text-gray-400 mb-4">Δώσε το email σου και θα σου στείλουμε σύνδεσμο επαναφοράς.</p>

          <div class="space-y-3">
            <input
              v-model="resetEmail"
              type="email"
              required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="you@example.com"
            />

            <p v-if="resetError" class="text-red-400 text-xs">{{ resetError }}</p>

            <div class="flex gap-3">
              <button
                type="button"
                @click="showReset = false"
                class="flex-1 px-4 py-2.5 rounded-lg border border-gray-600 text-sm font-medium text-gray-300 hover:bg-gray-700/60 transition-colors"
              >
                Ακύρωση
              </button>
              <button
                type="button"
                :disabled="resetLoading || !resetEmail"
                @click="sendReset"
                class="flex-1 px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white text-sm font-medium transition-colors"
              >
                {{ resetLoading ? 'Αποστολή...' : 'Αποστολή' }}
              </button>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>
