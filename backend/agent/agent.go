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

const (
	maxIterationsSimple  = 3  // 简单任务的最大迭代次数
	maxIterationsMedium  = 7  // 中等任务的最大迭代次数
	maxIterationsComplex = 12 // 复杂任务的最大迭代次数
	maxIterationsEval    = 1  // 任务评估的最大迭代次数
)

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

Always prefer using tools over making up information. If you have a tool that can help, use it. Respond in the same language as the user's message.`
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
	ID          string        `json:"id"`
	LLMConfigID string        `json:"llm_config_id"` // 会话使用的LLM配置ID
	Messages    []ChatMessage `json:"messages"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
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
	var cancel context.CancelFunc
	if strings.HasPrefix(t.name, "browser_") {
		// 检查原始 context 是否已经 done
		select {
		case <-ctx.Done():
			logger.Warn(ctx, "Original context already done for tool %s: %v", t.name, ctx.Err())
			// 如果原始 context 已经取消，使用 Background 创建新的独立 context
			execCtx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
		default:
			// 原始 context 正常，从它派生
			execCtx, cancel = context.WithTimeout(ctx, 120*time.Second)
		}
		defer cancel()
		logger.Info(ctx, "Using extended timeout (120s) for browser tool: %s", t.name)
	}

	// 调用 MCP 服务器执行脚本
	result, err := t.mcpServer.CallTool(execCtx, t.name, args)
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool: %w", err)
	}

	// 处理返回结果，统一处理 data 字段
	var responseText string
	if resultMap, ok := result.(map[string]interface{}); ok {
		logger.Info(ctx, "[Agent MCPTool] Result map keys: %v", getMapKeys(resultMap))

		// 获取 message 字段作为主要响应
		if message, ok := resultMap["message"].(string); ok {
			responseText = message
			logger.Info(ctx, "[Agent MCPTool] Got message: %s", message)
		}

		// 检查并处理 data 字段
		if data, ok := resultMap["data"].(map[string]interface{}); ok {
			logger.Info(ctx, "[Agent MCPTool] Found data field with keys: %v", getMapKeys(data))

			// 特殊处理 semantic_tree（直接追加文本）
			if semanticTree, ok := data["semantic_tree"].(string); ok && semanticTree != "" {
				responseText += "\n\nSemantic Tree:\n" + semanticTree
				logger.Info(ctx, "Added semantic_tree to response for tool: %s (tree length: %d)", t.name, len(semanticTree))
			} else {
				// 其他数据类型（extract结果、page info等）序列化为 JSON
				if len(data) > 0 {
					dataJSON, err := json.MarshalIndent(data, "", "  ")
					if err == nil {
						responseText += "\n\nData:\n" + string(dataJSON)
						logger.Info(ctx, "[Agent MCPTool] Added data to response for tool: %s (data keys: %v)", t.name, getMapKeys(data))
					}
				}
			}
		} else {
			logger.Info(ctx, "[Agent MCPTool] No data field found in result")
		}
	} else {
		logger.Info(ctx, "[Agent MCPTool] Result is not a map, type: %T", result)
	}

	// 如果没有提取到文本响应，回退到 JSON 序列化
	if responseText == "" {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to serialize result: %w", err)
		}
		return string(resultJSON), nil
	}

	return responseText, nil
}

// getMapKeys 获取 map 的所有 key（辅助函数，用于日志）
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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

// AgentInstances 存储不同类型的 Agent 实例
type AgentInstances struct {
	SimpleAgent  *agent.Agent   // 简单任务 Agent (maxIterations=3)
	MediumAgent  *agent.Agent   // 中等任务 Agent (maxIterations=7)
	ComplexAgent *agent.Agent   // 复杂任务 Agent (maxIterations=12)
	EvalAgent    *agent.Agent   // 任务评估 Agent (maxIterations=1)
	LLMClient    interfaces.LLM // 会话专用的 LLM client
}

// AgentManager Agent 管理器
type AgentManager struct {
	db               *storage.BoltDB
	mcpServer        browsermcp.IMCPServer
	sessions         map[string]*ChatSession
	agents           map[string]*AgentInstances // sessionID -> Agent 实例集合
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
		agents:    make(map[string]*AgentInstances),
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

// ensureAgentInstances 确保会话的 Agent 实例已创建（按需创建）
func (am *AgentManager) ensureAgentInstances(sessionID, llmConfigID string) (*AgentInstances, error) {
	// 先检查是否已存在
	am.mu.RLock()
	agentInstances, ok := am.agents[sessionID]
	am.mu.RUnlock()

	if ok && agentInstances != nil {
		return agentInstances, nil
	}

	// 不存在，需要创建
	logger.Info(am.ctx, "Creating Agent instances for session %s (LLM: %s)", sessionID, llmConfigID)

	// 创建 LLM client（根据会话的 LLMConfigID）
	var llmClient interfaces.LLM
	if llmConfigID != "" {
		// 使用会话指定的 LLM 配置
		config, err := am.db.GetLLMConfig(llmConfigID)
		if err != nil {
			logger.Warn(am.ctx, "Failed to get LLM config %s: %v, using default", llmConfigID, err)
			// 如果配置不存在，使用当前的默认配置
			llmClient = am.llmClient
		} else {
			// 创建专门的 LLM client
			llmClient, err = CreateLLMClient(config)
			if err != nil {
				return nil, fmt.Errorf("failed to create LLM client from config %s: %w", llmConfigID, err)
			}
			logger.Info(am.ctx, "✓ Created LLM client for session %s: %s (%s)", sessionID, config.Model, config.Provider)
		}
	} else {
		// 旧会话，没有指定 LLM，使用当前的默认配置
		logger.Info(am.ctx, "Session %s has no LLM config, using default", sessionID)
		llmClient = am.llmClient
	}

	if llmClient == nil {
		return nil, fmt.Errorf("LLM client is not available")
	}

	// 创建 Agent 实例
	agentInstances, err := am.createAgentInstances(llmClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent instances: %w", err)
	}

	// 保存到 map
	am.mu.Lock()
	am.agents[sessionID] = agentInstances
	am.mu.Unlock()

	tools := am.toolReg.List()
	logger.Info(am.ctx, "✓ Created Agent instances for session %s on demand (simple: %d, medium: %d, complex: %d, eval: %d), tools: %d",
		sessionID, maxIterationsSimple, maxIterationsMedium, maxIterationsComplex, maxIterationsEval, len(tools))

	return agentInstances, nil
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
					ToolName:     getStringFromMap(tc, "tool_name"),
					Status:       getStringFromMap(tc, "status"),
					Message:      getStringFromMap(tc, "message"),
					Instructions: getStringFromMap(tc, "instructions"),
					Result:       getStringFromMap(tc, "result"),
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
			ID:          dbSession.ID,
			LLMConfigID: dbSession.LLMConfigID, // 从数据库加载 LLM 配置 ID
			Messages:    messages,
			CreatedAt:   dbSession.CreatedAt,
			UpdatedAt:   dbSession.UpdatedAt,
		}

		am.sessions[session.ID] = session

		// ✅ 不再提前创建 Agent 实例，改为按需创建（lazy load）
		logger.Info(am.ctx, "Loaded session %s with %d messages (LLM: %s)", session.ID, len(messages), session.LLMConfigID)
	}

	return nil
}

// createAgentInstance 创建指定 maxIterations 的 Agent 实例（使用指定的 LLM client）
func (am *AgentManager) createAgentInstance(llmClient interfaces.LLM, maxIter int) (*agent.Agent, error) {
	mem := memory.NewConversationBuffer()

	// 获取LazyMCP配置
	lazyMCPConfigs, err := am.GetLazyMCPConfigs()
	if err != nil {
		logger.Warn(am.ctx, "Failed to get lazy MCP configs: %v", err)
		lazyMCPConfigs = []agent.LazyMCPConfig{}
	}

	ag, err := agent.NewAgent(
		agent.WithLLM(llmClient),
		agent.WithMemory(mem),
		agent.WithTools(am.toolReg.List()...),
		agent.WithLazyMCPConfigs(lazyMCPConfigs),
		agent.WithSystemPrompt(am.GetSystemPrompt()),
		agent.WithRequirePlanApproval(false),
		agent.WithMaxIterations(maxIter),
		agent.WithLogger(NewAgentLogger()),
	)
	if err != nil {
		return nil, err
	}

	return ag, nil
}

// createEvalAgent 创建评估 Agent（不带任何工具）
func (am *AgentManager) createEvalAgent(llmClient interfaces.LLM) (*agent.Agent, error) {
	mem := memory.NewConversationBuffer()

	// ⚠️ 评估 Agent 不需要任何工具，只用于评估任务复杂度
	ag, err := agent.NewAgent(
		agent.WithLLM(llmClient),
		agent.WithMemory(mem),
		// ✅ 不传入任何工具
		agent.WithSystemPrompt("You are a task evaluation assistant. Your ONLY job is to analyze user requests and classify them. DO NOT call any tools, DO NOT perform any actions, ONLY return the evaluation JSON."),
		agent.WithRequirePlanApproval(false),
		agent.WithMaxIterations(1), // 评估只需要1次
		agent.WithLogger(NewAgentLogger()),
	)
	if err != nil {
		return nil, err
	}

	return ag, nil
}

// createAgentInstances 为会话创建所有类型的 Agent 实例（使用指定的 LLM client）
func (am *AgentManager) createAgentInstances(llmClient interfaces.LLM) (*AgentInstances, error) {
	// 创建简单任务 Agent
	simpleAgent, err := am.createAgentInstance(llmClient, maxIterationsSimple)
	if err != nil {
		return nil, fmt.Errorf("failed to create simple agent: %w", err)
	}

	// 创建中等任务 Agent
	mediumAgent, err := am.createAgentInstance(llmClient, maxIterationsMedium)
	if err != nil {
		return nil, fmt.Errorf("failed to create medium agent: %w", err)
	}

	// 创建复杂任务 Agent
	complexAgent, err := am.createAgentInstance(llmClient, maxIterationsComplex)
	if err != nil {
		return nil, fmt.Errorf("failed to create complex agent: %w", err)
	}

	// 创建任务评估 Agent（不带工具）
	evalAgent, err := am.createEvalAgent(llmClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create eval agent: %w", err)
	}

	return &AgentInstances{
		SimpleAgent:  simpleAgent,
		MediumAgent:  mediumAgent,
		ComplexAgent: complexAgent,
		EvalAgent:    evalAgent,
		LLMClient:    llmClient, // 保存 LLM client 引用
	}, nil
}

// CreateSession 创建新会话
func (am *AgentManager) CreateSession(llmConfigID string) *ChatSession {
	am.mu.Lock()
	defer am.mu.Unlock()

	session := &ChatSession{
		ID:          uuid.New().String(),
		LLMConfigID: llmConfigID,
		Messages:    []ChatMessage{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	am.sessions[session.ID] = session

	// 保存到数据库
	dbSession := &models.AgentSession{
		ID:          session.ID,
		LLMConfigID: llmConfigID,
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   session.UpdatedAt,
	}
	if err := am.db.SaveAgentSession(dbSession); err != nil {
		logger.Warn(am.ctx, "Failed to save session to database: %v", err)
	}

	// ✅ 不再提前创建 Agent 实例，改为在 SendMessage 时按需创建
	logger.Info(am.ctx, "✓ Created session %s (LLM: %s), Agent instances will be created on demand",
		session.ID, llmConfigID)

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

const (
	ComplexModeSimple  = "simple"
	ComplexModeMedium  = "medium"
	ComplexModeComplex = "complex"
)

// TaskComplexity 任务复杂度评估结果
type TaskComplexity struct {
	NeedTools      bool   `json:"need_tools"`                // 是否需要使用工具
	ComplexMode    string `json:"complex_mode"`              // simple, medium, complex, none
	Reasoning      string `json:"reasoning"`                 // 评估理由
	Confidence     string `json:"confidence"`                // 置信度: high, medium, low
	Explanation    string `json:"explanation"`               // 对用户的解释
	DirectResponse string `json:"direct_response,omitempty"` // ✨ 如果不需要工具，直接包含完整回复内容
}

// generateGreeting 生成友好的开场白回复
func (am *AgentManager) generateGreeting(ctx context.Context, sessionID, userMessage string, agentInstances *AgentInstances) (string, error) {
	if agentInstances == nil || agentInstances.EvalAgent == nil {
		// 如果 Agent 不可用，返回默认的开场白
		return "Got it, let me help you with that.", nil
	}

	// 构建生成开场白的提示词
	greetingPrompt := fmt.Sprintf(`Generate a brief, friendly greeting response for the user's request. The greeting should:
