import { computed, ref } from 'vue'

export type ThemeMode = 'auto' | 'dark' | 'light' | 'system'
export type ResolvedTheme = 'dark' | 'light'

const STORAGE_KEY = 'booknav-theme'
const DEFAULT_MODE: ThemeMode = 'auto'
const DAY_START_HOUR = 7
const NIGHT_START_HOUR = 19

const mode = ref<ThemeMode>(DEFAULT_MODE)
const resolved = ref<ResolvedTheme>('dark')
let initialized = false
let mediaQuery: MediaQueryList | null = null
let autoTimer: ReturnType<typeof window.setTimeout> | null = null

function isThemeMode(value: string | null): value is ThemeMode {
  return value === 'auto' || value === 'dark' || value === 'light' || value === 'system'
}

function prefersDark() {
  return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches
}

function isDaytime(date = new Date()) {
  const hour = date.getHours()
  return hour >= DAY_START_HOUR && hour < NIGHT_START_HOUR
}

function resolveTheme(next: ThemeMode): ResolvedTheme {
  if (next === 'system') return prefersDark() ? 'dark' : 'light'
  if (next === 'auto') return isDaytime() ? 'light' : 'dark'
  return next
}

function applyTheme(next: ResolvedTheme) {
  if (typeof document === 'undefined') return
  document.documentElement.dataset.theme = next
  document.documentElement.style.colorScheme = next
  const meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')
  if (meta) meta.content = next === 'dark' ? '#070b12' : '#e8f0f3'
}

function setResolved(next: ThemeMode) {
  resolved.value = resolveTheme(next)
  applyTheme(resolved.value)
}

function clearAutoTimer() {
  if (autoTimer === null || typeof window === 'undefined') return
  window.clearTimeout(autoTimer)
  autoTimer = null
}

function scheduleAutoTheme() {
  clearAutoTimer()
  if (mode.value !== 'auto' || typeof window === 'undefined') return

  const now = new Date()
  const next = new Date(now)
  next.setSeconds(0, 0)
  if (now.getHours() < DAY_START_HOUR) {
    next.setHours(DAY_START_HOUR, 0, 0, 0)
  } else if (now.getHours() < NIGHT_START_HOUR) {
    next.setHours(NIGHT_START_HOUR, 0, 0, 0)
  } else {
    next.setDate(next.getDate() + 1)
    next.setHours(DAY_START_HOUR, 0, 0, 0)
  }

  autoTimer = window.setTimeout(() => {
    setResolved('auto')
    scheduleAutoTheme()
  }, Math.max(next.getTime() - now.getTime() + 100, 1000))
}

function refreshAutoTheme() {
  if (mode.value !== 'auto') return
  setResolved('auto')
  scheduleAutoTheme()
}

function initialize() {
  if (initialized || typeof window === 'undefined') return
  initialized = true

  const stored = localStorage.getItem(STORAGE_KEY)
  mode.value = isThemeMode(stored) ? stored : DEFAULT_MODE
  setResolved(mode.value)
  scheduleAutoTheme()

  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', () => {
    if (mode.value === 'system') setResolved('system')
  })
  window.addEventListener('focus', refreshAutoTheme)
  document.addEventListener('visibilitychange', refreshAutoTheme)
}

export function useTheme() {
  initialize()

  function setTheme(next: ThemeMode) {
    mode.value = next
    localStorage.setItem(STORAGE_KEY, next)
    setResolved(next)
    scheduleAutoTheme()
  }

  return {
    mode: computed(() => mode.value),
    resolved: computed(() => resolved.value),
    setTheme,
  }
}
