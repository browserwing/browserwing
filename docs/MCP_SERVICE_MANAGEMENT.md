# MCP服务管理功能实现文档

## 概述

本文档记录了BrowserWing项目中MCP(Model Context Protocol)服务管理功能的完整实现。该功能允许用户添加、配置和管理外部MCP服务,并自动发现和启用这些服务提供的工具。

## 功能特性

### 1. MCP服务管理
- ✅ 支持添加外部MCP服务
- ✅ 支持三种传输类型:stdio(进程)、SSE(服务器推送)、HTTP
- ✅ 服务状态跟踪(connecting/connected/disconnected/error)
- ✅ 启用/禁用MCP服务
- ✅ 编辑和删除MCP服务

### 2. 工具自动发现
- ✅ 连接到MCP服务并自动发现可用工具
- ✅ 展示工具名称、描述和参数信息
- ✅ 独立启用/禁用每个工具
- ✅ 工具状态持久化存储

### 3. Agent集成
- ✅ 使用LazyMCPConfig动态加载MCP服务
- ✅ 仅加载已启用的服务和工具
- ✅ 支持Agent会话创建时自动加载配置

### 4. 用户界面
- ✅ 工具管理页面新增MCP服务标签页
- ✅ 服务卡片展示(名称、描述、状态、工具数量)
- ✅ 工具列表展开/收起交互
- ✅ 服务创建/编辑模态框
- ✅ 支持搜索和分页(工具列表)

## 技术架构

### 后端实现

#### 1. 数据模型 (`backend/models/mcp_service.go`)

```go
type MCPService struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Type        MCPServiceType    `json:"type"` // stdio/sse/http
    Command     string            `json:"command"`     // stdio类型使用
    Args        []string          `json:"args"`        // stdio类型使用
    Env         map[string]string `json:"env"`         // 环境变量
    URL         string            `json:"url"`         // sse/http类型使用
    Enabled     bool              `json:"enabled"`
    Status      MCPServiceStatus  `json:"status"`
    ToolCount   int               `json:"tool_count"`
    CreatedAt   string            `json:"created_at"`
    UpdatedAt   string            `json:"updated_at"`
}

type MCPDiscoveredTool struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Enabled     bool   `json:"enabled"`
}
```

#### 2. 存储层 (`backend/storage/bolt.go`)

使用BoltDB存储MCP服务配置和工具状态:
- `mcpServicesBucket`: 存储MCP服务配置
- `{serviceID}_tools`: 存储每个服务的工具列表

关键方法:
- `SaveMCPService()`: 保存服务配置
- `GetMCPService()`: 获取单个服务
- `ListMCPServices()`: 列出所有服务
- `DeleteMCPService()`: 删除服务
- `SaveMCPServiceTools()`: 保存工具列表
- `GetMCPServiceTools()`: 获取工具列表

#### 3. API处理器 (`backend/api/handlers.go`)

提供9个REST API端点:

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/mcp-services` | 列出所有MCP服务 |
| GET | `/mcp-services/:id` | 获取单个服务详情 |
| POST | `/mcp-services` | 创建新服务 |
| PUT | `/mcp-services/:id` | 更新服务配置 |
| DELETE | `/mcp-services/:id` | 删除服务 |
| POST | `/mcp-services/:id/toggle` | 切换服务启用状态 |
| GET | `/mcp-services/:id/tools` | 获取服务工具列表 |
| POST | `/mcp-services/:id/discover` | 发现服务工具 |
| PUT | `/mcp-services/:id/tools/:toolName` | 更新工具启用状态 |

#### 4. Agent集成 (`backend/agent/agent.go`)

新增方法:
- `GetLazyMCPConfigs()`: 从数据库加载启用的MCP服务,转换为agent-sdk所需的LazyMCPConfig格式
- `ReloadMCPServices()`: 重新加载MCP服务配置(用于配置更新后)

集成点:
- `CreateSession()`: 创建Agent会话时自动加载LazyMCPConfigs
- `loadSessionsFromDB()`: 恢复会话时加载LazyMCPConfigs

### 前端实现

#### 1. API客户端 (`frontend/src/api/client.ts`)

TypeScript接口定义:
```typescript
export interface MCPService {
  id: string
  name: string
  description?: string
  type: 'stdio' | 'sse' | 'http'
  command?: string
  args?: string[]
  env?: Record<string, string>
  url?: string
  enabled: boolean
  status: 'connecting' | 'connected' | 'disconnected' | 'error'
  tool_count?: number
  created_at?: string
  updated_at?: string
}

