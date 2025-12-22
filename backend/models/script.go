package models

import (
	"time"
)

// ScriptAction 脚本操作步骤
type ScriptAction struct {
	Type      string            `json:"type"`      // click, input, select, navigate, wait, sleep, extract_text, extract_attribute, extract_html, execute_js, upload_file
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

	// 数据抓取相关字段
	ExtractType   string `json:"extract_type,omitempty"`   // 抓取类型: text, attribute, html
	AttributeName string `json:"attribute_name,omitempty"` // 要抓取的属性名（用于 extract_attribute）
	JSCode        string `json:"js_code,omitempty"`        // 要执行的 JavaScript 代码（用于 execute_js）
	VariableName  string `json:"variable_name,omitempty"`  // 存储抓取数据的变量名（便于引用）
	ExtractedData string `json:"extracted_data,omitempty"` // 回放时抓取到的数据（运行时填充）

	// 文件上传相关字段
	FilePaths   []string `json:"file_paths,omitempty"`  // 要上传的文件路径（用于 upload_file）
	FileNames   []string `json:"file_names,omitempty"`  // 文件名（录制时记录，回放时可选）
	Description string   `json:"description,omitempty"` // 操作描述
	Multiple    bool     `json:"multiple,omitempty"`    // 是否支持多文件上传
	Accept      string   `json:"accept,omitempty"`      // 接受的文件类型

	// 滚动相关字段
	ScrollX int `json:"scroll_x,omitempty"` // 水平滚动位置（像素）
	ScrollY int `json:"scroll_y,omitempty"` // 垂直滚动位置（像素）
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

	// MCP 相关字段
	IsMCPCommand          bool                   `json:"is_mcp_command"`          // 是否作为 MCP 命令对外提供
	MCPCommandName        string                 `json:"mcp_command_name"`        // MCP 命令名称（如 "execute_script"）
	MCPCommandDescription string                 `json:"mcp_command_description"` // MCP 命令描述
	MCPInputSchema        map[string]interface{} `json:"mcp_input_schema"`        // MCP 命令输入参数 schema（JSON Schema 格式）
}

// PlayResult 脚本回放结果
type PlayResult struct {
	Success       bool                   `json:"success"`        // 是否成功
	Message       string                 `json:"message"`        // 结果消息
	ExtractedData map[string]interface{} `json:"extracted_data"` // 抓取到的数据，key 为变量名或 action 索引
	Errors        []string               `json:"errors"`         // 错误信息列表
}
