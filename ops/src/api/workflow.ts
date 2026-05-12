import http from './types'
import type { WorkflowDefinition } from './types'

export const workflowApi = {
  list: () => http.get<WorkflowDefinition[]>('/workflows'),
  get: (id: string) => http.get<WorkflowDefinition>(`/workflows/${id}`),
  create: (data: Partial<WorkflowDefinition>) => http.post<{ id: string }>('/workflows', data),
  update: (id: string, data: Partial<WorkflowDefinition>) => http.put(`/workflows/${id}`, data),
  delete: (id: string) => http.delete(`/workflows/${id}`),
  publish: (id: string) => http.post(`/workflows/${id}/publish`),
}
