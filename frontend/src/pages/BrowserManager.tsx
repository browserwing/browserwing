import { useState, useEffect, useCallback } from 'react'
import { api, Script, ScriptAction, BrowserConfig } from '../api/client'
import { Power, PowerOff, Loader, ExternalLink, RefreshCw, Save, Video, Play, Settings, Cookie } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import Toast from '../components/Toast'
import ConfirmDialog from '../components/ConfirmDialog'
import { useLanguage } from '../i18n'

interface BrowserStatus {
  is_running: boolean
  start_time?: string
  uptime?: string
  pages_count?: number
}

interface RecordingStatus {
  is_recording: boolean
  start_url?: string
  start_time?: string
  duration?: number
  in_page_stopped?: boolean
  actions?: ScriptAction[]
  count?: number
}

export default function BrowserManager() {
  const { t, language } = useLanguage()
  const navigate = useNavigate()
  const [status, setStatus] = useState<BrowserStatus>({ is_running: false })
  const [recordingStatus, setRecordingStatus] = useState<RecordingStatus>({ is_recording: false })
  const [startingBrowser, setStartingBrowser] = useState(false)
  const [stoppingBrowser, setStoppingBrowser] = useState(false)
  const [openingPage, setOpeningPage] = useState(false)
  const [savingCookies, setSavingCookies] = useState(false)
  const [recordingLoading, setRecordingLoading] = useState(false)
  const [executingScript, setExecutingScript] = useState(false)
  const [savingScript, setSavingScript] = useState(false)
  const [openUrl, setOpenUrl] = useState('https://www.xiaohongshu.com')
  const [message, setMessage] = useState('')
  const [showToast, setShowToast] = useState(false)
  const [toastType, setToastType] = useState<'success' | 'error' | 'info'>('info')
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; configId: string | null }>({ show: false, configId: null })
  
  const showMessage = useCallback((msg: string, type: 'success' | 'error' | 'info' = 'success') => {
    setMessage(msg)
    setToastType(type)
    setShowToast(true)
  }, [])
  
  const [scripts, setScripts] = useState<Script[]>([])
  const [recordedActions, setRecordedActions] = useState<ScriptAction[]>([])
  const [showSaveDialog, setShowSaveDialog] = useState(false)
  const [scriptName, setScriptName] = useState('')
  const [scriptDescription, setScriptDescription] = useState('')
  const [hasShownStopMessage, setHasShownStopMessage] = useState(false) // 标记是否已显示停止消息
  
  // 历史访问记录状态
  const [historyLinks, setHistoryLinks] = useState<string[]>([])

  // 浏览器配置相关状态
  const [showConfigModal, setShowConfigModal] = useState(false)
  const [configs, setConfigs] = useState<BrowserConfig[]>([])
  const [editingConfig, setEditingConfig] = useState<BrowserConfig | null>(null)
  const [configForm, setConfigForm] = useState({
    name: '',
    description: '',
    url_pattern: '',
    user_agent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36',
    use_stealth: null as boolean | null,
    headless: null as boolean | null,
    launch_args: [] as string[],
    is_default: false,
  })

  // 从localStorage加载历史访问记录
  useEffect(() => {
    const savedHistory = localStorage.getItem('browserHistory')
    if (savedHistory) {
      try {
        setHistoryLinks(JSON.parse(savedHistory))
      } catch (error) {
        console.error('Failed to parse browser history:', error)
      }
    }
  }, [])

  useEffect(() => {
    loadStatus()
    loadScripts()
    loadConfigs()
    
    // 定时刷新状态和录制状态
    const interval = setInterval(() => {
      loadStatus()
      loadRecordingStatus()
    }, 2000) // 每2秒刷新状态,以便及时响应页面内的录制操作
    
    return () => clearInterval(interval)
  }, [])

  const loadStatus = async () => {
    try {
      const response = await api.getBrowserStatus()
      setStatus(response.data)
    } catch (err) {
      console.error('获取浏览器状态失败:', err)
    }
  }

  const handleStart = async () => {
    try {
      setStartingBrowser(true)
      const response = await api.startBrowser()
      showMessage(t(response.data.message), 'success')
      await loadStatus()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.startError'), 'error')
    } finally {
      setStartingBrowser(false)
    }
  }

  const handleStop = async () => {
    try {
      setStoppingBrowser(true)
      showMessage(t('browser.messages.stopInfo'), 'info')
      const response = await api.stopBrowser()
      showMessage(t(response.data.message) + ' ' + t('browser.messages.stopSuccess'), 'success')
      await loadStatus()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.stopError'), 'error')
    } finally {
      setStoppingBrowser(false)
    }
  }

  // 保存历史访问记录到localStorage
  const saveToHistory = (url: string) => {
    if (!url || !url.trim()) return
    
    setHistoryLinks(prev => {
      // 移除已存在的相同URL
      const newHistory = prev.filter(item => item !== url)
      // 添加到历史记录开头
      newHistory.unshift(url)
      // 只保留最近10条记录
      const trimmedHistory = newHistory.slice(0, 10)
      // 保存到localStorage
      localStorage.setItem('browserHistory', JSON.stringify(trimmedHistory))
      return trimmedHistory
    })
  }

  const handleOpenPage = async (url?: string) => {
    const targetUrl = url || openUrl
    if (!targetUrl.trim()) {
      showMessage(t('browser.messages.urlRequired'), 'error')
      return
    }

    try {
      setOpeningPage(true)
      const response = await api.openBrowserPage(targetUrl)
      showMessage(t(response.data.message), 'success')
      // 将当前URL添加到历史记录
      saveToHistory(targetUrl)
      // 更新输入框的值
      setOpenUrl(targetUrl)
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.openError'), 'error')
    } finally {
      setOpeningPage(false)
    }
  }

  const handleSaveCookies = async () => {
    try {
      setSavingCookies(true)
      const response = await api.saveBrowserCookies()
      showMessage(`${t(response.data.message)} - ${t('browser.messages.cookieSaved', { count: response.data.count })}`, 'success')
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.cookieError'), 'error')
    } finally {
      setSavingCookies(false)
    }
  }

  const loadRecordingStatus = useCallback(async () => {
    try {
      const response = await api.getRecordingStatus()
      const status: RecordingStatus = response.data
      
      // 检测是否是页面内停止的录制
      if (status.in_page_stopped && !hasShownStopMessage) {
        // 显示保存对话框
        setRecordedActions(status.actions || [])
        if (status.actions && status.actions.length > 0) {
          setShowSaveDialog(true)
        } else {
          await cleanRecordingState()
        }
        showMessage(`${t('browser.messages.recordingStopped')} - ${t('browser.messages.recordStopSuccess', { count: status.count || 0 })}`, 'success')
        setHasShownStopMessage(true) // 标记已显示消息
        
        // 立即清除后端的 in_page_stopped 状态，避免重复弹出
        await api.clearInPageRecordingState()
      }
      
      setRecordingStatus(status)
    } catch (err) {
      console.error('获取录制状态失败:', err)
    }
  }, [hasShownStopMessage, showMessage, t])

  const loadScripts = async () => {
    try {
      const response = await api.getScripts()
      setScripts(response.data.scripts || [])
    } catch (err) {
      console.error('获取脚本列表失败:', err)
    }
  }

  // 加载浏览器配置列表
  const loadConfigs = async () => {
    try {
      const response = await api.getBrowserConfigs()
      setConfigs(response.data.configs || [])
    } catch (err) {
      console.error('获取浏览器配置失败:', err)
    }
  }

  // 保存配置
  const handleSaveConfig = async () => {
    if (!configForm.name.trim()) {
      showMessage(t('browser.config.nameRequired'), 'error')
      return
    }

    try {
      if (editingConfig) {
        await api.updateBrowserConfig(editingConfig.id, configForm)
        showMessage(t('browser.config.updateSuccess'), 'success')
      } else {
        await api.createBrowserConfig(configForm)
        showMessage(t('browser.config.createSuccess'), 'success')
      }
      setShowConfigModal(false)
      setEditingConfig(null)
      await loadConfigs()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('browser.config.saveError'), 'error')
    }
  }

  // 删除配置
  const handleDeleteConfig = async () => {
    if (!deleteConfirm.configId) return

    try {
      await api.deleteBrowserConfig(deleteConfirm.configId)
      showMessage(t('browser.config.deleteSuccess'), 'success')
      await loadConfigs()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('browser.config.deleteError'), 'error')
    } finally {
      setDeleteConfirm({ show: false, configId: null })
    }
  }

  const handleStartRecording = async () => {
    try {
      setRecordingLoading(true)
      const response = await api.startRecording()
      showMessage(t(response.data.message), 'success')
      await loadRecordingStatus()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.recordStartError'), 'error')
    } finally {
      setRecordingLoading(false)
    }
  }

  const handleStopRecording = async () => {
    try {
      setRecordingLoading(true)
      const response = await api.stopRecording()
      setRecordedActions(response.data.actions || [])
      showMessage(`${t(response.data.message)} - ${t('browser.messages.recordStopSuccess', { count: response.data.count })}`, 'success')
      if (response.data.actions && response.data.actions.length > 0) {
        setShowSaveDialog(true)
      } else {
        await cleanRecordingState()
      }
      await loadRecordingStatus()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.recordStopError'), 'error')
    } finally {
      setRecordingLoading(false)
    }
  }

  const handleSaveScript = async () => {
    if (!scriptName.trim()) {
      showMessage(t('browser.messages.scriptNameRequired'), 'error')
      return
    }

    try {
      setSavingScript(true)
      const response = await api.saveScript({
        name: scriptName,
        description: scriptDescription,
        url: recordingStatus.start_url || openUrl,
        actions: recordedActions,
      })
      showMessage(t(response.data.message), 'success')
      await cleanRecordingState()
      await loadScripts()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.scriptSaveError'), 'error')
    } finally {
      setSavingScript(false)
    }
  }

  const cleanRecordingState = async () => {
    setShowSaveDialog(false)
    setScriptName('')
    setScriptDescription('')
    setRecordedActions([])
    setHasShownStopMessage(false)
  }

  const handlePlayScript = async (scriptId: string) => {
    try {
      setExecutingScript(true)
      const response = await api.playScript(scriptId)
      showMessage(t(response.data.message), 'success')
    } catch (err: any) {
      showMessage(t(err.response?.data?.error || 'browser.messages.scriptPlayError'), 'error')
    } finally {
      setExecutingScript(false)
    }
  }

  // 根据语言动态生成快速访问链接
  const getQuickLinks = () => {
    // 中文环境使用中国常用网站
    if (language.startsWith('zh')) {
      return [
        { name: t('browser.quickLinks.xiaohongshu'), url: 'https://www.xiaohongshu.com' },
        { name: t('browser.quickLinks.wechat'), url: 'https://mp.weixin.qq.com' },
        { name: t('browser.quickLinks.bilibili'), url: 'https://www.bilibili.com' },
        { name: t('browser.quickLinks.zhihu'), url: 'https://www.zhihu.com' },
        { name: t('browser.quickLinks.csdn'), url: 'https://www.csdn.net' },
        { name: t('browser.quickLinks.weibo'), url: 'https://weibo.com' },
        { name: t('browser.quickLinks.twitter'), url: 'https://x.com' },
        { name: t('browser.quickLinks.reddit'), url: 'https://www.reddit.com' },
        { name: t('browser.quickLinks.jike'), url: 'https://web.okjike.com' },
      ]
    }
    // 英文环境使用国际常用网站
    else if (language === 'en') {
      return [
        { name: 'Google', url: 'https://www.google.com' },
        { name: 'YouTube', url: 'https://www.youtube.com' },
        { name: 'Facebook', url: 'https://www.facebook.com' },
        { name: 'Twitter', url: 'https://x.com' },
        { name: 'Reddit', url: 'https://www.reddit.com' },
        { name: 'LinkedIn', url: 'https://www.linkedin.com' },
        { name: 'GitHub', url: 'https://www.github.com' },
        { name: 'Wikipedia', url: 'https://www.wikipedia.org' },
        { name: 'Amazon', url: 'https://www.amazon.com' },
      ]
    }
    // 其他语言默认使用英文网站
    else {
      return [
        { name: 'Google', url: 'https://www.google.com' },
        { name: 'YouTube', url: 'https://www.youtube.com' },
        { name: 'Facebook', url: 'https://www.facebook.com' },
        { name: 'Twitter', url: 'https://x.com' },
        { name: 'Reddit', url: 'https://www.reddit.com' },
        { name: 'LinkedIn', url: 'https://www.linkedin.com' },
        { name: 'GitHub', url: 'https://www.github.com' },
        { name: 'Wikipedia', url: 'https://www.wikipedia.org' },
      ]
    }
  }
  
  const quickLinks = getQuickLinks()

  return (
    <div className="space-y-6 lg:space-y-8 animate-fade-in">
      {/* Status and Control Card */}
      <div className="card">
        <div className="flex items-center justify-between mb-5 pb-3 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg lg:text-xl font-semibold text-gray-900 dark:text-gray-100 flex items-center space-x-2">
            <div className={`w-3 h-3 rounded-full ${status.is_running ? 'bg-gray-900 dark:bg-gray-100' : 'bg-gray-300 dark:bg-gray-600'}`}></div>
            <span>{t('browser.control.title')}</span>
          </h2>
          <div className="flex items-center space-x-2">
            <button
              onClick={() => {
                setEditingConfig(null)
                setConfigForm({
                  name: '',
                  description: '',
                  url_pattern: '',
                  user_agent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36',
                  use_stealth: null,
                  headless: null,
                  launch_args: [],
                  is_default: false,
                })
                setShowConfigModal(true)
              }}
              className="btn-ghost p-2"
              title={t('browser.control.advancedSettings')}
            >
              <Settings className="w-5 h-5" />
            </button>
            <button
              onClick={loadStatus}
              className="btn-ghost p-2"
              title={t('browser.control.refreshStatus')}
            >
              <RefreshCw className="w-5 h-5" />
            </button>
          </div>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-6 mb-6">
          <div className="space-y-2">
            <div className="text-[15px] text-gray-500 dark:text-gray-400">{t('browser.status.label')}</div>
            <div className={`text-xl lg:text-2xl font-bold ${status.is_running ? 'text-gray-900 dark:text-gray-100' : 'text-gray-400 dark:text-gray-500'}`}>
              {status.is_running ? t('browser.status.running') : t('browser.status.stopped')}
            </div>
          </div>
          {status.is_running && (
            <>
              <div className="space-y-2">
                <div className="text-sm text-gray-500">{t('browser.status.startTime')}</div>
                <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {status.start_time ? new Date(status.start_time).toLocaleTimeString('zh-CN') : '-'}
                </div>
              </div>
              <div className="space-y-2">
                <div className="text-sm text-gray-500">{t('browser.status.uptime')}</div>
                <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {status.uptime ? status.uptime.replace(/(\d+)(\.\d+)([a-z])/g, '$1$3') : '-'}
                </div>
              </div>
              <div className="space-y-2">
                <div className="text-sm text-gray-500">打开页面数</div>
                <div className="text-lg font-bold text-gray-900 dark:text-gray-100">
                  {status.pages_count ?? 0}
                </div>
              </div>
            </>
          )}
        </div>

        <div className="flex items-center space-x-4">
          {!status.is_running ? (
            <button
              onClick={handleStart}
              disabled={startingBrowser}
              className="btn-primary flex items-center space-x-2"
            >
              {startingBrowser ? (
                <>
                  <Loader className="w-5 h-5 animate-spin" />
                  <span>{t('browser.control.starting')}</span>
                </>
              ) : (
                <>
                  <Power className="w-5 h-5" />
                    <span>{t('browser.control.start')}</span>
                </>
              )}
            </button>
          ) : (
            <>
              <button
                onClick={handleStop}
                disabled={stoppingBrowser}
                className="btn-danger flex items-center space-x-2"
              >
                {stoppingBrowser ? (
                  <>
                    <Loader className="w-5 h-5 animate-spin" />
                      <span>{t('browser.control.stopping')}</span>
                  </>
                ) : (
                  <>
                    <PowerOff className="w-5 h-5" />
                        <span>{t('browser.control.stop')}</span>
                  </>
                )}
              </button>
              
              <button
                onClick={handleSaveCookies}
                disabled={savingCookies}
                className="btn-secondary flex items-center space-x-2"
              >
                {savingCookies ? (
                  <Loader className="w-5 h-5 animate-spin" />
                ) : (
                  <>
                    <Save className="w-5 h-5" />
                        <span>{t('browser.control.saveCookies')}</span>
                  </>
                )}
              </button>

                <button
                  onClick={() => navigate('/cookies')}
                  className="btn-secondary flex items-center space-x-2"
                >
                  <Cookie className="w-5 h-5" />
                  <span>{t('browser.control.cookieManagement')}</span>
                </button>

              {!recordingStatus.is_recording ? (
                <button
                  onClick={handleStartRecording}
                  disabled={recordingLoading}
                  className="btn-secondary flex items-center space-x-2"
                >
                  {recordingLoading ? (
                    <Loader className="w-5 h-5 animate-spin" />
                  ) : (
                    <>
                      <Video className="w-5 h-5" />
                          <span>{t('browser.recording.start')}</span>
                    </>
                  )}
                </button>
              ) : (
                <button
                  onClick={handleStopRecording}
                  disabled={recordingLoading}
                  className="btn-primary flex items-center space-x-2"
                >
                  {recordingLoading ? (
                    <Loader className="w-5 h-5 animate-spin" />
                  ) : (
                    <>
                      <Video className="w-5 h-5" />
                            <span>{t('browser.recording.stop')}</span>
                    </>
                  )}
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* Open Page Card */}
      {status.is_running && (
        <>
          <div className="card">
            <h2 className="text-xl font-bold text-gray-900 dark:text-gray-100 mb-4 flex items-center space-x-2">
              <ExternalLink className="w-5 h-5 text-gray-900 dark:text-gray-100" />
              <span>{t('browser.page.title')}</span>
            </h2>

            <div className="space-y-4">
              <div className="flex items-center space-x-3">
                <input
                  type="url"
                  value={openUrl}
                  onChange={(e) => setOpenUrl(e.target.value)}
                  placeholder={t('browser.page.placeholder')}
                  className="input flex-1"
                />
                <button
                  onClick={() => handleOpenPage()}
                  disabled={openingPage}
                  className="btn-primary whitespace-nowrap"
                >
                  {openingPage ? <Loader className="w-5 h-5 animate-spin" /> : t('browser.page.open')}
                </button>
              </div>

              <div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">{t('browser.page.quickAccess')}:</p>
                <div className="flex flex-wrap gap-2">
                  {quickLinks.map((link) => (
                    <button
                      key={link.name}
                      onClick={() => {
                        handleOpenPage(link.url);
                      }}
                      className="px-4 py-2 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium transition-colors"
                    >
                      {link.name}
                    </button>
                  ))}
                </div>
              </div>

              {/* 历史访问记录 */}
              {historyLinks.length > 0 && (
                <div className="mt-6">
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">{t('browser.page.recentVisits')}:</p>
                  <div className="flex flex-wrap gap-2">
                    {historyLinks.map((url, index) => {
                      // 去除http和https协议部分
                      const displayUrl = url.replace(/^https?:\/\//, '');
                      return (
                        <button
                          key={index}
                          onClick={() => {
                            handleOpenPage(url);
                          }}
                          className="px-4 py-2 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium transition-colors truncate max-w-[180px]"
                          title={url}
                        >
                          {displayUrl}
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Execute Script Card */}
          <div className="card">
            <h2 className="text-xl lg:text-2xl font-bold text-gray-900 dark:text-gray-100 mb-5 flex items-center space-x-2">
              <Play className="w-6 h-6 text-gray-900 dark:text-gray-100" />
              <span>{t('browser.script.execute')}</span>
            </h2>

            <div className="space-y-4">
              <div className="flex items-center space-x-3">
                <select
                  value=""
                  onChange={(e) => {
                    if (e.target.value) {
                      handlePlayScript(e.target.value)
                      e.target.value = ''
                    }
                  }}
                  disabled={executingScript || scripts.length === 0}
                  className="input flex-1"
                >
                  <option value="">{t('browser.script.selectScript')}</option>
                  {scripts.map((script) => (
                    <option key={script.id} value={script.id}>
                      {script.name} ({script.actions.length} {t('browser.script.steps')})
                    </option>
                  ))}
                </select>
                <button
                  onClick={loadScripts}
                  className="btn-ghost p-2"
                  title={t('browser.script.refresh')}
                  disabled={executingScript}
                >
                  <RefreshCw className={`w-5 h-5 ${executingScript ? 'animate-spin' : ''}`} />
                </button>
              </div>

              {scripts.length === 0 && (
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {t('browser.script.noScripts')}
                </p>
              )}
            </div>
          </div>
        </>
      )}

      {/* Save Script Dialog */}
      {showSaveDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-md w-full p-6">
            <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100 mb-4">{t('browser.script.saveRecording')}</h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {t('browser.script.name')} <span className="text-gray-900 dark:text-gray-100">*</span>
                </label>
                <input
                  type="text"
                  value={scriptName}
                  onChange={(e) => setScriptName(e.target.value)}
                  placeholder={t('browser.script.namePlaceholder')}
                  className="input w-full"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {t('browser.script.description')}
                </label>
                <textarea
                  value={scriptDescription}
                  onChange={(e) => setScriptDescription(e.target.value)}
                  placeholder={t('browser.script.descriptionPlaceholder')}
                  rows={3}
                  className="input w-full"
                />
              </div>

              <div className="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg">
                <div className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
                  <div className="flex justify-between">
                    <span>{t('browser.script.startUrl')}:</span>
                    <span className="font-mono text-xs">{recordingStatus.start_url}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>{t('browser.script.steps')}:</span>
                    <span className="font-bold text-gray-900 dark:text-gray-100">{recordedActions.length} {t('browser.script.stepsUnit')}</span>
                  </div>
                </div>
              </div>

              <div className="flex items-center space-x-3 pt-4">
                <button
                  onClick={handleSaveScript}
                  disabled={savingScript}
                  className="btn-primary flex-1 flex items-center justify-center space-x-2"
                >
                  {savingScript ? (
                    <>
                      <Loader className="w-5 h-5 animate-spin" />
                      <span>{t('browser.script.saving')}</span>
                    </>
                  ) : (
                    <>
                      <Save className="w-5 h-5" />
                        <span>{t('browser.script.save')}</span>
                    </>
                  )}
                </button>
                <button
                  onClick={async () => {
                    await cleanRecordingState()
                  }}
                  disabled={savingScript}
                  className="btn-ghost flex-1"
                >
                  {t('browser.script.cancel')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}


      {/* Instructions Card */}
      <div className="card bg-gray-50 dark:bg-gray-900 border-gray-200 dark:border-gray-700">
        <h3 className="text-lg lg:text-xl font-bold text-gray-900 dark:text-gray-100 mb-4">{t('browser.instructions.title')}</h3>
        <ul className="space-y-3 text-[15px] text-gray-700 dark:text-gray-300">
          <li className="flex items-start space-x-2">
            <span className="text-gray-900 dark:text-gray-100 font-bold">1.</span>
            <span>{t('browser.instructions.step1')}</span>
          </li>
          <li className="flex items-start space-x-2">
            <span className="text-gray-900 dark:text-gray-100 font-bold">2.</span>
            <span>{t('browser.instructions.step2')}</span>
          </li>
          <li className="flex items-start space-x-2">
            <span className="text-gray-900 dark:text-gray-100 font-bold">3.</span>
            <span dangerouslySetInnerHTML={{ __html: t('browser.instructions.step3') }} />
          </li>
          <li className="flex items-start space-x-2">
            <span className="text-gray-900 dark:text-gray-100 font-bold">4.</span>
            <span>{t('browser.instructions.step4')}</span>
          </li>
          <li className="flex items-start space-x-2">
            <span className="text-gray-900 dark:text-gray-100 font-bold">5.</span>
            <span dangerouslySetInnerHTML={{ __html: t('browser.instructions.step5') }} />
          </li>
        </ul>
        
        <div className="mt-5 p-4 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
          <p className="text-sm text-gray-700 dark:text-gray-300" dangerouslySetInnerHTML={{ __html: t('browser.instructions.protection') }} />
        </div>
      </div>

      {/* 浏览器配置模态框 */}
      {showConfigModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4 overflow-y-auto" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-4xl w-full p-6 my-8 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('browser.config.title')}</h3>
              <button
                onClick={() => {
                  setShowConfigModal(false)
                  setEditingConfig(null)
                }}
                className="btn-ghost p-2"
              >
                ✕
              </button>
            </div>

            <div className="space-y-6">
              {/* 配置列表 */}
              <div>
                <h4 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3">{t('browser.config.savedConfigs')}</h4>
                <div className="space-y-2 max-h-40 overflow-y-auto">
                  {configs.length === 0 ? (
                    <div className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">{t('browser.config.noConfigs')}</div>
                  ) : (
                    configs.map((config) => (
                      <div
                        key={config.id}
                        className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-900 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
                      >
                        <div className="flex-1">
                          <div className="flex items-center space-x-2 flex-wrap">
                            <span className="font-medium text-gray-900 dark:text-gray-100">{config.name}</span>
                            {config.is_default && (
                              <span className="px-2 py-0.5 bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 text-xs rounded-full">{t('browser.config.default')}</span>
                            )}
                            {config.use_stealth === true && (
                              <span className="px-2 py-0.5 bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300 text-xs rounded-full">Stealth</span>
                            )}
                            {config.use_stealth === false && (
                              <span className="px-2 py-0.5 bg-orange-100 text-orange-700 text-xs rounded-full">{t('browser.config.stealthDisabled')}</span>
                            )}
                            {config.headless === true && (
                              <span className="px-2 py-0.5 bg-purple-100 text-purple-700 text-xs rounded-full">{t('browser.config.headless')}</span>
                            )}
                            {config.headless === false && (
                              <span className="px-2 py-0.5 bg-gray-100 text-gray-700 text-xs rounded-full">{t('browser.config.headlessDisabled')}</span>
                            )}
                            {config.url_pattern && (
                              <span className="px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded-full font-mono">
                                {config.url_pattern}
                              </span>
                            )}
                          </div>
                          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{config.description}</p>
                        </div>
                        <div className="flex items-center space-x-2">
                          <button
                            onClick={() => {
                              setEditingConfig(config)
                              setConfigForm({
                                name: config.name,
                                description: config.description,
                                url_pattern: config.url_pattern,
                                user_agent: config.user_agent,
                                use_stealth: config.use_stealth,
                                headless: config.headless,
                                launch_args: config.launch_args || [],
                                is_default: config.is_default,
                              })
                            }}
                            className="btn-ghost text-sm px-3 py-1"
                          >
                            {t('browser.config.edit')}
                          </button>
                          {!config.is_default && (
                            <button
                              onClick={() => setDeleteConfirm({ show: true, configId: config.id })}
                              className="btn-ghost text-sm px-3 py-1 text-red-600 hover:bg-red-50"
                            >
                              {t('browser.config.delete')}
                            </button>
                          )}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </div>

              <div className="border-t dark:border-gray-700 pt-6">
                <h4 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
                  {editingConfig ? t('browser.config.editConfig') : t('browser.config.newConfig')}
                </h4>

                {/* 配置名称 */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.configName')} <span className="text-red-500 dark:text-red-400">*</span>
                  </label>
                  <input
                    type="text"
                    value={configForm.name}
                    onChange={(e) => setConfigForm({ ...configForm, name: e.target.value })}
                    placeholder={t('browser.config.configNamePlaceholder')}
                    className="input w-full"
                  />
                </div>

                {/* 配置描述 */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.configDescription')}
                  </label>
                  <textarea
                    value={configForm.description}
                    onChange={(e) => setConfigForm({ ...configForm, description: e.target.value })}
                    placeholder={t('browser.config.configDescriptionPlaceholder')}
                    rows={2}
                    className="input w-full"
                  />
                </div>

                {/* URL模式 */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.urlPattern')}
                  </label>
                  <input
                    type="text"
                    value={configForm.url_pattern}
                    onChange={(e) => setConfigForm({ ...configForm, url_pattern: e.target.value })}
                    placeholder={t('browser.config.urlPatternPlaceholder')}
                    className="input w-full font-mono text-sm"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1" dangerouslySetInnerHTML={{ __html: t('browser.config.urlPatternHint') }} />
                </div>

                {/* User Agent */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.userAgent')}
                  </label>
                  <input
                    type="text"
                    value={configForm.user_agent}
                    onChange={(e) => setConfigForm({ ...configForm, user_agent: e.target.value })}
                    placeholder={t('browser.config.userAgentPlaceholder')}
                    className="input w-full"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('browser.config.userAgentHint')}
                  </p>
                </div>

                {/* Stealth 模式 */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.stealthMode')}
                  </label>
                  <select
                    value={configForm.use_stealth === null ? 'default' : (configForm.use_stealth ? 'enabled' : 'disabled')}
                    onChange={(e) => {
                      const value = e.target.value
                      setConfigForm({
                        ...configForm,
                        use_stealth: value === 'default' ? null : value === 'enabled'
                      })
                    }}
                    className="input w-full"
                  >
                    <option value="default">{t('browser.config.stealthDefault')}</option>
                    <option value="enabled">{t('browser.config.stealthEnabled')}</option>
                    <option value="disabled">{t('browser.config.stealthDisabledOption')}</option>
                  </select>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('browser.config.stealthHint')}
                  </p>
                </div>

                {/* Headless 模式 */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.headlessMode')}
                  </label>
                  <select
                    value={configForm.headless === null ? 'default' : (configForm.headless ? 'enabled' : 'disabled')}
                    onChange={(e) => {
                      const value = e.target.value
                      setConfigForm({
                        ...configForm,
                        headless: value === 'default' ? null : value === 'enabled'
                      })
                    }}
                    className="input w-full"
                  >
                    <option value="default">{t('browser.config.headlessDefault')}</option>
                    <option value="enabled">{t('browser.config.headlessEnabled')}</option>
                    <option value="disabled">{t('browser.config.headlessDisabledOption')}</option>
                  </select>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('browser.config.headlessHint')}
                  </p>
                </div>

                {/* 启动参数 */}
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('browser.config.launchArgs')}
                  </label>
                  <textarea
                    value={configForm.launch_args.join('\n')}
                    onChange={(e) => setConfigForm({
                      ...configForm,
                      launch_args: e.target.value.split('\n').filter(l => l.trim())
                    })}
                    rows={8}
                    className="input w-full font-mono text-sm"
                    placeholder={t('browser.config.launchArgsPlaceholder')}
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {t('browser.config.launchArgsHint')}
                  </p>
                </div>

                {/* 设为默认 */}
                <div className="mb-6">
                  <label className="flex items-center space-x-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={configForm.is_default}
                      onChange={(e) => setConfigForm({ ...configForm, is_default: e.target.checked })}
                      className="w-4 h-4 text-gray-900 dark:text-gray-100 border-gray-300 dark:border-gray-600 rounded focus:ring-gray-900 dark:focus:ring-gray-100"
                    />
                    <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {t('browser.config.setAsDefault')}
                    </span>
                  </label>
                </div>

                {/* 操作按钮 */}
                <div className="flex items-center justify-end space-x-3 pt-4 border-t dark:border-gray-700">
                  <button
                    onClick={() => {
                      setShowConfigModal(false)
                      setEditingConfig(null)
                    }}
                    className="btn-ghost"
                  >
                    {t('browser.config.cancel')}
                  </button>
                  <button
                    onClick={handleSaveConfig}
                    disabled={!configForm.name.trim()}
                    className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {editingConfig ? t('browser.config.update') : t('browser.config.create')}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {showToast && (
        <Toast
          message={message}
          type={toastType}
          onClose={() => setShowToast(false)}
        />
      )}

      {/* Delete Confirmation Dialog */}
      {deleteConfirm.show && (
        <ConfirmDialog
          title={t('browser.config.deleteConfirm')}
          message={t('browser.config.deleteMessage')}
          confirmText={t('browser.config.delete')}
          cancelText={t('browser.config.cancel')}
          onConfirm={handleDeleteConfig}
          onCancel={() => setDeleteConfirm({ show: false, configId: null })}
        />
      )}
    </div>
  )
}
