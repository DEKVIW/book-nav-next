<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import type { Website } from '@/shared/types/models'
import { useCardTooltip } from '@/shared/composables/useCardTooltip'
import { skinForId } from '@/shared/mecha/skins'

const rootEl = ref<HTMLElement | null>(null)

const props = defineProps<{
  site: Website
  draggable?: boolean
  isLocated?: boolean
  isLocating?: boolean
}>()

const emit = defineEmits<{
  open: [site: Website]
  context: [e: MouseEvent, site: Website]
  dragstart: [e: DragEvent, site: Website]
  dragover: [e: DragEvent, site: Website]
  drop: [e: DragEvent, site: Website]
}>()

const tip = useCardTooltip()
const skin = computed(() => skinForId(props.site.id))
/** 图片加载失败时回退字母（原先只 hide img，圆形变空） */
const iconBroken = ref(false)

watch(
  () => props.site.icon,
  () => {
    iconBroken.value = false
  },
)

const showIcon = computed(() => !!(props.site.icon && String(props.site.icon).trim() && !iconBroken.value))

function letterFallback(title: string) {
  const t = title.trim()
  return t ? t.slice(0, 1).toUpperCase() : '?'
}

function onClick(e: MouseEvent) {
  e.preventDefault()
  tip.hide()
  emit('open', props.site)
}

function onContext(e: MouseEvent) {
  e.preventDefault()
  tip.hide()
  emit('context', e, props.site)
}

function onEnter(e: MouseEvent) {
  const desc = (props.site.description || '').trim()
  if (!desc) return
  tip.scheduleShow(e.currentTarget as HTMLElement, desc, props.site.title)
}

function onLeave() {
  tip.hide()
}

// 本卡卸载时若 tip 挂在本卡上才收起（避免列表 diff 误伤其它卡的 tip）
onBeforeUnmount(() => {
  tip.hideIf(rootEl.value)
})
</script>

<template>
  <a
    ref="rootEl"
    class="site-card hull"
    :class="{
      'site-card--drag': draggable,
      'site-card--located': isLocated,
      'site-card--locating': isLocating,
    }"
    :href="site.url"
    :data-id="site.id"
    :data-skin="skin.id"
    :draggable="!!draggable"
    :style="{
      '--skin-accent': skin.accent,
      '--skin-accent-dim': skin.accentDim,
      '--skin-wash': skin.wash,
      '--skin-art': `url(${skin.silhouette})`,
    }"
    @click="onClick"
    @contextmenu="onContext"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
    @dragstart="
      tip.hide();
      draggable && emit('dragstart', $event, site)
    "
    @dragover.prevent="draggable && emit('dragover', $event, site)"
    @drop.prevent="draggable && emit('drop', $event, site)"
  >
    <div class="hull-corners" aria-hidden="true">
      <span class="c-tl" /><span class="c-tr" /><span class="c-bl" /><span class="c-br" />
    </div>

    <!-- Art layer: absolute, top-right, OUT of document flow (never grows card) -->
    <span class="site-card__art" aria-hidden="true" />

    <!-- Readability scrim over left/mid content only -->
    <span class="site-card__scrim" aria-hidden="true" />

    <div class="site-card__icon" aria-hidden="true">
      <img
        v-if="showIcon"
        :src="site.icon"
        alt=""
        loading="lazy"
        referrerpolicy="no-referrer"
        @error="iconBroken = true"
      />
      <span v-else class="site-card__letter">{{ letterFallback(site.title) }}</span>
    </div>

    <div class="site-card__body">
      <div class="site-card__row">
        <h3 class="site-card__title">{{ site.title }}</h3>
        <span v-if="site.is_private" class="badge badge--private">PRIV</span>
        <span v-if="!site.is_valid" class="badge badge--dead">FAIL</span>
      </div>
      <p class="site-card__desc">{{ site.description || site.url }}</p>
    </div>
  </a>
</template>

