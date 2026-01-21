package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// ExtractSemanticTree 从页面提取语义树（基于 Accessibility Tree）
func ExtractSemanticTree(ctx context.Context, page *rod.Page) (*SemanticTree, error) {
	logger.Info(ctx, "[ExtractSemanticTree] Starting extraction")

	// 检查 context 是否已经取消
	select {
	case <-ctx.Done():
		logger.Info(ctx, "[ExtractSemanticTree] Context already done: %v", ctx.Err())
		return nil, ctx.Err()
	default:
		logger.Info(ctx, "[ExtractSemanticTree] Context is active")
	}

	// 先禁用再启用，确保状态干净
	logger.Info(ctx, "[ExtractSemanticTree] Disabling accessibility...")
	_ = proto.AccessibilityDisable{}.Call(page)

	// 启用 Accessibility 域
	logger.Info(ctx, "[ExtractSemanticTree] Enabling accessibility...")
	err := proto.AccessibilityEnable{}.Call(page)
	if err != nil {
		logger.Info(ctx, "[ExtractSemanticTree] Failed to enable accessibility: %v", err)
		return nil, fmt.Errorf("failed to enable accessibility: %w", err)
	}
	logger.Info(ctx, "[ExtractSemanticTree] Accessibility enabled")

	// 确保函数结束时禁用
	defer func() {
		logger.Info(ctx, "[ExtractSemanticTree] Cleaning up - disabling accessibility")
		_ = proto.AccessibilityDisable{}.Call(page)
	}()

	// 检查 context
	select {
	case <-ctx.Done():
		logger.Info(ctx, "[ExtractSemanticTree] Context done before getting tree: %v", ctx.Err())
		return nil, ctx.Err()
	default:
	}

	// 获取 Accessibility Tree，不限制深度（让它获取完整树）
	// 但我们会在后续处理时过滤
	logger.Info(ctx, "[ExtractSemanticTree] Getting full AX tree...")
	axTree, err := proto.AccessibilityGetFullAXTree{}.Call(page)
	if err != nil {
		logger.Info(ctx, "[ExtractSemanticTree] Failed to get AX tree: %v", err)
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}
	logger.Info(ctx, "[ExtractSemanticTree] Got AX tree with %d nodes", len(axTree.Nodes))

	if len(axTree.Nodes) == 0 {
		logger.Info(ctx, "[ExtractSemanticTree] AX tree is empty")
		return nil, fmt.Errorf("accessibility tree is empty")
	}

	// 构建语义树
	logger.Info(ctx, "[ExtractSemanticTree] Building semantic tree...")
	tree := &SemanticTree{
		Elements:     make(map[string]*SemanticNode),
		AXNodeMap:    make(map[proto.AccessibilityAXNodeID]*proto.AccessibilityAXNode),
		BackendIDMap: make(map[proto.DOMBackendNodeID]*SemanticNode),
	}

	// 构建 AX Node 映射
	logger.Info(ctx, "[ExtractSemanticTree] Building AX node map...")
	for _, axNode := range axTree.Nodes {
		tree.AXNodeMap[axNode.NodeID] = axNode
	}
	logger.Info(ctx, "[ExtractSemanticTree] AX node map built with %d nodes", len(tree.AXNodeMap))

	// 转换为语义节点
	// 注意：不要过度过滤，保留所有节点，在后续查询时再过滤
	logger.Info(ctx, "[ExtractSemanticTree] Converting to semantic nodes...")
	nodeCount := 0
	for i, axNode := range axTree.Nodes {
		// 检查 context
		select {
		case <-ctx.Done():
			logger.Info(ctx, "[ExtractSemanticTree] Context cancelled during node conversion at node %d/%d", i, len(axTree.Nodes))
			return nil, ctx.Err()
		default:
		}

		semanticNode := buildSemanticNodeFromAXNode(axNode, tree)
		if semanticNode != nil {
			tree.Elements[semanticNode.ID] = semanticNode
			if semanticNode.BackendNodeID > 0 {
				tree.BackendIDMap[semanticNode.BackendNodeID] = semanticNode
			}
			nodeCount++
		}

		// 每100个节点输出一次进度
		if (i+1)%100 == 0 {
			logger.Info(ctx, "[ExtractSemanticTree] Processed %d/%d nodes, kept %d", i+1, len(axTree.Nodes), nodeCount)
		}
	}
	logger.Info(ctx, "[ExtractSemanticTree] Converted %d nodes to %d semantic nodes", len(axTree.Nodes), nodeCount)

	// 检查 cursor: pointer 元素并标记为可点击
	logger.Info(ctx, "[ExtractSemanticTree] Checking cursor:pointer elements...")
	err = markCursorPointerElements(ctx, page, tree)
	if err != nil {
		logger.Warn(ctx, "[ExtractSemanticTree] Failed to mark cursor:pointer elements: %v", err)
		// 不返回错误，继续处理
	}

	// 构建根节点
	if len(axTree.Nodes) > 0 {
		tree.Root = tree.Elements[string(axTree.Nodes[0].NodeID)]
	}

	logger.Info(ctx, "[ExtractSemanticTree] Semantic tree extraction completed successfully")
	return tree, nil
}

