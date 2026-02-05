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
	executor2 "github.com/browserwing/browserwing/executor"
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
	executor       *executor2.Executor // Executor 实例
	config         *config.Config
	llmManager     *llm.Manager
	mcpServer      MCPHTTPHandler // MCP 服务器（使用 interface{} 避免循环依赖）
	agentManager   interface{}    // Agent 管理器（用于 LLM 配置更新后的热加载）
	scheduler      interface{}    // 定时任务调度器
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
		executor:       executor2.NewExecutor(browserMgr), // 初始化 Executor
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
		URL        string `json:"url" binding:"required"`
		Language   string `json:"language"`    // 前端当前语言
		InstanceID string `json:"instance_id"` // 指定实例ID，空字符串表示使用当前实例
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	if !h.browserManager.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.browserNotRunning"})
		return
	}

	if err := h.browserManager.OpenPage(req.URL, req.Language, req.InstanceID); err != nil {
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

// DeleteCookie 删除单个Cookie
func (h *Handler) DeleteCookie(c *gin.Context) {
	var req struct {
		ID     string `json:"id" binding:"required"`
		Name   string `json:"name" binding:"required"`
		Domain string `json:"domain" binding:"required"`
		Path   string `json:"path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 获取现有的cookie store
	cookieStore, err := h.db.GetCookies(req.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.cookieNotFound"})
		return
	}

	// 查找并删除匹配的cookie（通过 name + domain + path 唯一标识）
	found := false
	updatedCookies := make([]*proto.NetworkCookie, 0, len(cookieStore.Cookies))
	for _, cookie := range cookieStore.Cookies {
		if cookie.Name == req.Name && cookie.Domain == req.Domain && cookie.Path == req.Path {
			found = true
			continue // 跳过要删除的cookie
		}
		updatedCookies = append(updatedCookies, cookie)
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.cookieNotFound"})
		return
	}

	// 保存更新后的cookies
	cookieStore.Cookies = updatedCookies
	if err := h.db.SaveCookies(cookieStore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteCookieFailed"})
		return
	}

	logger.Info(c.Request.Context(), "Deleted cookie: %s (domain: %s, path: %s)", req.Name, req.Domain, req.Path)
	c.JSON(http.StatusOK, gin.H{
		"message": "success.cookieDeleted",
		"count":   len(updatedCookies),
	})
}

// BatchDeleteCookies 批量删除Cookies
func (h *Handler) BatchDeleteCookies(c *gin.Context) {
	type CookieIdentifier struct {
		Name   string `json:"name" binding:"required"`
		Domain string `json:"domain" binding:"required"`
		Path   string `json:"path" binding:"required"`
	}

	var req struct {
		ID      string             `json:"id" binding:"required"`
		Cookies []CookieIdentifier `json:"cookies" binding:"required"` // 要删除的cookie标识列表（name+domain+path）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	if len(req.Cookies) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	// 获取现有的cookie store
	cookieStore, err := h.db.GetCookies(req.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.cookieNotFound"})
		return
	}

	// 构建要删除的cookie的集合（使用 name+domain+path 作为key）
	deleteSet := make(map[string]bool)
	for _, ci := range req.Cookies {
		key := ci.Name + "|" + ci.Domain + "|" + ci.Path
		deleteSet[key] = true
	}

	// 过滤掉要删除的cookies
	updatedCookies := make([]*proto.NetworkCookie, 0, len(cookieStore.Cookies))
	deletedCount := 0
	for _, cookie := range cookieStore.Cookies {
		key := cookie.Name + "|" + cookie.Domain + "|" + cookie.Path
		if deleteSet[key] {
			deletedCount++
			continue // 跳过要删除的cookie
		}
		updatedCookies = append(updatedCookies, cookie)
	}

	if deletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.cookieNotFound"})
		return
	}

	// 保存更新后的cookies
	cookieStore.Cookies = updatedCookies
	if err := h.db.SaveCookies(cookieStore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteCookiesFailed"})
		return
	}

	logger.Info(c.Request.Context(), "Batch deleted %d cookies", deletedCount)
	c.JSON(http.StatusOK, gin.H{
		"message":       "success.cookiesDeleted",
		"deleted_count": deletedCount,
		"remaining":     len(updatedCookies),
	})
}

// ============= 脚本录制和回放相关 API =============

// StartRecording 开始录制操作
func (h *Handler) StartRecording(c *gin.Context) {
	var req struct {
		InstanceID string `json:"instance_id"` // 指定实例ID，空字符串表示使用当前实例
	}
	// 尝试解析请求体，如果失败或为空则使用默认值
	_ = c.ShouldBindJSON(&req)

	if !h.browserManager.IsRunning() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.browserNotRunning"})
		return
	}

	if err := h.browserManager.StartRecording(c.Request.Context(), req.InstanceID); err != nil {
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
		Variables             map[string]string       `json:"variables"`
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
		Variables:       req.Variables,
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
		Variables             map[string]string      `json:"variables"`
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
	if req.Variables != nil {
		script.Variables = req.Variables
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
		Params     map[string]string `json:"params"`
		InstanceID string            `json:"instance_id"` // 指定实例ID，空字符串表示使用当前实例
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有请求体或解析失败,使用空参数
		req.Params = make(map[string]string)
	}

	// 创建脚本副本并合并参数
	scriptToRun := script.Copy()

	// 合并参数：先使用脚本预设变量，再用外部传入的参数覆盖
	mergedParams := make(map[string]string)

	// 1. 首先添加脚本的预设变量
	if scriptToRun.Variables != nil {
		for key, value := range scriptToRun.Variables {
			mergedParams[key] = value
		}
		for key := range scriptToRun.Variables {
			if _, ok := req.Params[key]; ok {
				scriptToRun.Variables[key] = req.Params[key]
			}
		}
	}

	// 2. 外部传入的参数会覆盖预设变量
	for key, value := range req.Params {
		mergedParams[key] = value
	}

	// 如果有参数（包括预设变量和外部参数），替换占位符
	if len(mergedParams) > 0 {

		// 如果用户提供了 url 参数,使用它;否则替换 URL 中的占位符
		if urlParam, ok := mergedParams["url"]; ok && urlParam != "" {
			scriptToRun.URL = urlParam
		} else {
			scriptToRun.URL = replacePlaceholders(scriptToRun.URL, mergedParams)
		}

		// 复制 actions 数组以避免修改原始数据
		scriptToRun.Actions = make([]models.ScriptAction, len(script.Actions))
		copy(scriptToRun.Actions, script.Actions)

		// 替换所有 action 中的占位符
		for i := range scriptToRun.Actions {
			scriptToRun.Actions[i].Selector = replacePlaceholders(scriptToRun.Actions[i].Selector, mergedParams)
			scriptToRun.Actions[i].XPath = replacePlaceholders(scriptToRun.Actions[i].XPath, mergedParams)
			scriptToRun.Actions[i].Value = replacePlaceholders(scriptToRun.Actions[i].Value, mergedParams)
			scriptToRun.Actions[i].URL = replacePlaceholders(scriptToRun.Actions[i].URL, mergedParams)
			scriptToRun.Actions[i].JSCode = replacePlaceholders(scriptToRun.Actions[i].JSCode, mergedParams)

			// 替换文件路径中的占位符
			if len(scriptToRun.Actions[i].FilePaths) > 0 {
				newFilePaths := make([]string, len(scriptToRun.Actions[i].FilePaths))
				for j, path := range scriptToRun.Actions[i].FilePaths {
					newFilePaths[j] = replacePlaceholders(path, mergedParams)
				}
				scriptToRun.Actions[i].FilePaths = newFilePaths
			}
		}
	}

	// 执行回放
	result, page, err := h.browserManager.PlayScript(c.Request.Context(), scriptToRun, req.InstanceID)
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

	if req.Provider == "ollama" {
		req.APIKey = "ollama"
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

	// 如果数据库中没有配置，创建并保存默认配置
	if len(configs) == 0 {
		defaultConfig := h.browserManager.GetDefaultBrowserConfig()
		if err := h.db.SaveBrowserConfig(defaultConfig); err != nil {
			logger.Error(c.Request.Context(), "Failed to save default browser config: %v", err)
			c.JSON(500, gin.H{"error": "Failed to create default configuration"})
			return
		}
		configs = []models.BrowserConfig{*defaultConfig}
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
	actionsJSON := fmt.Sprintf("Script Variables: %+v\nActions: %s", script.Variables, script.GetActionsWithoutSemanticInfoJSON())

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

// ResetPrompt 重置系统提示词为默认值
func (h *Handler) ResetPrompt(c *gin.Context) {
	id := c.Param("id")

	// 检查是否是系统提示词
	prompt, err := h.db.GetPrompt(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.promptNotFound"})
		return
	}

	if prompt.Type != models.PromptTypeSystem {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.onlySystemPromptCanReset"})
		return
	}

	// 获取最新的系统提示词
	latestPrompt := models.GetSystemPromptByID(id)
	if latestPrompt == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.systemPromptNotFound"})
		return
	}

	// 重置为最新版本，保留原始CreatedAt
	latestPrompt.CreatedAt = prompt.CreatedAt
	latestPrompt.UpdatedAt = prompt.CreatedAt // 重置后UpdatedAt等于CreatedAt，表示未修改

	if err := h.db.SavePrompt(latestPrompt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.resetPromptFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.promptReset",
		"data":    latestPrompt,
	})
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
	noPagination := false // 是否禁用分页

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil {
			if ps == 0 {
				// page_size=0 表示不分页，返回所有数据
				noPagination = true
			} else if ps > 0 && ps <= 1000 {
				pageSize = ps
			}
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

	// 应用分页（如果启用）
	if !noPagination {
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

// ============= 认证相关 API =============

// CheckAuth 检查是否需要认证
func (h *Handler) CheckAuth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled": h.config.Auth.Enabled,
	})
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	// 获取用户
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "error.invalidCredentials"})
		return
	}

	// 验证密码
	if user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "error.invalidCredentials"})
		return
	}

	// 生成JWT Token
	token, err := GenerateJWT(user.ID, user.Username, h.config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.generateTokenFailed"})
		return
	}

	// 清空密码字段，不返回给前端
	user.Password = ""

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  user,
	})
}

// ============= 用户管理 API =============

// ListUsers 列出所有用户
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.db.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.listUsersFailed"})
		return
	}

	// 清空所有用户的密码字段
	for _, user := range users {
		user.Password = ""
	}

	c.JSON(http.StatusOK, users)
}

// GetUser 获取用户信息
func (h *Handler) GetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.db.GetUser(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.userNotFound"})
		return
	}

	// 清空密码字段
	user.Password = ""

	c.JSON(http.StatusOK, user)
}

// CreateUser 创建用户
func (h *Handler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	// 检查用户名是否已存在
	_, err := h.db.GetUserByUsername(req.Username)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.usernameExists"})
		return
	}

	// 创建用户
	user := &models.User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Password:  req.Password, // 实际应该加密
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.createUserFailed"})
		return
	}

	// 清空密码字段，不返回给前端
	user.Password = ""

	c.JSON(http.StatusOK, user)
}

// UpdatePassword 更新密码
func (h *Handler) UpdatePassword(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	// 获取用户
	user, err := h.db.GetUser(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.userNotFound"})
		return
	}

	// 验证旧密码
	if user.Password != req.OldPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidOldPassword"})
		return
	}

	// 更新密码
	user.Password = req.NewPassword
	user.UpdatedAt = time.Now()

	if err := h.db.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updatePasswordFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.passwordUpdated"})
}

// DeleteUser 删除用户
func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// 删除用户的所有API密钥
	apiKeys, err := h.db.ListApiKeysByUser(id)
	if err == nil {
		for _, key := range apiKeys {
			h.db.DeleteApiKey(key.ID)
		}
	}

	if err := h.db.DeleteUser(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteUserFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.userDeleted"})
}

// ============= ApiKey 管理 API =============

// ListApiKeys 列出API密钥
func (h *Handler) ListApiKeys(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "error.unauthorized"})
		return
	}

	apiKeys, err := h.db.ListApiKeysByUser(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.listApiKeysFailed"})
		return
	}

	c.JSON(http.StatusOK, apiKeys)
}

// GetApiKey 获取API密钥
func (h *Handler) GetApiKey(c *gin.Context) {
	id := c.Param("id")

	apiKey, err := h.db.GetApiKey(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.apiKeyNotFound"})
		return
	}

	// 验证是否是当前用户的API密钥
	userID, _ := c.Get("user_id")
	if apiKey.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "error.forbidden"})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// CreateApiKey 创建API密钥
func (h *Handler) CreateApiKey(c *gin.Context) {
	var req models.CreateApiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "error.unauthorized"})
		return
	}

	// 生成随机API密钥
	apiKeyValue := "bw_" + uuid.New().String()

	apiKey := &models.ApiKey{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Key:         apiKeyValue,
		Description: req.Description,
		UserID:      userID.(string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.CreateApiKey(apiKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.createApiKeyFailed"})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// DeleteApiKey 删除API密钥
func (h *Handler) DeleteApiKey(c *gin.Context) {
	id := c.Param("id")

	// 获取API密钥
	apiKey, err := h.db.GetApiKey(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.apiKeyNotFound"})
		return
	}

	// 验证是否是当前用户的API密钥
	userID, _ := c.Get("user_id")
	if apiKey.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "error.forbidden"})
		return
	}

	if err := h.db.DeleteApiKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteApiKeyFailed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.apiKeyDeleted"})
}

// ============= Claude Skills 相关 API =============

// ExportScriptsSkill 导出脚本为 Claude Skills 的 SKILL.md 格式
func (h *Handler) ExportScriptsSkill(c *gin.Context) {
	var req struct {
		ScriptIDs []string `json:"script_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	// 获取脚本列表
	var scripts []*models.Script
	var err error

	if len(req.ScriptIDs) == 0 {
		// 如果没有指定脚本ID，导出所有脚本
		scripts, err = h.db.ListScripts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error.listScriptsFailed"})
			return
		}
	} else {
		// 导出指定的脚本
		for _, id := range req.ScriptIDs {
			script, err := h.db.GetScript(id)
			if err != nil {
				continue // 跳过不存在的脚本
			}
			scripts = append(scripts, script)
		}
	}

	if len(scripts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.noScriptsToExport"})
		return
	}

	// 获取服务端地址
	host := c.Request.Host

	// 判断是否导出所有脚本
	isExportAll := len(req.ScriptIDs) == 0

	// 收集脚本 ID
	scriptIDs := make([]string, len(scripts))
	for i, script := range scripts {
		scriptIDs[i] = script.ID
	}

	// 生成 SKILL.md 内容
	skillContent := generateSkillMD(scripts, host, isExportAll, scriptIDs)

	fileName := fmt.Sprintf("SKILL_%s.md", time.Now().Format("20060102150405"))

	// 返回内容
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.String(http.StatusOK, skillContent)
}

// GetScriptsSummary 获取脚本摘要信息（用于 Claude Skills）
func (h *Handler) GetScriptsSummary(c *gin.Context) {
	// 获取脚本 ID 列表（可选）
	idsParam := c.Query("ids")
	var filterIDs []string
	if idsParam != "" {
		filterIDs = strings.Split(idsParam, ",")
	}

	// 获取所有脚本
	allScripts, err := h.db.ListScripts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.listScriptsFailed"})
		return
	}

	// 如果有 ID 过滤，只返回指定的脚本
	var scripts []*models.Script
	if len(filterIDs) > 0 {
		idSet := make(map[string]bool)
		for _, id := range filterIDs {
			idSet[strings.TrimSpace(id)] = true
		}
		for _, script := range allScripts {
			if idSet[script.ID] {
				scripts = append(scripts, script)
			}
		}
	} else {
		scripts = allScripts
	}

	// 构建摘要信息
	type ParameterInfo struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Type        string `json:"type,omitempty"`
		Required    bool   `json:"required,omitempty"`
		Default     string `json:"default,omitempty"`
	}

	type ScriptSummary struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  []ParameterInfo `json:"parameters,omitempty"`
		Tags        []string        `json:"tags,omitempty"`
		Group       string          `json:"group,omitempty"`
	}

	summaries := make([]ScriptSummary, 0, len(scripts))
	for _, script := range scripts {
		summary := ScriptSummary{
			ID:          script.ID,
			Name:        script.Name,
			Description: script.Description,
			Tags:        script.Tags,
			Group:       script.Group,
		}

		// 从 MCP Input Schema 提取参数信息
		if script.MCPInputSchema != nil {
			if properties, ok := script.MCPInputSchema["properties"].(map[string]interface{}); ok {
				var required []string
				if req, ok := script.MCPInputSchema["required"].([]interface{}); ok {
					for _, r := range req {
						if rStr, ok := r.(string); ok {
							required = append(required, rStr)
						}
					}
				}
				requiredSet := make(map[string]bool)
				for _, r := range required {
					requiredSet[r] = true
				}

				for paramName, paramSchema := range properties {
					if paramMap, ok := paramSchema.(map[string]interface{}); ok {
						param := ParameterInfo{
							Name:     paramName,
							Required: requiredSet[paramName],
						}
						if desc, ok := paramMap["description"].(string); ok {
							param.Description = desc
						}
						if paramType, ok := paramMap["type"].(string); ok {
							param.Type = paramType
						}
						if defVal, ok := paramMap["default"].(string); ok {
							param.Default = defVal
						}
						summary.Parameters = append(summary.Parameters, param)
					}
				}
			}
		}

		// 如果没有 MCP Schema，从 Variables 提取
		if len(summary.Parameters) == 0 && len(script.Variables) > 0 {
			for varName, varValue := range script.Variables {
				param := ParameterInfo{
					Name:    varName,
					Default: varValue,
					Type:    "string",
				}
				summary.Parameters = append(summary.Parameters, param)
			}
		}

		summaries = append(summaries, summary)
	}

	c.JSON(http.StatusOK, gin.H{
		"scripts": summaries,
		"total":   len(summaries),
	})
}

