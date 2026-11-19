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
              <p v-if="activeBill" class="text-xs text-gray-400 truncate">{{ activeBill.user_provider?.nickname || activeBill.user_provider?.provider?.name }} — {{ formatAmount(activeBill.amount) }}</p>
              <p v-else class="text-xs text-gray-400">Επίλεξε λογαριασμό</p>
            </div>
            <button v-if="!isRunning" @click="close" class="w-8 h-8 flex items-center justify-center text-gray-500 hover:text-white rounded-xl hover:bg-white/[0.06] transition-colors">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" /></svg>
            </button>
          </div>

          <div class="p-5 space-y-4 max-h-[70vh] overflow-y-auto">

            <!-- Bill picker (when no bill pre-selected) -->
            <template v-if="phase === 'pick_bill'">
              <div v-if="pickerLoading" class="space-y-2">
                <div v-for="i in 3" :key="i" class="h-14 rounded-xl bg-white/[0.04] animate-pulse" />
              </div>
              <div v-else-if="pickerBills.length === 0" class="text-center py-6 text-sm text-gray-500">
                Δεν βρέθηκαν λογαριασμοί.
              </div>
              <div v-else class="space-y-1.5">
                <button
                  v-for="b in pickerBills" :key="b.id"
                  @click="pickBill(b)"
                  class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl border border-white/[0.07] hover:border-white/[0.15] hover:bg-white/[0.04] transition-colors text-left"
                >
                  <div class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0 text-xs font-black text-white"
                    :style="{ backgroundColor: b.user_provider?.provider?.color || '#6366f1' }">
                    {{ (b.user_provider?.nickname || b.user_provider?.provider?.name || '?').charAt(0).toUpperCase() }}
                  </div>
                  <div class="flex-1 min-w-0">
                    <p class="text-sm font-semibold text-gray-100 truncate">{{ b.user_provider?.nickname || b.user_provider?.provider?.name }}</p>
                    <p class="text-xs text-gray-500">{{ formatDate(b.due_date) }}</p>
                  </div>
                  <div class="flex items-center gap-2 shrink-0">
                    <span class="text-sm font-bold"
                      :class="b.status === 'overdue' ? 'text-red-400' : b.status === 'paid' ? 'text-gray-500' : 'text-white'">
                      {{ formatAmount(b.amount) }}
                    </span>
                    <span class="text-[10px] px-1.5 py-0.5 rounded-full font-semibold"
                      :class="b.status === 'overdue' ? 'bg-red-500/15 text-red-400' : b.status === 'paid' ? 'bg-white/[0.06] text-gray-500' : 'bg-amber-500/15 text-amber-400'">
                      {{ b.status === 'overdue' ? 'Ληξ.' : b.status === 'paid' ? 'Πλ.' : 'Εκκρ.' }}
                    </span>
                  </div>
                </button>
              </div>
            </template>

            <!-- Setup form (before start) -->
            <template v-else-if="phase === 'setup'">

              <!-- Credential selector -->
              <div>
                <label class="block text-xs font-bold text-gray-500 uppercase tracking-widest mb-2">Κωδικοί σύνδεσης</label>
                <div v-if="credentials.length === 0" class="text-xs text-gray-500 py-2">
                  Δεν βρέθηκαν αποθηκευμένοι κωδικοί.
                  <button @click="emit('update:modelValue', false); router.push({ name: 'credentials' })" class="text-blue-400 ml-1 hover:underline">Πρόσθεσε →</button>
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

              <!-- Card selector -->
              <div>
                <label class="block text-xs font-bold text-gray-500 uppercase tracking-widest mb-2">
                  Κάρτα πληρωμής
                  <span class="text-gray-600 font-normal normal-case ml-1">(προαιρετικό αν υπάρχει RF)</span>
                </label>
                <div v-if="cards.length === 0" class="text-xs text-gray-500 py-2">
                  Δεν βρέθηκαν κάρτες.
                  <button @click="emit('update:modelValue', false); router.push({ name: 'cards' })" class="text-blue-400 ml-1 hover:underline">Πρόσθεσε →</button>
                </div>
                <div v-else class="space-y-1.5">
                  <button
                    v-for="card in cards" :key="card.id"
                    @click="selectedCardId = selectedCardId === card.id ? '' : card.id"
                    class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl border transition-colors text-left"
                    :class="selectedCardId === card.id
                      ? 'bg-indigo-600/15 border-indigo-500/40 text-white'
                      : 'border-white/[0.07] text-gray-300 hover:border-white/[0.15] hover:bg-white/[0.04]'"
                  >
                    <div class="w-7 h-7 rounded-lg flex items-center justify-center shrink-0 text-xs font-black text-white"
                      :style="{ backgroundColor: card.color || '#6366f1' }">
                      {{ card.card_type === 'visa' ? 'V' : card.card_type === 'mastercard' ? 'M' : '💳' }}
                    </div>
                    <div class="flex-1 min-w-0">
                      <span class="text-sm font-medium block truncate">{{ card.nickname }}</span>
                      <span class="text-xs text-gray-500">•••• {{ card.last_four }}</span>
                    </div>
                    <span v-if="!card.has_secure_details" class="text-[10px] text-amber-500/80 shrink-0">χωρίς πλήρη στοιχεία</span>
                    <span v-else class="text-[10px] text-emerald-500/80 shrink-0">✓ έτοιμη</span>
                  </button>
                  <p v-if="selectedCardId && !selectedCard?.has_secure_details" class="text-[10px] text-amber-400 mt-1">
                    Η κάρτα δεν έχει αποθηκευμένα πλήρη στοιχεία.
                    <button @click="emit('update:modelValue', false); router.push({ name: 'cards' })" class="text-amber-300 underline">Πρόσθεσε τα →</button>
                  </p>
                </div>
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

              <!-- OTP / user input -->
              <div v-if="phase === 'otp'" class="bg-amber-500/10 border border-amber-500/30 rounded-xl p-4">
                <p class="text-sm font-bold text-amber-300 mb-1">Απαιτείται εισαγωγή</p>
                <p class="text-xs text-amber-400/70 mb-3">{{ inputPrompt }}</p>
                <div class="flex gap-2">
                  <input
                    v-model="otpInput"
                    placeholder="Απάντηση..."
                    maxlength="20"
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
import { useRouter } from 'vue-router'
import { auth } from '@/firebase'
import api from '@/api/client'
import type { Bill } from '@/types'

