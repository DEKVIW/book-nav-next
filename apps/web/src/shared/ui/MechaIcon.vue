<script setup lang="ts">
/**
 * @deprecated Use AppIcon (Lucide). Kept as thin alias for residual imports.
 */
import AppIcon from './AppIcon.vue'

withDefaults(
  defineProps<{
    name: string
    size?: number
  }>(),
  { size: 18 },
)
</script>

<template>
  <AppIcon :name="name" :size="size" />
</template>
