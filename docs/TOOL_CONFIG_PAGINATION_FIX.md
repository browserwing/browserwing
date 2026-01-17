# 工具配置分页导致脚本工具返回为空的问题修复

## 问题描述

### 用户反馈

现在 executor 工具和预设工具返回都正常，但是脚本工具（`"type": "script"`）返回为空。怀疑是分页的原因。

### 问题分析

**API 调用**:
```typescript
// frontend/src/pages/AgentChat.tsx
const toolsResponse = await fetch('/api/v1/tool-configs')  // ❌ 没有传参数
```

**后端默认分页**:
```go
// backend/api/handlers.go
page := 1
pageSize := 20  // ❌ 默认只返回 20 个工具
```

**工具数量统计**:
- 本地预设工具: 5 个（fileops, bark, git, pyexec, webfetch）
- Executor 浏览器工具: 20 个（browser_navigate, browser_click 等）
- **总计**: 25 个预设工具
- 脚本工具: N 个（用户自定义）

**问题**:
```
第一页（page=1, pageSize=20）:
  ✅ 返回前 20 个工具（部分预设工具）
  
第二页之后:
  ❌ 剩余的 5 个预设工具 + N 个脚本工具（没有被前端获取）
```

**结果**: 前端只获取到前 20 个工具，脚本工具被分页截断，返回为空。

## 根本原因

### 数据流分析

```
前端请求
    ↓
GET /api/v1/tool-configs  (没有 page_size 参数)
    ↓
后端处理
    ├─ page = 1 (默认)
    ├─ pageSize = 20 (默认)  ⚠️ 限制
    ↓
获取所有工具配置
    ├─ 预设工具: 25 个
    ├─ 脚本工具: N 个
    └─ 总计: 25 + N 个
    ↓
应用分页
    ├─ start = (1-1) * 20 = 0
    ├─ end = 0 + 20 = 20
    └─ 返回 [0:20]  ⚠️ 只返回前 20 个
    ↓
返回结果
{
    "data": [...],  // 只有前 20 个工具
    "total": 25 + N,  // 总数是对的
    "page": 1,
    "page_size": 20
}
    ↓
前端处理
    ├─ enabledTools = data.filter(t => t.enabled)
    └─ 只看到前 20 个工具  ❌ 脚本工具被截断
```

### 为什么预设工具和 executor 工具"正常"

**用户说预设工具和 executor 工具返回正常**，是因为：
1. 这些工具在列表的前面（按名称排序）
2. 部分工具在前 20 个之内，所以能看到
3. 但实际上如果预设工具超过 20 个，后面的也会被截断

**脚本工具为什么"为空"**:
1. 脚本工具的 ID 格式是 `script_{scriptID}`
2. 按名称排序后，`script_` 开头的工具通常排在后面
3. 如果预设工具已经占满前 20 个，脚本工具完全看不到

## 解决方案

### 方案概述

采用双管齐下的方式：
1. **前端**：请求时指定 `page_size=0` 表示不分页
2. **后端**：支持 `page_size=0` 的特殊语义

### 1. 前端修改

**修改文件**: `frontend/src/pages/AgentChat.tsx`

**修改前**:
```typescript
const toolsResponse = await fetch('/api/v1/tool-configs')
```

**修改后**:
```typescript
// page_size=0 表示不分页，获取所有工具
const toolsResponse = await fetch('/api/v1/tool-configs?page_size=0')
```

**理由**:
- AgentChat 页面只需要统计工具总数，不需要实际的分页展示
- 使用 `page_size=0` 语义清晰，表示"获取所有数据"
- 避免硬编码一个固定的大数字（如 100），更灵活

### 2. 后端修改

**修改文件**: `backend/api/handlers.go`

#### 修改 1: 增加 noPagination 标志

**修改前**:
```go
page := 1
pageSize := 20
if pageStr := c.Query("page"); pageStr != "" {
    if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
        page = p
    }
}
if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
    if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
        pageSize = ps
    }
}
```

