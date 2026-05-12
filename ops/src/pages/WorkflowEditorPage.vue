<template>
  <div class="h-full flex flex-col -m-6">
    <!-- Top Toolbar -->
    <div class="h-12 flex-shrink-0 border-b border-border bg-bg-surface px-4 flex items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <button class="btn btn-ghost text-xs" @click="router.push('/workflows')">← 返回</button>
        <div class="w-px h-5 bg-border"></div>
        <input
          v-model="workflowName"
          class="bg-transparent border-none text-sm font-medium text-text-primary focus:outline-none w-48"
          placeholder="处理流名称..."
        />
        <span :class="['badge text-xs', isDirty ? 'badge-orange' : 'badge-green']">
          {{ isDirty ? '未保存' : '已保存' }}
        </span>
      </div>

      <div class="flex items-center gap-2">
        <button class="btn btn-ghost text-xs px-2" @click="undo" title="撤销">↩️</button>
        <button class="btn btn-ghost text-xs px-2" @click="redo" title="重做">↪️</button>
        <div class="w-px h-5 bg-border"></div>
        <button class="btn btn-ghost text-xs px-2" @click="() => { zoomOut() }">➖</button>
        <span class="text-xs text-text-muted w-12 text-center">{{ Math.round(zoom * 100) }}%</span>
        <button class="btn btn-ghost text-xs px-2" @click="() => { zoomIn() }">➕</button>
        <button class="btn btn-ghost text-xs px-2" @click="() => { fitView() }">⊡</button>
        <div class="w-px h-5 bg-border"></div>
        <button class="btn btn-secondary text-xs" @click="saveWorkflow">💾 保存</button>
        <button class="btn btn-primary text-xs" @click="publishWorkflow">🚀 发布</button>
      </div>
    </div>

    <!-- Canvas Area -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Left: Plugin Palette -->
      <div class="w-56 flex-shrink-0 border-r border-border bg-bg-surface overflow-y-auto">
        <div class="px-3 py-3 border-b border-border">
          <div class="text-xs font-medium text-text-muted uppercase tracking-wider mb-1">插件面板</div>
          <div class="text-xs text-text-muted">拖拽插件到画布</div>
        </div>
        <div class="p-2 space-y-1">
          <div
            v-for="plugin in availablePlugins"
            :key="plugin.name"
            class="plugin-item p-2.5 rounded-md cursor-grab active:cursor-grabbing border border-transparent hover:border-accent-blue hover:bg-bg-elevated transition-all text-sm"
            :draggable="true"
            @dragstart="(e) => onDragStart(e, plugin)"
          >
            <div class="flex items-center gap-2">
              <span class="text-accent-blue">🔌</span>
              <div class="flex-1 min-w-0">
                <div class="text-text-primary font-medium truncate text-xs">{{ plugin.name }}</div>
                <div class="text-text-muted text-xs">{{ plugin.nodeTypes?.join(', ') || '通用' }}</div>
              </div>
            </div>
          </div>
          <div v-if="!availablePlugins.length" class="px-3 py-6 text-center text-xs text-text-muted">
            暂无插件<br>请先启动 Worker
          </div>
        </div>
      </div>

      <!-- Center: Vue Flow Canvas -->
      <div class="flex-1 relative bg-bg-base" ref="canvasContainer">
        <VueFlow
          v-model:nodes="nodes"
          v-model:edges="edges"
          :connection-line-style="{ stroke: '#58a6ff', strokeWidth: 2 }"
          :default-viewport="{ zoom: 1 }"
          :min-zoom="0.2"
          :max-zoom="2"
          :snap-to-grid="true"
          :snap-grid="[16, 16]"
          fit-view-on-init
          @dragover="onDragOver"
          @drop="onDrop"
          @node-click="onNodeClick"
          @pane-click="onPaneClick"
          @nodes-change="onNodesChange"
          @edges-change="onEdgesChange"
          @connect="onConnect"
          ref="vueFlowRef"
        >
          <Background pattern-color="#30363d" :gap="16" />
          <Controls />
          <MiniMap />
        </VueFlow>

        <!-- Right: Node Config Panel -->
        <transition name="slide">
          <div
            v-if="activeNode"
            class="absolute top-0 right-0 bottom-0 w-72 bg-bg-surface border-l border-border overflow-y-auto"
          >
            <div class="px-4 py-3 border-b border-border flex items-center justify-between">
              <div class="text-sm font-medium text-text-primary">节点配置</div>
              <button class="btn btn-ghost text-xs px-1" @click="activeNodeId = null">✕</button>
            </div>
            <div class="p-4 space-y-4">
              <div>
                <label class="label">节点名称</label>
                <input v-model="activeNode.data.name" class="input" placeholder="节点名称" @input="isDirty = true" />
              </div>
              <div>
                <label class="label">插件</label>
                <div class="text-sm text-text-secondary">{{ activeNode.data.plugin }}</div>
              </div>
              <div>
                <label class="label">依赖节点</label>
                <div class="space-y-1">
                  <div
                    v-for="dep in (activeNode.data.dependsOn || [])"
                    :key="dep"
                    class="flex items-center justify-between px-2 py-1 bg-bg-elevated rounded text-xs"
                  >
                    <span class="text-text-secondary">{{ dep }}</span>
                    <button class="text-accent-red hover:text-red-400" @click="removeDep(dep)">✕</button>
                  </div>
                  <div v-if="!(activeNode.data.dependsOn?.length)" class="text-xs text-text-muted">
                    无依赖（可作为起始节点）
                  </div>
                </div>
              </div>
              <div>
                <label class="label">插件配置 (JSON)</label>
                <textarea
                  v-model="nodeConfigJson"
                  class="input font-mono text-xs resize-none"
                  rows="6"
                  placeholder='{ "key": "value" }'
                  @input="onConfigChange"
                ></textarea>
              </div>
              <div class="pt-2 border-t border-border">
                <button class="btn btn-danger w-full text-xs" @click="deleteActiveNode">
                  🗑️ 删除节点
                </button>
              </div>
            </div>
          </div>
        </transition>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import { workflowApi, pluginApi } from '@/api'
