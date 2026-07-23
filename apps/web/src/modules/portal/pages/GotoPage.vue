<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { apiPost } from '@/shared/api/client'
import type { Website } from '@/shared/types/models'

const SKIP_KEY = 'booknav_skip_transition'

const route = useRoute()
const site = ref<Website | null>(null)
const countdown = ref(0)
const error = ref('')
const rememberChoice = ref(true)
const showDesc = ref(true)
const themeColor = ref('#6e8efb')
const ad1 = ref('')
const ad2 = ref('')
let timer: number | undefined

const skipRemembered = computed(() => {
  try {
    return localStorage.getItem(SKIP_KEY) === '1'
  } catch {
    return false
  }
})

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
    rememberChoice.value = data.remember_choice !== false
    showDesc.value = data.show_description !== false
    if (data.color) themeColor.value = data.color
    ad1.value = data.ad1 || ''
    ad2.value = data.ad2 || ''

    if (!data.enable_transition || data.countdown <= 0 || skipRemembered.value) {
      window.location.href = data.website.url
      return
    }
    countdown.value = data.countdown
    timer = window.setInterval(() => {
      countdown.value -= 1
      if (countdown.value <= 0) {
        window.clearInterval(timer)
        window.location.href = data.website.url
      }
    }, 1000)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '加载失败'
  }
}

function jumpNow() {
  if (site.value) window.location.href = site.value.url
}

function skipForever() {
  try {
    localStorage.setItem(SKIP_KEY, '1')
  } catch {
    /* ignore */
  }
  jumpNow()
}

onMounted(load)
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="goto-page" :style="{ '--goto-accent': themeColor }">
    <div class="hud panel">
      <h1>即将跳转</h1>
      <p v-if="error" class="error">{{ error }}</p>
      <template v-else-if="site">
        <p class="title">{{ site.title }}</p>
        <p class="url">{{ site.url }}</p>
        <p v-if="showDesc && site.description" class="desc">{{ site.description }}</p>
        <div v-if="ad1" class="ad" v-html="ad1" />
        <div class="ring">{{ countdown }}</div>
        <div class="actions">
          <button type="button" class="m-btn m-btn--primary" @click="jumpNow">立即前往</button>
          <button v-if="rememberChoice" type="button" class="m-btn m-btn--ghost" @click="skipForever">
            不再显示
          </button>
          <RouterLink class="m-btn m-btn--ghost" to="/">返回</RouterLink>
        </div>
        <div v-if="ad2" class="ad ad--bottom" v-html="ad2" />
      </template>
      <p v-else class="muted">加载中…</p>
    </div>
  </div>
</template>

<style scoped>
.goto-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  background: var(--bg-void, #05080f);
}
.hud {
  text-align: center;
  padding: var(--space-8, 2rem);
  width: min(440px, 92vw);
}
.title {
  font-size: 1.25rem;
  font-weight: 650;
  margin: 0.5rem 0 0.25rem;
}
.url {
  font-size: 12px;
  opacity: 0.55;
  word-break: break-all;
}
.desc {
  font-size: 13px;
  opacity: 0.75;
  margin: 0.75rem 0;
  line-height: 1.5;
}
.ring {
  width: 72px;
  height: 72px;
  margin: 1.25rem auto;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-size: 1.5rem;
  font-weight: 700;
  border: 2px solid var(--goto-accent, #6e8efb);
  color: var(--goto-accent, #6e8efb);
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}
.ad {
  margin: 12px 0;
  font-size: 12px;
  text-align: left;
}
.ad--bottom {
  margin-top: 16px;
  opacity: 0.85;
}
.error {
  color: #f87171;
}
.muted {
  opacity: 0.6;
}
</style>
