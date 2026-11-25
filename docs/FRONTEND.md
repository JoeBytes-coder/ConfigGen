# ConfigGen 前端设计与实现规范

> 适用范围：web/src、web/index.html、web/vite.config.ts、web/package.json，以及 web/dist 的构建与提交约定。
> 变更前端代码时必须同步更新本文档；后端契约见 docs/BACKEND.md，总览见 docs/DESIGN.md。

## 1. 技术栈与版本

| 项 | 选型 | 说明 |
| --- | --- | --- |
| 框架 | Vue 3.5（Composition API + script setup） | 全部视图使用 setup 语法 |
| 构建 | Vite 7 + @vitejs/plugin-vue 6 | npm run build 输出 dist |
| 语言 | TypeScript 5.9（strict） | 构建不做类型检查，但代码必须保持类型正确 |
| 状态 | Pinia 3（setup store） | meta / generator / history 三个 store |
| 路由 | Vue Router 4（history 模式） | 后端 spaFallback 回退 index |
| UI | Naive UI 2.42 | 按需 import 组件，不全局注册 |
| 图 | vis-data 7 + vis-network 9 | 资源关系图 |
| 样式 | 原生 CSS + CSS 变量 + scoped style | 无预处理器、无 CSS 框架 |

## 2. 启动、构建与嵌入

### 2.1 开发

~~~
cd web
npm install
npm run dev
~~~

- Vite 端口 5173。
- vite.config.ts 将 /api 代理到 http://localhost:8080。
- 先启动 Go 后端，再启动 Vite，前后端联调。

### 2.2 生产构建

~~~
cd web
npm run build
~~~

- 输出 web/dist；该目录已提交到仓库，供 go:embed 使用。
- 后端 web/embed.go 使用 //go:embed all:dist，二进制内嵌静态资源。
- 改完前端必须重新构建并提交 dist，否则二进制仍是旧 UI。
- 路由按页面懒加载，dist 中会有 GeneratorView/HistoryView/RecordDetailView 等独立 chunk。

### 2.3 体积现状

最近一次构建：ResultPanel chunk 约 552KB（主要来自 Naive UI + vis-network），超过 Vite 500KB 警告线；其余 chunk 均在 220KB 以下。可接受的优化方向：按需拆分图表库、将 vis-network 改为动态 import。

## 3. 目录结构与职责

~~~
web/
├── index.html               # 入口；内联 SVG favicon；标题 ConfigGen
├── vite.config.ts           # @ 别名、/api 代理、outDir=dist
├── tsconfig.json            # strict、@/* -> src/*、noEmit
├── embed.go                 # //go:embed all:dist
└── src/
    ├── main.ts              # createApp + Pinia + Router + 全局样式
    ├── App.vue              # 主题、布局、页头导航、消息 Provider
    ├── theme.ts             # Naive UI 主题令牌
    ├── router/index.ts      # 3 个页面 + 兜底重定向
    ├── api/types.ts         # 与后端 JSON 一一对应的类型
    ├── api/client.ts        # fetch 封装与错误归一
    ├── constants/k8s.ts     # 分类标签/颜色、类型展示元信息
    ├── stores/meta.ts       # 资源目录
    ├── stores/generator.ts  # 表单、Selection、Plan、Render
    ├── stores/history.ts    # 历史列表与分页
    ├── components/
    │   ├── EnvVarEditor.vue   # 键值编辑器
    │   ├── ResourcePicker.vue # 目录驱动的资源选择
    │   ├── ResourceGraph.vue  # Plan 拓扑图
    │   └── ResultPanel.vue    # 多文件结果与复制/下载
    ├── views/
    │   ├── GeneratorView.vue    # 生成器主页
    │   ├── HistoryView.vue      # 历史列表
    │   └── RecordDetailView.vue # 记录详情
    ├── utils/format.ts        # 时间、镜像展示
    └── styles/main.css        # CSS 变量与全局布局
~~~

## 4. 数据契约

所有类型定义在 api/types.ts，字段必须与后端 JSON（snake_case）一致。

### 4.1 核心类型

