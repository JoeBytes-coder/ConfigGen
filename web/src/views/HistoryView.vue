<script setup lang="ts">
import { h, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NTag,
  useMessage,
} from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'

import type { ConfigRecord } from '@/api/types'
import { TYPE_META } from '@/constants/k8s'
import { useHistoryStore } from '@/stores/history'
import { formatDateTime, imageLabel } from '@/utils/format'

const router = useRouter()
const message = useMessage()
const store = useHistoryStore()

onMounted(() => {
  store.fetchRecords().catch((e) => message.error(e instanceof Error ? e.message : '加载失败'))
})

const columns: DataTableColumns<ConfigRecord> = [
  { title: 'ID', key: 'id', width: 70 },
  {
    title: '类型',
    key: 'type',
    width: 140,
    render: (row) =>
      h(
        NTag,
        { size: 'small', color: TYPE_META[row.request.type] ?? undefined, bordered: true },
        { default: () => TYPE_META[row.request.type]?.label ?? row.request.type },
      ),
  },
  { title: '应用名称', key: 'app', width: 140, render: (row) => row.request.app_name || '-' },
  {
    title: '镜像',
    key: 'image',
    width: 200,
    render: (row) => imageLabel(row.request.image, row.request.tag),
  },
  {
    title: '资源',
    key: 'resources',
    width: 90,
    render: (row) =>
      row.request.k8s_resource?.length ? `${row.request.k8s_resource.length} 种` : '-',
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 180,
    render: (row) => formatDateTime(row.created_at),
  },
  {
    title: '操作',
    key: 'actions',
    width: 90,
    render: (row) =>
      h(
        NButton,
        { size: 'small', quaternary: true, type: 'primary', onClick: () => goDetail(row.id) },
        { default: () => '查看' },
      ),
  },
]

function goDetail(id: number) {
  router.push(`/history/${id}`)
}

async function prev() {
  if (store.page > 1) await store.goToPage(store.page - 1)
}

async function next() {
  if (store.hasMore) await store.goToPage(store.page + 1)
}
</script>

<template>
  <n-card title="历史记录" size="small">
    <template #header-extra>
      <n-button size="small" quaternary type="primary" :loading="store.loading" @click="store.refresh()">
        刷新
      </n-button>
    </template>

    <n-data-table
      :columns="columns"
      :data="store.records"
      :loading="store.loading"
      :row-props="(row: ConfigRecord) => ({ class: 'table-clickable', onClick: () => goDetail(row.id) })"
    />

    <n-empty
      v-if="!store.loading && store.records.length === 0"
      description="暂无生成记录，去首页生成第一份配置吧"
      style="padding: 32px 0"
    />

    <template #action>
      <div style="display: flex; align-items: center; justify-content: flex-end; gap: 12px">
        <span class="page-indicator">第 {{ store.page }} 页</span>
        <n-button size="small" :disabled="store.page <= 1" @click="prev">上一页</n-button>
        <n-button size="small" :disabled="!store.hasMore" @click="next">下一页</n-button>
      </div>
    </template>
  </n-card>
</template>
