import { useState, useEffect, useCallback } from 'react'
import api, { Script, ScriptAction, RecordingConfig, ScriptExecution } from '../api/client'
import { RefreshCw, Play, Trash2, Clock, FileCode, ChevronDown, ChevronUp, Edit2, X, Check, ExternalLink, GripVertical, Download, Upload, CheckSquare, Square, Copy, Tag, Folder, HelpCircle, Clipboard } from 'lucide-react'
import Toast from '../components/Toast'
import ConfirmDialog from '../components/ConfirmDialog'
import ScriptParamsDialog from '../components/ScriptParamsDialog'
import { useLanguage } from '../i18n'
import { extractScriptParameters } from '../utils/scriptParamsExtractor'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'

export default function ScriptManager() {
  const { t } = useLanguage()

  // 标签页状态
  const [activeTab, setActiveTab] = useState<'scripts' | 'executions'>('scripts')

  const [scripts, setScripts] = useState<Script[]>([])
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')
  const [expandedScriptId, setExpandedScriptId] = useState<string | null>(null)
  const [editingScript, setEditingScript] = useState<Script | null>(null)
  const [editingActions, setEditingActions] = useState<ScriptAction[]>([])
  const [extractedData, setExtractedData] = useState<Record<string, any> | null>(null)
  const [showExtractedData, setShowExtractedData] = useState(false)
  const [showToast, setShowToast] = useState(false)
  const [toastType, setToastType] = useState<'success' | 'error' | 'info'>('info')
  const [selectedScripts, setSelectedScripts] = useState<Set<string>>(new Set())
  const [deleteConfirm, setDeleteConfirm] = useState<{ show: boolean; scriptId: string | null }>({ show: false, scriptId: null })

  // 分页和过滤
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(20)
  const [totalScripts, setTotalScripts] = useState(0)
  const [filterGroup, setFilterGroup] = useState<string>('')
  const [filterTag, setFilterTag] = useState<string>('')
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [availableGroups, setAvailableGroups] = useState<string[]>([])
  const [availableTags, setAvailableTags] = useState<string[]>([])

  // 批量操作
  const [showBatchGroupDialog, setShowBatchGroupDialog] = useState(false)
  const [showBatchTagDialog, setShowBatchTagDialog] = useState(false)
  const [batchGroupInput, setBatchGroupInput] = useState('')
  const [batchTagsInput, setBatchTagsInput] = useState('')

  // MCP 配置相关
  const [showMCPConfig, setShowMCPConfig] = useState(false)
  const [mcpConfigScript, setMCPConfigScript] = useState<Script | null>(null)
  const [mcpCommandName, setMCPCommandName] = useState('')
  const [mcpCommandDescription, setMCPCommandDescription] = useState('')
  const [mcpInputSchemaText, setMCPInputSchemaText] = useState('')

  // Tutorial modal
  const [showTutorial, setShowTutorial] = useState(false)
  const [copiedItem, setCopiedItem] = useState<string | null>(null)

  // 录制配置相关
  const [showRecordingConfig, setShowRecordingConfig] = useState(false)
  const [recordingConfig, setRecordingConfig] = useState<RecordingConfig | null>(null)

  // 执行记录相关
  const [executions, setExecutions] = useState<ScriptExecution[]>([])
  const [totalExecutions, setTotalExecutions] = useState(0)
  const [executionSearchQuery, setExecutionSearchQuery] = useState('')
  const [successFilter, setSuccessFilter] = useState<'all' | 'success' | 'failed'>('all')
  const [selectedExecutions, setSelectedExecutions] = useState<Set<string>>(new Set())
  const [executionDeleteConfirm, setExecutionDeleteConfirm] = useState<{ show: boolean; executionId: string | null }>({ show: false, executionId: null })
  const [expandedExecutionId, setExpandedExecutionId] = useState<string | null>(null)

  // 参数对话框相关
  const [showParamsDialog, setShowParamsDialog] = useState(false)
  const [paramsDialogScript, setParamsDialogScript] = useState<Script | null>(null)
  const [scriptParameters, setScriptParameters] = useState<string[]>([])

  // 导入确认相关
  const [showImportConfirm, setShowImportConfirm] = useState(false)
  const [importData, setImportData] = useState<any>(null)
  const [duplicateScriptIds, setDuplicateScriptIds] = useState<string[]>([])

  // 服务器配置
  const [mcpPort, setMcpPort] = useState<string>('8081')

  const showMessage = useCallback((msg: string, type: 'success' | 'error' | 'info' = 'info') => {
    setMessage(msg)
    setToastType(type)
    setShowToast(true)
  }, [])

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  useEffect(() => {
    // 加载服务器配置
    const loadServerConfig = async () => {
      try {
        const response = await api.getServerConfig()
        if (response.data.mcp_http_port) {
          setMcpPort(response.data.mcp_http_port)
        }
      } catch (err) {
        console.error('加载服务器配置失败:', err)
      }
    }
    loadServerConfig()

    if (activeTab === 'scripts') {
      loadScripts()
      loadRecordingConfig()
    } else if (activeTab === 'executions') {
      loadExecutions()
    }
  }, [activeTab, currentPage, filterGroup, filterTag, searchQuery, successFilter])

  const loadRecordingConfig = async () => {
    try {
      const response = await api.getRecordingConfig()
      setRecordingConfig(response.data)
    } catch (err) {
      console.error('加载录制配置失败:', err)
    }
  }

  const loadScripts = async () => {
    try {
      const params: any = {
        page: currentPage,
        page_size: pageSize,
      }
      if (filterGroup) params.group = filterGroup
      if (filterTag) params.tag = filterTag

      const response = await api.getScripts(params)
      let allScripts = response.data.scripts || []

      // 前端搜索过滤
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase()
        allScripts = allScripts.filter(script =>
          script.name.toLowerCase().includes(query) ||
          script.description?.toLowerCase().includes(query) ||
          script.url.toLowerCase().includes(query)
        )
      }

      setScripts(allScripts)
      setTotalScripts(searchQuery.trim() ? allScripts.length : response.data.total || 0)

      const execresponse = await api.listScriptExecutions(params)
      setTotalExecutions(execresponse.data.total || 0)

      // 收集所有分组和标签
      const groups = new Set<string>()
      const tags = new Set<string>()

      allScripts.forEach(script => {
        if (script.group) groups.add(script.group)
        if (script.tags) {
          script.tags.forEach(tag => tags.add(tag))
        }
      })

      setAvailableGroups(Array.from(groups).sort())
      setAvailableTags(Array.from(tags).sort())
    } catch (err) {
      console.error('加载脚本列表失败:', err)
    }
  }

  const handlePlayScript = async (scriptId: string, params?: Record<string, string>) => {
    const script = scripts.find(s => s.id === scriptId)
    if (!script) return

    // 如果没有提供参数,先检查脚本是否需要参数
    if (!params) {
      const requiredParams = extractScriptParameters(script)
      if (requiredParams.length > 0) {
        // 需要参数,显示参数对话框
        setParamsDialogScript(script)
        setScriptParameters(requiredParams)
        setShowParamsDialog(true)
        return
      }
    }

    // 执行脚本
    try {
      setLoading(true)
      setExtractedData(null)
      setShowExtractedData(false)
      const response = await api.playScript(scriptId, params)
      showMessage(t(response.data.message), 'success')

      // 检查是否有抓取的数据
      if (response.data.result && response.data.result.extracted_data) {
        const data = response.data.result.extracted_data
        if (Object.keys(data).length > 0) {
          setExtractedData(data)
          setShowExtractedData(true)
        }
      }
    } catch (err: any) {
      showMessage(t(err.response?.data?.error) || t('script.messages.playError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleParamsDialogConfirm = (params: Record<string, string>) => {
    setShowParamsDialog(false)
    if (paramsDialogScript) {
      handlePlayScript(paramsDialogScript.id, params)
    }
  }

  const handleParamsDialogCancel = () => {
    setShowParamsDialog(false)
    setParamsDialogScript(null)
    setScriptParameters([])
  }

  const handleDeleteScript = async () => {
    if (!deleteConfirm.scriptId) return

    try {
      setLoading(true)
      const response = await api.deleteScript(deleteConfirm.scriptId)
      showMessage(t(response.data.message), 'success')
      await loadScripts()
    } catch (err: any) {
      showMessage(t(err.response?.data?.error) || t('script.messages.deleteError'), 'error')
    } finally {
      setLoading(false)
      setDeleteConfirm({ show: false, scriptId: null })
    }
  }

  const handleOpenMCPConfig = (script: Script) => {
    setMCPConfigScript(script)
    setMCPCommandName(script.mcp_command_name || '')
    setMCPCommandDescription(script.mcp_command_description || '')

    // 加载 input schema，如果存在则格式化为 JSON
    if (script.mcp_input_schema) {
      setMCPInputSchemaText(JSON.stringify(script.mcp_input_schema, null, 2))
    } else {
      // 自动根据脚本中的占位符 ${xxx} 生成输入参数定义
      const placeholderPattern = /\$\{([^}]+)\}/g
      const inputVariables = new Set<string>()

      // 从脚本 URL 和所有 action 的各个字段中提取占位符
      const extractPlaceholders = (text: string | undefined) => {
        if (!text) return
        let match
        while ((match = placeholderPattern.exec(text)) !== null) {
          inputVariables.add(match[1])
        }
      }

      // 检查脚本 URL
      extractPlaceholders(script.url)

      // 检查每个 action 的各个字段
      script.actions.forEach(action => {
        extractPlaceholders(action.selector)
        extractPlaceholders(action.xpath)
        extractPlaceholders(action.value)
        extractPlaceholders(action.url)
        extractPlaceholders(action.js_code)

        // 检查文件路径数组
        if (action.file_paths) {
          action.file_paths.forEach(path => extractPlaceholders(path))
        }
      })

      if (inputVariables.size > 0) {
        const properties: Record<string, any> = {}
        inputVariables.forEach(varName => {
          properties[varName] = {
            type: 'string',
            description: `${varName} ${t('script.param.description')}`
          }
        })

        const autoSchema = {
          type: 'object',
          properties,
          required: []
        }

        setMCPInputSchemaText(JSON.stringify(autoSchema, null, 2))
      } else {
        setMCPInputSchemaText('')
      }
    }
    setShowMCPConfig(true)
  }

  const handleSaveMCPConfig = async () => {
    if (!mcpConfigScript) return

    try {
      setLoading(true)

      // 解析 input schema JSON
      let inputSchema: Record<string, any> | undefined
      if (mcpInputSchemaText.trim()) {
        try {
          inputSchema = JSON.parse(mcpInputSchemaText)
        } catch (err) {
          showMessage(t('script.messages.mcpInvalidJSON'), 'error')
          setLoading(false)
          return
        }
      }

      const response = await api.toggleScriptMCPCommand(mcpConfigScript.id, {
        is_mcp_command: true,
        mcp_command_name: mcpCommandName,
        mcp_command_description: mcpCommandDescription,
        mcp_input_schema: inputSchema,
      })
      showMessage(t(response.data.message), 'success')
      await loadScripts()
      setShowMCPConfig(false)
    } catch (err: any) {
      showMessage(t(err.response?.data?.error) || t('script.messages.mcpSetError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleCancelMCP = async () => {
    if (!mcpConfigScript) return

    try {
      setLoading(true)
      await api.toggleScriptMCPCommand(mcpConfigScript.id, {
        is_mcp_command: false,
        mcp_command_name: mcpConfigScript.mcp_command_name || '',
        mcp_command_description: mcpConfigScript.mcp_command_description || '',
        mcp_input_schema: mcpConfigScript.mcp_input_schema,
      })
      showMessage(t('script.messages.mcpCancelled'), 'success')
      await loadScripts()
      setShowMCPConfig(false)
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('script.messages.updateError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleGenerateMCPConfig = async () => {
    if (!mcpConfigScript) return

    try {
      setLoading(true)
      const response = await api.request<{ result: string }>('POST', `/scripts/${mcpConfigScript.id}/mcp/generate`)

      // 解析 LLM 返回的 JSON
      try {
        // 提取 JSON 内容（可能包含在代码块中）
        let jsonText = response.data.result
        const jsonMatch = jsonText.match(/```(?:json)?\s*([\s\S]*?)```/) || jsonText.match(/\{[\s\S]*\}/)
        if (jsonMatch) {
          jsonText = jsonMatch[1] || jsonMatch[0]
        }

        const config = JSON.parse(jsonText.trim())

        // 自动填充表单
        if (config.command_name) {
          setMCPCommandName(config.command_name)
        }
        if (config.command_description) {
          setMCPCommandDescription(config.command_description)
        }
        if (config.input_schema) {
          setMCPInputSchemaText(JSON.stringify(config.input_schema, null, 2))
        }

        showMessage(t('script.mcp.generateSuccess'), 'success')
      } catch (parseErr) {
        console.error('Failed to parse LLM response:', parseErr)
        showMessage(t('script.mcp.generateParseError'), 'error')
      }
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('script.mcp.generateError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  // 处理复制并显示反馈
  const handleCopyToClipboard = (text: string, itemId: string) => {
    navigator.clipboard.writeText(text)
    setCopiedItem(itemId)
    setTimeout(() => setCopiedItem(null), 2000)
  }

  // 复制脚本的 curl 命令
  const handleCopyCurl = (scriptId: string) => {
    const script = scripts.find(s => s.id === scriptId)
    if (!script) return

    // 检查脚本是否有参数
    const requiredParams = extractScriptParameters(script)

    let curlCommand = `curl -X POST http://${window.location.hostname}:${mcpPort}/api/v1/scripts/${scriptId}/play \\
  -H "Content-Type: application/json"`

    // 如果有参数，添加 data 部分
    if (requiredParams.length > 0) {
      const paramsExample: Record<string, string> = {}
      requiredParams.forEach(param => {
        paramsExample[param] = `<${param}>`
      })

      curlCommand += ` \\
  -d '${JSON.stringify({ params: paramsExample }, null, 2)}'`
    }

    navigator.clipboard.writeText(curlCommand)
    showMessage(t('script.card.curlCopied'), 'success')
  }

  const handleToggleMCP = async (scriptId: string) => {
    try {
      const script = scripts.find(s => s.id === scriptId)
      if (!script) return

      // 无论是否是MCP命令，都打开配置对话框
      handleOpenMCPConfig(script)
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('script.messages.updateError'), 'error')
    }
  }

  const handleEditScript = (script: Script) => {
    setEditingScript(script)
    setEditingActions([...script.actions])
    setExpandedScriptId(script.id) // 自动展开操作列表
  }

  const handleSaveEditedScript = async () => {
    if (!editingScript) return

    try {
      setLoading(true)
      await api.updateScript(editingScript.id, {
        name: editingScript.name,
        description: editingScript.description,
        url: editingScript.url,
        actions: editingActions,
      })
      showMessage(t('script.messages.updateSuccess'), 'success')
      setEditingScript(null)
      setEditingActions([])
      await loadScripts()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('script.messages.updateError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteAction = (index: number) => {
    setEditingActions(editingActions.filter((_, i) => i !== index))
  }

  const handleDuplicateAction = (index: number) => {
    const actionToDuplicate = editingActions[index]
    const duplicatedAction = { ...actionToDuplicate, timestamp: Date.now() }
    const newActions = [...editingActions]
    newActions.splice(index + 1, 0, duplicatedAction)
    setEditingActions(newActions)
    showMessage(t('script.messages.stepsCopied'), 'success')
  }

  const handleUpdateActionValue = (index: number, field: keyof ScriptAction, value: string | number | string[]) => {
    setEditingActions(
      editingActions.map((action, i) =>
        i === index ? { ...action, [field]: value } : action
      )
    )
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event

    if (over && active.id !== over.id) {
      setEditingActions((items) => {
        const oldIndex = items.findIndex((_, i) => i.toString() === active.id)
        const newIndex = items.findIndex((_, i) => i.toString() === over.id)
        return arrayMove(items, oldIndex, newIndex)
      })
    }
  }

  const handleAddAction = (type: string) => {
    const newAction: ScriptAction = {
      type,
      timestamp: Date.now(),
      selector: '',
      xpath: '',
      value: '',
      url: '',
      duration: type === 'sleep' ? 1000 : undefined,
    }

    // 为数据抓取类型设置默认值
    if (type === 'extract_text' || type === 'extract_html' || type === 'extract_attribute') {
      newAction.variable_name = `data_${editingActions.length}`
      newAction.extract_type = type.replace('extract_', '')
      if (type === 'extract_attribute') {
        newAction.attribute_name = 'href'
      }
    }

    // 为执行 JS 类型设置默认值
    if (type === 'execute_js') {
      newAction.variable_name = `result_${editingActions.length}`
      newAction.js_code = 'return document.title;'
    }

    // 为文件上传类型设置默认值
    if (type === 'upload_file') {
      newAction.file_paths = []
      newAction.description = t('script.action.uploadFileDefault')
    }

    // 为滚动类型设置默认值
    if (type === 'scroll') {
      newAction.scroll_x = 0
      newAction.scroll_y = 0
    }

    // 为键盘事件类型设置默认值
    if (type === 'keyboard') {
      newAction.key = 'enter'
      newAction.description = t('script.action.keyboardDefault')
    }

    // 为打开新标签页类型设置默认值
    if (type === 'open_tab') {
      newAction.url = 'https://'
    }

    // 为切换标签页类型设置默认值
    if (type === 'switch_tab') {
      newAction.value = '0'
    }

    setEditingActions([...editingActions, newAction])
  }

  const toggleScriptExpand = (scriptId: string) => {
    setExpandedScriptId(expandedScriptId === scriptId ? null : scriptId)
  }

  const toggleScriptSelection = (scriptId: string) => {
    const newSelected = new Set(selectedScripts)
    if (newSelected.has(scriptId)) {
      newSelected.delete(scriptId)
    } else {
      newSelected.add(scriptId)
    }
    setSelectedScripts(newSelected)
  }

  const toggleSelectAll = () => {
    if (selectedScripts.size === scripts.length) {
      setSelectedScripts(new Set())
    } else {
      setSelectedScripts(new Set(scripts.map(s => s.id)))
    }
  }

  const handleExportScripts = () => {
    if (selectedScripts.size === 0) {
      showMessage(t('script.message.selectAtLeastOne'), 'info')
      return
    }

    const scriptsToExport = scripts.filter(s => selectedScripts.has(s.id)).map(script => {
      // 导出时不包含分组和标签
      const { group, tags, ...scriptData } = script
      return scriptData
    })

    const exportData = {
      version: '1.0',
      exported_at: new Date().toISOString(),
      scripts: scriptsToExport
    }

    const jsonStr = JSON.stringify(exportData, null, 2)
    const blob = new Blob([jsonStr], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `scripts-export-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
    showMessage(t('script.exportSuccess', { count: selectedScripts.size }), 'success')
    setSelectedScripts(new Set())
  }

  const handleImportScripts = () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.json'
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return

      try {
        const text = await file.text()
        const data = JSON.parse(text)

        if (!data.scripts || !Array.isArray(data.scripts)) {
          showMessage(t('script.messages.invalidFormat'), 'error')
          return
        }

        // 检查是否有重复的ID
        const existingIds = new Set(scripts.map(s => s.id))
        const duplicateIds = data.scripts
          .filter((script: any) => script.id && existingIds.has(script.id))
          .map((script: any) => script.name)

        if (duplicateIds.length > 0) {
          // 有重复ID，显示确认对话框
          setImportData(data)
          setDuplicateScriptIds(duplicateIds)
          setShowImportConfirm(true)
        } else {
          // 没有重复ID，直接导入
          await performImport(data, false)
        }
      } catch (err) {
        console.error('解析文件失败:', err)
        showMessage(t('script.messages.invalidFormat'), 'error')
      }
    }
    input.click()
  }

  const performImport = async (data: any, overwrite: boolean) => {
    try {
      setLoading(true)
      let successCount = 0
      let failCount = 0
      const existingIds = new Set(scripts.map(s => s.id))
      const existingNames = new Set(scripts.map(s => s.name))

      for (const script of data.scripts) {
        try {
          if (overwrite && script.id && existingIds.has(script.id)) {
            // 覆盖现有脚本 - 只更新导入数据中存在的字段
            const updateData: any = {
              name: script.name,
              description: script.description || '',
              url: script.url,
              actions: script.actions,
            }
            
            // 如果导入数据包含MCP信息，则更新MCP信息
            if (script.is_mcp_command !== undefined) {
              updateData.is_mcp_command = script.is_mcp_command
            }
            if (script.mcp_command_name !== undefined) {
              updateData.mcp_command_name = script.mcp_command_name
            }
            if (script.mcp_command_description !== undefined) {
              updateData.mcp_command_description = script.mcp_command_description
            }
            if (script.mcp_input_schema !== undefined) {
              updateData.mcp_input_schema = script.mcp_input_schema
            }
            
            await api.updateScript(script.id, updateData)
            successCount++
          } else {
            // 创建新脚本
            // 检查名称是否重复，重复则添加后缀
            const scriptName = existingNames.has(script.name)
              ? script.name + t('script.imported')
              : script.name

            const scriptID = existingIds.has(script.id)
              ? ''
              : script.id

            const createData: any = {
              id: scriptID,
              name: scriptName,
              description: script.description || '',
              url: script.url,
              actions: script.actions,
              tags: script.tags || [],
              can_publish: script.can_publish || false,
              can_fetch: script.can_fetch || false,
            }
            
            // 如果导入数据包含MCP信息，则设置MCP信息
            if (script.is_mcp_command !== undefined) {
              createData.is_mcp_command = script.is_mcp_command
            }
            if (script.mcp_command_name !== undefined) {
              createData.mcp_command_name = script.mcp_command_name
            }
            if (script.mcp_command_description !== undefined) {
              createData.mcp_command_description = script.mcp_command_description
            }
            if (script.mcp_input_schema !== undefined) {
              createData.mcp_input_schema = script.mcp_input_schema
            }
            
            await api.createScript(createData)
            successCount++
          }
        } catch (err) {
          console.error('导入脚本失败:', script.name, err)
          failCount++
        }
      }

      await loadScripts()

      if (failCount === 0) {
        showMessage(t('script.importSuccess', { count: successCount }), 'success')
      } else {
        showMessage(t('script.importPartial', { success: successCount, failed: failCount }), 'info')
      }
    } catch (err) {
      console.error('导入失败:', err)
      showMessage(t('script.messages.invalidFormat'), 'error')
    } finally {
      setLoading(false)
      setShowImportConfirm(false)
      setImportData(null)
      setDuplicateScriptIds([])
    }
  }

  const handleImportOverwrite = () => {
    if (importData) {
      performImport(importData, true)
    }
  }

  const handleImportCreateNew = () => {
    if (importData) {
      performImport(importData, false)
    }
  }

  const handleBatchSetGroup = async () => {
    if (selectedScripts.size === 0) {
      showMessage(t('script.messages.selectAtLeastOne'), 'info')
      return
    }

    if (!batchGroupInput.trim()) {
      showMessage(t('script.messages.groupNameRequired'), 'error')
      return
    }

    try {
      setLoading(true)
      const response = await api.batchSetGroup(Array.from(selectedScripts), batchGroupInput.trim())
      showMessage(t(response.data.message, { count: response.data.count }), 'success')
      setShowBatchGroupDialog(false)
      setBatchGroupInput('')
      setSelectedScripts(new Set())
      await loadScripts()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('script.messages.batchGroupError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleBatchAddTags = async () => {
    if (selectedScripts.size === 0) {
      showMessage(t('script.messages.selectAtLeastOne'), 'info')
      return
    }

    const tags = batchTagsInput.split(',').map(t => t.trim()).filter(t => t)
    if (tags.length === 0) {
      showMessage(t('script.messages.tagsRequired'), 'error')
      return
    }

    try {
      setLoading(true)
      const response = await api.batchAddTags(Array.from(selectedScripts), tags)
      showMessage(t(response.data.message, { count: response.data.count }), 'success')
      setShowBatchTagDialog(false)
      setBatchTagsInput('')
      setSelectedScripts(new Set())
      await loadScripts()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('script.messages.batchTagsError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleBatchDelete = async () => {
    if (selectedScripts.size === 0) {
      showMessage(t('script.messages.selectAtLeastOne'), 'info')
      return
    }

    if (!confirm(t('script.batchDeleteConfirm', { count: selectedScripts.size }))) {
      return
    }

    try {
      setLoading(true)
      const response = await api.batchDeleteScripts(Array.from(selectedScripts))
      showMessage(t(response.data.message, { count: response.data.count }), 'success')
      setSelectedScripts(new Set())
      await loadScripts()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('script.messages.batchDeleteError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleSaveRecordingConfig = async () => {
    if (!recordingConfig) return

    try {
      setLoading(true)
      await api.updateRecordingConfig(recordingConfig)
      showMessage(t('script.messages.recordingConfigUpdated'), 'success')
      setShowRecordingConfig(false)
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('script.messages.recordingConfigError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  // 执行记录相关函数
  const loadExecutions = async () => {
    try {
      setLoading(true)
      const params: any = {
        page: currentPage,
        page_size: pageSize,
      }

      if (successFilter !== 'all') {
        params.success = successFilter === 'success'
      }

      if (executionSearchQuery.trim()) {
        params.search = executionSearchQuery.trim()
      }

      const response = await api.listScriptExecutions(params)
      setExecutions(response.data.executions || [])
      setTotalExecutions(response.data.total || 0)
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('execution.messages.loadError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleSearchExecutions = () => {
    setCurrentPage(1)
    loadExecutions()
  }

  const handleDeleteExecution = async () => {
    if (!executionDeleteConfirm.executionId) return

    try {
      setLoading(true)
      await api.deleteScriptExecution(executionDeleteConfirm.executionId)
      showMessage(t('execution.messages.deleteSuccess'), 'success')
      await loadExecutions()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('execution.messages.deleteError'), 'error')
    } finally {
      setLoading(false)
      setExecutionDeleteConfirm({ show: false, executionId: null })
    }
  }

  const handleBatchDeleteExecutions = async () => {
    if (selectedExecutions.size === 0) {
      showMessage(t('execution.messages.selectAtLeastOne'), 'info')
      return
    }

    if (!confirm(t('execution.batchDeleteConfirm', { count: selectedExecutions.size.toString() }))) {
      return
    }

    try {
      setLoading(true)
      const response = await api.batchDeleteScriptExecutions(Array.from(selectedExecutions))
      showMessage(t('execution.messages.batchDeleteSuccess', { message: response.data.message }), 'success')
      setSelectedExecutions(new Set())
      await loadExecutions()
    } catch (err: any) {
      showMessage(err.response?.data?.error || t('execution.messages.batchDeleteError'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const toggleExecutionSelection = (executionId: string) => {
    const newSelected = new Set(selectedExecutions)
    if (newSelected.has(executionId)) {
      newSelected.delete(executionId)
    } else {
      newSelected.add(executionId)
    }
    setSelectedExecutions(newSelected)
  }

  const toggleSelectAllExecutions = () => {
    if (selectedExecutions.size === executions.length) {
      setSelectedExecutions(new Set())
    } else {
      setSelectedExecutions(new Set(executions.map(e => e.id)))
    }
  }

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`
    const seconds = Math.floor(ms / 1000)
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}m ${remainingSeconds}s`
  }

  const formatDateTime = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const totalPages = activeTab === 'scripts'
    ? Math.ceil(totalScripts / pageSize)
    : Math.ceil(totalExecutions / pageSize)

  return (
    <div className="space-y-6 lg:space-y-8 animate-fade-in">
      {/* Header */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl lg:text-3xl font-bold text-gray-900 dark:text-gray-100">{t('script.title')}</h1>
            <p className="text-[15px] text-gray-600 dark:text-gray-400 mt-2">
              {activeTab === 'scripts'
                ? `${t('script.subtitle')} (${t('script.total', { count: totalScripts })})`
                : t('execution.subtitle')
              }
            </p>
          </div>
          {activeTab === 'scripts' && (
            <div className="flex items-center space-x-3">
              <button
                onClick={() => setShowRecordingConfig(true)}
                className="btn-secondary flex items-center space-x-2"
                disabled={loading}
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
                <span>{t('script.recordingConfig.title')}</span>
              </button>
              <button
                onClick={handleImportScripts}
                className="btn-secondary flex items-center space-x-2"
                disabled={loading}
              >
                <Upload className="w-5 h-5" />
                <span>{t('script.import')}</span>
              </button>
              <button
                onClick={loadScripts}
                className="btn-secondary flex items-center space-x-2"
                disabled={loading}
              >
                <RefreshCw className={`w-5 h-5 ${loading ? 'animate-spin' : ''}`} />
                <span>{t('common.refresh')}</span>
              </button>
              <button
                onClick={() => setShowTutorial(true)}
                className="btn-secondary flex items-center space-x-2"
                disabled={loading}
              >
                <HelpCircle className="w-5 h-5" />
                <span>{t('script.tutorial.title')}</span>
              </button>
            </div>
          )}
          {activeTab === 'executions' && (
            <button
              onClick={loadExecutions}
              className="btn-secondary flex items-center space-x-2"
              disabled={loading}
            >
              <RefreshCw className={`w-5 h-5 ${loading ? 'animate-spin' : ''}`} />
              <span>{t('common.refresh')}</span>
            </button>
          )}
        </div>

        {/* 标签页切换 */}
        <div className="border-b border-gray-200 dark:border-gray-700">
          <nav className="-mb-px flex space-x-8">
            <button
              onClick={() => { setActiveTab('scripts'); setCurrentPage(1); }}
              className={`py-3 px-1 border-b-2 font-medium text-sm transition-colors ${activeTab === 'scripts'
                ? 'border-gray-900 text-gray-900 dark:text-gray-100 dark:border-gray-100'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:hover:text-gray-300 dark:hover:border-gray-700'
                }`}
            >
              <div className="flex items-center space-x-2">
                <FileCode className="w-4 h-4" />
                <span>{t('script.title')}</span>
                <span className="ml-2 py-0.5 px-2 rounded-full text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400">
                  {totalScripts}
                </span>
              </div>
            </button>
            <button
              onClick={() => { setActiveTab('executions'); setCurrentPage(1); }}
              className={`py-3 px-1 border-b-2 font-medium text-sm transition-colors ${activeTab === 'executions'
                ? 'border-gray-900 text-gray-900 dark:text-gray-100 dark:border-gray-100'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:hover:text-gray-300 dark:hover:border-gray-700'
                }`}
            >
              <div className="flex items-center space-x-2">
                <Clock className="w-4 h-4" />
                <span>{t('execution.title')}</span>
                <span className="ml-2 py-0.5 px-2 rounded-full text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400">
                  {totalExecutions}
                </span>
              </div>
            </button>
          </nav>
        </div>

        {/* 脚本列表的过滤和批量操作栏 */}
        {activeTab === 'scripts' && (
          <div className="flex items-center justify-between bg-gray-50 dark:bg-gray-900 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
            <div className="flex items-center space-x-4">
              {/* 搜索框 */}
              <div className="relative">
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => { setSearchQuery(e.target.value); setCurrentPage(1); }}
                  placeholder={t('script.search.placeholder')}
                  className="pl-3 pr-8 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100 w-64 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                />
                {searchQuery && (
                  <button
                    onClick={() => { setSearchQuery(''); setCurrentPage(1); }}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300"
                  >
                    <X className="w-4 h-4" />
                  </button>
                )}
              </div>

              {/* 分组过滤 */}
              <div className="flex items-center space-x-2">
                <select
                  value={filterGroup}
                  onChange={(e) => { setFilterGroup(e.target.value); setCurrentPage(1); }}
                  className="px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                >
                  <option value="">{t('script.filter.allGroups')}</option>
                  {availableGroups.map(group => (
                    <option key={group} value={group}>{group}</option>
                  ))}
                </select>
              </div>

              {/* 标签过滤 */}
              <div className="flex items-center space-x-2">
                <select
                  value={filterTag}
                  onChange={(e) => { setFilterTag(e.target.value); setCurrentPage(1); }}
                  className="px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                >
                  <option value="">{t('script.filter.allTags')}</option>
                  {availableTags.map(tag => (
                    <option key={tag} value={tag}>{tag}</option>
                  ))}
                </select>
              </div>

              {(filterGroup || filterTag || searchQuery) && (
                <button
                  onClick={() => { setFilterGroup(''); setFilterTag(''); setSearchQuery(''); setCurrentPage(1); }}
                  className="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 underline"
                >
                  {t('script.filter.clear')}
                </button>
              )}
            </div>

            {/* 批量操作 */}
            {selectedScripts.size > 0 && (
              <div className="flex items-center space-x-3">
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {t('script.selected', { count: selectedScripts.size })}
                </span>
                <button
                  onClick={() => setShowBatchGroupDialog(true)}
                  className="btn-secondary text-sm flex items-center space-x-1.5"
                  disabled={loading}
                >
                  <Folder className="w-4 h-4" />
                  <span>{t('script.batch.setGroup')}</span>
                </button>
                <button
                  onClick={() => setShowBatchTagDialog(true)}
                  className="btn-secondary text-sm flex items-center space-x-1.5"
                  disabled={loading}
                >
                  <Tag className="w-4 h-4" />
                  <span>{t('script.batch.addTags')}</span>
                </button>
                <button
                  onClick={handleExportScripts}
                  className="btn-secondary text-sm flex items-center space-x-1.5"
                  disabled={loading}
                >
                  <Download className="w-4 h-4" />
                  <span>{t('common.export')}</span>
                </button>
                <button
                  onClick={handleBatchDelete}
                  className="btn-secondary text-sm text-red-600 hover:bg-red-50 flex items-center space-x-1.5"
                  disabled={loading}
                >
                  <Trash2 className="w-4 h-4" />
                  <span>{t('common.delete')}</span>
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Toast Notification */}
      {showToast && message && (
        <Toast
          message={message}
          type={toastType}
          onClose={() => setShowToast(false)}
        />
      )}

      {/* Script Parameters Dialog */}
      <ScriptParamsDialog
        isOpen={showParamsDialog}
        parameters={scriptParameters}
        onConfirm={handleParamsDialogConfirm}
        onCancel={handleParamsDialogCancel}
        scriptName={paramsDialogScript?.name}
      />

      {/* Extracted Data Display */}
      {showExtractedData && extractedData && (
        <div className="card bg-gradient-to-br from-emerald-50 to-teal-50 border-2 border-emerald-200">
          <div className="flex items-start justify-between mb-4">
            <div>
              <h3 className="text-lg font-bold text-emerald-900 flex items-center space-x-2">
                <span>{t('script.extractedData.title')}</span>
              </h3>
              <p className="text-sm text-emerald-700 mt-1">
                {t('script.extractedData.count', { count: Object.keys(extractedData).length })}
              </p>
            </div>
            <button
              onClick={() => setShowExtractedData(false)}
              className="p-2 text-emerald-700 hover:bg-emerald-100 rounded transition-colors"
              title={t('common.close')}
            >
              <X className="w-4 h-4" />
            </button>
          </div>

          <div className="space-y-3">
            {Object.entries(extractedData).map(([key, value]) => (
              <div
                key={key}
                className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-emerald-200 dark:border-emerald-800 hover:border-emerald-300 dark:hover:border-emerald-700 transition-colors"
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center space-x-2 mb-2">
                      <span className="px-2 py-1 bg-emerald-100 dark:bg-emerald-900/30 text-emerald-800 dark:text-emerald-300 text-xs font-mono rounded">
                        {key}
                      </span>
                      <span className="text-xs text-gray-500 dark:text-gray-400">
                        {typeof value === 'object' ? 'JSON' : typeof value}
                      </span>
                    </div>
                    <div className="bg-gray-50 dark:bg-gray-900 rounded p-3 overflow-auto max-h-40">
                      <pre className="text-sm text-gray-800 dark:text-gray-200 whitespace-pre-wrap break-all font-mono">
                        {typeof value === 'object'
                          ? JSON.stringify(value, null, 2)
                          : String(value)}
                      </pre>
                    </div>
                  </div>
                  <button
                    onClick={() => {
                      const textValue = typeof value === 'object'
                        ? JSON.stringify(value, null, 2)
                        : String(value)
                      navigator.clipboard.writeText(textValue)
                      showMessage(t('script.extractedData.copied'), 'success')
                    }}
                    className="ml-3 p-2 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-100 dark:hover:bg-emerald-900/30 rounded transition-colors"
                    title={t('common.copy')}
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </button>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-4 flex items-center justify-end space-x-3">
            <button
              onClick={() => {
                const jsonStr = JSON.stringify(extractedData, null, 2)
                const blob = new Blob([jsonStr], { type: 'application/json' })
                const url = URL.createObjectURL(blob)
                const a = document.createElement('a')
                a.href = url
                a.download = `extracted-data-${Date.now()}.json`
                a.click()
                URL.revokeObjectURL(url)
                showMessage(t('script.messages.dataExportedJSON'), 'success')
              }}
              className="px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors text-sm font-medium"
            >
              {t('script.extractedData.exportJSON')}
            </button>
            <button
              onClick={() => {
                const csv = Object.entries(extractedData)
                  .map(([key, value]) => `"${key}","${String(value).replace(/"/g, '""')}"`)
                  .join('\n')
                const csvContent = '变量名,值\n' + csv
                const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
                const url = URL.createObjectURL(blob)
                const a = document.createElement('a')
                a.href = url
                a.download = `extracted-data-${Date.now()}.csv`
                a.click()
                URL.revokeObjectURL(url)
                showMessage(t('script.messages.dataExportedCSV'), 'success')
              }}
              className="px-4 py-2 bg-white dark:bg-gray-800 text-emerald-700 dark:text-emerald-400 border-2 border-emerald-600 dark:border-emerald-500 rounded-lg hover:bg-emerald-50 dark:hover:bg-emerald-900/30 transition-colors text-sm font-medium"
            >
              {t('script.extractedData.exportCSV')}
            </button>
          </div>
        </div>
      )}

      {/* Scripts List */}
      {activeTab === 'scripts' && (
        <>
          <div className="card">
            {scripts.length === 0 ? (
              <div className="text-center py-12 text-gray-500 dark:text-gray-400">
                <FileCode className="w-16 h-16 mx-auto mb-4 opacity-30" />
                <p className="text-lg font-medium">{t('script.noScripts')}</p>
                <p className="text-sm mt-2">{t('script.noScriptsHint')}</p>
              </div>
            ) : (
              <>
                {/* 全选按钮 */}
                <div className="flex items-center justify-between pb-3 mb-3 border-b border-gray-200 dark:border-gray-700">
                  <button
                    onClick={toggleSelectAll}
                    className="flex items-center space-x-2 text-sm text-gray-700 dark:text-gray-300 hover:text-primary-600 transition-colors"
                  >
                    {selectedScripts.size === scripts.length ? (
                      <CheckSquare className="w-4 h-4" />
                    ) : (
                      <Square className="w-4 h-4" />
                    )}
                    <span>{selectedScripts.size === scripts.length ? t('script.deselectAll') : t('script.selectAll')}</span>
                  </button>
                  <span className="text-xs text-gray-500 dark:text-gray-400">{t('script.total', { count: scripts.length })}</span>
                </div>
                <div className="space-y-3">
                  {scripts.map((script) => {
                    const isExpanded = expandedScriptId === script.id
                    const isEditing = editingScript?.id === script.id

                    return (
                      <div
                        key={script.id}
                        className={`bg-gray-50 dark:bg-gray-900 rounded-lg border-2 transition-colors ${selectedScripts.has(script.id)
                          ? 'border-primary-400 bg-primary-50/30 dark:bg-primary-900/30'
                          : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                          }`}
                      >
                        <div className="p-4">
                          <div className="flex items-start justify-between">
                            {/* 选择框 */}
                            <button
                              onClick={() => toggleScriptSelection(script.id)}
                              className="p-1 mr-3 mt-1 text-gray-600 dark:text-gray-400 hover:text-primary-600 transition-colors"
                              title={selectedScripts.has(script.id) ? t('script.card.deselect') : t('script.card.select')}
                            >
                              {selectedScripts.has(script.id) ? (
                                <CheckSquare className="w-5 h-5 text-primary-600" />
                              ) : (
                                <Square className="w-5 h-5" />
                              )}
                            </button>
                            <div className="flex-1">
                              {isEditing ? (
                                <div className="space-y-2 mb-2">
                                  <input
                                    type="text"
                                    value={editingScript.name}
                                    onChange={(e) =>
                                      setEditingScript({ ...editingScript, name: e.target.value })
                                    }
                                    className="input w-full font-bold"
                                    placeholder={t('script.editor.scriptNamePlaceholder')}
                                  />
                                  <input
                                    type="text"
                                    value={editingScript.url}
                                    onChange={(e) =>
                                      setEditingScript({ ...editingScript, url: e.target.value })
                                    }
                                    className="input w-full text-sm"
                                    placeholder={t('script.editor.scriptUrlPlaceholder')}
                                  />
                                  <textarea
                                    value={editingScript.description}
                                    onChange={(e) =>
                                      setEditingScript({ ...editingScript, description: e.target.value })
                                    }
                                    className="input w-full text-sm"
                                    placeholder={t('script.editor.scriptDescPlaceholder')}
                                    rows={2}
                                  />
                                </div>
                              ) : (
                                <>
                                  <h3 className="font-bold text-gray-900 dark:text-gray-100">{script.name}</h3>
                                  {script.description && (
                                    <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{script.description}</p>
                                  )}
                                </>
                              )}
                              <div className="flex items-center space-x-4 mt-2 text-xs text-gray-500 dark:text-gray-400">
                                <span className="flex items-center space-x-1">
                                  <ExternalLink className="w-3 h-3" />
                                  <span className="truncate max-w-xs">{script.url}</span>
                                </span>
                                <span className="flex items-center space-x-1">
                                  <FileCode className="w-3 h-3" />
                                  <span>{t('script.card.actionsCount', { count: script.actions.length })}</span>
                                </span>
                                <span className="flex items-center space-x-1">
                                  <Clock className="w-3 h-3" />
                                  <span>{(script.duration / 1000).toFixed(1)}{t('script.card.durationUnit')}</span>
                                </span>
                                {script.group && (
                                  <span className="flex items-center space-x-1">
                                    <Folder className="w-3 h-3" />
                                    <span>{script.group}</span>
                                  </span>
                                )}
                              </div>
                              {script.tags && script.tags.length > 0 && (
                                <div className="flex items-center space-x-2 mt-2">
                                  <Tag className="w-3 h-3 text-gray-400 dark:text-gray-500" />
                                  {script.tags.map((tag) => (
                                    <span
                                      key={tag}
                                      className="px-2 py-0.5 bg-gray-50 text-gray-700 dark:bg-gray-700 dark:text-gray-300 text-xs rounded border border-gray-200 dark:border-gray-600"
                                    >
                                      {tag}
                                    </span>
                                  ))}
                                </div>
                              )}
                            </div>
                            <div className="flex items-center space-x-2 ml-4">
                              <button
                                onClick={() => toggleScriptExpand(script.id)}
                                className="p-2 text-gray-600 hover:bg-gray-200 rounded transition-colors"
                                title={isExpanded ? t('script.card.collapse') : t('script.card.expand')}
                              >
                                {isExpanded ? (
                                  <ChevronUp className="w-4 h-4" />
                                ) : (
                                  <ChevronDown className="w-4 h-4" />
                                )}
                              </button>
                              {isEditing ? (
                                <>
                                  <button
                                    onClick={handleSaveEditedScript}
                                    disabled={loading}
                                    className="btn-primary p-2"
                                    title={t('script.card.saveEdit')}
                                  >
                                    <Check className="w-4 h-4" />
                                  </button>
                                  <button
                                    onClick={() => {
                                      setEditingScript(null)
                                      setEditingActions([])
                                    }}
                                    disabled={loading}
                                    className="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors"
                                    title={t('script.card.cancelEdit')}
                                  >
                                    <X className="w-4 h-4" />
                                  </button>
                                </>
                              ) : (
                                <>
                                  <button
                                    onClick={() => handleEditScript(script)}
                                    disabled={loading}
                                    className="p-2 text-gray-900 dark:text-gray-100 hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors"
                                    title={t('script.card.editScript')}
                                  >
                                    <Edit2 className="w-4 h-4" />
                                  </button>
                                  <button
                                    onClick={() => handlePlayScript(script.id)}
                                    disabled={loading}
                                    className="btn-primary p-2"
                                    title={t('script.card.playScript')}
                                  >
                                    <Play className="w-4 h-4" />
                                  </button>
                                  <button
                                    onClick={() => handleCopyCurl(script.id)}
                                    disabled={loading}
                                    className="p-2 text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors"
                                    title={t('script.card.copyCurl')}
                                  >
                                    <Clipboard className="w-4 h-4" />
                                  </button>
                                  <button
                                    onClick={() => handleToggleMCP(script.id)}
                                    disabled={loading}
                                    className={`p-2 rounded transition-colors ${script.is_mcp_command
                                      ? 'bg-purple-100 text-purple-700 hover:bg-purple-200'
                                      : 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
                                      }`}
                                    title={script.is_mcp_command ? t('script.card.mcpCancel', { name: script.mcp_command_name || '' }) : t('script.card.mcpSet')}
                                  >
                                    <ExternalLink className="w-4 h-4" />
                                  </button>
                                  <button
                                    onClick={() => setDeleteConfirm({ show: true, scriptId: script.id })}
                                    disabled={loading}
                                    className="btn-danger p-2"
                                    title={t('script.card.deleteScript')}
                                  >
                                    <Trash2 className="w-4 h-4" />
                                  </button>
                                </>
                              )}
                            </div>
                          </div>
                        </div>

                        {/* Expanded Actions List */}
                        {isExpanded && (
                          <div className="border-t border-gray-200 dark:border-gray-700 px-4 pb-4">
                            <div className="mt-3">
                              <div className="flex items-center justify-between mb-3">
                                <h4 className="text-base font-semibold text-gray-800 dark:text-gray-200 flex items-center">
                                  <FileCode className="w-5 h-5 mr-2" />
                                  {t('script.editor.actionsTitle')}
                                </h4>
                                {isEditing && (
                                  <div className="flex items-center space-x-1">
                                    <button
                                      onClick={() => handleAddAction('click')}
                                      className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors"
                                      title={t('script.editor.addClick')}
                                    >
                                      + Click
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('input')}
                                      className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors"
                                      title={t('script.editor.addInput')}
                                    >
                                      + Input
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('select')}
                                      className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors"
                                      title={t('script.editor.addSelect')}
                                    >
                                      + Select
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('navigate')}
                                      className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors"
                                      title={t('script.editor.addNavigate')}
                                    >
                                      + Navigate
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('wait')}
                                      className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors"
                                      title={t('script.editor.addWait')}
                                    >
                                      + Wait
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('sleep')}
                                      className="text-xs px-2 py-1 bg-emerald-100 dark:bg-emerald-900 hover:bg-emerald-200 dark:hover:bg-emerald-800 text-emerald-700 dark:text-emerald-300 rounded transition-colors"
                                      title={t('script.editor.addSleep')}
                                    >
                                      + Sleep
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('scroll')}
                                      className="text-xs px-2 py-1 bg-indigo-100 dark:bg-indigo-900 hover:bg-indigo-200 dark:hover:bg-indigo-800 text-indigo-700 dark:text-indigo-300 rounded transition-colors"
                                      title={t('script.editor.addScroll')}
                                    >
                                      + Scroll
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('extract_text')}
                                      className="text-xs px-2 py-1 bg-blue-100 dark:bg-blue-900 hover:bg-blue-200 dark:hover:bg-blue-800 text-blue-700 dark:text-blue-300 rounded transition-colors"
                                      title={t('script.editor.addExtractText')}
                                    >
                                      + Extract Text
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('extract_html')}
                                      className="text-xs px-2 py-1 bg-blue-100 dark:bg-blue-900 hover:bg-blue-200 dark:hover:bg-blue-800 text-blue-700 dark:text-blue-300 rounded transition-colors"
                                      title={t('script.editor.addExtractHtml')}
                                    >
                                      + Extract HTML
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('extract_attribute')}
                                      className="text-xs px-2 py-1 bg-blue-100 dark:bg-blue-900 hover:bg-blue-200 dark:hover:bg-blue-800 text-blue-700 dark:text-blue-300 rounded transition-colors"
                                      title={t('script.editor.addExtractAttr')}
                                    >
                                      + Extract Attr
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('execute_js')}
                                      className="text-xs px-2 py-1 bg-purple-100 dark:bg-purple-900 hover:bg-purple-200 dark:hover:bg-purple-800 text-purple-700 dark:text-purple-300 rounded transition-colors"
                                      title={t('script.editor.addExecuteJs')}
                                    >
                                      + Execute JS
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('upload_file')}
                                      className="text-xs px-2 py-1 bg-orange-100 dark:bg-orange-900 hover:bg-orange-200 dark:hover:bg-orange-800 text-orange-700 dark:text-orange-300 rounded transition-colors"
                                      title={t('script.editor.addUploadFile')}
                                    >
                                      + Upload File
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('keyboard')}
                                      className="text-xs px-2 py-1 bg-indigo-100 dark:bg-indigo-900 hover:bg-indigo-200 dark:hover:bg-indigo-800 text-indigo-700 dark:text-indigo-300 rounded transition-colors"
                                      title={t('script.editor.addKeyboard')}
                                    >
                                      + Keyboard
                                    </button>
                                    <button
                                      onClick={() => handleAddAction('open_tab')}
                                      className="text-xs px-2 py-1 bg-purple-100 dark:bg-purple-900 hover:bg-purple-200 dark:hover:bg-purple-800 text-purple-700 dark:text-purple-300 rounded transition-colors"
                                      title="添加打开新标签页步骤"
                                    >
                                      + 新标签页
                                    </button>
                                  </div>
                                )}
                              </div>
                              {(isEditing ? editingActions : script.actions).length === 0 ? (
                                <p className="text-sm text-gray-500 dark:text-gray-400 italic">{t('script.editor.noActions')}</p>
                              ) : isEditing ? (
                                <DndContext
                                  sensors={sensors}
                                  collisionDetection={closestCenter}
                                  onDragEnd={handleDragEnd}
                                >
                                  <SortableContext
                                    items={editingActions.map((_, i) => i.toString())}
                                    strategy={verticalListSortingStrategy}
                                  >
                                    <div className="space-y-2">
                                      {editingActions.map((action, index) => (
                                        <SortableActionItem
                                          key={index}
                                          id={index.toString()}
                                          action={action}
                                          index={index}
                                          onUpdate={handleUpdateActionValue}
                                          onDelete={handleDeleteAction}
                                          onDuplicate={handleDuplicateAction}
                                        />
                                      ))}
                                    </div>
                                  </SortableContext>
                                </DndContext>
                              ) : (
                                <div className="space-y-2">
                                  {script.actions.map((action, index) => (
                                    <ActionItemView key={index} action={action} index={index} />
                                  ))}
                                </div>
                              )}
                            </div>
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              </>
            )}
          </div>

          {/* 分页控件 */}
          {totalScripts > pageSize && (
            <div className="flex items-center justify-between">
              <div className="text-sm text-gray-600 dark:text-gray-400">
                {((currentPage - 1) * pageSize) + 1} - {Math.min(currentPage * pageSize, totalScripts)} {t('script.pagination.of')} {t('script.pagination.total')} {totalScripts} {t('script.pagination.items')}
              </div>
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                  className="px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg text-sm hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('script.pagination.prevPage')}
                </button>
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {currentPage} {t('script.pagination.of')} {Math.ceil(totalScripts / pageSize)} {t('script.pagination.totalPages')}
                </span>
                <button
                  onClick={() => setCurrentPage(p => Math.min(Math.ceil(totalScripts / pageSize), p + 1))}
                  disabled={currentPage >= Math.ceil(totalScripts / pageSize)}
                  className="px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg text-sm hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('script.pagination.nextPage')}
                </button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Execution History List */}
      {activeTab === 'executions' && (
        <>
          {/* 搜索和过滤 */}
          <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="flex flex-col lg:flex-row gap-4">
              <div className="flex-1">
                <input
                  type="text"
                  placeholder={t('execution.search.placeholder')}
                  value={executionSearchQuery}
                  onChange={(e) => setExecutionSearchQuery(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSearchExecutions()}
                  className="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100 focus:border-gray-900 dark:focus:border-gray-100 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                />
              </div>
              <div className="flex gap-2">
                <select
                  value={successFilter}
                  onChange={(e) => setSuccessFilter(e.target.value as any)}
                  className="px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-100 focus:border-gray-900 dark:focus:border-gray-100 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                >
                  <option value="all">{t('execution.filter.allStatus')}</option>
                  <option value="success">{t('execution.filter.success')}</option>
                  <option value="failed">{t('execution.filter.failed')}</option>
                </select>
                <button
                  onClick={handleSearchExecutions}
                  className="px-6 py-2.5 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition-colors"
                >
                  {t('common.search')}
                </button>
              </div>
            </div>
          </div>

          {/* 批量操作 */}
          {selectedExecutions.size > 0 && (
            <div className="bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg p-4 flex items-center justify-between">
              <span className="text-gray-900 dark:text-gray-100">
                {t('execution.selected', { count: selectedExecutions.size.toString() })}
              </span>
              <button
                onClick={handleBatchDeleteExecutions}
                className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
              >
                {t('execution.batchDelete')}
              </button>
            </div>
          )}

          {/* 执行记录列表 */}
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
            {loading && executions.length === 0 ? (
              <div className="p-12 text-center text-gray-500 dark:text-gray-400">{t('execution.loading')}</div>
            ) : executions.length === 0 ? (
              <div className="p-12 text-center text-gray-500 dark:text-gray-400">{t('execution.noRecords')}</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
                    <tr>
                      <th className="px-6 py-3 text-left">
                        <input
                          type="checkbox"
                          checked={selectedExecutions.size === executions.length && executions.length > 0}
                          onChange={toggleSelectAllExecutions}
                          className="w-4 h-4 text-gray-900 rounded focus:ring-gray-900"
                        />
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{t('execution.table.scriptName')}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{t('execution.table.startTime')}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{t('execution.table.duration')}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{t('execution.table.steps')}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{t('execution.table.status')}</th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{t('execution.table.actions')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {executions.map((execution) => (
                      <>
                        <tr key={execution.id} className="hover:bg-gray-50 dark:hover:bg-gray-900">
                          <td className="px-6 py-4">
                            <input
                              type="checkbox"
                              checked={selectedExecutions.has(execution.id)}
                              onChange={() => toggleExecutionSelection(execution.id)}
                              className="w-4 h-4 text-gray-900 dark:text-gray-100 rounded focus:ring-gray-900 dark:focus:ring-gray-100"
                            />
                          </td>
                          <td className="px-6 py-4 text-sm font-medium text-gray-900 dark:text-gray-100">
                            {execution.script_name}
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
                            {formatDateTime(execution.start_time)}
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-400">
                            {formatDuration(execution.duration)}
                          </td>
                          <td className="px-6 py-4 text-sm">
                            <div className="flex items-center gap-2">
                              <span className="text-green-600">{t('execution.steps.success', { count: execution.success_steps.toString() })}</span>
                              <span className="text-gray-400">/</span>
                              <span className="text-red-600">{t('execution.steps.failed', { count: execution.failed_steps.toString() })}</span>
                              <span className="text-gray-400">/</span>
                              <span className="text-gray-600 dark:text-gray-400">{t('execution.steps.total', { count: execution.total_steps.toString() })}</span>
                            </div>
                          </td>
                          <td className="px-6 py-4">
                            {execution.success ? (
                              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                                {t('execution.status.success')}
                              </span>
                            ) : (
                              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                                {t('execution.status.failed')}
                              </span>
                            )}
                          </td>
                          <td className="px-6 py-4 text-sm">
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => setExpandedExecutionId(expandedExecutionId === execution.id ? null : execution.id)}
                                className="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
                              >
                                {expandedExecutionId === execution.id ? t('execution.action.hideDetails') : t('execution.action.viewDetails')}
                              </button>
                              <button
                                onClick={() => setExecutionDeleteConfirm({ show: true, executionId: execution.id })}
                                className="text-red-600 hover:text-red-800"
                              >
                                {t('execution.action.delete')}
                              </button>
                            </div>
                          </td>
                        </tr>
                        {/* 展开的详情 */}
                        {expandedExecutionId === execution.id && (
                          <tr>
                            <td colSpan={7} className="px-6 py-4 bg-gray-50 dark:bg-gray-900">
                              <div className="space-y-4 max-h-96 overflow-y-auto">
                                <div>
                                  <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{t('execution.details.executionInfo')}</h4>
                                  <div className="grid grid-cols-2 gap-4 text-sm">
                                    <div>
                                      <span className="text-gray-600">{t('execution.details.executionId')}: </span>
                                      <span className="font-mono text-gray-900 dark:text-gray-100">{execution.id}</span>
                                    </div>
                                    <div>
                                      <span className="text-gray-600">{t('execution.details.scriptId')}: </span>
                                      <span className="font-mono text-gray-900 dark:text-gray-100">{execution.script_id}</span>
                                    </div>
                                    <div>
                                      <span className="text-gray-600">{t('execution.details.endTime')}: </span>
                                      <span className="text-gray-900">{formatDateTime(execution.end_time)}</span>
                                    </div>
                                    <div>
                                      <span className="text-gray-600">{t('execution.details.message')}: </span>
                                      <span className="text-gray-900">{execution.message}</span>
                                    </div>
                                  </div>
                                </div>

                                {execution.error_msg && (
                                  <div>
                                    <h4 className="text-sm font-medium text-red-700 mb-2">{t('execution.details.errorInfo')}</h4>
                                    <div className="bg-red-50 border border-red-200 rounded-lg p-3 max-h-48 overflow-y-auto">
                                      <pre className="text-xs text-red-800 whitespace-pre-wrap">{execution.error_msg}</pre>
                                    </div>
                                  </div>
                                )}

                                {execution.extracted_data && Object.keys(execution.extracted_data).length > 0 && (
                                  <div>
                                    <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{t('execution.details.extractedData')}</h4>
                                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-3 max-h-64 overflow-y-auto">
                                      <pre className="text-xs text-gray-800 dark:text-gray-200 whitespace-pre-wrap">
                                        {JSON.stringify(execution.extracted_data, null, 2)}
                                      </pre>
                                    </div>
                                  </div>
                                )}

                                {execution.video_path && (
                                  <div>
                                    <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{t('execution.details.executionVideo')}</h4>
                                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-3">
                                      <img src={execution.video_path} alt="Execution Video" className="max-w-full h-auto rounded-md" />
                                    </div>
                                  </div>
                                )}
                              </div>
                            </td>
                          </tr>
                        )}
                      </>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-4">
              <div className="flex items-center justify-between">
                <div className="text-sm text-gray-600 dark:text-gray-400">
                  {t('execution.pagination.total', {
                    total: totalExecutions.toString(),
                    current: currentPage.toString(),
                    totalPages: totalPages.toString()
                  })}
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                    disabled={currentPage === 1}
                    className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {t('execution.pagination.prev')}
                  </button>
                  <button
                    onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                    disabled={currentPage === totalPages}
                    className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {t('execution.pagination.next')}
                  </button>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {/* Delete Confirmation Dialog */}
      {executionDeleteConfirm.show && (
        <ConfirmDialog
          title={t('execution.deleteConfirm.title')}
          message={t('execution.deleteConfirm.message')}
          confirmText={t('execution.deleteConfirm.confirm')}
          cancelText={t('execution.deleteConfirm.cancel')}
          onConfirm={handleDeleteExecution}
          onCancel={() => setExecutionDeleteConfirm({ show: false, executionId: null })}
        />
      )}

      {/* Delete Confirmation Dialog */}
      {deleteConfirm.show && (
        <ConfirmDialog
          title={t('script.deleteTitle')}
          message={t('script.deleteConfirm')}
          confirmText={t('common.delete')}
          cancelText={t('common.cancel')}
          onConfirm={handleDeleteScript}
          onCancel={() => setDeleteConfirm({ show: false, scriptId: null })}
        />
      )}

      {/* Import Confirmation Dialog */}
      {showImportConfirm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
                {t('script.import.duplicateTitle')}
              </h3>
              <p className="text-gray-600 dark:text-gray-400 mb-4">
                {t('script.import.duplicateMessage', { count: duplicateScriptIds.length })}
              </p>
              <div className="bg-gray-50 dark:bg-gray-900 rounded p-3 mb-4 max-h-32 overflow-y-auto">
                <p className="text-sm text-gray-700 dark:text-gray-300 font-mono">
                  {duplicateScriptIds.join(', ')}
                </p>
              </div>
              <div className="flex items-center justify-end space-x-3">
                <button
                  onClick={() => {
                    setShowImportConfirm(false)
                    setImportData(null)
                    setDuplicateScriptIds([])
                  }}
                  className="btn-secondary"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={handleImportCreateNew}
                  className="btn-secondary"
                >
                  {t('script.import.createNew')}
                </button>
                <button
                  onClick={handleImportOverwrite}
                  className="btn-danger"
                >
                  {t('script.import.overwrite')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 批量设置分组对话框 */}
      {showBatchGroupDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-bold text-gray-900 dark:text-gray-100">{t('script.dialog.batchGroup.title')}</h3>
                <button onClick={() => setShowBatchGroupDialog(false)} className="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300">
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('script.dialog.batchGroup.message', { count: selectedScripts.size })}
                  </label>
                  <input
                    type="text"
                    value={batchGroupInput}
                    onChange={(e) => setBatchGroupInput(e.target.value)}
                    placeholder={t('script.dialog.batchGroup.placeholder')}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-400 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500"
                    onKeyPress={(e) => e.key === 'Enter' && handleBatchSetGroup()}
                  />
                  {availableGroups.length > 0 && (
                    <div className="mt-2">
                      <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">{t('script.dialog.batchGroup.existing')}</p>
                      <div className="flex flex-wrap gap-1">
                        {availableGroups.map(group => (
                          <button
                            key={group}
                            onClick={() => setBatchGroupInput(group)}
                            className="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-900 dark:text-gray-100 rounded transition-colors"
                          >
                            {group}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>
              <div className="mt-6 flex justify-end space-x-3">
                <button onClick={() => setShowBatchGroupDialog(false)} className="btn-secondary">
                  {t('common.cancel')}
                </button>
                <button onClick={handleBatchSetGroup} disabled={loading || !batchGroupInput.trim()} className="btn-primary">
                  {t('common.confirm')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 批量添加标签对话框 */}
      {showBatchTagDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-bold text-gray-900 dark:text-gray-100">{t('script.dialog.batchTags.title')}</h3>
                <button onClick={() => setShowBatchTagDialog(false)} className="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300">
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('script.dialog.batchTags.message', { count: selectedScripts.size })}
                  </label>
                  <input
                    type="text"
                    value={batchTagsInput}
                    onChange={(e) => setBatchTagsInput(e.target.value)}
                    placeholder={t('script.dialog.batchTags.placeholder')}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-gray-900 dark:focus:ring-gray-400 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500"
                    onKeyPress={(e) => e.key === 'Enter' && handleBatchAddTags()}
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t('script.dialog.batchTags.example')}</p>
                  {availableTags.length > 0 && (
                    <div className="mt-2">
                      <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">{t('script.dialog.batchTags.existing')}</p>
                      <div className="flex flex-wrap gap-1">
                        {availableTags.map(tag => (
                          <button
                            key={tag}
                            onClick={() => {
                              const current = batchTagsInput.split(',').map(t => t.trim()).filter(t => t)
                              if (!current.includes(tag)) {
                                setBatchTagsInput([...current, tag].join(', '))
                              }
                            }}
                            className="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-900 dark:text-gray-100 rounded transition-colors"
                          >
                            {tag}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>
              <div className="mt-6 flex justify-end space-x-3">
                <button onClick={() => setShowBatchTagDialog(false)} className="btn-secondary">
                  {t('common.cancel')}
                </button>
                <button onClick={handleBatchAddTags} disabled={loading || !batchTagsInput.trim()} className="btn-primary">
                  {t('common.confirm')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* MCP Configuration Dialog */}
      {showMCPConfig && mcpConfigScript && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[9999]" style={{ margin: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex items-center justify-between mb-6">
                <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100">
                  {mcpConfigScript.is_mcp_command ? t('script.mcp.dialogTitle.edit') : t('script.mcp.dialogTitle.create')}
                </h3>
                <button
                  onClick={() => setShowMCPConfig(false)}
                  className="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="space-y-5">
                <div>
                  <label className="block text-base font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('script.mcp.scriptName')}
                  </label>
                  <input
                    type="text"
                    value={mcpConfigScript.name}
                    disabled
                    className="w-full px-4 py-2.5 text-base border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-900 text-gray-600 dark:text-gray-400"
                  />
                </div>

                <div>
                  <label className="block text-base font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('script.mcp.commandName')} <span className="text-red-500 dark:text-red-400">*</span>
                  </label>
                  <input
                    type="text"
                    value={mcpCommandName}
                    onChange={(e) => setMCPCommandName(e.target.value)}
                    placeholder={t('script.mcp.commandNamePlaceholder')}
                    className="w-full px-4 py-2.5 text-base border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-gray-500 dark:focus:ring-gray-400 focus:border-transparent bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                  />
                  <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                    {t('script.mcp.commandNameHint')}
                  </p>
                </div>

                <div>
                  <label className="block text-base font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('script.mcp.commandDescription')}
                  </label>
                  <textarea
                    value={mcpCommandDescription}
                    onChange={(e) => setMCPCommandDescription(e.target.value)}
                    placeholder={t('script.mcp.commandDescriptionPlaceholder')}
                    rows={3}
                    className="w-full px-4 py-2.5 text-base border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-purple-500 dark:focus:ring-purple-400 focus:border-transparent bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                  />
                </div>

                <div>
                  <label className="block text-base font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {t('script.editor.inputSchema')}
                  </label>
                  <textarea
                    value={mcpInputSchemaText}
                    onChange={(e) => setMCPInputSchemaText(e.target.value)}
                    placeholder={t('script.mcp.inputSchemaPlaceholder')}
                    rows={12}
                    className="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-gray-500 dark:focus:ring-gray-400 focus:border-transparent font-mono text-sm leading-relaxed bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
                  />
                  <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                    {t('script.mcp.inputSchemaHint')}
                  </p>
                </div>

                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-lg p-4">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <h4 className="text-base font-medium text-blue-900 dark:text-blue-100 mb-2">💡 {t('script.mcp.aiAssist')}</h4>
                      <p className="text-sm text-blue-700 dark:text-blue-300 mb-3">
                        {t('script.mcp.aiAssistDesc')}
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={handleGenerateMCPConfig}
                    disabled={loading}
                    className="w-full px-4 py-2.5 text-base font-medium bg-blue-600 hover:bg-blue-700 text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center space-x-2"
                  >
                    <span>{loading ? t('script.mcp.generating') : t('script.mcp.generateConfig')}</span>
                  </button>
                </div>

                <div className="bg-gray-50 dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                  <h4 className="text-base font-medium text-gray-900 dark:text-gray-100 mb-2">{t('script.mcp.tipsTitle')}</h4>
                  <ul className="text-sm text-gray-700 dark:text-gray-300 space-y-1.5">
                    <li>• {t('script.mcp.tip1')}</li>
                    <li>• {t('script.mcp.tip2')}</li>
                    <li>• {t('script.mcp.tip3')}</li>
                    <li>• {t('script.mcp.tip4')}</li>
                  </ul>
                </div>
              </div>

              <div className="mt-6 flex justify-end space-x-3">
                {mcpConfigScript.is_mcp_command && (
                  <button
                    onClick={handleCancelMCP}
                    disabled={loading}
                    className="px-5 py-2.5 text-base font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {loading ? t('script.editor.processing') : t('script.mcp.cancelCommand')}
                  </button>
                )}
                <button
                  onClick={handleSaveMCPConfig}
                  disabled={loading || !mcpCommandName.trim()}
                  className="px-5 py-2.5 text-base font-medium bg-gray-900 text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {loading ? t('script.editor.saving') : t('common.save')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 录制配置对话框 */}
      {showRecordingConfig && recordingConfig && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('script.recordingConfig.title')}</h2>
                <button
                  onClick={() => setShowRecordingConfig(false)}
                  className="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300"
                >
                  <X className="w-6 h-6" />
                </button>
              </div>

              <div className="space-y-6">
                {/* 启用开关 */}
                <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                  <div>
                    <h3 className="text-base font-medium text-gray-900 dark:text-gray-100">{t('script.recordingConfig.enabled')}</h3>
                    <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{t('script.recordingConfig.enabledDesc')}</p>
                  </div>
                  <button
                    onClick={() => setRecordingConfig({ ...recordingConfig, enabled: !recordingConfig.enabled })}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${recordingConfig.enabled ? 'bg-gray-900' : 'bg-gray-300'
                      }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${recordingConfig.enabled ? 'translate-x-6' : 'translate-x-1'
                        }`}
                    />
                  </button>
                </div>

                {/* 帧率设置 */}
                <div>
                  <label className="block text-base font-medium text-gray-900 dark:text-gray-100 mb-2">
                    {t('script.recordingConfig.frameRate')}
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="60"
                    value={recordingConfig.frame_rate}
                    onChange={(e) => setRecordingConfig({ ...recordingConfig, frame_rate: parseInt(e.target.value) || 15 })}
                    className="w-full px-4 py-2.5 text-base border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-blue-500"
                  />
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{t('script.recordingConfig.frameRateDesc')}</p>
                </div>

                {/* 质量设置 */}
                <div>
                  <label className="block text-base font-medium text-gray-900 dark:text-gray-100 mb-2">
                    {t('script.recordingConfig.quality')}
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="100"
                    value={recordingConfig.quality}
                    onChange={(e) => setRecordingConfig({ ...recordingConfig, quality: parseInt(e.target.value) || 70 })}
                    className="w-full px-4 py-2.5 text-base border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-blue-500"
                  />
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{t('script.recordingConfig.qualityDesc')}</p>
                </div>

                {/* 输出格式 */}
                <div>
                  <label className="block text-base font-medium text-gray-900 dark:text-gray-100 mb-2">
                    {t('script.recordingConfig.format')}
                  </label>
                  <select
                    value={recordingConfig.format}
                    onChange={(e) => setRecordingConfig({ ...recordingConfig, format: e.target.value })}
                    className="w-full px-4 py-2.5 text-base border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-blue-500"
                  >
                    <option value="mp4">MP4</option>
                    <option value="webm">WebM</option>
                  </select>
                </div>

                {/* 输出目录 */}
                <div>
                  <label className="block text-base font-medium text-gray-900 dark:text-gray-100 mb-2">
                    {t('script.recordingConfig.outputDir')}
                  </label>
                  <input
                    type="text"
                    value={recordingConfig.output_dir}
                    onChange={(e) => setRecordingConfig({ ...recordingConfig, output_dir: e.target.value })}
                    className="w-full px-4 py-2.5 text-base border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-blue-500"
                  />
                  <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{t('script.recordingConfig.outputDirDesc')}</p>
                </div>

                <div className="bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg p-4">
                  <h4 className="text-base font-medium text-gray-900 dark:text-gray-100 mb-2">{t('script.recordingConfig.note')}</h4>
                  <ul className="text-sm text-gray-700 dark:text-gray-300 space-y-1.5">
                    <li>• {t('script.recordingConfig.noteItem1')}</li>
                    <li>• {t('script.recordingConfig.noteItem2')}</li>
                    <li>• {t('script.recordingConfig.noteItem3')}</li>
                  </ul>
                </div>
              </div>

              <div className="mt-6 flex justify-end space-x-3">
                <button
                  onClick={() => setShowRecordingConfig(false)}
                  className="px-5 py-2.5 text-base font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={handleSaveRecordingConfig}
                  disabled={loading}
                  className="px-5 py-2.5 text-base font-medium bg-gray-900 dark:bg-blue-600 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {loading ? t('script.editor.saving') : t('common.save')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Tutorial Modal */}
      {showTutorial && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4" style={{ marginTop: 0, marginBottom: 0 }}>
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-2xl w-full max-h-[90vh] flex flex-col">
            {/* Fixed Header */}
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between">
                <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t('script.tutorial.title')}</h2>
                <button onClick={() => setShowTutorial(false)} className="text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300">
                  <X className="w-6 h-6" />
                </button>
              </div>
            </div>

            {/* Scrollable Content */}
            <div className="p-6 overflow-y-auto flex-1">
              <div className="space-y-6">
                {/* Parameter Placeholders Section */}
                <div>
                  <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-3">{t('script.tutorial.params.title')}</h3>
                  <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mb-4">
                    <div className="flex items-center space-x-2 mb-2">
                      <div className="bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-200 rounded-full p-2">
                        <FileCode className="w-5 h-5" />
                      </div>
                      <h4 className="text-lg font-medium text-blue-900 dark:text-blue-200">{t('script.tutorial.params.usage')}</h4>
                    </div>
                    <p className="text-gray-700 dark:text-gray-300 mb-3">{t('script.tutorial.params.description')}</p>
                    <div className="space-y-3">
                      <div>
                        <h5 className="font-medium text-gray-800 dark:text-gray-200 mb-1">{t('script.tutorial.params.exampleUrl')}</h5>
                        <code className="bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg p-3 block font-mono text-sm dark:text-gray-200">
                          https://example.com/search?q=${'{'}search_query{'}'}
                        </code>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-800 dark:text-gray-200 mb-1">{t('script.tutorial.params.exampleAction')}</h5>
                        <code className="bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg p-3 block font-mono text-sm dark:text-gray-200">
                          ${'{'}username{'}'}
                        </code>
                      </div>
                    </div>
                  </div>
                </div>
                {/* MCP Configuration Section */}
                <div>
                  <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-3">{t('script.tutorial.mcp.title')}</h3>
                  <div className="bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg p-4 mb-4">
                    <div className="flex items-center space-x-2 mb-2">
                      <div className="bg-purple-100 dark:bg-purple-800 text-purple-800 dark:text-purple-200 rounded-full p-2">
                        <ExternalLink className="w-5 h-5" />
                      </div>
                      <h4 className="text-lg font-medium text-purple-900 dark:text-purple-200">{t('script.tutorial.mcp.usage')}</h4>
                    </div>
                    <p className="text-gray-700 dark:text-gray-300 mb-3">{t('script.tutorial.mcp.description')}</p>
                    <div className="space-y-3">
                      <div>
                        <h5 className="font-medium text-gray-800 dark:text-gray-200 mb-1">{t('script.tutorial.mcp.enable')}</h5>
                        <ol className="list-decimal list-inside text-sm text-gray-700 dark:text-gray-300 space-y-1 pl-2">
                          <li>{t('script.tutorial.mcp.step1')}</li>
                          <li>{t('script.tutorial.mcp.step2')}</li>
                          <li>{t('script.tutorial.mcp.step3')}</li>
                        </ol>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-800 dark:text-gray-200 mb-1">{t('script.tutorial.mcp.integration')}</h5>
                        <p className="text-sm text-gray-700 dark:text-gray-300 mb-2">{t('script.tutorial.mcp.integrationDesc')}</p>
                        <div className="grid grid-cols-1 md:grid-cols-1 gap-3">
                          <div
                            className="bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg p-3 cursor-pointer hover:shadow-md transition-shadow"
                            onClick={() => handleCopyToClipboard(JSON.stringify({
                              mcpServers: {
                                browserwing: {
                                  url: `http://${window.location.hostname}:${mcpPort}/api/v1/mcp/message`
                                }
                              }
                            }, null, 2), 'mcp-config')}
                          >
                            <h6 className="font-medium text-gray-800 dark:text-gray-200 mb-1 flex items-center justify-between">
                              Cursor / Claude Desktop
                              {copiedItem === 'mcp-config' ? (
                                <Check className="w-4 h-4 text-green-500" />
                              ) : (
                                <Clipboard className="w-4 h-4 text-gray-500 dark:text-gray-400" />
                              )}
                            </h6>
                            <p className="text-xs text-gray-600 dark:text-gray-400 mb-2">{t('script.tutorial.mcp.copyDesc')}</p>
                            <pre className="bg-gray-50 dark:bg-gray-950 rounded p-2 text-xs font-mono overflow-x-auto dark:text-gray-300">
                              {`{
  "mcpServers": {
    "browserwing": {
      "url": "http://${window.location.hostname}:${mcpPort}/api/v1/mcp/message"
    }
  }
}`}
                            </pre>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                {/* HTTP API Section */}
                <div>
                  <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-3">{t('script.tutorial.http.title')}</h3>
                  <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4 mb-4">
                    <div className="flex items-center space-x-2 mb-2">
                      <div className="bg-green-100 dark:bg-green-800 text-green-800 dark:text-green-200 rounded-full p-2">
                        <FileCode className="w-5 h-5" />
                      </div>
                      <h4 className="text-lg font-medium text-green-900 dark:text-green-200">{t('script.tutorial.http.usage')}</h4>
                    </div>
                    <p className="text-gray-700 dark:text-gray-300 mb-3">{t('script.tutorial.http.description')}</p>
                    <div className="space-y-3">
                      <div>
                        <h5 className="font-medium text-gray-800 dark:text-gray-200 mb-1">{t('script.tutorial.http.endpoint')}</h5>
                        <code className="bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg p-3 block font-mono text-sm dark:text-gray-200">
                          POST http://{window.location.hostname}:{mcpPort}/api/v1/scripts/{'{script_id}'}/play
                        </code>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-800 dark:text-gray-200 mb-1">{t('script.tutorial.http.curlExample')}</h5>
                        <div
                          className="bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg p-3 cursor-pointer hover:shadow-md transition-shadow relative group"
                          onClick={() => {
                            const curlExample = `curl -X POST http://${window.location.hostname}:${mcpPort}/api/v1/scripts/{script_id}/play \\
  -H "Content-Type: application/json"`;
                            handleCopyToClipboard(curlExample, 'curl-basic');
                          }}
                        >
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs text-gray-600 dark:text-gray-400">{t('script.tutorial.mcp.copyDesc')}</span>
                            {copiedItem === 'curl-basic' ? (
                              <Check className="w-4 h-4 text-green-500" />
                            ) : (
                              <Clipboard className="w-4 h-4 text-gray-500 dark:text-gray-400" />
                            )}
                          </div>
                          <pre className="bg-gray-50 dark:bg-gray-950 rounded p-2 text-xs font-mono overflow-x-auto dark:text-gray-300">
                            {`curl -X POST http://${window.location.hostname}:${mcpPort}/api/v1/scripts/{script_id}/play \\
  -H "Content-Type: application/json"`}
                          </pre>
                        </div>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-800 dark:text-gray-200 mb-1">{t('script.tutorial.http.withParams')}</h5>
                        <div
                          className="bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded-lg p-3 cursor-pointer hover:shadow-md transition-shadow relative group"
                          onClick={() => {
                            const curlWithParams = `curl -X POST http://${window.location.hostname}:${mcpPort}/api/v1/scripts/{script_id}/play \\
  -H "Content-Type: application/json" \\
  -d '{
    "params": {
      "username": "test_user",
      "password": "secret123"
    }
  }'`;
                            handleCopyToClipboard(curlWithParams, 'curl-with-params');
                          }}
                        >
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-xs text-gray-600 dark:text-gray-400">{t('script.tutorial.mcp.copyDesc')}</span>
                            {copiedItem === 'curl-with-params' ? (
                              <Check className="w-4 h-4 text-green-500" />
                            ) : (
                              <Clipboard className="w-4 h-4 text-gray-500 dark:text-gray-400" />
                            )}
                          </div>
                          <pre className="bg-gray-50 dark:bg-gray-950 rounded p-2 text-xs font-mono overflow-x-auto dark:text-gray-300">
                            {`curl -X POST http://${window.location.hostname}:${mcpPort}/api/v1/scripts/{script_id}/play \\
  -H "Content-Type: application/json" \\
  -d '{
    "params": {
      "username": "test_user",
      "password": "secret123"
    }
  }'`}
                          </pre>
                        </div>
                      </div>
                      <div>
                        <h5 className="font-medium text-gray-800 dark:text-gray-200 mb-1">{t('script.tutorial.http.responseFormat')}</h5>
                        <p className="text-sm text-gray-700 dark:text-gray-300 mb-2">{t('script.tutorial.http.responseDesc')}</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// 可排序的 Action 项组件
interface SortableActionItemProps {
  id: string
  action: ScriptAction
  index: number
  onUpdate: (index: number, field: keyof ScriptAction, value: string | number | string[]) => void
  onDelete: (index: number) => void
  onDuplicate: (index: number) => void
}

function SortableActionItem({ id, action, index, onUpdate, onDelete, onDuplicate }: SortableActionItemProps) {
  const { t } = useLanguage()
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 text-base shadow-sm"
    >
      <div className="flex items-start gap-3">
        <button
          className="cursor-grab active:cursor-grabbing p-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded mt-1"
          {...attributes}
          {...listeners}
        >
          <GripVertical className="w-5 h-5 text-gray-400 dark:text-gray-500" />
        </button>
        <div className="flex-1 space-y-3">
          <div className="flex items-center space-x-2">
            <span className="font-mono text-sm bg-gray-100 dark:bg-gray-700 dark:text-gray-300 px-2.5 py-1 rounded font-medium">
              #{index + 1}
            </span>
            <span className="font-semibold text-base text-gray-900 dark:text-gray-100">{t(action.type)}</span>
          </div>
          <div className="space-y-3">
            {action.type !== 'sleep' && action.type !== 'wait' && action.type !== 'execute_js' && action.type !== 'upload_file' && action.type !== 'scroll' && action.type !== 'keyboard' && (
              <>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.selector')}</label>
                  <input
                    type="text"
                    value={action.selector}
                    onChange={(e) => onUpdate(index, 'selector', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="CSS 选择器"
                  />
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">XPath:</label>
                  <input
                    type="text"
                    value={action.xpath || ''}
                    onChange={(e) => onUpdate(index, 'xpath', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="XPath 路径"
                  />
                </div>
              </>
            )}
            {action.type === 'upload_file' && (
              <>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.selector')}</label>
                  <input
                    type="text"
                    value={action.selector}
                    onChange={(e) => onUpdate(index, 'selector', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="CSS 选择器"
                  />
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">XPath:</label>
                  <input
                    type="text"
                    value={action.xpath || ''}
                    onChange={(e) => onUpdate(index, 'xpath', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="XPath 路径"
                  />
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.filePaths')}</label>
                  <textarea
                    value={action.file_paths?.join('\n') || ''}
                    onChange={(e) => onUpdate(index, 'file_paths', e.target.value.split('\n').filter(p => p.trim()))}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-gray-50 dark:bg-gray-700 dark:text-gray-100 focus:bg-white dark:focus:bg-gray-600 focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
                    placeholder="/path/to/file1.jpg&#10;/path/to/file2.png"
                    rows={5}
                  />
                  <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('script.action.filePathHint')}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.description')}</label>
                  <input
                    type="text"
                    value={action.description || ''}
                    onChange={(e) => onUpdate(index, 'description', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder={t('script.action.uploadDesc')}
                  />
                </div>
              </>
            )}
            {(action.type === 'input' || action.type === 'select') && (
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.value')}</label>
                <input
                  type="text"
                  value={action.value || ''}
                  onChange={(e) => onUpdate(index, 'value', e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder={t('script.action.inputPlaceholder')}
                />
              </div>
            )}
            {action.type === 'navigate' && (
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">URL:</label>
                <input
                  type="text"
                  value={action.url || ''}
                  onChange={(e) => onUpdate(index, 'url', e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="https://example.com"
                />
              </div>
            )}
            {action.type === 'sleep' && (
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.delay')}</label>
                <input
                  type="number"
                  value={action.duration || 1000}
                  onChange={(e) => onUpdate(index, 'duration', parseInt(e.target.value) || 1000)}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="1000"
                  min="0"
                  step="100"
                />
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                  {((action.duration || 1000) / 1000).toFixed(1)} {t('script.action.delaySeconds')}
                </p>
              </div>
            )}
            {action.type === 'scroll' && (
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.scrollX')}</label>
                  <input
                    type="number"
                    value={action.scroll_x || 0}
                    onChange={(e) => onUpdate(index, 'scroll_x', parseInt(e.target.value) || 0)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="0"
                    min="0"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t('script.action.scrollXHint')}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.scrollY')}</label>
                  <input
                    type="number"
                    value={action.scroll_y || 0}
                    onChange={(e) => onUpdate(index, 'scroll_y', parseInt(e.target.value) || 0)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="0"
                    min="0"
                  />
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t('script.action.scrollYHint')}</p>
                </div>
              </div>
            )}
            {(action.type === 'extract_text' || action.type === 'extract_html' || action.type === 'extract_attribute') && (
              <>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.variableName')}</label>
                  <input
                    type="text"
                    value={action.variable_name || ''}
                    onChange={(e) => onUpdate(index, 'variable_name', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="data_0"
                  />
                  <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('script.action.variableHint')}</p>
                </div>
                {action.type === 'extract_attribute' && (
                  <div>
                    <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.attributeName')}</label>
                    <input
                      type="text"
                      value={action.attribute_name || ''}
                      onChange={(e) => onUpdate(index, 'attribute_name', e.target.value)}
                      className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="href, src, data-id 等"
                    />
                  </div>
                )}
              </>
            )}
            {action.type === 'execute_js' && (
              <>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1 block">{t('script.action.jsCode')}</label>
                  <div className="relative">
                    <textarea
                      value={action.js_code || ''}
                      onChange={(e) => onUpdate(index, 'js_code', e.target.value)}
                      className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-gray-900 dark:bg-gray-950 text-green-400 dark:text-green-300 focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors resize-y"
                      placeholder="return document.title;"
                      rows={8}
                      style={{ minHeight: '120px' }}
                    />
                    <div className="absolute top-2 right-2 text-xs text-gray-500 dark:text-gray-400 bg-gray-800 dark:bg-gray-900 px-2 py-1 rounded">
                      JavaScript
                    </div>
                  </div>
                  <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('script.action.jsHint')}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.variableName')}</label>
                  <input
                    type="text"
                    value={action.variable_name || ''}
                    onChange={(e) => onUpdate(index, 'variable_name', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="result_0"
                  />
                </div>
              </>
            )}
            {action.type === 'keyboard' && (
              <>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.key')}</label>
                  <select
                    value={action.key || 'enter'}
                    onChange={(e) => onUpdate(index, 'key', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                    <option value="enter">Enter (回车键)</option>
                    <option value="tab">Tab (切换)</option>
                    <option value="backspace">Backspace (删除)</option>
                    <option value="ctrl+a">Ctrl+A (全选)</option>
                    <option value="ctrl+c">Ctrl+C (复制)</option>
                    <option value="ctrl+v">Ctrl+V (粘贴)</option>
                  </select>
                  <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('script.action.keyHint')}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.selector')} ({t('script.action.optional')})</label>
                  <input
                    type="text"
                    value={action.selector || ''}
                    onChange={(e) => onUpdate(index, 'selector', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder="CSS 选择器（可选）"
                  />
                  <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{t('script.action.keyboardSelectorHint')}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">{t('script.action.description')} ({t('script.action.optional')})</label>
                  <input
                    type="text"
                    value={action.description || ''}
                    onChange={(e) => onUpdate(index, 'description', e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    placeholder={t('script.action.keyboardDescPlaceholder')}
                  />
                </div>
              </>
            )}
            {action.type === 'open_tab' && (
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">URL</label>
                <input
                  type="text"
                  value={action.url || ''}
                  onChange={(e) => onUpdate(index, 'url', e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg font-mono bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="https://example.com"
                />
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">在新标签页中打开此 URL</p>
              </div>
            )}
            {action.type === 'switch_tab' && (
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">标签页索引</label>
                <input
                  type="number"
                  value={action.value || '0'}
                  onChange={(e) => onUpdate(index, 'value', e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="0"
                  min="0"
                />
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">0 表示第一个标签页，1 表示第二个标签页，以此类推</p>
              </div>
            )}
          </div>

          {/* 语义信息展示（编辑模式） */}
          {(action.intent || action.accessibility || action.context || action.evidence) && (
            <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700 space-y-3">
              <div className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide flex items-center space-x-2">
                <span>{t('script.action.semanticInfo')}</span>
                <span className="text-xs font-normal text-gray-400 dark:text-gray-500">({t('script.action.readOnly')})</span>
              </div>

              {action.intent && (action.intent.verb || action.intent.object) && (
                <div className="bg-purple-50 dark:bg-purple-900/20 rounded-lg p-3 text-sm">
                  <div className="font-medium text-purple-700 dark:text-purple-300 mb-1">Intent</div>
                  <div className="text-gray-700 dark:text-gray-300">
                    {action.intent.verb && <code className="bg-white dark:bg-gray-800 px-2 py-1 rounded text-xs">{action.intent.verb}</code>}
                    {action.intent.verb && action.intent.object && ' → '}
                    {action.intent.object && <span className="font-medium">{action.intent.object}</span>}
                  </div>
                </div>
              )}

              {action.accessibility && (action.accessibility.role || action.accessibility.name) && (
                <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-3 text-sm">
                  <div className="font-medium text-blue-700 dark:text-blue-300 mb-1">Accessibility</div>
                  <div className="text-gray-700 dark:text-gray-300 space-y-1">
                    {action.accessibility.role && (
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Role:</span>{' '}
                        <code className="bg-white dark:bg-gray-800 px-2 py-1 rounded text-xs">{action.accessibility.role}</code>
                      </div>
                    )}
                    {action.accessibility.name && (
                      <div>
                        <span className="text-xs text-gray-500 dark:text-gray-400">Name:</span>{' '}
                        <span className="font-medium">"{action.accessibility.name}"</span>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {action.context && (
                <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-3 text-sm space-y-2">
                  <div className="font-medium text-green-700 dark:text-green-300">Context</div>
                  {action.context.nearby_text && action.context.nearby_text.length > 0 && (
                    <div>
                      <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Nearby Text:</div>
                      <div className="flex flex-wrap gap-1">
                        {action.context.nearby_text.slice(0, 3).map((text, i) => (
                          <span key={i} className="inline-block bg-white dark:bg-gray-800 px-2 py-1 rounded text-xs text-gray-700 dark:text-gray-300">
                            {text}
                          </span>
                        ))}
                        {action.context.nearby_text.length > 3 && (
                          <span className="inline-block text-xs text-gray-500">+{action.context.nearby_text.length - 3}</span>
                        )}
                      </div>
                    </div>
                  )}
                  {action.context.ancestor_tags && action.context.ancestor_tags.length > 0 && (
                    <div>
                      <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Ancestors:</div>
                      <code className="block bg-white dark:bg-gray-800 px-2 py-1 rounded text-xs text-gray-700 dark:text-gray-300">
                        {action.context.ancestor_tags.slice(0, 5).join(' > ')}
                        {action.context.ancestor_tags.length > 5 && ' ...'}
                      </code>
                    </div>
                  )}
                  {action.context.form_hint && (
                    <div>
                      <div className="text-xs text-gray-500 dark:text-gray-400 mb-1">Form Type:</div>
                      <code className="bg-white dark:bg-gray-800 px-2 py-1 rounded text-xs text-gray-700 dark:text-gray-300">
                        {action.context.form_hint}
                      </code>
                    </div>
                  )}
                </div>
              )}

              {action.evidence && action.evidence.confidence !== undefined && (
                <div className="bg-orange-50 dark:bg-orange-900/20 rounded-lg p-3 text-sm">
                  <div className="font-medium text-orange-700 dark:text-orange-300 mb-2">Evidence</div>
                  <div className="flex items-center space-x-3">
                    <span className="text-xs text-gray-500 dark:text-gray-400">Confidence:</span>
                    <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2 max-w-xs">
                      <div
                        className={`h-2 rounded-full transition-all ${action.evidence.confidence >= 0.8 ? 'bg-green-500' :
                            action.evidence.confidence >= 0.6 ? 'bg-yellow-500' :
                              'bg-orange-500'
                          }`}
                        style={{ width: `${action.evidence.confidence * 100}%` }}
                      />
                    </div>
                    <span className="text-xs font-mono font-medium text-gray-700 dark:text-gray-300">
                      {(action.evidence.confidence * 100).toFixed(0)}%
                    </span>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
        <div className="flex flex-col gap-2">
          <button
            onClick={() => onDuplicate(index)}
            className="p-2 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded-lg transition-colors"
            title={t('script.action.duplicateStep')}
          >
            <Copy className="w-5 h-5" />
          </button>
          <button
            onClick={() => onDelete(index)}
            className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30 rounded-lg transition-colors"
            title={t('script.action.deleteStep')}
          >
            <Trash2 className="w-5 h-5" />
          </button>
        </div>
      </div>
    </div>
  )
}

// 只读的 Action 项组件
interface ActionItemViewProps {
  action: ScriptAction
  index: number
}

function ActionItemView({ action, index }: ActionItemViewProps) {
  const { t } = useLanguage()

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 text-base shadow-sm">
      <div className="space-y-2">
        <div className="flex items-center space-x-2">
          <span className="font-mono text-sm bg-gray-100 dark:bg-gray-900 px-2.5 py-1 rounded font-medium text-gray-900 dark:text-gray-100">
            #{index + 1}
          </span>
          <span className="font-semibold text-base text-gray-900 dark:text-gray-100">{t(action.type)}</span>
        </div>
        {action.selector && (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <span className="font-medium">{t('script.action.selector')}</span>{' '}
            <code className="bg-gray-100 dark:bg-gray-900 px-2 py-1 rounded text-sm">{action.selector}</code>
          </div>
        )}
        {action.xpath && (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <span className="font-medium">XPath:</span>{' '}
            <code className="bg-gray-100 dark:bg-gray-900 px-2 py-1 rounded text-sm">{action.xpath}</code>
          </div>
        )}
        {action.value && (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <span className="font-medium">{t('script.action.value')}</span>{' '}
            <span className="text-gray-800 dark:text-gray-200">{action.value}</span>
          </div>
        )}
        {action.url && (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <span className="font-medium">URL:</span>{' '}
            <span className="text-gray-800 dark:text-gray-200">{action.url}</span>
          </div>
        )}
        {action.type === 'sleep' && action.duration !== undefined && (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <span className="font-medium">{t('script.action.delay')}</span>{' '}
            <span className="text-gray-800 dark:text-gray-200">{action.duration}ms ({(action.duration / 1000).toFixed(1)}{t('script.action.delaySeconds')})</span>
          </div>
        )}
        {action.type === 'scroll' && (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <span className="font-medium">{t('script.action.scrollPosition')}</span>{' '}
            <span className="text-gray-800 dark:text-gray-200">
              X: {action.scroll_x || 0}, Y: {action.scroll_y || 0}
            </span>
          </div>
        )}
        {(action.type === 'extract_text' || action.type === 'extract_html' || action.type === 'extract_attribute') && (
          <>
            {action.variable_name && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.variableName')}</span>{' '}
                <code className="bg-emerald-100 dark:bg-emerald-900 text-emerald-800 dark:text-emerald-300 px-2 py-1 rounded text-sm">{action.variable_name}</code>
              </div>
            )}
            {action.type === 'extract_attribute' && action.attribute_name && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.attributeName')}</span>{' '}
                <code className="bg-gray-100 dark:bg-gray-900 px-2 py-1 rounded text-sm">{action.attribute_name}</code>
              </div>
            )}
          </>
        )}
        {action.type === 'execute_js' && (
          <>
            {action.js_code && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.jsCode')}</span>
                <pre className="bg-gray-100 dark:bg-gray-900 px-3 py-2 rounded-lg mt-1 text-sm font-mono overflow-x-auto text-gray-900 dark:text-gray-100">{action.js_code}</pre>
              </div>
            )}
            {action.variable_name && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.variableName')}</span>{' '}
                <code className="bg-emerald-100 dark:bg-emerald-900 text-emerald-800 dark:text-emerald-300 px-2 py-1 rounded text-sm">{action.variable_name}</code>
              </div>
            )}
          </>
        )}
        {action.type === 'upload_file' && (
          <>
            {action.file_names && action.file_names.length > 0 && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.fileName')}</span>{' '}
                <span className="text-gray-800 dark:text-gray-200">{action.file_names.join(', ')}</span>
              </div>
            )}
            {action.file_paths && action.file_paths.length > 0 && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.filePath')}</span>{' '}
                {action.file_paths.map((path, i) => (
                  <code key={i} className="bg-gray-100 dark:bg-gray-900 px-2 py-1 rounded text-sm mr-1">{path}</code>
                ))}
              </div>
            )}
            {action.description && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.description')}</span>{' '}
                <span className="text-gray-800 dark:text-gray-200">{action.description}</span>
              </div>
            )}
            {action.multiple && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.multipleFiles')}</span>{' '}
                <span className="text-emerald-600 dark:text-emerald-400">{t('script.action.yes')}</span>
              </div>
            )}
            {action.accept && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.acceptTypes')}</span>{' '}
                <code className="bg-gray-100 dark:bg-gray-900 px-2 py-1 rounded text-sm">{action.accept}</code>
              </div>
            )}
          </>
        )}
        {action.type === 'keyboard' && (
          <>
            {action.key && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.key')}</span>{' '}
                <kbd className="bg-gray-100 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 px-2.5 py-1.5 rounded font-mono text-sm font-semibold text-gray-900 dark:text-gray-100 shadow-sm">
                  {action.key}
                </kbd>
              </div>
            )}
            {action.description && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium">{t('script.action.description')}</span>{' '}
                <span className="text-gray-800 dark:text-gray-200">{action.description}</span>
              </div>
            )}
          </>
        )}
        {action.type === 'open_tab' && action.url && (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <span className="font-medium">URL:</span>{' '}
            <span className="text-gray-800 dark:text-gray-200">{action.url}</span>
          </div>
        )}
        {action.type === 'switch_tab' && action.value && (
          <div className="text-sm text-gray-600 dark:text-gray-400">
            <span className="font-medium">标签页索引:</span>{' '}
            <span className="text-gray-800 dark:text-gray-200">{action.value}</span>
          </div>
        )}

        {/* 语义信息展示 */}
        {(action.intent || action.accessibility || action.context || action.evidence) && (
          <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700 space-y-2">
            <div className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">
              {t('script.action.semanticInfo')}
            </div>

            {action.intent && (action.intent.verb || action.intent.object) && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium text-purple-600 dark:text-purple-400">Intent:</span>{' '}
                <span className="text-gray-800 dark:text-gray-200">
                  {action.intent.verb && <code className="bg-purple-50 dark:bg-purple-900/30 px-1.5 py-0.5 rounded text-xs">{action.intent.verb}</code>}
                  {action.intent.verb && action.intent.object && ' → '}
                  {action.intent.object && <span className="font-medium">{action.intent.object}</span>}
                </span>
              </div>
            )}

            {action.accessibility && (action.accessibility.role || action.accessibility.name) && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium text-blue-600 dark:text-blue-400">Accessibility:</span>{' '}
                <span className="text-gray-800 dark:text-gray-200">
                  {action.accessibility.role && (
                    <code className="bg-blue-50 dark:bg-blue-900/30 px-1.5 py-0.5 rounded text-xs mr-2">
                      {action.accessibility.role}
                    </code>
                  )}
                  {action.accessibility.name && (
                    <span className="font-medium">"{action.accessibility.name}"</span>
                  )}
                </span>
              </div>
            )}

            {action.context && (
              <div className="text-sm text-gray-600 dark:text-gray-400 space-y-1">
                {action.context.nearby_text && action.context.nearby_text.length > 0 && (
                  <div>
                    <span className="font-medium text-green-600 dark:text-green-400">Nearby Text:</span>{' '}
                    <div className="mt-1 flex flex-wrap gap-1">
                      {action.context.nearby_text.slice(0, 3).map((text, i) => (
                        <span key={i} className="inline-block bg-green-50 dark:bg-green-900/30 px-2 py-1 rounded text-xs text-gray-700 dark:text-gray-300">
                          {text}
                        </span>
                      ))}
                      {action.context.nearby_text.length > 3 && (
                        <span className="inline-block text-xs text-gray-500">+{action.context.nearby_text.length - 3} more</span>
                      )}
                    </div>
                  </div>
                )}
                {action.context.ancestor_tags && action.context.ancestor_tags.length > 0 && (
                  <div>
                    <span className="font-medium text-green-600 dark:text-green-400">Ancestors:</span>{' '}
                    <code className="bg-green-50 dark:bg-green-900/30 px-2 py-1 rounded text-xs text-gray-700 dark:text-gray-300">
                      {action.context.ancestor_tags.slice(0, 5).join(' > ')}
                      {action.context.ancestor_tags.length > 5 && ' ...'}
                    </code>
                  </div>
                )}
                {action.context.form_hint && (
                  <div>
                    <span className="font-medium text-green-600 dark:text-green-400">Form Type:</span>{' '}
                    <code className="bg-green-50 dark:bg-green-900/30 px-2 py-1 rounded text-xs text-gray-700 dark:text-gray-300">
                      {action.context.form_hint}
                    </code>
                  </div>
                )}
              </div>
            )}

            {action.evidence && action.evidence.confidence !== undefined && (
              <div className="text-sm text-gray-600 dark:text-gray-400">
                <span className="font-medium text-orange-600 dark:text-orange-400">Confidence:</span>{' '}
                <div className="inline-flex items-center space-x-2">
                  <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2 w-32">
                    <div
                      className={`h-2 rounded-full ${action.evidence.confidence >= 0.8 ? 'bg-green-500' :
                          action.evidence.confidence >= 0.6 ? 'bg-yellow-500' :
                            'bg-orange-500'
                        }`}
                      style={{ width: `${action.evidence.confidence * 100}%` }}
                    />
                  </div>
                  <span className="text-xs font-mono text-gray-700 dark:text-gray-300">
                    {(action.evidence.confidence * 100).toFixed(0)}%
                  </span>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
