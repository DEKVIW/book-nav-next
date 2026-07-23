<script setup lang="ts">
import { computed } from 'vue'
import { ICON_PATHS, type IconName } from './AdminIcons'

const props = withDefaults(
  defineProps<{ name: IconName; size?: number }>(),
  { size: 18 },
)

const d = computed(() => ICON_PATHS[props.name] || '')
</script>

<template>
  <svg
    class="admin-icon"
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
.admin-icon {
  display: block;
  flex-shrink: 0;
}
</style>
