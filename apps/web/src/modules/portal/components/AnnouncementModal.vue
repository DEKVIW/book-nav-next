<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { sanitizeHtml, looksLikeHtml } from '@/shared/utils/sanitizeHtml'
import { markdownToHtml, looksLikeMarkdown } from '@/shared/utils/markdownLite'
import AppIcon from '@/shared/ui/AppIcon.vue'

const props = defineProps<{
  enabled: boolean
  title?: string
  content?: string
  /** 已知晓后隐藏天数，默认 7 */
  rememberDays?: number
}>()

const STORAGE_KEY = 'booknav_announcement_dismissed_until'
const visible = ref(false)

/**
 * Content pipeline:
 * 1) HTML (legacy) → sanitize + strip style/class → host mecha CSS
 * 2) Markdown → structure-only HTML → same CSS
 * 3) plain text → pre-wrap
 */
const rendered = computed(() => {
  const c = (props.content || '').trim()
  if (!c) return { mode: 'empty' as const, html: '' }
  if (looksLikeHtml(c)) {
    return { mode: 'html' as const, html: sanitizeHtml(c, { stripPresentation: true }) }
  }
  if (looksLikeMarkdown(c)) {
    return { mode: 'md' as const, html: markdownToHtml(c) }
  }
  // also try MD if user mixed simple markers without HTML tags
  if (/\*\*|\[.+\]\(https?:\/\//.test(c)) {
    return { mode: 'md' as const, html: markdownToHtml(c) }
  }
  return { mode: 'text' as const, html: '' }
})

const days = computed(() => (props.rememberDays && props.rememberDays > 0 ? props.rememberDays : 7))

function shouldShow(): boolean {
  if (!props.enabled) return false
  if (!props.title && !props.content) return false
  try {
    const until = localStorage.getItem(STORAGE_KEY)
    if (until && Date.now() < Number(until)) return false
  } catch {
    /* ignore */
  }
  return true
}

function openIfNeeded() {
  visible.value = shouldShow()
}

function dismiss(remember: boolean) {
  if (remember) {
    const until = Date.now() + days.value * 24 * 60 * 60 * 1000
    try {
      localStorage.setItem(STORAGE_KEY, String(until))
    } catch {
      /* ignore */
    }
  }
  visible.value = false
}

onMounted(openIfNeeded)
watch(
  () => [props.enabled, props.title, props.content],
  () => openIfNeeded(),
)
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="ann-mask" role="dialog" aria-modal="true" @click.self="dismiss(false)">
      <div class="ann-dialog hull hull--elevated">
        <div class="hull-corners" aria-hidden="true">
          <span class="c-tl" /><span class="c-tr" /><span class="c-bl" /><span class="c-br" />
        </div>
        <div class="ann-scan" aria-hidden="true" />

        <header class="ann-head">
          <div class="ann-head__row">
            <div class="ann-head__badge">
              <AppIcon name="bell" :size="13" />
              <span>NOTICE</span>
            </div>
            <h2 class="ann-head__title">{{ title || '公告' }}</h2>
          </div>
          <button type="button" class="ann-close" aria-label="关闭" @click="dismiss(false)">
            <AppIcon name="x" :size="16" />
          </button>
        </header>

        <div class="ann-body">
          <div
            v-if="rendered.mode === 'html' || rendered.mode === 'md'"
            class="ann-content"
            v-html="rendered.html"
          />
          <p v-else class="ann-text">{{ content }}</p>
        </div>

        <footer class="ann-foot">
          <button type="button" class="m-btn m-btn--ghost" @click="dismiss(false)">关闭</button>
          <button type="button" class="m-btn m-btn--primary" @click="dismiss(true)">
            知道了，{{ days }} 天内不再显示
          </button>
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.ann-mask {
  position: fixed;
  inset: 0;
  z-index: 850;
  display: grid;
  place-items: center;
  padding: 20px;
  background:
    radial-gradient(ellipse 60% 50% at 50% 40%, rgba(61, 231, 255, 0.08), transparent 70%),
    rgba(2, 6, 14, 0.78);
  backdrop-filter: blur(10px);
  animation: ann-fade 180ms ease-out;
}
@keyframes ann-fade {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.ann-dialog {
  width: min(520px, 100%);
  max-height: min(80vh, 640px);
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
  animation: ann-rise 220ms var(--ease-out, ease-out);
}
@keyframes ann-rise {
  from {
    opacity: 0;
    transform: translateY(10px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

.ann-scan {
  pointer-events: none;
  position: absolute;
  inset: 0;
  z-index: 1;
  background: linear-gradient(
    105deg,
    transparent 40%,
    rgba(61, 231, 255, 0.04) 50%,
    transparent 60%
  );
}

.ann-head {
  position: relative;
  z-index: 3;
  padding: 14px 48px 14px 18px;
  border-bottom: 1px solid var(--stroke-dim);
  background: linear-gradient(180deg, rgba(61, 231, 255, 0.07), transparent);
}
.ann-head__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 12px;
  min-width: 0;
}
.ann-head__badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  padding: 3px 9px;
  border: 1px solid var(--energy-dim);
  border-radius: 2px;
  color: var(--energy);
  font-family: var(--font-display);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.14em;
  background: rgba(61, 231, 255, 0.1);
  box-shadow: 0 0 12px rgba(61, 231, 255, 0.12);
}
.ann-head__title {
  margin: 0;
  min-width: 0;
  font-family: var(--font-display);
  font-size: 0.95rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: var(--energy);
  text-shadow: 0 0 18px rgba(61, 231, 255, 0.25);
}

.ann-close {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-muted);
  border-radius: var(--radius-sm);
  padding: 0;
}
.ann-close:hover {
  color: var(--energy);
  border-color: var(--energy-dim);
  background: rgba(61, 231, 255, 0.08);
}

.ann-body {
  position: relative;
  z-index: 3;
  padding: 16px 18px 14px;
  overflow-y: auto;
  flex: 1;
  min-height: 72px;
  background:
    linear-gradient(135deg, rgba(255, 176, 32, 0.05), rgba(61, 231, 255, 0.03)),
    var(--bg-inset);
}

.ann-text {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.75;
  white-space: pre-wrap;
}

/* Shared mecha styling for sanitized HTML + Markdown output */
.ann-content {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.75;
  overflow-wrap: anywhere;
}
.ann-content :deep(*) {
  background: transparent !important;
  border-color: transparent !important;
  box-shadow: none !important;
  max-width: 100%;
}
.ann-content :deep(div),
.ann-content :deep(p),
.ann-content :deep(span),
.ann-content :deep(li) {
  margin: 0 0 0.55em;
  padding: 0 !important;
  color: var(--text-secondary) !important;
  font-size: inherit !important;
  line-height: inherit !important;
  font-family: inherit !important;
}
.ann-content :deep(div:last-child),
.ann-content :deep(p:last-child) {
  margin-bottom: 0;
}
.ann-content :deep(h1),
.ann-content :deep(h2),
.ann-content :deep(h3) {
  margin: 0 0 0.5em;
  color: var(--text-primary) !important;
  font-family: var(--font-display);
  font-weight: 600;
  letter-spacing: 0.04em;
}
.ann-content :deep(h1) {
  font-size: 1.1rem;
}
.ann-content :deep(h2) {
  font-size: 1rem;
}
.ann-content :deep(h3) {
  font-size: 0.95rem;
}
.ann-content :deep(strong),
.ann-content :deep(b) {
  color: var(--amber) !important;
  font-weight: 650;
}
.ann-content :deep(em),
.ann-content :deep(i) {
  color: var(--text-primary) !important;
  font-style: normal;
  font-weight: 500;
}
.ann-content :deep(a) {
  color: var(--energy) !important;
  text-decoration: none !important;
  border-bottom: 1px solid var(--energy-dim) !important;
  font-weight: 500;
}
.ann-content :deep(a:hover) {
  color: #a8f8ff !important;
  border-bottom-color: var(--energy) !important;
}
.ann-content :deep(ul),
.ann-content :deep(ol) {
  margin: 0 0 0.55em;
  padding-left: 1.2em;
}
.ann-content :deep(code) {
  font-family: var(--font-mono);
  font-size: 0.9em;
  color: var(--energy) !important;
  padding: 0 4px;
  border: 1px solid var(--stroke-dim) !important;
  border-radius: 2px;
  background: rgba(0, 0, 0, 0.25) !important;
}
.ann-content :deep(img) {
  display: none !important;
}
.ann-content :deep(br) {
  content: '';
}

.ann-foot {
  position: relative;
  z-index: 3;
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px 16px;
  border-top: 1px solid var(--stroke-dim);
  background: rgba(0, 0, 0, 0.22);
}
</style>
