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
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/services/browser"
	"github.com/browserwing/browserwing/storage"
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

	// 禁用 agent-sdk-go 内部 zerolog 的 Debug 和 Info 日志
	// 只允许 Warn 及以上级别的日志输出
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

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

	// 初始化 Agent 管理器
	agentManager, err := agent.NewAgentManager(db, mcpServer)
	if err != nil {
		log.Printf("Warning: Failed to initialize Agent manager: %v", err)
	} else {
		log.Println("✓ Agent manager initialized successfully")
	}

	// 创建HTTP处理器
	handler := api.NewHandler(db, browserManager, cfg, llmManager)

	// 将 MCP 服务器实例注入到 Handler
	handler.SetMCPServer(mcpServer)

	// 将 Agent 管理器注入到 Handler (用于 LLM 配置更新后的热加载)
	handler.SetAgentManager(agentManager)

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
	setupGracefulShutdown(browserManager, db, mcpServer, agentManager)

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
func setupGracefulShutdown(browserManager *browser.Manager, db *storage.BoltDB, mcpServer mcp.IMCPServer, agentManager *agent.AgentManager) {
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
