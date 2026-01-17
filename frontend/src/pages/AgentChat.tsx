import { useState, useEffect, useRef } from 'react'
import { Send, Loader2, Bot, Wrench, CheckCircle2, XCircle, Trash2, MessageSquarePlus, Copy, Check, ChevronDown, StopCircle } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import Toast from '../components/Toast'
import MarkdownRenderer from '../components/MarkdownRenderer'
import { useLanguage } from '../i18n'

// 创建带认证的 fetch wrapper
const authFetch = async (url: string, options: RequestInit = {}) => {
  const token = localStorage.getItem('token')
  const headers = {
    ...options.headers,
    ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
  }
  return fetch(url, { ...options, headers })
}

// 消息类型
interface ToolCall {
  tool_name: string
  status: 'calling' | 'success' | 'error'
  message?: string
  instructions?: string  // 工具调用说明
  arguments?: Record<string, any>  // 工具调用参数
  result?: string  // 工具执行结果
  timestamp?: string  // 调用时间戳
}

interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: string
  tool_calls?: ToolCall[]
}

interface ChatSession {
  id: string
  messages: ChatMessage[]
  created_at: string
  updated_at: string
}

interface StreamChunk {
  type: 'message' | 'tool_call' | 'done' | 'error'
  content?: string
  tool_call?: ToolCall
  error?: string
  message_id?: string
}

