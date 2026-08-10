<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useLocale } from '@/composables/useLocale'

const router = useRouter()
const auth = useAuthStore()
const { t } = useLocale()

const name = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const googleLoading = ref(false)

async function loginGoogle() {
  googleLoading.value = true
  error.value = ''
  try {
    await auth.loginWithGoogle()
    router.push({ name: 'dashboard' })
  } catch {
    error.value = t('register.error_google')
  } finally {
    googleLoading.value = false
  }
}

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.register(name.value, email.value, password.value)
    router.push({ name: 'dashboard' })
  } catch (e: any) {
    error.value = e.response?.data?.error || t('register.error_google')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-[#0d1117] flex flex-col items-center justify-center px-4 py-12 relative overflow-hidden">
    <div class="absolute -top-32 left-1/2 -translate-x-1/2 w-[700px] h-[360px] bg-blue-950/50 rounded-full blur-3xl pointer-events-none" aria-hidden="true" />

    <div class="relative w-full max-w-[360px]">
      <!-- Logo -->
      <div class="flex flex-col items-center mb-8">
        <div class="w-12 h-12 bg-blue-600 rounded-2xl flex items-center justify-center mb-3 shadow-xl shadow-blue-600/25">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="white" class="w-6 h-6">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"/>
          </svg>
        </div>
        <h1 class="text-xl font-bold text-white tracking-tight">BillTracker</h1>
        <p class="text-xs text-gray-500 mt-1">{{ t('register.subtitle') }}</p>
      </div>

      <!-- Card -->
      <div class="bg-[#111119] border border-white/[0.06] rounded-2xl p-7 shadow-2xl">
        <h2 class="text-base font-semibold text-white mb-5">{{ t('register.title') }}</h2>

        <!-- Google -->
        <button
          type="button"
          @click="loginGoogle"
          :disabled="googleLoading || loading"
          class="w-full flex items-center justify-center gap-2.5 px-4 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.06] hover:bg-white/[0.04] text-sm font-medium text-gray-200 transition-colors disabled:opacity-55 mb-5"
        >
          <svg viewBox="0 0 24 24" class="w-4 h-4 shrink-0">
            <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
            <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
            <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
            <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
          </svg>
          {{ googleLoading ? t('register.google_loading') : t('register.google') }}
        </button>

        <div class="flex items-center gap-3 mb-5">
          <div class="flex-1 h-px bg-white/[0.04]" />
          <span class="text-xs text-gray-600">{{ t('register.or_email') }}</span>
          <div class="flex-1 h-px bg-white/[0.04]" />
        </div>

        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5 tracking-wide">{{ t('register.name') }}</label>
            <input
              v-model="name"
              type="text"
              required
              autofocus
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              :placeholder="t('register.name_placeholder')"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5 tracking-wide">{{ t('register.email') }}</label>
            <input
              v-model="email"
              type="email"
              required
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5 tracking-wide">{{ t('register.password') }}</label>
            <input
              v-model="password"
              type="password"
              required
              minlength="8"
              class="w-full px-3.5 py-2.5 rounded-xl border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
              :placeholder="t('register.password_placeholder')"
            />
          </div>

          <div v-if="error" class="flex items-start gap-2.5 text-sm text-red-400 bg-red-500/8 border border-red-500/20 rounded-xl px-3.5 py-2.5">
            <svg class="w-4 h-4 mt-0.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
            </svg>
            {{ error }}
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full bg-blue-600 hover:bg-blue-500 active:bg-blue-700 disabled:opacity-55 text-white font-semibold py-2.5 rounded-xl transition-colors text-sm shadow-lg shadow-blue-600/15 mt-1"
          >
            {{ loading ? t('register.submitting') : t('register.submit') }}
          </button>
        </form>

        <p class="text-center text-sm text-gray-500 mt-6">
          {{ t('register.has_account') }}
          <RouterLink to="/login" class="text-blue-400 hover:text-blue-300 font-medium transition-colors">{{ t('register.login_link') }}</RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>
