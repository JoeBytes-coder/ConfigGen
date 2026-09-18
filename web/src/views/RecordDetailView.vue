<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NPageHeader,
  NResult,
  NSpin,
  NTag,
} from 'naive-ui'

import { getConfig } from '@/api/client'
import type { ConfigRecord } from '@/api/types'
import ResourceGraph from '@/components/ResourceGraph.vue'
import ResultPanel from '@/components/ResultPanel.vue'
import { TYPE_META } from '@/constants/k8s'
import { formatDateTime, imageLabel } from '@/utils/format'

const route = useRoute()
const router = useRouter()

const record = ref<ConfigRecord | null>(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    record.value = await getConfig(route.params.id as string)
  } catch (e) {
    error.value = e instanceof Error ? e.message : '记录不存在'
  } finally {
    loading.value = false
  }
})

const isK8sLike = computed(
  () => record.value?.request.type === 'k8s' || record.value?.request.type === 'kustomize',
)

const envText = computed(() => {
  const env = record.value?.request.env
  if (!env || Object.keys(env).length === 0) return '-'
  return Object.entries(env)
    .map(([k, v]) => `${k}=${v}`)
    .join('，')
})
</script>

<template>
  <div class="detail-page">
    <n-page-header title="记录详情" @back="router.back()" style="margin-bottom: 16px" />

    <n-spin :show="loading">
      <n-result
        v-if="error"
        status="404"
        title="记录不存在"
        :description="error"
        style="padding: 48px 0"
      >
        <template #footer>
          <n-button type="primary" @click="router.push('/history')">返回历史记录</n-button>
        </template>
      </n-result>

      <template v-else-if="record">
        <n-card size="small" title="请求参数" style="margin-bottom: 16px">
          <n-descriptions bordered :column="2" label-placement="left" size="small">
            <n-descriptions-item label="类型">
              <n-tag size="small" :color="TYPE_META[record.request.type]" bordered>
                {{ TYPE_META[record.request.type]?.label ?? record.request.type }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="应用名称">{{ record.request.app_name }}</n-descriptions-item>
            <n-descriptions-item label="镜像">
              {{ imageLabel(record.request.image, record.request.tag) }}
            </n-descriptions-item>
            <n-descriptions-item label="端口">{{ record.request.port || '-' }}</n-descriptions-item>
            <n-descriptions-item v-if="record.request.replicas" label="副本数">
              {{ record.request.replicas }}
            </n-descriptions-item>
            <n-descriptions-item label="创建时间">
              {{ formatDateTime(record.created_at) }}
            </n-descriptions-item>
            <n-descriptions-item label="环境变量" :span="2">{{ envText }}</n-descriptions-item>
            <n-descriptions-item v-if="record.request.k8s_resource?.length" label="K8s 资源" :span="2">
              <n-tag
                v-for="kind in record.request.k8s_resource"
                :key="kind"
                size="small"
                style="margin-right: 6px"
              >
                {{ kind }}
              </n-tag>
            </n-descriptions-item>
          </n-descriptions>
        </n-card>

        <result-panel :record="record" />

        <n-card
          v-if="isK8sLike && (record.result.resources?.length ?? 0) > 0"
          size="small"
          title="资源关系图"
          style="margin-top: 16px"
        >
          <resource-graph :nodes="record.result.resources ?? []" :edges="record.result.edges ?? []" />
        </n-card>
      </template>
    </n-spin>
  </div>
</template>

<style scoped>
.detail-page {
  max-width: 920px;
  margin: 0 auto;
}
</style>
