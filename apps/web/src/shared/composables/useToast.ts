import { ref } from 'vue'

export type ToastKind = 'info' | 'success' | 'error'

export interface ToastItem {
  id: number
  message: string
  kind: ToastKind
}

const toasts = ref<ToastItem[]>([])
let seq = 1

export function useToast() {
  function push(message: string, kind: ToastKind = 'info', ms = 2800) {
    const id = seq++
    toasts.value.push({ id, message, kind })
    window.setTimeout(() => {
      toasts.value = toasts.value.filter((t) => t.id !== id)
    }, ms)
  }
  return {
    toasts,
    info: (m: string) => push(m, 'info'),
    success: (m: string) => push(m, 'success'),
    error: (m: string) => push(m, 'error'),
  }
}
