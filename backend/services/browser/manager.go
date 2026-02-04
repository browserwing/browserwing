package browser

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/browserwing/browserwing/config"
	"github.com/browserwing/browserwing/llm"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/storage"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

//go:embed scripts/float_button.js
var floatButtonScript string

//go:embed scripts/xhr_interceptor.js
var xhrInterceptorScriptForManager string

// resolveWebSocketURL 从 HTTP control URL 解析 WebSocket URL
// 如果输入已经是 ws:// 或 wss:// URL，则直接返回
// 如果是 http:// 或 https:// URL，则查询 /json/version 获取 webSocketDebuggerUrl
func resolveWebSocketURL(controlURL string) (string, error) {
	// 如果已经是 WebSocket URL，直接返回
	if strings.HasPrefix(controlURL, "ws://") || strings.HasPrefix(controlURL, "wss://") {
		return controlURL, nil
	}

	// HTTP URL，需要查询 /json/version
	versionURL := strings.TrimRight(controlURL, "/") + "/json/version"
	resp, err := http.Get(versionURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch browser version from %s: %w", versionURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d from %s: %s", resp.StatusCode, versionURL, string(body))
	}

	var result struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response from %s: %w", versionURL, err)
	}

	if result.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("webSocketDebuggerUrl not found in response from %s", versionURL)
	}

	return result.WebSocketDebuggerURL, nil
}

// BrowserInstanceRuntime 浏览器实例运行时信息
type BrowserInstanceRuntime struct {
	instance   *models.BrowserInstance // 实例配置
	browser    *rod.Browser            // 浏览器对象
	launcher   *launcher.Launcher      // 启动器（仅本地模式）
	activePage *rod.Page               // 当前活动页面
	startTime  time.Time               // 启动时间
}

// Manager 浏览器管理器
type Manager struct {
	config     *config.Config
	db         *storage.BoltDB
	llmManager *llm.Manager
	mu         sync.Mutex
	recorder   *Recorder

	// 多实例管理
	instances         map[string]*BrowserInstanceRuntime // 实例 ID -> 运行时信息
	currentInstanceID string                             // 当前活动实例 ID

	// 共享配置
	defaultBrowserConfig   *models.BrowserConfig   // 默认浏览器配置
	siteConfigs            []*models.BrowserConfig // 网站特定配置列表
	lastRecordedActions    []models.ScriptAction   // 最后一次录制的动作(用于页面内停止录制)
	lastRecordedStartURL   string                  // 最后一次录制的起始URL(用于页面内停止录制)
	lastDownloadedFiles    []models.DownloadedFile // 最后一次录制下载的文件(用于页面内停止录制)
	inPageRecordingStopped bool                    // 标记是否是页面内停止的录制
	currentLanguage        string                  // 当前前端语言设置
	downloadPath           string                  // 下载目录路径

	// 向后兼容（废弃）
	browser    *rod.Browser
	launcher   *launcher.Launcher
	isRunning  bool
	startTime  time.Time
	activePage *rod.Page
}

// NewManager 创建浏览器管理器
func NewManager(cfg *config.Config, db *storage.BoltDB, llmManager *llm.Manager) *Manager {
	recorder := NewRecorder()
	// 设置 API 服务器端口
	if cfg.Server != nil && cfg.Server.Port != "" {
		recorder.SetAPIServerPort(cfg.Server.Port)
	}

	// 设置 LLM 管理器
	if llmManager != nil {
		recorder.SetLLMManager(llmManager)
	}

	return &Manager{
		config:     cfg,
		db:         db,
		llmManager: llmManager,
		recorder:   recorder,
		instances:  make(map[string]*BrowserInstanceRuntime),
	}
}

