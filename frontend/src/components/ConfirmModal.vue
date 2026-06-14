<script setup lang="ts">
import { useConfirm } from '@/composables/useConfirm'

const { visible, options, accept, cancel } = useConfirm()
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-100"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="visible"
        class="fixed inset-0 bg-black/40 backdrop-blur-sm flex items-center justify-center z-[200] px-4"
        @click.self="cancel"
      >
        <Transition
          enter-active-class="transition-all duration-150"
          enter-from-class="opacity-0 scale-95"
          enter-to-class="opacity-100 scale-100"
          leave-active-class="transition-all duration-100"
          leave-from-class="opacity-100 scale-100"
          leave-to-class="opacity-0 scale-95"
          appear
        >
          <div class="bg-white rounded-2xl shadow-2xl w-full max-w-sm p-6">
            <!-- Icon + Title -->
            <div class="flex items-start gap-4 mb-4">
              <div
                class="w-10 h-10 rounded-full flex items-center justify-center shrink-0"
                :class="options.variant === 'warning' ? 'bg-amber-100' : 'bg-red-100'"
              >
                <svg
                  v-if="options.variant !== 'warning'"
                  class="w-5 h-5 text-red-500"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                <svg
                  v-else
                  class="w-5 h-5 text-amber-500"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>
              <div>
                <h3 class="text-base font-semibold text-gray-900">
                  {{ options.title ?? (options.variant === 'warning' ? 'Προσοχή' : 'Διαγραφή') }}
                </h3>
                <p class="text-sm text-gray-500 mt-1">{{ options.message }}</p>
              </div>
            </div>

            <!-- Buttons -->
            <div class="flex gap-3 mt-6">
              <button
                @click="cancel"
                class="flex-1 px-4 py-2.5 rounded-xl border border-gray-200 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
              >
                Ακύρωση
              </button>
              <button
                @click="accept"
                class="flex-1 px-4 py-2.5 rounded-xl text-sm font-medium text-white transition-colors"
                :class="options.variant === 'warning' ? 'bg-amber-500 hover:bg-amber-600' : 'bg-red-500 hover:bg-red-600'"
              >
                {{ options.confirmLabel ?? 'Διαγραφή' }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