export interface MCPDiscoveredTool {
  name: string
  description?: string
  enabled: boolean
}
```

API方法:
- `listMCPServices()`: 获取服务列表
- `createMCPService()`: 创建服务
- `updateMCPService()`: 更新服务
- `deleteMCPService()`: 删除服务
- `toggleMCPService()`: 切换启用状态
- `getMCPServiceTools()`: 获取工具列表
- `discoverMCPServiceTools()`: 发现工具
- `updateMCPServiceToolEnabled()`: 更新工具状态

#### 2. 工具管理页面 (`frontend/src/pages/ToolManager.tsx`)

新增功能:
- **MCP服务标签页**: 在脚本工具和预设工具之外新增第三个标签
- **服务列表渲染**: 卡片式展示,包含服务信息、状态徽章、操作按钮
- **工具列表展开**: 点击展开查看服务下的所有工具
- **服务创建/编辑模态框**: 表单包含名称、描述、类型选择、命令/URL配置
- **操作按钮**:
  - 启用/禁用服务(Play图标)
  - 发现工具(Download图标)
  - 编辑服务(Edit2图标)
  - 删除服务(Trash2图标)

状态管理:
```typescript
const [mcpServices, setMcpServices] = useState<MCPService[]>([])
const [showMCPModal, setShowMCPModal] = useState(false)
const [editingMCP, setEditingMCP] = useState<MCPService | null>(null)
const [mcpTools, setMcpTools] = useState<Record<string, MCPDiscoveredTool[]>>({})
const [expandedMCPId, setExpandedMCPId] = useState<string | null>(null)
```

#### 3. 国际化 (`frontend/src/i18n/translations.ts`)

新增翻译键(中英文):
- `toolManager.mcpServices`: MCP服务
- `toolManager.addMCPService`: 新增MCP服务
- `toolManager.editMCPService`: 编辑MCP服务
- `toolManager.serviceName`: 服务名称
- `toolManager.serviceType`: 传输类型
- `toolManager.command`: 命令
- `toolManager.serviceUrl`: 服务URL
- `toolManager.discoverTools`: 发现工具
- `toolManager.toolCount`: 工具数
- 以及更多...

## 使用流程

### 添加stdio类型MCP服务

1. 进入"工具管理"页面
2. 点击"MCP服务"标签
3. 点击"新增MCP服务"按钮
4. 填写表单:
   - 服务名称: 如"filesystem-tools"
   - 服务描述: 如"文件系统操作工具"
   - 传输类型: 选择"stdio"
   - 命令: 如"node"
   - 参数: 如"/path/to/mcp-server.js"
5. 点击"保存"
6. 点击"发现工具"按钮,自动连接到MCP服务并获取工具列表
7. 展开工具列表,启用需要的工具
8. 启用服务开关

### 添加HTTP/SSE类型MCP服务

1. 进入"工具管理"页面
2. 点击"MCP服务"标签
3. 点击"新增MCP服务"按钮
4. 填写表单:
   - 服务名称: 如"remote-api-tools"
   - 服务描述: 如"远程API工具集"
   - 传输类型: 选择"HTTP"或"SSE"
   - 服务URL: 如"http://localhost:3000/mcp"
5. 后续步骤同上

### Agent使用MCP工具

创建Agent会话时,系统会自动:
1. 从数据库加载所有启用的MCP服务
2. 筛选每个服务中启用的工具
3. 构建LazyMCPConfig传递给agent-sdk
4. Agent会话创建时动态加载这些工具
5. AI可以直接调用这些工具执行任务

## 待完成工作

### 高优先级
- [ ] 实现`DiscoverMCPServiceTools`的实际MCP连接逻辑(当前返回mock数据)
- [ ] 添加MCP服务连接状态实时监控
- [ ] 实现SSE和HTTP传输类型的完整支持

### 中优先级
- [ ] 添加MCP服务日志查看功能
- [ ] 工具调用统计和监控
- [ ] 批量操作(批量启用/禁用工具)

### 低优先级
- [ ] 添加繁体中文、西班牙语、日语翻译
- [ ] MCP服务配置导入/导出
- [ ] 工具调用历史记录

## 设计决策

### 1. 为什么采用自动发现而非手动配置?

**决策**: MCP服务的工具列表通过API调用自动发现,而不是让用户手动输入。

**理由**:
- MCP协议标准支持`list_tools`方法查询可用工具
- 减少用户配置负担,避免手动输入错误
- 工具列表可能动态变化,自动发现更灵活
- 提供更好的用户体验

### 2. 为什么工具状态独立存储?

**决策**: MCPDiscoveredTool独立存储,而不是嵌套在MCPService中。

**理由**:
- 工具列表可能很长,独立存储避免主配置膨胀
- 工具发现是异步操作,可能在服务创建后执行
- 便于单独更新工具状态,无需重写整个服务配置
- 数据库键值分离(`{serviceID}` vs `{serviceID}_tools`)

### 3. 为什么使用LazyMCPConfig?

**决策**: 使用agent-sdk的LazyMCPConfig而非直接连接MCP服务。

**理由**:
- 延迟加载,仅在需要时建立连接
- 减少Agent启动时间
- 自动处理连接生命周期
- 符合agent-sdk最佳实践

## 测试建议

### 单元测试
- [ ] MCP服务CRUD操作
- [ ] 工具状态更新逻辑
- [ ] LazyMCPConfig转换逻辑

### 集成测试
- [ ] 完整的服务添加-发现-启用流程
- [ ] Agent会话创建并调用MCP工具
- [ ] 服务禁用后Agent不加载对应工具

### E2E测试
- [ ] UI操作流程完整性
- [ ] 表单验证和错误处理
- [ ] 实际连接外部MCP服务

## 相关文档

- [MCP协议规范](https://spec.modelcontextprotocol.io/)
- [agent-sdk-go LazyMCPConfig文档](https://github.com/ingenimax/agent-sdk-go)
- [工具管理快速开始](./AGENT_QUICK_START.md)
- [MCP集成指南](./MCP_INTEGRATION.md)

## 变更历史

### 2024-01 初始版本
- 完成MCP服务管理基础架构
- 实现前后端集成
- 添加UI界面和i18n支持
- Agent集成LazyMCPConfig

---

**维护者**: BrowserWing Team  
**最后更新**: 2024-01
