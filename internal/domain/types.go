package domain

import "time"

// CatalogVersion 随资源目录结构变化递增；历史记录保存生成时使用的版本。
const CatalogVersion = "2.0.0"

// ConfigType 是生成产物的类型；当前仅支持 Kubernetes。
type ConfigType string

const TypeK8s ConfigType = "k8s"

func (t ConfigType) Valid() bool { return t == TypeK8s }

// Port 描述一个应用端口，容器端口与服务端口分离，避免端口漂移。
type Port struct {
	Name      string `json:"name"`
	Container int    `json:"container"`
	Service   int    `json:"service"`
}

type ResourceRequirements struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

type ProbeConfig struct {
	Enabled       bool   `json:"enabled"`
	Path          string `json:"path,omitempty"`
	ReadinessPath string `json:"readiness_path,omitempty"`
}

// AppContext 是生成一份配置所需的全部应用级参数（唯一事实源）。
type AppContext struct {
	Name      string               `json:"name"`
	Namespace string               `json:"namespace,omitempty"`
	Image     string               `json:"image,omitempty"`
	Tag       string               `json:"tag,omitempty"`
	Ports     []Port               `json:"ports,omitempty"`
	Replicas  int32                `json:"replicas,omitempty"`
	Env       map[string]string    `json:"env,omitempty"`
	SecretEnv map[string]string    `json:"secret_env,omitempty"`
	Resources ResourceRequirements `json:"resources,omitempty"`
	Probe     ProbeConfig          `json:"probe,omitempty"`
	Host      string               `json:"host,omitempty"`
	TLSSecret string               `json:"tls_secret,omitempty"`
}

// ImageRef 返回 image:tag。
func (a AppContext) ImageRef() string {
	if a.Image == "" {
		return ""
	}
	if a.Tag == "" {
		return a.Image
	}
	return a.Image + ":" + a.Tag
}

// Selection 是前端提交的选择：应用参数 + 资源类型集合。
type Selection struct {
	Type      ConfigType `json:"type"`
	App       AppContext `json:"app"`
	Resources []string   `json:"resources,omitempty"`
}

type Artifact struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
}

// Ref 是解析后的跨资源引用（kind + name + 可选端口名）。
type Ref struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Port string `json:"port,omitempty"`
}

// Node 是 Plan 中的一个资源节点。
type Node struct {
	Kind      string         `json:"kind"`
	Name      string         `json:"name"`
	Namespace string         `json:"namespace,omitempty"`
	Level     int            `json:"level"`
	AutoAdded bool           `json:"auto_added"`
	Refs      map[string]Ref `json:"refs,omitempty"`
}

type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Diagnostic 是解析/校验阶段产生的可解释诊断。
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Kind     string   `json:"kind,omitempty"`
	Message  string   `json:"message"`
	Hint     string   `json:"hint,omitempty"`
}

// Plan 是「选择 -> 依赖解析 -> 命名/引用绑定」的结果，渲染只读 Plan。
type Plan struct {
	Type           ConfigType   `json:"type"`
	App            AppContext   `json:"app"`
	Nodes          []Node       `json:"nodes"`
	Edges          []Edge       `json:"edges"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
	Valid          bool         `json:"valid"`
	CatalogVersion string       `json:"catalog_version"`
}

// Record 是一次生成的完整快照。
type Record struct {
	ID             int64      `json:"id"`
	Type           ConfigType `json:"type"`
	App            AppContext `json:"app"`
	Selection      Selection  `json:"selection"`
	Plan           Plan       `json:"plan"`
	Files          []Artifact `json:"files"`
	Bundle         string     `json:"bundle"`
	CatalogVersion string     `json:"catalog_version"`
	CreatedAt      time.Time  `json:"created_at"`
}

// RecordSummary 是历史列表用的轻量投影，避免列表返回整份配置。
type RecordSummary struct {
	ID        int64      `json:"id"`
	Type      ConfigType `json:"type"`
	AppName   string     `json:"app_name"`
	Namespace string     `json:"namespace,omitempty"`
	Kinds     []string   `json:"kinds"`
	CreatedAt time.Time  `json:"created_at"`
}
