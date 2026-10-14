<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import AppIcon from '@/shared/ui/AppIcon.vue'
import { createSearchShare } from '@/shared/api/searchShares'
import { useToast } from '@/shared/composables/useToast'
import type { Website } from '@/shared/types/models'
import type { CreateSearchShareResponse, SearchShareMode } from '@/shared/types/searchShare'

const props = defineProps<{
  open: boolean
  query: string
  mode: SearchShareMode
  summary?: string
  websites: Website[]
  isAdmin: boolean
  guestMaxDays: number
  guestMaxItems: number
}>()

const emit = defineEmits<{
  close: []
  created: [result: CreateSearchShareResponse]
}>()

const toast = useToast()
const title = ref('')
const expiresInDays = ref(7)
const submitting = ref(false)
const result = ref<CreateSearchShareResponse | null>(null)
const includePrivate = ref(false)

const shareUrl = computed(() => {
  if (!result.value) return ''
  return `${window.location.origin}${result.value.share_path}`
})

const eligibleWebsites = computed(() =>
  props.websites.filter((site) => site.is_valid && (includePrivate.value || !site.is_private)),
)
const shareLimit = computed(() => (props.isAdmin ? 40 : Math.max(1, props.guestMaxItems || 20)))
const shareableWebsites = computed(() => eligibleWebsites.value.slice(0, shareLimit.value))
const publicCount = computed(() => shareableWebsites.value.length)
const filteredOutCount = computed(() => props.websites.length - eligibleWebsites.value.length)
const limitExcludedCount = computed(() => eligibleWebsites.value.length - shareableWebsites.value.length)
const shareableWebsiteIDs = computed(() => new Set(shareableWebsites.value.map((site) => site.id)))
const modeLabel = computed(() => (props.mode === 'ai' ? 'AI 搜索' : '普通搜索'))
const guestExpiryOptions = computed(() => {
  const options = [1, props.guestMaxDays]
  return [...new Set(options)].sort((left, right) => left - right)
})

watch(
  () => props.open,
  (open) => {
    if (open) {
      title.value = `${props.mode === 'ai' ? 'AI' : '搜索'}：${props.query}`
      expiresInDays.value = props.isAdmin ? 7 : props.guestMaxDays
      result.value = null
      submitting.value = false
      includePrivate.value = false
    }
  },
)

async function submit() {
  if (submitting.value || publicCount.value === 0) return
  submitting.value = true
  try {
    const created = await createSearchShare({
      title: title.value.trim() || props.query,
      query: props.query,
      mode: props.mode,
      summary: props.summary || '',
      website_ids: shareableWebsites.value.map((site) => site.id),
      expires_in_days: expiresInDays.value,
      include_private: props.isAdmin && includePrivate.value,
    })
    result.value = created
    emit('created', created)
    const excluded = created.excluded_count + filteredOutCount.value + limitExcludedCount.value
    if (excluded > 0) {
      toast.info(`已自动排除 ${excluded} 条不可分享或超出上限的链接`)
    }
    await copyLink()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '分享创建失败')
  } finally {
    submitting.value = false
  }
}

async function copyLink() {
  if (!shareUrl.value) return
  try {
    await navigator.clipboard.writeText(shareUrl.value)
    toast.success('分享链接已复制')
  } catch {
    toast.error('复制失败，请手动复制链接')
  }
}

