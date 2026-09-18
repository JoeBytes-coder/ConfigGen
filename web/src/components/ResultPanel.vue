<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NCard, NEmpty, NTag, useMessage } from 'naive-ui'

import type { ConfigRecord } from '@/api/types'
import { TYPE_META } from '@/constants/k8s'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{ record: ConfigRecord | null }>()

const message = useMessage()
const copied = ref(false)

const FILENAMES: Record<string, string> = {
  dockerfile: 'Dockerfile',
  compose: 'docker-compose.yaml',
  k8s: 'kubernetes.yaml',
  kustomize: 'kustomize.yaml',
}

const filename = computed(() => FILENAMES[props.record?.result.type ?? ''] ?? 'config.yaml')

async function copy() {
  const text = props.record?.result.config ?? ''
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    // 非安全上下文降级方案
    const ta = document.createElement('textarea')
    ta.value = text
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }
  copied.value = true
  message.success('已复制到剪贴板')
  setTimeout(() => (copied.value = false), 2000)
}

function download() {
  const text = props.record?.result.config ?? ''
  if (!text) return
  const mime = props.record?.result.type === 'dockerfile' ? 'text/plain' : 'application/x-yaml'
  const blob = new Blob([text], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename.value
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
</script>

<template>
  <n-card title="生成结果" size="small" class="result-card">
    <template #header-extra>
      <n-button v-if="record" size="small" quaternary type="primary" @click="copy">
        {{ copied ? '已复制' : '复制' }}
      </n-button>
      <n-button v-if="record" size="small" type="primary" secondary @click="download">下载</n-button>
    </template>

    <n-empty v-if="!record" description="填写表单并生成配置后，结果将展示在这里" style="padding: 48px 0" />

    <template v-else>
      <div class="result-meta">
        <n-tag size="small" :color="TYPE_META[record.result.type]" :bordered="true">
          {{ TYPE_META[record.result.type]?.label ?? record.result.type }}
        </n-tag>
        <span class="meta-item">{{ filename }}</span>
        <span class="meta-item">记录 #{{ record.id }}</span>
        <span class="meta-item">{{ formatDateTime(record.created_at) }}</span>
      </div>
      <pre class="config-pre">{{ record.result.config }}</pre>
    </template>
  </n-card>
</template>
