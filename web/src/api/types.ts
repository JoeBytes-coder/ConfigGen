export type ConfigType = 'k8s'

export interface EnvPair {
  key: string
  value: string
}

export interface Port {
  name: string
  container: number
  service: number
}

export interface ResourceRequirements {
  requests?: Record<string, string>
  limits?: Record<string, string>
}

export interface ProbeConfig {
  enabled: boolean
  path?: string
  readiness_path?: string
}

export interface AppContext {
  name: string
  namespace?: string
  image?: string
  tag?: string
  ports?: Port[]
  replicas?: number
  env?: Record<string, string>
  secret_env?: Record<string, string>
  resources?: ResourceRequirements
  probe?: ProbeConfig
  host?: string
  tls_secret?: string
}

export interface Selection {
  type: ConfigType
  app: AppContext
  resources?: string[]
}

export interface Ref {
  kind: string
  name: string
  port?: string
}

export interface PlanNode {
  kind: string
  name: string
  namespace?: string
  level: number
  auto_added: boolean
  refs?: Record<string, Ref>
}

export interface PlanEdge {
  from: string
  to: string
  relation: string
}

export type Severity = 'error' | 'warning' | 'info'

export interface Diagnostic {
  severity: Severity
  code: string
  kind?: string
  message: string
  hint?: string
}

export interface Plan {
  type: ConfigType
  app: AppContext
  nodes: PlanNode[]
  edges: PlanEdge[]
  diagnostics: Diagnostic[]
  valid: boolean
  catalog_version: string
}

export interface Artifact {
  path: string
  kind: string
  content: string
}

export interface ConfigRecord {
  id: number
  type: ConfigType
  app: AppContext
  selection: Selection
  plan: Plan
  files: Artifact[]
  bundle: string
  catalog_version: string
  created_at: string
}

export interface RecordSummary {
  id: number
  type: ConfigType
  app_name: string
  namespace?: string
  kinds: string[]
  created_at: string
}

export interface CatalogRequire {
  kind: string
  relation: string
  mode: string
  cardinality: string
}

export interface CatalogKind {
  kind: string
  api_version: string
  category: string
  scope: 'namespaced' | 'cluster'
  suffix: string
  level: number
  needs_image: boolean
  needs_port: boolean
  needs_replicas: boolean
  requires?: CatalogRequire[]
}

export interface Catalog {
  version: string
  kinds: CatalogKind[]
}

export interface ListResponse {
  records: RecordSummary[]
  total: number
  offset: number
  limit: number
}

export interface ApiError extends Error {
  diagnostics?: Diagnostic[]
}
