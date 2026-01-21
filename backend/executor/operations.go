package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// Navigate 导航到指定 URL
func (e *Executor) Navigate(ctx context.Context, url string, opts *NavigateOptions) (*OperationResult, error) {
	logger.Info(ctx, "[Navigate] Starting navigation to %s", url)

	if !e.Browser.IsRunning() {
		err := e.Browser.Start(ctx)
		if err != nil {
			return &OperationResult{
				Success:   false,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}, err
		}
		logger.Info(ctx, "[Navigate] Browser started")
	}
	logger.Info(ctx, "[Navigate] Browser is running")

	if opts == nil {
		opts = &NavigateOptions{
			WaitUntil: "load",
			Timeout:   60 * time.Second, // 增加默认超时到60秒
		}
	}
	logger.Info(ctx, "[Navigate] Using timeout: %v, wait_until: %s", opts.Timeout, opts.WaitUntil)

	// 获取或创建页面
	logger.Info(ctx, "[Navigate] Getting active page...")
	page := e.Browser.GetActivePage()
	if page == nil {
		logger.Info(ctx, "[Navigate] No active page, creating new page...")
		// 如果没有活动页面，通过 OpenPage 创建新页面（会自动导航）
		err := e.Browser.OpenPage(url, "", true)
		if err != nil {
			logger.Error(ctx, "[Navigate] Failed to open page: %s", err.Error())
			return &OperationResult{
				Success:   false,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}, err
		}
		logger.Info(ctx, "[Navigate] Page opened successfully")

		page = e.Browser.GetActivePage()
		if page == nil {
			logger.Error(ctx, "[Navigate] Failed to get active page after opening")
			return &OperationResult{
				Success:   false,
				Error:     "Failed to get active page",
				Timestamp: time.Now(),
			}, fmt.Errorf("failed to get active page")
		}
		logger.Info(ctx, "[Navigate] Got active page")
	} else {
		logger.Info(ctx, "[Navigate] Using existing page, navigating...")
		// 如果已有活动页面，直接导航
		err := page.Timeout(opts.Timeout).Navigate(url)
		if err != nil {
			logger.Error(ctx, "[Navigate] Failed to navigate to page: %s", err.Error())
			return &OperationResult{
				Success:   false,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}, err
		}
		logger.Info(ctx, "[Navigate] Navigation completed")
	}

	// 等待页面加载 - 使用 panic 恢复机制防止 rod 库内部错误
	logger.Info(ctx, "[Navigate] Waiting for page load (condition: %s)...", opts.WaitUntil)
	waitErr := safeWaitForPageLoad(ctx, page, opts.WaitUntil)
	if waitErr != nil {
		logger.Warn(ctx, "[Navigate] Wait for page load failed: %v (continuing anyway)", waitErr)
		// 不返回错误,因为页面可能已经部分加载了,继续处理
	} else {
		logger.Info(ctx, "[Navigate] Page load completed")
	}

	logger.Info(ctx, "[Navigate] Successfully navigated to %s", url)

	// 获取页面语义树（带超时控制）
	// 注意：这里同步调用，但用带超时的 context
	var semanticTreeText string

	logger.Info(ctx, "[Navigate] Starting semantic tree extraction...")
	// 创建一个带超时的 context（10秒超时）
	treeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 直接调用，不使用 goroutine 避免资源竞争
	tree, err := e.GetSemanticTree(treeCtx)
	if err != nil {
		if err == context.DeadlineExceeded {
			logger.Warn(ctx, "[Navigate] Semantic tree extraction timed out after 10s")
		} else if err != context.Canceled {
			logger.Warn(ctx, "[Navigate] Failed to extract semantic tree: %s", err.Error())
		}
		// 不影响导航成功，只是没有语义树
	} else if tree != nil {
		semanticTreeText = tree.SerializeToSimpleText()
		logger.Info(ctx, "[Navigate] Successfully extracted semantic tree with %d elements", len(tree.Elements))
	} else {
		logger.Warn(ctx, "[Navigate] Semantic tree is nil")
	}

	result := &OperationResult{
		Success:   true,
		Message:   "Successfully navigated to " + url,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"url": url,
		},
	}

	// 如果获取到语义树，添加到返回结果中
	if semanticTreeText != "" {
		result.Data["semantic_tree"] = semanticTreeText
	}

	return result, nil
}

