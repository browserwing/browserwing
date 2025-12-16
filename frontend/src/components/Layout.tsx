import { Outlet, Link, useLocation } from 'react-router-dom'
import { Chrome, FileCode, Brain, BookText, Languages, MessageSquare, Sun, Moon } from 'lucide-react'
import { useLanguage, LANGUAGES } from '../i18n'
import { useState, useRef, useEffect } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { CURRENT_VERSION, fetchLatestVersion, hasNewVersion, isVersionDismissed, dismissVersion, VersionInfo } from '../utils/version'
import VersionUpdateDialog from './VersionUpdateDialog'

export default function Layout() {
  const location = useLocation()
  const { t, language, setLanguage } = useLanguage()
  const { theme, toggleTheme } = useTheme()
  const [showLangMenu, setShowLangMenu] = useState(false)
  const [showUpdateDialog, setShowUpdateDialog] = useState(false)
  const [latestVersionInfo, setLatestVersionInfo] = useState<VersionInfo | null>(null)
  const langMenuRef = useRef<HTMLDivElement>(null)

  const isActive = (path: string) => {
    return location.pathname === path
  }

  const navItems = [
    { path: '/agent', labelKey: 'nav.agent', icon: MessageSquare },
    { path: '/browser', labelKey: 'nav.browser', icon: Chrome },
    { path: '/scripts', labelKey: 'nav.scripts', icon: FileCode },
    { path: '/llm', labelKey: 'nav.llm', icon: Brain },
    { path: '/prompts', labelKey: 'nav.prompts', icon: BookText },
  ]

  // 点击外部关闭语言菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (langMenuRef.current && !langMenuRef.current.contains(event.target as Node)) {
        setShowLangMenu(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // 检查版本更新
  useEffect(() => {
    const checkVersion = async () => {
      const versionInfo = await fetchLatestVersion()
      if (versionInfo && hasNewVersion(CURRENT_VERSION, versionInfo.version)) {
        // 检查用户是否已经关闭过这个版本的更新提示
        if (!isVersionDismissed(versionInfo.version)) {
          setLatestVersionInfo(versionInfo)
          setShowUpdateDialog(true)
        }
      }
    }
    // 延迟2秒后检查，避免影响首次加载
    const timer = setTimeout(checkVersion, 2000)
    return () => clearTimeout(timer)
  }, [])

  const handleDismissUpdate = () => {
    if (latestVersionInfo) {
      dismissVersion(latestVersionInfo.version)
      setShowUpdateDialog(false)
    }
  }

  return (
    <div className="min-h-screen flex flex-col relative bg-gradient-to-br from-gray-50 via-white to-blue-50 dark:from-gray-900 dark:via-gray-900 dark:to-gray-800">
      {/* 背景装饰元素 */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none" style={{ zIndex: 0 }}>
        <div className="absolute -top-24 left-1/4 w-[500px] h-[500px] rounded-full opacity-30 blur-3xl dark:opacity-20" style={{ background: 'radial-gradient(circle, rgba(59, 130, 246, 0.15) 0%, transparent 70%)' }} />
        <div className="absolute -bottom-24 right-1/4 w-[500px] h-[500px] rounded-full opacity-30 blur-3xl dark:opacity-20" style={{ background: 'radial-gradient(circle, rgba(168, 85, 247, 0.15) 0%, transparent 70%)' }} />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[700px] h-[700px] rounded-full opacity-20 blur-3xl dark:opacity-10" style={{ background: 'radial-gradient(circle, rgba(59, 130, 246, 0.1) 0%, rgba(168, 85, 247, 0.1) 100%)' }} />
      </div>

      {/* Header - Notion风格 */}
      <header className="sticky top-0 bg-white/70 dark:bg-gray-900/70 border-b border-gray-200/60 dark:border-gray-700/60" style={{ zIndex: 50, backdropFilter: 'blur(12px)', WebkitBackdropFilter: 'blur(12px)' }}>
        <div className="max-w-[1400px] 2xl:max-w-[1600px] mx-auto px-6 lg:px-10 xl:px-12">
          <div className="flex items-center justify-between h-16 lg:h-18">
            <Link to="/" className="flex items-center space-x-3 group">
              {/* 飞翔的浏览器 Logo */}
              <div className="w-10 h-10 lg:w-11 lg:h-11 bg-gray-900 dark:bg-gray-100 rounded-lg flex items-center justify-center group-hover:bg-gray-800 dark:group-hover:bg-gray-200 transition-all duration-300 group-hover:scale-105">
                <svg className="w-9 h-9 lg:w-10 lg:h-10 text-white dark:text-gray-900" viewBox="0 0 28 28" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <g transform="rotate(-15 14 14)">
                    {/* 浏览器窗口主体 */}
                    <rect x="7" y="9" width="14" height="10" rx="2" stroke="currentColor" strokeWidth="2" fill="none" />
                    {/* 浏览器地址栏 */}
                    <line x1="9" y1="12" x2="19" y2="12" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />

                    {/* 左翅膀 - 三条羽毛 */}
                    <path d="M7 13C5 12 3 11.5 1 12.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
                    <path d="M7 14.5C5.5 14 4 13.5 2.5 14" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
                    <path d="M7 16C6 15.5 5 15.5 4 16" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />

                    {/* 右翅膀 - 三条羽毛 */}
                    <path d="M21 13C23 12 25 11.5 27 12.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
                    <path d="M21 14.5C22.5 14 24 13.5 25.5 14" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
                    <path d="M21 16C22 15.5 23 15.5 24 16" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
                  </g>
                </svg>
              </div>
              <span className="text-xl lg:text-2xl font-semibold text-gray-900 dark:text-gray-100">{t('layout.appName')}</span>
            </Link>
            
            <nav className="flex items-center space-x-2">
              {navItems.map((item) => {
                // const Icon = item.icon
                const active = isActive(item.path)
                return (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={`
                      flex items-center space-x-2 px-4 lg:px-5 py-2.5 rounded-lg text-[15px] lg:text-base font-medium transition-all duration-150
                      ${active
                      ? 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-gray-100'
                      : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-gray-100'
                      }
                    `}
                  >
                    {/* <Icon className="w-4.5 h-4.5" /> */}
                    <span>{t(item.labelKey)}</span>
                  </Link>
                )
              })}

              {/* 深色模式切换按钮 */}
              <button
                onClick={toggleTheme}
                className="flex items-center justify-center w-9 h-9 rounded-lg text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-gray-100 transition-all duration-150"
                title={theme === 'light' ? '切换到深色模式' : '切换到浅色模式'}
                aria-label="Toggle theme"
              >
                {theme === 'light' ? (
                  <Moon className="w-5 h-5" />
                ) : (
                  <Sun className="w-5 h-5" />
                )}
              </button>

              {/* 语言切换 */}
              <div className="relative ml-2" ref={langMenuRef}>
                <button
                  onClick={() => setShowLangMenu(!showLangMenu)}
                  className="flex items-center space-x-1.5 px-3 py-2 rounded-lg text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-gray-100 transition-all duration-150"
                  title="切换语言 / Switch Language"
                >
                  <Languages className="w-4 h-4" />
                  <span className="text-sm font-medium">{LANGUAGES.find(l => l.code === language)?.nativeName}</span>
                </button>

                {showLangMenu && (
                  <div className="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50">
                    {LANGUAGES.map((lang) => (
                      <button
                        key={lang.code}
                        onClick={() => {
                          setLanguage(lang.code)
                          setShowLangMenu(false)
                        }}
                        className={`w-full text-left px-4 py-2 text-sm transition-colors ${language === lang.code
                          ? 'bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100 font-medium'
                          : 'text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700'
                          }`}
                      >
                        {lang.nativeName}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </nav>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-[1400px] 2xl:max-w-[1600px] mx-auto px-6 lg:px-10 xl:px-12 py-4 lg:py-6 w-full min-h-[calc(100vh-10rem)]">
        <Outlet />
      </main>

      {/* Footer - 简洁固定 */}
      <footer className="mt-auto py-4 lg:py-4 border-t border-gray-200/60 dark:border-gray-700/60 bg-white/50 dark:bg-gray-900/50" style={{ backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)' }}>
        <div className="max-w-[1400px] 2xl:max-w-[1600px] mx-auto px-6 lg:px-10 xl:px-12">
          <div className="flex items-center justify-between text-[15px] text-gray-500 dark:text-gray-400">
            <div className="flex items-center space-x-4">
              <p>{t('layout.copyright')}</p>
              <span className="text-gray-400 dark:text-gray-600">•</span>
              <p className="text-sm">{t('layout.version')} {CURRENT_VERSION}</p>
            </div>
            <div className="flex items-center space-x-6">
              <a href="https://github.com/browserwing/browserwing" target="_blank" rel="noopener noreferrer" className="hover:text-gray-900 dark:hover:text-gray-100 transition-colors">{t('layout.github')}</a>
              <a href="https://browserwing.com/forums" target="_blank" rel="noopener noreferrer" className="hover:text-gray-900 dark:hover:text-gray-100 transition-colors">{t('layout.community')}</a>
              <a href="https://browserwing.com/docs" target="_blank" rel="noopener noreferrer" className="hover:text-gray-900 dark:hover:text-gray-100 transition-colors">{t('layout.docs')}</a>
              <a href="https://browserwing.com/donate" target="_blank" rel="noopener noreferrer" className="hover:text-gray-900 dark:hover:text-gray-100 transition-colors">{t('layout.donate')}</a>
            </div>
          </div>
        </div>
      </footer>

      {/* 版本更新弹窗 */}
      {showUpdateDialog && latestVersionInfo && (
        <VersionUpdateDialog
          versionInfo={latestVersionInfo}
          onClose={() => setShowUpdateDialog(false)}
          onDismiss={handleDismissUpdate}
        />
      )}
    </div>
  )
}
