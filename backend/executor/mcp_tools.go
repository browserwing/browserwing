package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/browserwing/browserwing/pkg/logger"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPToolRegistry MCP 工具注册表
type MCPToolRegistry struct {
	executor  *Executor
	mcpServer *server.MCPServer
}

// NewMCPToolRegistry 创建 MCP 工具注册表
func NewMCPToolRegistry(executor *Executor, mcpServer *server.MCPServer) *MCPToolRegistry {
	return &MCPToolRegistry{
		executor:  executor,
		mcpServer: mcpServer,
	}
}

// RegisterAllTools 注册所有工具到 MCP 服务器
func (r *MCPToolRegistry) RegisterAllTools() error {
	// 注册导航工具
	if err := r.registerNavigateTool(); err != nil {
		return fmt.Errorf("failed to register navigate tool: %w", err)
	}

	// 注册点击工具
	if err := r.registerClickTool(); err != nil {
		return fmt.Errorf("failed to register click tool: %w", err)
	}

	// 注册输入工具
	if err := r.registerTypeTool(); err != nil {
		return fmt.Errorf("failed to register type tool: %w", err)
	}

	// 注册选择工具
	if err := r.registerSelectTool(); err != nil {
		return fmt.Errorf("failed to register select tool: %w", err)
	}

	// 注册提取工具
	if err := r.registerExtractTool(); err != nil {
		return fmt.Errorf("failed to register extract tool: %w", err)
	}

	// 注册语义树工具
	if err := r.registerGetSemanticTreeTool(); err != nil {
		return fmt.Errorf("failed to register semantic tree tool: %w", err)
	}

	// 注册页面信息工具
	if err := r.registerGetPageInfoTool(); err != nil {
		return fmt.Errorf("failed to register page info tool: %w", err)
	}

	// 注册等待工具
	if err := r.registerWaitForTool(); err != nil {
		return fmt.Errorf("failed to register wait tool: %w", err)
	}

	// 注册滚动工具
	if err := r.registerScrollTool(); err != nil {
		return fmt.Errorf("failed to register scroll tool: %w", err)
	}

	// 注册截图工具
	if err := r.registerScreenshotTool(); err != nil {
		return fmt.Errorf("failed to register screenshot tool: %w", err)
	}

	// 注册执行脚本工具
	if err := r.registerEvaluateTool(); err != nil {
		return fmt.Errorf("failed to register evaluate tool: %w", err)
	}

	// 注册按键工具
	if err := r.registerPressKeyTool(); err != nil {
		return fmt.Errorf("failed to register press key tool: %w", err)
	}

	// 注册调整窗口工具
	if err := r.registerResizeTool(); err != nil {
		return fmt.Errorf("failed to register resize tool: %w", err)
	}

	// 注册拖拽工具
	if err := r.registerDragTool(); err != nil {
		return fmt.Errorf("failed to register drag tool: %w", err)
	}

	// 注册关闭页面工具
	if err := r.registerClosePageTool(); err != nil {
		return fmt.Errorf("failed to register close page tool: %w", err)
	}

	// 注册文件上传工具
	if err := r.registerFileUploadTool(); err != nil {
		return fmt.Errorf("failed to register file upload tool: %w", err)
	}

	// 注册对话框处理工具
	if err := r.registerHandleDialogTool(); err != nil {
		return fmt.Errorf("failed to register handle dialog tool: %w", err)
	}

	// 注册控制台消息工具
	if err := r.registerGetConsoleMessagesTool(); err != nil {
		return fmt.Errorf("failed to register console messages tool: %w", err)
	}

	// 注册网络请求工具
	if err := r.registerGetNetworkRequestsTool(); err != nil {
		return fmt.Errorf("failed to register network requests tool: %w", err)
	}

	return nil
}