// Click 点击元素
func (e *Executor) Click(ctx context.Context, identifier string, opts *ClickOptions) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		logger.Error(ctx, "Failed to get active page")
		return nil, fmt.Errorf("no active page")
	}

	if opts == nil {
		opts = &ClickOptions{
			WaitVisible: true,
			WaitEnabled: true,
			Timeout:     10 * time.Second,
			Button:      "left",
			ClickCount:  1,
		}
	}

	// 查找元素（带超时）
	elem, err := e.findElementWithTimeout(ctx, page, identifier, opts.Timeout)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Element not found: %s", identifier),
			Timestamp: time.Now(),
		}, err
	}

	// 等待元素可见
	if opts.WaitVisible {
		elem = elem.Timeout(opts.Timeout)
		if err := elem.WaitVisible(); err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Element not visible: %s (timeout after %v)", identifier, opts.Timeout),
				Timestamp: time.Now(),
			}, err
		}
	}

	// 等待元素可用
	if opts.WaitEnabled {
		elem = elem.Timeout(opts.Timeout)
		if err := elem.WaitEnabled(); err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Element not enabled: %s (timeout after %v)", identifier, opts.Timeout),
				Timestamp: time.Now(),
			}, err
		}
	}

	// 滚动到元素
	if err := elem.ScrollIntoView(); err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to scroll to element: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	// 点击
	var button proto.InputMouseButton
	switch opts.Button {
	case "right":
		button = proto.InputMouseButtonRight
	case "middle":
		button = proto.InputMouseButtonMiddle
	default:
		button = proto.InputMouseButtonLeft
	}

	for i := 0; i < opts.ClickCount; i++ {
		if err := elem.Click(button, 1); err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to click element: %s", err.Error()),
				Timestamp: time.Now(),
			}, err
		}
		if i < opts.ClickCount-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 同时返回当前的页面语义树
	tree, err := e.GetSemanticTree(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to get semantic tree: %s", err.Error())
	}
	var semanticTreeText string
	if tree != nil {
		semanticTreeText = tree.SerializeToSimpleText()
	}

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully clicked element: %s", identifier),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"semantic_tree": semanticTreeText,
		},
	}, nil
}

// Type 在元素中输入文本
func (e *Executor) Type(ctx context.Context, identifier string, text string, opts *TypeOptions) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if opts == nil {
		opts = &TypeOptions{
			Clear:       true,
			WaitVisible: true,
			Timeout:     10 * time.Second,
			Delay:       0,
		}
	}

	// 查找元素（带超时）
	elem, err := e.findElementWithTimeout(ctx, page, identifier, opts.Timeout)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Element not found: %s", identifier),
			Timestamp: time.Now(),
		}, err
	}

	// 等待元素可见
	if opts.WaitVisible {
		elem = elem.Timeout(opts.Timeout)
		if err := elem.WaitVisible(); err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Element not visible: %s (timeout after %v)", identifier, opts.Timeout),
				Timestamp: time.Now(),
			}, err
		}
	}

	// 聚焦元素
	if err := elem.Focus(); err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to focus element: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	// 清空现有内容
	if opts.Clear {
		if err := elem.SelectAllText(); err == nil {
			page.Keyboard.Press(input.Backspace)
		}
	}

	// 输入文本
	if opts.Delay > 0 {
		// 逐字符输入
		for _, char := range text {
			if err := elem.Input(string(char)); err != nil {
				return &OperationResult{
					Success:   false,
					Error:     fmt.Sprintf("Failed to input text: %s", err.Error()),
					Timestamp: time.Now(),
				}, err
			}
			time.Sleep(opts.Delay)
		}
	} else {
		// 一次性输入
		if err := elem.Input(text); err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to input text: %s", err.Error()),
				Timestamp: time.Now(),
			}, err
		}
	}

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully typed into element: %s", identifier),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"text": text,
		},
	}, nil
}

// Select 选择下拉框选项
func (e *Executor) Select(ctx context.Context, identifier string, value string, opts *SelectOptions) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if opts == nil {
		opts = &SelectOptions{
			WaitVisible: true,
			Timeout:     10 * time.Second,
		}
	}

	// 查找元素（带超时）
	elem, err := e.findElementWithTimeout(ctx, page, identifier, opts.Timeout)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Element not found: %s", identifier),
			Timestamp: time.Now(),
		}, err
	}

	// 等待元素可见
	if opts.WaitVisible {
		elem = elem.Timeout(opts.Timeout)
		if err := elem.WaitVisible(); err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Element not visible: %s (timeout after %v)", identifier, opts.Timeout),
				Timestamp: time.Now(),
			}, err
		}
	}

	// 选择选项
	if err := elem.Select([]string{value}, true, rod.SelectorTypeText); err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to select option: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully selected option: %s", value),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": value,
		},
	}, nil
}

