# ConfigGen

ConfigGen 是一个基于 Go 语言开发的云原生配置生成工具,提供表单化输入界面,根据用户配置自动生成标准化的 **Docker Compose**、**Kubernetes 资源清单**、**Dockerfile** 和 **Kustomize** 配置文件,并为 K8s 资源自动推导可视化拓扑关系图。本工具旨在解决云原生部署过程中配置文件编写复杂、格式易错、维护困难等问题,通过结构化表单简化配置生成流程。

## 功能特性

- **四种配置类型**: Docker Compose、Kubernetes(支持 20 种资源类型)、Dockerfile、Kustomize
- **Web 界面**: Vue 3 单页应用(Vite + Pinia + Naive UI + Vue Router,主色 `#7DAEE1`),表单化输入、结果一键复制与按类型命名下载、K8s 资源拓扑图、历史记录与详情页
- **资源关系图**: K8s / Kustomize 生成时自动推导资源节点与关联关系(Service→Deployment、Ingress→Service、RBAC 绑定等),前端以 vis-network 渲染拓扑图
- **REST API**: 提供完整的 REST API 接口,支持程序化调用
- **数据持久化**: 使用 SQLite(WAL 模式)存储配置记录,支持分页查询
- **模板驱动**: 基于 Go text/template,全部模板与前端资源嵌入二进制(`go:embed`),单文件部署,无运行时依赖
- **确定性输出**: 环境变量按键名排序渲染,输出可复现
- **Clean Architecture**: 四层架构,确保代码可维护性和可扩展性

## 项目结构

项目采用 Clean Architecture 分层设计:

```
configgen/
├── cmd/configgen/          # 应用程序入口(DI 组装、优雅停机)
├── config/                 # 运行时配置:.env 加载、监听地址解析
├── internal/
│   ├── domain/             # 领域层:实体和接口
│   │   ├── types.go        # 核心数据结构(ConfigRequest/ConfigResult/ConfigRecord)
│   │   ├── store.go        # Store 存储接口
│   │   └── generator.go    # ConfigGenerator 接口
│   ├── usecase/            # 用例层:默认值填充(Prepare)、生成与持久化
│   │   └── config_generator.go
│   ├── infrastructure/     # 基础设施层:外部实现
│   │   ├── generators/     # 四种生成器 + 注册表
│   │   │   └── templates/  # 嵌入式生成模板(20 种 K8s 资源等)
│   │   └── storage/        # SQLite 存储实现
│   └── presentation/       # 表示层:Gin HTTP 处理器
│       ├── server.go
│       └── util.go
├── web/                    # Vue 3 前端(src/ 源码 + dist/ 构建产物,go:embed 嵌入)
├── dockerfile              # 多阶段 Docker 构建
├── .env.example            # 环境变量示例
├── AGENTS.md
├── README.md
├── go.mod
└── LICENSE
```

`db/` 是运行时 SQLite 数据目录(默认 `db/configgen.db`),已被 `.gitignore` 忽略。

### 架构说明

- **Domain**: 包含业务实体和核心接口,不依赖外部框架。
- **Use Case**: 实现业务规则,协调领域对象。
- **Infrastructure**: 实现外部接口,如数据库、文件系统。
- **Presentation**: 处理用户输入和输出,依赖 Use Case。

依赖方向: `presentation → usecase → domain ← infrastructure`

## 安装

### 前置要求

- Go 1.25+
- C 编译器(SQLite 驱动 `mattn/go-sqlite3` 依赖 cgo;Windows 需要 MinGW/gcc,macOS/Linux 通常自带)
- Node.js ≥ 20.19(仅重新构建前端时需要;仓库已提交 `web/dist` 构建产物,纯 Go 构建无需 Node)

### 安装步骤

1. 克隆仓库:
   ```bash
   git clone https://github.com/your-repo/configgen.git
   cd configgen
   ```

2. 下载依赖:
   ```bash
   go mod tidy
   ```

3. 构建后端(已包含前端构建产物):
   ```bash
   go build ./cmd/configgen
   ```

