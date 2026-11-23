<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useCardsStore } from '@/stores/cards'
import { useLocale } from '@/composables/useLocale'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import api from '@/api/client'
import type { Card } from '@/types'

const store = useCardsStore()
const { locale } = useLocale()
const { toast } = useToast()
const { confirm } = useConfirm()

const showModal = ref(false)
const saving = ref(false)
const editingId = ref<string | null>(null)

const CARD_COLORS = ['#6366f1', '#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6']

type CardType = 'visa' | 'mastercard' | 'amex' | 'other'
const form = ref({
  nickname: '',
  last_four: '',
  expiry_mm: new Date().getMonth() + 1,
  expiry_yy: new Date().getFullYear(),
  card_type: 'visa' as CardType,
  is_default: false,
  color: CARD_COLORS[0]!,
  // AI payment details (optional, stored encrypted)
  aiNumber: '',
  aiCVV: '',
  aiCardholder: '',
  aiEnabled: false,
})

const isExpired = (card: Card) => {
  const now = new Date()
  return card.expiry_yy < now.getFullYear() ||
    (card.expiry_yy === now.getFullYear() && card.expiry_mm < now.getMonth() + 1)
}

const cardTypeLabel = (type: CardType) =>
  type === 'visa' ? 'Visa' :
  type === 'mastercard' ? 'Mastercard' :
  type === 'amex' ? 'Amex' : 'Άλλη'

function openCreate() {
  editingId.value = null
  form.value = {
    nickname: '',
    last_four: '',
    expiry_mm: new Date().getMonth() + 1,
    expiry_yy: new Date().getFullYear() + 2,
    card_type: 'visa',
    is_default: store.cards.length === 0,
    color: CARD_COLORS[store.cards.length % CARD_COLORS.length]!,
    aiNumber: '',
    aiCVV: '',
    aiCardholder: '',
    aiEnabled: false,
  }
  showModal.value = true
}

function openEdit(card: Card) {
  editingId.value = card.id
  form.value = {
    nickname: card.nickname,
    last_four: card.last_four,
    expiry_mm: card.expiry_mm,
    expiry_yy: card.expiry_yy,
    card_type: card.card_type as CardType,
    is_default: card.is_default,
    color: card.color || CARD_COLORS[0]!,
    aiNumber: '',
    aiCVV: '',
    aiCardholder: '',
    aiEnabled: false,
  }
  showModal.value = true
}

async function submit() {
  const lastFour = form.value.aiNumber
    ? form.value.aiNumber.replace(/\s/g, '').slice(-4)
    : form.value.last_four
  if (!form.value.nickname || lastFour.length !== 4) return
  saving.value = true
  try {
    let cardId = editingId.value
    if (cardId) {
      await store.updateCard(cardId, {
        nickname: form.value.nickname,
        expiry_mm: form.value.expiry_mm,
        expiry_yy: form.value.expiry_yy,
        card_type: form.value.card_type,
        is_default: form.value.is_default,
        color: form.value.color,
      })
    } else {
      const created = await store.createCard({
        nickname: form.value.nickname,
        last_four: lastFour,
        expiry_mm: form.value.expiry_mm,
        expiry_yy: form.value.expiry_yy,
        card_type: form.value.card_type,
        is_default: form.value.is_default,
        color: form.value.color,
      })
      cardId = created.id
    }
    // Save AI payment details if provided
    const rawNum = form.value.aiNumber.replace(/\s/g, '')
    if (cardId && rawNum.length >= 15 && form.value.aiCVV) {
      const expiry = `${String(form.value.expiry_mm).padStart(2, '0')}/${String(form.value.expiry_yy).slice(-2)}`
      await api.post(`/cards/${cardId}/secure`, {
        number: rawNum,
        expiry,
        cvv: form.value.aiCVV,
        cardholder_name: form.value.aiCardholder.toUpperCase(),
      })
      await store.fetchCards()
      toast(locale.value === 'el' ? 'Κάρτα & AI στοιχεία αποθηκεύτηκαν ✓' : 'Card & AI details saved ✓', 'success')
    } else {
      toast(locale.value === 'el' ? (editingId.value ? 'Κάρτα αποθηκεύτηκε' : 'Κάρτα προστέθηκε') : (editingId.value ? 'Card saved' : 'Card added'), 'success')
    }
    showModal.value = false
  } catch {
    toast(locale.value === 'el' ? 'Αποτυχία αποθήκευσης' : 'Failed to save', 'error')
  } finally {
    saving.value = false
  }
}

