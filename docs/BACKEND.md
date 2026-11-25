# ConfigGen 后端设计与实现规范

> 适用范围：cmd、config、internal/domain、internal/usecase、internal/infrastructure、internal/presentation。
> 变更后端代码时必须同步更新本文档；前端规范见 docs/FRONTEND.md，总览见 docs/DESIGN.md。

## 1. 技术栈与运行形态

| 项 | 选型 | 说明 |
| --- | --- | --- |
| 语言 | Go 1.25 | go.mod 中 go 1.25.0 |
| HTTP | gin-gonic/gin v1.9.0 | 路由、Recovery、静态 SPA |
| K8s 类型 | k8s.io/api v0.31.0 + apimachinery v0.31.0 | 渲染直接构造官方对象 |
| YAML | sigs.k8s.io/yaml v1.4.0 | JSON tag 序列化为 YAML |
| 存储 | modernc.org/sqlite v1.34.5 | 纯 Go，无 cgo |
| 日志 | log/slog | JSON handler，统一 http_request 日志 |
| 前端 | go:embed all:dist | 单二进制分发 |

运行形态：单进程 HTTP 服务 + 本地 SQLite + 内嵌 SPA。无外部中间件、无运行时代码生成、无模板热加载。

## 2. 分层与依赖

~~~
cmd/configgen/main.go        DI、http.Server、优雅停机
config/                      .env 加载、监听地址
internal/domain/             实体、目录、解析、校验、错误、Store 接口
internal/usecase/            Plan / Render / 查询 / 健康检查
internal/infrastructure/
  builders/                  26 种资源的类型化 builder
  render/                    K8s 渲染与产物校验
  storage/                   SQLite 快照存储
internal/presentation/       Gin 路由、JSON 解析、错误映射、SPA
~~~

强制规则：

1. 依赖方向只能是 presentation → usecase → domain ← infrastructure。
2. domain 不得导入 gin、database/sql、k8s.io/api、text/template。
3. builder 不得做依赖推导、不得自行拼资源名；名字与引用只能来自 domain.Plan。
4. presentation 不得包含业务校验；只负责解码、调用 usecase、映射 HTTP 状态码。
5. store 接口定义在 domain，实现放在 infrastructure。

## 3. 领域模型

### 3.1 AppContext

应用级唯一参数源，所有资源共用：

| 字段 | 类型 | JSON | 语义 |
| --- | --- | --- | --- |
| Name | string | name | 应用名，作为资源名前缀，DNS-1123，最大 44 |
| Namespace | string | namespace | 目标命名空间，DNS-1123，最大 63 |
| Image | string | image | 镜像名，不能含空白 |
| Tag | string | tag | 镜像标签 |
| Ports | []Port | ports | 端口列表，含 name/container/service |
| Replicas | int32 | replicas | 副本数，1-100 |
| Env | map | env | 普通环境变量 |
| SecretEnv | map | secret_env | 写入 Secret 的键值 |
| Resources | ResourceRequirements | resources | requests/limits，CPU/内存字符串 |
| Probe | ProbeConfig | probe | enabled/path/readiness_path |
| Host | string | host | Ingress 域名，可空 |
| TLSSecret | string | tls_secret | Ingress TLS Secret 名 |

Port 结构：name（端口名，DNS label 最大 15）、container（容器端口）、service（服务端口）。
ImageRef() 返回 image:tag；image 为空返回空串。

### 3.2 Selection

| 字段 | 类型 | JSON | 说明 |
| --- | --- | --- | --- |
| Type | ConfigType | type | 固定 k8s |
| App | AppContext | app | 应用参数 |
| Resources | []string | resources | K8s 资源 Kind 集合（仅 k8s 家族） |

### 3.3 Plan 与节点

- Node：Kind、Name、Namespace、Level、AutoAdded、Refs（逻辑键 → Ref{Kind,Name,Port}）。
- Edge：From、To、Relation，单位是 Kind（每个 Kind 最多一个节点）。
- Diagnostic：Severity（error/warning/info）、Code、Kind、Message、Hint。
- Plan：Type、App、Nodes、Edges、Diagnostics、Valid、CatalogVersion。

Plan 是渲染的唯一输入。任何渲染所需信息（名字、命名空间、引用、端口、环境变量）都必须已经写入 Plan。

