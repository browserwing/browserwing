package api

import (
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(handler *Handler, agentHandler interface{}, frontendFS fs.FS, embedMode, isDebug bool) *gin.Engine {
	var r *gin.Engine
	if isDebug {
		gin.SetMode(gin.DebugMode)
		r = gin.Default()
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	}

	// CORS配置 - 允许所有来源（因为录制时可能访问任何网站）
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false, // AllowAllOrigins 为 true 时必须设置为 false
	}))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.Static("/files/recordings", "./recordings")

	// API路由组
	api := r.Group("/api/v1")
	{
		// 提示词相关
		prompts := api.Group("/prompts")
		{
			prompts.GET("", handler.ListPrompts)
			prompts.GET("/:id", handler.GetPrompt)
			prompts.POST("", handler.CreatePrompt)
			prompts.PUT("/:id", handler.UpdatePrompt)
			prompts.DELETE("/:id", handler.DeletePrompt)
		}

		// 浏览器相关
		browserAPI := api.Group("/browser")
		{
			browserAPI.POST("/start", handler.StartBrowser)
			browserAPI.POST("/stop", handler.StopBrowser)
			browserAPI.GET("/status", handler.BrowserStatus)
			browserAPI.POST("/open", handler.OpenBrowserPage)
			browserAPI.POST("/cookies/save", handler.SaveBrowserCookies)
			browserAPI.POST("/cookies/import", handler.ImportBrowserCookies)

			// 录制相关
			browserAPI.POST("/record/start", handler.StartRecording)
			browserAPI.POST("/record/stop", handler.StopRecording)
			browserAPI.GET("/record/status", handler.GetRecordingStatus)
			browserAPI.POST("/record/clear-state", handler.ClearInPageRecordingState)
		} // Cookie 管理
		api.GET("/cookies/:id", handler.GetCookies)

		// 浏览器配置管理
		browserConfigs := api.Group("/browser-configs")
		{
			browserConfigs.GET("", handler.ListBrowserConfigs)
			browserConfigs.GET("/:id", handler.GetBrowserConfig)
			browserConfigs.POST("", handler.CreateBrowserConfig)
			browserConfigs.PUT("/:id", handler.UpdateBrowserConfig)
			browserConfigs.DELETE("/:id", handler.DeleteBrowserConfig)
		}

		// 脚本相关
		scripts := api.Group("/scripts")
		{
			scripts.GET("", handler.ListScripts)
			scripts.GET("/:id", handler.GetScript)
			scripts.POST("", handler.SaveScript)
			scripts.PUT("/:id", handler.UpdateScript)
			scripts.DELETE("/:id", handler.DeleteScript)
			scripts.POST("/:id/play", handler.PlayScript)
			scripts.GET("/play/result", handler.GetPlayResult) // 获取回放抓取的数据

			// MCP 命令相关
			scripts.POST("/:id/mcp/generate", handler.GenerateMCPConfig) // AI 生成 MCP 配置
			scripts.POST("/:id/mcp", handler.ToggleScriptMCPCommand)     // 设置/取消 MCP 命令

			// 批量操作
			scripts.POST("/batch/group", handler.BatchSetGroup)       // 批量设置分组
			scripts.POST("/batch/tags", handler.BatchAddTags)         // 批量添加标签
			scripts.POST("/batch/delete", handler.BatchDeleteScripts) // 批量删除
		}

		// 脚本执行记录相关
		executions := api.Group("/script-executions")
		{
			executions.GET("", handler.ListScriptExecutions)                      // 列出执行记录（支持分页和搜索）
			executions.GET("/:id", handler.GetScriptExecution)                    // 获取单个执行记录
			executions.DELETE("/:id", handler.DeleteScriptExecution)              // 删除执行记录
			executions.POST("/batch/delete", handler.BatchDeleteScriptExecutions) // 批量删除
		}

		// MCP 服务相关
		mcp := api.Group("/mcp")
		{
			mcp.GET("/status", handler.GetMCPStatus)             // 获取 MCP 服务状态
			mcp.GET("/commands", handler.ListMCPCommands)        // 列出所有 MCP 命令
			mcp.GET("/commands_all", handler.ListMCPCommandsAll) // 列出所有 MCP 命令
			mcp.POST("/message", handler.HandleMCPMessage)       // HTTP 模式：处理 MCP 请求
			mcp.GET("/sse", handler.HandleMCPSSE)                // SSE 模式：长连接
		}

		// LLM 配置管理
		llmConfigs := api.Group("/llm-configs")
		{
			llmConfigs.GET("", handler.ListLLMConfigs)
			llmConfigs.GET("/:id", handler.GetLLMConfig)
			llmConfigs.POST("", handler.CreateLLMConfig)
			llmConfigs.PUT("/:id", handler.UpdateLLMConfig)
			llmConfigs.DELETE("/:id", handler.DeleteLLMConfig)
			llmConfigs.POST("/test", handler.TestLLMConfig)
		}

		// 录制配置管理
		api.GET("/recording-config", handler.GetRecordingConfig)
		api.PUT("/recording-config", handler.UpdateRecordingConfig)

		// 工具配置管理
		toolConfigs := api.Group("/tool-configs")
		{
			toolConfigs.GET("", handler.ListToolConfigs)       // 列出所有工具配置
			toolConfigs.GET("/:id", handler.GetToolConfig)     // 获取单个工具配置
			toolConfigs.PUT("/:id", handler.UpdateToolConfig)  // 更新工具配置
			toolConfigs.POST("/sync", handler.SyncToolConfigs) // 同步工具配置
		}

		// Agent 聊天相关
		if agentHandler != nil {
			type AgentHandlerInterface interface {
				CreateSession(c *gin.Context)
				GetSession(c *gin.Context)
				ListSessions(c *gin.Context)
				DeleteSession(c *gin.Context)
				SendMessage(c *gin.Context)
				SetLLMConfig(c *gin.Context)
				ReloadLLM(c *gin.Context)
				GetMCPStatus(c *gin.Context)
			}

			if ah, ok := agentHandler.(AgentHandlerInterface); ok {
				agentAPI := api.Group("/agent")
				{
					agentAPI.POST("/sessions", ah.CreateSession)            // 创建会话
					agentAPI.GET("/sessions", ah.ListSessions)              // 列出会话
					agentAPI.GET("/sessions/:id", ah.GetSession)            // 获取会话
					agentAPI.DELETE("/sessions/:id", ah.DeleteSession)      // 删除会话
					agentAPI.POST("/sessions/:id/messages", ah.SendMessage) // 发送消息 (SSE流式)
					agentAPI.POST("/llm/set", ah.SetLLMConfig)              // 设置 LLM 配置
					agentAPI.POST("/llm/reload", ah.ReloadLLM)              // 重新加载 LLM 配置
					agentAPI.GET("/mcp/status", ah.GetMCPStatus)            // 获取 MCP 状态
				}
			}
		}
	}

	// 嵌入模式下提供静态文件服务
	if embedMode && frontendFS != nil {
		// 静态文件处理
		r.NoRoute(func(c *gin.Context) {
			path := strings.TrimPrefix(c.Request.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			// 尝试读取文件
			file, err := frontendFS.Open(path)
			if err != nil {
				// 文件不存在，返回 index.html（用于 SPA 路由）
				file, err = frontendFS.Open("index.html")
				if err != nil {
					c.String(http.StatusNotFound, "404 page not found")
					return
				}
			}
			defer file.Close()

			// 读取文件信息
			stat, err := file.Stat()
			if err != nil {
				c.String(http.StatusInternalServerError, "Internal server error")
				return
			}

			// 如果是目录，尝试返回 index.html
			if stat.IsDir() {
				file.Close()
				indexPath := path + "/index.html"
				if path == "" || path == "." {
					indexPath = "index.html"
				}
				file, err = frontendFS.Open(indexPath)
				if err != nil {
					c.String(http.StatusNotFound, "404 page not found")
					return
				}
				defer file.Close()
				stat, _ = file.Stat()
			}

			// 使用 http.ServeContent 自动处理 MIME 类型和缓存
			http.ServeContent(c.Writer, c.Request, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
		})
	}

	return r
}

// io.ReadSeeker 接口定义
type readSeeker interface {
	io.Reader
	io.Seeker
}
