<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useBillsStore } from '@/stores/bills'
import { useBudgetStore } from '@/stores/budget'
import CalculatorModal from '@/components/CalculatorModal.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const billsStore = useBillsStore()
const budgetStore = useBudgetStore()

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
    label: 'Dashboard',
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
]
</script>

<template>
  <div class="min-h-screen bg-gray-900 flex">

    <!-- Desktop sidebar -->
    <aside
      class="hidden md:flex flex-col fixed inset-y-0 z-30 bg-gray-800 border-r border-gray-700 transition-[width] duration-200 overflow-hidden"
      :class="sidebarOpen ? 'w-56' : 'w-14'"
    >
      <!-- Header with toggle -->
      <div class="flex items-center gap-2 px-3 py-4 border-b border-gray-700 min-h-[60px]">
        <button
          @click="toggleSidebar"
          class="p-1.5 rounded-lg hover:bg-gray-700 text-gray-400 shrink-0 transition-colors"
          :title="sidebarOpen ? 'Σύμπτυξη' : 'Ανάπτυξη'"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12" />
          </svg>
        </button>
        <div v-show="sidebarOpen" class="overflow-hidden whitespace-nowrap">
          <h1 class="text-base font-bold text-gray-50">BillTracker</h1>
          <p class="text-xs text-gray-400 truncate max-w-[140px]">{{ auth.user?.email }}</p>
        </div>
      </div>

      <nav class="flex-1 px-2 py-4 space-y-1 overflow-y-auto">
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="flex items-center gap-3 px-2.5 py-2 rounded-lg text-sm font-medium transition-colors"
          :class="route.name === item.name
            ? 'bg-blue-900/20 text-blue-400'
            : 'text-gray-400 hover:bg-gray-700'"
          :title="!sidebarOpen ? item.label : undefined"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
            stroke-width="1.75" stroke="currentColor" class="w-5 h-5 shrink-0"
            v-html="item.icon" />
          <span v-show="sidebarOpen" class="whitespace-nowrap">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="px-2 py-4 border-t border-gray-700">
        <button
          @click="logout"
          class="flex items-center gap-3 px-2.5 py-2 w-full rounded-lg text-sm font-medium text-gray-400 hover:bg-red-900/20 hover:text-red-400 transition-colors"
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
    <header class="md:hidden fixed top-0 inset-x-0 z-30 bg-gray-800 border-b border-gray-700 flex items-center justify-between px-4 h-14">
      <h1 class="text-base font-bold text-gray-50">BillTracker</h1>
      <button @click="logout" class="p-2 text-gray-400 hover:text-red-400 transition-colors">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15M12 9l-3 3m0 0 3 3m-3-3h12.75"/>
        </svg>
      </button>
    </header>

    <!-- Main content -->
    <main
      class="flex-1 pt-14 md:pt-0 pb-20 md:pb-0 p-4 md:p-8 min-w-0 transition-[margin] duration-200"
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
    <nav class="md:hidden fixed bottom-0 inset-x-0 z-30 bg-gray-800 border-t border-gray-700"
      style="padding-bottom: env(safe-area-inset-bottom)">
      <div class="flex">
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="flex-1 flex flex-col items-center justify-center gap-0.5 py-2 transition-colors"
          :class="route.name === item.name ? 'text-blue-400' : 'text-gray-400'"
        >
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
            stroke-width="1.75" stroke="currentColor" class="w-5 h-5"
            v-html="item.icon" />
          <span class="text-[10px] font-medium leading-tight">
            {{ item.name === 'dashboard' ? 'Αρχική' :
               item.name === 'credentials' ? 'Κωδικοί' :
               item.label.split(' ')[0] }}
          </span>
        </RouterLink>
      </div>
    </nav>

  </div>
</template>
