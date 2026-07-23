<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'

defineProps<{
  x: number
  y: number
  isAdmin: boolean
}>()

const emit = defineEmits<{
  visit: []
  copy: []
  edit: []
  add: []
  remove: []
  close: []
}>()

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
function onClick() {
  emit('close')
}
onMounted(() => {
  window.addEventListener('keydown', onKey)
  window.addEventListener('click', onClick)
  window.addEventListener('scroll', onClick, true)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('click', onClick)
  window.removeEventListener('scroll', onClick, true)
})
</script>

<template>
  <div
    class="ctx panel"
    :style="{ left: `${x}px`, top: `${y}px` }"
    @click.stop
  >
    <button type="button" @click="emit('visit')">访问链接</button>
    <button type="button" @click="emit('copy')">复制链接</button>
    <template v-if="isAdmin">
      <hr />
      <button type="button" @click="emit('edit')">编辑</button>
      <button type="button" @click="emit('add')">快速添加</button>
      <button type="button" class="danger" @click="emit('remove')">删除</button>
    </template>
  </div>
</template>

<style scoped>
.ctx {
  position: fixed;
  z-index: 900;
  min-width: 160px;
  padding: 6px;
  display: grid;
  gap: 2px;
}
.ctx button {
  border: 0;
  background: transparent;
  color: var(--text-primary);
  text-align: left;
  padding: 8px 10px;
  font: inherit;
  font-size: var(--text-sm);
  cursor: pointer;
  border-radius: var(--radius-sm);
}
.ctx button:hover {
  background: var(--bg-panel-elevated);
  color: var(--glow-cyan);
}
.ctx .danger:hover {
  color: var(--danger);
}
hr {
  border: 0;
  border-top: 1px solid var(--stroke-dim);
  margin: 4px 0;
}
</style>
