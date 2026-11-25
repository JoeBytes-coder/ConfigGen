import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'

import { planSelection, renderSelection } from '@/api/client'
import type { AppContext, ConfigRecord, Diagnostic, EnvPair, Selection } from '@/api/types'
import {
  canGenerate as canGenerateState,
  diagnosticsOf,
  errorOf,
  initialGenState,
  isPlanning,
  isRendering,
  planOf,
  recordOf,
  transition,
} from '@/stores/generatorMachine'
import type { GenEvent, GenState } from '@/stores/generatorMachine'
import { useMetaStore } from '@/stores/meta'

function pairsToObject(pairs: EnvPair[]): Record<string, string> | undefined {
  const out: Record<string, string> = {}
  for (const pair of pairs) {
    const key = pair.key.trim()
    if (key) out[key] = pair.value
  }
  return Object.keys(out).length > 0 ? out : undefined
}

function buildResources(cpuReq: string, memReq: string, cpuLim: string, memLim: string) {
  const requests: Record<string, string> = {}
  const limits: Record<string, string> = {}
  if (cpuReq.trim()) requests.cpu = cpuReq.trim()
  if (memReq.trim()) requests.memory = memReq.trim()
  if (cpuLim.trim()) limits.cpu = cpuLim.trim()
  if (memLim.trim()) limits.memory = memLim.trim()
  if (Object.keys(requests).length === 0 && Object.keys(limits).length === 0) return undefined
  return {
    requests: Object.keys(requests).length > 0 ? requests : undefined,
    limits: Object.keys(limits).length > 0 ? limits : undefined,
  }
}

export const useGeneratorStore = defineStore('generator', () => {
  const meta = useMetaStore()

  const appName = ref('myapp')
  const namespace = ref('')
  const image = ref('nginx')
  const tag = ref('1.25')
  const port = ref(8080)
  const replicas = ref(2)
  const host = ref('')
  const tlsSecret = ref('')

  const envPairs = ref<EnvPair[]>([])
  const secretPairs = ref<EnvPair[]>([])

  const cpuRequest = ref('100m')
  const memRequest = ref('128Mi')
  const cpuLimit = ref('500m')
  const memLimit = ref('512Mi')

  const probeEnabled = ref(false)
  const probePath = ref('/healthz')
  const readinessPath = ref('/readyz')

  const resources = ref<string[]>([])

  // 交互状态由 FSM 统一表达；以下派生值只读。
  const machine = ref<GenState>(initialGenState)
  const status = computed(() => machine.value.status)
  const plan = computed(() => planOf(machine.value))
  const planDiagnostics = computed(() => diagnosticsOf(machine.value))
  const planError = computed(() => errorOf(machine.value))
  const planning = computed(() => isPlanning(machine.value))
  const rendering = computed(() => isRendering(machine.value))
  const record = computed(() => recordOf(machine.value))
  const canGenerate = computed(() => canGenerateState(machine.value))

  const hasWorkload = computed(() => resources.value.some((kind) => meta.workloadKinds.has(kind)))

  function dispatch(event: GenEvent) {
    machine.value = transition(machine.value, event)
  }

  function buildSelection(): Selection {
    const app: AppContext = {
      name: appName.value.trim(),
      image: image.value.trim(),
      tag: tag.value.trim(),
    }
    if (port.value > 0) {
      app.ports = [{ name: 'http', container: port.value, service: port.value }]
    }
    app.replicas = replicas.value
    if (namespace.value.trim()) app.namespace = namespace.value.trim()
    if (host.value.trim()) app.host = host.value.trim()
    if (tlsSecret.value.trim()) app.tls_secret = tlsSecret.value.trim()
    const env = pairsToObject(envPairs.value)
    if (env) app.env = env
    const secretEnv = pairsToObject(secretPairs.value)
    if (secretEnv) app.secret_env = secretEnv
    const resourceRequirements = buildResources(cpuRequest.value, memRequest.value, cpuLimit.value, memLimit.value)
    if (resourceRequirements) app.resources = resourceRequirements
    app.probe = {
      enabled: probeEnabled.value,
      path: probePath.value,
      readiness_path: readinessPath.value,
    }

    return { type: 'k8s', app, resources: [...resources.value] }
  }

  let planRequestId = 0

  async function refreshPlan() {
    const requestId = ++planRequestId
    if (!appName.value.trim()) {
      dispatch({ type: 'PLAN_FAILED', error: '请填写应用名称', diagnostics: [] })
      return
    }
    if (resources.value.length === 0) {
      dispatch({ type: 'SELECTION_CHANGED', hasResources: false })
      return
    }
    dispatch({ type: 'SELECTION_CHANGED', hasResources: true })
    try {
      const result = await planSelection(buildSelection())
      // 只接受最新一次请求的结果，避免快速输入时旧响应覆盖新 Plan。
      if (requestId !== planRequestId) return
      dispatch({ type: 'PLAN_SUCCEEDED', plan: result })
    } catch (e) {
      if (requestId !== planRequestId) return
      const diags = (e as { diagnostics?: Diagnostic[] }).diagnostics
      dispatch({
        type: 'PLAN_FAILED',
        error: e instanceof Error ? e.message : '依赖解析失败',
        diagnostics: Array.isArray(diags) ? diags : [],
      })
    }
  }

  let planTimer: number | undefined

  function schedulePlan() {
    if (planTimer) window.clearTimeout(planTimer)
    planTimer = window.setTimeout(() => {
      void refreshPlan()
    }, 300)
  }

  function onSelectionChanged() {
    dispatch({ type: 'SELECTION_CHANGED', hasResources: resources.value.length > 0 })
    if (resources.value.length === 0) return
    schedulePlan()
  }

  async function generate(): Promise<ConfigRecord> {
    if (!canGenerate.value) {
      throw new Error('当前状态不可生成，请先完成依赖解析')
    }
    dispatch({ type: 'RENDER_STARTED' })
    try {
      const result = await renderSelection(buildSelection())
      if (machine.value.status !== 'rendering') {
        throw new Error('生成期间选择已变更，请重新生成')
      }
      dispatch({ type: 'RENDER_SUCCEEDED', plan: result.plan, record: result })
      return result
    } catch (e) {
      dispatch({ type: 'RENDER_FAILED' })
      throw e
    }
  }

  watch(
    [
      appName, namespace, image, tag, port, replicas, host, tlsSecret,
      envPairs, secretPairs, cpuRequest, memRequest, cpuLimit, memLimit,
      probeEnabled, probePath, readinessPath, resources,
    ],
    onSelectionChanged,
    { deep: true },
  )

  return {
    appName, namespace, image, tag, port, replicas, host, tlsSecret,
    envPairs, secretPairs, cpuRequest, memRequest, cpuLimit, memLimit,
    probeEnabled, probePath, readinessPath,
    resources, hasWorkload, status, canGenerate, plan, planError, planDiagnostics, planning, record, rendering,
    buildSelection, refreshPlan, generate,
  }
})
