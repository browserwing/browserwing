package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/browserwing/browserwing/executor"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/services/browser"
	"github.com/browserwing/browserwing/storage"
)

// MCPServer 使用 mcp-go 库实现的 MCP 服务器
type MCPServer struct {
	storage       *storage.BoltDB
	browserMgr    *browser.Manager
	scripts       map[string]*models.Script // scriptID -> Script
	scriptsByName map[string]*models.Script // commandName -> Script
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc

	// mcp-go server instance
	mcpServer            *server.MCPServer
	streamableHTTPServer *server.StreamableHTTPServer
	sseServer            *server.SSEServer

	// Executor 实例，提供通用浏览器自动化能力
	executor     *executor.Executor
	toolRegistry *executor.MCPToolRegistry
}

// NewMCPServer 创建使用 mcp-go 的 MCP 服务器
func NewMCPServer(storage *storage.BoltDB, browserMgr *browser.Manager) *MCPServer {
	ctx, cancel := context.WithCancel(context.Background())

	s := &MCPServer{
		storage:       storage,
		browserMgr:    browserMgr,
		scripts:       make(map[string]*models.Script),
		scriptsByName: make(map[string]*models.Script),
		ctx:           ctx,
		cancel:        cancel,
	}

	// 创建 mcp-go server
	s.mcpServer = server.NewMCPServer(
		"browserwing",
		"0.0.1",
		server.WithToolCapabilities(true),
	)

	// 创建 Streamable HTTP server
	s.streamableHTTPServer = server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath("/api/v1/mcp/message"),
		server.WithStateful(true),
	)

	// 创建 SSE server
	s.sseServer = server.NewSSEServer(
		s.mcpServer,
		server.WithSSEEndpoint("/api/v1/mcp/sse"),
		server.WithMessageEndpoint("/api/v1/mcp/sse_message"),
	)

	// 初始化 Executor 和工具注册表
	s.executor = executor.NewExecutor(browserMgr)
	s.toolRegistry = executor.NewMCPToolRegistry(s.executor, s.mcpServer)

	return s
}

func (s *MCPServer) StartStreamableHTTPServer(port string) error {
	go func() {
		newServer := server.NewStreamableHTTPServer(
			s.mcpServer,
			server.WithEndpointPath("/mcp"),
			server.WithStateful(true),
		)
		if err := newServer.Start(port); err != nil {
			logger.Error(s.ctx, "Failed to start streamable HTTP server: %v", err)
		}
		logger.Info(s.ctx, "Streamable HTTP server started on %s", port)
	}()
	return nil
}

// Start 启动 MCP 服务
func (s *MCPServer) Start() error {
	logger.Info(s.ctx, "MCP server started")

	// 加载所有标记为 MCP 命令的脚本
	if err := s.loadMCPScripts(); err != nil {
		return fmt.Errorf("failed to load MCP scripts: %w", err)
	}

	// 注册所有脚本为工具
	if err := s.registerAllTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	// 注册 Executor 工具
	if err := s.toolRegistry.RegisterAllTools(); err != nil {
		return fmt.Errorf("failed to register executor tools: %w", err)
	}
	
	logger.Info(s.ctx, "Registered %d executor tools", len(s.toolRegistry.GetToolMetadata()))

	return nil
}

// Stop 停止 MCP 服务
func (s *MCPServer) Stop() {
	logger.Info(s.ctx, "MCP server stopped")
	s.cancel()
}

// loadMCPScripts 加载所有 MCP 脚本
func (s *MCPServer) loadMCPScripts() error {
	scripts, err := s.storage.ListScripts()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, script := range scripts {
		if script.IsMCPCommand && script.MCPCommandName != "" {
			s.scripts[script.ID] = script
			s.scriptsByName[script.MCPCommandName] = script
			count++
		}
	}

	logger.Info(s.ctx, "Loaded %d MCP commands", count)
	return nil
}

