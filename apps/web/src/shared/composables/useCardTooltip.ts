import { ref } from 'vue'

/**
 * 全局单例 Tooltip 状态（所有卡片共享）。
 * Host 组件负责 Teleport 渲染。
 *
 * 必须在路由切换 / 页面隐藏 / 目标节点卸载时 hide，
 * 否则会出现「进后台后前台 tip 残留在 body 上」的幽灵气泡。
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

function cancel() {
  if (showTimer !== undefined) {
    window.clearTimeout(showTimer)
    showTimer = undefined
  }
}

/** Module-level hide — used by listeners and exports (must exist before ensureListeners). */
function hide() {
  cancel()
  visible.value = false
  activeEl = null
}

function placeNear(el: HTMLElement) {
  const rect = el.getBoundingClientRect()
  // Detached or display:none → zero box; don't keep a floating tip
  if (rect.width === 0 && rect.height === 0) {
    hide()
    return
  }
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
  if (!visible.value || !activeEl) return
  if (!activeEl.isConnected) {
    hide()
    return
  }
  placeNear(activeEl)
}

function onDocClick(e: Event) {
  if (!visible.value && showTimer === undefined) return
  const t = e.target as Node | null
  if (activeEl && t && activeEl.contains(t)) return
  hide()
}

function onVisibility() {
  if (document.visibilityState === 'hidden') hide()
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') hide()
}

function ensureListeners() {
  if (listenersBound || typeof window === 'undefined') return
  window.addEventListener('scroll', onScrollOrResize, true)
  window.addEventListener('resize', onScrollOrResize)
  document.addEventListener('click', onDocClick, true)
  document.addEventListener('keydown', onKeydown)
  document.addEventListener('visibilitychange', onVisibility)
  window.addEventListener('blur', hide)
  listenersBound = true
}

export function useCardTooltip() {
  ensureListeners()

  function scheduleShow(el: HTMLElement, tip: string, tipTitle = '') {
    cancel()
    if (!tip.trim()) return
    activeEl = el
    showTimer = window.setTimeout(() => {
      if (!activeEl || !activeEl.isConnected) {
        hide()
        return
      }
      text.value = tip
      title.value = tipTitle
      placeNear(activeEl)
      visible.value = true
    }, DELAY)
  }

  /** Only clear if this element owns the open tip (safe for list re-render unmount). */
  function hideIf(el: HTMLElement | null) {
    if (!el) return
    if (activeEl === el) hide()
  }

  return {
    visible,
    text,
    title,
    x,
    y,
    scheduleShow,
    hide,
    hideIf,
    cancel,
  }
}

/** Imperative hide for router / layout. */
export function hideCardTooltip() {
  hide()
}
