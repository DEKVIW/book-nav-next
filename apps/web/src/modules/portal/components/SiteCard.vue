<script setup lang="ts">
import type { Website } from '@/shared/types/models'
import { useCardTooltip } from '@/shared/composables/useCardTooltip'

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
    :draggable="!!draggable"
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
  display: flex;
  gap: var(--space-3);
  align-items: flex-start;
  min-height: var(--card-min-height);
  padding: 14px 16px 16px;
  text-decoration: none;
  color: inherit;
  cursor: pointer; /* 可点打开；draggable 时由 cursors.css 覆盖为 grab */
  overflow: hidden;
  transition:
    border-color var(--dur-fast) var(--ease-out),
    box-shadow var(--dur-fast) var(--ease-out),
    transform var(--dur-fast) var(--ease-out);
  user-select: none;
}
.site-card:hover {
  border-color: var(--stroke-bright);
  box-shadow: var(--glow-sm), var(--panel-shadow);
  transform: translateY(-1px);
}
.site-card:active {
  transform: scale(0.985);
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
    linear-gradient(135deg, rgba(61, 231, 255, 0.08), transparent 60%),
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
  color: var(--energy);
  text-shadow: 0 0 12px var(--energy-glow);
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
  font-size: 12px;
  color: var(--text-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.45;
}

.site-card__rail {
  position: absolute;
  left: 14px;
  right: 14px;
  bottom: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--energy-dim), transparent);
  opacity: 0.45;
  transition: opacity var(--dur-fast);
  z-index: 1;
}
.site-card:hover .site-card__rail {
  opacity: 1;
  background: linear-gradient(90deg, transparent, var(--energy), transparent);
  box-shadow: 0 0 12px var(--energy-glow);
}
</style>
