<script setup lang="ts">
/** @deprecated Prefer AppIcon */
import AppIcon from '@/shared/ui/AppIcon.vue'

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
