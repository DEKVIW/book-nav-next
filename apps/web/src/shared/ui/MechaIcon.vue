<script setup lang="ts">
import { computed } from 'vue'
import { MECHA_ICON_PATHS, type MechaIconName } from '@/shared/mecha/icons'

const props = withDefaults(
  defineProps<{
    name: MechaIconName | string
    size?: number
  }>(),
  { size: 18 },
)

const d = computed(() => MECHA_ICON_PATHS[props.name as MechaIconName] || MECHA_ICON_PATHS.bolt)
</script>

<template>
  <svg
    class="mecha-icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="currentColor"
    aria-hidden="true"
  >
    <path :d="d" />
  </svg>
</template>

<style scoped>
.mecha-icon {
  display: block;
  flex-shrink: 0;
  opacity: 0.92;
}
</style>