// Start 启动浏览器
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("browser is already running")
	}

	logger.Info(ctx, "Starting browser...")

	// 加载默认配置
	defaultConfig, err := m.db.GetDefaultBrowserConfig()
	if err != nil {
		logger.Warn(ctx, "Default browser configuration not found, using system defaults")
		defaultConfig = m.getDefaultBrowserConfig()
	}
	m.defaultBrowserConfig = defaultConfig

	// 加载所有网站特定配置
	allConfigs, err := m.db.ListBrowserConfigs()
	if err != nil {
		logger.Warn(ctx, "Failed to load site configurations: %v", err)
		m.siteConfigs = []*models.BrowserConfig{}
	} else {
		// 过滤出有URL模式的配置
		m.siteConfigs = []*models.BrowserConfig{}
		for i := range allConfigs {
			if allConfigs[i].URLPattern != "" && !allConfigs[i].IsDefault {
				m.siteConfigs = append(m.siteConfigs, &allConfigs[i])
			}
		}
		logger.Info(ctx, "Loaded %d site-specific configurations", len(m.siteConfigs))
	}

	logger.Info(ctx, fmt.Sprintf("Using default configuration: %s", defaultConfig.Name))

	var url string
	var browser *rod.Browser

	// 检查是否配置了远程 Chrome URL
	if m.config.Browser != nil && m.config.Browser.ControlURL != "" {
		// 使用远程 Chrome
		controlURL := m.config.Browser.ControlURL
		logger.Info(ctx, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Info(ctx, "Using remote Chrome browser")
		logger.Info(ctx, fmt.Sprintf("Control URL: %s", controlURL))
		logger.Info(ctx, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// 解析 WebSocket URL
		wsURL, err := resolveWebSocketURL(controlURL)
		if err != nil {
			return fmt.Errorf("failed to resolve WebSocket URL from %s: %w", controlURL, err)
		}
		logger.Info(ctx, fmt.Sprintf("Resolved WebSocket URL: %s", wsURL))
		url = wsURL

		// 连接到远程浏览器
		browser = rod.New().ControlURL(url)
	} else {
		// 启动本地浏览器
		logger.Info(ctx, "Starting local Chrome browser...")

		// 创建启动器
		// 根据配置决定是否使用 headless 模式
		headless := false // 默认不使用 headless
		if defaultConfig.Headless != nil {
			headless = *defaultConfig.Headless
		}
		logger.Info(ctx, fmt.Sprintf("Headless mode: %v", headless))

		l := launcher.New().
			Headless(headless).
			Devtools(false).
			Leakless(false)

		if defaultConfig.Proxy != "" {
			l = l.Proxy(defaultConfig.Proxy)
			logger.Info(ctx, fmt.Sprintf("Using proxy: %s", defaultConfig.Proxy))
		}

		// 打印启动参数
		logger.Info(ctx, fmt.Sprintf("Number of launch arguments: %d", len(defaultConfig.LaunchArgs)))
		for i, arg := range defaultConfig.LaunchArgs {
			logger.Info(ctx, fmt.Sprintf("  [%d] %s", i+1, arg))
		}

		// 应用默认配置的启动参数
		for _, arg := range defaultConfig.LaunchArgs {
			// 移除前导的--如果存在
			arg = strings.TrimPrefix(arg, "--")

			// 检查是否是key=value格式
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				l = l.Set(flags.Flag(parts[0]), parts[1])
			} else {
				// 单个flag
				l = l.Set(flags.Flag(arg))
			}
		}

		// 设置浏览器路径
		if m.config.Browser != nil && m.config.Browser.BinPath != "" {
			l = l.Bin(m.config.Browser.BinPath)
			logger.Info(ctx, fmt.Sprintf("Using browser path: %s", m.config.Browser.BinPath))
		}

		// 设置用户数据目录 - 关键：这会保存登录状态
		if m.config.Browser != nil && m.config.Browser.UserDataDir != "" {
			userDataDir := m.config.Browser.UserDataDir

			// 确保目录存在
			if err := os.MkdirAll(userDataDir, 0o755); err != nil {
				logger.Warn(ctx, fmt.Sprintf("Failed to create user data directory: %v", err))
				logger.Warn(ctx, "Will not use user data directory")
			} else {
				// 检查目录是否可写
				testFile := userDataDir + "/.test"
				if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
					logger.Warn(ctx, fmt.Sprintf("User data directory is not writable: %v", err))
					logger.Warn(ctx, "Will not use user data directory, may cause startup failure")
				} else {
					os.Remove(testFile)

					// 清理可能存在的锁文件（在启动前）
					logger.Info(ctx, "Checking and cleaning up lock files before launch...")
					if err := m.cleanupSingletonLock(ctx, userDataDir); err != nil {
						logger.Warn(ctx, "Failed to cleanup singleton lock: %v", err)
					}

					l = l.UserDataDir(userDataDir)
					logger.Info(ctx, fmt.Sprintf("✓ Using user data directory: %s", userDataDir))
				}
			}
		} else {
			logger.Warn(ctx, "User data directory not configured, login state will not be saved")
		}

		logger.Info(ctx, "Starting browser process...")
		// 启动浏览器
		var err error
		url, err = l.Launch()
		if err != nil {
			errMsg := err.Error()
			logger.Error(ctx, "Failed to start browser, detailed error: %v", err)

			// 如果是 SingletonLock 错误，尝试清理并给出提示
			if m.config.Browser != nil && m.config.Browser.UserDataDir != "" && strings.Contains(errMsg, "SingletonLock") {
				logger.Error(ctx, "Browser launch failed due to SingletonLock, attempting cleanup...")
				m.cleanupSingletonLock(ctx, m.config.Browser.UserDataDir)
				return fmt.Errorf("failed to launch browser (SingletonLock issue): %w\nTip: The lock files have been cleaned up. Please try starting the browser again", err)
			}

			// 检查是否是因为 Chrome 已经在运行
			if strings.Contains(errMsg, "会话") || strings.Contains(errMsg, "session") || strings.Contains(errMsg, "already") {
				logger.Error(ctx, "")
				logger.Error(ctx, "❌ Error reason: Chrome browser is already running with the same user data directory")
				logger.Error(ctx, "")
				logger.Error(ctx, "Solution:")
				logger.Error(ctx, "  1. Close all Chrome browser windows")
				logger.Error(ctx, "  2. Open Task Manager and end all chrome.exe processes")
				logger.Error(ctx, "  3. Then click the 'Start Browser' button again")
				logger.Error(ctx, "")
				logger.Error(ctx, "Or modify user_data_dir in config.toml to another directory")
				logger.Error(ctx, "")
				return fmt.Errorf("Chrome is already running with the same user data directory, please close all Chrome windows and try again")
			}

			logger.Error(ctx, "Possible reasons:")
			logger.Error(ctx, "  1. Incorrect Chrome path")
			logger.Error(ctx, "  2. Insufficient permissions or invalid path for user data directory")
			logger.Error(ctx, "  3. Chrome is being used by another process")
			logger.Error(ctx, "  4. Blocked by system firewall or security software")
			return fmt.Errorf("failed to start browser: %w", err)
		}

		logger.Info(ctx, fmt.Sprintf("Browser control URL: %s", url))

		// 连接到浏览器
		browser = rod.New().ControlURL(url)

		// 保存 launcher 实例用于后续清理
		m.launcher = l
	}
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect browser: %w", err)
	}

	// 获取并显示浏览器版本信息
	version, err := browser.Version()
	if err != nil {
		logger.Warn(ctx, "Failed to get browser version: %v", err)
	} else {
		logger.Info(ctx, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logger.Info(ctx, "Browser version information:")
		logger.Info(ctx, "  Product: %s", version.Product)
		logger.Info(ctx, "  User-Agent: %s", version.UserAgent)
		logger.Info(ctx, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}

	// 尝试从数据库加载保存的 Cookie
	if m.db != nil {
		cookieStore, err := m.db.GetCookies("browser")
		if err == nil && cookieStore != nil && len(cookieStore.Cookies) > 0 {
			// 将 NetworkCookie 转换为 NetworkCookieParam
			cookieParams := make([]*proto.NetworkCookieParam, 0, len(cookieStore.Cookies))
			for _, cookie := range cookieStore.Cookies {
				cookieParams = append(cookieParams, &proto.NetworkCookieParam{
					Name:     cookie.Name,
					Value:    cookie.Value,
					Domain:   cookie.Domain,
					Path:     cookie.Path,
					Secure:   cookie.Secure,
					HTTPOnly: cookie.HTTPOnly,
					SameSite: cookie.SameSite,
					Expires:  cookie.Expires,
				})
			}

			// 设置 Cookie 到浏览器
			if err := browser.SetCookies(cookieParams); err != nil {
				logger.Warn(ctx, "Failed to set Cookie: %v", err)
			} else {
				logger.Info(ctx, "Loaded %d saved Cookies", len(cookieParams))
			}
		} else {
			logger.Info(ctx, "No saved Cookies found")
		}
	}

	downloadPath := "./downloads"
	// 获取绝对路径
	absDownloadPath, err := os.Getwd()
	if err == nil {
		downloadPath = absDownloadPath + "/downloads"
	}
	// 判断文件夹是否存在，不存在则创建
	if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
		err := os.MkdirAll(downloadPath, 0o755)
		if err != nil {
			logger.Warn(ctx, "Failed to create download directory: %v", err)
		} else {
			logger.Info(ctx, "Download directory created: %s", downloadPath)
		}
	}

	downloadBehavior := &proto.BrowserSetDownloadBehavior{
		Behavior:      proto.BrowserSetDownloadBehaviorBehaviorAllow,
		DownloadPath:  downloadPath, // ⚠ 必须是已存在目录
		EventsEnabled: true,
	}
	err = downloadBehavior.Call(browser)
	if err != nil {
		logger.Warn(ctx, "Failed to set download behavior: %v", err)
	} else {
		logger.Info(ctx, "Download behavior set: %s, path: %s", downloadBehavior.Behavior, downloadBehavior.DownloadPath)
	}

	// 保存下载路径到 Manager 和 Recorder
	m.downloadPath = downloadPath
	m.recorder.SetDownloadPath(downloadPath)

	// 授予剪贴板权限，避免粘贴时弹出权限请求
	grantPermissions := &proto.BrowserGrantPermissions{
		Permissions: []proto.BrowserPermissionType{
			proto.BrowserPermissionTypeClipboardReadWrite,
			proto.BrowserPermissionTypeClipboardSanitizedWrite,
		},
	}
	err = grantPermissions.Call(browser)
	if err != nil {
		logger.Warn(ctx, "Failed to grant clipboard permissions: %v", err)
	} else {
		logger.Info(ctx, "✓ Clipboard permissions granted (read/write)")
	}

	m.browser = browser
	m.isRunning = true
	m.startTime = time.Now()

	logger.Info(ctx, "Browser started successfully")
	return nil
}

// Stop 停止浏览器
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("browser is not running")
	}

	ctx := context.Background()

	// 检查是否是远程模式
	isRemoteMode := m.config.Browser != nil && m.config.Browser.ControlURL != ""

	if isRemoteMode {
		logger.Info(ctx, "Disconnecting from remote browser...")
	} else {
		logger.Info(ctx, "Closing browser...")
	}

	// 1. 先关闭所有页面，让浏览器有机会保存数据
	if m.browser != nil {
		if !isRemoteMode {
			// 仅在本地模式下关闭页面
			pages, err := m.browser.Pages()
			if err == nil {
				for _, page := range pages {
					_ = page.Close()
				}
				logger.Info(ctx, fmt.Sprintf("Closed %d pages", len(pages)))
			}

			// 2. 等待一下，让浏览器保存数据
			time.Sleep(1 * time.Second)
		}

		// 3. 优雅关闭浏览器连接
		if err := m.browser.Close(); err != nil {
			logger.Warn(ctx, fmt.Sprintf("Error when closing browser connection: %v", err))
		}
	}

	// 4. 仅在本地模式下关闭浏览器进程
	if !isRemoteMode {
		// 再等待一下，确保数据完全写入磁盘
		time.Sleep(1 * time.Second)

		// 5. ⚠️ 重要：不调用 launcher.Cleanup()，因为它会删除用户数据目录！
		// 浏览器进程会在连接关闭后自动退出
		// 如果需要强制杀死进程，可以调用 launcher.Kill() 而不是 Cleanup()
		if m.launcher != nil {
			// 只杀死进程，不清理目录
			m.launcher.Kill()
			logger.Info(ctx, "Browser process terminated")
		}

		// 6. 清理本地浏览器的锁文件
		if m.config.Browser != nil && m.config.Browser.UserDataDir != "" {
			// 等待浏览器完全退出
			time.Sleep(500 * time.Millisecond)

			if err := m.cleanupSingletonLock(ctx, m.config.Browser.UserDataDir); err != nil {
				logger.Warn(ctx, "Failed to cleanup singleton lock after stop: %v", err)
			} else {
				logger.Info(ctx, "✓ Cleaned up singleton lock files")
			}
		}
	}

	m.browser = nil
	m.launcher = nil
	m.isRunning = false

	if isRemoteMode {
		logger.Info(ctx, "Disconnected from remote browser successfully")
	} else {
		logger.Info(ctx, "Browser fully closed, user data saved")
	}
	return nil
}

