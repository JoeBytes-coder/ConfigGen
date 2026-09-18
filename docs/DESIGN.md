# ConfigGen 设计文档

> 本文档描述 ConfigGen 的整体架构与前端设计。功能清单与 API 用法以 `README.md` 为准,本文不重复罗列。

## 1. 系统概览

ConfigGen 是一个云原生配置生成工具:用户通过 Web 界面或 REST API 提交参数,后端基于模板生成 Docker Compose / Kubernetes / Dockerfile / Kustomize 配置,记录持久化到 SQLite,并为 K8s 资源推导拓扑关系图。

```
┌─────────────────────────────────────────────────────┐
│                    浏览器                             │
│  Vue 3 SPA(Naive UI / Pinia / Vue Router)          │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP / JSON
┌──────────────────────▼──────────────────────────────┐
│              presentation(Gin)                      │
│  REST API(/api/v1/*)+ SPA 静态服务与路由回退        │
├─────────────────────────────────────────────────────┤
│              usecase(业务逻辑)                      │
│  Prepare 默认值 → GenerateAndSave → 查询             │
├─────────────────────────────────────────────────────┤
│              domain(实体与接口)                     │
├─────────────────────────────────────────────────────┤
│   infrastructure               │ infrastructure      │
│   generators(注册表 + 模板)   │ storage(SQLite)   │
└─────────────────────────────────────────────────────┘
```

依赖方向: `presentation → usecase → domain ← infrastructure`,前端通过 REST API 与后端解耦。

## 2. 前端架构

### 2.1 技术栈

| 技术 | 职责 |
|------|------|
| Vue 3(Composition API + `<script setup>`) | 视图层 |
| Vite 7 | 开发服务器与生产构建 |
| TypeScript | 类型约束(编辑器支持,构建不做类型阻塞) |
| Pinia | 状态管理(生成器表单、历史记录) |
| Vue Router 4(history 模式) | SPA 路由 |
| Naive UI | 组件库,通过 `n-config-provider` 定制主题 |
| vis-network | K8s 资源拓扑图渲染 |

### 2.2 目录结构

```
web/
├── embed.go              # //go:embed all:dist,构建产物嵌入二进制
├── package.json          # 前端依赖与脚本
├── vite.config.ts        # Vite 配置(@ 别名、/api 开发代理)
├── tsconfig.json
├── index.html            # Vite 入口(内联 SVG favicon)
├── src/
│   ├── main.ts           # 应用装配:Pinia + Router
│   ├── App.vue           # 主题注入 + 布局(页头/导航/内容)
│   ├── theme.ts          # Naive UI 主题令牌(#7DAEE1 主色体系)
│   ├── router/index.ts   # 路由表
│   ├── api/              # 类型定义(types.ts)+ fetch 封装(client.ts)
│   ├── stores/           # Pinia:generator(表单与结果)、history(记录列表)
│   ├── constants/k8s.ts  # 20 种资源类型、节点配色、类型展示元信息
│   ├── components/       # EnvVarEditor / ResourcePicker / ResultPanel / ResourceGraph
│   ├── views/            # GeneratorView / HistoryView / RecordDetailView
│   └── styles/main.css   # 全局样式与设计令牌(CSS 变量)
└── dist/                 # Vite 构建产物(提交进仓库,供纯 Go 构建使用)
```

### 2.3 路由设计

| 路径 | 页面 | 说明 |
|------|------|------|
| `/` | GeneratorView | 配置表单 + 生成结果 + 拓扑图(主页) |
| `/history` | HistoryView | 生成记录列表(服务端分页) |
| `/history/:id` | RecordDetailView | 记录详情:请求参数、结果、拓扑图 |
| `*` | 重定向 `/` | 兜底 |

Vue Router 使用 history 模式,后端 `spaFallback` 对未匹配路径回退到 `index.html`(见 §4)。

### 2.4 状态管理

- **generator store**:四种配置类型共用的表单字段;`k8s` 与 `kustomize` 各自维护独立的资源勾选集合(与旧版行为一致);`buildPayload()` 负责与后端请求结构对齐 —— 环境变量仅在非空时发送,`replicas` 仅在勾选工作负载资源时发送,dockerfile 附加字段按需携带。生成结果保存在 `lastRecord`。
- **history store**:记录列表与服务端分页。后端 `/configs` 不返回总数,采用「多取一条」策略判断是否有下一页。

