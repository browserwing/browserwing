package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/browserwing/browserwing/agent"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/storage"
	"github.com/go-rod/rod"
)

// ScriptPlayer 脚本播放器接口
type ScriptPlayer interface {
	PlayScript(scriptID string, variables map[string]string, instanceID string) (*models.PlayResult, error)
}

// AgentExecutor Agent 执行器接口
type AgentExecutor interface {
	ExecuteAgentTask(ctx context.Context, sessionID, llmID, prompt string) (string, error)
}

// DefaultTaskExecutor 默认任务执行器
type DefaultTaskExecutor struct {
	db            *storage.BoltDB
	scriptPlayer  ScriptPlayer
	agentExecutor AgentExecutor
}

// NewDefaultTaskExecutor 创建默认任务执行器
func NewDefaultTaskExecutor(db *storage.BoltDB, scriptPlayer ScriptPlayer, agentExecutor AgentExecutor) *DefaultTaskExecutor {
	return &DefaultTaskExecutor{
		db:            db,
		scriptPlayer:  scriptPlayer,
		agentExecutor: agentExecutor,
	}
}

// ExecuteScript 执行脚本任务
func (e *DefaultTaskExecutor) ExecuteScript(ctx context.Context, task *models.ScheduledTask) (map[string]interface{}, error) {
	if task.ScriptID == "" {
		return nil, fmt.Errorf("script ID is empty")
	}

	log.Printf("[TaskExecutor] Executing script task: %s (script: %s)", task.Name, task.ScriptID)

	// 执行脚本
	result, err := e.scriptPlayer.PlayScript(task.ScriptID, task.ScriptVariables, task.BrowserInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	if !result.Success {
		return result.ExtractedData, fmt.Errorf("script execution failed: %s", result.Message)
	}

	return result.ExtractedData, nil
}

// ExecuteAgent 执行 Agent 任务
func (e *DefaultTaskExecutor) ExecuteAgent(ctx context.Context, task *models.ScheduledTask) (map[string]interface{}, error) {
	if task.AgentPrompt == "" {
		return nil, fmt.Errorf("agent prompt is empty")
	}

	log.Printf("[TaskExecutor] Executing agent task: %s", task.Name)

	// 如果没有指定会话 ID，创建一个临时会话
	sessionID := task.AgentSessionID
	if sessionID == "" {
		// 使用任务 ID 作为会话标识
		sessionID = fmt.Sprintf("task_%s", task.ID)
	}

	// 执行 Agent 任务
	response, err := e.agentExecutor.ExecuteAgentTask(ctx, sessionID, task.AgentLLMID, task.AgentPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to execute agent task: %w", err)
	}

	// 将响应转换为 map
	result := map[string]interface{}{
		"response": response,
		"prompt":   task.AgentPrompt,
	}

	return result, nil
}

// BrowserManagerInterface 浏览器管理器接口（避免循环依赖）
// 注意：这里我们不定义完整接口，而是在 adapter 中处理类型转换
type BrowserManagerInterface interface {
	IsRunning() bool
	Start(ctx context.Context) error
}

// AgentManagerInterface Agent 管理器接口（避免循环依赖）
type AgentManagerInterface interface{}

// RealScriptPlayer 真实脚本播放器（使用浏览器管理器）
type RealScriptPlayer struct {
	db             *storage.BoltDB
	browserManager interface{} // 使用 interface{} 避免循环依赖
}

// NewRealScriptPlayer 创建真实脚本播放器
func NewRealScriptPlayer(db *storage.BoltDB, browserManager interface{}) *RealScriptPlayer {
	return &RealScriptPlayer{
		db:             db,
		browserManager: browserManager,
	}
}

// PlayScript 播放脚本
func (p *RealScriptPlayer) PlayScript(scriptID string, variables map[string]string, instanceID string) (result *models.PlayResult, err error) {
	// 添加 recover 捕获 panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[RealScriptPlayer] Panic recovered: %v", r)
			err = fmt.Errorf("script execution panicked: %v", r)
			result = nil
		}
	}()

	ctx := context.Background()

	// 获取脚本
	script, err := p.db.GetScript(scriptID)
	if err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	log.Printf("[RealScriptPlayer] Playing script: %s (ID: %s)", script.Name, scriptID)

	// 类型断言获取 browserManager（使用接口定义避免循环依赖）
	type browserMgr interface {
		IsRunning() bool
		Start(ctx context.Context) error
		PlayScript(ctx context.Context, script *models.Script, instanceID string) (*models.PlayResult, *rod.Page, error)
		CloseActivePage(ctx context.Context, page *rod.Page) error
	}

	bm, ok := p.browserManager.(browserMgr)
	if !ok {
		// 记录详细错误信息帮助调试
		log.Printf("[RealScriptPlayer] ERROR: Browser manager type assertion failed. Type: %T", p.browserManager)
		return nil, fmt.Errorf("invalid browser manager type: %T", p.browserManager)
	}

	// 确保浏览器正在运行
	if !bm.IsRunning() {
		log.Printf("[RealScriptPlayer] Browser not running, starting...")
		if err := bm.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start browser: %w", err)
		}
	}

	// 创建脚本副本并替换变量
	scriptToRun := script.Copy()

	// 合并参数：先使用脚本预设变量，再用外部传入的参数覆盖
	mergedParams := make(map[string]string)
	if scriptToRun.Variables != nil {
		for key, value := range scriptToRun.Variables {
			mergedParams[key] = value
		}
	}
	for key, value := range variables {
		mergedParams[key] = value
	}

	// 替换占位符
	if len(mergedParams) > 0 {
		scriptToRun.URL = replacePlaceholders(scriptToRun.URL, mergedParams)
		scriptToRun.Actions = make([]models.ScriptAction, len(script.Actions))
		copy(scriptToRun.Actions, script.Actions)

		for i := range scriptToRun.Actions {
			scriptToRun.Actions[i].Selector = replacePlaceholders(scriptToRun.Actions[i].Selector, mergedParams)
			scriptToRun.Actions[i].XPath = replacePlaceholders(scriptToRun.Actions[i].XPath, mergedParams)
			scriptToRun.Actions[i].Value = replacePlaceholders(scriptToRun.Actions[i].Value, mergedParams)
			scriptToRun.Actions[i].URL = replacePlaceholders(scriptToRun.Actions[i].URL, mergedParams)
			scriptToRun.Actions[i].JSCode = replacePlaceholders(scriptToRun.Actions[i].JSCode, mergedParams)

			if len(scriptToRun.Actions[i].FilePaths) > 0 {
				newFilePaths := make([]string, len(scriptToRun.Actions[i].FilePaths))
				for j, path := range scriptToRun.Actions[i].FilePaths {
					newFilePaths[j] = replacePlaceholders(path, mergedParams)
				}
				scriptToRun.Actions[i].FilePaths = newFilePaths
			}
		}
	}

	// 执行脚本
	result, page, err := bm.PlayScript(ctx, scriptToRun, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	// 关闭页面
	if err := bm.CloseActivePage(ctx, page); err != nil {
		return nil, fmt.Errorf("failed to close page: %w", err)
	}

	return result, nil
}

