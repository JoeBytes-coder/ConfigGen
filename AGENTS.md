# ConfigGen — AGENTS.md

## 项目定位

Go 编写的 Kubernetes 配置生成平台：Web 选择资源 → domain.Resolve 自动处理依赖/命名/引用/排序 → infrastructure/builders + render 输出可直接 apply 的多文件 YAML。当前只支持 Kubernetes 资源，不生成 Compose/Dockerfile/Kustomize。

## 文档规范（必读）

- docs/DESIGN.md：架构、ADR、维护流程、全局验收标准。
- docs/BACKEND.md：后端分层、领域模型、资源目录、解析算法、builder、API、存储、测试与扩展指南。
- docs/FRONTEND.md：前端技术栈、数据契约、store、组件、交互时序、样式、构建与开发规范。
- 任何影响架构、API 契约、资源目录、默认值、构建方式或前端交互的改动，必须同步更新对应文档；只改代码不改文档视为未完成。

## 架构与依赖方向

~~~
cmd/configgen/main.go        DI 组装 + http.Server（超时/优雅停机）+ slog
config/                      .env 加载、监听地址
internal/domain/             实体、资源目录、依赖解析、校验、Store 接口（纯领域，无框架）
internal/usecase/            唯一业务入口：Plan / Render / GetRecord / ListRecords / Health
internal/infrastructure/
  builders/                  26 种资源的类型化构造（k8s.io/api）
  render/                    渲染与产物校验
  storage/                   SQLite（modernc.org/sqlite，纯 Go）
internal/presentation/       Gin 路由、严格 JSON、错误映射、SPA 静态服务
web/                         Vue 3 + Vite + Pinia + Naive UI，dist 由 go:embed 嵌入
~~~

依赖方向：presentation → usecase → domain ← infrastructure。domain 不导入 Gin/SQL/模板。

## 关键设计（不要破坏）

1. 先 Plan 后 Render：domain.Resolve(Selection) 返回 Plan（nodes/edges/diagnostics）；render.Render(plan) 只读 Plan。禁止在 builder 里做依赖推导。
2. 目录是单一事实源：资源名 = appName + "-" + catalog suffix；作用域、层级、Needs、Requires 全在 internal/domain/catalog.go。新增资源不改前端。
3. 引用只来自 PlanNode.Refs：builder 不得自己拼 "app-xxx" 名字。跨资源引用由 resolver 绑定，保证图与 YAML 一致。
4. 输出即验收：render 后必须通过 validateK8sArtifacts（YAML 可解析、kind/name 非空、kind/name 不重复）。
5. 确定性：节点按 (level, kind, name) 排序；环境变量/端口顺序稳定；同一 Selection 两次渲染字节一致。
6. 自动补全可见：自动添加的节点 auto_added=true 且有 info 诊断，绝不静默添加。
7. 业务校验在 domain：名称/端口/环境变量/DNS-1123 校验在 domain；presentation 只解码与映射状态码。
8. 前端状态机：generator store 的状态只能由 generatorMachine.transition 迁移；组件只读 status/canGenerate，禁止自行组合 planning/planError/record 等布尔条件。

## 命令

| 用途 | 命令 |
| --- | --- |
| 构建 | go build -o configgen.exe ./cmd/configgen |
| 测试 | go test ./... |
| 覆盖率 | go test -cover ./... |
| 静态检查 | go vet ./... |
| 格式 | gofmt -w cmd config internal |
| 前端开发 | cd web && npm run dev |
| 前端构建 | cd web && npm run build（产物 web/dist 需提交） |
| 前端状态机测试 | cd web && npm run test:fsm（Node 22.6+） |

本地没有 cgo 依赖，不需要 gcc。

## 添加一种 K8s 资源

1. 在 internal/domain/catalog.go 增加 Spec（Kind/APIVersion/Category/Scope/Suffix/Level/Needs/Requires）。
2. 在 internal/infrastructure/builders/ 增加 builder 并在 init() 中 register。
3. 如需要复杂联动（目标唯一性、自动补全），在 internal/domain/resolve.go 的 workloadRules 中加规则。
4. 在 internal/infrastructure/render/render_test.go 的 all-kinds 选择中加入该资源，确保 YAML 合法。
5. 前端无需改动（目录由 /api/v1/catalog 驱动）。

不要新增 text/template 模板；builder 直接构造 k8s.io/api 对象。

## API 约定

- GET /api/v1/catalog
- POST /api/v1/plan（200；非法 400；依赖冲突 422；响应含 diagnostics/plan）
- POST /api/v1/render（201 Record）
- GET /api/v1/records?offset&limit（摘要 + total）
- GET /api/v1/records/:id
- GET /health（DB ping）

请求体严格解码（DisallowUnknownFields，1 MiB 上限）。错误分类用 domain.AppError（bad_request→400，unprocessable→422）。

## 测试

- internal/domain/resolve_test.go：依赖闭包、命名、排序、命名空间、HPA 冲突。
- internal/infrastructure/render/render_test.go：全资源 YAML 合法性、引用、探针、HPA。
- internal/presentation/server_test.go：状态码、诊断、未知字段、列表。
- internal/infrastructure/storage/sqlite_test.go：save/find/list/count。
- web/src/stores/generatorMachine.test.ts：前端状态迁移（happy path、失败、编辑失效、清空资源）。

新增行为必须补测试；禁止只做 strings.Contains 的 YAML 断言，必须反序列化。

## 已知坑

- Service headless 仅当唯一工作负载是 StatefulSet。
- Secret 的密钥来自 app.secret_env；为空时只挂载空 Secret 并给 warning。
- 端口：app.ports 为空时，只有 Service/Ingress 会补默认 80；纯工作负载不生成 ports 段。
- Job/CronJob 固定 restartPolicy=Never，且不输出端口与探针（否则命名端口探针会失败）。
- secret_env 非空时会自动补 Secret（即使未勾选），避免密钥被静默丢弃。
- resources.requests/limits 的 quantity 在 domain 校验，非法值返回 400（invalid_resource_quantity）。
- ConfigMap 与 Secret 的 envFrom 互斥于内联 env。
- SQLite 驱动为 modernc.org/sqlite（driver name 仍是 sqlite），DSN 用 _pragma 参数。
- 表名 render_records；旧 config_records 不迁移。
- web/dist 已提交；改前端后必须 npm run build，否则二进制里是旧 UI。
- 生成后编辑任意表单字段会清空旧结果（SELECTION_CHANGED → planning），这是状态机的预期行为，不是 bug。

## 历史修复（不要回退）

1. 端口 0 时不再生成 containerPort: 0（纯工作负载不输出 ports 段）。
2. k8s 环境变量不再被丢弃（内联 env 或 ConfigMap envFrom）。
3. 空 k8s_resource 从 500 改为 400。
4. 依赖节点去重；HPA 多目标返回 422。
5. 构建不再依赖 cgo，Docker/Release 产物可启动。