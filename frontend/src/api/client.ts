import axios from 'axios'
import { convertToI18nKey } from '../utils/backendMessageConverter'

// 动态构建 API URL
const getApiBaseUrl = () => {
  // 如果是嵌入式运行（生产环境），使用相对路径
  // 这样前端会自动使用当前页面的地址和端口
  if (import.meta.env.PROD) {
    return '/api/v1'
  }

  // 开发环境：优先使用完整的 API URL
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL
  }
  
  // 开发环境：使用端口号构建（用于 Vite 代理）
  const port = import.meta.env.VITE_API_PORT || '8080'
  const host = import.meta.env.VITE_API_HOST || 'localhost'
  return `http://${host}:${port}/api/v1`
}

const API_BASE_URL = getApiBaseUrl()

const client = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 添加响应拦截器，将后端返回的消息转换为i18n键
client.interceptors.response.use(
  (response) => {
    // 检查响应数据中是否包含message字段
    if (response.data && typeof response.data === 'object') {
      if (response.data.message && typeof response.data.message === 'string') {
        response.data.message = convertToI18nKey(response.data.message)
      }
      if (response.data.error && typeof response.data.error === 'string') {
        response.data.error = convertToI18nKey(response.data.error)
      }
    }
    return response
  },
  (error) => {
    // 处理错误响应
    if (error.response && error.response.data) {
      if (error.response.data.error && typeof error.response.data.error === 'string') {
        error.response.data.error = convertToI18nKey(error.response.data.error)
      }
      if (error.response.data.message && typeof error.response.data.message === 'string') {
        error.response.data.message = convertToI18nKey(error.response.data.message)
      }
    }
    return Promise.reject(error)
  }
)

export interface Article {
  id: string
  title: string
  content: string
  edited_content: string
  fetcher_name: string
  fetched_data: string
  prompt: string
  llm_model: string
  status: string
  group: string
  tags: string[]
  created_at: string
  updated_at: string
  published_at?: string
}

export interface ArticleListItem {
  id: string
  title: string
  fetcher_name: string
  status: string
  group: string
  tags: string[]
  created_at: string
}

export interface Fetcher {
  name: string
  description: string
  params?: FetcherParam[]
}

export interface FetcherParam {
  name: string
  description: string
  field: string
  is_must: boolean
  form_type: number
  options?: string[]
}

export interface Publisher {
  name: string
  description: string
}

export interface PublisherConfig {
  publisher_name: string
  publish_params: Record<string, string>
}

export interface BrowserConfig {
  id: string
  name: string
  description: string
  url_pattern: string  // URL正则匹配模式
  user_agent: string
  use_stealth: boolean | null  // null表示使用默认值
  headless: boolean | null     // null表示使用默认值(false)
  launch_args: string[]
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface GenerateRequest {
  fetcher_name: string
  fetch_params: Record<string, string>
  prompt: string
  llm_name?: string
  
  // 新增：发布相关（支持多个发布者）
  publish_immediately?: boolean
  publishers?: PublisherConfig[]
  
  // 新增：定时任务相关
  is_scheduled?: boolean
  schedule_type?: 'once' | 'daily'
  schedule_time?: string
  task_name?: string
}

export interface PublishRequest {
  publisher_name: string
  params: Record<string, string>
}

export interface PublishResult {
  success: boolean
  platform: string
  url: string
  message: string
  publish_id: string
}

export interface LLMConfig {
  name: string
  provider: string
  model: string
}

export interface Prompt {
  id: string
  name: string
  description: string
  content: string
  type: 'system' | 'custom'  // 系统预设或用户自定义
  created_at: string
  updated_at: string
}

export interface CreatePromptRequest {
  name: string
  description: string
  content: string
}

export interface UpdatePromptRequest {
  name: string
  description: string
  content: string
}

export interface ScriptAction {
  type: string
  timestamp: number
  selector: string
  xpath?: string
  value?: string
  url?: string
  duration?: number
  x?: number
  y?: number
  text?: string
  tag_name?: string
  attrs?: Record<string, string>
  // 键盘事件相关字段
  key?: string
  // 数据抓取相关字段
  extract_type?: string
  attribute_name?: string
  js_code?: string
  variable_name?: string
  extracted_data?: string
  // 文件上传相关字段
  file_paths?: string[]
  file_names?: string[]
  description?: string
  multiple?: boolean
  accept?: string
  // 滚动相关字段
  scroll_x?: number
  scroll_y?: number

