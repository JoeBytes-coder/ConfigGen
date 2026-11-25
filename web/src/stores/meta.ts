import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { getCatalog } from '@/api/client'
import type { CatalogKind } from '@/api/types'
import { CATEGORY_ORDER } from '@/constants/k8s'

/** K8s 资源目录（后端 /api/v1/catalog），分组、依赖提示、副本门控都以它为准。 */
export const useMetaStore = defineStore('meta', () => {
  const kinds = ref<CatalogKind[]>([])
  const version = ref('')
  const loaded = ref(false)
  const error = ref('')

  const byCategory = computed<Record<string, CatalogKind[]>>(() => {
    const groups: Record<string, CatalogKind[]> = {}
    for (const k of kinds.value) {
      ;(groups[k.category] ??= []).push(k)
    }
    const ordered: Record<string, CatalogKind[]> = {}
    for (const category of CATEGORY_ORDER) {
      if (groups[category]) ordered[category] = groups[category]
    }
    for (const category of Object.keys(groups)) {
      if (!ordered[category]) ordered[category] = groups[category]
    }
    return ordered
  })

  const kindToMeta = computed(() => new Map(kinds.value.map((k) => [k.kind, k])))

  const workloadKinds = computed(() => {
    const set = new Set<string>()
    for (const k of kinds.value) {
      if (k.category === 'workload') set.add(k.kind)
    }
    return set
  })

  async function fetchMeta() {
    error.value = ''
    try {
      const catalog = await getCatalog()
      kinds.value = catalog.kinds
      version.value = catalog.version
      loaded.value = true
    } catch (e) {
      error.value = e instanceof Error ? e.message : '资源目录加载失败'
    }
  }

  return { kinds, version, loaded, error, byCategory, kindToMeta, workloadKinds, fetchMeta }
})
