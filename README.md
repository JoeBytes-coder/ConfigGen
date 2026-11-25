# ConfigGen

ConfigGen 是一个用 Go 实现的云原生配置生成平台：在 Web 页面选择需要的 Kubernetes 资源，后端自动解析资源之间的依赖、分配资源名、绑定跨资源引用，最终输出可直接 kubectl apply 的多文件 YAML。

## 核心能力

- **26 种 K8s 资源**：工作负载、自动伸缩、网络、配置、存储、RBAC、策略、调度、集群九类，目录驱动。
- **依赖自动处理**：选择 Ingress 会自动补齐 Service 与工作负载；StatefulSet 自动补 Headless Service；HPA 校验目标唯一性；RoleBinding 自动补 ServiceAccount/Role；NetworkPolicy 自动放行 DNS。
- **两阶段生成**：先 Plan（依赖闭包、命名、引用绑定、拓扑排序、诊断），再 Render（类型化 K8s 对象 + YAML 序列化），Plan 可单独预览。
- **输出即验收**：每份产物都会反序列化校验，要求 YAML 合法、kind/name 齐全、无重复资源；生成失败不会被标记为可执行。
- **多文件产物**：按资源分文件返回，同时提供可直接 apply 的 bundle。
- **可解释**：自动补充的资源标记 auto_added，诊断分为 error / warning / info，并给出修复提示。
- **单二进制、无 cgo**：SQLite 使用纯 Go 驱动 modernc.org/sqlite，前端构建产物通过 go:embed 嵌入，多平台交叉编译可用。
- **历史快照**：保存 Selection、Plan、文件与资源目录版本，历史详情可回看当时的完整结果。

## 项目结构

~~~
configgen/
├── cmd/configgen/                 # 入口：配置加载、DI 组装、HTTP Server、优雅停机
├── config/                        # .env 加载、监听地址解析
├── internal/
│   ├── domain/                    # 领域层（无外部框架依赖）
│   │   ├── types.go               # AppContext / Selection / Plan / Record
│   │   ├── catalog.go             # 26 种资源目录：作用域/命名后缀/层级/依赖
│   │   ├── resolve.go             # 选择 -> 依赖闭包 -> 命名/绑定/排序/诊断
│   │   ├── validate.go            # DNS-1123、端口、环境变量等校验
│   │   ├── errors.go              # bad_request / unprocessable 错误分类
│   │   └── store.go               # Store 接口
│   ├── usecase/                   # Plan / Render / 查询 用例
│   ├── infrastructure/
│   │   ├── builders/              # 每种 K8s 资源一个类型化 builder
│   │   ├── render/                # K8s 渲染与产物校验
│   │   └── storage/               # SQLite 快照存储
│   └── presentation/              # Gin 路由、严格 JSON 解析、错误映射、SPA
├── web/                           # Vue 3 + Vite + Pinia + Naive UI（dist 由 go:embed 嵌入）
├── docs/                          # DESIGN / BACKEND / FRONTEND 三份设计与维护规范
└── go.mod
~~~

依赖方向：presentation → usecase → domain ← infrastructure。

## 文档

| 文档 | 内容 |
| --- | --- |
| [docs/DESIGN.md](docs/DESIGN.md) | 架构总览、设计决策（ADR）、维护流程、全局验收标准 |
| [docs/BACKEND.md](docs/BACKEND.md) | 后端分层、领域模型、资源目录、依赖解析、builder、渲染、API、存储、测试与扩展 |
| [docs/FRONTEND.md](docs/FRONTEND.md) | 前端技术栈、数据契约、Pinia store、组件、交互时序、样式与构建规范 |

修改代码时必须同步更新对应文档；详细规则见 AGENTS.md 的「文档规范」。

## 快速开始

要求 Go 1.25+，不再需要 C 编译器（SQLite 为纯 Go 驱动）。

~~~bash
go build -o configgen.exe ./cmd/configgen
./configgen.exe -addr=:8080
~~~

验证：

~~~bash
curl http://localhost:8080/health
# {"status":"ok"}
~~~

打开 http://localhost:8080 使用 Web 界面。

### 前端开发

~~~bash
cd web
npm install
npm run dev      # http://localhost:5173，/api 代理到 :8080
npm run build    # 输出到 web/dist，供 go:embed 嵌入
npm run test:fsm # 前端状态机单测（Node 22.6+）
~~~

## API

所有请求/响应均为 JSON；POST 请求体上限 1 MiB，未知字段会被拒绝。

### GET /api/v1/catalog

返回资源目录与目录版本，前端据此渲染选择器、分组与依赖提示。

~~~json
{
  "version": "2.0.0",
  "kinds": [
    {
      "kind": "Deployment",
      "api_version": "apps/v1",
      "category": "workload",
      "scope": "namespaced",
      "suffix": "deployment",
      "level": 2,
      "needs_image": true,
      "needs_port": false,
      "needs_replicas": true
    }
  ]
}
~~~

### POST /api/v1/plan

只做依赖解析，不落库。返回 nodes / edges / diagnostics / valid，前端用于实时预览。

