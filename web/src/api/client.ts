import type { ConfigRecord, ConfigType, GeneratePayload, GraphData, ListResponse } from './types'

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init)
  const body = await res.json().catch(() => null)
  if (!res.ok) {
    const detail = body?.details ? `（${body.details}）` : ''
    throw new Error(body?.error ? `${body.error}${detail}` : `请求失败（HTTP ${res.status}）`)
  }
  return body as T
}

export function generateConfig(type: ConfigType, payload: GeneratePayload): Promise<ConfigRecord> {
  return request<ConfigRecord>(`/api/v1/generate/${type}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

export function listConfigs(offset: number, limit: number): Promise<ListResponse> {
  return request<ListResponse>(`/api/v1/configs?offset=${offset}&limit=${limit}`)
}

export function getConfig(id: number | string): Promise<ConfigRecord> {
  return request<ConfigRecord>(`/api/v1/configs/${id}`)
}

export function getGraph(id: number | string): Promise<GraphData> {
  return request<GraphData>(`/api/v1/configs/${id}/graph`)
}
