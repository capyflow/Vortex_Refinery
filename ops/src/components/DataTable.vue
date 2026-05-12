<template>
  <div class="card">
    <!-- Header -->
    <div v-if="$slots.header || title" class="px-4 py-3 border-b border-border flex items-center justify-between">
      <div class="text-sm font-medium text-text-primary">{{ title }}</div>
      <slot name="header-actions" />
    </div>

    <!-- Table -->
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border">
            <th
              v-for="col in columns"
              :key="col.key"
              class="px-4 py-3 text-left text-xs font-medium text-text-muted uppercase tracking-wider"
              :class="col.align === 'right' ? 'text-right' : 'text-left'"
            >
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(row, idx) in data"
            :key="keyField ? row[keyField] : idx"
            class="border-b border-border/50 hover:bg-bg-elevated/50 transition-colors"
          >
            <td
              v-for="col in columns"
              :key="col.key"
              class="px-4 py-3"
              :class="col.align === 'right' ? 'text-right' : 'text-left'"
            >
              <slot :name="`cell-${col.key}`" :row="row" :value="row[col.key]">
                {{ row[col.key] }}
              </slot>
            </td>
          </tr>
          <tr v-if="!data || data.length === 0">
            <td :colspan="columns.length" class="px-4 py-12 text-center text-text-muted">
              <slot name="empty">暂无数据</slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Footer / Pagination -->
    <div v-if="$slots.footer || total > 0" class="px-4 py-3 border-t border-border flex items-center justify-between">
      <div class="text-xs text-text-muted">
        共 {{ total }} 条
      </div>
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  columns: { key: string; label: string; align?: 'left' | 'right' }[]
  data: Record<string, any>[]
  keyField?: string
  total?: number
  title?: string
}>(), {
  total: 0,
})
</script>
