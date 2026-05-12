# Vortex_Refinery 产品设计文档

## 1. 项目概述

**项目名称**: Vortex_Refinery  
**项目类型**: 事件驱动的文件处理工作流引擎  
**核心定位**: 接收外部事件 → 匹配触发条件 → 执行预编排的工作流 → 输出处理结果  
**技术栈**: Go (Master/Worker)、Redis Streams (消息队列)、MongoDB (状态存储)

---

## 2. 系统架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Vortex_Refinery                                  │
│                                                                               │
│   ┌────────────┐      ┌──────────────┐      ┌─────────────────────────────┐  │
│   │  External  │      │   Webhook    │      │   Other Event Sources       │  │
│   │  Systems   │─────►│   Adapter    │      │   (Kafka/AMQP/自定义)       │  │
│   └────────────┘      └──────┬───────┘      └──────────────┬──────────────┘  │
│                               │                               │                 │
│                               └───────────────┬───────────────┘                 │
│                                           │                                     │
│                                           ▼                                     │
│                              ┌────────────────────────┐                        │
│                              │     Redis Streams      │                        │
│                              │      (事件总线)          │                        │
│                              └────────────┬───────────┘                        │
│                                          │                                     │
│          ┌───────────────────────────────┼───────────────────────────────┐    │
│          │                               │                               │    │
│          ▼                               ▼                               ▼    │
│  ┌───────────────┐              ┌───────────────┐              ┌───────────────┐│
│  │    Master     │◄────────────►│    Worker     │◄────────────►│    Worker     ││
│  │    Node       │    gRPC      │    Node 1     │              │    Node N     ││
│  │  (调度中心)    │              │  (插件执行)    │              │  (插件执行)    ││
│  └───────┬───────┘              └───────────────┘              └───────────────┘│
│          │                                                                        │
│          │                     ┌────────────────────┐                             │
│          └────────────────────►│      MongoDB       │                             │
│                                │   (状态存储)         │                             │
│                                └────────────────────┘                             │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 核心组件职责

### 3.1 Webhook Adapter
- **职责**: 接收外部 HTTP Webhook 请求，转换为标准内部 Event，写入 Redis Stream
- **输入**: HTTP POST (任意格式的外部事件)
- **输出**: `202 Accepted`，异步处理
- **事件格式**:
```json
{
  "event_id": "uuid-v4",
  "event_type": "file.uploaded",
  "timestamp": "2025-01-01T10:00:00Z",
  "payload": {
    "file_path": "/uploads/video.mp4",
    "file_type": "video/mp4",
    "file_size": 104857600,
    "metadata": {
      "user_id": "user-123",
      "bucket": "input"
    }
  }
}
```

### 3.2 Event Bus (Redis Streams)
- **职责**: 统一事件入口，支持多消费者组订阅，事件持久化
- **Stream Key**: `vortex:events`
- **Consumer Group**: `vortex:masters`

### 3.3 Master Node
- **职责**:
  - 消费 Redis Stream 中的 Event
  - 根据 Event 匹配触发条件，查找对应工作流定义
  - 创建 WorkflowInstance 入库
  - 解析工作流节点依赖，按序派发 Task 到 Redis Stream
  - 接收 Worker 执行结果汇报
  - 维护工作流实例状态机
- **端口**: gRPC `0.0.0.0:50051`

### 3.4 Worker Node
- **职责**:
  - 向 Master 注册插件列表（启动时 + 插件变更时）
  - 从 Redis Stream 拉取分配给自己的 Task
  - 调用本地 Plugin Host 执行插件
  - 上报执行结果给 Master
  - 支持插件热更新（重新注册即生效）
- **端口**: 无需暴露（拉取模式）

### 3.5 Plugin Host
- **职责**:
  - 管理本地插件生命周期（加载/卸载/热更新）
  - 插件发现：扫描本地插件目录，调用插件注册接口
  - 插件调用：标准化输入输出格式
  - 插件隔离：插件可调用外部工具（exec.Command），可进程隔离

### 3.6 MongoDB Collections
- `workflow_definitions` — 工作流定义（模板）
- `workflow_instances` — 工作流实例（运行时）
- `task_records` — 任务记录（节点执行历史）
- `worker_registry` — Worker 节点注册表

---

## 4. 数据模型

