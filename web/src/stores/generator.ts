import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'

import { generateConfig } from '@/api/client'
import type { ConfigRecord, ConfigType, EnvPair, GeneratePayload } from '@/api/types'
import { WORKLOAD_KINDS } from '@/constants/k8s'

/**
 * 生成器表单与结果状态。
 * k8s 与 kustomize 各自维护一份独立的资源勾选集合（与旧版行为一致）。
 */
export const useGeneratorStore = defineStore('generator', () => {
  // ---- 表单状态 ----
  const type = ref<ConfigType>('k8s')
  const appName = ref('myapp')
  const image = ref('nginx')
  const tag = ref('latest')
  const port = ref(80)
  const envPairs = ref<EnvPair[]>([])
  const replicas = ref(1)
  const k8sResources = ref<string[]>([])
  const kustomizeResources = ref<string[]>([])
  const dockerfile = reactive({ baseImage: '', workDir: '/app', cmd: '' })

  // ---- 结果状态 ----
  const generating = ref(false)
  const lastRecord = ref<ConfigRecord | null>(null)

  const isK8sLike = computed(() => type.value === 'k8s' || type.value === 'kustomize')

  const resources = computed({
    get: () => (type.value === 'kustomize' ? kustomizeResources.value : k8sResources.value),
    set: (v: string[]) => {
      if (type.value === 'kustomize') {
        kustomizeResources.value = v
      } else {
        k8sResources.value = v
      }
    },
  })

  const hasWorkload = computed(() => resources.value.some((k) => WORKLOAD_KINDS.has(k)))

  function buildPayload(): GeneratePayload {
    const payload: GeneratePayload = {
      type: type.value,
      app_name: appName.value.trim(),
      image: image.value.trim(),
      tag: tag.value.trim(),
      port: port.value ?? 0,
    }

    const env: Record<string, string> = {}
    for (const p of envPairs.value) {
      const key = p.key.trim()
      if (key) env[key] = p.value.trim()
    }
    if (Object.keys(env).length > 0) payload.env = env

    if (isK8sLike.value) {
      payload.k8s_resource = [...resources.value]
      // replicas 仅在勾选了工作负载资源时发送（与旧版行为一致）
      if (hasWorkload.value) payload.replicas = replicas.value
    }

    if (type.value === 'dockerfile') {
      const baseImage = dockerfile.baseImage.trim()
      const workDir = dockerfile.workDir.trim()
      const cmd = dockerfile.cmd.trim()
      if (baseImage) payload.dockerfile_base_image = baseImage
      if (workDir) payload.dockerfile_workdir = workDir
      if (cmd) payload.dockerfile_cmd = cmd.split(/\s+/).filter(Boolean)
    }

    return payload
  }

  async function generate(): Promise<ConfigRecord> {
    generating.value = true
    try {
      lastRecord.value = await generateConfig(type.value, buildPayload())
      return lastRecord.value
    } finally {
      generating.value = false
    }
  }

  function resetResult() {
    lastRecord.value = null
  }

  return {
    type,
    appName,
    image,
    tag,
    port,
    envPairs,
    replicas,
    k8sResources,
    kustomizeResources,
    dockerfile,
    generating,
    lastRecord,
    isK8sLike,
    resources,
    hasWorkload,
    buildPayload,
    generate,
    resetResult,
  }
})