### 3.4 Record 与摘要

- Record：ID、Type、App、Selection、Plan、Files、Bundle、CatalogVersion、CreatedAt。
- RecordSummary：ID、Type、AppName、Namespace、Kinds、CreatedAt，仅用于列表。
- Artifact：Path、Kind、Content（多文件产物）。


## 4. 资源目录（Catalog）

文件：internal/domain/catalog.go，版本常量 CatalogVersion = "2.0.0"。

Spec 字段：Kind、APIVersion、Category、Scope（namespaced/cluster）、Suffix、Level、Needs（Image/Port/Replicas）、Requires（Kind/Relation/Mode/Cardinality）。

### 4.1 全部 26 种资源

| Kind | apiVersion | 分类 | 作用域 | 后缀 | 层级 | Needs | Requires |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Deployment | apps/v1 | workload | ns | deployment | 2 | image,replicas | - |
| StatefulSet | apps/v1 | workload | ns | statefulset | 2 | image,replicas | - |
| DaemonSet | apps/v1 | workload | ns | daemonset | 2 | image | - |
| CronJob | batch/v1 | workload | ns | cronjob | 2 | image | - |
| Job | batch/v1 | workload | ns | job | 2 | image | - |
| ReplicaSet | apps/v1 | workload | ns | replicaset | 2 | image,replicas | - |
| HorizontalPodAutoscaler | autoscaling/v2 | autoscaling | ns | hpa | 3 | - | - |
| Service | v1 | networking | ns | service | 3 | port | - |
| Ingress | networking.k8s.io/v1 | networking | ns | ingress | 4 | port | Service auto_add one |
| IngressClass | networking.k8s.io/v1 | networking | cluster | ingressclass | 0 | - | - |
| EndpointSlice | discovery.k8s.io/v1 | networking | ns | endpointslice | 4 | port | Service auto_add one |
| NetworkPolicy | networking.k8s.io/v1 | networking | ns | netpol | 3 | - | - |
| ConfigMap | v1 | config | ns | configmap | 0 | - | - |
| Secret | v1 | config | ns | secret | 0 | - | - |
| PersistentVolumeClaim | v1 | storage | ns | pvc | 1 | - | - |
| StorageClass | storage.k8s.io/v1 | storage | cluster | storageclass | 0 | - | - |
| ServiceAccount | v1 | rbac | ns | sa | 0 | - | - |
| Role | rbac.authorization.k8s.io/v1 | rbac | ns | role | 1 | - | - |
| RoleBinding | rbac.authorization.k8s.io/v1 | rbac | ns | rolebinding | 1 | - | ServiceAccount auto_add one |
| ClusterRole | rbac.authorization.k8s.io/v1 | rbac | cluster | clusterrole | 0 | - | - |
| ClusterRoleBinding | rbac.authorization.k8s.io/v1 | rbac | cluster | clusterrolebinding | 1 | - | ServiceAccount / ClusterRole auto_add |
| ResourceQuota | v1 | policy | ns | quota | 0 | - | - |
| LimitRange | v1 | policy | ns | limits | 0 | - | - |
| PodDisruptionBudget | policy/v1 | policy | ns | pdb | 3 | - | - |
| PriorityClass | scheduling.k8s.io/v1 | scheduling | cluster | priorityclass | 0 | - | - |
| Namespace | v1 | cluster | cluster | namespace | 0 | - | - |

### 4.2 目录维护规则

- Suffix 在目录内唯一，决定资源名 appName-suffix。
- Level 决定渲染顺序，必须是 0-4。
- Needs 只能声明该资源真正需要的应用参数：Service/Ingress 不声明 image；纯配置资源不声明 port。
- Requires 只用于存在性依赖；复杂规则（HPA 目标唯一、PDB/NetworkPolicy 需要工作负载、RoleBinding 的 Role 选择）放在 resolve.go 的 workloadRules。
- 新增 Kind 必须同时提供 builder 与测试，否则 build/test 必须失败。
- 前端不硬编码 Kind 列表，全部通过 GET /api/v1/catalog 获取。

## 5. 解析器（Resolve）

入口：domain.Resolve(sel Selection) (Plan, error)。纯函数，无 IO、无全局状态（catalog 只读）。

