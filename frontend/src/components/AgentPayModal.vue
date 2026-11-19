<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="modelValue" class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-end sm:items-center justify-center z-[200] px-4 pb-4 sm:pb-0" @click.self="close">
        <div class="bg-[#0d0d14] border border-white/[0.08] rounded-2xl shadow-2xl w-full max-w-md overflow-hidden">

          <!-- Header -->
          <div class="flex items-center gap-3 px-5 pt-5 pb-4 border-b border-white/[0.06]">
            <div class="w-10 h-10 rounded-xl flex items-center justify-center shrink-0 bg-blue-600/20 border border-blue-500/30">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5 text-blue-400">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09Z" />
              </svg>
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-bold text-white">AI Αυτόματη Πληρωμή</p>
              <p class="text-xs text-gray-400 truncate">{{ bill?.user_provider?.nickname || bill?.user_provider?.provider?.name }} — {{ formatAmount(bill?.amount ?? 0) }}</p>
            </div>
            <button v-if="!isRunning" @click="close" class="w-8 h-8 flex items-center justify-center text-gray-500 hover:text-white rounded-xl hover:bg-white/[0.06] transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" /></svg>
            </button>
          </div>

          <div class="p-5 space-y-4 max-h-[70vh] overflow-y-auto">

            <!-- Setup form (before start) -->
            <template v-if="phase === 'setup'">

              <!-- Credential selector -->
              <div>
                <label class="block text-xs font-bold text-gray-500 uppercase tracking-widest mb-2">Κωδικοί σύνδεσης</label>
                <div v-if="credentials.length === 0" class="text-xs text-gray-500 py-2">
                  Δεν βρέθηκαν αποθηκευμένοι κωδικοί.
                  <RouterLink :to="{ name: 'credentials' }" class="text-blue-400 ml-1">Πρόσθεσε →</RouterLink>
                </div>
                <div v-else class="space-y-1.5">
                  <button
                    v-for="cred in credentials" :key="cred.id"
                    @click="selectedCredId = selectedCredId === cred.id ? '' : cred.id"
                    class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl border transition-colors text-left"
                    :class="selectedCredId === cred.id
                      ? 'bg-blue-600/15 border-blue-500/40 text-white'
                      : 'border-white/[0.07] text-gray-300 hover:border-white/[0.15] hover:bg-white/[0.04]'"
                  >
                    <div class="w-7 h-7 rounded-lg flex items-center justify-center shrink-0"
                      :class="selectedCredId === cred.id ? 'bg-blue-600/30' : 'bg-white/[0.06]'">
                      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-3.5 h-3.5">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 0 1 3 3m3 0a6 6 0 0 1-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 0 1 21.75 8.25Z" />
                      </svg>
                    </div>
                    <span class="text-sm font-medium">{{ cred.label }}</span>
                    <span v-if="cred.url" class="text-xs text-gray-500 ml-auto truncate max-w-[100px]">{{ cred.url }}</span>
                  </button>
                </div>
              </div>

              <!-- Card details -->
              <div>
                <label class="block text-xs font-bold text-gray-500 uppercase tracking-widest mb-2">Κάρτα πληρωμής <span class="text-gray-600 font-normal normal-case">(προαιρετικό αν υπάρχει RF)</span></label>
                <div class="grid grid-cols-2 gap-2">
                  <input
                    v-model="cardNumber"
                    placeholder="1234 5678 9012 3456"
                    maxlength="19"
                    class="col-span-2 px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 font-mono tracking-widest"
                    @input="formatCardNumber"
                  />
                  <input v-model="cardExpiry" placeholder="MM/YY" maxlength="5"
                    class="px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 font-mono" />
                  <input v-model="cardCVV" placeholder="CVV" maxlength="4" type="password"
                    class="px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 font-mono" />
                  <input v-model="cardHolder" placeholder="Ονοματεπώνυμο κατόχου" maxlength="50"
                    class="col-span-2 px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 uppercase" />
                </div>
                <p class="text-[10px] text-gray-600 mt-1.5">Τα στοιχεία χρησιμοποιούνται μόνο κατά τη διάρκεια της πληρωμής και δεν αποθηκεύονται.</p>
              </div>

              <!-- Start button -->
              <button
                @click="startPayment"
                :disabled="!canStart"
                class="w-full py-3.5 rounded-xl font-bold text-sm transition-all"
                :class="canStart
                  ? 'bg-blue-600 hover:bg-blue-500 text-white'
                  : 'bg-white/[0.04] text-gray-600 cursor-not-allowed'"
              >
                Ξεκίνα αυτόματη πληρωμή
              </button>
            </template>

            <!-- Running phase -->
            <template v-else-if="phase === 'running' || phase === 'otp'">

              <!-- Status log -->
              <div class="space-y-2">
                <div
                  v-for="(event, i) in eventLog" :key="i"
                  class="flex items-start gap-2.5 text-sm"
                >
                  <span class="mt-0.5 text-base shrink-0">{{ eventIcon(event.type) }}</span>
                  <span :class="eventColor(event.type)">{{ event.message }}</span>
                </div>

                <!-- Spinner for running -->
                <div v-if="phase === 'running'" class="flex items-center gap-2 text-sm text-gray-500 pt-1">
                  <div class="w-3 h-3 rounded-full border-2 border-blue-500 border-t-transparent animate-spin shrink-0"></div>
                  <span>Ο agent εργάζεται...</span>
                </div>
              </div>

              <!-- OTP input -->
              <div v-if="phase === 'otp'" class="bg-amber-500/10 border border-amber-500/30 rounded-xl p-4">
                <p class="text-sm font-bold text-amber-300 mb-1">Απαιτείται OTP</p>
                <p class="text-xs text-amber-400/70 mb-3">Εισάγετε τον κωδικό που λάβατε στο κινητό σας.</p>
                <div class="flex gap-2">
                  <input
                    v-model="otpInput"
                    placeholder="000000"
                    maxlength="8"
                    class="flex-1 px-3 py-2.5 rounded-xl bg-white/[0.06] border border-white/[0.1] text-white text-center text-lg font-mono tracking-widest focus:outline-none focus:border-amber-500/50"
                    @keyup.enter="submitOTP"
                  />
                  <button
                    @click="submitOTP"
                    :disabled="otpInput.length < 4"
                    class="px-4 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-400 text-black font-bold text-sm disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
                  >
                    Αποστολή
                  </button>
                </div>
              </div>
            </template>

            <!-- Result phases -->
            <template v-else-if="phase === 'success'">
              <div class="text-center py-4">
                <div class="w-16 h-16 rounded-full bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center mx-auto mb-4">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-8 h-8 text-emerald-400">
                    <path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
                  </svg>
                </div>
                <p class="text-lg font-bold text-white mb-1">Πληρώθηκε!</p>
                <p class="text-sm text-gray-400">{{ successRef || 'Η πληρωμή ολοκληρώθηκε επιτυχώς.' }}</p>
                <button @click="close" class="mt-5 px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-sm transition-colors">
                  Κλείσιμο
                </button>
              </div>
            </template>

            <template v-else-if="phase === 'error'">
              <div class="text-center py-4">
                <div class="w-16 h-16 rounded-full bg-red-500/15 border border-red-500/30 flex items-center justify-center mx-auto mb-4">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-8 h-8 text-red-400">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
                  </svg>
                </div>
                <p class="text-lg font-bold text-white mb-1">Αποτυχία</p>
                <p class="text-sm text-gray-400">{{ errorMessage }}</p>
                <div class="flex gap-2 justify-center mt-5">
                  <button @click="resetToSetup" class="px-5 py-2.5 rounded-xl border border-white/[0.1] text-gray-300 font-semibold text-sm hover:bg-white/[0.05] transition-colors">
                    Ξαναπροσπάθησε
                  </button>
                  <button @click="close" class="px-5 py-2.5 rounded-xl bg-white/[0.06] text-gray-300 font-semibold text-sm hover:bg-white/[0.1] transition-colors">
                    Κλείσιμο
                  </button>
                </div>
              </div>
            </template>

          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { auth } from '@/firebase'
