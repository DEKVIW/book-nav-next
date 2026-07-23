import { ref } from 'vue'

/**
 * 全局单例 Tooltip 状态（所有卡片共享）。
 * Host 组件负责 Teleport 渲染。
 */
const visible = ref(false)
const text = ref('')
const title = ref('')
const x = ref(0)
const y = ref(0)

let showTimer: number | undefined
let activeEl: HTMLElement | null = null
let listenersBound = false

const DELAY = 280

function placeNear(el: HTMLElement) {
  const rect = el.getBoundingClientRect()
  const pad = 10
  const maxW = Math.min(300, window.innerWidth - 24)
  let left = rect.left + rect.width / 2 - maxW / 2
  left = Math.max(pad, Math.min(left, window.innerWidth - maxW - pad))

  let top = rect.bottom + 10
  const estH = 88
  if (top + estH > window.innerHeight - pad) {
    top = Math.max(pad, rect.top - estH - 10)
  }
  x.value = Math.round(left)
  y.value = Math.round(top)
}

function onScrollOrResize() {
  if (visible.value && activeEl) placeNear(activeEl)
}

function ensureListeners() {
  if (listenersBound || typeof window === 'undefined') return
  window.addEventListener('scroll', onScrollOrResize, true)
  window.addEventListener('resize', onScrollOrResize)
  listenersBound = true
}

export function useCardTooltip() {
  ensureListeners()

  function scheduleShow(el: HTMLElement, tip: string, tipTitle = '') {
    cancel()
    if (!tip.trim()) return
    activeEl = el
    showTimer = window.setTimeout(() => {
      if (!activeEl) return
      text.value = tip
      title.value = tipTitle
      placeNear(activeEl)
      visible.value = true
    }, DELAY)
  }

  function hide() {
    cancel()
    visible.value = false
    activeEl = null
  }

  function cancel() {
    if (showTimer !== undefined) {
      window.clearTimeout(showTimer)
      showTimer = undefined
    }
  }

  return {
    visible,
    text,
    title,
    x,
    y,
    scheduleShow,
    hide,
    cancel,
  }
}
