<script setup lang="ts">
import { computed } from 'vue'
import { getLucideIcon } from '@/shared/icons/registry'

const props = withDefaults(
  defineProps<{
    name?: string | null
    size?: number
    strokeWidth?: number
  }>(),
  { size: 18, strokeWidth: 1.75 },
)

const Comp = computed(() => getLucideIcon(props.name))
</script>

<template>
  <component
    :is="Comp"
    class="app-icon"
    :size="size"
    :stroke-width="strokeWidth"
    :absolute-stroke-width="false"
    aria-hidden="true"
  />
</template>

<style scoped>
.app-icon {
  display: block;
  flex-shrink: 0;
}
</style>
