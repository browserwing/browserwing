package agent

import (
	"context"
	"strings"
	"time"

	"github.com/browserwing/browserwing/pkg/logger"
)

// SendMessageInterface 是为了适配外部接口而添加的方法
// 接收 chan<- any 并转换为内部使用的 chan<- StreamChunk
func (am *AgentManager) SendMessageInterface(ctx context.Context, sessionID, userMessage string, streamChan chan<- any) error {
	// 如果是AI控制临时会话（以ai_control_开头），先创建临时会话
	if strings.HasPrefix(sessionID, "ai_control_") {
		// 检查会话是否已存在
		if _, err := am.GetSession(sessionID); err != nil {
			// 会话不存在，创建临时会话
			am.mu.Lock()
			session := &ChatSession{
				ID:          sessionID,
				LLMConfigID: "", // 使用默认配置
				Messages:    []ChatMessage{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			am.sessions[sessionID] = session
			am.mu.Unlock()
			
			logger.Info(ctx, "Created temporary AI control session: %s (will not be saved to database)", sessionID)
			
			// 延迟清理：10分钟后自动删除临时会话
			go func() {
				time.Sleep(10 * time.Minute)
				am.mu.Lock()
				delete(am.sessions, sessionID)
				am.mu.Unlock()
				logger.Info(ctx, "Cleaned up temporary AI control session: %s", sessionID)
			}()
		}
	}
	
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
