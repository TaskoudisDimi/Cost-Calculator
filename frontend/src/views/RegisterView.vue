<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

const name = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.register(name.value, email.value, password.value)
    router.push({ name: 'dashboard' })
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Σφάλμα κατά την εγγραφή.'
  } finally {
    loading.value = false
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
        <h2 class="text-xl font-semibold text-gray-100 mb-6">Εγγραφή</h2>

        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Όνομα</label>
            <input
              v-model="name"
              type="text"
              required
              class="w-full px-4 py-2.5 rounded-lg border border-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              placeholder="Γιώργης Παπαδόπουλος"
            />
          </div>

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
            <label class="block text-sm font-medium text-gray-300 mb-1">Κωδικός</label>
            <input
              v-model="password"
              type="password"
              required
              minlength="8"
              class="w-full px-4 py-2.5 rounded-lg border border-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              placeholder="Τουλάχιστον 8 χαρακτήρες"
            />
          </div>

          <p v-if="error" class="text-red-400 text-sm">{{ error }}</p>

          <button
            type="submit"
            :disabled="loading"
            class="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white font-medium py-2.5 rounded-lg transition-colors text-sm"
          >
            {{ loading ? 'Εγγραφή...' : 'Δημιουργία λογαριασμού' }}
          </button>
        </form>

        <p class="text-center text-sm text-gray-400 mt-6">
          Έχεις ήδη λογαριασμό;
          <RouterLink to="/login" class="text-blue-400 hover:underline font-medium">Σύνδεση</RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>
