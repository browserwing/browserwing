package browser

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/browserwing/browserwing/llm"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

//go:embed scripts/recorder.js
var recorderScript string

//go:embed scripts/iframe_recorder.js
var iframeRecorderScript string

//go:embed scripts/iframe_listener.js
var iframeMessageListenerScript string

// Recorder 浏览器操作录制器
type Recorder struct {
	mu              sync.Mutex
	isRecording     bool
	startTime       time.Time
	startURL        string
	actions         []models.ScriptAction
	page            *rod.Page
	syncTicker      *time.Ticker
	syncStopChan    chan bool
	lastSyncedCount int
	apiServerPort   string       // API 服务器端口
	llmManager      *llm.Manager // LLM 管理器
}

// NewRecorder 创建录制器
func NewRecorder() *Recorder {
	return &Recorder{
		actions:       make([]models.ScriptAction, 0),
		apiServerPort: "8080", // 默认端口
	}
}

// SetLLMManager 设置 LLM 管理器
func (r *Recorder) SetLLMManager(manager *llm.Manager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llmManager = manager
}

// SetAPIServerPort 设置 API 服务器端口
func (r *Recorder) SetAPIServerPort(port string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.apiServerPort = port
}

// StartRecording 开始录制
func (r *Recorder) StartRecording(ctx context.Context, page *rod.Page, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRecording {
		return fmt.Errorf("recording is already in progress")
	}

	r.isRecording = true
	r.startTime = time.Now()
	r.startURL = url
	r.actions = make([]models.ScriptAction, 0)
	r.page = page

	logger.Info(ctx, "Preparing to inject recording script into page...")

	// 等待页面完全加载
	if err := page.WaitLoad(); err != nil {
		logger.Warn(ctx, "Failed to wait for page to load: %v", err)
	}

	// 等待一下让页面稳定
	time.Sleep(500 * time.Millisecond)

	// 禁用 CSP 以允许向 localhost API 发送请求
	// 这对于像 Twitter 这样有严格 CSP 策略的网站是必需的
	err := proto.PageSetBypassCSP{Enabled: true}.Call(page)
	if err != nil {
		logger.Warn(ctx, "Failed to disable CSP: %v", err)
	} else {
		logger.Info(ctx, "✓ CSP restrictions disabled, can call localhost API")
	}

	// 先测试页面是否可以执行脚本
	testResult, testErr := page.Eval(`() => { return 1 + 1; }`)
	if testErr != nil {
		logger.Error(ctx, "Page script execution test failed: %v", testErr)
		r.isRecording = false
		return fmt.Errorf("page does not support script execution: %w", testErr)
	}
	logger.Info(ctx, "Page script test result: %v", testResult.Value)

	// 设置录制模式标志,让脚本知道这是录制模式
	_, err = page.Eval(`() => { window.__browserwingRecordingMode__ = true; }`)
	if err != nil {
		logger.Warn(ctx, "Failed to set recording mode flag: %v", err)
	}

	// 注入录制脚本 - 使用立即执行函数表达式
	_, err = page.Eval(`() => { ` + recorderScript + ` return true; }`)
	if err != nil {
		r.isRecording = false
		logger.Error(ctx, "Failed to inject script, error details: %v", err)

		// 尝试检查页面状态
		pageInfo, _ := page.Info()
		if pageInfo != nil {
			logger.Error(ctx, "Page URL: %s", pageInfo.URL)
		}

		return fmt.Errorf("failed to inject recording script: %w", err)
	}

	logger.Info(ctx, "✓ Recording script injected successfully")

	// 验证注入是否成功
	checkResult, checkErr := page.Eval(`() => window.__browserwingRecorder__`)
	if checkErr == nil && checkResult != nil {
		logger.Info(ctx, "✓ Recorder status verified: %v", checkResult.Value)
	}

	// 注入 iframe 消息监听器
	_, err = page.Eval(`() => { ` + iframeMessageListenerScript + ` return true; }`)
	if err != nil {
		logger.Warn(ctx, "Failed to inject iframe message listener: %v", err)
	} else {
		logger.Info(ctx, "✓ iframe message listener injected successfully")
	}

	// 为所有现有的 iframe 注入录制脚本
	r.injectIframeRecorders(ctx, page)

	// 监听新创建的 iframe
	go r.watchForNewIframes(ctx, page)

	// 监听页面导航事件,在新页面自动重新注入录制脚本
	go r.watchForPageNavigation(ctx, page)

	logger.Info(ctx, "Starting recording operation, URL: %s", url)

	// 启动定期同步协程，每500ms同步一次浏览器中的操作（更频繁，减少丢失风险）
	r.syncTicker = time.NewTicker(500 * time.Millisecond)
	r.syncStopChan = make(chan bool)
	r.lastSyncedCount = 0

	go r.syncActionsFromBrowser(ctx)

	return nil
}

