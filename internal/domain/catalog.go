package domain

// Scope 表示资源作用域。
type Scope string

const (
	Namespaced Scope = "namespaced"
	Cluster    Scope = "cluster"
)

// RequireMode 描述依赖缺失时的处理方式。
type RequireMode string

const (
	ModeAutoAdd RequireMode = "auto_add"
	ModeMust    RequireMode = "must"
)

type Cardinality string

const (
	CardinalityOne        Cardinality = "one"
	CardinalityAtLeastOne Cardinality = "at_least_one"
)

// Require 声明一个资源对另一种资源的依赖。
type Require struct {
	Kind     string      `json:"kind"`
	Relation string      `json:"relation"`
	Mode     RequireMode `json:"mode"`
	Cardinal Cardinality `json:"cardinality"`
}

// Needs 声明该资源需要哪些应用级参数。
type Needs struct {
	Image    bool `json:"image"`
	Port     bool `json:"port"`
	Replicas bool `json:"replicas"`
}

// Spec 是资源目录条目，命名/作用域/依赖/排序的唯一事实源。
type Spec struct {
	Kind       string
	APIVersion string
	Category   string
	Scope      Scope
	Suffix     string
	Level      int
	Needs      Needs
	Requires   []Require
}

// 分类常量，前端据此分组。
const (
	CategoryWorkload    = "workload"
	CategoryAutoscaling = "autoscaling"
	CategoryNetworking  = "networking"
	CategoryConfig      = "config"
	CategoryStorage     = "storage"
	CategoryRBAC        = "rbac"
	CategoryPolicy      = "policy"
	CategoryScheduling  = "scheduling"
	CategoryCluster     = "cluster"
)

