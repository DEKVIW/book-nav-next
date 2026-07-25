<script setup lang="ts">
import { reactive, watch, ref, computed } from 'vue'
import type { Category, Website } from '@/shared/types/models'

const props = defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  site?: Website | null
  categories: Category[]
  initialUrl?: string
  loading?: boolean
  /** 抓取进度文案，如「正在解析页面…」 */
  progressText?: string
  /** AI 已启用且当前用户可用（管理员快加场景） */
  aiAvailable?: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: [payload: Record<string, unknown>]
  fetchMeta: [url: string]
  translate: [payload: { field: 'title' | 'description'; text: string }]
  enhance: [payload: { url: string; title: string; description: string }]
}>()

const form = reactive({
  title: '',
  url: '',
  description: '',
  icon: '',
  category_id: null as number | null,
  is_private: false,
  is_featured: false,
})

const iconBroken = ref(false)
/** keep originals so user can restore after translate/AI */
const originalTitle = ref('')
const originalDescription = ref('')
const aiBusy = ref<'translate-title' | 'translate-desc' | 'enhance' | null>(null)

const showAI = computed(() => !!props.aiAvailable)
const canTranslateTitle = computed(() => showAI.value && !!form.title.trim())
const canTranslateDesc = computed(() => showAI.value && !!form.description.trim())
const canEnhance = computed(() => showAI.value && (!!form.url.trim() || !!form.title.trim()))
const canRestoreTitle = computed(() => !!originalTitle.value && originalTitle.value !== form.title)
const canRestoreDesc = computed(
  () => !!originalDescription.value && originalDescription.value !== form.description,
)

watch(
  () => [props.open, props.site, props.initialUrl] as const,
  () => {
    if (!props.open) return
    iconBroken.value = false
    aiBusy.value = null
    originalTitle.value = ''
    originalDescription.value = ''
    if (props.mode === 'edit' && props.site) {
      form.title = props.site.title
      form.url = props.site.url
      form.description = props.site.description || ''
      form.icon = props.site.icon || ''
      form.category_id = props.site.category_id ?? null
      form.is_private = props.site.is_private
      form.is_featured = props.site.is_featured
      originalTitle.value = form.title
      originalDescription.value = form.description
    } else {
      form.title = ''
      form.url = props.initialUrl || ''
      form.description = ''
      form.icon = ''
      form.category_id = props.categories[0]?.id ?? null
      form.is_private = false
      form.is_featured = false
    }
  },
  { immediate: true },
)

function flatCategories(list: Category[], depth = 0): { id: number; label: string }[] {
  const out: { id: number; label: string }[] = []
  for (const c of list) {
    out.push({ id: c.id, label: `${'— '.repeat(depth)}${c.name}` })
    if (c.children?.length) out.push(...flatCategories(c.children, depth + 1))
  }
  return out
}

function onSubmit() {
  emit('submit', {
    title: form.title,
    url: form.url,
    description: form.description,
    icon: form.icon,
    category_id: form.category_id,
    is_private: form.is_private,
    is_featured: form.is_featured,
    force: true,
  })
}

function onUrlBlur() {
  const u = form.url.trim()
  if (!u || props.loading) return
  if (props.mode === 'create' && !form.title.trim()) {
    emit('fetchMeta', u)
  }
}

function onTranslate(field: 'title' | 'description') {
  const text = field === 'title' ? form.title.trim() : form.description.trim()
  if (!text || aiBusy.value || props.loading) return
  if (field === 'title' && !originalTitle.value) originalTitle.value = form.title
  if (field === 'description' && !originalDescription.value) originalDescription.value = form.description
  aiBusy.value = field === 'title' ? 'translate-title' : 'translate-desc'
  emit('translate', { field, text })
}

function onEnhance() {
  if (!canEnhance.value || aiBusy.value || props.loading) return
  if (form.title && !originalTitle.value) originalTitle.value = form.title
  if (form.description && !originalDescription.value) originalDescription.value = form.description
  aiBusy.value = 'enhance'
  emit('enhance', {
    url: form.url.trim(),
    title: form.title.trim(),
    description: form.description.trim(),
  })
}

function restoreTitle() {
  if (originalTitle.value) form.title = originalTitle.value
}
function restoreDesc() {
  if (originalDescription.value) form.description = originalDescription.value
}