| 类型 | 关键字段 |
| --- | --- |
| ConfigType | 固定 k8s |
| Port | name、container、service |
| AppContext | name、namespace、image、tag、ports、replicas、env、secret_env、resources、probe、host、tls_secret |
| Selection | type、app、resources |
| Ref | kind、name、port |
| PlanNode | kind、name、namespace、level、auto_added、refs |
| PlanEdge | from、to、relation |
| Diagnostic | severity（error/warning/info）、code、kind、message、hint |
| Plan | type、app、nodes、edges、diagnostics、valid、catalog_version |
| Artifact | path、kind、content |
| ConfigRecord | id、type、app、selection、plan、files、bundle、catalog_version、created_at |
| RecordSummary | id、type、app_name、namespace、kinds、created_at |
| CatalogKind | kind、api_version、category、scope、suffix、level、needs_image、needs_port、needs_replicas、requires |
| ApiError | Error + diagnostics? |

### 4.2 API 客户端

api/client.ts 的统一约定：

- request 使用 fetch，非 2xx 时读取 body.error / body.details 拼成中文错误。
- 将 body.diagnostics 挂到 Error 上（ApiError.diagnostics），供生成器 store 展示。
- 提供 getCatalog、planSelection、renderSelection、listRecords、getRecord。
- 不在客户端做字段转换；snake_case 直接透传。

## 5. 状态管理

### 5.1 meta store

职责：资源目录与派生索引。

| 成员 | 类型 | 说明 |
| --- | --- | --- |
| kinds | ref<CatalogKind[]> | 目录原始数据 |
| version | ref<string> | catalog_version |
| loaded | ref<boolean> | 是否加载成功 |
| error | ref<string> | 加载失败信息 |
| byCategory | computed | 按分类分组，顺序由 CATEGORY_ORDER 决定 |
| kindToMeta | computed | Map<Kind, CatalogKind> |
| workloadKinds | computed | category=workload 的 Kind 集合 |
| fetchMeta | action | GET /api/v1/catalog，失败写 error |

约定：App.vue onMounted 调用一次；ResourcePicker/ResourceGraph 只消费，不重复请求。禁止在前端硬编码资源 Kind 列表。

### 5.2 generator store

职责：表单状态、Selection 构造、Plan 预览、最终生成。

表单状态：

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| appName | myapp | 应用名 |
| namespace | 空 | 目标命名空间 |
| image / tag | nginx / 1.25 | 镜像 |
| port | 8080 | 单端口，写入 ports[0] 的 container 与 service |
| replicas | 2 | 无工作负载时输入禁用，但值仍随请求发送 |
| host / tlsSecret | 空 | Ingress 域名与 TLS |
| envPairs / secretPairs | [] | EnvPair 列表 |
| cpuRequest / memRequest / cpuLimit / memLimit | 100m/128Mi/500m/512Mi | 资源配额 |
| probeEnabled / probePath / readinessPath | false / /healthz / /readyz | 探针 |
| resources | [] | 资源 Kind 勾选集合 |

派生与动作：

- status：empty / planning / planned / plan_error / rendering / rendered，是唯一的交互状态源。
- plan、planError、planDiagnostics、planning、rendering、record：全部由 status 派生的只读 computed，组件不得自行组合布尔值。
- canGenerate：仅 planned / rendered 为 true，生成按钮与提交校验都使用它。
- hasWorkload：勾选集合是否包含 workload 分类（由 meta 驱动，副本数输入据此禁用）。
- buildSelection()：构造 Selection；空端口不发送；空 env/secret 不发送；固定 type=k8s，发送 resources/replicas。
- refreshPlan()：名称空时 dispatch PLAN_FAILED；资源空时 dispatch SELECTION_CHANGED(false)；否则 POST /plan，使用递增请求序号只接受最新响应，成功 dispatch PLAN_SUCCEEDED，失败 dispatch PLAN_FAILED。
- schedulePlan()：300ms 防抖；watch 表单字段与 resources（deep）。
- onSelectionChanged()：任何表单变更立即 dispatch SELECTION_CHANGED（旧结果失效），资源非空时再防抖请求。
- generate()：仅 canGenerate 可调用；dispatch RENDER_STARTED → POST /render → 成功 RENDER_SUCCEEDED（record + plan），失败 RENDER_FAILED 回到 planned；若生成期间选择已变化则丢弃本次结果并提示重新生成。

约定：

- 前端不做依赖推导、不拼接资源名，一切以 /plan 返回为准。
- 自动补充节点必须展示，不允许隐藏。
- 输入变化即预览，但正式生成仍由用户点击。
- 生成结果属于当前 Selection：编辑任意字段会清空旧 record，避免“旧结果 + 新表单”误导。

