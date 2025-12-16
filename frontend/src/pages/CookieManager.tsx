import { useState, useEffect, useCallback } from 'react'
import { api } from '../api/client'
import { ArrowLeft, Upload, Trash2, Edit2, Plus, Save, X, Search, Download } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import Toast from '../components/Toast'
import ConfirmDialog from '../components/ConfirmDialog'
import { useLanguage } from '../i18n'

interface Cookie {
  name: string
  value: string
  domain: string
  path: string
  secure: boolean
  httpOnly: boolean
  sameSite?: string
  expires?: number
}

interface CookieStore {
  id: string
  platform: string
  cookies: Cookie[]
  created_at?: string
  updated_at?: string
}

export default function CookieManager() {
  const navigate = useNavigate()
  const { t } = useLanguage()
  const [cookieStore, setCookieStore] = useState<CookieStore | null>(null)
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState('')
  const [showToast, setShowToast] = useState(false)
  const [toastType, setToastType] = useState<'success' | 'error' | 'info'>('info')
  const [showImportDialog, setShowImportDialog] = useState(false)
  const [importJson, setImportJson] = useState('')
  const [importing, setImporting] = useState(false)
  const [editingCookie, setEditingCookie] = useState<Cookie | null>(null)
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; cookieName: string | null }>({ 
    show: false, 
    cookieName: null 
  })

  const showMessage = useCallback((msg: string, type: 'success' | 'error' | 'info' = 'success') => {
    setMessage(msg)
    setToastType(type)
    setShowToast(true)
  }, [])

  useEffect(() => {
    loadCookies()
  }, [])

  const loadCookies = async () => {
    try {
      setLoading(true)
      const response = await api.getCookies('browser')
      setCookieStore(response.data)
    } catch (err: any) {
      if (err.response?.status === 404) {
        // 没有保存的 Cookie，创建空的
        setCookieStore({
          id: 'browser',
          platform: 'browser',
          cookies: []
        })
      } else {
        showMessage(t('cookie.messages.loadError') + ': ' + (err.response?.data?.error || err.message), 'error')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleImport = async () => {
    if (!importJson.trim()) {
      showMessage(t('cookie.messages.inputRequired'), 'error')
      return
    }

    try {
      setImporting(true)
      const parsed = JSON.parse(importJson)
      
      if (!Array.isArray(parsed)) {
        showMessage(t('cookie.messages.jsonArrayError'), 'error')
        return
      }

      // 验证并转换 EditThisCookie 格式
      const cookies: Cookie[] = parsed.map((item: any) => ({
        name: item.name || '',
        value: item.value || '',
        domain: item.domain || '',
        path: item.path || '/',
        secure: item.secure || false,
        httpOnly: item.httpOnly || false,
        sameSite: item.sameSite || 'Lax',
        expires: item.expirationDate || item.expires || undefined
      }))

      // 合并到现有 Cookie（替换同名的）
      const existingCookies = cookieStore?.cookies || []
      const cookieMap = new Map<string, Cookie>()
      
      // 先添加现有的
      existingCookies.forEach(c => {
        const key = `${c.domain}:${c.name}`
        cookieMap.set(key, c)
      })
      
      // 再添加/覆盖新的
      cookies.forEach(c => {
        const key = `${c.domain}:${c.name}`
        cookieMap.set(key, c)
      })

      const mergedCookies = Array.from(cookieMap.values())

      // 保存到数据库
      await api.importBrowserCookies({ cookies: mergedCookies })
      showMessage(t('cookie.messages.importSuccess', { count: cookies.length }), 'success')
      setShowImportDialog(false)
      setImportJson('')
      await loadCookies()
    } catch (err: any) {
      if (err instanceof SyntaxError) {
        showMessage(t('cookie.messages.jsonError') + ': ' + err.message, 'error')
      } else {
        showMessage(err.response?.data?.error || t('cookie.messages.importError'), 'error')
      }
    } finally {
      setImporting(false)
    }
  }

  const handleSaveCookie = async () => {
    if (!editingCookie) return

    if (!editingCookie.name.trim() || !editingCookie.domain.trim()) {
      showMessage(t('cookie.messages.nameRequired'), 'error')
      return
    }

    try {
      const existingCookies = cookieStore?.cookies || []
      let updatedCookies: Cookie[]

      if (editingCookie === cookieStore?.cookies.find(c => 
        c.name === editingCookie.name && c.domain === editingCookie.domain
      )) {
        // 更新现有 Cookie
        updatedCookies = existingCookies.map(c =>
          c.name === editingCookie.name && c.domain === editingCookie.domain
            ? editingCookie
            : c
        )
      } else {
        // 添加新 Cookie
        updatedCookies = [...existingCookies, editingCookie]
      }

      await api.importBrowserCookies({ cookies: updatedCookies })
      showMessage(t('cookie.messages.saveSuccess'), 'success')
      setShowEditDialog(false)
      setEditingCookie(null)
      await loadCookies()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('cookie.messages.saveError'), 'error')
    }
  }

  const handleDeleteCookie = async () => {
    if (!deleteConfirm.cookieName) return

    try {
      const existingCookies = cookieStore?.cookies || []
      const updatedCookies = existingCookies.filter(c => 
        c.name !== deleteConfirm.cookieName
      )

      await api.importBrowserCookies({ cookies: updatedCookies })
      showMessage(t('cookie.messages.deleteSuccess'), 'success')
      setDeleteConfirm({ show: false, cookieName: null })
      await loadCookies()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('cookie.messages.deleteError'), 'error')
    }
  }

  const handleAddNew = () => {
    setEditingCookie({
      name: '',
      value: '',
      domain: '',
      path: '/',
      secure: true,
      httpOnly: false,
      sameSite: 'Lax'
    })
    setShowEditDialog(true)
  }

  const handleExport = () => {
    if (!cookieStore || cookieStore.cookies.length === 0) {
      showMessage(t('cookie.messages.noCookiesToExport'), 'error')
      return
    }

    try {
      // 转换为 EditThisCookie 兼容格式
      const exportCookies = cookieStore.cookies.map(cookie => ({
        name: cookie.name,
        value: cookie.value,
        domain: cookie.domain,
        path: cookie.path,
        secure: cookie.secure,
        httpOnly: cookie.httpOnly,
        sameSite: cookie.sameSite || 'Lax',
        expires: cookie.expires || null,
        session: cookie.expires === undefined
      }))

      const jsonData = JSON.stringify(exportCookies, null, 2)
      const blob = new Blob([jsonData], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `cookies-${new Date().toISOString().slice(0, 10)}.json`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)

      showMessage(t('cookie.messages.exportSuccess', { count: cookieStore.cookies.length }), 'success')
    } catch (err: any) {
      showMessage(t('cookie.messages.exportError') + ': ' + err.message, 'error')
    }
  }

  const filteredCookies = cookieStore?.cookies.filter(cookie =>
    cookie.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    cookie.domain.toLowerCase().includes(searchTerm.toLowerCase()) ||
    cookie.value.toLowerCase().includes(searchTerm.toLowerCase())
  ) || []

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">{t('cookie.messages.loading')}</div>
      </div>
    )
  }

  return (
    <div className="space-y-6 lg:space-y-8 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <button
            onClick={() => navigate('/browser')}
            className="btn-ghost p-2"
            title={t('cookie.back')}
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div>
            <h1 className="text-2xl lg:text-3xl font-bold text-gray-900 dark:text-gray-100">{t('cookie.title')}</h1>
            <p className="text-[15px] text-gray-500 mt-2">
              {t('cookie.subtitle')}
            </p>
          </div>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={() => handleExport()}
            className="btn-secondary flex items-center space-x-2"
            disabled={!cookieStore || cookieStore.cookies.length === 0}
            title={t('cookie.export')}
          >
            <Download className="w-5 h-5" />
            <span>{t('cookie.export')}</span>
          </button>
          <button
            onClick={() => setShowImportDialog(true)}
            className="btn-secondary flex items-center space-x-2"
          >
            <Upload className="w-5 h-5" />
            <span>{t('cookie.import')}</span>
          </button>
          <button
            onClick={handleAddNew}
            className="btn-primary flex items-center space-x-2"
          >
            <Plus className="w-5 h-5" />
            <span>{t('cookie.add')}</span>
          </button>
        </div>
      </div>

      {/* Stats Card */}
      <div className="card">
        <div className="grid grid-cols-3 gap-6 lg:gap-8">
          <div className="space-y-2.5">
            <div className="text-[15px] text-gray-500">{t('cookie.stats.total')}</div>
            <div className="text-2xl lg:text-3xl font-bold text-gray-900 dark:text-gray-100">
              {cookieStore?.cookies.length || 0}
            </div>
          </div>
          <div className="space-y-2.5">
            <div className="text-[15px] text-gray-500">{t('cookie.stats.domains')}</div>
            <div className="text-2xl lg:text-3xl font-bold text-gray-900 dark:text-gray-100">
              {new Set(cookieStore?.cookies.map(c => c.domain)).size || 0}
            </div>
          </div>
          <div className="space-y-2.5">
            <div className="text-[15px] text-gray-500">{t('cookie.stats.lastUpdated')}</div>
            <div className="text-[15px] font-medium text-gray-900 dark:text-gray-100">
              {cookieStore?.updated_at 
                ? new Date(cookieStore.updated_at).toLocaleString('zh-CN')
                : t('cookie.stats.neverUpdated')
              }
            </div>
          </div>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="card">
        <div className="flex items-center space-x-4">
          <Search className="w-5 h-5 text-gray-400" />
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder={t('cookie.search.placeholder')}
            className="input flex-1"
          />
        </div>
      </div>

      {/* Cookie List */}
      <div className="card">
        <h2 className="text-lg lg:text-xl font-semibold text-gray-900 dark:text-gray-100 mb-5">{t('cookie.list.title')}</h2>
        
        {filteredCookies.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-500 dark:text-gray-400 mb-4">
              {searchTerm ? t('cookie.list.emptySearch') : t('cookie.list.empty')}
            </p>
            {!searchTerm && (
              <button
                onClick={() => setShowImportDialog(true)}
                className="btn-primary"
              >
                {t('cookie.list.importNow')}
              </button>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600">
                <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">{t('cookie.table.name')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">{t('cookie.table.domain')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">{t('cookie.table.path')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">{t('cookie.table.value')}</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">{t('cookie.table.attributes')}</th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase">{t('cookie.table.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {filteredCookies.map((cookie, index) => (
                  <tr key={index} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-4 py-3 text-sm font-medium text-gray-900 dark:text-gray-100">
                      {cookie.name}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300 font-mono">
                      {cookie.domain}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400 font-mono">
                      {cookie.path}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400 font-mono max-w-xs truncate">
                      {cookie.value}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <div className="flex items-center space-x-2">
                        {cookie.secure && (
                          <span className="px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 text-xs rounded">
                            Secure
                          </span>
                        )}
                        {cookie.httpOnly && (
                          <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 text-xs rounded">
                            HttpOnly
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-right">
                      <div className="flex items-center justify-end space-x-2">
                        <button
                          onClick={() => {
                            setEditingCookie({ ...cookie })
                            setShowEditDialog(true)
                          }}
                          className="p-2 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded-lg transition-colors"
                          title={t('cookie.action.edit')}
                        >
                          <Edit2 className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => setDeleteConfirm({ show: true, cookieName: cookie.name })}
                          className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 rounded-lg transition-colors"
                          title={t('cookie.action.delete')}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Import Dialog */}
      {showImportDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-3xl w-full p-6 max-h-[90vh] overflow-y-auto">
            <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100 mb-4">{t('cookie.import.title')}</h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {t('cookie.import.jsonLabel')} <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={importJson}
                  onChange={(e) => setImportJson(e.target.value)}
                  placeholder={t('cookie.import.jsonPlaceholder')}
                  rows={12}
                  className="input w-full font-mono text-sm"
                />
              </div>

              <div className="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                <h4 className="text-sm font-semibold text-blue-900 dark:text-blue-200 mb-2">{t('cookie.import.howto.title')}</h4>
                <div className="text-xs text-blue-800 dark:text-blue-300 space-y-2">
                  <div>
                    <strong>{t('cookie.import.howto.step1.title')}</strong>
                    <ol className="list-decimal list-inside ml-2 mt-1 space-y-1">
                      <li>{t('cookie.import.howto.step1.item1')}</li>
                      <li>{t('cookie.import.howto.step1.item2')}</li>
                      <li>{t('cookie.import.howto.step1.item3')}</li>
                    </ol>
                  </div>

                  <div className="pt-2 border-t border-blue-200 dark:border-blue-700">
                    <strong>{t('cookie.import.howto.step2.title')}</strong>
                    <ol className="list-decimal list-inside ml-2 mt-1 space-y-1">
                      <li>{t('cookie.import.howto.step2.item1')}</li>
                      <li>{t('cookie.import.howto.step2.item2')}</li>
                      <li>{t('cookie.import.howto.step2.item3')}</li>
                      <li>{t('cookie.import.howto.step2.item4')}</li>
                      <li>{t('cookie.import.howto.step2.item5')}</li>
                    </ol>
                  </div>

                  <div className="pt-2 border-t border-blue-200 dark:border-blue-700">
                    <strong>{t('cookie.import.howto.tips.title')}</strong>
                    <ul className="list-disc list-inside ml-2 mt-1">
                      <li>{t('cookie.import.howto.tips.item1')}</li>
                      <li>{t('cookie.import.howto.tips.item2')}</li>
                    </ul>
                  </div>
                </div>
              </div>

              <div className="flex items-center space-x-3 pt-4">
                <button
                  onClick={handleImport}
                  disabled={importing || !importJson.trim()}
                  className="btn-primary flex-1 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {importing ? t('cookie.import.importing') : t('cookie.import.button')}
                </button>
                <button
                  onClick={() => {
                    setShowImportDialog(false)
                    setImportJson('')
                  }}
                  disabled={importing}
                  className="btn-ghost flex-1"
                >
                  {t('common.cancel')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Edit/Add Dialog */}
      {showEditDialog && editingCookie && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-2xl w-full p-6 max-h-[90vh] overflow-y-auto">
            <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100 mb-4">
              {editingCookie.name ? t('cookie.editTitle') : t('cookie.addTitle')}
            </h3>

            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {t('cookie.edit.nameLabel')} <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={editingCookie.name}
                    onChange={(e) => setEditingCookie({ ...editingCookie, name: e.target.value })}
                    className="input w-full"
                    placeholder={t('cookie.edit.namePlaceholder')}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {t('cookie.edit.domainLabel')} <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={editingCookie.domain}
                    onChange={(e) => setEditingCookie({ ...editingCookie, domain: e.target.value })}
                    className="input w-full"
                    placeholder={t('cookie.edit.domainPlaceholder')}
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {t('cookie.edit.valueLabel')} <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={editingCookie.value}
                  onChange={(e) => setEditingCookie({ ...editingCookie, value: e.target.value })}
                  className="input w-full font-mono text-sm"
                  rows={3}
                  placeholder={t('cookie.edit.valuePlaceholder')}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {t('cookie.edit.pathLabel')}
                  </label>
                  <input
                    type="text"
                    value={editingCookie.path}
                    onChange={(e) => setEditingCookie({ ...editingCookie, path: e.target.value })}
                    className="input w-full"
                    placeholder={t('cookie.edit.pathPlaceholder')}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {t('cookie.edit.sameSiteLabel')}
                  </label>
                  <select
                    value={editingCookie.sameSite || 'Lax'}
                    onChange={(e) => setEditingCookie({ ...editingCookie, sameSite: e.target.value })}
                    className="input w-full"
                  >
                    <option value="Lax">Lax</option>
                    <option value="Strict">Strict</option>
                    <option value="None">None</option>
                  </select>
                </div>
              </div>

              <div className="flex items-center space-x-6">
                <label className="flex items-center space-x-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={editingCookie.secure}
                    onChange={(e) => setEditingCookie({ ...editingCookie, secure: e.target.checked })}
                    className="w-4 h-4 text-gray-900 border-gray-300 rounded focus:ring-gray-900"
                  />
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('cookie.edit.secureLabel')}</span>
                </label>
                <label className="flex items-center space-x-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={editingCookie.httpOnly}
                    onChange={(e) => setEditingCookie({ ...editingCookie, httpOnly: e.target.checked })}
                    className="w-4 h-4 text-gray-900 border-gray-300 rounded focus:ring-gray-900"
                  />
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('cookie.edit.httpOnlyLabel')}</span>
                </label>
              </div>

              <div className="flex items-center space-x-3 pt-4 border-t dark:border-gray-700">
                <button
                  onClick={handleSaveCookie}
                  className="btn-primary flex-1 flex items-center justify-center space-x-2"
                >
                  <Save className="w-5 h-5" />
                  <span>{t('common.save')}</span>
                </button>
                <button
                  onClick={() => {
                    setShowEditDialog(false)
                    setEditingCookie(null)
                  }}
                  className="btn-ghost flex-1 flex items-center justify-center space-x-2"
                >
                  <X className="w-5 h-5" />
                  <span>{t('common.cancel')}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deleteConfirm.show && (
        <ConfirmDialog
          title={t('cookie.deleteTitle')}
          message={t('cookie.deleteConfirm', { name: deleteConfirm.cookieName || '' })}
          confirmText={t('common.delete')}
          cancelText={t('common.cancel')}
          onConfirm={handleDeleteCookie}
          onCancel={() => setDeleteConfirm({ show: false, cookieName: null })}
        />
      )}

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
