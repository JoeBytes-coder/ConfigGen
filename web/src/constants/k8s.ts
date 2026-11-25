/** 资源分类展示元信息（资源清单与依赖由后端 /api/v1/catalog 提供） */
export const CATEGORY_COLORS: Record<string, string> = {
  workload: '#C7DFF5',
  autoscaling: '#FBF3C4',
  networking: '#CBEBCB',
  config: '#FCE3C3',
  storage: '#DAD0EF',
  rbac: '#E2CCE8',
  policy: '#F8CFDC',
  scheduling: '#C4EBF0',
  cluster: '#E3F0E4',
}

export const CATEGORY_LABELS: Record<string, string> = {
  workload: '工作负载',
  autoscaling: '自动伸缩',
  networking: '网络',
  config: '配置',
  storage: '存储',
  rbac: 'RBAC',
  policy: '策略',
  scheduling: '调度',
  cluster: '集群',
}

export const CATEGORY_ORDER = [
  'workload',
  'autoscaling',
  'networking',
  'config',
  'storage',
  'rbac',
  'policy',
  'scheduling',
  'cluster',
]