### 5.3 生成器状态机（generatorMachine.ts）

状态与迁移集中在一个纯函数 transition(state, event) 中；store 只负责异步副作用与 dispatch。

| 状态 | 含义 | canGenerate |
| --- | --- | --- |
| empty | 未选资源 / 初始 | false |
| planning | 正在或等待解析 | false |
| planned | 有合法 Plan | true |
| plan_error | 解析/校验失败 | false |
| rendering | 正在生成 | false |
| rendered | 已有结果 | true |

| 事件 | 迁移 |
| --- | --- |
| SELECTION_CHANGED(hasResources) | true → planning；false → empty（任何状态） |
| PLAN_SUCCEEDED | planning → planned |
| PLAN_FAILED | planning/empty → plan_error |
| RENDER_STARTED | planned/rendered → rendering |
| RENDER_SUCCEEDED | rendering → rendered |
| RENDER_FAILED | rendering → planned（保留 plan） |
| RESET | 任意 → empty |

未在表中声明的事件一律忽略，例如 plan_error 下的 RENDER_STARTED 不会改变状态。这样“错误态不可生成”“生成中不可重复生成”“编辑使旧结果失效”都由迁移表保证，而不是散落在组件里。

测试：

~~~bash
cd web
npm run test:fsm   # Node 22.6+（--experimental-strip-types）；构建仍支持 Node 20.19+
~~~

### 5.4 history store

| 成员 | 说明 |
| --- | --- |
| records | RecordSummary[] |
| total | 服务端返回总数 |
| page | 当前页，从 1 开始 |
| totalPages | ceil(total / 10) |
| fetchRecords | 按 offset=(page-1)*10 请求 |
| goToPage / refresh | 翻页与刷新 |

约定：列表只使用摘要，不请求完整记录；详情页单独请求 /records/:id。

## 6. 页面与组件规范

### 6.1 App.vue

- NConfigProvider 注入 themeOverrides、zhCN/dateZhCN、inline-theme-disabled。
- NMessageProvider 包裹全局（useMessage 依赖它）。
- 页头：品牌 { } Logo、ConfigGen、副标题“云原生配置生成器”、导航“生成配置 / 历史记录”。
- 内容区固定最大宽度 1120px（main.css 的 .page）。
- onMounted 拉取资源目录。

### 6.2 路由

| 路径 | 组件 | 说明 |
| --- | --- | --- |
| / | GeneratorView | 生成器 |
| /history | HistoryView | 历史列表 |
| /history/:id(\d+) | RecordDetailView | 详情 |
| 其他 | 重定向 / | 兜底 |

history 模式；刷新由后端 spaFallback 返回 index.html。

### 6.3 GeneratorView

布局：CSS Grid，左 7 右 5；<=960px 单列；右列 sticky。

左侧表单顺序（不可随意重排）：

1. 应用名称（提示 DNS-1123）。
2. 镜像 + 标签（两列）。
3. 端口 + 副本数（两列；副本数在未选择工作负载时禁用）。
4. 命名空间（k8s 家族）。
5. 域名 / TLS Secret（仅 k8s，TLS 依赖域名）。
6. 资源类型选择（ResourcePicker，依赖自动补齐）。
7. 资源配额（CPU/Memory requests/limits）。
8. HTTP 探针开关与路径。
9. 环境变量（EnvVarEditor）。
10. 密钥（EnvVarEditor；填写后后端自动补 Secret）。
11. 生成按钮（loading；仅 canGenerate 为 true 时可用）。

右侧：

- 依赖解析卡片：status=empty 时显示中性空态；planning spinner；planError 告警；资源/关系/自动补充 tag；诊断列表；资源关系图。
- ResultPanel：多文件结果。

行为约定：

- 提交前只做最小前置校验（名称非空、至少一个资源、canGenerate 为 true）；其余交给后端。
- 生成后编辑任意字段 → 状态回到 planning，旧结果立即清空；生成期间修改选择会取消本次结果并提示重新生成。
- 生成成功 message.success 显示文件数；失败显示后端错误。
- 诊断颜色映射：error→error、warning→warning、其他→info。

### 6.4 ResourcePicker

Props：modelValue: string[]；Emits：update:modelValue。
逻辑：

- 目录未加载显示 spinner；加载失败显示 NAlert + 重试按钮。
- 顶部：全选、清空、已选 N / 总数。
- 按 meta.byCategory 分组；每组标题 + “全组/取消全组”。
- Tag 可勾选；切换时 emits 新数组，不直接改 props。

