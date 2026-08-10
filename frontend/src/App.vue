<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import ConfirmModal from '@/components/ConfirmModal.vue'

const authStore = useAuthStore()
</script>

<template>
  <!-- Full-page splash while Firebase resolves auth state -->
  <div v-if="authStore.loading" class="fixed inset-0 bg-[#0b0b10] flex items-center justify-center z-50">
    <div class="flex flex-col items-center gap-4">
      <p class="text-[22px] font-bold tracking-tight select-none">
        <span class="text-blue-400">Bill</span><span class="text-white">Tracker</span>
      </p>
      <div class="w-5 h-5 animate-spin rounded-full border-2 border-white/10 border-t-blue-500" />
    </div>
  </div>
  <template v-else>
    <RouterView />
    <ConfirmModal />
  </template>
</template>
