<script setup lang="ts">
import { useToast } from '@/shared/composables/useToast'
const { toasts } = useToast()
</script>

<template>
  <div class="toast-host" aria-live="polite">
    <div
      v-for="t in toasts"
      :key="t.id"
      class="toast"
      :class="`toast--${t.kind}`"
    >
      {{ t.message }}
    </div>
  </div>
</template>

<style scoped>
.toast-host {
  position: fixed;
  right: 16px;
  top: 72px;
  z-index: 1000;
  display: grid;
  gap: 8px;
  max-width: min(360px, 92vw);
}
.toast {
  padding: 10px 14px;
  border: 1px solid var(--stroke-dim);
  background: var(--bg-panel-elevated);
  color: var(--text-primary);
  font-size: var(--text-sm);
  box-shadow: var(--panel-shadow);
  border-radius: var(--radius-sm);
  animation: in 180ms var(--ease-out);
}
.toast--success {
  border-color: color-mix(in srgb, var(--success) 50%, var(--stroke-dim));
}
.toast--error {
  border-color: color-mix(in srgb, var(--danger) 50%, var(--stroke-dim));
}
@keyframes in {
  from {
    opacity: 0;
    transform: translateX(12px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
</style>