// syncActionsFromBrowser 定期从浏览器同步录制的操作
func (r *Recorder) syncActionsFromBrowser(ctx context.Context) {
	for {
		select {
		case <-r.syncTicker.C:
			r.mu.Lock()
			if !r.isRecording || r.page == nil {
				r.mu.Unlock()
				return
			}

			// 检查是否有 AI 提取请求
			r.checkAndProcessAIRequest(ctx)

			// 从浏览器获取当前录制的所有操作（优先从 sessionStorage 读取，因为它能跨页面保存）
			result, err := r.page.Eval(`() => {
				try {
					// 先尝试从 sessionStorage 获取（跨页面持久化）
					var saved = sessionStorage.getItem('__browserwing_actions__');
					if (saved) {
						return JSON.parse(saved);
					}
				} catch (e) {
					console.error('[BrowserWing] sessionStorage read error:', e);
				}
				// 回退到内存中的数据
				return window.__recordedActions__ || [];
			}`)
			if err == nil && result != nil {
				var actions []models.ScriptAction
				jsonData, _ := json.Marshal(result.Value)
				if json.Unmarshal(jsonData, &actions) == nil {
					// 只保存新增的操作
					if len(actions) > r.lastSyncedCount {
						newActions := actions[r.lastSyncedCount:]
						r.actions = append(r.actions, newActions...)
						if len(newActions) > 0 {
							logger.Info(ctx, "Synced %d new actions, total %d actions", len(newActions), len(r.actions))
						}
						r.lastSyncedCount = len(actions)
					}
				}
			}
			r.mu.Unlock()

		case <-r.syncStopChan:
			return
		}
	}
}

// checkAndProcessAIRequest 检查并处理 AI 提取请求
func (r *Recorder) checkAndProcessAIRequest(ctx context.Context) {
	if r.llmManager == nil {
		return
	}

	// 检查是否有待处理的 AI 请求
	result, err := r.page.Eval(`() => {
		if (window.__aiExtractionRequest__) {
			var req = window.__aiExtractionRequest__;
			delete window.__aiExtractionRequest__; // 立即清除请求，避免重复处理
			return req;
		}
		return null;
	}`)

	if err != nil || result == nil {
		return
	}

	// 检查返回值是否为 null
	if result.Value.Nil() {
		return
	}

	// 解析请求
	var requestData map[string]interface{}
	jsonData, _ := json.Marshal(result.Value)
	if err := json.Unmarshal(jsonData, &requestData); err != nil {
		logger.Warn(ctx, "Failed to parse AI request: %v", err)
		return
	}

	html, _ := requestData["html"].(string)
	description, _ := requestData["description"].(string)
	requestType, _ := requestData["type"].(string) // "extract" 或 "formfill"

	if html == "" {
		logger.Warn(ctx, "AI request missing HTML content")
		return
	}

	// 处理表单填充请求
	if requestType == "formfill" {
		logger.Info(ctx, "Received AI form fill request, HTML length: %d", len(html))
		r.handleFormFillRequest(ctx, html, description)
		return
	}

	// 处理数据提取请求（默认）
	logger.Info(ctx, "Received AI extraction request, HTML length: %d", len(html))

	// 获取默认 LLM 提取器
	extractor, err := r.llmManager.GetDefault()
	if err != nil {
		logger.Error(ctx, "Failed to get default LLM: %v", err)
		_, _ = r.page.Eval(fmt.Sprintf(`() => {
			window.__aiExtractionResponse__ = {
				success: false,
				error: %q
			};
		}`, err.Error()))
		return
	}

	// 调用 LLM 生成代码
	extractResult, err := extractor.GenerateExtractionJS(ctx, llm.ExtractionRequest{
		HTML:        html,
		Description: description,
	})
	if err != nil {
		logger.Error(ctx, "AI code generation failed: %v", err)
		// 将错误返回给页面
		_, _ = r.page.Eval(fmt.Sprintf(`() => {
			window.__aiExtractionResponse__ = {
				success: false,
				error: %q
			};
		}`, err.Error()))
		return
	}

	logger.Info(ctx, "✓ AI code generation successful, length: %d", len(extractResult.JavaScript))

	// 将结果返回给页面
	jsCode := extractResult.JavaScript
	// 转义 JavaScript 代码中的特殊字符
	jsCode = escapeJSString(jsCode)

	_, _ = r.page.Eval(fmt.Sprintf(`() => {
		window.__aiExtractionResponse__ = {
			success: true,
			javascript: %q,
			used_model: %q
		};
	}`, jsCode, extractResult.UsedModel))

	logger.Info(ctx, "✓ AI response set to page")
}