// IsRunning 检查浏览器是否运行
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否有当前实例ID
	if m.currentInstanceID == "" {
		return m.isRunning // 向后兼容：如果没有实例ID，使用旧逻辑
	}

	// 检查当前实例是否真的在运行
	runtime, exists := m.instances[m.currentInstanceID]
	return exists && runtime != nil && runtime.browser != nil
}

// GetActivePage 获取当前活动页面
func (m *Manager) GetActivePage() *rod.Page {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activePage
}

// CloseActivePage 关闭当前活动页面
func (m *Manager) CloseActivePage(ctx context.Context, page *rod.Page) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning || m.browser == nil {
		return fmt.Errorf("browser is not running")
	}

	if page == nil {
		logger.Warn(ctx, "No active page to close")
		return nil
	}

	logger.Info(ctx, "Closing active page...")
	if err := page.Close(); err != nil {
		return fmt.Errorf("failed to close active page: %w", err)
	}

	logger.Info(ctx, "Active page closed")
	return nil
}

// Status 获取浏览器状态
func (m *Manager) Status() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	status := map[string]interface{}{
		"is_running": m.isRunning,
	}

	if m.isRunning {
		status["start_time"] = m.startTime.Format(time.RFC3339)
		status["uptime"] = time.Since(m.startTime).String()

		// 获取浏览器页面数量
		if m.browser != nil {
			pages, err := m.browser.Pages()
			if err == nil {
				status["pages_count"] = len(pages)
			}
		}
	}

	return status
}

func (m *Manager) setPageWindow(page *rod.Page) {
	ctx := context.Background()

	// 获取屏幕尺寸
	screenInfo, err := page.Eval(`() => ({
		width: window.screen.availWidth,
		height: window.screen.availHeight
	})`)

	var windowWidth, windowHeight int
	var viewportWidth, viewportHeight int

	if err == nil && screenInfo != nil {
		if info, ok := screenInfo.Value.Val().(map[string]interface{}); ok {
			screenWidth := int(info["width"].(float64))
			screenHeight := int(info["height"].(float64))

			logger.Info(ctx, "Detected screen size: %dx%d", screenWidth, screenHeight)

			// 窗口大小设置为屏幕大小的 90%
			windowWidth = int(float64(screenWidth) * 0.9)
			windowHeight = int(float64(screenHeight) * 0.9)

			// viewport 大小为窗口大小减去浏览器边框和工具栏 (约 120 像素宽度, 100 像素高度)
			viewportWidth = windowWidth - 120
			viewportHeight = windowHeight - 100

			logger.Info(ctx, "Calculated window size: %dx%d", windowWidth, windowHeight)
			logger.Info(ctx, "Calculated viewport size: %dx%d", viewportWidth, viewportHeight)
		} else {
			logger.Warn(ctx, "Failed to parse screen info, using default sizes")
			windowWidth = 1400
			windowHeight = 900
			viewportWidth = 1280
			viewportHeight = 800
		}
	} else {
		logger.Warn(ctx, "Failed to get screen size: %v, using default sizes", err)
		windowWidth = 1400
		windowHeight = 900
		viewportWidth = 1280
		viewportHeight = 800
	}

	// 设置浏览器窗口（外壳）
	page.MustSetWindow(0, 0, windowWidth, windowHeight)

	// 设置页面 viewport（CSS 布局尺寸）
	page.MustSetViewport(
		viewportWidth,  // width
		viewportHeight, // height
		1,              // deviceScaleFactor
		false,          // desktop
	)
}

// OpenPage 打开一个新页面
// instanceID: 指定实例ID，空字符串表示使用当前实例
func (m *Manager) OpenPage(url string, language string, instanceID string, norecord ...bool) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 捕获 panic 并转换为错误
	defer func() {
		if r := recover(); r != nil {
			ctx := context.Background()
			logger.Error(ctx, "Panic in OpenPage: %v", r)
			err = fmt.Errorf("failed to open page: browser connection may be closed (panic: %v)", r)
		}
	}()

	var noRecord bool
	if len(norecord) > 0 {
		noRecord = norecord[0]
	}

	// 获取指定实例的浏览器
	browser, _, instance, err := m.getInstanceBrowser(instanceID)
	if err != nil {
		return err
	}

	// 使用实际的实例ID（可能从空字符串转换为 default）
	if instance != nil {
		instanceID = instance.ID
	} else if instanceID == "" {
		// 向后兼容：如果没有 instance 对象，使用 currentInstanceID
		instanceID = m.currentInstanceID
	}

	// 检查浏览器连接是否仍然有效
	ctx := context.Background()
	if err := checkBrowserConnection(browser); err != nil {
		logger.Error(ctx, "Browser connection check failed: %v", err)
		return fmt.Errorf("browser connection is closed or invalid: %w", err)
	}

	// 保存当前语言设置,用于后续注入脚本时的文本替换
	if language == "" {
		language = "zh-CN" // 默认简体中文
	}
	m.currentLanguage = language

	// 根据URL匹配配置
	config := m.getConfigForURL(url)
	logger.Info(ctx, fmt.Sprintf("URL: %s, using configuration: %s, language: %s", url, config.Name, language))

	var page *rod.Page

	// 根据配置决定是否使用 stealth
	useStealth := true // 默认使用stealth
	if config.UseStealth != nil {
		useStealth = *config.UseStealth
	}

	if useStealth {
		page = stealth.MustPage(browser)
		logger.Info(ctx, "Using Stealth mode")
	} else {
		page = browser.MustPage()
		logger.Info(ctx, "Not using Stealth mode")
	}

	m.setPageWindow(page)

	// 设置 User Agent
	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"
	}
	page = page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: userAgent,
	})

	// 导航到目标 URL（设置60秒超时）
	if err := page.Timeout(60 * time.Second).Navigate(url); err != nil {
		return fmt.Errorf("failed to navigate to page: %w", err)
	}

	if err := page.Timeout(60 * time.Second).WaitLoad(); err != nil {
		logger.Warn(ctx, "Failed to wait for page load: %v", err)
	}

	// 为当前页面授予剪贴板权限
	pageInfo, _ := page.Info()
	if pageInfo != nil {
		grantPagePermissions := &proto.BrowserGrantPermissions{
			Origin: pageInfo.URL,
			Permissions: []proto.BrowserPermissionType{
				proto.BrowserPermissionTypeClipboardReadWrite,
				proto.BrowserPermissionTypeClipboardSanitizedWrite,
			},
		}
		if err := grantPagePermissions.Call(browser); err != nil {
			logger.Warn(ctx, "Failed to grant clipboard permissions for page: %v", err)
		} else {
			logger.Info(ctx, "✓ Clipboard permissions granted for page: %s", pageInfo.URL)
		}
	}

	if !noRecord {
		// 注入浮动录制按钮
		time.Sleep(500 * time.Millisecond) // 等待页面稳定
		// 替换浮动按钮脚本中的多语言占位符
		localizedFloatButtonScript := ReplaceI18nPlaceholders(floatButtonScript, m.currentLanguage, FloatButtonI18n)
		_, err := page.Eval(`() => { ` + localizedFloatButtonScript + ` return true; }`)
		if err != nil {
			logger.Warn(ctx, "Failed to inject float button script: %v", err)
		} else {
			logger.Info(ctx, "✓ Float recording button injected successfully (language: %s)", m.currentLanguage)

			// 设置 API 端口信息
			if m.config.Server != nil && m.config.Server.Port != "" {
				apiPort := m.config.Server.Port
				setPortScript := fmt.Sprintf(`() => { window.__browserwingAPIPort__ = "%s"; }`, apiPort)
				if _, err := page.Eval(setPortScript); err != nil {
					logger.Warn(ctx, "Failed to set API port: %v", err)
				}
			}
		}
		// 启动轮询检查页面内的录制请求
		go m.checkInPageRecordingRequests(ctx, page)
	}

	// 保存当前活动页面到指定实例
	if err := m.setInstanceActivePage(instanceID, page); err != nil {
		logger.Warn(ctx, "Failed to set active page: %v", err)
	}

	logger.Info(ctx, fmt.Sprintf("Page opened: %s", url))
	return nil
}