// buildSemanticNodeFromAXNode 从 Accessibility Node 构建语义节点
func buildSemanticNodeFromAXNode(axNode *proto.AccessibilityAXNode, tree *SemanticTree) *SemanticNode {
	// 获取 Role
	var role string
	if axNode.Role != nil {
		role = getAXValueString(axNode.Role)
	}

	// 创建节点（不在这里过滤，保留所有节点）
	node := &SemanticNode{
		ID:         string(axNode.NodeID),
		AXNodeID:   axNode.NodeID,
		Role:       role,
		Type:       role, // 保持兼容性
		Attributes: make(map[string]string),
		Metadata:   make(map[string]interface{}),
		Children:   make([]*SemanticNode, 0),
	}

	// 记录是否被忽略
	if axNode.Ignored {
		node.Metadata["ignored"] = true
	}

	// 设置 BackendNodeID
	if axNode.BackendDOMNodeID > 0 {
		node.BackendNodeID = axNode.BackendDOMNodeID
	}

	// 获取名称（通常是元素的主要标识）
	if axNode.Name != nil {
		nameStr := getAXValueString(axNode.Name)
		node.Label = nameStr
		node.Text = nameStr
	}

	// 获取描述
	if axNode.Description != nil {
		node.Description = getAXValueString(axNode.Description)
	}

	// 获取值
	if axNode.Value != nil {
		node.Value = getAXValueString(axNode.Value)
	}

	// 处理属性
	if axNode.Properties != nil {
		for _, prop := range axNode.Properties {
			key := string(prop.Name)
			value := getAXValueString(prop.Value)
			node.Attributes[key] = value

			// 设置特定属性
			switch key {
			case "placeholder":
				node.Placeholder = value
			case "disabled":
				if value == "true" {
					node.IsEnabled = false
				} else {
					node.IsEnabled = true
				}
			case "focused":
				if value == "true" {
					node.Metadata["focused"] = true
				}
			case "readonly":
				node.Metadata["readonly"] = value == "true"
			case "required":
				node.Metadata["required"] = value == "true"
			}
		}
	}

	// 检查是否可交互
	node.IsInteractive = isInteractiveRole(node.Role)

	// 设置子节点引用（注意：此时子节点可能还没有被创建，所以暂时只记录 ID）
	// 子节点关系会在所有节点创建完成后由外部代码处理
	if axNode.ChildIDs != nil {
		// 记录子节点 ID 到 metadata 中
		node.Metadata["childIDs"] = axNode.ChildIDs
	}

	return node
}

