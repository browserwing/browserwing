import { Link } from 'react-router-dom'
import { Chrome, FileCode, Download, Copy, Check, Settings, ChevronRight, Zap, ArrowRight } from 'lucide-react'
import { useLanguage } from '../i18n'
import { useState } from 'react'

export default function Dashboard() {
  const { t } = useLanguage()
  const [copiedMCP, setCopiedMCP] = useState(false)
  const [downloadingSkill, setDownloadingSkill] = useState(false)

  const mcpConfig = `{
  "mcpServers": {
    "browserwing": {
      "url": "http://${window.location.host}/api/v1/mcp/message"
    }
  }
}`

  const handleCopyMCP = () => {
    navigator.clipboard.writeText(mcpConfig)
    setCopiedMCP(true)
    setTimeout(() => setCopiedMCP(false), 2000)
  }

  const handleDownloadSkill = async () => {
    setDownloadingSkill(true)
    try {
      const response = await fetch('/api/v1/executor/export/skill')
      if (!response.ok) throw new Error('Download failed')
      
      const blob = await response.blob()
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = 'BROWSERWING_SKILL.md'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(link.href)
    } catch (error) {
      console.error('Failed to download SKILL.md:', error)
    } finally {
      setTimeout(() => setDownloadingSkill(false), 1000)
    }
  }

  return (
    <div className="max-w-6xl mx-auto space-y-20 py-12 lg:py-16">
      {/* Hero Section */}
      <div className="text-center space-y-6 animate-fade-in">
        <h1 className="text-5xl lg:text-6xl font-bold text-gray-900 dark:text-gray-100 tracking-tight">
          {t('dashboard.hero.title')}
        </h1>
        <p className="text-lg text-gray-600 dark:text-gray-400 max-w-2xl mx-auto">
          {t('dashboard.hero.subtitle')}
        </p>
      </div>

      {/* 快速集成 - MCP & Skills */}
      <div className="space-y-8">
        <div className="text-center space-y-2">
          <h2 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            {t('dashboard.quickstart.title')}
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            {t('dashboard.quickstart.subtitle')}
          </p>
        </div>
        
        <div className="grid md:grid-cols-2 gap-6">
          {/* MCP Server Config Card */}
          <div className="card group hover:border-gray-400 dark:hover:border-gray-600 transition-colors">
            <div className="flex items-center space-x-3 mb-4">
              <div className="w-10 h-10 bg-gray-100 dark:bg-gray-800 rounded-lg flex items-center justify-center flex-shrink-0">
                <Zap className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {t('dashboard.quickstart.mcp.title')}
                </h3>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {t('dashboard.quickstart.mcp.subtitle')}
                </p>
              </div>
            </div>
            
            <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-4 mb-4 relative border border-gray-200 dark:border-gray-700" style={{ height: '192px' }}>
              <pre className="text-xs text-gray-700 dark:text-gray-300 overflow-x-auto whitespace-pre font-mono">
                {mcpConfig}
              </pre>
              <button
                onClick={handleCopyMCP}
                className="absolute top-2 right-2 p-2 bg-white dark:bg-gray-700 rounded-md border border-gray-200 dark:border-gray-600 hover:border-gray-900 dark:hover:border-gray-400 transition-colors"
                title={t('dashboard.quickstart.mcp.copy')}
              >
                {copiedMCP ? (
                  <Check className="w-4 h-4 text-gray-900 dark:text-gray-100" />
                ) : (
                  <Copy className="w-4 h-4 text-gray-600 dark:text-gray-400" />
                )}
              </button>
            </div>

            <button
              onClick={handleCopyMCP}
              className="w-full btn-primary inline-flex items-center justify-center space-x-2"
            >
              {copiedMCP ? (
                <>
                  <Check className="w-5 h-5" />
                  <span>{t('dashboard.quickstart.mcp.copied')}</span>
                </>
              ) : (
                <>
                  <Copy className="w-5 h-5" />
                  <span>{t('dashboard.quickstart.mcp.copyButton')}</span>
                </>
              )}
            </button>

            <p className="text-xs text-gray-500 dark:text-gray-400 mt-3 text-center">
              {t('dashboard.quickstart.mcp.instruction')}
            </p>
          </div>

          {/* Skills Card */}
          <div className="card group hover:border-gray-400 dark:hover:border-gray-600 transition-colors">
            <div className="flex items-center space-x-3 mb-4">
              <div className="w-10 h-10 bg-gray-100 dark:bg-gray-800 rounded-lg flex items-center justify-center flex-shrink-0">
                <Download className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {t('dashboard.quickstart.skill.title')}
                </h3>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {t('dashboard.quickstart.skill.subtitle')}
                </p>
              </div>
            </div>

            <div className="bg-gray-50 dark:bg-gray-800/50 rounded-lg p-8 mb-4 text-center border border-gray-200 dark:border-gray-700">
              <div className="w-16 h-16 bg-gray-200 dark:bg-gray-700 rounded-lg flex items-center justify-center mx-auto mb-4">
                <FileCode className="w-8 h-8 text-gray-700 dark:text-gray-300" />
              </div>
              <p className="text-sm text-gray-700 dark:text-gray-300 mb-2">
                {t('dashboard.quickstart.skill.description')}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400 font-mono">
                SKILL.md
              </p>
            </div>

            <button
              onClick={handleDownloadSkill}
              disabled={downloadingSkill}
              className="w-full btn-primary inline-flex items-center justify-center space-x-2 disabled:opacity-50"
            >
              <Download className="w-5 h-5" />
              <span>
                {downloadingSkill 
                  ? t('dashboard.quickstart.skill.downloading')
                  : t('dashboard.quickstart.skill.downloadButton')
                }
              </span>
            </button>

            <p className="text-xs text-gray-500 dark:text-gray-400 mt-3 text-center">
              {t('dashboard.quickstart.skill.instruction')}
            </p>
          </div>
        </div>
      </div>

      {/* 3 步快速开始 */}
      <div className="space-y-8">
        <div className="text-center space-y-2">
          <h2 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            {t('dashboard.steps.title')}
          </h2>
        </div>
        
        <div className="grid md:grid-cols-3 gap-6">
          {[
            { 
              num: '1', 
              icon: <Copy className="w-5 h-5" />,
              title: t('dashboard.steps.step1.title'), 
              desc: t('dashboard.steps.step1.desc') 
            },
            { 
              num: '2', 
              icon: <ArrowRight className="w-5 h-5" />,
              title: t('dashboard.steps.step2.title'), 
              desc: t('dashboard.steps.step2.desc') 
            },
            { 
              num: '3', 
              icon: <Zap className="w-5 h-5" />,
              title: t('dashboard.steps.step3.title'), 
              desc: t('dashboard.steps.step3.desc') 
            },
          ].map((step, idx) => (
            <div key={idx} className="card">
              <div className="flex items-start space-x-3 mb-3">
                <div className="w-10 h-10 bg-gray-100 dark:bg-gray-800 rounded-lg flex items-center justify-center flex-shrink-0">
                  <div className="text-gray-700 dark:text-gray-300">
                    {step.icon}
                  </div>
                </div>
                <div className="flex-1">
                  <div className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                    {t('dashboard.steps.step')} {step.num}
                  </div>
                  <h4 className="font-semibold text-gray-900 dark:text-gray-100 mb-2">
                    {step.title}
                  </h4>
                  <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
                    {step.desc}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* 扩展功能 */}
      <div className="space-y-8">
        <div className="text-center space-y-2">
          <h2 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            {t('dashboard.advanced.title')}
          </h2>
          <p className="text-gray-600 dark:text-gray-400">
            {t('dashboard.advanced.subtitle')}
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-6">
          <Link to="/browser" className="card group hover:border-gray-400 dark:hover:border-gray-600 transition-colors">
            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-10 h-10 bg-gray-100 dark:bg-gray-800 rounded-lg flex items-center justify-center">
                <Chrome className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                  {t('dashboard.advanced.browser.title')}
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed mb-3">
                  {t('dashboard.advanced.browser.desc')}
                </p>
                <div className="flex items-center text-gray-500 dark:text-gray-400 text-sm group-hover:text-gray-900 dark:group-hover:text-gray-100 transition-colors">
                  <span>{t('dashboard.advanced.learnMore')}</span>
                  <ChevronRight className="w-4 h-4 ml-1" />
                </div>
              </div>
            </div>
          </Link>

          <Link to="/scripts" className="card group hover:border-gray-400 dark:hover:border-gray-600 transition-colors">
            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-10 h-10 bg-gray-100 dark:bg-gray-800 rounded-lg flex items-center justify-center">
                <FileCode className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                  {t('dashboard.advanced.scripts.title')}
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed mb-3">
                  {t('dashboard.advanced.scripts.desc')}
                </p>
                <div className="flex items-center text-gray-500 dark:text-gray-400 text-sm group-hover:text-gray-900 dark:group-hover:text-gray-100 transition-colors">
                  <span>{t('dashboard.advanced.learnMore')}</span>
                  <ChevronRight className="w-4 h-4 ml-1" />
                </div>
              </div>
            </div>
          </Link>

          <Link to="/llm" className="card group hover:border-gray-400 dark:hover:border-gray-600 transition-colors">
            <div className="flex items-start space-x-3">
              <div className="flex-shrink-0 w-10 h-10 bg-gray-100 dark:bg-gray-800 rounded-lg flex items-center justify-center">
                <Settings className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                  {t('dashboard.advanced.config.title')}
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed mb-3">
                  {t('dashboard.advanced.config.desc')}
                </p>
                <div className="flex items-center text-gray-500 dark:text-gray-400 text-sm group-hover:text-gray-900 dark:group-hover:text-gray-100 transition-colors">
                  <span>{t('dashboard.advanced.learnMore')}</span>
                  <ChevronRight className="w-4 h-4 ml-1" />
                </div>
              </div>
            </div>
          </Link>
        </div>
      </div>
    </div>
  )
}