// getConfigForURL 根据URL获取匹配的配置
func (m *Manager) getConfigForURL(url string) *models.BrowserConfig {
	ctx := context.Background()
	logger.Info(ctx, fmt.Sprintf("Starting URL matching: %s, total %d site configurations", url, len(m.siteConfigs)))

	// 遍历所有网站特定配置，找到第一个匹配的
	for _, config := range m.siteConfigs {
		if config.URLPattern != "" {
			logger.Info(ctx, fmt.Sprintf("Trying to match pattern: %s (configuration: %s)", config.URLPattern, config.Name))
			// 使用正则表达式匹配
			matched, err := regexp.MatchString(config.URLPattern, url)
			if err != nil {
				logger.Info(ctx, fmt.Sprintf("Regular expression error: %v", err))
			} else if matched {
				logger.Info(ctx, fmt.Sprintf("✓ URL %s matched pattern %s (configuration: %s)", url, config.URLPattern, config.Name))
				return config
			} else {
				logger.Info(ctx, "✗ Not matched")
			}
		}
	}

	// 没有匹配的，返回默认配置
	logger.Info(ctx, "No matching site configuration found, using default configuration")

	// 如果默认配置未初始化，尝试加载或创建一个
	if m.defaultBrowserConfig == nil {
		logger.Info(ctx, "Default configuration not initialized, loading from database")
		defaultConfig, err := m.db.GetDefaultBrowserConfig()
		if err != nil {
			logger.Warn(ctx, "Failed to load default configuration, using system defaults")
			defaultConfig = m.getDefaultBrowserConfig()
		}
		m.defaultBrowserConfig = defaultConfig

		// 同时加载网站特定配置
		allConfigs, err := m.db.ListBrowserConfigs()
		if err != nil {
			logger.Warn(ctx, "Failed to load site configurations: %v", err)
			m.siteConfigs = []*models.BrowserConfig{}
		} else {
			m.siteConfigs = []*models.BrowserConfig{}
			for i := range allConfigs {
				if allConfigs[i].URLPattern != "" && !allConfigs[i].IsDefault {
					m.siteConfigs = append(m.siteConfigs, &allConfigs[i])
				}
			}
			logger.Info(ctx, "Loaded %d site-specific configurations", len(m.siteConfigs))
		}
	}

	return m.defaultBrowserConfig
}

// GetCurrentPageCookies 获取当前活动页面的所有 Cookie
func (m *Manager) GetCurrentPageCookies() (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning || m.browser == nil {
		return nil, fmt.Errorf("browser is not running")
	}

	// 获取浏览器的所有 Cookie
	cookies, err := m.browser.GetCookies()
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	return cookies, nil
}

// StartRecording 开始录制操作
// StartRecording 开始录制操作
// instanceID: 指定实例ID，空字符串表示使用当前实例
func (m *Manager) StartRecording(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	currentLang := m.currentLanguage
	if currentLang == "" {
		currentLang = "zh-CN" // 默认简体中文
	}
	m.mu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取指定实例的浏览器和活动页面
	_, activePage, _, err := m.getInstanceBrowser(instanceID)
	if err != nil {
		return err
	}

	if activePage == nil {
		return fmt.Errorf("please open a page first")
	}

	// 获取当前页面URL
	info, err := activePage.Info()
	if err != nil {
		return fmt.Errorf("failed to get page info: %w", err)
	}

	err = m.recorder.StartRecording(ctx, activePage, info.URL, currentLang)
	if err != nil {
		return err
	}

	// 启动录制后,显示录制UI面板
	_, _ = activePage.Eval(`() => {
		window.__isRecordingActive__ = true;
		if (typeof createRecorderUI === 'function') createRecorderUI();
		if (typeof createHighlightElement === 'function') createHighlightElement();
	}`)

	return nil
}

// StopRecording 停止录制
func (m *Manager) StopRecording(ctx context.Context) ([]models.ScriptAction, []models.DownloadedFile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	actions, err := m.recorder.StopRecording(ctx)
	if err != nil {
		return nil, nil, err
	}

	// 获取下载文件信息
	downloadedFiles := m.recorder.GetDownloadedFiles()

	return actions, downloadedFiles, nil
}

// IsRecording 检查是否正在录制
func (m *Manager) IsRecording() bool {
	return m.recorder.IsRecording()
}

// GetRecordingInfo 获取录制信息
func (m *Manager) GetRecordingInfo() map[string]interface{} {
	info := m.recorder.GetRecordingInfo()

	// 如果是页面内停止的录制,添加标记和actions
	m.mu.Lock()
	if m.inPageRecordingStopped {
		info["in_page_stopped"] = true
		info["actions"] = m.lastRecordedActions
		info["count"] = len(m.lastRecordedActions)
		info["downloaded_files"] = m.lastDownloadedFiles
		// 使用持久化的start_url
		if m.lastRecordedStartURL != "" {
			info["start_url"] = m.lastRecordedStartURL
		}
		// 不要清除标记,让前端显示完保存对话框后主动调用清除
	}
	m.mu.Unlock()

	return info
}

// ClearInPageRecordingState 清除页面内录制状态(供前端保存或取消后调用)
func (m *Manager) ClearInPageRecordingState() {
	m.mu.Lock()
	m.inPageRecordingStopped = false
	m.lastRecordedActions = nil
	m.lastRecordedStartURL = ""
	m.lastDownloadedFiles = nil
	m.mu.Unlock()
}

