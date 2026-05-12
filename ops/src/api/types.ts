import axios from 'axios'

// Typed HTTP client — interceptor returns res.data directly (not AxiosResponse)
const http = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

http.interceptors.response.use(
  (res) => res.data,
  (err) => {
    const msg = err.response?.data?.error || err.message || '请求失败'
    return Promise.reject(new Error(msg))
  }
)

export default http

// ─── Types ────────────────────────────────────────────────────────────────────

export interface PluginInfo {
  name: string
  version: string
  status: 'healthy' | 'unhealthy' | 'unknown'
  registeredAt: string
  nodeTypes: string[]
}

export interface WorkflowDefinition {
  id: string
  name: string
  version: number
  trigger: Trigger
  nodes: WorkflowNode[]
  onComplete?: CallbackAction
  onFailure?: CallbackAction
  createdAt: string
  updatedAt: string
  published: boolean
}

export interface Trigger {
  eventType: string
  conditions: Record<string, any>
}

export interface WorkflowNode {
  id: string
  name: string
  plugin: string
  dependsOn: string[]
  config: Record<string, any>
}

export interface CallbackAction {
  action: string
  config: Record<string, any>
}

export interface WorkflowInstance {
  id: string
  workflowId: string
  workflowName: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  event: any
  currentNodes: string[]
  completedNodes: string[]
  failedNodes: string[]
  createdAt: string
  updatedAt: string
}

export interface TaskRecord {
  id: string
  instanceId: string
  nodeId: string
  plugin: string
  status: string
  input: Record<string, string>
  output: Record<string, string>
  error: string
  workerId: string
  createdAt: string
  startedAt?: string
  completedAt?: string
  retryCount: number
}

export interface WorkerInfo {
  workerId: string
  plugins: string[]
  status: 'online' | 'offline'
  registeredAt: string
  lastHeartbeat: string
}

export interface DashboardStats {
  totalPlugins: number
  totalWorkflows: number
  runningInstances: number
  triggeredToday: number
}