// GetText 获取元素文本
func (e *Executor) GetText(ctx context.Context, identifier string) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	// 使用默认10秒超时
	elem, err := e.findElementWithTimeout(ctx, page, identifier, 10*time.Second)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Element not found: %s", identifier),
			Timestamp: time.Now(),
		}, err
	}

	text, err := elem.Text()
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to get text: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully retrieved text",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"text": text,
		},
	}, nil
}

// GetValue 获取元素值
func (e *Executor) GetValue(ctx context.Context, identifier string) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	// 使用默认10秒超时
	elem, err := e.findElementWithTimeout(ctx, page, identifier, 10*time.Second)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Element not found: %s", identifier),
			Timestamp: time.Now(),
		}, err
	}

	value, err := elem.Property("value")
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to get value: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully retrieved value",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"value": value.String(),
		},
	}, nil
}

// WaitFor 等待元素
func (e *Executor) WaitFor(ctx context.Context, identifier string, opts *WaitForOptions) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if opts == nil {
		opts = &WaitForOptions{
			Timeout: 30 * time.Second,
			State:   "visible",
		}
	}

	// 查找元素（带超时）
	elem, err := e.findElementWithTimeout(ctx, page, identifier, opts.Timeout)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Element not found: %s", identifier),
			Timestamp: time.Now(),
		}, err
	}

	elem = elem.Timeout(opts.Timeout)

	switch opts.State {
	case "visible":
		err = elem.WaitVisible()
	case "hidden":
		err = elem.WaitInvisible()
	case "enabled":
		err = elem.WaitEnabled()
	default:
		err = elem.WaitLoad()
	}

	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Wait failed for state '%s': %s (timeout after %v)", opts.State, err.Error(), opts.Timeout),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully waited for element: %s", identifier),
		Timestamp: time.Now(),
	}, nil
}

// Extract 提取数据
func (e *Executor) Extract(ctx context.Context, opts *ExtractOptions) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if opts == nil {
		return &OperationResult{
			Success:   false,
			Error:     "Extract options required",
			Timestamp: time.Now(),
		}, fmt.Errorf("extract options required")
	}

	var result interface{}

	if opts.Multiple {
		// 提取多个元素
		elements, err := page.Elements(opts.Selector)
		if err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to find elements: %s", err.Error()),
				Timestamp: time.Now(),
			}, err
		}

		results := make([]map[string]interface{}, 0, len(elements))
		for _, elem := range elements {
			data, err := e.extractElementData(elem, opts)
			if err != nil {
				continue
			}
			results = append(results, data)
		}
		result = results
	} else {
		// 提取单个元素
		elem, err := page.Element(opts.Selector)
		if err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to find element: %s", err.Error()),
				Timestamp: time.Now(),
			}, err
		}

		data, err := e.extractElementData(elem, opts)
		if err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Failed to extract data: %s", err.Error()),
				Timestamp: time.Now(),
			}, err
		}
		result = data
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully extracted data",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"result": result,
		},
	}, nil
}

// extractElementData 提取元素数据
func (e *Executor) extractElementData(elem *rod.Element, opts *ExtractOptions) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch opts.Type {
	case "text":
		text, err := elem.Text()
		if err != nil {
			return nil, err
		}
		data["text"] = text

	case "html":
		html, err := elem.HTML()
		if err != nil {
			return nil, err
		}
		data["html"] = html

	case "attribute":
		if opts.Attr != "" {
			attr, err := elem.Attribute(opts.Attr)
			if err != nil || attr == nil {
				return nil, err
			}
			data[opts.Attr] = *attr
		}

	case "property":
		if opts.Attr != "" {
			prop, err := elem.Property(opts.Attr)
			if err != nil {
				return nil, err
			}
			data[opts.Attr] = prop.String()
		}

	default:
		// 提取指定字段
		if len(opts.Fields) > 0 {
			for _, field := range opts.Fields {
				switch field {
				case "text":
					if text, err := elem.Text(); err == nil {
						data["text"] = text
					}
				case "html":
					if html, err := elem.HTML(); err == nil {
						data["html"] = html
					}
				case "value":
					if val, err := elem.Property("value"); err == nil {
						data["value"] = val.String()
					}
				case "href":
					if href, err := elem.Attribute("href"); err == nil && href != nil {
						data["href"] = *href
					}
				case "src":
					if src, err := elem.Attribute("src"); err == nil && src != nil {
						data["src"] = *src
					}
				}
			}
		} else {
			// 默认提取文本
			text, err := elem.Text()
			if err != nil {
				return nil, err
			}
			data["text"] = text
		}
	}

	return data, nil
}

