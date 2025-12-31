package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdkagent "github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/mcp"
	"github.com/browserwing/browserwing/agent"
	localtools "github.com/browserwing/browserwing/agent/tools"
	"github.com/browserwing/browserwing/config"
	"github.com/browserwing/browserwing/llm"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/services/browser"
	"github.com/browserwing/browserwing/storage"
	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod/lib/proto"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/server"
)

// 使用类型断言访问 MCP 服务器的 ServeHTTP 方法
type MCPHTTPHandler interface {
	GetStatus() map[string]interface{}
	RegisterScript(*models.Script) error
	UnregisterScript(string)
	ServeSteamableHTTP(http.ResponseWriter, *http.Request)
	GetSSEServer() *server.SSEServer
}

type Handler struct {
	db             *storage.BoltDB
	browserManager *browser.Manager
	config         *config.Config
	llmManager     *llm.Manager
	mcpServer      MCPHTTPHandler // MCP 服务器（使用 interface{} 避免循环依赖）
	agentManager   interface{}    // Agent 管理器（用于 LLM 配置更新后的热加载）
}

func NewHandler(
	db *storage.BoltDB,
	browserMgr *browser.Manager,
	cfg *config.Config,
	llmMgr *llm.Manager,
) *Handler {
	return &Handler{
		db:             db,
		browserManager: browserMgr,
		config:         cfg,
		llmManager:     llmMgr,
		mcpServer:      nil, // 将在主程序中设置
	}
}

// ============= 浏览器控制相关 API =============

// StartBrowser 启动浏览器
func (h *Handler) StartBrowser(c *gin.Context) {
	if h.browserManager.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.browserAlreadyRunning"})
		return
	}

	if err := h.browserManager.Start(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.startBrowserFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.browserStarted",
		"status":  h.browserManager.Status(),
	})
}

// StopBrowser 停止浏览器
func (h *Handler) StopBrowser(c *gin.Context) {
	if !h.browserManager.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.browserNotRunning"})
		return
	}

	// 获取当前浏览器的所有 Cookie
	cookies, err := h.browserManager.GetCurrentPageCookies()
	if err != nil {
		logger.Error(c.Request.Context(), "Failed to get current page cookies: %v", err)
	} else {
		// 保存到数据库，使用固定 ID "browser"
		cookieStore := &models.CookieStore{
			ID:       "browser",
			Platform: "browser",
			Cookies:  cookies.([]*proto.NetworkCookie),
		}

		if err := h.db.SaveCookies(cookieStore); err != nil {
			logger.Error(c.Request.Context(), "Failed to save cookies: %v", err)
		} else {
			logger.Info(c.Request.Context(), "Saved %d cookies", len(cookieStore.Cookies))
		}
	}

	if err := h.browserManager.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.stopBrowserFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.browserStopped",
	})
}

// BrowserStatus 获取浏览器状态
func (h *Handler) BrowserStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.browserManager.Status())
}

// OpenBrowserPage 在浏览器中打开页面
func (h *Handler) OpenBrowserPage(c *gin.Context) {
	var req struct {
		URL      string `json:"url" binding:"required"`
		Language string `json:"language"` // 前端当前语言
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	if !h.browserManager.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.browserNotRunning"})
		return
	}

	if err := h.browserManager.OpenPage(req.URL, req.Language); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.openPageFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.pageOpened",
		"url":     req.URL,
	})
}

// SaveBrowserCookies 保存浏览器Cookie
func (h *Handler) SaveBrowserCookies(c *gin.Context) {
	if !h.browserManager.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.browserNotRunning"})
		return
	}

	// 获取当前浏览器的所有 Cookie
	cookies, err := h.browserManager.GetCurrentPageCookies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getCookiesFailed"})
		return
	}

	// 保存到数据库，使用固定 ID "browser"
	cookieStore := &models.CookieStore{
		ID:       "browser",
		Platform: "browser",
		Cookies:  cookies.([]*proto.NetworkCookie),
	}

	if err := h.db.SaveCookies(cookieStore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.saveCookiesFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.cookiesSaved",
		"count":   len(cookieStore.Cookies),
	})
}

// ImportBrowserCookies 导入Cookie
func (h *Handler) ImportBrowserCookies(c *gin.Context) {
	var req struct {
		Cookies []map[string]interface{} `json:"cookies"`
		URL     string                   `json:"url"` // 可选，用于日志记录
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	if len(req.Cookies) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 转换为 NetworkCookie 格式
	cookies := make([]*proto.NetworkCookie, 0, len(req.Cookies))
	for _, cookieMap := range req.Cookies {
		cookie := &proto.NetworkCookie{}

		// 解析必需字段
		if name, ok := cookieMap["name"].(string); ok {
			cookie.Name = name
		}
		if value, ok := cookieMap["value"].(string); ok {
			cookie.Value = value
		}
		if domain, ok := cookieMap["domain"].(string); ok {
			cookie.Domain = domain
		}
		if path, ok := cookieMap["path"].(string); ok {
			cookie.Path = path
		} else {
			cookie.Path = "/"
		}

		// 解析可选字段
		if secure, ok := cookieMap["secure"].(bool); ok {
			cookie.Secure = secure
		}
		if httpOnly, ok := cookieMap["httpOnly"].(bool); ok {
			cookie.HTTPOnly = httpOnly
		}
		if sameSite, ok := cookieMap["sameSite"].(string); ok {
			cookie.SameSite = proto.NetworkCookieSameSite(sameSite)
		}
		if expires, ok := cookieMap["expires"].(float64); ok {
			cookie.Expires = proto.TimeSinceEpoch(expires)
		}

		// 只添加有效的 Cookie（至少有 name 和 value）
		if cookie.Name != "" && cookie.Value != "" {
			cookies = append(cookies, cookie)
		}
	}

	if len(cookies) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.noValidCookies"})
		return
	}

	// 保存到数据库
	cookieStore := &models.CookieStore{
		ID:       "browser",
		Platform: "browser",
		Cookies:  cookies,
	}

	if err := h.db.SaveCookies(cookieStore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.saveCookiesFailed"})
		return
	}

	if req.URL != "" {
		logger.Info(c.Request.Context(), "Imported %d cookies (target URL: %s)", len(cookies), req.URL)
	} else {
		logger.Info(c.Request.Context(), "Imported %d cookies", len(cookies))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.cookiesImported",
		"count":   len(cookies),
	})
}

// GetCookies 获取保存的 Cookie
func (h *Handler) GetCookies(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	cookieStore, err := h.db.GetCookies(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.cookieNotFound"})
		return
	}

	c.JSON(http.StatusOK, cookieStore)
}

// ============= 脚本录制和回放相关 API =============

// StartRecording 开始录制操作
func (h *Handler) StartRecording(c *gin.Context) {
	if !h.browserManager.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.browserNotRunning"})
		return
	}

	if err := h.browserManager.StartRecording(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.startRecordingFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.recordingStarted"})
}