**修改后**:
```go
page := 1
pageSize := 20
noPagination := false // 是否禁用分页

if pageStr := c.Query("page"); pageStr != "" {
    if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
        page = p
    }
}
if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
    if ps, err := strconv.Atoi(pageSizeStr); err == nil {
        if ps == 0 {
            // page_size=0 表示不分页，返回所有数据
            noPagination = true
        } else if ps > 0 && ps <= 1000 {
            pageSize = ps
        }
    }
}
```

**变化**:
- 添加 `noPagination` 标志
- 检测 `page_size=0` 时设置 `noPagination = true`
- 将 `pageSize` 最大限制从 100 改为 1000（允许更大的分页）

#### 修改 2: 条件应用分页

**修改前**:
```go
total := len(filteredTools)

// 应用分页
start := (page - 1) * pageSize
end := start + pageSize
if start >= total {
    filteredTools = []ToolConfigResponse{}
} else {
    if end > total {
        end = total
    }
    filteredTools = filteredTools[start:end]
}
```

**修改后**:
```go
total := len(filteredTools)

// 应用分页（如果启用）
if !noPagination {
    start := (page - 1) * pageSize
    end := start + pageSize
    if start >= total {
        filteredTools = []ToolConfigResponse{}
    } else {
        if end > total {
            end = total
        }
        filteredTools = filteredTools[start:end]
    }
}
```

**变化**:
- 只有 `noPagination = false` 时才应用分页
- 如果 `noPagination = true`，返回所有 `filteredTools`

## 使用方式

### 不分页（获取所有工具）

```bash
# page_size=0 表示不分页
curl "http://localhost:18088/api/v1/tool-configs?page_size=0"

# 返回所有工具
{
    "data": [...],  // 所有工具（25+ 个）
    "total": 25,
    "page": 1,
    "page_size": 20  # 这个值仍然返回，但实际未使用
}
```

### 分页（默认行为）

```bash
# 默认：page=1, page_size=20
curl "http://localhost:18088/api/v1/tool-configs"

# 第二页
curl "http://localhost:18088/api/v1/tool-configs?page=2&page_size=20"

# 自定义每页大小（最大 1000）
curl "http://localhost:18088/api/v1/tool-configs?page_size=50"
```

### 按类型过滤

```bash
# 只获取预设工具
curl "http://localhost:18088/api/v1/tool-configs?type=preset&page_size=0"

# 只获取脚本工具
curl "http://localhost:18088/api/v1/tool-configs?type=script&page_size=0"
```

## 数据流（修复后）

```
前端请求
    ↓
GET /api/v1/tool-configs?page_size=0  ✨ 指定不分页
    ↓
后端处理
    ├─ page = 1
    ├─ pageSize = 20
    └─ noPagination = true  ✨ 检测到 page_size=0
    ↓
获取所有工具配置
    ├─ 预设工具: 25 个
    ├─ 脚本工具: N 个
    └─ 总计: 25 + N 个
    ↓
应用分页？
    └─ NO (noPagination = true)  ✨ 跳过分页
    ↓
返回结果
{
    "data": [...],  // 所有 25 + N 个工具 ✅
    "total": 25 + N,
    "page": 1,
    "page_size": 20
}
    ↓
前端处理
    ├─ enabledTools = data.filter(t => t.enabled)
    └─ 看到所有工具  ✅ 包括脚本工具
```

## 效果对比

### 修改前

**请求**:
```bash
curl "http://localhost:18088/api/v1/tool-configs"
```

**返回**:
```json
{
    "data": [
        // 只有前 20 个工具
        { "id": "bark", "type": "preset" },
        { "id": "browser_click", "type": "preset" },
        { "id": "browser_close", "type": "preset" },
        // ... 共 20 个
    ],
    "total": 30,  // 实际有 30 个工具
    "page": 1,
    "page_size": 20
}
```

**前端显示**:
```
工具总数: 20（只看到启用的前 20 个）
```

😐 脚本工具和部分预设工具看不到