约定：不得硬编码分组或 Kind；分组标题来自 CATEGORY_LABELS，颜色来自 CATEGORY_COLORS。

### 6.5 ResourceGraph

Props：nodes: PlanNode[]、edges: PlanEdge[]。

实现：

- DataSet 来自 vis-data，Network 来自 vis-network。
- 节点 id 使用 Kind；label 为 Kind + 换行 + 资源名。
- 节点底色按目录 category 从 CATEGORY_COLORS 取；自动补充节点边框为橙色 #D99A4E，其余 #6E9CD3。
- 边只渲染两端都存在的节点；label 为 relation；箭头指向目标。
- physics barnesHut（gravitationalConstant=-6000，springLength=140），高度 280px。
- onMounted 渲染；watch nodes/edges deep 更新；onBeforeUnmount destroy。

约定：图只做展示，不参与依赖计算；布局变化不得改变数据。

### 6.6 ResultPanel

Props：record: ConfigRecord | null。

功能：

- 无记录显示 NEmpty。
- 记录元信息：文件数、记录 ID、格式化时间。
- 文件 tab：每个 Artifact 一个 tab（name=path，tab=path）。
- 单文件操作：复制、下载（文件名取 path）。
- 整包操作：复制全部 bundle、下载 bundle。
- 文件名规则：整包统一下载为 bundle.yaml。
- 复制优先 navigator.clipboard，失败降级 textarea + execCommand。
- 下载使用 Blob + a[download]，用后 revokeObjectURL。

约定：不在前端拆分或合并文件；后端返回什么就展示什么。

### 6.7 HistoryView

- 头部显示总数与刷新按钮。
- NDataTable 列：ID、应用、命名空间、资源列表、时间、操作（查看）。
- 操作用 h(NButton) 渲染，router.push('/history/'+id)。
- NPagination 绑定 page/totalPages。
- 错误显示 NEmpty；空列表显示“暂无生成记录”。

### 6.8 RecordDetailView

- 顶部返回历史 + 刷新。
- 概览卡：应用名、类型、记录 ID、时间、镜像、命名空间、catalog 版本、文件数、自动补充资源。
- 诊断卡：按 severity 渲染 NAlert。
- 关系图卡：复用 ResourceGraph。
- 结果卡：复用 ResultPanel。
- 请求参数卡：pre 展示 selection JSON。

### 6.9 EnvVarEditor

Props：value: EnvPair[]；Emits：update:value。
使用 NDynamicInput preset=pair，键值占位 KEY/VALUE；父组件负责把 pairs 转成对象（空 key 丢弃）。

## 7. 交互时序

~~~
App onMounted
  ├─ meta.fetchMeta() → GET /catalog
  └─ GeneratorView onMounted → generator.refreshPlan() → POST /plan
用户修改表单
  └─ watch → schedulePlan(300ms) → POST /plan → plan/diagnostics/edges
用户点击生成
  └─ generate() → POST /render → record(files/bundle/plan) → ResultPanel
切换页面
  └─ HistoryView → GET /records；详情 → GET /records/:id
~~~

## 8. 样式规范

### 8.1 设计令牌

Naive UI 主题（theme.ts）：

| 令牌 | 值 | 用途 |
| --- | --- | --- |
| primaryColor | #7DAEE1 | 主按钮、选中、链接 |
| primaryColorHover | #93BEE8 | 悬停 |
| primaryColorPressed | #6698CF | 按下 |
| bodyColor | #F5F8FC | 页面背景 |
| cardColor | #FFFFFF | 卡片 |
| borderColor | #DFE9F5 | 边框/分割线 |
| textColor1/2/3 | #2C3E5D / #44557A / #8296B3 | 标题/正文/弱化 |
| borderRadius | 8px | 统一圆角 |

CSS 变量（main.css）：--cg-primary、--cg-primary-deep、--cg-text、--cg-text-secondary、--cg-text-muted、--cg-border、--cg-bg。

### 8.2 布局与写法

