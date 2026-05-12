# Vortex Refinery

事件驱动的分布式文件处理工作流引擎。

```
外部系统 ──Webhook──► Redis Streams ──事件总线──► Master (调度)
                                                      │
                                         ┌────────────┼────────────┐
                                         ▼            ▼            ▼
                                      Worker     Worker     Worker...
                                      (插件执行)   (插件执行)
```

## 核心特性

- **事件触发**：支持 Webhook 接收外部事件（文件上传、API 调用等），自动匹配工作流
- **可视化编排**：Vue 3 Ops Dashboard，拖拽式插件节点连线，无需编写 YAML
- **插件化执行**：Worker 注册插件能力，Master 按需分发任务，水平扩展
- **状态持久化**：MongoDB 存储工作流定义/实例/任务记录，Redis Streams 作消息总线
- **热更新**：处理流编辑 → 保存即刻生效，无需重启服务

## 系统架构

| 组件 | 语言 | 职责 |
|------|------|------|
| **Master** | Go | 消费事件、匹配工作流、调度任务（gRPC） |
| **Worker** | Go | 注册插件、执行任务、汇报结果 |
| **Ops Dashboard** | Vue 3 | 处理流编排、实例监控、插件管理 |
| **Redis Streams** | — | 事件总线、任务分发 |
| **MongoDB** | — | 持久化存储（工作流/实例/任务/Worker 注册表） |

## 快速开始

### 前置依赖

- Go 1.21+
- Node.js 18+ & npm
- Docker & Docker Compose（Redis + MongoDB）

### 1. 启动基础设施

```bash
# 启动 Redis 和 MongoDB
docker run -d --name vortex-redis -p 6379:6379 redis:latest redis-server --requirepass 1433223qQ
docker run -d --name vortex-mongo -p 27017:27017 mongo:7
```

### 2. 克隆并编译

```bash
git clone https://github.com/capyflow/Vortex_Refinery.git
cd Vortex_Refinery

# 编译 Master 和 Worker
go build -o bin/master ./cmd/master
go build -o bin/worker ./cmd/worker

# 安装前端依赖
cd ops && npm install && cd ..
```

### 3. 配置

编辑 `config/master.yaml`（或复制一份）：

```yaml
server:
  grpc_addr: "0.0.0.0:50051"
  http_addr: "0.0.0.0:8088"      # Ops API 监听地址

redis:
  addr: "localhost:6379"
  password: "1433223qQ"          # 修改为你的 Redis 密码
  stream_key: "vortex:events"
  task_stream_key: "vortex:tasks"

mongodb:
  uri: "mongodb://localhost:27017"
  database: "vortex_refinery"

workflows:
  dir: "./workflows"             # 处理流 JSON 文件存储目录
```

### 4. 启动服务

```bash
# 终端 1：启动 Master（REST API :8088 + gRPC :50051）
./bin/master -config config/master.yaml

# 终端 2：启动 Worker（注册到 Master）
./bin/worker -config config/master.yaml

# 终端 3：启动前端 Dashboard
cd ops && npm run dev
```

访问 `http://localhost:5173/ops` 打开 Dashboard。

### 5. 触发一个工作流

```bash
# 模拟文件上传事件
curl -X POST http://localhost:8088/adapter/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "file.uploaded",
    "payload": {
      "file_path": "/uploads/test.csv",
      "file_type": "text/csv"
    }
  }'
```

## 项目结构

```
Vortex_Refinery/
├── cmd/
│   ├── master/main.go           # Master 入口
│   └── worker/main.go           # Worker 入口
├── internal/
│   ├── api/                     # REST API Server (Ops 后端)
│   │   ├── server.go
│   │   └── handler.go
│   ├── master/
│   │   ├── master.go            # Master 主逻辑
│   │   └── dispatcher.go       # 任务分发
│   ├── worker/
│   │   ├── executor.go          # 任务执行
│   │   ├── fetcher.go          # 任务拉取
│   │   └── registrar.go        # Worker 注册
│   ├── plugin/
│   │   ├── host.go             # 插件宿主
│   │   ├── loader.go           # 插件加载
│   │   └── interface.go        # 插件接口定义
│   ├── store/                   # MongoDB 存储层
│   │   ├── workflow.go
│   │   ├── instance.go
│   │   ├── task.go
│   │   └── worker.go
│   ├── bus/                     # Redis Streams 事件总线
│   └── adapter/                # Webhook 适配器
├── pkg/
│   └── types/types.go          # 核心类型定义
├── plugins/
│   └── example/                 # 示例插件
├── ops/                         # Vue 3 Ops Dashboard
│   ├── src/
│   │   ├── pages/               # Dashboard 页面
│   │   │   ├── DashboardPage.vue
│   │   │   ├── WorkflowEditorPage.vue   # 拖拽编排画板
│   │   │   ├── WorkflowListPage.vue
│   │   │   ├── InstancesPage.vue
│   │   │   ├── PluginsPage.vue
│   │   │   └── WorkersPage.vue
│   │   ├── api/                # Axios API 层
│   │   ├── stores/             # Pinia 状态管理
│   │   └── layouts/            # 布局组件
│   └── vite.config.ts          # Vite 配置（含 API 代理）
├── config/
│   ├── config.go               # 配置加载
│   └── master.yaml             # Master 配置
└── workflows/                  # 处理流 JSON 文件存储
```

## REST API

基础 URL: `http://localhost:8088`

| 方法 | 路径 | 描述 |
|------|------|------|
| `GET` | `/health` | 健康检查 |
| `GET` | `/api/stats` | Dashboard 统计数据 |
| `GET` | `/api/workflows` | 列出所有处理流 |
| `POST` | `/api/workflows` | 新建处理流 |
| `GET` | `/api/workflows/:id` | 获取处理流详情 |
| `PUT` | `/api/workflows/:id` | 更新处理流 |
| `DELETE` | `/api/workflows/:id` | 删除处理流 |
| `POST` | `/api/workflows/:id/publish` | 发布处理流 |
| `GET` | `/api/instances` | 列出处理流实例 |
| `POST` | `/api/instances/:id/retry` | 重试实例 |
| `GET` | `/api/plugins` | 列出已注册插件 |
| `POST` | `/api/plugins/:name/refresh` | 刷新插件健康状态 |
| `GET` | `/api/workers` | 列出 Worker |
| `POST` | `/adapter/webhook` | 接收外部事件（触发工作流）|

## 工作流数据结构

```json
{
  "id": "wf-xxx-xxx",
  "name": "CSV 处理流",
  "trigger": {
    "eventType": "file.uploaded",
    "conditions": { "file_type": "text/csv" }
  },
  "nodes": [
    {
      "id": "node-1",
      "name": "读取 CSV",
      "plugin": "csv-reader",
      "dependsOn": [],
      "config": { "path": "${file_path}" }
    },
    {
      "id": "node-2",
      "name": "过滤数据",
      "plugin": "json-filter",
      "dependsOn": ["node-1"],
      "config": { "expression": "amount > 100" }
    }
  ],
  "published": true,
  "createdAt": "2025-01-01T00:00:00Z",
  "updatedAt": "2025-01-01T00:00:00Z"
}
```

## 开发

```bash
# 编译所有 Go 代码
go build ./...

# 运行前端开发服务器
cd ops && npm run dev

# 构建前端生产版本
cd ops && npm run build
```

## 技术栈

- **后端**: Go 1.21+, gRPC, Redis Streams, MongoDB
- **前端**: Vue 3, Vite, TypeScript, Tailwind CSS, Vue Flow, Pinia, Axios
- **基础设施**: Redis, MongoDB (Docker)
