<script setup lang="ts">
defineProps<{
  empty?: string
  loading?: boolean
  isEmpty?: boolean
}>()
</script>

<template>
  <div class="c-card">
    <div v-if="loading" class="c-empty">加载中…</div>
    <template v-else>
      <div class="c-table-wrap">
        <table class="c-table">
          <thead>
            <slot name="head" />
          </thead>
          <tbody>
            <slot />
          </tbody>
        </table>
      </div>
      <div v-if="isEmpty" class="c-empty">{{ empty || '暂无数据' }}</div>
      <div v-if="$slots.footer && !isEmpty" class="c-pagination">
        <slot name="footer" />
      </div>
    </template>
  </div>
</template>
