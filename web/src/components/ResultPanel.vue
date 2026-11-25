<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NCard, NEmpty, NTabPane, NTabs, useMessage } from 'naive-ui'

import type { ConfigRecord } from '@/api/types'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{ record: ConfigRecord | null }>()
const message = useMessage()
const active = ref('')

const files = computed(() => props.record?.files ?? [])

watch(
  () => props.record,
  (record) => {
    active.value = record?.files?.[0]?.path ?? ''
  },
  { immediate: true },
)

async function copyText(text: string, label: string) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = text
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
  }
  message.success(label + '已复制')
}

function download(name: string, text: string) {
  if (!text) return
  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = name
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
</script>

<template>
  <n-card title="生成结果" size="small">
    <template #header-extra>
      <template v-if="record">
        <n-button size="small" quaternary type="primary" @click="copyText(record.bundle, '全部配置')">复制全部</n-button>
        <n-button size="small" type="primary" secondary @click="download('bundle.yaml', record.bundle)">下载</n-button>
      </template>
    </template>

    <n-empty v-if="!record" description="填写表单并生成配置后，结果将展示在这里" style="padding: 40px 0" />

    <template v-else>
      <div class="result-meta">
        <span class="meta-item">Kubernetes</span>
        <span class="meta-item">{{ files.length }} 个文件</span>
        <span class="meta-item">记录 #{{ record.id }}</span>
        <span class="meta-item">{{ formatDateTime(record.created_at) }}</span>
      </div>

      <n-tabs v-model:value="active" type="line" size="small" animated>
        <n-tab-pane v-for="file in files" :key="file.path" :name="file.path" :tab="file.path">
          <div class="file-actions">
            <n-button size="tiny" quaternary type="primary" @click="copyText(file.content, file.path)">复制</n-button>
            <n-button size="tiny" quaternary @click="download(file.path, file.content)">下载</n-button>
          </div>
          <pre class="config-pre">{{ file.content }}</pre>
        </n-tab-pane>
      </n-tabs>
    </template>
  </n-card>
</template>

<style scoped>
.result-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--cg-text-muted);
}
.meta-item {
  color: var(--cg-text-secondary);
}
.file-actions {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
  margin-bottom: 4px;
}
.config-pre {
  margin: 0;
  max-height: 420px;
  overflow: auto;
  padding: 12px;
  background: #f7fafd;
  border: 1px solid var(--cg-border);
  border-radius: 8px;
  font-family: Consolas, 'JetBrains Mono', monospace;
  font-size: 12px;
  line-height: 1.6;
  color: #2c3e5d;
}
</style>
