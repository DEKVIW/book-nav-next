<script setup lang="ts">
import type { Website } from '@/shared/types/models'

defineProps<{
  open: boolean
  website: Website | null
}>()

const emit = defineEmits<{
  view: []
  force: []
  cancel: []
}>()
</script>

<template>
  <div v-if="open" class="mask" @click.self="emit('cancel')">
    <div class="modal panel">
      <p class="eyebrow">DUPLICATE SIGNAL</p>
      <h2>链接已存在</h2>
      <p v-if="website" class="info">
        <strong>{{ website.title }}</strong><br />
        <span>{{ website.url }}</span><br />
        <span v-if="website.category_name">分类：{{ website.category_name }}</span>
      </p>
      <div class="actions">
        <button type="button" class="m-btn m-btn--ghost" @click="emit('cancel')">取消</button>
        <button type="button" class="m-btn m-btn--ghost" @click="emit('view')">查看已有</button>
        <button type="button" class="m-btn m-btn--primary" @click="emit('force')">仍然添加</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: 850;
  background: rgba(0, 0, 0, 0.55);
  display: grid;
  place-items: center;
  padding: 16px;
}
.modal {
  width: min(420px, 100%);
  padding: 20px;
}
.eyebrow {
  margin: 0;
  color: var(--glow-amber);
  font-family: var(--font-display);
  font-size: var(--text-xs);
  letter-spacing: 0.14em;
}
h2 {
  margin: 8px 0;
}
.info {
  color: var(--text-secondary);
  font-size: var(--text-sm);
  line-height: 1.6;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