// escapeJSString 转义 JavaScript 字符串中的特殊字符
func escapeJSString(s string) string {
	// Go 的 %q 格式化会自动转义特殊字符
	return s
}

// handleFormFillRequest 处理表单填充请求
func (r *Recorder) handleFormFillRequest(ctx context.Context, html, description string) {
	// 获取默认 LLM 提取器
	extractor, err := r.llmManager.GetDefault()
	if err != nil {
		logger.Error(ctx, "Failed to get default LLM: %v", err)
		_, _ = r.page.Eval(fmt.Sprintf(`() => {
			window.__aiFormFillResponse__ = {
				success: false,
				error: %q
			};
		}`, err.Error()))
		return
	}

	// 调用 LLM 生成表单填充代码
	fillResult, err := extractor.GenerateFormFillJS(ctx, llm.FormFillRequest{
		HTML:        html,
		Description: description,
	})
	if err != nil {
		logger.Error(ctx, "AI form fill code generation failed: %v", err)
		_, _ = r.page.Eval(fmt.Sprintf(`() => {
			window.__aiFormFillResponse__ = {
				success: false,
				error: %q
			};
		}`, err.Error()))
		return
	}

	logger.Info(ctx, "✓ AI form fill code generation successful, length: %d", len(fillResult.JavaScript))

	// 将结果返回给页面
	jsCode := fillResult.JavaScript
	jsCode = escapeJSString(jsCode)

	_, _ = r.page.Eval(fmt.Sprintf(`() => {
		window.__aiFormFillResponse__ = {
			success: true,
			javascript: %q,
			used_model: %q
		};
	}`, jsCode, fillResult.UsedModel))

	logger.Info(ctx, "✓ AI form fill response set to page")
}

// StopRecording 停止录制并返回操作列表
func (r *Recorder) StopRecording(ctx context.Context) ([]models.ScriptAction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRecording {
		return nil, fmt.Errorf("recording is not in progress")
	}

	// 停止同步协程
	if r.syncTicker != nil {
		r.syncTicker.Stop()
		close(r.syncStopChan)
	}

	// 最后一次同步：从页面获取录制的操作
	if r.page != nil {
		logger.Info(ctx, "Performing final sync...")

		// 先检查录制器是否还存在
		checkResult, _ := r.page.Eval(`() => {
			var savedCount = 0;
			try {
				var saved = sessionStorage.getItem('__browserwing_actions__');
				if (saved) {
					savedCount = JSON.parse(saved).length;
				}
			} catch (e) {}
			
			return {
				recorderExists: !!window.__browserwingRecorder__,
				actionsCount: window.__recordedActions__ ? window.__recordedActions__.length : -1,
				sessionStorageCount: savedCount,
				actionsType: typeof window.__recordedActions__
			}
		}`)
		if checkResult != nil {
			logger.Info(ctx, "Recorder status check: %+v", checkResult.Value)
		}

		result, err := r.page.Eval(`() => {
			try {
				// 优先从 sessionStorage 获取完整数据
				var saved = sessionStorage.getItem('__browserwing_actions__');
				if (saved) {
					return JSON.parse(saved);
				}
			} catch (e) {
				console.error('[BrowserWing] sessionStorage read error:', e);
			}
			return window.__recordedActions__ || [];
		}`)
		if err != nil {
			logger.Warn(ctx, "Failed to get recording actions: %v", err)
		} else {
			logger.Info(ctx, "Result type received: %T", result.Value)
			// 解析 JSON 数据
			var actions []models.ScriptAction
			jsonData, err := json.Marshal(result.Value)
			if err == nil {
				logger.Info(ctx, "JSON serialization successful, data length: %d", len(jsonData))
				if err := json.Unmarshal(jsonData, &actions); err == nil {
					// 合并最后的操作（可能有新的）
					if len(actions) > r.lastSyncedCount {
						newActions := actions[r.lastSyncedCount:]
						r.actions = append(r.actions, newActions...)
						logger.Info(ctx, "Final sync of %d new actions", len(newActions))
					}
					logger.Info(ctx, "Recording completed, total %d actions", len(r.actions))
				} else {
					logger.Error(ctx, "JSON deserialization failed: %v", err)
				}
			} else {
				logger.Error(ctx, "JSON serialization failed: %v", err)
			}
		}

		// 清理注入的脚本、UI面板和 sessionStorage
		_, _ = r.page.Eval(`() => { 
			// 移除录制器 UI 面板
			if (window.__recorderUI__ && window.__recorderUI__.panel) {
				try {
					window.__recorderUI__.panel.remove();
				} catch(e) {
					console.error('[BrowserWing] Remove panel error:', e);
				}
			}
			// 移除高亮元素
			if (window.__highlightElement__) {
				try {
					window.__highlightElement__.remove();
				} catch(e) {}
			}
			// 清理全局变量
			delete window.__browserwingRecorder__; 
			delete window.__recordedActions__; 
			delete window.__recorderUI__;
			delete window.__highlightElement__;
			delete window.__selectedElement__;
			delete window.__extractMode__;
			delete window.__aiExtractMode__;
			delete window.__lastInputTime__;
			delete window.__inputTimers__;
			// 清理 sessionStorage
			try { sessionStorage.removeItem('__browserwing_actions__'); } catch(e) {}
		}`)

		// 恢复 CSP 限制
		_ = proto.PageSetBypassCSP{Enabled: false}.Call(r.page)
		logger.Info(ctx, "✓ CSP restrictions restored")
	} else {
		logger.Warn(ctx, "Page object is nil")
	}

	r.isRecording = false
	actions := r.actions
	r.page = nil

	logger.Info(ctx, "Final return of %d actions", len(actions))

	return actions, nil
}