async function deleteCard(card: Card) {
  const ok = await confirm(
    locale.value === 'el'
      ? `Διαγραφή κάρτας "${card.nickname}";`
      : `Delete card "${card.nickname}"?`,
  )
  if (!ok) return
  await store.deleteCard(card.id)
  toast(locale.value === 'el' ? 'Κάρτα διαγράφηκε' : 'Card deleted', 'success')
}

async function setDefault(card: Card) {
  await store.updateCard(card.id, { is_default: true })
  toast(locale.value === 'el' ? 'Προεπιλεγμένη κάρτα ορίστηκε' : 'Default card set', 'success')
}

// Secure card storage for AI agent
const secureCardId = ref<string | null>(null)
const secureForm = ref({ number: '', expiry: '', cvv: '', cardholder_name: '' })
const secureSaving = ref(false)

async function toggleSecure(card: Card) {
  if ((card as any).has_secure_details) {
    const ok = await confirm(locale.value === 'el'
      ? 'Διαγραφή αποθηκευμένων στοιχείων κάρτας για AI agent;'
      : 'Remove stored secure card details for AI agent?')
    if (!ok) return
    await api.delete(`/cards/${card.id}/secure`)
    await store.fetchCards()
    toast(locale.value === 'el' ? 'Στοιχεία διαγράφηκαν' : 'Secure details removed', 'success')
  } else {
    secureCardId.value = card.id
    secureForm.value = { number: '', expiry: '', cvv: '', cardholder_name: '' }
  }
}

async function saveSecure() {
  if (!secureCardId.value) return
  secureSaving.value = true
  try {
    await api.post(`/cards/${secureCardId.value}/secure`, {
      number: secureForm.value.number.replace(/\s/g, ''),
      expiry: secureForm.value.expiry,
      cvv: secureForm.value.cvv,
      cardholder_name: secureForm.value.cardholder_name.toUpperCase(),
    })
    await store.fetchCards()
    secureCardId.value = null
    toast(locale.value === 'el' ? 'Στοιχεία αποθηκεύτηκαν κρυπτογραφημένα ✓' : 'Secure details saved ✓', 'success')
  } catch {
    toast(locale.value === 'el' ? 'Σφάλμα αποθήκευσης' : 'Save failed', 'error')
  } finally {
    secureSaving.value = false
  }
}

function formatSecureCardNumber(e: Event) {
  const v = (e.target as HTMLInputElement).value.replace(/\D/g, '').substring(0, 16)
  secureForm.value.number = v.replace(/(.{4})/g, '$1 ').trim()
}

function formatAiNumber(e: Event) {
  const v = (e.target as HTMLInputElement).value.replace(/\D/g, '').substring(0, 16)
  form.value.aiNumber = v.replace(/(.{4})/g, '$1 ').trim()
  // Auto-detect card type from first digit
  const d = v.charAt(0)
  if (d === '4') form.value.card_type = 'visa'
  else if (d === '5') form.value.card_type = 'mastercard'
  else if (d === '3') form.value.card_type = 'amex'
}

onMounted(() => store.fetchCards())

const months = Array.from({ length: 12 }, (_, i) => i + 1)
const years = Array.from({ length: 15 }, (_, i) => new Date().getFullYear() + i)
</script>