// replacePlaceholders 替换占位符 ${xxx}
func replacePlaceholders(text string, params map[string]string) string {
	if text == "" || len(params) == 0 {
		return text
	}

	result := text
	for key, value := range params {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// RealAgentExecutor 真实 Agent 执行器（使用 Agent 管理器）
type RealAgentExecutor struct {
	agentManager interface{} // 使用 interface{} 避免循环依赖
}

// NewRealAgentExecutor 创建真实 Agent 执行器
func NewRealAgentExecutor(agentManager interface{}) *RealAgentExecutor {
	return &RealAgentExecutor{
		agentManager: agentManager,
	}
}

// ExecuteAgentTask 执行 Agent 任务
func (e *RealAgentExecutor) ExecuteAgentTask(ctx context.Context, sessionID, llmID, prompt string) (result string, err error) {
	// 添加 recover 捕获 panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[RealAgentExecutor] Panic recovered: %v", r)
			err = fmt.Errorf("agent execution panicked: %v", r)
		}
	}()

	log.Printf("[RealAgentExecutor] Executing agent task with prompt: %s", prompt)

	// 类型断言获取 agentManager（使用接口定义避免循环依赖）
	type agentMgr interface {
		GetSession(sessionID string) (*agent.ChatSession, error)
		CreateSession(llmConfigID string) *agent.ChatSession
		SendMessage(ctx context.Context, sessionID, userMessage string, streamChan chan<- agent.StreamChunk) error
	}
	
	am, ok := e.agentManager.(agentMgr)
	if !ok {
		// 记录详细错误信息帮助调试
		log.Printf("[RealAgentExecutor] ERROR: Agent manager type assertion failed. Type: %T", e.agentManager)
		return "", fmt.Errorf("invalid agent manager type: %T", e.agentManager)
	}

	// 检查会话是否存在，如果不存在则创建
	session, err := am.GetSession(sessionID)
	if err != nil {
		// 会话不存在，创建新会话
		log.Printf("[RealAgentExecutor] Session %s not found, creating new session with LLM: %s", sessionID, llmID)
		session = am.CreateSession(llmID)
		sessionID = session.ID  // 使用新创建的会话 ID
		log.Printf("[RealAgentExecutor] Created new session with ID: %s", sessionID)
	}

	// 创建一个通道来接收流式响应（但我们不需要流式，只要最终结果）
	streamChan := make(chan agent.StreamChunk, 100)
	
	// 在 goroutine 中收集响应
	responseChan := make(chan string, 1)
	go func() {
		var responseBuilder strings.Builder
		for chunk := range streamChan {
			// 从 StreamChunk 中提取内容
			if chunk.Type == "message" && chunk.Content != "" {
				responseBuilder.WriteString(chunk.Content)
			}
		}
		responseChan <- responseBuilder.String()
	}()

	// 发送消息（注意：SendMessage 内部会 defer close(streamChan)，所以这里不需要关闭）
	err = am.SendMessage(ctx, sessionID, prompt, streamChan)
	if err != nil {
		return "", fmt.Errorf("failed to send message to agent: %w", err)
	}

	// 等待响应收集完成
	response := <-responseChan
	log.Printf("[RealAgentExecutor] Agent response received (length: %d)", len(response))
	
	return response, nil
}

