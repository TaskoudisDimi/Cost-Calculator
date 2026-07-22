import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface ToastItem {
  id: number
  message: string
  type: ToastType
}

const toasts = ref<ToastItem[]>([])
let _id = 0

export function useToast() {
  function toast(message: string, type: ToastType = 'success', ms = 3500) {
    const id = ++_id
    toasts.value = [...toasts.value, { id, message, type }]
    setTimeout(() => {
      toasts.value = toasts.value.filter(t => t.id !== id)
    }, ms)
  }

  return { toasts, toast }
}
