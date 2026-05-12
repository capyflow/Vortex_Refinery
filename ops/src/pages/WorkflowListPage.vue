<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-semibold text-text-primary">处理流</h2>
        <p class="text-sm text-text-secondary mt-0.5">编排和管理数据处理工作流</p>
      </div>
      <button class="btn btn-primary text-sm" @click="createWorkflow">
        + 新建处理流
      </button>
    </div>

    <!-- Workflow Cards -->
    <div v-if="workflows.length" class="grid grid-cols-3 gap-4">
      <div
        v-for="wf in workflows"
        :key="wf.id"
        class="card p-4 hover:border-border-hover transition-colors cursor-pointer group"
        @click="editWorkflow(wf.id)"
      >
        <div class="flex items-start justify-between mb-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-1">
              <span class="text-text-primary font-medium truncate">{{ wf.name }}</span>
              <span :class="['badge text-xs', wf.published ? 'badge-green' : 'badge-gray']">
                {{ wf.published ? '已发布' : '草稿' }}
              </span>
            </div>
            <div class="text-xs text-text-muted font-mono">v{{ wf.version }} · {{ wf.id.slice(0, 8) }}</div>
          </div>
        </div>

        <!-- Trigger info -->
        <div class="mb-3">
          <div class="text-xs text-text-muted mb-1">触发条件</div>
          <div class="badge badge-blue text-xs">{{ wf.trigger?.eventType || '-' }}</div>
        </div>

        <!-- Nodes count -->
        <div class="flex items-center gap-4 text-xs text-text-secondary mb-3">
          <span>🔀 {{ wf.nodes?.length || 0 }} 个节点</span>
          <span>📅 {{ formatTime(wf.updatedAt) }}</span>
        </div>

        <!-- Actions -->
        <div class="flex items-center gap-2 pt-3 border-t border-border/50">
          <button class="btn btn-secondary text-xs flex-1" @click.stop="editWorkflow(wf.id)">
            ✏️ 编辑
          </button>
          <button
            v-if="!wf.published"
            class="btn btn-primary text-xs flex-1"
            @click.stop="publishWorkflow(wf)"
          >
            🚀 发布
          </button>
          <button class="btn btn-danger text-xs" @click.stop="deleteWorkflow(wf)">
            🗑️
          </button>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <EmptyState
      v-else
      emoji="🔀"
      title="暂无处理流"
      description="创建第一个处理流，将插件编排成数据处理管道"
    >
      <template #action>
        <button class="btn btn-primary" @click="createWorkflow">+ 新建处理流</button>
      </template>
    </EmptyState>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import EmptyState from '@/components/EmptyState.vue'
import { workflowApi } from '@/api'

const router = useRouter()
const workflows = ref<any[]>([])

async function loadWorkflows() {
  try {
    workflows.value = (await workflowApi.list()) as any[]
  } catch (e) {
    workflows.value = []
  }
}

function createWorkflow() {
  router.push('/workflows/editor')
}

function editWorkflow(id: string) {
  router.push(`/workflows/editor/${id}`)
}

async function publishWorkflow(wf: any) {
  try {
    await workflowApi.publish(wf.id)
    await loadWorkflows()
  } catch (e: any) {
    console.error('Publish failed:', e.message)
  }
}

async function deleteWorkflow(wf: any) {
  if (!confirm(`确定删除处理流 "${wf.name}" 吗？`)) return
  try {
    await workflowApi.delete(wf.id)
    await loadWorkflows()
  } catch (e: any) {
    console.error('Delete failed:', e.message)
  }
}

function formatTime(ts: string) {
  if (!ts) return '-'
  return new Date(ts).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
}

onMounted(loadWorkflows)
</script>
