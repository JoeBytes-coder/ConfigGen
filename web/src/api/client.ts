import type {
  Catalog,
  ConfigRecord,
  ListResponse,
  Selection,
} from './types'

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init)
  const body = await res.json().catch(() => null)
  if (!res.ok) {
    const detail = body?.details ? '（' + body.details + '）' : ''
    const message = (body?.error ?? '请求失败（HTTP ' + res.status + '）') + detail
    const error = new Error(message) as Error & { diagnostics?: unknown }
    error.diagnostics = body?.diagnostics
    throw error
  }
  return body as T
}

export function getCatalog(): Promise<Catalog> {
  return request<Catalog>('/api/v1/catalog')
}

export function planSelection(selection: Selection): Promise<import('./types').Plan> {
  return request('/api/v1/plan', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(selection),
  })
}

export function renderSelection(selection: Selection): Promise<ConfigRecord> {
  return request<ConfigRecord>('/api/v1/render', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(selection),
  })
}

export function listRecords(offset: number, limit: number): Promise<ListResponse> {
  return request<ListResponse>('/api/v1/records?offset=' + offset + '&limit=' + limit)
}

export function getRecord(id: number | string): Promise<ConfigRecord> {
  return request<ConfigRecord>('/api/v1/records/' + id)
}
