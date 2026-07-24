<script setup lang="ts">
/**
 * 过渡页 — 行为对齐旧站；视觉：Canvas 浩瀚星空 + 跃迁舱
 *
 * 1. 首页 target=_blank → 新标签打开本页
 * 2. 倒计时结束 / 立即前往 → location.replace(外链)
 * 3. 首页标签保持
 */
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { apiPost } from '@/shared/api/client'
import type { Website } from '@/shared/types/models'
import SpaceSky from '@/shared/ui/SpaceSky.vue'

const SKIP_KEY = 'booknav_skip_transition'

const route = useRoute()
const site = ref<Website | null>(null)
const views = ref<number | null>(null)
const countdown = ref(0)
const totalSeconds = ref(0)
const error = ref('')
const rememberChoice = ref(true)
const showDesc = ref(true)
const themeColor = ref('#3de7ff')
const ad1 = ref('')
const ad2 = ref('')
const skipChecked = ref(false)
const progress = ref(0)
const jumping = ref(false)
const iconBroken = ref(false)

let rafId = 0
let startMs = 0
let ended = false
let targetUrl = ''

const skipRemembered = computed(() => {
  try {
    return localStorage.getItem(SKIP_KEY) === '1'
  } catch {
    return false
  }
})

const ringOffset = computed(() => {
  const c = 2 * Math.PI * 52
  return c * (1 - Math.min(progress.value, 100) / 100)
})

const statusLabel = computed(() => {
  if (jumping.value) return '跃迁启动'
  if (countdown.value <= 0) return '校准完成'
  return '航道校准中'
})

function goTarget(url: string) {
  if (ended || !url) return
  ended = true
  jumping.value = true
  if (rafId) cancelAnimationFrame(rafId)
  window.location.replace(url)
}

function persistSkipIfNeeded() {
  if (!rememberChoice.value || !skipChecked.value) return
  try {
    localStorage.setItem(SKIP_KEY, '1')
  } catch {
    /* ignore */
  }
}

function jumpNow() {
  persistSkipIfNeeded()
  goTarget(targetUrl || site.value?.url || '')
}

function tick() {
  if (ended) return
  const elapsed = (Date.now() - startMs) / 1000
  const total = totalSeconds.value
  const percent = Math.min((elapsed / total) * 100, 100)
  progress.value = percent
  const remain = Math.ceil(total - elapsed)
  countdown.value = remain > 0 ? remain : 0
  if (elapsed >= total) {
    persistSkipIfNeeded()
    goTarget(targetUrl)
    return
  }
  rafId = requestAnimationFrame(tick)
}

async function load() {
  const id = route.params.id
  try {
    const data = await apiPost<{
      website: Website
      enable_transition: boolean
      countdown: number
      remember_choice?: boolean
      show_description?: boolean
      color?: string
      ad1?: string
      ad2?: string
    }>(`/api/v1/portal/websites/${id}/visit`)

    site.value = data.website
    targetUrl = data.website.url
    views.value = typeof data.website.views === 'number' ? data.website.views : null
    rememberChoice.value = data.remember_choice !== false
    showDesc.value = data.show_description !== false
    if (data.color) themeColor.value = data.color
    ad1.value = data.ad1 || ''
    ad2.value = data.ad2 || ''
    iconBroken.value = false

    if (!data.enable_transition || data.countdown <= 0 || skipRemembered.value) {
      goTarget(data.website.url)
      return
    }

    totalSeconds.value = data.countdown
    countdown.value = data.countdown
    startMs = Date.now()
    rafId = requestAnimationFrame(tick)

    try {
      const origin = new URL(data.website.url).origin
      const link = document.createElement('link')
      link.rel = 'dns-prefetch'
      link.href = origin
      document.head.appendChild(link)
    } catch {
      /* ignore */
    }
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '加载失败'
  }
}

onMounted(load)
onUnmounted(() => {
  if (rafId) cancelAnimationFrame(rafId)
})
</script>