  // 语义信息字段（用于自愈）
  intent?: {
    verb?: string
    object?: string
  }
  accessibility?: {
    role?: string
    name?: string
    value?: string
  }
  context?: {
    nearby_text?: string[]
    ancestor_tags?: string[]
    form_hint?: string
  }
  evidence?: {
    backend_dom_node_id?: number
    ax_node_id?: string
    confidence?: number
  }
}

export interface Script {
  id: string
  name: string
  description: string
  url: string
  actions: ScriptAction[]
  created_at: string
  updated_at: string
  tags?: string[]
  group?: string
  duration: number
  can_publish?: boolean
  can_fetch?: boolean
  is_mcp_command?: boolean
  mcp_command_name?: string
  mcp_command_description?: string
  mcp_input_schema?: Record<string, any>
}

export interface SaveScriptRequest {
  id: string
  name: string
  description: string
  url: string
  actions: ScriptAction[]
  tags?: string[]
  can_publish?: boolean
  can_fetch?: boolean
}

export interface PlayResult {
  success: boolean
  message: string
  extracted_data?: Record<string, any>
  errors?: string[]
}

export interface ScriptExecution {
  id: string
  script_id: string
  script_name: string
  start_time: string
  end_time: string
  duration: number
  success: boolean
  message: string
  error_msg: string
  total_steps: number
  success_steps: number
  failed_steps: number
  extracted_data?: Record<string, any>
  video_path?: string  // 录制视频路径
  created_at: string
}

export interface RecordingConfig {
  id: string
  enabled: boolean
  frame_rate: number
  quality: number
  format: string
  output_dir: string
  created_at: string
  updated_at: string
}

export interface ToolConfig {
  id: string
  name: string
  type: 'preset' | 'script'
  description: string
  enabled: boolean
  parameters: Record<string, any>
  script_id?: string
  created_at: string
  updated_at: string
}

export interface PresetToolParameterSchema {
  name: string
  type: string
  description: string
  required: boolean
  default?: string
}

export interface PresetToolMetadata {
  id: string
  name: string
  description: string
  parameters: PresetToolParameterSchema[]
}

export interface ToolConfigResponse extends ToolConfig {
  metadata?: PresetToolMetadata
  script?: Script
}

export interface MCPService {
  id: string
  name: string
  description: string
  type: 'stdio' | 'sse' | 'http'
  command?: string
  args?: string[]
  url?: string
  env?: Record<string, string>
  enabled: boolean
  status: 'disconnected' | 'connecting' | 'connected' | 'error'
  tool_count: number
  last_error?: string
  created_at: string
  updated_at: string
}

export interface MCPDiscoveredTool {
  name: string
  description: string
  enabled: boolean
  schema: Record<string, any>
}

export interface Task {
  id: string
  name: string
  type: 'generate' | 'generate_and_publish' | 'publish'
  schedule_type: 'once' | 'daily'
  cron_spec: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  
  fetcher_name?: string
  fetch_params?: Record<string, string>
  prompt?: string
  llm_name?: string
  
  publisher_name?: string
  publish_params?: Record<string, string>
  article_id?: string
  
  execution_count: number
  last_executed_at?: string
  next_execute_at?: string
  