- 页面统一 .page（最大宽 1120px、居中）；内容区 padding 24px。
- 页头高度 60px；品牌 Logo 34px 圆角渐变。
- 组件样式必须 scoped；全局只允许 main.css 定义布局、重置、CSS 变量。
- 间距优先 4 的倍数；卡片间距 16px。
- 复杂两列布局用 grid，并用 960px 断点降级单列。
- 不引入 Tailwind/UnoCSS；不写内联颜色，除图表节点色（来自 constants）。
- 字体栈：Inter、PingFang SC、Microsoft YaHei；代码块使用 Consolas。
- 结果代码块：背景 #F7FAFD、1px 边框、最大高度 420px、可滚动。

### 8.3 文案

- 界面统一中文，语气与后端诊断一致（“已自动补充”“依赖无法满足”）。
- 错误提示必须可操作：说明原因或下一步（如 DNS-1123、HPA 多目标）。
- 不使用 alert/confirm/prompt；统一 useMessage。

## 9. 错误处理与提示

- API 错误由 client 统一抛出 Error；页面 catch 后用 message.error 展示。
- 依赖解析错误（400/422）在 planError + diagnostics 面板展示，不弹窗。
- 生成中禁用按钮（loading）；切换类型清理旧结果。
- 资源目录加载失败：选择器内显示重试，不阻塞其他区域。

## 10. 开发规范

1. 只用 script setup + Composition API；不写 Options API。
2. 组件 props/emits 必须显式类型；禁止 any（必要时用具体联合类型）。
3. 从 Naive UI 按需 import 组件；不注册全局组件。
4. 服务端数据只通过 api/client；组件不直接 fetch。
5. 业务规则（依赖、名称、端口）不得写在前端；只做展示与最小前置校验。
6. 资源 Kind、分类不硬编码；从 meta store 读取。
7. 新页面必须使用懒加载路由。
8. 修改 API 类型时同步 api/types.ts 与 docs/BACKEND.md；字段保持 snake_case。
9. 所有新增交互必须有 loading/empty/error 三态。
10. 不提交 web/node_modules；必须提交 web/dist（构建产物）。

## 11. 常见改动 Recipe

### 11.1 增加一个表单字段

1. api/types.ts 的 AppContext 加字段（与后端一致）。
2. stores/generator.ts 加 ref、buildSelection 写入、watch 列表加入该 ref。
3. GeneratorView.vue 加输入控件（放在语义正确的位置）。
4. 若后端需要新校验/默认值，先改 docs/BACKEND.md 与后端。
5. npm run build，提交 dist。

### 11.2 增加一个页面

1. views 下新建 .vue。
2. router/index.ts 增加懒加载路由。
3. App.vue 导航增加入口（如有必要）。
4. 后端 NoRoute 已支持 history 模式，无需新增路由。
5. 构建并更新文档。

### 11.3 调整资源目录展示

- 分类标签/颜色：constants/k8s.ts 的 CATEGORY_LABELS/CATEGORY_COLORS。
- 分组顺序：CATEGORY_ORDER。
- 新增分类必须同时更新后端 catalog 的 Category 常量与前端这两个映射。

### 11.4 增加前端测试（建议）

当前无前端测试。建议：

- 单元：Vitest + @vue/test-utils，覆盖 stores（buildSelection、防抖、错误态）与 ResultPanel 文件操作。
- 端到端：Playwright，覆盖“选择 Ingress → 自动补充 → 生成 → 文件 tab → 历史详情”。
- 契约：用后端真实 JSON fixture 校验 api/types.ts 的反序列化。

## 12. 验收清单（前端）

- 首次加载能拉到 catalog，选择器分组正确。
- 修改任意表单字段 300ms 内触发 /plan，诊断与图同步更新。
- 自动补充节点有橙色边框，并在“自动补充”区域列出。
- 400/422 时显示错误与诊断，不产生 record。
- 生成成功后文件 tab 数量等于 files.length，复制/下载内容一致。
- 历史列表分页与总数正确；详情能还原 Plan 与文件。
- 960px 以下布局切为单列，右列不再 sticky。
- 构建产物已更新，二进制中不再出现旧 chunk 名。

## 13. 已知限制与后续

- 无前端自动化测试；无 E2E。
- ResultPanel chunk 偏大（Naive UI + vis-network），可动态加载图组件。
- 表单未覆盖所有资源参数（StorageClass provisioner、IngressClass controller、CronJob schedule、PVC 容量等仍是后端固定值）。
- 无 i18n、无暗色主题、无移动端专门适配（仅响应式降级）。
- 依赖解析防抖为固定 300ms，未做请求取消；快速输入时以最后一次响应为准（后端无副作用，可接受）。