1. Acknowledge their request
2. Show understanding of what they want
3. Be warm and professional
4. Be brief (1-2 sentences max)
5. Indicate you're about to help them
6. IMPORTANT: Respond in the SAME LANGUAGE as the user's request

User request: "%s"

Examples of good greetings (match the language):
- For Chinese: 收到，我将帮您查询今天的GitHub热门项目。
- For Chinese: 好的，让我来分析这个网站的性能数据。
- For English: Got it, I'll help you find today's trending GitHub projects.
- For English: Sure, let me analyze the website performance data for you.

Generate ONLY the greeting text (no JSON, no explanation), and respond in the same language as the user's request.`, userMessage)

	// 创建评估上下文
	greetingCtx := multitenancy.WithOrgID(ctx, "browserwing")
	greetingCtx = context.WithValue(greetingCtx, memory.ConversationIDKey, sessionID+"_greeting")

	// 使用评估 Agent 生成开场白
	greeting, err := agentInstances.EvalAgent.Run(greetingCtx, greetingPrompt)
	if err != nil {
		logger.Warn(ctx, "[Greeting] Failed to generate greeting: %v, using default", err)
		return "Got it, let me help you with that.", nil
	}

	// 清理可能的多余空白和换行
	greeting = strings.TrimSpace(greeting)

	logger.Info(ctx, "[Greeting] Generated greeting: %s", greeting)

	return greeting, nil
}

