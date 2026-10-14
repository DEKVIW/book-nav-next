import { computed, ref } from 'vue'

export type ThemeMode = 'dark' | 'light' | 'system'
export type ResolvedTheme = 'dark' | 'light'

const STORAGE_KEY = 'booknav-theme'
const DEFAULT_MODE: ThemeMode = 'dark'

const mode = ref<ThemeMode>(DEFAULT_MODE)
const resolved = ref<ResolvedTheme>(DEFAULT_MODE)
let initialized = false
let mediaQuery: MediaQueryList | null = null

function isThemeMode(value: string | null): value is ThemeMode {
  return value === 'dark' || value === 'light' || value === 'system'
}

function prefersDark() {
  return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches
}

function resolveTheme(next: ThemeMode): ResolvedTheme {
  return next === 'system' ? (prefersDark() ? 'dark' : 'light') : next
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

function initialize() {
  if (initialized || typeof window === 'undefined') return
  initialized = true

  const stored = localStorage.getItem(STORAGE_KEY)
  mode.value = isThemeMode(stored) ? stored : DEFAULT_MODE
  setResolved(mode.value)

  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', () => {
    if (mode.value === 'system') setResolved('system')
  })
}

export function useTheme() {
  initialize()

  function setTheme(next: ThemeMode) {
    mode.value = next
    localStorage.setItem(STORAGE_KEY, next)
    setResolved(next)
  }

  return {
    mode: computed(() => mode.value),
    resolved: computed(() => resolved.value),
    setTheme,
  }
}