### 4.1 工作流定义 (WorkflowDefinition)
```json
{
  "id": "wf-001",
  "name": "视频处理工作流",
  "version": 1,
  "trigger": {
    "event_type": "file.uploaded",
    "conditions": {
      "file_type": {"$in": ["video/mp4", "video/webm"]}
    }
  },
  "nodes": [
    {
      "id": "node-1",
      "name": "视频转码",
      "plugin": "video-transcode",
      "config": {
        "output_format": "mp4",
        "bitrate": "1M"
      }
    },
    {
      "id": "node-2",
      "name": "提取缩略图",
      "plugin": "thumbnail-extract",
      "depends_on": ["node-1"],
      "config": {
        "timestamp": "00:00:05",
        "format": "jpg"
      }
    },
    {
      "id": "node-3",
      "name": "上传CDN",
      "plugin": "cdn-upload",
      "depends_on": ["node-1", "node-2"],
      "config": {
        "bucket": "output"
      }
    }
  ],
  "on_complete": {
    "action": "webhook",
    "config": {
      "url": "https://callback.example.com/complete"
    }
  },
  "on_failure": {
    "action": "webhook",
    "config": {
      "url": "https://callback.example.com/failed"
    }
  }
}
```

### 4.2 工作流实例 (WorkflowInstance)
```json
{
  "id": "wfi-uuid",
  "workflow_id": "wf-001",
  "status": "running",
  "event": { ... },
  "current_nodes": ["node-1"],
  "completed_nodes": [],
  "failed_nodes": [],
  "created_at": "2025-01-01T10:00:00Z",
  "updated_at": "2025-01-01T10:00:05Z"
}
```

**状态机**:
```
pending → running → completed
                  ↘ failed
```

### 4.3 任务记录 (TaskRecord)
```json
{
  "id": "task-uuid",
  "instance_id": "wfi-uuid",
  "node_id": "node-1",
  "plugin": "video-transcode",
  "status": "pending",
  "input": {
    "file_path": "/uploads/video.mp4"
  },
  "output": null,
  "error": null,
  "worker_id": null,
  "created_at": "2025-01-01T10:00:00Z",
  "started_at": null,
  "completed_at": null,
  "retry_count": 0
}
```

### 4.4 Worker 注册表 (WorkerRegistry)
```json
{
  "worker_id": "worker-001",
  "plugins": ["video-transcode", "thumbnail-extract"],
  "status": "online",
  "registered_at": "2025-01-01T09:00:00Z",
  "last_heartbeat": "2025-01-01T10:00:00Z"
}
```

---

## 5. 接口设计

### 5.1 Worker → Master

#### 5.1.1 插件注册
```
POST /api/v1/workers/register
Body: {
  "worker_id": "worker-001",
  "plugins": ["video-transcode", "thumbnail-extract"]
}
Response: 200 OK
```

#### 5.1.2 心跳
```
POST /api/v1/workers/heartbeat
Body: {
  "worker_id": "worker-001"
}
Response: 200 OK
```

#### 5.1.3 任务结果上报
```
POST /api/v1/tasks/report
Body: {
  "task_id": "task-uuid",
  "status": "completed",
  "output": {
    "file_path": "/outputs/video.mp4"
  },
  "error": null
}
Response: 200 OK
```

### 5.2 Master → Worker (gRPC)
```
rpc ExecuteTask(TaskRequest) returns (TaskResponse)
message TaskRequest {
  string task_id = 1;
  string plugin = 2;
  google.protobuf.Struct config = 3;
  map<string, string> context = 4;  // 上游节点输出
}
message TaskResponse {
  string task_id = 1;
  bool success = 2;
  map<string, string> output = 3;
  string error = 4;
}
```

### 5.3 Webhook Adapter → Event Bus
```
LPUSH vortex:events '{"event_id":"uuid","event_type":"file.uploaded","payload":{...}}'
```

### 5.4 Master → Redis Stream (任务派发)
```
XADD vortex:tasks worker-{node_id} '*' task_id="task-uuid" plugin="video-transcode" config="{...}"
```

---

## 6. 插件系统

### 6.1 插件接口
```go
type Plugin interface {
    // 插件唯一标识
    Name() string
    // 执行处理
    Execute(ctx context.Context, input []byte, config json.RawMessage) ([]byte, error)
    // 健康检查
    Health() error
}
```

