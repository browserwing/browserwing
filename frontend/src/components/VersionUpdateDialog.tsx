import { X, Download, ExternalLink } from 'lucide-react'
import { VersionInfo } from '../utils/version'
import { useLanguage } from '../i18n'

interface VersionUpdateDialogProps {
  versionInfo: VersionInfo
  onClose: () => void
  onDismiss: () => void
}

export default function VersionUpdateDialog({ versionInfo, onClose, onDismiss }: VersionUpdateDialogProps) {
  const { t } = useLanguage()

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[100]" onClick={onClose}>
      <div
        className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-md w-full mx-4 overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="bg-gradient-to-r from-blue-500 to-purple-500 px-6 py-4 text-white">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <div className="w-10 h-10 bg-white/20 rounded-lg flex items-center justify-center">
                <Download className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-lg font-semibold">{t('version.newVersionAvailable')}</h3>
                <p className="text-sm text-white/80">Version {versionInfo.version}</p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="text-white/80 hover:text-white transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 py-5">
          <div className="mb-4">
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-1">
              {t('version.releaseDate')}: {new Date(versionInfo.releaseDate).toLocaleDateString()}
            </p>
          </div>

          {versionInfo.features && versionInfo.features.length > 0 && (
            <div className="mb-6">
              <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
                {t('version.whatsNew')}
              </h4>
              <ul className="space-y-2">
                {versionInfo.features.map((feature, index) => (
                  <li key={index} className="flex items-start space-x-2 text-sm text-gray-700 dark:text-gray-300">
                    <span className="text-blue-500 mt-0.5">•</span>
                    <span>{feature}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center space-x-3">
            <a
              href={versionInfo.downloadUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex-1 flex items-center justify-center space-x-2 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2.5 rounded-lg font-medium transition-colors"
            >
              <Download className="w-4 h-4" />
              <span>{t('version.downloadNow')}</span>
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
            <button
              onClick={onDismiss}
              className="flex-1 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 px-4 py-2.5 rounded-lg font-medium transition-colors"
            >
              {t('version.skipThisVersion')}
            </button>
          </div>

          <button
            onClick={onClose}
            className="w-full mt-2 text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 py-2 transition-colors"
          >
            {t('version.remindLater')}
          </button>
        </div>
      </div>
    </div>
  )
}
