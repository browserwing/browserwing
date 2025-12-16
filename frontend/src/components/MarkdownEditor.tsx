import { useState, useEffect } from 'react'
import MDEditor, { commands } from '@uiw/react-md-editor'
import '@uiw/react-md-editor/markdown-editor.css'
import '@uiw/react-markdown-preview/markdown.css'

interface MarkdownEditorProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
}

export default function MarkdownEditor({ value, onChange, placeholder }: MarkdownEditorProps) {
  const [isFullscreen, setIsFullscreen] = useState(false)

  // 处理全屏切换
  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen)
  }

  // 监听 ESC 键退出全屏
  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isFullscreen) {
        setIsFullscreen(false)
      }
    }
    window.addEventListener('keydown', handleEsc)
    return () => window.removeEventListener('keydown', handleEsc)
  }, [isFullscreen])

  // 阻止 body 滚动
  useEffect(() => {
    if (isFullscreen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => {
      document.body.style.overflow = ''
    }
  }, [isFullscreen])

  // 复制按钮命令
  const copyCommand = {
    name: 'copy',
    keyCommand: 'copy',
    buttonProps: { 'aria-label': '复制内容' },
    icon: (
      <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
        <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
      </svg>
    ),
    execute: () => {
      navigator.clipboard.writeText(value).then(() => {
        const btn = document.querySelector('[aria-label="复制内容"]')
        if (btn) {
          const originalTitle = btn.getAttribute('title')
          btn.setAttribute('title', '已复制!')
          setTimeout(() => {
            btn.setAttribute('title', originalTitle || '')
          }, 2000)
        }
      })
    },
  }

  // 自定义工具栏命令 (移除默认全屏按钮)
  const customCommands = [
    commands.bold,
    commands.italic,
    commands.strikethrough,
    commands.hr,
    commands.divider,
    commands.title,
    commands.divider,
    commands.link,
    commands.quote,
    commands.code,
    commands.codeBlock,
    commands.image,
    commands.divider,
    commands.unorderedListCommand,
    commands.orderedListCommand,
    commands.checkedListCommand,
    commands.divider,
    copyCommand,
    commands.divider,
    commands.codeEdit,
    commands.codeLive,
    commands.codePreview,
  ]

  return (
    <div 
      className={`markdown-editor-container ${isFullscreen ? 'fullscreen-mode' : ''}`} 
      data-color-mode="light"
    >
      {/* 自定义全屏按钮 */}
      <button
        onClick={toggleFullscreen}
        className="custom-fullscreen-btn"
        title={isFullscreen ? '退出全屏 (ESC)' : '全屏'}
        type="button"
      >
        {isFullscreen ? (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
            <path d="M5 16h3v3h2v-5H5v2zm3-8H5v2h5V5H8v3zm6 11h2v-3h3v-2h-5v5zm2-11V5h-2v5h5V8h-3z"/>
          </svg>
        ) : (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
            <path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/>
          </svg>
        )}
      </button>

      <div className="markdown-editor-wrapper">
        <MDEditor
          value={value}
          onChange={(val) => onChange(val || '')}
          preview="preview"
          height="100%"
          textareaProps={{
            placeholder: placeholder || '使用 Markdown 格式编写内容...',
          }}
          previewOptions={{
            className: 'markdown-preview-content',
          }}
          commands={customCommands}
          extraCommands={[]}
        />
      </div>
    </div>
  )
}