// StopRecording 停止录制
func (h *Handler) StopRecording(c *gin.Context) {
	actions, downloadedFiles, err := h.browserManager.StopRecording(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.stopRecordingFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "success.recordingStopped",
		"actions":          actions,
		"count":            len(actions),
		"downloaded_files": downloadedFiles,
	})
}

// GetRecordingStatus 获取录制状态
func (h *Handler) GetRecordingStatus(c *gin.Context) {
	info := h.browserManager.GetRecordingInfo()
	c.JSON(http.StatusOK, info)
}

// ClearInPageRecordingState 清除页面内录制状态
func (h *Handler) ClearInPageRecordingState(c *gin.Context) {
	h.browserManager.ClearInPageRecordingState()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SaveScript 保存脚本
func (h *Handler) SaveScript(c *gin.Context) {
	var req struct {
		ID                    string                  `json:"id"` // 可选，更新时使用
		Name                  string                  `json:"name" binding:"required"`
		Description           string                  `json:"description"`
		URL                   string                  `json:"url" binding:"required"`
		Actions               []models.ScriptAction   `json:"actions" binding:"required"`
		DownloadedFiles       []models.DownloadedFile `json:"downloaded_files"` // 下载的文件列表
		Tags                  []string                `json:"tags"`
		IsMCPCommand          *bool                   `json:"is_mcp_command"`
		MCPCommandName        string                  `json:"mcp_command_name"`
		MCPCommandDescription string                  `json:"mcp_command_description"`
		MCPInputSchema        map[string]interface{}  `json:"mcp_input_schema"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 计算录制时长
	var duration int64
	if len(req.Actions) > 0 {
		duration = req.Actions[len(req.Actions)-1].Timestamp - req.Actions[0].Timestamp
	}

	id := req.ID
	if id == "" {
		id = uuid.New().String()
	}

	script := &models.Script{
		ID:              id,
		Name:            req.Name,
		Description:     req.Description,
		URL:             req.URL,
		Actions:         req.Actions,
		DownloadedFiles: req.DownloadedFiles, // 保存下载文件信息
		Tags:            req.Tags,
		Duration:        duration,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 如果提供了 MCP 相关字段，则设置
	if req.IsMCPCommand != nil {
		script.IsMCPCommand = *req.IsMCPCommand
	}
	if req.MCPCommandName != "" {
		script.MCPCommandName = req.MCPCommandName
	}
	if req.MCPCommandDescription != "" {
		script.MCPCommandDescription = req.MCPCommandDescription
	}
	if req.MCPInputSchema != nil {
		script.MCPInputSchema = req.MCPInputSchema
	}

	if err := h.db.SaveScript(script); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.saveScriptFailed"})
		return
	}

	// 同步 MCP 注册状态
	h.syncMCPRegistration(c, script)

	c.JSON(http.StatusOK, gin.H{
		"message": "success.scriptSaved",
		"script":  script,
	})
}

// ListScripts 列出所有脚本（支持分页和过滤）
func (h *Handler) ListScripts(c *gin.Context) {
	// 获取分页参数
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := fmt.Sscanf(p, "%d", &page); err == nil && parsed == 1 && page > 0 {
			// page is valid
		} else {
			page = 1
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := fmt.Sscanf(ps, "%d", &pageSize); err == nil && parsed == 1 && pageSize > 0 && pageSize <= 100 {
			// pageSize is valid
		} else {
			pageSize = 20
		}
	}

	// 获取过滤参数
	group := c.Query("group")
	tag := c.Query("tag")

	scripts, err := h.db.ListScripts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get script list: " + err.Error()})
		return
	}

	// 应用过滤
	filteredScripts := make([]*models.Script, 0)
	for _, script := range scripts {
		// 按分组过滤
		if group != "" && script.Group != group {
			continue
		}
		// 按标签过滤
		if tag != "" {
			hasTag := false
			for _, t := range script.Tags {
				if t == tag {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		filteredScripts = append(filteredScripts, script)
	}

	total := len(filteredScripts)

	// 应用分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		filteredScripts = []*models.Script{}
	} else {
		if end > total {
			end = total
		}
		filteredScripts = filteredScripts[start:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"scripts":   filteredScripts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetScript 获取单个脚本详情
func (h *Handler) GetScript(c *gin.Context) {
	id := c.Param("id")
	script, err := h.db.GetScript(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.scriptNotFound"})
		return
	}

	c.JSON(http.StatusOK, script)
}

// UpdateScript 更新脚本
func (h *Handler) UpdateScript(c *gin.Context) {
	id := c.Param("id")

	// 检查脚本是否存在
	script, err := h.db.GetScript(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.scriptNotFound"})
		return
	}

	var req struct {
		Name                  string                 `json:"name"`
		Description           string                 `json:"description"`
		URL                   string                 `json:"url"`
		Actions               []models.ScriptAction  `json:"actions"`
		Tags                  []string               `json:"tags"`
		IsMCPCommand          *bool                  `json:"is_mcp_command"`
		MCPCommandName        *string                `json:"mcp_command_name"`
		MCPCommandDescription *string                `json:"mcp_command_description"`
		MCPInputSchema        map[string]interface{} `json:"mcp_input_schema"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 更新字段
	if req.Name != "" {
		script.Name = req.Name
	}
	if req.Description != "" {
		script.Description = req.Description
	}
	if req.URL != "" {
		script.URL = req.URL
	}
	if req.Actions != nil {
		script.Actions = req.Actions
		// 重新计算时长
		if len(req.Actions) > 0 {
			script.Duration = req.Actions[len(req.Actions)-1].Timestamp - req.Actions[0].Timestamp
		}
	}
	if req.Tags != nil {
		script.Tags = req.Tags
	}

	// 如果提供了 MCP 相关字段，则更新（使用指针类型来区分未提供和提供了false）
	if req.IsMCPCommand != nil {
		script.IsMCPCommand = *req.IsMCPCommand
	}
	if req.MCPCommandName != nil {
		script.MCPCommandName = *req.MCPCommandName
	}
	if req.MCPCommandDescription != nil {
		script.MCPCommandDescription = *req.MCPCommandDescription
	}
	if req.MCPInputSchema != nil {
		script.MCPInputSchema = req.MCPInputSchema
	}

	if err := h.db.UpdateScript(script); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updateScriptFailed"})
		return
	}

	// 同步 MCP 注册状态
	h.syncMCPRegistration(c, script)

	c.JSON(http.StatusOK, gin.H{
		"message": "success.scriptUpdated",
		"script":  script,
	})
}

// DeleteScript 删除脚本
func (h *Handler) DeleteScript(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteScript(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteScriptFailed"})
		return
	}

	if err := h.db.DeleteToolConfigByScriptID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteScriptFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.scriptDeleted"})
}

// PlayScript 回放脚本
func (h *Handler) PlayScript(c *gin.Context) {
	id := c.Param("id")

	// 检查浏览器是否运行
	if !h.browserManager.IsRunning() {
		logger.Info(c, "Browser not running, starting...")
		if err := h.browserManager.Start(c); err != nil {
			logger.Error(c.Request.Context(), "Failed to start browser: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error.playScriptFailed"})
			return
		}
	}

	// 获取脚本
	script, err := h.db.GetScript(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.scriptNotFound"})
		return
	}

	// 解析请求体中的参数
	var req struct {
		Params map[string]string `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有请求体或解析失败,使用空参数
		req.Params = make(map[string]string)
	}

	// 如果提供了参数,创建脚本副本并替换占位符
	scriptToRun := script.Copy()
	if len(req.Params) > 0 {

		// 如果用户提供了 url 参数,使用它;否则替换 URL 中的占位符
		if urlParam, ok := req.Params["url"]; ok && urlParam != "" {
			scriptToRun.URL = urlParam
		} else {
			scriptToRun.URL = replacePlaceholders(scriptToRun.URL, req.Params)
		}

		// 复制 actions 数组以避免修改原始数据
		scriptToRun.Actions = make([]models.ScriptAction, len(script.Actions))
		copy(scriptToRun.Actions, script.Actions)

		// 替换所有 action 中的占位符
		for i := range scriptToRun.Actions {
			scriptToRun.Actions[i].Selector = replacePlaceholders(scriptToRun.Actions[i].Selector, req.Params)
			scriptToRun.Actions[i].XPath = replacePlaceholders(scriptToRun.Actions[i].XPath, req.Params)
			scriptToRun.Actions[i].Value = replacePlaceholders(scriptToRun.Actions[i].Value, req.Params)
			scriptToRun.Actions[i].URL = replacePlaceholders(scriptToRun.Actions[i].URL, req.Params)
			scriptToRun.Actions[i].JSCode = replacePlaceholders(scriptToRun.Actions[i].JSCode, req.Params)

			// 替换文件路径中的占位符
			if len(scriptToRun.Actions[i].FilePaths) > 0 {
				newFilePaths := make([]string, len(scriptToRun.Actions[i].FilePaths))
				for j, path := range scriptToRun.Actions[i].FilePaths {
					newFilePaths[j] = replacePlaceholders(path, req.Params)
				}
				scriptToRun.Actions[i].FilePaths = newFilePaths
			}
		}
	}

	// 执行回放
	result, page, err := h.browserManager.PlayScript(c.Request.Context(), scriptToRun)
	if err != nil {
		logger.Error(c.Request.Context(), "Failed to play script: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.playScriptFailed",
			"result": result,
		})
		return
	}

	// 关闭页面
	if err := h.browserManager.CloseActivePage(c.Request.Context(), page); err != nil {
		logger.Warn(c.Request.Context(), "Failed to close page: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.scriptPlaybackCompleted",
		"script":  script.Name,
		"result":  result,
	})
}

// GetPlayResult 获取上次脚本回放的抓取数据
func (h *Handler) GetPlayResult(c *gin.Context) {
	if !h.browserManager.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.browserNotRunning"})
		return
	}

	scriptID := c.Query("script_id")
	if scriptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.scriptIDRequired"})
		return
	}

	execution, err := h.db.GetLatestScriptExecutionByScriptID(scriptID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.executionNotFound"})
		return
	}

	extractedData := execution.ExtractedData

	c.JSON(http.StatusOK, gin.H{
		"data": extractedData,
	})
}

// ============= LLM 配置管理相关处理器 =============

// ListLLMConfigs 列出所有 LLM 配置
func (h *Handler) ListLLMConfigs(c *gin.Context) {
	configs, err := h.db.ListLLMConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getLLMConfigsFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

// GetLLMConfig 获取单个 LLM 配置
func (h *Handler) GetLLMConfig(c *gin.Context) {
	id := c.Param("id")

	config, err := h.db.GetLLMConfig(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.llmConfigNotFound"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// CreateLLMConfig 创建 LLM 配置
func (h *Handler) CreateLLMConfig(c *gin.Context) {
	var req models.LLMConfigModel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 验证必填字段
	if req.Name == "" || req.Provider == "" || req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.llmConfigRequiredFields"})
		return
	}

	// 使用 name 作为 ID
	req.ID = req.Name
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	// 通过 Manager 添加配置
	if err := h.llmManager.Add(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.createLLMConfigFailed"})
		return
	}

	// 如果是默认配置或启用的配置，通知 Agent 重新加载
	if (req.IsDefault || req.IsActive) && h.agentManager != nil {
		if am, ok := h.agentManager.(interface{ ReloadLLM() error }); ok {
			if err := am.ReloadLLM(); err != nil {
				logger.Warn(c.Request.Context(), "Agent failed to reload LLM: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, req)
}

// UpdateLLMConfig 更新 LLM 配置
func (h *Handler) UpdateLLMConfig(c *gin.Context) {
	id := c.Param("id")

	var req models.LLMConfigModel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 确保 ID 一致
	req.ID = id
	req.UpdatedAt = time.Now()

	// 通过 Manager 更新配置
	if err := h.llmManager.Update(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updateLLMConfigFailed"})
		return
	}

	// 如果是默认配置或启用的配置，通知 Agent 重新加载
	if (req.IsDefault || req.IsActive) && h.agentManager != nil {
		if am, ok := h.agentManager.(interface{ ReloadLLM() error }); ok {
			if err := am.ReloadLLM(); err != nil {
				logger.Warn(c.Request.Context(), "Agent failed to reload LLM: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, req)
}

// DeleteLLMConfig 删除 LLM 配置
func (h *Handler) DeleteLLMConfig(c *gin.Context) {
	id := c.Param("id")

	// 通过 Manager 删除配置
	if err := h.llmManager.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteLLMConfigFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.llmConfigDeleted"})
}

// TestLLMConfig 测试 LLM 配置连接
func (h *Handler) TestLLMConfig(c *gin.Context) {
	var req models.LLMConfigModel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "error.invalidParams",
			"error":   err.Error(),
		})
		return
	}

	// 验证必填字段
	if req.Provider == "" || req.APIKey == "" || req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "error.llmConfigRequiredFields",
		})
		return
	}

	// 创建临时 LLM 客户端进行测试
	client, err := agent.CreateLLMClient(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "llm.messages.testError",
			"error":   err.Error(),
		})
		return
	}

	// 创建 Agent 实例并发送简单的测试消息
	ctx := c.Request.Context()
	testPrompt := "Reply with 'OK' if you can read this message."

	// 使用 agent-sdk-go 的接口进行简单测试
	// 创建一个临时 Agent 进行测试
	ag, err := sdkagent.NewAgent(
		sdkagent.WithLLM(client),
		sdkagent.WithMaxIterations(1),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "llm.messages.testError",
			"error":   "Failed to create agent: " + err.Error(),
		})
		return
	}

	// 使用非流式方法测试
	response, err := ag.Run(ctx, testPrompt)
	if err != nil {
		logger.Error(ctx, "LLM test failed: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "llm.messages.testError",
			"error":   err.Error(),
		})
		return
	}

	logger.Info(ctx, "LLM test successful: %s", response)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "llm.messages.testSuccess",
		"response": response,
	})
}

// ============= 浏览器配置管理相关 API =============

// ListBrowserConfigs 列出所有浏览器配置
func (h *Handler) ListBrowserConfigs(c *gin.Context) {
	configs, err := h.db.ListBrowserConfigs()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"configs": configs,
		"count":   len(configs),
	})
}

// GetBrowserConfig 获取单个浏览器配置
func (h *Handler) GetBrowserConfig(c *gin.Context) {
	id := c.Param("id")

	config, err := h.db.GetBrowserConfig(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "error.configNotFound"})
		return
	}

	c.JSON(200, config)
}

// CreateBrowserConfig 创建浏览器配置
func (h *Handler) CreateBrowserConfig(c *gin.Context) {
	var config models.BrowserConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 生成ID
	config.ID = fmt.Sprintf("config_%d", time.Now().Unix())

	if err := h.db.SaveBrowserConfig(&config); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "browser.config.createSuccess",
		"config":  config,
	})
}

// UpdateBrowserConfig 更新浏览器配置
func (h *Handler) UpdateBrowserConfig(c *gin.Context) {
	id := c.Param("id")

	var config models.BrowserConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	config.ID = id

	if err := h.db.SaveBrowserConfig(&config); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "browser.config.updateSuccess",
		"config":  config,
	})
}

// DeleteBrowserConfig 删除浏览器配置
func (h *Handler) DeleteBrowserConfig(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteBrowserConfig(id); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "browser.config.deleteSuccess"})
}

// ============= MCP 相关 API =============

// SetMCPServer 设置 MCP 服务器实例
func (h *Handler) SetMCPServer(mcpServer MCPHTTPHandler) {
	h.mcpServer = mcpServer
}

// SetAgentManager 设置 Agent 管理器实例
func (h *Handler) SetAgentManager(agentManager interface{}) {
	h.agentManager = agentManager
}

// GenerateMCPConfig 使用 LLM 自动生成 MCP 配置
func (h *Handler) GenerateMCPConfig(c *gin.Context) {
	id := c.Param("id")

	// 获取脚本
	script, err := h.db.GetScript(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.scriptNotFound"})
		return
	}

	// 获取默认的 LLM 配置
	llmConfigs, err := h.db.ListLLMConfigs()
	if err != nil || len(llmConfigs) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No LLM configuration available"})
		return
	}

	// 构建提示词
	actionsJSON, _ := fmt.Sprintf("%+v", script.Actions), ""

	// 调用 LLM
	extractor, err := h.llmManager.GetDefault()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get LLM extractor: " + err.Error()})
		return
	}

	resp, err := extractor.GetMCPInfo(c.Request.Context(), script.Name, script.Description, script.URL, actionsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": resp,
	})
}

// ToggleScriptMCPCommand 设置/取消脚本为 MCP 命令
func (h *Handler) ToggleScriptMCPCommand(c *gin.Context) {
	scriptID := c.Param("id")

	var req struct {
		IsMCPCommand          bool                   `json:"is_mcp_command"`
		MCPCommandName        string                 `json:"mcp_command_name"`
		MCPCommandDescription string                 `json:"mcp_command_description"`
		MCPInputSchema        map[string]interface{} `json:"mcp_input_schema"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "error.invalidParams"})
		return
	}

	// 获取脚本
	script, err := h.db.GetScript(scriptID)
	if err != nil {
		c.JSON(404, gin.H{"error": "error.scriptNotFound"})
		return
	}

	// 如果要启用 MCP 命令，需要验证命令名称
	if req.IsMCPCommand {
		if req.MCPCommandName == "" {
			c.JSON(400, gin.H{"error": "error.mcpCommandNameEmpty"})
			return
		}

		// 检查命令名是否已被其他脚本使用
		scripts, err := h.db.ListScripts()
		if err == nil {
			for _, s := range scripts {
				if s.ID != scriptID && s.IsMCPCommand && s.MCPCommandName == req.MCPCommandName {
					c.JSON(400, gin.H{"error": "error.mcpCommandNameUsed"})
					return
				}
			}
		}
	}

	// 更新脚本
	script.IsMCPCommand = req.IsMCPCommand
	script.MCPCommandName = req.MCPCommandName
	script.MCPCommandDescription = req.MCPCommandDescription
	script.MCPInputSchema = req.MCPInputSchema

	if err := h.db.UpdateScript(script); err != nil {
		c.JSON(500, gin.H{"error": "error.updateScriptFailed"})
		return
	}

	// 同步 MCP 注册状态
	h.syncMCPRegistration(c, script)

	messageKey := "success.mcpCommandDisabled"
	if req.IsMCPCommand {
		messageKey = "success.mcpCommandSet"
	}

	c.JSON(200, gin.H{
		"message": messageKey,
		"script":  script,
	})
}

