import { useState, useEffect } from 'react'
import api, { ScriptExecution } from '../api/client'
import { useLanguage } from '../i18n'
import Toast from '../components/Toast'
import ConfirmDialog from '../components/ConfirmDialog'

export default function ScriptExecutionHistory() {
    const { t } = useLanguage()
  const [executions, setExecutions] = useState<ScriptExecution[]>([])
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')
  const [showToast, setShowToast] = useState(false)
  const [toastType, setToastType] = useState<'success' | 'error' | 'info'>('info')
  
  // 分页和过滤
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(20)
  const [totalExecutions, setTotalExecutions] = useState(0)
  const [searchQuery, setSearchQuery] = useState('')
  const [successFilter, setSuccessFilter] = useState<'all' | 'success' | 'failed'>('all')
  
  // 选中的执行记录
  const [selectedExecutions, setSelectedExecutions] = useState<Set<string>>(new Set())
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; executionId: string | null }>({ show: false, executionId: null })
  
  // 展开的执行记录详情
  const [expandedExecutionId, setExpandedExecutionId] = useState<string | null>(null)

  const showMessage = (msg: string, type: 'success' | 'error' | 'info' = 'info') => {
    setMessage(msg)
    setToastType(type)
    setShowToast(true)
  }

  useEffect(() => {
    loadExecutions()
  }, [currentPage, successFilter])

  const loadExecutions = async () => {
    try {
      setLoading(true)
      const params: any = {
        page: currentPage,
        page_size: pageSize,
      }
      
      if (successFilter !== 'all') {
        params.success = successFilter === 'success'
      }
      
      if (searchQuery.trim()) {
        params.search = searchQuery.trim()
      }

      const response = await api.listScriptExecutions(params)
      setExecutions(response.data.executions || [])
      setTotalExecutions(response.data.total || 0)
    } catch (err: any) {
        showMessage(err.response?.data?.error || t('execution.messages.loadError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = () => {
    setCurrentPage(1)
    loadExecutions()
  }

  const handleDeleteExecution = async () => {
    if (!deleteConfirm.executionId) return

    try {
      setLoading(true)
      await api.deleteScriptExecution(deleteConfirm.executionId)
        showMessage(t('execution.messages.deleteSuccess'), 'success')
      await loadExecutions()
    } catch (err: any) {
        showMessage(err.response?.data?.error || t('execution.messages.deleteError'), 'error')
    } finally {
      setLoading(false)
      setDeleteConfirm({ show: false, executionId: null })
    }
  }

  const handleBatchDelete = async () => {
    if (selectedExecutions.size === 0) {
        showMessage(t('execution.messages.selectAtLeastOne'), 'info')
      return
    }

      if (!confirm(t('execution.batchDeleteConfirm', { count: selectedExecutions.size.toString() }))) {
      return
    }

    try {
      setLoading(true)
      const response = await api.batchDeleteScriptExecutions(Array.from(selectedExecutions))
        showMessage(t(response.data.message, { count: response.data.count }), 'success')
      setSelectedExecutions(new Set())
      await loadExecutions()
    } catch (err: any) {
        showMessage(err.response?.data?.error || t('execution.messages.batchDeleteError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const toggleExecutionSelection = (executionId: string) => {
    const newSelected = new Set(selectedExecutions)
    if (newSelected.has(executionId)) {
      newSelected.delete(executionId)
    } else {
      newSelected.add(executionId)
    }
    setSelectedExecutions(newSelected)
  }

  const toggleSelectAll = () => {
    if (selectedExecutions.size === executions.length) {
      setSelectedExecutions(new Set())
    } else {
      setSelectedExecutions(new Set(executions.map(e => e.id)))
    }
  }

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`
    const seconds = Math.floor(ms / 1000)
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}m ${remainingSeconds}s`
  }

  const formatDateTime = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const totalPages = Math.ceil(totalExecutions / pageSize)

  return (
    <div className="space-y-6 lg:space-y-8 animate-fade-in">
      {/* 页头 */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
                  <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('execution.title')}</h1>
                  <p className="mt-1 text-sm text-gray-600">{t('execution.subtitle')}</p>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
        <div className="flex flex-col lg:flex-row gap-4">
          <div className="flex-1">
            <input
              type="text"
                          placeholder={t('execution.search.placeholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              className="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
            />
          </div>
          <div className="flex gap-2">
            <select
              value={successFilter}
              onChange={(e) => setSuccessFilter(e.target.value as any)}
              className="px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
            >
                          <option value="all">{t('execution.filter.allStatus')}</option>
                          <option value="success">{t('execution.filter.success')}</option>
                          <option value="failed">{t('execution.filter.failed')}</option>
            </select>
            <button
              onClick={handleSearch}
              className="px-6 py-2.5 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition-colors"
            >
                          {t('common.search')}
            </button>
          </div>
        </div>
      </div>

      {/* 批量操作 */}
      {selectedExecutions.size > 0 && (
        <div className="bg-gray-50 border border-gray-200 dark:bg-gray-800 dark:border-gray-700 rounded-lg p-4 flex items-center justify-between">
          <span className="text-gray-800 dark:text-gray-200">
                      {t('execution.selected', { count: selectedExecutions.size.toString() })}
          </span>
          <button
            onClick={handleBatchDelete}
            className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
          >
                      {t('execution.batchDelete')}
          </button>
        </div>
      )}

      {/* 执行记录列表 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading && executions.length === 0 ? (
                  <div className="p-12 text-center text-gray-500">{t('execution.loading')}</div>
        ) : executions.length === 0 ? (
                      <div className="p-12 text-center text-gray-500">{t('execution.noRecords')}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-6 py-3 text-left">
                    <input
                      type="checkbox"
                      checked={selectedExecutions.size === executions.length && executions.length > 0}
                      onChange={toggleSelectAll}
                      className="w-4 h-4 text-gray-900 rounded focus:ring-gray-500"
                    />
                  </th>
                                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t('execution.table.scriptName')}</th>
                                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t('execution.table.startTime')}</th>
                                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t('execution.table.duration')}</th>
                                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t('execution.table.steps')}</th>
                                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t('execution.table.status')}</th>
                                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{t('execution.table.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {executions.map((execution) => (
                  <>
                    <tr key={execution.id} className="hover:bg-gray-50">
                      <td className="px-6 py-4">
                        <input
                          type="checkbox"
                          checked={selectedExecutions.has(execution.id)}
                          onChange={() => toggleExecutionSelection(execution.id)}
                          className="w-4 h-4 text-gray-900 rounded focus:ring-gray-500"
                        />
                      </td>
                      <td className="px-6 py-4 text-sm font-medium text-gray-900 dark:text-gray-100">
                        {execution.script_name}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-600">
                        {formatDateTime(execution.start_time)}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-600">
                        {formatDuration(execution.duration)}
                      </td>
                      <td className="px-6 py-4 text-sm">
                        <div className="flex items-center gap-2">
                                    <span className="text-green-600">{t('execution.steps.success', { count: execution.success_steps.toString() })}</span>
                          <span className="text-gray-400">/</span>
                                    <span className="text-red-600">{t('execution.steps.failed', { count: execution.failed_steps.toString() })}</span>
                          <span className="text-gray-400">/</span>
                                    <span className="text-gray-600">{t('execution.steps.total', { count: execution.total_steps.toString() })}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        {execution.success ? (
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                                        {t('execution.status.success')}
                          </span>
                        ) : (
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                                            {t('execution.status.failed')}
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-sm">
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => setExpandedExecutionId(expandedExecutionId === execution.id ? null : execution.id)}
                            className="text-gray-900 hover:text-gray-700 dark:text-gray-300 dark:hover:text-gray-100"
                          >
                                        {expandedExecutionId === execution.id ? t('execution.action.hideDetails') : t('execution.action.viewDetails')}
                          </button>
                          <button
                            onClick={() => setDeleteConfirm({ show: true, executionId: execution.id })}
                            className="text-red-600 hover:text-red-800"
                          >
                                        {t('execution.action.delete')}
                          </button>
                        </div>
                      </td>
                    </tr>
                    {/* 展开的详情 */}
                    {expandedExecutionId === execution.id && (
                      <tr>
                        <td colSpan={7} className="px-6 py-4 bg-gray-50">
                          <div className="space-y-4">
                            <div>
                                            <h4 className="text-sm font-medium text-gray-700 mb-2">{t('execution.details.executionInfo')}</h4>
                              <div className="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                                    <span className="text-gray-600">{t('execution.details.executionId')}</span>
                                  <span className="font-mono text-gray-900 dark:text-gray-100">{execution.id}</span>
                                </div>
                                <div>
                                                    <span className="text-gray-600">{t('execution.details.scriptId')}</span>
                                  <span className="font-mono text-gray-900 dark:text-gray-100">{execution.script_id}</span>
                                </div>
                                <div>
                                                    <span className="text-gray-600">{t('execution.details.endTime')}</span>
                                  <span className="text-gray-900">{formatDateTime(execution.end_time)}</span>
                                </div>
                                <div>
                                                    <span className="text-gray-600">{t('execution.details.message')}</span>
                                  <span className="text-gray-900">{execution.message}</span>
                                </div>
                              </div>
                            </div>

                            {execution.error_msg && (
                              <div>
                                                <h4 className="text-sm font-medium text-red-700 mb-2">{t('execution.details.errorInfo')}</h4>
                                <div className="bg-red-50 border border-red-200 rounded-lg p-3">
                                  <pre className="text-xs text-red-800 whitespace-pre-wrap">{execution.error_msg}</pre>
                                </div>
                              </div>
                            )}

                            {execution.extracted_data && Object.keys(execution.extracted_data).length > 0 && (
                              <div>
                                                <h4 className="text-sm font-medium text-gray-700 mb-2">{t('execution.details.extractedData')}</h4>
                                <div className="bg-white border border-gray-200 rounded-lg p-3">
                                  <pre className="text-xs text-gray-800 whitespace-pre-wrap">
                                    {JSON.stringify(execution.extracted_data, null, 2)}
                                  </pre>
                                </div>
                              </div>
                            )}

                            {execution.video_path && execution.video_path.trim() !== '' && (
                              <div>
                                <h4 className="text-sm font-medium text-gray-700 mb-2">{t('execution.details.executionVideo')}</h4>
                                <div className="bg-white border border-gray-200 rounded-lg p-3">
                                  <img src={execution.video_path} alt="Execution Video" className="max-w-full h-auto rounded-md" />
                                </div>
                              </div>
                            )}
                          </div>
                        </td>
                      </tr>
                    )}
                  </>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* 分页 */}
      {totalPages > 1 && (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-600">
                          {t('execution.pagination.total', {
                              total: totalExecutions.toString(),
                              current: currentPage.toString(),
                              totalPages: totalPages.toString()
                          })}
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                disabled={currentPage === 1}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                              {t('execution.pagination.prev')}
              </button>
              <button
                onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                disabled={currentPage === totalPages}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                              {t('execution.pagination.next')}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Toast 消息 */}
      {showToast && message && (
        <Toast
          message={message}
          type={toastType}
          onClose={() => setShowToast(false)}
        />
      )}

      {/* 删除确认对话框 */}
      {deleteConfirm.show && (
        <ConfirmDialog
                  title={t('execution.deleteConfirm.title')}
                  message={t('execution.deleteConfirm.message')}
                  confirmText={t('execution.deleteConfirm.confirm')}
                  cancelText={t('execution.deleteConfirm.cancel')}
          onConfirm={handleDeleteExecution}
          onCancel={() => setDeleteConfirm({ show: false, executionId: null })}
        />
      )}
    </div>
  )
}