function openPreview() {
  if (shareUrl.value) window.open(shareUrl.value, '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <div v-if="open" class="share-dialog-mask" @click.self="emit('close')">
    <section class="share-dialog" role="dialog" aria-modal="true" aria-labelledby="share-dialog-title">
      <header class="share-dialog__head">
        <div>
          <p class="share-dialog__eyebrow"><AppIcon name="link" :size="14" /> SEARCH SHARE</p>
          <h2 id="share-dialog-title">分享搜索结果</h2>
        </div>
        <button type="button" class="share-dialog__close" aria-label="关闭" @click="emit('close')">
          <AppIcon name="x" :size="18" />
        </button>
      </header>

      <div v-if="!result" class="share-dialog__body">
        <div class="share-dialog__summary">
          <div class="share-dialog__query">
            <span class="share-dialog__label">搜索词</span>
            <strong>{{ query }}</strong>
          </div>
          <span class="share-dialog__mode" :class="{ 'is-ai': mode === 'ai' }">{{ modeLabel }}</span>
        </div>

        <label class="share-dialog__field">
          <span>分享标题</span>
          <input v-model="title" maxlength="80" type="text" placeholder="给这份结果起个名字" />
        </label>

        <label class="share-dialog__field">
          <span>有效期</span>
          <select v-model.number="expiresInDays">
            <option v-for="days in props.isAdmin ? [1, 7] : guestExpiryOptions" :key="days" :value="days">{{ days }} 天</option>
            <option v-if="props.isAdmin" :value="30">30 天</option>
            <option v-if="props.isAdmin" :value="365">365 天</option>
            <option v-if="props.isAdmin" :value="0">永久</option>
          </select>
        </label>

        <label v-if="props.isAdmin && websites.some((site) => site.is_private)" class="share-dialog__toggle">
          <input v-model="includePrivate" type="checkbox" />
          <span>
            <strong>包含私人链接</strong>
            <small>任何获得分享链接的人都可以访问这些私人网站。</small>
          </span>
        </label>

        <div class="share-dialog__notice">
          <AppIcon name="shield" :size="16" />
          <span v-if="includePrivate">已包含私人链接，请只把链接发送给可信任的人；失效链接仍会自动排除。</span>
          <span v-else>只会分享公开且有效的链接，私人链接和失效链接会被自动排除。</span>
        </div>

        <div class="share-dialog__results">
          <div class="share-dialog__results-head">
            <span>结果预览</span>
            <strong>{{ publicCount }} / {{ websites.length }}</strong>
          </div>
          <div class="share-dialog__result-list">
            <div v-for="site in websites.slice(0, 8)" :key="site.id" class="share-dialog__result-row">
              <img v-if="site.icon" :src="site.icon" alt="" loading="lazy" referrerpolicy="no-referrer" />
              <span v-else class="share-dialog__result-letter">{{ site.title.slice(0, 1).toUpperCase() }}</span>
              <span class="share-dialog__result-title">{{ site.title }}</span>
              <span v-if="!site.is_valid || (site.is_private && !includePrivate) || !shareableWebsiteIDs.has(site.id)" class="share-dialog__result-state">
                {{ !site.is_valid ? '已失效' : site.is_private && !includePrivate ? '私人' : '超出上限' }}
              </span>
            </div>
          </div>
          <p v-if="filteredOutCount > 0" class="share-dialog__more">
            将自动排除 {{ filteredOutCount }} 条私人或失效链接
          </p>
          <p v-if="limitExcludedCount > 0" class="share-dialog__more">
            当前最多分享 {{ shareLimit }} 条，超出部分将按当前顺序排除
          </p>
          <p v-if="websites.length > 8" class="share-dialog__more">还有 {{ websites.length - 8 }} 条结果</p>
        </div>
      </div>

      <div v-else class="share-dialog__success">
        <div class="share-dialog__success-icon"><AppIcon name="check" :size="24" /></div>
        <h3>分享链接已生成</h3>
        <p>已收录 {{ result.item_count }} 条公开结果，可在后台分享管理中撤销。</p>
        <div class="share-dialog__url">
          <span>{{ shareUrl }}</span>
          <button type="button" class="m-btn m-btn--ghost" @click="copyLink">
            <AppIcon name="link" :size="15" /> 复制
          </button>
        </div>
        <div class="share-dialog__actions">
          <button type="button" class="m-btn m-btn--primary" @click="openPreview">
            <AppIcon name="external-link" :size="15" /> 打开预览
          </button>
          <button type="button" class="m-btn m-btn--ghost" @click="emit('close')">完成</button>
        </div>
      </div>

      <footer v-if="!result" class="share-dialog__foot">
        <button type="button" class="m-btn m-btn--ghost" @click="emit('close')">取消</button>
        <button type="button" class="m-btn m-btn--primary" :disabled="submitting || publicCount === 0" @click="submit">
          <AppIcon name="link" :size="15" /> {{ submitting ? '生成中…' : '生成分享链接' }}
        </button>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.share-dialog-mask {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgba(2, 7, 15, 0.72);
  backdrop-filter: blur(10px);
}
.share-dialog {
  width: min(560px, 100%);
  max-height: min(760px, calc(100vh - 40px));
  overflow: auto;
  color: var(--text-primary);
  background: var(--bg-panel-solid);
  border: 1px solid var(--stroke-mid);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.42), 0 0 0 1px color-mix(in srgb, var(--energy) 12%, transparent);
}
.share-dialog__head,
.share-dialog__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--stroke-dim);
}
.share-dialog__foot {
  justify-content: flex-end;
  border-top: 1px solid var(--stroke-dim);
  border-bottom: 0;
}
.share-dialog__eyebrow {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0 0 5px;
  color: var(--energy);
  font-size: 11px;
  letter-spacing: 0.16em;
  font-family: var(--font-display);
}
.share-dialog h2,
.share-dialog h3 {
  margin: 0;
  color: var(--text-primary);
}
.share-dialog h2 { font-size: 20px; }
.share-dialog__close {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid var(--stroke-dim);
  cursor: pointer;
}
.share-dialog__close:hover { color: var(--energy); border-color: var(--energy); }
.share-dialog__body,
.share-dialog__success { padding: 20px; }
.share-dialog__summary {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  background: var(--bg-inset);
  border: 1px solid var(--stroke-dim);
}
.share-dialog__query { min-width: 0; display: grid; gap: 4px; }
.share-dialog__label { color: var(--text-muted); font-size: 12px; }
.share-dialog__query strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.share-dialog__mode,
.share-dialog__result-state {
  flex: 0 0 auto;
  color: var(--energy);
  font-size: 12px;
  border: 1px solid color-mix(in srgb, var(--energy) 36%, transparent);
  padding: 4px 8px;
}
.share-dialog__mode.is-ai { color: #d09bff; border-color: rgba(208, 155, 255, 0.45); }
.share-dialog__field { display: grid; gap: 7px; margin-top: 16px; color: var(--text-secondary); font-size: 13px; }
.share-dialog__field input,
.share-dialog__field select {
  width: 100%;
  min-height: 40px;
  box-sizing: border-box;
  color: var(--text-primary);
  background: var(--bg-input);
  border: 1px solid var(--stroke-dim);
  padding: 0 12px;
  outline: none;
}
.share-dialog__field input:focus,
.share-dialog__field select:focus { border-color: var(--energy); box-shadow: 0 0 0 3px color-mix(in srgb, var(--energy) 15%, transparent); }
.share-dialog__toggle {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-top: 16px;
  padding: 12px;
  color: var(--text-primary);
  background: color-mix(in srgb, var(--amber) 9%, transparent);
  border: 1px solid color-mix(in srgb, var(--amber) 35%, var(--stroke-dim));
  font-size: 13px;
}
.share-dialog__toggle input { margin-top: 2px; accent-color: var(--amber); }
.share-dialog__toggle span { display: grid; gap: 4px; min-width: 0; }
.share-dialog__toggle small { color: var(--text-secondary); font-size: 11px; line-height: 1.5; }
.share-dialog__notice {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  margin-top: 16px;
  padding: 11px 12px;
  color: var(--text-secondary);
  background: color-mix(in srgb, var(--energy) 8%, transparent);
  border-left: 2px solid var(--energy);
  font-size: 12px;
  line-height: 1.55;
}
.share-dialog__notice svg { flex-shrink: 0; color: var(--energy); }
.share-dialog__results { margin-top: 18px; }
.share-dialog__results-head { display: flex; justify-content: space-between; margin-bottom: 8px; color: var(--text-secondary); font-size: 12px; }
.share-dialog__results-head strong { color: var(--energy); }
.share-dialog__result-list { display: grid; gap: 6px; max-height: 176px; overflow: auto; }
.share-dialog__result-row { display: flex; align-items: center; gap: 9px; min-width: 0; padding: 7px 9px; background: var(--bg-inset); border: 1px solid var(--stroke-dim); }
.share-dialog__result-row img,
.share-dialog__result-letter { width: 24px; height: 24px; flex: 0 0 24px; object-fit: cover; display: grid; place-items: center; color: var(--energy); background: var(--bg-panel); font-size: 11px; }
.share-dialog__result-title { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-primary); font-size: 13px; }
.share-dialog__result-state { margin-left: auto; padding: 2px 6px; color: var(--text-muted); border-color: var(--stroke-dim); font-size: 10px; }
.share-dialog__more { margin: 8px 0 0; color: var(--text-muted); font-size: 12px; }
.share-dialog__success { text-align: center; }
.share-dialog__success-icon { display: grid; place-items: center; width: 54px; height: 54px; margin: 4px auto 14px; color: var(--energy); border: 1px solid var(--energy); background: color-mix(in srgb, var(--energy) 10%, transparent); }
.share-dialog__success p { margin: 9px 0 18px; color: var(--text-secondary); font-size: 13px; }
.share-dialog__url { display: flex; align-items: center; gap: 8px; padding: 9px; text-align: left; background: var(--bg-inset); border: 1px solid var(--stroke-dim); }
.share-dialog__url span { min-width: 0; flex: 1; overflow: hidden; color: var(--text-secondary); font: 12px/1.4 var(--font-mono); text-overflow: ellipsis; white-space: nowrap; }
.share-dialog__actions { display: flex; justify-content: center; gap: 9px; margin-top: 18px; }
@media (max-width: 600px) {
  .share-dialog-mask { padding: 10px; }
  .share-dialog { max-height: calc(100vh - 20px); }
  .share-dialog__head, .share-dialog__body, .share-dialog__success, .share-dialog__foot { padding-left: 16px; padding-right: 16px; }
  .share-dialog__actions, .share-dialog__foot { flex-wrap: wrap; }
  .share-dialog__actions > *, .share-dialog__foot > * { flex: 1; }
}
@media (prefers-reduced-motion: reduce) {
  .share-dialog-mask { backdrop-filter: none; }
}
</style>
