package browser

import (
	"context"
	_ "embed"
	"fmt"
	"os"
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

// Manager 浏览器管理器
type Manager struct {
	config                 *config.Config
	db                     *storage.BoltDB
	llmManager             *llm.Manager
	browser                *rod.Browser
	launcher               *launcher.Launcher
	mu                     sync.Mutex
	isRunning              bool
	startTime              time.Time
	recorder               *Recorder
	activePage             *rod.Page               // 当前活动页面
	defaultBrowserConfig   *models.BrowserConfig   // 默认浏览器配置
	siteConfigs            []*models.BrowserConfig // 网站特定配置列表
	lastRecordedActions    []models.ScriptAction   // 最后一次录制的动作(用于页面内停止录制)
	lastRecordedStartURL   string                  // 最后一次录制的起始URL(用于页面内停止录制)
	lastDownloadedFiles    []models.DownloadedFile // 最后一次录制下载的文件(用于页面内停止录制)
	inPageRecordingStopped bool                    // 标记是否是页面内停止的录制
	currentLanguage        string                  // 当前前端语言设置
	downloadPath           string                  // 下载目录路径
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
				l = l.UserDataDir(userDataDir)
				logger.Info(ctx, fmt.Sprintf("✓ Using user data directory: %s", userDataDir))
			}
		}
	} else {
		logger.Warn(ctx, "User data directory not configured, login state will not be saved")
	}

	logger.Info(ctx, "Starting browser process...")
	// 启动浏览器
	url, err := l.Launch()
	if err != nil {
		errMsg := err.Error()
		logger.Error(ctx, "Failed to start browser, detailed error: %v", err)

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
	browser := rod.New().ControlURL(url)
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

	m.launcher = l
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
	logger.Info(ctx, "Closing browser...")

	// 1. 先关闭所有页面，让浏览器有机会保存数据
	if m.browser != nil {
		pages, err := m.browser.Pages()
		if err == nil {
			for _, page := range pages {
				_ = page.Close()
			}
			logger.Info(ctx, fmt.Sprintf("Closed %d pages", len(pages)))
		}

		// 2. 等待一下，让浏览器保存数据
		time.Sleep(1 * time.Second)

		// 3. 优雅关闭浏览器连接
		if err := m.browser.Close(); err != nil {
			logger.Warn(ctx, fmt.Sprintf("Error when closing browser connection: %v", err))
		}
	}

	// 4. 再等待一下，确保数据完全写入磁盘
	time.Sleep(1 * time.Second)

	// 5. ⚠️ 重要：不调用 launcher.Cleanup()，因为它会删除用户数据目录！
	// 浏览器进程会在连接关闭后自动退出
	// 如果需要强制杀死进程，可以调用 launcher.Kill() 而不是 Cleanup()
	if m.launcher != nil {
		// 只杀死进程，不清理目录
		m.launcher.Kill()
		logger.Info(ctx, "Browser process terminated")
	}

	m.browser = nil
	m.launcher = nil
	m.isRunning = false

	logger.Info(ctx, "Browser fully closed, user data saved")
	return nil
}

// IsRunning 检查浏览器是否运行
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isRunning
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
func (m *Manager) OpenPage(url string, language string, norecord ...bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var noRecord bool
	if len(norecord) > 0 {
		noRecord = norecord[0]
	}

	if !m.isRunning || m.browser == nil {
		return fmt.Errorf("browser is not running")
	}

	// 保存当前语言设置,用于后续注入脚本时的文本替换
	if language == "" {
		language = "zh-CN" // 默认简体中文
	}
	m.currentLanguage = language

	// 根据URL匹配配置
	config := m.getConfigForURL(url)
	ctx := context.Background()
	logger.Info(ctx, fmt.Sprintf("URL: %s, using configuration: %s, language: %s", url, config.Name, language))

	var page *rod.Page

	// 根据配置决定是否使用 stealth
	useStealth := true // 默认使用stealth
	if config.UseStealth != nil {
		useStealth = *config.UseStealth
	}

	if useStealth {
		page = stealth.MustPage(m.browser)
		logger.Info(ctx, "Using Stealth mode")
	} else {
		page = m.browser.MustPage()
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
		if err := grantPagePermissions.Call(m.browser); err != nil {
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

	// 保存当前活动页面
	m.activePage = page

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
func (m *Manager) StartRecording(ctx context.Context) error {
	m.mu.Lock()
	currentLang := m.currentLanguage
	if currentLang == "" {
		currentLang = "zh-CN" // 默认简体中文
	}
	m.mu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning || m.browser == nil {
		return fmt.Errorf("browser is not running")
	}

	if m.activePage == nil {
		return fmt.Errorf("please open a page first")
	}

	// 获取当前页面URL
	info, err := m.activePage.Info()
	if err != nil {
		return fmt.Errorf("failed to get page info: %w", err)
	}

	err = m.recorder.StartRecording(ctx, m.activePage, info.URL, currentLang)
	if err != nil {
		return err
	}

	// 启动录制后,显示录制UI面板
	_, _ = m.activePage.Eval(`() => {
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
func (m *Manager) PlayScript(ctx context.Context, script *models.Script) (*models.PlayResult, *rod.Page, error) {
	if !m.isRunning || m.browser == nil {
		return nil, nil, fmt.Errorf("browser is not running")
	}

	// 创建执行记录
	executionID := fmt.Sprintf("%s-%d", script.ID, time.Now().UnixNano())
	execution := &models.ScriptExecution{
		ID:         executionID,
		ScriptID:   script.ID,
		ScriptName: script.Name,
		StartTime:  time.Now(),
		TotalSteps: len(script.Actions),
		CreatedAt:  time.Now(),
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
	var page *rod.Page

	// 根据配置决定是否使用 stealth
	useStealth := true // 默认使用stealth
	if config.UseStealth != nil {
		useStealth = *config.UseStealth
	}

	if useStealth {
		page = stealth.MustPage(m.browser)
		logger.Info(ctx, "Replay using Stealth mode")
	} else {
		page = m.browser.MustPage()
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
		if err := grantPlayPermissions.Call(m.browser); err != nil {
			logger.Warn(ctx, "Failed to grant clipboard permissions for playback: %v", err)
		} else {
			logger.Info(ctx, "✓ Clipboard permissions granted for playback")
		}
	}

	player := NewPlayer()

	// 设置下载路径并启动下载监听
	if m.downloadPath != "" {
		player.SetDownloadPath(m.downloadPath)
		player.StartDownloadListener(ctx, m.browser)
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
	playErr := player.PlayScript(ctx, page, script)

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
					return req;
				}
				return null;
			}`)

			if err == nil && result != nil && !result.Value.Nil() {
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
