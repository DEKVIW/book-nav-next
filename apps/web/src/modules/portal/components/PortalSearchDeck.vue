<script setup lang="ts">
import { computed } from 'vue'
import AppIcon from '@/shared/ui/AppIcon.vue'

const props = withDefaults(
  defineProps<{
    modelValue: string
    aiAvailable?: boolean
    useAi?: boolean
  }>(),
  { aiAvailable: false, useAi: false },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:useAi': [value: boolean]
  submit: [query: string]
  clear: []
}>()

const aiActive = computed(() => props.aiAvailable && props.useAi)
const placeholder = computed(() =>
  aiActive.value ? 'AI 目标锁定：输入问题…' : '搜索网站标题、描述或链接…',
)

function updateValue(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}

function toggleAi() {
  if (props.aiAvailable) emit('update:useAi', !props.useAi)
}

function submit() {
  const query = props.modelValue.trim()
  if (!query) {
    emit('clear')
    return
  }
  emit('submit', query)
}

function clear() {
  emit('update:modelValue', '')
  emit('clear')
}
</script>

<template>
  <form class="search-deck" :class="{ 'search-deck--ai': aiActive }" @submit.prevent="submit">
    <span class="search-deck__pulse" aria-hidden="true" />
    <span class="search-deck__icon" :class="{ 'search-deck__icon--ai': aiActive }" aria-hidden="true">
      <AppIcon :name="aiActive ? 'sparkles' : 'radar'" :size="18" />
    </span>
    <input
      :value="modelValue"
      type="search"
      class="search-deck__input"
      :placeholder="placeholder"
      enterkeyhint="search"
      @input="updateValue"
    />
    <button
      v-if="modelValue"
      type="button"
      class="search-deck__clear"
      aria-label="清除搜索"
      title="清除搜索"
      @click="clear"
    >
      <AppIcon name="x" :size="15" />
    </button>
    <button
      v-if="aiAvailable"
      type="button"
      class="search-deck__mode"
      :class="{ 'search-deck__mode--ai': aiActive }"
      :aria-pressed="aiActive"
      :title="aiActive ? '关闭 AI 智能搜索' : '开启 AI 智能搜索'"
      @click="toggleAi"
    >
      <AppIcon name="sparkles" :size="15" />
      <span>AI</span>
    </button>
    <button type="submit" class="search-deck__submit" aria-label="执行搜索" title="执行搜索">
      <AppIcon name="search" :size="16" />
    </button>
  </form>
</template>

<style scoped>
.search-deck {
  position: relative;
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  min-height: 46px;
  padding: 5px;
  overflow: hidden;
  border: 1px solid var(--stroke-dim);
  background:
    linear-gradient(90deg, color-mix(in srgb, var(--energy) 5%, transparent), transparent 42%),
    var(--bg-input);
  box-shadow: var(--inset-shadow);
  clip-path: polygon(9px 0, 100% 0, 100% calc(100% - 9px), calc(100% - 9px) 100%, 0 100%, 0 9px);
  transition: border-color var(--dur-base), box-shadow var(--dur-base), background var(--dur-base);
}

.search-deck::after {
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent, rgba(61, 231, 255, 0.12), transparent);
  content: '';
  pointer-events: none;
  transform: translateX(-100%);
  transition: transform var(--dur-slow) var(--ease-out);
}

.search-deck:focus-within,
.search-deck:hover {
  border-color: var(--energy-dim);
  box-shadow: var(--inset-shadow), 0 0 0 1px color-mix(in srgb, var(--energy) 16%, transparent), 0 0 22px var(--energy-glow);
}

.search-deck:focus-within::after {
  transform: translateX(100%);
}

