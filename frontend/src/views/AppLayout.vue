<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

function logout() {
  auth.logout()
  router.push({ name: 'login' })
}

const navItems = [
  { name: 'dashboard', label: 'Dashboard', icon: '◉' },
  { name: 'bills', label: 'Λογαριασμοί', icon: '◈' },
  { name: 'expenses', label: 'Αγορές', icon: '◍' },
  { name: 'providers', label: 'Πάροχοι', icon: '◎' },
]
</script>

<template>
  <div class="min-h-screen bg-gray-50 flex">
    <!-- Sidebar -->
    <aside class="w-56 bg-white border-r border-gray-200 flex flex-col fixed inset-y-0">
      <div class="px-5 py-5 border-b border-gray-200">
        <h1 class="text-lg font-bold text-gray-900">BillTracker</h1>
        <p class="text-xs text-gray-400 mt-0.5 truncate">{{ auth.user?.email }}</p>
      </div>

      <nav class="flex-1 px-3 py-4 space-y-1">
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors"
          :class="route.name === item.name
            ? 'bg-blue-50 text-blue-700'
            : 'text-gray-600 hover:bg-gray-100'"
        >
          <span class="text-base">{{ item.icon }}</span>
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="px-3 py-4 border-t border-gray-200">
        <button
          @click="logout"
          class="flex items-center gap-3 px-3 py-2 w-full rounded-lg text-sm font-medium text-gray-600 hover:bg-red-50 hover:text-red-600 transition-colors"
        >
          <span>⎋</span>
          Αποσύνδεση
        </button>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 ml-56 p-8">
      <RouterView />
    </main>
  </div>
</template>
