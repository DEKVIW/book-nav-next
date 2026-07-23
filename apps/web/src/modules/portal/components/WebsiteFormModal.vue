<script setup lang="ts">
import { reactive, watch, ref } from 'vue'
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
}>()

const emit = defineEmits<{
  close: []
  submit: [payload: Record<string, unknown>]
  fetchMeta: [url: string]
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

watch(
  () => [props.open, props.site, props.initialUrl] as const,
  () => {
    if (!props.open) return
    iconBroken.value = false
    if (props.mode === 'edit' && props.site) {
      form.title = props.site.title
      form.url = props.site.url
      form.description = props.site.description || ''
      form.icon = props.site.icon || ''
      form.category_id = props.site.category_id ?? null
      form.is_private = props.site.is_private
      form.is_featured = props.site.is_featured
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
  // 新建且标题仍空时，失焦自动抓取
  if (props.mode === 'create' && !form.title.trim()) {
    emit('fetchMeta', u)
  }
}

defineExpose({
  applyMeta(meta: { title?: string; description?: string; icon_url?: string; url?: string }) {
    if (meta.title) form.title = String(meta.title)
    if (meta.description) form.description = String(meta.description)
    if (meta.icon_url) {
      form.icon = String(meta.icon_url)
      iconBroken.value = false
    }
    if (meta.url) form.url = String(meta.url)
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
      <header>
        <p class="hud-label">{{ mode === 'create' ? 'QUICK ADD' : 'EDIT' }}</p>
        <h2>{{ mode === 'create' ? '快速添加链接' : '编辑链接' }}</h2>
        <p class="sub">
          {{ mode === 'create' ? '粘贴网址后自动检测重复并抓取标题、描述与图标' : '修改后保存即可' }}
        </p>
      </header>

      <div v-if="loading" class="progress">
        <div class="progress__bar" />
        <span>{{ progressText || '正在处理…' }}</span>
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
              :disabled="!form.url || loading"
              @click="emit('fetchMeta', form.url)"
            >
              抓取信息
            </button>
          </div>
        </label>

        <label>
          <span>标题</span>
          <input v-model="form.title" class="m-input" required placeholder="自动填充或手动填写" />
        </label>

        <label>
          <span>描述</span>
          <textarea v-model="form.description" class="m-input area" rows="3" placeholder="可选" />
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
          <button type="submit" class="m-btn m-btn--primary" :disabled="loading">
            {{ loading ? '请稍候…' : mode === 'create' ? '添加' : '保存' }}
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
header {
  margin-bottom: 14px;
}
h2 {
  margin: 6px 0 4px;
  font-size: var(--text-lg);
}
.sub {
  margin: 0;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.5;
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
