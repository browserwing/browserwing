import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface MarkdownRendererProps {
  content: string
  className?: string
}

export default function MarkdownRenderer({ content, className = '' }: MarkdownRendererProps) {
  return (
    <div className={`prose prose-sm dark:prose-invert max-w-none [&_hr]:!my-3 [&_p]:!my-2 [&_br]:!leading-tight ${className}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          // 自定义链接样式
          a: ({ node, ...props }) => (
            <a {...props} className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 underline break-words overflow-wrap-anywhere" target="_blank" rel="noopener noreferrer" />
          ),
          // 自定义代码块样式（改进版）
          code: ({ node, className, children, ...props }: any) => {
            const inline = !className

            if (inline) {
              return (
                <code {...props} className="bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-200 px-1.5 py-0.5 rounded text-sm font-mono">
                  {children}
                </code>
              )
            }

            return (
              <code {...props} className="block bg-gray-900 dark:bg-gray-950 text-gray-100 p-4 rounded-lg text-sm font-mono leading-relaxed overflow-x-auto whitespace-pre-wrap break-words">
                {children}
              </code>
            )
          },
          // 自定义表格样式
          table: ({ node, ...props }) => (
            <div className="overflow-x-auto">
              <table {...props} className="min-w-full divide-y divide-gray-300 dark:divide-gray-700" />
            </div>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
