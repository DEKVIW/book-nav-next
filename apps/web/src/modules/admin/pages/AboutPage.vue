<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { apiGet } from '@/shared/api/client'
import AppIcon from '@/shared/ui/AppIcon.vue'

/** Project identity & author links (static product metadata). */
const PROJECT = {
  name: 'BookNav Next',
  tagline: '自托管个人网址导航',
  stack: 'Go · Vue 3 · SQLite · Docker',
  author: 'yilan',
  year: 2026,
  license: '见源码仓库',
} as const

type LinkItem = {
  key: string
  label: string
  href: string
  icon: string
  mono?: string
}

const links: LinkItem[] = [
  {
    key: 'github',
    label: '源码仓库',
    href: 'https://github.com/DEKVIW/book-nav-next',
    icon: 'github',
  },
  {
    key: 'blog',
    label: '作者博客',
    href: 'https://blog.yilanapp.com/',
    icon: 'globe',
  },
  {
    key: 'docker',
    label: '容器镜像',
    href: 'https://hub.docker.com/r/yilan666/booknav-next',
    icon: 'package',
    mono: 'yilan666/booknav-next',
  },
  {
    key: 'issues',
    label: '问题反馈',
    href: 'https://github.com/DEKVIW/book-nav-next/issues',
    icon: 'external-link',
  },
  {
    key: 'legacy',
    label: '旧版仓库（已停更）',
    href: 'https://github.com/DEKVIW/book-nav',
    icon: 'archive',
  },
]

type VersionInfo = {
  version?: string
  commit?: string
  build_time?: string
  uptime_sec?: number
}

const loading = ref(true)
const version = ref<VersionInfo>({})

onMounted(async () => {
  try {
    version.value = (await apiGet<VersionInfo>('/api/v1/version')) || {}
  } catch {
    version.value = {}
  } finally {
    loading.value = false
  }
})

const versionLabel = computed(() => version.value.version || '—')
const commitLabel = computed(() => {
  const c = version.value.commit
  if (!c || c === 'unknown') return '—'
  return c.length > 12 ? c.slice(0, 7) : c
})
const buildTimeLabel = computed(() => {
  const t = version.value.build_time
  if (!t || t === 'unknown') return '—'
  return t
})
const uptimeLabel = computed(() => {
  const s = version.value.uptime_sec
  if (s == null || Number.isNaN(s)) return '—'
  if (s < 60) return `${s}s`
  if (s < 3600) return `${Math.floor(s / 60)}m ${s % 60}s`
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  return `${h}h ${m}m`
})
</script>

<template>
  <div class="admin-page about-page">
    <header class="page-header">
      <h1>关于</h1>
    </header>

    <section class="about-hero c-card">
      <div class="about-hero__mark" aria-hidden="true">
        <AppIcon name="compass" :size="28" />
      </div>
      <div class="about-hero__body">
        <h2 class="about-hero__title">{{ PROJECT.name }}</h2>
        <p class="about-hero__tag">{{ PROJECT.tagline }} · {{ PROJECT.stack }}</p>
      </div>
    </section>

    <section class="c-card">
      <div class="c-card__body">
        <h3 class="c-card__title">运行信息</h3>
        <div v-if="loading" class="c-empty c-empty--sm">加载中…</div>
        <dl v-else class="about-meta">
          <div class="about-meta__row">
            <dt>版本</dt>
            <dd class="mono">{{ versionLabel }}</dd>
          </div>
          <div class="about-meta__row">
            <dt>Commit</dt>
            <dd class="mono">{{ commitLabel }}</dd>
          </div>
          <div class="about-meta__row">
            <dt>构建时间</dt>
            <dd class="mono">{{ buildTimeLabel }}</dd>
          </div>
          <div class="about-meta__row">
            <dt>运行时长</dt>
            <dd class="mono">{{ uptimeLabel }}</dd>
          </div>
        </dl>
      </div>
    </section>

    <section class="c-card">
      <div class="c-card__body">
        <h3 class="c-card__title">相关链接</h3>
        <ul class="about-links">
          <li v-for="item in links" :key="item.key">
            <a
              class="about-link"
              :href="item.href"
              target="_blank"
              rel="noopener noreferrer"
            >
              <span class="about-link__icon">
                <AppIcon :name="item.icon" :size="18" />
              </span>
              <span class="about-link__text">
                <strong>{{ item.label }}</strong>
                <span v-if="item.mono" class="about-link__mono mono">{{ item.mono }}</span>
              </span>
              <AppIcon name="external-link" :size="14" class="about-link__ext" />
            </a>
          </li>
        </ul>
      </div>
    </section>

    <section class="c-card">
      <div class="c-card__body">
        <h3 class="c-card__title">作者</h3>
        <div class="about-author">
          <div class="about-author__avatar" aria-hidden="true">Y</div>
          <div class="about-author__name">{{ PROJECT.author }}</div>
        </div>
        <p class="about-copy">
          © {{ PROJECT.year }} {{ PROJECT.author }} · {{ PROJECT.license }}
        </p>
      </div>
    </section>
  </div>
</template>

<style scoped>
.about-hero {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  padding: 20px 22px;
}
.about-hero__mark {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
  color: var(--console-primary);
  background: var(--console-primary-soft);
  border: 1px solid color-mix(in srgb, var(--console-primary) 35%, transparent);
}
.about-hero__title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 650;
  letter-spacing: 0.02em;
}
.about-hero__tag {
  margin: 6px 0 0;
  color: var(--console-text-2);
  font-size: 13px;
}

.about-meta {
  margin: 0;
  display: grid;
  gap: 0;
}
.about-meta__row {
  display: grid;
  grid-template-columns: 100px 1fr;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--console-border);
  font-size: 13px;
}
.about-meta__row:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}
.about-meta__row:first-child {
  padding-top: 0;
}
.about-meta dt {
  margin: 0;
  color: var(--console-text-3);
  font-weight: 500;
}
.about-meta dd {
  margin: 0;
  color: var(--console-text);
  word-break: break-all;
}
.mono {
  font-family: var(--console-mono);
  font-size: 12.5px;
}

.about-links {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 8px;
}
.about-link {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--console-radius);
  border: 1px solid var(--console-border);
  background: var(--console-surface-2);
  color: inherit;
  text-decoration: none;
  transition:
    border-color 0.15s,
    background 0.15s;
}
.about-link:hover {
  border-color: color-mix(in srgb, var(--console-primary) 45%, var(--console-border));
  background: color-mix(in srgb, var(--console-primary) 8%, var(--console-surface-2));
}
.about-link__icon {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
  color: var(--console-primary);
  flex-shrink: 0;
}
.about-link__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}
.about-link__text strong {
  font-size: 14px;
  font-weight: 600;
}
.about-link__mono {
  margin-top: 2px;
  color: var(--console-text-2);
}
.about-link__ext {
  color: var(--console-text-3);
  flex-shrink: 0;
  opacity: 0.7;
}

.about-author {
  display: flex;
  gap: 12px;
  align-items: center;
}
.about-author__avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  font-weight: 700;
  font-size: 15px;
  color: #fff;
  background: linear-gradient(145deg, #6e8efb, #3d5afe);
  flex-shrink: 0;
}
.about-author__name {
  font-size: 16px;
  font-weight: 650;
}
.about-copy {
  margin: 16px 0 0;
  padding-top: 14px;
  border-top: 1px solid var(--console-border);
  font-size: 12px;
  color: var(--console-text-3);
}

.c-empty--sm {
  padding: 12px 0;
  font-size: 13px;
}
</style>
