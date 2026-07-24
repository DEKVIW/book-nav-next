<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import ToastHost from '@/shared/ui/ToastHost.vue'
import CardTooltipHost from '@/shared/ui/CardTooltipHost.vue'
import SpaceSky from '@/shared/ui/SpaceSky.vue'
import { useAuthStore } from '@/shared/stores/auth'

const auth = useAuthStore()
const route = useRoute()

/** 前台星空：后台与过渡页自管背景时关闭，避免叠两层 */
const showAmbientSky = computed(() => {
  const p = route.path
  if (p.startsWith('/admin')) return false
  if (p.startsWith('/goto')) return false
  return true
})

onMounted(() => {
  auth.fetchMe()
})
</script>

<template>
  <div class="mecha-void app-root" :class="{ 'app-root--sky': showAmbientSky }">
    <SpaceSky v-if="showAmbientSky" intensity="full" />
    <RouterView />
    <ToastHost />
    <CardTooltipHost />
  </div>
</template>

<style scoped>
.app-root {
  min-height: 100vh;
  position: relative;
}
/* 有星空时略压低机库贴图，让星点/星云露出来 */
.app-root--sky :deep(.space-sky) {
  z-index: 0;
}
</style>