// findElement 查找元素（支持多种方式），带超时支持
func (e *Executor) findElement(ctx context.Context, page *rod.Page, identifier string) (*rod.Element, error) {
	return e.findElementWithTimeout(ctx, page, identifier, 10*time.Second)
}

// findElementWithTimeout 查找元素（支持多种方式），带自定义超时
func (e *Executor) findElementWithTimeout(ctx context.Context, page *rod.Page, identifier string, timeout time.Duration) (*rod.Element, error) {
	// 设置超时
	timeoutPage := page.Timeout(timeout)

	// 0. 尝试解析语义树索引格式，例如 "Input Element [1]", "Clickable Element [2]", "[3]"
	if elem, err := e.findElementBySemanticIndex(ctx, page, identifier); err == nil && elem != nil {
		return elem, nil
	}

	// 1. 尝试作为 CSS 选择器
	if elem, err := timeoutPage.Element(identifier); err == nil {
		return elem, nil
	}

	// 2. 尝试作为 XPath
	if elem, err := timeoutPage.ElementX(identifier); err == nil {
		return elem, nil
	}

	// 3. 尝试通过文本查找
	if elem, err := timeoutPage.ElementR("button", identifier); err == nil {
		return elem, nil
	}
	if elem, err := timeoutPage.ElementR("a", identifier); err == nil {
		return elem, nil
	}

	// 4. 尝试通过 aria-label 查找
	selector := fmt.Sprintf("[aria-label*='%s']", identifier)
	if elem, err := timeoutPage.Element(selector); err == nil {
		return elem, nil
	}

	// 5. 尝试通过 placeholder 查找
	selector = fmt.Sprintf("[placeholder*='%s']", identifier)
	if elem, err := timeoutPage.Element(selector); err == nil {
		return elem, nil
	}

	return nil, fmt.Errorf("element not found: %s (timeout after %v)", identifier, timeout)
}

// findElementBySemanticIndex 通过语义树索引查找元素
// 支持格式：
// - "Input Element [1]"
// - "Clickable Element [2]"
// - "[3]"
// - "input [1]"
// - "clickable [2]"
func (e *Executor) findElementBySemanticIndex(ctx context.Context, page *rod.Page, identifier string) (*rod.Element, error) {
	// 使用正则解析索引格式
	identifier = strings.TrimSpace(identifier)

	// 尝试匹配不同的索引格式
	var elementType string
	var index int

	// 格式 1: "Input Element [N]" 或 "Clickable Element [N]"
	if strings.Contains(identifier, "Input Element") {
		_, err := fmt.Sscanf(identifier, "Input Element [%d]", &index)
		if err == nil {
			elementType = "input"
		}
	} else if strings.Contains(identifier, "Clickable Element") {
		_, err := fmt.Sscanf(identifier, "Clickable Element [%d]", &index)
		if err == nil {
			elementType = "clickable"
		}
	} else if strings.HasPrefix(strings.ToLower(identifier), "input [") {
		// 格式 2: "input [N]"
		_, err := fmt.Sscanf(identifier, "input [%d]", &index)
		if err != nil {
			_, err = fmt.Sscanf(identifier, "Input [%d]", &index)
		}
		if err == nil {
			elementType = "input"
		}
	} else if strings.HasPrefix(strings.ToLower(identifier), "clickable [") {
		// 格式 3: "clickable [N]"
		_, err := fmt.Sscanf(identifier, "clickable [%d]", &index)
		if err != nil {
			_, err = fmt.Sscanf(identifier, "Clickable [%d]", &index)
		}
		if err == nil {
			elementType = "clickable"
		}
	} else if strings.HasPrefix(identifier, "[") && strings.HasSuffix(identifier, "]") {
		// 格式 4: "[N]" - 需要从上下文推断类型，默认尝试输入元素
		_, err := fmt.Sscanf(identifier, "[%d]", &index)
		if err == nil {
			elementType = "any" // 尝试所有类型
		}
	}

	if index <= 0 {
		return nil, fmt.Errorf("invalid semantic index")
	}

	// 获取语义树
	tree, err := e.GetSemanticTree(ctx)
	if err != nil {
		return nil, err
	}

	// 根据类型查找对应的元素
	var targetNode *SemanticNode

	switch elementType {
	case "input":
		inputs := tree.GetInputElements()
		if index > 0 && index <= len(inputs) {
			targetNode = inputs[index-1]
		}
	case "clickable":
		clickables := tree.GetClickableElements()
		if index > 0 && index <= len(clickables) {
			targetNode = clickables[index-1]
		}
	case "any":
		// 先尝试输入元素
		inputs := tree.GetInputElements()
		if index > 0 && index <= len(inputs) {
			targetNode = inputs[index-1]
		}
		// 如果没找到，再尝试可点击元素
		if targetNode == nil {
			clickables := tree.GetClickableElements()
			if index > 0 && index <= len(clickables) {
				targetNode = clickables[index-1]
			}
		}
	}

	if targetNode == nil {
		return nil, fmt.Errorf("element not found at index %d", index)
	}

	// 通过语义节点查找实际的 Rod 元素
	return GetElementFromPage(ctx, page, targetNode)
}