### 5.1 执行顺序（不可调换）

1. 校验 Type 合法。
2. normalizeApp：trim、端口默认命名（http、http-2…）、container/service 互补。
3. 资源列表去重；未知 Kind 报错。
4. expand：按 Catalog.Requires 做依赖闭包，最多 12 轮。
5. workloadRules：Service/StatefulSet/HPA/PDB/NetworkPolicy/RoleBinding 联动。
6. applyDefaults：按最终 Kind 集合补 image/tag、port、replicas。
7. validateK8s：名称、命名空间、端口、环境变量、域名、TLS。
8. allocateNames：Kind → 名字。
9. resolveNamespace：确定目标命名空间与 Namespace 资源名。
10. buildNodes：绑定 Refs。
11. 按 (Level, Kind, Name) 排序，buildEdges。
12. semanticDiags：语义告警。
13. Valid = 无 error；有 error 时按 conflict 标记返回 400 或 422。

### 5.2 默认值规则

| 条件 | 默认值 |
| --- | --- |
| 任一选中资源 Needs.Image 且 image 为空 | nginx |
| 同上且 tag 为空 | latest |
| 任一选中资源 Needs.Port 且 ports 为空 | [{name: http, container: 80, service: 80}] |
| 任一选中资源 Needs.Replicas 且 replicas < 1 | 1 |
| 端口 container 为 0 | 取 service |
| 端口 service 为 0 | 取 container |
| 端口名为空 | 第一个 http，其余 http-N |

纯工作负载不补端口：Deployment-only 不生成 ports 段，避免 containerPort: 0。

### 5.3 依赖规则（workloadRules）

| 条件 | 行为 | 诊断 |
| --- | --- | --- |
| StatefulSet 且无 Service | 自动补 Service | info auto_added |
| Service 且无任何工作负载 | 自动补 Deployment | info auto_added |
| HPA 且无可伸缩目标 | 有 DaemonSet/Job/CronJob 时 error；否则补 Deployment | 422 / info |
| HPA 且有多个可伸缩目标 | error | 422 hpa_ambiguous_target |
| PDB 且无工作负载 | 自动补 Deployment | info |
| NetworkPolicy 且无工作负载 | 自动补 Deployment | info |
| RoleBinding 且无 Role/ClusterRole | 自动补 Role | info |
| RoleBinding 同时有 Role 与 ClusterRole | 绑定 Role | warning role_ref_ambiguous |
| ClusterRoleBinding 缺 SA/ClusterRole | 由 Catalog.Requires 自动补 | info |
| secret_env 非空且未选 Secret | 自动补 Secret，并以 envFrom 注入 | info auto_added |

### 5.4 命名与命名空间

- 名字：appName + "-" + suffix；Namespace 特殊：若 app.namespace 非空，Namespace 资源名即 app.namespace。
- 选择 Namespace 且 app.namespace 为空：使用 appName-namespace，并写回 App.Namespace，所有 namespaced 资源写入 metadata.namespace。
- 选择 Namespace 且 app.namespace 非空：Namespace 资源名 = app.namespace。
- 未选择 Namespace 但 app.namespace 非空：所有 namespaced 资源写入该 namespace，并提示需要已存在。
- cluster-scoped 资源永不写 metadata.namespace。

### 5.5 引用绑定（Refs）

| 节点 | refs 键 |
| --- | --- |
| 工作负载 | configmap、secret、pvc、sa；StatefulSet 额外 service |
| Service | workload（primaryWorkload） |
| Ingress | service、ingressclass |
| HPA | workload（唯一可伸缩目标） |
| PDB / NetworkPolicy | workload（primaryWorkload） |
| EndpointSlice | service |
| PVC | storageclass |
| RoleBinding | sa、role（无 Role 时 clusterrole） |
| ClusterRoleBinding | sa、clusterrole |

primaryWorkload 顺序：Deployment, StatefulSet, DaemonSet, ReplicaSet, Job, CronJob。
可伸缩目标顺序：Deployment, StatefulSet, ReplicaSet。

### 5.6 边关系