// evaluateTaskComplexity 评估任务复杂度
func (am *AgentManager) evaluateTaskComplexity(ctx context.Context, sessionID, userMessage string) (*TaskComplexity, error) {
	am.mu.RLock()
	agentInstances, ok := am.agents[sessionID]
	am.mu.RUnlock()

	if !ok || agentInstances == nil || agentInstances.EvalAgent == nil {
		return nil, fmt.Errorf("eval agent for session %s is not initialized", sessionID)
	}

	// 构建评估提示词
	evalPrompt := fmt.Sprintf(`Analyze the following user request and determine:
1. Does it need to use tools (browser automation, web search, calculations, etc.)?
2. If yes, estimate the number of tool calls and classify complexity

User request: "%s"

**STEP 1: Determine if tools are needed**

NO TOOLS NEEDED (need_tools: false):
- Greetings, casual chat, small talk
- General knowledge questions that LLM can answer directly
- Asking for explanations, definitions, or advice
- Examples:
  * "Hi" / "Hello" / "你好" → Just greeting
  * "What is AI?" → LLM knowledge
  * "How do I learn programming?" → LLM advice
  * "Tell me a joke" → LLM generation
  * "What's the capital of France?" → LLM knowledge

TOOLS NEEDED (need_tools: true):
- Real-time information (weather, news, stock prices)
- Web browsing, clicking, form filling
- Searching the web
- Calculations, data processing
- Examples:
  * "Search for today's trending GitHub repositories" → need web_search
  * "Open Baidu and search for AI news" → need browser automation
  * "What's the weather now?" → need real-time data

**STEP 2: If tools needed, classify complexity**

**SIMPLE (1-3 tool calls):**
- Single tool call tasks
- Direct web searches or calculations
- Examples:
  * "Search for trending GitHub repos" → 1 call (web_search)
  * "Calculate 123 * 456" → 1 call (calculate)

**MEDIUM (4-7 tool calls):**
- Browser automation with multiple steps
- Examples:
  * "Open Baidu, search for 'AI news', click first result" → 4-5 calls
  * "Fill a simple form and submit" → 4-6 calls

**COMPLEX (8+ tool calls):**
- Multi-page workflows with data processing
- Examples:
  * "Compare prices across 3 sites" → 12+ calls
  * "Automate complete registration flow" → 10+ calls

Response format (JSON only, no explanation, no markdown):
{
  "need_tools": true/false,
  "complex_mode": "simple/medium/complex",
  "reasoning": "Brief explanation",
  "confidence": "high/medium/low",
  "explanation": "Short user-friendly explanation in same language as user",
  "direct_response": "REQUIRED if need_tools is false: Complete answer"
}

**IMPORTANT:**
- If need_tools is false:
  * Set complex_mode to "none"
  * YOU MUST include "direct_response" with the complete answer
  * The "direct_response" should be natural and in the same language as user

- If need_tools is true:
  * Set complex_mode appropriately (simple/medium/complex)
  * DO NOT include "direct_response"`, userMessage)

	// 创建评估上下文
	evalCtx := multitenancy.WithOrgID(ctx, "browserwing")
	evalCtx = context.WithValue(evalCtx, memory.ConversationIDKey, sessionID+"_eval")

	logger.Info(ctx, "[TaskEval] Evaluating task complexity for message: %s", userMessage)

	// 使用评估 Agent
	response, err := agentInstances.EvalAgent.Run(evalCtx, evalPrompt)
	if err != nil {
		logger.Warn(ctx, "[TaskEval] Failed to evaluate task complexity: %v, defaulting to no tools", err)
		return &TaskComplexity{
			NeedTools:   false, // ✅ 评估失败时默认不使用工具
			ComplexMode: "none",
			Reasoning:   "Evaluation failed, defaulting to direct response",
			Confidence:  "low",
			Explanation: "评估失败，直接回复",
		}, nil
	}

	logger.Info(ctx, "[TaskEval] Raw response: %s", response)

	// 🔍 调试：输出原始响应的前 200 个字符
	if len(response) > 200 {
		logger.Info(ctx, "[TaskEval] Raw response preview (first 200 chars): %s...", response[:200])
	}

	response = strings.TrimSpace(response)
	// 移除 ```json 和 ``` 标签
	response = strings.ReplaceAll(response, "```json", "")
	response = strings.ReplaceAll(response, "```", "")
	response = strings.TrimSpace(response)

	// 🔍 调试：输出清理后的响应
	logger.Info(ctx, "[TaskEval] Cleaned response: %s", response)
	if response == "" {
		logger.Warn(ctx, "[TaskEval] Empty response, defaulting to no tools")
		return &TaskComplexity{
			NeedTools:   false, // ✅ 空响应时默认不使用工具
			ComplexMode: "none",
			Reasoning:   "Empty response, defaulting to direct response",
			Confidence:  "low",
			Explanation: "评估结果为空，直接回复",
		}, nil
	}

	// 解析 JSON 响应
	var complexity TaskComplexity
	if err := json.Unmarshal([]byte(response), &complexity); err != nil {
		logger.Warn(ctx, "[TaskEval] Failed to parse JSON response: %v", err)
		logger.Warn(ctx, "[TaskEval] Response content: %s", response)
		logger.Warn(ctx, "[TaskEval] Defaulting to no tools")
		return &TaskComplexity{
			NeedTools:   false, // ✅ 解析失败时默认不使用工具
			ComplexMode: "none",
			Reasoning:   "Failed to parse evaluation result",
			Confidence:  "low",
			Explanation: "评估结果解析失败，直接回复",
		}, nil
	}

	// 🔍 调试：检查解析后的值
	logger.Info(ctx, "[TaskEval] Parsed result: NeedTools=%v, ComplexMode='%s', Reasoning='%s'",
		complexity.NeedTools,
		complexity.ComplexMode,
		complexity.Reasoning)

	// 🔍 验证：检查必需字段是否为空
	if complexity.ComplexMode == "" && complexity.Reasoning == "" {
		logger.Warn(ctx, "[TaskEval] ⚠️ Warning: All fields are empty after parsing! Using default.")
		logger.Warn(ctx, "[TaskEval] This usually means LLM returned wrong format.")
		return &TaskComplexity{
			NeedTools:   false,
			ComplexMode: "none",
			Reasoning:   "LLM returned invalid format",
			Confidence:  "low",
			Explanation: "评估返回格式错误，直接回复",
		}, nil
	}

	logger.Info(ctx, "[TaskEval] Task evaluated as %s (confidence: %s): %s",
		complexity.ComplexMode,
		complexity.Confidence,
		complexity.Reasoning)

	// 🔍 调试日志：输出完整的评估结果
	logger.Info(ctx, "[TaskEval] ✓ Evaluation result: NeedTools=%v, ComplexMode=%s, Confidence=%s",
		complexity.NeedTools,
		complexity.ComplexMode,
		complexity.Confidence)

	return &complexity, nil
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

	// 保存用户消息到数据库（跳过AI控制临时会话）
	if len(sessionID) < 11 || sessionID[:11] != "ai_control_" {
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
	}

	// 确保 Agent 实例已创建（按需创建）
	agentInstances, err := am.ensureAgentInstances(sessionID, session.LLMConfigID)
	if err != nil {
		streamChan <- StreamChunk{
			Type:  "error",
			Error: fmt.Sprintf("Failed to create Agent instances: %v", err),
		}
		return fmt.Errorf("failed to create agent instances: %w", err)
	}

	// 创建主要的助手消息（用于工具调用和最终回复）
	assistantMsg := ChatMessage{
		ID:        uuid.New().String(),
		Role:      "assistant",
		Content:   "",
		Timestamp: time.Now(),
		ToolCalls: []*ToolCall{},
	}

	// 发送新的消息 ID
	streamChan <- StreamChunk{
		Type:      "message",
		Content:   "",
		MessageID: assistantMsg.ID,
	}

	// 评估任务复杂度（在后台进行）
	complexity, err := am.evaluateTaskComplexity(ctx, sessionID, userMessage)
	if err != nil {
		logger.Warn(ctx, "Failed to evaluate task complexity: %v, using direct response", err)
		complexity = &TaskComplexity{
			NeedTools:   false, // ✅ 评估错误时默认不使用工具
			ComplexMode: "none",
			Reasoning:   "Evaluation error, defaulting to direct response",
			Confidence:  "low",
			Explanation: "评估失败，直接回复",
		}
	}

	// 🔍 调试日志：输出评估结果和判断逻辑
	logger.Info(ctx, "[SendMessage] Complexity evaluation: NeedTools=%v, ComplexMode=%s, Message='%s'",
		complexity.NeedTools,
		complexity.ComplexMode,
		userMessage)

	// 如果不需要工具，直接用 LLM 生成回复
	if !complexity.NeedTools {
		logger.Info(ctx, "[SendMessage] ✓ Taking direct response path (no tools needed)")
		logger.Info(ctx, "[DirectLLM] Task doesn't need tools: %s", complexity.Reasoning)

		// ✨ 优化：如果评估结果中包含直接回复，直接使用，无需再调用 LLM
		if complexity.DirectResponse != "" {
			logger.Info(ctx, "[DirectLLM] ⚡ Using direct response from evaluation (1 LLM call): %d chars", len(complexity.DirectResponse))

			// 将回复内容分段流式发送（模拟流式效果，提升用户体验）
			assistantMsg.Content = complexity.DirectResponse
			chunkSize := 20 // 每次发送 20 个字符
			for i := 0; i < len(complexity.DirectResponse); i += chunkSize {
				end := i + chunkSize
				if end > len(complexity.DirectResponse) {
					end = len(complexity.DirectResponse)
				}
				chunk := complexity.DirectResponse[i:end]
				streamChan <- StreamChunk{
					Type:      "message",
					Content:   chunk,
					MessageID: assistantMsg.ID,
				}
				// 小延迟，模拟自然的打字效果
				time.Sleep(10 * time.Millisecond)
			}

			logger.Info(ctx, "[DirectLLM] ✓ Direct response completed (from evaluation)")

		} else {
			// 降级：如果没有直接回复，使用 SimpleAgent 生成（2次 LLM 调用）
			logger.Warn(ctx, "[DirectLLM] No direct response in evaluation, falling back to SimpleAgent (2 LLM calls)")

			// 创建多租户上下文
			directCtx := multitenancy.WithOrgID(ctx, "browserwing")
			directCtx = context.WithValue(directCtx, memory.ConversationIDKey, sessionID)

			// 使用 SimpleAgent 的流式运行
			streamEvents, err := agentInstances.SimpleAgent.RunStream(directCtx, userMessage)
			if err != nil {
				logger.Warn(ctx, "Direct response failed: %v, falling back to agent with tools", err)
				complexity.NeedTools = true // 降级到使用带工具的 agent
				goto needTools              // 跳转到需要工具的流程
			}

			// 处理流式事件
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
				case interfaces.AgentEventError:
					// 错误
					logger.Warn(ctx, "Direct response error: %s", event.Content)
					streamChan <- StreamChunk{
						Type:  "error",
						Error: event.Content,
					}
					return fmt.Errorf("direct response error: %s", event.Content)
				case interfaces.AgentEventComplete:
					// 完成
					logger.Info(ctx, "[DirectLLM] ✓ Direct response completed: %d chars", len(assistantMsg.Content))
				}
			}
		}

		// 完成消息
		streamChan <- StreamChunk{
			Type:      "done",
			MessageID: assistantMsg.ID,
		}

		// 保存助手消息
		am.mu.Lock()
		session.Messages = append(session.Messages, assistantMsg)
		session.UpdatedAt = time.Now()
		am.mu.Unlock()

		// 保存到数据库（跳过AI控制临时会话）
		if len(sessionID) < 11 || sessionID[:11] != "ai_control_" {
			dbAssistantMsg := &models.AgentMessage{
				ID:        assistantMsg.ID,
				SessionID: sessionID,
				Role:      assistantMsg.Role,
				Content:   assistantMsg.Content,
				Timestamp: assistantMsg.Timestamp,
				ToolCalls: []map[string]interface{}{},
			}
			if err := am.db.SaveAgentMessage(dbAssistantMsg); err != nil {
				logger.Warn(am.ctx, "Failed to save assistant message to database: %v", err)
			}
		}

		return nil
	}