// Hover 鼠标悬停
func (e *Executor) Hover(ctx context.Context, identifier string, opts *HoverOptions) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if opts == nil {
		opts = &HoverOptions{
			WaitVisible: true,
			Timeout:     10 * time.Second,
		}
	}

	// 查找元素
	elem, err := e.findElementWithTimeout(ctx, page, identifier, opts.Timeout)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Element not found: %s", identifier),
			Timestamp: time.Now(),
		}, err
	}

	// 等待元素可见
	if opts.WaitVisible {
		elem = elem.Timeout(opts.Timeout)
		if err := elem.WaitVisible(); err != nil {
			return &OperationResult{
				Success:   false,
				Error:     fmt.Sprintf("Element not visible: %s", identifier),
				Timestamp: time.Now(),
			}, err
		}
	}

	// 悬停
	if err := elem.Hover(); err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to hover: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully hovered over element: %s", identifier),
		Timestamp: time.Now(),
	}, nil
}

// ScrollToBottom 滚动到页面底部
func (e *Executor) ScrollToBottom(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	// 使用安全的 Eval 包装,防止 panic
	err := safeScrollEval(ctx, page)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to scroll: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully scrolled to bottom",
		Timestamp: time.Now(),
	}, nil
}

// GoBack 后退
func (e *Executor) GoBack(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if err := page.NavigateBack(); err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to go back: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully navigated back",
		Timestamp: time.Now(),
	}, nil
}

// GoForward 前进
func (e *Executor) GoForward(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if err := page.NavigateForward(); err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to go forward: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully navigated forward",
		Timestamp: time.Now(),
	}, nil
}

// Reload 刷新页面
func (e *Executor) Reload(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if err := page.Reload(); err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to reload: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully reloaded page",
		Timestamp: time.Now(),
	}, nil
}

// Screenshot 截图
func (e *Executor) Screenshot(ctx context.Context, opts *ScreenshotOptions) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if opts == nil {
		opts = &ScreenshotOptions{
			FullPage: false,
			Quality:  80,
			Format:   "png",
		}
	}

	var format proto.PageCaptureScreenshotFormat
	if opts.Format == "jpeg" || opts.Format == "jpg" {
		format = proto.PageCaptureScreenshotFormatJpeg
	} else {
		format = proto.PageCaptureScreenshotFormatPng
	}

	var data []byte
	var err error

	if opts.FullPage {
		data, err = page.Screenshot(opts.FullPage, &proto.PageCaptureScreenshot{
			Format:  format,
			Quality: &opts.Quality,
		})
	} else {
		data, err = page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format:  format,
			Quality: &opts.Quality,
		})
	}

	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to take screenshot: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully captured screenshot (%d bytes)", len(data)),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"data":   data,
			"format": opts.Format,
			"size":   len(data),
		},
	}, nil
}

