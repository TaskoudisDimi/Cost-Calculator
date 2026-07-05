<script setup lang="ts">
import { ref, computed, onMounted, unref } from 'vue'
import { useBillsStore } from '@/stores/bills'
import { useConfirm } from '@/composables/useConfirm'
import { categoryLabel } from '@/utils/format'

const { confirm } = useConfirm()

const store = useBillsStore()
const showModal = ref(false)
const modalMode = ref<'list' | 'template'>('list')
const error = ref('')
const saving = ref(false)

const listForm = ref({ provider_id: '', nickname: '', account_num: '' })
const templateForm = ref({
  name: '',
  category: 'other',
  customCategory: '',
  color: '#6B7280',
  payment_url: '',
})

const colorOptions = [
  '#3B82F6', '#EF4444', '#10B981', '#F59E0B',
  '#8B5CF6', '#EC4899', '#F97316', '#6B7280',
]

const seededCategoryOrder = ['energy', 'water', 'telecom', 'streaming', 'subscription', 'housing', 'finance', 'car', 'other']

const builtinCategories = [
  { value: 'energy', label: 'Ρεύμα' },
  { value: 'water', label: 'Νερό' },
  { value: 'telecom', label: 'Τηλεπικοινωνίες' },
  { value: 'streaming', label: 'Streaming' },
  { value: 'subscription', label: 'Συνδρομές' },
  { value: 'housing', label: 'Κατοικία' },
  { value: 'finance', label: 'Οικονομικά' },
  { value: 'car', label: 'Αυτοκίνητο' },
  { value: 'other', label: 'Άλλο' },
  { value: '__custom__', label: '+ Νέα κατηγορία…' },
]

const allCategories = computed(() => {
  const templates = unref(store.providerTemplates)
  const templateCats = new Set(Array.isArray(templates) ? templates.map(t => t.category) : [])
  const extra = [...templateCats].filter(c => !seededCategoryOrder.includes(c))
  return [...seededCategoryOrder, ...extra]
})

onMounted(async () => {
  await Promise.all([store.fetchProviders(), store.fetchUserProviders(), store.fetchProviderTemplates()])
})

function providersByCategory(cat: string) {
  const providers = unref(store.providers)
  return Array.isArray(providers) ? providers.filter(p => p.category === cat) : []
}

function templatesByCategory(cat: string) {
  const templates = unref(store.providerTemplates)
  return Array.isArray(templates) ? templates.filter(t => t.category === cat) : []
}

function isAdded(providerId: string) {
  const ups = unref(store.userProviders)
  return Array.isArray(ups) ? ups.some(up => up.provider_id === providerId) : false
}

function hasAnythingInCategory(cat: string) {
  return providersByCategory(cat).length > 0 || templatesByCategory(cat).length > 0
}

function openListModal(providerId = '') {
  modalMode.value = 'list'
  listForm.value = { provider_id: providerId, nickname: '', account_num: '' }
  error.value = ''
  showModal.value = true
}

function openTemplateModal() {
  modalMode.value = 'template'
  templateForm.value = { name: '', category: 'other', customCategory: '', color: '#6B7280', payment_url: '' }
  error.value = ''
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  error.value = ''
}

async function addFromList() {
  error.value = ''
  saving.value = true
  try {
    await store.addUserProvider({
      provider_id: listForm.value.provider_id,
      nickname: listForm.value.nickname,
      account_num: listForm.value.account_num,
    })
    closeModal()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Σφάλμα.'
  } finally {
    saving.value = false
  }
}

async function activateTemplate(templateId: string) {
  if (isAdded(templateId)) return
  try {
    await store.addUserProvider({ provider_id: templateId })
  } catch {
    // silently ignore
  }
}

async function saveTemplate() {
  error.value = ''
  saving.value = true
  try {
    const cat = templateForm.value.category === '__custom__'
      ? templateForm.value.customCategory.trim()
      : templateForm.value.category
    await store.createProviderTemplate({
      name: templateForm.value.name,
      category: cat || 'other',
      color: templateForm.value.color,
      payment_url: templateForm.value.payment_url,
    })
    closeModal()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Σφάλμα.'
  } finally {
    saving.value = false
  }
}

async function removeProvider(id: string) {
  if (!await confirm({ message: 'Θα αφαιρεθεί ο πάροχος. Η ενέργεια δεν αναιρείται.', confirmLabel: 'Αφαίρεση' })) return
  await store.deleteUserProvider(id)
}