// injectIframeRecorders 为页面中所有 iframe 注入录制脚本
func (r *Recorder) injectIframeRecorders(ctx context.Context, page *rod.Page) {
	// 使用 rod 的 Elements 方法获取所有 iframe
	iframes, err := page.Elements("iframe")
	if err != nil {
		logger.Warn(ctx, "Failed to detect iframes: %v", err)
		return
	}

	if len(iframes) == 0 {
		logger.Info(ctx, "No iframes in page")
		return
	}

	logger.Info(ctx, "Detected %d iframes, preparing to inject recording script", len(iframes))

	// 为每个 iframe 注入脚本
	for i, iframeElement := range iframes {
		// 获取 iframe 的页面上下文
		frame, err := iframeElement.Frame()
		if err != nil {
			logger.Warn(ctx, "Failed to get Frame for iframe #%d: %v", i, err)
			continue
		}

		// 等待 iframe 加载
		if err := frame.WaitLoad(); err != nil {
			logger.Warn(ctx, "Failed to wait for iframe #%d to load: %v", i, err)
		}

		// 在 iframe 的页面上下文中注入录制脚本
		_, err = frame.Eval(`() => { ` + iframeRecorderScript + ` return true; }`)
		if err != nil {
			logger.Warn(ctx, "Failed to inject script into iframe #%d: %v", i, err)
		} else {
			logger.Info(ctx, "✓ Recording script injected into iframe #%d successfully", i)
		}
	}
}

// watchForNewIframes 监听新创建的 iframe 并自动注入录制脚本
func (r *Recorder) watchForNewIframes(ctx context.Context, page *rod.Page) {
	// 记录已经处理过的 iframe 数量
	processedIframeCount := 0

	// 使用定时轮询检测新的 iframe（每秒检查一次）
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查录制是否还在进行
			if !r.IsRecording() {
				return
			}

			// 获取当前所有 iframe
			iframes, err := page.Elements("iframe")
			if err != nil {
				continue
			}

			// 如果有新的 iframe
			if len(iframes) > processedIframeCount {
				logger.Info(ctx, "Detected %d new iframes", len(iframes)-processedIframeCount)

				// 为新的 iframe 注入脚本
				for i := processedIframeCount; i < len(iframes); i++ {
					iframeElement := iframes[i]

					// 获取 iframe 的页面上下文
					frame, err := iframeElement.Frame()
					if err != nil {
						logger.Warn(ctx, "Failed to get Frame for new iframe #%d: %v", i, err)
						continue
					}

					// 等待 iframe 加载
					if err := frame.WaitLoad(); err != nil {
						logger.Warn(ctx, "Failed to wait for new iframe #%d to load: %v", i, err)
					}

					// 在 iframe 的页面上下文中注入录制脚本
					_, err = frame.Eval(`() => { ` + iframeRecorderScript + ` return true; }`)
					if err != nil {
						logger.Warn(ctx, "Failed to inject script into new iframe #%d: %v", i, err)
					} else {
						logger.Info(ctx, "✓ Recording script injected into new iframe #%d successfully", i)
					}
				}

				processedIframeCount = len(iframes)
			}

		case <-ctx.Done():
			return
		}
	}
}