defineExpose({
  applyMeta(meta: { title?: string; description?: string; icon_url?: string; url?: string }) {
    if (meta.title) {
      form.title = String(meta.title)
      originalTitle.value = form.title
    }
    if (meta.description) {
      form.description = String(meta.description)
      originalDescription.value = form.description
    }
    if (meta.icon_url) {
      form.icon = String(meta.icon_url)
      iconBroken.value = false
    }
    if (meta.url) form.url = String(meta.url)
  },
  applyTranslate(field: 'title' | 'description', text: string) {
    if (field === 'title') form.title = text
    else form.description = text
    aiBusy.value = null
  },
  applyEnhance(meta: { title?: string; description?: string }) {
    if (meta.title) form.title = String(meta.title)
    if (meta.description) form.description = String(meta.description)
    aiBusy.value = null
  },
  clearAiBusy() {
    aiBusy.value = null
  },
  setUrl(url: string) {
    form.url = url
  },
  getForm() {
    return { ...form }
  },
})
</script>

<template>
  <div v-if="open" class="mecha-mask" @click.self="emit('close')">
    <div class="modal hull hull--elevated">
      <div class="hull-corners" aria-hidden="true">
        <span class="c-tl" /><span class="c-tr" /><span class="c-bl" /><span class="c-br" />
      </div>
      <header class="modal-head">
        <div class="modal-head__row">
          <span class="modal-head__mark" aria-hidden="true" />
          <h2 class="modal-head__title">
            {{ mode === 'create' ? '快速添加链接' : '编辑链接' }}
          </h2>
          <span class="modal-head__beam" aria-hidden="true" />
          <span class="modal-head__end" aria-hidden="true" />
        </div>
      </header>

      <div v-if="loading || aiBusy" class="progress">
        <div class="progress__bar" />
        <span>
          <template v-if="aiBusy === 'translate-title'">正在翻译标题…</template>
          <template v-else-if="aiBusy === 'translate-desc'">正在翻译描述…</template>
          <template v-else-if="aiBusy === 'enhance'">正在 AI 补全站点信息…</template>
          <template v-else>{{ progressText || '正在处理…' }}</template>
        </span>
      </div>

      <form class="form" @submit.prevent="onSubmit">
        <label>
          <span>网址 URL</span>
          <div class="row">
            <input
              v-model="form.url"
              class="m-input"
              required
              placeholder="https://example.com"
              @blur="onUrlBlur"
            />
            <button
              type="button"
              class="m-btn m-btn--ghost"
              :disabled="!form.url || loading || !!aiBusy"
              @click="emit('fetchMeta', form.url)"
            >
              抓取信息
            </button>
          </div>
        </label>

        <label>
          <div class="label-row">
            <span>标题</span>
            <span v-if="showAI" class="ai-actions">
              <button
                type="button"
                class="link-btn"
                :disabled="!canTranslateTitle || loading || !!aiBusy"
                @click="onTranslate('title')"
              >
                译成中文
              </button>
              <button
                v-if="canRestoreTitle"
                type="button"
                class="link-btn link-btn--muted"
                :disabled="loading || !!aiBusy"
                @click="restoreTitle"
              >
                还原
              </button>
            </span>
          </div>
          <input v-model="form.title" class="m-input" required placeholder="自动填充或手动填写" />
        </label>

        <label>
          <div class="label-row">
            <span>描述</span>
            <span v-if="showAI" class="ai-actions">
              <button
                type="button"
                class="link-btn"
                :disabled="!canTranslateDesc || loading || !!aiBusy"
                @click="onTranslate('description')"
              >
                译成中文
              </button>
              <button
                type="button"
                class="link-btn"
                :disabled="!canEnhance || loading || !!aiBusy"
                @click="onEnhance"
              >
                AI 补描述
              </button>
              <button
                v-if="canRestoreDesc"
                type="button"
                class="link-btn link-btn--muted"
                :disabled="loading || !!aiBusy"
                @click="restoreDesc"
              >
                还原
              </button>
            </span>
          </div>
          <textarea v-model="form.description" class="m-input area" rows="3" placeholder="可选；可抓取后译中文或 AI 补全" />
          <span v-if="showAI" class="hint">AI 结果请核对；失败不影响添加</span>
        </label>

        <label>
          <span>图标</span>
          <div class="icon-row">
            <div class="icon-preview">
              <img
                v-if="form.icon && !iconBroken"
                :src="form.icon"
                alt=""
                referrerpolicy="no-referrer"
                @error="iconBroken = true"
              />
              <span v-else class="icon-fallback">{{ (form.title || '?').slice(0, 1) }}</span>
            </div>
            <input v-model="form.icon" class="m-input" placeholder="图标 URL（可自动抓取）" @input="iconBroken = false" />
          </div>
        </label>

        <label>
          <span>分类</span>
          <select v-model="form.category_id" class="m-input">
            <option :value="null">未分类</option>
            <option v-for="c in flatCategories(categories)" :key="c.id" :value="c.id">
              {{ c.label }}
            </option>
          </select>
        </label>

        <div class="checks">
          <label class="check"><input v-model="form.is_private" type="checkbox" /> 私有</label>
          <label class="check"><input v-model="form.is_featured" type="checkbox" /> 精选</label>
        </div>

        <footer>
          <button type="button" class="m-btn m-btn--ghost" @click="emit('close')">取消</button>
          <button type="submit" class="m-btn m-btn--primary" :disabled="loading || !!aiBusy">
            {{ loading || aiBusy ? '请稍候…' : mode === 'create' ? '添加' : '保存' }}
          </button>
        </footer>
      </form>
    </div>
  </div>