async function deleteTemplate(id: string) {
  if (!await confirm({ message: 'Θα διαγραφεί το πρότυπο πάροχου.', confirmLabel: 'Διαγραφή' })) return
  await store.deleteProviderTemplate(id)
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-5">
      <h2 class="text-xl md:text-2xl font-bold text-gray-50">Πάροχοί μου</h2>
      <div class="flex gap-2">
        <button
          @click="openTemplateModal"
          class="bg-gray-800 border border-gray-600 hover:border-blue-400 text-gray-300 text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          + Νέο πρότυπο
        </button>
        <button
          @click="openListModal()"
          class="bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          + Από λίστα
        </button>
      </div>
    </div>

    <!-- Active user providers -->
    <div v-if="store.userProviders.length > 0" class="mb-8">
      <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">Ενεργοί</h3>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div
          v-for="up in store.userProviders"
          :key="up.id"
          class="bg-gray-800 rounded-xl border border-gray-700 p-4 flex items-center justify-between"
        >
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-xl flex items-center justify-center text-white text-sm font-bold"
              :style="{ backgroundColor: up.provider?.color }"
            >
              {{ (up.nickname || up.provider?.name || '?').charAt(0).toUpperCase() }}
            </div>
            <div>
              <p class="text-sm font-semibold text-gray-50">{{ up.nickname || up.provider?.name }}</p>
              <p class="text-xs text-gray-400">{{ categoryLabel(up.provider?.category) }}</p>
              <p v-if="up.account_num" class="text-xs text-gray-400">{{ up.account_num }}</p>
            </div>
          </div>
          <button
            @click="removeProvider(up.id)"
            class="text-gray-300 hover:text-red-400 transition-colors text-lg leading-none"
          >
            ✕
          </button>
        </div>
      </div>
    </div>

    <!-- Providers grid by category -->
    <div v-for="cat in allCategories" :key="cat">
      <div v-if="hasAnythingInCategory(cat)" class="mb-6">
        <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">
          {{ categoryLabel(cat) }}
        </h3>
        <div class="grid grid-cols-3 sm:grid-cols-4 gap-3">
          <!-- Seeded providers -->
          <div
            v-for="p in providersByCategory(cat)"
            :key="p.id"
            class="bg-gray-800 rounded-xl border p-3 flex flex-col items-center gap-2 transition-all"
            :class="isAdded(p.id) ? 'border-green-200 opacity-60' : 'border-gray-700 hover:border-blue-300 cursor-pointer'"
            @click="!isAdded(p.id) && openListModal(p.id)"
          >
            <div
              class="w-10 h-10 rounded-xl flex items-center justify-center text-white text-sm font-bold"
              :style="{ backgroundColor: p.color }"
            >
              {{ (p.name || '?').charAt(0) }}
            </div>
            <p class="text-xs font-medium text-gray-300 text-center">{{ p.name }}</p>
            <span v-if="isAdded(p.id)" class="text-xs text-green-400 font-medium">✓ Ενεργός</span>
          </div>

          <!-- User template providers -->
          <div
            v-for="t in templatesByCategory(cat)"
            :key="t.id"
            class="bg-gray-800 rounded-xl border p-3 flex flex-col items-center gap-2 transition-all relative"
            :class="isAdded(t.id) ? 'border-green-200 opacity-60' : 'border-dashed border-gray-600 hover:border-blue-300 cursor-pointer'"
            @click="activateTemplate(t.id)"
          >
            <!-- Delete template button -->
            <button
              @click.stop="deleteTemplate(t.id)"
              class="absolute top-1.5 right-1.5 w-5 h-5 flex items-center justify-center text-gray-300 hover:text-red-400 transition-colors text-xs leading-none"
              title="Διαγραφή προτύπου"
            >
              ✕
            </button>
            <div
              class="w-10 h-10 rounded-xl flex items-center justify-center text-white text-sm font-bold"
              :style="{ backgroundColor: t.color }"
            >
              {{ (t.name || '?').charAt(0) }}
            </div>
            <p class="text-xs font-medium text-gray-300 text-center">{{ t.name }}</p>
            <span v-if="isAdded(t.id)" class="text-xs text-green-400 font-medium">✓ Ενεργός</span>
            <span v-else class="text-xs text-gray-400">Δικός μου</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/40 flex items-end sm:items-center justify-center z-50 px-4 pb-4 sm:pb-0" @click.self="closeModal">
      <div class="bg-gray-800 rounded-2xl shadow-xl w-full max-w-sm p-6">

        <!-- Mode toggle -->
        <div class="flex rounded-lg border border-gray-700 p-1 mb-5">
          <button
            @click="modalMode = 'list'"
            class="flex-1 py-1.5 rounded-md text-sm font-medium transition-colors"
            :class="modalMode === 'list' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-gray-300'"
          >
            Από λίστα
          </button>
          <button
            @click="modalMode = 'template'"
            class="flex-1 py-1.5 rounded-md text-sm font-medium transition-colors"
            :class="modalMode === 'template' ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-gray-300'"
          >
            Νέο πρότυπο
          </button>
        </div>

        <!-- From list -->
        <form v-if="modalMode === 'list'" @submit.prevent="addFromList" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Πάροχος</label>
            <select
              v-model="listForm.provider_id"
              required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="" disabled>Επίλεξε</option>
              <option v-for="p in store.providers" :key="p.id" :value="p.id" :disabled="isAdded(p.id)">
                {{ p.name }}{{ isAdded(p.id) ? ' (ήδη ενεργός)' : '' }}
              </option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Παρατσούκλι (προαιρετικό)</label>
            <input v-model="listForm.nickname" type="text"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="π.χ. ΔΕΗ σπίτι" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Αρ. λογαριασμού (προαιρετικό)</label>
            <input v-model="listForm.account_num" type="text"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="π.χ. 12345678" />
          </div>
          <p v-if="error" class="text-red-400 text-sm">{{ error }}</p>
          <div class="flex gap-3 pt-1">
            <button type="button" @click="closeModal"
              class="flex-1 px-4 py-2.5 rounded-lg border border-gray-600 text-sm font-medium text-gray-300 hover:bg-gray-700/60">Ακύρωση</button>
            <button type="submit" :disabled="saving"
              class="flex-1 px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium disabled:opacity-60">
              {{ saving ? 'Αποθήκευση…' : 'Προσθήκη' }}
            </button>
          </div>
        </form>

        <!-- New template form -->
        <form v-else @submit.prevent="saveTemplate" class="space-y-4">
          <p class="text-xs text-gray-400 -mt-2">Δημιουργήσε ένα δικό σου πρότυπο παρόχου. Θα εμφανίζεται στο grid και μπορείς να το ενεργοποιήσεις με ένα κλικ.</p>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Όνομα <span class="text-red-400">*</span></label>
            <input v-model="templateForm.name" type="text" required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="π.χ. Ενοίκιο γραφείου, Δάνειο αυτοκινήτου…" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Κατηγορία</label>
            <select v-model="templateForm.category"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option v-for="c in builtinCategories" :key="c.value" :value="c.value">{{ c.label }}</option>
            </select>
          </div>
          <div v-if="templateForm.category === '__custom__'">
            <label class="block text-sm font-medium text-gray-300 mb-1">Όνομα κατηγορίας <span class="text-red-400">*</span></label>
            <input v-model="templateForm.customCategory" type="text" required
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="π.χ. Αθλητισμός, Εκπαίδευση…" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Χρώμα</label>
            <div class="flex gap-2 flex-wrap">
              <button
                v-for="col in colorOptions"
                :key="col"
                type="button"
                @click="templateForm.color = col"
                class="w-8 h-8 rounded-full border-2 transition-all"
                :style="{ backgroundColor: col }"
                :class="templateForm.color === col ? 'border-gray-900 scale-110' : 'border-transparent'"
              />
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">URL πληρωμής (προαιρετικό)</label>
            <input v-model="templateForm.payment_url" type="url"
              class="w-full px-3 py-2.5 rounded-lg border border-gray-600 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="https://…" />
          </div>
          <p v-if="error" class="text-red-400 text-sm">{{ error }}</p>
          <div class="flex gap-3 pt-1">
            <button type="button" @click="closeModal"
              class="flex-1 px-4 py-2.5 rounded-lg border border-gray-600 text-sm font-medium text-gray-300 hover:bg-gray-700/60">Ακύρωση</button>
            <button type="submit" :disabled="saving"
              class="flex-1 px-4 py-2.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium disabled:opacity-60">
              {{ saving ? 'Αποθήκευση…' : 'Αποθήκευση' }}
            </button>
          </div>
        </form>

      </div>
    </div>
  </div>
</template>