// Catalog 是全部受支持的 K8s 资源（26 种）。
var Catalog = []Spec{
	{Kind: "Deployment", APIVersion: "apps/v1", Category: CategoryWorkload, Scope: Namespaced, Suffix: "deployment", Level: 2, Needs: Needs{Image: true, Replicas: true}},
	{Kind: "StatefulSet", APIVersion: "apps/v1", Category: CategoryWorkload, Scope: Namespaced, Suffix: "statefulset", Level: 2, Needs: Needs{Image: true, Replicas: true}},
	{Kind: "DaemonSet", APIVersion: "apps/v1", Category: CategoryWorkload, Scope: Namespaced, Suffix: "daemonset", Level: 2, Needs: Needs{Image: true}},
	{Kind: "CronJob", APIVersion: "batch/v1", Category: CategoryWorkload, Scope: Namespaced, Suffix: "cronjob", Level: 2, Needs: Needs{Image: true}},
	{Kind: "Job", APIVersion: "batch/v1", Category: CategoryWorkload, Scope: Namespaced, Suffix: "job", Level: 2, Needs: Needs{Image: true}},
	{Kind: "ReplicaSet", APIVersion: "apps/v1", Category: CategoryWorkload, Scope: Namespaced, Suffix: "replicaset", Level: 2, Needs: Needs{Image: true, Replicas: true}},

	{Kind: "HorizontalPodAutoscaler", APIVersion: "autoscaling/v2", Category: CategoryAutoscaling, Scope: Namespaced, Suffix: "hpa", Level: 3},

	{Kind: "Service", APIVersion: "v1", Category: CategoryNetworking, Scope: Namespaced, Suffix: "service", Level: 3, Needs: Needs{Port: true}},
	{Kind: "Ingress", APIVersion: "networking.k8s.io/v1", Category: CategoryNetworking, Scope: Namespaced, Suffix: "ingress", Level: 4, Needs: Needs{Port: true},
		Requires: []Require{{Kind: "Service", Relation: "routes-to", Mode: ModeAutoAdd, Cardinal: CardinalityOne}}},
	{Kind: "IngressClass", APIVersion: "networking.k8s.io/v1", Category: CategoryNetworking, Scope: Cluster, Suffix: "ingressclass", Level: 0},
	{Kind: "EndpointSlice", APIVersion: "discovery.k8s.io/v1", Category: CategoryNetworking, Scope: Namespaced, Suffix: "endpointslice", Level: 4, Needs: Needs{Port: true},
		Requires: []Require{{Kind: "Service", Relation: "belongs-to", Mode: ModeAutoAdd, Cardinal: CardinalityOne}}},
	{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1", Category: CategoryNetworking, Scope: Namespaced, Suffix: "netpol", Level: 3},

	{Kind: "ConfigMap", APIVersion: "v1", Category: CategoryConfig, Scope: Namespaced, Suffix: "configmap", Level: 0},
	{Kind: "Secret", APIVersion: "v1", Category: CategoryConfig, Scope: Namespaced, Suffix: "secret", Level: 0},

	{Kind: "PersistentVolumeClaim", APIVersion: "v1", Category: CategoryStorage, Scope: Namespaced, Suffix: "pvc", Level: 1},
	{Kind: "StorageClass", APIVersion: "storage.k8s.io/v1", Category: CategoryStorage, Scope: Cluster, Suffix: "storageclass", Level: 0},

	{Kind: "ServiceAccount", APIVersion: "v1", Category: CategoryRBAC, Scope: Namespaced, Suffix: "sa", Level: 0},
	{Kind: "Role", APIVersion: "rbac.authorization.k8s.io/v1", Category: CategoryRBAC, Scope: Namespaced, Suffix: "role", Level: 1},
	{Kind: "RoleBinding", APIVersion: "rbac.authorization.k8s.io/v1", Category: CategoryRBAC, Scope: Namespaced, Suffix: "rolebinding", Level: 1,
		Requires: []Require{{Kind: "ServiceAccount", Relation: "grants-to", Mode: ModeAutoAdd, Cardinal: CardinalityOne}}},
	{Kind: "ClusterRole", APIVersion: "rbac.authorization.k8s.io/v1", Category: CategoryRBAC, Scope: Cluster, Suffix: "clusterrole", Level: 0},
	{Kind: "ClusterRoleBinding", APIVersion: "rbac.authorization.k8s.io/v1", Category: CategoryRBAC, Scope: Cluster, Suffix: "clusterrolebinding", Level: 1,
		Requires: []Require{
			{Kind: "ServiceAccount", Relation: "grants-to", Mode: ModeAutoAdd, Cardinal: CardinalityOne},
			{Kind: "ClusterRole", Relation: "binds", Mode: ModeAutoAdd, Cardinal: CardinalityOne},
		}},

	{Kind: "ResourceQuota", APIVersion: "v1", Category: CategoryPolicy, Scope: Namespaced, Suffix: "quota", Level: 0},
	{Kind: "LimitRange", APIVersion: "v1", Category: CategoryPolicy, Scope: Namespaced, Suffix: "limits", Level: 0},
	{Kind: "PodDisruptionBudget", APIVersion: "policy/v1", Category: CategoryPolicy, Scope: Namespaced, Suffix: "pdb", Level: 3},

	{Kind: "PriorityClass", APIVersion: "scheduling.k8s.io/v1", Category: CategoryScheduling, Scope: Cluster, Suffix: "priorityclass", Level: 0},

	{Kind: "Namespace", APIVersion: "v1", Category: CategoryCluster, Scope: Cluster, Suffix: "namespace", Level: 0},
}

var catalogByKind = func() map[string]Spec {
	m := make(map[string]Spec, len(Catalog))
	for _, s := range Catalog {
		m[s.Kind] = s
	}
	return m
}()

func SpecByKind(kind string) (Spec, bool) {
	s, ok := catalogByKind[kind]
	return s, ok
}

// WorkloadKinds 是全部工作负载类型。
var WorkloadKinds = []string{"Deployment", "StatefulSet", "DaemonSet", "CronJob", "Job", "ReplicaSet"}

// ScalableKinds 是 HPA 可指向的工作负载。
var ScalableKinds = []string{"Deployment", "StatefulSet", "ReplicaSet"}

// KindView 是暴露给前端的目录条目。
type KindView struct {
	Kind          string    `json:"kind"`
	APIVersion    string    `json:"api_version"`
	Category      string    `json:"category"`
	Scope         Scope     `json:"scope"`
	Suffix        string    `json:"suffix"`
	Level         int       `json:"level"`
	NeedsImage    bool      `json:"needs_image"`
	NeedsPort     bool      `json:"needs_port"`
	NeedsReplicas bool      `json:"needs_replicas"`
	Requires      []Require `json:"requires,omitempty"`
}

// CatalogPayload 是 GET /api/v1/catalog 的响应。
type CatalogPayload struct {
	Version string     `json:"version"`
	Kinds   []KindView `json:"kinds"`
}

func CatalogView() CatalogPayload {
	views := make([]KindView, 0, len(Catalog))
	for _, s := range Catalog {
		views = append(views, KindView{
			Kind: s.Kind, APIVersion: s.APIVersion, Category: s.Category, Scope: s.Scope,
			Suffix: s.Suffix, Level: s.Level,
			NeedsImage: s.Needs.Image, NeedsPort: s.Needs.Port, NeedsReplicas: s.Needs.Replicas,
			Requires: s.Requires,
		})
	}
	return CatalogPayload{Version: CatalogVersion, Kinds: views}
}