  created_at: string
  updated_at: string
  deleted_at?: string
}

export interface TaskListItem {
  id: string
  name: string
  type: 'generate' | 'generate_and_publish' | 'publish'
  schedule_type: 'once' | 'daily'
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  execution_count: number
  last_executed_at?: string
  next_execute_at?: string
  created_at: string
  updated_at: string
}

export interface TaskExecution {
  id: string
  task_id: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  started_at: string
  ended_at?: string
  article_id?: string
  publish_result?: string
  error?: string
  created_at: string
}

export interface LLMConfig {
  id: string
  name: string
  provider: string
  api_key: string
  model: string
  base_url?: string
  is_default: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateLLMConfigRequest {
  name: string
  provider: string
  api_key: string
  model: string
  base_url?: string
  is_default?: boolean
  is_active?: boolean
}

export interface UpdateLLMConfigRequest {
  name?: string
  provider?: string
  api_key?: string
  model?: string
  base_url?: string
  is_default?: boolean
  is_active?: boolean
}

export interface TestLLMConfigRequest {
  name: string
  provider: string
  api_key: string
  model: string
  base_url?: string
}

export interface ServerConfig {
  mcp_http_port: string
  port: string
  host: string
}

export const api = {
  // 配置相关
  getServerConfig: () =>
    client.get<ServerConfig>('/config'),

  // 文章相关
  getArticles: (params?: { group?: string; tags?: string[]; sortBy?: string; sortOrder?: string }) => 
    client.get<ArticleListItem[]>('/articles', { params }),
  
  getArticle: (id: string) => 
    client.get<Article>(`/articles/${id}`),
  
  generateArticle: (data: GenerateRequest) => 
    client.post<Article | { article?: Article; publish_result?: PublishResult; warning?: string; message?: string; task?: Task }>('/articles/generate', data),
  
  updateArticle: (id: string, data: Partial<Article>) => 
    client.put<Article>(`/articles/${id}`, data),
  
  deleteArticle: (id: string) => 
    client.delete(`/articles/${id}`),
  
  publishArticle: (id: string, data: PublishRequest) => 
    client.post<PublishResult>(`/articles/${id}/publish`, data),

  // 抓取器和发布器
  getFetchers: () => 
    client.get<Fetcher[]>('/fetchers'),
  
  getFetcherParams: (name: string) =>
    client.get<Fetcher>(`/fetchers/${name}`),
  
  getPublishers: () => 
    client.get<Publisher[]>('/publishers'),

  // LLM 相关
  getLLMs: () =>
    client.get<LLMConfig[]>('/llms'),

  // 提示词相关
  getPrompts: (excludeSystem?: boolean) =>
    client.get<{ data: Prompt[] }>('/prompts', {
      params: excludeSystem ? { exclude_system: 'true' } : {}
    }),

  getPrompt: (id: string) =>
    client.get<{ data: Prompt }>(`/prompts/${id}`),

  createPrompt: (data: CreatePromptRequest) =>
    client.post<{ data: Prompt }>('/prompts', data),

  updatePrompt: (id: string, data: UpdatePromptRequest) =>
    client.put<{ data: Prompt }>(`/prompts/${id}`, data),

  deletePrompt: (id: string) =>
    client.delete(`/prompts/${id}`),

  // 浏览器相关
  startBrowser: () =>
    client.post<{ message: string; status: any }>('/browser/start'),

  stopBrowser: () =>
    client.post<{ message: string }>('/browser/stop'),

  getBrowserStatus: () =>
    client.get<any>('/browser/status'),

  openBrowserPage: (url: string, language?: string) =>
    client.post<{ message: string; url: string }>('/browser/open', { url, language }),


  saveBrowserCookies: () =>
    client.post<{ message: string; count: number }>('/browser/cookies/save'),

  getCookies: (id: string) =>
    client.get<any>(`/cookies/${id}`),

  importBrowserCookies: (data: { cookies: any[] }) =>
    client.post<{ message: string; count: number }>('/browser/cookies/import', data),

  // 录制相关
  startRecording: () =>
    client.post<{ message: string }>('/browser/record/start'),

  stopRecording: () =>
    client.post<{ message: string; actions: ScriptAction[]; count: number }>('/browser/record/stop'),

  getRecordingStatus: () =>
    client.get<{ is_recording: boolean; start_url?: string; start_time?: string; duration?: number }>('/browser/record/status'),

  clearInPageRecordingState: () =>
    client.post<{ success: boolean }>('/browser/record/clear-state'),

  // 脚本相关
  getScripts: (params?: { page?: number; page_size?: number; group?: string; tag?: string }) =>
    client.get<{ scripts: Script[]; total: number; page: number; page_size: number }>('/scripts', { params }),

  getScript: (id: string) =>
    client.get<Script>(`/scripts/${id}`),

  saveScript: (data: SaveScriptRequest) =>
    client.post<{ message: string; script: Script }>('/scripts', data),

  createScript: (data: SaveScriptRequest) =>
    client.post<{ message: string; script: Script }>('/scripts', data),

  updateScript: (id: string, data: Partial<SaveScriptRequest>) =>
    client.put<{ message: string; script: Script }>(`/scripts/${id}`, data),

  deleteScript: (id: string) =>
    client.delete<{ message: string }>(`/scripts/${id}`),

  playScript: (id: string, params?: Record<string, string>) =>
    client.post<{ message: string; script: string; result: PlayResult }>(`/scripts/${id}/play`, { params }),

  // 脚本批量操作
  batchSetGroup: (scriptIds: string[], group: string) =>
    client.post<{ message: string; count: number }>('/scripts/batch/group', { script_ids: scriptIds, group }),

  batchAddTags: (scriptIds: string[], tags: string[]) =>
    client.post<{ message: string; count: number }>('/scripts/batch/tags', { script_ids: scriptIds, tags }),

  batchDeleteScripts: (scriptIds: string[]) =>
    client.post<{ message: string; count: number }>('/scripts/batch/delete', { script_ids: scriptIds }),

  // AI 提取相关
  generateExtractionJS: (data: { html: string; description?: string }) =>
    client.post<{ javascript: string; used_model: string; message: string }>('/browser/generate-extraction-js', data),

  // 任务相关
  listTasks: () =>
    client.get<{ data: TaskListItem[] }>('/tasks'),

  getTask: (id: string) =>
    client.get<{ data: Task }>(`/tasks/${id}`),

  deleteTask: (id: string) =>
    client.delete<{ message: string }>(`/tasks/${id}`),

  cancelTask: (id: string) =>
    client.post<{ message: string }>(`/tasks/${id}/cancel`),

  listTaskExecutions: (taskId: string) =>
    client.get<{ data: TaskExecution[] }>(`/tasks/${taskId}/executions`),

  getTaskExecution: (executionId: string) =>
    client.get<{ data: TaskExecution }>(`/tasks/executions/${executionId}`),

  // LLM 配置管理
  listLLMConfigs: () =>
    client.get<{ configs: LLMConfig[] }>('/llm-configs'),

  getLLMConfig: (id: string) =>
    client.get<LLMConfig>(`/llm-configs/${id}`),

  createLLMConfig: (data: CreateLLMConfigRequest) =>
    client.post<LLMConfig>('/llm-configs', data),

  updateLLMConfig: (id: string, data: UpdateLLMConfigRequest) =>
    client.put<LLMConfig>(`/llm-configs/${id}`, data),

  deleteLLMConfig: (id: string) =>
    client.delete<{ message: string }>(`/llm-configs/${id}`),

  testLLMConfig: (data: TestLLMConfigRequest) =>
    client.post<{ success: boolean; message: string }>('/llm-configs/test', data),

  // 浏览器配置管理
  getBrowserConfigs: () =>
    client.get<{ configs: BrowserConfig[]; count: number }>('/browser-configs'),

  getBrowserConfig: (id: string) =>
    client.get<BrowserConfig>(`/browser-configs/${id}`),

  createBrowserConfig: (config: Omit<BrowserConfig, 'id' | 'created_at' | 'updated_at'>) =>
    client.post<{ message: string; config: BrowserConfig }>('/browser-configs', config),

  updateBrowserConfig: (id: string, config: Partial<BrowserConfig>) =>
    client.put<{ message: string; config: BrowserConfig }>(`/browser-configs/${id}`, config),

  deleteBrowserConfig: (id: string) =>
    client.delete<{ message: string }>(`/browser-configs/${id}`),

  // MCP 命令管理
  toggleScriptMCPCommand: (scriptId: string, data: {
    is_mcp_command: boolean
    mcp_command_name: string
    mcp_command_description: string
    mcp_input_schema?: Record<string, any>
  }) =>
    client.post<{ message: string; script: Script }>(`/scripts/${scriptId}/mcp`, data),

  getMCPStatus: () =>
    client.get<{ running: boolean; commands: any[]; command_count: number }>('/mcp/status'),

  listMCPCommands: () =>
    client.get<{ commands: any[]; count: number }>('/mcp/commands'),

  // 脚本执行记录
  listScriptExecutions: (params?: {
    page?: number
    page_size?: number
    script_id?: string
    search?: string
    success?: boolean
  }) =>
    client.get<{
      executions: ScriptExecution[]
      total: number
      page: number
      page_size: number
    }>('/script-executions', { params }),

  getScriptExecution: (id: string) =>
    client.get<ScriptExecution>(`/script-executions/${id}`),

  deleteScriptExecution: (id: string) =>
    client.delete<{ message: string }>(`/script-executions/${id}`),

  batchDeleteScriptExecutions: (ids: string[]) =>
    client.post<{ message: string; count: number }>('/script-executions/batch/delete', { ids }),

  // 录制配置相关
  getRecordingConfig: () => client.get<RecordingConfig>('/recording-config'),
  updateRecordingConfig: (config: RecordingConfig) => client.put('/recording-config', config),

  // 工具配置相关
  listToolConfigs: (params?: { page?: number; page_size?: number; search?: string; type?: string }) =>
    client.get<{ data: ToolConfigResponse[]; total: number; page: number; page_size: number }>('/tool-configs', { params }),
  getToolConfig: (id: string) => client.get<ToolConfig>(`/tool-configs/${id}`),
  updateToolConfig: (id: string, data: { enabled?: boolean; parameters?: Record<string, any> }) =>
    client.put<ToolConfig>(`/tool-configs/${id}`, data),
  syncToolConfigs: () => client.post<{ message: string }>('/tool-configs/sync'),

  // MCP服务相关
  listMCPServices: () => client.get<{ data: MCPService[] }>('/mcp-services'),
  getMCPService: (id: string) => client.get<MCPService>(`/mcp-services/${id}`),
  createMCPService: (data: Partial<MCPService>) =>
    client.post<{ message: string; service: MCPService }>('/mcp-services', data),
  updateMCPService: (id: string, data: Partial<MCPService>) =>
    client.put<{ message: string; service: MCPService }>(`/mcp-services/${id}`, data),
  deleteMCPService: (id: string) => client.delete<{ message: string }>(`/mcp-services/${id}`),
  toggleMCPService: (id: string, enabled: boolean) =>
    client.post<{ message: string; service: MCPService }>(`/mcp-services/${id}/toggle`, { enabled }),
  getMCPServiceTools: (id: string) =>
    client.get<{ data: MCPDiscoveredTool[] }>(`/mcp-services/${id}/tools`),
  discoverMCPServiceTools: (id: string) =>
    client.post<{ message: string; tools: MCPDiscoveredTool[] }>(`/mcp-services/${id}/discover`),
  updateMCPServiceToolEnabled: (id: string, toolName: string, enabled: boolean) =>
    client.put<{ message: string }>(`/mcp-services/${id}/tools/${toolName}`, { enabled }),

  // 通用请求方法
  request: <T = any>(method: 'GET' | 'POST' | 'PUT' | 'DELETE', url: string, data?: any) => {
    const fullUrl = url.startsWith('/') ? url : `/${url}`
    switch (method) {
      case 'GET':
        return client.get<T>(fullUrl)
      case 'POST':
        return client.post<T>(fullUrl, data)
      case 'PUT':
        return client.put<T>(fullUrl, data)
      case 'DELETE':
        return client.delete<T>(fullUrl)
      default:
        throw new Error(`Unsupported method: ${method}`)
    }
  },
}

export default api




