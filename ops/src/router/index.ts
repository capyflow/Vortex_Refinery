import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/pages/DashboardPage.vue'),
        meta: { title: 'Dashboard', icon: '📊' }
      },
      {
        path: 'plugins',
        name: 'plugins',
        component: () => import('@/pages/PluginsPage.vue'),
        meta: { title: '插件管理', icon: '🔌' }
      },
      {
        path: 'workflows',
        name: 'workflows',
        component: () => import('@/pages/WorkflowListPage.vue'),
        meta: { title: '处理流', icon: '🔀' }
      },
      {
        path: 'workflows/editor',
        name: 'workflow-editor-new',
        component: () => import('@/pages/WorkflowEditorPage.vue'),
        meta: { title: '编排处理流', icon: '🎨' }
      },
      {
        path: 'workflows/editor/:id',
        name: 'workflow-editor',
        component: () => import('@/pages/WorkflowEditorPage.vue'),
        meta: { title: '编排处理流', icon: '🎨' }
      },
      {
        path: 'instances',
        name: 'instances',
        component: () => import('@/pages/InstancesPage.vue'),
        meta: { title: '处理流实例', icon: '▶️' }
      },
      {
        path: 'workers',
        name: 'workers',
        component: () => import('@/pages/WorkersPage.vue'),
        meta: { title: 'Worker 状态', icon: '⚙️' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory('/ops'),
  routes
})

export default router
