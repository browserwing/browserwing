import { X } from 'lucide-react'
import { useEffect } from 'react'
import { createPortal } from 'react-dom'

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  children: React.ReactNode
  type?: 'info' | 'warning' | 'danger' | 'success'
}

export function Modal({ isOpen, onClose, title, children, type = 'info' }: ModalProps) {
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = 'unset'
    }
    return () => {
      document.body.style.overflow = 'unset'
    }
  }, [isOpen])

  if (!isOpen) return null

  const bgColors = {
    info: 'bg-blue-50 dark:bg-blue-950',
    warning: 'bg-amber-50 dark:bg-amber-950',
    danger: 'bg-red-50 dark:bg-red-950',
    success: 'bg-emerald-50 dark:bg-emerald-950',
  }

  const modalContent = (
    <div 
      className="fixed inset-0 z-[100] flex items-center justify-center p-4"
      onClick={onClose}
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40 dark:bg-black/60 backdrop-blur-sm"></div>
      
      {/* Modal */}
      <div 
        className="relative bg-white dark:bg-gray-800 rounded-3xl shadow-2xl max-w-lg w-full transform transition-all animate-scale-in overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className={`flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700 ${bgColors[type]}`}>
          <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100">{title}</h3>
          <button
            onClick={onClose}
            className="p-2 hover:bg-white/50 dark:hover:bg-gray-700/50 rounded-xl transition-colors"
          >
            <X className="w-5 h-5 text-gray-500 dark:text-gray-400" />
          </button>
        </div>
        
        {/* Body */}
        <div className="p-6 bg-white dark:bg-gray-800">
          {children}
        </div>
      </div>
    </div>
  )

  return createPortal(modalContent, document.body)
}

interface ConfirmModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  type?: 'warning' | 'danger'
}

export function ConfirmModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = '确认',
  cancelText = '取消',
  type = 'warning',
}: ConfirmModalProps) {
  const handleConfirm = () => {
    onConfirm()
    onClose()
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} type={type}>
      <p className="text-gray-700 dark:text-gray-300 mb-6 leading-relaxed">{message}</p>
      <div className="flex items-center justify-end space-x-3">
        <button onClick={onClose} className="btn-secondary">
          {cancelText}
        </button>
        <button 
          onClick={handleConfirm} 
          className={type === 'danger' ? 'btn-danger' : 'btn-primary'}
        >
          {confirmText}
        </button>
      </div>
    </Modal>
  )
}

interface AlertModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  message: string
  type?: 'info' | 'success' | 'warning' | 'danger'
}

export function AlertModal({
  isOpen,
  onClose,
  title,
  message,
  type = 'info',
}: AlertModalProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} type={type}>
      <p className="text-gray-700 mb-6 leading-relaxed whitespace-pre-line">{message}</p>
      <div className="flex justify-end">
        <button onClick={onClose} className="btn-primary">
          确定
        </button>
      </div>
    </Modal>
  )
}