// registerAllTools 注册所有脚本为 MCP 工具
func (s *MCPServer) registerAllTools() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, script := range s.scripts {
		if err := s.registerTool(script); err != nil {
			logger.Warn(s.ctx, "Failed to register tool %s: %v", script.MCPCommandName, err)
			continue
		}
	}

	return nil
}

// registerTool 注册单个脚本为工具
func (s *MCPServer) registerTool(script *models.Script) error {
	opts := []mcpgo.ToolOption{
		mcpgo.WithDescription(script.MCPCommandDescription),
	}

	// 如果脚本有 InputSchema，添加参数
	if script.MCPInputSchema != nil {

		debugSchema, _ := json.Marshal(script.MCPInputSchema)
		logger.Info(s.ctx, "MCP input schema: %s", string(debugSchema))

		if props, ok := script.MCPInputSchema["properties"].(map[string]interface{}); ok {
			for propName, propDef := range props {
				if propDefMap, ok := propDef.(map[string]interface{}); ok {
					desc := ""
					if d, ok := propDefMap["description"].(string); ok {
						desc = d
					}

					propType := ""
					if t, ok := propDefMap["type"].(string); ok {
						propType = t
					}

					// 根据类型添加参数
					switch propType {
					case "string":
						opts = append(opts, mcpgo.WithString(propName, mcpgo.Description(desc)))
					case "number", "integer":
						opts = append(opts, mcpgo.WithNumber(propName, mcpgo.Description(desc)))
					case "boolean":
						opts = append(opts, mcpgo.WithBoolean(propName, mcpgo.Description(desc)))
					}
				}
			}
		}
	}

	// 创建工具处理器
	handler := s.createToolHandler(script)

	tool := mcpgo.NewTool(script.MCPCommandName, opts...)

	// 注册工具
	s.mcpServer.AddTool(tool, handler)
	return nil
}

// createToolHandler 创建工具处理器
func (s *MCPServer) createToolHandler(script *models.Script) func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return func(ctx context.Context, request mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		logger.Info(ctx, "Executing MCP command: %s (script: %s)", script.MCPCommandName, script.Name)
		logger.Info(ctx, "MCP command arguments: %v", request.Params.Arguments)

		// 检查浏览器是否运行
		if !s.browserMgr.IsRunning() {
			logger.Info(ctx, "Browser not running, starting...")
			if err := s.browserMgr.Start(ctx); err != nil {
				return mcpgo.NewToolResultError(fmt.Sprintf("Failed to start browser: %v", err)), nil
			}
			logger.Info(ctx, "Browser started successfully")
		}

		// 创建脚本副本并替换占位符
		scriptToRun := script.Copy()

		// 合并参数：先使用脚本预设变量，再用外部传入的参数覆盖
		params := make(map[string]string)

		// 1. 首先添加脚本的预设变量
		if scriptToRun.Variables != nil {
			for key, value := range scriptToRun.Variables {
				params[key] = value
			}
		}

		// 2. 外部传入的参数会覆盖预设变量
		if request.Params.Arguments != nil {
			if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
				for key, value := range argsMap {
					params[key] = fmt.Sprintf("%v", value)
				}
				for key := range scriptToRun.Variables {
					if _, ok := params[key]; ok {
						scriptToRun.Variables[key] = params[key]
					}
				}
			}
		}

		// 替换 URL 中的占位符
		if urlParam, ok := params["url"]; ok && urlParam != "" {
			scriptToRun.URL = urlParam
		} else {
			scriptToRun.URL = s.replacePlaceholders(scriptToRun.URL, params)
		}

		// 替换所有 action 中的占位符
		for i := range scriptToRun.Actions {
			scriptToRun.Actions[i].Selector = s.replacePlaceholders(scriptToRun.Actions[i].Selector, params)
			scriptToRun.Actions[i].XPath = s.replacePlaceholders(scriptToRun.Actions[i].XPath, params)
			scriptToRun.Actions[i].Value = s.replacePlaceholders(scriptToRun.Actions[i].Value, params)
			scriptToRun.Actions[i].URL = s.replacePlaceholders(scriptToRun.Actions[i].URL, params)
			scriptToRun.Actions[i].JSCode = s.replacePlaceholders(scriptToRun.Actions[i].JSCode, params)

			for j := range scriptToRun.Actions[i].FilePaths {
				scriptToRun.Actions[i].FilePaths[j] = s.replacePlaceholders(scriptToRun.Actions[i].FilePaths[j], params)
			}
		}

		// 执行脚本
		playResult, page, err := s.browserMgr.PlayScript(ctx, scriptToRun)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("Failed to execute script: %v", err)), nil
		}

		// 关闭页面
		if err := s.browserMgr.CloseActivePage(ctx, page); err != nil {
			logger.Warn(ctx, "Failed to close page: %v", err)
		}

		resultData := map[string]interface{}{
			"success": playResult.Success,
			"message": playResult.Message,
		}
		if len(playResult.ExtractedData) > 0 {
			resultData["extracted_data"] = playResult.ExtractedData
		}

		return mcpgo.NewToolResultJSON(resultData)
	}
}