<template>
  <div class="warp" :style="{ '--warp-accent': themeColor }" :class="{ 'warp--launch': jumping }">
    <SpaceSky intensity="full" />

    <main class="stage">
      <div class="bay">
        <div class="bay__corners" aria-hidden="true">
          <i class="cn cn-tl" /><i class="cn cn-tr" /><i class="cn cn-bl" /><i class="cn cn-br" />
        </div>

        <header class="bay__head">
          <span class="bay__code">WARP · JUMP</span>
          <span class="bay__status">{{ statusLabel }}</span>
        </header>

        <p class="bay__banner">您即将离开本站，进入外部航道</p>

        <div class="bay__body">
          <p v-if="error" class="error">{{ error }}</p>

          <template v-else-if="site">
            <div class="target">
              <div class="target__icon">
                <img
                  v-if="site.icon && !iconBroken"
                  :src="site.icon"
                  alt=""
                  referrerpolicy="no-referrer"
                  @error="iconBroken = true"
                />
                <span v-else class="target__letter">{{ (site.title || '?').slice(0, 1) }}</span>
              </div>
              <div class="target__meta">
                <h1 class="target__title">{{ site.title }}</h1>
                <p class="target__url">{{ site.url }}</p>
              </div>
            </div>

            <div v-if="showDesc && site.description" class="brief">
              {{ site.description }}
            </div>

            <div v-if="ad1" class="ad" v-html="ad1" />

            <section class="launch">
              <div class="launch__ring">
                <svg viewBox="0 0 120 120" aria-hidden="true">
                  <circle class="launch__track" cx="60" cy="60" r="52" />
                  <circle
                    class="launch__prog"
                    cx="60"
                    cy="60"
                    r="52"
                    :style="{ strokeDashoffset: ringOffset }"
                  />
                </svg>
                <div class="launch__core">
                  <span class="launch__num">{{ jumping ? 'GO' : countdown }}</span>
                  <span class="launch__unit">{{ jumping ? 'JUMP' : 'SEC' }}</span>
                </div>
              </div>

              <div
                class="launch__bar"
                role="progressbar"
                :aria-valuenow="Math.round(progress)"
                aria-valuemin="0"
                aria-valuemax="100"
              >
                <div class="launch__bar-fill" :style="{ width: progress + '%' }" />
              </div>
              <p class="launch__hint">
                将在 <em>{{ countdown }}</em> 秒后自动跃迁
              </p>

              <button type="button" class="launch__btn" :disabled="jumping" @click="jumpNow">
                {{ jumping ? '点火中…' : '立即跃迁' }}
              </button>
            </section>

            <div v-if="ad2" class="ad" v-html="ad2" />
          </template>

          <p v-else class="muted">航道加载中…</p>
        </div>

        <footer class="bay__foot">
          <span v-if="views != null" class="bay__views">访问 {{ views }}</span>
          <span v-else />
          <label v-if="rememberChoice" class="skip">
            <input v-model="skipChecked" type="checkbox" />
            不再显示过渡页
          </label>
        </footer>
      </div>

      <p class="stage__back">
        <RouterLink to="/">← 返回机库</RouterLink>
      </p>
    </main>
  </div>
</template>

<style scoped>
.warp {
  --warp-accent: #3de7ff;
  --ring: 326.726;
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  background: #040b18;
  color: #e8f1ff;
  font-family: 'Noto Sans SC', 'Segoe UI', system-ui, sans-serif;
}

.stage {
  position: relative;
  z-index: 2;
  width: min(440px, 100%);
  margin: 0 auto;
  padding: 32px 16px 28px;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  min-height: 100vh;
  justify-content: center;
  box-sizing: border-box;
}

.bay {
  position: relative;
  border-radius: 16px;
  background: linear-gradient(165deg, rgba(16, 28, 44, 0.9), rgba(8, 14, 24, 0.94));
  border: 1px solid color-mix(in srgb, var(--warp-accent) 22%, rgba(100, 140, 180, 0.2));
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.35),
    0 28px 80px rgba(0, 0, 0, 0.55),
    0 0 48px color-mix(in srgb, var(--warp-accent) 10%, transparent);
  backdrop-filter: blur(14px) saturate(1.1);
  overflow: hidden;
}
.bay__corners {
  pointer-events: none;
  position: absolute;
  inset: 0;
}
.cn {
  position: absolute;
  width: 14px;
  height: 14px;
  border-color: color-mix(in srgb, var(--warp-accent) 65%, transparent);
  border-style: solid;
}
.cn-tl {
  top: 10px;
  left: 10px;
  border-width: 2px 0 0 2px;
}
.cn-tr {
  top: 10px;
  right: 10px;
  border-width: 2px 2px 0 0;
}
.cn-bl {
  bottom: 10px;
  left: 10px;
  border-width: 0 0 2px 2px;
}
.cn-br {
  bottom: 10px;
  right: 10px;
  border-width: 0 2px 2px 0;
}

.bay__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px 12px;
  border-bottom: 1px solid rgba(90, 130, 170, 0.16);
}
.bay__code {
  font-family: Orbitron, 'Segoe UI', sans-serif;
  font-size: 11px;
  letter-spacing: 0.16em;
  font-weight: 600;
  color: var(--warp-accent);
}
.bay__status {
  font-size: 11px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  color: #7a92ad;
}

.bay__banner {
  margin: 0;
  padding: 10px 20px;
  font-size: 13px;
  color: #b8c9de;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--warp-accent) 12%, transparent),
    transparent 85%
  );
  border-bottom: 1px solid rgba(90, 130, 170, 0.12);
}

.bay__body {
  padding: 18px 20px 8px;
}

