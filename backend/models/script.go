package models

import (
	"time"
)

// DownloadedFile 下载的文件信息
type DownloadedFile struct {
	FileName     string    `json:"file_name"`     // 文件名
	FilePath     string    `json:"file_path"`     // 完整文件路径
	URL          string    `json:"url"`           // 下载URL
	MimeType     string    `json:"mime_type"`     // MIME类型
	Size         int64     `json:"size"`          // 文件大小（字节）
	DownloadTime time.Time `json:"download_time"` // 下载时间
}

// ScriptAction 脚本操作步骤（v2 - 支持语义与自愈，向后兼容）
type ScriptAction struct {
	// =========================
	// 原有字段（保持不变）
	// =========================
	Type      string            `json:"type"`      // click, input, select, navigate, wait, sleep, extract_text, extract_attribute, extract_html, execute_js, upload_file, scroll, keyboard, open_tab, switch_tab
	Timestamp int64             `json:"timestamp"` // 时间戳（毫秒）
	Selector  string            `json:"selector"`  // CSS选择器
	XPath     string            `json:"xpath"`     // XPath选择器（更可靠）
	Value     string            `json:"value"`     // 输入值或选择值
	URL       string            `json:"url"`       // 导航URL
	Duration  int               `json:"duration"`  // 延迟时长（毫秒，用于 sleep 类型）
	X         int               `json:"x"`         // 鼠标X坐标
	Y         int               `json:"y"`         // 鼠标Y坐标
	Text      string            `json:"text"`      // 元素文本内容
	TagName   string            `json:"tag_name"`  // 元素标签名
	Attrs     map[string]string `json:"attrs"`     // 元素属性

	// 键盘事件相关字段
	Key string `json:"key,omitempty"` // 键盘按键（用于 keyboard 类型，如 "ctrl+c", "enter"）

	// 数据抓取相关字段
	ExtractType   string `json:"extract_type,omitempty"`   // text, attribute, html
	AttributeName string `json:"attribute_name,omitempty"` // 抓取的属性名
	JSCode        string `json:"js_code,omitempty"`        // JS 代码
	VariableName  string `json:"variable_name,omitempty"`  // 变量名
	ExtractedData string `json:"extracted_data,omitempty"` // 回放时填充

	// 文件上传相关字段
	FilePaths   []string `json:"file_paths,omitempty"`
	FileNames   []string `json:"file_names,omitempty"`
	Description string   `json:"description,omitempty"` // 人类可读描述
	Multiple    bool     `json:"multiple,omitempty"`
	Accept      string   `json:"accept,omitempty"`
	Remark      string   `json:"remark,omitempty"` // 操作备注

	// 滚动相关字段
	ScrollX int `json:"scroll_x,omitempty"`
	ScrollY int `json:"scroll_y,omitempty"`

	// =========================
	// 新增字段（v2，自愈核心）
	// =========================

	// ① 操作意图（结构化，而不是自由文本）
	Intent *ActionIntent `json:"intent,omitempty"`

	// ② Accessibility 语义信息（最重要）
	Accessibility *AccessibilityInfo `json:"accessibility,omitempty"`

	// ③ 上下文锚点（用于消歧 & 自愈）
	Context *ActionContext `json:"context,omitempty"`

	// ④ 录制证据（debug / 自愈评分用）
	Evidence *ActionEvidence `json:"evidence,omitempty"`
}

type ActionIntent struct {
	Verb   string `json:"verb,omitempty"`   // click, input, select, submit
	Object string `json:"object,omitempty"` // login button, email input
}

type AccessibilityInfo struct {
	Role  string `json:"role,omitempty"`  // button, textbox, link
	Name  string `json:"name,omitempty"`  // Sign In, Email
	Value string `json:"value,omitempty"` // 输入框当前值（可选）
}

type ActionContext struct {
	NearbyText   []string `json:"nearby_text,omitempty"`   // 附近可见文本
	AncestorTags []string `json:"ancestor_tags,omitempty"` // form, section
	FormHint     string   `json:"form_hint,omitempty"`     // login, search
}

type ActionEvidence struct {
	BackendDOMNodeID int64   `json:"backend_dom_node_id,omitempty"`
	AXNodeID         string  `json:"ax_node_id,omitempty"`
	Confidence       float64 `json:"confidence,omitempty"` // 录制时匹配置信度
}

// Script 自动化脚本
type Script struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	URL         string         `json:"url"`     // 起始URL
	Actions     []ScriptAction `json:"actions"` // 操作步骤
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Tags        []string       `json:"tags"`        // 标签
	Group       string         `json:"group"`       // 分组
	Duration    int64          `json:"duration"`    // 录制时长（毫秒）
	CanPublish  bool           `json:"can_publish"` // 是否可作为发布器使用
	CanFetch    bool           `json:"can_fetch"`   // 是否可作为抓取器使用

	// 下载文件信息
	DownloadedFiles []DownloadedFile `json:"downloaded_files,omitempty"` // 录制过程中下载的文件列表

	// MCP 相关字段
	IsMCPCommand          bool                   `json:"is_mcp_command"`          // 是否作为 MCP 命令对外提供
	MCPCommandName        string                 `json:"mcp_command_name"`        // MCP 命令名称（如 "execute_script"）
	MCPCommandDescription string                 `json:"mcp_command_description"` // MCP 命令描述
	MCPInputSchema        map[string]interface{} `json:"mcp_input_schema"`        // MCP 命令输入参数 schema（JSON Schema 格式）
}

func (s *Script) Copy() *Script {
	return &Script{
		ID:                    s.ID,
		Name:                  s.Name,
		Description:           s.Description,
		URL:                   s.URL,
		Actions:               s.Actions,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
		Tags:                  s.Tags,
		Group:                 s.Group,
		Duration:              s.Duration,
		CanPublish:            s.CanPublish,
		CanFetch:              s.CanFetch,
		DownloadedFiles:       s.DownloadedFiles,
		IsMCPCommand:          s.IsMCPCommand,
		MCPCommandName:        s.MCPCommandName,
		MCPCommandDescription: s.MCPCommandDescription,
		MCPInputSchema:        s.MCPInputSchema,
	}
}

// PlayResult 脚本回放结果
type PlayResult struct {
	Success       bool                   `json:"success"`        // 是否成功
	Message       string                 `json:"message"`        // 结果消息
	ExtractedData map[string]interface{} `json:"extracted_data"` // 抓取到的数据，key 为变量名或 action 索引
	Errors        []string               `json:"errors"`         // 错误信息列表
}