### 2.5 API 层约定

`src/api/client.ts` 统一封装 fetch:非 2xx 响应抛出携带 `error`/`details` 的 Error,由页面以 Naive UI message 呈现。类型定义 `types.ts` 与后端 `internal/domain/types.go` 的 JSON 字段一一对应(snake_case)。

## 3. 视觉设计

### 3.1 设计原则

贴合「云原生配置生成器」的开发者工具定位:

- **克制干净**:浅色底 + 白色卡片,信息层级靠留白与字重而非重装饰;
- **主色克制使用**:主色 #7DAEE1 用于交互态(按钮、选中、链接)而非大面积铺色;
- **技术感细节**:配置输出用等宽字体展示、拓扑图保留资源分类配色、页面右上角品牌 Logo 为 `{}` 字形渐变方块;
- **中文界面**,组件文案与后端提示保持一致的中文语气。

### 3.2 色彩令牌

| 令牌 | 值 | 用途 |
|------|----|------|
| 主色 | `#7DAEE1` | 品牌色、主按钮、选中态 |
| 主色 Hover | `#93BEE8` | 悬停 |
| 主色 Pressed | `#6698CF` | 按下 |
| 页面背景 | `#F5F8FC` | body / 布局底色 |
| 卡片背景 | `#FFFFFF` | 卡片、弹层 |
| 边框 | `#DFE9F5` | 输入框、分割线 |
| 主文本 | `#2C3E5D` | 标题 |
| 次级文本 | `#44557A` / `#8296B3` | 正文 / 辅助说明 |
| 代码块背景 | `#F7FAFD` | 配置输出 |

主题通过 `src/theme.ts` 注入 Naive UI `GlobalThemeOverrides`,并在 `main.css` 中以 CSS 变量(`--cg-*`)同步一份供自定义样式使用。拓扑图节点按资源分类使用柔和色系(`constants/k8s.ts` 的 `KIND_COLORS`),描边统一使用主色系 `#6E9CD3`。

## 4. 构建与部署

### 4.1 生产构建(单二进制)

```
cd web && npm install && npm run build   # 产出 web/dist
go build -o configgen ./cmd/configgen    # dist 经 go:embed 嵌入
```

`web/dist` 提交进仓库,保证在没有 Node 环境的机器上也能纯 Go 构建(仓库中的 dist 若为占位页,则构建后提示执行 npm 构建)。`web/node_modules` 不入库。

### 4.2 前后端联调

```
终端 1:go run ./cmd/configgen           # 后端 :8080
终端 2:cd web && npm run dev            # Vite :5173,/api 代理到 :8080
```

访问 http://localhost:5173 进行开发,享受热更新;后端 API 与生产环境完全一致。

### 4.3 SPA 服务(Go 端)

`presentation/server.go` 的 `spaFallback` 处理所有未匹配路由:

1. `/api/*` 未匹配 → JSON 404(不返回 SPA 文档);
2. 命中 `dist` 内的静态文件(如 `/assets/*.js`)→ 直接返回文件(带 MIME 与协商缓存);
3. 其余路径 → 返回 `index.html`,由 Vue Router 接管(客户端路由刷新可用)。

`web/embed.go` 使用 `//go:embed all:dist` 嵌入整个构建产物;`embed.FS` 经 `io/fs.Sub` 得到 `dist` 子树供静态服务使用。

## 5. 关键数据流

```
表单提交 → generator store buildPayload()
        → POST /api/v1/generate/{type}
        → 后端 Prepare(默认值) → validate → 生成模板 → 关系抽取 → SQLite 保存
        → 返回 ConfigRecord(id/request/result/resources/edges)
        → ResultPanel 展示配置(复制/下载)
        → ResourceGraph 用 result 内联的 resources/edges 渲染 vis-network
```

历史页 `GET /api/v1/configs?offset&limit` → 表格;详情页 `GET /api/v1/configs/:id` → 描述列表 + 结果 + 拓扑图。

## 6. 未实现功能(规划中)

Nginx 配置、Docker Daemon 配置、Containerd 配置、ZIP 打包下载、模板热更新、外部插件系统、i18n 多语言。详见 `AGENTS.md` 的「Not implemented」章节。
