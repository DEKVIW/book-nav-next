<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import AppIcon from './AppIcon.vue'
import { useTheme, type ThemeMode } from '@/shared/composables/useTheme'

const root = ref<HTMLElement | null>(null)
const open = ref(false)
const { mode, resolved, setTheme } = useTheme()

const options: { value: ThemeMode; label: string; description: string; icon: string }[] = [
  { value: 'auto', label: '自动', description: '按时间自动切换', icon: 'clock' },
  { value: 'light', label: '亮色', description: '白昼云海机甲', icon: 'sun' },
  { value: 'dark', label: '深色', description: '深空机甲舰桥', icon: 'moon' },
  { value: 'system', label: '跟随系统', description: '跟随设备主题', icon: 'monitor' },
]

const activeLabel = computed(() => options.find((item) => item.value === mode.value)?.label || '自动')
const triggerIcon = computed(() => {
  if (mode.value === 'auto') return 'clock'
  if (mode.value === 'system') return 'monitor'
  return resolved.value === 'dark' ? 'moon' : 'sun'
})

function toggle() {
  open.value = !open.value
}

function select(next: ThemeMode) {
  setTheme(next)
  open.value = false
}

function onPointerDown(event: PointerEvent) {
  if (!root.value?.contains(event.target as Node)) open.value = false
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') open.value = false
}

onMounted(() => {
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div ref="root" class="theme-menu">
    <button
      type="button"
      class="theme-menu__trigger"
      :aria-label="`主题：${activeLabel}`"
      :aria-expanded="open"
      aria-haspopup="menu"
      :title="`主题：${activeLabel}`"
      @click.stop="toggle"
    >
      <AppIcon :name="triggerIcon" :size="17" />
      <span class="theme-menu__trigger-label">{{ activeLabel }}</span>
      <AppIcon name="chevron-down" :size="13" class="theme-menu__chevron" :class="{ 'is-open': open }" />
    </button>

    <div v-if="open" class="theme-menu__panel hull" role="menu" aria-label="选择主题">
      <div class="theme-menu__heading">
        <span class="hud-label">THEME CONTROL</span>
        <span class="theme-menu__current">{{ activeLabel }}</span>
      </div>
      <button
        v-for="item in options"
        :key="item.value"
        type="button"
        class="theme-menu__option"
        :class="{ 'is-active': mode === item.value }"
        role="menuitemradio"
        :aria-checked="mode === item.value"
        @click="select(item.value)"
      >
        <span class="theme-menu__option-icon"><AppIcon :name="item.icon" :size="16" /></span>
        <span class="theme-menu__option-copy">
          <strong>{{ item.label }}</strong>
          <small>{{ item.description }}</small>
        </span>
        <AppIcon v-if="mode === item.value" name="check" :size="16" class="theme-menu__check" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.theme-menu {
  position: relative;
  z-index: 3;
}

.theme-menu__trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 9px;
  border: 1px solid var(--stroke-dim);
  background: color-mix(in srgb, var(--bg-panel-solid) 78%, transparent);
  color: var(--text-secondary);
  cursor: pointer;
  clip-path: polygon(5px 0, 100% 0, 100% calc(100% - 5px), calc(100% - 5px) 100%, 0 100%, 0 5px);
  transition: border-color var(--dur-fast), color var(--dur-fast), background var(--dur-fast);
}

.theme-menu__trigger:hover,
.theme-menu__trigger[aria-expanded='true'] {
  border-color: var(--stroke-bright);
  color: var(--text-primary);
  background: color-mix(in srgb, var(--energy) 8%, var(--bg-panel-solid));
}

.theme-menu__trigger-label {
  font-size: var(--text-xs);
  white-space: nowrap;
}

.theme-menu__chevron {
  transition: transform var(--dur-fast);
}

.theme-menu__chevron.is-open {
  transform: rotate(180deg);
}

.theme-menu__panel {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 216px;
  padding: 8px;
  background: var(--bg-panel-elevated);
  box-shadow: var(--panel-shadow), 0 14px 34px rgba(15, 40, 54, 0.18);
}

.theme-menu__heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 4px 7px 8px;
  border-bottom: 1px solid var(--stroke-dim);
}

.theme-menu__current {
  color: var(--text-muted);
  font-size: 11px;
}

.theme-menu__option {
  display: grid;
  grid-template-columns: 28px 1fr 16px;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 46px;
  padding: 6px 7px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  text-align: left;
  cursor: pointer;
}

.theme-menu__option:hover,
.theme-menu__option.is-active {
  border-color: var(--energy-dim);
  background: color-mix(in srgb, var(--energy) 8%, transparent);
  color: var(--text-primary);
}

.theme-menu__option-icon {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  color: var(--energy);
  background: color-mix(in srgb, var(--energy) 10%, transparent);
  border: 1px solid var(--energy-dim);
}

.theme-menu__option-copy {
  min-width: 0;
}

.theme-menu__option-copy strong,
.theme-menu__option-copy small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.theme-menu__option-copy strong {
  font-size: 13px;
  font-weight: 600;
}

.theme-menu__option-copy small {
  margin-top: 1px;
  color: var(--text-muted);
  font-size: 11px;
}

.theme-menu__check {
  color: var(--energy);
}

@media (max-width: 640px) {
  .theme-menu__trigger-label,
  .theme-menu__chevron {
    display: none;
  }

  .theme-menu__trigger {
    width: 36px;
    justify-content: center;
    padding: 0;
  }

  .theme-menu__panel {
    position: fixed;
    top: calc(var(--top-rail-height) + 8px);
    right: 12px;
  }
}
</style>
