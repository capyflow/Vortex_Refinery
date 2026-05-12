import http from './types'
import type { WorkflowInstance, TaskRecord } from './types'

export const instanceApi = {
  list: (params?: { workflowId?: string; status?: string }) =>
    http.get<WorkflowInstance[]>('/instances', { params }),
  get: (id: string) => http.get<WorkflowInstance>(`/instances/${id}`),
  retry: (id: string) => http.post(`/instances/${id}/retry`),
  tasks: (id: string) => http.get<TaskRecord[]>(`/instances/${id}/tasks`),
}
