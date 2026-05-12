import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { WorkflowDefinition, WorkflowNode } from '@/api/types'

export const useWorkflowStore = defineStore('workflow', () => {
  // Current workflow being edited
  const currentWorkflow = ref<Partial<WorkflowDefinition> | null>(null)
  const nodes = ref<WorkflowNode[]>([])
  const isDirty = ref(false)

  function setWorkflow(wf: Partial<WorkflowDefinition>) {
    currentWorkflow.value = wf
    nodes.value = wf.nodes || []
    isDirty.value = false
  }

  function updateNodes(newNodes: WorkflowNode[]) {
    nodes.value = newNodes
    isDirty.value = true
  }

  function clear() {
    currentWorkflow.value = null
    nodes.value = []
    isDirty.value = false
  }

  return { currentWorkflow, nodes, isDirty, setWorkflow, updateNodes, clear }
})
