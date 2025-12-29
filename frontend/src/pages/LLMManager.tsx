import { useState, useEffect } from 'react'
import { Plus, Trash2, Edit, X, Star, TestTube, Loader, BookText } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { api, LLMConfig, CreateLLMConfigRequest } from '../api/client'
import Toast from '../components/Toast'
import ConfirmDialog from '../components/ConfirmDialog'
import { useLanguage } from '../i18n'

export default function LLMManager() {
  const { t } = useLanguage()
  const navigate = useNavigate()

  // LLM 提供商列表
  const PROVIDERS = [
  // 国际模型
    { value: 'openai', label: 'OpenAI', group: 'international' },
    { value: 'claude', label: 'Anthropic Claude', group: 'international' },
    { value: 'gemini', label: 'Google Gemini', group: 'international' },
    { value: 'mistral', label: 'Mistral AI', group: 'international' },
    { value: 'deepseek', label: 'DeepSeek', group: 'international' },
    { value: 'groq', label: 'Groq', group: 'international' },
    { value: 'cohere', label: 'Cohere', group: 'international' },
    { value: 'xai', label: 'xAI', group: 'international' },
    { value: 'together', label: 'together.ai', group: 'international' },
    { value: 'novita', label: 'novita.ai', group: 'international' },
    { value: 'openrouter', label: 'OpenRouter', group: 'international' },

    // 国内模型
    { value: 'qwen', label: '通义千问', group: 'domestic' },
    { value: 'siliconflow', label: '硅基流动', group: 'domestic' },
    { value: 'doubao', label: '豆包 (字节跳动)', group: 'domestic' },
    { value: 'ernie', label: '文心一言 (百度)', group: 'domestic' },
    { value: 'spark', label: '讯飞星火', group: 'domestic' },
    { value: 'chatglm', label: 'ChatGLM (智谱)', group: 'domestic' },
    { value: '360', label: '360智脑', group: 'domestic' },
    { value: 'hunyuan', label: '腾讯混元', group: 'domestic' },
    { value: 'moonshot', label: 'Moonshot AI', group: 'domestic' },
    { value: 'baichuan', label: '百川大模型', group: 'domestic' },
    { value: 'minimax', label: 'MINIMAX', group: 'domestic' },
    { value: 'yi', label: '零一万物', group: 'domestic' },
    { value: 'stepfun', label: '阶跃星辰', group: 'domestic' },
    { value: 'coze', label: 'Coze', group: 'domestic' },

    // 本地模型
    { value: 'ollama', label: 'Ollama', group: 'local' },
  ]

  const [configs, setConfigs] = useState<LLMConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [showAddForm, setShowAddForm] = useState(false)
  const [formData, setFormData] = useState<CreateLLMConfigRequest>({
    name: '',
    provider: 'openai',
    api_key: '',
    model: '',
    base_url: '',
    is_default: false,
    is_active: true,
  })
  const [testingId, setTestingId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' | 'info' } | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; configId: string | null }>({ show: false, configId: null })

  // 获取 provider 的显示名称
  const getProviderLabel = (value: string) => {
    const provider = PROVIDERS.find(p => p.value === value)
    return provider ? provider.label : value
  }

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'info') => {
    setToast({ message, type })
  }

  useEffect(() => {
    loadConfigs()
  }, [])

  const loadConfigs = async () => {
    try {
      setLoading(true)
      const response = await api.listLLMConfigs()
      setConfigs(response.data.configs || [])
    } catch (error) {
      console.error('加载 LLM 配置失败:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = async () => {
    if (!formData.name || !formData.provider || !formData.model || !formData.api_key) {
      showToast(t('llm.messages.fillRequired'), 'error')
      return
    }

    try {
      await api.createLLMConfig(formData)
      await loadConfigs()
      setShowAddForm(false)
      setFormData({
        name: '',
        provider: 'openai',
        api_key: '',
        model: '',
        base_url: '',
        is_default: false,
        is_active: true,
      })
      showToast(t('llm.messages.addSuccess'), 'success')
    } catch (error: any) {
      showToast(t('llm.messages.addError') + ': ' + (error.response?.data?.error || error.message), 'error')
    }
  }

  const handleUpdate = async (id: string, updates: Partial<LLMConfig>) => {
    try {
      await api.updateLLMConfig(id, updates)
      await loadConfigs()
      setEditingId(null)
      showToast(t('llm.messages.updateSuccess'), 'success')
    } catch (error: any) {
      showToast(t('llm.messages.updateError') + ': ' + (error.response?.data?.error || error.message), 'error')
    }
  }

  const handleDelete = async () => {
    if (!deleteConfirm.configId) return

    try {
      await api.deleteLLMConfig(deleteConfirm.configId)
      await loadConfigs()
      showToast(t('llm.messages.deleteSuccess'), 'success')
    } catch (error: any) {
      showToast(t('llm.messages.deleteError') + ': ' + (error.response?.data?.error || error.message), 'error')
    } finally {
      setDeleteConfirm({ show: false, configId: null })
    }
  }

  const handleTest = async (config: LLMConfig) => {
    setTestingId(config.id)
    try {
      const result = await api.testLLMConfig({
        name: config.name,
        provider: config.provider,
        api_key: config.api_key,
        model: config.model,
        base_url: config.base_url,
      })
      showToast(t(result.data.message), result.data.success ? 'success' : 'error')
    } catch (error: any) {
      showToast(t('llm.messages.testError') + ': ' + (error.response?.data?.error || error.message), 'error')
    } finally {
      setTestingId(null)
    }
  }

  const toggleActive = async (config: LLMConfig) => {
    await handleUpdate(config.id, { ...config, is_active: !config.is_active })
  }

  const setDefault = async (config: LLMConfig) => {
    await handleUpdate(config.id, { ...config, is_default: true })
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">{t('llm.loading')}</div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">{t('llm.title')}</h1>
          <p className="mt-1 text-sm text-gray-500">
            {t('llm.subtitle')}
          </p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => setShowAddForm(true)}
            className="inline-flex items-center px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition-colors"
          >
            <Plus className="w-4 h-4 mr-2" />
            {t('llm.addConfig')}
          </button>
          <button
            onClick={() => navigate('/prompts')}
            className="inline-flex items-center px-4 py-2 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
          >
            <BookText className="w-4 h-4 mr-2" />
            {t('nav.prompts')}
          </button>
        </div>
      </div>

      {/* Add Form */}
      {showAddForm && (
        <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-6 space-y-4">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100">{t('llm.addConfigTitle')}</h2>
            <button
              onClick={() => setShowAddForm(false)}
              className="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('llm.nameRequired')} <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100"
                placeholder={t('llm.namePlaceholder')}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('llm.providerRequired')} <span className="text-red-500">*</span>
              </label>
              <select
                value={formData.provider}
                onChange={(e) => setFormData({ ...formData, provider: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100"
              >
                <optgroup label={t('llm.groupInternational')}>
                  {PROVIDERS.filter(p => p.group === 'international' || p.group === 'domestic').map(p => (
                    <option key={p.value} value={p.value}>{p.label}</option>
                  ))}
                </optgroup>
                <optgroup label={t('llm.groupLocal')}>
                  {PROVIDERS.filter(p => p.group === 'local').map(p => (
                    <option key={p.value} value={p.value}>{p.label}</option>
                  ))}
                </optgroup>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('llm.modelRequired')} <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={formData.model}
                onChange={(e) => setFormData({ ...formData, model: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100"
                placeholder={t('llm.modelPlaceholder')}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('llm.apiKeyRequired')} <span className="text-red-500">*</span>
              </label>
              <input
                type="password"
                value={formData.api_key}
                onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100"
                placeholder={t('llm.apiKeyPlaceholder')}
              />
            </div>

            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {t('llm.baseUrl')}
              </label>
              <input
                type="text"
                value={formData.base_url}
                onChange={(e) => setFormData({ ...formData, base_url: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100"
                placeholder={t('llm.baseUrlPlaceholder')}
              />
            </div>
          </div>

          <div className="flex items-center space-x-4">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={formData.is_default}
                onChange={(e) => setFormData({ ...formData, is_default: e.target.checked })}
                className="mr-2"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">{t('llm.setDefault')}</span>
            </label>
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={formData.is_active}
                onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                className="mr-2"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">{t('llm.enable')}</span>
            </label>
          </div>

          <div className="flex justify-end space-x-2 pt-4 border-t dark:border-gray-700">
            <button
              onClick={() => setShowAddForm(false)}
              className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              onClick={handleAdd}
              className="px-4 py-2 bg-gray-900 dark:bg-gray-700 text-white rounded-md hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors"
            >
              {t('common.add')}
            </button>
          </div>
        </div>
      )}

      {/* List */}
      <div className="space-y-3">
        {configs.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <p className="text-gray-500 dark:text-gray-400">{t('llm.noConfigs')}</p>
            <button
              onClick={() => setShowAddForm(true)}
              className="mt-4 text-gray-900 dark:text-gray-100 hover:underline"
            >
              {t('llm.addFirst')}
            </button>
          </div>
        ) : (
          configs.map((config) => (
            <div
              key={config.id}
              className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4 hover:border-gray-300 dark:hover:border-gray-600 transition-colors"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-4 flex-1">
                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => toggleActive(config)}
                      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 ${config.is_active
                        ? 'bg-gray-900'
                        : 'bg-gray-200'
                      }`}
                      role="switch"
                      aria-checked={config.is_active}
                    >
                      <span
                        className={`inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform duration-200 ${config.is_active ? 'translate-x-6' : 'translate-x-1'
                        }`}
                      />
                    </button>
                  </div>

                  <div className="flex-1">
                    <div className="flex items-center space-x-2">
                      <h3 className="font-medium text-gray-900 dark:text-gray-100">{config.name}</h3>
                      <span className="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded">
                        {getProviderLabel(config.provider)}
                      </span>
                    </div>
                    <p className="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{config.model}</p>
                  </div>
                </div>

                <div className="flex items-center space-x-2">
                  {!config.is_default && config.is_active && (
                    <button
                      onClick={() => setDefault(config)}
                      className="p-2 text-gray-400 hover:text-yellow-500 transition-colors"
                      title={t('llm.setDefault')}
                    >
                      <Star className="w-4 h-4" />
                    </button>
                  )}
                  {config.is_default && (
                    <button
                      className="p-2 text-gray-400 hover:text-yellow-500 transition-colors"
                    >                    
                      <Star className="w-4 h-4 text-yellow-500 fill-yellow-500" />
                    </button>
                  )}                  
                  <button
                    onClick={() => handleTest(config)}
                    disabled={testingId === config.id}
                    className="p-2 text-gray-400 hover:text-blue-500 transition-colors disabled:opacity-50"
                    title={t('llm.testConnection')}
                  >
                    {testingId === config.id ? (
                      <Loader className="w-4 h-4 animate-spin" />
                    ) : (
                      <TestTube className="w-4 h-4" />
                    )}
                  </button>
                  <button
                    onClick={() => setEditingId(config.id)}
                    className="p-2 text-gray-400 hover:text-gray-600 transition-colors"
                    title={t('llm.edit')}
                  >
                    <Edit className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => setDeleteConfirm({ show: true, configId: config.id })}
                    className="p-2 text-gray-400 hover:text-red-500 transition-colors"
                    title={t('llm.delete')}
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {editingId === config.id && (
                <div className="mt-4 pt-4 border-t dark:border-gray-700 space-y-3">
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                        {t('llm.apiKey')}
                      </label>
                      <input
                        type="password"
                        defaultValue={config.api_key}
                        onBlur={(e) => {
                          if (e.target.value !== config.api_key) {
                            handleUpdate(config.id, { ...config, api_key: e.target.value })
                          }
                        }}
                        className="w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                        {t('llm.baseUrl')}
                      </label>
                      <input
                        type="text"
                        defaultValue={config.base_url}
                        onBlur={(e) => {
                          if (e.target.value !== config.base_url) {
                            handleUpdate(config.id, { ...config, base_url: e.target.value })
                          }
                        }}
                        className="w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100"
                      />
                    </div>
                  </div>
                  <div className="flex justify-end">
                    <button
                      onClick={() => setEditingId(null)}
                      className="px-3 py-1 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
                    >
                      {t('llm.done')}
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))
        )}
      </div>

      {/* Toast */}
      {toast && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast(null)}
        />
      )}

      {/* Delete Confirmation Dialog */}
      {deleteConfirm.show && (
        <ConfirmDialog
          title={t('llm.deleteConfirmTitle')}
          message={t('llm.deleteConfirm')}
          confirmText={t('common.delete')}
          cancelText={t('common.cancel')}
          onConfirm={handleDelete}
          onCancel={() => setDeleteConfirm({ show: false, configId: null })}
        />
      )}
    </div>
  )
}
