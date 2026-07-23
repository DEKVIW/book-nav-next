<script setup lang="ts">
import { computed } from 'vue'
import type { Website } from '@/shared/types/models'
import { useCardTooltip } from '@/shared/composables/useCardTooltip'
import { skinForId } from '@/shared/mecha/skins'

const props = defineProps<{
  site: Website
  draggable?: boolean
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
</script>

<template>
  <a
    class="site-card hull"
    :class="{ 'site-card--drag': draggable }"
    :href="site.url"
    :data-id="site.id"
    :data-skin="skin.id"
    :draggable="!!draggable"
    :style="{
      '--skin-accent': skin.accent,
      '--skin-accent-dim': skin.accentDim,
      '--skin-tint': skin.tint,
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

    <!-- Mecha silhouette: bottom-left, fades upward -->
    <div
      class="site-card__mecha"
      aria-hidden="true"
      :style="{ backgroundImage: `url(${skin.silhouette})` }"
    />

    <div class="site-card__icon" aria-hidden="true">
      <img
        v-if="site.icon"
        :src="site.icon"
        alt=""
        loading="lazy"
        referrerpolicy="no-referrer"
        @error="($event.target as HTMLImageElement).style.display = 'none'"
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

    <div class="site-card__rail" aria-hidden="true" />
  </a>
</template>

<style scoped>
.site-card {
  --skin-accent: var(--energy);
  --skin-accent-dim: var(--energy-dim);
  --skin-tint: transparent;
  display: flex;
  gap: var(--space-3);
  align-items: flex-start;
  min-height: var(--card-min-height);
  padding: 14px 16px 16px;
  text-decoration: none;
  color: inherit;
  cursor: pointer;
  overflow: hidden;
  position: relative;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.03) 0%, transparent 28%),
    linear-gradient(135deg, var(--skin-tint), transparent 55%),
    var(--bg-panel-solid);
  transition:
    border-color var(--dur-fast) var(--ease-out),
    box-shadow var(--dur-fast) var(--ease-out),
    transform var(--dur-fast) var(--ease-out);
  user-select: none;
}
.site-card:hover {
  border-color: color-mix(in srgb, var(--skin-accent) 55%, var(--stroke-dim));
  box-shadow:
    0 0 0 1px var(--skin-accent-dim),
    0 0 20px color-mix(in srgb, var(--skin-accent) 18%, transparent),
    var(--panel-shadow);
  transform: translateY(-1px);
}
.site-card:active {
  transform: scale(0.985);
}

.site-card__mecha {
  pointer-events: none;
  position: absolute;
  left: -4px;
  bottom: -6px;
  width: 72px;
  height: 96px;
  z-index: 0;
  background-repeat: no-repeat;
  background-position: left bottom;
  background-size: contain;
  opacity: 0.22;
  filter: drop-shadow(0 0 8px var(--skin-accent-dim));
  -webkit-mask-image: linear-gradient(to top, #000 35%, transparent 95%);
  mask-image: linear-gradient(to top, #000 35%, transparent 95%);
  transition: opacity var(--dur-base) var(--ease-out);
}
.site-card:hover .site-card__mecha {
  opacity: 0.38;
}

.site-card__icon {
  position: relative;
  z-index: 1;
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--skin-accent) 14%, transparent), transparent 60%),
    var(--bg-inset);
  border: 1px solid var(--stroke-dim);
  clip-path: polygon(6px 0, 100% 0, 100% calc(100% - 6px), calc(100% - 6px) 100%, 0 100%, 0 6px);
  overflow: hidden;
}
.site-card__icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.site-card__letter {
  font-family: var(--font-display);
  font-size: 1rem;
  color: var(--skin-accent);
  text-shadow: 0 0 12px var(--skin-accent-dim);
}

.site-card__body {
  position: relative;
  z-index: 1;
  min-width: 0;
  flex: 1;
  padding-top: 1px;
}
.site-card__row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.site-card__title {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--text-primary);
}
.site-card__desc {
  margin: 6px 0 0;
  font-size: var(--text-xs);
  color: var(--text-muted);
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.site-card__rail {
  position: absolute;
  top: 10px;
  right: 0;
  bottom: 10px;
  width: 2px;
  background: linear-gradient(
    180deg,
    transparent,
    var(--skin-accent-dim),
    transparent
  );
  opacity: 0.7;
  z-index: 1;
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
  background: rgba(255, 176, 32, 0.1);
}
.badge--dead {
  color: var(--danger);
  border: 1px solid rgba(255, 77, 106, 0.35);
  background: rgba(255, 77, 106, 0.1);
}
</style>
