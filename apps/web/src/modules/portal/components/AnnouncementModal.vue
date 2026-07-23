<script setup lang="ts">
/**
 * 公告弹窗 — 对齐旧站「弹层公告 + N 天不再显示」
 * content 支持 HTML（消毒后渲染）
 */
import { computed, onMounted, ref, watch } from 'vue'
import { sanitizeHtml, looksLikeHtml } from '@/shared/utils/sanitizeHtml'

const props = defineProps<{
  enabled: boolean
  title?: string
  content?: string
  /** 已知晓后隐藏天数，默认 7 */
  rememberDays?: number
}>()

const STORAGE_KEY = 'booknav_announcement_dismissed_until'
const visible = ref(false)

const htmlBody = computed(() => {
  const c = props.content || ''
  if (looksLikeHtml(c)) return sanitizeHtml(c)
  return ''
})

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
    const days = props.rememberDays && props.rememberDays > 0 ? props.rememberDays : 7
    const until = Date.now() + days * 24 * 60 * 60 * 1000
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
        <header class="ann-head">
          <p class="hud-label">NOTICE</p>
          <h2>{{ title || '公告' }}</h2>
          <button type="button" class="ann-close" aria-label="关闭" @click="dismiss(false)">×</button>
        </header>
        <div class="ann-body">
          <div v-if="htmlBody" class="ann-html" v-html="htmlBody" />
          <p v-else class="ann-text">{{ content }}</p>
        </div>
        <footer class="ann-foot">
          <button type="button" class="m-btn m-btn--ghost" @click="dismiss(false)">关闭</button>
          <button type="button" class="m-btn m-btn--primary" @click="dismiss(true)">
            知道了，{{ rememberDays || 7 }} 天内不再显示
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
  background: rgba(2, 6, 14, 0.72);
  backdrop-filter: blur(8px);
  animation: fade-in 180ms ease-out;
}
@keyframes fade-in {
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
}
.ann-head {
  position: relative;
  padding: 18px 44px 12px 20px;
  border-bottom: 1px solid var(--stroke-dim);
}
.ann-head h2 {
  margin: 6px 0 0;
  font-size: 1.15rem;
}
.ann-close {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 32px;
  height: 32px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
  border-radius: 6px;
}
.ann-close:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.06);
}
.ann-body {
  padding: 16px 20px;
  overflow-y: auto;
  flex: 1;
  min-height: 80px;
}
.ann-text {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.65;
  white-space: pre-wrap;
}
.ann-html {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}
.ann-html :deep(a) {
  color: var(--energy);
}
.ann-html :deep(div),
.ann-html :deep(p) {
  margin: 0 0 8px;
}
.ann-foot {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px 16px;
  border-top: 1px solid var(--stroke-dim);
  background: rgba(0, 0, 0, 0.15);
}
</style>