~~~json
{
  "type": "k8s",
  "app": {
    "name": "demo",
    "namespace": "prod",
    "image": "nginx",
    "tag": "1.25",
    "ports": [{ "name": "http", "container": 8080, "service": 80 }],
    "replicas": 2,
    "env": { "FOO": "bar" },
    "resources": {
      "requests": { "cpu": "100m", "memory": "128Mi" },
      "limits": { "cpu": "500m", "memory": "512Mi" }
    },
    "probe": { "enabled": true, "path": "/healthz" }
  },
  "resources": ["Ingress"]
}
~~~

响应中的关键字段：

~~~json
{
  "nodes": [
    { "kind": "Deployment", "name": "demo-deployment", "level": 2, "auto_added": true },
    { "kind": "Service", "name": "demo-service", "level": 3, "auto_added": true },
    { "kind": "Ingress", "name": "demo-ingress", "level": 4, "auto_added": false }
  ],
  "edges": [
    { "from": "Service", "to": "Deployment", "relation": "selects" },
    { "from": "Ingress", "to": "Service", "relation": "routes-to" }
  ],
  "diagnostics": [
    { "severity": "info", "code": "auto_added", "kind": "Service", "message": "已自动补充 Service" }
  ],
  "valid": true,
  "catalog_version": "2.0.0"
}
~~~

### POST /api/v1/render

请求体与 plan 相同；解析后渲染并持久化，返回记录。

~~~json
{
  "id": 1,
  "type": "k8s",
  "app": { "...": "..." },
  "selection": { "...": "..." },
  "plan": { "...": "..." },
  "files": [
    { "path": "deployment.yaml", "kind": "Deployment", "content": "apiVersion: apps/v1" }
  ],
  "bundle": "---",
  "catalog_version": "2.0.0",
  "created_at": "2026-11-24T07:19:45Z"
}
~~~

### 历史记录

- GET /api/v1/records?offset=0&limit=20：轻量摘要（不返回完整配置），附带 total。
- GET /api/v1/records/:id：完整快照。

### 错误语义

- 400：请求/参数非法（名称不符合 DNS-1123、端口越界、未知资源、资源配额 quantity 非法等），响应含 diagnostics。
- 422：依赖无法满足（例如 HPA 同时指向多个可伸缩工作负载）。
- 500：内部错误（数据库、渲染异常）。

### GET /health

数据库可用时返回 200，否则 503。

## 依赖处理规则

| 触发 | 自动行为 |
| --- | --- |
| 选择 Ingress | 自动补 Service，再补 Deployment |
| 选择 Service 且无工作负载 | 自动补 Deployment |
| 选择 StatefulSet 且无 Service | 自动补 Headless Service（clusterIP: None） |
| 选择 HPA | 必须存在唯一可伸缩目标（Deployment/StatefulSet/ReplicaSet），否则 422 |
| 选择 PDB / NetworkPolicy | 无工作负载时自动补 Deployment |
| 选择 RoleBinding | 自动补 ServiceAccount；无 Role/ClusterRole 时补最小权限 Role |
| 选择 ClusterRoleBinding | 自动补 ServiceAccount 与 ClusterRole |
| 选择 ConfigMap | 工作负载通过 envFrom 引用它 |
| 填写 secret_env | 即使未勾选 Secret 也会自动补 Secret，并以 envFrom 注入 |
| 选择 Secret | 有 secret_env 时 envFrom，否则以只读卷挂载 |
| 选择 PVC | 工作负载挂载到 /data；有 StorageClass 时填充 storageClassName |
| 选择 Namespace | 所有 namespaced 资源写入 metadata.namespace |
| 选择 IngressClass | Ingress 填充 ingressClassName |

渲染顺序按层级（Namespace/ConfigMap/Secret/SA 等 → PVC/RBAC → 工作负载 → Service/HPA/PDB/NetworkPolicy → Ingress/EndpointSlice），同层按 kind/name 稳定排序，输出可复现。

## 配置项

| 配置 | 说明 |
| --- | --- |
| -addr | 监听地址，优先级：-addr > ADDR > PORT > :8080 |
| CONFIGGEN_DB | SQLite 文件路径，默认 db/configgen.db |
| .env | 启动时加载根目录 .env（已被 gitignore） |

## 开发与测试

~~~bash
gofmt -l cmd config internal
go vet ./...
go test ./...
go test -cover ./...
~~~

当前覆盖：domain ~73%、render ~72%、storage ~72.5%、presentation ~68%。

CI（.github/workflows/go-ci.yml）执行 gofmt 检查、go vet、go test、构建冒烟与 gosec 扫描；打 v* 标签时发布多平台二进制并构建 ghcr.io 镜像。

## Docker

~~~bash
docker build -f dockerfile -t configgen .
docker run --rm -p 8080:8080 -v configgen-db:/data configgen
~~~

镜像使用 CGO_ENABLED=0 构建，纯 Go SQLite 可正常运行。

## 已知限制

- 生成的是生产骨架：StorageClass 使用 no-provisioner、IngressClass 使用 ingress-nginx 标识、CronJob 固定每小时、PVC 固定 10Gi、Secret 内容需自行填充；固定值资源、被忽略的 host/探针等都会以 warning 提示。探针默认关闭且不单独告警。
- 不包含认证/鉴权，默认面向内网或本机使用。
- 未实现服务端 dry-run、Helm Chart/values 导出、ZIP 打包、i18n。
- 数据库表已从 config_records 迁移为 render_records，旧记录不会自动迁移。

## License

MIT
