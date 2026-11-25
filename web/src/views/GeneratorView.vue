<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import {
  NAlert,
  NButton,
  NCard,
  NDivider,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSwitch,
  NTag,
  NText,
  useMessage,
} from 'naive-ui'

import EnvVarEditor from '@/components/EnvVarEditor.vue'
import ResourceGraph from '@/components/ResourceGraph.vue'
import ResourcePicker from '@/components/ResourcePicker.vue'
import ResultPanel from '@/components/ResultPanel.vue'
import type { Diagnostic } from '@/api/types'
import { useGeneratorStore } from '@/stores/generator'

const message = useMessage()
const store = useGeneratorStore()
const {
  appName, namespace, image, tag, port, replicas, host, tlsSecret,
  envPairs, secretPairs, cpuRequest, memRequest, cpuLimit, memLimit,
  probeEnabled, probePath, readinessPath,
  resources, hasWorkload, status, canGenerate, plan, planError, planDiagnostics, planning, record, rendering,
} = storeToRefs(store)

const graphExpanded = ref(true)
const autoAdded = computed(() => plan.value?.nodes.filter((node) => node.auto_added) ?? [])
const diagnostics = computed<Diagnostic[]>(() => planDiagnostics.value ?? [])

onMounted(() => {
  void store.refreshPlan()
})

function severityType(severity: string) {
  if (severity === 'error') return 'error'
  if (severity === 'warning') return 'warning'
  return 'info'
}

async function submit() {
  if (!appName.value.trim()) {
    message.warning('请填写应用名称')
    return
  }
  if (resources.value.length === 0) {
    message.warning('请至少选择一种 Kubernetes 资源')
    return
  }
  if (!canGenerate.value) {
    message.warning('请先完成依赖解析')
    return
  }
  try {
    const result = await store.generate()
    message.success('已生成 ' + result.files.length + ' 个文件')
  } catch (e) {
    message.error(e instanceof Error ? e.message : '生成失败，请重试')
  }
}
</script>