import api from '@/api/client'
import type { Bill } from '@/types'

const props = defineProps<{
  modelValue: boolean
  bill: Bill | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'paid', billId: string): void
}>()

function formatAmount(v: number): string {
  return new Intl.NumberFormat('el-GR', { style: 'currency', currency: 'EUR' }).format(v)
}

type Phase = 'setup' | 'running' | 'otp' | 'success' | 'error'

const phase = ref<Phase>('setup')
const credentials = ref<{ id: string; label: string; url: string }[]>([])
const selectedCredId = ref('')
const cardNumber = ref('')
const cardExpiry = ref('')
const cardCVV = ref('')
const cardHolder = ref('')
const eventLog = ref<{ type: string; message: string }[]>([])
const otpInput = ref('')
const successRef = ref('')
const errorMessage = ref('')
const currentJobId = ref('')
const isRunning = computed(() => phase.value === 'running' || phase.value === 'otp')

let eventSource: EventSource | null = null

const canStart = computed(() => {
  return selectedCredId.value !== '' || props.bill?.payment_code
})

watch(() => props.modelValue, async (v) => {
  if (v) {
    resetToSetup()
    await loadCredentials()
  } else {
    cleanup()
  }
})

async function loadCredentials() {
  try {
    const { data } = await api.get('/agent/credentials')
    credentials.value = data
  } catch {
    credentials.value = []
  }
}

