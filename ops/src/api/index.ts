import http from './types'

// ─── Plugin API ───────────────────────────────────────────────────────────────
export const pluginApi = {
  list: (): Promise<any[]> => http.get('/plugins'),
  get: (name: string): Promise<any> => http.get(`/plugins/${name}`),
  refreshHealth: (name: string): Promise<any> => http.post(`/plugins/${name}/refresh`),
}

// ─── Workflow API ─────────────────────────────────────────────────────────────
export const workflowApi = {
  list: (): Promise<any[]> => http.get('/workflows'),
  get: (id: string): Promise<any> => http.get(`/workflows/${id}`),
  create: (data: any): Promise<{ id: string }> => http.post('/workflows', data),
  update: (id: string, data: any): Promise<void> => http.put(`/workflows/${id}`, data),
  delete: (id: string): Promise<void> => http.delete(`/workflows/${id}`),
  publish: (id: string): Promise<void> => http.post(`/workflows/${id}/publish`),
}

// ─── Instance API ─────────────────────────────────────────────────────────────
export const instanceApi = {
  list: (params?: Record<string, string>): Promise<any[]> =>
    http.get('/instances', { params }),
  get: (id: string): Promise<any> => http.get(`/instances/${id}`),
  retry: (id: string): Promise<void> => http.post(`/instances/${id}/retry`),
  tasks: (id: string): Promise<any[]> => http.get(`/instances/${id}/tasks`),
}

// ─── Dashboard API ────────────────────────────────────────────────────────────
export const dashboardApi = {
  stats: (): Promise<any> => http.get('/stats'),
  workers: (): Promise<any[]> => http.get('/workers'),
  listInstances: (): Promise<any[]> => http.get('/instances'),
}