.search-deck--ai {
  border-color: color-mix(in srgb, var(--magenta) 58%, var(--stroke-dim));
  background:
    linear-gradient(90deg, color-mix(in srgb, var(--magenta) 10%, transparent), transparent 48%),
    var(--bg-input);
  box-shadow: var(--inset-shadow), 0 0 0 1px color-mix(in srgb, var(--magenta) 17%, transparent), 0 0 24px color-mix(in srgb, var(--magenta) 19%, transparent);
}

.search-deck--ai::after {
  background: linear-gradient(90deg, transparent, color-mix(in srgb, var(--magenta) 28%, transparent), transparent);
}

.search-deck__pulse {
  width: 5px;
  height: 5px;
  flex-shrink: 0;
  margin-left: 3px;
  border-radius: 50%;
  background: var(--energy);
  box-shadow: 0 0 10px var(--energy);
  transition: background var(--dur-base), box-shadow var(--dur-base);
}

.search-deck--ai .search-deck__pulse {
  background: var(--magenta);
  box-shadow: 0 0 10px var(--magenta);
}

.search-deck__icon {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  color: var(--energy);
  transition: color var(--dur-base), transform var(--dur-base);
}

.search-deck__icon--ai {
  color: var(--magenta);
  transform: rotate(-8deg) scale(1.08);
}

.search-deck__input {
  min-width: 0;
  flex: 1;
  height: 34px;
  padding: 0 2px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text-primary);
  font: inherit;
  font-size: var(--text-sm);
}

.search-deck__input::placeholder {
  color: var(--text-muted);
  opacity: 0.9;
}

.search-deck__clear,
.search-deck__mode,
.search-deck__submit {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  min-width: 34px;
  height: 34px;
  padding: 0 7px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: color var(--dur-fast), border-color var(--dur-fast), background var(--dur-fast), box-shadow var(--dur-fast);
}

.search-deck__clear:hover {
  color: var(--text-primary);
  border-color: var(--stroke-dim);
  background: var(--bg-panel);
}

.search-deck__mode {
  gap: 4px;
  color: var(--text-secondary);
  border-color: var(--stroke-dim);
  background: color-mix(in srgb, var(--bg-panel-solid) 62%, transparent);
  font-family: var(--font-display);
  font-size: 10px;
  letter-spacing: 0.08em;
}

.search-deck__mode:hover,
.search-deck__mode--ai {
  color: var(--magenta);
  border-color: color-mix(in srgb, var(--magenta) 58%, var(--stroke-dim));
  background: color-mix(in srgb, var(--magenta) 12%, var(--bg-panel-solid));
  box-shadow: 0 0 14px color-mix(in srgb, var(--magenta) 16%, transparent);
}

.search-deck__submit {
  color: var(--energy);
  border-color: var(--energy-dim);
  background: color-mix(in srgb, var(--energy) 10%, transparent);
}

.search-deck__submit:hover {
  color: var(--text-primary);
  border-color: var(--energy);
  background: color-mix(in srgb, var(--energy) 18%, transparent);
  box-shadow: 0 0 14px var(--energy-glow);
}

.search-deck--ai .search-deck__submit {
  color: var(--magenta);
  border-color: color-mix(in srgb, var(--magenta) 52%, var(--stroke-dim));
  background: color-mix(in srgb, var(--magenta) 10%, transparent);
}

@media (max-width: 899px) {
  .search-deck {
    min-height: 52px;
    padding: 6px;
  }

  .search-deck__input {
    height: 38px;
  }

  .search-deck__clear,
  .search-deck__mode,
  .search-deck__submit {
    min-width: 38px;
    height: 38px;
  }
}

@media (max-width: 360px) {
  .search-deck {
    gap: 5px;
  }

  .search-deck__mode {
    min-width: 40px;
    padding-inline: 5px;
  }

  .search-deck__mode span {
    font-size: 9px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .search-deck,
  .search-deck::after,
  .search-deck__pulse,
  .search-deck__icon,
  .search-deck__clear,
  .search-deck__mode,
  .search-deck__submit {
    transition: none;
  }
}
</style>
