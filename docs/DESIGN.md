# ConfigGen 设计总览与维护规范

> 本文是项目设计文档的入口，负责跨前后端的架构、决策与维护流程。
> 后端实现细节见 [BACKEND.md](BACKEND.md)，前端实现细节见 [FRONTEND.md](FRONTEND.md)。

## 1. 目标与非目标

### 目标

1. 在 Web 页面选择 Kubernetes 资源，自动处理资源之间的依赖，输出可直接 kubectl apply 的多文件 YAML。
2. 依赖、命名、引用、排序、校验全部由后端负责，前端只做输入与展示。
3. 生成物可复现、可解释、可追溯（历史快照含目录版本）。
4. 单二进制分发：前端构建产物与全部逻辑内嵌，无 cgo、无外部中间件。

### 非目标（当前版本）

- 不做 Helm Chart 生成 / 发布生命周期管理。
- 不做多租户、鉴权、配额中心。
- 不做服务端 dry-run、集群连接、GitOps 推送。
- 不做资源模板热加载、外部插件、i18n。

## 2. 文档地图

| 文档 | 内容 | 读者 |
| --- | --- | --- |
| docs/DESIGN.md（本文） | 架构、数据流、决策记录、维护流程、验收标准 | 所有人 |
| docs/BACKEND.md | 分层、领域模型、目录、解析算法、builder、API、存储、测试、扩展 | 后端开发/维护 |
| docs/FRONTEND.md | 技术栈、数据契约、store、组件、交互、样式、构建、开发规范 | 前端开发/维护 |
| README.md | 使用说明、API 示例、快速开始、已知限制 | 使用者 |
| AGENTS.md | 仓库导航、命令、不可破坏的约定 | 后续 AI/开发者 |

文档同步规则：**任何影响架构、API 契约、资源目录、默认值、构建方式或前端交互的改动，必须在同一个变更中更新对应文档。** 只改代码不改文档视为未完成。

## 3. 系统架构

~~~
┌──────────────────────────────────────────────────────────┐
│ 浏览器：Vue 3 SPA（Pinia / Naive UI / vis-network）       │
└───────────────┬──────────────────────────────────────────┘
                │ JSON（snake_case，严格字段）
┌───────────────▼──────────────────────────────────────────┐
│ presentation（Gin）                                       │
│ 路由 / 严格解码 / 错误映射 / SPA 静态服务                 │
├──────────────────────────────────────────────────────────┤
│ usecase：Plan / Render / Record 查询 / Health             │
├──────────────────────────────────────────────────────────┤
│ domain：实体 + Catalog + Resolve + Validate + Store 接口  │
├──────────────────────────────────────────────────────────┤
│ infrastructure                                            │
│  builders（k8s.io/api） / render / storage（SQLite）      │
└──────────────────────────────────────────────────────────┘
~~~

依赖方向：presentation → usecase → domain ← infrastructure。domain 不依赖任何框架或外部 IO。

## 4. 运行时数据流

~~~
Selection（type + app + resources）
   │ domain.Resolve：纯函数，无 IO
   ▼
Plan（nodes / edges / diagnostics / valid）
   │ render.Render：builder 构造对象 + YAML 序列化 + 产物校验
   ▼
[]Artifact + bundle
   │ usecase.Render
   ▼
Record（Selection + Plan + Files + CatalogVersion）──► SQLite 快照
~~~

前端有两条调用：

- 输入变化（300ms 防抖）→ POST /api/v1/plan → 只预览依赖与诊断。
- 点击生成 → POST /api/v1/render → 返回并持久化完整记录。

## 5. 仓库结构

~~~
cmd/configgen/            进程入口、DI、HTTP Server、优雅停机
config/                   .env 与监听地址
internal/domain/          唯一事实源（catalog/resolve/validate/types）
internal/usecase/         业务入口
internal/infrastructure/  builders / render / storage
internal/presentation/    Gin 路由与 SPA
web/src/                  Vue 源码
web/dist/                 构建产物（提交、嵌入）
docs/                     本组文档
~~~

## 6. 跨层约定

### 6.1 命名

- 资源名 = appName + "-" + catalog.Suffix，唯一且符合 DNS-1123。
- 引用只能是 Node.Refs 中的 (Kind, Name)，禁止在 builder 中拼名字。
- JSON 字段统一 snake_case；前端类型与后端结构一一对应。

### 6.2 错误