4. (可选)重新构建前端:
   ```bash
   cd web
   npm install
   npm run build
   ```

## 使用方法

### 运行服务器

```bash
./configgen -addr=:8080
```

启动时会自动加载根目录 `.env` 文件(如果存在);已设置的环境变量优先于 `.env` 中的值。

### Web 界面

访问 `http://localhost:8080` 进入 Web 界面(Vue 3 + Naive UI 单页应用,主色 `#7DAEE1`):

- **生成配置页**: 按类型(Dockerfile / Docker Compose / Kubernetes / Kustomize)填写参数、添加环境变量、勾选 K8s 资源,生成后查看结果并复制/下载;K8s / Kustomize 类型附带资源拓扑关系图
- **历史记录页**: 分页浏览全部生成记录,点击查看详情(请求参数、生成结果、拓扑图)

### API 使用

#### 生成配置

- **POST /api/v1/generate**: 通用端点,`type` 由请求体指定,缺省为 `k8s`
- **POST /api/v1/generate/compose**: 生成 Docker Compose 配置
- **POST /api/v1/generate/k8s**: 生成 Kubernetes 配置
- **POST /api/v1/generate/dockerfile**: 生成 Dockerfile
- **POST /api/v1/generate/kustomize**: 生成 Kustomize 配置

类型化端点会强制使用 URL 中的类型,忽略请求体里的 `type` 字段。

请求字段说明:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `type` | string | 是* | `compose` / `k8s` / `dockerfile` / `kustomize`;通用端点缺省为 `k8s` |
| `app_name` | string | 是 | 应用名,1-50 字符,用作资源名前缀 |
| `image` | string | 是 | 镜像名;k8s 请求中留空时自动填 `nginx` |
| `tag` | string | 是 | 镜像标签;k8s 请求中留空时自动填 `latest` |
| `port` | int | 否 | 0-65535;含 Service/Ingress 的 k8s 请求缺省为 80 |
| `env` | map | 否 | 环境变量,按键名排序输出 |
| `replicas` | int | 否 | 副本数 1-100,k8s 工作负载使用,缺省为 1 |
| `k8s_resource` | []string | k8s 必填 | K8s 资源类型列表,见下文 20 种 |
| `dockerfile_base_image` | string | 否 | 基础镜像,缺省为 `image:tag` |
| `dockerfile_workdir` | string | 否 | 工作目录,缺省为 `/app` |
| `dockerfile_cmd` | []string | 否 | 容器启动命令,缺省为 `["./<app_name>"]` |

支持的 20 种 K8s 资源类型:
`Deployment`、`StatefulSet`、`DaemonSet`、`CronJob`、`Service`、`Ingress`、`ConfigMap`、`Secret`、`Namespace`、`PersistentVolumeClaim`、`ServiceAccount`、`Role`、`RoleBinding`、`ClusterRole`、`ClusterRoleBinding`、`HorizontalPodAutoscaler`、`ResourceQuota`、`LimitRange`、`NetworkPolicy`、`StorageClass`

Kubernetes 请求示例:
```json
POST /api/v1/generate/k8s
{
  "app_name": "my-app",
  "image": "nginx",
  "tag": "latest",
  "port": 80,
  "replicas": 3,
  "k8s_resource": ["Deployment", "Service"]
}
```

Dockerfile 请求示例:
```json
POST /api/v1/generate/dockerfile
{
  "app_name": "my-app",
  "image": "golang",
  "tag": "1.25",
  "port": 8080,
  "dockerfile_base_image": "alpine:3.19",
  "dockerfile_workdir": "/app",
  "dockerfile_cmd": ["./my-app"]
}
```

响应示例:
```json
{
  "id": 1,
  "request": {...},
  "result": {
    "type": "k8s",
    "config": "---\napiVersion: apps/v1\nkind: Deployment\n...",
    "resources": [{"kind": "Deployment", "name": "my-app-deployment"}, ...],
    "edges": [{"source_kind": "Service", "target_kind": "Deployment", "relation": "selects", ...}]
  },
  "created_at": "2023-01-01T00:00:00Z"
}
```

#### 查询配置记录