<style scoped>
.site-card {
  --skin-accent: #5ef0ff;
  --skin-accent-dim: rgba(94, 240, 255, 0.45);
  --skin-wash: rgba(40, 140, 200, 0.12);
  /* fixed card metrics — art cannot affect height */
  height: 96px;
  min-height: 96px;
  max-height: 96px;
  box-sizing: border-box;
  display: flex;
  gap: 14px;
  align-items: center; /* 图标与文字垂直居中 */
  padding: 14px 16px;
  padding-right: 78px; /* reserve right for art */
  text-decoration: none;
  color: inherit;
  cursor: pointer;
  overflow: hidden;
  position: relative;
  isolation: isolate;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.05) 0%, transparent 40%),
    linear-gradient(90deg, rgba(10, 18, 28, 0.92) 0%, rgba(10, 18, 28, 0.75) 55%, rgba(10, 18, 28, 0.35) 100%),
    linear-gradient(135deg, var(--skin-wash), transparent 60%),
    var(--bg-panel-solid);
  border-color: color-mix(in srgb, var(--skin-accent) 22%, var(--stroke-dim));
  /* only color transitions — avoid layout-affecting transforms that jitter neighbors */
  transition: border-color 0.15s ease, box-shadow 0.15s ease, filter 0.15s ease;
  user-select: none;
}
.site-card:hover {
  border-color: color-mix(in srgb, var(--skin-accent) 65%, var(--stroke-dim));
  box-shadow:
    0 0 0 1px var(--skin-accent-dim),
    0 8px 28px rgba(0, 0, 0, 0.45),
    0 0 24px color-mix(in srgb, var(--skin-accent) 16%, transparent);
  filter: brightness(1.04);
}
.site-card:active {
  filter: brightness(0.98);
}
.site-card--located {
  border-color: var(--energy);
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--energy) 70%, transparent),
    0 8px 28px rgba(0, 0, 0, 0.45),
    0 0 30px color-mix(in srgb, var(--energy) 24%, transparent);
}
.site-card--located::before {
  content: '';
  position: absolute;
  z-index: 3;
  left: 0;
  top: 12px;
  bottom: 12px;
  width: 3px;
  background: var(--energy);
  box-shadow: 0 0 14px color-mix(in srgb, var(--energy) 70%, transparent);
}
.site-card--located .hull-corners span {
  border-color: var(--energy);
}
.site-card--locating {
  animation: site-card-locate-pulse 0.8s ease-in-out 4;
}
@keyframes site-card-locate-pulse {
  50% {
    border-color: var(--energy);
    box-shadow:
      0 0 0 1px color-mix(in srgb, var(--energy) 85%, transparent),
      0 8px 28px rgba(0, 0, 0, 0.45),
      0 0 42px color-mix(in srgb, var(--energy) 36%, transparent);
    filter: brightness(1.12);
  }
}
@media (prefers-reduced-motion: reduce) {
  .site-card--locating {
    animation: none;
  }
}

/* TOP-RIGHT action art — absolute, no layout size */
.site-card__art {
  pointer-events: none;
  position: absolute;
  z-index: 0;
  right: -6px;
  top: -4px;
  width: 92px;
  height: 100px;
  background-image: var(--skin-art);
  background-repeat: no-repeat;
  background-position: right top;
  background-size: contain;
  opacity: 0.78;
  filter: drop-shadow(0 0 10px var(--skin-accent-dim));
  /* clearest at top-right character area; fade toward content */
  -webkit-mask-image: linear-gradient(225deg, #000 28%, rgba(0, 0, 0, 0.55) 55%, transparent 88%);
  mask-image: linear-gradient(225deg, #000 28%, rgba(0, 0, 0, 0.55) 55%, transparent 88%);
  transition: opacity 0.15s ease;
}
.site-card:hover .site-card__art {
  opacity: 0.95;
}

/* extra text protection over left 65% */
.site-card__scrim {
  pointer-events: none;
  position: absolute;
  inset: 0;
  z-index: 0;
  background: linear-gradient(
    90deg,
    rgba(8, 14, 22, 0.55) 0%,
    rgba(8, 14, 22, 0.25) 48%,
    transparent 72%
  );
}

/* 圆形站点图标：更大、垂直居中、favicon 更清晰 */
.site-card__icon {
  position: relative;
  z-index: 2;
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background:
    linear-gradient(145deg, color-mix(in srgb, var(--skin-accent) 22%, transparent), transparent 70%),
    var(--bg-inset);
  border: 1px solid color-mix(in srgb, var(--skin-accent) 35%, var(--stroke-dim));
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.2), 0 4px 12px rgba(0, 0, 0, 0.25);
  overflow: hidden;
}
.site-card__icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 50%;
}
.site-card__letter {
  font-family: var(--font-display);
  font-size: 1.05rem;
  font-weight: 700;
  color: var(--skin-accent);
  text-shadow: 0 0 10px var(--skin-accent-dim);
}

.site-card__body {
  position: relative;
  z-index: 2;
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding-top: 0;
}
.site-card__row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.site-card__title {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: 650;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: #f2f7ff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.65);
}
.site-card__desc {
  margin: 5px 0 0;
  font-size: var(--text-xs);
  color: #c5d4e8;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.55);
}
.badge {
  font-size: 9px;
  font-family: var(--font-mono);
  letter-spacing: 0.06em;
  padding: 2px 5px;
  border-radius: 2px;
  flex-shrink: 0;
}
.badge--private {
  color: var(--amber);
  border: 1px solid var(--amber-dim);
  background: rgba(255, 176, 32, 0.12);
}
.badge--dead {
  color: var(--danger);
  border: 1px solid rgba(255, 77, 106, 0.4);
  background: rgba(255, 77, 106, 0.12);
}
</style>