needTools:
	// 需要工具的流程标签

	// 需要工具，根据评估结果选择合适的 Agent
	logger.Info(ctx, "[SendMessage] ✓ Taking agent path (tools needed)")

	var ag *agent.Agent
	switch complexity.ComplexMode {
	case ComplexModeComplex:
		ag = agentInstances.ComplexAgent
		logger.Info(ctx, "Using COMPLEX agent (max iterations: %d) for task: %s", maxIterationsComplex, complexity.Reasoning)
	case ComplexModeMedium:
		ag = agentInstances.MediumAgent
		logger.Info(ctx, "Using MEDIUM agent (max iterations: %d) for task: %s", maxIterationsMedium, complexity.Reasoning)
	default:
		ag = agentInstances.SimpleAgent
		logger.Info(ctx, "Using SIMPLE agent (max iterations: %d) for task: %s", maxIterationsSimple, complexity.Reasoning)
	}

	if ag == nil {
		streamChan <- StreamChunk{
			Type:  "error",
			Error: fmt.Sprintf("Selected agent for session %s is not initialized", sessionID),
		}
		return fmt.Errorf("selected agent for session %s is not initialized", sessionID)
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
	// 保存助手消息到数据库（跳过AI控制临时会话）
	if len(sessionID) < 11 || sessionID[:11] != "ai_control_" {
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
			ID:          sessionID,
			LLMConfigID: session.LLMConfigID, // ✅ 保留 LLMConfigID
			CreatedAt:   session.CreatedAt,
			UpdatedAt:   session.UpdatedAt,
		}
		if err := am.db.SaveAgentSession(dbSession); err != nil {
			logger.Warn(am.ctx, "Failed to update session timestamp: %v", err)
		}
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
		// 过滤掉脚本回放时创建的AI控制临时会话
		// 这些会话ID以"ai_control_"开头，不应该显示在前端
		if len(session.ID) >= 11 && session.ID[:11] == "ai_control_" {
			continue
		}
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

// SendMessageGeneric 是 SendMessage 的泛型包装版本
// 用于适配外部接口，接收 chan<- any 并转换为内部使用的 chan<- StreamChunk
func (am *AgentManager) SendMessageGeneric(ctx context.Context, sessionID, userMessage string, streamChan chan<- any) error {
	// 创建一个内部的 StreamChunk 通道
	internalChan := make(chan StreamChunk, 100)
	
	// 启动一个 goroutine 来转换通道类型
	go func() {
		defer close(streamChan)
		for chunk := range internalChan {
			streamChan <- chunk
		}
	}()
	
	// 调用原始的 SendMessage 方法
	return am.SendMessage(ctx, sessionID, userMessage, internalChan)
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