.target {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  margin-bottom: 14px;
  border-radius: 12px;
  background: rgba(5, 12, 22, 0.65);
  border: 1px solid rgba(90, 130, 170, 0.2);
}
.target__icon {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  border-radius: 12px;
  overflow: hidden;
  display: grid;
  place-items: center;
  background: #0a1420;
  border: 1px solid color-mix(in srgb, var(--warp-accent) 30%, transparent);
}
.target__icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.target__letter {
  font-family: Orbitron, sans-serif;
  font-weight: 700;
  color: var(--warp-accent);
  font-size: 1.1rem;
}
.target__meta {
  min-width: 0;
  flex: 1;
}
.target__title {
  margin: 0 0 4px;
  font-size: 1.05rem;
  font-weight: 650;
  color: #f0f6ff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.target__url {
  margin: 0;
  font-size: 12px;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
  color: #5c7390;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.brief {
  margin: 0 0 16px;
  padding: 12px 14px;
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.55;
  color: #8fa6c2;
  background: rgba(5, 12, 22, 0.45);
  border: 1px solid rgba(90, 130, 170, 0.14);
  max-height: 96px;
  overflow: auto;
}

.ad {
  margin: 0 0 12px;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 12px;
  color: #8fa6c2;
  border: 1px dashed rgba(90, 130, 170, 0.22);
  background: rgba(0, 0, 0, 0.18);
}

.launch {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 8px 0 18px;
}

.launch__ring {
  position: relative;
  width: 132px;
  height: 132px;
}
.launch__ring svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}
.launch__track {
  fill: none;
  stroke: rgba(90, 130, 170, 0.18);
  stroke-width: 5;
}
.launch__prog {
  fill: none;
  stroke: var(--warp-accent);
  stroke-width: 5;
  stroke-linecap: round;
  stroke-dasharray: var(--ring);
  filter: drop-shadow(0 0 8px color-mix(in srgb, var(--warp-accent) 55%, transparent));
  transition: stroke-dashoffset 0.08s linear;
}
.launch__core {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.launch__num {
  font-family: Orbitron, sans-serif;
  font-size: 2.1rem;
  font-weight: 700;
  line-height: 1;
  color: var(--warp-accent);
  text-shadow: 0 0 20px color-mix(in srgb, var(--warp-accent) 50%, transparent);
}
.launch__unit {
  margin-top: 4px;
  font-size: 10px;
  letter-spacing: 0.2em;
  color: #5c7390;
  font-family: Orbitron, sans-serif;
}

.launch__bar {
  width: 100%;
  height: 6px;
  border-radius: 999px;
  background: rgba(10, 18, 30, 0.9);
  border: 1px solid rgba(90, 130, 170, 0.2);
  overflow: hidden;
}
.launch__bar-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--warp-accent) 50%, #0a2030),
    var(--warp-accent)
  );
  box-shadow: 0 0 12px color-mix(in srgb, var(--warp-accent) 35%, transparent);
  transition: width 0.08s linear;
}

.launch__hint {
  margin: 0;
  font-size: 14px;
  color: #8fa6c2;
  text-align: center;
}
.launch__hint em {
  font-style: normal;
  font-family: Orbitron, sans-serif;
  font-weight: 700;
  font-size: 1.15em;
  color: var(--warp-accent);
  margin: 0 2px;
}

.launch__btn {
  width: 100%;
  height: 44px;
  margin-top: 2px;
  border: 1px solid color-mix(in srgb, var(--warp-accent) 50%, transparent);
  border-radius: 10px;
  font-family: Orbitron, sans-serif;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.12em;
  color: #041018;
  cursor: pointer;
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--warp-accent) 95%, #fff),
    var(--warp-accent)
  );
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.2),
    0 8px 24px color-mix(in srgb, var(--warp-accent) 25%, transparent);
  transition:
    filter 0.15s,
    transform 0.15s,
    box-shadow 0.15s;
}
.launch__btn:hover:not(:disabled) {
  filter: brightness(1.06);
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.2),
    0 10px 32px color-mix(in srgb, var(--warp-accent) 35%, transparent);
}
.launch__btn:active:not(:disabled) {
  transform: scale(0.98);
}
.launch__btn:disabled {
  opacity: 0.75;
  cursor: wait;
}

.bay__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 18px 14px;
  border-top: 1px solid rgba(90, 130, 170, 0.14);
  font-size: 12px;
  color: #5c7390;
}
.bay__views {
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}
.skip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  user-select: none;
  color: #8fa6c2;
}
.skip input {
  accent-color: var(--warp-accent);
}

.stage__back {
  margin: 18px 0 0;
  text-align: center;
  font-size: 13px;
}
.stage__back a {
  color: #5c7390;
  text-decoration: none;
}
.stage__back a:hover {
  color: var(--warp-accent);
}

.error {
  color: #ff6b8a;
  text-align: center;
}
.muted {
  color: #5c7390;
  text-align: center;
  padding: 28px;
}

.warp--launch .launch__num {
  animation: pulse-go 0.45s ease infinite alternate;
}
@keyframes pulse-go {
  from {
    opacity: 0.7;
  }
  to {
    opacity: 1;
    text-shadow: 0 0 28px var(--warp-accent);
  }
}
</style>
