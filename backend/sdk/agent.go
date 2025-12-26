package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/browserwing/browserwing/agent"
	"github.com/browserwing/browserwing/models"
)

// AgentClient Agent 客户端
type AgentClient struct {
	client *Client
}

// AgentSession Agent 会话
type AgentSession struct {
	ID        string          `json:"id"`
	Messages  []AgentMessage  `json:"messages"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}

// AgentMessage Agent 消息
type AgentMessage struct {
	ID        string     `json:"id"`
	Role      string     `json:"role"` // user, assistant, system
	Content   string     `json:"content"`
	Timestamp int64      `json:"timestamp"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall 工具调用信息
type ToolCall struct {
	ToolName string `json:"tool_name"`
	Status   string `json:"status"` // calling, success, error
	Message  string `json:"message,omitempty"`
}

// MessageChunk 流式消息块
type MessageChunk struct {
	Type      string    `json:"type"` // message, tool_call, done, error
	Content   string    `json:"content,omitempty"`
	ToolCall  *ToolCall `json:"tool_call,omitempty"`
	Error     string    `json:"error,omitempty"`
	MessageID string    `json:"message_id,omitempty"`
}

// CreateSession 创建新的 Agent 会话
func (ac *AgentClient) CreateSession(ctx context.Context) (string, error) {
	if ac.client.agentManager == nil {
		return "", fmt.Errorf("agent manager not initialized")
	}

	session := ac.client.agentManager.CreateSession()
	if session == nil {
		return "", fmt.Errorf("failed to create session")
	}

	return session.ID, nil
}

// GetSession 获取会话信息
func (ac *AgentClient) GetSession(ctx context.Context, sessionID string) (*AgentSession, error) {
	if ac.client.agentManager == nil {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	session, err := ac.client.agentManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// 转换为 SDK 模型
	messages := make([]AgentMessage, 0, len(session.Messages))
	for _, msg := range session.Messages {
		toolCalls := make([]ToolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ToolName: tc.ToolName,
				Status:   tc.Status,
				Message:  tc.Message,
			})
		}

		messages = append(messages, AgentMessage{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp.Unix(),
			ToolCalls: toolCalls,
		})
	}

	return &AgentSession{
		ID:        session.ID,
		Messages:  messages,
		CreatedAt: session.CreatedAt.Unix(),
		UpdatedAt: session.UpdatedAt.Unix(),
	}, nil
}

// ListSessions 列出所有会话
func (ac *AgentClient) ListSessions(ctx context.Context) ([]*AgentSession, error) {
	if ac.client.agentManager == nil {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	sessions := ac.client.agentManager.ListSessions()
	
	// 转换为 SDK 模型
	result := make([]*AgentSession, 0, len(sessions))
	for _, session := range sessions {
		messages := make([]AgentMessage, 0, len(session.Messages))
		for _, msg := range session.Messages {
			toolCalls := make([]ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, ToolCall{
					ToolName: tc.ToolName,
					Status:   tc.Status,
					Message:  tc.Message,
				})
			}

			messages = append(messages, AgentMessage{
				ID:        msg.ID,
				Role:      msg.Role,
				Content:   msg.Content,
				Timestamp: msg.Timestamp.Unix(),
				ToolCalls: toolCalls,
			})
		}

		result = append(result, &AgentSession{
			ID:        session.ID,
			Messages:  messages,
			CreatedAt: session.CreatedAt.Unix(),
			UpdatedAt: session.UpdatedAt.Unix(),
		})
	}

	return result, nil
}

