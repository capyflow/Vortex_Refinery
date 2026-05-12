<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-semibold text-text-primary">Worker 状态</h2>
        <p class="text-sm text-text-secondary mt-0.5">查看已注册 Worker 的健康状态</p>
      </div>
      <button class="btn btn-secondary text-xs" @click="loadWorkers">🔄 刷新</button>
    </div>

    <DataTable :columns="columns" :data="workers" key-field="workerId">
      <template #cell-workerId="{ row }">
        <span class="font-mono text-xs text-accent-blue">{{ row.workerId }}</span>
      </template>
      <template #cell-plugins="{ row }">
        <div class="flex flex-wrap gap-1">
          <span v-for="p in row.plugins" :key="p" class="badge badge-blue text-xs">{{ p }}</span>
          <span v-if="!row.plugins?.length" class="text-text-muted text-xs">-</span>
        </div>
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :status="row.status" />
      </template>
      <template #cell-lastHeartbeat="{ row }">
        <span class="text-text-secondary text-xs">{{ formatTime(row.lastHeartbeat) }}</span>
      </template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import DataTable from '@/components/DataTable.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { dashboardApi } from '@/api'

const columns = [
  { key: 'workerId', label: 'Worker ID' },
  { key: 'plugins', label: '支持的插件' },
  { key: 'status', label: '状态' },
  { key: 'registeredAt', label: '注册时间' },
  { key: 'lastHeartbeat', label: '最后心跳' },
]

const workers = ref<any[]>([])

async function loadWorkers() {
  try {
    workers.value = (await dashboardApi.workers()) as any[]
  } catch {
    workers.value = []
  }
}

function formatTime(ts: string) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString('zh-CN', { hour12: false })
}

onMounted(loadWorkers)
</script>