</template>

<style scoped>
.modal {
  width: min(480px, 100%);
  padding: 22px 20px 18px;
  position: relative;
}
/* 单行机甲标题：中文主标题 + 能量轨 */
.modal-head {
  margin: 0 0 16px;
  padding: 0 0 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--stroke-dim) 80%, transparent);
  position: relative;
}
.modal-head::after {
  content: '';
  position: absolute;
  left: 0;
  bottom: -1px;
  width: 72px;
  height: 2px;
  background: linear-gradient(
    90deg,
    var(--energy, #3de7ff),
    color-mix(in srgb, var(--energy, #3de7ff) 20%, transparent)
  );
  box-shadow: 0 0 10px color-mix(in srgb, var(--energy, #3de7ff) 45%, transparent);
}
.modal-head__row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 28px;
}
.modal-head__mark {
  width: 10px;
  height: 10px;
  flex-shrink: 0;
  background: var(--energy, #3de7ff);
  clip-path: polygon(50% 0, 100% 50%, 50% 100%, 0 50%);
  box-shadow: 0 0 8px color-mix(in srgb, var(--energy, #3de7ff) 55%, transparent);
}
.modal-head__title {
  margin: 0;
  flex-shrink: 0;
  font-family: var(--font-display, inherit);
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0.06em;
  color: var(--text-primary, #e8eef7);
  line-height: 1.2;
  text-shadow: 0 0 18px color-mix(in srgb, var(--energy, #3de7ff) 22%, transparent);
}
.modal-head__beam {
  flex: 1;
  height: 1px;
  min-width: 24px;
  background: repeating-linear-gradient(
    90deg,
    color-mix(in srgb, var(--energy, #3de7ff) 35%, transparent) 0 4px,
    transparent 4px 8px
  );
  opacity: 0.7;
}
.modal-head__end {
  width: 8px;
  height: 8px;
  flex-shrink: 0;
  border: 1px solid color-mix(in srgb, var(--energy, #3de7ff) 55%, transparent);
  background: color-mix(in srgb, var(--energy, #3de7ff) 12%, transparent);
  clip-path: polygon(2px 0, 100% 0, 100% calc(100% - 2px), calc(100% - 2px) 100%, 0 100%, 0 2px);
}
.progress {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  padding: 8px 10px;
  background: var(--bg-inset);
  border: 1px solid var(--stroke-dim);
  font-size: 12px;
  color: var(--text-secondary);
}
.progress__bar {
  width: 14px;
  height: 14px;
  border: 2px solid var(--energy-dim);
  border-top-color: var(--energy);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
.form {
  display: grid;
  gap: 12px;
}
.form label {
  display: grid;
  gap: 6px;
  font-size: var(--text-xs);
  color: var(--text-muted);
}
.label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.ai-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}
.link-btn {
  border: none;
  background: transparent;
  color: var(--energy, #3de7ff);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  padding: 0;
  line-height: 1.2;
}
.link-btn:hover:not(:disabled) {
  text-decoration: underline;
}
.link-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.link-btn--muted {
  color: var(--text-muted);
  font-weight: 500;
}
.hint {
  font-size: 11px;
  color: var(--text-muted);
  opacity: 0.85;
}
.row {
  display: flex;
  gap: 8px;
}
.row .m-input {
  flex: 1;
}
.area {
  height: auto;
  padding: 10px 12px;
  resize: vertical;
}
.icon-row {
  display: flex;
  gap: 10px;
  align-items: center;
}
.icon-preview {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border: 1px solid var(--stroke-dim);
  background: var(--bg-inset);
  display: grid;
  place-items: center;
  overflow: hidden;
  clip-path: polygon(4px 0, 100% 0, 100% calc(100% - 4px), calc(100% - 4px) 100%, 0 100%, 0 4px);
}
.icon-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.icon-fallback {
  font-family: var(--font-display);
  color: var(--energy);
}
.checks {
  display: flex;
  gap: 16px;
}
.check {
  display: flex !important;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary) !important;
  font-size: var(--text-sm) !important;
}
footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
</style>
