import { X } from 'lucide-react'
import { useLanguage } from '../i18n'

interface ConfirmDialogProps {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  confirmButtonClass?: string
  onConfirm: () => void
  onCancel: () => void
}

export default function ConfirmDialog({
  title,
  message,
  confirmText,
  cancelText,
  confirmButtonClass = 'btn-danger',
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const { t } = useLanguage()

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center animate-fade-in" style={{ marginTop: 0, marginBottom: 0 }}>
      {/* 背景遮罩 */}
      <div
        className="absolute inset-0 bg-black bg-opacity-50 dark:bg-opacity-70 backdrop-blur-sm"
        onClick={onCancel}
      />
      
      {/* 对话框 */}
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-2xl max-w-md w-full mx-4 animate-scale-in">
        {/* 头部 */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-bold text-gray-900 dark:text-gray-100">{title}</h3>
          <button
            onClick={onCancel}
            className="p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
        
        {/* 内容 */}
        <div className="p-6">
          <p className="text-gray-700 dark:text-gray-300">{message}</p>
        </div>
        
        {/* 底部按钮 */}
        <div className="flex items-center justify-end space-x-3 p-6 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 rounded-b-lg">
          <button
            onClick={onCancel}
            className="btn-secondary"
          >
            {cancelText || t('common.cancel')}
          </button>
          <button
            onClick={onConfirm}
            className={confirmButtonClass}
          >
            {confirmText || t('common.confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}
