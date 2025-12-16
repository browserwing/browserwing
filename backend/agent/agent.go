package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/memory"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools"
	browsermcp "github.com/browserwing/browserwing/mcp"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/storage"
	"github.com/google/uuid"

	// 本地工具包
	localtools "github.com/browserwing/browserwing/agent/tools"
)

const (
	defSystemPrompt = `You are a helpful AI assistant with access to various tools. When users ask questions or make requests, you should:

1. Analyze if any of your available tools can help answer the question
2. Use the appropriate tools to gather information
3. Provide a comprehensive answer based on the tool results

Always prefer using tools over making up information. If you have a tool that can help, use it.`
)

// ChatMessage 聊天消息
type ChatMessage struct {
	ID        string      `json:"id"`
	Role      string      `json:"role"` // user, assistant, system
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
	ToolCalls []*ToolCall `json:"tool_calls,omitempty"` // 工具调用信息
}

// ToolCall 工具调用信息
type ToolCall struct {
	ToolName string `json:"tool_name"`
	Status   string `json:"status"` // calling, success, error
	Message  string `json:"message,omitempty"`
}

// ChatSession 聊天会话
type ChatSession struct {
	ID        string        `json:"id"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// StreamChunk 流式响应数据块
type StreamChunk struct {
	Type      string    `json:"type"` // message, tool_call, done, error
	Content   string    `json:"content,omitempty"`
	ToolCall  *ToolCall `json:"tool_call,omitempty"`
	Error     string    `json:"error,omitempty"`
	MessageID string    `json:"message_id,omitempty"`
}

// MCPTool 实现 interfaces.Tool 接口,用于调用本地 MCP 服务
type MCPTool struct {
	name        string
	description string
	inputSchema map[string]interface{}
	mcpServer   *browsermcp.MCPServer
}

func (t *MCPTool) Name() string {
	return t.name
}

func (t *MCPTool) Description() string {
	return t.description
}

func (t *MCPTool) InputSchema() map[string]interface{} {
	return t.inputSchema
}

func (t *MCPTool) Execute(ctx context.Context, input string) (string, error) {
	logger.Info(ctx, "Calling MCP tool: %s, input: %s, Parameters: %+v", t.name, input, t.Parameters())

	// 解析输入参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("failed to parse input parameters: %w", err)
	}

	// 调用 MCP 服务器执行脚本
	result, err := t.mcpServer.CallTool(ctx, t.name, args)
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool: %w", err)
	}

	// 返回 JSON 格式的结果
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to serialize result: %w", err)
	}

	return string(resultJSON), nil
}

// Run implements interfaces.Tool.Run
func (t *MCPTool) Run(ctx context.Context, input string) (string, error) {
	return t.Execute(ctx, input)
}

// Parameters implements interfaces.Tool.Parameters
func (t *MCPTool) Parameters() map[string]interfaces.ParameterSpec {
	// 将 inputSchema 转换为 ParameterSpec
	params := make(map[string]interfaces.ParameterSpec)
	properties, _ := t.inputSchema["properties"].(map[string]interface{})
	for name, schema := range properties {
		schemaMap, ok := schema.(map[string]interface{})
		if !ok {
			continue
		}

		spec := interfaces.ParameterSpec{
			Required: false,
		}

		if typeVal, ok := schemaMap["type"].(string); ok {
			spec.Type = typeVal
		}
		if descVal, ok := schemaMap["description"].(string); ok {
			spec.Description = descVal
		}
		if reqVal, ok := schemaMap["required"].(bool); ok {
			spec.Required = reqVal
		}

		params[name] = spec
	}
	return params
}

// AgentManager Agent 管理器
type AgentManager struct {
	db         *storage.BoltDB
	mcpServer  *browsermcp.MCPServer
	sessions   map[string]*ChatSession
	agents     map[string]*agent.Agent // sessionID -> Agent 实例
	llmClient  interfaces.LLM
	toolReg    *tools.Registry
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	mcpWatcher *time.Ticker // MCP 命令监听器
}

// NewAgentManager 创建 Agent 管理器
func NewAgentManager(db *storage.BoltDB, mcpServer *browsermcp.MCPServer) (*AgentManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	am := &AgentManager{
		db:        db,
		mcpServer: mcpServer,
		sessions:  make(map[string]*ChatSession),
		agents:    make(map[string]*agent.Agent),
		toolReg:   tools.NewRegistry(),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 从数据库加载默认 LLM 配置
	if err := am.LoadLLMFromDatabase(); err != nil {
		logger.Warn(ctx, "Failed to load LLM configuration: %v (Please configure in LLM Management page)", err)
	}

	// 从数据库加载持久化的会话
	if err := am.loadSessionsFromDB(); err != nil {
		logger.Warn(ctx, "Failed to load session: %v", err)
	}

	// 初始化 MCP 工具
	if err := am.initMCPTools(); err != nil {
		logger.Warn(ctx, "Failed to initialize MCP tools: %v", err)
	}

	// 启动 MCP 命令监听器
	am.startMCPWatcher()

	return am, nil
}

// initMCPTools 初始化 MCP 工具
func (am *AgentManager) initMCPTools() error {
	if am.mcpServer == nil {
		return fmt.Errorf("MCP server is not initialized")
	}

	// 初始化预设工具
	if err := am.initPresetTools(); err != nil {
		logger.Warn(am.ctx, "Failed to initialize preset tools: %v", err)
	}

	// 获取所有工具配置
	toolConfigs, err := am.db.ListToolConfigs()
	if err != nil {
		logger.Warn(am.ctx, "Failed to list tool configs: %v", err)
		toolConfigs = []*models.ToolConfig{}
	}

	// 构建脚本工具配置映射
	scriptToolConfigMap := make(map[string]*models.ToolConfig)
	for _, cfg := range toolConfigs {
		if cfg.Type == models.ToolTypeScript {
			scriptToolConfigMap[cfg.ScriptID] = cfg
		}
	}

	// 获取所有 MCP 命令脚本
	scripts, err := am.db.ListScripts()
	if err != nil {
		return fmt.Errorf("failed to list scripts: %w", err)
	}

	count := 0
	for _, script := range scripts {
		if !script.IsMCPCommand || script.MCPCommandName == "" {
			continue
		}

		// 检查该脚本工具是否被禁用
		if cfg, exists := scriptToolConfigMap[script.ID]; exists && !cfg.Enabled {
			continue
		}

		// 创建 MCP 工具
		tool := &MCPTool{
			name:        script.MCPCommandName,
			description: script.MCPCommandDescription,
			inputSchema: script.MCPInputSchema,
			mcpServer:   am.mcpServer,
		}

		// 注册到工具注册表
		am.toolReg.Register(tool)
		count++
	}

	if count == 0 {
		logger.Warn(am.ctx, "⚠ No MCP command scripts found, please create and enable MCP commands in Script Management page")
	}

	return nil
}

// initPresetTools 初始化预设工具
func (am *AgentManager) initPresetTools() error {
	return localtools.InitPresetTools(am.ctx, am.toolReg, am.db)
}

// startMCPWatcher 启动 MCP 命令监听器
func (am *AgentManager) startMCPWatcher() {
	// 每 5 秒检查一次 MCP 命令是否有更新
	am.mcpWatcher = time.NewTicker(5 * time.Second)

	go func() {
		for {
			select {
			case <-am.ctx.Done():
				am.mcpWatcher.Stop()
				return
			case <-am.mcpWatcher.C:
				// 重新加载 MCP 工具列表
				if err := am.refreshMCPTools(); err != nil {
					logger.Warn(am.ctx, "Failed to refresh MCP tool list: %v", err)
				}
			}
		}
	}()

	logger.Info(am.ctx, "✓ MCP command listener has started")
}

// refreshMCPTools 刷新 MCP 工具列表
func (am *AgentManager) refreshMCPTools() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// 重新初始化工具注册表
	am.toolReg = tools.NewRegistry()
	if err := am.initMCPTools(); err != nil {
		return err
	}

	// Note: agent.Agent 不支持动态更新工具
	// 新会话会在创建时自动使用最新的工具列表

	return nil
}

// LoadLLMFromDatabase 从数据库加载默认 LLM 配置
func (am *AgentManager) LoadLLMFromDatabase() error {
	// 获取默认的 LLM 配置
	configs, err := am.db.ListLLMConfigs()
	if err != nil {
		return fmt.Errorf("failed to list LLM configs: %w", err)
	}

	if len(configs) == 0 {
		return fmt.Errorf("no available LLM configs")
	}

	// 查找默认配置或第一个激活的配置
	var selectedConfig *models.LLMConfigModel
	for _, cfg := range configs {
		if !cfg.IsActive {
			continue
		}
		if cfg.IsDefault {
			selectedConfig = cfg
			break
		}
		if selectedConfig == nil {
			selectedConfig = cfg
		}
	}

	if selectedConfig == nil {
		return fmt.Errorf("no active LLM config found")
	}

	return am.SetLLMConfig(selectedConfig)
}

// SetLLMConfig 设置 LLM 配置
func (am *AgentManager) SetLLMConfig(config *models.LLMConfigModel) error {
	// 验证配置
	if err := ValidateLLMConfig(config); err != nil {
		return fmt.Errorf("failed to validate LLM config: %w", err)
	}

	// 创建 LLM 客户端
	client, err := CreateLLMClient(config)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	am.mu.Lock()
	am.llmClient = client
	am.mu.Unlock()

	logger.Info(am.ctx, "✓ LLM configuration loaded successfully: %s", GetProviderInfo(config))

	// 检查模型是否支持工具调用
	if !SupportsToolCalling(config.Provider, config.Model) {
		logger.Warn(am.ctx, "⚠ Warning: Model %s (%s) may not support function calling", config.Model, config.Provider)
		logger.Warn(am.ctx, "  Recommended models that support function calling: GPT-4o, Claude-3.5-Sonnet, Gemini-1.5-Pro, Qwen-Max, etc.")
	}

	return nil
}

// ReloadLLM 重新加载 LLM 配置 (用于配置更新后的热加载)
func (am *AgentManager) ReloadLLM() error {
	return am.LoadLLMFromDatabase()
}

func (am *AgentManager) GetSystemPrompt() string {
	dbSystemPrompt, err := am.db.GetPrompt(models.SystemPromptAIAgentID)
	if err != nil {
		logger.Warn(am.ctx, "Failed to get system prompt: %v", err)
		return defSystemPrompt
	}
	return dbSystemPrompt.Content
}

// loadSessionsFromDB 从数据库加载持久化的会话
func (am *AgentManager) loadSessionsFromDB() error {
	// 加载所有会话
	dbSessions, err := am.db.ListAgentSessions()
	if err != nil {
		return fmt.Errorf("failed to list agent sessions: %w", err)
	}

	logger.Info(am.ctx, "Loaded %d sessions from database", len(dbSessions))

	for _, dbSession := range dbSessions {
		// 加载会话的消息
		dbMessages, err := am.db.ListAgentMessages(dbSession.ID)
		if err != nil {
			logger.Warn(am.ctx, "Failed to load messages for session %s: %v", dbSession.ID, err)
			continue
		}

		// 转换为 ChatMessage
		messages := make([]ChatMessage, 0, len(dbMessages))
		for _, dbMsg := range dbMessages {
			toolCalls := make([]*ToolCall, 0, len(dbMsg.ToolCalls))
			for _, tc := range dbMsg.ToolCalls {
				toolCalls = append(toolCalls, &ToolCall{
					ToolName: tc["tool_name"].(string),
					Status:   tc["status"].(string),
					Message:  tc["message"].(string),
				})
			}

			messages = append(messages, ChatMessage{
				ID:        dbMsg.ID,
				Role:      dbMsg.Role,
				Content:   dbMsg.Content,
				Timestamp: dbMsg.Timestamp,
				ToolCalls: toolCalls,
			})
		}

		// 创建会话对象
		session := &ChatSession{
			ID:        dbSession.ID,
			Messages:  messages,
			CreatedAt: dbSession.CreatedAt,
			UpdatedAt: dbSession.UpdatedAt,
		}

		am.sessions[session.ID] = session

		// 为会话创建 Agent 实例
		if am.llmClient != nil {
			mem := memory.NewConversationBuffer()

			ag, err := agent.NewAgent(
				agent.WithLLM(am.llmClient),
				agent.WithMemory(mem),
				agent.WithTools(am.toolReg.List()...),
				agent.WithSystemPrompt(am.GetSystemPrompt()),
				agent.WithRequirePlanApproval(false),
				agent.WithMaxIterations(5), // 增加最大迭代次数
			)
			if err != nil {
				logger.Warn(am.ctx, "Failed to create Agent for session %s: %v", session.ID, err)
			} else {
				am.agents[session.ID] = ag
			}
		}
	}

	return nil
}

// CreateSession 创建新会话
func (am *AgentManager) CreateSession() *ChatSession {
	am.mu.Lock()
	defer am.mu.Unlock()

	session := &ChatSession{
		ID:        uuid.New().String(),
		Messages:  []ChatMessage{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	am.sessions[session.ID] = session

	// 保存到数据库
	dbSession := &models.AgentSession{
		ID:        session.ID,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
	if err := am.db.SaveAgentSession(dbSession); err != nil {
		logger.Warn(am.ctx, "Failed to save session to database: %v", err)
	}

	// 为新会话创建 Agent 实例
	if am.llmClient != nil {
		// 创建会话内存
		mem := memory.NewConversationBuffer()

		// 获取工具列表
		tools := am.toolReg.List()

		// 创建 Agent - 禁用执行计划审批
		ag, err := agent.NewAgent(
			agent.WithLLM(am.llmClient),
			agent.WithMemory(mem),
			agent.WithTools(tools...),
			agent.WithSystemPrompt(am.GetSystemPrompt()),
			agent.WithRequirePlanApproval(false), // 禁用执行计划审批
			agent.WithMaxIterations(5),           // 增加最大迭代次数，避免过早触发 final call
		)
		if err != nil {
			logger.Warn(am.ctx, "Failed to create Agent for session %s: %v", session.ID, err)
		} else {
			am.agents[session.ID] = ag
			logger.Info(am.ctx, "✓ Created Agent for session %s, tools count: %d", session.ID, len(tools))
			for _, tool := range tools {
				logger.Debug(am.ctx, "  - %s: %s", tool.Name(), tool.Description())
			}
		}
	}

	return session
}

// GetSession 获取会话
func (am *AgentManager) GetSession(sessionID string) (*ChatSession, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	session, ok := am.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("Session not found: %s", sessionID)
	}

	return session, nil
}

// SendMessage 发送消息 (流式)
func (am *AgentManager) SendMessage(sessionID, userMessage string, streamChan chan<- StreamChunk) error {
	defer close(streamChan)

	// 检查 LLM 是否已配置
	if am.llmClient == nil {
		streamChan <- StreamChunk{
			Type:  "error",
			Error: "LLM is not configured, please configure it in the LLM management page",
		}
		return fmt.Errorf("LLM is not configured")
	}

	// 获取会话
	session, err := am.GetSession(sessionID)
	if err != nil {
		streamChan <- StreamChunk{
			Type:  "error",
			Error: err.Error(),
		}
		return err
	}

	// 添加用户消息
	userMsg := ChatMessage{
		ID:        uuid.New().String(),
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now(),
	}

	am.mu.Lock()
	session.Messages = append(session.Messages, userMsg)
	session.UpdatedAt = time.Now()
	am.mu.Unlock()

	// 保存用户消息到数据库
	dbUserMsg := &models.AgentMessage{
		ID:        userMsg.ID,
		SessionID: sessionID,
		Role:      userMsg.Role,
		Content:   userMsg.Content,
		Timestamp: userMsg.Timestamp,
	}
	if err := am.db.SaveAgentMessage(dbUserMsg); err != nil {
		logger.Warn(am.ctx, "Failed to save user message to database: %v", err)
	}

	// 获取 Agent 实例
	am.mu.RLock()
	ag, ok := am.agents[sessionID]
	am.mu.RUnlock()

	if !ok {
		streamChan <- StreamChunk{
			Type:  "error",
			Error: fmt.Sprintf("Agent for session %s is not initialized", sessionID),
		}
		return fmt.Errorf("agent for session %s is not initialized", sessionID)
	}

	// 创建助手消息
	assistantMsg := ChatMessage{
		ID:        uuid.New().String(),
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
		ToolCalls: []*ToolCall{},
	}

	// 创建多租户上下文
	ctx := multitenancy.WithOrgID(am.ctx, "browserpilot")
	ctx = context.WithValue(ctx, memory.ConversationIDKey, sessionID)

	// 使用 Agent 流式处理消息
	streamEvents, err := ag.RunStream(ctx, userMessage)
	if err != nil {
		streamChan <- StreamChunk{
			Type:  "error",
			Error: err.Error(),
		}
		return err
	}

	// 发送消息 ID
	streamChan <- StreamChunk{
		Type:      "message",
		Content:   "",
		MessageID: assistantMsg.ID,
	}

	// 处理流式事件
	toolCallMap := make(map[string]*ToolCall) // 用于跟踪工具调用状态

	for event := range streamEvents {
		switch event.Type {
		case interfaces.AgentEventContent:
			// 文本内容
			assistantMsg.Content += event.Content
			streamChan <- StreamChunk{
				Type:      "message",
				Content:   event.Content,
				MessageID: assistantMsg.ID,
			}

		case interfaces.AgentEventToolResult:
			logger.Warn(ctx, "Received unhandled tool result event: %+v", event)
			if event.ToolCall == nil {
				logger.Error(ctx, "Tool result event missing ToolCall information")
				continue
			}
			tc := event.ToolCall
			toolCall, exists := toolCallMap[tc.Name]
			if !exists {
				continue
			}

			// 更新工具调用状态
			switch tc.Status {
			case "executing":
				toolCall.Status = "calling"
				toolCall.Message = "执行中..."
			case "completed":
				toolCall.Status = "success"
				toolCall.Message = "调用成功"
			case "error":
				toolCall.Status = "error"
				toolCall.Message = "调用失败"
			}

			// 发送工具调用状态
			streamChan <- StreamChunk{
				Type:     "tool_call",
				ToolCall: toolCall,
			}

		case interfaces.AgentEventToolCall:
			logger.Warn(ctx, "Received unhandled tool call event: %+v", event)
			// 工具调用
			if event.ToolCall == nil {
				logger.Error(ctx, "Tool call event missing ToolCall information")
				continue
			}
			tc := event.ToolCall

			// 获取或创建工具调用记录
			toolCall, exists := toolCallMap[tc.Name]
			if !exists {
				toolCall = &ToolCall{
					ToolName: tc.Name,
					Status:   "calling",
				}
				toolCallMap[tc.Name] = toolCall
				assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, toolCall)
			}
			// 发送工具调用状态
			streamChan <- StreamChunk{
				Type:     "tool_call",
				ToolCall: toolCall,
			}
		case interfaces.AgentEventThinking:
			// 思考过程(可选择性展示)
			logger.Debug(ctx, "Agent thinking: %s", event.ThinkingStep)

		case interfaces.AgentEventError:
			// 错误
			streamChan <- StreamChunk{
				Type:  "error",
				Error: event.Error.Error(),
			}
			return event.Error
		}
	}

	// 保存助手消息
	am.mu.Lock()
	session.Messages = append(session.Messages, assistantMsg)
	session.UpdatedAt = time.Now()
	am.mu.Unlock()

	// 保存助手消息到数据库
	var toolCallsData []map[string]interface{}
	for _, tc := range assistantMsg.ToolCalls {
		toolCallsData = append(toolCallsData, map[string]interface{}{
			"tool_name": tc.ToolName,
			"status":    tc.Status,
			"message":   tc.Message,
		})
	}
	dbAssistantMsg := &models.AgentMessage{
		ID:        assistantMsg.ID,
		SessionID: sessionID,
		Role:      assistantMsg.Role,
		Content:   assistantMsg.Content,
		Timestamp: assistantMsg.Timestamp,
		ToolCalls: toolCallsData,
	}
	if err := am.db.SaveAgentMessage(dbAssistantMsg); err != nil {
		logger.Warn(am.ctx, "Failed to save assistant message to database: %v", err)
	}

	// 更新会话时间戳
	dbSession := &models.AgentSession{
		ID:        sessionID,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
	if err := am.db.SaveAgentSession(dbSession); err != nil {
		logger.Warn(am.ctx, "Failed to update session timestamp: %v", err)
	}

	// 发送完成信号
	streamChan <- StreamChunk{
		Type:      "done",
		MessageID: assistantMsg.ID,
	}

	return nil
}

// ListSessions 列出所有会话
func (am *AgentManager) ListSessions() []*ChatSession {
	am.mu.RLock()
	defer am.mu.RUnlock()

	sessions := make([]*ChatSession, 0, len(am.sessions))
	for _, session := range am.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// DeleteSession 删除会话
func (am *AgentManager) DeleteSession(sessionID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, ok := am.sessions[sessionID]; !ok {
		return fmt.Errorf("Session not found: %s", sessionID)
	}

	delete(am.sessions, sessionID)
	delete(am.agents, sessionID)

	// 从数据库删除
	if err := am.db.DeleteAgentSession(sessionID); err != nil {
		logger.Warn(am.ctx, "Failed to delete session from database: %v", err)
	}

	return nil
}

// GetMCPStatus 获取 MCP 状态
func (am *AgentManager) GetMCPStatus() map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	status := map[string]interface{}{
		"connected":  am.toolReg != nil,
		"tools":      []string{},
		"tool_count": 0,
	}

	if am.toolReg != nil {
		toolList := am.toolReg.List()
		toolNames := make([]string, len(toolList))
		for i, tool := range toolList {
			toolNames[i] = tool.Name()
		}
		status["tools"] = toolNames
		status["tool_count"] = len(toolList)
	}

	return status
}

// Stop 停止 Agent 管理器
func (am *AgentManager) Stop() {
	logger.Info(am.ctx, "Agent manager stopped")

	if am.mcpWatcher != nil {
		am.mcpWatcher.Stop()
	}

	am.cancel()
}
