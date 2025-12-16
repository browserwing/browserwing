import { ReactNode, useState, useCallback } from 'react'
import { LanguageContext, Language } from './index'
import { translations } from './translations'

const STORAGE_KEY = 'browserwing-language'

interface LanguageProviderProps {
  children: ReactNode
}

export const LanguageProvider = ({ children }: LanguageProviderProps) => {
  const [language, setLanguageState] = useState<Language>(() => {
    // 从 localStorage 读取保存的语言
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved && (saved === 'zh-CN' || saved === 'zh-TW' || saved === 'en' || saved === 'es' || saved === 'ja')) {
      return saved as Language
    }
    
    // 如果没有保存，尝试从浏览器语言检测
    const browserLang = navigator.language
    if (browserLang.startsWith('zh-CN') || browserLang === 'zh') {
      return 'zh-CN'
    } else if (browserLang.startsWith('zh-TW') || browserLang.startsWith('zh-HK')) {
      return 'zh-TW'
    } else if (browserLang.startsWith('es')) {
      return 'es'
    } else if (browserLang.startsWith('ja')) {
      return 'ja'
    } else if (browserLang.startsWith('en')) {
      return 'en'
    }
    
    // 默认简体中文
    return 'zh-CN'
  })

  const setLanguage = useCallback((lang: Language) => {
    setLanguageState(lang)
    localStorage.setItem(STORAGE_KEY, lang)
  }, [])

  const t = useCallback((key: string, params?: Record<string, string | number>): string => {
    const translation = translations[language]
    let text = (translation as any)[key] || key
    
    // 替换参数
    if (params) {
      Object.entries(params).forEach(([paramKey, paramValue]) => {
        text = text.replace(new RegExp(`\\{${paramKey}\\}`, 'g'), String(paramValue))
      })
    }
    
    return text
  }, [language])

  return (
    <LanguageContext.Provider value={{ language, setLanguage, t }}>
      {children}
    </LanguageContext.Provider>
  )
}
