<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NAlert, NButton, NCard, NEmpty, NSpin, NTag } from 'naive-ui'

import { getRecord } from '@/api/client'
import type { ConfigRecord, Diagnostic } from '@/api/types'
import ResourceGraph from '@/components/ResourceGraph.vue'
import ResultPanel from '@/components/ResultPanel.vue'
import { formatDateTime } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const record = ref<ConfigRecord | null>(null)
const loading = ref(false)
const error = ref('')

const diagnostics = computed<Diagnostic[]>(() => record.value?.plan.diagnostics ?? [])
const autoAdded = computed(() => record.value?.plan.nodes.filter((node) => node.auto_added) ?? [])
const selectionJSON = computed(() => (record.value ? JSON.stringify(record.value.selection, null, 2) : ''))

function severityType(severity: string) {
  if (severity === 'error') return 'error'
  if (severity === 'warning') return 'warning'
  return 'info'
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    record.value = await getRecord(String(route.params.id))
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="detail">
    <div class="detail-head">
      <n-button quaternary size="small" @click="router.push('/history')">返回历史记录</n-button>
      <n-button size="small" quaternary type="primary" :loading="loading" @click="load">刷新</n-button>
    </div>

    <n-alert v-if="error" type="error" :bordered="false">{{ error }}</n-alert>
    <n-spin v-else-if="loading" size="small" style="width: 100%; padding: 32px 0">
      <n-empty description="正在加载记录…" />
    </n-spin>

    <template v-else-if="record">
      <n-card size="small" class="summary-card">
        <div class="summary-head">
          <span class="app-name">{{ record.app.name }}</span>
          <span class="created">记录 #{{ record.id }} · {{ formatDateTime(record.created_at) }}</span>
        </div>
        <div class="summary-meta">
          <span>镜像：{{ record.app.image || '-' }}:{{ record.app.tag || '-' }}</span>
          <span>命名空间：{{ record.app.namespace || 'default' }}</span>
          <span>Catalog：{{ record.catalog_version }}</span>
          <span>文件：{{ record.files.length }}</span>
        </div>
        <div v-if="autoAdded.length" class="auto-added">
          <span>自动补充：</span>
          <n-tag v-for="node in autoAdded" :key="node.kind" size="tiny" type="warning">{{ node.kind }}</n-tag>
        </div>
      </n-card>

      <n-card v-if="diagnostics.length" title="诊断" size="small">
        <div class="diag-list">
          <n-alert
            v-for="(item, index) in diagnostics"
            :key="index"
            :type="severityType(item.severity)"
            :bordered="false"
            size="small"
          >
            {{ item.message }}
          </n-alert>
        </div>
      </n-card>

      <n-card v-if="record.plan.nodes.length" title="资源关系图" size="small">
        <resource-graph :nodes="record.plan.nodes" :edges="record.plan.edges" />
      </n-card>

      <result-panel :record="record" />

      <n-card title="请求参数" size="small">
        <pre class="json-pre">{{ selectionJSON }}</pre>
      </n-card>
    </template>

    <n-empty v-else description="记录不存在" style="padding: 32px 0" />
  </div>
</template>

<style scoped>
.detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.detail-head {
  display: flex;
  justify-content: space-between;
}
.summary-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.app-name {
  font-size: 18px;
  font-weight: 600;
}
.created {
  font-size: 12px;
  color: var(--cg-text-muted);
}
.summary-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 10px;
  font-size: 12px;
  color: var(--cg-text-secondary);
}
.auto-added {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
  font-size: 12px;
  color: var(--cg-text-muted);
}
.diag-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.json-pre {
  margin: 0;
  max-height: 320px;
  overflow: auto;
  padding: 12px;
  background: #f7fafd;
  border: 1px solid var(--cg-border);
  border-radius: 8px;
  font-family: Consolas, monospace;
  font-size: 12px;
}
</style>