- **GET /api/v1/configs?offset=0&limit=20**: 分页查询配置记录列表(`limit` 上限 100,默认 20)
- **GET /api/v1/configs/:id**: 查询单条完整记录

响应示例:
```json
{
  "id": 1,
  "request": {...},
  "result": {...},
  "created_at": "2023-01-01T00:00:00Z"
}
```

#### 查询资源关系图

K8s / Kustomize 类型的记录附带资源节点(`resources`)与关联边(`edges`),用于拓扑可视化:

- **GET /api/v1/configs/:id/resources**: 资源节点列表
- **GET /api/v1/configs/:id/edges**: 资源关联边列表
- **GET /api/v1/configs/:id/graph**: 组合返回 `{"resources": [...], "edges": [...]}`

关系规则基于请求的资源类型集合推导,涵盖 Service→Deployment(`selects`)、Ingress→Service(`routes-to`)、Deployment→ConfigMap/Secret(`mounts`)、HPA→Deployment(`scales`)、RBAC 绑定(`binds`/`grants-to`)等。

#### 健康检查

**GET /health**

响应:
```json
{"status": "ok"}
```

## 配置选项

| 配置 | 说明 |
|------|------|
| `-addr` | 服务器监听地址。优先级: `-addr` 参数 > `ADDR` 环境变量 > `PORT` 环境变量 > 默认 `:8080` |
| `CONFIGGEN_DB` | SQLite 数据库文件路径,默认 `db/configgen.db` |
| `SENTRY_DSN` | 可选,设置后启用 Sentry 错误追踪 |
| `.env` | 启动时自动加载根目录 `.env` 文件(已被 gitignore,模板见 `.env.example`) |

## 测试

运行所有测试:
```bash
go test ./...
```

运行覆盖率测试:
```bash
go test -cover ./...
```

## 开发

### 代码质量检查

与 CI 保持一致的检查:

```bash
# 格式化(CI 会校验 gofmt)
go fmt ./...

# 静态分析
go vet ./...

# 测试
go test ./...
```

CI(`.github/workflows/go-ci.yml`)在 push main / PR 时执行 gofmt 检查、`go vet`、`go test`、构建冒烟和 gosec 安全扫描;打 `v*` 标签时自动发布 Linux/macOS 多平台二进制,并构建推送多架构 Docker 镜像到 ghcr.io。

### 前端开发

前端为 Vue 3 + Vite + Pinia + Naive UI + Vue Router 的单页应用,源码位于 `web/src`:

```bash
cd web
npm install
npm run dev      # Vite 开发服务器 http://localhost:5173,/api 代理到本地 Go 后端
npm run build    # 构建产物输出到 web/dist,由 go:embed 嵌入二进制
```

联调时先启动 Go 后端(`go run ./cmd/configgen`),再启动 Vite 开发服务器。系统与前端设计详见 [docs/DESIGN.md](docs/DESIGN.md)。

### 添加新生成器

1. 在 `internal/infrastructure/generators/` 实现新生成器,实现 `Generator` 接口(`interface.go`)。
2. 在生成器的 `init()` 中调用 `Register("type_name", &MyGenerator{})` 完成自注册。
3. 在 `internal/infrastructure/generators/templates/<type>/` 添加模板并通过 `//go:embed` 嵌入。
4. 在 `internal/domain/types.go` 的 `Type` 字段 `oneof` 校验标签中加入新类型(类型合法性由该校验控制)。

## 部署

### Docker 构建

使用根目录的 `dockerfile`(多阶段构建,基于 `golang:1.25-alpine3.23`),也可直接使用 CI 发布到 ghcr.io 的多架构镜像(linux/amd64、linux/arm64)。

### Kubernetes 部署

参考 `internal/infrastructure/generators/templates/k8s/` 中的示例配置。

## 贡献

欢迎贡献!请遵循以下步骤:

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 致谢

- [Gin](https://gin-gonic.com/) - Web 框架
- [SQLite](https://www.sqlite.org/) - 数据库
- [vis-network](https://visjs.github.io/vis-network/) - 拓扑图可视化
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) - 架构模式
