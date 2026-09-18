/** 后端支持的 20 种 K8s 资源类型 */
export const K8S_KINDS = [
  'Namespace',
  'Deployment',
  'StatefulSet',
  'DaemonSet',
  'Service',
  'Ingress',
  'ConfigMap',
  'Secret',
  'PersistentVolumeClaim',
  'StorageClass',
  'CronJob',
  'ServiceAccount',
  'HorizontalPodAutoscaler',
  'ResourceQuota',
  'LimitRange',
  'NetworkPolicy',
  'Role',
  'RoleBinding',
  'ClusterRole',
  'ClusterRoleBinding',
] as const

/** 工作负载类型：勾选后 replicas 才会随请求发送 */
export const WORKLOAD_KINDS = new Set(['Deployment', 'StatefulSet', 'DaemonSet', 'CronJob'])

/** 拓扑图节点配色（按资源分类的柔和色系，与主色 #7DAEE1 协调） */
export const KIND_COLORS: Record<string, string> = {
  Namespace: '#E3F0E4',
  Deployment: '#C7DFF5',
  StatefulSet: '#C7DFF5',
  DaemonSet: '#C7DFF5',
  CronJob: '#C7DFF5',
  Service: '#CBEBCB',
  Ingress: '#FBF3C4',
  ConfigMap: '#FCE3C3',
  Secret: '#F9D4D6',
  PersistentVolumeClaim: '#DAD0EF',
  StorageClass: '#DAD0EF',
  ServiceAccount: '#C4EBF0',
  HorizontalPodAutoscaler: '#EFF3C8',
  ResourceQuota: '#F8CFDC',
  LimitRange: '#F8CFDC',
  NetworkPolicy: '#CBEBE2',
  Role: '#E2CCE8',
  RoleBinding: '#E2CCE8',
  ClusterRole: '#E2CCE8',
  ClusterRoleBinding: '#E2CCE8',
}

/** 配置类型的展示元信息 */
export const TYPE_META: Record<string, { label: string; color: string; textColor: string; border: string }> = {
  dockerfile: { label: 'Dockerfile', color: '#EFEBFA', textColor: '#6F5CC7', border: '#D8CEF0' },
  compose: { label: 'Docker Compose', color: '#E1F3EA', textColor: '#2F8F63', border: '#C4E6D5' },
  k8s: { label: 'Kubernetes', color: '#E7F0FA', textColor: '#3D6EA8', border: '#CADFF2' },
  kustomize: { label: 'Kustomize', color: '#F7E8F1', textColor: '#A5588A', border: '#ECCFE0' },
}