### 6.2 插件目录结构
```
plugins/
├── video-transcode/
│   ├── plugin.json       # 插件元信息
│   ├── main.go           # 插件实现
│   └── transcode/        # 辅助包
├── thumbnail-extract/
│   ├── plugin.json
│   └── main.go
└── cdn-upload/
    ├── plugin.json
    └── main.go
```

### 6.3 插件元信息 (plugin.json)
```json
{
  "name": "video-transcode",
  "version": "1.0.0",
  "description": "视频转码处理",
  "entry": "./main",
  "dependencies": ["ffmpeg"],
  "config_schema": {
    "output_format": {"type": "string", "required": true},
    "bitrate": {"type": "string", "required": false}
  }
}
```

### 6.4 插件自注册流程
1. Worker 扫描 `plugins/` 目录
2. 解析 `plugin.json`，获取插件名称和入口
3. 加载插件 SO 或启动子进程
4. 调用 `Register(host *PluginHost)` 向 Worker 注册
5. Worker 汇总本地插件列表，向 Master 注册
6. Master 更新 `worker_registry`

### 6.5 热更新流程
1. 新插件放入 `plugins/` 目录
2. Worker 检测到变更，重新扫描
3. 调用新插件 `Register()`
4. 向 Master 重新注册（相同 worker_id，覆盖）
5. Master 下次派发任务时使用新插件列表

---

## 7. 工作流执行流程

### 7.1 完整流程时序

```
1. 外部系统 POST Webhook → Webhook Adapter
2. Webhook Adapter → LPUSH vortex:events (Event)
3. Master XREADGROUP vortex:events → 获取 Event
4. Master 查询 MongoDB，匹配 WorkflowDefinition
5. Master 创建 WorkflowInstance (status=pending)
6. Master 解析工作流节点依赖，按层序生成 Task 列表
7. Master XADD vortex:tasks 派发 Node-1 的 Task
8. Worker XREADGROUP vortex:tasks → 获取 Task
9. Worker 调用本地 plugin.Execute()
10. Worker POST /api/v1/tasks/report → Master
11. Master 更新 TaskRecord，更新 WorkflowInstance.current_nodes
12. Master 根据依赖关系，派发 Node-2 Task
13. 重复 8-12 直到所有节点完成
14. Master 执行 on_complete Webhook
15. WorkflowInstance status=completed
```

### 7.2 节点依赖拓扑排序
```
     [Node-1] ──┬──→ [Node-3]
                │
[Event] ──→ [Node-2] ──┴──→ [Node-3]
                           │
                      [Node-4]
```
派发顺序: Event → (Node-1, Node-2) 并行 → Node-3 → Node-4

---

## 8. 项目结构

```
Vortex_Refinery/
├── cmd/
│   ├── master/
│   │   └── main.go           # Master 入口
│   └── worker/
│       └── main.go           # Worker 入口
├── internal/
│   ├── adapter/              # Webhook Adapter
│   │   ├── server.go
│   │   └── handler.go
│   ├── bus/                  # Redis Streams 封装
│   │   ├── event.go
│   │   └── task.go
│   ├── master/               # Master 核心逻辑
│   │   ├── dispatcher.go    # 任务派发
│   │   ├── matcher.go       # 触发条件匹配
│   │   ├── scheduler.go     # 拓扑排序/调度
│   │   └── reporter.go      # 结果处理
│   ├── worker/               # Worker 核心逻辑
│   │   ├── fetcher.go       # 任务拉取
│   │   ├── executor.go      # 插件调用
│   │   └── registrar.go     # 插件注册
│   ├── plugin/              # 插件接口 & 加载器
│   │   ├── interface.go
│   │   ├── loader.go
│   │   └── host.go
│   └── store/               # MongoDB 操作
│       ├── mongo.go
│       ├── workflow.go
│       ├── instance.go
│       ├── task.go
│       └── worker.go
├── pkg/
│   └── types/               # 公共类型定义
│       ├── event.go
│       ├── workflow.go
│       └── task.go
├── plugins/                 # 内置插件示例
│   └── example/
├── api/                     # API 协议定义 (proto)
│   └── vortex.proto
├── config/
│   └── config.go            # 配置加载
├── go.mod
├── go.sum
└── README.md
```

