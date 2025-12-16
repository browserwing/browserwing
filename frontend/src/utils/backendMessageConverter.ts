// 定义后端硬编码消息到i18n键的映射

export const backendMessageToI18nKey: Record<string, string> = {
  // browser相关
  'Browser is already running': 'error.browserAlreadyRunning',
  'Failed to start browser: ': 'error.startBrowserFailed',
  'Browser started successfully': 'success.browserStarted',
  'Browser is not running': 'error.browserNotRunning',
  'Browser stopped successfully': 'success.browserStopped',
  'Failed to stop browser': 'error.stopBrowserFailed',
  'Invalid URL': 'error.invalidUrl',
  'Failed to open URL': 'error.openUrlFailed',
  'Page opened successfully': 'success.pageOpened',
  'Browser not running': 'error.browserNotRunning',
  
  // cookies相关
  'No valid cookies parsed': 'error.noValidCookies',
  'Failed to save cookies: ': 'error.saveCookiesFailed',
  'Cookies imported successfully': 'success.cookiesImported',
  
  // 录制相关
  'Recording started': 'success.recordingStarted',
  'Recording stopped': 'success.recordingStopped',
  
  // 脚本相关
  'Invalid request parameters: ': 'error.invalidParams',
  'Failed to save script: ': 'error.saveScriptFailed',
  'Script saved': 'success.scriptSaved',
  'Script not found': 'error.scriptNotFound',
  'Failed to get script list: ': 'error.getScriptsFailed',
  
  // agent相关
  'Session deleted': 'agent.sessionDeleted',
  'Message content cannot be empty': 'error.messageEmpty',
  'Streaming response not supported': 'error.streamingNotSupported',
  'Failed to send message': 'error.sendMessageFailed',
  'LLM configuration not found': 'error.llmConfigNotFound',
  'Failed to get LLM configuration': 'error.getLlmConfigFailed',
  
  // 其他通用错误
  'Internal Server Error': 'error.internalServerError',
  'Bad Request': 'error.badRequest',
  'Unauthorized': 'error.unauthorized',
  'Forbidden': 'error.forbidden',
  'Not Found': 'error.notFound',
}

// 将后端消息转换为i18n键
export const convertToI18nKey = (message: string): string => {
  // 检查是否完全匹配
  if (backendMessageToI18nKey[message]) {
    return backendMessageToI18nKey[message]
  }
  
  // 检查是否以某个错误消息开头
  for (const [key, value] of Object.entries(backendMessageToI18nKey)) {
    if (message.startsWith(key)) {
      return value
    }
  }
  
  // 如果没有匹配的键，返回原始消息
  return message
}