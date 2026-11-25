import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { listRecords } from '@/api/client'
import type { RecordSummary } from '@/api/types'

const PAGE_SIZE = 10

export const useHistoryStore = defineStore('history', () => {
  const records = ref<RecordSummary[]>([])
  const total = ref(0)
  const loading = ref(false)
  const page = ref(1)
  const error = ref('')

  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

  async function fetchRecords() {
    loading.value = true
    error.value = ''
    try {
      const offset = (page.value - 1) * PAGE_SIZE
      const res = await listRecords(offset, PAGE_SIZE)
      records.value = res.records
      total.value = res.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : '加载失败'
      records.value = []
      total.value = 0
    } finally {
      loading.value = false
    }
  }

  async function goToPage(target: number) {
    if (target < 1) return
    page.value = target
    await fetchRecords()
  }

  async function refresh() {
    await fetchRecords()
  }

  return { records, total, loading, page, error, pageSize: PAGE_SIZE, totalPages, fetchRecords, goToPage, refresh }
})
