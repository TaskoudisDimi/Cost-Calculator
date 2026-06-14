import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  createUserWithEmailAndPassword,
  signInWithEmailAndPassword,
  signOut,
  onAuthStateChanged,
  updateProfile,
  type User as FirebaseUser,
} from 'firebase/auth'
import { auth } from '@/firebase'

export const useAuthStore = defineStore('auth', () => {
  const firebaseUser = ref<FirebaseUser | null>(null)
  const loading = ref(true)

  const isAuthenticated = computed(() => !!firebaseUser.value)

  const user = computed(() => firebaseUser.value
    ? {
        id: firebaseUser.value.uid,
        name: firebaseUser.value.displayName ?? firebaseUser.value.email ?? '',
        email: firebaseUser.value.email ?? '',
      }
    : null
  )

  // Listen for Firebase auth state changes
  onAuthStateChanged(auth, (fbUser) => {
    firebaseUser.value = fbUser
    loading.value = false
  })

  async function getToken(): Promise<string> {
    if (!firebaseUser.value) throw new Error('not authenticated')
    return firebaseUser.value.getIdToken()
  }

  async function login(email: string, password: string) {
    await signInWithEmailAndPassword(auth, email, password)
  }

  async function register(name: string, email: string, password: string) {
    const { user: fbUser } = await createUserWithEmailAndPassword(auth, email, password)
    await updateProfile(fbUser, { displayName: name })
    firebaseUser.value = fbUser
  }

  async function logout() {
    await signOut(auth)
    firebaseUser.value = null
  }

  return { user, firebaseUser, loading, isAuthenticated, getToken, login, register, logout }
})