| From | refs 键 | relation |
| --- | --- | --- |
| Service | workload | selects |
| HPA | workload | scales |
| PDB | workload | protects |
| NetworkPolicy | workload | targets |
| Ingress | service | routes-to |
| Ingress | ingressclass | uses-class |
| 工作负载 | configmap/secret/pvc | mounts |
| 工作负载 | sa | uses-sa |
| EndpointSlice | service | belongs-to |
| PVC | storageclass | uses-class |
| RoleBinding/ClusterRoleBinding | role/clusterrole | binds |
| RoleBinding/ClusterRoleBinding | sa | grants-to |

边只从 Ref 生成，因此“图”与“YAML 引用”天然一致。

### 5.7 校验规则

| 项 | 规则 | 错误码 |
| --- | --- | --- |
| app.name | DNS-1123，1-44 | invalid_app_name |
| namespace | DNS-1123，1-63 | invalid_namespace |
| image/tag | 在需要镜像时非空，不能含空白 | image_required/tag_required/invalid_image |
| port | container/service 均在 1-65535 | invalid_port |
| port.name | DNS label 1-15，且唯一 | invalid_port_name/duplicate_port_name |
| env/secret_env key | 字母数字及 - _ . | invalid_env_key |
| host | 不能含空白 | invalid_host |
| tls_secret | DNS-1123 | invalid_tls_secret |
| resources | k8s 家族至少一个（trim 后） | no_resources |
| resources.requests/limits | 必须能解析为 Kubernetes quantity（apimachinery resource） | invalid_resource_quantity |
| Kind | 必须在目录中 | unknown_kind |

### 5.8 语义诊断

| Code | 级别 | 触发 |
| --- | --- | --- |
| netpol_default_deny | warning | 选择 NetworkPolicy |
| placeholder_provisioner | warning | 选择 StorageClass |
| placeholder_controller | warning | 选择 IngressClass |
| secret_empty | warning | 选择 Secret 但 secret_env 为空 |
| configmap_empty | warning | 选择 ConfigMap 但 env 为空 |
| hpa_no_requests | warning | HPA 存在但未设置 CPU requests |
| pdb_single_replica | warning | PDB 且 replicas <= 1 |
| service_job | warning | Service 与 Job/CronJob 同时存在 |
| ingress_any_host | info | Ingress 未填 host |
| tls_without_host | warning | 配置 TLS 但未填 host |
| namespace_external | info | 指定 namespace 但未创建 Namespace 资源 |
| namespace_created | info | 自动创建 Namespace |
| host_ignored | warning | 填写 host 但未选择 Ingress |
| tls_ignored | warning | 填写 tls_secret 但未选择 Ingress |
| probe_ignored_no_workload | warning | 开启探针但未选择工作负载 |
| probe_ignored_batch | warning | 仅选择 Job/CronJob 时开启探针 |
| probe_ignored_no_ports | warning | 工作负载无端口时开启探针 |
| cronjob_default_schedule | warning | 选择 CronJob（固定 0 * * * *） |
| pvc_default_size | warning | 选择 PVC（固定 10Gi） |
| endpointslice_empty | warning | 选择 EndpointSlice（endpoints 为空） |
| quota_placeholder | warning | 选择 ResourceQuota（固定配额） |
| limitrange_placeholder | warning | 选择 LimitRange（固定默认值） |
| priorityclass_placeholder | warning | 选择 PriorityClass（固定 value 1000） |
| multiple_workloads | warning | 同时选择多个工作负载 |
| statefulset_service_not_headless | warning | 多工作负载下 StatefulSet 的 Service 非 Headless |
| namespace_default | info | 未显式指定命名空间 |

## 6. 构建器（builders）

接口：type Builder func(app domain.AppContext, n domain.Node) (any, error)。
注册：init() 中 register("Kind", builder)；Build 按 n.Kind 分派，未注册返回错误。

### 6.1 共享 podSpec

podSpec 统一生成容器模板，规则如下：

- 容器名 = app.name，镜像 = image:tag。
- ports 仅在 withPorts=true 且 app.ports 非空时生成。
- resources 仅在 requests/limits 非空时生成；quantity 已在 domain 校验，非法值在 /plan 阶段返回 400。
- ConfigMap 存在：envFrom configMapRef；否则若 env 非空，内联 env（按 key 排序）。
- Secret 存在且 secret_env 非空：envFrom secretRef。
- Secret 存在且 secret_env 为空：只读卷 secret-volume 挂载 /etc/secret。
- PVC 存在：卷 data 挂载 /data。
- SA 存在：serviceAccountName。
- probe.enabled、存在端口且 withPorts=true：liveness/readiness HTTPGet，初始延迟 5s、周期 10s、超时 2s、失败阈值 3。Job/CronJob 不输出端口，因此也不生成探针。
- 探针端口使用第一个端口名。

