import { useState, useEffect } from 'react'
import api, { Prompt } from '../api/client'
import { BookText, Plus, Edit2, Trash2, Save, X, Shield, ChevronDown, ChevronUp } from 'lucide-react'
import Toast from '../components/Toast'
import ConfirmDialog from '../components/ConfirmDialog'
import { useLanguage } from '../i18n'

export default function PromptManage() {
  const { t } = useLanguage()
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [isCreating, setIsCreating] = useState(false)
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())
  const [showModal, setShowModal] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; promptId: string | null; prompt: Prompt | null }>({ show: false, promptId: null, prompt: null })
  const [showToast, setShowToast] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastType, setToastType] = useState<'success' | 'error' | 'info'>('info')
  
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    content: '',
  })

  const showMessage = (msg: string, type: 'success' | 'error' | 'info' = 'info') => {
    setToastMessage(msg)
    setToastType(type)
    setShowToast(true)
  }

  useEffect(() => {
    loadPrompts()
  }, [])

  const loadPrompts = async () => {
    try {
      setLoading(true)
      const response = await api.getPrompts()
      setPrompts(response.data.data || [])
    } catch (err) {
      setError(t('prompt.messages.loadError'))
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = () => {
    setIsCreating(true)
    setEditingId(null)
    setFormData({ name: '', description: '', content: '' })
    setShowModal(true)
  }

  const handleEdit = (prompt: Prompt) => {
    setEditingId(prompt.id)
    setIsCreating(false)
    setFormData({
      name: prompt.name,
      description: prompt.description,
      content: prompt.content,
    })
    setShowModal(true)
  }

  const handleCancel = () => {
    setIsCreating(false)
    setEditingId(null)
    setFormData({ name: '', description: '', content: '' })
    setShowModal(false)
  }

  const handleSave = async () => {
    if (!formData.name || !formData.content) {
      alert(t('prompt.messages.fillRequired'))
      return
    }

    try {
      setLoading(true)
      if (isCreating) {
        await api.createPrompt(formData)
      } else if (editingId) {
        await api.updatePrompt(editingId, formData)
      }
      await loadPrompts()
      handleCancel()
    } catch (err) {
      setError(t('prompt.messages.saveError'))
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteConfirm.promptId || !deleteConfirm.prompt) return

    // 系统提示词不能删除
    if (deleteConfirm.prompt.type === 'system') {
      showMessage(t('prompt.messages.systemNoDelete'), 'error')
      setDeleteConfirm({ show: false, promptId: null, prompt: null })
      return
    }

    try {
      setLoading(true)
      await api.deletePrompt(deleteConfirm.promptId)
      await loadPrompts()
      showMessage(t('prompt.messages.deleteSuccess'), 'success')
    } catch (err) {
      showMessage(t('prompt.messages.deleteError'), 'error')
      console.error(err)
    } finally {
      setLoading(false)
      setDeleteConfirm({ show: false, promptId: null, prompt: null })
    }
  }

  const toggleExpand = (id: string) => {
    setExpandedIds(prev => {
      const newSet = new Set(prev)
      if (newSet.has(id)) {
        newSet.delete(id)
      } else {
        newSet.add(id)
      }
      return newSet
    })
  }

  return (
    <>
      <div className="space-y-6 lg:space-y-8 animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl lg:text-3xl font-semibold text-gray-900 dark:text-gray-100">{t('prompt.title')}</h1>
            <p className="mt-2 text-[15px] text-gray-500">
              {t('prompt.subtitle')}
            </p>
          </div>
          {/* <button
            onClick={handleCreate}
            className="btn-primary flex items-center space-x-2"
          >
            <Plus className="w-5 h-5" />
            <span>{t('prompt.create')}</span>
          </button> */}
        </div>

        {/* Error Message */}
        {error && (
          <div className="p-4 lg:p-5 bg-red-50/80 dark:bg-red-900/20 border-2 border-red-200/80 dark:border-red-800 rounded-xl flex items-start space-x-3">
            <svg className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p className="text-red-800 dark:text-red-300 text-[15px]">{error}</p>
          </div>
        )}

        {/* Prompts List */}
        <div className="space-y-5 lg:space-y-6">
        {loading && prompts.length === 0 ? (
            <div className="card text-center py-16">
              <div className="inline-flex items-center justify-center w-14 h-14 bg-gray-100 dark:bg-gray-700 rounded-2xl mb-5 animate-spin">
                <BookText className="w-7 h-7 text-gray-400 dark:text-gray-500" />
            </div>
              <p className="text-[15px] text-gray-500 dark:text-gray-400">{t('prompt.loading')}</p>
          </div>
        ) : prompts.length === 0 ? (
              <div className="card text-center py-20">
                <div className="inline-flex items-center justify-center w-24 h-24 bg-amber-50 dark:bg-amber-900/20 rounded-3xl mb-8">
                  <BookText className="w-12 h-12 text-amber-500 dark:text-amber-400" />
            </div>
                <h3 className="text-2xl lg:text-3xl font-bold text-gray-900 dark:text-gray-100 mb-4">{t('prompt.noTemplates')}</h3>
                <p className="text-gray-600 dark:text-gray-400 mb-8 text-lg">{t('prompt.noTemplatesHint')}</p>
            <button onClick={handleCreate} className="btn-primary inline-flex items-center space-x-2">
              <Plus className="w-5 h-5" />
              <span>{t('prompt.createTemplate')}</span>
            </button>
          </div>
        ) : (
                prompts.map((prompt, idx) => {
                  const isExpanded = expandedIds.has(prompt.id)
                  return (
            <div
              key={prompt.id}
                    className={`group card-hover animate-scale-in ${
                prompt.type === 'system' ? 'border-2 border-blue-200 dark:border-blue-800 bg-blue-50/30 dark:bg-blue-900/20' : ''
              }`}
              style={{animationDelay: `${idx * 0.05}s`}}
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex-1">
                  <div className="flex items-center space-x-3">
                    <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100 group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
                      {prompt.name}
                    </h3>
                    {prompt.type === 'system' && (
                      <span className="inline-flex items-center space-x-1 px-2.5 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 text-xs font-medium rounded-lg border border-blue-200 dark:border-blue-800">
                        <Shield className="w-3.5 h-3.5" />
                        <span>{t('prompt.systemPreset')}</span>
                      </span>
                    )}
                  </div>
                  {prompt.description && (
                    <p className="text-sm text-gray-600 dark:text-gray-400 mt-2 flex items-center space-x-2">
                      <span className="w-1 h-1 bg-gray-400 dark:bg-gray-500 rounded-full"></span>
                      <span>{prompt.description}</span>
                    </p>
                  )}
                </div>
                <div className="flex space-x-2 ml-4">
                  <button
                          onClick={() => toggleExpand(prompt.id)}
                          className="p-2.5 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-xl transition-all hover:scale-110 active:scale-95"
                          title={isExpanded ? t('prompt.collapse') : t('prompt.expand')}
                        >
                          {isExpanded ? <ChevronUp className="w-5 h-5" /> : <ChevronDown className="w-5 h-5" />}
                        </button>
                        <button
                    onClick={() => handleEdit(prompt)}
                          className="p-2.5 text-primary-600 dark:text-primary-400 hover:bg-primary-50 dark:hover:bg-primary-900/30 rounded-xl transition-all hover:scale-110 active:scale-95"
                    title={t('prompt.edit')}
                  >
                    <Edit2 className="w-5 h-5" />
                  </button>
                  <button
                            onClick={() => setDeleteConfirm({ show: true, promptId: prompt.id, prompt })}
                          disabled={prompt.type === 'system'}
                          className={`p-2.5 rounded-xl transition-all hover:scale-110 active:scale-95 ${
                      prompt.type === 'system' 
                        ? 'text-gray-400 dark:text-gray-600 cursor-not-allowed' 
                        : 'text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30'
                    }`}
                    title={prompt.type === 'system' ? t('prompt.systemCannotDelete') : t('prompt.delete')}
                  >
                    <Trash2 className="w-5 h-5" />
                  </button>
                </div>
              </div>
              
                    {isExpanded && (
                      <div className="bg-gradient-to-br from-gray-50 to-white dark:from-gray-800 dark:to-gray-900 rounded-xl p-5 border border-gray-200 dark:border-gray-700 group-hover:border-gray-300 dark:group-hover:border-gray-600 transition-all animate-fade-in mb-4">
                        <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap font-mono leading-relaxed">
                          {prompt.content}
                        </p>
                      </div>
                    )}

                    <div className="flex items-center justify-between pt-4 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400">
                <div className="flex items-center space-x-4">
                  <span className="flex items-center space-x-1">
                    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                    </svg>
                    <span>{t('prompt.created')}: {new Date(prompt.created_at).toLocaleString()}</span>
                  </span>
                  {prompt.updated_at !== prompt.created_at && (
                    <span className="flex items-center space-x-1">
                      <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                      </svg>
                      <span>{t('prompt.updated')}: {new Date(prompt.updated_at).toLocaleString()}</span>
                    </span>
                  )}
                </div>
              </div>
            </div>
                )
              })
        )}
        </div>
      </div>

      {/* Modal for Create/Edit - Outside of space-y container */}
      {showModal && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in" onClick={handleCancel}>
          <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-y-auto animate-scale-in" onClick={(e) => e.stopPropagation()}>
            <div className="sticky top-0 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 rounded-t-2xl">
              <div className="flex items-center justify-between">
                <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                  {isCreating ? t('prompt.addTitle') : t('prompt.editTitle')}
                </h3>
                <button
                  onClick={handleCancel}
                  className="p-2 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-all"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
            </div>

            <div className="px-6 py-6">
              <div className="grid md:grid-cols-2 gap-6">
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('prompt.nameRequired')} <span className="text-red-600">*</span>
                    </label>
                    <input
                      type="text"
                      value={formData.name}
                      onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                      className="input"
                      placeholder={t('prompt.namePlaceholder')}
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      {t('prompt.descriptionOptional')}
                    </label>
                    <input
                      type="text"
                      value={formData.description}
                      onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                      className="input"
                      placeholder={t('prompt.descriptionPlaceholder')}
                    />
                  </div>

                  <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                    <p className="text-sm text-blue-800 dark:text-blue-300">
                      <strong>{t('prompt.tip')}:</strong> {t('prompt.tipContent')}
                    </p>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('prompt.contentRequired')} <span className="text-red-600">*</span>
                  </label>
                  <textarea
                    value={formData.content}
                    onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                    rows={13}
                    className="input resize-none font-mono text-sm"
                    placeholder={t('prompt.contentPlaceholder')}
                  />
                </div>
              </div>
            </div>

            <div className="sticky bottom-0 bg-gray-50 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4 rounded-b-2xl flex justify-end space-x-3">
              <button
                onClick={handleCancel}
                className="btn-secondary flex items-center space-x-2"
              >
                <X className="w-4 h-4" />
                <span>{t('prompt.cancel')}</span>
              </button>
              <button
                onClick={handleSave}
                disabled={loading}
                className="btn-primary flex items-center space-x-2 disabled:opacity-50"
              >
                <Save className="w-4 h-4" />
                <span>{t('prompt.save')}</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {showToast && (
        <Toast
          message={toastMessage}
          type={toastType}
          onClose={() => setShowToast(false)}
        />
      )}

      {/* Delete Confirmation Dialog */}
      {deleteConfirm.show && (
        <ConfirmDialog
          title={t('prompt.deleteConfirmTitle')}
          message={t('prompt.deleteConfirm')}
          confirmText={t('common.delete')}
          cancelText={t('common.cancel')}
          onConfirm={handleDelete}
          onCancel={() => setDeleteConfirm({ show: false, promptId: null, prompt: null })}
        />
      )}
    </>
  )
}
