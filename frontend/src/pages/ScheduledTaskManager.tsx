import { useState, useEffect } from 'react'
import { Clock, Plus, Edit2, Trash2, Power, PowerOff, History, Calendar, Timer, Code, ChevronDown, ChevronUp } from 'lucide-react'
import { useLanguage } from '../i18n'
import * as api from '../api/client'
import type { ScheduledTask, TaskExecution, Script, LLMConfig } from '../api/client'
import Toast from '../components/Toast'
import ConfirmDialog from '../components/ConfirmDialog'
import { extractScriptParameters } from '../utils/scriptParamsExtractor'

export default function ScheduledTaskManager() {
  const { t, language } = useLanguage()
  const [activeTab, setActiveTab] = useState<'tasks' | 'executions'>('tasks')
  
  // 任务列表相关
  const [tasks, setTasks] = useState<ScheduledTask[]>([])
  const [loading, setLoading] = useState(false)
  const [currentPage, setCurrentPage] = useState(1)
  const [totalTasks, setTotalTasks] = useState(0)
  const [searchQuery, setSearchQuery] = useState('')
  const pageSize = 20

  // 执行记录相关
  const [executions, setExecutions] = useState<TaskExecution[]>([])
  const [totalExecutions, setTotalExecutions] = useState(0)
  const [executionSearchQuery, setExecutionSearchQuery] = useState('')
  const [successFilter, setSuccessFilter] = useState<'all' | 'success' | 'failed'>('all')
  const [expandedExecutionResults, setExpandedExecutionResults] = useState<Set<string>>(new Set())
  const [executionPage, setExecutionPage] = useState(1)
  const [showDeleteExecutionConfirm, setShowDeleteExecutionConfirm] = useState(false)
  const [executionToDelete, setExecutionToDelete] = useState<string | null>(null)

  // UI状态
  const [showToast, setShowToast] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastType, setToastType] = useState<'success' | 'error' | 'info'>('info')
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [taskToDelete, setTaskToDelete] = useState<string | null>(null)
  
  // 创建/编辑对话框
  const [showTaskDialog, setShowTaskDialog] = useState(false)
  const [editingTask, setEditingTask] = useState<ScheduledTask | null>(null)
  const [taskForm, setTaskForm] = useState({
    name: '',
    description: '',
    enabled: true,
    schedule_type: 'every' as api.ScheduleType,
    schedule_config: '',
    execution_type: 'script' as api.ExecutionType,
    script_id: '',
    script_variables: {} as Record<string, string>,
    agent_prompt: '',
    agent_llm_id: '',
    browser_instance_id: '',
  })

  // 选择器数据
  const [scripts, setScripts] = useState<Script[]>([])
  const [llmConfigs, setLLMConfigs] = useState<LLMConfig[]>([])
  const [selectedScript, setSelectedScript] = useState<Script | null>(null)
  const [scriptParams, setScriptParams] = useState<string[]>([])

  useEffect(() => {
    if (activeTab === 'tasks') {
      loadTasks()
    } else {
      loadExecutions()
    }
  }, [activeTab, currentPage, searchQuery, successFilter, executionPage, executionSearchQuery])

  useEffect(() => {
    loadScripts()
    loadLLMConfigs()
  }, [])

  // 当选择脚本变化时，更新选中的脚本对象和参数列表
  useEffect(() => {
    if (taskForm.script_id) {
      const script = scripts.find(s => s.id === taskForm.script_id)
      setSelectedScript(script || null)
      
      if (script) {
        // 使用 extractScriptParameters 提取脚本参数（包括 ${xxx} 占位符和 variables）
        const params = extractScriptParameters(script)
        setScriptParams(params)
        
        // 初始化变量，保留已有值
        if (params.length > 0) {
          const initialVars: Record<string, string> = {}
          params.forEach(param => {
            // 优先使用已填写的值，其次是 script.variables 中的默认值
            initialVars[param] = taskForm.script_variables[param] || 
                                  (script.variables && script.variables[param]) || ''
          })
          setTaskForm(prev => ({ ...prev, script_variables: initialVars }))
        } else {
          setTaskForm(prev => ({ ...prev, script_variables: {} }))
        }
      }
    } else {
      setSelectedScript(null)
      setScriptParams([])
    }
  }, [taskForm.script_id, scripts])

  const loadTasks = async () => {
    setLoading(true)
    try {
      const data = await api.listScheduledTasks(currentPage, pageSize, searchQuery)
      setTasks(data.tasks || [])
      setTotalTasks(data.total)
    } catch (error: any) {
      showMessage(t(error.response?.data?.error || 'error.getTaskListFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const loadExecutions = async () => {
    setLoading(true)
    try {
      const data = await api.listTaskExecutions(executionPage, pageSize, '', executionSearchQuery, successFilter)
      setExecutions(data.executions || [])
      setTotalExecutions(data.total)
    } catch (error: any) {
      showMessage(t(error.response?.data?.error || 'error.getExecutionsFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const loadScripts = async () => {
    try {
      const response = await api.api.getScripts({ page: 1, page_size: 1000 })
      setScripts(response.data.scripts || [])
    } catch (error) {
      console.error('Failed to load scripts:', error)
    }
  }

  const loadLLMConfigs = async () => {
    try {
      const response = await api.api.listLLMConfigs()
      setLLMConfigs(response.data.configs || [])
    } catch (error) {
      console.error('Failed to load LLM configs:', error)
    }
  }

  const showMessage = (message: string, type: 'success' | 'error' | 'info') => {
    setToastMessage(message)
    setToastType(type)
    setShowToast(true)
  }

  // 将 ISO 时间字符串转换为 datetime-local 格式（本地时间）
  const isoToLocalDatetime = (isoString: string): string => {
    if (!isoString) return ''
    const date = new Date(isoString)
    // 获取本地时间的各个部分
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    return `${year}-${month}-${day}T${hours}:${minutes}`
  }

  // 将 datetime-local 格式（本地时间）转换为 ISO 字符串
  const localDatetimeToISO = (localDatetime: string): string => {
    if (!localDatetime) return ''
    // datetime-local 的值已经是本地时间，直接创建 Date 对象
    const date = new Date(localDatetime)
    return date.toISOString()
  }

  const handleCreateTask = () => {
    setEditingTask(null)
    setTaskForm({
      name: '',
      description: '',
      enabled: true,
      schedule_type: 'every',
      schedule_config: '',
      execution_type: 'script',
      script_id: '',
      script_variables: {},
      agent_prompt: '',
      agent_llm_id: '',
      browser_instance_id: '',
    })
    setShowTaskDialog(true)
  }

  const handleEditTask = (task: ScheduledTask) => {
    setEditingTask(task)
    setTaskForm({
      name: task.name,
      description: task.description || '',
      enabled: task.enabled,
      schedule_type: task.schedule_type,
      schedule_config: task.schedule_config,
      execution_type: task.execution_type,
      script_id: task.script_id || '',
      script_variables: task.script_variables || {},
      agent_prompt: task.agent_prompt || '',
      agent_llm_id: task.agent_llm_id || '',
      browser_instance_id: task.browser_instance_id || '',
    })
    setShowTaskDialog(true)
  }

  const handleSaveTask = async () => {
    try {
      if (editingTask) {
        await api.updateScheduledTask(editingTask.id, taskForm)
        showMessage(t('success.taskUpdated'), 'success')
      } else {
        await api.createScheduledTask(taskForm)
        showMessage(t('success.taskCreated'), 'success')
      }
      setShowTaskDialog(false)
      loadTasks()
    } catch (error: any) {
      showMessage(t(error.response?.data?.error || 'error.saveFailed'), 'error')
    }
  }

  const handleToggleTask = async (task: ScheduledTask) => {
    try {
      await api.toggleScheduledTask(task.id)
      showMessage(t('success.taskToggled'), 'success')
      loadTasks()
    } catch (error: any) {
      showMessage(t(error.response?.data?.error || 'error.operationFailed'), 'error')
    }
  }

  const handleDeleteTask = async () => {
    if (!taskToDelete) return
    try {
      await api.deleteScheduledTask(taskToDelete)
      showMessage(t('success.taskDeleted'), 'success')
      setShowDeleteConfirm(false)
      setTaskToDelete(null)
      loadTasks()
    } catch (error: any) {
      showMessage(t(error.response?.data?.error || 'error.deleteFailed'), 'error')
    }
  }

  const formatDateTime = (dateTime: string | null | undefined) => {
    if (!dateTime) return '-'
    // 根据语言习惯返回日期时间格式
    if (language === 'zh-CN') {
      return new Date(dateTime).toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })
    }
    return new Date(dateTime).toLocaleString()
  }

  const toggleExecutionResult = (executionId: string) => {
    setExpandedExecutionResults(prev => {
      const newSet = new Set(prev)
      if (newSet.has(executionId)) {
        newSet.delete(executionId)
      } else {
        newSet.add(executionId)
      }
      return newSet
    })
  }

  const handleDeleteExecution = async () => {
    if (!executionToDelete) return
    try {
      await api.deleteTaskExecution(executionToDelete)
      showMessage(t('success.executionDeleted'), 'success')
      setShowDeleteExecutionConfirm(false)
      setExecutionToDelete(null)
      loadExecutions()
    } catch (error: any) {
      showMessage(t(error.response?.data?.error || 'error.deleteFailed'), 'error')
    }
  }

  // 渲染调度配置输入框（根据类型提供更好的UI）
  const renderScheduleConfigInput = () => {
    switch (taskForm.schedule_type) {
      case 'at':
        return (
          <div>
            <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300 flex items-center space-x-2">
              <Calendar className="w-4 h-4" />
              <span>{t('task.scheduleConfig')}</span>
            </label>
            <input
              type="datetime-local"
              value={isoToLocalDatetime(taskForm.schedule_config)}
              onChange={(e) => {
                const value = e.target.value
                setTaskForm({ ...taskForm, schedule_config: localDatetimeToISO(value) })
              }}
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
            />
            <p className="mt-1 text-xs text-gray-500">{t('task.scheduleConfig.at.hint')}</p>
          </div>
        )
      case 'every':
        return (
          <div>
            <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300 flex items-center space-x-2">
              <Timer className="w-4 h-4" />
              <span>{t('task.scheduleConfig')}</span>
            </label>
            <div className="flex items-center space-x-2">
              <input
                type="number"
                min="1"
                value={taskForm.schedule_config ? parseInt(taskForm.schedule_config) : ''}
                onChange={(e) => {
                  const value = e.target.value
                  const unit = taskForm.schedule_config.slice(-1) || 'm'
                  setTaskForm({ ...taskForm, schedule_config: value ? `${value}${unit}` : '' })
                }}
                className="flex-1 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                placeholder="5"
              />
              <select
                value={taskForm.schedule_config.slice(-1) || 'm'}
                onChange={(e) => {
                  const num = taskForm.schedule_config ? parseInt(taskForm.schedule_config) : ''
                  setTaskForm({ ...taskForm, schedule_config: num ? `${num}${e.target.value}` : '' })
                }}
                className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
              >
                <option value="s">秒</option>
                <option value="m">分钟</option>
                <option value="h">小时</option>
              </select>
            </div>
            <p className="mt-1 text-xs text-gray-500">{t('task.scheduleConfig.every.hint')}</p>
          </div>
        )
      case 'cron':
        return (
          <div>
            <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300 flex items-center space-x-2">
              <Code className="w-4 h-4" />
              <span>{t('task.scheduleConfig')}</span>
            </label>
            <input
              type="text"
              value={taskForm.schedule_config}
              onChange={(e) => setTaskForm({ ...taskForm, schedule_config: e.target.value })}
              placeholder="0 */5 * * * *"
              className="w-full px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
            />
            <p className="mt-1 text-xs text-gray-500">{t('task.scheduleConfig.cron.hint')}</p>
          </div>
        )
    }
  }

  return (
    <div className="space-y-6 lg:space-y-8 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">{t('task.title')}</h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">{t('task.subtitle')}</p>
        </div>
        {activeTab === 'tasks' && (
          <button
            onClick={handleCreateTask}
            className="flex items-center space-x-2 px-4 py-2 bg-gray-900 hover:bg-gray-800 dark:bg-gray-700 dark:hover:bg-gray-600 text-white rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span>{t('task.create')}</span>
          </button>
        )}
      </div>

      {/* Tabs */}
      <div className="flex space-x-4 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setActiveTab('tasks')}
          className={`pb-3 px-1 border-b-2 transition-colors ${
            activeTab === 'tasks'
              ? 'border-gray-900 dark:border-gray-100 text-gray-900 dark:text-gray-100 font-medium'
              : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'
          }`}
        >
          <Clock className="w-4 h-4 inline mr-2" />
          {t('task.title')}
        </button>
        <button
          onClick={() => setActiveTab('executions')}
          className={`pb-3 px-1 border-b-2 transition-colors ${
            activeTab === 'executions'
              ? 'border-gray-900 dark:border-gray-100 text-gray-900 dark:text-gray-100 font-medium'
              : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'
          }`}
        >
          <History className="w-4 h-4 inline mr-2" />
          {t('task.executions')}
        </button>
      </div>

      {/* Tasks Tab */}
      {activeTab === 'tasks' && (
        <div className="space-y-4">
          {/* Search - 缩小尺寸 */}
          <input
            type="text"
            placeholder={t('common.search')}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full max-w-md px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 placeholder-gray-500 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
          />

          {/* Task List */}
          {loading ? (
            <div className="text-center py-8 text-gray-600 dark:text-gray-400">{t('common.loading')}</div>
          ) : tasks.length === 0 ? (
            <div className="text-center py-12 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <Clock className="w-12 h-12 mx-auto text-gray-600 mb-4" />
              <p className="text-gray-600 dark:text-gray-600">{t('task.noTasks')}</p>
            </div>
          ) : (
            <div className="space-y-3">
              {tasks.map((task) => (
                <div
                  key={task.id}
                  className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-5 border border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-2">
                        <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100">{task.name}</h3>
                        <span
                          className={`px-2 py-0.5 text-xs rounded ${
                            task.enabled
                              ? 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100'
                              : 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
                          }`}
                        >
                          {task.enabled ? t('task.enabled') : t('task.disabled')}
                        </span>
                        <span className="px-2 py-0.5 text-xs rounded bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300">
                          {t(`task.scheduleType.${task.schedule_type}`)}
                        </span>
                        <span className="px-2 py-0.5 text-xs rounded bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300">
                          {t(`task.executionType.${task.execution_type}`)}
                        </span>
                      </div>
                      {task.description && (
                        <p className="text-gray-600 dark:text-gray-400 text-sm mb-3">{task.description}</p>
                      )}
                      <div className="grid grid-cols-2 gap-3 text-sm">
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">{t('task.scheduleConfig')}: </span>
                          <span className="text-gray-900 dark:text-gray-100">{task.schedule_config}</span>
                        </div>
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">{t('task.nextExecution')}: </span>
                          <span className="text-gray-900 dark:text-gray-100">{formatDateTime(task.next_execution_time)}</span>
                        </div>
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">{t('task.lastExecution')}: </span>
                          <span className="text-gray-900 dark:text-gray-100">{formatDateTime(task.last_execution_time)}</span>
                        </div>
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">{t('task.executionCount')}: </span>
                          <span className="text-gray-900 dark:text-gray-100">{task.execution_count}</span>
                          <span className="text-gray-500 mx-2">|</span>
                          <span className="text-gray-500 dark:text-gray-400">{t('task.successCount')}: </span>
                          <span className="text-gray-900 dark:text-gray-100">{task.success_count}</span>
                          <span className="text-gray-500 mx-2">|</span>
                          <span className="text-gray-500 dark:text-gray-400">{t('task.failedCount')}: </span>
                          <span className="text-gray-600 dark:text-gray-400">{task.failed_count}</span>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center space-x-1 ml-4">
                      <button
                        onClick={() => handleToggleTask(task)}
                        className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                        title={task.enabled ? 'Disable' : 'Enable'}
                      >
                        {task.enabled ? (
                          <Power className="w-4 h-4 text-gray-700 dark:text-gray-300" />
                        ) : (
                          <PowerOff className="w-4 h-4 text-gray-400" />
                        )}
                      </button>
                      <button
                        onClick={() => handleEditTask(task)}
                        className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                      >
                        <Edit2 className="w-4 h-4 text-gray-700 dark:text-gray-300" />
                      </button>
                      <button
                        onClick={() => {
                          setTaskToDelete(task.id)
                          setShowDeleteConfirm(true)
                        }}
                        className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                      >
                        <Trash2 className="w-4 h-4 text-gray-700 dark:text-gray-300" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Pagination */}
          {totalTasks > pageSize && (
            <div className="flex justify-center space-x-2">
              <button
                onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                disabled={currentPage === 1}
                className="px-4 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-800"
              >
                {t('common.previous')}
              </button>
              <span className="px-4 py-2 text-sm text-gray-700 dark:text-gray-300">
                {currentPage} / {Math.ceil(totalTasks / pageSize)}
              </span>
              <button
                onClick={() => setCurrentPage((p) => p + 1)}
                disabled={currentPage >= Math.ceil(totalTasks / pageSize)}
                className="px-4 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-800"
              >
                {t('common.next')}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Executions Tab */}
      {activeTab === 'executions' && (
        <div className="space-y-4">
          {/* Filters - 缩小尺寸 */}
          <div className="flex space-x-3">
            <input
              type="text"
              placeholder={t('common.search')}
              value={executionSearchQuery}
              onChange={(e) => setExecutionSearchQuery(e.target.value)}
              className="flex-1 max-w-md px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 placeholder-gray-500 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
            />
            <select
              value={successFilter}
              onChange={(e) => setSuccessFilter(e.target.value as 'all' | 'success' | 'failed')}
              className="px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
            >
              <option value="all">{t('common.all')}</option>
              <option value="success">{t('task.status.success')}</option>
              <option value="failed">{t('task.status.failed')}</option>
            </select>
          </div>

          {/* Execution List */}
          {loading ? (
            <div className="text-center py-8 text-gray-600 dark:text-gray-400">{t('common.loading')}</div>
          ) : executions.length === 0 ? (
            <div className="text-center py-12 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <History className="w-12 h-12 mx-auto text-gray-600 mb-4" />
              <p className="text-gray-600 dark:text-gray-600">{t('task.noExecutions')}</p>
            </div>
          ) : (
            <div className="space-y-3">
              {executions.map((execution) => (
                <div
                  key={execution.id}
                  className="bg-white dark:bg-gray-800 rounded-lg shadow-sm p-5 border border-gray-200 dark:border-gray-700"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-2">
                        <h3 className="text-base font-semibold text-gray-900 dark:text-gray-100">{execution.task_name}</h3>
                        <span
                          className={`px-2 py-0.5 text-xs rounded ${
                            execution.success
                              ? 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100'
                              : 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300'
                          }`}
                        >
                          {execution.success ? t('task.status.success') : t('task.status.failed')}
                        </span>
                      </div>
                      <div className="grid grid-cols-2 gap-3 text-sm">
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">{t('task.startTime')}: </span>
                          <span className="text-gray-900 dark:text-gray-100">{formatDateTime(execution.start_time)}</span>
                        </div>
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">{t('task.duration')}: </span>
                          <span className="text-gray-900 dark:text-gray-100">{execution.duration}ms</span>
                        </div>
                      </div>
                      {execution.message && (
                        <div className="mt-2 text-sm">
                          <span className="font-medium text-gray-700 dark:text-gray-300">{t('task.message')}: </span>
                          <span className="text-gray-600 dark:text-gray-400">{execution.message}</span>
                        </div>
                      )}

                      {/* 结果数据展示 */}
                      {execution.result_data && Object.keys(execution.result_data).length > 0 && (
                        <div className="mt-3 border-t border-gray-200 dark:border-gray-700 pt-3">
                          <button
                            onClick={() => toggleExecutionResult(execution.id)}
                            className="flex items-center space-x-2 text-sm text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 transition-colors"
                          >
                            {expandedExecutionResults.has(execution.id) ? (
                              <ChevronUp className="w-4 h-4" />
                            ) : (
                              <ChevronDown className="w-4 h-4" />
                            )}
                            <span className="font-medium">
                              {expandedExecutionResults.has(execution.id) ? '隐藏结果数据' : '显示结果数据'}
                              <span className="ml-1 text-xs text-gray-500">
                                ({Object.keys(execution.result_data).length} 项)
                              </span>
                            </span>
                          </button>
                          
                          {expandedExecutionResults.has(execution.id) && (
                            <div className="mt-2 bg-gray-50 dark:bg-gray-900 rounded-lg p-3 overflow-auto max-h-96">
                              <pre className="text-xs text-gray-800 dark:text-gray-200 whitespace-pre-wrap break-words font-mono">
                                {JSON.stringify(execution.result_data, null, 2)}
                              </pre>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                    <div className="flex items-center ml-4">
                      <button
                        onClick={() => {
                          setExecutionToDelete(execution.id)
                          setShowDeleteExecutionConfirm(true)
                        }}
                        className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                        title={t('common.delete')}
                      >
                        <Trash2 className="w-4 h-4 text-gray-700 dark:text-gray-300" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Pagination */}
          {totalExecutions > pageSize && (
            <div className="flex justify-center space-x-2">
              <button
                onClick={() => setExecutionPage((p) => Math.max(1, p - 1))}
                disabled={executionPage === 1}
                className="px-4 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-800"
              >
                {t('common.previous')}
              </button>
              <span className="px-4 py-2 text-sm text-gray-700 dark:text-gray-300">
                {executionPage} / {Math.ceil(totalExecutions / pageSize)}
              </span>
              <button
                onClick={() => setExecutionPage((p) => p + 1)}
                disabled={executionPage >= Math.ceil(totalExecutions / pageSize)}
                className="px-4 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 dark:hover:bg-gray-800"
              >
                {t('common.next')}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Task Dialog */}
      {showTaskDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto p-6">
            <h2 className="text-2xl font-bold mb-4 text-gray-900 dark:text-gray-100">
              {editingTask ? t('common.edit') : t('task.create')}
            </h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300">{t('task.name')}</label>
                <input
                  type="text"
                  value={taskForm.name}
                  onChange={(e) => setTaskForm({ ...taskForm, name: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300">{t('task.description')}</label>
                <textarea
                  value={taskForm.description}
                  onChange={(e) => setTaskForm({ ...taskForm, description: e.target.value })}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                  rows={2}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300">{t('task.scheduleType')}</label>
                  <select
                    value={taskForm.schedule_type}
                    onChange={(e) => setTaskForm({ ...taskForm, schedule_type: e.target.value as api.ScheduleType, schedule_config: '' })}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                  >
                    <option value="at">{t('task.scheduleType.at')}</option>
                    <option value="every">{t('task.scheduleType.every')}</option>
                    <option value="cron">{t('task.scheduleType.cron')}</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300">{t('task.executionType')}</label>
                  <select
                    value={taskForm.execution_type}
                    onChange={(e) => setTaskForm({ ...taskForm, execution_type: e.target.value as api.ExecutionType })}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                  >
                    <option value="script">{t('task.executionType.script')}</option>
                    <option value="agent">{t('task.executionType.agent')}</option>
                  </select>
                </div>
              </div>

              {/* 调度配置输入 - 优化UI */}
              {renderScheduleConfigInput()}

              {taskForm.execution_type === 'script' && (
                <>
                  <div>
                    <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300">{t('task.selectScript')}</label>
                    <select
                      value={taskForm.script_id}
                      onChange={(e) => setTaskForm({ ...taskForm, script_id: e.target.value })}
                      className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                    >
                      <option value="">-- {t('task.selectScript')} --</option>
                      {scripts.map((script) => (
                        <option key={script.id} value={script.id}>
                          {script.name}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* 脚本参数输入 */}
                  {scriptParams.length > 0 && (
                    <div className="border border-gray-300 dark:border-gray-600 rounded-lg p-4 space-y-3 bg-gray-50 dark:bg-gray-900">
                      <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300">脚本参数</h4>
                      {scriptParams.map((param) => {
                        const defaultValue = selectedScript?.variables?.[param] || ''
                        return (
                          <div key={param}>
                            <label className="block text-xs font-medium mb-1 text-gray-600 dark:text-gray-400">{param}</label>
                            <input
                              type="text"
                              value={taskForm.script_variables[param] || ''}
                              onChange={(e) => setTaskForm({
                                ...taskForm,
                                script_variables: { ...taskForm.script_variables, [param]: e.target.value }
                              })}
                              placeholder={defaultValue ? `默认值: ${defaultValue}` : `请输入 ${param}`}
                              className="w-full px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                            />
                          </div>
                        )
                      })}
                    </div>
                  )}
                </>
              )}

              {taskForm.execution_type === 'agent' && (
                <>
                  <div>
                    <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300">{t('task.agentPrompt')}</label>
                    <textarea
                      value={taskForm.agent_prompt}
                      onChange={(e) => setTaskForm({ ...taskForm, agent_prompt: e.target.value })}
                      placeholder={t('task.agentPrompt.placeholder')}
                      className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                      rows={3}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1.5 text-gray-700 dark:text-gray-300">{t('task.selectLLM')}</label>
                    <select
                      value={taskForm.agent_llm_id}
                      onChange={(e) => setTaskForm({ ...taskForm, agent_llm_id: e.target.value })}
                      className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-gray-400 dark:focus:ring-gray-500 focus:border-transparent"
                    >
                      <option value="">-- {t('task.selectLLM')} --</option>
                      {llmConfigs.map((llm) => (
                        <option key={llm.id} value={llm.id}>
                          {llm.name}
                        </option>
                      ))}
                    </select>
                  </div>
                </>
              )}

              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="enabled"
                  checked={taskForm.enabled}
                  onChange={(e) => setTaskForm({ ...taskForm, enabled: e.target.checked })}
                  className="rounded border-gray-300 dark:border-gray-600"
                />
                <label htmlFor="enabled" className="text-sm text-gray-700 dark:text-gray-300">
                  {t('task.enabled')}
                </label>
              </div>
            </div>

            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={() => setShowTaskDialog(false)}
                className="px-4 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                onClick={handleSaveTask}
                className="px-4 py-2 text-sm bg-gray-900 hover:bg-gray-800 dark:bg-gray-700 dark:hover:bg-gray-600 text-white rounded-lg transition-colors"
              >
                {t('common.save')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Task Confirmation Dialog */}
      {showDeleteConfirm && (
        <ConfirmDialog
          title={t('common.confirmDelete')}
          message={t('common.confirmDeleteMessage', { name: tasks.find((t) => t.id === taskToDelete)?.name || '' })}
          confirmText={t('common.delete')}
          cancelText={t('common.cancel')}
          onConfirm={handleDeleteTask}
          onCancel={() => setShowDeleteConfirm(false)}
        />
      )}

      {/* Delete Execution Confirmation Dialog */}
      {showDeleteExecutionConfirm && (
        <ConfirmDialog
          title={t('common.confirmDelete')}
          message={t('task.confirmDeleteExecution')}
          confirmText={t('common.delete')}
          cancelText={t('common.cancel')}
          onConfirm={handleDeleteExecution}
          onCancel={() => setShowDeleteExecutionConfirm(false)}
        />
      )}

      {/* Toast */}
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
