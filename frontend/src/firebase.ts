import { initializeApp } from 'firebase/app'
import { getAuth } from 'firebase/auth'
import { getFirestore } from 'firebase/firestore'
import { getMessaging, getToken, onMessage, isSupported } from 'firebase/messaging'

const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID,
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID,
  appId: import.meta.env.VITE_FIREBASE_APP_ID,
}

export const firebaseApp = initializeApp(firebaseConfig)
export const auth = getAuth(firebaseApp)
export const db = getFirestore(firebaseApp)

export async function getMessagingIfSupported() {
  try {
    const supported = await isSupported()
    if (!supported) return null
    return getMessaging(firebaseApp)
  } catch {
    return null
  }
}

export async function requestNotificationToken(): Promise<string | null> {
  const vapidKey = import.meta.env.VITE_FIREBASE_VAPID_KEY
  if (!vapidKey) return null

  try {
    const messaging = await getMessagingIfSupported()
    if (!messaging) return null

    const permission = await Notification.requestPermission()
    if (permission !== 'granted') return null

    const sw = await navigator.serviceWorker.register('/firebase-messaging-sw.js')
    const token = await getToken(messaging, { vapidKey, serviceWorkerRegistration: sw })
    return token || null
  } catch {
    return null
  }
}

export { onMessage, getMessaging }
