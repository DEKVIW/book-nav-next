<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import AppIcon from '@/shared/ui/AppIcon.vue'
import SpaceSky from '@/shared/ui/SpaceSky.vue'
import SiteCard from '@/modules/portal/components/SiteCard.vue'
import { getPublicSearchShare } from '@/shared/api/searchShares'
import { useTheme } from '@/shared/composables/useTheme'
import type { Website } from '@/shared/types/models'
import type { PublicSearchShareResponse } from '@/shared/types/searchShare'

type Phase = 'loading' | 'ready' | 'typing' | 'searching' | 'matching' | 'revealing' | 'complete' | 'error'

const route = useRoute()
const { resolved } = useTheme()
const data = ref<PublicSearchShareResponse | null>(null)
const phase = ref<Phase>('loading')
const revealedCount = ref(0)
const errorMessage = ref('')
const reducedMotion = ref(false)
let timers: number[] = []

const token = computed(() => String(route.params.token || ''))
const websites = computed<Website[]>(() => data.value?.websites || [])
const shownWebsites = computed(() => websites.value.slice(0, revealedCount.value))
const isAI = computed(() => data.value?.share.mode === 'ai')
const phaseText = computed(() => {
  if (phase.value === 'loading') return '正在接收分享'
  if (phase.value === 'ready') return '分享已就绪'
  if (phase.value === 'typing') return '正在输入搜索词'
  if (phase.value === 'searching') return isAI.value ? '正在理解搜索意图' : '正在检索导航'
  if (phase.value === 'matching') return isAI.value ? '正在匹配相关资源' : '正在整理结果'
  if (phase.value === 'revealing') return '结果正在展开'
  if (phase.value === 'complete') return '搜索结果已抵达'
  return '分享暂时无法打开'
})

function clearTimers() {
  for (const timer of timers) window.clearTimeout(timer)
  timers = []
}

function finishPlayback() {
  phase.value = 'complete'
  revealedCount.value = websites.value.length
}

function skipPlayback() {
  clearTimers()
  finishPlayback()
}

function startPlayback() {
  clearTimers()
  revealedCount.value = 0
  if (!data.value) return
  if (reducedMotion.value) {
    finishPlayback()
    return
  }

  phase.value = 'ready'
  const stages: Phase[] = isAI.value ? ['typing', 'searching', 'matching', 'revealing'] : ['typing', 'searching', 'revealing']
  stages.forEach((stage, index) => {
    timers.push(
      window.setTimeout(() => {
        phase.value = stage
        if (stage === 'revealing') {
          const count = websites.value.length
          for (let i = 1; i <= count; i += 1) {
            timers.push(
              window.setTimeout(() => {
                revealedCount.value = i
                if (i === count) phase.value = 'complete'
              }, i * 115),
            )
          }
        }
      }, 520 + index * 620),
    )
  })
}

function openWebsite(site: Website) {
  window.open(site.url, '_blank', 'noopener,noreferrer')
}

function replay() {
  startPlayback()
}

async function load() {
  phase.value = 'loading'
  errorMessage.value = ''
  try {
    data.value = await getPublicSearchShare(token.value)
    startPlayback()
  } catch (error) {
    phase.value = 'error'
    errorMessage.value = error instanceof Error ? error.message : '分享不存在或已失效'
  }
}

onMounted(() => {
  reducedMotion.value = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  void load()
})

onBeforeUnmount(clearTimers)
</script>