import { uuid } from '@/utils/uuid'

const router = useRouter()
const route = useRoute()

// ─── Vue Flow ───────────────────────────────────────────────────────────────
const vueFlowRef = ref<any>(null)
const canvasContainer = ref<HTMLElement | null>(null)
const { zoomIn, zoomOut, fitView, getViewport } = useVueFlow()

const nodes = ref<any[]>([])
const edges = ref<any[]>([])
const zoom = ref(1)

// ─── Active Node ────────────────────────────────────────────────────────────
const activeNodeId = ref<string | null>(null)
const activeNode = computed(() =>
  activeNodeId.value ? nodes.value.find(n => n.id === activeNodeId.value) : null
)

const nodeConfigJson = ref('{}')
function onConfigChange() {
  if (!activeNode.value) return
  try {
    activeNode.value.data.config = JSON.parse(nodeConfigJson.value)
    isDirty.value = true
  } catch { /* invalid JSON */ }
}

// Watch activeNode to sync config JSON
watch(activeNode, (node) => {
  if (node) {
    nodeConfigJson.value = JSON.stringify(node.data.config || {}, null, 2)
  }
}, { immediate: true })

// ─── Workflow Meta ───────────────────────────────────────────────────────────
const workflowId = ref<string | null>(null)
const workflowName = ref('未命名处理流')
const isDirty = ref(false)

// ─── Plugin Palette ─────────────────────────────────────────────────────────
const availablePlugins = ref<any[]>([])

// ─── Undo/Redo ───────────────────────────────────────────────────────────────
const history = ref<{ nodes: any[]; edges: any[] }[]>([])
const historyIndex = ref(-1)
let suppressHistory = false

function pushHistory() {
  if (suppressHistory) return
  history.value = history.value.slice(0, historyIndex.value + 1)
  history.value.push({
    nodes: JSON.parse(JSON.stringify(nodes.value)),
    edges: JSON.parse(JSON.stringify(edges.value))
  })
  historyIndex.value = history.value.length - 1
  isDirty.value = true
}

function undo() {
  if (historyIndex.value <= 0) return
  historyIndex.value--
  const s = history.value[historyIndex.value]
  suppressHistory = true
  nodes.value = JSON.parse(JSON.stringify(s.nodes))
  edges.value = JSON.parse(JSON.stringify(s.edges))
  suppressHistory = false
}

function redo() {
  if (historyIndex.value >= history.value.length - 1) return
  historyIndex.value++
  const s = history.value[historyIndex.value]
  suppressHistory = true
  nodes.value = JSON.parse(JSON.stringify(s.nodes))
  edges.value = JSON.parse(JSON.stringify(s.edges))
  suppressHistory = false
}

// ─── Drag & Drop ────────────────────────────────────────────────────────────
let dragPlugin: any = null

function onDragStart(e: DragEvent, plugin: any) {
  dragPlugin = plugin
  e.dataTransfer?.setData('application/vue-flow', JSON.stringify(plugin))
  e.dataTransfer!.effectAllowed = 'move'
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  e.dataTransfer!.dropEffect = 'move'
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  const pluginStr = e.dataTransfer?.getData('application/vue-flow')
  if (!pluginStr) return
  const plugin = JSON.parse(pluginStr)

  const bounds = canvasContainer.value?.getBoundingClientRect()
  if (!bounds) return

  const viewport = getViewport()
  const x = (e.clientX - bounds.left - viewport.x) / viewport.zoom
  const y = (e.clientY - bounds.top - viewport.y) / viewport.zoom

  const nodeId = 'node-' + uuid()
  const newNode = {
    id: nodeId,
    type: 'default',
    position: { x, y },
    data: {
      label: plugin.name,
      name: plugin.name,
      plugin: plugin.name,
      config: {},
      dependsOn: [],
    }
  }

  pushHistory()
  nodes.value = [...nodes.value, newNode]
  activeNodeId.value = nodeId
  dragPlugin = null
}

