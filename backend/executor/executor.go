package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/browserwing/browserwing/services/browser"
	"github.com/go-rod/rod"
)

// Executor 提供通用的浏览器自动化能力
// 类似 agent-browser，提供语义化的浏览器操作接口
type Executor struct {
	Browser *browser.Manager
	ctx     context.Context
}

// NewExecutor 创建 Executor 实例
func NewExecutor(browser *browser.Manager) *Executor {
	return &Executor{
		Browser: browser,
		ctx:     context.Background(),
	}
}

// WithContext 设置上下文
func (e *Executor) WithContext(ctx context.Context) *Executor {
	e.ctx = ctx
	return e
}

// ========== 页面管理 ==========

// GetPage 获取当前活动页面
func (e *Executor) GetPage() *Page {
	rodPage := e.Browser.GetActivePage()
	if rodPage == nil {
		return nil
	}

	info, err := rodPage.Info()
	if err != nil {
		return nil
	}

	page := &Page{
		RodPage:     rodPage,
		URL:         info.URL,
		Title:       info.Title,
		LastUpdated: time.Now(),
	}

	return page
}

// GetSemanticTree 获取页面的语义树
func (e *Executor) GetSemanticTree(ctx context.Context) (*SemanticTree, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	return ExtractSemanticTree(ctx, page)
}

// RefreshSemanticTree 刷新语义树
func (e *Executor) RefreshSemanticTree(ctx context.Context, page *Page) error {
	tree, err := ExtractSemanticTree(ctx, page.RodPage)
	if err != nil {
		return err
	}

	page.SemanticTree = tree
	page.LastUpdated = time.Now()
	return nil
}

// ========== 智能元素查找 ==========

// FindElementByLabel 通过标签查找元素
func (e *Executor) FindElementByLabel(ctx context.Context, label string) (*SemanticNode, error) {
	tree, err := e.GetSemanticTree(ctx)
	if err != nil {
		return nil, err
	}

	node := tree.FindElementByLabel(label)
	if node == nil {
		return nil, fmt.Errorf("element not found with label: %s", label)
	}

	return node, nil
}

// FindElementsByType 通过类型查找元素
func (e *Executor) FindElementsByType(ctx context.Context, elemType string) ([]*SemanticNode, error) {
	tree, err := e.GetSemanticTree(ctx)
	if err != nil {
		return nil, err
	}

	return tree.FindElementsByType(elemType), nil
}

// GetClickableElements 获取所有可点击元素
func (e *Executor) GetClickableElements(ctx context.Context) ([]*SemanticNode, error) {
	tree, err := e.GetSemanticTree(ctx)
	if err != nil {
		return nil, err
	}

	return tree.GetClickableElements(), nil
}

// GetInputElements 获取所有输入元素
func (e *Executor) GetInputElements(ctx context.Context) ([]*SemanticNode, error) {
	tree, err := e.GetSemanticTree(ctx)
	if err != nil {
		return nil, err
	}

	return tree.GetInputElements(), nil
}

// ========== 智能操作方法 ==========

// ClickByLabel 通过标签点击元素
func (e *Executor) ClickByLabel(ctx context.Context, label string) (*OperationResult, error) {
	node, err := e.FindElementByLabel(ctx, label)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, err
	}

	return e.Click(ctx, node.Selector, nil)
}

// TypeByLabel 通过标签输入文本
func (e *Executor) TypeByLabel(ctx context.Context, label string, text string) (*OperationResult, error) {
	node, err := e.FindElementByLabel(ctx, label)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, err
	}

	return e.Type(ctx, node.Selector, text, nil)
}

// SelectByLabel 通过标签选择选项
func (e *Executor) SelectByLabel(ctx context.Context, label string, value string) (*OperationResult, error) {
	node, err := e.FindElementByLabel(ctx, label)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, err
	}

	return e.Select(ctx, node.Selector, value, nil)
}

// ========== 页面信息获取 ==========

// GetPageInfo 获取页面信息
func (e *Executor) GetPageInfo(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return &OperationResult{
			Success:   false,
			Error:     "No active page",
			Timestamp: time.Now(),
		}, fmt.Errorf("no active page")
	}

	info, err := page.Info()
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully retrieved page info",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"url":   info.URL,
			"title": info.Title,
		},
	}, nil
}

// GetPageContent 获取页面内容
func (e *Executor) GetPageContent(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return &OperationResult{
			Success:   false,
			Error:     "No active page",
			Timestamp: time.Now(),
		}, fmt.Errorf("no active page")
	}

	html, err := page.HTML()
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully retrieved page content",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"html": html,
		},
	}, nil
}

