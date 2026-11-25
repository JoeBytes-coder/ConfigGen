<script setup lang="ts">
import { h, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { NButton, NCard, NDataTable, NEmpty, NPagination } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useRouter } from 'vue-router'

import type { RecordSummary } from '@/api/types'
import { useHistoryStore } from '@/stores/history'
import { formatDateTime } from '@/utils/format'

const router = useRouter()
const store = useHistoryStore()
const { records, total, loading, page, error, totalPages } = storeToRefs(store)

onMounted(() => {
  void store.fetchRecords()
})

const columns: DataTableColumns<RecordSummary> = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '应用', key: 'app_name' },
  { title: '命名空间', key: 'namespace', render: (row: RecordSummary) => row.namespace || '-' },
  { title: '资源', key: 'kinds', render: (row: RecordSummary) => row.kinds.join(', ') },
  { title: '时间', key: 'created_at', render: (row: RecordSummary) => formatDateTime(row.created_at) },
  {
    title: '操作',
    key: 'actions',
    width: 90,
    render: (row: RecordSummary) =>
      h(
        NButton,
        { size: 'tiny', text: true, type: 'primary', onClick: () => router.push('/history/' + row.id) },
        { default: () => '查看' },
      ),
  },
]

async function onPage(target: number) {
  await store.goToPage(target)
}
</script>

<template>
  <n-card title="历史记录" size="small">
    <template #header-extra>
      <span class="total">共 {{ total }} 条</span>
      <n-button size="small" quaternary type="primary" :loading="loading" @click="store.refresh()">刷新</n-button>
    </template>

    <n-empty v-if="error" :description="error" style="padding: 32px 0" />
    <template v-else>
      <n-data-table :columns="columns" :data="records" :loading="loading" size="small" :bordered="false">
        <template #empty>
          <n-empty description="暂无生成记录" style="padding: 32px 0" />
        </template>
      </n-data-table>
      <div class="pager">
        <n-pagination :page="page" :page-count="totalPages" @update:page="onPage" />
      </div>
    </template>
  </n-card>
</template>

<style scoped>
.total {
  font-size: 12px;
  color: var(--cg-text-muted);
  margin-right: 8px;
}
.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