// ─── Node/Edge Events ───────────────────────────────────────────────────────
function onNodeClick(e: any) {
  activeNodeId.value = e.node.id
}

function onPaneClick() {
  activeNodeId.value = null
}

function onConnect(params: any) {
  pushHistory()
  edges.value = [...edges.value, {
    id: 'edge-' + uuid(),
    source: params.source,
    target: params.target,
    animated: false,
  }]
  // Update target node's dependsOn
  const targetNode = nodes.value.find(n => n.id === params.target)
  if (targetNode) {
    targetNode.data.dependsOn = [...(targetNode.data.dependsOn || []), params.source]
  }
}

function onNodesChange(changes: any[]) {
  changes.forEach(change => {
    if (change.type === 'remove') {
      nodes.value.forEach(n => {
        n.data.dependsOn = (n.data.dependsOn || []).filter((id: string) => id !== change.id)
      })
      edges.value = edges.value.filter(e => e.source !== change.id && e.target !== change.id)
    }
  })
}

function onEdgesChange(changes: any[]) {
  changes.forEach(change => {
    if (change.type === 'remove') {
      const edge = edges.value.find(e => e.id === change.id)
      if (edge) {
        const targetNode = nodes.value.find(n => n.id === edge.target)
        if (targetNode) {
          targetNode.data.dependsOn = (targetNode.data.dependsOn || []).filter((id: string) => id !== edge.source)
        }
      }
      edges.value = edges.value.filter(e => e.id !== change.id)
    }
  })
}

function removeDep(depId: string) {
  if (!activeNode.value) return
  pushHistory()
  activeNode.value.data.dependsOn = (activeNode.value.data.dependsOn || []).filter((id: string) => id !== depId)
  edges.value = edges.value.filter(e => !(e.source === depId && e.target === activeNode.value!.id))
}

function deleteActiveNode() {
  if (!activeNode.value) return
  const id = activeNode.value.id
  pushHistory()
  nodes.value = nodes.value.filter(n => n.id !== id)
  edges.value = edges.value.filter(e => e.source !== id && e.target !== id)
  nodes.value.forEach(n => {
    n.data.dependsOn = (n.data.dependsOn || []).filter((depId: string) => depId !== id)
  })
  activeNodeId.value = null
}

// ─── Save / Publish ─────────────────────────────────────────────────────────
async function saveWorkflow() {
  try {
    const payload = {
      name: workflowName.value,
      nodes: nodes.value.map(n => ({
        id: n.id,
        name: n.data.name,
        plugin: n.data.plugin,
        dependsOn: n.data.dependsOn || [],
        config: n.data.config || {},
      })),
      trigger: { eventType: 'manual', conditions: {} },
    }

    if (workflowId.value) {
      await workflowApi.update(workflowId.value, payload)
    } else {
      const res = await workflowApi.create(payload) as any
      workflowId.value = res.id
    }

    isDirty.value = false
  } catch (e: any) {
    console.error('Save failed:', e.message)
    alert('保存失败: ' + e.message)
  }
}

async function publishWorkflow() {
  await saveWorkflow()
  if (!workflowId.value) return
  try {
    await workflowApi.publish(workflowId.value)
    alert('发布成功！')
  } catch (e: any) {
    alert('发布失败: ' + e.message)
  }
}

// ─── Init ────────────────────────────────────────────────────────────────────
onMounted(async () => {
  try {
    availablePlugins.value = (await pluginApi.list()) as any[]
  } catch {}

  const id = route.params.id as string | undefined
  if (id) {
    try {
      const wf = await workflowApi.get(id) as any
      workflowId.value = wf.id
      workflowName.value = wf.name
      nodes.value = (wf.nodes || []).map((n: any, i: number) => ({
        id: n.id,
        type: 'default',
        position: { x: 200 + (i % 3) * 220, y: 150 + Math.floor(i / 3) * 180 },
        data: {
          label: n.name,
          name: n.name,
          plugin: n.plugin,
          config: n.config || {},
          dependsOn: n.dependsOn || [],
        }
      }))
      edges.value = []
      nodes.value.forEach(node => {
        (node.data.dependsOn || []).forEach((depId: string) => {
          edges.value.push({ id: 'e-' + depId + '-' + node.id, source: depId, target: node.id })
        })
      })
      pushHistory()
    } catch {}
  } else {
    pushHistory()
  }
})
</script>

<style scoped>
.slide-enter-active, .slide-leave-active {
  transition: transform 0.15s ease;
}
.slide-enter-from, .slide-leave-to {
  transform: translateX(100%);
}
</style>
