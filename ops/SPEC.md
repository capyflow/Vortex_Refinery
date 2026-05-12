# Vortex Refinery — Ops Dashboard SPEC

## 1. Concept & Vision

一个专业的工作流编排运营后台，面向运维/开发人员。界面干净、现代，以深色主题为主，强调数据可视化与操作效率。核心场景是通过拖拽画板快速编排处理流，将插件像积木一样拼装成完整的数据处理管道。

## 2. Design Language

- **Aesthetic**: 深色技术风 (Dark Tech) — 灵感来自 Vercel/GitHub Dark，适合长时间操作
- **Color Palette**:
  - Background: `#0d1117` (主背景)
  - Surface: `#161b22` (卡片/面板)
  - Border: `#30363d`
  - Primary: `#58a6ff` (蓝，操作按钮)
  - Accent: `#238636` (绿，成功/健康)
  - Danger: `#f85149` (红，错误/失败)
  - Warning: `#d29922` (橙)
  - Text Primary: `#e6edf3`
  - Text Secondary: `#8b949e`
- **Typography**: Inter (Google Fonts), 等宽字体用于代码/ID
- **Spacing**: 8px grid, 16px/24px padding
- **Motion**: 轻微过渡 (150ms ease)，拖拽时节点有阴影提升效果

## 3. Layout & Structure

```
┌─────────────────────────────────────────────────────────┐
│  Sidebar (220px)  │  Main Content Area                   │
│  ─────────────    │  ─────────────────────────────────  │
│  Logo / Name      │  Top Bar (breadcrumb + actions)      │
│                   │                                       │
│  ○ Dashboard      │  Page Content                        │
│  ○ 插件管理        │                                       │
│  ○ 处理流编排      │                                       │
│  ○ 处理流实例      │                                       │
│  ○ Worker 状态     │                                       │
└─────────────────────────────────────────────────────────┘
```

## 4. Pages & Features

### 4.1 插件管理 (`/plugins`)
- **表格展示**: 插件名称、版本、状态（健康/不健康）、注册时间、支持的节点类型
- **操作**: 查看详情、刷新健康状态
- **状态徽章**: 🟢 健康 / 🔴 不健康 / 🟡 未知

### 4.2 处理流编排 (`/workflows/editor/:id?`)
- **左侧面板**: 插件列表（可拖拽）
- **中间画板**: Vue Flow 拖拽画布，支持：
  - 从左侧拖拽插件到画板生成节点
  - 节点之间通过拖尾连线形成依赖关系
  - 节点可删除、选中高亮
  - 画布支持缩放、平移
  - 节点右键菜单（配置、删除）
- **右侧面板**: 节点配置（选中节点后显示配置表单）
- **顶部工具栏**: 保存 / 发布 / 撤销 / 重做 / 缩放控制
- **依赖连线**: `DependsOn` 关系通过 Vue Flow 的 edges 渲染

**节点数据结构**:
```json
{
  "id": "node-uuid",
  "type": "plugin",
  "position": { "x": 100, "y": 200 },
  "data": {
    "plugin": "example-processor",
    "name": "Example Node",
    "config": { "operation": "process" }
  }
}
```

**边数据结构**:
```json
{
  "id": "edge-uuid",
  "source": "node-1",
  "target": "node-2",
  "animated": false
}
```

### 4.3 处理流列表 (`/workflows`)
- **卡片列表**: 每个处理流一张卡片，显示名称、版本、触发条件、节点数量、状态
- **操作**: 新建 / 编辑 / 删除 / 发布

### 4.4 处理流实例 (`/instances`)
- **表格**: 实例ID、工作流名称、状态（pending/running/completed/failed）、触发时间、耗时
- **操作**: 查看详情（节点执行状态）、重试

### 4.5 Dashboard (`/`)
- 统计卡片: 总插件数、总处理流数、运行中实例数、今日触发数
- 最近实例列表

## 5. Component Inventory

### 通用
- `Sidebar` — 侧边导航
- `TopBar` — 面包屑 + 页面标题 + 全局操作
- `StatusBadge` — 状态徽章
- `DataTable` — 通用表格（排序、分页）
- `EmptyState` — 空数据状态

### 处理流编排
- `WorkflowCanvas` — Vue Flow 画布主组件
- `PluginPalette` — 左侧可拖拽插件列表
- `NodeConfigPanel` — 右侧节点配置面板
- `WorkflowToolbar` — 顶部工具栏

## 6. Technical Approach

### 前端
- **Framework**: Vue 3 + Composition API + `<script setup>`
- **Build**: Vite
- **State**: Pinia
- **Routing**: Vue Router 4
- **UI**: Tailwind CSS v3
- **流程图**: `@vue-flow/core` + `@vue-flow/background` + `@vue-flow/controls`
- **HTTP**: Axios
- **部署**: 独立静态资源，`vite build` 输出到 `ops/dist/`

### 后端 REST API (Go)
需要新增以下端点，附加到现有 master 服务或独立 ops 服务：

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/plugins` | 列出所有已注册插件 |
| GET | `/api/plugins/:name` | 插件详情 |
| GET | `/api/workflows` | 列出所有处理流定义 |
| POST | `/api/workflows` | 创建处理流 |
| GET | `/api/workflows/:id` | 获取处理流详情 |
| PUT | `/api/workflows/:id` | 更新处理流 |
| DELETE | `/api/workflows/:id` | 删除处理流 |
| POST | `/api/workflows/:id/publish` | 发布处理流 |
| GET | `/api/instances` | 列出处理流实例 |
| GET | `/api/instances/:id` | 实例详情 |
| POST | `/api/instances/:id/retry` | 重试实例 |
| GET | `/api/stats` | Dashboard 统计数据 |

### 数据流
1. 前端通过 Axios 调用 Go REST API
2. Go API 读写 MongoDB（store 层已存在）
3. 处理流定义存储在 `workflows` collection
4. 实例数据存储在 `instances` collection
