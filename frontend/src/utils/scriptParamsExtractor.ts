import { Script, ScriptAction } from '../api/client'

/**
 * 从文本中提取所有 ${xxx} 格式的占位符
 */
export function extractPlaceholdersFromText(text: string | undefined | null): string[] {
  if (!text) return []
  
  const placeholderPattern = /\$\{([^}]+)\}/g
  const placeholders: string[] = []
  let match
  
  while ((match = placeholderPattern.exec(text)) !== null) {
    const placeholder = match[1]
    if (!placeholders.includes(placeholder)) {
      placeholders.push(placeholder)
    }
  }
  
  return placeholders
}

/**
 * 从脚本中提取所有占位符参数
 */
export function extractScriptParameters(script: Script): string[] {
  const allPlaceholders = new Set<string>()
  
  // 从 URL 中提取
  extractPlaceholdersFromText(script.url).forEach(p => allPlaceholders.add(p))
  
  // 从每个 action 中提取
  script.actions.forEach((action: ScriptAction) => {
    extractPlaceholdersFromText(action.selector).forEach(p => allPlaceholders.add(p))
    extractPlaceholdersFromText(action.xpath).forEach(p => allPlaceholders.add(p))
    extractPlaceholdersFromText(action.value).forEach(p => allPlaceholders.add(p))
    extractPlaceholdersFromText(action.url).forEach(p => allPlaceholders.add(p))
    extractPlaceholdersFromText(action.js_code).forEach(p => allPlaceholders.add(p))
    
    // 从文件路径中提取
    if (action.file_paths) {
      action.file_paths.forEach((path: string) => {
        extractPlaceholdersFromText(path).forEach(p => allPlaceholders.add(p))
      })
    }
  })
  
  const variables = Array.from(allPlaceholders).sort()
  if (script.variables) {
    Object.keys(script.variables).forEach(variable => {
      if (!variables.includes(variable)) {
        variables.push(variable)
      }
    })
  }
  console.log('variables', variables)
  return variables
}

/**
 * 检查脚本是否包含参数占位符
 */
export function hasScriptParameters(script: Script): boolean {
  return extractScriptParameters(script).length > 0
}