// registerNavigateTool 注册导航工具
func (r *MCPToolRegistry) registerNavigateTool() error {
	tool := mcpgo.NewTool(
		"browser_navigate",
		mcpgo.WithDescription("Navigate to a URL in the browser"),
		mcpgo.WithString("url", mcpgo.Required(), mcpgo.Description("The URL to navigate to")),
		mcpgo.WithString("wait_until", mcpgo.Description("Wait condition: load, domcontentloaded, networkidle (default: load)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		logger.Info(ctx, "[MCP Handler] browser_navigate called")

		args := request.Params.Arguments.(map[string]interface{})
		url, _ := args["url"].(string)
		logger.Info(ctx, "[MCP Handler] URL: %s", url)

		// 检查 context
		select {
		case <-ctx.Done():
			logger.Info(ctx, "[MCP Handler] Context already done: %v", ctx.Err())
			return mcpgo.NewToolResultError(fmt.Sprintf("context error: %v", ctx.Err())), nil
		default:
			logger.Info(ctx, "[MCP Handler] Context is active")
		}

		opts := &NavigateOptions{
			WaitUntil: "load",
			Timeout:   60 * time.Second, // 设置默认超时
		}
		if waitUntil, ok := args["wait_until"].(string); ok && waitUntil != "" {
			opts.WaitUntil = waitUntil
		}
		logger.Info(ctx, "[MCP Handler] Options: WaitUntil=%s, Timeout=%v", opts.WaitUntil, opts.Timeout)

		logger.Info(ctx, "[MCP Handler] Calling executor.Navigate...")
		result, err := r.executor.Navigate(ctx, url, opts)
		if err != nil {
			logger.Info(ctx, "[MCP Handler] Navigate failed: %v", err)
			return mcpgo.NewToolResultError(err.Error()), nil
		}
		logger.Info(ctx, "[MCP Handler] Navigate succeeded")

		// 构建返回文本，包含消息和语义树
		var responseText string
		responseText = result.Message

		// 如果有语义树数据，添加到响应中
		if semanticTree, ok := result.Data["semantic_tree"].(string); ok && semanticTree != "" {
			responseText += "\n\nSemantic Tree:\n" + semanticTree
		}

		return mcpgo.NewToolResultText(responseText), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerClickTool 注册点击工具
func (r *MCPToolRegistry) registerClickTool() error {
	tool := mcpgo.NewTool(
		"browser_click",
		mcpgo.WithDescription("Click an element on the page. Can use CSS selector, XPath, or element label/text."),
		mcpgo.WithString("identifier", mcpgo.Required(), mcpgo.Description("Element identifier: CSS selector, XPath, label, or text")),
		mcpgo.WithBoolean("wait_visible", mcpgo.Description("Wait for element to be visible (default: true)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		identifier, _ := args["identifier"].(string)

		opts := &ClickOptions{
			WaitVisible: true,
			WaitEnabled: true,
			Timeout:     10 * time.Second,
			Button:      "left",
			ClickCount:  1,
		}
		if waitVisible, ok := args["wait_visible"].(bool); ok {
			opts.WaitVisible = waitVisible
		}

		result, err := r.executor.Click(ctx, identifier, opts)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		// 构建返回文本，包含消息和语义树
		var responseText string
		responseText = result.Message

		// 如果有语义树数据，添加到响应中
		if semanticTree, ok := result.Data["semantic_tree"].(string); ok && semanticTree != "" {
			responseText += "\n\nSemantic Tree:\n" + semanticTree
		}

		return mcpgo.NewToolResultText(responseText), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerTypeTool 注册输入工具
func (r *MCPToolRegistry) registerTypeTool() error {
	tool := mcpgo.NewTool(
		"browser_type",
		mcpgo.WithDescription("Type text into an input field. Can use CSS selector, XPath, or element label."),
		mcpgo.WithString("identifier", mcpgo.Required(), mcpgo.Description("Element identifier: CSS selector, XPath, label, or placeholder")),
		mcpgo.WithString("text", mcpgo.Required(), mcpgo.Description("Text to type")),
		mcpgo.WithBoolean("clear", mcpgo.Description("Clear existing text before typing (default: true)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		identifier, _ := args["identifier"].(string)
		text, _ := args["text"].(string)

		opts := &TypeOptions{
			Clear:       true,
			WaitVisible: true,
			Timeout:     10 * time.Second,
			Delay:       0,
		}
		if clear, ok := args["clear"].(bool); ok {
			opts.Clear = clear
		}

		result, err := r.executor.Type(ctx, identifier, text, opts)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerSelectTool 注册选择工具
func (r *MCPToolRegistry) registerSelectTool() error {
	tool := mcpgo.NewTool(
		"browser_select",
		mcpgo.WithDescription("Select an option from a dropdown menu"),
		mcpgo.WithString("identifier", mcpgo.Required(), mcpgo.Description("Select element identifier")),
		mcpgo.WithString("value", mcpgo.Required(), mcpgo.Description("Option value or text to select")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		identifier, _ := args["identifier"].(string)
		value, _ := args["value"].(string)

		opts := &SelectOptions{
			WaitVisible: true,
			Timeout:     10 * time.Second,
		}

		result, err := r.executor.Select(ctx, identifier, value, opts)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerExtractTool 注册提取工具
func (r *MCPToolRegistry) registerExtractTool() error {
	tool := mcpgo.NewTool(
		"browser_extract",
		mcpgo.WithDescription("Extract data from elements on the page"),
		mcpgo.WithString("selector", mcpgo.Required(), mcpgo.Description("CSS selector for elements to extract")),
		mcpgo.WithString("type", mcpgo.Description("Extract type: text, html, attribute, property (default: text)")),
		mcpgo.WithBoolean("multiple", mcpgo.Description("Extract from multiple elements (default: false)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		selector, _ := args["selector"].(string)

		opts := &ExtractOptions{
			Selector: selector,
			Type:     "text",
			Multiple: false,
		}

		if extractType, ok := args["type"].(string); ok && extractType != "" {
			opts.Type = extractType
		}
		if multiple, ok := args["multiple"].(bool); ok {
			opts.Multiple = multiple
		}

		result, err := r.executor.Extract(ctx, opts)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		// 序列化结果为 JSON
		data, _ := json.Marshal(result.Data["result"])
		return mcpgo.NewToolResultText(string(data)), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerGetSemanticTreeTool 注册语义树工具
func (r *MCPToolRegistry) registerGetSemanticTreeTool() error {
	tool := mcpgo.NewTool(
		"browser_get_semantic_tree",
		mcpgo.WithDescription("Get the semantic tree of interactive elements on the page"),
		mcpgo.WithBoolean("simple", mcpgo.Description("Return simplified text format suitable for LLMs (default: true)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})

		simple := true
		if simpleArg, ok := args["simple"].(bool); ok {
			simple = simpleArg
		}

		tree, err := r.executor.GetSemanticTree(ctx)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		if simple {
			// 返回简化的文本格式
			text := tree.SerializeToSimpleText()
			return mcpgo.NewToolResultText(text), nil
		}

		// 返回完整的 JSON 格式
		data, _ := json.Marshal(tree)
		return mcpgo.NewToolResultText(string(data)), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerGetPageInfoTool 注册页面信息工具
func (r *MCPToolRegistry) registerGetPageInfoTool() error {
	tool := mcpgo.NewTool(
		"browser_get_page_info",
		mcpgo.WithDescription("Get information about the current page (URL, title, etc.)"),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		result, err := r.executor.GetPageInfo(ctx)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		data, _ := json.Marshal(result.Data)
		return mcpgo.NewToolResultText(string(data)), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerWaitForTool 注册等待工具
func (r *MCPToolRegistry) registerWaitForTool() error {
	tool := mcpgo.NewTool(
		"browser_wait_for",
		mcpgo.WithDescription("Wait for an element to appear or change state"),
		mcpgo.WithString("identifier", mcpgo.Required(), mcpgo.Description("Element identifier")),
		mcpgo.WithString("state", mcpgo.Description("Wait state: visible, hidden, enabled (default: visible)")),
		mcpgo.WithNumber("timeout", mcpgo.Description("Timeout in seconds (default: 30)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		identifier, _ := args["identifier"].(string)

		opts := &WaitForOptions{
			State:   "visible",
			Timeout: 30 * time.Second,
		}

		if state, ok := args["state"].(string); ok && state != "" {
			opts.State = state
		}

		if timeout, ok := args["timeout"].(float64); ok && timeout > 0 {
			opts.Timeout = time.Duration(timeout) * time.Second
		}

		result, err := r.executor.WaitFor(ctx, identifier, opts)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerScrollTool 注册滚动工具
func (r *MCPToolRegistry) registerScrollTool() error {
	tool := mcpgo.NewTool(
		"browser_scroll",
		mcpgo.WithDescription("Scroll the page or to an element"),
		mcpgo.WithString("direction", mcpgo.Description("Scroll direction: bottom, top, or element identifier")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		direction, _ := args["direction"].(string)

		var result *OperationResult
		var err error

		switch direction {
		case "bottom":
			result, err = r.executor.ScrollToBottom(ctx)
		case "top":
			// Scroll to top
			page := r.executor.GetRodPage()
			if page != nil {
				_, err = page.Eval(`() => window.scrollTo(0, 0)`)
				if err == nil {
					result = &OperationResult{
						Success: true,
						Message: "Scrolled to top",
					}
				}
			}
		default:
			// 滚动到元素
			page := r.executor.GetRodPage()
			if page != nil {
				elem, findErr := r.executor.findElement(ctx, page, direction)
				if findErr != nil {
					return mcpgo.NewToolResultError(findErr.Error()), findErr
				}
				err = ScrollToElement(ctx, elem)
				if err == nil {
					result = &OperationResult{
						Success: true,
						Message: fmt.Sprintf("Scrolled to element: %s", direction),
					}
				}
			}
		}

		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerScreenshotTool 注册截图工具
func (r *MCPToolRegistry) registerScreenshotTool() error {
	tool := mcpgo.NewTool(
		"browser_take_screenshot",
		mcpgo.WithDescription("Take a screenshot of the current page"),
		mcpgo.WithBoolean("full_page", mcpgo.Description("Capture full page (default: false)")),
		mcpgo.WithString("format", mcpgo.Description("Image format: png or jpeg (default: png)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})

		opts := &ScreenshotOptions{
			FullPage: false,
			Quality:  80,
			Format:   "png",
		}

		if fullPage, ok := args["full_page"].(bool); ok {
			opts.FullPage = fullPage
		}
		if format, ok := args["format"].(string); ok && format != "" {
			opts.Format = format
		}

		result, err := r.executor.Screenshot(ctx, opts)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerEvaluateTool 注册执行脚本工具
func (r *MCPToolRegistry) registerEvaluateTool() error {
	tool := mcpgo.NewTool(
		"browser_evaluate",
		mcpgo.WithDescription("Execute JavaScript code in the browser context"),
		mcpgo.WithString("script", mcpgo.Required(), mcpgo.Description("JavaScript code to execute")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		script, _ := args["script"].(string)

		result, err := r.executor.Evaluate(ctx, script)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		// 返回执行结果
		if resultData, ok := result.Data["result"]; ok {
			return mcpgo.NewToolResultText(fmt.Sprintf("%s\nResult: %v", result.Message, resultData)), nil
		}
		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerPressKeyTool 注册按键工具
func (r *MCPToolRegistry) registerPressKeyTool() error {
	tool := mcpgo.NewTool(
		"browser_press_key",
		mcpgo.WithDescription("Press a keyboard key"),
		mcpgo.WithString("key", mcpgo.Required(), mcpgo.Description("Key to press (e.g., Enter, Tab, ArrowUp, etc.)")),
		mcpgo.WithBoolean("ctrl", mcpgo.Description("Hold Ctrl key")),
		mcpgo.WithBoolean("shift", mcpgo.Description("Hold Shift key")),
		mcpgo.WithBoolean("alt", mcpgo.Description("Hold Alt key")),
		mcpgo.WithBoolean("meta", mcpgo.Description("Hold Meta key (Command/Windows)")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		key, _ := args["key"].(string)

		opts := &PressKeyOptions{}
		if ctrl, ok := args["ctrl"].(bool); ok {
			opts.Ctrl = ctrl
		}
		if shift, ok := args["shift"].(bool); ok {
			opts.Shift = shift
		}
		if alt, ok := args["alt"].(bool); ok {
			opts.Alt = alt
		}
		if meta, ok := args["meta"].(bool); ok {
			opts.Meta = meta
		}

		result, err := r.executor.PressKey(ctx, key, opts)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerResizeTool 注册调整窗口工具
func (r *MCPToolRegistry) registerResizeTool() error {
	tool := mcpgo.NewTool(
		"browser_resize",
		mcpgo.WithDescription("Resize the browser window"),
		mcpgo.WithNumber("width", mcpgo.Required(), mcpgo.Description("Window width in pixels")),
		mcpgo.WithNumber("height", mcpgo.Required(), mcpgo.Description("Window height in pixels")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})

		width := 0
		height := 0

		if w, ok := args["width"].(float64); ok {
			width = int(w)
		}
		if h, ok := args["height"].(float64); ok {
			height = int(h)
		}

		if width <= 0 || height <= 0 {
			return mcpgo.NewToolResultError("Invalid width or height"), nil
		}

		result, err := r.executor.Resize(ctx, width, height)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerDragTool 注册拖拽工具
func (r *MCPToolRegistry) registerDragTool() error {
	tool := mcpgo.NewTool(
		"browser_drag",
		mcpgo.WithDescription("Drag an element to another element"),
		mcpgo.WithString("from_identifier", mcpgo.Required(), mcpgo.Description("Source element identifier")),
		mcpgo.WithString("to_identifier", mcpgo.Required(), mcpgo.Description("Target element identifier")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		fromIdentifier, _ := args["from_identifier"].(string)
		toIdentifier, _ := args["to_identifier"].(string)

		result, err := r.executor.Drag(ctx, fromIdentifier, toIdentifier)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerClosePageTool 注册关闭页面工具
func (r *MCPToolRegistry) registerClosePageTool() error {
	tool := mcpgo.NewTool(
		"browser_close",
		mcpgo.WithDescription("Close the current browser page/tab"),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		result, err := r.executor.ClosePage(ctx)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerFileUploadTool 注册文件上传工具
func (r *MCPToolRegistry) registerFileUploadTool() error {
	tool := mcpgo.NewTool(
		"browser_file_upload",
		mcpgo.WithDescription("Upload files to a file input element"),
		mcpgo.WithString("identifier", mcpgo.Required(), mcpgo.Description("File input element identifier")),
		mcpgo.WithArray("file_paths", mcpgo.Required(), mcpgo.Description("Array of file paths to upload")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		identifier, _ := args["identifier"].(string)

		var filePaths []string
		if paths, ok := args["file_paths"].([]interface{}); ok {
			for _, p := range paths {
				if path, ok := p.(string); ok {
					filePaths = append(filePaths, path)
				}
			}
		}

		if len(filePaths) == 0 {
			return mcpgo.NewToolResultError("No file paths provided"), nil
		}

		result, err := r.executor.FileUpload(ctx, identifier, filePaths)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerHandleDialogTool 注册对话框处理工具
func (r *MCPToolRegistry) registerHandleDialogTool() error {
	tool := mcpgo.NewTool(
		"browser_handle_dialog",
		mcpgo.WithDescription("Configure how to handle JavaScript dialogs (alert, confirm, prompt)"),
		mcpgo.WithBoolean("accept", mcpgo.Required(), mcpgo.Description("Whether to accept the dialog")),
		mcpgo.WithString("text", mcpgo.Description("Text to enter for prompt dialogs")),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := request.Params.Arguments.(map[string]interface{})
		accept := false
		if a, ok := args["accept"].(bool); ok {
			accept = a
		}

		text := ""
		if t, ok := args["text"].(string); ok {
			text = t
		}

		result, err := r.executor.HandleDialog(ctx, accept, text)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerGetConsoleMessagesTool 注册控制台消息工具
func (r *MCPToolRegistry) registerGetConsoleMessagesTool() error {
	tool := mcpgo.NewTool(
		"browser_console_messages",
		mcpgo.WithDescription("Get console messages from the browser"),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		result, err := r.executor.GetConsoleMessages(ctx)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// registerGetNetworkRequestsTool 注册网络请求工具
func (r *MCPToolRegistry) registerGetNetworkRequestsTool() error {
	tool := mcpgo.NewTool(
		"browser_network_requests",
		mcpgo.WithDescription("Get network requests made by the page"),
	)

	handler := func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		result, err := r.executor.GetNetworkRequests(ctx)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}

		return mcpgo.NewToolResultText(result.Message), nil
	}

	r.mcpServer.AddTool(tool, handler)
	return nil
}

// GetToolMetadata 获取所有工具的元数据（用于文档生成）
func (r *MCPToolRegistry) GetToolMetadata() []ToolMetadata {
	return GetExecutorToolsMetadata()
}

// GetExecutorToolsMetadata 获取 Executor 工具元数据列表（包级别函数，方便外部调用）
func GetExecutorToolsMetadata() []ToolMetadata {
	return []ToolMetadata{
		{
			Name:        "browser_navigate",
			Description: "Navigate to a URL in the browser",
			Category:    "Navigation",
			Parameters: []ToolParameter{
				{Name: "url", Type: "string", Required: true, Description: "The URL to navigate to"},
				{Name: "wait_until", Type: "string", Required: false, Description: "Wait condition: load, domcontentloaded, networkidle"},
			},
		},
		{
			Name:        "browser_click",
			Description: "Click an element on the page",
			Category:    "Interaction",
			Parameters: []ToolParameter{
				{Name: "identifier", Type: "string", Required: true, Description: "Element identifier"},
				{Name: "wait_visible", Type: "boolean", Required: false, Description: "Wait for element to be visible"},
			},
		},
		{
			Name:        "browser_type",
			Description: "Type text into an input field",
			Category:    "Interaction",
			Parameters: []ToolParameter{
				{Name: "identifier", Type: "string", Required: true, Description: "Element identifier"},
				{Name: "text", Type: "string", Required: true, Description: "Text to type"},
				{Name: "clear", Type: "boolean", Required: false, Description: "Clear existing text"},
			},
		},
		{
			Name:        "browser_select",
			Description: "Select an option from a dropdown",
			Category:    "Interaction",
			Parameters: []ToolParameter{
				{Name: "identifier", Type: "string", Required: true, Description: "Select element identifier"},
				{Name: "value", Type: "string", Required: true, Description: "Option value or text"},
			},
		},
		{
			Name:        "browser_extract",
			Description: "Extract data from elements",
			Category:    "Data",
			Parameters: []ToolParameter{
				{Name: "selector", Type: "string", Required: true, Description: "CSS selector"},
				{Name: "type", Type: "string", Required: false, Description: "Extract type: text, html, attribute"},
				{Name: "multiple", Type: "boolean", Required: false, Description: "Extract multiple elements"},
			},
		},
		{
			Name:        "browser_get_semantic_tree",
			Description: "Get semantic tree of interactive elements",
			Category:    "Analysis",
			Parameters: []ToolParameter{
				{Name: "simple", Type: "boolean", Required: false, Description: "Return simplified format"},
			},
		},
		{
			Name:        "browser_get_page_info",
			Description: "Get page information",
			Category:    "Analysis",
			Parameters:  []ToolParameter{},
		},
		{
			Name:        "browser_wait_for",
			Description: "Wait for element state",
			Category:    "Synchronization",
			Parameters: []ToolParameter{
				{Name: "identifier", Type: "string", Required: true, Description: "Element identifier"},
				{Name: "state", Type: "string", Required: false, Description: "Wait state: visible, hidden, enabled"},
				{Name: "timeout", Type: "number", Required: false, Description: "Timeout in seconds"},
			},
		},
		{
			Name:        "browser_scroll",
			Description: "Scroll the page",
			Category:    "Navigation",
			Parameters: []ToolParameter{
				{Name: "direction", Type: "string", Required: false, Description: "Direction: bottom, top, or element identifier"},
			},
		},
		{
			Name:        "browser_take_screenshot",
			Description: "Take a screenshot of the current page",
			Category:    "Capture",
			Parameters: []ToolParameter{
				{Name: "full_page", Type: "boolean", Required: false, Description: "Capture full page"},
				{Name: "format", Type: "string", Required: false, Description: "Image format: png or jpeg"},
			},
		},
		{
			Name:        "browser_evaluate",
			Description: "Execute JavaScript code in the browser context",
			Category:    "Scripting",
			Parameters: []ToolParameter{
				{Name: "script", Type: "string", Required: true, Description: "JavaScript code to execute"},
			},
		},
		{
			Name:        "browser_press_key",
			Description: "Press a keyboard key",
			Category:    "Interaction",
			Parameters: []ToolParameter{
				{Name: "key", Type: "string", Required: true, Description: "Key to press (Enter, Tab, ArrowUp, etc.)"},
				{Name: "ctrl", Type: "boolean", Required: false, Description: "Hold Ctrl key"},
				{Name: "shift", Type: "boolean", Required: false, Description: "Hold Shift key"},
				{Name: "alt", Type: "boolean", Required: false, Description: "Hold Alt key"},
				{Name: "meta", Type: "boolean", Required: false, Description: "Hold Meta key"},
			},
		},
		{
			Name:        "browser_resize",
			Description: "Resize the browser window",
			Category:    "Window",
			Parameters: []ToolParameter{
				{Name: "width", Type: "number", Required: true, Description: "Window width in pixels"},
				{Name: "height", Type: "number", Required: true, Description: "Window height in pixels"},
			},
		},
		{
			Name:        "browser_drag",
			Description: "Drag an element to another element",
			Category:    "Interaction",
			Parameters: []ToolParameter{
				{Name: "from_identifier", Type: "string", Required: true, Description: "Source element identifier"},
				{Name: "to_identifier", Type: "string", Required: true, Description: "Target element identifier"},
			},
		},
		{
			Name:        "browser_close",
			Description: "Close the current browser page/tab",
			Category:    "Window",
			Parameters:  []ToolParameter{},
		},
		{
			Name:        "browser_file_upload",
			Description: "Upload files to a file input element",
			Category:    "Interaction",
			Parameters: []ToolParameter{
				{Name: "identifier", Type: "string", Required: true, Description: "File input element identifier"},
				{Name: "file_paths", Type: "array", Required: true, Description: "Array of file paths to upload"},
			},
		},
		{
			Name:        "browser_handle_dialog",
			Description: "Configure how to handle JavaScript dialogs",
			Category:    "Dialog",
			Parameters: []ToolParameter{
				{Name: "accept", Type: "boolean", Required: true, Description: "Whether to accept the dialog"},
				{Name: "text", Type: "string", Required: false, Description: "Text for prompt dialogs"},
			},
		},
		{
			Name:        "browser_console_messages",
			Description: "Get console messages from the browser",
			Category:    "Debug",
			Parameters:  []ToolParameter{},
		},
		{
			Name:        "browser_network_requests",
			Description: "Get network requests made by the page",
			Category:    "Debug",
			Parameters:  []ToolParameter{},
		},
	}
}

// ToolMetadata 工具元数据
type ToolMetadata struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Parameters  []ToolParameter `json:"parameters"`
}

// ToolParameter 工具参数
type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}
