<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-semibold text-text-primary">插件管理</h2>
        <p class="text-sm text-text-secondary mt-0.5">管理系统中已注册的所有插件</p>
      </div>
      <button class="btn btn-secondary text-xs" @click="loadPlugins">
        🔄 刷新列表
      </button>
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="plugins"
      key-field="name"
    >
      <template #empty>
        <div class="py-8">
          <EmptyState
            emoji="🔌"
            title="暂无插件"
            description="Worker 注册后会显示在这里"
          />
        </div>
      </template>

      <template #cell-name="{ row }">
        <div class="flex items-center gap-2">
          <span class="text-accent-blue">🔌</span>
          <span class="font-medium text-text-primary">{{ row.name }}</span>
        </div>
      </template>

      <template #cell-status="{ row }">
        <StatusBadge :status="row.status" />
      </template>

      <template #cell-version="{ row }">
        <span class="font-mono text-xs text-text-muted">v{{ row.version || '?' }}</span>
      </template>

      <template #cell-nodeTypes="{ row }">
        <div class="flex flex-wrap gap-1">
          <span v-for="t in row.nodeTypes" :key="t" class="badge badge-gray text-xs">{{ t }}</span>
          <span v-if="!row.nodeTypes?.length" class="text-text-muted text-xs">-</span>
        </div>
      </template>

      <template #cell-registeredAt="{ row }">
        <span class="text-text-secondary text-xs">{{ formatTime(row.registeredAt) }}</span>
      </template>

      <template #cell-actions="{ row }">
        <div class="flex items-center gap-2">
          <button
            class="btn btn-ghost text-xs px-2 py-1"
            @click="refreshHealth(row.name)"
            title="刷新健康状态"
          >
            🔄
          </button>
        </div>
      </template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import DataTable from '@/components/DataTable.vue'
import EmptyState from '@/components/EmptyState.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import { pluginApi } from '@/api'

const columns = [
  { key: 'name', label: '插件名称' },
  { key: 'status', label: '状态' },
  { key: 'version', label: '版本' },
  { key: 'nodeTypes', label: '节点类型' },
  { key: 'registeredAt', label: '注册时间' },
  { key: 'actions', label: '操作' },
]

const plugins = ref<any[]>([])

async function loadPlugins() {
  try {
    plugins.value = await pluginApi.list()
  } catch (e: any) {
    console.error('Failed to load plugins:', e.message)
    plugins.value = []
  }
}

async function refreshHealth(name: string) {
  try {
    await pluginApi.refreshHealth(name)
    await loadPlugins()
  } catch (e: any) {
    console.error('Failed to refresh:', e.message)
  }
}

function formatTime(ts: string) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString('zh-CN', { hour12: false })
}

onMounted(loadPlugins)
</script>
