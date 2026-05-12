<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-semibold text-text-primary">处理流实例</h2>
        <p class="text-sm text-text-secondary mt-0.5">查看所有处理流执行记录</p>
      </div>
      <div class="flex items-center gap-2">
        <select v-model="filterStatus" class="input w-32 text-xs">
          <option value="">全部状态</option>
          <option value="pending">等待中</option>
          <option value="running">运行中</option>
          <option value="completed">已完成</option>
          <option value="failed">失败</option>
        </select>
      </div>
    </div>

    <!-- Table -->
    <DataTable :columns="columns" :data="instances" key-field="id">
      <template #cell-status="{ row }">
        <StatusBadge :status="row.status" />
      </template>
      <template #cell-id="{ row }">
        <span class="font-mono text-xs text-text-muted">{{ row.id.slice(0, 12) }}...</span>
      </template>
      <template #cell-workflowName="{ row }">
        <span class="text-text-primary">{{ row.workflowName }}</span>
      </template>
      <template #cell-createdAt="{ row }">
        <span class="text-text-secondary text-xs">{{ formatTime(row.createdAt) }}</span>
      </template>
      <template #cell-updatedAt="{ row }">
        <span class="text-text-secondary text-xs">{{ formatTime(row.updatedAt) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center gap-2">
          <button
            v-if="row.status === 'failed'"
            class="btn btn-secondary text-xs"
            @click="retryInstance(row.id)"
          >
            🔄 重试
          </button>
          <button class="btn btn-ghost text-xs" @click="viewDetail(row.id)">
            查看
          </button>
        </div>
      </template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import DataTable from '@/components/DataTable.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { instanceApi } from '@/api'

const columns = [
  { key: 'id', label: '实例 ID' },
  { key: 'workflowName', label: '处理流' },
  { key: 'status', label: '状态' },
  { key: 'createdAt', label: '创建时间' },
  { key: 'updatedAt', label: '更新时间' },
  { key: 'actions', label: '操作' },
]

const instances = ref<any[]>([])
const filterStatus = ref('')

async function loadInstances() {
  try {
    instances.value = (await instanceApi.list(filterStatus.value ? { status: filterStatus.value } : undefined)) as any[]
  } catch {
    instances.value = []
  }
}

async function retryInstance(id: string) {
  try {
    await instanceApi.retry(id)
    await loadInstances()
  } catch (e: any) {
    alert('重试失败: ' + e.message)
  }
}

function viewDetail(id: string) {
  alert('实例详情: ' + id)
}

function formatTime(ts: string) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString('zh-CN', { hour12: false })
}

onMounted(loadInstances)
</script>
