package mcp

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

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

	return s
}

func (s *MCPServer) StartHTTPServer(port string) error {
	go func(port string) {
		logger.Info(s.ctx, "MCP server started on port %s", port)
		err := s.streamableHTTPServer.Start(port)
		if err != nil {
			logger.Error(s.ctx, "Failed to start MCP server: %v", err)
		}
	}(port)
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
	// 构建工具定义
	tool := mcpgo.NewTool(
		script.MCPCommandName,
		mcpgo.WithDescription(script.MCPCommandDescription),
		mcpgo.WithString("url", mcpgo.Description("URL to navigate (optional, overrides script URL)")),
	)

	// 如果脚本有 InputSchema，添加参数
	if script.MCPInputSchema != nil {
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
						tool = mcpgo.NewTool(
							script.MCPCommandName,
							mcpgo.WithDescription(script.MCPCommandDescription),
							mcpgo.WithString(propName, mcpgo.Description(desc)),
						)
					case "number", "integer":
						tool = mcpgo.NewTool(
							script.MCPCommandName,
							mcpgo.WithDescription(script.MCPCommandDescription),
							mcpgo.WithNumber(propName, mcpgo.Description(desc)),
						)
					case "boolean":
						tool = mcpgo.NewTool(
							script.MCPCommandName,
							mcpgo.WithDescription(script.MCPCommandDescription),
							mcpgo.WithBoolean(propName, mcpgo.Description(desc)),
						)
					}
				}
			}
		}
	}

	// 创建工具处理器
	handler := s.createToolHandler(script)

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
		scriptToRun := *script

		// 将 arguments 转换为 map[string]string
		params := make(map[string]string)
		if request.Params.Arguments != nil {
			if argsMap, ok := request.Params.Arguments.(map[string]interface{}); ok {
				for key, value := range argsMap {
					params[key] = fmt.Sprintf("%v", value)
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
		playResult, err := s.browserMgr.PlayScript(ctx, &scriptToRun)
		if err != nil {
			return mcpgo.NewToolResultError(fmt.Sprintf("Failed to execute script: %v", err)), nil
		}

		// 关闭页面
		if err := s.browserMgr.CloseActivePage(ctx); err != nil {
			logger.Warn(ctx, "Failed to close page: %v", err)
		}

		// 构建结果
		resultText := fmt.Sprintf("Success: %v\nMessage: %s", playResult.Success, playResult.Message)
		if len(playResult.ExtractedData) > 0 {
			resultText += fmt.Sprintf("\nExtracted Data: %v", playResult.ExtractedData)
		}

		return mcpgo.NewToolResultText(resultText), nil
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
	s.mu.RLock()
	script, exists := s.scriptsByName[name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("command not found: %s", name)
	}

	logger.Info(ctx, "CallTool: Executing MCP command: %s (script: %s)", name, script.Name)

	// 检查浏览器是否运行
	if !s.browserMgr.IsRunning() {
		logger.Info(ctx, "Browser not running, starting...")
		if err := s.browserMgr.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start browser: %w", err)
		}
	}

	// 创建脚本副本并替换占位符
	scriptToRun := *script
	params := make(map[string]string)
	for key, value := range arguments {
		params[key] = fmt.Sprintf("%v", value)
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
	playResult, err := s.browserMgr.PlayScript(ctx, &scriptToRun)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	// 关闭页面
	if err := s.browserMgr.CloseActivePage(ctx); err != nil {
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