// SimplScriptPlayer 简单脚本播放器（用于测试或简化场景）
type SimpleScriptPlayer struct {
	db *storage.BoltDB
}

// NewSimpleScriptPlayer 创建简单脚本播放器
func NewSimpleScriptPlayer(db *storage.BoltDB) *SimpleScriptPlayer {
	return &SimpleScriptPlayer{db: db}
}

// PlayScript 播放脚本
func (p *SimpleScriptPlayer) PlayScript(scriptID string, variables map[string]string, instanceID string) (*models.PlayResult, error) {
	// 这是一个简化的实现，仅用于测试
	script, err := p.db.GetScript(scriptID)
	if err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	log.Printf("[SimpleScriptPlayer] Playing script: %s (ID: %s)", script.Name, scriptID)

	// 返回一个模拟的成功结果
	return &models.PlayResult{
		Success: true,
		Message: fmt.Sprintf("Script '%s' executed successfully (mock)", script.Name),
		ExtractedData: map[string]interface{}{
			"script_id":   scriptID,
			"script_name": script.Name,
			"variables":   variables,
		},
	}, nil
}

// SimpleAgentExecutor 简单 Agent 执行器（用于测试或简化场景）
type SimpleAgentExecutor struct{}

// NewSimpleAgentExecutor 创建简单 Agent 执行器
func NewSimpleAgentExecutor() *SimpleAgentExecutor {
	return &SimpleAgentExecutor{}
}

// ExecuteAgentTask 执行 Agent 任务
func (e *SimpleAgentExecutor) ExecuteAgentTask(ctx context.Context, sessionID, llmID, prompt string) (string, error) {
	log.Printf("[SimpleAgentExecutor] Executing agent task with prompt: %s", prompt)

	// 这是一个模拟实现
	response := fmt.Sprintf("Agent response for prompt: %s (mock)", prompt)
	return response, nil
}

// MarshalResultToMap 将结果转换为 map
func MarshalResultToMap(result interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(data, &resultMap); err != nil {
		return nil, err
	}

	return resultMap, nil
}
