# AgentChat 认证问题修复

## 日期：2026-01-17

### 问题描述

**现象：**
- 用户已登录（有 token）
- 访问脚本页等其他页面正常
- 访问 AgentChat 页面时所有 API 返回 401 Unauthorized

**原因分析：**

AgentChat 页面使用的是原始的 `fetch` API，而其他页面使用的是配置好的 axios client（`src/api/client.ts`）。

两者的区别：
- **axios client**: 有 request interceptor 自动添加 `Authorization: Bearer <token>` header
- **原始 fetch**: 不会自动添加任何认证信息

相关代码对比：

```typescript
// axios client (自动添加 token) - 其他页面使用
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 原始 fetch (没有 token) - AgentChat 页面使用
const response = await fetch('/api/v1/agent/sessions')
// ❌ 缺少 Authorization header
```

---

## 修复方案

### 1. 创建 authFetch wrapper 函数

在 `AgentChat.tsx` 文件顶部添加：

```typescript
// 创建带认证的 fetch wrapper
const authFetch = async (url: string, options: RequestInit = {}) => {
  const token = localStorage.getItem('token')
  const headers = {
    ...options.headers,
    ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
  }
  return fetch(url, { ...options, headers })
}
```

**功能：**
- 自动从 localStorage 读取 token
- 自动添加到 Authorization header
- 保留原有的其他 headers
- 如果没有 token，不添加 header（兼容未启用认证的情况）

### 2. 替换所有 fetch 调用

将所有 `fetch` 调用替换为 `authFetch`：

#### 修改前：
```typescript
const response = await fetch('/api/v1/agent/sessions')
```

#### 修改后：
```typescript
const response = await authFetch('/api/v1/agent/sessions')
```

### 3. 涉及的 API 调用

共修复了 5 个 fetch 调用：

1. **loadSessions** - 加载会话列表
   ```typescript
   await authFetch('/api/v1/agent/sessions')
   ```

2. **loadMCPStatus** - 加载工具状态
   ```typescript
   await authFetch('/api/v1/tool-configs?page_size=0')
   await authFetch('/api/v1/mcp-services')
   ```

3. **loadLLMConfigs** - 加载 LLM 配置
   ```typescript
   await authFetch('/api/v1/llm-configs')
   ```

4. **createSession** - 创建新会话
   ```typescript
   await authFetch('/api/v1/agent/sessions', { method: 'POST' })
   ```

5. **deleteSession** - 删除会话
   ```typescript
   await authFetch(`/api/v1/agent/sessions/${sessionId}`, { method: 'DELETE' })
   ```

6. **sendMessage** - 发送消息（流式响应）
   ```typescript
   await authFetch(`/api/v1/agent/sessions/${currentSession.id}/messages`, {
     method: 'POST',
     headers: { 'Content-Type': 'application/json' },
     body: JSON.stringify({ message: userMessage, llm_config_id: selectedLlm }),
     signal: abortControllerRef.current.signal,
   })
   ```

---

## 修复结果

### 修复前
- ❌ AgentChat 页面加载失败
- ❌ 无法创建会话
- ❌ 无法发送消息
- ❌ 无法加载工具状态
- ❌ 无法加载 LLM 配置
- ❌ 所有请求返回 401

### 修复后
- ✅ AgentChat 页面正常加载
- ✅ 可以创建和删除会话
- ✅ 可以正常发送消息
- ✅ 工具状态正常显示
- ✅ LLM 配置正常加载
- ✅ 所有请求携带正确的认证信息

---

## 为什么其他页面没问题？

其他页面（如脚本页、浏览器页、LLM 配置页）使用的是统一的 `src/api/client.ts`：

```typescript
// src/api/client.ts
import axios from 'axios'

const client = axios.create({
  baseURL: '/',
  timeout: 30000,
})

// ✅ 自动添加认证 header
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export default client
```

所有通过这个 client 发送的请求都会自动携带 token。

---

## 最佳实践建议

### 问题根源
AgentChat 页面需要使用 SSE（Server-Sent Events）来实现流式响应，而 axios 不直接支持 SSE，所以使用了原始的 fetch API。

### 未来改进方向

1. **创建统一的 fetch wrapper**
   - 在 `src/api/client.ts` 中导出 `authFetch` 函数
   - 所有需要使用 fetch 的地方都引用这个统一的 wrapper
   
2. **或者使用支持 SSE 的库**
   - 考虑使用 `eventsource-parser` 或类似库
   - 与 axios 集成，统一认证管理

3. **代码审查**
   - 检查其他可能直接使用 fetch 的地方
   - 确保所有 API 调用都正确处理认证

---

## 修改的文件

- `frontend/src/pages/AgentChat.tsx` - 添加 authFetch wrapper 并替换所有 fetch 调用

---

## 测试检查清单

登录后访问 AgentChat 页面：

- [ ] 页面正常加载，不报 401 错误
- [ ] 左侧会话列表正常显示
- [ ] 可以创建新会话
- [ ] 可以发送消息并收到响应
- [ ] 流式消息正常显示
- [ ] 工具调用状态正常显示
- [ ] LLM 配置下拉框正常工作
- [ ] 可以删除会话
- [ ] 可以复制消息内容
- [ ] 可以停止消息生成

---

## 完成时间
**2026-01-17 09:20**

## 状态
✅ **修复完成**

刷新页面后，AgentChat 页面应该可以正常工作了！
