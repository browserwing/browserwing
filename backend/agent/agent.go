package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/memory"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools"
	"github.com/browserwing/browserwing/executor"
	browsermcp "github.com/browserwing/browserwing/mcp"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/storage"
	"github.com/google/uuid"

	// 本地工具包
	localtools "github.com/browserwing/browserwing/agent/tools"
)

// 导入全局工具结果存储
var toolResultStore = localtools.GlobalToolResultStore

const maxIterations = 4

// getStringFromMap 从 map 中安全地获取字符串值
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
	ToolName     string                 `json:"tool_name"`
	Status       string                 `json:"status"` // calling, success, error
	Message      string                 `json:"message,omitempty"`
	Instructions string                 `json:"instructions,omitempty"` // 工具调用说明（为什么调用、如何使用）
	Arguments    map[string]interface{} `json:"arguments,omitempty"`    // 工具调用参数
	Result       string                 `json:"result,omitempty"`       // 工具执行结果
	Timestamp    time.Time              `json:"timestamp"`              // 调用时间戳
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
	mcpServer   browsermcp.IMCPServer
}

func (t *MCPTool) Name() string {
	return t.name
}

func (t *MCPTool) Description() string {
	return t.description
}

func (t *MCPTool) InputSchema() map[string]interface{} {
	// 注意：现在 MCPTool 会被 ToolWrapper 包装
	// ToolWrapper 会自动添加 instructions 参数
	// 这里直接返回原始 schema 即可
	return t.inputSchema
}

func (t *MCPTool) Execute(ctx context.Context, input string) (string, error) {
	logger.Info(ctx, "Calling MCP tool: %s, input: %s, Parameters: %+v", t.name, input, t.Parameters())

	// 解析输入参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("failed to parse input parameters: %w", err)
	}

	// 为浏览器操作创建更长超时的 context（浏览器启动和导航需要更多时间）
	// Executor 工具（browser_*）使用 120 秒超时，其他工具使用原有 context
	execCtx := ctx
	if strings.HasPrefix(t.name, "browser_") {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		logger.Info(ctx, "Using extended timeout (120s) for browser tool: %s", t.name)
	}

	// 调用 MCP 服务器执行脚本
	result, err := t.mcpServer.CallTool(execCtx, t.name, args)
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
	db               *storage.BoltDB
	mcpServer        browsermcp.IMCPServer
	sessions         map[string]*ChatSession
	agents           map[string]*agent.Agent // sessionID -> Agent 实例
	llmClient        interfaces.LLM
	currentLLMConfig *models.LLMConfigModel // 当前使用的 LLM 配置
	toolReg          *tools.Registry
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	mcpWatcher       *time.Ticker // MCP 命令监听器
}

