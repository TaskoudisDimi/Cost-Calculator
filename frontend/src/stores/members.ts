import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api/client'
import type { Member } from '@/types'

export const useMembersStore = defineStore('members', () => {
  const members = ref<Member[]>([])
  const loading = ref(false)

  async function fetchMembers() {
    loading.value = true
    try {
      const { data } = await api.get<Member[]>('/members')
      members.value = data
    } finally {
      loading.value = false
    }
  }

  async function createMember(name: string, color: string): Promise<Member> {
    const { data } = await api.post<Member>('/members', { name, color })
    members.value = [...members.value, data]
    return data
  }

  async function updateMember(id: string, payload: { name?: string; color?: string }): Promise<Member> {
    const { data } = await api.put<Member>(`/members/${id}`, payload)
    members.value = members.value.map(m => m.id === id ? data : m)
    return data
  }

  async function deleteMember(id: string) {
    await api.delete(`/members/${id}`)
    members.value = members.value.filter(m => m.id !== id)
  }

  return { members, loading, fetchMembers, createMember, updateMember, deleteMember }
})