<template>
  <main class="share-page" :class="{ 'share-page--complete': phase === 'complete' }">
    <SpaceSky class="share-page__sky" intensity="ambient" :theme="resolved" />
    <div class="share-page__veil" aria-hidden="true" />

    <div class="share-shell">
      <header class="share-topbar">
        <RouterLink to="/" class="share-brand" aria-label="返回书签导航">
          <AppIcon name="bookmark" :size="17" />
          <span>BOOKNAV / SHARE</span>
        </RouterLink>
        <span class="share-topbar__status"><span class="share-topbar__dot" /> {{ phaseText }}</span>
      </header>

      <section v-if="phase !== 'error' && data" class="share-hero">
        <div class="share-hero__eyebrow"><AppIcon :name="isAI ? 'sparkles' : 'search'" :size="15" /> {{ isAI ? 'AI SEARCH TRANSMISSION' : 'SEARCH TRANSMISSION' }}</div>
        <h1>{{ data?.share.title || '一份搜索结果分享' }}</h1>
        <p class="share-hero__query">“{{ data?.share.query }}”</p>
        <p v-if="data?.share.summary && phase === 'complete'" class="share-hero__summary">{{ data.share.summary }}</p>
      </section>

      <section v-if="phase !== 'error' && data" class="share-console" aria-live="polite">
        <div class="share-console__line">
          <span class="share-console__label">{{ isAI ? 'AI PIPELINE' : 'NAVIGATION INDEX' }}</span>
          <span class="share-console__mode">{{ isAI ? 'HYBRID' : 'KEYWORD' }}</span>
        </div>
        <div class="share-console__progress"><span :style="{ width: `${phase === 'complete' ? 100 : Math.min(92, 15 + revealedCount / Math.max(websites.length, 1) * 70)}%` }" /></div>
        <div class="share-console__message"><span class="share-console__pulse" :class="{ 'is-done': phase === 'complete' }" /> {{ phaseText }}</div>
      </section>

      <section v-if="phase !== 'error' && shownWebsites.length" class="share-results" aria-label="分享结果">
        <div class="share-results__head">
          <div>
            <span class="share-results__eyebrow">PUBLIC RESULTS</span>
            <h2>已为你整理 {{ websites.length }} 条链接</h2>
          </div>
          <span class="share-results__count">{{ shownWebsites.length }} / {{ websites.length }}</span>
        </div>
        <div class="share-results-grid">
          <SiteCard
            v-for="(site, index) in shownWebsites"
            :key="site.id"
            :site="site"
            :is-located="phase === 'complete' && index === 0"
            :is-locating="phase === 'revealing' && index === shownWebsites.length - 1"
            @open="openWebsite"
          />
        </div>
      </section>

      <section v-if="phase === 'error'" class="share-error">
        <div class="share-error__icon"><AppIcon name="link" :size="26" /></div>
        <h1>这份分享已不可用</h1>
        <p>{{ errorMessage }}</p>
        <RouterLink to="/" class="m-btn m-btn--primary"><AppIcon name="home" :size="15" /> 返回导航</RouterLink>
      </section>

      <footer class="share-footer">
        <button v-if="phase !== 'error' && phase !== 'complete'" type="button" class="m-btn m-btn--ghost" @click="skipPlayback">
          <AppIcon name="chevrons-right" :size="15" /> 跳过演示
        </button>
        <button v-if="phase === 'complete'" type="button" class="m-btn m-btn--ghost" @click="replay">
          <AppIcon name="activity" :size="15" /> 重播演示
        </button>
        <RouterLink to="/" class="share-footer__back">打开我的导航</RouterLink>
      </footer>
    </div>
  </main>
</template>

