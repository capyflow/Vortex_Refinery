import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { PluginInfo } from '@/api/types'

export const usePluginStore = defineStore('plugins', () => {
  const plugins = ref<PluginInfo[]>([])
  const loading = ref(false)

  function setPlugins(list: PluginInfo[]) {
    plugins.value = list
  }

  return { plugins, loading, setPlugins }
})
