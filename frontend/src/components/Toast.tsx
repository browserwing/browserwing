import { useEffect } from 'react'
import { X, CheckCircle, AlertCircle, Info } from 'lucide-react'

interface ToastProps {
  message: string
  type?: 'success' | 'error' | 'info'
  onClose: () => void
  duration?: number
}

export default function Toast({ message, type = 'info', onClose, duration = 3000 }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onClose()
    }, duration)

    return () => clearTimeout(timer)
  }, [duration, onClose])

  const getIcon = () => {
    switch (type) {
      case 'success':
        return <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
      case 'error':
        return <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
      default:
        return <Info className="w-5 h-5 text-blue-600 dark:text-blue-400" />
    }
  }

  const getStyles = () => {
    switch (type) {
      case 'success':
        return 'bg-white dark:bg-gray-800 border-emerald-200 dark:border-emerald-800 shadow-lg'
      case 'error':
        return 'bg-white dark:bg-gray-800 border-red-200 dark:border-red-800 shadow-lg'
      default:
        return 'bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 shadow-lg'
    }
  }

  return (
    <div className={`fixed top-6 right-6 z-50 animate-slide-in-right ${getStyles()} border rounded-xl p-4 min-w-[320px] max-w-[480px]`}>
      <div className="flex items-start space-x-3">
        <div className="flex-shrink-0 mt-0.5">
          {getIcon()}
        </div>
        <div className="flex-1 text-sm text-gray-900 dark:text-gray-100">
          {message}
        </div>
        <button
          onClick={onClose}
          className="flex-shrink-0 p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors"
        >
          <X className="w-4 h-4 text-gray-500 dark:text-gray-400" />
        </button>
      </div>
    </div>
  )
}