function formatCardNumber(e: Event) {
  const input = e.target as HTMLInputElement
  const v = input.value.replace(/\D/g, '').substring(0, 16)
  cardNumber.value = v.replace(/(.{4})/g, '$1 ').trim()
}

async function startPayment() {
  if (!props.bill) return

  phase.value = 'running'
  eventLog.value = []
  currentJobId.value = ''

  const body = {
    bill_id: props.bill.id,
    credential_id: selectedCredId.value,
    card_number: cardNumber.value.replace(/\s/g, ''),
    card_expiry: cardExpiry.value,
    card_cvv: cardCVV.value,
    card_holder: cardHolder.value.toUpperCase(),
  }

  const user = auth.currentUser
  const token = user ? await user.getIdToken() : ''
  const baseURL = (import.meta.env.VITE_API_URL as string | undefined) ?? ''
  const url = `${baseURL}/api/agent/pay/${props.bill.id}`

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  })

  if (!response.ok || !response.body) {
    phase.value = 'error'
    errorMessage.value = 'Αδύνατη η σύνδεση με τον agent.'
    return
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    const text = decoder.decode(value)
    const lines = text.split('\n')

    for (const line of lines) {
      if (!line.startsWith('data: ')) continue
      try {
        const event = JSON.parse(line.slice(6))
        handleAgentEvent(event)
      } catch {}
    }
  }
}

function handleAgentEvent(event: Record<string, any>) {
  const type = event.type as string
  const message = event.message as string

  if (event.job_id && !currentJobId.value) {
    currentJobId.value = event.job_id
  }

  if (['started', 'navigating', 'agent_message', 'info', 'action'].includes(type)) {
    if (message) eventLog.value.push({ type, message })
  }

  if (type === 'otp_required') {
    phase.value = 'otp'
    eventLog.value.push({ type, message: message || 'Απαιτείται OTP' })
  }

  if (type === 'otp_received') {
    phase.value = 'running'
    otpInput.value = ''
  }

  if (type === 'success') {
    phase.value = 'success'
    successRef.value = event.reference || message || ''
    if (props.bill) emit('paid', props.bill.id)
  }

  if (type === 'error') {
    phase.value = 'error'
    errorMessage.value = message || 'Άγνωστο σφάλμα'
  }

  if (type === 'done' && phase.value === 'running') {
    phase.value = 'error'
    errorMessage.value = 'Ο agent τερμάτισε χωρίς επιβεβαίωση.'
  }
}

async function submitOTP() {
  if (!otpInput.value || !currentJobId.value) return
  phase.value = 'running'
  try {
    await api.post(`/agent/otp/${currentJobId.value}`, { otp: otpInput.value })
  } catch (e) {
    console.error('OTP submit failed', e)
  }
  otpInput.value = ''
}

function eventIcon(type: string): string {
  const icons: Record<string, string> = {
    started: '🚀',
    navigating: '🌐',
    agent_message: '🤖',
    action: '👆',
    otp_required: '📱',
    otp_received: '✅',
    info: 'ℹ️',
    success: '✅',
    error: '❌',
  }
  return icons[type] ?? '·'
}

function eventColor(type: string): string {
  if (type === 'success') return 'text-emerald-400'
  if (type === 'error') return 'text-red-400'
  if (type === 'otp_required') return 'text-amber-400'
  if (type === 'agent_message') return 'text-blue-300'
  return 'text-gray-400'
}

function resetToSetup() {
  phase.value = 'setup'
  eventLog.value = []
  otpInput.value = ''
  successRef.value = ''
  errorMessage.value = ''
  currentJobId.value = ''
  cleanup()
}

function cleanup() {
  eventSource?.close()
  eventSource = null
}

function close() {
  if (isRunning.value) return
  emit('update:modelValue', false)
}

onUnmounted(cleanup)
</script>

<style scoped>
.modal-fade-enter-active, .modal-fade-leave-active { transition: opacity 0.2s ease; }
.modal-fade-enter-from, .modal-fade-leave-to { opacity: 0; }
</style>
