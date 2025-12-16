package models

import "time"

// AgentSession Agent 聊天会话
type AgentSession struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentMessage Agent 聊天消息
type AgentMessage struct {
	ID        string                   `json:"id"`
	SessionID string                   `json:"session_id"`
	Role      string                   `json:"role"` // user, assistant, system
	Content   string                   `json:"content"`
	Timestamp time.Time                `json:"timestamp"`
	ToolCalls []map[string]interface{} `json:"tool_calls,omitempty"` // 工具调用信息
}
