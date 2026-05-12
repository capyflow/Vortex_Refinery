<template>
  <div class="flex h-screen overflow-hidden">
    <!-- Sidebar -->
    <aside class="w-56 flex-shrink-0 bg-bg-surface border-r border-border flex flex-col">
      <!-- Logo -->
      <div class="px-4 py-5 border-b border-border">
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 rounded-lg bg-accent-blue/20 flex items-center justify-center">
            <span class="text-accent-blue text-sm">⚡</span>
          </div>
          <div>
            <div class="text-sm font-semibold text-text-primary">Vortex Refinery</div>
            <div class="text-xs text-text-muted">Ops Dashboard</div>
          </div>
        </div>
      </div>

      <!-- Nav -->
      <nav class="flex-1 px-2 py-4 space-y-0.5 overflow-y-auto">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors"
          :class="isActive(item.path)
            ? 'bg-accent-blue/10 text-accent-blue'
            : 'text-text-secondary hover:text-text-primary hover:bg-bg-elevated'"
        >
          <span>{{ item.icon }}</span>
          <span>{{ item.label }}</span>
        </router-link>
      </nav>

      <!-- Footer -->
      <div class="px-4 py-3 border-t border-border">
        <div class="text-xs text-text-muted">v1.0.0</div>
      </div>
    </aside>

    <!-- Main -->
    <main class="flex-1 flex flex-col overflow-hidden">
      <!-- Top bar -->
      <header class="h-14 flex-shrink-0 border-b border-border bg-bg-surface px-6 flex items-center justify-between">
        <div class="flex items-center gap-2 text-sm">
          <span class="text-text-muted">{{ activeMeta?.icon }}</span>
          <span class="text-text-primary font-medium">{{ activeMeta?.title }}</span>
        </div>
        <div class="flex items-center gap-3">
          <button class="btn btn-ghost text-xs" @click="refresh">
            🔄 刷新
          </button>
        </div>
      </header>

      <!-- Page content -->
      <div class="flex-1 overflow-auto p-6">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

const navItems = [
  { path: '/', label: 'Dashboard', icon: '📊' },
  { path: '/plugins', label: '插件管理', icon: '🔌' },
  { path: '/workflows', label: '处理流', icon: '🔀' },
  { path: '/instances', label: '处理流实例', icon: '▶️' },
  { path: '/workers', label: 'Worker 状态', icon: '⚙️' },
]

const activeMeta = computed(() => route.meta as { title: string; icon: string } | undefined)

function isActive(path: string) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

function refresh() {
  window.location.reload()
}
</script>
