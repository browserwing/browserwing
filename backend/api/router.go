package api

import (
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/browserwing/browserwing/config"
	"github.com/browserwing/browserwing/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

	// TraceID 中间件 - 必须在其他中间件之前
	r.Use(TraceIDMiddleware())

	// CORS配置 - 允许所有来源（因为录制时可能访问任何网站）
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Trace-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Trace-ID"},
		AllowCredentials: false, // AllowAllOrigins 为 true 时必须设置为 false
	}))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.Static("/files/recordings", "./recordings")

	// 认证相关API（不需要认证）
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", handler.Login)
		auth.GET("/check", handler.CheckAuth) // 检查是否需要认证
	}

	// API路由组（需要JWT认证）
	api := r.Group("/api/v1")
	api.Use(JWTAuthenticationMiddleware(handler.config, handler.db))
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
		}

		// Cookie 管理
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
			scripts.GET("/play/result", handler.GetPlayResult) // 获取回放抓取的数据

			// MCP 命令相关
			scripts.POST("/:id/mcp/generate", handler.GenerateMCPConfig) // AI 生成 MCP 配置
			scripts.POST("/:id/mcp", handler.ToggleScriptMCPCommand)     // 设置/取消 MCP 命令

			// 批量操作
			scripts.POST("/batch/group", handler.BatchSetGroup)       // 批量设置分组
			scripts.POST("/batch/tags", handler.BatchAddTags)         // 批量添加标签
			scripts.POST("/batch/delete", handler.BatchDeleteScripts) // 批量删除

			// Claude Skills 导出
			scripts.POST("/export/skill", handler.ExportScriptsSkill) // 导出 SKILL.md
			scripts.GET("/summary", handler.GetScriptsSummary)        // 获取脚本摘要（用于 Claude Skills）
		}

		// PlayScript接口使用JWT或ApiKey认证（支持内部和外部调用）
		scriptsPlay := r.Group("/api/v1/scripts")
		scriptsPlay.Use(JWTOrApiKeyAuthenticationMiddleware(handler.config, handler.db))
		{
			scriptsPlay.POST("/:id/play", handler.PlayScript)
		}

		// 脚本执行记录相关
		executions := api.Group("/script-executions")
		{
			executions.GET("", handler.ListScriptExecutions)                      // 列出执行记录（支持分页和搜索）
			executions.GET("/:id", handler.GetScriptExecution)                    // 获取单个执行记录
			executions.DELETE("/:id", handler.DeleteScriptExecution)              // 删除执行记录
			executions.POST("/batch/delete", handler.BatchDeleteScriptExecutions) // 批量删除
		}

		// MCP 服务相关（管理接口）
		mcp := api.Group("/mcp")
		{
			mcp.GET("/status", handler.GetMCPStatus)             // 获取 MCP 服务状态
			mcp.GET("/commands", handler.ListMCPCommands)        // 列出所有 MCP 命令
			mcp.GET("/commands_all", handler.ListMCPCommandsAll) // 列出所有 MCP 命令
		}

		// MCP SSE端点（使用ApiKey认证，供外部MCP客户端调用）
		mcpSSE := r.Group("/api/v1/mcp")
		mcpSSE.Use(ApiKeyAuthenticationMiddleware(handler.config, handler.db))
		{
			mcpSSE.Any("/sse", gin.WrapH(handler.mcpServer.GetSSEServer().SSEHandler()))
			mcpSSE.Any("/sse_message", gin.WrapH(handler.mcpServer.GetSSEServer().MessageHandler()))
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

		// MCP服务管理
		mcpServices := api.Group("/mcp-services")
		{
			mcpServices.GET("", handler.ListMCPServices)                                 // 列出所有MCP服务
			mcpServices.GET("/:id", handler.GetMCPService)                               // 获取单个MCP服务
			mcpServices.POST("", handler.CreateMCPService)                               // 创建MCP服务
			mcpServices.PUT("/:id", handler.UpdateMCPService)                            // 更新MCP服务
			mcpServices.DELETE("/:id", handler.DeleteMCPService)                         // 删除MCP服务
			mcpServices.POST("/:id/toggle", handler.ToggleMCPService)                    // 启用/禁用MCP服务
			mcpServices.GET("/:id/tools", handler.GetMCPServiceTools)                    // 获取MCP服务的工具列表
			mcpServices.POST("/:id/discover", handler.DiscoverMCPServiceTools)           // 发现MCP服务的工具
			mcpServices.PUT("/:id/tools/:toolName", handler.UpdateMCPServiceToolEnabled) // 更新工具启用状态
		}

		// 用户管理
		users := api.Group("/users")
		{
			users.GET("", handler.ListUsers)                   // 列出所有用户
			users.GET("/:id", handler.GetUser)                 // 获取用户信息
			users.POST("", handler.CreateUser)                 // 创建用户
			users.PUT("/:id/password", handler.UpdatePassword) // 更新密码
			users.DELETE("/:id", handler.DeleteUser)           // 删除用户
		}

		// ApiKey管理
		apiKeys := api.Group("/api-keys")
		{
			apiKeys.GET("", handler.ListApiKeys)         // 列出所有API密钥
			apiKeys.GET("/:id", handler.GetApiKey)       // 获取API密钥
			apiKeys.POST("", handler.CreateApiKey)       // 创建API密钥
			apiKeys.DELETE("/:id", handler.DeleteApiKey) // 删除API密钥
		}

		// Executor HTTP API（使用 JWT 或 ApiKey 认证，支持外部调用）
		executorAPI := r.Group("/api/v1/executor")
		executorAPI.Use(JWTOrApiKeyAuthenticationMiddleware(handler.config, handler.db))
		{
			// 帮助和命令列表
			executorAPI.GET("/help", handler.ExecutorHelp)                // 获取所有可用命令和使用说明
			executorAPI.GET("/export/skill", handler.ExportExecutorSkill) // 导出 SKILL.md 文件

			// 页面导航和操作
			executorAPI.POST("/navigate", handler.ExecutorNavigate)               // 导航到 URL
			executorAPI.POST("/click", handler.ExecutorClick)                     // 点击元素
			executorAPI.POST("/type", handler.ExecutorType)                       // 输入文本
			executorAPI.POST("/select", handler.ExecutorSelect)                   // 选择下拉框
			executorAPI.POST("/hover", handler.ExecutorHover)                     // 鼠标悬停
			executorAPI.POST("/wait", handler.ExecutorWaitFor)                    // 等待元素
			executorAPI.POST("/scroll-to-bottom", handler.ExecutorScrollToBottom) // 滚动到底部
			executorAPI.POST("/go-back", handler.ExecutorGoBack)                  // 后退
			executorAPI.POST("/go-forward", handler.ExecutorGoForward)            // 前进
			executorAPI.POST("/reload", handler.ExecutorReload)                   // 刷新页面
			executorAPI.POST("/press-key", handler.ExecutorPressKey)              // 按键
			executorAPI.POST("/resize", handler.ExecutorResize)                   // 调整窗口大小

			// 数据提取和获取
			executorAPI.POST("/get-text", handler.ExecutorGetText)           // 获取元素文本
			executorAPI.POST("/get-value", handler.ExecutorGetValue)         // 获取元素值
			executorAPI.POST("/extract", handler.ExecutorExtract)            // 提取数据
			executorAPI.GET("/page-info", handler.ExecutorGetPageInfo)       // 获取页面信息
			executorAPI.GET("/page-content", handler.ExecutorGetPageContent) // 获取页面内容
			executorAPI.GET("/page-text", handler.ExecutorGetPageText)       // 获取页面文本

			// 语义树和元素查找
			executorAPI.GET("/semantic-tree", handler.ExecutorGetSemanticTree)           // 获取语义树
			executorAPI.GET("/clickable-elements", handler.ExecutorGetClickableElements) // 获取可点击元素
			executorAPI.GET("/input-elements", handler.ExecutorGetInputElements)         // 获取输入元素

			// 高级功能
			executorAPI.POST("/screenshot", handler.ExecutorScreenshot) // 截图
			executorAPI.POST("/evaluate", handler.ExecutorEvaluate)     // 执行 JavaScript
			executorAPI.POST("/batch", handler.ExecutorBatch)           // 批量执行操作
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

			// 如果是以 mcp 开头的，需要ApiKey认证
			if strings.HasPrefix(path, "api/v1/mcp/message") {
				// 验证ApiKey
				if handler.config.Auth.Enabled {
					apiKey := c.GetHeader("X-BrowserWing-Key")
					if apiKey == "" {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "error.unauthorized"})
						return
					}
					_, err := handler.db.GetApiKeyByKey(apiKey)
					if err != nil {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "error.invalidApiKey"})
						return
					}
				}
				handler.mcpServer.ServeSteamableHTTP(c.Writer, c.Request)
				return
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
	} else {
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/v1/mcp/message") {
				// 验证ApiKey
				if handler.config.Auth.Enabled {
					apiKey := c.GetHeader("X-BrowserWing-Key")
					if apiKey == "" {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "error.unauthorized"})
						return
					}
					_, err := handler.db.GetApiKeyByKey(apiKey)
					if err != nil {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "error.invalidApiKey"})
						return
					}
				}
				handler.mcpServer.ServeSteamableHTTP(c.Writer, c.Request)
				return
			}
			c.String(http.StatusNotFound, "404 page not found")
		})
	}

	return r
}

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateJWT 生成JWT Token
func GenerateJWT(userID, username string, config *config.Config) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour * 7)), // 7天过期
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Auth.AppKey))
}