// getAXValueString 获取 AX Value 的字符串表示
func getAXValueString(value *proto.AccessibilityAXValue) string {
	if value == nil {
		return ""
	}

	// gson.JSON 可以通过 String() 方法转换为字符串
	// 但需要去除 JSON 字符串的引号
	str := value.Value.String()

	// 如果是带引号的字符串，去除引号
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	return str
}

// isInteractiveRole 判断角色是否可交互
func isInteractiveRole(role string) bool {
	interactiveRoles := map[string]bool{
		"button":           true,
		"link":             true,
		"textbox":          true,
		"searchbox":        true,
		"combobox":         true,
		"checkbox":         true,
		"radio":            true,
		"slider":           true,
		"spinbutton":       true,
		"switch":           true,
		"tab":              true,
		"menuitem":         true,
		"menuitemcheckbox": true,
		"menuitemradio":    true,
		"option":           true,
		"treeitem":         true,
		"gridcell":         true,
	}

	return interactiveRoles[role]
}

// FindElementByLabel 通过标签查找元素
func (tree *SemanticTree) FindElementByLabel(label string) *SemanticNode {
	label = strings.ToLower(strings.TrimSpace(label))

	for _, node := range tree.Elements {
		if strings.Contains(strings.ToLower(node.Label), label) {
			return node
		}
		if strings.Contains(strings.ToLower(node.Text), label) {
			return node
		}
		if strings.Contains(strings.ToLower(node.Placeholder), label) {
			return node
		}
	}

	return nil
}

// FindElementByType 通过类型查找元素
func (tree *SemanticTree) FindElementsByType(elemType string) []*SemanticNode {
	result := make([]*SemanticNode, 0)

	for _, node := range tree.Elements {
		if node.Type == elemType {
			result = append(result, node)
		}
	}

	return result
}

// FindElementByID 通过 ID 查找元素
func (tree *SemanticTree) FindElementByID(id string) *SemanticNode {
	return tree.Elements[id]
}

// GetVisibleElements 获取所有可见元素
func (tree *SemanticTree) GetVisibleElements() []*SemanticNode {
	result := make([]*SemanticNode, 0)

	for _, node := range tree.Elements {
		if node.IsVisible {
			result = append(result, node)
		}
	}

	return result
}

// markCursorPointerElements 标记所有 cursor:pointer 的元素为可点击
func markCursorPointerElements(ctx context.Context, page *rod.Page, tree *SemanticTree) error {
	// 执行 JavaScript 获取所有 cursor:pointer 元素的信息
	script := `
	() => {
		const elements = [];
		const allElements = document.querySelectorAll('*');
		
		for (const elem of allElements) {
			const style = window.getComputedStyle(elem);
			if (style.cursor === 'pointer') {
				// 获取元素的文本内容（截断到合理长度）
				let text = elem.textContent || '';
				text = text.trim().substring(0, 100);
				
				// 获取元素的标识信息
				const id = elem.id || '';
				const className = elem.className || '';
				const tagName = elem.tagName.toLowerCase();
				
				elements.push({
					text: text,
					id: id,
					className: typeof className === 'string' ? className : '',
					tagName: tagName
				});
			}
		}
		
		return elements;
	}
	`

	// 使用安全的 Eval 调用,防止 rod 库 panic
	var cursorPointerElements []map[string]interface{}
	err := safePageEvalUnmarshal(ctx, page, script, &cursorPointerElements)
	if err != nil {
		return fmt.Errorf("failed to execute cursor pointer detection script: %w", err)
	}

	logger.Info(ctx, "[markCursorPointerElements] Found %d elements with cursor:pointer", len(cursorPointerElements))

	// 标记树中的节点
	markedCount := 0
	for _, elem := range cursorPointerElements {
		text, _ := elem["text"].(string)
		id, _ := elem["id"].(string)
		className, _ := elem["className"].(string)
		tagName, _ := elem["tagName"].(string)

		// 尝试在语义树中找到匹配的节点
		for _, node := range tree.Elements {
			// 跳过已经被标记为可点击的节点
			if clickable, ok := node.Metadata["cursor_pointer"].(bool); ok && clickable {
				continue
			}

			// 匹配逻辑：基于文本、ID、className
			matched := false
			if id != "" && node.Attributes["id"] == id {
				matched = true
			} else if text != "" && (strings.Contains(node.Text, text) || strings.Contains(text, node.Text)) {
				// 文本匹配（允许部分匹配）
				if len(text) > 5 && len(node.Text) > 5 { // 避免太短的文本误匹配
					matched = true
				}
			} else if node.Label != "" && text != "" && strings.Contains(text, node.Label) {
				matched = true
			}

			if matched {
				// 标记为 cursor:pointer 元素
				node.Metadata["cursor_pointer"] = true
				node.Metadata["cursor_pointer_tag"] = tagName
				if className != "" {
					node.Metadata["cursor_pointer_class"] = className
				}
				markedCount++
			}
		}
	}

	logger.Info(ctx, "[markCursorPointerElements] Marked %d nodes as cursor:pointer", markedCount)
	return nil
}

