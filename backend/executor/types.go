package executor

import (
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Page 表示一个浏览器页面及其上下文
type Page struct {
	RodPage              *rod.Page
	URL                  string
	Title                string
	AccessibilitySnapshot *AccessibilitySnapshot
	LastUpdated          time.Time
}

// AccessibilitySnapshot 表示页面的可访问性快照（基于 Accessibility Tree）
type AccessibilitySnapshot struct {
	Root         *AccessibilityNode                                      // 根节点
	Elements     map[string]*AccessibilityNode                           // AXNodeID -> Node 映射
	AXNodeMap    map[proto.AccessibilityAXNodeID]*proto.AccessibilityAXNode // AXNodeID -> AXNode 映射
	BackendIDMap map[proto.DOMBackendNodeID]*AccessibilityNode           // BackendNodeID -> Node 映射
}

// AccessibilityNode 表示页面中的一个可访问性节点（基于 Accessibility Node）
type AccessibilityNode struct {
	ID            string                        // 节点 ID（字符串形式的 AXNodeID）
	AXNodeID      proto.AccessibilityAXNodeID   // Accessibility 节点 ID
	BackendNodeID proto.DOMBackendNodeID        // DOM Backend 节点 ID
	Role          string                        // Accessibility Role（如 button, link, textbox 等）
	Label         string                        // 节点标签/名称
	Description   string                        // 节点描述
	Value         string                        // 节点值
	Text          string                        // 文本内容
	Placeholder   string                        // placeholder 属性
	Attributes    map[string]string             // 所有属性
	IsInteractive bool                          // 是否可交互
	IsEnabled     bool                          // 是否启用（非 disabled）
	Children      []*AccessibilityNode          // 子节点
	Metadata      map[string]interface{}        // 其他元数据
	
	// 保留兼容性字段
	Type       string           // 保留，映射到 Role
	Selector   string           // 保留，但可能为空
	XPath      string           // 保留，但可能为空
	Position   *ElementPosition // 保留，但可能为空
	IsVisible  bool             // 保留，但可能不准确
}

// ElementPosition 元素位置信息
type ElementPosition struct {
	X      float64 // X 坐标
	Y      float64 // Y 坐标
	Width  float64 // 宽度
	Height float64 // 高度
}

// OperationResult 操作结果
type OperationResult struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NavigateOptions 导航选项
type NavigateOptions struct {
	WaitUntil string        // 等待条件：load, domcontentloaded, networkidle
	Timeout   time.Duration // 超时时间
}

// ClickOptions 点击选项
type ClickOptions struct {
	WaitVisible bool          // 等待元素可见
	WaitEnabled bool          // 等待元素可用
	Timeout     time.Duration // 超时时间
	Button      string        // 鼠标按钮：left, right, middle
	ClickCount  int           // 点击次数
}

// TypeOptions 输入选项
type TypeOptions struct {
	Clear       bool          // 是否先清空
	WaitVisible bool          // 等待元素可见
	Timeout     time.Duration // 超时时间
	Delay       time.Duration // 每个字符之间的延迟
}

// SelectOptions 选择选项
type SelectOptions struct {
	WaitVisible bool          // 等待元素可见
	Timeout     time.Duration // 超时时间
}

// WaitForOptions 等待选项
type WaitForOptions struct {
	Timeout time.Duration // 超时时间
	State   string        // 等待状态：visible, hidden, attached, detached
}

// ScreenshotOptions 截图选项
type ScreenshotOptions struct {
	FullPage bool   // 是否截取完整页面
	Quality  int    // 质量 (0-100)
	Format   string // 格式：png, jpeg
}

// ExtractOptions 提取选项
type ExtractOptions struct {
	Selector string   // CSS 选择器
	Type     string   // 提取类型：text, html, attribute, property
	Attr     string   // 属性名（type=attribute 时使用）
	Multiple bool     // 是否提取多个元素
	Fields   []string // 要提取的字段列表
}

// HoverOptions 鼠标悬停选项
type HoverOptions struct {
	WaitVisible bool          // 等待元素可见
	Timeout     time.Duration // 超时时间
}

// PressKeyOptions 按键选项
type PressKeyOptions struct {
	Ctrl  bool // Ctrl 键
	Shift bool // Shift 键
	Alt   bool // Alt 键
	Meta  bool // Meta 键 (Command on Mac, Windows key on Windows)
}

