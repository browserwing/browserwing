package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/browserwing/browserwing/agent"
	"github.com/browserwing/browserwing/api"
	"github.com/browserwing/browserwing/config"
	"github.com/browserwing/browserwing/llm"
	"github.com/browserwing/browserwing/mcp"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/scheduler"
	"github.com/browserwing/browserwing/services/browser"
	"github.com/browserwing/browserwing/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// 构建信息变量，通过Makefile的LDFLAGS注入
var (
	Version   = "v0.1.0"
	BuildTime = ""
	GoVersion = ""
)

func main() {
	// 命令行参数
	port := flag.String("port", "", "Server port (default: 8080)")
	host := flag.String("host", "", "Server host (default: 0.0.0.0)")
	configPath := flag.String("config", "config.toml", "Path to config file (default: config.toml)")
	version := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// 显示版本信息
	if *version {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Go Version: %s\n", GoVersion)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Failed to load config file, using default config: %v", err)
	}

	logger.InitLogger(cfg.Log)

	// 完全禁用 agent-sdk-go 内部 zerolog 的日志输出
	// 避免在终端输出调试信息
	zerolog.SetGlobalLevel(zerolog.Disabled)

	// 优先级: 命令行参数 > 环境变量 > 配置文件
	if *port != "" {
		cfg.Server.Port = *port
	} else if envPort := os.Getenv("PORT"); envPort != "" {
		cfg.Server.Port = envPort
	}

	if *host != "" {
		cfg.Server.Host = *host
	} else if envHost := os.Getenv("HOST"); envHost != "" {
		cfg.Server.Host = envHost
	}

	// 确保数据库目录存在
	dbDir := filepath.Dir(cfg.Database.Path)
	err = os.MkdirAll(dbDir, 0o755)
	if err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// 初始化数据库
	db, err := storage.NewBoltDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("✓ Database initialization successful")

	// 检查并更新系统提示词（自动升级未修改的prompt）
	if err := db.CheckAndUpdateSystemPrompts(); err != nil {
		log.Printf("Warning: Failed to update system prompts: %v", err)
	} else {
		log.Println("✓ System prompts checked and updated")
	}

	// 初始化默认浏览器实例
	err = initDefaultBrowserInstance(db, cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize default browser instance: %v", err)
	} else {
		log.Println("✓ Default browser instance initialized successfully")
	}

	// 初始化默认用户（如果启用了认证）
	if cfg.Auth.Enabled {
		err = initDefaultUser(db, cfg)
		if err != nil {
			log.Printf("Warning: Failed to initialize default user: %v", err)
		} else {
			log.Println("✓ Default user initialized successfully")
		}
	}

	// 初始化 LLM 管理器
	llmManager := llm.NewManager(db)
	// 从配置文件加载 LLM 配置
	err = llmManager.LoadFromConfig(cfg)
	if err != nil {
		log.Printf("Warning: Failed to load LLM config from file: %v", err)
	} else {
		log.Printf("✓ LLM manager initialized successfully, loaded %d configs", len(llmManager.List()))
	}

	// 初始化浏览器管理器
	browserManager := browser.NewManager(cfg, db, llmManager)
	log.Println("✓ Browser manager initialized successfully")

	// 初始化 MCP 服务器 (使用 mcp-go 库)
	mcpServer := mcp.NewMCPServer(db, browserManager)
	err = mcpServer.Start()
	if err != nil {
		log.Printf("Warning: Failed to start MCP server: %v", err)
	} else {
		log.Println("✓ MCP server initialized successfully")
	}

	if cfg.Server.MCPPort != "" {
		host := ""
		if cfg.Server.MCPHost != "" {
			host = cfg.Server.MCPHost
		}
		err = mcpServer.StartStreamableHTTPServer(host + ":" + cfg.Server.MCPPort)
		if err != nil {
			log.Printf("Warning: Failed to start streamable HTTP server: %v", err)
		} else {
			log.Println("✓ Streamable HTTP server initialized successfully")
		}
	}

	// 初始化 Agent 管理器
	agentManager, err := agent.NewAgentManager(db, mcpServer)
	if err != nil {
		log.Printf("Warning: Failed to initialize Agent manager: %v", err)
	} else {
		log.Println("✓ Agent manager initialized successfully")
	}

	// 将 Agent 管理器注入到浏览器管理器
	browserManager.SetAgentManager(agentManager)

	// 初始化 AI 探索器
	explorer := browser.NewExplorer(browserManager, db)
	explorer.SetAgentManager(agentManager)
	explorer.SetExecutorRecorder(mcpServer.GetExecutor())
	log.Println("✓ AI Explorer initialized successfully")

	// 创建HTTP处理器
	handler := api.NewHandler(db, browserManager, cfg, llmManager)

	// 注入版本信息
	handler.SetVersionInfo(Version, BuildTime, GoVersion)

	// 将 MCP 服务器实例注入到 Handler
	handler.SetMCPServer(mcpServer)

	// 将 Agent 管理器注入到 Handler (用于 LLM 配置更新后的热加载)
	handler.SetAgentManager(agentManager)

	// 将 AI 探索器注入到 Handler
	handler.SetExplorer(explorer)

	// 初始化定时任务执行器（使用真实的浏览器管理器和 Agent 管理器）
	scriptPlayer := scheduler.NewRealScriptPlayer(db, browserManager)
	agentExecutor := scheduler.NewRealAgentExecutor(agentManager)
	taskExecutor := scheduler.NewDefaultTaskExecutor(db, scriptPlayer, agentExecutor)

	// 初始化定时任务调度器
	taskScheduler := scheduler.NewScheduler(db, taskExecutor)
	err = taskScheduler.Start()
	if err != nil {
		log.Printf("Warning: Failed to start scheduler: %v", err)
	} else {
		log.Println("✓ Task scheduler initialized successfully")
	}

	// 将调度器注入到 Handler
	handler.SetScheduler(taskScheduler)

	// 创建 Agent HTTP 处理器
	agentHandler := agent.NewHandler(agentManager)

	// 获取前端文件系统
	frontendFS, err := GetFrontendFS()
	embedMode := IsEmbedMode()
	if err != nil && embedMode {
		log.Printf("Warning: Failed to load frontend filesystem: %v", err)
	}

	router := api.SetupRouter(handler, agentHandler, frontendFS, embedMode, cfg.Debug)

	// 设置优雅退出
	setupGracefulShutdown(browserManager, db, mcpServer, agentManager, taskScheduler)

	// 启动服务器
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("🚀 BrowserWing server started at http://%s", addr)

	go openBrowser("http://127.0.0.1:" + cfg.Server.Port)

	if embedMode {
		log.Printf("📦 Running mode: Embedded (Frontend packed)")
		log.Printf("🌐 Access: http://%s", addr)
	} else {
		log.Printf("📦 Running mode: Development (Frontend needs to be started separately)")
		log.Printf("📝 API Documentation: http://%s/health", addr)
	}

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// setupGracefulShutdown 设置优雅退出，自动关闭浏览器
func setupGracefulShutdown(browserManager *browser.Manager, db *storage.BoltDB, mcpServer mcp.IMCPServer, agentManager *agent.AgentManager, taskScheduler interface{}) {
	sigChan := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("\nReceived exit signal: %v", sig)
		log.Println("Exiting gracefully...")

		// 创建超时上下文，最多等待 10 秒
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 停止定时任务调度器
		if taskScheduler != nil {
			log.Println("Stopping task scheduler...")
			type Scheduler interface {
				Stop()
			}
			if s, ok := taskScheduler.(Scheduler); ok {
				s.Stop()
				log.Println("✓ Task scheduler stopped")
			}
		}

		// 停止 Agent 管理器
		if agentManager != nil {
			log.Println("Stopping Agent manager...")
			agentManager.Stop()
			log.Println("✓ Agent manager stopped")
		}

		// 停止 MCP 服务器
		if mcpServer != nil {
			log.Println("Stopping MCP server...")
			mcpServer.Stop()
			log.Println("✓ MCP server stopped")
		}

		// 检查并关闭浏览器
		if browserManager.IsRunning() {
			log.Println("Browser is running, closing...")
			if err := browserManager.Stop(); err != nil {
				log.Printf("Failed to close browser: %v", err)
			} else {
				log.Println("✓ Browser closed")
			}
		} else {
			log.Println("Browser is not running, no need to close")
		}

		// 关闭数据库
		if db != nil {
			log.Println("Closing database...")
			if err := db.Close(); err != nil {
				log.Printf("Failed to close database: %v", err)
			} else {
				log.Println("✓ Database closed")
			}
		}

		// 等待或超时
		select {
		case <-ctx.Done():
			log.Println("Cleanup timeout, force exit")
		case <-time.After(500 * time.Millisecond):
			log.Println("Cleanup completed")
		}

		log.Println("Program exited")
		os.Exit(0)
	}()

	log.Println("✓ Graceful shutdown mechanism started (Ctrl+C will automatically close the browser)")
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux / freebsd...
		cmd = exec.Command("xdg-open", url)
	}

	_ = cmd.Start() // 不阻塞，忽略错误（有些环境可能没有 GUI）
}

