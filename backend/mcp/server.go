package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/services/browser"
	"github.com/browserwing/browserwing/storage"
)

// MCPRequest MCP 协议请求
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// MCPResponse MCP 协议响应
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError MCP 错误
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolInfo MCP 工具信息
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPServer MCP 服务器
type MCPServer struct {
	storage       *storage.BoltDB
	browserMgr    *browser.Manager
	scripts       map[string]*models.Script // scriptID -> Script
	scriptsByName map[string]*models.Script // commandName -> Script
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewMCPServer 创建 MCP 服务器
func NewMCPServer(storage *storage.BoltDB, browserMgr *browser.Manager) *MCPServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &MCPServer{
		storage:       storage,
		browserMgr:    browserMgr,
		scripts:       make(map[string]*models.Script),
		scriptsByName: make(map[string]*models.Script),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start 启动 MCP 服务
func (s *MCPServer) Start() error {
	logger.Info(s.ctx, "MCP server started")

	// 加载所有标记为 MCP 命令的脚本
	if err := s.loadMCPScripts(); err != nil {
		return fmt.Errorf("failed to load MCP scripts: %w", err)
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

	logger.Info(s.ctx, "Registered MCP command: %s (script: %s)", script.MCPCommandName, script.Name)
	return nil
}

// UnregisterScript 取消注册脚本
func (s *MCPServer) UnregisterScript(scriptID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if script, exists := s.scripts[scriptID]; exists {
		delete(s.scriptsByName, script.MCPCommandName)
		delete(s.scripts, scriptID)
		logger.Info(s.ctx, "Unregistered MCP command: %s", script.MCPCommandName)
	}
}

// GetTools 获取所有可用的工具列表（MCP 协议）
func (s *MCPServer) GetTools() []ToolInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]ToolInfo, 0, len(s.scripts))
	for _, script := range s.scripts {
		// 使用脚本定义的 InputSchema，如果没有则使用默认的空 schema
		inputSchema := script.MCPInputSchema
		if inputSchema == nil {
			inputSchema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			}
		}

		tools = append(tools, ToolInfo{
			Name:        script.MCPCommandName,
			Description: script.MCPCommandDescription,
			InputSchema: inputSchema,
		})
	}

	return tools
}

// CallTool 调用工具（执行脚本）
func (s *MCPServer) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error) {
	s.mu.RLock()
	script, exists := s.scriptsByName[name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("command not found: %s", name)
	}

	logger.Info(ctx, "Executing MCP command: %s (script: %s)", name, script.Name)
	logger.Info(ctx, "MCP command arguments: %v", arguments)

	// 检查浏览器是否运行，如果未运行则自动启动
	if !s.browserMgr.IsRunning() {
		logger.Info(ctx, "Browser not running, starting...")
		if err := s.browserMgr.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start browser: %w", err)
		}
		logger.Info(ctx, "Browser started successfully")
	}

	// 创建脚本副本并替换占位符
	scriptToRun := *script

	// 将 arguments 转换为 map[string]string 以便替换占位符
	params := make(map[string]string)
	for key, value := range arguments {
		// 将各种类型的值转换为字符串
		switch v := value.(type) {
		case string:
			params[key] = v
		case float64:
			params[key] = fmt.Sprintf("%v", v)
		case bool:
			params[key] = fmt.Sprintf("%v", v)
		default:
			params[key] = fmt.Sprintf("%v", v)
		}
	}

	// 如果用户提供了 url 参数，使用它；否则替换 URL 中的占位符
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

		// 替换文件路径中的占位符
		for j := range scriptToRun.Actions[i].FilePaths {
			scriptToRun.Actions[i].FilePaths[j] = s.replacePlaceholders(scriptToRun.Actions[i].FilePaths[j], params)
		}
	}

	// 执行脚本（使用 Manager 的 PlayScript 方法）
	playResult, err := s.browserMgr.PlayScript(ctx, &scriptToRun)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	// 执行完成后，关闭创建的页面（避免页面堆积）
	if err := s.browserMgr.CloseActivePage(ctx); err != nil {
		logger.Warn(ctx, "Failed to close page (does not affect result): %v", err)
	}

	// 返回执行结果
	result := map[string]interface{}{
		"success": playResult.Success,
		"message": playResult.Message,
	}

	// 如果有抓取的数据，添加到结果中
	if len(playResult.ExtractedData) > 0 {
		result["extracted_data"] = playResult.ExtractedData
	}

	return result, nil
}

// HandleRequest 处理 MCP 请求（stdio 模式）
func (s *MCPServer) HandleRequest(reader io.Reader, writer io.Writer) error {
	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(writer)

	for {
		var req MCPRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to parse request: %w", err)
		}

		resp := s.processRequest(req)
		if err := encoder.Encode(resp); err != nil {
			return fmt.Errorf("failed to send response: %w", err)
		}
	}
}

// processRequest 处理单个 MCP 请求
func (s *MCPServer) processRequest(req MCPRequest) MCPResponse {
	ctx := s.ctx

	switch req.Method {
	case "initialize":
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "browserpilot",
					"version": "1.0.0",
				},
			},
		}

	case "tools/list":
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.GetTools(),
			},
		}

	case "tools/call":
		toolName, _ := req.Params["name"].(string)
		arguments, _ := req.Params["arguments"].(map[string]interface{})

		result, err := s.CallTool(ctx, toolName, arguments)
		if err != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &MCPError{
					Code:    -32603,
					Message: err.Error(),
				},
			}
		}

		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("%v", result),
					},
				},
			},
		}

	default:
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Unknown method: %s", req.Method),
			},
		}
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

// ServeHTTP 处理 HTTP/SSE 模式的 MCP 请求
func (s *MCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 设置 CORS 头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == "POST" {
		// 处理单个请求
		s.handleHTTPRequest(w, r)
		return
	}

	if r.Method == "GET" {
		// SSE 模式 - 用于长连接和事件推送
		s.handleSSE(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleHTTPRequest 处理 HTTP POST 请求
func (s *MCPServer) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32700,
				Message: "Parse error: " + err.Error(),
			},
		})
		return
	}

	resp := s.processRequest(req)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleSSE 处理 Server-Sent Events 连接
func (s *MCPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	// 设置 SSE 头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 发送初始连接消息
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	// 保持连接，等待客户端断开
	<-r.Context().Done()
}

// replacePlaceholders 替换字符串中的占位符
// 支持 ${field} 格式，例如 ${keyword}, ${page}, ${category} 等
func (s *MCPServer) replacePlaceholders(text string, params map[string]string) string {
	if text == "" {
		return text
	}

	// 替换所有占位符
	result := text
	for key, value := range params {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// 使用正则处理剩余未替换的占位符（替换为空字符串）
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	result = re.ReplaceAllString(result, "")

	return result
}