// generateSkillMD 生成 SKILL.md 内容
func generateSkillMD(scripts []*models.Script, host string, isExportAll bool, scriptIDs []string) string {
	var sb strings.Builder

	skillDescription := "Execute browser automation scripts via HTTP API. Scripts include: "

	for i, script := range scripts {
		// 最多10个脚本描述
		if i >= 10 {
			break
		}
		description := script.Description
		if description == "" && script.Name != "" {
			description = script.Name
		}
		if description == "" && script.MCPCommandDescription != "" {
			description = script.MCPCommandDescription
		}
		if i == len(scripts)-1 || i == 9 {
			skillDescription += description
		} else {
			skillDescription += description + ","
		}
	}

	// YAML Frontmatter
	sb.WriteString("---\n")
	sb.WriteString("name: browserwing-scripts\n")
	sb.WriteString("description: " + skillDescription + "\n")
	sb.WriteString("---\n\n")

	// 主标题
	sb.WriteString("# BrowserWing Automation Scripts\n\n")

	// 简介
	sb.WriteString("## Overview\n\n")
	sb.WriteString("BrowserWing provides browser automation capabilities through HTTP APIs. You can execute pre-configured scripts to automate web tasks.\n\n")
	sb.WriteString(fmt.Sprintf("**Total Scripts Available:** %d\n\n", len(scripts)))
	sb.WriteString(fmt.Sprintf("**API Base URL:** `http://%s/api/v1`\n\n", host))

	// API 端点基础信息
	sb.WriteString("## API Endpoints\n\n")

	// 判断是否需要调用接口获取脚本列表
	needListAPI := len(scripts) >= 5

	if needListAPI {
		// 1. 获取脚本摘要列表
		sb.WriteString("### 1. Get Scripts Summary\n\n")
		sb.WriteString("Get summary information for available scripts (name, description, parameters).\n\n")
		sb.WriteString("```bash\n")
		if isExportAll {
			sb.WriteString(fmt.Sprintf("curl -X GET 'http://%s/api/v1/scripts/summary'\n", host))
		} else {
			// 限制只能获取导出的脚本
			idsParam := strings.Join(scriptIDs, ",")
			sb.WriteString(fmt.Sprintf("curl -X GET 'http://%s/api/v1/scripts/summary?ids=%s'\n", host, idsParam))
		}
		sb.WriteString("```\n\n")
		sb.WriteString("**Response Example:**\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"scripts\": [\n")
		sb.WriteString("    {\n")
		sb.WriteString("      \"id\": \"script-id\",\n")
		sb.WriteString("      \"name\": \"Script Name\",\n")
		sb.WriteString("      \"description\": \"What this script does\",\n")
		sb.WriteString("      \"url\": \"https://target-website.com\",\n")
		sb.WriteString("      \"parameters\": [\n")
		sb.WriteString("        {\n")
		sb.WriteString("          \"name\": \"username\",\n")
		sb.WriteString("          \"description\": \"User login name\",\n")
		sb.WriteString("          \"type\": \"string\",\n")
		sb.WriteString("          \"required\": true\n")
		sb.WriteString("        }\n")
		sb.WriteString("      ],\n")
		sb.WriteString("      \"tags\": [\"login\", \"authentication\"],\n")
		sb.WriteString("      \"group\": \"User Management\"\n")
		sb.WriteString("    }\n")
		sb.WriteString("  ],\n")
		sb.WriteString("  \"total\": 10\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n\n")
		if !isExportAll {
			sb.WriteString("**Note:** This endpoint is restricted to only return the scripts included in this SKILL.md file.\n\n")
		}
	}

	// 执行脚本接口
	apiNumber := 1
	if needListAPI {
		apiNumber = 2
	}
	sb.WriteString(fmt.Sprintf("### %d. Execute Script\n\n", apiNumber))
	sb.WriteString("Run a script with optional parameters.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/scripts/{{script_id}}/play' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\n")
	sb.WriteString("    \"params\": {\n")
	sb.WriteString("      \"parameter_name\": \"value\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }'\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**Request Body:**\n")
	sb.WriteString("- `params` (optional): Object mapping parameter names to values. Check script's parameters to see which are required.\n\n")
	sb.WriteString("**Response:**\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"message\": \"success.scriptPlaybackCompleted\",\n")
	sb.WriteString("  \"result\": {\n")
	sb.WriteString("    \"success\": true,\n")
	sb.WriteString("    \"extracted_data\": {\n")
	sb.WriteString("      \"variable_name\": \"extracted_value\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	// 如果脚本数量少于 5 个，直接列出所有脚本信息
	if len(scripts) < 5 {
		sb.WriteString("## Available Scripts\n\n")
		for i, script := range scripts {
			sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, script.Name))
			sb.WriteString(fmt.Sprintf("**ID:** `%s`\n\n", script.ID))
			if script.Description != "" {
				sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", script.Description))
			}
			if script.URL != "" {
				sb.WriteString(fmt.Sprintf("**Target URL:** `%s`\n\n", script.URL))
			}

			// 显示参数（从 MCP Schema 或 Variables 提取）
			hasParams := false
			if script.MCPInputSchema != nil {
				if properties, ok := script.MCPInputSchema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
					sb.WriteString("**Parameters:**\n")
					hasParams = true

					var required []string
					if req, ok := script.MCPInputSchema["required"].([]interface{}); ok {
						for _, r := range req {
							if rStr, ok := r.(string); ok {
								required = append(required, rStr)
							}
						}
					}
					requiredSet := make(map[string]bool)
					for _, r := range required {
						requiredSet[r] = true
					}

					for paramName, paramSchema := range properties {
						if paramMap, ok := paramSchema.(map[string]interface{}); ok {
							isRequired := requiredSet[paramName]
							requiredText := ""
							if isRequired {
								requiredText = " **(required)**"
							}

							sb.WriteString(fmt.Sprintf("- `%s`%s", paramName, requiredText))

							if paramType, ok := paramMap["type"].(string); ok {
								sb.WriteString(fmt.Sprintf(" - Type: %s", paramType))
							}
							if desc, ok := paramMap["description"].(string); ok {
								sb.WriteString(fmt.Sprintf(" - %s", desc))
							}
							if defVal, ok := paramMap["default"]; ok {
								sb.WriteString(fmt.Sprintf(" - Default: `%v`", defVal))
							}
							sb.WriteString("\n")
						}
					}
					sb.WriteString("\n")
				}
			}

			if !hasParams && len(script.Variables) > 0 {
				sb.WriteString("**Parameters:**\n")
				for varName, varValue := range script.Variables {
					if varValue != "" {
						sb.WriteString(fmt.Sprintf("- `%s` - Default: `%s`\n", varName, varValue))
					} else {
						sb.WriteString(fmt.Sprintf("- `%s` **(required)**\n", varName))
					}
				}
				sb.WriteString("\n")
			}

			if len(script.Tags) > 0 {
				sb.WriteString(fmt.Sprintf("**Tags:** %s\n\n", strings.Join(script.Tags, ", ")))
			}

			// 执行示例
			sb.WriteString("**Execution Example:**\n")
			sb.WriteString("```bash\n")
			sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/scripts/%s/play'", host, script.ID))

			// 根据参数生成示例
			needParams := false
			if script.MCPInputSchema != nil {
				if properties, ok := script.MCPInputSchema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
					needParams = true
				}
			} else if len(script.Variables) > 0 {
				needParams = true
			}

			if needParams {
				sb.WriteString(" \\\n  -H 'Content-Type: application/json' \\\n  -d '{\n    \"params\": {\n")

				if script.MCPInputSchema != nil {
					if properties, ok := script.MCPInputSchema["properties"].(map[string]interface{}); ok {
						i := 0
						propCount := len(properties)
						for paramName := range properties {
							comma := ","
							if i == propCount-1 {
								comma = ""
							}
							sb.WriteString(fmt.Sprintf("      \"%s\": \"<value>\"%s\n", paramName, comma))
							i++
						}
					}
				} else {
					i := 0
					varCount := len(script.Variables)
					for varName := range script.Variables {
						comma := ","
						if i == varCount-1 {
							comma = ""
						}
						sb.WriteString(fmt.Sprintf("      \"%s\": \"<value>\"%s\n", varName, comma))
						i++
					}
				}
				sb.WriteString("    }\n  }'")
			}
			sb.WriteString("\n```\n\n")
		}
	}

	// 使用说明
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("**Step-by-step workflow:**\n\n")
	sb.WriteString("1. **Understand the request:** Identify what web automation task the user needs.\n\n")

	if needListAPI {
		sb.WriteString("2. **Fetch script list:** Call `GET /api/v1/scripts/summary` to see all available scripts with their parameters.\n\n")
		sb.WriteString("3. **Select appropriate script:** Based on script names, descriptions, parameters, and target URLs, choose the most relevant script.\n\n")
		sb.WriteString("4. **Collect parameters:** Check the script's `parameters` array. For each parameter where `required: true`, ask the user for a value or determine it from context.\n\n")
		sb.WriteString("5. **Execute script:** Call `POST /api/v1/scripts/{script_id}/play` with parameters in the request body.\n\n")
		sb.WriteString("6. **Present results:** Parse the response and show:\n")
	} else {
		sb.WriteString("2. **Select script:** Choose from the available scripts listed above based on the user's needs.\n\n")
		sb.WriteString("3. **Check parameters:** Review the script's parameter list. Ask the user for required parameter values.\n\n")
		sb.WriteString("4. **Execute script:** Call `POST /api/v1/scripts/{script_id}/play` with parameters in the request body.\n\n")
		sb.WriteString("5. **Present results:** Parse the response and show:\n")
	}

	sb.WriteString("   - Execution status (`success: true/false`)\n")
	sb.WriteString("   - Extracted data (in `extracted_data` field)\n")
	sb.WriteString("   - Any error messages\n\n")

	// 完整示例
	if len(scripts) > 0 {
		example := scripts[0]
		sb.WriteString("## Complete Example\n\n")

		if example.Description != "" {
			sb.WriteString(fmt.Sprintf("**User Request:** \"%s\"\n\n", example.Description))
		} else {
			sb.WriteString(fmt.Sprintf("**User Request:** \"Run the %s script\"\n\n", example.Name))
		}

		sb.WriteString("**Your Actions:**\n\n")

		stepNum := 1
		if needListAPI {
			sb.WriteString(fmt.Sprintf("%d. Fetch script list:\n", stepNum))
			sb.WriteString("```bash\n")
			if isExportAll {
				sb.WriteString(fmt.Sprintf("curl -X GET 'http://%s/api/v1/scripts/summary'\n", host))
			} else {
				idsParam := strings.Join(scriptIDs, ",")
				sb.WriteString(fmt.Sprintf("curl -X GET 'http://%s/api/v1/scripts/summary?ids=%s'\n", host, idsParam))
			}
			sb.WriteString("```\n\n")
			stepNum++

			sb.WriteString(fmt.Sprintf("%d. Found script: `%s` (ID: `%s`)\n\n", stepNum, example.Name, example.ID))
			stepNum++
		} else {
			sb.WriteString(fmt.Sprintf("%d. Select script: `%s` (ID: `%s`)\n\n", stepNum, example.Name, example.ID))
			stepNum++
		}

		// 检查是否有参数
		hasParams := false
		if example.MCPInputSchema != nil {
			if properties, ok := example.MCPInputSchema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
				hasParams = true
			}
		} else if len(example.Variables) > 0 {
			hasParams = true
		}

		if hasParams {
			sb.WriteString(fmt.Sprintf("%d. Collect required parameters from user.\n\n", stepNum))
			stepNum++
		}

		sb.WriteString(fmt.Sprintf("%d. Execute the script", stepNum))
		if hasParams {
			sb.WriteString(":\n")
			sb.WriteString("```bash\n")
			sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/scripts/%s/play' \\\n", host, example.ID))
			sb.WriteString("  -H 'Content-Type: application/json' \\\n")
			sb.WriteString("  -d '{\n    \"params\": {\n")

			if example.MCPInputSchema != nil {
				if properties, ok := example.MCPInputSchema["properties"].(map[string]interface{}); ok {
					i := 0
					propCount := len(properties)
					for paramName := range properties {
						comma := ","
						if i == propCount-1 {
							comma = ""
						}
						sb.WriteString(fmt.Sprintf("      \"%s\": \"value\"%s\n", paramName, comma))
						i++
					}
				}
			} else {
				i := 0
				varCount := len(example.Variables)
				for varName := range example.Variables {
					comma := ","
					if i == varCount-1 {
						comma = ""
					}
					sb.WriteString(fmt.Sprintf("      \"%s\": \"value\"%s\n", varName, comma))
					i++
				}
			}
			sb.WriteString("    }\n  }'\n")
		} else {
			sb.WriteString(":\n")
			sb.WriteString("```bash\n")
			sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/scripts/%s/play'\n", host, example.ID))
		}
		sb.WriteString("```\n\n")
		stepNum++

		sb.WriteString(fmt.Sprintf("%d. Present the extracted data to the user.\n\n", stepNum))
	}

	// 最佳实践
	sb.WriteString("## Guidelines\n\n")
	if needListAPI {
		sb.WriteString("- **Always fetch script list first** to see available scripts and their parameters\n")
		sb.WriteString("- **Check parameter requirements** - look at the `required` field for each parameter\n")
	} else {
		sb.WriteString("- **Review script details** carefully including parameters and their requirements\n")
	}
	sb.WriteString("- **Match URLs carefully** - ensure the script's target URL matches the user's request\n")
	sb.WriteString("- **Check parameter descriptions** - they provide context about what values are expected\n")
	sb.WriteString("- **Handle errors gracefully** - if execution fails, explain the error to the user\n")
	sb.WriteString("- **Present data clearly** - format extracted data in a readable way for the user\n")
	sb.WriteString("- **Don't assume parameters** - if a required parameter is unclear, ask the user for clarification\n")
	if !isExportAll {
		sb.WriteString("- **Stay within scope** - only use the scripts provided in this SKILL.md file\n")
	}
	sb.WriteString("\n")

	// 注意事项
	sb.WriteString("## Important Notes\n\n")
	sb.WriteString("- The browser must be started before executing scripts\n")
	sb.WriteString("- Scripts run in the actual browser, so execution may take a few seconds\n")
	sb.WriteString("- Some scripts may require authentication cookies or specific browser state\n")
	sb.WriteString("- Always replace `<host>` with the actual BrowserWing API host address\n\n")

	return sb.String()
}

