import { ref } from 'vue'
import { defineStore } from 'pinia'
import api from '@/api/client'
import type { Card } from '@/types'

export const useCardsStore = defineStore('cards', () => {
  const cards = ref<Card[]>([])
  const loading = ref(false)

  async function fetchCards() {
    loading.value = true
    try {
      const { data } = await api.get<Card[]>('/cards')
      cards.value = data
    } finally {
      loading.value = false
    }
  }

  async function createCard(payload: Omit<Card, 'id' | 'user_id' | 'created_at' | 'updated_at'>): Promise<Card> {
    const { data } = await api.post<Card>('/cards', payload)
    cards.value.push(data)
    return data
  }

  async function updateCard(id: string, payload: Partial<Omit<Card, 'id' | 'user_id' | 'last_four' | 'created_at' | 'updated_at'>>): Promise<Card> {
    const { data } = await api.put<Card>(`/cards/${id}`, payload)
    const idx = cards.value.findIndex(c => c.id === id)
    if (idx !== -1) {
      cards.value[idx] = data
      // If this card is now default, unset others locally
      if (data.is_default) {
        cards.value.forEach((c, i) => { if (i !== idx) c.is_default = false })
      }
    }
    return data
  }

  async function deleteCard(id: string) {
    await api.delete(`/cards/${id}`)
    cards.value = cards.value.filter(c => c.id !== id)
  }

  function cardById(id: string): Card | undefined {
    return cards.value.find(c => c.id === id)
  }

  return { cards, loading, fetchCards, createCard, updateCard, deleteCard, cardById }
})
