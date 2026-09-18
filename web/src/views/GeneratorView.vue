<script setup lang="ts">
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import {
  NAlert,
  NButton,
  NCard,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NTabs,
  NTabPane,
  useMessage,
} from 'naive-ui'

import EnvVarEditor from '@/components/EnvVarEditor.vue'
import ResourceGraph from '@/components/ResourceGraph.vue'
import ResourcePicker from '@/components/ResourcePicker.vue'
import ResultPanel from '@/components/ResultPanel.vue'
import { useGeneratorStore } from '@/stores/generator'

const message = useMessage()
const store = useGeneratorStore()
const {
  type,
  appName,
  image,
  tag,
  port,
  envPairs,
  replicas,
  dockerfile,
  generating,
  lastRecord,
  isK8sLike,
  resources,
  hasWorkload,
} = storeToRefs(store)

const graphExpanded = ref(true)

const showGraph = computed(
  () => isK8sLike && !!lastRecord.value && (lastRecord.value.result.resources?.length ?? 0) > 0,
)

function onTypeChange(v: string) {
  type.value = v as typeof type.value
  store.resetResult()
}

async function submit() {
  if (!appName.value.trim()) {
    message.warning('请填写应用名称')
    return
  }
  if (!image.value.trim() || !tag.value.trim()) {
    message.warning('请填写镜像名称与标签')
    return
  }
  if (!port.value || port.value < 1) {
    message.warning('请填写端口（1 - 65535）')
    return
  }
  if (isK8sLike && resources.value.length === 0) {
    message.warning('请选择至少一种 Kubernetes 资源类型')
    return
  }

  try {
    await store.generate()
    message.success('配置生成成功')
  } catch (e) {
    message.error(e instanceof Error ? e.message : '生成失败，请重试')
  }
}
</script>

<template>
  <div class="generator-layout">
    <!-- 左列：配置表单 -->
    <n-card title="配置参数" size="small">
      <n-form label-placement="top" :show-feedback="false" class="generator-form">
        <n-form-item label="应用名称">
          <n-input v-model:value="appName" placeholder="例如 myapp，用作资源名前缀" />
        </n-form-item>

        <div class="form-grid">
          <n-form-item label="镜像">
            <n-input v-model:value="image" placeholder="例如 nginx" />
          </n-form-item>
          <n-form-item label="标签">
            <n-input v-model:value="tag" placeholder="例如 latest" />
          </n-form-item>
        </div>

        <n-form-item label="端口">
          <n-input-number v-model:value="port" :min="1" :max="65535" style="width: 100%" />
        </n-form-item>

        <n-form-item label="环境变量">
          <env-var-editor v-model:value="envPairs" />
        </n-form-item>

        <n-tabs
          type="segment"
          animated
          :value="type"
          class="type-tabs"
          @update:value="onTypeChange"
        >
          <n-tab-pane name="dockerfile" tab="Dockerfile">
            <n-form-item label="基础镜像">
              <n-input v-model:value="dockerfile.baseImage" placeholder="例如 golang:1.25-alpine" />
            </n-form-item>
            <n-form-item label="工作目录">
              <n-input v-model:value="dockerfile.workDir" placeholder="/app" />
            </n-form-item>
            <n-form-item label="启动命令">
              <n-input v-model:value="dockerfile.cmd" placeholder="例如 ./myapp --port 8080" />
            </n-form-item>
            <n-alert type="info" :bordered="false" size="small">
              基础镜像与启动命令留空时，将分别使用「镜像:标签」与应用名称
            </n-alert>
          </n-tab-pane>

          <n-tab-pane name="compose" tab="Docker Compose">
            <n-alert type="info" :bordered="false">
              Docker Compose 配置使用上方公共字段（应用名称、镜像、标签、端口）与环境变量
            </n-alert>
          </n-tab-pane>

          <n-tab-pane name="k8s" tab="Kubernetes">
            <n-form-item label="副本数">
              <n-input-number
                v-model:value="replicas"
                :min="1"
                :max="100"
                style="width: 100%"
                :disabled="!hasWorkload"
              />
            </n-form-item>
            <n-form-item label="Kubernetes 资源类型（可多选）">
              <resource-picker v-model="resources" />
            </n-form-item>
          </n-tab-pane>

          <n-tab-pane name="kustomize" tab="Kustomize">
            <n-form-item label="副本数">
              <n-input-number
                v-model:value="replicas"
                :min="1"
                :max="100"
                style="width: 100%"
                :disabled="!hasWorkload"
              />
            </n-form-item>
            <n-form-item label="Kubernetes 资源类型（可多选）">
              <resource-picker v-model="resources" />
            </n-form-item>
          </n-tab-pane>
        </n-tabs>

        <n-button type="primary" block size="large" :loading="generating" class="submit-btn" @click="submit">
          {{ generating ? '生成中…' : '生成配置' }}
        </n-button>
      </n-form>
    </n-card>

    <!-- 右列：结果与拓扑图 -->
    <div class="result-column">
      <result-panel :record="lastRecord" />

      <n-card v-if="showGraph" size="small" class="graph-card">
        <template #header>
          资源关系图
          <n-button size="tiny" quaternary style="margin-left: 8px" @click="graphExpanded = !graphExpanded">
            {{ graphExpanded ? '收起' : '展开' }}
          </n-button>
        </template>
        <resource-graph
          v-if="graphExpanded && lastRecord"
          :nodes="lastRecord.result.resources ?? []"
          :edges="lastRecord.result.edges ?? []"
        />
      </n-card>
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

.generator-form :deep(.n-form-item) {
  margin-bottom: 16px;
}

.type-tabs {
  margin-top: 4px;
}

.type-tabs :deep(.n-tab-pane) {
  padding-top: 16px;
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
</style>