// PlayScript 回放脚本
// instanceID: 指定实例ID，空字符串表示使用当前实例
func (m *Manager) PlayScript(ctx context.Context, script *models.Script, instanceID string) (result *models.PlayResult, page *rod.Page, err error) {
	// 捕获 panic 并转换为错误
	defer func() {
		if r := recover(); r != nil {
			logger.Error(ctx, "Panic in PlayScript: %v", r)
			err = fmt.Errorf("failed to play script: browser connection may be closed (panic: %v)", r)
			result = nil
			page = nil
		}
	}()

	// 获取指定实例的浏览器
	browser, _, instance, err := m.getInstanceBrowser(instanceID)
	if err != nil {
		return nil, nil, err
	}

	// 检查浏览器连接是否仍然有效
	if err := checkBrowserConnection(browser); err != nil {
		logger.Error(ctx, "Browser connection check failed: %v", err)
		return nil, nil, fmt.Errorf("browser connection is closed or invalid: %w", err)
	}

	// 确定使用的实例ID（从 instance 对象获取，可能从空字符串转换为 default）
	usedInstanceID := instanceID
	instanceName := ""
	if instance != nil {
		usedInstanceID = instance.ID
		instanceName = instance.Name
	} else if usedInstanceID == "" {
		// 向后兼容：如果没有 instance 对象，使用 currentInstanceID
		usedInstanceID = m.currentInstanceID
	}

	// 创建执行记录
	executionID := fmt.Sprintf("%s-%d", script.ID, time.Now().UnixNano())
	execution := &models.ScriptExecution{
		ID:           executionID,
		ScriptID:     script.ID,
		ScriptName:   script.Name,
		InstanceID:   usedInstanceID,
		InstanceName: instanceName,
		StartTime:    time.Now(),
		TotalSteps:   len(script.Actions),
		CreatedAt:    time.Now(),
	}

	// 根据脚本的URL匹配配置
	scriptURL := script.URL
	if scriptURL == "" && len(script.Actions) > 0 {
		// 如果脚本没有URL，尝试从第一个action获取
		scriptURL = script.Actions[0].URL
	}

	config := m.getConfigForURL(scriptURL)
	logger.Info(ctx, fmt.Sprintf("Replay script URL: %s, using configuration: %s", scriptURL, config.Name))

	// 创建新页面用于回放
	// 根据配置决定是否使用 stealth
	useStealth := true // 默认使用stealth
	if config.UseStealth != nil {
		useStealth = *config.UseStealth
	}

	if useStealth {
		page = stealth.MustPage(browser)
		logger.Info(ctx, "Replay using Stealth mode")
	} else {
		page = browser.MustPage()
		logger.Info(ctx, "Replay not using Stealth mode")
	}

	m.setPageWindow(page)

	// 设置 User Agent
	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"
	}
	page = page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: userAgent,
	})

	// 为回放页面授予剪贴板权限
	if scriptURL != "" {
		grantPlayPermissions := &proto.BrowserGrantPermissions{
			Origin: scriptURL,
			Permissions: []proto.BrowserPermissionType{
				proto.BrowserPermissionTypeClipboardReadWrite,
				proto.BrowserPermissionTypeClipboardSanitizedWrite,
			},
		}
		if err := grantPlayPermissions.Call(browser); err != nil {
			logger.Warn(ctx, "Failed to grant clipboard permissions for playback: %v", err)
		} else {
			logger.Info(ctx, "✓ Clipboard permissions granted for playback")
		}
	}

	// 创建播放器，传入当前语言设置
	currentLang := m.currentLanguage
	if currentLang == "" {
		currentLang = "zh-CN" // 默认简体中文
	}
	player := NewPlayer(currentLang)

	// 设置下载路径并启动下载监听
	if m.downloadPath != "" {
		player.SetDownloadPath(m.downloadPath)
		player.StartDownloadListener(ctx, browser)
		logger.Info(ctx, "Download tracking enabled for playback, path: %s", m.downloadPath)
	}

	// 检查是否需要录制视频
	recordingConfig := m.db.GetDefaultRecordingConfig()
	var videoPath string
	if recordingConfig.Enabled {
		// 创建输出目录
		outputDir := recordingConfig.OutputDir
		if outputDir == "" {
			outputDir = "recordings"
		}
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			logger.Warn(ctx, "Failed to create recording directory: %v", err)
		} else {
			// 生成视频文件名
			timestamp := time.Now().Format("20060102_150405")
			// 强制使用 gif 格式
			videoPath = fmt.Sprintf("%s/%s_%s.gif", outputDir, script.Name, timestamp)

			// 开始录制
			frameRate := recordingConfig.FrameRate
			if frameRate <= 0 {
				frameRate = 15
			}
			quality := recordingConfig.Quality
			if quality <= 0 || quality > 100 {
				quality = 70
			}

			logger.Info(ctx, "Starting video recording: %s (frame rate: %d, quality: %d)", videoPath, frameRate, quality)
			if err := player.StartVideoRecording(page, videoPath, frameRate, quality); err != nil {
				logger.Warn(ctx, "Failed to start video recording: %v", err)
				videoPath = "" // 清空路径，表示录制失败
			}
		}
	}

	// 执行回放
	playErr := player.PlayScript(ctx, page, script, m.currentLanguage)

	// 停止下载监听
	if m.downloadPath != "" {
		player.StopDownloadListener(ctx)
	}

	// 停止视频录制
	if videoPath != "" {
		logger.Info(ctx, "Stopping video recording")
		if err := player.StopVideoRecording(videoPath, recordingConfig.FrameRate); err != nil {
			logger.Warn(ctx, "Failed to stop video recording: %v", err)
		} else {
			execution.VideoPath = videoPath
			logger.Info(ctx, "Video saved: %s", videoPath)
		}
	}

	// 记录结束时间和耗时
	execution.EndTime = time.Now()
	execution.Duration = execution.EndTime.Sub(execution.StartTime).Milliseconds()

	// 记录统计信息
	execution.SuccessSteps = player.GetSuccessCount()
	execution.FailedSteps = player.GetFailCount()
	execution.ExtractedData = player.GetExtractedData()

	// 判断是否成功
	if playErr != nil {
		execution.Success = false
		execution.ErrorMsg = playErr.Error()
		execution.Message = "Script execution failed"
	} else {
		execution.Success = true
		execution.Message = "Script execution successful"
	}

	// 保存执行记录到数据库
	if m.db != nil {
		if err := m.db.SaveScriptExecution(execution); err != nil {
			logger.Warn(ctx, "Failed to save script execution record: %v", err)
		} else {
			logger.Info(ctx, "Script execution record saved: %s", executionID)
		}
	}

	// 如果执行失败，返回错误
	if playErr != nil {
		return &models.PlayResult{
			Success: false,
			Message: playErr.Error(),
			Errors:  []string{playErr.Error()},
		}, page, playErr
	}

	// 返回回放结果，包含抓取的数据
	extractedData := player.GetExtractedData()
	logger.Info(ctx, "[PlayScript] Extracted data length: %d", len(extractedData))
	if len(extractedData) > 0 {
		keys := make([]string, 0, len(extractedData))
		for k := range extractedData {
			keys = append(keys, k)
		}
		logger.Info(ctx, "[PlayScript] Extracted data keys: %v", keys)
	}

	// 添加下载的文件路径到提取数据
	downloadedFiles := player.GetDownloadedFiles()
	if len(downloadedFiles) > 0 {
		extractedData["downloaded_files"] = downloadedFiles
		logger.Info(ctx, "[PlayScript] Downloaded files count: %d", len(downloadedFiles))
		for i, file := range downloadedFiles {
			logger.Info(ctx, "[PlayScript] Downloaded file #%d: %s", i+1, file)
		}
	}

	return &models.PlayResult{
		Success:       true,
		Message:       "Script replay completed",
		ExtractedData: extractedData,
	}, page, nil
}

// checkInPageRecordingRequests 检查页面内的录制控制请求
func (m *Manager) checkInPageRecordingRequests(ctx context.Context, page *rod.Page) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查是否有录制开始请求
			result, err := page.Eval(`() => {
				if (window.__startRecordingRequest__) {
					var req = window.__startRecordingRequest__;
					delete window.__startRecordingRequest__;
					return { type: 'start_recording', data: req };
				}
				if (window.__screenshotRequest__) {
					var req = window.__screenshotRequest__;
					delete window.__screenshotRequest__;
					return { type: 'screenshot', data: req };
				}
				return null;
			}`)

			if err == nil && result != nil && !result.Value.Nil() {
				// 解析请求类型
				resultMap, ok := result.Value.Val().(map[string]interface{})
				if !ok {
					continue
				}

				reqType, _ := resultMap["type"].(string)
				
				if reqType == "start_recording" {
					logger.Info(ctx, "Detected in-page recording start request")

					// 获取当前页面URL
					info, err := page.Info()
					if err == nil {
						// 获取当前语言设置
						currentLang := m.currentLanguage
						if currentLang == "" {
							currentLang = "zh-CN"
						}
						// 开始录制
						if err := m.recorder.StartRecording(ctx, page, info.URL, currentLang); err != nil {
							logger.Error(ctx, "Failed to start recording from in-page request: %v", err)
						} else {
							logger.Info(ctx, "✓ Recording started from in-page button")
							// 通知页面显示录制UI
							_, _ = page.Eval(`() => {
								window.__isRecordingActive__ = true;
								if (typeof createRecorderUI === 'function') createRecorderUI();
								if (typeof createHighlightElement === 'function') createHighlightElement();
							}`)
						}
					} else {
						logger.Error(ctx, "Failed to get page info for in-page recording start: %v", err)
					}
				} else if reqType == "screenshot" {
					logger.Info(ctx, "Detected in-page screenshot request")
					
					// 提取截图请求数据（仅用于日志）
					data, _ := resultMap["data"].(map[string]interface{})
					mode, _ := data["mode"].(string)
					
					if mode == "" {
						mode = "viewport"
					}
					
					// 注意：不在后端添加 action，因为前端已经通过 recordAction() 添加了
					// 停止录制时会从前端的 window.__recordedActions__ 同步过来
					// 这样避免重复添加截图操作
					logger.Info(ctx, "Screenshot action will be synced from frontend: mode=%s", mode)
				}
			}

			// 检查是否有录制停止请求
			stopResult, err := page.Eval(`() => {
				if (window.__stopRecordingRequest__) {
					var req = window.__stopRecordingRequest__;
					delete window.__stopRecordingRequest__;
					return req;
				}
				return null;
			}`)

			if err == nil && stopResult != nil && !stopResult.Value.Nil() {
				logger.Info(ctx, "Detected in-page recording stop request")

				// 获取录制信息(包含start_url)
				recInfo := m.recorder.GetRecordingInfo()

				// 停止录制并获取下载文件信息
				actions, err := m.recorder.StopRecording(ctx)
				downloadedFiles := m.recorder.GetDownloadedFiles()

				if err != nil {
					logger.Error(ctx, "Failed to stop recording from in-page request: %v", err)
				} else {
					logger.Info(ctx, "✓ Recording stopped from in-page button, %d actions recorded, %d files downloaded",
						len(actions), len(downloadedFiles))
					// 保存录制结果、下载文件和URL,供前端获取
					m.mu.Lock()
					m.lastRecordedActions = actions
					m.lastDownloadedFiles = downloadedFiles
					m.inPageRecordingStopped = true
					// 保存录制时的URL到持久化字段
					if startURL, ok := recInfo["start_url"].(string); ok && startURL != "" {
						m.lastRecordedStartURL = startURL
						logger.Info(ctx, "Saved start URL: %s", startURL)
					}
					m.mu.Unlock()

					// 通知页面:录制已停止
					_, _ = page.Eval(`() => {
						window.__recordingStoppedByInPage__ = true;
					}`)
				}
			}

		case <-ctx.Done():
			return
		}

		// 如果页面不再是活动页面,停止轮询
		m.mu.Lock()
		isActive := m.activePage == page
		m.mu.Unlock()
		if !isActive {
			return
		}
	}
}

