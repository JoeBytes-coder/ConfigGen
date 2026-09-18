import { defineStore } from 'pinia'
import { ref } from 'vue'

import { listConfigs } from '@/api/client'
import type { ConfigRecord } from '@/api/types'

const PAGE_SIZE = 10

/**
 * 历史记录列表状态。
 * 后端 /configs 不返回总数，这里用「多取一条」的方式判断是否存在下一页。
 */
export const useHistoryStore = defineStore('history', () => {
  const records = ref<ConfigRecord[]>([])
  const loading = ref(false)
  const page = ref(1)
  const hasMore = ref(false)
  const error = ref('')

  async function fetchRecords() {
    loading.value = true
    error.value = ''
    try {
      const offset = (page.value - 1) * PAGE_SIZE
      const res = await listConfigs(offset, PAGE_SIZE + 1)
      hasMore.value = res.records.length > PAGE_SIZE
      records.value = res.records.slice(0, PAGE_SIZE)
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载失败'
      records.value = []
      hasMore.value = false
    } finally {
      loading.value = false
    }
  }

  async function goToPage(p: number) {
    if (p < 1) return
    page.value = p
    await fetchRecords()
  }

  async function refresh() {
    await fetchRecords()
  }

  return { records, loading, page, hasMore, error, pageSize: PAGE_SIZE, fetchRecords, goToPage, refresh }
})
