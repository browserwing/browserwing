import { createContext, useContext } from 'react'

export type Language = 'zh-CN' | 'zh-TW' | 'en' | 'es' | 'ja'

export interface LanguageContextType {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: string, params?: Record<string, string | number>) => string
}

export const LanguageContext = createContext<LanguageContextType | undefined>(undefined)

export const useLanguage = () => {
  const context = useContext(LanguageContext)
  if (!context) {
    throw new Error('useLanguage must be used within a LanguageProvider')
  }
  return context
}

export const LANGUAGES = [
  { code: 'zh-CN' as Language, name: '简体中文', nativeName: '简体中文' },
  { code: 'zh-TW' as Language, name: '繁體中文', nativeName: '繁體中文' },
  { code: 'en' as Language, name: 'English', nativeName: 'English' },
  { code: 'es' as Language, name: 'Español', nativeName: 'Español' },
  { code: 'ja' as Language, name: '日本語', nativeName: '日本語' },
]