// watchForPageNavigation 监听页面导航事件，在新页面自动重新注入录制脚本
func (r *Recorder) watchForPageNavigation(ctx context.Context, page *rod.Page) {
	// 记录上一次的 URL
	var lastURL string
	result, err := page.Eval(`() => window.location.href`)
	if err == nil && result != nil && result.Value.Str() != "" {
		lastURL = result.Value.Str()
	}

	logger.Info(ctx, "Started watching for page navigation, initial URL: %s", lastURL)

	// 使用定时轮询检测 URL 变化（每 300ms 检查一次）
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查录制是否还在进行
			if !r.IsRecording() {
				return
			}

			// 获取当前 URL
			result, err := page.Eval(`() => window.location.href`)
			if err != nil {
				continue
			}

			if result == nil || result.Value.Str() == "" {
				continue
			}

			currentURL := result.Value.Str()

			// 检测到 URL 变化（页面导航/跳转）
			if currentURL != lastURL {
				logger.Info(ctx, "Page navigation detected: %s -> %s", lastURL, currentURL)
				lastURL = currentURL

				// 等待新页面加载稳定
				time.Sleep(800 * time.Millisecond)

				// 检查录制模式标志是否存在
				checkResult, _ := page.Eval(`() => window.__browserwingRecordingMode__`)
				needsReinjection := false

				if checkResult == nil || !checkResult.Value.Bool() {
					logger.Info(ctx, "Recording mode flag missing after navigation, will reinject")
					needsReinjection = true
				}

				// 检查录制器是否存在
				recorderCheck, _ := page.Eval(`() => window.__browserwingRecorder__`)
				if recorderCheck == nil || !recorderCheck.Value.Bool() {
					logger.Info(ctx, "Recorder script missing after navigation, will reinject")
					needsReinjection = true
				}

				// 如果需要重新注入
				if needsReinjection {
					// 禁用 CSP
					err := proto.PageSetBypassCSP{Enabled: true}.Call(page)
					if err != nil {
						logger.Warn(ctx, "Failed to disable CSP after navigation: %v", err)
					}

					// 重新设置录制模式标志
					_, err = page.Eval(`() => { window.__browserwingRecordingMode__ = true; }`)
					if err != nil {
						logger.Warn(ctx, "Failed to set recording mode flag after navigation: %v", err)
					}

					// 重新注入录制脚本
					_, err = page.Eval(`() => { ` + recorderScript + ` return true; }`)
					if err != nil {
						logger.Error(ctx, "Failed to reinject recording script after navigation: %v", err)
					} else {
						logger.Info(ctx, "✓ Recording script reinjected successfully after navigation")
					}

					// 重新注入 iframe 消息监听器
					_, err = page.Eval(`() => { ` + iframeMessageListenerScript + ` return true; }`)
					if err != nil {
						logger.Warn(ctx, "Failed to reinject iframe message listener: %v", err)
					}

					// 为新页面的 iframe 注入录制脚本
					r.injectIframeRecorders(ctx, page)
				} else {
					logger.Info(ctx, "Recording script still active after navigation, no reinjection needed")
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

// IsRecording 检查是否正在录制
func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRecording
}

// GetRecordingInfo 获取录制信息
func (r *Recorder) GetRecordingInfo() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := map[string]interface{}{
		"is_recording": r.isRecording,
	}

	if r.isRecording {
		info["start_url"] = r.startURL
		info["start_time"] = r.startTime.Format(time.RFC3339)
		info["duration"] = time.Since(r.startTime).Seconds()
	}

	return info
}

// GetStartURL 获取录制的起始URL
func (r *Recorder) GetStartURL() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.startURL
}
