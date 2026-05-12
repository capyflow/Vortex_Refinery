import http from './types'

export const pluginApi = {
  list: () => http.get<any[]>('/plugins'),
  get: (name: string) => http.get<any>(`/plugins/${name}`),
  refreshHealth: (name: string) => http.post<any>(`/plugins/${name}/refresh`),
}