### 6.2 各资源实现约定

| Kind | 关键实现 |
| --- | --- |
| Deployment | replicas 默认 1；selector app=name；RollingUpdate 默认策略 |
| StatefulSet | serviceName 取 refs.service，否则 app-service；replicas 默认 1 |
| DaemonSet | 无 replicas；包含端口 |
| ReplicaSet | replicas 默认 1 |
| Job | backoffLimit=3；restartPolicy=Never；不输出端口与探针 |
| CronJob | schedule 固定 0 * * * *；restartPolicy=Never；不输出端口与探针 |
| Service | ports 来自 app.ports；targetPort=container；工作负载为 StatefulSet 时 clusterIP=None |
| Ingress | path / + Prefix；backend 指向 refs.service 与第一个 service 端口；host 可空；host+tls_secret 生成 TLS；有 IngressClass 时设置 ingressClassName |
| IngressClass | controller 固定 k8s.io/ingress-nginx |
| HPA | min=replicas（至少 1），max=max(5, 2*min)，CPU 70% |
| PDB | replicas>1 用 minAvailable=1，否则 maxUnavailable=1 |
| NetworkPolicy | podSelector app=name；ingress 同命名空间；egress 同命名空间 + kube-system DNS 53/UDP+TCP |
| EndpointSlice | addressType IPv4；endpoints 为空；ports 来自 app.ports；label kubernetes.io/service-name |
| ConfigMap | data = app.env |
| Secret | type Opaque；stringData = app.secret_env |
| PVC | ReadWriteOnce，10Gi；有 StorageClass 时 storageClassName |
| StorageClass | provisioner kubernetes.io/no-provisioner；Retain；WaitForFirstConsumer |
| ServiceAccount | 仅 metadata |
| Role/ClusterRole | pods 的 get/list/watch |
| RoleBinding | roleRef 取 Role 或 ClusterRole；subject ServiceAccount |
| ClusterRoleBinding | subject 的 namespace 为空时用 default |
| ResourceQuota | requests 4CPU/8Gi，limits 8CPU/16Gi |
| LimitRange | 默认 500m/512Mi，默认请求 100m/128Mi |
| PriorityClass | value 1000，globalDefault=false |
| Namespace | 仅 metadata |

新增 builder 必须：

1. 返回 k8s.io/api 官方类型，并设置 TypeMeta（apiVersion/kind）。
2. 使用 objectMeta(app, n) 统一 metadata；cluster-scoped 用只带 Name/Labels 的 ObjectMeta。
3. 所有名字来自 n.Name / n.Refs，禁止字符串拼接。
4. 在 init() 注册；在 render 全量测试中被覆盖。

## 7. 渲染层（render）

入口：render.Render(plan domain.Plan) ([]domain.Artifact, string, error)。

### 7.1 K8s

- 逐节点 Build → sigs.k8s.io/yaml.Marshal。
- 文件名：strings.ToLower(Kind) + ".yaml"（如 deployment.yaml、networkpolicy.yaml）。
- bundle：每份文件前加一行 ---，顺序与节点一致。
- 产物校验：必须是可解析 map，kind 与 metadata.name 非空，kind/name 不得重复。校验失败视为内部错误（不会落库）。

## 8. 用例层

文件：internal/usecase/service.go。唯一业务入口。

| 方法 | 行为 |
| --- | --- |
| Plan(sel) | domain.Resolve，不落库 |
| Render(sel) | Resolve → render.Render → 组装 Record → store.Save → 回填 ID |
| GetRecord(id) | store.Find |
| ListRecords(offset, limit) | store.List + store.Count，返回摘要与总数 |
| Health() | store.Ping |

约定：Render 的渲染错误映射为 400（NewBadRequest），存储错误原样上抛（500）。

## 9. 存储层

文件：internal/infrastructure/storage/sqlite.go。

### 9.1 DSN

