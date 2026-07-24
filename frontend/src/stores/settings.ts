import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api/client'
import { activeCurrency } from '@/utils/format'

export interface UserSettings {
  notif_enabled: boolean
  notif_days: number
  currency: string
}

export const useSettingsStore = defineStore('settings', () => {
  const notifEnabled = ref(false)
  const notifDays = ref(3)
  const currency = ref('EUR')
  const loading = ref(false)

  async function fetchSettings() {
    loading.value = true
    try {
      const { data } = await api.get<UserSettings>('/settings')
      notifEnabled.value = data.notif_enabled
      notifDays.value = data.notif_days
      currency.value = data.currency || 'EUR'
      activeCurrency.value = data.currency || 'EUR'
    } finally {
      loading.value = false
    }
  }

  async function saveSettings() {
    await api.put('/settings', {
      notif_enabled: notifEnabled.value,
      notif_days: notifDays.value,
      currency: currency.value,
    })
    activeCurrency.value = currency.value
  }

  async function deleteAccount() {
    await api.delete('/account')
  }

  return { notifEnabled, notifDays, currency, loading, fetchSettings, saveSettings, deleteAccount }
})