// DeleteSession 删除会话
func (ac *AgentClient) DeleteSession(ctx context.Context, sessionID string) error {
	if ac.client.agentManager == nil {
		return fmt.Errorf("agent manager not initialized")
	}

	if err := ac.client.agentManager.DeleteSession(sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// SendMessage 发送消息(非流式)
// 返回完整的回复内容
func (ac *AgentClient) SendMessage(ctx context.Context, sessionID, message string) (string, error) {
	if ac.client.agentManager == nil {
		return "", fmt.Errorf("agent manager not initialized")
	}

	// 创建通道接收流式响应
	streamChan := make(chan agent.StreamChunk, 10)
	
	// 启动发送消息的 goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- ac.client.agentManager.SendMessage(ctx, sessionID, message, streamChan)
	}()

	// 收集所有消息块
	var fullResponse string
	var lastError string

	for chunk := range streamChan {
		switch chunk.Type {
		case "message":
			fullResponse += chunk.Content
		case "error":
			lastError = chunk.Error
		case "done":
			// 流式传输完成
			break
		}
	}

	// 等待 goroutine 完成
	if err := <-errChan; err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	if lastError != "" {
		return "", fmt.Errorf("agent error: %s", lastError)
	}

	return fullResponse, nil
}

// SendMessageStream 发送消息(流式)
// callback 函数会被多次调用,每次传递一个消息块
func (ac *AgentClient) SendMessageStream(ctx context.Context, sessionID, message string, callback func(*MessageChunk)) error {
	if ac.client.agentManager == nil {
		return fmt.Errorf("agent manager not initialized")
	}

	// 创建通道接收流式响应
	streamChan := make(chan agent.StreamChunk, 10)
	
	// 启动发送消息的 goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- ac.client.agentManager.SendMessage(ctx, sessionID, message, streamChan)
	}()

	// 处理流式响应
	for chunk := range streamChan {
		var toolCall *ToolCall
		if chunk.ToolCall != nil {
			toolCall = &ToolCall{
				ToolName: chunk.ToolCall.ToolName,
				Status:   chunk.ToolCall.Status,
				Message:  chunk.ToolCall.Message,
			}
		}

		callback(&MessageChunk{
			Type:      chunk.Type,
			Content:   chunk.Content,
			ToolCall:  toolCall,
			Error:     chunk.Error,
			MessageID: chunk.MessageID,
		})

		// 如果是错误或完成,提前返回
		if chunk.Type == "error" || chunk.Type == "done" {
			break
		}
	}

	// 等待 goroutine 完成
	if err := <-errChan; err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// SendMessageStreamReader 发送消息(流式, io.Reader 接口)
// 返回一个 io.ReadCloser,可以像读取文件一样读取流式响应
func (ac *AgentClient) SendMessageStreamReader(ctx context.Context, sessionID, message string) (io.ReadCloser, error) {
	if ac.client.agentManager == nil {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	// 创建 pipe
	pr, pw := io.Pipe()

	// 启动后台处理
	go func() {
		defer pw.Close()

		// 创建通道接收流式响应
		streamChan := make(chan agent.StreamChunk, 10)
		
		// 启动发送消息的 goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- ac.client.agentManager.SendMessage(ctx, sessionID, message, streamChan)
		}()

		// 将流式响应写入 pipe
		encoder := json.NewEncoder(pw)
		for chunk := range streamChan {
			var toolCall *ToolCall
			if chunk.ToolCall != nil {
				toolCall = &ToolCall{
					ToolName: chunk.ToolCall.ToolName,
					Status:   chunk.ToolCall.Status,
					Message:  chunk.ToolCall.Message,
				}
			}

			msgChunk := &MessageChunk{
				Type:      chunk.Type,
				Content:   chunk.Content,
				ToolCall:  toolCall,
				Error:     chunk.Error,
				MessageID: chunk.MessageID,
			}

			if err := encoder.Encode(msgChunk); err != nil {
				pw.CloseWithError(fmt.Errorf("failed to encode chunk: %w", err))
				return
			}

			// 如果是错误或完成,提前返回
			if chunk.Type == "error" || chunk.Type == "done" {
				break
			}
		}

		// 等待 goroutine 完成
		if err := <-errChan; err != nil {
			pw.CloseWithError(fmt.Errorf("failed to send message: %w", err))
		}
	}()

	return pr, nil
}

// SetLLMConfig 设置 LLM 配置
func (ac *AgentClient) SetLLMConfig(ctx context.Context, config *LLMConfig) error {
	if ac.client.agentManager == nil {
		return fmt.Errorf("agent manager not initialized")
	}

	// 转换为内部模型
	llmConfig := &models.LLMConfigModel{
		Provider: config.Provider,
		APIKey:   config.APIKey,
		Model:    config.Model,
		BaseURL:  config.BaseURL,
	}

	if err := ac.client.agentManager.SetLLMConfig(llmConfig); err != nil {
		return fmt.Errorf("failed to set LLM config: %w", err)
	}

	return nil
}

// GetLLMConfig 获取当前 LLM 配置
func (ac *AgentClient) GetLLMConfig(ctx context.Context) (*LLMConfig, error) {
	if ac.client.agentManager == nil {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	// 注意: AgentManager 不提供 GetCurrentLLMConfig 方法
	// 这里返回配置中的 LLM 配置
	if ac.client.config.LLMConfig == nil {
		return nil, fmt.Errorf("no LLM config set")
	}

	return ac.client.config.LLMConfig, nil
}
