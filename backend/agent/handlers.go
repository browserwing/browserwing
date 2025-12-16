package agent

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Handler Agent HTTP 处理器
type Handler struct {
	manager *AgentManager
}

// NewHandler 创建 Agent 处理器
func NewHandler(manager *AgentManager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// CreateSession 创建新会话
func (h *Handler) CreateSession(c *gin.Context) {
	session := h.manager.CreateSession()

	c.JSON(http.StatusOK, gin.H{
		"session": session,
	})
}

// GetSession 获取会话
func (h *Handler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")

	session, err := h.manager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session": session,
	})
}

// ListSessions 列出所有会话
func (h *Handler) ListSessions(c *gin.Context) {
	sessions := h.manager.ListSessions()

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// DeleteSession 删除会话
func (h *Handler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")

	if err := h.manager.DeleteSession(sessionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "agent.sessionDeleted",
	})
}

// SendMessage 发送消息 (SSE 流式响应)
func (h *Handler) SendMessage(c *gin.Context) {
	sessionID := c.Param("id")

	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.messageEmpty"})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // 禁用 nginx 缓冲

	// 创建流式通道
	streamChan := make(chan StreamChunk, 10)

	// 获取 ResponseWriter
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.streamingNotSupported"})
		return
	}

	// 在后台处理消息
	go func() {
		if err := h.manager.SendMessage(sessionID, req.Message, streamChan); err != nil {
			logger.Warn(c.Request.Context(), "Failed to send message: %v", err)
		}
	}()

	// 发送流式数据
	for chunk := range streamChan {
		data, err := json.Marshal(chunk)
		if err != nil {
			logger.Warn(c.Request.Context(), "Failed to serialize data chunk: %v", err)
			continue
		}

		// SSE 格式: data: {json}\n\n
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
	}
}

// SetLLMConfig 设置 LLM 配置
func (h *Handler) SetLLMConfig(c *gin.Context) {
	var req struct {
		ConfigID string `json:"config_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.configIdEmpty"})
		return
	}

	// 从数据库获取 LLM 配置
	config, err := h.manager.db.GetLLMConfig(req.ConfigID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.configNotFound"})
		return
	}

	// 设置 LLM 配置
	if err := h.manager.SetLLMConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "agent.llmConfigSet",
		"config":  GetProviderInfo(config),
	})
}

// ReloadLLM 重新加载 LLM 配置 (用于配置更新后的热加载)
func (h *Handler) ReloadLLM(c *gin.Context) {
	if err := h.manager.ReloadLLM(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "agent.llmConfigReloaded",
	})
}

// GetMCPStatus 获取 MCP 状态
func (h *Handler) GetMCPStatus(c *gin.Context) {
	status := h.manager.GetMCPStatus()

	c.JSON(http.StatusOK, status)
}