// replacePlaceholders 替换字符串中的占位符
func (s *MCPServer) replacePlaceholders(text string, params map[string]string) string {
	if text == "" {
		return text
	}

	result := text
	for key, value := range params {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// 清理未替换的占位符
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	result = re.ReplaceAllString(result, "")

	return result
}

// RegisterScript 注册脚本为 MCP 命令
func (s *MCPServer) RegisterScript(script *models.Script) error {
	if !script.IsMCPCommand || script.MCPCommandName == "" {
		return fmt.Errorf("script is not marked as MCP command or missing command name")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查命令名是否已存在
	if existing, exists := s.scriptsByName[script.MCPCommandName]; exists && existing.ID != script.ID {
		return fmt.Errorf("command name '%s' is already used by script '%s'", script.MCPCommandName, existing.Name)
	}

	s.scripts[script.ID] = script
	s.scriptsByName[script.MCPCommandName] = script

	// 注册工具
	if err := s.registerTool(script); err != nil {
		return fmt.Errorf("failed to register tool: %w", err)
	}

	logger.Info(s.ctx, "Registered MCP command: %s (script: %s)", script.MCPCommandName, script.Name)
	return nil
}

// UnregisterScript 取消注册脚本
func (s *MCPServer) UnregisterScript(scriptID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if script, exists := s.scripts[scriptID]; exists {
		// TODO: mcp-go 可能需要添加删除工具的方法
		delete(s.scriptsByName, script.MCPCommandName)
		delete(s.scripts, scriptID)
		logger.Info(s.ctx, "Unregistered MCP command: %s", script.MCPCommandName)
	}
}

// GetStatus 获取 MCP 服务状态
func (s *MCPServer) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	commands := make([]map[string]string, 0, len(s.scripts))
	for _, script := range s.scripts {
		commands = append(commands, map[string]string{
			"name":        script.MCPCommandName,
			"description": script.MCPCommandDescription,
			"script_name": script.Name,
			"script_id":   script.ID,
		})
	}

	return map[string]interface{}{
		"running":       true,
		"commands":      commands,
		"command_count": len(s.scripts),
	}
}

// CallTool 直接调用工具（用于 Agent）
func (s *MCPServer) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	// 检查是否是 Executor 工具（以 "browser_" 开头）
	if strings.HasPrefix(name, "browser_") {
		return s.callExecutorTool(ctx, name, arguments)
	}

	// 处理脚本工具
	s.mu.RLock()
	script, exists := s.scriptsByName[name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("command not found: %s", name)
	}

	logger.Info(ctx, "CallTool: Executing MCP command: %s (script: %s), arguments %+v", name, script.Name, arguments)

	// 检查浏览器是否运行
	if !s.browserMgr.IsRunning() {
		logger.Info(ctx, "Browser not running, starting...")
		if err := s.browserMgr.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start browser: %w", err)
		}
	}

	// 创建脚本副本并替换占位符
	scriptToRun := script.Copy()
	params := make(map[string]string)
	for key, value := range scriptToRun.Variables {
		params[key] = value
	}

	for key, value := range arguments {
		params[key] = fmt.Sprintf("%v", value)
	}

	for key := range scriptToRun.Variables {
		if _, ok := params[key]; ok {
			scriptToRun.Variables[key] = params[key]
		}
	}

	// 替换占位符
	if urlParam, ok := params["url"]; ok && urlParam != "" {
		scriptToRun.URL = urlParam
	} else {
		scriptToRun.URL = s.replacePlaceholders(scriptToRun.URL, params)
	}

	for i := range scriptToRun.Actions {
		scriptToRun.Actions[i].Selector = s.replacePlaceholders(scriptToRun.Actions[i].Selector, params)
		scriptToRun.Actions[i].XPath = s.replacePlaceholders(scriptToRun.Actions[i].XPath, params)
		scriptToRun.Actions[i].Value = s.replacePlaceholders(scriptToRun.Actions[i].Value, params)
		scriptToRun.Actions[i].URL = s.replacePlaceholders(scriptToRun.Actions[i].URL, params)
		scriptToRun.Actions[i].JSCode = s.replacePlaceholders(scriptToRun.Actions[i].JSCode, params)

		for j := range scriptToRun.Actions[i].FilePaths {
			scriptToRun.Actions[i].FilePaths[j] = s.replacePlaceholders(scriptToRun.Actions[i].FilePaths[j], params)
		}
	}

	// 执行脚本
	playResult, page, err := s.browserMgr.PlayScript(ctx, scriptToRun)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	// 关闭页面
	if err := s.browserMgr.CloseActivePage(ctx, page); err != nil {
		logger.Warn(ctx, "Failed to close page: %v", err)
	}

	// 返回结果
	result := map[string]interface{}{
		"success": playResult.Success,
		"message": playResult.Message,
	}

	if len(playResult.ExtractedData) > 0 {
		result["extracted_data"] = playResult.ExtractedData
	}

	return result, nil
}

