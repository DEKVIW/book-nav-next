<script setup lang="ts">
import { watch } from 'vue'
import { useRoute } from 'vue-router'
import { useCardTooltip, hideCardTooltip } from '@/shared/composables/useCardTooltip'

const { visible, text, title, x, y } = useCardTooltip()
const route = useRoute()

// SPA 路由切换时前台卡片会卸载，mouseleave 不一定触发 → 必须清全局 tip
watch(
  () => route.fullPath,
  () => hideCardTooltip(),
)
</script>

<template>
  <Teleport to="body">
    <div
      v-show="visible"
      class="m-tooltip hull"
      :class="{ 'is-visible': visible }"
      :style="{ left: `${x}px`, top: `${y}px` }"
      role="tooltip"
    >
      <div class="hull-corners" aria-hidden="true">
        <span class="c-tl" /><span class="c-tr" /><span class="c-bl" /><span class="c-br" />
      </div>
      <span v-if="title" class="m-tooltip__title">{{ title }}</span>
      <span class="m-tooltip__body">{{ text }}</span>
    </div>
  </Teleport>
</template>

<style>
.m-tooltip {
  position: fixed;
  z-index: var(--z-tooltip);
  max-width: min(320px, 70vw);
  padding: 10px 12px;
  pointer-events: none;
  opacity: 0;
  transform: translate(-50%, -100%) translateY(-10px) scale(0.98);
  transition:
    opacity var(--dur-fast) var(--ease-out),
    transform var(--dur-fast) var(--ease-out);
  font-size: 12px;
  line-height: 1.45;
  color: var(--text-secondary);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent 40%),
    var(--bg-panel-elevated);
  border: 1px solid var(--stroke-mid);
  box-shadow: var(--glow-sm), var(--panel-shadow);
  clip-path: polygon(8px 0, 100% 0, 100% calc(100% - 8px), calc(100% - 8px) 100%, 0 100%, 0 8px);
}
.m-tooltip.is-visible {
  opacity: 1;
  transform: translate(-50%, -100%) translateY(-12px) scale(1);
}
.m-tooltip__title {
  display: block;
  margin-bottom: 4px;
  font-size: 11px;
  font-weight: 650;
  letter-spacing: 0.04em;
  color: var(--energy);
  font-family: var(--font-display);
}
.m-tooltip__body {
  display: block;
  color: var(--text-secondary);
}
.m-tooltip .hull-corners span {
  width: 8px;
  height: 8px;
  border-color: var(--energy-dim);
}
</style>