// initDefaultBrowserInstance 初始化默认浏览器实例
func initDefaultBrowserInstance(db *storage.BoltDB, cfg *config.Config) error {
	// 检查是否已存在默认实例
	defaultInstance, err := db.GetDefaultBrowserInstance()
	if err == nil && defaultInstance != nil {
		log.Printf("Default browser instance already exists: %s (ID: %s)", defaultInstance.Name, defaultInstance.ID)

		// 同步配置文件中的浏览器配置到默认实例
		if cfg.Browser != nil {
			needUpdate := false

			// 检查并更新 ControlURL
			if cfg.Browser.ControlURL != "" && defaultInstance.ControlURL != cfg.Browser.ControlURL {
				log.Printf("Syncing control URL from config: %s -> %s", defaultInstance.ControlURL, cfg.Browser.ControlURL)
				defaultInstance.ControlURL = cfg.Browser.ControlURL
				// 如果配置了远程 URL，切换为 remote 类型
				if defaultInstance.Type != "remote" {
					defaultInstance.Type = "remote"
					log.Printf("Switching instance type to remote due to control URL")
				}
				needUpdate = true
			} else if cfg.Browser.ControlURL == "" && defaultInstance.Type == "remote" {
				// 如果配置中移除了 ControlURL，但实例仍是 remote 类型，切换回 local
				log.Printf("Control URL removed from config, switching to local mode")
				defaultInstance.Type = "local"
				defaultInstance.ControlURL = ""
				needUpdate = true
			}

			// 检查并更新 BinPath（仅 local 模式）
			if defaultInstance.Type == "local" && cfg.Browser.BinPath != "" && defaultInstance.BinPath != cfg.Browser.BinPath {
				log.Printf("Syncing bin path from config: %s -> %s", defaultInstance.BinPath, cfg.Browser.BinPath)
				defaultInstance.BinPath = cfg.Browser.BinPath
				needUpdate = true
			}

			// 检查并更新 UserDataDir（仅 local 模式）
			if defaultInstance.Type == "local" && cfg.Browser.UserDataDir != "" && defaultInstance.UserDataDir != cfg.Browser.UserDataDir {
				log.Printf("Syncing user data dir from config: %s -> %s", defaultInstance.UserDataDir, cfg.Browser.UserDataDir)
				defaultInstance.UserDataDir = cfg.Browser.UserDataDir
				needUpdate = true
			}

			// 如果有配置变化，保存实例
			if needUpdate {
				log.Printf("Updating default browser instance with config changes")
				return db.SaveBrowserInstance(defaultInstance)
			}
		}

		return nil
	}

	// 查找默认 Chrome 路径
	var binPath string
	var userDataDir string

	// 创建默认实例
	useStealth := true
	headless := false

	// 根据环境自动设置 headless
	display := os.Getenv("DISPLAY")
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")
	if runtime.GOOS == "linux" && display == "" && waylandDisplay == "" {
		headless = true
		log.Println("Detected headless environment, enabling headless mode for default instance")
	}

	browserType := "local"
	controlURL := ""

	if cfg.Browser.ControlURL != "" {
		browserType = "remote"
		controlURL = cfg.Browser.ControlURL
	} else {

		// 获取默认浏览器路径（参考 config.go 的逻辑）
		commonPaths := []string{
			"/usr/bin/google-chrome",
			"/usr/bin/chromium-browser",
			"/usr/bin/chromium",
			"/usr/bin/google-chrome-stable",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
		}

		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				binPath = path
				log.Printf("Found browser at: %s", binPath)
				break
			}
		}

		// 如果配置中有指定路径，优先使用配置的路径
		if cfg.Browser != nil && cfg.Browser.BinPath != "" {
			binPath = cfg.Browser.BinPath
			log.Printf("Using browser path from config: %s", binPath)
		}

		// 设置默认用户数据目录
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			userDataDir = filepath.Join(homeDir, ".browserwing", "default-profile")
		}
	}

	instance := &models.BrowserInstance{
		ID:          "default",
		Name:        "默认浏览器",
		Description: "系统默认浏览器实例",
		Type:        browserType,
		ControlURL:  controlURL,
		BinPath:     binPath,
		UserDataDir: userDataDir,
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36",
		UseStealth:  &useStealth,
		Headless:    &headless,
		LaunchArgs: []string{
			"disable-blink-features=AutomationControlled",
			"excludeSwitches=enable-automation",
			"no-first-run",
			"no-default-browser-check",
			"window-size=1920,1080",
			"start-maximized",
		},
		IsDefault: true,
		IsActive:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 保存到数据库
	if err := db.SaveBrowserInstance(instance); err != nil {
		return fmt.Errorf("failed to save default browser instance: %w", err)
	}

	log.Printf("Created default browser instance: %s (BinPath: %s, UserDataDir: %s)",
		instance.Name, instance.BinPath, instance.UserDataDir)
	return nil
}

// initDefaultUser 初始化默认用户
func initDefaultUser(db *storage.BoltDB, cfg *config.Config) error {
	// 检查是否已存在用户
	users, err := db.ListUsers()
	if err != nil {
		log.Printf("Warning: Failed to list users: %v", err)
		return err
	}

	log.Printf("Current user count: %d", len(users))

	// 如果已有用户，显示现有用户信息（不显示密码）
	if len(users) > 0 {
		log.Printf("Existing users:")
		for _, u := range users {
			log.Printf("  - Username: %s, ID: %s", u.Username, u.ID)
		}
		log.Printf("Default user already exists, skipping creation")
		return nil
	}

	// 创建默认用户
	log.Printf("Creating default user: username=%s, password=%s", cfg.Auth.DefaultUsername, cfg.Auth.DefaultPassword)
	defaultUser := &models.User{
		ID:        uuid.New().String(),
		Username:  cfg.Auth.DefaultUsername,
		Password:  cfg.Auth.DefaultPassword,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = db.CreateUser(defaultUser)
	if err != nil {
		log.Printf("Error: Failed to create default user: %v", err)
		return err
	}

	log.Printf("✓ Created default user: username=%s, id=%s", defaultUser.Username, defaultUser.ID)
	return nil
}