// GetPageText 获取页面文本
func (e *Executor) GetPageText(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return &OperationResult{
			Success:   false,
			Error:     "No active page",
			Timestamp: time.Now(),
		}, fmt.Errorf("no active page")
	}

	// 使用安全的 Eval 调用,防止 panic
	result, err := safeGetPageText(ctx, page)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully retrieved page text",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"text": result,
		},
	}, nil
}

// safeGetPageText 安全地获取页面文本,捕获可能的 panic
func safeGetPageText(ctx context.Context, page *rod.Page) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during get page text: %v", r)
			text = ""
		}
	}()

	result, evalErr := page.Eval(`() => document.body.innerText`)
	if evalErr != nil {
		return "", evalErr
	}

	if result != nil {
		text = result.Value.Str()
	}

	return text, nil
}

// ========== 批量操作 ==========

// ExecuteBatch 批量执行操作
func (e *Executor) ExecuteBatch(ctx context.Context, operations []Operation) (*BatchResult, error) {
	results := &BatchResult{
		Operations: make([]OperationResult, 0, len(operations)),
		StartTime:  time.Now(),
	}

	for i, op := range operations {
		var result *OperationResult
		var err error

		switch op.Type {
		case "navigate":
			url, _ := op.Params["url"].(string)
			result, err = e.Navigate(ctx, url, nil)

		case "click":
			identifier, _ := op.Params["identifier"].(string)
			result, err = e.Click(ctx, identifier, nil)

		case "type":
			identifier, _ := op.Params["identifier"].(string)
			text, _ := op.Params["text"].(string)
			result, err = e.Type(ctx, identifier, text, nil)

		case "select":
			identifier, _ := op.Params["identifier"].(string)
			value, _ := op.Params["value"].(string)
			result, err = e.Select(ctx, identifier, value, nil)

		case "wait":
			identifier, _ := op.Params["identifier"].(string)
			result, err = e.WaitFor(ctx, identifier, nil)

		default:
			result = &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Unknown operation type: %s", op.Type),
				Timestamp: time.Now(),
			}
			err = fmt.Errorf("unknown operation type: %s", op.Type)
		}

		if result != nil {
			results.Operations = append(results.Operations, *result)
		}

		if err != nil && op.StopOnError {
			results.Failed = i + 1
			break
		}

		if result != nil && result.Success {
			results.Success++
		} else {
			results.Failed++
		}
	}

	results.EndTime = time.Now()
	results.Duration = results.EndTime.Sub(results.StartTime)

	return results, nil
}

// Operation 批量操作定义
type Operation struct {
	Type        string                 `json:"type"`
	Params      map[string]interface{} `json:"params"`
	StopOnError bool                   `json:"stop_on_error"`
}

// BatchResult 批量操作结果
type BatchResult struct {
	Operations []OperationResult `json:"operations"`
	Success    int               `json:"success"`
	Failed     int               `json:"failed"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Duration   time.Duration     `json:"duration"`
}

// ========== 辅助方法 ==========

// EnsurePageReady 确保页面就绪
func (e *Executor) EnsurePageReady(ctx context.Context) error {
	page := e.Browser.GetActivePage()
	if page == nil {
		return fmt.Errorf("no active page")
	}

	// 等待页面加载完成
	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("failed to wait for page load: %w", err)
	}

	// 注入辅助脚本
	if err := InjectAccessibilityHelpers(ctx, page); err != nil {
		return fmt.Errorf("failed to inject helpers: %w", err)
	}

	return nil
}

// HighlightElementByLabel 高亮显示元素
func (e *Executor) HighlightElementByLabel(ctx context.Context, label string) error {
	node, err := e.FindElementByLabel(ctx, label)
	if err != nil {
		return err
	}

	page := e.Browser.GetActivePage()
	if page == nil {
		return fmt.Errorf("no active page")
	}

	return HighlightElement(ctx, page, node.Selector)
}

// GetRodPage 获取 Rod Page（供内部使用）
func (e *Executor) GetRodPage() *rod.Page {
	return e.Browser.GetActivePage()
}

// IsReady 检查 Executor 是否就绪
func (e *Executor) IsReady() bool {
	return e.Browser.IsRunning() && e.Browser.GetActivePage() != nil
}

// WaitUntilReady 等待 Executor 就绪
func (e *Executor) WaitUntilReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if e.IsReady() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for executor to be ready")
}
