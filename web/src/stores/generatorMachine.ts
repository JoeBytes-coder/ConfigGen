import type { ConfigRecord, Diagnostic, Plan } from '@/api/types'

/** 生成器交互状态；所有 UI 组合都必须能从这些状态推导出来。 */
export type GenStatus = 'empty' | 'planning' | 'planned' | 'plan_error' | 'rendering' | 'rendered'

export type GenState =
  | { status: 'empty' }
  | { status: 'planning' }
  | { status: 'planned'; plan: Plan }
  | { status: 'plan_error'; error: string; diagnostics: Diagnostic[] }
  | { status: 'rendering'; plan: Plan }
  | { status: 'rendered'; plan: Plan; record: ConfigRecord }

export type GenEvent =
  | { type: 'SELECTION_CHANGED'; hasResources: boolean }
  | { type: 'PLAN_SUCCEEDED'; plan: Plan }
  | { type: 'PLAN_FAILED'; error: string; diagnostics: Diagnostic[] }
  | { type: 'RENDER_STARTED' }
  | { type: 'RENDER_SUCCEEDED'; plan: Plan; record: ConfigRecord }
  | { type: 'RENDER_FAILED' }
  | { type: 'RESET' }

export const initialGenState: GenState = { status: 'empty' }

/**
 * 纯迁移函数：只允许表中声明的状态变化，其余事件一律忽略。
 * 异步副作用（请求、防抖、提示）留在 store，保证迁移可单测。
 */
export function transition(state: GenState, event: GenEvent): GenState {
  if (event.type === 'RESET') return { status: 'empty' }
  if (event.type === 'SELECTION_CHANGED') {
    return event.hasResources ? { status: 'planning' } : { status: 'empty' }
  }
  switch (event.type) {
    case 'PLAN_SUCCEEDED':
      return state.status === 'planning' ? { status: 'planned', plan: event.plan } : state
    case 'PLAN_FAILED':
      return state.status === 'planning' || state.status === 'empty'
        ? { status: 'plan_error', error: event.error, diagnostics: event.diagnostics }
        : state
    case 'RENDER_STARTED':
      return state.status === 'planned' || state.status === 'rendered'
        ? { status: 'rendering', plan: state.plan }
        : state
    case 'RENDER_SUCCEEDED':
      return state.status === 'rendering' ? { status: 'rendered', plan: event.plan, record: event.record } : state
    case 'RENDER_FAILED':
      return state.status === 'rendering' ? { status: 'planned', plan: state.plan } : state
    default:
      return state
  }
}

export function planOf(state: GenState): Plan | null {
  switch (state.status) {
    case 'planned':
    case 'rendering':
    case 'rendered':
      return state.plan
    default:
      return null
  }
}

export function diagnosticsOf(state: GenState): Diagnostic[] {
  if (state.status === 'plan_error') return state.diagnostics
  return planOf(state)?.diagnostics ?? []
}

export function errorOf(state: GenState): string {
  return state.status === 'plan_error' ? state.error : ''
}

export function recordOf(state: GenState): ConfigRecord | null {
  return state.status === 'rendered' ? state.record : null
}

export function isPlanning(state: GenState): boolean {
  return state.status === 'planning'
}

export function isRendering(state: GenState): boolean {
  return state.status === 'rendering'
}

export function canGenerate(state: GenState): boolean {
  return state.status === 'planned' || state.status === 'rendered'
}