<template>
  <div class="max-w-2xl mx-auto px-4 pt-6 pb-20">

    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-black text-gray-50 tracking-tight">
          {{ locale === 'el' ? 'Κάρτες Πληρωμής' : 'Payment Cards' }}
        </h1>
        <p class="text-sm text-gray-500 mt-0.5">
          {{ locale === 'el'
            ? 'Διαχείριση καρτών για τους λογαριασμούς σου'
            : 'Manage cards for your bills' }}
        </p>
      </div>
      <button
        @click="openCreate"
        class="flex items-center gap-2 px-4 py-2 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white text-sm font-bold transition-all shadow-sm shadow-blue-900/40"
      >
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" class="w-4 h-4">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        {{ locale === 'el' ? 'Προσθήκη' : 'Add card' }}
      </button>
    </div>

    <!-- Notice -->
    <div class="mb-5 px-4 py-3 rounded-xl bg-blue-500/8 border border-blue-500/20 flex gap-3">
      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-5 h-5 text-blue-400 shrink-0 mt-0.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z" />
      </svg>
      <div>
        <p class="text-xs font-medium text-blue-300">
          {{ locale === 'el' ? 'Ασφάλεια & AI Πληρωμές' : 'Security & AI Payments' }}
        </p>
        <p class="text-xs text-blue-400/80 mt-0.5">
          {{ locale === 'el'
            ? 'Μπορείς να αποθηκεύσεις τον πλήρη αριθμό κάρτας κρυπτογραφημένο (AES-256) ώστε ο AI agent να κάνει αυτόματες πληρωμές. Δεν εμφανίζεται ποτέ στην οθόνη.'
            : 'You can optionally store the full card number encrypted (AES-256) so the AI agent can make automatic payments. It is never shown on screen.' }}
        </p>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="!store.loading && store.cards.length === 0" class="text-center py-16">
      <div class="w-14 h-14 rounded-2xl bg-white/[0.04] border border-white/[0.06] flex items-center justify-center mx-auto mb-4">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-7 h-7 text-gray-600">
          <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 0 0 2.25-2.25V6.75A2.25 2.25 0 0 0 19.5 4.5h-15a2.25 2.25 0 0 0-2.25 2.25v10.5A2.25 2.25 0 0 0 4.5 19.5Z" />
        </svg>
      </div>
      <p class="text-sm text-gray-500">
        {{ locale === 'el' ? 'Δεν έχεις προσθέσει κάρτες ακόμα.' : 'No cards added yet.' }}
      </p>
    </div>

    <!-- Card list -->
    <div v-else class="space-y-3">
      <div
        v-for="card in store.cards"
        :key="card.id"
        class="relative overflow-hidden rounded-2xl border"
        :class="isExpired(card) ? 'border-red-500/20 opacity-60' : 'border-white/[0.08]'"
      >
        <!-- Card visual -->
        <div class="relative h-32 p-4 flex flex-col justify-between" :style="{ background: `linear-gradient(135deg, ${card.color}22 0%, ${card.color}08 100%)` }">
          <!-- Background stripe -->
          <div class="absolute inset-0 opacity-[0.04]" style="background: repeating-linear-gradient(-45deg, #fff, #fff 1px, transparent 1px, transparent 8px)" />

          <div class="flex items-start justify-between relative">
            <div>
              <p class="text-xs font-medium uppercase tracking-widest" :style="{ color: card.color }">
                {{ cardTypeLabel(card.card_type as CardType) }}
              </p>
              <p class="text-sm font-semibold text-gray-200 mt-0.5">{{ card.nickname }}</p>
            </div>
            <div class="flex items-center gap-1.5">
              <span v-if="card.is_default" class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full bg-blue-500/20 text-blue-300 border border-blue-500/30">
                {{ locale === 'el' ? 'Προεπιλογή' : 'Default' }}
              </span>
              <span v-if="isExpired(card)" class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full bg-red-500/20 text-red-400 border border-red-500/30">
                {{ locale === 'el' ? 'Ληγμένη' : 'Expired' }}
              </span>
            </div>
          </div>

          <div class="flex items-end justify-between relative">
            <p class="text-base font-mono text-gray-300 tracking-widest">•••• •••• •••• {{ card.last_four }}</p>
            <p class="text-xs text-gray-500">
              {{ String(card.expiry_mm).padStart(2, '0') }}/{{ String(card.expiry_yy).slice(-2) }}
            </p>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex items-center gap-2 px-4 py-3 border-t border-white/[0.05] bg-white/[0.015]">
          <button
            v-if="!card.is_default"
            @click="setDefault(card)"
            class="text-xs text-gray-500 hover:text-blue-400 transition-colors"
          >
            {{ locale === 'el' ? 'Ορισμός ως προεπιλογή' : 'Set as default' }}
          </button>
          <!-- AI Agent secure storage toggle -->
          <button
            @click="toggleSecure(card)"
            class="text-xs transition-colors flex items-center gap-1"
            :class="card.has_secure_details
              ? 'text-emerald-500 hover:text-emerald-400'
              : 'text-gray-600 hover:text-blue-400'"
            :title="card.has_secure_details ? 'Αποθηκευμένη για agent' : 'Αποθήκευσε πλήρη στοιχεία για AI agent'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-3.5 h-3.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09Z" />
            </svg>
            {{ card.has_secure_details ? 'AI ✓' : 'AI' }}
          </button>
          <div class="flex-1" />
          <button
            @click="openEdit(card)"
            class="w-8 h-8 flex items-center justify-center rounded-lg text-gray-500 hover:text-gray-300 hover:bg-white/[0.06] transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Z" />
            </svg>
          </button>
          <button
            @click="deleteCard(card)"
            class="w-8 h-8 flex items-center justify-center rounded-lg text-gray-500 hover:text-red-400 hover:bg-red-500/8 transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4">
              <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
            </svg>
          </button>
        </div>
      </div>
    </div>

  </div>

  <!-- Add/Edit modal -->
  <div v-if="showModal" class="fixed inset-0 z-50 flex items-end sm:items-center justify-center px-4 pb-4 sm:pb-0" @click.self="showModal = false">
    <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="showModal = false" />
    <div class="relative z-10 w-full max-w-sm bg-[#13131e] rounded-2xl border border-white/[0.08] shadow-2xl p-5">
      <h3 class="text-sm font-semibold text-gray-50 mb-5">
        {{ editingId
          ? (locale === 'el' ? 'Επεξεργασία κάρτας' : 'Edit card')
          : (locale === 'el' ? 'Προσθήκη κάρτας' : 'Add card') }}
      </h3>

      <div class="space-y-4">
        <!-- Nickname -->
        <div>
          <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ locale === 'el' ? 'Όνομα' : 'Name' }}</label>
          <input v-model="form.nickname" type="text" :placeholder="locale === 'el' ? 'π.χ. Eurobank Visa' : 'e.g. Eurobank Visa'"
            class="w-full px-3 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.03] text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>

        <!-- Card number (full or last 4) -->
        <div v-if="!editingId">
          <label class="block text-xs font-medium text-gray-400 mb-1.5">
            {{ locale === 'el' ? 'Αριθμός κάρτας' : 'Card number' }}
          </label>
          <input
            :value="form.aiNumber"
            @input="formatAiNumber"
            type="text" inputmode="numeric" maxlength="19"
            :placeholder="locale === 'el' ? '1234 5678 9012 3456' : '1234 5678 9012 3456'"
            class="w-full px-3 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.03] text-sm font-mono tracking-widest text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p class="text-[11px] text-gray-600 mt-1">{{ locale === 'el' ? 'Αποθηκεύεται κρυπτογραφημένος για AI πληρωμές. Τα τελευταία 4 ψηφία εξάγονται αυτόματα.' : 'Stored encrypted for AI payments. Last 4 digits extracted automatically.' }}</p>
        </div>

        <!-- Last four (edit mode only) + type row -->
        <div class="grid gap-3" :class="editingId ? 'grid-cols-2' : 'grid-cols-1'">
          <div v-if="editingId">
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ locale === 'el' ? 'Τελευταία 4 ψηφία' : 'Last 4 digits' }}</label>
            <input v-model="form.last_four" type="text" maxlength="4" inputmode="numeric" :placeholder="locale === 'el' ? 'π.χ. 1234' : 'e.g. 1234'"
              disabled
              class="w-full px-3 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.03] text-sm font-mono text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ locale === 'el' ? 'Τύπος' : 'Type' }}</label>
            <select v-model="form.card_type"
              class="w-full px-3 py-2.5 rounded-xl border border-white/[0.08] bg-[#13131e] text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option value="visa">Visa</option>
              <option value="mastercard">Mastercard</option>
              <option value="amex">Amex</option>
              <option value="other">{{ locale === 'el' ? 'Άλλη' : 'Other' }}</option>
            </select>
          </div>
        </div>

        <!-- CVV + Cardholder (shown when number entered) -->
        <div v-if="!editingId && form.aiNumber.replace(/\s/g,'').length >= 15" class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">CVV</label>
            <input v-model="form.aiCVV" type="password" maxlength="4" placeholder="•••"
              class="w-full px-3 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.03] text-sm font-mono text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ locale === 'el' ? 'Όνομα κατόχου' : 'Cardholder' }}</label>
            <input v-model="form.aiCardholder" type="text" maxlength="30" placeholder="FULL NAME"
              class="w-full px-3 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.03] text-sm uppercase font-mono text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
        </div>

        <!-- Expiry -->
        <div>
          <label class="block text-xs font-medium text-gray-400 mb-1.5">{{ locale === 'el' ? 'Ημερομηνία λήξης' : 'Expiry' }}</label>
          <div class="grid grid-cols-2 gap-3">
            <select v-model.number="form.expiry_mm"
              class="w-full px-3 py-2.5 rounded-xl border border-white/[0.08] bg-[#13131e] text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option v-for="m in months" :key="m" :value="m">{{ String(m).padStart(2, '0') }}</option>
            </select>
            <select v-model.number="form.expiry_yy"
              class="w-full px-3 py-2.5 rounded-xl border border-white/[0.08] bg-[#13131e] text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option v-for="y in years" :key="y" :value="y">{{ y }}</option>
            </select>
          </div>
        </div>

        <!-- Color picker -->
        <div>
          <label class="block text-xs font-medium text-gray-400 mb-2">{{ locale === 'el' ? 'Χρώμα' : 'Color' }}</label>
          <div class="flex gap-2 flex-wrap">
            <button v-for="c in CARD_COLORS" :key="c" @click="form.color = c"
              class="w-7 h-7 rounded-lg transition-all border-2"
              :style="{ backgroundColor: c }"
              :class="form.color === c ? 'border-white scale-110' : 'border-transparent opacity-60 hover:opacity-100'" />
          </div>
        </div>

        <!-- Default toggle -->
        <label class="flex items-center gap-3 cursor-pointer">
          <div class="relative w-9 h-5">
            <input type="checkbox" v-model="form.is_default" class="sr-only" />
            <div class="w-9 h-5 rounded-full transition-colors" :class="form.is_default ? 'bg-blue-600' : 'bg-white/[0.1]'" />
            <div class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform" :class="form.is_default ? 'translate-x-4' : ''" />
          </div>
          <span class="text-sm text-gray-300">{{ locale === 'el' ? 'Προεπιλεγμένη κάρτα' : 'Set as default card' }}</span>
        </label>
      </div>

      <!-- Actions -->
      <div class="flex gap-3 mt-6">
        <button @click="showModal = false" class="flex-1 py-2.5 rounded-xl border border-white/[0.08] bg-white/[0.03] text-sm text-gray-400 hover:text-gray-200 transition-colors">
          {{ locale === 'el' ? 'Ακύρωση' : 'Cancel' }}
        </button>
        <button @click="submit" :disabled="saving || !form.nickname || (editingId ? false : form.aiNumber.replace(/\s/g,'').length < 15 && form.last_four.length !== 4)"
          class="flex-1 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-40 text-white text-sm font-semibold transition-colors">
          {{ saving ? '...' : (locale === 'el' ? 'Αποθήκευση' : 'Save') }}
        </button>
      </div>
    </div>
  </div>

  <!-- Secure card details modal for AI agent -->
  <div v-if="secureCardId" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0" @click.self="secureCardId = null">
    <div class="bg-[#111119] border border-white/[0.08] rounded-2xl shadow-2xl w-full max-w-sm p-5">
      <div class="flex items-center gap-3 mb-4">
        <div class="w-9 h-9 rounded-xl bg-blue-600/15 border border-blue-500/30 flex items-center justify-center shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.75" stroke="currentColor" class="w-4 h-4 text-blue-400">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09Z" />
          </svg>
        </div>
        <div>
          <p class="text-sm font-bold text-white">Στοιχεία για AI Agent</p>
          <p class="text-xs text-gray-500">Αποθηκεύονται κρυπτογραφημένα (AES-256)</p>
        </div>
      </div>

      <div class="space-y-2 mb-4">
        <input
          v-model="secureForm.number"
          @input="formatSecureCardNumber"
          placeholder="1234 5678 9012 3456"
          maxlength="19"
          class="w-full px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 font-mono tracking-widest"
        />
        <div class="grid grid-cols-2 gap-2">
          <input v-model="secureForm.expiry" placeholder="MM/YY" maxlength="5"
            class="px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 font-mono" />
          <input v-model="secureForm.cvv" placeholder="CVV" maxlength="4" type="password"
            class="px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 font-mono" />
        </div>
        <input v-model="secureForm.cardholder_name" placeholder="ΟΝΟΜΑΤΕΠΩΝΥΜΟ" maxlength="50"
          class="w-full px-3 py-2.5 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50 uppercase font-mono" />
      </div>

      <div class="flex gap-2">
        <button @click="secureCardId = null"
          class="flex-1 py-2.5 rounded-xl border border-white/[0.08] text-sm text-gray-400 hover:text-white hover:bg-white/[0.05] transition-colors">
          Ακύρωση
        </button>
        <button @click="saveSecure" :disabled="secureSaving || !secureForm.number || !secureForm.cvv"
          class="flex-1 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 disabled:opacity-40 text-white text-sm font-bold transition-colors">
          {{ secureSaving ? '...' : 'Αποθήκευση' }}
        </button>
      </div>
    </div>
  </div>

</template>
