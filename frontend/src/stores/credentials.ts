import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  collection, query, where, orderBy,
  addDoc, updateDoc, deleteDoc, doc,
  onSnapshot, serverTimestamp,
  type Unsubscribe,
} from 'firebase/firestore'
import { db, auth } from '@/firebase'
import type { UserCredential } from '@/types'

export const useCredentialsStore = defineStore('credentials', () => {
  const credentials = ref<UserCredential[]>([])
  let unsub: Unsubscribe | null = null

  function startSync() {
    const uid = auth.currentUser?.uid
    if (!uid || unsub) return

    const q = query(
      collection(db, 'userCredentials'),
      where('user_id', '==', uid),
      orderBy('created_at', 'desc'),
    )

    unsub = onSnapshot(q,
      snap => {
        credentials.value = snap.docs.map(d => {
          const data = d.data()
          return {
            id: d.id,
            user_id: data.user_id ?? '',
            label: data.label ?? '',
            username: data.username ?? '',
            password: data.password ?? '',
            url: data.url ?? '',
            notes: data.notes ?? '',
            created_at: data.created_at?.toDate?.()?.toISOString?.() ?? '',
            updated_at: data.updated_at?.toDate?.()?.toISOString?.() ?? '',
          } as UserCredential
        })
      },
      err => console.error('[credentials] sync error:', err),
    )
  }

  function stopSync() {
    unsub?.()
    unsub = null
    credentials.value = []
  }

  async function create(payload: {
    label: string
    username?: string
    password?: string
    url?: string
    notes?: string
  }) {
    const uid = auth.currentUser?.uid
    if (!uid) return
    await addDoc(collection(db, 'userCredentials'), {
      label: payload.label,
      username: payload.username ?? '',
      password: payload.password ?? '',
      url: payload.url ?? '',
      notes: payload.notes ?? '',
      user_id: uid,
      created_at: serverTimestamp(),
      updated_at: serverTimestamp(),
    })
  }

  async function update(id: string, payload: {
    label?: string
    username?: string
    password?: string
    url?: string
    notes?: string
  }) {
    await updateDoc(doc(db, 'userCredentials', id), {
      ...payload,
      updated_at: serverTimestamp(),
    })
  }

  async function remove(id: string) {
    await deleteDoc(doc(db, 'userCredentials', id))
  }

  return { credentials, startSync, stopSync, create, update, remove }
})