// isHeadlessEnvironment 检测当前环境是否为无GUI环境
func isHeadlessEnvironment() bool {
	// 1. 优先检查是否在 Docker 容器中
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// 2. 检查 cgroup 文件是否包含 docker 或 containerd 标识（仅限 Linux）
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			return true
		}
	}

	// 3. 根据操作系统类型判断
	osType := strings.ToLower(os.Getenv("GOOS"))
	if osType == "" {
		// 如果 GOOS 环境变量不存在，使用 runtime.GOOS
		osType = strings.ToLower(runtime.GOOS)
	}

	// Windows 和 macOS 默认有 GUI 环境
	if osType == "windows" || osType == "darwin" {
		return false
	}

	// 4. Linux 环境下检查 DISPLAY 和 WAYLAND_DISPLAY 环境变量
	if osType == "linux" {
		display := os.Getenv("DISPLAY")
		waylandDisplay := os.Getenv("WAYLAND_DISPLAY")

		// 如果两个环境变量都为空，则认为是无GUI环境
		if display == "" && waylandDisplay == "" {
			return true
		}
	}

	// 默认认为有 GUI 环境
	return false
}

// GetDefaultBrowserConfig 获取默认浏览器配置（公开方法）
func (m *Manager) GetDefaultBrowserConfig() *models.BrowserConfig {
	return m.getDefaultBrowserConfig()
}