// GetClickableElements 获取所有可点击元素（基于 Accessibility Role 和 cursor:pointer）
func (tree *SemanticTree) GetClickableElements() []*SemanticNode {
	result := make([]*SemanticNode, 0)

	clickableRoles := map[string]bool{
		"button":           true,
		"link":             true,
		"menuitem":         true,
		"menuitemcheckbox": true,
		"menuitemradio":    true,
		"tab":              true,
		"checkbox":         true,
		"radio":            true,
		"switch":           true,
		"treeitem":         true,
	}

	for _, node := range tree.Elements {
		// 跳过被忽略的节点
		if ignored, ok := node.Metadata["ignored"].(bool); ok && ignored {
			continue
		}

		// 跳过没有 BackendNodeID 的节点（无法操作）
		if node.BackendNodeID == 0 {
			continue
		}

		isClickable := false

		// 1. 基于 Accessibility Role 判断
		if clickableRoles[node.Role] {
			// 至少要有名称或文本
			if node.Label != "" || node.Text != "" || node.Description != "" {
				isClickable = true
			}
		}

		// 2. 检查是否有 cursor:pointer 标记
		if cursorPointer, ok := node.Metadata["cursor_pointer"].(bool); ok && cursorPointer {
			// cursor:pointer 元素也应该有一定的标识（避免添加空元素）
			if node.Label != "" || node.Text != "" || node.Description != "" || node.Attributes["id"] != "" {
				isClickable = true
			}
		}

		if isClickable {
			result = append(result, node)
		}
	}

	return result
}

// GetInputElements 获取所有输入元素（基于 Accessibility Role）
func (tree *SemanticTree) GetInputElements() []*SemanticNode {
	result := make([]*SemanticNode, 0)

	inputRoles := map[string]bool{
		"textbox":    true,
		"searchbox":  true,
		"combobox":   true,
		"spinbutton": true,
		"slider":     true,
	}

	for _, node := range tree.Elements {
		// 跳过被忽略的节点
		if ignored, ok := node.Metadata["ignored"].(bool); ok && ignored {
			continue
		}

		// 跳过没有 BackendNodeID 的节点（无法操作）
		if node.BackendNodeID == 0 {
			continue
		}

		// 基于 Accessibility Role 判断
		if inputRoles[node.Role] {
			result = append(result, node)
		}
	}

	return result
}

