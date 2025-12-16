import { useState, useEffect } from 'react'
import { Settings, RefreshCw, Wrench, FileCode } from 'lucide-react'
import { api, ToolConfigResponse } from '../api/client'
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

  useEffect(() => {
    loadTools()
  }, [])

  const loadTools = async () => {
    try {
      setLoading(true)
      const response = await api.listToolConfigs()
      setTools(response.data || [])
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
    <div className="border border-gray-300 dark:border-gray-700">
      {/* 顶部标题栏 */}
      <div className="border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-6 py-4">
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
      </div>

      {/* 内容区域 */}
      <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-[calc(100vh-12rem)]">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-gray-500 dark:text-gray-400">{t('common.loading')}</div>
          </div>
        ) : (
          <div className="space-y-8">
            {/* 预设工具 */}
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{t('toolManager.presetTools')}</h2>
              {presetTools.length === 0 ? (
                <div className="text-center py-8 text-gray-500 dark:text-gray-400">{t('toolManager.noTools')}</div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {presetTools.map(renderToolCard)}
                </div>
              )}
            </div>

            {/* 脚本工具 */}
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{t('toolManager.scriptTools')}</h2>
              {scriptTools.length === 0 ? (
                <div className="text-center py-8 text-gray-500 dark:text-gray-400">{t('toolManager.noTools')}</div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {scriptTools.map(renderToolCard)}
                </div>
              )}
            </div>
          </div>
        )}
      </div>

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
