import { useState, useEffect } from 'react'
import { Settings, RefreshCw, Wrench, FileCode, Search, Server, Plus, Trash2, Power, Edit2, ChevronDown } from 'lucide-react'
import { api, ToolConfigResponse, MCPService, MCPDiscoveredTool } from '../api/client'
import Toast from '../components/Toast'
import { Modal } from '../components/Modal'
import { useLanguage } from '../i18n'

export default function ToolManager() {
  const { t } = useLanguage()
  
  const [tools, setTools] = useState<ToolConfigResponse[]>([])
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' | 'info' } | null>(null)
  const [configModal, setConfigModal] = useState<{ show: boolean; tool: ToolConfigResponse | null }>({
    show: false,
    tool: null,
  })
  const [parameters, setParameters] = useState<Record<string, any>>({})
  const [activeTab, setActiveTab] = useState<'script' | 'preset' | 'mcp'>('script')
  const [searchQuery, setSearchQuery] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(10)

  // MCP服务相关state
  const [mcpServices, setMCPServices] = useState<MCPService[]>([])
  const [showMCPModal, setShowMCPModal] = useState(false)
  const [editingMCP, setEditingMCP] = useState<MCPService | null>(null)
  const [mcpTools, setMCPTools] = useState<Record<string, MCPDiscoveredTool[]>>({})
  const [expandedMCPId, setExpandedMCPId] = useState<string | null>(null)

  useEffect(() => {
    // 初始加载时同时加载工具和MCP服务
    loadTools()
    loadMCPServices()
  }, [])

  // 当切换tab时，如果数据为空则重新加载
  useEffect(() => {
    if (activeTab === 'mcp' && mcpServices.length === 0) {
      loadMCPServices()
    } else if (activeTab !== 'mcp' && tools.length === 0) {
      loadTools()
    }
  }, [activeTab])

  const loadTools = async () => {
    try {
      setLoading(true)
      const response = await api.listToolConfigs()
      setTools(response.data.data || [])
    } catch (error: any) {
      console.error('Failed to load tools:', error)
      showToast(t('error.getLLMConfigsFailed'), 'error')
      setTools([])
    } finally {
      setLoading(false)
    }
  }

  const syncTools = async () => {
    try {
      setSyncing(true)
      await api.syncToolConfigs()
      showToast(t('toolManager.syncSuccess'), 'success')
      await loadTools()
    } catch (error: any) {
      showToast(t('toolManager.syncFailed'), 'error')
    } finally {
      setSyncing(false)
    }
  }

  // MCP服务相关函数
  const loadMCPServices = async () => {
    try {
      setLoading(true)
      const response = await api.listMCPServices()
      const services = response.data.data || []
      setMCPServices(services)
      
      // 自动加载所有服务的工具列表
      const toolsData: Record<string, MCPDiscoveredTool[]> = {}
      await Promise.all(
        services.map(async (service) => {
          try {
            const toolsResponse = await api.getMCPServiceTools(service.id)
            toolsData[service.id] = toolsResponse.data.data || []
          } catch (error) {
            console.error(`Failed to load tools for service ${service.id}:`, error)
            toolsData[service.id] = []
          }
        })
      )
      setMCPTools(toolsData)
    } catch (error: any) {
      console.error('Failed to load MCP services:', error)
      showToast(t('error.getMCPServicesFailed'), 'error')
      setMCPServices([])
    } finally {
      setLoading(false)
    }
  }

  const handleCreateMCP = () => {
    setEditingMCP({
      id: '',
      name: '',
      description: '',
      type: 'stdio',
      command: '',
      args: [],
      enabled: true,
      status: 'disconnected',
      tool_count: 0,
      created_at: '',
      updated_at: '',
    } as MCPService)
    setShowMCPModal(true)
  }

  const handleEditMCP = (service: MCPService) => {
    setEditingMCP(service)
    setShowMCPModal(true)
  }

  const handleSaveMCP = async () => {
    if (!editingMCP) return

    try {
      setLoading(true)
      if (editingMCP.id) {
        await api.updateMCPService(editingMCP.id, editingMCP)
        showToast(t('mcpService.updateSuccess'), 'success')
      } else {
        // 清理不需要的字段
        const { id, created_at, updated_at, tool_count, status, ...createData } = editingMCP
        await api.createMCPService(createData)
        showToast(t('mcpService.createSuccess'), 'success')
      }
      setShowMCPModal(false)
      setEditingMCP(null)
      await loadMCPServices()

      // 等待后端自动发现工具（给点时间）
      setTimeout(async () => {
        await loadMCPServices() // 重新加载以获取更新的tool_count
      }, 2000)
    } catch (error: any) {
      const errorMsg = error.response?.data?.details || error.response?.data?.error || error.message || t('mcpService.saveFailed')
      console.error('Save MCP service error:', error.response?.data || error)
      showToast(errorMsg, 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteMCP = async (id: string) => {
    if (!confirm(t('mcpService.deleteConfirm'))) return

    try {
      await api.deleteMCPService(id)
      showToast(t('mcpService.deleteSuccess'), 'success')
      await loadMCPServices()
    } catch (error: any) {
      showToast(t('mcpService.deleteFailed'), 'error')
    }
  }

  const handleToggleMCP = async (service: MCPService) => {
    try {
      await api.toggleMCPService(service.id, !service.enabled)
      showToast(t('mcpService.toggleSuccess'), 'success')
      await loadMCPServices()
    } catch (error: any) {
      showToast(t('mcpService.toggleFailed'), 'error')
    }
  }

  const handleToggleMCPTool = async (serviceId: string, toolName: string, currentEnabled: boolean) => {
    try {
      await api.updateMCPServiceToolEnabled(serviceId, toolName, !currentEnabled)
      showToast(t('mcpService.toolToggleSuccess'), 'success')
      // 重新加载该服务的工具列表
      const response = await api.getMCPServiceTools(serviceId)
      setMCPTools(prev => ({ ...prev, [serviceId]: response.data.data || [] }))
    } catch (error: any) {
      showToast(t('mcpService.toolToggleFailed'), 'error')
    }
  }

  const toggleMCPExpand = (serviceId: string) => {
    if (expandedMCPId === serviceId) {
      setExpandedMCPId(null)
    } else {
      setExpandedMCPId(serviceId)
    }
  }

  const toggleTool = async (tool: ToolConfigResponse) => {
    try {
      await api.updateToolConfig(tool.id, { enabled: !tool.enabled })
      showToast(t('toolManager.updateSuccess'), 'success')
      await loadTools()
    } catch (error: any) {
      showToast(t('toolManager.updateFailed'), 'error')
    }
  }

  const openConfigModal = (tool: ToolConfigResponse) => {
    setConfigModal({ show: true, tool })
    setParameters(tool.parameters || {})
  }

  const closeConfigModal = () => {
    setConfigModal({ show: false, tool: null })
    setParameters({})
  }

  const saveParameters = async () => {
    if (!configModal.tool) return

    try {
      await api.updateToolConfig(configModal.tool.id, { parameters })
      showToast(t('toolManager.updateSuccess'), 'success')
      closeConfigModal()
      await loadTools()
    } catch (error: any) {
      showToast(t('toolManager.updateFailed'), 'error')
    }
  }

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
    setToast({ message, type })
  }

  // 过滤和分页
  const currentTools = (tools || []).filter(t => t.type === activeTab)
  const filteredTools = currentTools.filter(tool => {
    if (!searchQuery) return true
    const searchLower = searchQuery.toLowerCase()
    return (
      tool.name.toLowerCase().includes(searchLower) ||
      tool.description.toLowerCase().includes(searchLower)
    )
  })

  const totalPages = Math.ceil(filteredTools.length / pageSize)
  const paginatedTools = filteredTools.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize
  )

  // 当切换tab或搜索时重置到第一页
  useEffect(() => {
    setCurrentPage(1)
  }, [activeTab, searchQuery])

  const presetTools = (tools || []).filter(t => t.type === 'preset')
  const scriptTools = (tools || []).filter(t => t.type === 'script')

  const renderToolCard = (tool: ToolConfigResponse) => (
    <div
      key={tool.id}
      className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-5 hover:shadow-md transition-shadow"
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-3 mb-2">
            <div className="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg">
              {tool.type === 'preset' ? (
                <Wrench className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              ) : (
                <FileCode className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              )}
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{tool.name}</h3>
              <span className="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded">
                {t(`toolManager.toolType.${tool.type}`)}
              </span>
            </div>
          </div>
          
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">{tool.description}</p>

          {/* 脚本工具显示关联脚本信息 */}
          {tool.type === 'script' && tool.script && (
            <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
              <span className="font-medium">{t('script.title')}: </span>
              <span>{tool.script.name}</span>
            </div>
          )}

          {/* 预设工具显示可配置参数 */}
          {tool.type === 'preset' && tool.metadata && tool.metadata.parameters.length > 0 && (
            <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
              <span className="font-medium">{t('toolManager.parameterConfig')}: </span>
              <span>{tool.metadata.parameters.length} {t('toolManager.parameterName')}</span>
            </div>
          )}
        </div>

        <div className="flex flex-col items-end gap-2 ml-4">
          {/* 启用/禁用开关 */}
          <button
            onClick={() => toggleTool(tool)}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              tool.enabled ? 'bg-gray-900 dark:bg-gray-700' : 'bg-gray-300 dark:bg-gray-600'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                tool.enabled ? 'translate-x-6' : 'translate-x-1'
              }`}
            />
          </button>
          <span className={`text-xs font-medium ${tool.enabled ? 'text-green-600 dark:text-green-400' : 'text-gray-400 dark:text-gray-500'}`}>
            {tool.enabled ? t('toolManager.enabled') : t('toolManager.disabled')}
          </span>

          {/* 配置按钮 */}
          {tool.type === 'preset' && tool.metadata && tool.metadata.parameters.length > 0 && (
            <button
              onClick={() => openConfigModal(tool)}
              className="mt-2 flex items-center gap-1 px-3 py-1.5 text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
            >
              <Settings className="w-4 h-4" />
              {t('toolManager.configure')}
            </button>
          )}
        </div>
      </div>
    </div>
  )

  return (
    <div className="space-y-6 lg:space-y-8 animate-fade-in">
      {/* Header */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('toolManager.title')}</h1>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{t('toolManager.subtitle')}</p>
          </div>
          <button
            onClick={syncTools}
            disabled={syncing}
            className="flex items-center gap-2 px-4 py-2 bg-gray-900 dark:bg-gray-700 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${syncing ? 'animate-spin' : ''}`} />
            {t('toolManager.syncTools')}
          </button>
        </div>

        {/* Tab 导航 */}
        <div className="border-b border-gray-200 dark:border-gray-700">
          <div className="flex gap-8">
            <button
              onClick={() => setActiveTab('script')}
              className={`py-3 px-1 border-b-2 font-medium text-sm transition-colors ${activeTab === 'script'
                  ? 'border-gray-900 dark:border-gray-100 text-gray-900 dark:text-gray-100'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }`}
            >
              <div className="flex items-center gap-2">
                <FileCode className="w-4 h-4" />
                {t('toolManager.scriptTools')} ({scriptTools.length})
              </div>
            </button>
            <button
              onClick={() => setActiveTab('preset')}
              className={`py-3 px-1 border-b-2 font-medium text-sm transition-colors ${activeTab === 'preset'
                  ? 'border-gray-900 dark:border-gray-100 text-gray-900 dark:text-gray-100'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }`}
            >
              <div className="flex items-center gap-2">
                <Wrench className="w-4 h-4" />
                {t('toolManager.presetTools')} ({presetTools.length})
              </div>
            </button>
            <button
              onClick={() => setActiveTab('mcp')}
              className={`py-3 px-1 border-b-2 font-medium text-sm transition-colors ${activeTab === 'mcp'
                ? 'border-gray-900 dark:border-gray-100 text-gray-900 dark:text-gray-100'
                : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }`}
            >
              <div className="flex items-center gap-2">
                <Server className="w-4 h-4" />
                {t('toolManager.mcpServices')} ({mcpServices.length})
              </div>
            </button>
          </div>
        </div>

        {/* 搜索栏和操作按钮 */}
        <div className="flex items-center justify-between gap-4">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={t('common.search') || '搜索...'}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
            />
          </div>
          {activeTab === 'mcp' && (
            <button
              onClick={handleCreateMCP}
              className="flex items-center gap-2 px-4 py-2 bg-gray-900 dark:bg-gray-700 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors"
            >
              <Plus className="w-4 h-4" />
              {t('toolManager.addMCPService') || '新增MCP服务'}
            </button>
          )}
        </div>
      </div>

      {/* 内容区域 */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <div className="text-gray-500 dark:text-gray-400">{t('common.loading')}</div>
        </div>
      ) : activeTab === 'mcp' ? (
        /* MCP服务列表 */
        <>
          {mcpServices.length === 0 ? (
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-12 text-center">
              <div className="text-gray-500 dark:text-gray-400">
                {t('toolManager.noMCPServices') || '暂无MCP服务'}
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {mcpServices.map((service) => (
                <div key={service.id} className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
                          {service.name}
                        </h3>
                        <span className={`px-2 py-1 text-xs rounded-full ${service.status === 'connected'
                          ? 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-100'
                          : service.status === 'disconnected'
                            ? 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300'
                            : service.status === 'connecting'
                              ? 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-100'
                              : 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-100'
                          }`}>
                          {service.status}
                        </span>
                        <span className="px-2 py-1 text-xs rounded-full bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-100">
                          {service.type}
                        </span>
                      </div>
                      {service.description && (
                        <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                          {service.description}
                        </p>
                      )}
                      {service.last_error && (
                        <p className="text-sm text-red-600 dark:text-red-400 mb-2 font-mono">
                          {t('mcpService.lastError') || '错误'}: {service.last_error}
                        </p>
                      )}
                      <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
                        <span>{t('toolManager.toolCount') || '工具数'}: {service.tool_count || 0}</span>
                        {service.type === 'stdio' && service.command && (
                          <span className="font-mono">{service.command}</span>
                        )}
                        {(service.type === 'sse' || service.type === 'http') && service.url && (
                          <span className="font-mono truncate max-w-md">{service.url}</span>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => handleToggleMCP(service)}
                        className={`p-2 rounded-lg transition-colors ${service.enabled
                          ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-200 hover:bg-green-200 dark:hover:bg-green-800'
                          : 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'
                          }`}
                        title={service.enabled ? t('common.disable') : t('common.enable')}
                      >
                        <Power className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleEditMCP(service)}
                        className="p-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                        title={t('common.edit')}
                      >
                        <Edit2 className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDeleteMCP(service.id)}
                        className="p-2 bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-200 rounded-lg hover:bg-red-200 dark:hover:bg-red-800 transition-colors"
                        title={t('common.delete')}
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>

                  {/* 工具列表 */}
                  {mcpTools[service.id] && mcpTools[service.id].length > 0 && (
                    <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                      <button
                        onClick={() => toggleMCPExpand(service.id)}
                        className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 mb-3"
                      >
                        <ChevronDown className={`w-4 h-4 transition-transform ${expandedMCPId === service.id ? 'rotate-180' : ''}`} />
                        {t('toolManager.tools') || '工具列表'} ({mcpTools[service.id].length})
                      </button>
                      {expandedMCPId === service.id && (
                        <div className="space-y-2">
                          {mcpTools[service.id].map((tool) => (
                            <div key={tool.name} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-900 rounded-lg">
                              <div className="flex-1">
                                <div className="font-medium text-sm text-gray-900 dark:text-gray-100">
                                  {tool.name}
                                </div>
                                {tool.description && (
                                  <div className="text-xs text-gray-600 dark:text-gray-400 mt-1">
                                    {tool.description}
                                  </div>
                                )}
                              </div>
                              <button
                                onClick={() => handleToggleMCPTool(service.id, tool.name, tool.enabled)}
                                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 ${tool.enabled
                                  ? 'bg-gray-900 dark:bg-gray-700'
                                  : 'bg-gray-200 dark:bg-gray-600'
                                }`}
                                role="switch"
                                aria-checked={tool.enabled}
                              >
                                <span
                                  className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform duration-200 ${tool.enabled ? 'translate-x-6' : 'translate-x-1'
                                  }`}
                                />
                              </button>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </>
        ) : (
            /* 工具列表 */
            <>
              {paginatedTools.length === 0 ? (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-12 text-center">
                  <div className="text-gray-500 dark:text-gray-400">
                    {searchQuery ? t('common.noSearchResults') || '未找到匹配的工具' : t('toolManager.noTools')}
                  </div>
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {paginatedTools.map(renderToolCard)}
                </div>
              )}

              {/* 分页控制 */}
              {totalPages > 1 && (
                <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-4">
                  <div className="flex items-center justify-between">
                    <button
                      onClick={() => setCurrentPage(1)}
                      disabled={currentPage === 1}
                      className="px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {t('common.firstPage') || '首页'}
                    </button>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                        disabled={currentPage === 1}
                        className="px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {t('common.previous') || '上一页'}
                      </button>
                      <span className="text-sm text-gray-600 dark:text-gray-400">
                        {currentPage} / {totalPages}
                      </span>
                      <button
                        onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                        disabled={currentPage === totalPages}
                        className="px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {t('common.next') || '下一页'}
                      </button>
                </div>
                    <button
                      onClick={() => setCurrentPage(totalPages)}
                      disabled={currentPage === totalPages}
                      className="px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {t('common.lastPage') || '末页'}
                    </button>
                  </div>
                </div>
              )}
        </>
      )}

      {/* 参数配置弹窗 */}
      <Modal
        isOpen={configModal.show}
        onClose={closeConfigModal}
        title={`${t('toolManager.parameterConfig')} - ${configModal.tool?.name || ''}`}
      >
        <div className="space-y-4">
          {configModal.tool?.metadata && configModal.tool.metadata.parameters.length > 0 ? (
            configModal.tool.metadata.parameters.map((param) => (
              <div key={param.name}>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {param.description}
                  {param.required && <span className="text-red-500 ml-1">*</span>}
                  <span className="text-xs text-gray-500 dark:text-gray-400 ml-2">
                    ({param.required ? t('toolManager.parameterRequired') : t('toolManager.parameterOptional')})
                  </span>
                </label>
                <input
                  type="text"
                  value={parameters[param.name] || param.default || ''}
                  onChange={(e) => setParameters({ ...parameters, [param.name]: e.target.value })}
                  placeholder={param.default || ''}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {t('toolManager.parameterName')}: <code className="bg-gray-100 dark:bg-gray-700 px-1 py-0.5 rounded">{param.name}</code>
                </p>
              </div>
            ))
          ) : (
            <div className="text-center py-4 text-gray-500 dark:text-gray-400">{t('toolManager.noParameters')}</div>
          )}

          <div className="flex justify-end gap-3 mt-6 pt-4 border-t dark:border-gray-700">
            <button
              onClick={closeConfigModal}
              className="px-4 py-2 text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={saveParameters}
              className="px-4 py-2 bg-gray-900 dark:bg-gray-700 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors"
            >
              {t('common.save')}
            </button>
          </div>
        </div>
      </Modal>

      {/* MCP服务配置弹窗 */}
      <Modal
        isOpen={showMCPModal}
        onClose={() => {
          setShowMCPModal(false)
          setEditingMCP(null)
        }}
        title={editingMCP ? t('toolManager.editMCPService') || '编辑MCP服务' : t('toolManager.addMCPService') || '新增MCP服务'}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {t('toolManager.serviceName') || '服务名称'} <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={editingMCP?.name || ''}
              onChange={(e) => setEditingMCP(prev => prev ? { ...prev, name: e.target.value } : { id: '', name: e.target.value, type: 'stdio', enabled: false, status: 'disconnected', description: '', tool_count: 0, created_at: '', updated_at: '' })}
              placeholder={t('toolManager.serviceNamePlaceholder') || '请输入服务名称'}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {t('toolManager.serviceDescription') || '服务描述'}
            </label>
            <textarea
              value={editingMCP?.description || ''}
              onChange={(e) => setEditingMCP(prev => prev ? { ...prev, description: e.target.value } : null)}
              placeholder={t('toolManager.serviceDescriptionPlaceholder') || '请输入服务描述'}
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {t('toolManager.serviceType') || '传输类型'} <span className="text-red-500">*</span>
            </label>
            <select
              value={editingMCP?.type || 'stdio'}
              onChange={(e) => setEditingMCP(prev => prev ? { ...prev, type: e.target.value as 'stdio' | 'sse' | 'http' } : null)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
            >
              <option value="stdio">stdio (进程通信)</option>
              <option value="sse">SSE (服务器推送)</option>
              <option value="http">HTTP</option>
            </select>
          </div>

          {editingMCP?.type === 'stdio' ? (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {t('toolManager.command') || '命令'} <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={editingMCP?.command || ''}
                  onChange={(e) => setEditingMCP(prev => prev ? { ...prev, command: e.target.value } : null)}
                  placeholder="node"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {t('toolManager.commandArgs') || '参数'}
                </label>
                <input
                  type="text"
                  value={editingMCP?.args?.join(' ') || ''}
                  onChange={(e) => setEditingMCP(prev => prev ? { ...prev, args: e.target.value.split(' ').filter(Boolean) } : null)}
                  placeholder="/path/to/server.js"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {t('toolManager.commandArgsHint') || '多个参数用空格分隔'}
                </p>
              </div>
            </>
          ) : (
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('toolManager.serviceUrl') || '服务URL'} <span className="text-red-500">*</span>
              </label>
              <input
                type="url"
                value={editingMCP?.url || ''}
                onChange={(e) => setEditingMCP(prev => prev ? { ...prev, url: e.target.value } : null)}
                placeholder="http://localhost:3000"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500"
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                {t('toolManager.serviceUrlHint') || '输入MCP服务器的基础URL,例如: http://localhost:3000'}
              </p>
            </div>
          )}

          {/* 自动发现工具提示 */}
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
            <div className="flex items-start gap-2">
              <svg className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="text-sm text-blue-800 dark:text-blue-300">
                {t('mcpService.autoDiscoverHint') || '创建或更新服务后将自动发现工具'}
              </p>
            </div>
          </div>

          <div className="flex justify-end gap-3 mt-6 pt-4 border-t dark:border-gray-700">
            <button
              onClick={() => {
                setShowMCPModal(false)
                setEditingMCP(null)
              }}
              className="px-4 py-2 text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleSaveMCP}
              disabled={!editingMCP?.name || (editingMCP.type === 'stdio' ? !editingMCP.command : !editingMCP.url)}
              className="px-4 py-2 bg-gray-900 dark:bg-gray-700 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {t('common.save')}
            </button>
          </div>
        </div>
      </Modal>

      {/* Toast 提示 */}
      {toast && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast(null)}
        />
      )}
    </div>
  )
}
