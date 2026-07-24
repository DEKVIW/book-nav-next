<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from '@/shared/ui/AppIcon.vue'
import { CATEGORY_ICON_OPTIONS } from '@/shared/icons/registry'

const model = defineModel<string>({ default: 'folder' })

const q = ref('')

const filtered = computed(() => {
  const s = q.value.trim().toLowerCase()
  if (!s) return CATEGORY_ICON_OPTIONS
  return CATEGORY_ICON_OPTIONS.filter(
    (o) => o.name.includes(s) || o.label.includes(s) || o.label.toLowerCase().includes(s),
  )
})

function pick(name: string) {
  model.value = name
}
</script>

<template>
  <div class="icon-picker">
    <div class="icon-picker__current">
      <span class="icon-picker__preview">
        <AppIcon :name="model" :size="20" />
      </span>
      <div class="icon-picker__meta">
        <span class="icon-picker__name">{{ model }}</span>
        <input
          v-model="q"
          type="search"
          class="c-input icon-picker__search"
          placeholder="搜索图标…"
        />
      </div>
    </div>
    <div class="icon-picker__grid" role="listbox" :aria-label="'选择分类图标'">
      <button
        v-for="opt in filtered"
        :key="opt.name"
        type="button"
        class="icon-picker__cell"
        :class="{ 'icon-picker__cell--active': model === opt.name }"
        :title="`${opt.label} (${opt.name})`"
        role="option"
        :aria-selected="model === opt.name"
        @click="pick(opt.name)"
      >
        <AppIcon :name="opt.name" :size="18" />
      </button>
      <p v-if="!filtered.length" class="icon-picker__empty">无匹配图标</p>
    </div>
  </div>
</template>

<style scoped>
.icon-picker {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: min(100%, 360px);
}
.icon-picker__current {
  display: flex;
  align-items: center;
  gap: 10px;
}
.icon-picker__preview {
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  color: var(--energy);
  background: rgba(61, 231, 255, 0.1);
  border: 1px solid rgba(61, 231, 255, 0.28);
  border-radius: 10px;
  flex-shrink: 0;
}
.icon-picker__meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.icon-picker__name {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-muted);
}
.icon-picker__search {
  max-width: 220px;
  height: 32px !important;
  font-size: 12px !important;
}
.icon-picker__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(40px, 1fr));
  gap: 6px;
  max-height: 180px;
  overflow-y: auto;
  padding: 8px;
  background: var(--bg-inset);
  border: 1px solid var(--stroke-dim);
  border-radius: 10px;
}
.icon-picker__cell {
  width: 100%;
  aspect-ratio: 1;
  display: grid;
  place-items: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  transition:
    background 0.12s,
    color 0.12s,
    border-color 0.12s;
}
.icon-picker__cell:hover {
  color: var(--energy);
  background: rgba(61, 231, 255, 0.08);
}
.icon-picker__cell--active {
  color: var(--energy);
  background: rgba(61, 231, 255, 0.14);
  border-color: rgba(61, 231, 255, 0.35);
  box-shadow: 0 0 10px rgba(61, 231, 255, 0.15);
}
.icon-picker__empty {
  grid-column: 1 / -1;
  margin: 0;
  padding: 12px;
  text-align: center;
  color: var(--text-muted);
  font-size: 12px;
}
</style>