// Evaluate 执行 JavaScript 代码
func (e *Executor) Evaluate(ctx context.Context, script string) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	// 使用安全的执行方式,防止 rod 库 panic
	var result interface{}
	err := safeEvaluate(ctx, page, script, &result)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to execute script: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully executed script",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"result": result,
		},
	}, nil
}

// PressKey 按键
func (e *Executor) PressKey(ctx context.Context, key string, opts *PressKeyOptions) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	if opts == nil {
		opts = &PressKeyOptions{}
	}

	// 创建 keyboard
	keyboard := page.Keyboard

	// 按下修饰键
	if opts.Ctrl {
		keyboard.Press(input.ControlLeft)
		defer keyboard.Release(input.ControlLeft)
	}
	if opts.Shift {
		keyboard.Press(input.ShiftLeft)
		defer keyboard.Release(input.ShiftLeft)
	}
	if opts.Alt {
		keyboard.Press(input.AltLeft)
		defer keyboard.Release(input.AltLeft)
	}
	if opts.Meta {
		keyboard.Press(input.MetaLeft)
		defer keyboard.Release(input.MetaLeft)
	}

	// 按下并释放目标键
	var keyCode input.Key
	switch strings.ToLower(key) {
	case "enter", "return":
		keyCode = input.Enter
	case "tab":
		keyCode = input.Tab
	case "escape", "esc":
		keyCode = input.Escape
	case "backspace":
		keyCode = input.Backspace
	case "delete":
		keyCode = input.Delete
	case "arrowup", "up":
		keyCode = input.ArrowUp
	case "arrowdown", "down":
		keyCode = input.ArrowDown
	case "arrowleft", "left":
		keyCode = input.ArrowLeft
	case "arrowright", "right":
		keyCode = input.ArrowRight
	case "home":
		keyCode = input.Home
	case "end":
		keyCode = input.End
	case "pageup":
		keyCode = input.PageUp
	case "pagedown":
		keyCode = input.PageDown
	case "space":
		keyCode = input.Space
	default:
		// 单个字符
		if len(key) == 1 {
			err := keyboard.Type(input.Key(key[0]))
			if err != nil {
				return &OperationResult{
					Success:   false,
					Error:     fmt.Sprintf("Failed to press key: %s", err.Error()),
					Timestamp: time.Now(),
				}, err
			}
			return &OperationResult{
				Success:   true,
				Message:   fmt.Sprintf("Successfully pressed key: %s", key),
				Timestamp: time.Now(),
			}, nil
		}
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Unknown key: %s", key),
			Timestamp: time.Now(),
		}, fmt.Errorf("unknown key: %s", key)
	}

	if err := keyboard.Press(keyCode); err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to press key: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}
	keyboard.Release(keyCode)

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully pressed key: %s", key),
		Timestamp: time.Now(),
	}, nil
}

// Resize 调整浏览器窗口大小
func (e *Executor) Resize(ctx context.Context, width, height int) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  width,
		Height: height,
	})
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to resize window: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully resized window to %dx%d", width, height),
		Timestamp: time.Now(),
	}, nil
}

// GetConsoleMessages 获取控制台消息
func (e *Executor) GetConsoleMessages(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	// 收集控制台消息
	messages := []map[string]interface{}{}

	// 监听控制台事件
	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		msg := map[string]interface{}{
			"type":      e.Type,
			"timestamp": time.Now().Format(time.RFC3339),
			"args":      []string{},
		}
		for _, arg := range e.Args {
			if arg.Value.Val() != nil {
				msg["args"] = append(msg["args"].([]string), fmt.Sprintf("%v", arg.Value.Val()))
			}
		}
		messages = append(messages, msg)
	})()

	// 等待一小段时间以收集消息
	time.Sleep(100 * time.Millisecond)

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Retrieved %d console messages", len(messages)),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"messages": messages,
		},
	}, nil
}

// HandleDialog 处理对话框（alert, confirm, prompt）
func (e *Executor) HandleDialog(ctx context.Context, accept bool, text string) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	// 设置对话框处理器
	go page.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
		if accept {
			proto.PageHandleJavaScriptDialog{
				Accept:     true,
				PromptText: text,
			}.Call(page)
		} else {
			proto.PageHandleJavaScriptDialog{
				Accept: false,
			}.Call(page)
		}
	})()

	return &OperationResult{
		Success:   true,
		Message:   "Dialog handler configured",
		Timestamp: time.Now(),
	}, nil
}