// GetMCPStatus 获取 MCP 服务状态
func (h *Handler) GetMCPStatus(c *gin.Context) {
	if h.mcpServer == nil {
		c.JSON(200, gin.H{
			"running":       false,
			"commands":      []interface{}{},
			"command_count": 0,
		})
		return
	}

	status := h.mcpServer.GetStatus()
	c.JSON(200, status)
}

// ListMCPCommandsAll 列出所有 MCP 命令
func (h *Handler) ListMCPCommandsAll(c *gin.Context) {
	scripts, err := h.db.ListScripts()
	if err != nil {
		c.JSON(500, gin.H{"error": "error.getScriptListFailed"})
		return
	}

	commands := []map[string]interface{}{}
	for _, script := range scripts {
		if script.IsMCPCommand {
			commands = append(commands, map[string]interface{}{
				"id":          script.ID,
				"name":        script.Name,
				"command":     script.MCPCommandName,
				"description": script.MCPCommandDescription,
				"schema":      script.MCPInputSchema,
				"created_at":  script.CreatedAt,
			})
		}
	}

	c.JSON(200, gin.H{
		"commands": commands,
		"count":    len(commands),
	})
}

// ListMCPCommands 列出所有 MCP 命令
func (h *Handler) ListMCPCommands(c *gin.Context) {
	scripts, err := h.db.ListScripts()
	if err != nil {
		c.JSON(500, gin.H{"error": "error.getScriptListFailed"})
		return
	}

	commands := []map[string]interface{}{}
	for _, script := range scripts {
		if script.IsMCPCommand {
			commands = append(commands, map[string]interface{}{
				"id":          script.ID,
				"name":        script.Name,
				"command":     script.MCPCommandName,
				"description": script.MCPCommandDescription,
				"created_at":  script.CreatedAt,
			})
		}
	}

	c.JSON(200, gin.H{
		"commands": commands,
		"count":    len(commands),
	})
}