// getDefaultBrowserConfig 获取默认浏览器配置
func (m *Manager) getDefaultBrowserConfig() *models.BrowserConfig {
	useStealth := true
	// 根据环境自动设置 headless 默认值
	// 如果是无GUI环境（Docker、Linux服务器等），默认使用 headless 模式
	headless := isHeadlessEnvironment()

	// 记录环境检测结果
	osType := runtime.GOOS
	display := os.Getenv("DISPLAY")
	waylandDisplay := os.Getenv("WAYLAND_DISPLAY")

	logger.Info(context.Background(),
		"detected browser environment: OS=%s, DISPLAY=%s, WAYLAND_DISPLAY=%s, headless=%v",
		osType, display, waylandDisplay, headless)

	return &models.BrowserConfig{
		ID:          "default",
		Name:        "默认配置",
		Description: "系统默认浏览器配置，适用于所有网站",
		URLPattern:  "", // 空表示默认配置
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
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ==================== 多实例管理 ====================

// checkBrowserConnection 检查浏览器连接是否仍然有效
func checkBrowserConnection(browser *rod.Browser) error {
	if browser == nil {
		return fmt.Errorf("browser is nil")
	}

	// 尝试获取浏览器页面列表，这是一个轻量级的检查
	// 如果连接已关闭，这个调用会失败
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := browser.Context(ctx).Pages()
	if err != nil {
		return fmt.Errorf("failed to get browser pages: connection may be closed: %w", err)
	}

	return nil
}

// cleanupSingletonLock 清理 Chrome 的进程单例锁文件
// 这些锁文件在 Chrome 异常退出时可能没有被清理，导致无法启动新实例
func (m *Manager) cleanupSingletonLock(ctx context.Context, userDataDir string) error {
	// Chrome 在用户数据目录中创建的锁文件
	lockFiles := []string{
		"SingletonLock",
		"SingletonCookie",
		"SingletonSocket",
	}

	var cleanedFiles []string
	var failedFiles []string

	for _, lockFile := range lockFiles {
		lockPath := filepath.Join(userDataDir, lockFile)

		// 检查文件是否存在
		if _, err := os.Stat(lockPath); err == nil {
			// 尝试删除锁文件，最多重试 3 次
			deleted := false
			for attempt := 1; attempt <= 3; attempt++ {
				if err := os.Remove(lockPath); err != nil {
					if attempt < 3 {
						// 等待一小段时间后重试
						time.Sleep(100 * time.Millisecond)
						continue
					}
					// 最后一次尝试失败
					logger.Warn(ctx, "Failed to remove lock file %s after %d attempts: %v", lockFile, attempt, err)
					failedFiles = append(failedFiles, lockFile)
				} else {
					deleted = true
					break
				}
			}

			if deleted {
				cleanedFiles = append(cleanedFiles, lockFile)
			}
		}
	}

	if len(cleanedFiles) > 0 {
		logger.Info(ctx, "Cleaned up lock files: %v", cleanedFiles)
	}

	if len(failedFiles) > 0 {
		logger.Warn(ctx, "Failed to clean some lock files: %v (may need manual cleanup or process is still running)", failedFiles)
	}

	return nil
}

// getInstanceBrowser 获取指定实例的浏览器和活动页面
// 如果 instanceID 为空，则使用当前实例
// 如果 default 实例未运行，会自动启动它
// 返回: browser, activePage, instance, error
func (m *Manager) getInstanceBrowser(instanceID string) (*rod.Browser, *rod.Page, *models.BrowserInstance, error) {
	// 如果没有指定实例ID，使用当前实例
	if instanceID == "" {
		instanceID = m.currentInstanceID
	}

	// 如果还是空，说明没有运行中的实例
	if instanceID == "" {
		// 向后兼容：检查旧的 browser 字段
		if m.isRunning && m.browser != nil {
			return m.browser, m.activePage, nil, nil
		}

		// 尝试使用 default 实例
		instanceID = "default"
		ctx := context.Background()
		logger.Info(ctx, "No current instance, attempting to use default instance")
	}

	// 获取实例运行时信息
	runtime, exists := m.instances[instanceID]
	if !exists || runtime == nil {
		// 如果是 default 实例且未运行，尝试自动启动
		if instanceID == "default" {
			ctx := context.Background()
			logger.Info(ctx, "Default instance not running, attempting to auto-start...")

			// 调用内部启动函数（调用者已持有锁）
			err := m.startInstanceInternal(ctx, "default")
			if err != nil {
				logger.Error(ctx, "Failed to auto-start default instance: %v", err)
				return nil, nil, nil, fmt.Errorf("default instance not running and failed to start: %w", err)
			}

			logger.Info(ctx, "✓ Default instance auto-started successfully")

			// 重新获取运行时信息
			runtime, exists = m.instances["default"]
			if !exists || runtime == nil {
				return nil, nil, nil, fmt.Errorf("default instance started but runtime not found")
			}

			return runtime.browser, runtime.activePage, runtime.instance, nil
		}

		return nil, nil, nil, fmt.Errorf("instance %s is not running", instanceID)
	}

	return runtime.browser, runtime.activePage, runtime.instance, nil
}

// setInstanceActivePage 设置指定实例的活动页面
func (m *Manager) setInstanceActivePage(instanceID string, page *rod.Page) error {
	// 如果没有指定实例ID，使用当前实例
	if instanceID == "" {
		instanceID = m.currentInstanceID
	}

	// 如果还是空，说明没有运行中的实例
	if instanceID == "" {
		// 向后兼容：设置旧的 activePage 字段
		if m.isRunning && m.browser != nil {
			m.activePage = page
			return nil
		}
		return fmt.Errorf("no running instance available")
	}

	// 获取实例运行时信息
	runtime, exists := m.instances[instanceID]
	if !exists || runtime == nil {
		return fmt.Errorf("instance %s is not running", instanceID)
	}

	runtime.activePage = page

	// 如果是当前实例，也更新向后兼容的字段
	if instanceID == m.currentInstanceID {
		m.activePage = page
	}

	return nil
}

// StartInstance 启动指定浏览器实例
func (m *Manager) StartInstance(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.startInstanceInternal(ctx, instanceID)
}

// startInstanceInternal 内部启动函数，调用者必须已持有锁
func (m *Manager) startInstanceInternal(ctx context.Context, instanceID string) error {
	// 检查实例是否已启动
	if runtime, exists := m.instances[instanceID]; exists && runtime != nil {
		return fmt.Errorf("instance %s is already running", instanceID)
	}

	// 从数据库加载实例配置
	instance, err := m.db.GetBrowserInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to load instance: %w", err)
	}

	logger.Info(ctx, "Starting browser instance: %s (%s)", instance.Name, instance.Type)

	var browser *rod.Browser
	var launcherObj *launcher.Launcher
	var url string

	if instance.Type == "remote" {
		// 远程模式
		if instance.ControlURL == "" {
			return fmt.Errorf("control_url is required for remote instance")
		}
		controlURL := instance.ControlURL
		logger.Info(ctx, "Connecting to remote browser: %s", controlURL)

		// 解析 WebSocket URL
		wsURL, err := resolveWebSocketURL(controlURL)
		if err != nil {
			return fmt.Errorf("failed to resolve WebSocket URL from %s: %w", controlURL, err)
		}
		logger.Info(ctx, "Resolved WebSocket URL: %s", wsURL)
		url = wsURL

		browser = rod.New().ControlURL(url)
	} else {
		// 本地模式
		logger.Info(ctx, "Starting local browser instance...")

		// 创建启动器
		headless := false
		if instance.Headless != nil {
			headless = *instance.Headless
		}

		l := launcher.New().
			Headless(headless).
			Devtools(false).
			Leakless(false)

		// 设置代理
		if instance.Proxy != "" {
			l = l.Proxy(instance.Proxy)
			logger.Info(ctx, "Using proxy: %s", instance.Proxy)
		}

		// 设置启动参数
		launchArgs := instance.LaunchArgs
		if len(launchArgs) == 0 {
			// 使用默认启动参数
			launchArgs = []string{
				"disable-blink-features=AutomationControlled",
				"excludeSwitches=enable-automation",
				"no-first-run",
				"no-default-browser-check",
				"window-size=1920,1080",
				"start-maximized",
			}
		}

		for _, arg := range launchArgs {
			arg = strings.TrimPrefix(arg, "--")
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				l = l.Set(flags.Flag(parts[0]), parts[1])
			} else {
				l = l.Set(flags.Flag(arg))
			}
		}

		// 设置浏览器路径
		binPath := instance.BinPath
		if binPath == "" {
			// 如果没有指定路径，尝试查找系统中的 Chrome
			logger.Info(ctx, "BinPath not specified, searching for system Chrome...")
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
					logger.Info(ctx, "Found Chrome at: %s", binPath)
					break
				}
			}

			// 如果配置文件中有指定路径，优先使用
			if m.config.Browser != nil && m.config.Browser.BinPath != "" {
				binPath = m.config.Browser.BinPath
				logger.Info(ctx, "Using browser path from config: %s", binPath)
			}
		}

		if binPath != "" {
			l = l.Bin(binPath)
			logger.Info(ctx, "Using browser path: %s", binPath)
		} else {
			logger.Warn(ctx, "No browser path found, will use launcher default (may download Chrome)")
		}

		// 设置用户数据目录
		if instance.UserDataDir != "" {
			if err := os.MkdirAll(instance.UserDataDir, 0o755); err != nil {
				logger.Warn(ctx, "Failed to create user data directory: %v", err)
			} else {
				// 主动清理可能存在的锁文件（在启动前）
				logger.Info(ctx, "Checking and cleaning up lock files before launch...")
				if err := m.cleanupSingletonLock(ctx, instance.UserDataDir); err != nil {
					logger.Warn(ctx, "Failed to cleanup singleton lock: %v", err)
				}

				l = l.UserDataDir(instance.UserDataDir)
				logger.Info(ctx, "Using user data directory: %s", instance.UserDataDir)
			}
		}

		// 启动浏览器
		url, err = l.Launch()
		if err != nil {
			// 如果启动失败，尝试再次清理锁文件并给出提示
			if instance.UserDataDir != "" && strings.Contains(err.Error(), "SingletonLock") {
				logger.Error(ctx, "Browser launch failed due to SingletonLock, attempting cleanup...")
				m.cleanupSingletonLock(ctx, instance.UserDataDir)
				return fmt.Errorf("failed to launch browser (SingletonLock issue): %w\nTip: The lock files have been cleaned up. Please try starting the instance again", err)
			}
			return fmt.Errorf("failed to launch browser: %w", err)
		}

		browser = rod.New().ControlURL(url)
		launcherObj = l
		logger.Info(ctx, "Browser launched: %s", url)
	}

	// 连接浏览器
	if err := browser.Connect(); err != nil {
		if launcherObj != nil {
			launcherObj.Kill()
		}
		return fmt.Errorf("failed to connect browser: %w", err)
	}

	// 关键：在浏览器连接后立即设置XHR拦截器，确保所有页面（包括后续打开的）都会自动监听XHR
	// 这样用户在点击"开始录制"之前打开的页面，也能捕获到XHR请求
	logger.Info(ctx, "Setting up XHR interceptor for all pages...")

	// 获取所有现有页面并注入XHR拦截器
	pages, err := browser.Pages()
	if err == nil {
		for _, page := range pages {
			if page != nil {
				// 为每个现有页面设置EvalOnNewDocument
				_, evalErr := page.EvalOnNewDocument(xhrInterceptorScriptForManager)
				if evalErr != nil {
					logger.Warn(ctx, "Failed to set EvalOnNewDocument for existing page: %v", evalErr)
				}

				// 立即在现有页面注入（以防页面已经加载）
				_, injectErr := page.Eval(`() => { ` + xhrInterceptorScriptForManager + ` return true; }`)
				if injectErr != nil {
					logger.Warn(ctx, "Failed to inject XHR interceptor into existing page: %v", injectErr)
				}
			}
		}
		logger.Info(ctx, "✓ XHR interceptor configured for %d existing pages", len(pages))
	}

	// 为浏览器设置默认的EvalOnNewDocument（影响所有新页面）
	// 注意：rod目前没有直接在Browser级别设置EvalOnNewDocument的API
	// 所以我们需要在页面创建时处理，见下面的页面监听逻辑

	logger.Info(ctx, "✓ XHR interceptor setup completed")

	// 设置下载行为
	if m.downloadPath == "" {
		downloadPath := "./downloads"
		absDownloadPath, err := os.Getwd()
		if err == nil {
			downloadPath = absDownloadPath + "/downloads"
		}
		os.MkdirAll(downloadPath, 0o755)
		m.downloadPath = downloadPath
		m.recorder.SetDownloadPath(downloadPath)
	}

	downloadBehavior := &proto.BrowserSetDownloadBehavior{
		Behavior:      proto.BrowserSetDownloadBehaviorBehaviorAllow,
		DownloadPath:  m.downloadPath,
		EventsEnabled: true,
	}
	if err := downloadBehavior.Call(browser); err != nil {
		logger.Warn(ctx, "Failed to set download behavior: %v", err)
	}

	// 授予剪贴板权限
	grantPermissions := &proto.BrowserGrantPermissions{
		Permissions: []proto.BrowserPermissionType{
			proto.BrowserPermissionTypeClipboardReadWrite,
			proto.BrowserPermissionTypeClipboardSanitizedWrite,
		},
	}
	if err := grantPermissions.Call(browser); err != nil {
		logger.Warn(ctx, "Failed to grant clipboard permissions: %v", err)
	}

	// 创建运行时信息
	runtime := &BrowserInstanceRuntime{
		instance:  instance,
		browser:   browser,
		launcher:  launcherObj,
		startTime: time.Now(),
	}

	m.instances[instanceID] = runtime

	// 更新实例状态为运行中
	instance.IsActive = true
	instance.UpdatedAt = time.Now()
	if err := m.db.SaveBrowserInstance(instance); err != nil {
		logger.Warn(ctx, "Failed to update instance status: %v", err)
	}

	// 如果是第一个启动的实例或者是默认实例，设置为当前实例
	if m.currentInstanceID == "" || instance.IsDefault {
		m.currentInstanceID = instanceID
	}

	// 如果启动的是当前实例，更新向后兼容的旧字段
	if m.currentInstanceID == instanceID {
		m.browser = browser
		m.launcher = launcherObj
		m.isRunning = true
		m.startTime = runtime.startTime
	}

	// 启动新页面监听，自动为新打开的页面注入XHR拦截器
	go m.watchForNewPagesXHR(ctx, browser, instanceID)

	logger.Info(ctx, "✓ Browser instance started: %s", instance.Name)
	return nil
}