const router = useRouter()

const props = defineProps<{
  modelValue: boolean
  bill?: Bill | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'paid', billId: string): void
}>()

function formatAmount(v: number): string {
  return new Intl.NumberFormat('el-GR', { style: 'currency', currency: 'EUR' }).format(v)
}

function formatDate(d: string): string {
  return new Date(d).toLocaleDateString('el-GR', { day: 'numeric', month: 'short' })
}

type Phase = 'pick_bill' | 'setup' | 'running' | 'otp' | 'success' | 'error'
type CardSummary = { id: string; nickname: string; last_four: string; card_type: string; color: string; has_secure_details: boolean; is_default: boolean }

const phase = ref<Phase>('setup')
const credentials = ref<{ id: string; label: string; url: string }[]>([])
const cards = ref<CardSummary[]>([])
const selectedCredId = ref('')
const selectedCardId = ref('')
const eventLog = ref<{ type: string; message: string }[]>([])
const otpInput = ref('')
const successRef = ref('')
const errorMessage = ref('')
const currentJobId = ref('')
const isRunning = computed(() => phase.value === 'running' || phase.value === 'otp')
const inputPrompt = ref('Εισάγετε τον OTP που λάβατε στο κινητό σας.')
const selectedCard = computed(() => cards.value.find(c => c.id === selectedCardId.value) ?? null)

const internalBill = ref<Bill | null>(null)
const activeBill = computed(() => props.bill ?? internalBill.value)

const pickerBills = ref<Bill[]>([])
const pickerLoading = ref(false)

let eventSource: EventSource | null = null

const canStart = computed(() => {
  return selectedCredId.value !== '' || activeBill.value?.payment_code
})

watch(() => props.modelValue, async (v) => {
  if (v) {
    internalBill.value = null
    if (!props.bill) {
      phase.value = 'pick_bill'
      pickerLoading.value = true
      await Promise.all([loadCredentials(), loadCards(), fetchPickerBills()])
      pickerLoading.value = false
    } else {
      resetToSetup()
      await Promise.all([loadCredentials(), loadCards()])
    }
  } else {
    cleanup()
  }
})

async function fetchPickerBills() {
  try {
    const { data } = await api.get('/bills')
    const order: Record<string, number> = { overdue: 0, pending: 1, paid: 2 }
    pickerBills.value = [...data].sort((a: Bill, b: Bill) =>
      (order[a.status] ?? 1) - (order[b.status] ?? 1)
    )
  } catch {
    pickerBills.value = []
  }
}

function pickBill(bill: Bill) {
  internalBill.value = bill
  phase.value = 'setup'
}

async function loadCredentials() {
  try {
    const { data } = await api.get('/agent/credentials')
    credentials.value = data
  } catch {
    credentials.value = []
  }
}

async function loadCards() {
  try {
    const { data } = await api.get('/cards')
    cards.value = data
    // Auto-select default card
    const def = data.find((c: CardSummary) => c.is_default)
    if (def) selectedCardId.value = def.id
  } catch {
    cards.value = []
  }
}

async function startPayment() {
  if (!activeBill.value) return

  phase.value = 'running'
  eventLog.value = []
  currentJobId.value = ''

  const body = {
    credential_id: selectedCredId.value,
    card_id: selectedCardId.value,
  }

  const user = auth.currentUser
  const token = user ? await user.getIdToken() : ''
  const baseURL = (import.meta.env.VITE_API_URL as string | undefined) ?? ''
  const url = `${baseURL}/api/agent/pay/${activeBill.value.id}`

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
    inputPrompt.value = message || 'Εισάγετε τον OTP που λάβατε στο κινητό σας.'
    eventLog.value.push({ type, message: message || 'Απαιτείται OTP' })
  }

  if (type === 'user_input_required') {
    phase.value = 'otp'
    inputPrompt.value = message || 'Απαιτείται εισαγωγή στοιχείου από τον χρήστη.'
    eventLog.value.push({ type: 'otp_required', message: message || 'Απαιτείται εισαγωγή' })
  }

  if (type === 'otp_received') {
    phase.value = 'running'
    otpInput.value = ''
  }

  if (type === 'success') {
    phase.value = 'success'
    successRef.value = event.reference || message || ''
    if (activeBill.value) emit('paid', activeBill.value.id)
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
  phase.value = activeBill.value ? 'setup' : 'pick_bill'
  eventLog.value = []
  otpInput.value = ''
  successRef.value = ''
  errorMessage.value = ''
  currentJobId.value = ''
  selectedCredId.value = ''
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
