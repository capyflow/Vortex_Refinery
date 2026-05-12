<template>
  <div class="space-y-6">
    <!-- Stats Cards -->
    <div class="grid grid-cols-4 gap-4">
      <div v-for="stat in stats" :key="stat.label" class="card p-5">
        <div class="flex items-center justify-between mb-3">
          <span class="text-2xl">{{ stat.icon }}</span>
          <span :class="['badge', stat.colorClass]">{{ stat.change }}</span>
        </div>
        <div class="text-2xl font-bold text-text-primary mb-0.5">{{ stat.value }}</div>
        <div class="text-sm text-text-secondary">{{ stat.label }}</div>
      </div>
    </div>

    <!-- Recent Instances -->
    <div class="card">
      <div class="px-4 py-3 border-b border-border flex items-center justify-between">
        <span class="text-sm font-medium text-text-primary">最近处理流实例</span>
        <router-link to="/instances" class="text-xs text-accent-blue hover:underline">查看全部 →</router-link>
      </div>
      <div class="divide-y divide-border/50">
        <div v-for="inst in recentInstances" :key="inst.id" class="px-4 py-3 flex items-center justify-between hover:bg-bg-elevated/30 transition-colors">
          <div class="flex items-center gap-3">
            <StatusBadge :status="inst.status" />
            <div>
              <div class="text-sm text-text-primary">{{ inst.workflowName }}</div>
              <div class="text-xs text-text-muted font-mono">{{ inst.id.slice(0, 12) }}...</div>
            </div>
          </div>
          <div class="text-xs text-text-muted">{{ formatTime(inst.createdAt) }}</div>
        </div>
        <div v-if="!recentInstances.length" class="px-4 py-8 text-center text-text-muted text-sm">
          暂无运行中的实例
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { dashboardApi } from '@/api'

const stats = ref([
  { icon: '🔌', label: '插件总数', value: '-', change: '', colorClass: 'badge-blue' },
  { icon: '🔀', label: '处理流', value: '-', change: '', colorClass: 'badge-green' },
  { icon: '▶️', label: '运行中', value: '-', change: '', colorClass: 'badge-orange' },
  { icon: '📈', label: '今日触发', value: '-', change: '', colorClass: 'badge-blue' },
])

const recentInstances = ref<any[]>([])

onMounted(async () => {
  try {
    const data = await dashboardApi.stats() as any
    stats.value[0].value = data.totalPlugins
    stats.value[1].value = data.totalWorkflows
    stats.value[2].value = data.runningInstances
    stats.value[3].value = data.triggeredToday
  } catch (e) {
    // API not ready yet
  }

  try {
    const instances = (await dashboardApi.listInstances()) as any[]
    recentInstances.value = instances.slice(0, 5)
  } catch (e) {
    recentInstances.value = []
  }
})

function formatTime(ts: string) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString('zh-CN', { hour12: false })
}
</script>