---

## 9. 配置项

### 9.1 Master 配置 (config/master.yaml)
```yaml
server:
  grpc_addr: "0.0.0.0:50051"
  http_addr: "0.0.0.0:8080"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  stream_key: "vortex:events"
  task_stream_key: "vortex:tasks"

mongodb:
  uri: "mongodb://localhost:27017"
  database: "vortex_refinery"

worker:
  heartbeat_interval: 30s
  task_pull_interval: 5s
```

### 9.2 Worker 配置 (config/worker.yaml)
```yaml
master:
  grpc_addr: "localhost:50051"

redis:
  addr: "localhost:6379"
  task_stream_key: "vortex:tasks"

plugin:
  dir: "./plugins"
  reload_interval: 10s
```

---

## 10. 部署架构

### 10.1 最小化部署
```
┌─────────────────────────────────────────────────────┐
│                     单机部署                          │
│                                                     │
│  ┌──────────┐   ┌──────────┐   ┌──────────────────┐ │
│  │  Redis   │   │  MongoDB │   │  Vortex_Refinery │ │
│  └──────────┘   └──────────┘   │  Master + Worker │ │
│                                │  (混合部署)        │ │
│                                └──────────────────┘ │
└─────────────────────────────────────────────────────┘
```

### 10.2 生产部署
```
┌─────────────────────────────────────────────────────────────┐
│                        Kubernetes                           │
│                                                             │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────────┐   │
│  │   Redis     │   │   MongoDB   │   │  Vortex Master  │   │
│  │  (集群)      │   │  (集群)      │   │  (Deployment)   │   │
│  └─────────────┘   └─────────────┘   └────────┬────────┘   │
│                                                 │            │
│  ┌──────────────────────────────────────────────┼──────────┐ │
│  │                    Worker Pool                │          │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  │          │ │
│  │  │ Worker 1 │  │ Worker 2 │  │ Worker N │  │          │ │
│  │  │ (Deployment) │ (Deployment) │ (Deployment) │          │ │
│  │  └──────────┘  └──────────┘  └──────────┘               │ │
│  └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

---

## 11. 开发团队待办

### 11.1 第一阶段：核心骨架
- [ ] 项目初始化，Go Module 配置
- [ ] 配置加载器 (config/)
- [ ] MongoDB 连接 & 基础 CRUD (store/)
- [ ] Redis Streams 封装 (bus/)
- [ ] Master gRPC 服务端骨架
- [ ] Worker gRPC 客户端骨架
- [ ] Webhook Adapter HTTP 服务

### 11.2 第二阶段：核心流程
- [ ] 事件消费 → 工作流匹配逻辑 (master/matcher)
- [ ] WorkflowInstance 创建逻辑
- [ ] 拓扑排序调度器 (master/scheduler)
- [ ] 任务派发 (bus/task)
- [ ] Worker 任务拉取 & 执行
- [ ] 插件加载器 (plugin/loader)
- [ ] 任务结果上报 & 状态更新

### 11.3 第三阶段：插件系统
- [ ] 插件接口定义 (plugin/interface)
- [ ] 插件自注册机制
- [ ] 热更新支持
- [ ] 至少 3 个示例插件

### 11.4 第四阶段：完善功能
- [ ] 错误处理 & 重试机制
- [ ] 心跳 & Worker 健康检查
- [ ] on_complete / on_failure 回调
- [ ] 工作流定义 CRUD API
- [ ] 监控指标 (Prometheus)

### 11.5 第五阶段：测试 & 文档
- [ ] 单元测试 (>60% 覆盖率)
- [ ] 集成测试
- [ ] API 文档
- [ ] 部署文档

---

## 12. 风险 & 注意事项

1. **插件隔离性**: 当前设计插件与 Worker 同进程，插件崩溃可能影响 Worker。考虑后期支持 WASM 或子进程隔离。
2. **文件传递**: 当前设计中，文件通过共享存储路径传递。需要确保 Worker 节点能访问同一存储路径（S3/NFS）。
3. **分布式事务**: 多节点工作流的失败补偿机制需要后续迭代，当前版本失败即终止。
4. **规模预估**: Redis Stream 单 Stream 建议 QPS <10万，大规模需分区。