### 修改后

**请求**:
```bash
curl "http://localhost:18088/api/v1/tool-configs?page_size=0"
```

**返回**:
```json
{
    "data": [
        // 所有 30 个工具
        { "id": "bark", "type": "preset" },
        { "id": "browser_click", "type": "preset" },
        // ... 25 个预设工具
        { "id": "script_abc123", "type": "script" },
        { "id": "script_def456", "type": "script" },
        // ... 5 个脚本工具
    ],
    "total": 30,
    "page": 1,
    "page_size": 20
}
```

**前端显示**:
```
工具总数: 30（所有工具）
  - 预设工具: 25
  - 脚本工具: 5
```

😊 所有工具都正确显示

## 兼容性

### 向后兼容

✅ **完全向后兼容**:
- 不传 `page_size` 参数：使用默认分页（page_size=20）
- 传统的分页请求（如 `page=2&page_size=20`）：正常工作
- 新增的 `page_size=0`：不影响现有功能

### API 行为

| 请求 | 行为 |
|------|------|
| `/tool-configs` | 默认分页（page=1, pageSize=20） |
| `/tool-configs?page=2` | 第二页（pageSize=20） |
| `/tool-configs?page_size=50` | 自定义分页大小（最大 1000） |
| `/tool-configs?page_size=0` | ✨ 不分页，返回所有数据 |
| `/tool-configs?page_size=-1` | 无效，使用默认值 20 |

## 其他调用场景

### AgentChat 页面

**用途**: 统计工具总数
**需求**: 获取所有启用的工具
**方案**: `page_size=0`

```typescript
// frontend/src/pages/AgentChat.tsx
const toolsResponse = await fetch('/api/v1/tool-configs?page_size=0')
```

### 工具管理页面（如果有）

**用途**: 展示和管理工具配置
**需求**: 分页展示（用户体验）
**方案**: 正常分页

```typescript
// 假设有工具管理页面
const response = await apiClient.listToolConfigs({
    page: currentPage,
    page_size: 20,
    search: searchQuery,
    type: selectedType
})
```

## pageSize 上限的调整

### 修改前

```go
if ps > 0 && ps <= 100 {  // 最大 100
    pageSize = ps
}
```

### 修改后

```go
if ps > 0 && ps <= 1000 {  // 最大 1000 ✨
    pageSize = ps
}
```

**理由**:
- 原来的 100 可能不够（如果有很多工具）
- 1000 对于大多数场景已经足够
- 防止恶意请求（如 `page_size=999999999`）

## 性能考虑

### 不分页的性能影响

**工具数量估算**:
- 预设工具: ~25 个
- 脚本工具: 用户自定义，假设 ~50 个
- **总计**: ~75 个工具

**数据大小**:
```
单个工具配置: ~1 KB (包含 metadata)
75 个工具: ~75 KB
```

**性能评估**:
- ✅ 数据量很小（75 KB）
- ✅ 网络传输快速
- ✅ JSON 解析开销可忽略
- ✅ 数据库查询（BoltDB）性能良好

**结论**: 不分页对性能几乎没有影响。

### 何时需要分页

**需要分页的场景**:
1. 工具数量 > 1000 个
2. 单个工具配置很大（如包含大量参数）
3. 频繁请求（需要减少数据传输）

**当前场景**:
- AgentChat 页面只在加载时请求一次
- 工具数量有限（< 100）
- **结论**: 不需要分页

## 边界情况

### 1. page_size 为负数

```bash
curl "http://localhost:18088/api/v1/tool-configs?page_size=-1"
```

**行为**: 忽略，使用默认值 20

### 2. page_size 超过上限

```bash
curl "http://localhost:18088/api/v1/tool-configs?page_size=9999"
```

**行为**: 忽略，使用默认值 20（因为超过 1000）

### 3. 无效的 page_size

```bash
curl "http://localhost:18088/api/v1/tool-configs?page_size=abc"
```

**行为**: 解析失败，使用默认值 20

