package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/services/browser"
	"github.com/go-rod/rod"
)

// Executor 提供通用的浏览器自动化能力
// 类似 agent-browser，提供语义化的浏览器操作接口
type Executor struct {
	Browser *browser.Manager
	ctx     context.Context

	// RefID 缓存（用于稳定的元素引用）
	refIDMutex       sync.RWMutex
	refIDMap         map[string]*RefData // refID -> 语义化定位器数据
	refIDCounter     int
	refIDSnapshot    *AccessibilitySnapshot
	refIDPageURL     string    // 缓存生成时的页面URL，用于检测页面切换（比指针更可靠）
	refIDPageTarget  string    // Chrome DevTools Protocol target ID
	refIDTimestamp   time.Time
	refIDTTL         time.Duration

	Recorder *OperationRecorder
}

// NewExecutor 创建 Executor 实例
func NewExecutor(browser *browser.Manager) *Executor {
	return &Executor{
		Browser:  browser,
		ctx:      context.Background(),
		refIDMap: make(map[string]*RefData),
		refIDTTL: 300 * time.Second,
		Recorder: NewOperationRecorder(),
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

// InvalidateSnapshotCache clears the cached accessibility snapshot.
// This should be called when the active page changes (tab switch, new tab, navigation, etc.)
func (e *Executor) InvalidateSnapshotCache() {
	e.refIDMutex.Lock()
	defer e.refIDMutex.Unlock()
	e.refIDSnapshot = nil
	e.refIDPageURL = ""
	e.refIDPageTarget = ""
	e.refIDMap = make(map[string]*RefData)
	e.refIDCounter = 0
}

// GetAccessibilitySnapshot 获取页面的可访问性快照（带 RefID 缓存）
func (e *Executor) GetAccessibilitySnapshot(ctx context.Context) (*AccessibilitySnapshot, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	// 获取页面标识信息（用于缓存验证）
	pageInfo, err := page.Info()
	if err != nil {
		logger.Warn(ctx, "[GetAccessibilitySnapshot] Failed to get page info: %v", err)
	}
	pageURL := ""
	pageTarget := ""
	if pageInfo != nil {
		pageURL = pageInfo.URL
		pageTarget = string(pageInfo.TargetID) // Convert proto.TargetTargetID to string
	}

	// 检查缓存：TTL 有效 且 页面未切换（使用 URL + TargetID 比较而非指针）
	e.refIDMutex.RLock()
	cacheValid := e.refIDSnapshot != nil &&
		time.Since(e.refIDTimestamp) < e.refIDTTL &&
		e.refIDPageURL == pageURL &&
		e.refIDPageTarget == pageTarget
	if cacheValid {
		logger.Info(ctx, "[GetAccessibilitySnapshot] Using cached snapshot (age: %v, %d refs, URL: %s)",
			time.Since(e.refIDTimestamp), len(e.refIDMap), pageURL)
		cachedSnapshot := e.refIDSnapshot
		e.refIDMutex.RUnlock()
		return cachedSnapshot, nil
	}
	e.refIDMutex.RUnlock()

	// 获取新快照
	logger.Info(ctx, "[GetAccessibilitySnapshot] Fetching new accessibility snapshot (URL: %s, TargetID: %s)",
		pageURL, pageTarget)
	snapshot, err := GetAccessibilitySnapshot(ctx, page)
	if err != nil {
		return nil, err
	}

	// 生成 RefID 并缓存
	e.refIDMutex.Lock()
	defer e.refIDMutex.Unlock()

	e.refIDMap = make(map[string]*RefData)
	e.refIDCounter = 0
	e.assignRefIDs(snapshot)
	e.refIDSnapshot = snapshot
	e.refIDPageURL = pageURL
	e.refIDPageTarget = pageTarget
	e.refIDTimestamp = time.Now()

	logger.Info(ctx, "[GetAccessibilitySnapshot] Cached new snapshot with %d refs (TTL: %v)",
		len(e.refIDMap), e.refIDTTL)

	return snapshot, nil
}

// assignRefIDs 为快照中的元素分配 RefID（参考 agent-browser 的实现）
// 使用 role+name+nth 而非 BackendNodeID，以提高稳定性
func (e *Executor) assignRefIDs(snapshot *AccessibilitySnapshot) {
	// 跟踪 role:name 组合，用于处理重复元素
	roleNameCounter := make(map[string]int) // "button:Submit" -> 0, 1, 2...
	
	// 为可点击元素分配 refID（e1, e2, e3...）
	clickables := snapshot.GetClickableElements()
	logger.Info(context.Background(), "[assignRefIDs] Assigning RefIDs to %d clickable elements", len(clickables))
	clickableCount := 0
	
	for _, node := range clickables {
		// 构建 role:name key
		key := fmt.Sprintf("%s:%s", node.Role, node.Label)
		nth := roleNameCounter[key]
		roleNameCounter[key]++
		
		// 分配 RefID
		e.refIDCounter++
		refID := fmt.Sprintf("e%d", e.refIDCounter)
		node.RefID = refID
		
		// 存储语义化定位器数据（参考 agent-browser）
		// 存储尽可能多的信息以精确定位元素
		refData := &RefData{
			Role:       node.Role,
			Name:       node.Label,
			Nth:        nth,
			BackendID:  int(node.BackendNodeID),
			Attributes: make(map[string]string),
		}
		
		// 对于链接，存储 href
		if node.Role == "link" && node.Attributes != nil {
			if href, ok := node.Attributes["href"]; ok {
				refData.Href = href
			}
		}
		
		// 存储关键属性（id, class）
		if node.Attributes != nil {
			if id, ok := node.Attributes["id"]; ok && id != "" {
				refData.Attributes["id"] = id
			}
			if class, ok := node.Attributes["class"]; ok && class != "" {
				refData.Attributes["class"] = class
			}
		}
		
		e.refIDMap[refID] = refData
		clickableCount++
		
		// 记录前10个元素用于调试
		if clickableCount <= 10 {
			logger.Info(context.Background(), "[assignRefIDs] %s -> role=%s, name=%s, nth=%d, backendID=%d", 
				refID, node.Role, node.Label, nth, node.BackendNodeID)
		}
	}
	logger.Info(context.Background(), "[assignRefIDs] Assigned %d RefIDs to clickable elements", clickableCount)

	// 为输入元素分配 refID
	inputs := snapshot.GetInputElements()
	logger.Info(context.Background(), "[assignRefIDs] Processing %d input elements", len(inputs))
	inputCount := 0
	
	for _, node := range inputs {
		// 检查是否已分配（可点击元素中可能包含输入元素）
		if node.RefID != "" {
			logger.Info(context.Background(), "[assignRefIDs] Input element already has RefID: %s", node.RefID)
			continue
		}
		
		// 构建 role:name key
		key := fmt.Sprintf("%s:%s", node.Role, node.Label)
		nth := roleNameCounter[key]
		roleNameCounter[key]++
		
		// 分配 RefID
		e.refIDCounter++
		refID := fmt.Sprintf("e%d", e.refIDCounter)
		node.RefID = refID
		
		// 存储语义化定位器数据
		refData := &RefData{
			Role:       node.Role,
			Name:       node.Label,
			Nth:        nth,
			BackendID:  int(node.BackendNodeID),
			Attributes: make(map[string]string),
		}
		
		// 存储 placeholder（对于输入元素）
		if node.Placeholder != "" {
			refData.Placeholder = node.Placeholder
		}
		
		// 存储关键属性
		if node.Attributes != nil {
			if id, ok := node.Attributes["id"]; ok && id != "" {
				refData.Attributes["id"] = id
			}
			if class, ok := node.Attributes["class"]; ok && class != "" {
				refData.Attributes["class"] = class
			}
		}
		
		e.refIDMap[refID] = refData
		inputCount++
		
		if inputCount <= 5 {
			logger.Info(context.Background(), "[assignRefIDs] %s -> role=%s, name=%s, nth=%d, backendID=%d", 
				refID, node.Role, node.Label, nth, node.BackendNodeID)
		}
	}
	logger.Info(context.Background(), "[assignRefIDs] Assigned %d RefIDs to input elements", inputCount)
	logger.Info(context.Background(), "[assignRefIDs] Total RefIDs in map: %d (using semantic locators)", len(e.refIDMap))
}

// InvalidateRefIDCache 清除 RefID 缓存
func (e *Executor) InvalidateRefIDCache() {
	e.refIDMutex.Lock()
	defer e.refIDMutex.Unlock()

	e.refIDMap = make(map[string]*RefData)
	e.refIDCounter = 0
	e.refIDSnapshot = nil
}

// RefreshAccessibilitySnapshot 刷新可访问性快照
func (e *Executor) RefreshAccessibilitySnapshot(ctx context.Context, page *Page) error {
	snapshot, err := GetAccessibilitySnapshot(ctx, page.RodPage)
	if err != nil {
		return err
	}

	page.AccessibilitySnapshot = snapshot
	page.LastUpdated = time.Now()
	return nil
}

// ========== 智能元素查找 ==========

// FindElementByLabel 通过标签查找元素
func (e *Executor) FindElementByLabel(ctx context.Context, label string) (*AccessibilityNode, error) {
	snapshot, err := e.GetAccessibilitySnapshot(ctx)
	if err != nil {
		return nil, err
	}

	node := snapshot.FindElementByLabel(label)
	if node == nil {
		return nil, fmt.Errorf("element not found with label: %s", label)
	}

	return node, nil
}

// FindElementsByType 通过类型查找元素
func (e *Executor) FindElementsByType(ctx context.Context, elemType string) ([]*AccessibilityNode, error) {
	snapshot, err := e.GetAccessibilitySnapshot(ctx)
	if err != nil {
		return nil, err
	}

	return snapshot.FindElementsByType(elemType), nil
}

// GetClickableElements 获取所有可点击元素
func (e *Executor) GetClickableElements(ctx context.Context) ([]*AccessibilityNode, error) {
	snapshot, err := e.GetAccessibilitySnapshot(ctx)
	if err != nil {
		return nil, err
	}

	return snapshot.GetClickableElements(), nil
}

// GetInputElements 获取所有输入元素
func (e *Executor) GetInputElements(ctx context.Context) ([]*AccessibilityNode, error) {
	snapshot, err := e.GetAccessibilitySnapshot(ctx)
	if err != nil {
		return nil, err
	}

	return snapshot.GetInputElements(), nil
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

// GetPageInfo 获取页面信息（增强版，参考 playwright-mcp 和 agent-browser）
func (e *Executor) GetPageInfo(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return &OperationResult{
			Success:   false,
			Error:     "No active page",
			Timestamp: time.Now(),
		}, fmt.Errorf("no active page")
	}

	// 收集所有信息
	pageInfo := make(map[string]interface{})

	// 1. 基本信息
	info, err := page.Info()
	if err == nil {
		pageInfo["url"] = info.URL
		pageInfo["title"] = info.Title
	}

	// 2. 视口大小
	viewport, err := page.Eval(`() => ({
		width: window.innerWidth,
		height: window.innerHeight,
		devicePixelRatio: window.devicePixelRatio
	})`)
	if err == nil && viewport != nil {
		pageInfo["viewport"] = viewport.Value.Val()
	}

	// 3. 文档状态
	docState, err := page.Eval(`() => ({
		readyState: document.readyState,
		documentElement: !!document.documentElement,
		body: !!document.body
	})`)
	if err == nil && docState != nil {
		pageInfo["documentState"] = docState.Value.Val()
	}

	// 4. 元素统计（类似 agent-browser 的 get count）
	stats, err := page.Eval(`() => ({
		links: document.querySelectorAll('a').length,
		buttons: document.querySelectorAll('button, [role="button"]').length,
		inputs: document.querySelectorAll('input, textarea, select').length,
		images: document.querySelectorAll('img').length,
		scripts: document.querySelectorAll('script').length,
		forms: document.querySelectorAll('form').length,
		iframes: document.querySelectorAll('iframe').length,
		headings: document.querySelectorAll('h1, h2, h3, h4, h5, h6').length
	})`)
	if err == nil && stats != nil {
		pageInfo["elementCounts"] = stats.Value.Val()
	}

	// 5. 滚动信息
	scrollInfo, err := page.Eval(`() => ({
		scrollX: window.scrollX || window.pageXOffset || 0,
		scrollY: window.scrollY || window.pageYOffset || 0,
		scrollWidth: document.documentElement.scrollWidth,
		scrollHeight: document.documentElement.scrollHeight,
		isScrollable: document.documentElement.scrollHeight > window.innerHeight
	})`)
	if err == nil && scrollInfo != nil {
		pageInfo["scroll"] = scrollInfo.Value.Val()
	}

	// 6. 元数据（Open Graph, meta tags）
	metadata, err := page.Eval(`() => {
		const getMeta = (name) => {
			const meta = document.querySelector('meta[name="' + name + '"], meta[property="' + name + '"]');
			return meta ? meta.content : null;
		};
		return {
			description: getMeta('description'),
			keywords: getMeta('keywords'),
			author: getMeta('author'),
			ogTitle: getMeta('og:title'),
			ogDescription: getMeta('og:description'),
			ogImage: getMeta('og:image'),
			ogUrl: getMeta('og:url'),
			ogType: getMeta('og:type'),
			twitterCard: getMeta('twitter:card'),
			twitterTitle: getMeta('twitter:title'),
			twitterDescription: getMeta('twitter:description'),
			twitterImage: getMeta('twitter:image'),
			viewport: getMeta('viewport'),
			charset: document.characterSet || document.charset
		};
	}`)
	if err == nil && metadata != nil {
		pageInfo["metadata"] = metadata.Value.Val()
	}

	// 7. 性能信息（页面加载时间）
	perfInfo, err := page.Eval(`() => {
		if (!window.performance || !window.performance.timing) {
			return null;
		}
		const timing = window.performance.timing;
		return {
			navigationStart: timing.navigationStart,
			domContentLoadedTime: timing.domContentLoadedEventEnd - timing.navigationStart,
			loadTime: timing.loadEventEnd - timing.navigationStart,
			domInteractive: timing.domInteractive - timing.navigationStart,
			domComplete: timing.domComplete - timing.navigationStart
		};
	}`)
	if err == nil && perfInfo != nil && perfInfo.Value.Val() != nil {
		pageInfo["performance"] = perfInfo.Value.Val()
	}

	// 8. 交互元素快速统计（可点击、可输入）
	interactiveInfo, err := page.Eval(`() => {
		const clickableElements = document.querySelectorAll('a, button, [role="button"], [onclick], [role="link"]');
		const inputElements = document.querySelectorAll('input, textarea, select, [role="textbox"], [role="combobox"]');
		return {
			clickableElements: clickableElements.length,
			inputElements: inputElements.length,
			visibleInputs: Array.from(inputElements).filter(el => {
				const style = window.getComputedStyle(el);
				return style.display !== 'none' && style.visibility !== 'hidden';
			}).length
		};
	}`)
	if err == nil && interactiveInfo != nil {
		pageInfo["interactive"] = interactiveInfo.Value.Val()
	}

	// 9. 页面语言和方向
	langInfo, err := page.Eval(`() => ({
		language: document.documentElement.lang || null,
		direction: document.documentElement.dir || 'ltr'
	})`)
	if err == nil && langInfo != nil {
		pageInfo["language"] = langInfo.Value.Val()
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully retrieved enhanced page info",
		Timestamp: time.Now(),
		Data:      pageInfo,
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

// ========== 录制模式 ==========

// StartRecordMode 开启操作录制模式
func (e *Executor) StartRecordMode() {
	e.Recorder.Enable()
}

// StopRecordMode 停止录制模式并返回所有记录的操作（OperationRecord 格式）
func (e *Executor) StopRecordMode() []OperationRecord {
	ops := e.Recorder.GetOperations()
	e.Recorder.Disable()
	e.Recorder.Reset()
	return ops
}

// StopRecordModeAsOpRecords 停止录制模式并返回 models.OpRecord 格式
func (e *Executor) StopRecordModeAsOpRecords() []models.OpRecord {
	return convertToOpRecords(e.StopRecordMode())
}

// IsRecordMode 检查是否处于录制模式
func (e *Executor) IsRecordMode() bool {
	return e.Recorder.IsEnabled()
}

// GetRecordedOps 获取当前录制的操作（不清空）
func (e *Executor) GetRecordedOps() []OperationRecord {
	return e.Recorder.GetOperations()
}

// GetRecordedOpsAsOpRecords returns recorded ops in models.OpRecord format (no import cycle)
func (e *Executor) GetRecordedOpsAsOpRecords() []models.OpRecord {
	return convertToOpRecords(e.GetRecordedOps())
}

// NavigateForExplore navigates to a URL (used by explorer)
func (e *Executor) NavigateForExplore(ctx context.Context, url string) error {
	_, err := e.Navigate(ctx, url, nil)
	return err
}

func convertToOpRecords(ops []OperationRecord) []models.OpRecord {
	result := make([]models.OpRecord, len(ops))
	for i, op := range ops {
		result[i] = models.OpRecord{
			Type:          op.Type,
			Identifier:    op.Identifier,
			ResolvedXPath: op.ResolvedXPath,
			Value:         op.Value,
			URL:           op.URL,
			Key:           op.Key,
			JSCode:        op.JSCode,
			Success:       op.Success,
			Timestamp:     op.Timestamp,
			ElementInfo:   op.ElementInfo,
		}
	}
	return result
}

// recordOp 记录一个操作（内部调用）
func (e *Executor) recordOp(op OperationRecord) {
	e.Recorder.Record(op)
}

// getElementXPath 通过 JS 获取元素的稳定 Full XPath
func getElementXPath(elem *rod.Element) string {
	result, err := elem.Eval(`() => {
		function getXPath(el) {
			if (!el || el.nodeType !== 1) return '';
			if (el.id && document.querySelectorAll('#' + CSS.escape(el.id)).length === 1) {
				return '//*[@id="' + el.id + '"]';
			}
			var parts = [];
			while (el && el.nodeType === 1) {
				var idx = 0;
				var sib = el.previousSibling;
				while (sib) {
					if (sib.nodeType === 1 && sib.nodeName === el.nodeName) idx++;
					sib = sib.previousSibling;
				}
				var part = el.nodeName.toLowerCase();
				if (idx > 0 || el.nextSibling) {
					var total = 0;
					var next = el.nextSibling;
					while (next) {
						if (next.nodeType === 1 && next.nodeName === el.nodeName) total++;
						next = next.nextSibling;
					}
					if (idx > 0 || total > 0) {
						part += '[' + (idx + 1) + ']';
					}
				}
				parts.unshift(part);
				el = el.parentNode;
			}
			return '/' + parts.join('/');
		}
		return getXPath(this);
	}`)
	if err != nil {
		return ""
	}
	return result.Value.Str()
}

// getElementInfo 获取元素的基本信息（tag, text）
func getElementInfo(elem *rod.Element) map[string]string {
	info := make(map[string]string)
	tagResult, err := elem.Eval(`() => this.tagName ? this.tagName.toLowerCase() : ''`)
	if err == nil {
		info["tag"] = tagResult.Value.Str()
	}
	text, err := elem.Text()
	if err == nil && len(text) > 0 {
		if len(text) > 100 {
			text = text[:100]
		}
		info["text"] = text
	}
	return info
}
