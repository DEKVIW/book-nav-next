<script setup lang="ts">
defineProps<{
  empty?: string
  loading?: boolean
  isEmpty?: boolean
}>()
</script>

<template>
  <!--
    Keep table mounted so loading toggles never unmount the whole table (flash / blank).
    - has rows: always show table (even while loading)
    - empty + loading: show loading text
    - empty + idle: show empty text
  -->
  <div class="c-card admin-table-card">
    <div v-show="!isEmpty" class="c-table-wrap" :class="{ 'is-refreshing': loading && !isEmpty }">
      <table class="c-table">
        <thead>
          <slot name="head" />
        </thead>
        <tbody>
          <slot />
        </tbody>
      </table>
    </div>
    <div v-if="loading && isEmpty" class="c-empty">加载中…</div>
    <div v-else-if="isEmpty && !loading" class="c-empty">{{ empty || '暂无数据' }}</div>
    <div v-if="$slots.footer && !isEmpty" class="c-pagination">
      <slot name="footer" />
    </div>
  </div>
</template>

<style scoped>
.is-refreshing {
  opacity: 0.85;
  transition: opacity 0.15s ease;
}
</style>
