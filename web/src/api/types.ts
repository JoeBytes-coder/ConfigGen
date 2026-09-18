export type ConfigType = 'compose' | 'k8s' | 'dockerfile' | 'kustomize'

export interface EnvPair {
  key: string
  value: string
}

/** 与后端 internal/domain/types.go 的 ConfigRequest 对应 */
export interface GeneratePayload {
  type: ConfigType
  app_name: string
  image: string
  tag: string
  port: number
  env?: Record<string, string>
  replicas?: number
  k8s_resource?: string[]
  dockerfile_base_image?: string
  dockerfile_workdir?: string
  dockerfile_cmd?: string[]
}

export interface ResourceNode {
  kind: string
  name: string
}

export interface ResourceEdge {
  source_kind: string
  source_name: string
  target_kind: string
  target_name: string
  relation: string
}

export interface ConfigResult {
  type: ConfigType
  config: string
  resources?: ResourceNode[]
  edges?: ResourceEdge[]
}

export interface ConfigRecord {
  id: number
  request: GeneratePayload
  result: ConfigResult
  created_at: string
}

export interface GraphData {
  resources: ResourceNode[]
  edges: ResourceEdge[]
}

export interface ListResponse {
  records: ConfigRecord[]
  offset: number
  limit: number
}