// ListPrompts 列出所有提示词
func (h *Handler) ListPrompts(c *gin.Context) {
	prompts, err := h.db.ListPrompts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getPromptListFailed"})
		return
	}

	// 检查是否需要过滤系统提示词 (默认不过滤,显示所有)
	// 只有明确传递 exclude_system=true 时才过滤
	excludeSystem := c.Query("exclude_system") == "true"

	if excludeSystem {
		// 过滤掉系统预设的提示词，只返回用户自定义的
		userPrompts := make([]*models.Prompt, 0)
		for _, prompt := range prompts {
			// 如果Type为空(旧数据),也认为是用户自定义的
			if prompt.Type == models.PromptTypeCustom || prompt.Type == "" {
				userPrompts = append(userPrompts, prompt)
			}
		}
		c.JSON(http.StatusOK, gin.H{"data": userPrompts})
	} else {
		// 返回所有提示词
		c.JSON(http.StatusOK, gin.H{"data": prompts})
	}
}

// GetPrompt 获取单个提示词
func (h *Handler) GetPrompt(c *gin.Context) {
	id := c.Param("id")
	prompt, err := h.db.GetPrompt(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.promptNotFound"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": prompt})
}

// CreatePrompt 创建提示词
func (h *Handler) CreatePrompt(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Content     string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	prompt := &models.Prompt{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Type:        models.PromptTypeCustom, // 用户创建的都是自定义类型
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.SavePrompt(prompt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.savePromptFailed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": prompt})
}

// UpdatePrompt 更新提示词
func (h *Handler) UpdatePrompt(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Content     string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 检查提示词是否存在
	existingPrompt, err := h.db.GetPrompt(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.promptNotFound"})
		return
	}

	// 更新字段
	existingPrompt.Name = req.Name
	existingPrompt.Description = req.Description
	existingPrompt.Content = req.Content
	existingPrompt.UpdatedAt = time.Now()

	if err := h.db.UpdatePrompt(existingPrompt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updatePromptFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": existingPrompt})
}

// DeletePrompt 删除提示词
func (h *Handler) DeletePrompt(c *gin.Context) {
	id := c.Param("id")

	// 检查是否是系统提示词
	prompt, err := h.db.GetPrompt(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.promptNotFound"})
		return
	}

	if prompt.Type == models.PromptTypeSystem {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.systemPromptCannotDelete"})
		return
	}

	if err := h.db.DeletePrompt(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deletePromptFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.promptDeleted"})
}

// ============= 脚本批量操作相关 API =============

// BatchSetGroup 批量设置脚本分组
func (h *Handler) BatchSetGroup(c *gin.Context) {
	var req struct {
		ScriptIDs []string `json:"script_ids" binding:"required"`
		Group     string   `json:"group" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	successCount := 0
	for _, id := range req.ScriptIDs {
		script, err := h.db.GetScript(id)
		if err != nil {
			continue
		}
		script.Group = req.Group
		script.UpdatedAt = time.Now()
		if err := h.db.UpdateScript(script); err != nil {
			continue
		}
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "script.messages.batchGroupSuccess",
		"count":   successCount,
	})
}

// BatchAddTags 批量添加标签
func (h *Handler) BatchAddTags(c *gin.Context) {
	var req struct {
		ScriptIDs []string `json:"script_ids" binding:"required"`
		Tags      []string `json:"tags" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	successCount := 0
	for _, id := range req.ScriptIDs {
		script, err := h.db.GetScript(id)
		if err != nil {
			continue
		}

		// 合并标签，去重
		tagMap := make(map[string]bool)
		for _, tag := range script.Tags {
			tagMap[tag] = true
		}
		for _, tag := range req.Tags {
			tagMap[tag] = true
		}

		newTags := make([]string, 0, len(tagMap))
		for tag := range tagMap {
			newTags = append(newTags, tag)
		}

		script.Tags = newTags
		script.UpdatedAt = time.Now()
		if err := h.db.UpdateScript(script); err != nil {
			continue
		}
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "script.messages.batchTagsSuccess",
		"count":   successCount,
	})
}

// BatchDeleteScripts 批量删除脚本
func (h *Handler) BatchDeleteScripts(c *gin.Context) {
	var req struct {
		ScriptIDs []string `json:"script_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	successCount := 0
	for _, id := range req.ScriptIDs {
		if err := h.db.DeleteScript(id); err != nil {
			continue
		}
		if err := h.db.DeleteToolConfigByScriptID(id); err != nil {
			continue
		}
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "script.messages.batchDeleteSuccess",
		"count":   successCount,
	})
}

// ============= 脚本执行记录相关 API =============

// ListScriptExecutions 列出脚本执行记录（支持分页和搜索）
func (h *Handler) ListScriptExecutions(c *gin.Context) {
	// 获取分页参数
	page := 1
	pageSize := 20
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// 获取过滤参数
	scriptID := c.Query("script_id")    // 按脚本ID过滤
	searchQuery := c.Query("search")    // 搜索脚本名称
	successFilter := c.Query("success") // 按成功/失败过滤

	// 获取所有执行记录
	executions, err := h.db.ListScriptExecutions(scriptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getExecutionRecordsFailed"})
		return
	}

	// 应用搜索过滤
	filteredExecutions := make([]*models.ScriptExecution, 0)
	for _, exec := range executions {
		// 搜索过滤
		if searchQuery != "" {
			if !strings.Contains(strings.ToLower(exec.ScriptName), strings.ToLower(searchQuery)) {
				continue
			}
		}

		// 成功/失败过滤
		if successFilter != "" {
			isSuccess := successFilter == "true"
			if exec.Success != isSuccess {
				continue
			}
		}

		if exec.VideoPath != "" {
			exec.VideoPath = "/files/" + exec.VideoPath
		}

		filteredExecutions = append(filteredExecutions, exec)
	}

	total := len(filteredExecutions)

	// 应用分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		filteredExecutions = []*models.ScriptExecution{}
	} else {
		if end > total {
			end = total
		}
		filteredExecutions = filteredExecutions[start:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": filteredExecutions,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// GetScriptExecution 获取单个执行记录详情
func (h *Handler) GetScriptExecution(c *gin.Context) {
	id := c.Param("id")

	execution, err := h.db.GetScriptExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.executionRecordNotFound"})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// DeleteScriptExecution 删除执行记录
func (h *Handler) DeleteScriptExecution(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteScriptExecution(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteExecutionRecordFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.executionRecordDeleted"})
}

// BatchDeleteScriptExecutions 批量删除执行记录
func (h *Handler) BatchDeleteScriptExecutions(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.selectExecutionRecords"})
		return
	}

	successCount := 0
	for _, id := range req.IDs {
		if err := h.db.DeleteScriptExecution(id); err == nil {
			successCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "execution.messages.batchDeleteSuccess",
		"count":   successCount,
	})
}

// ============= 录制配置相关 API =============

// GetRecordingConfig 获取录制配置
func (h *Handler) GetRecordingConfig(c *gin.Context) {
	config := h.db.GetDefaultRecordingConfig()
	c.JSON(200, config)
}

// UpdateRecordingConfig 更新录制配置
func (h *Handler) UpdateRecordingConfig(c *gin.Context) {
	var req models.RecordingConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "error.invalidParams"})
		return
	}

	// 固定ID为 default
	req.ID = "default"

	// 验证参数
	if req.FrameRate <= 0 || req.FrameRate > 60 {
		c.JSON(400, gin.H{"error": "error.frameRateRange"})
		return
	}
	if req.Quality <= 0 || req.Quality > 100 {
		c.JSON(400, gin.H{"error": "error.qualityRange"})
		return
	}
	if req.Format == "" {
		req.Format = "mp4"
	}
	if req.OutputDir == "" {
		req.OutputDir = "recordings"
	}

	if err := h.db.SaveRecordingConfig(&req); err != nil {
		c.JSON(500, gin.H{"error": "error.saveConfigFailed"})
		return
	}

	c.JSON(200, gin.H{
		"message": "success.recordingConfigUpdated",
		"config":  req,
	})
}

// ============= 辅助函数 =============

// replacePlaceholders 替换字符串中的占位符
// 支持 ${field} 格式，例如 ${keyword}, ${page}, ${category} 等
func replacePlaceholders(text string, params map[string]string) string {
	if text == "" {
		return text
	}

	// 替换所有占位符
	result := text
	for key, value := range params {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// 注意：这里不移除未替换的占位符，保留它们以便调试
	return result
}

// syncMCPRegistration 同步 MCP 命令注册状态
// 如果脚本是 MCP 命令则注册，否则取消注册
func (h *Handler) syncMCPRegistration(ctx context.Context, script *models.Script) {
	if h.mcpServer == nil {
		return
	}

	if script.IsMCPCommand {
		if err := h.mcpServer.RegisterScript(script); err != nil {
			logger.Error(ctx, "Failed to register MCP command: %v", err)
		}
	} else {
		h.mcpServer.UnregisterScript(script.ID)
	}
}

// ============= 工具管理相关 API =============

// ListToolConfigs 列出所有工具配置
func (h *Handler) ListToolConfigs(c *gin.Context) {
	// 获取分页参数
	page := 1
	pageSize := 20
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// 获取搜索和过滤参数
	searchQuery := c.Query("search")
	toolType := c.Query("type") // 可选: preset, script

	// 获取工具配置
	toolConfigs, err := h.db.ListToolConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建配置映射以便快速查找
	configMap := make(map[string]*models.ToolConfig)
	for _, cfg := range toolConfigs {
		configMap[cfg.ID] = cfg
	}

	// 检查并初始化预设工具配置
	presetToolsMetadata := localtools.GetPresetToolsMetadata()
	for _, meta := range presetToolsMetadata {
		if _, exists := configMap[meta.ID]; !exists {
			config := &models.ToolConfig{
				ID:          meta.ID,
				Name:        meta.Name,
				Type:        models.ToolTypePreset,
				Description: meta.Description,
				Enabled:     true,
				Parameters:  make(map[string]interface{}),
			}
			if err := h.db.SaveToolConfig(config); err == nil {
				toolConfigs = append(toolConfigs, config)
				configMap[config.ID] = config
			}
		}
	}

	// 检查并初始化脚本工具配置
	scripts, _ := h.db.ListScripts()
	for _, script := range scripts {
		if !script.IsMCPCommand || script.MCPCommandName == "" {
			continue
		}
		toolID := "script_" + script.ID
		if _, exists := configMap[toolID]; !exists {
			config := &models.ToolConfig{
				ID:          toolID,
				Name:        script.MCPCommandName,
				Type:        models.ToolTypeScript,
				Description: script.MCPCommandDescription,
				Enabled:     true,
				Parameters:  make(map[string]interface{}),
				ScriptID:    script.ID,
			}
			if err := h.db.SaveToolConfig(config); err == nil {
				toolConfigs = append(toolConfigs, config)
				configMap[config.ID] = config
			}
		}
	}

	// 获取脚本信息，构建脚本工具的完整信息
	scriptMap := make(map[string]*models.Script)
	for _, script := range scripts {
		scriptMap[script.ID] = script
	}

	// 获取预设工具元数据
	presetMetaMap := make(map[string]models.PresetToolMetadata)
	for _, meta := range presetToolsMetadata {
		presetMetaMap[meta.ID] = meta
	}

	// 构建响应
	type ToolConfigResponse struct {
		*models.ToolConfig
		Metadata *models.PresetToolMetadata `json:"metadata,omitempty"` // 预设工具的元数据
		Script   *models.Script             `json:"script,omitempty"`   // 脚本工具关联的脚本
	}

	var allTools []ToolConfigResponse
	for _, cfg := range toolConfigs {
		resp := ToolConfigResponse{ToolConfig: cfg}
		if cfg.Type == models.ToolTypePreset {
			if meta, ok := presetMetaMap[cfg.ID]; ok {
				resp.Metadata = &meta
			}
		} else if cfg.Type == models.ToolTypeScript {
			if script, ok := scriptMap[cfg.ScriptID]; ok {
				resp.Script = script
			}
		}
		allTools = append(allTools, resp)
	}

	// 应用搜索和类型过滤
	var filteredTools []ToolConfigResponse
	for _, tool := range allTools {
		// 类型过滤
		if toolType != "" && string(tool.Type) != toolType {
			continue
		}

		// 搜索过滤
		if searchQuery != "" {
			searchLower := strings.ToLower(searchQuery)
			if !strings.Contains(strings.ToLower(tool.Name), searchLower) &&
				!strings.Contains(strings.ToLower(tool.Description), searchLower) {
				continue
			}
		}

		filteredTools = append(filteredTools, tool)
	}

	total := len(filteredTools)

	// 应用分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		filteredTools = []ToolConfigResponse{}
	} else {
		if end > total {
			end = total
		}
		filteredTools = filteredTools[start:end]
	}

	// 确保至少返回空数组而不是 null
	if filteredTools == nil {
		filteredTools = []ToolConfigResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      filteredTools,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetToolConfig 获取单个工具配置
func (h *Handler) GetToolConfig(c *gin.Context) {
	id := c.Param("id")
	config, err := h.db.GetToolConfig(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tool config not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// UpdateToolConfig 更新工具配置
func (h *Handler) UpdateToolConfig(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Enabled    *bool                  `json:"enabled"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取现有配置
	config, err := h.db.GetToolConfig(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tool config not found"})
		return
	}

	// 更新字段
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.Parameters != nil {
		config.Parameters = req.Parameters
	}

	// 保存
	if err := h.db.SaveToolConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// SyncToolConfigs 同步工具配置（确保数据库中有所有工具的配置）
func (h *Handler) SyncToolConfigs(c *gin.Context) {
	// 获取现有配置
	existingConfigs, err := h.db.ListToolConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	configMap := make(map[string]*models.ToolConfig)
	for _, cfg := range existingConfigs {
		configMap[cfg.ID] = cfg
	}

	// 同步预设工具
	presetToolsMetadata := localtools.GetPresetToolsMetadata()
	for _, meta := range presetToolsMetadata {
		if _, exists := configMap[meta.ID]; !exists {
			config := &models.ToolConfig{
				ID:          meta.ID,
				Name:        meta.Name,
				Type:        models.ToolTypePreset,
				Description: meta.Description,
				Enabled:     true,
				Parameters:  make(map[string]interface{}),
			}
			if err := h.db.SaveToolConfig(config); err != nil {
				logger.Warn(c.Request.Context(), "Failed to sync tool config: %s, error: %v", meta.ID, err)
			}
		}
	}

	// 同步脚本工具
	scripts, err := h.db.ListScripts()
	if err == nil {
		for _, script := range scripts {
			if !script.IsMCPCommand || script.MCPCommandName == "" {
				continue
			}

			toolID := "script_" + script.ID
			if _, exists := configMap[toolID]; !exists {
				config := &models.ToolConfig{
					ID:          toolID,
					Name:        script.MCPCommandName,
					Type:        models.ToolTypeScript,
					Description: script.MCPCommandDescription,
					Enabled:     true,
					Parameters:  make(map[string]interface{}),
					ScriptID:    script.ID,
				}
				if err := h.db.SaveToolConfig(config); err != nil {
					logger.Warn(c.Request.Context(), "Failed to sync script tool config: %s, error: %v", script.ID, err)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "toolManager.syncSuccess"})
}

// ============= MCP服务管理相关 API =============

// ListMCPServices 列出所有MCP服务
func (h *Handler) ListMCPServices(c *gin.Context) {
	services, err := h.db.ListMCPServices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getMCPServicesFailed"})
		return
	}

	if services == nil {
		services = []*models.MCPService{}
	}

	c.JSON(http.StatusOK, gin.H{"data": services})
}

// GetMCPService 获取单个MCP服务
func (h *Handler) GetMCPService(c *gin.Context) {
	id := c.Param("id")

	service, err := h.db.GetMCPService(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.mcpServiceNotFound"})
		return
	}

	c.JSON(http.StatusOK, service)
}

// CreateMCPService 创建MCP服务
func (h *Handler) CreateMCPService(c *gin.Context) {
	var req models.MCPService
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(c.Request.Context(), "Failed to bind MCP service JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "error.invalidParams",
			"details": err.Error(),
		})
		return
	}

	// 验证必填字段
	if req.Name == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.mcpServiceRequiredFields"})
		return
	}

	// 根据类型验证必填字段
	switch req.Type {
	case models.MCPServiceTypeStdio:
		if req.Command == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error.mcpServiceCommandRequired"})
			return
		}
	case models.MCPServiceTypeSSE, models.MCPServiceTypeHTTP:
		if req.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error.mcpServiceURLRequired"})
			return
		}
	}

	// 生成ID
	req.ID = fmt.Sprintf("mcp_%d", time.Now().Unix())
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.Status = models.MCPServiceStatusDisconnected
	req.ToolCount = 0

	if err := h.db.SaveMCPService(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.saveMCPServiceFailed"})
		return
	}

	// 自动发现工具（异步执行，不阻塞响应）
	go func() {
		ctx := context.Background()
		tools, err := h.discoverMCPTools(ctx, &req)
		if err != nil {
			logger.Warn(ctx, "Failed to auto-discover tools for MCP service %s: %v", req.Name, err)
			req.Status = models.MCPServiceStatusError
			req.LastError = fmt.Sprintf("Auto-discovery failed: %v", err)
		} else {
			req.Status = models.MCPServiceStatusConnected
			req.ToolCount = len(tools)
			req.LastError = ""
			if err := h.db.SaveMCPServiceTools(req.ID, tools); err != nil {
				logger.Error(ctx, "Failed to save discovered tools: %v", err)
			}
		}
		req.UpdatedAt = time.Now()
		h.db.SaveMCPService(&req)

		// 通知Agent重新加载MCP配置
		if h.agentManager != nil {
			type AgentManagerInterface interface {
				ReloadMCPServices() error
			}
			if am, ok := h.agentManager.(AgentManagerInterface); ok {
				am.ReloadMCPServices()
			}
		}
	}()

	// 通知Agent重新加载MCP配置
	if h.agentManager != nil {
		type AgentManagerInterface interface {
			ReloadMCPServices() error
		}
		if am, ok := h.agentManager.(AgentManagerInterface); ok {
			if err := am.ReloadMCPServices(); err != nil {
				logger.Warn(c.Request.Context(), "Failed to reload MCP services: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.mcpServiceCreated",
		"service": req,
	})
}

// UpdateMCPService 更新MCP服务
func (h *Handler) UpdateMCPService(c *gin.Context) {
	id := c.Param("id")

	var req models.MCPService
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 检查服务是否存在
	_, err := h.db.GetMCPService(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.mcpServiceNotFound"})
		return
	}

	// 确保ID一致
	req.ID = id
	req.UpdatedAt = time.Now()

	if err := h.db.SaveMCPService(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updateMCPServiceFailed"})
		return
	}

	// 自动重新发现工具（异步执行）
	go func() {
		ctx := context.Background()
		tools, err := h.discoverMCPTools(ctx, &req)
		if err != nil {
			logger.Warn(ctx, "Failed to re-discover tools for MCP service %s: %v", req.Name, err)
			req.Status = models.MCPServiceStatusError
			req.LastError = fmt.Sprintf("Re-discovery failed: %v", err)
		} else {
			req.Status = models.MCPServiceStatusConnected
			req.ToolCount = len(tools)
			req.LastError = ""
			if err := h.db.SaveMCPServiceTools(req.ID, tools); err != nil {
				logger.Error(ctx, "Failed to save re-discovered tools: %v", err)
			}
		}
		req.UpdatedAt = time.Now()
		h.db.SaveMCPService(&req)

		// 通知Agent重新加载MCP配置
		if h.agentManager != nil {
			type AgentManagerInterface interface {
				ReloadMCPServices() error
			}
			if am, ok := h.agentManager.(AgentManagerInterface); ok {
				am.ReloadMCPServices()
			}
		}
	}()

	// 通知Agent重新加载MCP配置
	if h.agentManager != nil {
		type AgentManagerInterface interface {
			ReloadMCPServices() error
		}
		if am, ok := h.agentManager.(AgentManagerInterface); ok {
			if err := am.ReloadMCPServices(); err != nil {
				logger.Warn(c.Request.Context(), "Failed to reload MCP services: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.mcpServiceUpdated",
		"service": req,
	})
}

// DeleteMCPService 删除MCP服务
func (h *Handler) DeleteMCPService(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteMCPService(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteMCPServiceFailed"})
		return
	}

	// 通知Agent重新加载MCP配置
	if h.agentManager != nil {
		type AgentManagerInterface interface {
			ReloadMCPServices() error
		}
		if am, ok := h.agentManager.(AgentManagerInterface); ok {
			if err := am.ReloadMCPServices(); err != nil {
				logger.Warn(c.Request.Context(), "Failed to reload MCP services: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.mcpServiceDeleted"})
}

// ToggleMCPService 启用/禁用MCP服务
func (h *Handler) ToggleMCPService(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	service, err := h.db.GetMCPService(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.mcpServiceNotFound"})
		return
	}

	service.Enabled = req.Enabled
	service.UpdatedAt = time.Now()

	if err := h.db.SaveMCPService(service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updateMCPServiceFailed"})
		return
	}

	// 通知Agent重新加载MCP配置
	if h.agentManager != nil {
		type AgentManagerInterface interface {
			ReloadMCPServices() error
		}
		if am, ok := h.agentManager.(AgentManagerInterface); ok {
			if err := am.ReloadMCPServices(); err != nil {
				logger.Warn(c.Request.Context(), "Failed to reload MCP services: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.mcpServiceToggled",
		"service": service,
	})
}

// GetMCPServiceTools 获取MCP服务的工具列表
func (h *Handler) GetMCPServiceTools(c *gin.Context) {
	id := c.Param("id")

	tools, err := h.db.GetMCPServiceTools(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getMCPServiceToolsFailed"})
		return
	}

	if tools == nil {
		tools = []models.MCPDiscoveredTool{}
	}

	c.JSON(http.StatusOK, gin.H{"data": tools})
}

// DiscoverMCPServiceTools 发现MCP服务的工具(连接服务并获取工具列表)
func (h *Handler) DiscoverMCPServiceTools(c *gin.Context) {
	id := c.Param("id")

	service, err := h.db.GetMCPService(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.mcpServiceNotFound"})
		return
	}

	// 更新状态为连接中
	service.Status = models.MCPServiceStatusConnecting
	service.LastError = ""
	h.db.SaveMCPService(service)

	// 实际连接MCP服务并发现工具
	discoveredTools, err := h.discoverMCPTools(c.Request.Context(), service)
	if err != nil {
		service.Status = models.MCPServiceStatusError
		service.LastError = fmt.Sprintf("Failed to discover tools: %v", err)
		h.db.SaveMCPService(service)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "error.discoverMCPServiceToolsFailed",
			"details": err.Error(),
		})
		return
	}

	// 保存发现的工具
	if err := h.db.SaveMCPServiceTools(id, discoveredTools); err != nil {
		service.Status = models.MCPServiceStatusError
		service.LastError = err.Error()
		h.db.SaveMCPService(service)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.saveMCPServiceToolsFailed"})
		return
	}

	// 更新服务状态
	service.Status = models.MCPServiceStatusConnected
	service.ToolCount = len(discoveredTools)
	service.UpdatedAt = time.Now()
	h.db.SaveMCPService(service)

	// 通知Agent重新加载MCP配置
	if h.agentManager != nil {
		type AgentManagerInterface interface {
			ReloadMCPServices() error
		}
		if am, ok := h.agentManager.(AgentManagerInterface); ok {
			if err := am.ReloadMCPServices(); err != nil {
				logger.Warn(c.Request.Context(), "Failed to reload MCP services: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.mcpServiceToolsDiscovered",
		"tools":   discoveredTools,
	})
}

// UpdateMCPServiceToolEnabled 更新单个工具的启用状态
func (h *Handler) UpdateMCPServiceToolEnabled(c *gin.Context) {
	id := c.Param("id")
	toolName := c.Param("toolName")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 获取工具列表
	tools, err := h.db.GetMCPServiceTools(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getMCPServiceToolsFailed"})
		return
	}

	// 查找并更新工具
	found := false
	for i := range tools {
		if tools[i].Name == toolName {
			tools[i].Enabled = req.Enabled
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.toolNotFound"})
		return
	}

	// 保存更新后的工具列表
	if err := h.db.SaveMCPServiceTools(id, tools); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.saveMCPServiceToolsFailed"})
		return
	}

	// 通知Agent重新加载MCP配置
	if h.agentManager != nil {
		type AgentManagerInterface interface {
			ReloadMCPServices() error
		}
		if am, ok := h.agentManager.(AgentManagerInterface); ok {
			if err := am.ReloadMCPServices(); err != nil {
				logger.Warn(c.Request.Context(), "Failed to reload MCP services: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.mcpServiceToolUpdated",
	})
}

// discoverMCPTools 连接MCP服务并发现工具
func (h *Handler) discoverMCPTools(ctx context.Context, service *models.MCPService) ([]models.MCPDiscoveredTool, error) {
	logger.Info(ctx, "Discovering tools for MCP service: %s (type: %s, url: %s)", service.Name, service.Type, service.URL)

	var mcpServer interfaces.MCPServer
	var err error

	// 根据服务类型创建MCP服务器连接
	switch service.Type {
	case models.MCPServiceTypeStdio:
		// 创建stdio类型的MCP服务器
		config := mcp.StdioServerConfig{
			Command: service.Command,
			Args:    service.Args,
		}
		// 转换环境变量
		if len(service.Env) > 0 {
			envSlice := make([]string, 0, len(service.Env))
			for k, v := range service.Env {
				envSlice = append(envSlice, k+"="+v)
			}
			config.Env = envSlice
		}

		mcpServer, err = mcp.NewStdioServer(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdio MCP server: %w", err)
		}

	case models.MCPServiceTypeSSE, models.MCPServiceTypeHTTP:
		// 创建HTTP/SSE类型的MCP服务器
		if service.URL == "" {
			return nil, fmt.Errorf("%s type requires URL", service.Type)
		}

		// 根据类型设置ProtocolType
		protocolType := mcp.StreamableHTTP
		if service.Type == models.MCPServiceTypeSSE {
			protocolType = mcp.SSE
		}

		config := mcp.HTTPServerConfig{
			BaseURL:      service.URL,
			ProtocolType: protocolType,
		}

		logger.Info(ctx, "Creating MCP HTTP server with BaseURL: %s, Path: %s, ProtocolType: %s", config.BaseURL, config.Path, config.ProtocolType)

		mcpServer, err = mcp.NewHTTPServer(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP/SSE MCP server: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported MCP service type: %s", service.Type)
	}

	// 确保关闭连接
	defer func() {
		if mcpServer != nil {
			mcpServer.Close()
		}
	}()

	// 初始化连接
	if err := mcpServer.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	// 获取工具列表
	tools, err := mcpServer.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	// 转换为我们的工具模型
	discoveredTools := make([]models.MCPDiscoveredTool, 0, len(tools))
	for _, tool := range tools {
		discoveredTool := models.MCPDiscoveredTool{
			Name:        tool.Name,
			Description: tool.Description,
			Enabled:     true, // 默认启用所有发现的工具
		}

		// 保存工具的输入Schema
		if tool.Schema != nil {
			// 将Schema转换为map[string]interface{}
			schemaMap, ok := tool.Schema.(map[string]interface{})
			if ok {
				discoveredTool.Schema = schemaMap
			} else {
				// 尝试JSON序列化再反序列化
				schemaBytes, err := json.Marshal(tool.Schema)
				if err == nil {
					var schemaObj map[string]interface{}
					if err := json.Unmarshal(schemaBytes, &schemaObj); err == nil {
						discoveredTool.Schema = schemaObj
					}
				}
			}
		}

		discoveredTools = append(discoveredTools, discoveredTool)
	}

	logger.Info(ctx, "Discovered %d tools from MCP service: %s", len(discoveredTools), service.Name)
	return discoveredTools, nil
}
