<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import ToastHost from '@/shared/ui/ToastHost.vue'
import CardTooltipHost from '@/shared/ui/CardTooltipHost.vue'
import { useAuthStore } from '@/shared/stores/auth'

const auth = useAuthStore()
onMounted(() => {
  auth.fetchMe()
})
</script>

<template>
  <div class="mecha-void app-root">
    <RouterView />
    <ToastHost />
    <CardTooltipHost />
  </div>
</template>

<style scoped>
.app-root {
  min-height: 100vh;
}
</style>