// JWTAuthenticationMiddleware JWT认证中间件
func JWTAuthenticationMiddleware(config *config.Config, db *storage.BoltDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Auth.Enabled {
			c.Next()
			return
		}

		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "error.unauthorized"})
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "error.unauthorized"})
			c.Abort()
			return
		}

		// 解析JWT Token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.Auth.AppKey), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "error.invalidToken"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "error.invalidToken"})
			c.Abort()
			return
		}

		// 验证用户是否存在
		user, err := db.GetUser(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "error.userNotFound"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Next()
	}
}

// ApiKeyAuthenticationMiddleware ApiKey认证中间件
func ApiKeyAuthenticationMiddleware(config *config.Config, db *storage.BoltDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Auth.Enabled {
			c.Next()
			return
		}

		apiKey := c.GetHeader("X-BrowserWing-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "error.unauthorized"})
			c.Abort()
			return
		}

		// 验证API Key
		key, err := db.GetApiKeyByKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "error.invalidApiKey"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", key.UserID)
		c.Set("api_key_id", key.ID)
		c.Next()
	}
}

// JWTOrApiKeyAuthenticationMiddleware JWT或ApiKey认证中间件（两者任一即可）
func JWTOrApiKeyAuthenticationMiddleware(config *config.Config, db *storage.BoltDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Auth.Enabled {
			c.Next()
			return
		}

		// 先尝试API Key认证
		apiKey := c.GetHeader("X-BrowserWing-Key")
		if apiKey != "" {
			key, err := db.GetApiKeyByKey(apiKey)
			if err == nil {
				// API Key验证成功
				c.Set("user_id", key.UserID)
				c.Set("api_key_id", key.ID)
				c.Next()
				return
			}
		}

		// API Key不存在或验证失败，尝试JWT Token
		tokenString := c.GetHeader("Authorization")
		if tokenString != "" {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
			if tokenString != "" {
				// 解析JWT Token
				token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(config.Auth.AppKey), nil
				})

				if err == nil && token.Valid {
					claims, ok := token.Claims.(*JWTClaims)
					if ok {
						// 验证用户是否存在
						user, err := db.GetUser(claims.UserID)
						if err == nil {
							// JWT验证成功
							c.Set("user_id", user.ID)
							c.Set("username", user.Username)
							c.Next()
							return
						}
					}
				}
			}
		}

		// 两种认证方式都失败
		c.JSON(http.StatusUnauthorized, gin.H{"error": "error.unauthorized"})
		c.Abort()
	}
}