// watchForNewPagesXHR 监听新页面创建并自动注入XHR拦截器
// 这确保了用户在点击"开始录制"之前打开的所有页面都能捕获XHR请求
func (m *Manager) watchForNewPagesXHR(ctx context.Context, browser *rod.Browser, instanceID string) {
	logger.Info(ctx, "Starting XHR interceptor watcher for instance: %s", instanceID)

	// 记录已处理的页面
	processedPages := make(map[string]bool)

	// 定时检查新页面（每秒检查一次）
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查实例是否还在运行
			m.mu.Lock()
			runtime, exists := m.instances[instanceID]
			m.mu.Unlock()

			if !exists || runtime == nil {
				logger.Info(ctx, "Instance %s stopped, stopping XHR watcher", instanceID)
				return
			}

			// 获取所有页面
			pages, err := browser.Pages()
			if err != nil {
				continue
			}

			// 为每个新页面注入XHR拦截器
			for _, page := range pages {
				if page == nil {
					continue
				}

				pageInfo, err := page.Info()
				if err != nil {
					continue
				}

				targetID := string(pageInfo.TargetID)

				// 跳过已处理的页面
				if processedPages[targetID] {
					continue
				}

				// 跳过特殊页面
				if strings.HasPrefix(pageInfo.URL, "chrome://") ||
					strings.HasPrefix(pageInfo.URL, "chrome-extension://") ||
					strings.HasPrefix(pageInfo.URL, "devtools://") ||
					strings.HasPrefix(pageInfo.URL, "about:") {
					processedPages[targetID] = true
					continue
				}

				// 标记为已处理
				processedPages[targetID] = true

				logger.Info(ctx, "Injecting XHR interceptor into new page: %s (URL: %s)", targetID, pageInfo.URL)

				// 为新页面设置EvalOnNewDocument（影响该页面内的iframe和导航）
				_, evalErr := page.EvalOnNewDocument(xhrInterceptorScriptForManager)
				if evalErr != nil {
					logger.Warn(ctx, "Failed to set EvalOnNewDocument for page %s: %v", targetID, evalErr)
				}

				// 立即在页面注入（以防页面已经加载）
				go func(p *rod.Page, tid string) {
					_, injectErr := p.Eval(`() => { ` + xhrInterceptorScriptForManager + ` return true; }`)
					if injectErr != nil {
						logger.Warn(ctx, "Failed to inject XHR interceptor into page %s: %v", tid, injectErr)
					} else {
						logger.Info(ctx, "✓ XHR interceptor injected into page: %s", tid)
					}
				}(page, targetID)
			}
		case <-ctx.Done():
			logger.Info(ctx, "Context cancelled, stopping XHR watcher for instance: %s", instanceID)
			return
		}
	}
}

// StopInstance 停止指定浏览器实例
func (m *Manager) StopInstance(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime, exists := m.instances[instanceID]
	if !exists || runtime == nil {
		return fmt.Errorf("instance %s is not running", instanceID)
	}

	logger.Info(ctx, "Stopping browser instance: %s", runtime.instance.Name)

	isRemote := runtime.instance.Type == "remote"

	// 关闭浏览器
	if runtime.browser != nil {
		if !isRemote {
			// 关闭所有页面
			if pages, err := runtime.browser.Pages(); err == nil {
				for _, page := range pages {
					_ = page.Close()
				}
			}
			time.Sleep(1 * time.Second)
		}

		if err := runtime.browser.Close(); err != nil {
			logger.Warn(ctx, "Error closing browser: %v", err)
		}
	}

	// 终止本地浏览器进程
	if !isRemote && runtime.launcher != nil {
		time.Sleep(1 * time.Second)
		runtime.launcher.Kill()
		logger.Info(ctx, "Browser process terminated")
	}

	// 清理本地实例的锁文件
	if !isRemote && runtime.instance.UserDataDir != "" {
		// 等待浏览器完全退出
		time.Sleep(500 * time.Millisecond)

		if err := m.cleanupSingletonLock(ctx, runtime.instance.UserDataDir); err != nil {
			logger.Warn(ctx, "Failed to cleanup singleton lock after stop: %v", err)
		} else {
			logger.Info(ctx, "✓ Cleaned up singleton lock files for stopped instance")
		}
	}

	// 更新实例状态
	runtime.instance.IsActive = false
	runtime.instance.UpdatedAt = time.Now()
	if err := m.db.SaveBrowserInstance(runtime.instance); err != nil {
		logger.Warn(ctx, "Failed to update instance status: %v", err)
	}

	// 删除运行时信息
	delete(m.instances, instanceID)

	// 如果停止的是当前实例，清空当前实例 ID
	if m.currentInstanceID == instanceID {
		m.currentInstanceID = ""

		// 向后兼容：清空旧字段
		m.browser = nil
		m.launcher = nil
		m.isRunning = false
		m.activePage = nil

		// 尝试切换到第一个运行中的实例
		for id := range m.instances {
			m.currentInstanceID = id
			runtime := m.instances[id]
			m.browser = runtime.browser
			m.launcher = runtime.launcher
			m.isRunning = true
			m.startTime = runtime.startTime
			break
		}
	}

	logger.Info(ctx, "✓ Browser instance stopped: %s", runtime.instance.Name)
	return nil
}

// SwitchInstance 切换当前活动实例
func (m *Manager) SwitchInstance(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 从数据库获取实例信息，验证实例是否存在
	instance, err := m.db.GetBrowserInstance(instanceID)
	if err != nil {
		return fmt.Errorf("instance %s not found: %w", instanceID, err)
	}

	// 设置当前实例ID（无论实例是否运行）
	m.currentInstanceID = instanceID

	// 检查实例是否运行
	runtime, exists := m.instances[instanceID]
	if exists && runtime != nil {
		// 实例正在运行，更新旧字段以保持向后兼容
		m.browser = runtime.browser
		m.launcher = runtime.launcher
		m.isRunning = true
		m.startTime = runtime.startTime
		m.activePage = runtime.activePage
		logger.Info(ctx, "Switched to running instance: %s", instance.Name)
	} else {
		// 实例未运行，清空旧字段
		m.browser = nil
		m.launcher = nil
		m.isRunning = false
		m.startTime = time.Time{}
		m.activePage = nil
		logger.Info(ctx, "Switched to stopped instance: %s (not running)", instance.Name)
	}

	return nil
}

// GetCurrentInstance 获取当前活动实例
func (m *Manager) GetCurrentInstance() *models.BrowserInstance {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentInstanceID == "" {
		return nil
	}

	// 首先尝试从运行时获取（如果实例正在运行）
	runtime, exists := m.instances[m.currentInstanceID]
	if exists && runtime != nil {
		return runtime.instance
	}

	// 如果实例未运行，从数据库获取
	instance, err := m.db.GetBrowserInstance(m.currentInstanceID)
	if err != nil {
		return nil
	}

	return instance
}

// GetInstanceRuntime 获取指定实例的运行时信息
func (m *Manager) GetInstanceRuntime(instanceID string) (*BrowserInstanceRuntime, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runtime, exists := m.instances[instanceID]
	if !exists || runtime == nil {
		return nil, fmt.Errorf("instance %s is not running", instanceID)
	}

	return runtime, nil
}

// ListRunningInstances 列出所有运行中的实例
func (m *Manager) ListRunningInstances() []*models.BrowserInstance {
	m.mu.Lock()
	defer m.mu.Unlock()

	var instances []*models.BrowserInstance
	for _, runtime := range m.instances {
		if runtime != nil && runtime.instance != nil {
			instances = append(instances, runtime.instance)
		}
	}

	return instances
}
