import { ref } from 'vue'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmLabel?: string
  variant?: 'danger' | 'warning'
}

const visible = ref(false)
const options = ref<ConfirmOptions>({ message: '' })
let resolveFn: ((value: boolean) => void) | null = null

export function useConfirm() {
  function confirm(opts: ConfirmOptions | string): Promise<boolean> {
    options.value = typeof opts === 'string' ? { message: opts } : opts
    visible.value = true
    return new Promise((resolve) => {
      resolveFn = resolve
    })
  }

  function accept() {
    visible.value = false
    resolveFn?.(true)
    resolveFn = null
  }

  function cancel() {
    visible.value = false
    resolveFn?.(false)
    resolveFn = null
  }

  return { visible, options, confirm, accept, cancel }
}
