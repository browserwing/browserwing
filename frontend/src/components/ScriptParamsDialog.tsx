import { useState, useEffect } from 'react'
import { X } from 'lucide-react'
import { useLanguage } from '../i18n'

interface ScriptParamsDialogProps {
  isOpen: boolean
  parameters: string[]
  onConfirm: (params: Record<string, string>) => void
  onCancel: () => void
  scriptName?: string
}

export default function ScriptParamsDialog({
  isOpen,
  parameters,
  onConfirm,
  onCancel,
  scriptName
}: ScriptParamsDialogProps) {
  const { t } = useLanguage()
  const [paramValues, setParamValues] = useState<Record<string, string>>({})

  // 初始化参数值
  useEffect(() => {
    if (isOpen && parameters.length > 0) {
      const initialValues: Record<string, string> = {}
      parameters.forEach(param => {
        initialValues[param] = ''
      })
      setParamValues(initialValues)
    }
  }, [isOpen, parameters])

  const handleInputChange = (paramName: string, value: string) => {
    setParamValues(prev => ({
      ...prev,
      [paramName]: value
    }))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    
    // 检查是否所有参数都已填写
    const hasEmptyValues = parameters.some(param => !paramValues[param]?.trim())
    if (hasEmptyValues) {
      alert(t('script.params.fillAllRequired') || '请填写所有必需参数')
      return
    }
    
    onConfirm(paramValues)
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4" style={{ marginTop: 0, marginBottom: 0 }}>
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-lg w-full max-h-[90vh] flex flex-col">
        {/* 标题栏 - Fixed Header */}
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {t('script.params.title') || '填写脚本参数'}
            </h2>
            <button
              onClick={onCancel}
              className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
            >
              <X className="w-6 h-6" />
            </button>
          </div>
        </div>

        {/* 内容区 - Scrollable */}
        <form onSubmit={handleSubmit} className="flex flex-col flex-1 overflow-hidden">
          <div className="p-6 overflow-y-auto flex-1">
            <div className="space-y-5">
              {scriptName && (
                <div className="p-4 bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg">
                  <p className="text-sm text-gray-700 dark:text-gray-300">
                    <strong className="font-semibold text-gray-900 dark:text-gray-100">{t('script.params.scriptName') || '脚本'}:</strong> {scriptName}
                  </p>
                </div>
              )}

              <div className="bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg p-4">
                <p className="text-sm text-gray-700 dark:text-gray-300">
                  {t('script.params.description') || '此脚本需要以下参数,请填写具体值:'}
                </p>
              </div>

              {parameters.map((param, index) => (
                <div key={index}>
                  <label className="block text-base font-medium text-gray-900 dark:text-gray-100 mb-2">
                    {param}
                    <span className="text-red-500 dark:text-red-400 ml-1">*</span>
                  </label>
                  <input
                    type="text"
                    value={paramValues[param] || ''}
                    onChange={(e) => handleInputChange(param, e.target.value)}
                    placeholder={`${t('script.params.enter') || '请输入'} ${param}`}
                    className="w-full px-4 py-2.5 text-base border border-gray-300 dark:border-gray-600 rounded-lg 
                             focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-500 focus:border-transparent
                             bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500
                             transition-colors"
                    required
                  />
                </div>
              ))}
            </div>
          </div>

          {/* 按钮区 - Fixed Footer */}
          <div className="p-6 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900">
            <div className="flex items-center justify-end gap-3">
              <button
                type="button"
                onClick={onCancel}
                className="px-5 py-2.5 text-base font-medium text-gray-700 dark:text-gray-300 
                         hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              >
                {t('common.cancel') || '取消'}
              </button>
              <button
                type="submit"
                className="px-5 py-2.5 text-base font-medium text-white 
                         bg-gray-900 dark:bg-gray-700 rounded-lg hover:bg-gray-800 dark:hover:bg-gray-600 
                         transition-colors"
              >
                {t('script.params.execute') || '执行'}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  )
}
