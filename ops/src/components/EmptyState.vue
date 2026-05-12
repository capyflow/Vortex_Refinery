<template>
  <div class="card p-8 text-center">
    <div class="text-4xl mb-3">{{ emoji }}</div>
    <div class="text-text-primary font-medium mb-1">{{ title }}</div>
    <div class="text-text-secondary text-sm">{{ description }}</div>
    <div v-if="$slots.action" class="mt-4">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  emoji?: string
  title: string
  description: string
}>()
</script>