- 默认 db/configgen.db；目录不存在时 MkdirAll。
- 非 :memory: 时加 file: 前缀并转斜杠。
- 追加：_pragma=busy_timeout(5000)、_pragma=journal_mode(WAL)、_pragma=foreign_keys(1)。
- db.SetMaxOpenConns(1)：SQLite 单写者，避免 database is locked。

### 9.2 表结构

表名 render_records：

| 列 | 类型 | 说明 |
| --- | --- | --- |
| id | INTEGER PK AUTOINCREMENT | 记录 ID |
| config_type | TEXT | 固定 k8s |
| app_name | TEXT | 应用名 |
| namespace | TEXT | 命名空间，默认空串 |
| selection_json | TEXT | Selection JSON |
| plan_json | TEXT | Plan JSON |
| files_json | TEXT | []Artifact JSON |
| bundle | TEXT | 合并后的 YAML |
| catalog_version | TEXT | 生成时目录版本 |
| created_at | TEXT | RFC3339Nano |

索引：idx_render_records_created(created_at DESC)。

旧表 config_records 不迁移，历史记录从新表开始。

### 9.3 序列化与时间

- 时间统一 UTC + time.RFC3339Nano 文本，读取时严格解析，失败返回错误。
- List 只查询摘要字段，并反序列化 plan_json 提取 Kinds；不返回完整 files。
- Save 使用单条 INSERT；Find/List/Count 失败均向上抛出。

## 10. 表现层

文件：internal/presentation/server.go。

### 10.1 路由

| 方法 | 路径 | 行为 | 成功码 |
| --- | --- | --- | --- |
| GET | /health | DB ping | 200 / 503 |
| GET | /api/v1/catalog | CatalogView | 200 |
| POST | /api/v1/plan | UseCase.Plan | 200 |
| POST | /api/v1/render | UseCase.Render | 201 |
| GET | /api/v1/records | 摘要 + total | 200 |
| GET | /api/v1/records/:id | 完整记录 | 200 / 404 |
| GET | / | SPA index | 200 |
| NoRoute | 非 /api/ 路径回退 index；/api/ 返回 JSON 404 | - | - |

### 10.2 请求解析

- c.Request.Body = http.MaxBytesReader(..., 1<<20)。
- json.Decoder + DisallowUnknownFields。
- 解析失败 400，响应包含 error 与 details。
- 不在 handler 内做业务校验。

### 10.3 错误映射

- domain.AppError Kind=bad_request → 400。
- domain.AppError Kind=unprocessable → 422。
- 其他错误 → 500，日志记录 error（不把内部细节返回给客户端）。
- 400/422 响应包含 diagnostics；若已有 Plan 节点，附带 plan。

### 10.4 日志与 SPA

- 每个请求输出 slog：method、path、status、duration_ms。
- 静态资源从 go:embed 的 web/dist 提供；未知客户端路由回退 index.html。
- /api/ 下未匹配路径返回 {"error":"not found"}，不返回 HTML。

## 11. 进程与配置

main.go：

- flag -addr；LoadDotEnv(".env")；GetListenAddr 优先级 -addr > ADDR > PORT > :8080。
- slog JSON handler，级别 Info。
- SQLite 初始化失败直接退出。
- http.Server：ReadHeaderTimeout 5s、ReadTimeout 15s、WriteTimeout 60s、IdleTimeout 60s。
- SIGINT/SIGTERM 触发 10s 超时优雅停机。
- 环境变量：PORT、ADDR、CONFIGGEN_DB；SENTRY_DSN 已移除。

## 12. 测试规范

| 包 | 测试文件 | 覆盖重点 |
| --- | --- | --- |
| domain | resolve_test.go | 闭包、命名、排序、Namespace、HPA 冲突、默认端口 |
| render | render_test.go | 全资源 YAML 合法、引用、envFrom、Volume、HPA |
| presentation | server_test.go | catalog/health、plan/render、400/422、未知字段、列表/详情 |
| storage | sqlite_test.go | save/find/list/count、JSON 往返、时间、Kinds 投影 |

当前覆盖率：domain 73.0%、render 72.0%、storage 72.5%、presentation 68.3%、config 88.9%。

要求：