export default function AgentChat() {
  const { t } = useLanguage()
  const navigate = useNavigate()
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [currentSession, setCurrentSession] = useState<ChatSession | null>(null)
  const [inputMessage, setInputMessage] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [showToast, setShowToast] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastType, setToastType] = useState<'success' | 'error' | 'info'>('info')
  const [mcpStatus, setMcpStatus] = useState<any>(null)
  const [llmConfigs, setLlmConfigs] = useState<any[]>([])
  const [selectedLlm, setSelectedLlm] = useState<string>('')
  const [showLlmDropdown, setShowLlmDropdown] = useState(false)
  const [copiedMessageId, setCopiedMessageId] = useState<string | null>(null)
  const [expandedToolCalls, setExpandedToolCalls] = useState<Set<string>>(new Set())
  
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const llmDropdownRef = useRef<HTMLDivElement>(null)
  const abortControllerRef = useRef<AbortController | null>(null)

  // 自动滚动到底部
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [currentSession?.messages])

  // 自动调整输入框高度
  useEffect(() => {
    const textarea = textareaRef.current
    if (!textarea) return

    // 重置高度以获取正确的 scrollHeight
    textarea.style.height = 'auto'
    
    // 计算新高度：最小1行，最大10行
    const lineHeight = 24 // leading-6 对应 24px
    const minHeight = lineHeight * 1 // 1行
    const maxHeight = lineHeight * 10 // 10行
    const newHeight = Math.min(Math.max(textarea.scrollHeight, minHeight), maxHeight)
    
    textarea.style.height = `${newHeight}px`
  }, [inputMessage])

  // 加载会话列表
  const loadSessions = async () => {
    try {
      const response = await authFetch('/api/v1/agent/sessions')
      const data = await response.json()
      const sessions = data.sessions || []
      // 按更新时间降序排列（最新的在前面）
      sessions.sort((a: ChatSession, b: ChatSession) => 
        new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
      )
      setSessions(sessions)
    } catch (error) {
      console.error('加载会话失败:', error)
    }
  }

  // 加载工具状态（包括所有类型的工具）
  const loadMCPStatus = async () => {
    try {
      // 加载工具配置（预设工具和脚本工具）
      // page_size=0 表示不分页，获取所有工具
      const toolsResponse = await authFetch('/api/v1/tool-configs?page_size=0')
      const toolsData = await toolsResponse.json()
      const enabledTools = (toolsData.data || []).filter((t: any) => t.enabled)
      
      // 加载MCP服务列表
      const mcpResponse = await authFetch('/api/v1/mcp-services')
      const mcpData = await mcpResponse.json()
      const mcpServices = mcpData.data || []
      
      // 计算所有MCP服务的工具总数
      const mcpToolCount = mcpServices.reduce((sum: number, service: any) => {
        return sum + (service.enabled ? (service.tool_count || 0) : 0)
      }, 0)
      
      // 合并统计：启用的预设/脚本工具 + 启用的MCP服务的工具
      const totalToolCount = enabledTools.length + mcpToolCount
      const hasConnected = mcpServices.some((s: any) => s.status === 'connected' && s.enabled)
      
      setMcpStatus({
        connected: hasConnected,
        tool_count: totalToolCount
      })
    } catch (error) {
      console.error('加载工具状态失败:', error)
    }
  }

  // 加载 LLM 配置列表
  const loadLLMConfigs = async () => {
    try {
      const response = await authFetch('/api/v1/llm-configs')
      const data = await response.json()
      const configs = data.configs || []
      setLlmConfigs(configs)
      
      // 设置默认选中的 LLM
      const defaultConfig = configs.find((c: any) => c.is_default && c.is_active)
      if (defaultConfig) {
        setSelectedLlm(defaultConfig.id)
      } else if (configs.length > 0) {
        setSelectedLlm(configs[0].id)
      }
    } catch (error) {
      console.error('加载 LLM 配置失败:', error)
    }
  }

  useEffect(() => {
    loadSessions()
    loadMCPStatus()
    loadLLMConfigs()
    
    // 定期刷新 MCP 状态
    const interval = setInterval(loadMCPStatus, 5000)
    return () => clearInterval(interval)
  }, [])

  // 点击外部关闭 LLM 下拉框
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (llmDropdownRef.current && !llmDropdownRef.current.contains(event.target as Node)) {
        setShowLlmDropdown(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // 创建新会话
  const createSession = async () => {
    try {
      const response = await authFetch('/api/v1/agent/sessions', {
        method: 'POST',
      })
      const data = await response.json()
      const newSession = data.session
      
      setSessions([newSession, ...sessions])
      setCurrentSession(newSession)
      
      showToastMessage(t('agentChat.sessionCreated'), 'success')
    } catch (error) {
      console.error('创建会话失败:', error)
      showToastMessage(t('agentChat.createSessionFailed'), 'error')
    }
  }

  // 删除会话
  const deleteSession = async (sessionId: string) => {
    try {
      await authFetch(`/api/v1/agent/sessions/${sessionId}`, {
        method: 'DELETE',
      })
      
      setSessions(sessions.filter(s => s.id !== sessionId))
      if (currentSession?.id === sessionId) {
        setCurrentSession(null)
      }
      
      showToastMessage(t('agentChat.sessionDeleted'), 'success')
    } catch (error) {
      console.error('删除会话失败:', error)
      showToastMessage(t('agentChat.deleteSessionFailed'), 'error')
    }
  }

  // 停止消息生成
  const stopGeneration = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
      abortControllerRef.current = null
    }
    setIsStreaming(false)
    showToastMessage(t('agentChat.generationStopped'), 'info')
  }

  // 发送消息
  const sendMessage = async () => {
    if (!inputMessage.trim() || !currentSession || isStreaming) {
      return
    }

    const userMessage = inputMessage.trim()
    setInputMessage('')
    setIsStreaming(true)

    // 创建 AbortController
    abortControllerRef.current = new AbortController()

    // 添加用户消息到界面
    const tempUserMsg: ChatMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: userMessage,
      timestamp: new Date().toISOString(),
    }

    setCurrentSession({
      ...currentSession,
      messages: [...currentSession.messages, tempUserMsg],
    })

    // 创建临时助手消息
    let assistantMsg: ChatMessage = {
      id: `temp-${Date.now()}`,
      role: 'assistant',
      content: '',
      timestamp: new Date().toISOString(),
      tool_calls: [],
    }

    try {
      const response = await authFetch(`/api/v1/agent/sessions/${currentSession.id}/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          message: userMessage,
          llm_config_id: selectedLlm || undefined, // 传递选中的 LLM ID
        }),
        signal: abortControllerRef.current.signal, // 添加 abort signal
      })

      if (!response.ok) {
        throw new Error('发送消息失败')
      }

      const reader = response.body?.getReader()
      const decoder = new TextDecoder()

      if (!reader) {
        throw new Error('无法获取响应流')
      }

      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        
        if (done) {
          break
        }

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (!line.trim() || !line.startsWith('data: ')) {
            continue
          }

          try {
            const chunk: StreamChunk = JSON.parse(line.substring(6))

            switch (chunk.type) {
              case 'message':
                // 文本内容
                // 如果收到新的 message_id，说明是新消息，需要创建新的消息对象
                if (chunk.message_id && chunk.message_id !== assistantMsg.id) {
                  // 如果已有消息ID且不同，创建新消息
                  if (assistantMsg.id) {
                    // 保存当前消息到会话中（如果还没保存的话）
                    setCurrentSession(prev => {
                      if (!prev) return prev
                      const messages = [...prev.messages]
                      const existingIndex = messages.findIndex(m => m.id === assistantMsg.id)
                      if (existingIndex === -1) {
                        messages.push({ ...assistantMsg })
                      }
                      return { ...prev, messages }
                    })
                  }
                  // 创建新的消息对象
                  assistantMsg = {
                    id: chunk.message_id,
                    role: 'assistant',
                    content: chunk.content || '',
                    timestamp: new Date().toISOString(),
                    tool_calls: [],
                  }
                } else {
                // 同一条消息，追加内容
                  if (chunk.message_id && !assistantMsg.id) {
                    assistantMsg.id = chunk.message_id
                  }
                  assistantMsg.content += chunk.content || ''
                }
                
                // 更新界面
                setCurrentSession(prev => {
                  if (!prev) {
                    console.warn('[消息更新] prev session 为 null')
                    return prev
                  }
                  const messages = [...prev.messages]
                  const lastMsg = messages[messages.length - 1]
                  
                  console.log('[消息更新] 当前消息数:', messages.length, '最后一条消息ID:', lastMsg?.id, '助手消息ID:', assistantMsg.id)
                  
                  if (lastMsg?.role === 'assistant' && lastMsg.id === assistantMsg.id) {
                    messages[messages.length - 1] = { ...assistantMsg }
                  } else {
                    messages.push({ ...assistantMsg })
                  }
                  
                  // 验证所有消息都有 id
                  const invalidMessages = messages.filter(m => !m || !m.id)
                  if (invalidMessages.length > 0) {
                    console.error('[消息更新错误] 发现无效消息:', invalidMessages)
                  }
                  
                  return {
                    ...prev,
                    messages,
                  }
                })
                break

              case 'tool_call':
                // 工具调用
                if (chunk.tool_call) {
                  console.log('收到工具调用事件:', {
                    tool_name: chunk.tool_call.tool_name,
                    status: chunk.tool_call.status,
                    instructions: chunk.tool_call.instructions,
                    arguments: chunk.tool_call.arguments,
                    result: chunk.tool_call.result ? `${chunk.tool_call.result.substring(0, 50)}...` : 'empty',
                  })

                  const existingIndex = assistantMsg.tool_calls?.findIndex(
                    tc => tc.tool_name === chunk.tool_call?.tool_name
                  ) ?? -1

                  if (existingIndex >= 0 && assistantMsg.tool_calls) {
                    assistantMsg.tool_calls[existingIndex] = chunk.tool_call
                  } else {
                    assistantMsg.tool_calls = [...(assistantMsg.tool_calls || []), chunk.tool_call]
                  }

                  // 更新界面
                  setCurrentSession(prev => {
                    if (!prev) {
                      console.warn('[工具调用更新] prev session 为 null')
                      return prev
                    }
                    const messages = [...prev.messages]
                    const lastMsg = messages[messages.length - 1]
                    
                    console.log('[工具调用更新] 工具:', chunk.tool_call?.tool_name, '当前消息数:', messages.length, '最后一条消息ID:', lastMsg?.id, '助手消息ID:', assistantMsg.id)
                    
                    // 检查是否是同一条消息（通过 id 和 role）
                    if (lastMsg?.role === 'assistant' && lastMsg.id === assistantMsg.id) {
                      messages[messages.length - 1] = { ...assistantMsg }
                    } else {
                      messages.push({ ...assistantMsg })
                    }
                    
                    // 验证所有消息都有 id
                    const invalidMessages = messages.filter(m => !m || !m.id)
                    if (invalidMessages.length > 0) {
                      console.error('[工具调用更新错误] 发现无效消息:', invalidMessages)
                    }
                    
                    return {
                      ...prev,
                      messages,
                    }
                  })
                }
                break

              case 'done':
                // 完成 - 单个消息完成，但不关闭整个流式状态
                // 流式状态会在整个连接结束时关闭
                break

              case 'error':
                // 错误
                showToastMessage(chunk.error || t('agentChat.sendMessageFailed'), 'error')
                setIsStreaming(false)
                break
            }
          } catch (e) {
            console.error('解析流数据失败:', e)
          }
        }
      }

      // 流式传输完成，关闭流式状态
      setIsStreaming(false)

      // 重新加载会话以获取完整数据
      console.log('[流式完成] 开始重新加载会话:', currentSession.id)
      const sessionResponse = await authFetch(`/api/v1/agent/sessions/${currentSession.id}`)
      const sessionData = await sessionResponse.json()
      const updatedSession = sessionData.session
      
      console.log('[流式完成] 获取到更新的会话数据:', {
        sessionId: updatedSession?.id,
        messagesCount: updatedSession?.messages?.length,
        messages: updatedSession?.messages?.map((m: any) => ({
          id: m?.id,
          role: m?.role,
          hasContent: !!m?.content,
          toolCallsCount: m?.tool_calls?.length
        }))
      })

      // 验证会话数据完整性
      if (updatedSession && updatedSession.messages) {
        // 过滤掉可能的无效消息
        updatedSession.messages = updatedSession.messages.filter((m: any) => m && m.id)
        console.log('[流式完成] 过滤后的消息数量:', updatedSession.messages.length)
      }

      setCurrentSession(updatedSession)

      // 更新会话列表中的该会话，并重新排序
      setSessions(prevSessions => {
        console.log('[流式完成] 更新会话列表，当前会话数:', prevSessions.length)
        const updatedSessions = prevSessions.map(s => 
          s && s.id === updatedSession?.id ? updatedSession : s
        ).filter(s => s && s.id) // 再次过滤，确保没有无效会话
        
        // 按更新时间降序排列
        return updatedSessions.sort((a, b) => 
          new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
        )
      })

    } catch (error: any) {
      // 如果是用户主动取消，不显示错误
      if (error.name === 'AbortError') {
        console.log('请求已取消')
        return
      }
      console.error('发送消息失败:', error)
      showToastMessage(t('agentChat.sendMessageFailed'), 'error')
      setIsStreaming(false)
    } finally {
      abortControllerRef.current = null
    }
  }

  // 显示 Toast 消息
  const showToastMessage = (message: string, type: 'success' | 'error' | 'info') => {
    setToastMessage(message)
    setToastType(type)
    setShowToast(true)
  }

  // 处理输入框回车
  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  // 格式化时间
  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  // 复制消息内容
  const copyMessage = async (content: string, messageId: string) => {
    try {
      await navigator.clipboard.writeText(content)
      setCopiedMessageId(messageId)
      setTimeout(() => setCopiedMessageId(null), 2000)
    } catch (error) {
      console.error('复制失败:', error)
      showToastMessage(t('agentChat.copyFailed'), 'error')
    }
  }

  // 切换工具调用详情展开状态
  const toggleToolCallExpand = (messageId: string, toolName: string) => {
    const key = `${messageId}-${toolName}`
    setExpandedToolCalls(prev => {
      const newSet = new Set(prev)
      if (newSet.has(key)) {
        newSet.delete(key)
      } else {
        newSet.add(key)
      }
      return newSet
    })
  }

  // 渲染工具调用状态（新版本，支持展开详情）
  const renderToolCall = (toolCall: ToolCall, messageId: string, isInline: boolean = false) => {
    console.log('渲染工具调用:', {
      tool_name: toolCall.tool_name,
      instructions: toolCall.instructions,
      has_arguments: !!toolCall.arguments,
      has_result: !!toolCall.result,
    })

    const statusIcons = {
      calling: <Loader2 className="w-4 h-4 animate-spin text-gray-600 dark:text-gray-400" />,
      success: <CheckCircle2 className="w-4 h-4 text-green-600 dark:text-green-400" />,
      error: <XCircle className="w-4 h-4 text-red-600 dark:text-red-400" />,
    }

    const key = `${messageId}-${toolCall.tool_name}`
    const isExpanded = expandedToolCalls.has(key)

    return (
      <div
        key={toolCall.tool_name}
        className={`${isInline ? 'my-3' : 'mt-2'} bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden`}
      >
        {/* 工具调用头部 */}
        <button
          onClick={() => toggleToolCallExpand(messageId, toolCall.tool_name)}
          className="w-full px-3 py-2 flex items-center gap-2 transition-colors text-left"
        >
          <Wrench className="w-4 h-4 text-gray-600 dark:text-gray-400 flex-shrink-0" />
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300 flex-shrink-0">
            {toolCall.tool_name}
          </span>
          {statusIcons[toolCall.status]}
          <ChevronDown
            className={`w-4 h-4 text-gray-500 dark:text-gray-400 ml-auto flex-shrink-0 transition-transform ${isExpanded ? 'rotate-180' : ''
              }`}
          />
        </button>

        {/* 工具调用详情（展开时显示）*/}
        {isExpanded && (
          <div className="px-3 pb-3 space-y-2 border-t border-gray-200 dark:border-gray-700 pt-2">
            {/* 参数 */}
            {toolCall.arguments && Object.keys(toolCall.arguments).length > 0 && (
              <div>
                <div className="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1">
                  {t('agentChat.toolParameters')}:
                </div>
                <pre className="text-xs bg-white dark:bg-gray-900 p-2 rounded border border-gray-200 dark:border-gray-700 overflow-x-auto">
                  {JSON.stringify(toolCall.arguments, null, 2)}
                </pre>
              </div>
            )}

            {/* 结果 */}
            {toolCall.result && (
              <div>
                <div className="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-1">
                  {t('agentChat.toolResult')}:
                </div>
                <pre className="text-xs bg-white dark:bg-gray-900 p-2 rounded border border-gray-200 dark:border-gray-700 overflow-x-auto max-h-40 overflow-y-auto">
                  {toolCall.result}
                </pre>
              </div>
            )}

            {/* 状态消息 */}
            {toolCall.message && (
              <div className="text-xs text-gray-500 dark:text-gray-400">
                {t('agentChat.status')}: {toolCall.message}
              </div>
            )}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="border border-gray-300 dark:border-gray-700 flex flex-col bg-gray-50 dark:bg-gray-900 h-[calc(100vh-11rem)] overflow-hidden">
      {/* 顶部状态栏 */}
      <div className="flex items-center justify-between px-6 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 flex-shrink-0">
        <div className="flex items-center gap-3">
          <Bot className="w-6 h-6 text-gray-900 dark:text-gray-100" />
          <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">{t('agentChat.title')}</h1>
        </div>

        <div className="flex items-center gap-4">
          {/* LLM 选择下拉框 */}
          <div className="relative" ref={llmDropdownRef}>
            <button
              onClick={() => setShowLlmDropdown(!showLlmDropdown)}
              className="flex items-center gap-2 px-3 py-1.5 bg-gray-100 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors text-sm"
              disabled={isStreaming}
            >
              <Bot className="w-4 h-4 text-gray-600 dark:text-gray-400" />
              <span className="text-gray-700 dark:text-gray-300">
                {llmConfigs.find(c => c.id === selectedLlm)?.model || t('agentChat.selectModel')}
              </span>
              <ChevronDown className="w-4 h-4 text-gray-600 dark:text-gray-400" />
            </button>

            {showLlmDropdown && (
              <div className="absolute top-full mt-2 right-0 w-64 bg-white dark:bg-gray-700 rounded-lg shadow-lg border border-gray-200 dark:border-gray-600 py-1 z-50 max-h-64 overflow-y-auto">
                {llmConfigs.filter(c => c && c.id && c.is_active).map(config => (
                  <button
                    key={config.id}
                    onClick={() => {
                      setSelectedLlm(config.id)
                      setShowLlmDropdown(false)
                    }}
                    className={`w-full text-left px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors ${
                      selectedLlm === config.id ? 'bg-gray-100 dark:bg-gray-600' : ''
                    }`}
                  >
                    <div className="font-medium text-gray-900 dark:text-gray-100">{config.model}</div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">{config.provider}</div>
                  </button>
                ))}
              </div>
            )}
          </div>
          
          {/* MCP 状态 */}
          {mcpStatus && (
            <button
              onClick={() => navigate('/tools')}
              className="flex items-center gap-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 px-3 py-2 rounded-lg transition-colors"
            >
              <div className={`w-2 h-2 rounded-full bg-green-400`} />
              <span className="text-gray-400 dark:text-gray-500">
                {t('agentChat.tools')} ({mcpStatus.tool_count || 0})
              </span>
            </button>
          )}

          {/* 新建会话按钮 */}
          <button
            onClick={createSession}
            className="flex items-center gap-2 px-4 py-2 bg-gray-900 dark:bg-gray-700 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors"
          >
            <MessageSquarePlus className="w-4 h-4" />
            <span>{t('agentChat.newSession')}</span>
          </button>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden min-h-0">
        {/* 会话列表 */}
        <div className="w-64 border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 overflow-y-auto flex-shrink-0">
          <div className="p-4">
            <h2 className="text-sm font-semibold text-gray-400 dark:text-gray-500 mb-3">{t('agentChat.sessionList')}</h2>
            <div className="space-y-2">
              {sessions.filter(s => {
                if (!s) {
                  console.warn('[会话列表] 发现 undefined 会话')
                  return false
                }
                if (!s.id) {
                  console.warn('[会话列表] 发现没有 id 的会话:', s)
                  return false
                }
                return true
              }).map(session => (
                <div
                  key={session.id}
                  className={`p-3 rounded-lg cursor-pointer transition-colors group ${
                    currentSession?.id === session.id
                    ? 'bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100'
                    : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700'
                  }`}
                  onClick={() => setCurrentSession(session)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="text-base font-medium truncate">
                        {session.messages?.[0]?.content?.substring(0, 30) || '新会话'}
                      </div>
                      <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                        {session.messages?.length || 0} {t('agentChat.messages')}
                      </div>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        deleteSession(session.id)
                      }}
                      className="opacity-0 group-hover:opacity-100 p-1 hover:bg-gray-200 dark:hover:bg-gray-600 rounded"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* 聊天区域 */}
        <div className="flex-1 flex flex-col bg-white dark:bg-gray-800 min-h-0">
          {currentSession ? (
            <>
              {/* 消息列表 */}
              <div className="flex-1 overflow-y-auto px-6 py-3 min-h-0">
                <div className="max-w-8xl mx-auto space-y-6">
                  {currentSession.messages.filter(m => {
                    if (!m) {
                      console.warn('[渲染警告] 发现 undefined 消息')
                      return false
                    }
                    if (!m.id) {
                      console.warn('[渲染警告] 发现没有 id 的消息:', m)
                      return false
                    }
                    return true
                  }).map((message, index) => (
                    <div
                      key={message.id || `temp-${index}`}
                      className={`flex ${
                        message.role === 'user' ? 'justify-end' : 'justify-start'
                      }`}
                    >
                      <div className="relative group max-w-2xl">
                        <div
                          className={`px-4 py-3 rounded-2xl ${message.role === 'user'
                            ? 'bg-gray-900 dark:bg-gray-900 text-white'
                            : 'bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-white'
                          }`}
                        >
                          {message.role === 'assistant' ? (
                            <>
                              {/* 工具调用说明和卡片（显示在内容上方）*/}
                              {message.tool_calls && message.tool_calls.length > 0 && (
                                <div className="space-y-3 mb-3">
                                  {message.tool_calls.filter(tc => {
                                    if (!tc) {
                                      console.warn('[渲染警告] 发现 undefined 工具调用，消息ID:', message.id)
                                      return false
                                    }
                                    if (!tc.tool_name) {
                                      console.warn('[渲染警告] 发现没有 tool_name 的工具调用:', tc, '消息ID:', message.id)
                                      return false
                                    }
                                    return true
                                  }).map(tc => (
                                    <div key={tc.tool_name}>
                                      {/* Instructions 显示在卡片上方 - 普通文字样式 */}
                                      {tc.instructions && (
                                        <div className="prose prose-sm dark:prose-invert max-w-none text-base">
                                          {tc.instructions}
                                        </div>
                                      )}
                                      {/* 工具调用卡片 */}
                                      {renderToolCall(tc, message.id || 'temp', true)}
                                    </div>
                                  ))}
                                </div>
                              )}

                              {/* 消息内容 - 支持 Markdown 渲染 */}
                              {message.content ? (
                                <>
                                  <MarkdownRenderer
                                    content={message.content}
                                    className="text-base"
                                  />
                                  {/* 如果正在流式传输，显示思考中的指示 */}
                                  {isStreaming && message.id === currentSession.messages[currentSession.messages.length - 1]?.id && (
                                    <div className="flex items-center gap-1.5 mt-2">
                                      <span className="w-2 h-2 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '0ms', animationDuration: '1.4s' }}></span>
                                      <span className="w-2 h-2 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '200ms', animationDuration: '1.4s' }}></span>
                                      <span className="w-2 h-2 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '400ms', animationDuration: '1.4s' }}></span>
                                    </div>
                                  )}
                                </>
                              ) : isStreaming ? (
                                <div className="flex items-center gap-1.5 py-3">
                                  <span className="w-2 h-2 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '0ms', animationDuration: '1.4s' }}></span>
                                  <span className="w-2 h-2 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '200ms', animationDuration: '1.4s' }}></span>
                                  <span className="w-2 h-2 bg-gray-400 dark:bg-gray-500 rounded-full animate-bounce" style={{ animationDelay: '400ms', animationDuration: '1.4s' }}></span>
                                  </div>
                              ) : null}
                            </>
                          ) : (
                            <div className="whitespace-pre-wrap break-words text-base">
                              {message.content}
                            </div>
                          )}

                          <div className="text-sm text-gray-400 dark:text-gray-500 mt-2">
                            {formatTime(message.timestamp)}
                          </div>
                        </div>

                        {/* 复制按钮 - 仅 AI 消息显示 */}
                        {message.role === 'assistant' && message.content && (
                          <button
                            onClick={() => copyMessage(message.content, message.id)}
                            className="absolute bottom-2 right-2 opacity-0 group-hover:opacity-100 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg p-1.5 hover:bg-gray-50 dark:hover:bg-gray-600 transition-opacity shadow-sm"
                            title={t('agentChat.copyMessage')}
                          >
                            {copiedMessageId === message.id ? (
                              <Check className="w-4 h-4 text-green-600 dark:text-green-400" />
                            ) : (
                              <Copy className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                            )}
                          </button>
                        )}
                      </div>
                    </div>
                  ))}

                  <div ref={messagesEndRef} />
                </div>
              </div>

              {/* 输入区域 - 固定在底部 */}
              <div className="px-6 py-3 bg-white dark:bg-gray-800 flex-shrink-0 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.1)] dark:shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.3)]">
                <div className="max-w-8xl mx-auto">
                  <div className="flex items-end gap-3">
                    {/* 输入框 */}
                    <div className="flex-1 flex items-end gap-3 bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-2xl px-4 py-2">
                      <textarea
                        ref={textareaRef}
                        value={inputMessage}
                        onChange={(e) => setInputMessage(e.target.value)}
                        onKeyPress={handleKeyPress}
                        placeholder={t('agentChat.inputPlaceholder')}
                        className="flex-1 bg-transparent text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500 resize-none outline-none py-2 leading-6 text-base overflow-y-auto"
                        rows={1}
                        style={{ minHeight: '24px', maxHeight: '240px' }}
                        disabled={isStreaming}
                      />
                      <button
                        onClick={isStreaming ? stopGeneration : sendMessage}
                        disabled={!isStreaming && !inputMessage.trim()}
                        className="flex-shrink-0 p-2 bg-gray-900 dark:bg-gray-700 text-white rounded-xl hover:bg-gray-800 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors mb-1"
                        title={isStreaming ? t('agentChat.stopGeneration') : t('agentChat.send')}
                      >
                        {isStreaming ? (
                          <StopCircle className="w-5 h-5" />
                        ) : (
                          <Send className="w-5 h-5" />
                        )}
                      </button>
                    </div>
                  </div>
                  <div className="text-xs text-gray-600 dark:text-gray-400 mt-2 text-center">
                    {t('agentChat.disclaimer')}
                  </div>
                </div>
              </div>
            </>
          ) : (
              <div className="flex-1 flex items-center justify-center text-gray-400 dark:text-gray-500">
              <div className="text-center">
                  <Bot className="w-16 h-16 mx-auto mb-4 opacity-30 text-gray-300 dark:text-gray-600" />
                <p className="text-lg">{t('agentChat.noSession')}</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Toast 提示 */}
      {showToast && (
        <Toast
          message={toastMessage}
          type={toastType}
          onClose={() => setShowToast(false)}
        />
      )}
    </div>
  )
}
