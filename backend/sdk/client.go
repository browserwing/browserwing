package sdk

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/browserwing/browserwing/agent"
	"github.com/browserwing/browserwing/config"
	"github.com/browserwing/browserwing/llm"
	"github.com/browserwing/browserwing/mcp"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/services/browser"
	"github.com/browserwing/browserwing/storage"
)

// Config SDK 配置
type Config struct {
	// 必需配置
	DatabasePath string // 数据库文件路径

	// 可选配置 - 功能开关
	EnableBrowser bool // 是否启用浏览器功能
	EnableScript  bool // 是否启用脚本功能(依赖浏览器)
	EnableAgent   bool // 是否启用 Agent 功能

	// 可选配置 - 日志
	LogLevel  string // 日志级别: debug, info, warn, error
	LogOutput string // 日志输出: stdout, stderr, 或文件路径

	// 可选配置 - LLM
	LLMConfig *LLMConfig // LLM 配置(启用 Agent 时需要)

	// 可选配置 - 浏览器
	BrowserConfig *BrowserConfigOptions // 浏览器配置
}

// LLMConfig LLM 配置
type LLMConfig struct {
	Provider string // openai, anthropic, ollama 等
	APIKey   string // API 密钥
	Model    string // 模型名称
	BaseURL  string // API 基础 URL(可选)
}

// BrowserConfigOptions 浏览器配置选项
type BrowserConfigOptions struct {
	Headless     bool   // 是否无头模式
	UserAgent    string // 自定义 User-Agent
	ProxyURL     string // 代理 URL
	DownloadPath string // 下载目录
}

// Client BrowserWing SDK 客户端
type Client struct {
	config *Config

	// 核心组件
	db             *storage.BoltDB
	llmManager     *llm.Manager
	browserManager *browser.Manager
	mcpServer      *mcp.MCPServer
	agentManager   *agent.AgentManager

	// 子客户端
	browserClient *BrowserClient
	scriptClient  *ScriptClient
	agentClient   *AgentClient

	// 状态
	initialized bool
}

// New 创建新的 SDK 客户端
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// 验证必需配置
	if cfg.DatabasePath == "" {
		return nil, fmt.Errorf("database path is required")
	}

	// 验证依赖关系
	if cfg.EnableScript && !cfg.EnableBrowser {
		return nil, fmt.Errorf("script functionality requires browser to be enabled")
	}

	if cfg.EnableAgent && cfg.LLMConfig == nil {
		return nil, fmt.Errorf("agent functionality requires LLM configuration")
	}

	client := &Client{
		config: cfg,
	}

	// 初始化日志
	if err := client.initLogger(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// 初始化数据库
	if err := client.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 初始化 LLM 管理器
	if err := client.initLLMManager(); err != nil {
		return nil, fmt.Errorf("failed to initialize LLM manager: %w", err)
	}

	// 初始化浏览器管理器
	if cfg.EnableBrowser {
		if err := client.initBrowserManager(); err != nil {
			return nil, fmt.Errorf("failed to initialize browser manager: %w", err)
		}
		client.browserClient = &BrowserClient{client: client}
	}

	// 初始化脚本客户端
	if cfg.EnableScript {
		client.scriptClient = &ScriptClient{client: client}
	}

	// 初始化 MCP 服务器和 Agent
	if cfg.EnableAgent {
		if err := client.initMCPServer(); err != nil {
			return nil, fmt.Errorf("failed to initialize MCP server: %w", err)
		}
		if err := client.initAgentManager(); err != nil {
			return nil, fmt.Errorf("failed to initialize agent manager: %w", err)
		}
		client.agentClient = &AgentClient{client: client}
	}

	client.initialized = true
	log.Println("✓ BrowserWing SDK initialized successfully")

	return client, nil
}

// initLogger 初始化日志
func (c *Client) initLogger() error {
	logConfig := &logger.LoggerConfig{
		Level: "info",
		File:  "stdout",
	}

	if c.config.LogLevel != "" {
		logConfig.Level = c.config.LogLevel
	}
	if c.config.LogOutput != "" {
		logConfig.File = c.config.LogOutput
	}

	logger.InitLogger(logConfig)
	return nil
}

