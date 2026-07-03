<script setup lang="ts">
import { ref, computed } from 'vue'

const isOpen = ref(false)

const display = ref('0')
const expression = ref('')
const justEvaluated = ref(false)
const hasError = ref(false)

function open() { isOpen.value = true }
function close() { isOpen.value = false }

function reset() {
  display.value = '0'
  expression.value = ''
  justEvaluated.value = false
  hasError.value = false
}

function pressDigit(d: string) {
  if (hasError.value) reset()
  if (justEvaluated.value) {
    display.value = d
    justEvaluated.value = false
    return
  }
  if (display.value === '0' && d !== '.') {
    display.value = d
  } else if (d === '.' && display.value.includes('.')) {
    return
  } else {
    if (display.value.length >= 15) return
    display.value += d
  }
}

function pressOp(op: string) {
  if (hasError.value) reset()
  justEvaluated.value = false
  const last = expression.value.slice(-1)
  const isLastOp = ['+', '-', '×', '÷'].includes(last)

  if (isLastOp) {
    expression.value = expression.value.slice(0, -1) + op
  } else {
    expression.value += display.value + op
    display.value = '0'
  }
}

function pressPercent() {
  const val = parseFloat(display.value)
  if (isNaN(val)) return
  display.value = String(val / 100)
}

function backspace() {
  if (hasError.value) { reset(); return }
  if (justEvaluated.value) { reset(); return }
  if (display.value.length <= 1 || display.value === '-0') {
    display.value = '0'
  } else {
    display.value = display.value.slice(0, -1)
  }
}

function evaluate() {
  if (hasError.value) { reset(); return }
  const full = expression.value + display.value
  if (!expression.value) return

  try {
    const normalized = full
      .replace(/×/g, '*')
      .replace(/÷/g, '/')

    // Safe eval: only allow digits, operators, dots, parentheses
    if (!/^[\d\s+\-*/().]+$/.test(normalized)) throw new Error('invalid')

    // eslint-disable-next-line no-new-func
    const result = Function('"use strict"; return (' + normalized + ')')()
    if (!isFinite(result)) throw new Error('overflow')

    const rounded = parseFloat(result.toFixed(10))
    display.value = String(rounded)
    expression.value = ''
    justEvaluated.value = true
  } catch {
    display.value = 'Σφάλμα'
    expression.value = ''
    hasError.value = true
  }
}

const expressionDisplay = computed(() => {
  if (!expression.value) return ' '
  return expression.value.replace(/\*/g, '×').replace(/\//g, '÷')
})

const buttons = [
  ['C', '⌫', '%', '÷'],
  ['7', '8', '9', '×'],
  ['4', '5', '6', '-'],
  ['1', '2', '3', '+'],
  ['0', '.', '='],
]

function handleButton(btn: string) {
  if (btn === 'C') { reset(); return }
  if (btn === '⌫') { backspace(); return }
  if (btn === '%') { pressPercent(); return }
  if (btn === '=') { evaluate(); return }
  if (['+', '-', '×', '÷'].includes(btn)) { pressOp(btn); return }
  pressDigit(btn)
}

function btnClass(btn: string): string {
  if (btn === '=') return 'bg-blue-600 hover:bg-blue-500 text-white font-bold col-span-1 row-span-1'
  if (['+', '-', '×', '÷'].includes(btn)) return 'bg-gray-600 hover:bg-gray-500 text-blue-300 font-semibold'
  if (btn === 'C') return 'bg-red-900/40 hover:bg-red-900/60 text-red-400 font-semibold'
  if (btn === '⌫') return 'bg-gray-700 hover:bg-gray-600 text-amber-400'
  if (btn === '%') return 'bg-gray-700 hover:bg-gray-600 text-amber-400'
  if (btn === '0') return 'bg-gray-700 hover:bg-gray-600 text-gray-100 col-span-2'
  return 'bg-gray-700 hover:bg-gray-600 text-gray-100'
}
</script>

<template>
  <!-- Floating trigger button -->
  <button
    @click="open"
    class="fixed bottom-24 right-4 md:bottom-6 z-40 w-12 h-12 rounded-full bg-blue-600 hover:bg-blue-500 text-white shadow-lg flex items-center justify-center transition-all duration-200 hover:scale-110"
    title="Αριθμομηχανή"
  >
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5">
      <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 15.75V18m-7.5-6.75h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V13.5Zm0 2.25h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V18Zm2.498-6.75h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V13.5Zm0 2.25h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V18Zm2.504-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5Zm0 2.25h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V18Zm2.498-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5ZM8.25 6h7.5v2.25h-7.5V6ZM12 2.25c-1.892 0-3.758.11-5.593.322C5.307 2.7 4.5 3.578 4.5 4.585v15.83c0 1.007.807 1.885 1.907 2.013a48.272 48.272 0 0 0 5.593.322c1.892 0 3.758-.11 5.593-.322 1.1-.128 1.907-1.006 1.907-2.013V4.585c0-1.007-.807-1.885-1.907-2.013A48.272 48.272 0 0 0 12 2.25Z"/>
    </svg>
  </button>

  <!-- Calculator Modal -->
  <Teleport to="body">
    <div v-if="isOpen" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center sm:justify-end sm:p-6 sm:pb-6">
      <!-- Backdrop -->
      <div class="absolute inset-0 bg-black/50" @click="close" />

      <!-- Calculator panel -->
      <div class="relative bg-gray-900 border border-gray-700 rounded-2xl shadow-2xl w-full max-w-xs mx-4 sm:mx-0 mb-4 sm:mb-0 overflow-hidden">

        <!-- Header -->
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-700">
          <span class="text-sm font-semibold text-gray-300">Αριθμομηχανή</span>
          <button @click="close" class="text-gray-500 hover:text-gray-300 transition-colors">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-5 h-5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <!-- Display -->
        <div class="px-4 pt-4 pb-2 text-right">
          <p class="text-xs text-gray-500 h-4 truncate text-right">{{ expressionDisplay }}</p>
          <p
            class="font-mono font-bold text-gray-50 mt-1 truncate"
            :class="display.length > 10 ? 'text-2xl' : 'text-4xl'"
            :style="{ color: hasError ? '#f87171' : undefined }"
          >
            {{ display }}
          </p>
        </div>

        <!-- Buttons -->
        <div class="p-3 grid grid-cols-4 gap-2">
          <template v-for="row in buttons" :key="row.join('')">
            <template v-for="btn in row" :key="btn">
              <button
                @click="handleButton(btn)"
                class="rounded-xl py-4 text-lg font-medium transition-all duration-100 active:scale-95"
                :class="btnClass(btn)"
              >
                {{ btn }}
              </button>
            </template>
          </template>
        </div>

      </div>
    </div>
  </Teleport>
</template>
