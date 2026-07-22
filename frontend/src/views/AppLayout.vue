<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useBillsStore } from '@/stores/bills'
import { useBudgetStore } from '@/stores/budget'
import { useToast } from '@/composables/useToast'
import CalculatorModal from '@/components/CalculatorModal.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const billsStore = useBillsStore()
const budgetStore = useBudgetStore()
const { toasts } = useToast()

const sidebarOpen = ref(true)

onMounted(() => {
  const saved = localStorage.getItem('sidebar-open')
  if (saved !== null) sidebarOpen.value = saved === 'true'

  billsStore.startRealtimeSync()
  budgetStore.startRealtimeSync()
})

onUnmounted(() => {
  billsStore.stopRealtimeSync()
  budgetStore.stopRealtimeSync()
})

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
  localStorage.setItem('sidebar-open', String(sidebarOpen.value))
}

function logout() {
  billsStore.stopRealtimeSync()
  budgetStore.stopRealtimeSync()
  auth.logout()
  router.push({ name: 'login' })
}

const navItems = [
  {
    name: 'dashboard',
    label: 'Αρχική',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12 12 2.25 21.75 12M4.5 9.75v10.125c0 .621.504 1.125 1.125 1.125H9.75v-4.875c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21h4.125c.621 0 1.125-.504 1.125-1.125V9.75"/>`,
  },
  {
    name: 'bills',
    label: 'Λογαριασμοί',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"/>`,
  },
  {
    name: 'expenses',
    label: 'Αγορές',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 0 0-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 0 0-16.536-1.84M7.5 14.25 5.106 5.272M6 20.25a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Zm12.75 0a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Z"/>`,
  },
  {
    name: 'providers',
    label: 'Πάροχοι',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 21h16.5M4.5 3h15M5.25 3v18m13.5-18v18M9 6.75h1.5m-1.5 3h1.5m-1.5 3h1.5m3-6H15m-1.5 3H15m-1.5 3H15M9 21v-3.375c0-.621.504-1.125 1.125-1.125h3.75c.621 0 1.125.504 1.125 1.125V21"/>`,
  },
  {
    name: 'credentials',
    label: 'Κωδικοί',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 0 1 3 3m3 0a6 6 0 0 1-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 0 1 21.75 8.25Z"/>`,
  },
  {
    name: 'settings',
    label: 'Ρυθμίσεις',
    icon: `<path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z"/><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"/>`,
    sidebarOnly: true,
  },
]
</script>