// NewAgentManager 创建 Agent 管理器
func NewAgentManager(db *storage.BoltDB, mcpServer browsermcp.IMCPServer) (*AgentManager, error) {
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

	// 初始化 Executor 工具配置（作为预设工具）
	if err := am.initExecutorToolConfigs(); err != nil {
		logger.Warn(am.ctx, "Failed to initialize executor tool configs: %v", err)
	}

	// 初始化 Executor 工具到 MCP
	if err := am.initExecutorTools(); err != nil {
		logger.Warn(am.ctx, "Failed to initialize executor tools: %v", err)
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

		// 包装工具以添加 instructions 参数和捕获执行结果
		wrappedTool := localtools.WrapTool(tool)
		
		// 注册到工具注册表
		am.toolReg.Register(wrappedTool)
		count++
		
		logger.Info(am.ctx, "Registered script MCP tool (wrapped): %s", script.MCPCommandName)
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

// initExecutorTools 初始化 Executor 工具
// initExecutorToolConfigs 初始化 Executor 工具配置（作为预设工具）
func (am *AgentManager) initExecutorToolConfigs() error {
	// 获取 Executor 工具元数据
	executorTools := executor.GetExecutorToolsMetadata()

	count := 0
	for _, meta := range executorTools {
		// 检查是否已存在配置
		existingConfig, err := am.db.GetToolConfig(meta.Name)
		if err == nil && existingConfig != nil {
			// 配置已存在，跳过
			continue
		}

		// 创建新的工具配置
		config := &models.ToolConfig{
			ID:          meta.Name,
			Name:        meta.Name,
			Type:        models.ToolTypePreset, // 标记为预设工具
			Description: meta.Description,
			Enabled:     true, // 默认启用
			Parameters:  make(map[string]interface{}),
		}

		// 添加分类信息到参数中
		if meta.Category != "" {
			config.Parameters["category"] = meta.Category
		}

		// 保存到数据库
		if err := am.db.SaveToolConfig(config); err != nil {
			logger.Warn(am.ctx, "Failed to save executor tool config %s: %v", meta.Name, err)
			continue
		}

		count++
	}

	if count > 0 {
		logger.Info(am.ctx, "✓ Initialized %d Executor tool configs", count)
	}
	return nil
}

func (am *AgentManager) initExecutorTools() error {
	// 获取 Executor 工具元数据
	executorTools := executor.GetExecutorToolsMetadata()

	// 获取所有工具配置
	toolConfigs, err := am.db.ListToolConfigs()
	if err != nil {
		logger.Warn(am.ctx, "Failed to list tool configs for executor tools: %v", err)
		toolConfigs = []*models.ToolConfig{}
	}

	// 构建配置映射
	configMap := make(map[string]*models.ToolConfig)
	for _, cfg := range toolConfigs {
		if cfg.Type == models.ToolTypePreset {
			configMap[cfg.ID] = cfg
		}
	}

	count := 0
	for _, meta := range executorTools {
		// 检查工具是否被启用
		if config, ok := configMap[meta.Name]; ok && !config.Enabled {
			logger.Info(am.ctx, "Executor tool %s is disabled, skipping", meta.Name)
			continue
		}

		// 为每个 Executor 工具创建 MCPTool 包装器
		tool := &MCPTool{
			name:        meta.Name,
			description: meta.Description,
			inputSchema: buildInputSchemaFromMetadata(meta),
			mcpServer:   am.mcpServer,
		}

		// 包装工具以添加 instructions 参数和捕获执行结果
		wrappedTool := localtools.WrapTool(tool)

		// 注册到工具注册表
		am.toolReg.Register(wrappedTool)
		count++
		
		logger.Debug(am.ctx, "Registered executor tool (wrapped): %s", meta.Name)
	}

	return nil
}

// buildInputSchemaFromMetadata 从工具元数据构建输入 schema
func buildInputSchemaFromMetadata(meta executor.ToolMetadata) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	for _, param := range meta.Parameters {
		prop := map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}
		properties[param.Name] = prop

		if param.Required {
			required = append(required, param.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
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
	am.currentLLMConfig = config
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
				toolCall := &ToolCall{
					ToolName: getStringFromMap(tc, "tool_name"),
					Status:   getStringFromMap(tc, "status"),
					Message:  getStringFromMap(tc, "message"),
					Instructions: getStringFromMap(tc, "instructions"),
					Result:  getStringFromMap(tc, "result"),
				}
				
				// 加载 arguments
				if args, ok := tc["arguments"].(map[string]interface{}); ok {
					toolCall.Arguments = args
				}
				
				// 加载 timestamp
				if tsStr, ok := tc["timestamp"].(string); ok {
					if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
						toolCall.Timestamp = ts
					}
				}
				
				toolCalls = append(toolCalls, toolCall)
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

			// 获取LazyMCP配置
			lazyMCPConfigs, err := am.GetLazyMCPConfigs()
			if err != nil {
				logger.Warn(am.ctx, "Failed to get lazy MCP configs: %v", err)
				lazyMCPConfigs = []agent.LazyMCPConfig{}
			}

			ag, err := agent.NewAgent(
				agent.WithLLM(am.llmClient),
				agent.WithMemory(mem),
				agent.WithTools(am.toolReg.List()...),
				agent.WithLazyMCPConfigs(lazyMCPConfigs),
				agent.WithSystemPrompt(am.GetSystemPrompt()),
				agent.WithRequirePlanApproval(false),
				agent.WithMaxIterations(maxIterations), // 增加最大迭代次数
				agent.WithLogger(NewAgentLogger()),
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

		// 获取LazyMCP配置
		lazyMCPConfigs, err := am.GetLazyMCPConfigs()
		if err != nil {
			logger.Warn(am.ctx, "Failed to get lazy MCP configs: %v", err)
			lazyMCPConfigs = []agent.LazyMCPConfig{}
		}

		// 创建 Agent - 禁用执行计划审批
		ag, err := agent.NewAgent(
			agent.WithLLM(am.llmClient),
			agent.WithMemory(mem),
			agent.WithTools(tools...),
			agent.WithLazyMCPConfigs(lazyMCPConfigs),
			agent.WithSystemPrompt(am.GetSystemPrompt()),
			agent.WithRequirePlanApproval(false),   // 禁用执行计划审批
			agent.WithMaxIterations(maxIterations), // 增加最大迭代次数，避免过早触发 final call
			agent.WithLogger(NewAgentLogger()),
		)
		if err != nil {
			logger.Warn(am.ctx, "Failed to create Agent for session %s: %v", session.ID, err)
		} else {
			am.agents[session.ID] = ag
			logger.Info(am.ctx, "✓ Created Agent for session %s, tools count: %d, lazy MCP services: %d", session.ID, len(tools), len(lazyMCPConfigs))
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
func (am *AgentManager) SendMessage(ctx context.Context, sessionID, userMessage string, streamChan chan<- StreamChunk) error {
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
	agentCtx := multitenancy.WithOrgID(ctx, "browserwing")
	agentCtx = context.WithValue(agentCtx, memory.ConversationIDKey, sessionID)

	// 使用 Agent 流式处理消息
	streamEvents, err := ag.RunStream(agentCtx, userMessage)
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

	for {
		select {
		case <-ctx.Done():
			// 客户端取消请求，停止处理
			logger.Info(ctx, "Request cancelled by client, stopping message processing")
			return ctx.Err()
		case event, ok := <-streamEvents:
			if !ok {
				// 流式事件通道已关闭，处理完成
				goto processingComplete
			}

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
				// 工具执行结果
				if event.ToolCall == nil {
					logger.Error(ctx, "Tool result event missing ToolCall information")
					continue
				}
				tc := event.ToolCall
				toolCall, exists := toolCallMap[tc.Name]
				if !exists {
					logger.Warn(ctx, "[ToolResult Event] Tool call not found in map: %s", tc.Name)
					continue
				}

				// 详细日志：查看事件的完整结构
				logger.Info(ctx, "[ToolResult Event] Tool: %s, Status: %s, Result length: %d", 
					tc.Name, tc.Status, len(tc.Result))
				logger.Info(ctx, "[ToolResult Event] ToolCall details - ID: %s, Arguments: %s, Result: %s", 
					tc.ID, tc.Arguments, tc.Result)
				logger.Info(ctx, "[ToolResult Event] Event.Content: %s", event.Content)
				if event.Metadata != nil {
					logger.Info(ctx, "[ToolResult Event] Event.Metadata: %+v", event.Metadata)
				}

				// 尝试从多个地方获取执行结果
				resultData := tc.Result
				if resultData == "" && event.Content != "" {
					resultData = event.Content
					logger.Info(ctx, "[ToolResult Event] Using event.Content as result")
				}
				if resultData == "" && event.Metadata != nil {
					if result, ok := event.Metadata["result"].(string); ok && result != "" {
						resultData = result
						logger.Info(ctx, "[ToolResult Event] Using metadata.result as result")
					}
				}
				// 最后尝试从全局存储中获取（工具包装器会保存结果）
				if resultData == "" {
					storedResult := toolResultStore.GetResult(tc.Name)
					if storedResult != "" {
						resultData = storedResult
						logger.Info(ctx, "[ToolResult Event] Using GlobalToolResultStore result (length: %d)", len(resultData))
					}
				}

				// 更新工具调用状态和结果
				switch tc.Status {
				case "executing":
					toolCall.Status = "calling"
					toolCall.Message = "执行中..."
				case "completed":
					toolCall.Status = "success"
					toolCall.Message = "调用成功"
					// 保存执行结果
					if resultData != "" {
						toolCall.Result = resultData
						logger.Info(ctx, "[ToolResult Event] Saved result (first 200 chars): %s", 
							resultData[:min(200, len(resultData))])
					} else {
						logger.Warn(ctx, "[ToolResult Event] Result is empty!")
					}
				case "error":
					toolCall.Status = "error"
					toolCall.Message = "调用失败"
					// 保存错误信息
					if resultData != "" {
						toolCall.Result = resultData
						logger.Info(ctx, "[ToolResult Event] Saved error result: %s", resultData)
					}
				}

				// 发送工具调用状态
				streamChan <- StreamChunk{
					Type:     "tool_call",
					ToolCall: toolCall,
				}

			case interfaces.AgentEventToolCall:
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
						ToolName:  tc.Name,
						Status:    "calling",
						Timestamp: time.Now(),
						Arguments: make(map[string]interface{}),
					}
					
					logger.Info(ctx, "[ToolCall Event] Tool: %s, Arguments JSON: %s", tc.Name, tc.Arguments)
					
					// 提取 instructions 和其他参数
					if tc.Arguments != "" {
						// 解析参数 JSON
						var args map[string]interface{}
						if err := json.Unmarshal([]byte(tc.Arguments), &args); err == nil {
							logger.Info(ctx, "[ToolCall Event] Parsed args: %+v", args)
							
							// 提取 instructions
							if instructions, ok := args["instructions"].(string); ok {
								toolCall.Instructions = instructions
								logger.Info(ctx, "[ToolCall Event] Found instructions: %s", instructions)
								// 从参数中移除 instructions，保留实际的工具参数
								delete(args, "instructions")
							} else {
								logger.Warn(ctx, "[ToolCall Event] No instructions found in args")
							}
							toolCall.Arguments = args
							logger.Info(ctx, "[ToolCall Event] Final toolCall - Instructions: %s, Args: %+v", 
								toolCall.Instructions, toolCall.Arguments)
						} else {
							logger.Error(ctx, "[ToolCall Event] Failed to parse arguments JSON: %v", err)
						}
					} else {
						logger.Warn(ctx, "[ToolCall Event] Arguments is empty")
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
	}

processingComplete:
	// 保存助手消息
	am.mu.Lock()
	session.Messages = append(session.Messages, assistantMsg)
	session.UpdatedAt = time.Now()
	am.mu.Unlock()

	// 保存助手消息到数据库
	var toolCallsData []map[string]interface{}
	for _, tc := range assistantMsg.ToolCalls {
		logger.Info(ctx, "Saving tool call to DB: name=%s, status=%s, instructions=%s, args=%+v, result_len=%d", 
			tc.ToolName, tc.Status, tc.Instructions, tc.Arguments, len(tc.Result))
		
		toolCallsData = append(toolCallsData, map[string]interface{}{
			"tool_name":    tc.ToolName,
			"status":       tc.Status,
			"message":      tc.Message,
			"instructions": tc.Instructions,
			"arguments":    tc.Arguments,
			"result":       tc.Result,
			"timestamp":    tc.Timestamp.Format(time.RFC3339),
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

type AgentLogger struct {
	logger logger.Logger
}

func NewAgentLogger() *AgentLogger {
	return &AgentLogger{
		logger: logger.GetDefaultLogger(),
	}
}

func (al *AgentLogger) fieldsToString(fields map[string]interface{}) string {
	fieldStr := ""
	for k, v := range fields {
		fieldStr += fmt.Sprintf("%s=%v ", k, v)
	}
	return fieldStr
}

func (al *AgentLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	al.logger.Info(ctx, "%s %s", msg, al.fieldsToString(fields))
}

func (al *AgentLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	al.logger.Warn(ctx, "%s %s", msg, al.fieldsToString(fields))
}

func (al *AgentLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	al.logger.Error(ctx, "%s %s", msg, al.fieldsToString(fields))
}

func (al *AgentLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	al.logger.Debug(ctx, "%s %s", msg, al.fieldsToString(fields))
}

// ReloadMCPServices 重新加载MCP服务配置
func (am *AgentManager) ReloadMCPServices() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// 重新初始化工具注册表
	am.toolReg = tools.NewRegistry()
	if err := am.initMCPTools(); err != nil {
		return fmt.Errorf("failed to init MCP tools: %w", err)
	}

	logger.Info(am.ctx, "✓ MCP services reloaded successfully")

	// Note: 现有会话的Agent实例不会自动更新
	// 新会话将自动使用最新的工具列表

	return nil
}

// GetLazyMCPConfigs 获取LazyMCP配置列表（用于Agent SDK）
func (am *AgentManager) GetLazyMCPConfigs() ([]agent.LazyMCPConfig, error) {
	// 从数据库加载MCP服务配置
	services, err := am.db.ListMCPServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP services: %w", err)
	}

	var lazyConfigs []agent.LazyMCPConfig
	for _, service := range services {
		if !service.Enabled {
			continue
		}

		// 构建LazyMCPConfig
		config := agent.LazyMCPConfig{
			Name: service.Name,
			Type: string(service.Type),
		}

		switch service.Type {
		case models.MCPServiceTypeStdio:
			config.Command = service.Command
			config.Args = service.Args
			// 转换环境变量格式 map[string]string -> []string
			if len(service.Env) > 0 {
				envSlice := make([]string, 0, len(service.Env))
				for k, v := range service.Env {
					envSlice = append(envSlice, k+"="+v)
				}
				config.Env = envSlice
			}
		case models.MCPServiceTypeSSE, models.MCPServiceTypeHTTP:
			// 支持SSE和HTTP类型的MCP服务
			if service.URL == "" {
				logger.Warn(am.ctx, "MCP service %s missing URL, skipping", service.Name)
				continue
			}
			config.URL = service.URL
		}
		// 从数据库加载该服务的工具配置
		tools, err := am.db.GetMCPServiceTools(service.ID)
		if err != nil {
			logger.Warn(am.ctx, "Failed to load tools for MCP service %s: %v", service.Name, err)
			continue
		}

		// 转换工具配置
		var toolConfigs []agent.LazyMCPToolConfig
		for _, tool := range tools {
			if !tool.Enabled {
				continue
			}
			toolConfigs = append(toolConfigs, agent.LazyMCPToolConfig{
				Name:        tool.Name,
				Description: tool.Description,
				Schema:      tool.Schema,
			})
		}
		config.Tools = toolConfigs

		// 只有当有工具时才添加配置
		if len(toolConfigs) > 0 {
			lazyConfigs = append(lazyConfigs, config)
		}
	}

	return lazyConfigs, nil
}