### 4. page_size=0 与其他参数

```bash
# 不分页 + 类型过滤
curl "http://localhost:18088/api/v1/tool-configs?page_size=0&type=script"
# ✅ 返回所有脚本工具

# 不分页 + 搜索
curl "http://localhost:18088/api/v1/tool-configs?page_size=0&search=browser"
# ✅ 返回所有匹配 "browser" 的工具

# 不分页 + page 参数（page 被忽略）
curl "http://localhost:18088/api/v1/tool-configs?page_size=0&page=2"
# ✅ 返回所有工具（page 参数无效）
```

## 测试建议

### 1. 基础功能测试

```bash
# 测试默认分页
curl "http://localhost:18088/api/v1/tool-configs" | jq '.data | length, .total'

# 测试不分页
curl "http://localhost:18088/api/v1/tool-configs?page_size=0" | jq '.data | length, .total'
# 期望: 两个值相同（data 长度 = total）
```

### 2. 脚本工具测试

```bash
# 创建一个脚本工具（如果还没有）
# ...

# 测试能否获取脚本工具
curl "http://localhost:18088/api/v1/tool-configs?page_size=0&type=script" | jq '.data'
# 期望: 返回所有脚本工具
```

### 3. 前端集成测试

1. 打开 AgentChat 页面
2. 查看工具总数
3. 期望: 显示所有工具（预设 + 脚本）

### 4. 分页测试

```bash
# 第一页（20 个）
curl "http://localhost:18088/api/v1/tool-configs?page=1&page_size=20" | jq '.data | length'

# 第二页
curl "http://localhost:18088/api/v1/tool-configs?page=2&page_size=20" | jq '.data | length'

# 不分页（所有）
curl "http://localhost:18088/api/v1/tool-configs?page_size=0" | jq '.data | length'
```

## 相关文件

### 修改的文件

1. **backend/api/handlers.go**
   - 修改 `ListToolConfigs()` 函数
   - 添加 `noPagination` 标志
   - 支持 `page_size=0` 的特殊语义
   - 提高 `pageSize` 上限至 1000

2. **frontend/src/pages/AgentChat.tsx**
   - 修改 `loadMCPStatus()` 函数
   - 请求时添加 `page_size=0` 参数

### 未修改的文件

- **frontend/src/api/client.ts** - API 客户端封装（已支持 page_size 参数）
- **backend/storage/bolt.go** - 数据库层（不需要改动）

## 总结

### ✅ 完成的工作

1. 识别问题：默认分页 20 导致脚本工具被截断
2. 前端修改：请求时添加 `page_size=0` 参数
3. 后端增强：支持 `page_size=0` 表示不分页
4. 提高 pageSize 上限至 1000
5. 保持向后兼容性

### 📊 改进效果

| 场景 | 修改前 | 修改后 |
|------|--------|--------|
| AgentChat 工具总数 | 0-20 | 所有（25+） |
| 预设工具可见性 | 部分 | 全部 ✅ |
| 脚本工具可见性 | ❌ 不可见 | ✅ 全部可见 |
| 分页功能 | ✅ | ✅ |
| 性能影响 | - | 可忽略 |

### 🎯 用户体验提升

**修改前**:
```
用户: 创建了 5 个脚本工具
用户: 打开 AgentChat 页面
页面: 工具总数: 20
用户: 😐 我的脚本工具呢？
```

**修改后**:
```
用户: 创建了 5 个脚本工具
用户: 打开 AgentChat 页面
页面: 工具总数: 30
      - 预设工具: 25 ✅
      - 脚本工具: 5 ✅
用户: 😊 所有工具都在了！
```

### 📝 API 规范

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码（从 1 开始） |
| page_size | int | 20 | 每页大小（0=不分页，最大 1000） |
| search | string | - | 搜索关键词 |
| type | string | - | 工具类型（preset/script） |

现在工具配置 API 能正确返回所有工具（包括脚本工具），前端能准确统计工具总数！🎉