<template>
  <div class="generator-layout">
    <n-card title="Kubernetes 配置参数" size="small">
      <n-form label-placement="top" :show-feedback="false">
        <n-form-item label="应用名称（DNS-1123，小写字母/数字/连字符）">
          <n-input v-model:value="appName" placeholder="例如 myapp" />
        </n-form-item>

        <div class="form-grid">
          <n-form-item label="镜像">
            <n-input v-model:value="image" placeholder="例如 nginx" />
          </n-form-item>
          <n-form-item label="标签">
            <n-input v-model:value="tag" placeholder="例如 1.25" />
          </n-form-item>
        </div>

        <div class="form-grid">
          <n-form-item label="端口">
            <n-input-number v-model:value="port" :min="1" :max="65535" style="width: 100%" />
          </n-form-item>
          <n-form-item label="副本数">
            <n-input-number v-model:value="replicas" :min="1" :max="100" style="width: 100%" :disabled="!hasWorkload" />
          </n-form-item>
        </div>

        <n-form-item label="命名空间">
          <n-input v-model:value="namespace" placeholder="留空则使用集群默认命名空间；选择 Namespace 资源可自动创建" />
        </n-form-item>

        <n-form-item label="域名（Ingress）">
          <n-input v-model:value="host" placeholder="例如 app.example.com，留空则匹配任意 Host" />
        </n-form-item>
        <n-form-item v-if="host" label="TLS Secret 名称">
          <n-input v-model:value="tlsSecret" placeholder="例如 app-tls" />
        </n-form-item>

        <n-form-item label="资源类型（可多选，依赖会自动补齐）">
          <resource-picker v-model="resources" />
        </n-form-item>

        <n-divider title-placement="left">资源配额</n-divider>
        <div class="form-grid">
          <n-form-item label="CPU requests">
            <n-input v-model:value="cpuRequest" placeholder="100m" />
          </n-form-item>
          <n-form-item label="Memory requests">
            <n-input v-model:value="memRequest" placeholder="128Mi" />
          </n-form-item>
        </div>
        <div class="form-grid">
          <n-form-item label="CPU limits">
            <n-input v-model:value="cpuLimit" placeholder="500m" />
          </n-form-item>
          <n-form-item label="Memory limits">
            <n-input v-model:value="memLimit" placeholder="512Mi" />
          </n-form-item>
        </div>

        <n-form-item label="HTTP 探针（liveness / readiness）">
          <n-switch v-model:value="probeEnabled" />
        </n-form-item>
        <div v-if="probeEnabled" class="form-grid">
          <n-form-item label="存活路径">
            <n-input v-model:value="probePath" placeholder="/healthz" />
          </n-form-item>
          <n-form-item label="就绪路径">
            <n-input v-model:value="readinessPath" placeholder="/readyz" />
          </n-form-item>
        </div>

        <n-divider title-placement="left">环境变量（写入 ConfigMap 或内联 env）</n-divider>
        <env-var-editor :value="envPairs" @update:value="envPairs = $event" />

        <n-divider title-placement="left">密钥（写入 Secret）</n-divider>
        <env-var-editor :value="secretPairs" @update:value="secretPairs = $event" />

        <n-button type="primary" block size="large" class="submit-btn" :loading="rendering" :disabled="!canGenerate" @click="submit">
          {{ rendering ? '生成中…' : '生成 Kubernetes 配置' }}
        </n-button>
      </n-form>
    </n-card>

    <div class="result-column">
      <n-card title="依赖解析" size="small">
        <template #header-extra>
          <n-tag v-if="plan" size="small" :type="plan.valid ? 'success' : 'error'">
            {{ plan.valid ? '可执行' : '存在错误' }}
          </n-tag>
        </template>

        <n-spin v-if="planning" size="small" style="width: 100%; padding: 12px 0">
          <n-text depth="3" style="font-size: 12px">正在解析资源依赖…</n-text>
        </n-spin>

        <n-alert v-if="planError" type="error" :bordered="false">{{ planError }}</n-alert>

        <n-empty
          v-if="status === 'empty'"
          description="选择资源类型后自动解析依赖"
          style="padding: 20px 0"
        />

        <template v-if="plan">
          <div class="plan-summary">
            <n-tag size="small" :bordered="false">资源 {{ plan.nodes.length }}</n-tag>
            <n-tag size="small" :bordered="false">关系 {{ plan.edges.length }}</n-tag>
            <n-tag v-if="autoAdded.length" size="small" type="warning" :bordered="false">
              自动补充 {{ autoAdded.length }}
            </n-tag>
          </div>

          <div v-if="autoAdded.length" class="auto-added">
            <n-text depth="3" style="font-size: 12px">自动补充：</n-text>
            <n-tag v-for="node in autoAdded" :key="node.kind" size="tiny" type="warning">{{ node.kind }}</n-tag>
          </div>

          <div v-if="diagnostics.length" class="diag-list">
            <n-alert
              v-for="(item, index) in diagnostics"
              :key="index"
              :type="severityType(item.severity)"
              :bordered="false"
              size="small"
              class="diag-item"
            >
              {{ item.message }}
              <n-text v-if="item.hint" depth="3">（{{ item.hint }}）</n-text>
            </n-alert>
          </div>

          <div v-if="plan.nodes.length" class="graph-block">
            <div class="graph-head">
              <span>资源关系图</span>
              <n-button size="tiny" quaternary @click="graphExpanded = !graphExpanded">
                {{ graphExpanded ? '收起' : '展开' }}
              </n-button>
            </div>
            <resource-graph v-if="graphExpanded" :nodes="plan.nodes" :edges="plan.edges" />
          </div>
        </template>
      </n-card>

      <result-panel :record="record" />
    </div>
  </div>
</template>

<style scoped>
.generator-layout {
  display: grid;
  grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
  gap: 16px;
  align-items: start;
}
@media (max-width: 960px) {
  .generator-layout {
    grid-template-columns: 1fr;
  }
}
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}
.submit-btn {
  margin-top: 20px;
}
.result-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
  position: sticky;
  top: 84px;
}
@media (max-width: 960px) {
  .result-column {
    position: static;
  }
}
.plan-summary {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}
.auto-added {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}
.diag-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 8px;
}
.diag-item {
  font-size: 12px;
}
.graph-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  color: var(--cg-text-secondary);
  margin: 8px 0 6px;
}
</style>