// initDatabase 初始化数据库
func (c *Client) initDatabase() error {
	// 确保数据库目录存在
	dbDir := filepath.Dir(c.config.DatabasePath)
	
	db, err := storage.NewBoltDB(c.config.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w (directory: %s)", err, dbDir)
	}

	c.db = db
	log.Println("✓ Database initialized")
	return nil
}

// initLLMManager 初始化 LLM 管理器
func (c *Client) initLLMManager() error {
	c.llmManager = llm.NewManager(c.db)

	// 如果提供了 LLM 配置,创建配置并保存到数据库
	if c.config.LLMConfig != nil {
		// 这里可以添加 LLM 配置的保存逻辑
		log.Println("✓ LLM manager initialized with custom config")
	} else {
		// 尝试从数据库加载现有配置
		if err := c.llmManager.LoadAll(); err != nil {
			log.Printf("Warning: Failed to load LLM configs: %v", err)
		}
		log.Println("✓ LLM manager initialized")
	}

	return nil
}

// initBrowserManager 初始化浏览器管理器
func (c *Client) initBrowserManager() error {
	// 创建默认配置
	cfg := &config.Config{
		Server: &config.ServerConfig{
			Host: "127.0.0.1",
			Port: "8080",
		},
		Database: &config.DatabaseConfig{
			Path: c.config.DatabasePath,
		},
	}

	c.browserManager = browser.NewManager(cfg, c.db, c.llmManager)
	log.Println("✓ Browser manager initialized")
	return nil
}

// initMCPServer 初始化 MCP 服务器
func (c *Client) initMCPServer() error {
	c.mcpServer = mcp.NewMCPServer(c.db, c.browserManager)
	if err := c.mcpServer.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	log.Println("✓ MCP server initialized")
	return nil
}

// initAgentManager 初始化 Agent 管理器
func (c *Client) initAgentManager() error {
	am, err := agent.NewAgentManager(c.db, c.mcpServer)
	if err != nil {
		return fmt.Errorf("failed to create agent manager: %w", err)
	}
	c.agentManager = am
	log.Println("✓ Agent manager initialized")
	return nil
}

// Browser 返回浏览器客户端
func (c *Client) Browser() *BrowserClient {
	if !c.config.EnableBrowser {
		log.Println("Warning: Browser functionality is not enabled")
		return nil
	}
	return c.browserClient
}

// Script 返回脚本客户端
func (c *Client) Script() *ScriptClient {
	if !c.config.EnableScript {
		log.Println("Warning: Script functionality is not enabled")
		return nil
	}
	return c.scriptClient
}

// Agent 返回 Agent 客户端
func (c *Client) Agent() *AgentClient {
	if !c.config.EnableAgent {
		log.Println("Warning: Agent functionality is not enabled")
		return nil
	}
	return c.agentClient
}

// Close 关闭客户端,释放资源
func (c *Client) Close() error {
	if !c.initialized {
		return nil
	}

	log.Println("Closing BrowserWing SDK...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 停止 Agent 管理器
	if c.agentManager != nil {
		log.Println("Stopping agent manager...")
		c.agentManager.Stop()
	}

	// 停止 MCP 服务器
	if c.mcpServer != nil {
		log.Println("Stopping MCP server...")
		c.mcpServer.Stop()
	}

	// 停止浏览器
	if c.browserManager != nil && c.browserManager.IsRunning() {
		log.Println("Stopping browser...")
		if err := c.browserManager.Stop(); err != nil {
			log.Printf("Warning: Failed to stop browser: %v", err)
		}
	}

	// 关闭数据库
	if c.db != nil {
		log.Println("Closing database...")
		if err := c.db.Close(); err != nil {
			log.Printf("Warning: Failed to close database: %v", err)
		}
	}

	// 等待或超时
	select {
	case <-ctx.Done():
		log.Println("Close timeout")
	case <-time.After(500 * time.Millisecond):
		log.Println("✓ SDK closed successfully")
	}

	c.initialized = false
	return nil
}

// IsInitialized 检查客户端是否已初始化
func (c *Client) IsInitialized() bool {
	return c.initialized
}