func (s *MCPServer) ServeSteamableHTTP(w http.ResponseWriter, r *http.Request) {
	logger.Info(r.Context(), "ServeHTTP: Method=%s, Path=%s, RemoteAddr=%s", r.Method, r.URL.Path, r.RemoteAddr)
	s.streamableHTTPServer.ServeHTTP(w, r)
}

func (s *MCPServer) GetSSEServer() *server.SSEServer {
	return s.sseServer
}

// callExecutorTool 调用 Executor 工具
func (s *MCPServer) callExecutorTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	logger.Info(ctx, "CallTool: Executing Executor tool: %s, arguments %+v", name, arguments)

	// 确保浏览器已启动
	if !s.browserMgr.IsRunning() {
		logger.Info(ctx, "Browser not running, starting...")
		if err := s.browserMgr.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start browser: %w", err)
		}
	}

	// 根据工具名调用相应的 Executor 方法
	switch name {
	case "browser_navigate":
		url, _ := arguments["url"].(string)
		waitUntil, _ := arguments["wait_until"].(string)
		
		opts := &executor.NavigateOptions{
			Timeout: 60 * time.Second, // 设置默认超时为 60 秒
		}
		if waitUntil != "" {
			opts.WaitUntil = waitUntil
		}
		
		result, err := s.executor.Navigate(ctx, url, opts)
		if err != nil {
			return nil, err
		}
		response := map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}
		// 如果有 Data 字段，包含它（特别是 semantic_tree）
		if len(result.Data) > 0 {
			response["data"] = result.Data
		}
		return response, nil

	case "browser_click":
		identifier, _ := arguments["identifier"].(string)
		waitVisible, _ := arguments["wait_visible"].(bool)
		
		opts := &executor.ClickOptions{
			WaitVisible: waitVisible,
			Timeout:     30 * time.Second, // 设置默认超时为 30 秒
		}
		
		result, err := s.executor.Click(ctx, identifier, opts)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}, nil

	case "browser_type":
		identifier, _ := arguments["identifier"].(string)
		text, _ := arguments["text"].(string)
		clear := true
		if clearArg, ok := arguments["clear"].(bool); ok {
			clear = clearArg
		}
		
		opts := &executor.TypeOptions{
			Clear:   clear,
			Timeout: 30 * time.Second, // 设置默认超时为 30 秒
		}
		
		result, err := s.executor.Type(ctx, identifier, text, opts)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}, nil

	case "browser_select":
		identifier, _ := arguments["identifier"].(string)
		value, _ := arguments["value"].(string)
		
		opts := &executor.SelectOptions{
			Timeout: 30 * time.Second, // 设置默认超时为 30 秒
		}
		
		result, err := s.executor.Select(ctx, identifier, value, opts)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}, nil

	case "browser_screenshot":
		fullPage, _ := arguments["full_page"].(bool)
		format, _ := arguments["format"].(string)
		if format == "" {
			format = "png"
		}
		
		opts := &executor.ScreenshotOptions{
			FullPage: fullPage,
			Format:   format,
		}
		
		result, err := s.executor.Screenshot(ctx, opts)
		if err != nil {
			return nil, err
		}
		response := map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}
		if len(result.Data) > 0 {
			response["data"] = result.Data
		}
		return response, nil

	case "browser_extract":
		selector, _ := arguments["selector"].(string)
		extractType, _ := arguments["type"].(string)
		if extractType == "" {
			extractType = "text"
		}
		multiple, _ := arguments["multiple"].(bool)
		
		opts := &executor.ExtractOptions{
			Selector: selector,
			Type:     extractType,
			Multiple: multiple,
		}
		
		result, err := s.executor.Extract(ctx, opts)
		if err != nil {
			return nil, err
		}
		response := map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}
		if len(result.Data) > 0 {
			response["data"] = result.Data
		}
		return response, nil

	case "browser_get_semantic_tree":
		simple := true
		if simpleArg, ok := arguments["simple"].(bool); ok {
			simple = simpleArg
		}
		
		tree, err := s.executor.GetSemanticTree(ctx)
		if err != nil {
			return nil, err
		}
		
		response := map[string]interface{}{
			"success": true,
			"message": "Successfully retrieved semantic tree",
		}
		
		if simple {
			response["data"] = map[string]interface{}{
				"semantic_tree": tree.SerializeToSimpleText(),
			}
		} else {
			response["data"] = map[string]interface{}{
				"semantic_tree": tree,
			}
		}
		
		return response, nil

	case "browser_get_page_info":
		result, err := s.executor.GetPageInfo(ctx)
		if err != nil {
			return nil, err
		}
		response := map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}
		if len(result.Data) > 0 {
			response["data"] = result.Data
		}
		return response, nil

	case "browser_wait_for":
		identifier, _ := arguments["identifier"].(string)
		state, _ := arguments["state"].(string)
		if state == "" {
			state = "visible"
		}
		
		opts := &executor.WaitForOptions{
			State:   state,
			Timeout: 30 * time.Second, // 设置默认超时为 30 秒
		}
		
		if timeout, ok := arguments["timeout"].(float64); ok && timeout > 0 {
			opts.Timeout = time.Duration(timeout) * time.Second
		}
		
		result, err := s.executor.WaitFor(ctx, identifier, opts)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"success": result.Success,
			"message": result.Message,
		}, nil

	case "browser_scroll":
		direction, _ := arguments["direction"].(string)
		if direction == "" || direction == "bottom" {
			result, err := s.executor.ScrollToBottom(ctx)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{
				"success": result.Success,
				"message": result.Message,
			}, nil
		}
		
		// 滚动到顶部或元素
		page := s.executor.GetRodPage()
		if page != nil {
			if direction == "top" {
				_, err := page.Eval(`() => window.scrollTo(0, 0)`)
				if err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"success": true,
					"message": "Scrolled to top",
				}, nil
			}
		}
		
		return map[string]interface{}{
			"success": false,
			"message": "Invalid scroll direction",
		}, nil

	default:
		return nil, fmt.Errorf("unknown executor tool: %s", name)
	}
}
