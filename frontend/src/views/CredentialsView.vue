<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useCredentialsStore } from '@/stores/credentials'
import { useConfirm } from '@/composables/useConfirm'
import { useLocale } from '@/composables/useLocale'
import type { UserCredential } from '@/types'

const store = useCredentialsStore()
const { confirm } = useConfirm()
const { t, locale } = useLocale()

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
    <div class="flex items-center justify-between mb-5">
      <div>
        <h2 class="text-xl md:text-2xl font-bold text-gray-50">{{ t('credentials.title') }}</h2>
        <p class="text-sm text-gray-400 mt-0.5">{{ t('credentials.subtitle') }}</p>
      </div>
      <button
        @click="openAdd"
        class="bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors whitespace-nowrap"
      >
        {{ t('credentials.new_btn') }}
      </button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="store.loading" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
      <div v-for="i in 3" :key="i" class="bg-gray-800 rounded-xl border border-gray-700 p-4 h-36 animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="store.credentials.length === 0"
      class="text-center py-20 bg-gray-800 rounded-xl border border-gray-700">
      <div class="text-5xl mb-3">🔑</div>
      <p class="text-gray-400 font-medium">{{ t('credentials.no_creds') }}</p>
      <p class="text-sm text-gray-400 mt-1">{{ t('credentials.no_creds_desc') }}</p>
      <button @click="openAdd"
        class="mt-4 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors">
        {{ t('credentials.new_btn') }}
      </button>
    </div>

    <!-- Cards grid -->
    <div v-else-if="!store.loading" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
      <div
        v-for="cred in store.credentials"
        :key="cred.id"
        class="bg-gray-800 rounded-xl border border-gray-700 p-4 shadow-sm flex flex-col gap-3"
      >
        <!-- Header -->
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <p class="font-semibold text-gray-50 truncate">{{ cred.label }}</p>
            <p v-if="cred.notes" class="text-xs text-gray-400 mt-0.5 truncate">{{ cred.notes }}</p>
          </div>
          <div class="flex items-center gap-1 shrink-0">
            <button @click="openEdit(cred)"
              class="p-1.5 rounded-lg text-gray-400 hover:text-blue-500 hover:bg-blue-900/20 transition-colors">
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
        <div v-if="cred.username" class="flex items-center gap-2 bg-gray-900/50 rounded-lg px-3 py-2">
          <span class="text-xs text-gray-400 w-16 shrink-0">{{ t('credentials.user_label') }}</span>
          <span class="flex-1 text-sm text-gray-100 font-mono truncate">{{ cred.username }}</span>
          <button
            @click="copyText(cred.username, cred.id + '-u')"
            class="shrink-0 text-xs px-2 py-1 rounded-md font-medium transition-colors"
            :class="copiedKey === cred.id + '-u'
              ? 'bg-green-900/30 text-green-400'
              : 'bg-gray-800 border border-gray-700 text-gray-400 hover:border-blue-300 hover:text-blue-400'"
          >
            {{ copiedKey === cred.id + '-u' ? t('common.copied') : t('common.copy') }}
          </button>
        </div>

        <!-- Password row -->
        <div v-if="cred.password" class="flex items-center gap-2 bg-gray-900/50 rounded-lg px-3 py-2">
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
              : 'bg-gray-800 border border-gray-700 text-gray-400 hover:border-blue-300 hover:text-blue-400'"
          >
            {{ copiedKey === cred.id + '-p' ? t('common.copied') : t('common.copy') }}
          </button>
        </div>

        <!-- Open URL -->
        <a v-if="cred.url" :href="cred.url" target="_blank" rel="noopener"
          class="flex items-center gap-1.5 text-xs text-blue-400 hover:text-blue-400 hover:underline mt-1">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-3.5 h-3.5 shrink-0">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v10.5A2.25 2.25 0 0 0 5.25 21h10.5A2.25 2.25 0 0 0 18 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25"/>
          </svg>
          {{ t('credentials.open_url') }}
        </a>
      </div>
    </div>

    <!-- Add / Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/40 flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0">
      <div class="bg-gray-800 rounded-2xl shadow-xl w-full max-w-md p-6">
        <h3 class="text-lg font-semibold text-gray-50 mb-5">
          {{ editingId ? t('credentials.modal_edit') : t('credentials.modal_new') }}
        </h3>

        <form @submit.prevent="submit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.label') }}</label>
            <input v-model="form.label" type="text" required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              :placeholder="t('credentials.label_placeholder')" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.username') }}</label>
            <input v-model="form.username" type="text" autocomplete="off"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="username or email" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.password') }}</label>
            <input v-model="form.password" type="text" autocomplete="new-password"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
              :placeholder="t('credentials.password').toLowerCase()" />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.url') }}</label>
            <input v-model="form.url" type="url"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="https://..." />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('credentials.notes') }}</label>
            <input v-model="form.notes" type="text"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              :placeholder="t('common.optional')" />
          </div>

          <div class="flex gap-3 pt-2">
            <button type="button" @click="showModal = false"
              class="flex-1 px-4 py-2.5 rounded-lg border border-gray-600 text-sm font-medium text-gray-300 hover:bg-gray-700/60 transition-colors">
              {{ t('common.cancel') }}
            </button>
            <button type="submit"
              class="flex-1 px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium transition-colors">
              {{ editingId ? t('common.save') : t('common.add') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
