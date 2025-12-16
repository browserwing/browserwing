import { Link } from 'react-router-dom'
import { Chrome, FileCode, Brain, ArrowRight, Zap, Lock, Globe } from 'lucide-react'
import { useLanguage } from '../i18n'

export default function Dashboard() {
  const { t } = useLanguage()
  return (
    <div className="space-y-20 lg:space-y-24 py-12 lg:py-16">
      {/* Hero Section - Notion风格 */}
      <div className="text-center space-y-6 lg:space-y-8 animate-fade-in">
        <h1 className="text-5xl lg:text-6xl xl:text-7xl font-bold text-gray-900 dark:text-gray-100 tracking-tight">
          {t('dashboard.hero.title')}
        </h1>
        <p className="text-lg lg:text-xl text-gray-600 dark:text-gray-400 max-w-2xl mx-auto">
          {t('dashboard.hero.subtitle')}
        </p>
        <div className="flex items-center justify-center gap-4 pt-4">
          <Link
            to="/browser"
            className="btn-primary inline-flex items-center space-x-2"
          >
            <span>{t('dashboard.hero.start')}</span>
            <ArrowRight className="w-5 h-5" />
          </Link>
          <Link
            to="/scripts"
            className="btn-secondary inline-flex items-center space-x-2"
          >
            <FileCode className="w-5 h-5" />
            <span>{t('dashboard.hero.manageScripts')}</span>
          </Link>
        </div>
      </div>

      {/* Feature Cards - 简洁风格 */}
      <div className="grid md:grid-cols-3 gap-5 lg:gap-6">
        <Link to="/browser" className="card-hover group">
          <div className="flex items-start space-x-4 lg:space-x-5">
            <div className="flex-shrink-0 w-12 h-12 lg:w-14 lg:h-14 bg-gray-900 dark:bg-gray-100 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform">
              <Chrome className="w-6 h-6 lg:w-7 lg:h-7 text-white dark:text-gray-900" />
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-lg lg:text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
                {t('dashboard.features.browser.title')}
              </h3>
              <p className="text-[15px] text-gray-600 dark:text-gray-400 leading-relaxed">
                {t('dashboard.features.browser.desc')}
              </p>
            </div>
          </div>
        </Link>

        <Link to="/scripts" className="card-hover group">
          <div className="flex items-start space-x-4 lg:space-x-5">
            <div className="flex-shrink-0 w-12 h-12 lg:w-14 lg:h-14 bg-gray-900 dark:bg-gray-100 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform">
              <FileCode className="w-6 h-6 lg:w-7 lg:h-7 text-white dark:text-gray-900" />
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-lg lg:text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
                {t('dashboard.features.scripts.title')}
              </h3>
              <p className="text-[15px] text-gray-600 dark:text-gray-400 leading-relaxed">
                {t('dashboard.features.scripts.desc')}
              </p>
            </div>
          </div>
        </Link>

        <Link to="/llm" className="card-hover group">
          <div className="flex items-start space-x-4 lg:space-x-5">
            <div className="flex-shrink-0 w-12 h-12 lg:w-14 lg:h-14 bg-gray-900 dark:bg-gray-100 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform">
              <Brain className="w-6 h-6 lg:w-7 lg:h-7 text-white dark:text-gray-900" />
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-lg lg:text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
                {t('dashboard.features.mcp.title')}
              </h3>
              <p className="text-[15px] text-gray-600 dark:text-gray-400 leading-relaxed">
                {t('dashboard.features.mcp.desc')}
              </p>
            </div>
          </div>
        </Link>
      </div>

      {/* Workflow - 简化版 */}
      <div className="card">
        <h2 className="text-2xl lg:text-3xl font-semibold text-gray-900 dark:text-gray-100 mb-8">{t('dashboard.workflow.title')}</h2>
        <div className="grid md:grid-cols-4 gap-8 lg:gap-10">
          {[
            { num: '1', title: t('dashboard.workflow.step1.title'), desc: t('dashboard.workflow.step1.desc') },
            { num: '2', title: t('dashboard.workflow.step2.title'), desc: t('dashboard.workflow.step2.desc') },
            { num: '3', title: t('dashboard.workflow.step3.title'), desc: t('dashboard.workflow.step3.desc') },
            { num: '4', title: t('dashboard.workflow.step4.title'), desc: t('dashboard.workflow.step4.desc') },
          ].map((step, idx) => (
            <div key={idx} className="group">
              <div className="flex items-center space-x-3 mb-3">
                <div className="w-10 h-10 bg-gray-100 dark:bg-gray-800 rounded-xl flex items-center justify-center text-base font-semibold text-gray-900 dark:text-gray-100 group-hover:bg-gray-900 dark:group-hover:bg-gray-100 group-hover:text-white dark:group-hover:text-gray-900 transition-colors">
                  {step.num}
                </div>
                <h4 className="font-semibold text-lg text-gray-900 dark:text-gray-100">{step.title}</h4>
              </div>
              <p className="text-[15px] text-gray-600 dark:text-gray-400 ml-13">{step.desc}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Features Highlight */}
      <div className="grid md:grid-cols-3 gap-6">
        <div className="card text-center">
          <div className="w-14 h-14 bg-gray-100 dark:bg-gray-700 rounded-xl flex items-center justify-center mx-auto mb-4">
            <Zap className="w-7 h-7 text-gray-600 dark:text-gray-400" />
          </div>
          <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">{t('dashboard.highlights.automation.title')}</h3>
          <p className="text-[15px] text-gray-600 dark:text-gray-400">
            {t('dashboard.highlights.automation.desc')}
          </p>
        </div>

        <div className="card text-center">
          <div className="w-14 h-14 bg-green-100 dark:bg-green-900 rounded-xl flex items-center justify-center mx-auto mb-4">
            <Lock className="w-7 h-7 text-green-600 dark:text-green-400" />
          </div>
          <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">{t('dashboard.highlights.security.title')}</h3>
          <p className="text-[15px] text-gray-600 dark:text-gray-400">
            {t('dashboard.highlights.security.desc')}
          </p>
        </div>

        <div className="card text-center">
          <div className="w-14 h-14 bg-purple-100 dark:bg-purple-900 rounded-xl flex items-center justify-center mx-auto mb-4">
            <Globe className="w-7 h-7 text-purple-600 dark:text-purple-400" />
          </div>
          <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">{t('dashboard.highlights.ai.title')}</h3>
          <p className="text-[15px] text-gray-600 dark:text-gray-400">
            {t('dashboard.highlights.ai.desc')}
          </p>
        </div>
      </div>
    </div>
  )
}

