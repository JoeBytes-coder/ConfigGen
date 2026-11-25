import assert from 'node:assert/strict'
import test from 'node:test'

import { canGenerate, errorOf, initialGenState, isPlanning, planOf, recordOf, transition } from './generatorMachine.ts'

const plan = {
  type: 'k8s',
  app: { name: 'demo' },
  nodes: [],
  edges: [],
  diagnostics: [],
  valid: true,
  catalog_version: '2.0.0',
} as any

const record = {
  id: 1,
  type: 'k8s',
  app: { name: 'demo' },
  selection: { type: 'k8s', app: { name: 'demo' } },
  plan,
  files: [],
  bundle: '',
  catalog_version: '2.0.0',
  created_at: '2026-01-01T00:00:00Z',
} as any

test('happy path: empty -> planning -> planned -> rendering -> rendered', () => {
  let state = transition(initialGenState, { type: 'SELECTION_CHANGED', hasResources: true })
  assert.equal(state.status, 'planning')
  assert.equal(isPlanning(state), true)

  state = transition(state, { type: 'PLAN_SUCCEEDED', plan })
  assert.equal(state.status, 'planned')
  assert.equal(canGenerate(state), true)
  assert.equal(planOf(state)?.catalog_version, '2.0.0')

  state = transition(state, { type: 'RENDER_STARTED' })
  assert.equal(state.status, 'rendering')

  state = transition(state, { type: 'RENDER_SUCCEEDED', plan, record })
  assert.equal(state.status, 'rendered')
  assert.equal(recordOf(state)?.id, 1)
})

test('plan failure blocks generation and keeps diagnostics', () => {
  let state = transition(initialGenState, { type: 'SELECTION_CHANGED', hasResources: true })
  state = transition(state, { type: 'PLAN_FAILED', error: '配置校验失败', diagnostics: [{ severity: 'error', code: 'x', message: 'bad' }] })
  assert.equal(state.status, 'plan_error')
  assert.equal(errorOf(state), '配置校验失败')
  assert.equal(canGenerate(state), false)
  assert.equal(transition(state, { type: 'RENDER_STARTED' }), state)
})

test('selection change from rendered invalidates result', () => {
  let state = transition(initialGenState, { type: 'SELECTION_CHANGED', hasResources: true })
  state = transition(state, { type: 'PLAN_SUCCEEDED', plan })
  state = transition(state, { type: 'RENDER_STARTED' })
  state = transition(state, { type: 'RENDER_SUCCEEDED', plan, record })
  state = transition(state, { type: 'SELECTION_CHANGED', hasResources: true })
  assert.equal(state.status, 'planning')
  assert.equal(recordOf(state), null)
})

test('render failure returns to planned and keeps plan', () => {
  let state = transition(initialGenState, { type: 'SELECTION_CHANGED', hasResources: true })
  state = transition(state, { type: 'PLAN_SUCCEEDED', plan })
  state = transition(state, { type: 'RENDER_STARTED' })
  state = transition(state, { type: 'RENDER_FAILED' })
  assert.equal(state.status, 'planned')
  assert.equal(planOf(state)?.valid, true)
})

test('clearing resources returns to empty; reset clears everything', () => {
  let state = transition(initialGenState, { type: 'SELECTION_CHANGED', hasResources: true })
  state = transition(state, { type: 'PLAN_SUCCEEDED', plan })
  state = transition(state, { type: 'SELECTION_CHANGED', hasResources: false })
  assert.equal(state.status, 'empty')
  state = transition({ status: 'rendered', plan, record }, { type: 'RESET' })
  assert.equal(state.status, 'empty')
})