- 输入/参数错误：400，响应含 diagnostics。
- 依赖冲突（HPA 多目标等）：422。
- 内部错误：500，不泄露内部细节。
- 错误分类由 domain.AppError 承载，presentation 只做映射。

### 6.3 确定性

- 节点按 (Level, Kind, Name) 排序；环境变量、端口按固定规则排序。
- 同一 Selection 两次渲染必须字节一致；bundle 与 files 内容一致。
- 目录版本 CatalogVersion 在资源目录语义变化时递增，并随记录保存。

### 6.4 安全与输入

- 请求体 1 MiB 上限；未知字段拒绝。
- 名称/端口/环境变量在 domain 校验，禁止把用户输入直接拼进 YAML 结构（使用类型化对象）。
- 生成的 Secret 不含默认口令；无内容时显式告警。

## 7. 设计决策记录（ADR）

| 编号 | 决策 | 原因 | 代价/约束 |
| --- | --- | --- | --- |
| ADR-1 | Plan 与 Render 分离，Resolve 为纯函数 | 依赖可单测、可预览、前端可复用；避免 builder 内推导 | 需要维护 Plan 中间模型与 Ref 绑定 |
| ADR-2 | 用 k8s.io/api 类型化对象替代 text/template | 编译期字段正确、YAML 由序列化器保证、无注入面 | 依赖体积较大；默认值需在 builder 显式设置 |
| ADR-3 | Catalog 作为单一事实源 | 校验/命名/默认值/关系/前端目录只维护一处 | 新增资源必须同时写 builder 与测试 |
| ADR-4 | SQLite 使用 modernc.org/sqlite（纯 Go） | 去掉 cgo 后 Docker/Release 可启动、可交叉编译 | 二进制略大；驱动行为需按 _pragma 配置 |
| ADR-5 | 前端 dist 提交并 go:embed | 纯 Go 构建无需 Node，单文件分发 | dist 可能过期，必须手动/CI 校验 |
| ADR-6 | 严格 JSON + 400/422/500 分类 | 客户端错误不再变成 500；契约稳定 | 前端字段必须与后端严格同步 |
| ADR-7 | 移除 Sentry/logrus/validator 死依赖 | 减少无用依赖与虚假能力声明；slog 统一日志 | 若未来需要错误追踪需重新接入中间件 |

## 8. 维护流程

### 8.1 变更前

1. 判断是否触及：资源目录、解析规则、API 契约、默认值、构建方式、前端交互。
2. 触及则先更新对应文档的“设计”部分，再改代码。
3. 新资源先加 catalog 与测试，再加 builder。

### 8.2 变更中

- 后端：保持 domain 纯函数；所有跨资源引用走 Ref；渲染后校验。
- 前端：不复制业务规则；类型与后端同步；保持 loading/empty/error 三态。
- 文档：代码与文档同一个变更提交。

### 8.3 变更后（提交前门禁）

~~~
gofmt -l cmd config internal     # 必须为空
go vet ./...                     # 必须通过
go test ./...                    # 必须全绿
go test -cover ./...             # 关注 domain/render 覆盖率
cd web && npm run build          # 必须更新 dist
CGO_ENABLED=0 go build ./cmd/configgen
~~~

### 8.4 版本与发布

- 资源目录语义变化：递增 CatalogVersion，并在文档中记录。
- 数据库结构变化：新增表/列需给出迁移说明；当前不迁移旧表。
- 发布：打 v* 标签，CI 产出多平台二进制与多架构镜像。
- 破坏性 API 变更：需同步前端与 README，并在本文记录决策。

## 9. 全局验收标准

- 选择 Ingress 时自动补齐 Service/Deployment，并给出 info 诊断。
- 空资源列表返回 400；HPA 多目标返回 422。
- 生成物为合法 YAML，kind/name 齐全，引用全部命中，名称符合 DNS-1123。
- 端口、环境变量、命名空间没有“静默丢失”。
- 历史详情可还原当时的 Plan 与文件。
- 前端能展示自动补充、诊断、关系图、多文件与下载。
- 构建不再依赖 cgo；Docker 镜像可启动。

## 10. 路线图

1. 生成物二次校验：kubeconform / kubectl --dry-run=server（可选连接集群）。
2. 参数化脚手架：StorageClass provisioner、IngressClass controller、CronJob schedule、PVC 容量、Secret 内容。
3. 导出能力：Helm values.yaml、ZIP 多文件包。
4. 工程化：前端单测 + E2E、CI 校验 dist 新鲜度、gosec 阻断策略。
5. 运维能力：鉴权、审计、记录搜索与清理策略。