<style scoped>
.share-page { position: relative; min-height: 100vh; overflow: hidden; color: var(--text-primary); background: var(--bg-app); }
.share-page__sky, .share-page__veil { position: fixed; inset: 0; pointer-events: none; }
.share-page__sky { z-index: 0; }
.share-page__veil { z-index: 1; background: radial-gradient(circle at 50% 15%, transparent 0 18%, color-mix(in srgb, var(--bg-app) 50%, transparent) 74%), linear-gradient(180deg, color-mix(in srgb, var(--bg-app) 35%, transparent), var(--bg-app)); }
.share-shell { position: relative; z-index: 2; width: min(1120px, calc(100% - 32px)); margin: 0 auto; padding: 22px 0 40px; }
.share-topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.share-brand, .share-footer__back { display: inline-flex; align-items: center; gap: 8px; color: var(--text-secondary); text-decoration: none; font: 11px var(--font-display); letter-spacing: .13em; }
.share-brand:hover, .share-footer__back:hover { color: var(--energy); }
.share-topbar__status { display: inline-flex; align-items: center; gap: 8px; color: var(--text-muted); font: 11px var(--font-mono); }
.share-topbar__dot { width: 7px; height: 7px; border-radius: 50%; background: var(--energy); box-shadow: 0 0 12px var(--energy); }
.share-hero { width: min(760px, 100%); margin: 11vh auto 30px; text-align: center; }
.share-hero__eyebrow, .share-results__eyebrow { color: var(--energy); font: 11px var(--font-display); letter-spacing: .18em; }
.share-hero__eyebrow { display: inline-flex; align-items: center; gap: 8px; }
.share-hero h1 { margin: 14px 0 10px; font-size: clamp(28px, 5vw, 52px); line-height: 1.12; letter-spacing: .02em; }
.share-hero__query { margin: 0; color: var(--text-secondary); font-size: 18px; }
.share-hero__summary { width: min(620px, 100%); margin: 16px auto 0; color: var(--text-secondary); line-height: 1.65; }
.share-console { width: min(680px, 100%); margin: 0 auto 34px; padding: 15px 17px; background: color-mix(in srgb, var(--bg-panel-solid) 86%, transparent); border: 1px solid var(--stroke-mid); box-shadow: 0 15px 50px rgba(0, 0, 0, .2); }
.share-console__line { display: flex; justify-content: space-between; gap: 12px; color: var(--text-muted); font: 11px var(--font-mono); }
.share-console__mode { color: var(--energy); }
.share-console__progress { height: 4px; margin: 12px 0; background: var(--bg-inset); overflow: hidden; }
.share-console__progress span { display: block; height: 100%; background: linear-gradient(90deg, var(--magenta), var(--energy)); transition: width .35s ease; }
.share-console__message { display: flex; align-items: center; gap: 9px; color: var(--text-secondary); font-size: 13px; }
.share-console__pulse { width: 8px; height: 8px; border-radius: 50%; background: var(--amber); box-shadow: 0 0 12px var(--amber); animation: share-pulse 1.15s ease-in-out infinite; }
.share-console__pulse.is-done { background: var(--energy); box-shadow: 0 0 12px var(--energy); animation: none; }
.share-results { padding: 22px; background: color-mix(in srgb, var(--bg-panel-solid) 80%, transparent); border: 1px solid var(--stroke-dim); }
.share-results__head { display: flex; align-items: end; justify-content: space-between; gap: 16px; margin-bottom: 17px; }
.share-results h2 { margin: 7px 0 0; font-size: 22px; }
.share-results__count { color: var(--energy); font: 12px var(--font-mono); }
.share-results-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.share-results-grid :deep(.site-card) { cursor: pointer; }
.share-footer { display: flex; align-items: center; justify-content: center; gap: 16px; padding-top: 24px; }
.share-error { width: min(520px, 100%); margin: 16vh auto 0; padding: 34px 24px; text-align: center; background: color-mix(in srgb, var(--bg-panel-solid) 84%, transparent); border: 1px solid var(--stroke-mid); }
.share-error__icon { display: grid; place-items: center; width: 58px; height: 58px; margin: 0 auto 16px; color: var(--danger); border: 1px solid color-mix(in srgb, var(--danger) 55%, transparent); }
.share-error h1 { margin: 0 0 10px; font-size: 25px; }
.share-error p { margin: 0 0 22px; color: var(--text-secondary); }
@keyframes share-pulse { 50% { opacity: .35; transform: scale(.72); } }
@media (max-width: 720px) {
  .share-shell { width: min(100% - 20px, 560px); padding-top: 15px; }
  .share-topbar { align-items: flex-start; }
  .share-topbar__status { max-width: 42%; text-align: right; line-height: 1.4; }
  .share-hero { margin-top: 9vh; }
  .share-hero h1 { font-size: 31px; }
  .share-results { padding: 15px; }
  .share-results-grid { grid-template-columns: 1fr; }
  .share-footer { flex-wrap: wrap; }
}
@media (prefers-reduced-motion: reduce) {
  .share-console__pulse { animation: none; }
  .share-console__progress span { transition: none; }
}
</style>
