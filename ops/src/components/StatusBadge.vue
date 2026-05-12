<template>
  <span :class="badgeClass">
    <span class="w-1.5 h-1.5 rounded-full" :class="dotClass"></span>
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: 'healthy' | 'unhealthy' | 'unknown' | 'pending' | 'running' | 'completed' | 'failed' | 'online' | 'offline'
}>()

const badgeClass = computed(() => {
  switch (props.status) {
    case 'healthy':
    case 'completed':
    case 'online':
      return 'badge badge-green'
    case 'unhealthy':
    case 'failed':
    case 'offline':
      return 'badge badge-red'
    case 'running':
    case 'pending':
      return 'badge badge-orange'
    default:
      return 'badge badge-gray'
  }
})

const dotClass = computed(() => {
  switch (props.status) {
    case 'healthy':
    case 'completed':
    case 'online':
      return 'bg-accent-green'
    case 'unhealthy':
    case 'failed':
    case 'offline':
      return 'bg-accent-red'
    case 'running':
    case 'pending':
      return 'bg-accent-orange animate-pulse'
    default:
      return 'bg-text-muted'
  }
})

const label = computed(() => {
  switch (props.status) {
    case 'healthy': return '健康'
    case 'unhealthy': return '不健康'
    case 'unknown': return '未知'
    case 'pending': return '等待中'
    case 'running': return '运行中'
    case 'completed': return '已完成'
    case 'failed': return '失败'
    case 'online': return '在线'
    case 'offline': return '离线'
    default: return props.status
  }
})
</script>