1. 新增规则必须补 domain 测试；新增资源必须进入 render 全量测试。
2. YAML 断言必须反序列化，不允许只用 strings.Contains 判断结构。
3. 错误路径必须覆盖 400/422 区分。
4. 涉及名称/端口/引用的改动必须覆盖“输出可直接 apply”的不变量。

## 13. 构建、CI 与发布

本地：

~~~
gofmt -w cmd config internal
go vet ./...
go test ./...
go build -o configgen.exe ./cmd/configgen
~~~

CI（.github/workflows/go-ci.yml）：

1. gofmt 检查。
2. go vet。
3. go test -v ./...。
4. CGO_ENABLED=0 go build 冒烟。
5. gosec 报告（当前不阻断）。
6. tag 发布：linux/darwin 多平台二进制 + ghcr.io 多架构镜像（纯 Go，可交叉编译）。

Docker：golang:1.25-alpine 构建，CGO_ENABLED=0，alpine 运行，GIN_MODE=release。

## 14. 扩展指南

### 14.1 新增一种资源

1. catalog.go 加 Spec（唯一 Suffix、Level 0-4、正确的 Needs/Scope）。
2. builders/ 新文件实现 Builder，init() register，设置 TypeMeta，使用 objectMeta/refs。
3. 需要联动时在 resolve.go workloadRules 加规则，并补语义诊断。
4. render_test 的 all-kinds 选择加入该 Kind；补关键字段断言。
5. 更新 docs/BACKEND.md 的资源表与 builder 约定。

### 14.2 新增依赖规则

1. 在 catalog Requires（简单存在性）或 workloadRules（唯一性/联动）实现。
2. 在 Ref 中记录关系，relationFor 增加关系名（若为新键）。
3. 补 domain 测试：自动补充、冲突、无副作用。
4. 更新文档的规则表。

### 14.3 修改请求模型

1. types.go 加字段，AppContext 保持“应用级唯一参数源”。
2. 若字段影响资源，更新 builder 与 podSpec。
3. 前端 types.ts 同步；前端表单与 buildSelection 同步。
4. 数据库历史快照存 JSON，无需迁移；但 Record 读取需兼容旧 JSON（字段缺失按零值）。
5. 目录版本 CatalogVersion 在语义变化时递增。

## 15. 不变量清单（代码评审必查）

- Plan 中每个 Ref 的 Kind 必须存在于 Nodes。
- 节点名唯一，Shape 为 appName-suffix；DNS-1123，长度 <= 63。
- 所有 namespaced 资源共享同一命名空间；cluster-scoped 不带 namespace。
- Service selector 与工作负载 pod labels 一致。
- Ingress backend 的端口存在于 Service。
- HPA 的 target 是唯一可伸缩工作负载。
- ConfigMap/Secret/PVC 的引用名与生成的资源名一致。
- 端口的 container/service 均在 1-65535；端口名唯一。
- 渲染两次字节一致；bundle 与 files 内容一致。
- 空资源列表返回 400，不返回 500。

## 16. 已知限制

- StorageClass 使用 no-provisioner、IngressClass 使用 ingress-nginx 标识、CronJob 固定每小时、Job backoff 3、PVC 10Gi、ResourceQuota/LimitRange/PriorityClass 固定值：生成的是骨架，均有对应 warning，需要按集群调整。
- Secret 内容为空时只生成空 Secret 并告警。
- 探针默认关闭；CPU requests 未设置时 HPA 告警但不阻断。
- 未实现服务端 dry-run、kubeconform、Helm values 导出、鉴权、多租户。
- 旧表 config_records 不迁移。

## 17. 排障

| 现象 | 排查 |
| --- | --- |
| 启动报 failed to create store | 检查 CONFIGGEN_DB 目录存在且可写；纯 Go SQLite 需要写 WAL/SHM |
| 422 且诊断为 hpa_ambiguous_target | 只保留一个 Deployment/StatefulSet/ReplicaSet，或移除 HPA |
| 生成的文件无法 apply | 检查诊断中的 placeholder 警告；确认 app_name 为小写 DNS-1123 |
| 历史列表为空 | 新表 render_records 从本次重构开始记录，旧数据不迁移 |
| 前端仍是旧界面 | web/dist 未重新构建；执行 cd web && npm run build 后重新编译二进制 |
