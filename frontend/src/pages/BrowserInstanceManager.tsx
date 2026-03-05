import { useState, useEffect } from 'react'
import { api, BrowserInstance } from '../api/client'
import { Plus, Power, PowerOff, Edit, Trash2, Settings, RefreshCw, ArrowLeft } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import Toast from '../components/Toast'
import ConfirmDialog from '../components/ConfirmDialog'
import { useLanguage } from '../i18n'

export default function BrowserInstanceManager() {
  const { t } = useLanguage()
  const navigate = useNavigate()
  const [instances, setInstances] = useState<BrowserInstance[]>([])
  const [loading, setLoading] = useState(false)
  const [showModal, setShowModal] = useState(false)
  const [editingInstance, setEditingInstance] = useState<BrowserInstance | null>(null)
  const [message, setMessage] = useState('')
  const [showToast, setShowToast] = useState(false)
  const [toastType, setToastType] = useState<'success' | 'error' | 'info'>('info')
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; instanceId: string | null }>({ 
    show: false, 
    instanceId: null 
  })
  const [currentInstanceId, setCurrentInstanceId] = useState<string>('')
  
  const [instanceForm, setInstanceForm] = useState({
    name: '',
    description: '',
    type: 'local' as 'local' | 'remote',
    bin_path: '',
    user_data_dir: '',
    control_url: '',
    user_agent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36',
    use_stealth: null as boolean | null,
    headless: null as boolean | null,
    no_sandbox: null as boolean | null,
    launch_args: [] as string[],
    proxy: '',
    is_default: false,
  })

  const showMessage = (msg: string, type: 'success' | 'error' | 'info' = 'success') => {
    setMessage(msg)
    setToastType(type)
    setShowToast(true)
  }

  useEffect(() => {
    loadInstances()
    loadCurrentInstance()

    // 定期刷新实例状态（每5秒）
    const intervalId = setInterval(() => {
      loadInstances()
      loadCurrentInstance()
    }, 5000)

    // 页面获得焦点时刷新
    const handleFocus = () => {
      loadInstances()
      loadCurrentInstance()
    }
    window.addEventListener('focus', handleFocus)

    // 清理函数
    return () => {
      clearInterval(intervalId)
      window.removeEventListener('focus', handleFocus)
    }
  }, [])

  const loadInstances = async () => {
    try {
      setLoading(true)
      const response = await api.listBrowserInstances()
      setInstances(response.data.instances || [])
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.loadError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const loadCurrentInstance = async () => {
    try {
      const response = await api.getCurrentBrowserInstance()
      if (response.data.instance) {
        setCurrentInstanceId(response.data.instance.id)
      }
    } catch (err) {
      // 没有当前实例，忽略错误
    }
  }

  const handleSaveInstance = async () => {
    if (!instanceForm.name.trim()) {
      showMessage(t('browser.instance.nameRequired'), 'error')
      return
    }

    if (instanceForm.type === 'local') {
      if (!instanceForm.user_data_dir.trim()) {
        showMessage(t('browser.instance.userDataDirRequired'), 'error')
        return
      }
    } else {
      if (!instanceForm.control_url.trim()) {
        showMessage(t('browser.instance.controlUrlRequired'), 'error')
        return
      }
    }

    try {
      if (editingInstance) {
        await api.updateBrowserInstance(editingInstance.id, instanceForm)
        showMessage(t('browser.instance.updateSuccess'), 'success')
      } else {
        await api.createBrowserInstance(instanceForm)
        showMessage(t('browser.instance.createSuccess'), 'success')
      }
      setShowModal(false)
      setEditingInstance(null)
      await loadInstances()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.instance.saveError'), 'error')
    }
  }

  const handleDeleteInstance = async () => {
    if (!deleteConfirm.instanceId) return

    try {
      await api.deleteBrowserInstance(deleteConfirm.instanceId)
      showMessage(t('browser.instance.deleteSuccess'), 'success')
      await loadInstances()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.instance.deleteError'), 'error')
    } finally {
      setDeleteConfirm({ show: false, instanceId: null })
    }
  }

  const handleStartInstance = async (id: string) => {
    try {
      await api.startBrowserInstance(id)
      showMessage(t('browser.instance.startSuccess'), 'success')
      await loadInstances()
      await loadCurrentInstance()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.instance.startError'), 'error')
    }
  }

  const handleStopInstance = async (id: string) => {
    try {
      await api.stopBrowserInstance(id)
      showMessage(t('browser.instance.stopSuccess'), 'success')
      await loadInstances()
      await loadCurrentInstance()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.instance.stopError'), 'error')
    }
  }

  const handleSwitchInstance = async (id: string) => {
    try {
      await api.switchBrowserInstance(id)
      showMessage(t('browser.instance.switchSuccess'), 'success')
      setCurrentInstanceId(id)
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.instance.switchError'), 'error')
    }
  }

  const openCreateModal = () => {
    setEditingInstance(null)
    setInstanceForm({
      name: '',
      description: '',
      type: 'local',
      bin_path: '',
      user_data_dir: '',
      control_url: '',
      user_agent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36',
      use_stealth: null,
      headless: null,
      no_sandbox: null,
      launch_args: [],
      proxy: '',
      is_default: false,
    })
    setShowModal(true)
  }

  const openEditModal = (instance: BrowserInstance) => {
    setEditingInstance(instance)
    setInstanceForm({
      name: instance.name,
      description: instance.description,
      type: instance.type,
      bin_path: instance.bin_path || '',
      user_data_dir: instance.user_data_dir || '',
      control_url: instance.control_url || '',
      user_agent: instance.user_agent || 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36',
      use_stealth: instance.use_stealth ?? null,
      headless: instance.headless ?? null,
      no_sandbox: instance.no_sandbox ?? null,
      launch_args: instance.launch_args || [],
      proxy: instance.proxy || '',
      is_default: instance.is_default,
    })
    setShowModal(true)
  }

  return (
    <div className="space-y-6 lg:space-y-8 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            {t('browser.instance.title')}
          </h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">
            {t('browser.instance.manage')}
          </p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => navigate('/browser')}
            className="btn-secondary flex items-center space-x-1.5"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>{t('common.back')}</span>
          </button>
          <button
            onClick={loadInstances}
            disabled={loading}
            className="btn-secondary flex items-center space-x-1.5"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            <span>{t('common.refresh')}</span>
          </button>
          <button
            onClick={openCreateModal}
            className="btn-secondary flex items-center space-x-1.5"
          >
            <Plus className="w-4 h-4" />
            <span>{t('browser.instance.create')}</span>
          </button>
        </div>
      </div>

      {/* Instance List */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {instances.map((instance) => (
          <div
            key={instance.id}
            className={`bg-white dark:bg-gray-800 rounded-lg border transition-colors ${
              instance.id === currentInstanceId 
                ? 'border-gray-900 dark:border-gray-100 shadow-sm' 
                : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
            }`}
          >
            {/* 头部：标题和操作按钮 */}
            <div className="p-4">
              <div className="flex items-start justify-between mb-3">
                <div className="flex-1 min-w-0">
                  <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100 truncate mb-2">
                    {instance.name}
                  </h3>
                  <div className="flex flex-wrap gap-1.5">
                    {instance.is_default && (
                      <span className="px-2 py-0.5 bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 text-xs rounded-full">
                        {t('browser.instance.default')}
                      </span>
                    )}
                    {instance.id === currentInstanceId && (
                      <span className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 text-xs rounded-full">
                        {t('browser.instance.current')}
                      </span>
                    )}
                    <span className={`px-2 py-0.5 rounded-full text-xs ${
                      instance.type === 'local' 
                        ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' 
                        : 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300'
                    }`}>
                      {t(`browser.instance.${instance.type}`)}
                    </span>
                    <span className={`px-2 py-0.5 rounded-full text-xs ${
                      instance.is_active 
                        ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' 
                        : 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
                    }`}>
                      {instance.is_active ? t('browser.instance.running') : t('browser.instance.stopped')}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-1 ml-2 flex-shrink-0">
                  <button
                    onClick={() => openEditModal(instance)}
                    disabled={instance.is_active}
                    className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    title={t('browser.instance.edit')}
                  >
                    <Edit className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                  </button>
                  {!instance.is_default && !instance.is_active && (
                    <button
                      onClick={() => setDeleteConfirm({ show: true, instanceId: instance.id })}
                      className="p-1.5 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                      title={t('browser.instance.delete')}
                    >
                      <Trash2 className="w-4 h-4 text-red-600 dark:text-red-400" />
                    </button>
                  )}
                </div>
              </div>

              {instance.description && (
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-3 line-clamp-2">
                  {instance.description}
                </p>
              )}

              {/* 路径信息 */}
              {instance.type === 'local' && instance.user_data_dir && (
                <div className="text-xs text-gray-500 dark:text-gray-400 truncate mb-3">
                  <span className="font-medium">{t('browser.instance.userDataDir')}:</span> {instance.user_data_dir}
                </div>
              )}
              {instance.type === 'remote' && instance.control_url && (
                <div className="text-xs text-gray-500 dark:text-gray-400 truncate mb-3">
                  <span className="font-medium">{t('browser.instance.controlUrl')}:</span> {instance.control_url}
                </div>
              )}

              {/* 主操作按钮 */}
              <div className="flex gap-2">
                {!instance.is_active ? (
                  <button
                    onClick={() => handleStartInstance(instance.id)}
                    className="flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm bg-gray-900 hover:bg-gray-800 dark:bg-gray-100 dark:hover:bg-gray-200 text-white dark:text-gray-900 rounded-md transition-colors"
                  >
                    <Power className="w-3.5 h-3.5" />
                    <span>{t('browser.instance.start')}</span>
                  </button>
                ) : (
                  <>
                    {instance.id !== currentInstanceId && (
                      <button
                        onClick={() => handleSwitchInstance(instance.id)}
                        className="flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-md transition-colors"
                      >
                        <Settings className="w-3.5 h-3.5" />
                        <span>{t('browser.instance.switch')}</span>
                      </button>
                    )}
                    <button
                      onClick={() => handleStopInstance(instance.id)}
                      className="flex-1 flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm border border-red-300 dark:border-red-800 hover:bg-red-50 dark:hover:bg-red-900/20 text-red-600 dark:text-red-400 rounded-md transition-colors"
                    >
                      <PowerOff className="w-3.5 h-3.5" />
                      <span>{t('browser.instance.stop')}</span>
                    </button>
                  </>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {instances.length === 0 && !loading && (
        <div className="text-center py-12 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg">
          <p className="text-gray-500 dark:text-gray-400 mb-4">
            {t('browser.instance.noInstances')}
          </p>
          <button onClick={openCreateModal} className="btn-primary">
            {t('browser.instance.createFirst')}
          </button>
        </div>
      )}

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4 overflow-y-auto" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-2xl w-full p-6 my-8 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {editingInstance ? t('browser.instance.editTitle') : t('browser.instance.createTitle')}
              </h3>
              <button
                onClick={() => {
                  setShowModal(false)
                  setEditingInstance(null)
                }}
                className="btn-ghost p-2"
              >
                ✕
              </button>
            </div>

            <div className="space-y-4">
              {/* 基本信息 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('browser.instance.name')} <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={instanceForm.name}
                  onChange={(e) => setInstanceForm({ ...instanceForm, name: e.target.value })}
                  placeholder={t('browser.instance.namePlaceholder')}
                  className="input w-full"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('browser.instance.description')}
                </label>
                <textarea
                  value={instanceForm.description}
                  onChange={(e) => setInstanceForm({ ...instanceForm, description: e.target.value })}
                  placeholder={t('browser.instance.descriptionPlaceholder')}
                  rows={2}
                  className="input w-full"
                />
              </div>

              {/* 实例类型 */}
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('browser.instance.type')} <span className="text-red-500">*</span>
                </label>
                <select
                  value={instanceForm.type}
                  onChange={(e) => setInstanceForm({ ...instanceForm, type: e.target.value as 'local' | 'remote' })}
                  className="input w-full"
                >
                  <option value="local">{t('browser.instance.local')}</option>
                  <option value="remote">{t('browser.instance.remote')}</option>
                </select>
              </div>

              {/* 本地模式配置 */}
              {instanceForm.type === 'local' && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('browser.instance.binPath')}
                    </label>
                    <input
                      type="text"
                      value={instanceForm.bin_path}
                      onChange={(e) => setInstanceForm({ ...instanceForm, bin_path: e.target.value })}
                      placeholder={t('browser.instance.binPathPlaceholder')}
                      className="input w-full font-mono text-sm"
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {t('browser.instance.binPathHint')}
                    </p>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('browser.instance.userDataDir')} <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      value={instanceForm.user_data_dir}
                      onChange={(e) => setInstanceForm({ ...instanceForm, user_data_dir: e.target.value })}
                      placeholder={t('browser.instance.userDataDirPlaceholder')}
                      className="input w-full font-mono text-sm"
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {t('browser.instance.userDataDirHint')}
                    </p>
                  </div>
                </>
              )}

              {/* 远程模式配置 */}
              {instanceForm.type === 'remote' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.instance.controlUrl')} <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={instanceForm.control_url}
                    onChange={(e) => setInstanceForm({ ...instanceForm, control_url: e.target.value })}
                    placeholder={t('browser.instance.controlUrlPlaceholder')}
                    className="input w-full font-mono text-sm"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('browser.instance.controlUrlHint')}
                  </p>
                </div>
              )}

              {/* 高级配置（可选） */}
              <div className="border-t dark:border-gray-700 pt-4">
                <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
                  {t('browser.instance.advancedSettings')}
                </h4>
                
                {/* Proxy */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.proxy')}
                  </label>
                  <input
                    type="text"
                    value={instanceForm.proxy}
                    onChange={(e) => setInstanceForm({ ...instanceForm, proxy: e.target.value })}
                    placeholder={t('browser.config.proxyPlaceholder')}
                    className="input w-full font-mono text-sm"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('browser.config.proxyHint')}
                  </p>
                </div>

                {/* Headless */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.headlessMode')}
                  </label>
                  <select
                    value={instanceForm.headless === null ? 'default' : (instanceForm.headless ? 'enabled' : 'disabled')}
                    onChange={(e) => {
                      const value = e.target.value
                      setInstanceForm({
                        ...instanceForm,
                        headless: value === 'default' ? null : value === 'enabled'
                      })
                    }}
                    className="input w-full"
                  >
                    <option value="default">{t('browser.config.headlessDefault')}</option>
                    <option value="enabled">{t('browser.config.headlessEnabled')}</option>
                    <option value="disabled">{t('browser.config.headlessDisabledOption')}</option>
                  </select>
                </div>

                {/* NoSandbox */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.noSandboxMode')}
                  </label>
                  <select
                    value={instanceForm.no_sandbox === null ? 'default' : (instanceForm.no_sandbox ? 'enabled' : 'disabled')}
                    onChange={(e) => {
                      const value = e.target.value
                      setInstanceForm({
                        ...instanceForm,
                        no_sandbox: value === 'default' ? null : value === 'enabled'
                      })
                    }}
                    className="input w-full"
                  >
                    <option value="default">{t('browser.config.noSandboxDefault')}</option>
                    <option value="enabled">{t('browser.config.noSandboxEnabled')}</option>
                    <option value="disabled">{t('browser.config.noSandboxDisabledOption')}</option>
                  </select>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('browser.config.noSandboxHint')}
                  </p>
                </div>
              </div>

              {/* 设为默认 */}
              <div>
                <label className="flex items-center space-x-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={instanceForm.is_default}
                    onChange={(e) => setInstanceForm({ ...instanceForm, is_default: e.target.checked })}
                    className="w-4 h-4 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600 rounded focus:ring-gray-900 dark:focus:ring-gray-100"
                  />
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {t('browser.instance.setAsDefault')}
                  </span>
                </label>
              </div>

              {/* 操作按钮 */}
              <div className="flex items-center justify-end space-x-3 pt-4 border-t dark:border-gray-700">
                <button
                  onClick={() => {
                    setShowModal(false)
                    setEditingInstance(null)
                  }}
                  className="btn-ghost"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={handleSaveInstance}
                  className="btn-primary"
                >
                  {editingInstance ? t('common.update') : t('common.create')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deleteConfirm.show && (
        <ConfirmDialog
          title={t('browser.instance.deleteConfirm')}
          message={t('browser.instance.deleteMessage')}
          confirmText={t('common.delete')}
          cancelText={t('common.cancel')}
          onConfirm={handleDeleteInstance}
          onCancel={() => setDeleteConfirm({ show: false, instanceId: null })}
        />
      )}

      {/* Toast */}
      {showToast && (
        <Toast
          message={message}
          type={toastType}
          onClose={() => setShowToast(false)}
        />
      )}
    </div>
  )
}