// ============= Executor HTTP API =============

// ExecutorHelp 获取所有可用命令的帮助信息
func (h *Handler) ExecutorHelp(c *gin.Context) {
	// 支持查询特定命令
	command := c.Query("command")

	commands := []map[string]interface{}{
		{
			"name":        "navigate",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/navigate",
			"description": "Navigate to a URL",
			"parameters": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Target URL to navigate to",
					"example":     "https://example.com",
				},
				"wait_until": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"description": "Wait condition: load, domcontentloaded, networkidle",
					"default":     "load",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"required":    false,
					"description": "Timeout in seconds",
					"default":     60,
				},
			},
			"example": map[string]interface{}{
				"url":        "https://example.com",
				"wait_until": "load",
			},
			"returns": "Operation result with semantic tree",
		},
		{
			"name":        "click",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/click",
			"description": "Click an element on the page",
			"parameters": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Element identifier: RefID (@e1, @e2 from snapshot), CSS selector, XPath, or text content",
					"example":     "@e1 or #button-id",
				},
				"wait_visible": map[string]interface{}{
					"type":        "boolean",
					"required":    false,
					"description": "Wait for element to be visible",
					"default":     true,
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"required":    false,
					"description": "Timeout in seconds",
					"default":     10,
				},
			},
			"example": map[string]interface{}{
				"identifier":   "#login-button",
				"wait_visible": true,
			},
			"returns": "Operation result with updated semantic tree",
		},
		{
			"name":        "type",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/type",
			"description": "Type text into an input element",
			"parameters": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Input element identifier",
					"example":     "@e3 or #email-input",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Text to type",
					"example":     "user@example.com",
				},
				"clear": map[string]interface{}{
					"type":        "boolean",
					"required":    false,
					"description": "Clear existing content first",
					"default":     true,
				},
			},
			"example": map[string]interface{}{
				"identifier": "#email-input",
				"text":       "user@example.com",
				"clear":      true,
			},
			"returns": "Operation result",
		},
		{
			"name":        "select",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/select",
			"description": "Select an option from a dropdown",
			"parameters": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Select element identifier",
				},
				"value": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Option value or text to select",
				},
			},
			"example": map[string]interface{}{
				"identifier": "#country-select",
				"value":      "United States",
			},
		},
		{
			"name":        "extract",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/extract",
			"description": "Extract data from page elements",
			"parameters": map[string]interface{}{
				"selector": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "CSS selector for elements to extract",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"description": "Extraction type: text, html, attribute, property",
				},
				"fields": map[string]interface{}{
					"type":        "array",
					"required":    false,
					"description": "Fields to extract: text, html, href, src, value",
					"example":     []string{"text", "href"},
				},
				"multiple": map[string]interface{}{
					"type":        "boolean",
					"required":    false,
					"description": "Extract multiple elements",
					"default":     false,
				},
			},
			"example": map[string]interface{}{
				"selector": ".product-item",
				"fields":   []string{"text", "href"},
				"multiple": true,
			},
			"returns": "Extracted data array or object",
		},
		{
			"name":        "wait",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/wait",
			"description": "Wait for an element to reach a certain state",
			"parameters": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Element identifier",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"description": "State to wait for: visible, hidden, enabled",
					"default":     "visible",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"required":    false,
					"description": "Timeout in seconds",
					"default":     30,
				},
			},
			"example": map[string]interface{}{
				"identifier": "#loading-spinner",
				"state":      "hidden",
				"timeout":    10,
			},
		},
		{
			"name":        "hover",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/hover",
			"description": "Hover mouse over an element",
			"parameters": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Element identifier",
				},
			},
			"example": map[string]interface{}{
				"identifier": ".dropdown-trigger",
			},
		},
		{
			"name":        "press-key",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/press-key",
			"description": "Press a keyboard key",
			"parameters": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Key to press: Enter, Tab, Escape, ArrowDown, etc.",
					"example":     "Enter",
				},
				"ctrl": map[string]interface{}{
					"type":        "boolean",
					"required":    false,
					"description": "Hold Ctrl key",
				},
				"shift": map[string]interface{}{
					"type":        "boolean",
					"required":    false,
					"description": "Hold Shift key",
				},
			},
			"example": map[string]interface{}{
				"key": "Enter",
			},
		},
		{
			"name":        "scroll-to-bottom",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/scroll-to-bottom",
			"description": "Scroll to the bottom of the page",
			"parameters":  map[string]interface{}{},
			"example":     map[string]interface{}{},
		},
		{
			"name":        "go-back",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/go-back",
			"description": "Navigate back in browser history",
			"parameters":  map[string]interface{}{},
		},
		{
			"name":        "go-forward",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/go-forward",
			"description": "Navigate forward in browser history",
			"parameters":  map[string]interface{}{},
		},
		{
			"name":        "reload",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/reload",
			"description": "Reload the current page",
			"parameters":  map[string]interface{}{},
		},
		{
			"name":        "get-text",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/get-text",
			"description": "Get text content of an element",
			"parameters": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Element identifier",
				},
			},
			"example": map[string]interface{}{
				"identifier": "h1",
			},
			"returns": "Text content",
		},
		{
			"name":        "get-value",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/get-value",
			"description": "Get value of an input element",
			"parameters": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Input element identifier",
				},
			},
			"returns": "Input value",
		},
		{
			"name":        "snapshot",
			"method":      "GET",
			"endpoint":    "/api/v1/executor/snapshot",
			"description": "Get the accessibility snapshot of the current page (all interactive elements)",
			"parameters":  map[string]interface{}{},
			"returns":     "Accessibility snapshot with all clickable and input elements",
			"note":        "Use this first to understand page structure and get element indices. The accessibility tree is cleaner than raw DOM.",
		},
		{
			"name":        "clickable-elements",
			"method":      "GET",
			"endpoint":    "/api/v1/executor/clickable-elements",
			"description": "Get all clickable elements (buttons, links, etc.)",
			"parameters":  map[string]interface{}{},
			"returns":     "Array of clickable elements with indices",
		},
		{
			"name":        "input-elements",
			"method":      "GET",
			"endpoint":    "/api/v1/executor/input-elements",
			"description": "Get all input elements (text boxes, selects, etc.)",
			"parameters":  map[string]interface{}{},
			"returns":     "Array of input elements with indices",
		},
		{
			"name":        "page-info",
			"method":      "GET",
			"endpoint":    "/api/v1/executor/page-info",
			"description": "Get current page URL and title",
			"parameters":  map[string]interface{}{},
			"returns":     "Page info (url, title)",
		},
		{
			"name":        "page-text",
			"method":      "GET",
			"endpoint":    "/api/v1/executor/page-text",
			"description": "Get all text content from the page",
			"parameters":  map[string]interface{}{},
			"returns":     "Page text content",
		},
		{
			"name":        "page-content",
			"method":      "GET",
			"endpoint":    "/api/v1/executor/page-content",
			"description": "Get full HTML content of the page",
			"parameters":  map[string]interface{}{},
			"returns":     "HTML content",
		},
		{
			"name":        "screenshot",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/screenshot",
			"description": "Take a screenshot of the page",
			"parameters": map[string]interface{}{
				"full_page": map[string]interface{}{
					"type":        "boolean",
					"required":    false,
					"description": "Capture full page or viewport only",
					"default":     false,
				},
				"format": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"description": "Image format: png or jpeg",
					"default":     "png",
				},
			},
			"returns": "Base64 encoded image data",
		},
		{
			"name":        "evaluate",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/evaluate",
			"description": "Execute JavaScript code on the page",
			"parameters": map[string]interface{}{
				"script": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "JavaScript code to execute",
					"example":     "() => document.title",
				},
			},
			"returns": "Script execution result",
		},
		{
			"name":        "batch",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/batch",
			"description": "Execute multiple operations in sequence",
			"parameters": map[string]interface{}{
				"operations": map[string]interface{}{
					"type":        "array",
					"required":    true,
					"description": "Array of operations to execute",
					"example": []map[string]interface{}{
						{
							"type":          "navigate",
							"params":        map[string]interface{}{"url": "https://example.com"},
							"stop_on_error": true,
						},
						{
							"type":          "click",
							"params":        map[string]interface{}{"identifier": "#button"},
							"stop_on_error": true,
						},
					},
				},
			},
			"returns": "Batch execution results",
		},
		{
			"name":        "tabs",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/tabs",
			"description": "Manage browser tabs (list, create, switch, close)",
			"parameters": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Tab action: list, new, switch, close",
					"example":     "list",
				},
				"url": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"description": "URL for new tab (required when action='new')",
					"example":     "https://example.com",
				},
				"index": map[string]interface{}{
					"type":        "number",
					"required":    false,
					"description": "Tab index for switch/close (0-based)",
					"example":     1,
				},
			},
			"example": map[string]interface{}{
				"action": "list",
			},
			"returns": "Tab operation result with tab information",
		},
		{
			"name":        "fill-form",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/fill-form",
			"description": "Intelligently fill out web forms with multiple fields",
			"parameters": map[string]interface{}{
				"fields": map[string]interface{}{
					"type":        "array",
					"required":    true,
					"description": "Array of form fields to fill",
					"example": []map[string]interface{}{
						{"name": "username", "value": "john@example.com"},
						{"name": "password", "value": "secret123"},
					},
				},
				"submit": map[string]interface{}{
					"type":        "boolean",
					"required":    false,
					"description": "Auto-submit form after filling",
					"default":     false,
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"required":    false,
					"description": "Timeout per field in seconds",
					"default":     10,
				},
			},
			"example": map[string]interface{}{
				"fields": []map[string]interface{}{
					{"name": "email", "value": "user@example.com"},
					{"name": "password", "value": "secret123"},
				},
				"submit": true,
			},
			"returns": "Form fill results with success/error details",
		},
		{
			"name":        "resize",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/resize",
			"description": "Resize the browser window",
			"parameters": map[string]interface{}{
				"width": map[string]interface{}{
					"type":        "number",
					"required":    true,
					"description": "Window width in pixels",
					"example":     1920,
				},
				"height": map[string]interface{}{
					"type":        "number",
					"required":    true,
					"description": "Window height in pixels",
					"example":     1080,
				},
			},
			"example": map[string]interface{}{
				"width":  1920,
				"height": 1080,
			},
			"returns": "Operation result",
		},
		{
			"name":        "console-messages",
			"method":      "GET",
			"endpoint":    "/api/v1/executor/console-messages",
			"description": "Get console messages from the browser (logs, warnings, errors)",
			"parameters":  map[string]interface{}{},
			"returns":     "Array of console messages with type, text, and timestamp",
			"note":        "Useful for debugging JavaScript errors or monitoring console output",
		},
		{
			"name":        "network-requests",
			"method":      "GET",
			"endpoint":    "/api/v1/executor/network-requests",
			"description": "Get network requests made by the page (XHR, Fetch, etc.)",
			"parameters":  map[string]interface{}{},
			"returns":     "Array of network requests with URL, method, status, and response",
			"note":        "Useful for API monitoring and debugging network issues",
		},
		{
			"name":        "handle-dialog",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/handle-dialog",
			"description": "Configure how to handle JavaScript dialogs (alert, confirm, prompt)",
			"parameters": map[string]interface{}{
				"accept": map[string]interface{}{
					"type":        "boolean",
					"required":    true,
					"description": "Whether to accept the dialog (true) or dismiss it (false)",
					"example":     true,
				},
				"text": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"description": "Text to enter for prompt dialogs",
					"example":     "User input text",
				},
			},
			"example": map[string]interface{}{
				"accept": true,
				"text":   "Hello",
			},
			"returns": "Operation result",
			"note":    "Must be called before the dialog appears. Affects next dialog interaction.",
		},
		{
			"name":        "file-upload",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/file-upload",
			"description": "Upload files to a file input element",
			"parameters": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "File input element identifier",
					"example":     "#file-input",
				},
				"file_paths": map[string]interface{}{
					"type":        "array",
					"required":    true,
					"description": "Array of file paths to upload (absolute paths)",
					"example":     []string{"/path/to/file1.pdf", "/path/to/file2.jpg"},
				},
			},
			"example": map[string]interface{}{
				"identifier": "#file-input",
				"file_paths": []string{"/path/to/document.pdf"},
			},
			"returns": "Operation result",
		},
		{
			"name":        "drag",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/drag",
			"description": "Drag an element to another element (drag and drop)",
			"parameters": map[string]interface{}{
				"from_identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Source element identifier to drag",
					"example":     "#drag-item",
				},
				"to_identifier": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Target element identifier to drop onto",
					"example":     "#drop-zone",
				},
			},
			"example": map[string]interface{}{
				"from_identifier": "#drag-item",
				"to_identifier":   "#drop-zone",
			},
			"returns": "Operation result",
		},
		{
			"name":        "close-page",
			"method":      "POST",
			"endpoint":    "/api/v1/executor/close-page",
			"description": "Close the current browser page/tab",
			"parameters":  map[string]interface{}{},
			"returns":     "Operation result",
			"note":        "Use with caution. After closing, you may need to switch to another tab.",
		},
	}

	// 如果指定了特定命令，只返回该命令的信息
	if command != "" {
		for _, cmd := range commands {
			if cmd["name"] == command {
				c.JSON(http.StatusOK, gin.H{
					"command": cmd,
				})
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Command not found",
		})
		return
	}

	// 返回所有命令
	c.JSON(http.StatusOK, gin.H{
		"total_commands": len(commands),
		"base_url":       "/api/v1/executor",
		"authentication": map[string]interface{}{
			"methods": []string{"JWT Token", "API Key"},
			"jwt":     "Authorization: Bearer <token>",
			"api_key": "X-BrowserWing-Key: <api-key>",
		},
		"workflow": []string{
			"1. Call GET /snapshot to understand page structure",
			"2. Use element RefIDs (@e1, @e2) or CSS selectors for operations",
			"3. Call appropriate operation endpoints (navigate, click, type, etc.)",
			"4. Extract data using /extract endpoint",
			"5. Use /batch for multiple operations",
		},
		"element_identifiers": map[string]interface{}{
			"refid":          "@e1, @e2, @e3 (from /snapshot, recommended)",
			"css_selector":   "#id, .class, button[type='submit']",
			"xpath":          "//button[@id='login'], //a[contains(text(), 'Login')]",
			"text_content":   "Login, Sign Up (will find button/link with this text)",
			"aria_label":     "Searches for elements with aria-label attribute",
			"recommendation": "Use /snapshot first to get RefIDs (@e1, @e2, etc.)",
		},
		"commands": commands,
		"examples": map[string]interface{}{
			"simple_workflow": map[string]interface{}{
				"description": "Navigate and click a button",
				"steps": []map[string]interface{}{
					{
						"step":     1,
						"action":   "Navigate",
						"endpoint": "POST /navigate",
						"payload":  map[string]interface{}{"url": "https://example.com"},
					},
					{
						"step":     2,
						"action":   "Get page structure",
						"endpoint": "GET /snapshot",
					},
					{
						"step":     3,
						"action":   "Click button",
						"endpoint": "POST /click",
						"payload":  map[string]interface{}{"identifier": "@e1"},
					},
				},
			},
			"data_extraction": map[string]interface{}{
				"description": "Search and extract results",
				"steps": []map[string]interface{}{
					{
						"step":     1,
						"endpoint": "POST /navigate",
						"payload":  map[string]interface{}{"url": "https://example.com/search"},
					},
					{
						"step":     2,
						"endpoint": "POST /type",
						"payload":  map[string]interface{}{"identifier": "#search", "text": "query"},
					},
					{
						"step":     3,
						"endpoint": "POST /press-key",
						"payload":  map[string]interface{}{"key": "Enter"},
					},
					{
						"step":     4,
						"endpoint": "POST /wait",
						"payload":  map[string]interface{}{"identifier": ".results", "state": "visible"},
					},
					{
						"step":     5,
						"endpoint": "POST /extract",
						"payload":  map[string]interface{}{"selector": ".item", "fields": []string{"text", "href"}, "multiple": true},
					},
				},
			},
		},
	})
}