<template>
  <div class="min-h-screen bg-gray-900 flex">

    <!-- Desktop sidebar -->
    <aside
      class="hidden md:flex flex-col fixed inset-y-0 z-30 bg-gray-950 border-r border-gray-800 transition-[width] duration-200 overflow-hidden"
      :class="sidebarOpen ? 'w-56' : 'w-14'"
    >
      <!-- Header -->
      <div class="flex items-center gap-3 px-3 py-4 border-b border-gray-800 min-h-[64px]">
        <button
          @click="toggleSidebar"
          class="p-1.5 rounded-lg hover:bg-gray-800 text-gray-500 shrink-0 transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12" />
          </svg>
        </button>
        <div v-show="sidebarOpen" class="overflow-hidden whitespace-nowrap min-w-0">
          <h1 class="text-sm font-bold text-gray-50 leading-tight">BillTracker</h1>
          <p class="text-xs text-gray-500 truncate">{{ auth.user?.email }}</p>
        </div>
      </div>

      <nav class="flex-1 px-2 py-3 space-y-0.5 overflow-y-auto">
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="flex items-center gap-3 px-2.5 py-2.5 rounded-xl text-sm font-medium transition-all"
          :class="route.name === item.name
            ? 'bg-blue-600/15 text-blue-400 shadow-sm'
            : 'text-gray-400 hover:bg-gray-800 hover:text-gray-200'"
          :title="!sidebarOpen ? item.label : undefined"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
            stroke-width="1.75" stroke="currentColor" class="w-5 h-5 shrink-0"
            v-html="item.icon" />
          <span v-show="sidebarOpen" class="whitespace-nowrap">{{ item.label }}</span>
          <span
            v-if="sidebarOpen && route.name === item.name"
            class="ml-auto w-1.5 h-1.5 rounded-full bg-blue-400"
          />
        </RouterLink>
      </nav>

      <div class="px-2 py-3 border-t border-gray-800">
        <button
          @click="logout"
          class="flex items-center gap-3 px-2.5 py-2.5 w-full rounded-xl text-sm font-medium text-gray-500 hover:bg-red-900/20 hover:text-red-400 transition-all"
          :title="!sidebarOpen ? 'Αποσύνδεση' : undefined"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5 shrink-0">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15M12 9l-3 3m0 0 3 3m-3-3h12.75"/>
          </svg>
          <span v-show="sidebarOpen" class="whitespace-nowrap">Αποσύνδεση</span>
        </button>
      </div>
    </aside>

    <!-- Mobile top header -->
    <header class="md:hidden fixed top-0 inset-x-0 z-30 bg-gray-950 border-b border-gray-800 flex items-center justify-between px-4 h-14">
      <div class="flex items-center gap-2">
        <div class="w-7 h-7 rounded-lg bg-blue-600 flex items-center justify-center">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="white" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"/>
          </svg>
        </div>
        <h1 class="text-base font-bold text-gray-50">BillTracker</h1>
      </div>
      <div class="flex items-center gap-1">
        <RouterLink
          :to="{ name: 'settings' }"
          class="p-2 rounded-lg transition-colors"
          :class="route.name === 'settings' ? 'text-blue-400 bg-blue-600/10' : 'text-gray-500 hover:text-gray-300 hover:bg-gray-800'"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z"/><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"/>
          </svg>
        </RouterLink>
        <button @click="logout" class="p-2 text-gray-500 hover:text-red-400 transition-colors rounded-lg hover:bg-gray-800">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15M12 9l-3 3m0 0 3 3m-3-3h12.75"/>
          </svg>
        </button>
      </div>
    </header>

    <!-- Main content -->
    <main
      class="flex-1 pt-14 md:pt-0 pb-20 md:pb-0 p-4 md:p-8 min-w-0 transition-[margin] duration-200 max-w-full"
      :class="sidebarOpen ? 'md:ml-56' : 'md:ml-14'"
    >
      <RouterView v-slot="{ Component }">
        <KeepAlive :max="5">
          <component :is="Component" />
        </KeepAlive>
      </RouterView>
    </main>

    <!-- Floating calculator -->
    <CalculatorModal />

    <!-- Mobile bottom tab bar -->
    <nav class="md:hidden fixed bottom-0 inset-x-0 z-30 bg-gray-950 border-t border-gray-800"
      style="padding-bottom: env(safe-area-inset-bottom)">
      <div class="flex">
        <RouterLink
          v-for="item in navItems.filter(i => !i.sidebarOnly)"
          :key="item.name"
          :to="{ name: item.name }"
          class="flex-1 flex flex-col items-center justify-center gap-1 py-2.5 transition-all relative"
          :class="route.name === item.name ? 'text-blue-400' : 'text-gray-500'"
        >
          <span
            v-if="route.name === item.name"
            class="absolute top-0 inset-x-3 h-0.5 rounded-full bg-blue-400"
          />
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
            stroke-width="1.75" stroke="currentColor" class="w-5 h-5"
            v-html="item.icon" />
          <span class="text-[10px] font-medium leading-tight">{{ item.label.split(' ')[0] }}</span>
        </RouterLink>
      </div>
    </nav>

    <!-- Toast container -->
    <Teleport to="body">
      <div class="fixed top-4 right-4 z-[200] flex flex-col gap-2 pointer-events-none w-72 max-w-[calc(100vw-2rem)]">
        <TransitionGroup
          enter-active-class="transition-all duration-300 ease-out"
          enter-from-class="opacity-0 translate-x-8 scale-95"
          enter-to-class="opacity-100 translate-x-0 scale-100"
          leave-active-class="transition-all duration-200 ease-in"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0 scale-95"
        >
          <div
            v-for="t in toasts"
            :key="t.id"
            class="flex items-center gap-3 px-4 py-3 rounded-xl shadow-xl border text-sm font-medium pointer-events-auto"
            :class="{
              'bg-gray-900 border-green-700/50 text-green-300': t.type === 'success',
              'bg-gray-900 border-red-700/50 text-red-300': t.type === 'error',
              'bg-gray-900 border-amber-700/50 text-amber-300': t.type === 'warning',
              'bg-gray-900 border-blue-700/50 text-blue-300': t.type === 'info',
            }"
          >
            <span class="shrink-0 text-base">
              {{ t.type === 'success' ? '✓' : t.type === 'error' ? '✕' : t.type === 'warning' ? '⚠' : 'ℹ' }}
            </span>
            {{ t.message }}
          </div>
        </TransitionGroup>
      </div>
    </Teleport>

  </div>
</template>
