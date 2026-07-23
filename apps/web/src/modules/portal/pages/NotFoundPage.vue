<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="nf">
    <h1>404</h1>
    <p>页面不存在</p>
    <RouterLink class="m-btn m-btn--primary" to="/">返回首页</RouterLink>
  </div>
</template>

<style scoped>
.nf {
  min-height: 100vh;
  display: grid;
  place-content: center;
  justify-items: center;
  gap: var(--space-3);
  background: var(--bg-void);
  color: var(--text-secondary);
}
h1 {
  margin: 0;
  font-family: var(--font-display);
  font-size: 4rem;
  color: var(--glow-cyan);
}
</style>