// FileUpload 上传文件
func (e *Executor) FileUpload(ctx context.Context, identifier string, filePaths []string) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	elem, err := e.findElementWithTimeout(ctx, page, identifier, 10*time.Second)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to find element: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	err = elem.SetFiles(filePaths)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to upload files: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Successfully uploaded %d file(s)", len(filePaths)),
		Timestamp: time.Now(),
	}, nil
}

// Drag 拖拽元素
func (e *Executor) Drag(ctx context.Context, fromIdentifier, toIdentifier string) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	fromElem, err := e.findElementWithTimeout(ctx, page, fromIdentifier, 10*time.Second)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to find source element: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	toElem, err := e.findElementWithTimeout(ctx, page, toIdentifier, 10*time.Second)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to find target element: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	// 获取源元素和目标元素的位置
	fromBox, err := fromElem.Shape()
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to get source element shape: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	toBox, err := toElem.Shape()
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to get target element shape: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	// 计算中心点
	fromRect := fromBox.Box()
	toRect := toBox.Box()
	fromCenter := proto.Point{
		X: fromRect.X + fromRect.Width/2,
		Y: fromRect.Y + fromRect.Height/2,
	}
	toCenter := proto.Point{
		X: toRect.X + toRect.Width/2,
		Y: toRect.Y + toRect.Height/2,
	}

	// 执行拖拽
	err = page.Mouse.MoveLinear(fromCenter, 10)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to move to source: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	err = page.Mouse.Down(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to mouse down: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	err = page.Mouse.MoveLinear(toCenter, 10)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to move to target: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	err = page.Mouse.Up(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to mouse up: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully dragged element",
		Timestamp: time.Now(),
	}, nil
}

// ClosePage 关闭当前页面
func (e *Executor) ClosePage(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	err := page.Close()
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to close page: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	return &OperationResult{
		Success:   true,
		Message:   "Successfully closed page",
		Timestamp: time.Now(),
	}, nil
}

// GetNetworkRequests 获取网络请求（需要先启用网络监控）
func (e *Executor) GetNetworkRequests(ctx context.Context) (*OperationResult, error) {
	page := e.Browser.GetActivePage()
	if page == nil {
		return nil, fmt.Errorf("no active page")
	}

	// 启用网络监控
	err := proto.NetworkEnable{}.Call(page)
	if err != nil {
		return &OperationResult{
			Success:   false,
			Error:     fmt.Sprintf("Failed to enable network monitoring: %s", err.Error()),
			Timestamp: time.Now(),
		}, err
	}

	requests := []map[string]interface{}{}

	// 监听网络请求
	go page.EachEvent(func(e *proto.NetworkRequestWillBeSent) {
		req := map[string]interface{}{
			"url":       e.Request.URL,
			"method":    e.Request.Method,
			"timestamp": time.Now().Format(time.RFC3339),
			"type":      e.Type,
		}
		requests = append(requests, req)
	})()

	// 等待一段时间收集请求
	time.Sleep(100 * time.Millisecond)

	return &OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Retrieved %d network requests", len(requests)),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"requests": requests,
		},
	}, nil
}

// safeWaitForPageLoad 安全地等待页面加载,捕获可能的 panic
func safeWaitForPageLoad(ctx context.Context, page *rod.Page, waitUntil string) (err error) {
	// 使用 defer recover 来捕获 rod 库可能产生的 panic
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during page wait: %v", r)
		}
	}()

	// 根据不同的等待条件执行相应的等待操作
	switch waitUntil {
	case "domcontentloaded", "load":
		err = page.WaitLoad()
	case "networkidle", "idle":
		page.WaitIdle(2 * time.Second)
	default:
		err = page.WaitLoad()
	}

	return err
}

// safeScrollEval 安全地执行滚动操作,捕获可能的 panic
func safeScrollEval(ctx context.Context, page *rod.Page) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during scroll eval: %v", r)
		}
	}()

	_, err = page.Eval(`() => {
		window.scrollTo(0, document.body.scrollHeight);
	}`)
	return err
}

// safeEvaluate 安全地执行 JavaScript,捕获可能的 panic
func safeEvaluate(ctx context.Context, page *rod.Page, script string, result interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during evaluate: %v", r)
		}
	}()

	evalResult, evalErr := page.Eval(script)
	if evalErr != nil {
		return evalErr
	}

	// 尝试解析结果
	if evalResult != nil {
		*result.(*interface{}) = evalResult.Value.Val()
	}

	return nil
}
