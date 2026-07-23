<script setup lang="ts">
import { useCardTooltip } from '@/shared/composables/useCardTooltip'

const { visible, text, title, x, y } = useCardTooltip()
</script>

<template>
  <Teleport to="body">
    <div
      class="m-tooltip"
      :class="{ 'is-visible': visible }"
      :style="{ left: `${x}px`, top: `${y}px` }"
      role="tooltip"
    >
      <span v-if="title" class="m-tooltip__title">{{ title }}</span>
      {{ text }}
    </div>
  </Teleport>
</template>