// SerializeToSimpleText 将语义树序列化为简单文本（用于 LLM）
func (tree *SemanticTree) SerializeToSimpleText() string {
	var builder strings.Builder
	builder.WriteString("Page Interactive Elements:\n")
	builder.WriteString("(Use the exact identifier like 'Clickable Element [1]' or 'Input Element [1]' to interact with elements)\n\n")

	// 按类型分组
	clickable := tree.GetClickableElements()
	inputs := tree.GetInputElements()

	if len(clickable) > 0 {
		builder.WriteString("Clickable Elements (use identifier like 'Clickable Element [N]'):\n")
		for i, node := range clickable {
			// 生成更明确的标识
			label := node.Label
			if label == "" {
				label = node.Text
			}
			if label == "" {
				label = node.Description
			}
			if label == "" {
				label = fmt.Sprintf("<%s>", node.Role)
			}

			// 格式：[索引] 标签 - 角色 - 描述
			builder.WriteString(fmt.Sprintf("  [%d] %s", i+1, label))
			if node.Role != "" {
				builder.WriteString(fmt.Sprintf(" (role: %s)", node.Role))
			}
			if node.Description != "" && node.Description != label {
				builder.WriteString(fmt.Sprintf(" - %s", node.Description))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	if len(inputs) > 0 {
		builder.WriteString("Input Elements (use identifier like 'Input Element [N]'):\n")
		for i, node := range inputs {
			// 生成更明确的标识
			label := node.Label
			if label == "" {
				label = node.Placeholder
			}
			if label == "" {
				label = node.Description
			}
			if label == "" {
				label = fmt.Sprintf("<%s>", node.Role)
			}

			// 格式：[索引] 标签 - 角色 - placeholder - value
			builder.WriteString(fmt.Sprintf("  [%d] %s", i+1, label))
			if node.Role != "" {
				builder.WriteString(fmt.Sprintf(" (role: %s)", node.Role))
			}
			if node.Placeholder != "" && node.Placeholder != label {
				builder.WriteString(fmt.Sprintf(" [placeholder: %s]", node.Placeholder))
			}
			if node.Value != "" {
				builder.WriteString(fmt.Sprintf(" [value: %s]", node.Value))
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// HighlightElement 在页面上高亮显示元素（用于调试）
func HighlightElement(ctx context.Context, page *rod.Page, selector string) error {
	elem, err := page.Element(selector)
	if err != nil {
		return err
	}

	// 添加高亮样式
	_, err = elem.Eval(`function() {
		this.style.outline = '3px solid red';
		this.style.outlineOffset = '2px';
		setTimeout(() => {
			this.style.outline = '';
			this.style.outlineOffset = '';
		}, 2000);
	}`)

	return err
}

// WaitForElement 等待元素出现
func WaitForElement(ctx context.Context, page *rod.Page, selector string, opts *WaitForOptions) error {
	if opts == nil {
		opts = &WaitForOptions{}
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * 1000000000 // 30秒
	}

	// 等待元素
	elem, err := page.Timeout(timeout).Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %s", selector)
	}

	// 根据状态等待
	switch opts.State {
	case "visible":
		return elem.WaitVisible()
	case "hidden":
		return elem.WaitInvisible()
	default:
		// 默认只等待元素存在
		return nil
	}
}

// EvaluateAccessibility 评估页面可访问性
func EvaluateAccessibility(ctx context.Context, page *rod.Page) (*AccessibilityReport, error) {
	report := &AccessibilityReport{
		Issues: make([]AccessibilityIssue, 0),
	}

	// 检查没有 alt 属性的图片
	images, _ := page.Elements("img:not([alt])")
	for _, img := range images {
		src, _ := img.Attribute("src")
		report.Issues = append(report.Issues, AccessibilityIssue{
			Type:     "missing-alt",
			Severity: "warning",
			Message:  "Image missing alt attribute",
			Element:  fmt.Sprintf("<img src='%s'>", *src),
		})
	}

	// 检查没有 label 的输入框
	inputs, _ := page.Elements("input:not([type='hidden']):not([aria-label]):not([id])")
	for range inputs {
		report.Issues = append(report.Issues, AccessibilityIssue{
			Type:     "missing-label",
			Severity: "error",
			Message:  "Input field missing label or aria-label",
		})
	}

	report.TotalIssues = len(report.Issues)
	return report, nil
}

// AccessibilityReport 可访问性报告
type AccessibilityReport struct {
	TotalIssues int                  `json:"total_issues"`
	Issues      []AccessibilityIssue `json:"issues"`
}

// AccessibilityIssue 可访问性问题
type AccessibilityIssue struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Element  string `json:"element,omitempty"`
}

// GetElementFromPage 从页面获取 Rod Element（基于 BackendNodeID）
func GetElementFromPage(ctx context.Context, page *rod.Page, node *SemanticNode) (*rod.Element, error) {
	if node.BackendNodeID == 0 {
		return nil, fmt.Errorf("node has no backend node ID")
	}

	// 使用 DOM.resolveNode 将 BackendNodeID 转换为 ObjectID
	obj, err := proto.DOMResolveNode{
		BackendNodeID: node.BackendNodeID,
	}.Call(page)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve backend node: %w", err)
	}

	if obj.Object.ObjectID == "" {
		return nil, fmt.Errorf("resolved object has no object ID")
	}

	// 创建 Rod Element
	elem, err := page.ElementFromObject(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to create element from object: %w", err)
	}

	return elem, nil
}

// GetElementByAXNodeID 通过 AX Node ID 获取 Rod Element
func GetElementByAXNodeID(ctx context.Context, page *rod.Page, tree *SemanticTree, axNodeID proto.AccessibilityAXNodeID) (*rod.Element, error) {
	// 从树中查找节点
	node, ok := tree.Elements[string(axNodeID)]
	if !ok {
		return nil, fmt.Errorf("AX node not found: %s", axNodeID)
	}

	return GetElementFromPage(ctx, page, node)
}

// InjectAccessibilityHelpers 注入辅助脚本
func InjectAccessibilityHelpers(ctx context.Context, page *rod.Page) error {
	// 注入辅助函数到页面
	_, err := page.Eval(`() => {
		// 添加用于元素高亮的辅助函数
		window.__highlightElement = function(selector) {
			const elem = document.querySelector(selector);
			if (elem) {
				elem.style.outline = '3px solid blue';
				elem.style.outlineOffset = '2px';
				setTimeout(() => {
					elem.style.outline = '';
					elem.style.outlineOffset = '';
				}, 1000);
			}
		};
		
		// 添加获取元素语义信息的辅助函数
		window.__getElementInfo = function(elem) {
			return {
				tag: elem.tagName.toLowerCase(),
				text: elem.innerText || elem.textContent,
				value: elem.value,
				visible: elem.offsetParent !== null,
				rect: elem.getBoundingClientRect()
			};
		};
	}`)

	return err
}

// ScrollToElement 滚动到元素位置
func ScrollToElement(ctx context.Context, elem *rod.Element) error {
	return elem.ScrollIntoView()
}

// GetElementScreenshot 获取元素截图
func GetElementScreenshot(ctx context.Context, elem *rod.Element) ([]byte, error) {
	return elem.Screenshot(proto.PageCaptureScreenshotFormatPng, 100)
}

// safePageEvalUnmarshal 安全地执行页面 JavaScript 并解析结果,捕获可能的 panic
func safePageEvalUnmarshal(ctx context.Context, page *rod.Page, script string, result interface{}) (err error) {
	// 使用 defer recover 来捕获 rod 库可能产生的 panic
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during page eval: %v", r)
		}
	}()

	// 执行 JavaScript
	evalResult, evalErr := page.Eval(script)
	if evalErr != nil {
		return evalErr
	}

	// 解析结果
	if evalResult != nil {
		if unmarshalErr := evalResult.Value.Unmarshal(result); unmarshalErr != nil {
			return fmt.Errorf("failed to unmarshal result: %w", unmarshalErr)
		}
	}

	return nil
}
