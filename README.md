# ConfigGen

ConfigGen 是一个基于 Go 语言开发的云原生配置生成工具，提供表单化输入界面，根据用户配置自动生成标准化的 Docker Compose、Kubernetes 资源清单、Nginx 配置、Docker Daemon 配置及 Containerd 配置文件。本工具旨在解决云原生部署过程中配置文件编写复杂、格式易错、维护困难等问题，通过结构化表单简化配置生成流程。

## 功能特性

- **多格式支持**: 生成 Docker Compose、Kubernetes YAML 等配置。
- **Web 界面**: 提供 HTML 界面进行配置输入。
- **REST API**: 提供完整的 REST API 接口，支持程序化调用。
- **数据持久化**: 使用 SQLite 存储配置记录。
- **模板驱动**: 基于 Go text/template，支持自定义模板。
- **Clean Architecture**: 采用 Clean Architecture 设计，确保代码可维护性和可扩展性。

## 项目结构

项目采用 Clean Architecture 分层设计：

```
configgen/
├── cmd/configgen/          # 应用程序入口
├── config/                 # 运行时配置加载 helpers
├── internal/
│   ├── domain/             # 领域层：实体和接口
│   │   ├── types.go        # 核心数据结构
│   │   └── store.go        # 存储接口
│   ├── usecase/            # 用例层：业务逻辑
│   │   └── config_generator.go
│   ├── infrastructure/     # 基础设施层：外部实现
│   │   ├── generators/     # 配置生成器实现与模板
│   │   │   └── templates/   # 嵌入式生成模板
│   │   └── storage/        # 数据存储实现
│   └── presentation/       # 表示层：HTTP 处理器
│       ├── server.go
│       └── util.go
├── web/                    # 前端页面和嵌入式静态资源
├── .env.example            # 示例环境变量文件
├── AGENTS.md
├── README.md
├── go.mod
└── LICENSE
```

`db/` 是运行时 SQLite 数据目录，通常由 `.gitignore` 忽略。

### 架构说明

- **Domain**: 包含业务实体和核心接口，不依赖外部框架。
- **Use Case**: 实现业务规则，协调领域对象。
- **Infrastructure**: 实现外部接口，如数据库、文件系统。
- **Presentation**: 处理用户输入和输出，依赖 Use Case。

## 安装

### 前置要求

- Go 1.21+
- SQLite3

### 安装步骤

1. 克隆仓库：
   ```bash
   git clone https://github.com/your-repo/configgen.git
   cd configgen
   ```

2. 下载依赖：
   ```bash
   go mod tidy
   ```

3. 构建：
   ```bash
   go build ./cmd/configgen
   ```

## 使用方法

### 运行服务器

```bash
./configgen -addr=:8080
```

或使用环境变量：
```bash
CONFIGGEN_DB=configgen.db ./configgen
```

### Web 界面

访问 `http://localhost:8080` 进入 Web 界面。

### API 使用

#### 生成配置

**POST /api/v1/generate**

请求体示例：
```json
{
  "type": "k8s",
  "app_name": "my-app",
  "image": "nginx",
  "tag": "latest",
  "port": 80,
  "replicas": 3,
  "k8s_resource": ["Deployment", "Service"]
}
```

响应示例：
```json
{
  "id": 1,
  "request": {...},
  "result": {
    "type": "k8s",
    "config": "---\napiVersion: apps/v1\nkind: Deployment\n..."
  },
  "created_at": "2023-01-01T00:00:00Z"
}
```

#### 生成特定类型配置

- **POST /api/v1/generate/compose**: 生成 Docker Compose 配置
- **POST /api/v1/generate/k8s**: 生成 Kubernetes 配置

#### 查询配置记录

**GET /api/v1/configs/:id**

响应示例：
```json
{
  "id": 1,
  "request": {...},
  "result": {...},
  "created_at": "2023-01-01T00:00:00Z"
}
```

#### 健康检查

**GET /health**

响应：
```json
{"status": "ok"}
```

## 配置选项

- `-addr`: 服务器监听地址（默认 `:8080`）
- `CONFIGGEN_DB`: 数据库文件路径（默认 `configgen.db`）

## 测试

运行所有测试：
```bash
go test ./...
```

运行覆盖率测试：
```bash
go test -cover ./...
```

## 开发

### 代码质量检查

```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...

# 安装 golangci-lint（如果未安装）
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 运行 linter
golangci-lint run
```

### 添加新生成器

1. 在 `internal/infrastructure/generators/` 实现新生成器。
2. 实现 `Generator` 接口。
3. 在 `registry.go` 注册新生成器。

## 部署

### Docker 构建

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build ./cmd/configgen

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/configgen .
CMD ["./configgen"]
```

### Kubernetes 部署

参考 `internal/infrastructure/generators/templates/k8s/` 中的示例配置。

## 贡献

欢迎贡献！请遵循以下步骤：

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
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) - 架构模式