// ExecutorNavigate 导航到指定 URL
func (h *Handler) ExecutorNavigate(c *gin.Context) {
	var req struct {
		URL       string `json:"url" binding:"required"`
		WaitUntil string `json:"wait_until"` // load, domcontentloaded, networkidle
		Timeout   int    `json:"timeout"`    // 秒
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	// 创建 executor 实例
	executor := h.executor.WithContext(c.Request.Context())

	// 设置选项
	var opts *executor2.NavigateOptions
	if req.WaitUntil != "" || req.Timeout > 0 {
		opts = &executor2.NavigateOptions{}
		if req.WaitUntil != "" {
			opts.WaitUntil = req.WaitUntil
		}
		if req.Timeout > 0 {
			opts.Timeout = time.Duration(req.Timeout) * time.Second
		}
	}

	// 执行导航
	result, err := executor.Navigate(c.Request.Context(), req.URL, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.navigationFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorClick 点击元素
func (h *Handler) ExecutorClick(c *gin.Context) {
	var req struct {
		Identifier  string `json:"identifier" binding:"required"` // CSS selector, XPath, label, etc.
		WaitVisible bool   `json:"wait_visible"`
		WaitEnabled bool   `json:"wait_enabled"`
		Timeout     int    `json:"timeout"` // 秒
		Button      string `json:"button"`  // left, right, middle
		ClickCount  int    `json:"click_count"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.ClickOptions{
		WaitVisible: req.WaitVisible,
		WaitEnabled: req.WaitEnabled,
		Button:      req.Button,
		ClickCount:  req.ClickCount,
	}
	if req.Timeout > 0 {
		opts.Timeout = time.Duration(req.Timeout) * time.Second
	}

	result, err := executor.Click(c.Request.Context(), req.Identifier, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.clickFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorType 输入文本
func (h *Handler) ExecutorType(c *gin.Context) {
	var req struct {
		Identifier  string `json:"identifier" binding:"required"` // CSS selector, XPath, label, etc.
		Text        string `json:"text" binding:"required"`
		Clear       bool   `json:"clear"`
		WaitVisible bool   `json:"wait_visible"`
		Timeout     int    `json:"timeout"` // 秒
		Delay       int    `json:"delay"`   // 毫秒
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.TypeOptions{
		Clear:       req.Clear,
		WaitVisible: req.WaitVisible,
	}
	if req.Timeout > 0 {
		opts.Timeout = time.Duration(req.Timeout) * time.Second
	}
	if req.Delay > 0 {
		opts.Delay = time.Duration(req.Delay) * time.Millisecond
	}

	result, err := executor.Type(c.Request.Context(), req.Identifier, req.Text, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.typeFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorSelect 选择下拉框选项
func (h *Handler) ExecutorSelect(c *gin.Context) {
	var req struct {
		Identifier  string `json:"identifier" binding:"required"`
		Value       string `json:"value" binding:"required"`
		WaitVisible bool   `json:"wait_visible"`
		Timeout     int    `json:"timeout"` // 秒
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.SelectOptions{
		WaitVisible: req.WaitVisible,
	}
	if req.Timeout > 0 {
		opts.Timeout = time.Duration(req.Timeout) * time.Second
	}

	result, err := executor.Select(c.Request.Context(), req.Identifier, req.Value, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.selectFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorGetText 获取元素文本
func (h *Handler) ExecutorGetText(c *gin.Context) {
	var req struct {
		Identifier string `json:"identifier" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GetText(c.Request.Context(), req.Identifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getTextFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorGetValue 获取元素值
func (h *Handler) ExecutorGetValue(c *gin.Context) {
	var req struct {
		Identifier string `json:"identifier" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GetValue(c.Request.Context(), req.Identifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getValueFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorWaitFor 等待元素
func (h *Handler) ExecutorWaitFor(c *gin.Context) {
	var req struct {
		Identifier string `json:"identifier" binding:"required"`
		State      string `json:"state"`   // visible, hidden, enabled
		Timeout    int    `json:"timeout"` // 秒
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.WaitForOptions{
		State: req.State,
	}
	if req.Timeout > 0 {
		opts.Timeout = time.Duration(req.Timeout) * time.Second
	}

	result, err := executor.WaitFor(c.Request.Context(), req.Identifier, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.waitForFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorExtract 提取数据
func (h *Handler) ExecutorExtract(c *gin.Context) {
	var req struct {
		Selector string   `json:"selector" binding:"required"`
		Type     string   `json:"type"`     // text, html, attribute, property
		Attr     string   `json:"attr"`     // 当 type 为 attribute 或 property 时使用
		Fields   []string `json:"fields"`   // 要提取的字段列表
		Multiple bool     `json:"multiple"` // 是否提取多个元素
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.ExtractOptions{
		Selector: req.Selector,
		Type:     req.Type,
		Attr:     req.Attr,
		Fields:   req.Fields,
		Multiple: req.Multiple,
	}

	result, err := executor.Extract(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.extractFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorHover 鼠标悬停
func (h *Handler) ExecutorHover(c *gin.Context) {
	var req struct {
		Identifier  string `json:"identifier" binding:"required"`
		WaitVisible bool   `json:"wait_visible"`
		Timeout     int    `json:"timeout"` // 秒
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.HoverOptions{
		WaitVisible: req.WaitVisible,
	}
	if req.Timeout > 0 {
		opts.Timeout = time.Duration(req.Timeout) * time.Second
	}

	result, err := executor.Hover(c.Request.Context(), req.Identifier, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.hoverFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorScrollToBottom 滚动到底部
func (h *Handler) ExecutorScrollToBottom(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.ScrollToBottom(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.scrollFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorGoBack 后退
func (h *Handler) ExecutorGoBack(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GoBack(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.goBackFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorGoForward 前进
func (h *Handler) ExecutorGoForward(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GoForward(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.goForwardFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorReload 刷新页面
func (h *Handler) ExecutorReload(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.Reload(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.reloadFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorScreenshot 截图
func (h *Handler) ExecutorScreenshot(c *gin.Context) {
	var req struct {
		FullPage bool   `json:"full_page"`
		Quality  int    `json:"quality"` // 1-100
		Format   string `json:"format"`  // png, jpeg
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.ScreenshotOptions{
		FullPage: req.FullPage,
		Quality:  req.Quality,
		Format:   req.Format,
	}

	result, err := executor.Screenshot(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.screenshotFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorEvaluate 执行 JavaScript
func (h *Handler) ExecutorEvaluate(c *gin.Context) {
	var req struct {
		Script string `json:"script" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.Evaluate(c.Request.Context(), req.Script)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.evaluateFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorPressKey 按键
func (h *Handler) ExecutorPressKey(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"` // enter, tab, escape, etc.
		Ctrl  bool   `json:"ctrl"`
		Shift bool   `json:"shift"`
		Alt   bool   `json:"alt"`
		Meta  bool   `json:"meta"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.PressKeyOptions{
		Ctrl:  req.Ctrl,
		Shift: req.Shift,
		Alt:   req.Alt,
		Meta:  req.Meta,
	}

	result, err := executor.PressKey(c.Request.Context(), req.Key, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.pressKeyFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorResize 调整窗口大小
func (h *Handler) ExecutorResize(c *gin.Context) {
	var req struct {
		Width  int `json:"width" binding:"required"`
		Height int `json:"height" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.Resize(c.Request.Context(), req.Width, req.Height)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.resizeFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorGetPageInfo 获取页面信息
func (h *Handler) ExecutorGetPageInfo(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GetPageInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getPageInfoFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorGetPageContent 获取页面内容
func (h *Handler) ExecutorGetPageContent(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GetPageContent(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getPageContentFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorGetPageText 获取页面文本
func (h *Handler) ExecutorGetPageText(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GetPageText(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getPageTextFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorGetAccessibilitySnapshot 获取可访问性快照
func (h *Handler) ExecutorGetAccessibilitySnapshot(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	snapshot, err := executor.GetAccessibilitySnapshot(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getAccessibilitySnapshotFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"snapshot": snapshot.SerializeToSimpleText(),
	})
}

// ExecutorGetClickableElements 获取可点击元素
func (h *Handler) ExecutorGetClickableElements(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	elements, err := executor.GetClickableElements(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getClickableElementsFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"elements": elements,
		"count":    len(elements),
	})
}

// ExecutorGetInputElements 获取输入元素
func (h *Handler) ExecutorGetInputElements(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	elements, err := executor.GetInputElements(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getInputElementsFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"elements": elements,
		"count":    len(elements),
	})
}

// ExecutorBatch 批量执行操作
func (h *Handler) ExecutorBatch(c *gin.Context) {
	var req struct {
		Operations []executor2.Operation `json:"operations" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.ExecuteBatch(c.Request.Context(), req.Operations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.batchExecutionFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorTabs 标签页管理
func (h *Handler) ExecutorTabs(c *gin.Context) {
	var req struct {
		Action string `json:"action" binding:"required"` // list, new, switch, close
		URL    string `json:"url"`                       // for new action
		Index  int    `json:"index"`                     // for switch/close action
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.TabsOptions{
		Action: executor2.TabsAction(req.Action),
		URL:    req.URL,
		Index:  req.Index,
	}

	result, err := executor.Tabs(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.tabsOperationFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorFillForm 批量填写表单
func (h *Handler) ExecutorFillForm(c *gin.Context) {
	var req struct {
		Fields  []executor2.FormField `json:"fields" binding:"required"`
		Submit  bool                  `json:"submit"`
		Timeout int                   `json:"timeout"` // 秒
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())

	opts := &executor2.FillFormOptions{
		Fields: req.Fields,
		Submit: req.Submit,
	}
	if req.Timeout > 0 {
		opts.Timeout = time.Duration(req.Timeout) * time.Second
	}

	result, err := executor.FillForm(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.fillFormFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorConsoleMessages 获取控制台消息
func (h *Handler) ExecutorConsoleMessages(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GetConsoleMessages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getConsoleMessagesFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorNetworkRequests 获取网络请求
func (h *Handler) ExecutorNetworkRequests(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.GetNetworkRequests(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.getNetworkRequestsFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorHandleDialog 处理对话框
func (h *Handler) ExecutorHandleDialog(c *gin.Context) {
	var req struct {
		Accept bool   `json:"accept" binding:"required"`
		Text   string `json:"text"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.HandleDialog(c.Request.Context(), req.Accept, req.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.handleDialogFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorFileUpload 文件上传
func (h *Handler) ExecutorFileUpload(c *gin.Context) {
	var req struct {
		Identifier string   `json:"identifier" binding:"required"`
		FilePaths  []string `json:"file_paths" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.FileUpload(c.Request.Context(), req.Identifier, req.FilePaths)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.fileUploadFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorDrag 拖拽元素
func (h *Handler) ExecutorDrag(c *gin.Context) {
	var req struct {
		FromIdentifier string `json:"from_identifier" binding:"required"`
		ToIdentifier   string `json:"to_identifier" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest"})
		return
	}

	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.Drag(c.Request.Context(), req.FromIdentifier, req.ToIdentifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.dragFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecutorClosePage 关闭当前页面
func (h *Handler) ExecutorClosePage(c *gin.Context) {
	executor := h.executor.WithContext(c.Request.Context())
	result, err := executor.ClosePage(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "error.closePageFailed",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExportExecutorSkill 导出 Executor API 为 Claude Skills 的 SKILL.md 格式
func (h *Handler) ExportExecutorSkill(c *gin.Context) {
	// 获取服务端地址
	host := c.Request.Host

	// 生成 SKILL.md 内容
	skillContent := generateExecutorSkillMD(host)

	fileName := fmt.Sprintf("EXECUTOR_SKILL_%s.md", time.Now().Format("20060102150405"))

	// 返回内容
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.String(http.StatusOK, skillContent)
}

// generateExecutorSkillMD 生成 Executor API 的 SKILL.md 内容
func generateExecutorSkillMD(host string) string {
	var sb strings.Builder

	// YAML Frontmatter
	sb.WriteString("---\n")
	sb.WriteString("name: browserwing-executor\n")
	sb.WriteString("description: Control browser automation through HTTP API. Supports page navigation, element interaction (click, type, select), data extraction, accessibility snapshot analysis, screenshot, JavaScript execution, and batch operations.\n")
	sb.WriteString("---\n\n")

	// 主标题
	sb.WriteString("# BrowserWing Executor API\n\n")

	// 简介
	sb.WriteString("## Overview\n\n")
	sb.WriteString("BrowserWing Executor provides comprehensive browser automation capabilities through HTTP APIs. You can control browser navigation, interact with page elements, extract data, and analyze page structure.\n\n")
	sb.WriteString(fmt.Sprintf("**API Base URL:** `http://%s/api/v1/executor`\n\n", host))
	sb.WriteString("**Authentication:** Use `X-BrowserWing-Key: <api-key>` header or `Authorization: Bearer <token>`\n\n")

	// 核心功能
	sb.WriteString("## Core Capabilities\n\n")
	sb.WriteString("- **Page Navigation:** Navigate to URLs, go back/forward, reload\n")
	sb.WriteString("- **Element Interaction:** Click, type, select, hover on page elements\n")
	sb.WriteString("- **Data Extraction:** Extract text, attributes, values from elements\n")
	sb.WriteString("- **Accessibility Analysis:** Get accessibility snapshot to understand page structure\n")
	sb.WriteString("- **Advanced Operations:** Screenshot, JavaScript execution, keyboard input\n")
	sb.WriteString("- **Batch Processing:** Execute multiple operations in sequence\n\n")

	// API 端点
	sb.WriteString("## API Endpoints\n\n")

	// 1. 发现命令
	sb.WriteString("### 1. Discover Available Commands\n\n")
	sb.WriteString("**IMPORTANT:** Always call this endpoint first to see all available commands and their parameters.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X GET 'http://%s/api/v1/executor/help'\n", host))
	sb.WriteString("```\n\n")
	sb.WriteString("**Response:** Returns complete list of all commands with parameters, examples, and usage guidelines.\n\n")
	sb.WriteString("**Query specific command:**\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X GET 'http://%s/api/v1/executor/help?command=extract'\n", host))
	sb.WriteString("```\n\n")

	// 2. 获取可访问性快照
	sb.WriteString("### 2. Get Accessibility Snapshot\n\n")
	sb.WriteString("**CRITICAL:** Always call this after navigation to understand page structure and get element RefIDs.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X GET 'http://%s/api/v1/executor/snapshot'\n", host))
	sb.WriteString("```\n\n")
	sb.WriteString("**Response Example:**\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"success\": true,\n")
	sb.WriteString("  \"snapshot_text\": \"Clickable Elements:\\n  @e1 Login (role: button)\\n  @e2 Sign Up (role: link)\\n\\nInput Elements:\\n  @e3 Email (role: textbox) [placeholder: your@email.com]\\n  @e4 Password (role: textbox)\"\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**Use Cases:**\n")
	sb.WriteString("- Understand what interactive elements are on the page\n")
	sb.WriteString("- Get element RefIDs (@e1, @e2, etc.) for precise identification\n")
	sb.WriteString("- See element labels, roles, and attributes\n")
	sb.WriteString("- The accessibility tree is cleaner than raw DOM and better for LLMs\n")
	sb.WriteString("- RefIDs are stable references that work reliably across page changes\n\n")

	// 3. 主要操作端点
	sb.WriteString("### 3. Common Operations\n\n")

	// Navigate
	sb.WriteString("#### Navigate to URL\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/navigate' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\"url\": \"https://example.com\"}'\n")
	sb.WriteString("```\n\n")

	// Click
	sb.WriteString("#### Click Element\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/click' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\"identifier\": \"@e1\"}'\n")
	sb.WriteString("```\n")
	sb.WriteString("**Identifier formats:**\n")
	sb.WriteString("- **RefID (Recommended):** `@e1`, `@e2` (from snapshot)\n")
	sb.WriteString("- **CSS Selector:** `#button-id`, `.class-name`\n")
	sb.WriteString("- **XPath:** `//button[@type='submit']`\n")
	sb.WriteString("- **Text:** `Login` (text content)\n\n")

	// Type
	sb.WriteString("#### Type Text\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/type' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\"identifier\": \"@e3\", \"text\": \"user@example.com\"}'\n")
	sb.WriteString("```\n\n")

	// Extract
	sb.WriteString("#### Extract Data\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/extract' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\n")
	sb.WriteString("    \"selector\": \".product-item\",\n")
	sb.WriteString("    \"fields\": [\"text\", \"href\"],\n")
	sb.WriteString("    \"multiple\": true\n")
	sb.WriteString("  }'\n")
	sb.WriteString("```\n\n")

	// Wait
	sb.WriteString("#### Wait for Element\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/wait' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\"identifier\": \".loading\", \"state\": \"hidden\", \"timeout\": 10}'\n")
	sb.WriteString("```\n\n")

	// Batch
	sb.WriteString("#### Batch Operations\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/batch' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\n")
	sb.WriteString("    \"operations\": [\n")
	sb.WriteString("      {\"type\": \"navigate\", \"params\": {\"url\": \"https://example.com\"}, \"stop_on_error\": true},\n")
	sb.WriteString("      {\"type\": \"click\", \"params\": {\"identifier\": \"@e1\"}, \"stop_on_error\": true},\n")
	sb.WriteString("      {\"type\": \"type\", \"params\": {\"identifier\": \"@e3\", \"text\": \"query\"}, \"stop_on_error\": true}\n")
	sb.WriteString("    ]\n")
	sb.WriteString("  }'\n")
	sb.WriteString("```\n\n")

	// 使用说明
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("**Step-by-step workflow:**\n\n")
	sb.WriteString("1. **Discover commands:** Call `GET /help` to see all available operations and their parameters (do this first if unsure).\n\n")
	sb.WriteString("2. **Navigate:** Use `POST /navigate` to open the target webpage.\n\n")
	sb.WriteString("3. **Analyze page:** Call `GET /snapshot` to understand page structure and get element RefIDs.\n\n")
	sb.WriteString("4. **Interact:** Use element RefIDs (like `@e1`, `@e2`) or CSS selectors to:\n")
	sb.WriteString("   - Click elements: `POST /click`\n")
	sb.WriteString("   - Input text: `POST /type`\n")
	sb.WriteString("   - Select options: `POST /select`\n")
	sb.WriteString("   - Wait for elements: `POST /wait`\n\n")
	sb.WriteString("5. **Extract data:** Use `POST /extract` to get information from the page.\n\n")
	sb.WriteString("6. **Present results:** Format and show extracted data to the user.\n\n")

	// 完整示例
	sb.WriteString("## Complete Example\n\n")
	sb.WriteString("**User Request:** \"Search for 'laptop' on example.com and get the first 5 results\"\n\n")
	sb.WriteString("**Your Actions:**\n\n")

	sb.WriteString("1. Navigate to search page:\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/navigate' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\"url\": \"https://example.com/search\"}'\n")
	sb.WriteString("```\n\n")

	sb.WriteString("2. Get page structure to find search input:\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X GET 'http://%s/api/v1/executor/snapshot'\n", host))
	sb.WriteString("```\n")
	sb.WriteString("Response shows: `@e3 Search (role: textbox) [placeholder: Search...]`\n\n")

	sb.WriteString("3. Type search query:\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/type' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\"identifier\": \"@e3\", \"text\": \"laptop\"}'\n")
	sb.WriteString("```\n\n")

	sb.WriteString("4. Press Enter to submit:\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/press-key' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\"key\": \"Enter\"}'\n")
	sb.WriteString("```\n\n")

	sb.WriteString("5. Wait for results to load:\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/wait' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\"identifier\": \".search-results\", \"state\": \"visible\", \"timeout\": 10}'\n")
	sb.WriteString("```\n\n")

	sb.WriteString("6. Extract search results:\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/extract' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\n")
	sb.WriteString("    \"selector\": \".result-item\",\n")
	sb.WriteString("    \"fields\": [\"text\", \"href\"],\n")
	sb.WriteString("    \"multiple\": true\n")
	sb.WriteString("  }'\n")
	sb.WriteString("```\n\n")

	sb.WriteString("7. Present the extracted data:\n")
	sb.WriteString("```\n")
	sb.WriteString("Found 15 results for 'laptop':\n")
	sb.WriteString("1. Gaming Laptop - $1299 (https://...)\n")
	sb.WriteString("2. Business Laptop - $899 (https://...)\n")
	sb.WriteString("...\n")
	sb.WriteString("```\n\n")

	// 关键命令速查
	sb.WriteString("## Key Commands Reference\n\n")

	// 导航类
	sb.WriteString("### Navigation\n")
	sb.WriteString("- `POST /navigate` - Navigate to URL\n")
	sb.WriteString("- `POST /go-back` - Go back in history\n")
	sb.WriteString("- `POST /go-forward` - Go forward in history\n")
	sb.WriteString("- `POST /reload` - Reload current page\n\n")

	// 元素交互类
	sb.WriteString("### Element Interaction\n")
	sb.WriteString("- `POST /click` - Click element (supports: RefID `@e1`, CSS selector, XPath, text content)\n")
	sb.WriteString("- `POST /type` - Type text into input (supports: RefID `@e3`, CSS selector, XPath)\n")
	sb.WriteString("- `POST /select` - Select dropdown option\n")
	sb.WriteString("- `POST /hover` - Hover over element\n")
	sb.WriteString("- `POST /wait` - Wait for element state (visible, hidden, enabled)\n")
	sb.WriteString("- `POST /press-key` - Press keyboard key (Enter, Tab, Ctrl+S, etc.)\n\n")

	// 数据提取类
	sb.WriteString("### Data Extraction\n")
	sb.WriteString("- `POST /extract` - Extract data from elements (supports multiple elements, custom fields)\n")
	sb.WriteString("- `POST /get-text` - Get element text content\n")
	sb.WriteString("- `POST /get-value` - Get input element value\n")
	sb.WriteString("- `GET /page-info` - Get page URL and title\n")
	sb.WriteString("- `GET /page-text` - Get all page text\n")
	sb.WriteString("- `GET /page-content` - Get full HTML\n\n")

	// 页面分析类
	sb.WriteString("### Page Analysis\n")
	sb.WriteString("- `GET /snapshot` - Get accessibility snapshot (⭐ **ALWAYS call after navigation**)\n")
	sb.WriteString("- `GET /clickable-elements` - Get all clickable elements\n")
	sb.WriteString("- `GET /input-elements` - Get all input elements\n\n")

	// 高级功能类
	sb.WriteString("### Advanced\n")
	sb.WriteString("- `POST /screenshot` - Take page screenshot (base64 encoded)\n")
	sb.WriteString("- `POST /evaluate` - Execute JavaScript code\n")
	sb.WriteString("- `POST /batch` - Execute multiple operations in sequence\n")
	sb.WriteString("- `POST /scroll-to-bottom` - Scroll to page bottom\n")
	sb.WriteString("- `POST /resize` - Resize browser window\n")
	sb.WriteString("- `POST /tabs` - Manage browser tabs (list, new, switch, close)\n")
	sb.WriteString("- `POST /fill-form` - Intelligently fill multiple form fields at once\n\n")

	// 调试和监控类
	sb.WriteString("### Debug & Monitoring\n")
	sb.WriteString("- `GET /console-messages` - Get browser console messages (logs, warnings, errors)\n")
	sb.WriteString("- `GET /network-requests` - Get network requests made by the page\n")
	sb.WriteString("- `POST /handle-dialog` - Configure JavaScript dialog (alert, confirm, prompt) handling\n")
	sb.WriteString("- `POST /file-upload` - Upload files to input elements\n")
	sb.WriteString("- `POST /drag` - Drag and drop elements\n")
	sb.WriteString("- `POST /close-page` - Close the current page/tab\n\n")

	// 元素定位方式
	sb.WriteString("## Element Identification\n\n")
	sb.WriteString("You can identify elements using:\n\n")
	sb.WriteString("1. **RefID (Recommended):** `@e1`, `@e2`, `@e3`\n")
	sb.WriteString("   - Most reliable method - stable across page changes\n")
	sb.WriteString("   - Get RefIDs from `/snapshot` endpoint\n")
	sb.WriteString("   - Valid for 5 minutes after snapshot\n")
	sb.WriteString("   - Example: `\"identifier\": \"@e1\"`\n")
	sb.WriteString("   - Works with multi-strategy fallback for robustness\n\n")
	sb.WriteString("2. **CSS Selector:** `#id`, `.class`, `button[type=\"submit\"]`\n")
	sb.WriteString("   - Standard CSS selectors\n")
	sb.WriteString("   - Example: `\"identifier\": \"#login-button\"`\n\n")
	sb.WriteString("3. **XPath:** `//button[@id='login']`, `//a[contains(text(), 'Submit')]`\n")
	sb.WriteString("   - XPath expressions for complex queries\n")
	sb.WriteString("   - Example: `\"identifier\": \"//button[@id='login']\"`\n\n")
	sb.WriteString("4. **Text Content:** `Login`, `Sign Up`, `Submit`\n")
	sb.WriteString("   - Searches buttons and links with matching text\n")
	sb.WriteString("   - Example: `\"identifier\": \"Login\"`\n\n")
	sb.WriteString("5. **ARIA Label:** Elements with `aria-label` attribute\n")
	sb.WriteString("   - Automatically searched\n\n")

	// Guidelines
	sb.WriteString("## Guidelines\n\n")
	sb.WriteString("**Before starting:**\n")
	sb.WriteString("- Call `GET /help` if you're unsure about available commands or their parameters\n")
	sb.WriteString("- Ensure browser is started (if not, it will auto-start on first operation)\n\n")
	sb.WriteString("**During automation:**\n")
	sb.WriteString("- **Always call `/snapshot` after navigation** to get page structure and RefIDs\n")
	sb.WriteString("- **Prefer RefIDs** (like `@e1`) over CSS selectors for reliability and stability\n")
	sb.WriteString("- **Re-snapshot after page changes** to get updated RefIDs\n")
	sb.WriteString("- **Use `/wait`** for dynamic content that loads asynchronously\n")
	sb.WriteString("- **Check element states** before interaction (visible, enabled)\n")
	sb.WriteString("- **Use `/batch`** for multiple sequential operations to improve efficiency\n\n")
	sb.WriteString("**Error handling:**\n")
	sb.WriteString("- If operation fails, check element identifier and try different format\n")
	sb.WriteString("- For timeout errors, increase timeout value\n")
	sb.WriteString("- If element not found, call `/snapshot` again to refresh page structure\n")
	sb.WriteString("- Explain errors clearly to user with suggested solutions\n\n")
	sb.WriteString("**Data extraction:**\n")
	sb.WriteString("- Use `fields` parameter to specify what to extract: `[\"text\", \"href\", \"src\"]`\n")
	sb.WriteString("- Set `multiple: true` to extract from multiple elements\n")
	sb.WriteString("- Format extracted data in a readable way for user\n\n")

	// 完整工作流示例
	sb.WriteString("## Complete Workflow Example\n\n")
	sb.WriteString("**Scenario:** User wants to login to a website\n\n")
	sb.WriteString("```\n")
	sb.WriteString("User: \"Please log in to example.com with username 'john' and password 'secret123'\"\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**Your Actions:**\n\n")

	sb.WriteString("**Step 1:** Navigate to login page\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("POST http://%s/api/v1/executor/navigate\n", host))
	sb.WriteString("{\"url\": \"https://example.com/login\"}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Step 2:** Get page structure\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("GET http://%s/api/v1/executor/snapshot\n", host))
	sb.WriteString("```\n")
	sb.WriteString("Response:\n")
	sb.WriteString("```\n")
	sb.WriteString("Clickable Elements:\n")
	sb.WriteString("  @e1 Login (role: button)\n")
	sb.WriteString("\n")
	sb.WriteString("Input Elements:\n")
	sb.WriteString("  @e2 Username (role: textbox)\n")
	sb.WriteString("  @e3 Password (role: textbox)\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Step 3:** Enter username\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("POST http://%s/api/v1/executor/type\n", host))
	sb.WriteString("{\"identifier\": \"@e2\", \"text\": \"john\"}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Step 4:** Enter password\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("POST http://%s/api/v1/executor/type\n", host))
	sb.WriteString("{\"identifier\": \"@e3\", \"text\": \"secret123\"}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Step 5:** Click login button\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("POST http://%s/api/v1/executor/click\n", host))
	sb.WriteString("{\"identifier\": \"@e1\"}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Step 6:** Wait for login success (optional)\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("POST http://%s/api/v1/executor/wait\n", host))
	sb.WriteString("{\"identifier\": \".welcome-message\", \"state\": \"visible\", \"timeout\": 10}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**Step 7:** Inform user\n")
	sb.WriteString("```\n")
	sb.WriteString("\"Successfully logged in to example.com!\"\n")
	sb.WriteString("```\n\n")

	// 批量操作示例
	sb.WriteString("## Batch Operation Example\n\n")
	sb.WriteString("**Scenario:** Fill out a form with multiple fields\n\n")
	sb.WriteString("Instead of making 5 separate API calls, use one batch operation:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("curl -X POST 'http://%s/api/v1/executor/batch' \\\n", host))
	sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	sb.WriteString("  -d '{\n")
	sb.WriteString("    \"operations\": [\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"type\": \"navigate\",\n")
	sb.WriteString("        \"params\": {\"url\": \"https://example.com/form\"},\n")
	sb.WriteString("        \"stop_on_error\": true\n")
	sb.WriteString("      },\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"type\": \"type\",\n")
	sb.WriteString("        \"params\": {\"identifier\": \"#name\", \"text\": \"John Doe\"},\n")
	sb.WriteString("        \"stop_on_error\": true\n")
	sb.WriteString("      },\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"type\": \"type\",\n")
	sb.WriteString("        \"params\": {\"identifier\": \"#email\", \"text\": \"john@example.com\"},\n")
	sb.WriteString("        \"stop_on_error\": true\n")
	sb.WriteString("      },\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"type\": \"select\",\n")
	sb.WriteString("        \"params\": {\"identifier\": \"#country\", \"value\": \"United States\"},\n")
	sb.WriteString("        \"stop_on_error\": true\n")
	sb.WriteString("      },\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"type\": \"click\",\n")
	sb.WriteString("        \"params\": {\"identifier\": \"#submit\"},\n")
	sb.WriteString("        \"stop_on_error\": true\n")
	sb.WriteString("      }\n")
	sb.WriteString("    ]\n")
	sb.WriteString("  }'\n")
	sb.WriteString("```\n\n")

	// 最佳实践
	sb.WriteString("## Best Practices\n\n")
	sb.WriteString("1. **Discovery first:** If unsure, call `/help` or `/help?command=<name>` to learn about commands\n")
	sb.WriteString("2. **Structure first:** Always call `/snapshot` after navigation to understand the page\n")
	sb.WriteString("3. **Use accessibility indices:** They're more reliable than CSS selectors (elements might have dynamic classes)\n")
	sb.WriteString("4. **Wait for dynamic content:** Use `/wait` before interacting with elements that load asynchronously\n")
	sb.WriteString("5. **Batch when possible:** Use `/batch` for multiple sequential operations\n")
	sb.WriteString("6. **Handle errors gracefully:** Provide clear explanations and suggestions when operations fail\n")
	sb.WriteString("7. **Verify results:** After operations, check if desired outcome was achieved\n\n")

	// 常见场景
	sb.WriteString("## Common Scenarios\n\n")

	sb.WriteString("### Form Filling\n")
	sb.WriteString("1. Navigate to form page\n")
	sb.WriteString("2. Get accessibility snapshot to find input elements and their RefIDs\n")
	sb.WriteString("3. Use `/type` for each field: `@e1`, `@e2`, etc.\n")
	sb.WriteString("4. Use `/select` for dropdowns\n")
	sb.WriteString("5. Click submit button using its RefID\n\n")

	sb.WriteString("### Data Scraping\n")
	sb.WriteString("1. Navigate to target page\n")
	sb.WriteString("2. Wait for content to load with `/wait`\n")
	sb.WriteString("3. Use `/extract` with CSS selector and `multiple: true`\n")
	sb.WriteString("4. Specify fields to extract: `[\"text\", \"href\", \"src\"]`\n\n")

	sb.WriteString("### Search Operations\n")
	sb.WriteString("1. Navigate to search page\n")
	sb.WriteString("2. Get accessibility snapshot to locate search input\n")
	sb.WriteString("3. Type search query into input\n")
	sb.WriteString("4. Press Enter or click search button\n")
	sb.WriteString("5. Wait for results\n")
	sb.WriteString("6. Extract results data\n\n")

	sb.WriteString("### Login Automation\n")
	sb.WriteString("1. Navigate to login page\n")
	sb.WriteString("2. Get accessibility snapshot to find RefIDs\n")
	sb.WriteString("3. Type username: `@e2`\n")
	sb.WriteString("4. Type password: `@e3`\n")
	sb.WriteString("5. Click login button: `@e1`\n")
	sb.WriteString("6. Wait for success indicator\n\n")

	// 重要提示
	sb.WriteString("## Important Notes\n\n")
	sb.WriteString("- Browser must be running (it will auto-start on first operation if needed)\n")
	sb.WriteString("- Operations are executed on the **currently active browser tab**\n")
	sb.WriteString("- Accessibility snapshot updates after each navigation and click operation\n")
	sb.WriteString("- All timeouts are in seconds\n")
	sb.WriteString("- Use `wait_visible: true` (default) for reliable element interaction\n")
	sb.WriteString(fmt.Sprintf("- Replace `%s` with actual API host address\n", host))
	sb.WriteString("- Authentication required: use `X-BrowserWing-Key` header or JWT token\n\n")

	// 故障排除
	sb.WriteString("## Troubleshooting\n\n")
	sb.WriteString("**Element not found:**\n")
	sb.WriteString("- Call `/snapshot` to see available elements\n")
	sb.WriteString("- Try different identifier format (accessibility index, CSS selector, text)\n")
	sb.WriteString("- Check if page has finished loading\n\n")
	sb.WriteString("**Timeout errors:**\n")
	sb.WriteString("- Increase timeout value in request\n")
	sb.WriteString("- Check if element actually appears on page\n")
	sb.WriteString("- Use `/wait` with appropriate state before interaction\n\n")
	sb.WriteString("**Extraction returns empty:**\n")
	sb.WriteString("- Verify CSS selector matches target elements\n")
	sb.WriteString("- Check if content has loaded (use `/wait` first)\n")
	sb.WriteString("- Try different extraction fields or type\n\n")

	// 快速参考
	sb.WriteString("## Quick Reference\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Discover commands\n")
	sb.WriteString(fmt.Sprintf("GET %s/api/v1/executor/help\n\n", host))
	sb.WriteString("# Navigate\n")
	sb.WriteString(fmt.Sprintf("POST %s/api/v1/executor/navigate {\"url\": \"...\"}\n\n", host))
	sb.WriteString("# Get page structure\n")
	sb.WriteString(fmt.Sprintf("GET %s/api/v1/executor/snapshot\n\n", host))
	sb.WriteString("# Click element\n")
	sb.WriteString(fmt.Sprintf("POST %s/api/v1/executor/click {\"identifier\": \"@e1\"}\n\n", host))
	sb.WriteString("# Type text\n")
	sb.WriteString(fmt.Sprintf("POST %s/api/v1/executor/type {\"identifier\": \"@e3\", \"text\": \"...\"}\n\n", host))
	sb.WriteString("# Extract data\n")
	sb.WriteString(fmt.Sprintf("POST %s/api/v1/executor/extract {\"selector\": \"...\", \"fields\": [...], \"multiple\": true}\n", host))
	sb.WriteString("```\n\n")

	// 响应格式
	sb.WriteString("## Response Format\n\n")
	sb.WriteString("All operations return:\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"success\": true,\n")
	sb.WriteString("  \"message\": \"Operation description\",\n")
	sb.WriteString("  \"timestamp\": \"2026-01-15T10:30:00Z\",\n")
	sb.WriteString("  \"data\": {\n")
	sb.WriteString("    // Operation-specific data\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**Error response:**\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"error\": \"error.operationFailed\",\n")
	sb.WriteString("  \"detail\": \"Detailed error message\"\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	return sb.String()
}

// ==================== 浏览器实例管理 ====================

// CreateBrowserInstance 创建浏览器实例
func (h *Handler) CreateBrowserInstance(c *gin.Context) {
	var instance models.BrowserInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest", "detail": err.Error()})
		return
	}

	// 生成ID
	if instance.ID == "" {
		instance.ID = fmt.Sprintf("instance-%d", time.Now().UnixNano())
	}

	// 设置时间
	instance.CreatedAt = time.Now()
	instance.UpdatedAt = time.Now()

	// 保存到数据库
	if err := h.db.SaveBrowserInstance(&instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.saveFailed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "success.instanceCreated",
		"instance": instance,
	})
}

// ListBrowserInstances 列出所有浏览器实例
func (h *Handler) ListBrowserInstances(c *gin.Context) {
	instances, err := h.db.ListBrowserInstances()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.loadFailed", "detail": err.Error()})
		return
	}

	// 标记运行中的实例
	runningInstances := h.browserManager.ListRunningInstances()
	runningIDs := make(map[string]bool)
	for _, inst := range runningInstances {
		runningIDs[inst.ID] = true
	}

	for i := range instances {
		instances[i].IsActive = runningIDs[instances[i].ID]
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": instances,
	})
}

// GetBrowserInstance 获取浏览器实例详情
func (h *Handler) GetBrowserInstance(c *gin.Context) {
	id := c.Param("id")

	instance, err := h.db.GetBrowserInstance(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.notFound", "detail": err.Error()})
		return
	}

	// 检查是否正在运行
	runtime, _ := h.browserManager.GetInstanceRuntime(id)
	if runtime != nil {
		instance.IsActive = true
	}

	c.JSON(http.StatusOK, gin.H{
		"instance": instance,
	})
}

// UpdateBrowserInstance 更新浏览器实例
func (h *Handler) UpdateBrowserInstance(c *gin.Context) {
	id := c.Param("id")

	var instance models.BrowserInstance
	if err := c.ShouldBindJSON(&instance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidRequest", "detail": err.Error()})
		return
	}

	instance.ID = id
	if err := h.db.UpdateBrowserInstance(id, &instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updateFailed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "success.instanceUpdated",
		"instance": instance,
	})
}

// DeleteBrowserInstance 删除浏览器实例
func (h *Handler) DeleteBrowserInstance(c *gin.Context) {
	id := c.Param("id")

	// 检查是否正在运行
	if runtime, _ := h.browserManager.GetInstanceRuntime(id); runtime != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.instanceRunning", "detail": "Please stop the instance before deleting"})
		return
	}

	if err := h.db.DeleteBrowserInstance(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteFailed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.instanceDeleted",
	})
}

// StartBrowserInstance 启动浏览器实例
func (h *Handler) StartBrowserInstance(c *gin.Context) {
	id := c.Param("id")

	ctx := context.Background()
	if err := h.browserManager.StartInstance(ctx, id); err != nil {
		logger.Error(ctx, "Failed to start browser instance: %s", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.startBrowserFailed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.browserStarted",
	})
}

// StopBrowserInstance 停止浏览器实例
func (h *Handler) StopBrowserInstance(c *gin.Context) {
	id := c.Param("id")

	ctx := context.Background()
	if err := h.browserManager.StopInstance(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.stopBrowserFailed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.browserStopped",
	})
}

// SwitchBrowserInstance 切换当前活动实例
func (h *Handler) SwitchBrowserInstance(c *gin.Context) {
	id := c.Param("id")

	ctx := context.Background()
	if err := h.browserManager.SwitchInstance(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.switchFailed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.instanceSwitched",
	})
}

// GetCurrentBrowserInstance 获取当前活动实例
func (h *Handler) GetCurrentBrowserInstance(c *gin.Context) {
	instance := h.browserManager.GetCurrentInstance()
	if instance == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.noActiveInstance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance": instance,
	})
}

// SetScheduler 设置调度器
func (h *Handler) SetScheduler(scheduler interface{}) {
	h.scheduler = scheduler
}

// ================== Scheduled Tasks API ==================

// CreateScheduledTask 创建定时任务
func (h *Handler) CreateScheduledTask(c *gin.Context) {
	var task models.ScheduledTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams", "details": err.Error()})
		return
	}

	// 设置ID和时间戳
	task.ID = uuid.New().String()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.ExecutionCount = 0
	task.SuccessCount = 0
	task.FailedCount = 0

	// 验证必填字段
	if task.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.taskNameRequired"})
		return
	}
	if task.ScheduleType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.scheduleTypeRequired"})
		return
	}
	if task.ScheduleConfig == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.scheduleConfigRequired"})
		return
	}
	if task.ExecutionType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.executionTypeRequired"})
		return
	}

	// 根据执行类型验证配置
	if task.ExecutionType == models.ExecutionTypeScript && task.ScriptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.scriptIdRequired"})
		return
	}
	if task.ExecutionType == models.ExecutionTypeAgent && task.AgentPrompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.agentPromptRequired"})
		return
	}

	// 如果有脚本ID，加载脚本名称
	if task.ScriptID != "" {
		script, err := h.db.GetScript(task.ScriptID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error.scriptNotFound"})
			return
		}
		task.ScriptName = script.Name
	}

	// 如果有LLM ID，加载LLM名称
	if task.AgentLLMID != "" {
		llmConfig, err := h.db.GetLLMConfig(task.AgentLLMID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error.llmConfigNotFound"})
			return
		}
		task.AgentLLMName = llmConfig.Name
	}

	// 保存到数据库
	if err := h.db.CreateScheduledTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.createTaskFailed", "details": err.Error()})
		return
	}

	// 如果任务启用，添加到调度器
	if task.Enabled && h.scheduler != nil {
		type Scheduler interface {
			AddTask(*models.ScheduledTask) error
		}
		if scheduler, ok := h.scheduler.(Scheduler); ok {
			if err := scheduler.AddTask(&task); err != nil {
				logger.Warn(context.Background(), "Failed to add task to scheduler: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.taskCreated",
		"task":    task,
	})
}

// ListScheduledTasks 列出定时任务
func (h *Handler) ListScheduledTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	searchQuery := c.Query("search")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	tasks, total, err := h.db.ListScheduledTasksWithPagination(page, pageSize, searchQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getTaskListFailed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// GetScheduledTask 获取单个定时任务
func (h *Handler) GetScheduledTask(c *gin.Context) {
	id := c.Param("id")
	task, err := h.db.GetScheduledTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.taskNotFound"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"task": task})
}

// UpdateScheduledTask 更新定时任务
func (h *Handler) UpdateScheduledTask(c *gin.Context) {
	id := c.Param("id")
	
	// 检查任务是否存在
	existingTask, err := h.db.GetScheduledTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.taskNotFound"})
		return
	}

	var task models.ScheduledTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams", "details": err.Error()})
		return
	}

	// 保持ID和创建时间
	task.ID = id
	task.CreatedAt = existingTask.CreatedAt
	task.UpdatedAt = time.Now()
	
	// 保持统计数据
	task.ExecutionCount = existingTask.ExecutionCount
	task.SuccessCount = existingTask.SuccessCount
	task.FailedCount = existingTask.FailedCount
	task.LastExecutionTime = existingTask.LastExecutionTime
	task.LastExecutionStatus = existingTask.LastExecutionStatus

	// 验证必填字段
	if task.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.taskNameRequired"})
		return
	}
	if task.ScheduleType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.scheduleTypeRequired"})
		return
	}
	if task.ScheduleConfig == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.scheduleConfigRequired"})
		return
	}
	if task.ExecutionType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.executionTypeRequired"})
		return
	}

	// 根据执行类型验证配置
	if task.ExecutionType == models.ExecutionTypeScript && task.ScriptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.scriptIdRequired"})
		return
	}
	if task.ExecutionType == models.ExecutionTypeAgent && task.AgentPrompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.agentPromptRequired"})
		return
	}

	// 如果有脚本ID，加载脚本名称
	if task.ScriptID != "" {
		script, err := h.db.GetScript(task.ScriptID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error.scriptNotFound"})
			return
		}
		task.ScriptName = script.Name
	}

	// 如果有LLM ID，加载LLM名称
	if task.AgentLLMID != "" {
		llmConfig, err := h.db.GetLLMConfig(task.AgentLLMID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error.llmConfigNotFound"})
			return
		}
		task.AgentLLMName = llmConfig.Name
	}

	// 更新数据库
	if err := h.db.UpdateScheduledTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updateTaskFailed", "details": err.Error()})
		return
	}

	// 重新加载调度器中的任务
	if h.scheduler != nil {
		type Scheduler interface {
			ReloadTask(string) error
		}
		if scheduler, ok := h.scheduler.(Scheduler); ok {
			if err := scheduler.ReloadTask(id); err != nil {
				logger.Warn(context.Background(), "Failed to reload task in scheduler: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.taskUpdated",
		"task":    task,
	})
}

// DeleteScheduledTask 删除定时任务
func (h *Handler) DeleteScheduledTask(c *gin.Context) {
	id := c.Param("id")

	// 从调度器移除任务
	if h.scheduler != nil {
		type Scheduler interface {
			RemoveTask(string)
		}
		if scheduler, ok := h.scheduler.(Scheduler); ok {
			scheduler.RemoveTask(id)
		}
	}

	// 从数据库删除
	if err := h.db.DeleteScheduledTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteTaskFailed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.taskDeleted"})
}

// ToggleScheduledTask 启用/禁用定时任务
func (h *Handler) ToggleScheduledTask(c *gin.Context) {
	id := c.Param("id")
	
	task, err := h.db.GetScheduledTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.taskNotFound"})
		return
	}

	// 切换启用状态
	task.Enabled = !task.Enabled
	task.UpdatedAt = time.Now()

	if err := h.db.UpdateScheduledTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.updateTaskFailed", "details": err.Error()})
		return
	}

	// 更新调度器
	if h.scheduler != nil {
		type Scheduler interface {
			ReloadTask(string) error
		}
		if scheduler, ok := h.scheduler.(Scheduler); ok {
			if err := scheduler.ReloadTask(id); err != nil {
				logger.Warn(context.Background(), "Failed to reload task in scheduler: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success.taskToggled",
		"task":    task,
	})
}

// ================== Task Executions API ==================

// ListTaskExecutions 列出任务执行记录
func (h *Handler) ListTaskExecutions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	taskID := c.Query("task_id")
	searchQuery := c.Query("search")
	successFilter := c.DefaultQuery("success", "all")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	executions, total, err := h.db.ListTaskExecutionsWithPagination(page, pageSize, taskID, searchQuery, successFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.getExecutionsFailed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// GetTaskExecution 获取单个执行记录
func (h *Handler) GetTaskExecution(c *gin.Context) {
	id := c.Param("id")
	execution, err := h.db.GetTaskExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "error.executionNotFound"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"execution": execution})
}

// DeleteTaskExecution 删除执行记录
func (h *Handler) DeleteTaskExecution(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteTaskExecution(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.deleteExecutionFailed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.executionDeleted"})
}

// BatchDeleteTaskExecutions 批量删除执行记录
func (h *Handler) BatchDeleteTaskExecutions(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.invalidParams"})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "error.noExecutionsSelected"})
		return
	}

	if err := h.db.BatchDeleteTaskExecutions(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error.batchDeleteFailed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success.executionsDeleted", "count": len(req.IDs)})
}
