import http from './types'
import type { DashboardStats, WorkerInfo } from './types'

export const dashboardApi = {
  stats: () => http.get<DashboardStats>('/stats'),
  workers: () => http.get<WorkerInfo[]>('/workers'),
  listInstances: () => http.get<any[]>('/instances'),
}
