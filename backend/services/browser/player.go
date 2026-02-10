package browser

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

//go:embed scripts/indicator.js
var indicatorScript string

// Player 脚本回放器
// BrowserManagerInterface 定义 Browser 管理器需要的接口
// 避免循环依赖
type BrowserManagerInterface interface {
	SetActivePage(page *rod.Page)
	GetActivePage() *rod.Page
}

type Player struct {
	extractedData     map[string]interface{}          // 存储抓取的数据
	successCount      int                             // 成功步骤数
	failCount         int                             // 失败步骤数
	recordingPage     *rod.Page                       // 录制的页面
	recordingOutputs  chan *proto.PageScreencastFrame // 录制帧通道
	recordingDone     chan bool                       // 录制完成信号
	pages             map[int]*rod.Page               // 多标签页支持 (key: tab index)
	currentPage       *rod.Page                       // 当前活动页面
	tabCounter        int                             // 标签页计数器
	downloadedFiles   []string                        // 下载的文件路径列表
	downloadPath      string                          // 下载目录路径
	downloadCtx       context.Context                 // 下载监听上下文
	downloadCancel    context.CancelFunc              // 取消下载监听
	currentScriptName string                          // 当前执行的脚本名称
	currentLang       string                          // 当前语言设置
	currentActions    []models.ScriptAction           // 当前执行的脚本动作列表
	currentStepIndex  int                             // 当前执行到的步骤索引
	agentManager      AgentManagerInterface           // Agent 管理器（用于 AI 控制功能）
	browserManager    BrowserManagerInterface         // Browser 管理器（用于同步活跃页面）
}

// highlightElement 高亮显示元素
func (p *Player) highlightElement(ctx context.Context, element *rod.Element) {
	if element == nil {
		return
	}

	// 添加高亮边框样式
	_, err := element.Eval(`() => {
		this.setAttribute('data-browserwing-original-style', this.style.cssText || '');
		this.style.outline = '3px solid #3b82f6';
		this.style.outlineOffset = '2px';
		this.style.boxShadow = '0 0 0 4px rgba(59, 130, 246, 0.3)';
	}`)
	if err != nil {
		logger.Warn(ctx, "Failed to highlight element: %v", err)
	}
}

// unhighlightElement 取消元素高亮
func (p *Player) unhighlightElement(ctx context.Context, element *rod.Element) {
	if element == nil {
		return
	}

	// 移除高亮样式，恢复原始样式
	_, err := element.Eval(`() => {
		const originalStyle = this.getAttribute('data-browserwing-original-style');
		if (originalStyle !== null) {
			this.style.cssText = originalStyle;
			this.removeAttribute('data-browserwing-original-style');
		} else {
			this.style.outline = '';
			this.style.outlineOffset = '';
			this.style.boxShadow = '';
		}
	}`)
	if err != nil {
		logger.Warn(ctx, "Failed to unhighlight element: %v", err)
	}
}

// showAIControlIndicator 显示 AI 控制指示器
func (p *Player) showAIControlIndicator(ctx context.Context, page *rod.Page, scriptName, currentLang string) {
	if page == nil {
		return
	}

	// 获取国际化文本
	titleText := getI18nText("ai.control.title", currentLang)
	scriptLabelText := getI18nText("ai.control.script", currentLang)
	readyText := getI18nText("ai.control.ready", currentLang)

	_, err := page.Eval(indicatorScript, scriptName, titleText, scriptLabelText, readyText)

	if err != nil {
		logger.Warn(ctx, "Failed to show AI control indicator: %v", err)
		logger.Warn(ctx, "Error details: %v", err)
	} else {
		logger.Info(ctx, "✓ AI control indicator displayed")
	}
}

// ensureAIControlIndicator 确保 AI 控制指示器存在（在页面导航后重新注入）
func (p *Player) ensureAIControlIndicator(ctx context.Context, page *rod.Page) {
	if page == nil {
		return
	}

	// 检查指示器是否存在
	exists, err := page.Eval(`() => {
		// 清理旧的闪烁定时器（如果有）
		if (window.__browserwingBlinkInterval__) {
			clearInterval(window.__browserwingBlinkInterval__);
			window.__browserwingBlinkInterval__ = null;
		}
		return document.getElementById('browserwing-ai-indicator') !== null;
	}`)
	if err != nil {
		logger.Warn(ctx, "Failed to check AI control indicator: %v", err)
		return
	}

	// 如果不存在，重新注入
	if exists != nil && !exists.Value.Bool() {
		logger.Info(ctx, "AI control indicator lost after navigation, re-injecting...")
		currentLang := p.currentLang
		if currentLang == "" {
			currentLang = "zh-CN"
		}
		p.showAIControlIndicator(ctx, page, p.currentScriptName, currentLang)

		// 如果有当前执行的脚本动作，重新初始化步骤列表
		if len(p.currentActions) > 0 {
			p.initAIControlSteps(ctx, page, p.currentActions)

			// 恢复之前步骤的状态
			for i := 0; i < p.currentStepIndex && i < len(p.currentActions); i++ {
				// 标记已完成的步骤为成功（简化处理）
				p.markStepCompleted(ctx, page, i+1, true)
			}

			// 如果当前正在执行某个步骤，也更新其状态
			if p.currentStepIndex > 0 && p.currentStepIndex <= len(p.currentActions) {
				p.updateAIControlStatus(ctx, page, p.currentStepIndex, len(p.currentActions), p.currentActions[p.currentStepIndex-1].Type)
			}
		}
	}
}

// initAIControlSteps 初始化步骤列表
func (p *Player) initAIControlSteps(ctx context.Context, page *rod.Page, actions []models.ScriptAction) {
	if page == nil {
		return
	}

	// 获取国际化文本
	stepText := getI18nText("ai.control.step", p.currentLang)

	// 构建步骤数据
	type stepData struct {
		Index  int    `json:"index"`
		Action string `json:"action"`
	}

	steps := make([]stepData, len(actions))
	for i, action := range actions {
		steps[i] = stepData{
			Index:  i + 1,
			Action: getActionDisplayText(action.Type, p.currentLang),
		}
	}

	_, err := page.Eval(`(stepText, stepsData) => {
		const container = document.getElementById('browserwing-ai-steps-container');
		if (!container) return false;
		
		// 清空容器
		container.innerHTML = '';
		
		// 添加所有步骤
		stepsData.forEach((step) => {
			const stepItem = document.createElement('div');
			stepItem.id = 'browserwing-ai-step-' + step.index;
			stepItem.className = '__browserwing-protected__';
			stepItem.style.cssText = 'padding: 12px 16px !important; border-bottom: 1px solid #e2e8f0 !important; display: flex !important; align-items: center !important; gap: 12px !important; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "SF Pro Display", Helvetica, Arial, sans-serif !important; background: white !important; transition: all 0.3s ease !important;';
			
			// 步骤序号
			const stepNum = document.createElement('div');
			stepNum.className = '__browserwing-protected__ step-number';
			stepNum.style.cssText = 'width: 28px !important; height: 28px !important; min-width: 28px !important; border-radius: 6px !important; background: #e2e8f0 !important; color: #94a3b8 !important; display: flex !important; align-items: center !important; justify-content: center !important; font-size: 12px !important; font-weight: 600 !important; transition: all 0.3s ease !important;';
			stepNum.textContent = step.index;
			
			// 步骤内容
			const stepContent = document.createElement('div');
			stepContent.className = '__browserwing-protected__ step-content';
			stepContent.style.cssText = 'flex: 1 !important; color: #64748b !important; font-size: 13px !important; font-weight: 500 !important; transition: all 0.3s ease !important;';
			stepContent.textContent = step.action;
			
			// 状态图标（初始为等待状态 - 时钟图标）
			const statusIcon = document.createElement('div');
			statusIcon.className = '__browserwing-protected__ browserwing-step-status';
			statusIcon.style.cssText = 'width: 24px !important; height: 24px !important; min-width: 24px !important; border-radius: 50% !important; background: #e2e8f0 !important; color: #94a3b8 !important; display: flex !important; align-items: center !important; justify-content: center !important; transition: all 0.3s ease !important;';
			statusIcon.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" style="width: 14px !important; height: 14px !important;"><path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10 10-4.5 10-10S17.5 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z"/></svg>';
			
			stepItem.appendChild(stepNum);
			stepItem.appendChild(stepContent);
			stepItem.appendChild(statusIcon);
			container.appendChild(stepItem);
		});
		
		return true;
	}`, stepText, steps)
	if err != nil {
		logger.Warn(ctx, "Failed to initialize AI control steps: %v", err)
	}
}

// updateAIControlStatus 更新步骤状态
func (p *Player) updateAIControlStatus(ctx context.Context, page *rod.Page, current, total int, actionType string) {
	if page == nil {
		return
	}

	_, err := page.Eval(`(stepIndex, status) => {
		const stepItem = document.getElementById('browserwing-ai-step-' + stepIndex);
		if (!stepItem) return false;
		
		const statusIcon = stepItem.querySelector('.browserwing-step-status');
		const stepNum = stepItem.querySelector('.step-number');
		const stepContent = stepItem.querySelector('.step-content');
		if (!statusIcon) return false;
		
		// 更新为执行中状态 - 鲜艳的蓝色高亮
		stepItem.setAttribute('style', 'padding: 12px 16px !important; border-bottom: 1px solid #e2e8f0 !important; display: flex !important; align-items: center !important; gap: 12px !important; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "SF Pro Display", Helvetica, Arial, sans-serif !important; background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%) !important; border-left: 4px solid #3b82f6 !important; box-shadow: 0 2px 8px rgba(59, 130, 246, 0.15) !important; transition: all 0.3s ease !important;');
		
		if (stepNum) {
			stepNum.setAttribute('style', 'width: 28px !important; height: 28px !important; min-width: 28px !important; border-radius: 6px !important; background: #3b82f6 !important; color: white !important; display: flex !important; align-items: center !important; justify-content: center !important; font-size: 12px !important; font-weight: 600 !important; transition: all 0.3s ease !important;');
		}
		
		if (stepContent) {
			stepContent.setAttribute('style', 'flex: 1 !important; color: #1e40af !important; font-size: 13px !important; font-weight: 600 !important; transition: all 0.3s ease !important;');
		}
		
		// 执行中的 SVG 图标（播放图标）
		statusIcon.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" style="width: 14px !important; height: 14px !important;"><path d="M8 5v14l11-7z"/></svg>';
		statusIcon.setAttribute('style', 'width: 24px !important; height: 24px !important; min-width: 24px !important; border-radius: 50% !important; background: #3b82f6 !important; color: white !important; display: flex !important; align-items: center !important; justify-content: center !important; animation: browserwing-ai-blink 1.5s ease-in-out infinite !important; transition: all 0.3s ease !important;');
		
		// 滚动到当前步骤
		stepItem.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
		
		return true;
	}`, current)
	if err != nil {
		logger.Warn(ctx, "Failed to update AI control status: %v", err)
	}
}

// markStepCompleted 标记步骤为已完成
func (p *Player) markStepCompleted(ctx context.Context, page *rod.Page, stepIndex int, success bool) {
	if page == nil {
		return
	}

	_, err := page.Eval(`(stepIndex, success) => {
		const stepItem = document.getElementById('browserwing-ai-step-' + stepIndex);
		if (!stepItem) return false;
		
		const statusIcon = stepItem.querySelector('.browserwing-step-status');
		const stepNum = stepItem.querySelector('.step-number');
		const stepContent = stepItem.querySelector('.step-content');
		if (!statusIcon) return false;
		
		if (success) {
			// 成功 - 鲜艳的绿色高亮
			stepItem.setAttribute('style', 'padding: 12px 16px !important; border-bottom: 1px solid #e2e8f0 !important; display: flex !important; align-items: center !important; gap: 12px !important; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "SF Pro Display", Helvetica, Arial, sans-serif !important; background: linear-gradient(135deg, #dcfce7 0%, #bbf7d0 100%) !important; border-left: 4px solid #10b981 !important; box-shadow: 0 2px 8px rgba(16, 185, 129, 0.15) !important; transition: all 0.3s ease !important;');
			
			if (stepNum) {
				stepNum.setAttribute('style', 'width: 28px !important; height: 28px !important; min-width: 28px !important; border-radius: 6px !important; background: #10b981 !important; color: white !important; display: flex !important; align-items: center !important; justify-content: center !important; font-size: 12px !important; font-weight: 600 !important; transition: all 0.3s ease !important;');
			}
			
			if (stepContent) {
				stepContent.setAttribute('style', 'flex: 1 !important; color: #15803d !important; font-size: 13px !important; font-weight: 500 !important; transition: all 0.3s ease !important;');
			}
			
			// 成功的 SVG 图标（对勾）
			statusIcon.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" style="width: 16px !important; height: 16px !important;"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>';
			statusIcon.setAttribute('style', 'width: 24px !important; height: 24px !important; min-width: 24px !important; border-radius: 50% !important; background: #10b981 !important; color: white !important; display: flex !important; align-items: center !important; justify-content: center !important; animation: none !important; transition: all 0.3s ease !important;');
		} else {
			// 失败 - 鲜艳的红色高亮
			stepItem.setAttribute('style', 'padding: 12px 16px !important; border-bottom: 1px solid #e2e8f0 !important; display: flex !important; align-items: center !important; gap: 12px !important; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "SF Pro Display", Helvetica, Arial, sans-serif !important; background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%) !important; border-left: 4px solid #ef4444 !important; box-shadow: 0 2px 8px rgba(239, 68, 68, 0.15) !important; transition: all 0.3s ease !important;');
			
			if (stepNum) {
				stepNum.setAttribute('style', 'width: 28px !important; height: 28px !important; min-width: 28px !important; border-radius: 6px !important; background: #ef4444 !important; color: white !important; display: flex !important; align-items: center !important; justify-content: center !important; font-size: 12px !important; font-weight: 600 !important; transition: all 0.3s ease !important;');
			}
			
			if (stepContent) {
				stepContent.setAttribute('style', 'flex: 1 !important; color: #b91c1c !important; font-size: 13px !important; font-weight: 500 !important; transition: all 0.3s ease !important;');
			}
			
			// 失败的 SVG 图标（叉号）
			statusIcon.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" style="width: 16px !important; height: 16px !important;"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>';
			statusIcon.setAttribute('style', 'width: 24px !important; height: 24px !important; min-width: 24px !important; border-radius: 50% !important; background: #ef4444 !important; color: white !important; display: flex !important; align-items: center !important; justify-content: center !important; animation: none !important; transition: all 0.3s ease !important;');
		}
		
		return true;
	}`, stepIndex, success)
	if err != nil {
		logger.Warn(ctx, "Failed to mark step completion: %v", err)
	}
}

// getI18nText 获取国际化文本
func getI18nText(key, lang string) string {
	// 翻译映射表
	translations := map[string]map[string]string{
		"zh-CN": {
			// AI 控制指示器
			"ai.control.title":     "Browserwing AI 控制中",
			"ai.control.script":    "执行脚本:",
			"ai.control.ready":     "准备执行脚本...",
			"ai.control.step":      "步骤",
			"ai.control.completed": "✓ 完成",
			"ai.control.success":   "成功",
			"ai.control.failed":    "失败",
			// 操作类型
			"action.click":             "点击元素",
			"action.input":             "输入文本",
			"action.select":            "选择选项",
			"action.navigate":          "页面导航",
			"action.wait":              "等待加载",
			"action.sleep":             "延迟等待",
			"action.extract_text":      "提取文本",
			"action.extract_html":      "提取HTML",
			"action.extract_attribute": "提取属性",
			"action.execute_js":        "执行JS",
			"action.upload_file":       "上传文件",
			"action.scroll":            "滚动页面",
			"action.keyboard":          "键盘事件",
			"action.screenshot":        "截图",
			"action.open_tab":          "打开新标签页",
			"action.switch_tab":        "切换标签页",
			"action.switch_active_tab": "切换到活跃标签页",
			"action.capture_xhr":       "捕获XHR请求",
			"action.ai_control":        "AI控制",
		},
		"zh-TW": {
			// AI 控制指示器
			"ai.control.title":     "Browserwing AI 控制中",
			"ai.control.script":    "執行腳本:",
			"ai.control.ready":     "準備執行腳本...",
			"ai.control.step":      "步驟",
			"ai.control.completed": "✓ 完成",
			"ai.control.success":   "成功",
			"ai.control.failed":    "失敗",
			// 操作類型
			"action.click":             "點擊元素",
			"action.input":             "輸入文字",
			"action.select":            "選擇選項",
			"action.navigate":          "頁面導航",
			"action.wait":              "等待載入",
			"action.sleep":             "延遲等待",
			"action.extract_text":      "提取文字",
			"action.extract_html":      "提取HTML",
			"action.extract_attribute": "提取屬性",
			"action.execute_js":        "執行JS",
			"action.upload_file":       "上傳檔案",
			"action.scroll":            "滾動頁面",
			"action.keyboard":          "鍵盤事件",
			"action.screenshot":        "截圖",
			"action.open_tab":          "打開新標籤頁",
			"action.switch_tab":        "切換標籤頁",
			"action.switch_active_tab": "切換到活躍標籤頁",
			"action.capture_xhr":       "捕獲XHR請求",
			"action.ai_control":        "AI控制",
		},
		"en": {
			// AI Control Indicator
			"ai.control.title":     "Browserwing AI Control",
			"ai.control.script":    "Executing Script:",
			"ai.control.ready":     "Preparing to execute script...",
			"ai.control.step":      "Step",
			"ai.control.completed": "✓ Completed",
			"ai.control.success":   "Success",
			"ai.control.failed":    "Failed",
			// Action Types
			"action.click":             "Click Element",
			"action.input":             "Input Text",
			"action.select":            "Select Option",
			"action.navigate":          "Navigate Page",
			"action.wait":              "Wait for Load",
			"action.sleep":             "Sleep",
			"action.extract_text":      "Extract Text",
			"action.extract_html":      "Extract HTML",
			"action.extract_attribute": "Extract Attribute",
			"action.execute_js":        "Execute JS",
			"action.upload_file":       "Upload File",
			"action.scroll":            "Scroll Page",
			"action.keyboard":          "Keyboard Event",
			"action.screenshot":        "Screenshot",
			"action.open_tab":          "Open New Tab",
			"action.switch_tab":        "Switch Tab",
			"action.switch_active_tab": "Switch to Active Tab",
			"action.capture_xhr":       "Capture XHR Request",
			"action.ai_control":        "AI Control",
		},
	}

	// 如果语言不存在，默认使用英文
	langMap, exists := translations[lang]
	if !exists {
		langMap = translations["en"]
	}

	// 返回翻译文本，如果不存在则返回 key
	if text, exists := langMap[key]; exists {
		return text
	}
	return key
}

// getActionDisplayText 获取操作的显示文本（支持国际化）
func getActionDisplayText(actionType, lang string) string {
	return getI18nText("action."+actionType, lang)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// elementContext 包含元素及其所在的页面上下文
type elementContext struct {
	element *rod.Element
	page    *rod.Page // 元素所在的 page（如果在 iframe 中，这是 frame）
}

// NewPlayer 创建回放器
func NewPlayer(currentLang string) *Player {
	return &Player{
		extractedData:   make(map[string]interface{}),
		successCount:    0,
		failCount:       0,
		pages:           make(map[int]*rod.Page),
		tabCounter:      0,
		downloadedFiles: make([]string, 0),
		currentLang:     currentLang,
	}
}

// SetDownloadPath 设置下载路径
func (p *Player) SetDownloadPath(downloadPath string) {
	p.downloadPath = downloadPath
}

// StartDownloadListener 启动下载事件监听
func (p *Player) StartDownloadListener(ctx context.Context, browser *rod.Browser) {
	if p.downloadPath == "" {
		logger.Warn(ctx, "Download path not set, skipping download listener")
		return
	}

	// 创建可取消的上下文
	p.downloadCtx, p.downloadCancel = context.WithCancel(ctx)

	logger.Info(ctx, "Starting download event listener for path: %s", p.downloadPath)

	// 记录每个下载的 GUID 到文件名的映射
	downloadMap := make(map[string]string)

	// 监听下载开始事件 (BrowserDownloadWillBegin)
	go browser.Context(p.downloadCtx).EachEvent(func(e *proto.BrowserDownloadWillBegin) {
		// 记录 GUID 和建议的文件名
		downloadMap[e.GUID] = e.SuggestedFilename
		logger.Info(ctx, "📥 Download will begin: %s (GUID: %s)", e.SuggestedFilename, e.GUID)
	})()

	// 监听下载进度事件 (BrowserDownloadProgress)
	go browser.Context(p.downloadCtx).EachEvent(func(e *proto.BrowserDownloadProgress) {
		if e.State == proto.BrowserDownloadProgressStateCompleted {
			// 下载完成，从映射中获取文件名
			fileName, exists := downloadMap[e.GUID]
			if !exists {
				logger.Warn(ctx, "Download completed but filename not found (GUID: %s)", e.GUID)
				return
			}
			// 如果是截图，则也不进行在这里监听返回，包含 browserwing_screenshot_ 前缀
			if strings.Contains(fileName, "browserwing_screenshot_") {
				return
			}

			// 构建完整路径
			fullPath := filepath.Join(p.downloadPath, fileName)

			// 检查文件是否实际存在（可能浏览器自动重命名了）
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				// 文件不存在，可能被重命名了（如 file.pdf -> file (1).pdf）
				// 尝试查找类似的文件
				if actualFile := p.findSimilarFile(fileName); actualFile != "" {
					fullPath = filepath.Join(p.downloadPath, actualFile)
					fileName = actualFile
					logger.Info(ctx, "File was renamed by browser: %s -> %s", downloadMap[e.GUID], actualFile)
				}
			}

			// 检查是否已经记录过这个文件
			alreadyRecorded := false
			for _, existing := range p.downloadedFiles {
				if existing == fullPath {
					alreadyRecorded = true
					break
				}
			}

			if !alreadyRecorded {
				p.downloadedFiles = append(p.downloadedFiles, fullPath)
				logger.Info(ctx, "✓ Download completed: %s (%.2f MB, GUID: %s)",
					fullPath, float64(e.TotalBytes)/(1024*1024), e.GUID)
			}

			// 清理映射
			delete(downloadMap, e.GUID)
		} else if e.State == proto.BrowserDownloadProgressStateCanceled {
			logger.Warn(ctx, "Download canceled (GUID: %s)", e.GUID)
			delete(downloadMap, e.GUID)
		}
	})()

	logger.Info(ctx, "Download event listener started")
}

// findSimilarFile 查找相似的文件名（处理浏览器自动重命名的情况）
func (p *Player) findSimilarFile(originalName string) string {
	entries, err := os.ReadDir(p.downloadPath)
	if err != nil {
		return ""
	}

	// 提取文件名和扩展名
	ext := filepath.Ext(originalName)
	nameWithoutExt := strings.TrimSuffix(originalName, ext)

	// 查找匹配的模式：file.pdf -> file (1).pdf, file (2).pdf, etc.
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 检查是否匹配 "原名 (数字).扩展名" 的模式
		if strings.HasPrefix(name, nameWithoutExt) && strings.HasSuffix(name, ext) {
			// 精确匹配或带数字后缀
			if name == originalName ||
				(len(name) > len(nameWithoutExt)+len(ext) &&
					name[len(nameWithoutExt)] == ' ' &&
					name[len(nameWithoutExt)+1] == '(') {
				return name
			}
		}
	}

	return ""
}

// StopDownloadListener 停止下载事件监听
func (p *Player) StopDownloadListener(ctx context.Context) {
	if p.downloadCancel != nil {
		p.downloadCancel()
		logger.Info(ctx, "Download event listener stopped")
	}

	// 记录最终下载的文件
	if len(p.downloadedFiles) > 0 {
		logger.Info(ctx, "✓ Total downloaded files: %d", len(p.downloadedFiles))
		for i, file := range p.downloadedFiles {
			logger.Info(ctx, "  #%d: %s", i+1, file)
		}
	} else {
		logger.Info(ctx, "No files downloaded during script execution")
	}
}

// GetDownloadedFiles 获取下载的文件列表
func (p *Player) GetDownloadedFiles() []string {
	return p.downloadedFiles
}

// GetExtractedData 获取抓取的数据
func (p *Player) GetExtractedData() map[string]interface{} {
	return p.extractedData
}

// GetSuccessCount 获取成功步骤数
func (p *Player) GetSuccessCount() int {
	return p.successCount
}

// GetFailCount 获取失败步骤数
func (p *Player) GetFailCount() int {
	return p.failCount
}

// ResetStats 重置统计信息
func (p *Player) ResetStats() {
	p.successCount = 0
	p.failCount = 0
	p.extractedData = make(map[string]interface{})
	// 注意：不清空录制相关字段，因为录制可能在 PlayScript 之前就已经启动
	// 录制字段只在 StopVideoRecording 中清空
}

// StartVideoRecording 开始视频录制（使用 Chrome DevTools Protocol）
func (p *Player) StartVideoRecording(page *rod.Page, outputPath string, frameRate, quality int) error {
	if page == nil {
		return fmt.Errorf("page is empty, cannot start recording")
	}

	p.recordingPage = page
	p.recordingOutputs = make(chan *proto.PageScreencastFrame, 100)
	p.recordingDone = make(chan bool)

	// 启动 screencast
	if frameRate <= 0 {
		frameRate = 15
	}
	if quality <= 0 || quality > 100 {
		quality = 70
	}

	ctx := page.GetContext()

	// 在启动 screencast 之前就开始监听事件，避免丢失帧
	// 这里立即捕获 page 变量，避免后续被修改
	capturedPage := page
	go p.saveScreencastFrames(ctx, capturedPage, outputPath)

	// 稍微等待一下，确保事件监听器已经启动
	time.Sleep(100 * time.Millisecond)

	// 启动屏幕录制
	format := proto.PageStartScreencastFormatJpeg
	err := proto.PageStartScreencast{
		Format:  format,
		Quality: &quality,
	}.Call(page)
	if err != nil {
		close(p.recordingDone) // 清理
		return fmt.Errorf("failed to start screencast: %w", err)
	}

	logger.Info(ctx, "Video recording started: frame rate=%d, quality=%d", frameRate, quality)
	return nil
}

// saveScreencastFrames 保存录制帧到文件（简化版 - 保存为图片序列）
func (p *Player) saveScreencastFrames(ctx context.Context, page *rod.Page, outputPath string) {
	if page == nil {
		logger.Warn(ctx, "Recording page is empty, cannot save frame")
		return
	}

	// 创建输出目录
	baseDir := strings.TrimSuffix(outputPath, ".gif") + "_frames"
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		logger.Warn(ctx, "Failed to create output directory: %v", err)
		return
	}

	logger.Info(ctx, "Start listening to recording frames, output directory: %s", baseDir)

	frameIndex := 0

	// 监听 screencast 帧事件
	// 注意：不要再嵌套 goroutine，这个函数本身就在 goroutine 中运行
	page.EachEvent(func(e *proto.PageScreencastFrame) {
		// 保存帧数据
		framePath := fmt.Sprintf("%s/frame_%05d.jpg", baseDir, frameIndex)
		data := []byte(e.Data)
		if err := os.WriteFile(framePath, data, 0o644); err != nil {
			logger.Warn(ctx, "Failed to save frame: %v", err)
		} else {
			if frameIndex%30 == 0 { // 每30帧打印一次日志
				logger.Info(ctx, "Saved %d frames", frameIndex)
			}
		}

		// 确认帧已处理
		_ = proto.PageScreencastFrameAck{
			SessionID: e.SessionID,
		}.Call(page)

		frameIndex++
	})()

	// 等待录制完成信号
	<-p.recordingDone
	logger.Info(ctx, "Recording completed, recorded %d frames, saved in: %s", frameIndex, baseDir)
}

// StopVideoRecording 停止视频录制
func (p *Player) StopVideoRecording(outputPath string, frameRate int) error {
	// 先保存 page 引用，避免在检查后被其他地方修改
	page := p.recordingPage
	done := p.recordingDone

	if page == nil && done == nil {
		return fmt.Errorf("no ongoing recording")
	}

	ctx := context.Background()
	if page != nil {
		ctx = page.GetContext()
	}
	logger.Info(ctx, "Stopping video recording...")

	// 先停止 screencast
	if page != nil {
		err := proto.PageStopScreencast{}.Call(page)
		if err != nil {
			logger.Warn(ctx, "Failed to stop screencast: %v", err)
		} else {
			logger.Info(ctx, "Screencast stopped")
		}
	}

	// 稍微等待一下，确保最后的帧被处理
	logger.Info(ctx, "Waiting for final frame processing to complete...")
	time.Sleep(500 * time.Millisecond)

	// 发送录制完成信号
	if done != nil {
		logger.Info(ctx, "Sending recording completion signal...")
		close(done)
	}

	// 清空录制状态
	p.recordingPage = nil
	p.recordingOutputs = nil
	p.recordingDone = nil

	// 将帧序列转换为 GIF
	if outputPath != "" {
		if err := p.convertFramesToGIF(ctx, outputPath, frameRate); err != nil {
			logger.Warn(ctx, "Failed to convert frames to GIF: %v", err)
			return err
		}
	}

	logger.Info(ctx, "Video recording stopped")
	return nil
}

// convertFramesToGIF 将帧序列转换为 GIF 动画
func (p *Player) convertFramesToGIF(ctx context.Context, outputPath string, frameRate int) error {
	baseDir := strings.TrimSuffix(outputPath, ".gif") + "_frames"

	// 检查帧目录是否存在
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return fmt.Errorf("frame directory does not exist: %s", baseDir)
	}

	if frameRate <= 0 {
		frameRate = 15
	}

	logger.Info(ctx, "Converting frame sequence to GIF...")
	logger.Info(ctx, "Input directory: %s", baseDir)
	logger.Info(ctx, "Output file: %s", outputPath)
	logger.Info(ctx, "Frame rate: %d", frameRate)

	// 读取所有帧文件
	files, err := filepath.Glob(filepath.Join(baseDir, "frame_*.jpg"))
	if err != nil {
		return fmt.Errorf("failed to read frame file: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no frame files found")
	}

	// 按文件名排序确保顺序正确
	sort.Strings(files)
	logger.Info(ctx, "Found %d frame files", len(files))

	// 为了控制 GIF 大小，我们可以跳帧
	// 如果帧数过多（>100），每隔一帧采样
	skipFrames := 1
	if len(files) > 150 {
		skipFrames = 3 // 每3帧取1帧
	} else if len(files) > 100 {
		skipFrames = 2 // 每2帧取1帧
	}

	if skipFrames > 1 {
		logger.Info(ctx, "To control file size, sample 1 frame every %d frames", skipFrames)
	}

	// 准备 GIF 数据结构
	gifData := &gif.GIF{}
	delay := 100 / frameRate // 每帧延迟时间（单位：1/100秒）

	// 处理每一帧
	processedFrames := 0
	for i, framePath := range files {
		// 跳帧处理
		if i%skipFrames != 0 {
			continue
		}

		// 读取 JPEG 帧
		frameFile, err := os.Open(framePath)
		if err != nil {
			logger.Warn(ctx, "Failed to open frame file: %v", err)
			continue
		}

		// 解码 JPEG
		img, err := jpeg.Decode(frameFile)
		frameFile.Close()
		if err != nil {
			logger.Warn(ctx, "Failed to decode frame: %v", err)
			continue
		}

		// 为了减小 GIF 体积，缩小图片尺寸
		// 将宽度缩放到 800px（保持宽高比）
		bounds := img.Bounds()
		origWidth := bounds.Dx()
		origHeight := bounds.Dy()

		targetWidth := 800
		if origWidth < targetWidth {
			targetWidth = origWidth
		}
		targetHeight := origHeight * targetWidth / origWidth

		// 创建缩小后的图片
		resized := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

		// 简单的最近邻缩放
		for y := range targetHeight {
			for x := 0; x < targetWidth; x++ {
				srcX := x * origWidth / targetWidth
				srcY := y * origHeight / targetHeight
				resized.Set(x, y, img.At(srcX, srcY))
			}
		}

		// 转换为调色板图片（GIF 需要）
		palettedImg := image.NewPaletted(resized.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(palettedImg, resized.Bounds(), resized, image.Point{})

		// 添加到 GIF
		gifData.Image = append(gifData.Image, palettedImg)
		gifData.Delay = append(gifData.Delay, delay)

		processedFrames++
		if processedFrames%10 == 0 {
			logger.Info(ctx, "Processed %d/%d frames", processedFrames, (len(files)+skipFrames-1)/skipFrames)
		}
	}

	if len(gifData.Image) == 0 {
		return fmt.Errorf("no frames were processed successfully")
	}

	logger.Info(ctx, "Processed %d frames in total, saving GIF...", len(gifData.Image))

	// 保存 GIF 文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	err = gif.EncodeAll(outFile, gifData)
	if err != nil {
		return fmt.Errorf("failed to encode GIF: %w", err)
	}

	logger.Info(ctx, "✓ GIF conversion completed: %s", outputPath)

	// 获取文件大小
	fileInfo, _ := os.Stat(outputPath)
	if fileInfo != nil {
		fileSizeMB := float64(fileInfo.Size()) / 1024 / 1024
		logger.Info(ctx, "GIF file size: %.2f MB", fileSizeMB)
	}

	// 删除帧目录以节省空间
	if err := os.RemoveAll(baseDir); err != nil {
		logger.Warn(ctx, "Failed to delete frame directory: %v", err)
	} else {
		logger.Info(ctx, "Temporary frame directory cleaned up")
	}

	return nil
}

// PlayScript 回放脚本
func (p *Player) PlayScript(ctx context.Context, page *rod.Page, script *models.Script, currentLang string) error {
	logger.Info(ctx, "Start playing script: %s", script.Name)
	logger.Info(ctx, "Target URL: %s", script.URL)
	logger.Info(ctx, "Total %d operation steps", len(script.Actions))

	// 确保语言设置有默认值
	if currentLang == "" {
		currentLang = "zh-CN"
	}
	logger.Info(ctx, "Using language: %s", currentLang)

	// AI 控制指示器将常驻显示，不再自动隐藏
	// defer p.hideAIControlIndicator(ctx, page)  // 注释掉自动隐藏

	// 重置统计和抓取数据
	p.ResetStats()

	// 初始化变量上下文（包含脚本预设变量）
	variables := make(map[string]string)
	if script.Variables != nil {
		for k, v := range script.Variables {
			variables[k] = v
			logger.Info(ctx, "Initialize variable: %s = %s", k, v)
		}
	}

	// 初始化多标签页支持
	p.pages = make(map[int]*rod.Page)
	p.tabCounter = 0
	p.pages[p.tabCounter] = page
	p.currentPage = page

	// 导航到起始URL
	if script.URL != "" {
		logger.Info(ctx, "Navigate to: %s", script.URL)
		if err := page.Navigate(script.URL); err != nil {
			return fmt.Errorf("navigation failed: %w", err)
		}
		if err := page.WaitLoad(); err != nil {
			logger.Warn(ctx, "Failed to wait for page to load: %v", err)
		}
		// 等待页面稳定
		time.Sleep(2 * time.Second)

		// 页面加载完成后，等待额外时间让 JavaScript 框架初始化完成
		logger.Info(ctx, "Waiting for page JavaScript to stabilize...")
		time.Sleep(1 * time.Second)
	}

	// 保存脚本名称和动作列表，用于后续重新注入时使用
	p.currentScriptName = script.Name
	p.currentActions = script.Actions
	p.currentStepIndex = 0

	// 在页面完全稳定后显示 AI 控制指示器
	p.showAIControlIndicator(ctx, page, script.Name, currentLang)

	// 初始化步骤列表
	p.initAIControlSteps(ctx, page, script.Actions)

	// 预先注入XHR拦截器，监听脚本中所有需要捕获的XHR请求
	// 这样可以避免在执行到capture_xhr action时才开始监听，导致漏掉前面的请求
	if err := p.injectXHRInterceptorForScript(ctx, page, script.Actions); err != nil {
		logger.Warn(ctx, "Failed to inject XHR interceptor: %v", err)
	}

	// 执行每个操作
	for i, action := range script.Actions {
		p.currentStepIndex = i
		logger.Info(ctx, "[%d/%d] Execute action: %s", i+1, len(script.Actions), action.Type)

		// 更新 AI 控制状态显示（标记为执行中）
		p.updateAIControlStatus(ctx, page, i+1, len(script.Actions), action.Type)

		// 检查条件执行
		if action.Condition != nil && action.Condition.Enabled {
			shouldExecute, err := p.evaluateCondition(ctx, action.Condition, variables)
			if err != nil {
				logger.Warn(ctx, "Failed to evaluate condition: %v", err)
			} else if !shouldExecute {
				logger.Info(ctx, "Skipping action due to condition not met: %s %s %s",
					action.Condition.Variable, action.Condition.Operator, action.Condition.Value)
				// 标记为跳过（视为成功）
				p.markStepCompleted(ctx, page, i+1, true)
				continue
			}
			logger.Info(ctx, "Condition met, executing action: %s %s %s",
				action.Condition.Variable, action.Condition.Operator, action.Condition.Value)
		}

		if err := p.executeAction(ctx, page, action); err != nil {
			logger.Warn(ctx, "Action execution failed (continuing with subsequent steps): %v", err)
			p.failCount++
			// 标记步骤为失败
			p.markStepCompleted(ctx, page, i+1, false)
			// 不要中断，继续执行下一步
		} else {
			p.successCount++
			// 标记步骤为成功
			p.markStepCompleted(ctx, page, i+1, true)

			// 如果 action 提取了数据，更新变量上下文
			if action.VariableName != "" && p.extractedData[action.VariableName] != nil {
				variables[action.VariableName] = fmt.Sprintf("%v", p.extractedData[action.VariableName])
				logger.Info(ctx, "Updated variable from extracted data: %s = %s", action.VariableName, variables[action.VariableName])
			}
		}
	}

	logger.Info(ctx, "Script playback completed - Success: %d, Failed: %d, Total: %d", p.successCount, p.failCount, len(script.Actions))
	if len(p.extractedData) > 0 {
		logger.Info(ctx, "Extracted %d data items", len(p.extractedData))
	}

	// 如果所有操作都失败了，返回错误
	if p.failCount > 0 && p.successCount == 0 {
		return fmt.Errorf("all operations failed")
	}

	return nil
}

// evaluateCondition 评估操作执行条件
func (p *Player) evaluateCondition(ctx context.Context, condition *models.ActionCondition, variables map[string]string) (bool, error) {
	if condition == nil {
		return true, nil
	}

	varName := condition.Variable
	operator := condition.Operator
	expectedValue := condition.Value

	// 处理 exists 和 not_exists 操作符
	if operator == "exists" {
		_, exists := variables[varName]
		return exists, nil
	}
	if operator == "not_exists" {
		_, exists := variables[varName]
		return !exists, nil
	}

	// 获取变量值
	actualValue, exists := variables[varName]
	if !exists {
		logger.Warn(ctx, "Variable not found for condition: %s", varName)
		return false, fmt.Errorf("variable not found: %s", varName)
	}

	// 根据操作符进行比较
	switch operator {
	case "=", "==":
		return actualValue == expectedValue, nil

	case "!=":
		return actualValue != expectedValue, nil

	case ">":
		// 尝试数值比较
		return compareNumeric(actualValue, expectedValue, ">")

	case "<":
		return compareNumeric(actualValue, expectedValue, "<")

	case ">=":
		return compareNumeric(actualValue, expectedValue, ">=")

	case "<=":
		return compareNumeric(actualValue, expectedValue, "<=")

	case "in":
		// 检查 actualValue 是否包含在 expectedValue 中（逗号分隔）
		values := strings.Split(expectedValue, ",")
		for _, v := range values {
			if strings.TrimSpace(v) == actualValue {
				return true, nil
			}
		}
		return false, nil

	case "not_in":
		values := strings.Split(expectedValue, ",")
		for _, v := range values {
			if strings.TrimSpace(v) == actualValue {
				return false, nil
			}
		}
		return true, nil

	case "contains":
		return strings.Contains(actualValue, expectedValue), nil

	case "not_contains":
		return !strings.Contains(actualValue, expectedValue), nil

	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// compareNumeric 数值比较辅助函数
func compareNumeric(actual, expected, operator string) (bool, error) {
	// 尝试将字符串转换为浮点数进行比较
	var actualNum, expectedNum float64
	_, err1 := fmt.Sscanf(actual, "%f", &actualNum)
	_, err2 := fmt.Sscanf(expected, "%f", &expectedNum)

	if err1 != nil || err2 != nil {
		// 如果无法转换为数字，则进行字符串比较
		switch operator {
		case ">":
			return actual > expected, nil
		case "<":
			return actual < expected, nil
		case ">=":
			return actual >= expected, nil
		case "<=":
			return actual <= expected, nil
		}
	}

	// 数值比较
	switch operator {
	case ">":
		return actualNum > expectedNum, nil
	case "<":
		return actualNum < expectedNum, nil
	case ">=":
		return actualNum >= expectedNum, nil
	case "<=":
		return actualNum <= expectedNum, nil
	}

	return false, fmt.Errorf("unsupported numeric operator: %s", operator)
}

// executeAction 执行单个操作
func (p *Player) executeAction(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	// 对于跨标签页的操作,使用 currentPage
	activePage := p.currentPage
	if activePage == nil {
		activePage = page
	}

	switch action.Type {
	case "open_tab":
		return p.executeOpenTab(ctx, page, action)
	case "switch_active_tab":
		return p.executeSwitchActiveTab(ctx)
	case "switch_tab":
		return p.executeSwitchTab(ctx, action)
	case "click":
		return p.executeClick(ctx, activePage, action)
	case "input":
		return p.executeInput(ctx, activePage, action)
	case "select":
		return p.executeSelect(ctx, activePage, action)
	case "navigate":
		return p.executeNavigate(ctx, activePage, action)
	case "wait":
		return p.executeWait(ctx, action)
	case "sleep":
		return p.executeSleep(ctx, action)
	case "extract_text":
		return p.executeExtractText(ctx, activePage, action)
	case "extract_html":
		return p.executeExtractHTML(ctx, activePage, action)
	case "extract_attribute":
		return p.executeExtractAttribute(ctx, activePage, action)
	case "execute_js":
		return p.executeJS(ctx, activePage, action)
	case "upload_file":
		return p.executeUploadFile(ctx, activePage, action)
	case "scroll":
		return p.executeScroll(ctx, activePage, action)
	case "keyboard":
		return p.executeKeyboard(ctx, activePage, action)
	case "screenshot":
		return p.executeScreenshot(ctx, activePage, action)
	case "capture_xhr":
		return p.executeCaptureXHR(ctx, activePage, action)
	case "ai_control":
		return p.executeAIControl(ctx, activePage, action)
	default:
		logger.Warn(ctx, "Unknown action type: %s", action.Type)
		return nil
	}
}

// executeClick 执行点击操作
func (p *Player) executeClick(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	selector := action.Selector
	if action.XPath != "" {
		selector = action.XPath
		logger.Info(ctx, "Click element (XPath): %s", selector)
	} else {
		logger.Info(ctx, "Click element (CSS): %s", selector)
	}

	if selector == "" {
		return fmt.Errorf("missing selector information")
	}

	// 重试机制：最多尝试3次
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			logger.Info(ctx, "Retrying attempt %d...", attempt)
			time.Sleep(time.Duration(attempt) * time.Second) // 递增等待时间
		}

		// 使用新的 findElementWithContext 方法（支持 iframe）
		elemCtx, err := p.findElementWithContext(ctx, page, action)
		if err != nil {
			if attempt < maxRetries {
				logger.Warn(ctx, "Element not found, waiting and retrying: %v", err)
				continue
			}
			return fmt.Errorf("element not found: %w", err)
		}

		// 从上下文中提取元素
		element := elemCtx.element

		// 等待元素变为可见和可交互
		if err := element.WaitVisible(); err != nil {
			logger.Warn(ctx, "Failed to wait for element to be visible: %v", err)
		}

		// 滚动到元素可见
		if err := element.ScrollIntoView(); err != nil {
			logger.Warn(ctx, "Failed to scroll to element: %v", err)
		}
		time.Sleep(300 * time.Millisecond)

		// 高亮显示元素
		p.highlightElement(ctx, element)
		defer p.unhighlightElement(ctx, element) // 检查元素是否可点击（pointer-events 不为 none）
		isClickable, _ := element.Eval(`() => {
			const style = window.getComputedStyle(this);
			return style.pointerEvents !== 'none' && style.display !== 'none' && style.visibility !== 'hidden';
		}`)

		if isClickable != nil && !isClickable.Value.Bool() {
			if attempt < maxRetries {
				logger.Warn(ctx, "Element not clickable (pointer-events or display/visibility), waiting and retrying")
				continue
			}
			// 最后一次尝试：尝试用 JavaScript 强制点击
			logger.Warn(ctx, "Element not clickable, trying JavaScript click")
			_, err := element.Eval(`() => this.click()`)
			if err != nil {
				return fmt.Errorf("javaScript click failed: %w", err)
			}
			return nil
		}

		// 尝试点击元素
		err = element.Click(proto.InputMouseButtonLeft, 1)
		if err == nil {
			logger.Info(ctx, "✓ Click successful")
			return nil
		}

		if attempt < maxRetries {
			logger.Warn(ctx, "Click failed, will retry: %v", err)
			continue
		}

		// 最后尝试：用 JavaScript 强制点击
		logger.Warn(ctx, "Regular click failed, trying JavaScript click")
		_, jsErr := element.Eval(`() => this.click()`)
		if jsErr != nil {
			return fmt.Errorf("click failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("click operation failed")
}

// executeInput 执行输入操作
func (p *Player) executeInput(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	selector := action.Selector
	if action.XPath != "" {
		selector = action.XPath
		logger.Info(ctx, "Input text (XPath): %s -> %s", selector, action.Value)
	} else {
		logger.Info(ctx, "Input text (CSS): %s -> %s", selector, action.Value)
	}

	// 使用新的 findElement 方法（支持 iframe）
	elementInfo, err := p.findElementWithContext(ctx, page, action)
	if err != nil {
		return fmt.Errorf("input box not found: %w", err)
	}

	element := elementInfo.element
	targetPage := elementInfo.page // 使用正确的 page（可能是 iframe 的 frame）

	// 等待元素可见
	if err := element.WaitVisible(); err != nil {
		logger.Warn(ctx, "Failed to wait for input element to be visible: %v", err)
	}

	// 滚动到元素可见
	if err := element.ScrollIntoView(); err != nil {
		logger.Warn(ctx, "Failed to scroll to element: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	// 高亮显示元素
	p.highlightElement(ctx, element)
	defer p.unhighlightElement(ctx, element)

	// 先点击获取焦点 - 添加重试逻辑
	clickSuccess := false
	for i := 0; i < 3; i++ {
		if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
			logger.Warn(ctx, "Failed to click input element (attempt %d/3): %v", i+1, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		clickSuccess = true
		logger.Info(ctx, "✓ Click on input element successful")
		break
	}
	if !clickSuccess {
		logger.Warn(ctx, "Multiple failed attempts to click input element, continuing with input")
	}
	time.Sleep(300 * time.Millisecond)

	// 显式聚焦元素
	if err := element.Focus(); err != nil {
		logger.Warn(ctx, "Failed to focus element: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// 检查是否是 contenteditable 元素
	isContentEditable := false
	contentEditableResult, _ := element.Eval(`() => this.contentEditable`)

	if contentEditableResult != nil && contentEditableResult.Value.String() == "true" {
		isContentEditable = true
		logger.Info(ctx, "Detected contenteditable element")
	}

	if isContentEditable {
		// 对于 contenteditable 元素，需要使用真实的键盘事件
		// 因为 Draft.js 等编辑器依赖键盘事件来更新内部状态
		logger.Info(ctx, "Using keyboard input to simulate contenteditable element")

		// 确保元素已获得焦点
		if err := element.Focus(); err != nil {
			logger.Warn(ctx, "Failed to focus element: %v", err)
		}
		time.Sleep(200 * time.Millisecond)

		// contenteditable 元素不支持 SelectAllText，直接使用快捷键清空
		// 使用 Ctrl+A 全选现有内容
		targetPage.KeyActions().Press(input.ControlLeft).Type('a').Release(input.ControlLeft).MustDo()
		time.Sleep(100 * time.Millisecond)

		// 按 Backspace 清空
		targetPage.KeyActions().Press(input.Backspace).MustDo()
		time.Sleep(100 * time.Millisecond)

		// 使用 targetPage.InsertText 方法输入文本（支持 Unicode 字符）
		// InsertText 会触发 beforeinput 和 input 事件，Draft.js 能正确响应
		err := targetPage.InsertText(action.Value)
		if err != nil {
			logger.Warn(ctx, "InsertText failed, trying character-by-character input: %v", err)
			// 回退方案：逐字符输入（只对 ASCII 字符有效）
			for _, char := range action.Value {
				if char < 128 {
					targetPage.KeyActions().Type(input.Key(char)).MustDo()
					time.Sleep(5 * time.Millisecond)
				}
			}
		}

		logger.Info(ctx, "✓ Keyboard input completed")

		// 等待一下让编辑器状态更新
		time.Sleep(300 * time.Millisecond)

	} else {
		// 传统输入框：先尝试清空内容，然后输入
		logger.Info(ctx, "Processing traditional input element")

		// 尝试全选文本（如果失败，使用其他方法清空）
		selectErr := element.SelectAllText()
		if selectErr != nil {
			logger.Warn(ctx, "SelectAllText failed: %v, trying other clearing methods", selectErr)

			// 方法1: 使用 JavaScript 清空
			_, jsErr := element.Eval(`() => { this.value = ''; this.textContent = ''; }`)
			if jsErr != nil {
				logger.Warn(ctx, "JavaScript clearing failed: %v", jsErr)
			}

			// 方法2: 使用快捷键清空
			targetPage.KeyActions().Press(input.ControlLeft).Type('a').Release(input.ControlLeft).MustDo()
			time.Sleep(50 * time.Millisecond)
			targetPage.KeyActions().Press(input.Backspace).MustDo()
			time.Sleep(50 * time.Millisecond)
		} else {
			logger.Info(ctx, "✓ Text selection successful")
		}

		// 尝试输入文本
		inputErr := element.Input(action.Value)
		if inputErr != nil {
			logger.Warn(ctx, "element.Input failed: %v, trying InsertText", inputErr)

			// 回退到 InsertText 方法
			insertErr := targetPage.InsertText(action.Value)
			if insertErr != nil {
				return fmt.Errorf("failed to input text (Input: %v, InsertText: %v)", inputErr, insertErr)
			}
			logger.Info(ctx, "✓ Input successful using InsertText")
		} else {
			logger.Info(ctx, "✓ Input successful using element.Input")
		}
	}

	// 触发额外的事件来确保编辑器识别内容变化
	// 这对富文本编辑器（如 CSDN）特别重要
	time.Sleep(200 * time.Millisecond)

	// 构建选择器参数（去掉 iframe 前缀，因为我们已经在正确的上下文中）
	elemSelector := action.Selector
	elemXPath := action.XPath

	// 如果是 iframe 元素，移除 "iframe " 前缀和 "//iframe" 前缀
	if len(elemSelector) > 7 && elemSelector[:7] == "iframe " {
		elemSelector = elemSelector[7:]
	}
	if len(elemXPath) > 8 && elemXPath[:8] == "//iframe" {
		elemXPath = elemXPath[8:]
	}

	_, triggerErr := targetPage.Eval(`(sel, xp, val) => {
		// 尝试找到元素
		let element = null;
		if (xp) {
			try {
				element = document.evaluate(xp, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
			} catch (e) {}
		}
		if (!element && sel && sel !== 'unknown') {
			try {
				element = document.querySelector(sel);
			} catch (e) {}
		}
		
		if (!element) {
			console.warn('[BrowserWing] Could not find element for event trigger');
			return false;
		}
		
		console.log('[BrowserWing] Triggering editor update events for:', element.tagName);
		
		// 1. 触发标准事件序列
		const events = ['input', 'change', 'keyup'];
		events.forEach(eventType => {
			try {
				const event = new Event(eventType, { bubbles: true, cancelable: true });
				element.dispatchEvent(event);
			} catch (e) {
				console.warn('Failed to dispatch ' + eventType, e);
			}
		});
		
		// 2. 对于 contenteditable，强制设置内容并触发更多事件
		if (element.contentEditable === 'true' || element.isContentEditable) {
			try {
				// 保存当前内容
				const currentContent = element.textContent || element.innerText || '';
				
				// 如果当前内容与预期不同，强制设置
				if (currentContent !== val && val) {
					console.log('[BrowserWing] Force setting content:', val.substring(0, 50));
					element.textContent = val;
				}
				
				// 触发 focus 确保编辑器激活
				element.focus();
				
				// 触发 InputEvent（现代编辑器依赖此事件）
				try {
					const inputEvent = new InputEvent('input', {
						bubbles: true,
						cancelable: true,
						inputType: 'insertText',
						data: val
					});
					element.dispatchEvent(inputEvent);
				} catch (e) {
					console.warn('InputEvent failed', e);
				}
				
				// 触发 compositionend（某些亚洲语言输入法编辑器需要）
				try {
					const compositionEvent = new CompositionEvent('compositionend', {
						bubbles: true,
						cancelable: true,
						data: val
					});
					element.dispatchEvent(compositionEvent);
				} catch (e) {
					console.warn('CompositionEvent failed', e);
				}
				
				// 触发 DOMCharacterDataModified（旧版编辑器可能需要）
				try {
					const mutationEvent = document.createEvent('MutationEvent');
					mutationEvent.initMutationEvent('DOMCharacterDataModified', true, false, element, '', val, '', 0);
					element.dispatchEvent(mutationEvent);
				} catch (e) {
					// DOMCharacterDataModified 已废弃，某些浏览器可能不支持
				}
				
				// 短暂失焦再聚焦，触发编辑器的验证逻辑
				setTimeout(() => {
					element.blur();
					const blurEvent = new Event('blur', { bubbles: true });
					element.dispatchEvent(blurEvent);
					
					setTimeout(() => {
						element.focus();
						const focusEvent = new Event('focus', { bubbles: true });
						element.dispatchEvent(focusEvent);
					}, 50);
				}, 100);
				
				console.log('[BrowserWing] Editor update events triggered successfully');
				
			} catch (e) {
				console.warn('Failed to update contenteditable', e);
			}
		}
		
		return true;
	}`, elemSelector, elemXPath, action.Value)

	if triggerErr != nil {
		logger.Warn(ctx, "Failed to trigger editor update event: %v", triggerErr)
	} else {
		logger.Info(ctx, "✓ Editor content update event triggered")
	}

	// 再等待一下确保编辑器完全响应
	time.Sleep(500 * time.Millisecond)

	logger.Info(ctx, "✓ Input successful")
	return nil
}

// executeSelect 执行选择操作
func (p *Player) executeSelect(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	selector := action.Selector
	if action.XPath != "" {
		selector = action.XPath
		logger.Info(ctx, "Select option (XPath): %s -> %s", selector, action.Value)
	} else {
		logger.Info(ctx, "Select option (CSS): %s -> %s", selector, action.Value)
	}

	// 使用新的 findElementWithContext 方法（支持 iframe）
	elemCtx, err := p.findElementWithContext(ctx, page, action)
	if err != nil {
		return fmt.Errorf("select box not found: %w", err)
	}

	// 从上下文中提取元素
	element := elemCtx.element

	// 等待元素可见
	if err := element.WaitVisible(); err != nil {
		logger.Warn(ctx, "Failed to wait for select element to be visible: %v", err)
	}

	// 滚动到元素可见
	if err := element.ScrollIntoView(); err != nil {
		logger.Warn(ctx, "Failed to scroll to element: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	// 高亮显示元素
	p.highlightElement(ctx, element)
	defer p.unhighlightElement(ctx, element)

	// 选择值
	if err := element.Select([]string{action.Value}, true, rod.SelectorTypeText); err != nil {
		return fmt.Errorf("failed to select option: %w", err)
	}

	logger.Info(ctx, "✓ Selection successful")
	return nil
}

// executeNavigate 执行导航操作
func (p *Player) executeNavigate(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	logger.Info(ctx, "Navigate to: %s", action.URL)

	if err := page.Navigate(action.URL); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return fmt.Errorf("failed to wait for page to load: %w", err)
	}

	p.ensureAIControlIndicator(ctx, page)

	return nil
}

// executeWait 执行等待操作
func (p *Player) executeWait(ctx context.Context, action models.ScriptAction) error {
	duration := time.Duration(action.Timestamp) * time.Millisecond
	logger.Info(ctx, "Wait for: %v", duration)
	time.Sleep(duration)
	return nil
}

// executeSleep 执行延迟操作
func (p *Player) executeSleep(ctx context.Context, action models.ScriptAction) error {
	duration := time.Duration(action.Duration) * time.Millisecond
	logger.Info(ctx, "Delay: %v", duration)
	time.Sleep(duration)
	return nil
}

// findElementWithContext 查找元素并返回其页面上下文（支持 iframe）
func (p *Player) findElementWithContext(ctx context.Context, page *rod.Page, action models.ScriptAction) (*elementContext, error) {
	selector := action.Selector
	xpath := action.XPath

	// 检查是否是 iframe 内的元素
	isIframeElement := false
	innerXPath := ""
	innerCSS := ""

	if xpath != "" && len(xpath) > 8 && xpath[:8] == "//iframe" {
		isIframeElement = true
		// 提取 iframe 后面的路径，例如 "//iframe//body" -> "//body"
		// 注意：xpath[8:] 会是 "//body"，不需要再加 "/"
		if len(xpath) > 8 {
			remaining := xpath[8:] // 例如 "//body"
			// 确保是有效的 XPath
			if len(remaining) > 0 && remaining[0] == '/' {
				innerXPath = remaining // 已经有前导 /，直接使用
			} else {
				innerXPath = "//" + remaining // 补充 //
			}
		}
	} else if selector != "" && len(selector) > 7 && selector[:7] == "iframe " {
		isIframeElement = true
		// 提取 iframe 后面的选择器，例如 "iframe body" -> "body"
		innerCSS = selector[7:]
	}

	// 如果是 iframe 内的元素
	if isIframeElement {
		logger.Info(ctx, "Detected element inside iframe, preparing to switch to iframe")
		logger.Info(ctx, "Inner iframe XPath: %s, CSS: %s", innerXPath, innerCSS)

		// 先找到所有 iframe
		iframes, err := page.Elements("iframe")
		if err != nil {
			return nil, fmt.Errorf("failed to find iframe: %w", err)
		}

		if len(iframes) == 0 {
			return nil, fmt.Errorf("no iframe found in page")
		}

		logger.Info(ctx, "Found %d iframes, attempting to find element in each", len(iframes))
		// 尝试在每个 iframe 中查找元素
		for i, iframe := range iframes {
			logger.Info(ctx, "Trying iframe #%d", i)

			// 获取 iframe 的 contentDocument
			frame, frameErr := iframe.Frame()
			if frameErr != nil {
				logger.Warn(ctx, "Failed to get Frame for iframe #%d: %v", i, frameErr)
				continue
			}

			// 等待 iframe 加载
			if err := frame.WaitLoad(); err != nil {
				logger.Warn(ctx, "Failed to wait for iframe #%d to load: %v", i, err)
			}

			// 在 iframe 中查找元素
			var element *rod.Element
			var findErr error

			if innerXPath != "" {
				// 使用 XPath 查找
				element, findErr = frame.Timeout(3 * time.Second).ElementX(innerXPath)
			} else if innerCSS != "" {
				// 使用 CSS 选择器查找
				element, findErr = frame.Timeout(3 * time.Second).Element(innerCSS)
			} else {
				logger.Warn(ctx, "Inner iframe element selector is empty")
				continue
			}

			if findErr == nil && element != nil {
				logger.Info(ctx, "✓ Found element in iframe #%d", i)
				// 返回元素及其所在的 frame 作为页面上下文
				return &elementContext{
					element: element,
					page:    frame,
				}, nil
			}

			logger.Warn(ctx, "Element not found in iframe #%d: %v", i, findErr)
		}

		return nil, fmt.Errorf("element not found in any iframe")
	}

	// 普通元素（非 iframe）
	var element *rod.Element
	var err error

	if xpath != "" {
		element, err = page.Timeout(5 * time.Second).ElementX(xpath)
		if err != nil && selector != "" && selector != "unknown" {
			logger.Warn(ctx, "XPath lookup failed, trying CSS: %v", err)
			element, err = page.Timeout(5 * time.Second).Element(selector)
		}
	} else if selector != "" && selector != "unknown" {
		element, err = page.Timeout(5 * time.Second).Element(selector)
	} else {
		return nil, fmt.Errorf("missing valid selector")
	}

	if err != nil {
		return nil, err
	}

	// 普通元素返回主页面作为上下文
	return &elementContext{
		element: element,
		page:    page,
	}, nil
}

// executeExtractText 执行文本抓取操作
func (p *Player) executeExtractText(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	logger.Info(ctx, "Extract text data: %s", action.Selector)

	elemCtx, err := p.findElementWithContext(ctx, page, action)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	element := elemCtx.element

	// 等待元素可见
	if err := element.WaitVisible(); err != nil {
		logger.Warn(ctx, "Failed to wait for element to be visible: %v", err)
	}

	// 高亮显示元素
	p.highlightElement(ctx, element)
	defer p.unhighlightElement(ctx, element)

	// 获取文本内容
	text, err := element.Text()
	if err != nil {
		return fmt.Errorf("failed to get text: %w", err)
	}

	// 存储抓取的数据
	varName := action.VariableName
	if varName == "" {
		varName = fmt.Sprintf("text_data_%d", len(p.extractedData))
	}
	p.extractedData[varName] = text

	logger.Info(ctx, "✓ Text extraction successful: %s = %s", varName, text)
	return nil
}

// executeExtractHTML 执行 HTML 抓取操作
func (p *Player) executeExtractHTML(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	logger.Info(ctx, "Extract HTML data: %s", action.Selector)

	elemCtx, err := p.findElementWithContext(ctx, page, action)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	element := elemCtx.element

	// 等待元素可见
	if err := element.WaitVisible(); err != nil {
		logger.Warn(ctx, "Failed to wait for element to be visible: %v", err)
	}

	// 高亮显示元素
	p.highlightElement(ctx, element)
	defer p.unhighlightElement(ctx, element)

	// 获取 HTML 内容
	html, err := element.HTML()
	if err != nil {
		return fmt.Errorf("failed to get HTML: %w", err)
	}

	// 存储抓取的数据
	varName := action.VariableName
	if varName == "" {
		varName = fmt.Sprintf("html_data_%d", len(p.extractedData))
	}
	p.extractedData[varName] = html

	logger.Info(ctx, "✓ HTML extraction successful: %s (length: %d)", varName, len(html))
	return nil
}

// executeExtractAttribute 执行属性抓取操作
func (p *Player) executeExtractAttribute(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	logger.Info(ctx, "Extract attribute data: %s[%s]", action.Selector, action.AttributeName)

	if action.AttributeName == "" {
		return fmt.Errorf("attribute name to extract not specified")
	}

	elemCtx, err := p.findElementWithContext(ctx, page, action)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	element := elemCtx.element

	// 等待元素可见
	if err := element.WaitVisible(); err != nil {
		logger.Warn(ctx, "Failed to wait for element to be visible: %v", err)
	}

	// 高亮显示元素
	p.highlightElement(ctx, element)
	defer p.unhighlightElement(ctx, element)

	// 获取属性值
	attrValue, err := element.Attribute(action.AttributeName)
	if err != nil {
		return fmt.Errorf("failed to get attribute: %w", err)
	}

	if attrValue == nil {
		return fmt.Errorf("attribute %s does not exist", action.AttributeName)
	}

	// 存储抓取的数据
	varName := action.VariableName
	if varName == "" {
		varName = fmt.Sprintf("attr_data_%d", len(p.extractedData))
	}
	p.extractedData[varName] = *attrValue

	logger.Info(ctx, "✓ Attribute extraction successful: %s = %s", varName, *attrValue)
	return nil
}

// executeJS 执行 JavaScript 并返回结果
func (p *Player) executeJS(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	logger.Info(ctx, "Execute JavaScript code")

	if action.JSCode == "" {
		return fmt.Errorf("JavaScript code to execute not specified")
	}

	jsCode := strings.TrimSpace(action.JSCode)
	logger.Info(ctx, "Original code length: %d characters", len(jsCode))

	// 对于包含多行语句的代码（如函数声明 + 调用），
	// 需要包装在一个函数表达式中
	// Rod 的 Eval 期望的是一个函数表达式： () => { ... }

	var wrappedCode string

	// 检查是否已经是函数表达式格式
	if strings.HasPrefix(jsCode, "() =>") || strings.HasPrefix(jsCode, "function()") {
		wrappedCode = jsCode
		logger.Info(ctx, "Already in function expression format")
	} else if strings.HasPrefix(jsCode, "(() =>") && (strings.HasSuffix(jsCode, ")()") || strings.HasSuffix(jsCode, ")();")) {
		// 已经是 IIFE：(() => { ... })()
		// 需要转换为函数表达式：() => { ... }
		// 去掉外层的 ( 和 )()
		if strings.HasSuffix(jsCode, ")();") {
			wrappedCode = jsCode[1 : len(jsCode)-4]
		} else {
			wrappedCode = jsCode[1 : len(jsCode)-3]
		}
		logger.Info(ctx, "Convert from IIFE format to function expression, wrappedCode: %s", wrappedCode)
	} else {
		// 包含普通代码或函数声明，包装为函数表达式
		// 关键：需要 return 最后的表达式结果
		// 如果代码包含函数调用（如 extractData()），需要确保返回它的结果

		// 检查代码最后是否有函数调用
		lines := strings.Split(jsCode, "\n")
		lastLine := strings.TrimSpace(lines[len(lines)-1])

		if strings.HasSuffix(lastLine, "()") || strings.HasSuffix(lastLine, "();") {
			// 最后一行是函数调用，需要 return 它
			// 去掉最后的分号（如果有）
			lastLine = strings.TrimSuffix(lastLine, ";")
			// 重新组合代码，在最后一行前加 return
			lines[len(lines)-1] = "return " + lastLine + ";"
			jsCode = strings.Join(lines, "\n")
			logger.Info(ctx, "Add return before the final function call")
		}

		wrappedCode = "() => { " + jsCode + " }"
		logger.Info(ctx, "Wrap as function expression format")
	}

	// 执行 JavaScript
	result, err := page.Eval(wrappedCode)
	if err != nil {
		// 如果失败，尝试记录详细信息
		logger.Error(ctx, "JavaScript execution failed, code snippet: %s...", wrappedCode[:min(200, len(wrappedCode))])
		return fmt.Errorf("failed to execute JavaScript: %w", err)
	}

	// 存储抓取的数据
	// 有可能是表单填充的动作，没有return，则不用存储数据
	if !strings.Contains(wrappedCode, "return") {
		logger.Info(ctx, "No return statement detected, skipping result storage")
		return nil
	}

	varName := action.VariableName
	if varName == "" {
		varName = fmt.Sprintf("js_result_%d", len(p.extractedData))
	}
	p.extractedData[varName] = result.Value

	logger.Info(ctx, "✓ JavaScript execution successful: %s", varName)
	return nil
}

// executeScroll 执行滚动操作
func (p *Player) executeScroll(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	scrollX := action.ScrollX
	scrollY := action.ScrollY

	logger.Info(ctx, "Scroll to position: X=%d, Y=%d", scrollX, scrollY)

	// 使用 JavaScript 执行滚动
	_, err := page.Eval(fmt.Sprintf(`() => {
		window.scrollTo(%d, %d);
		return true;
	}`, scrollX, scrollY))
	if err != nil {
		return fmt.Errorf("failed to scroll: %w", err)
	}

	// 等待滚动完成
	time.Sleep(500 * time.Millisecond)

	logger.Info(ctx, "✓ Scroll successful")
	return nil
}

// downloadFileFromURL 从 HTTP(S) URL 下载文件到临时目录
func (p *Player) downloadFileFromURL(ctx context.Context, url string) (string, error) {
	logger.Info(ctx, "Downloading file from URL: %s", url)

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 执行请求
	client := &http.Client{
		Timeout: 5 * time.Minute, // 5分钟超时
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	// 从 URL 中提取文件名
	urlPath := strings.TrimRight(url, "/")
	fileName := filepath.Base(urlPath)
	if fileName == "." || fileName == "/" || fileName == "" {
		fileName = "downloaded_file"
	}

	// 如果文件名没有扩展名，尝试从 Content-Type 推断
	if filepath.Ext(fileName) == "" {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "image/jpeg") || strings.Contains(contentType, "image/jpg") {
			fileName += ".jpg"
		} else if strings.Contains(contentType, "image/png") {
			fileName += ".png"
		} else if strings.Contains(contentType, "image/gif") {
			fileName += ".gif"
		} else if strings.Contains(contentType, "application/pdf") {
			fileName += ".pdf"
		}
	}

	// 创建临时文件
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fileName)

	// 创建目标文件
	out, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// 复制内容到文件
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tempFile) // 清理失败的文件
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	logger.Info(ctx, "✓ File downloaded successfully to: %s", tempFile)
	return tempFile, nil
}

// executeUploadFile 执行文件上传操作
func (p *Player) executeUploadFile(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	selector := action.Selector
	if action.XPath != "" {
		selector = action.XPath
		logger.Info(ctx, "Upload file to element (XPath): %s", selector)
	} else {
		logger.Info(ctx, "Upload file to element (CSS): %s", selector)
	}

	if selector == "" {
		return fmt.Errorf("missing selector information")
	}

	// 检查是否有文件路径
	if len(action.FilePaths) == 0 {
		return fmt.Errorf("no file paths specified for upload")
	}

	logger.Info(ctx, "Preparing to upload %d files: %v", len(action.FilePaths), action.FilePaths)

	// 处理 HTTP(S) 链接，先下载到本地
	localFilePaths := make([]string, 0, len(action.FilePaths))
	downloadedFiles := make([]string, 0) // 记录需要清理的临时文件

	for _, filePath := range action.FilePaths {
		// 检查是否是 HTTP(S) 链接
		if strings.HasPrefix(strings.ToLower(filePath), "http://") ||
			strings.HasPrefix(strings.ToLower(filePath), "https://") {
			// 下载文件到本地
			localPath, err := p.downloadFileFromURL(ctx, filePath)
			if err != nil {
				// 清理已下载的临时文件
				for _, tmpFile := range downloadedFiles {
					os.Remove(tmpFile)
				}
				return fmt.Errorf("failed to download file from %s: %w", filePath, err)
			}
			localFilePaths = append(localFilePaths, localPath)
			downloadedFiles = append(downloadedFiles, localPath)
		} else {
			// 本地文件路径，直接使用
			localFilePaths = append(localFilePaths, filePath)
		}
	}

	// 延迟清理下载的临时文件
	defer func() {
		for _, tmpFile := range downloadedFiles {
			if err := os.Remove(tmpFile); err != nil {
				logger.Warn(ctx, "Failed to cleanup temp file %s: %v", tmpFile, err)
			} else {
				logger.Info(ctx, "Cleaned up temp file: %s", tmpFile)
			}
		}
	}()

	logger.Info(ctx, "Local file paths ready: %v", localFilePaths)

	// 重试机制：最多尝试3次
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			logger.Info(ctx, "Retrying attempt %d...", attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		// 查找文件输入元素
		elemCtx, err := p.findElementWithContext(ctx, page, action)
		if err != nil {
			if attempt < maxRetries {
				logger.Warn(ctx, "Element not found, waiting and retrying: %v", err)
				continue
			}
			return fmt.Errorf("element not found: %w", err)
		}

		element := elemCtx.element

		// 验证元素类型（file input 通常是隐藏的，所以先验证类型再处理可见性）
		tagName, err := element.Eval(`() => this.tagName.toLowerCase()`)
		if err != nil {
			if attempt < maxRetries {
				logger.Warn(ctx, "Failed to get element tag, waiting and retrying: %v", err)
				continue
			}
			return fmt.Errorf("failed to get element tag: %w", err)
		}

		inputType, _ := element.Eval(`() => this.type`)
		if tagName.Value.String() != "input" || (inputType != nil && inputType.Value.String() != "file") {
			return fmt.Errorf("element is not a file input (tagName=%s, type=%s)",
				tagName.Value.String(),
				inputType.Value.String())
		}

		logger.Info(ctx, "Found file input element, preparing to upload files...")

		// file input 经常是隐藏的，不需要等待可见或滚动
		// 直接尝试设置文件即可

		// 高亮显示元素（即使是隐藏的也可以高亮其父元素）
		p.highlightElement(ctx, element)

		// 使用 SetFiles 设置文件（使用处理后的本地文件路径）
		err = element.SetFiles(localFilePaths)
		if err == nil {
			logger.Info(ctx, "✓ File upload successful")

			// 等待文件上传处理（等待可能的异步上传或验证）
			// 检查是否有 change 事件监听器被触发
			time.Sleep(1 * time.Second)

			// 可选：等待网络活动稳定（如果页面在上传后有 AJAX 请求）
			// 这里等待2秒，让页面处理文件选择后的逻辑
			logger.Info(ctx, "Waiting for file processing...")
			time.Sleep(2 * time.Second)

			// 取消高亮
			p.unhighlightElement(ctx, element)

			return nil
		}

		if attempt < maxRetries {
			logger.Warn(ctx, "Failed to set files, waiting and retrying: %v", err)
			p.unhighlightElement(ctx, element)
			continue
		}
		p.unhighlightElement(ctx, element)
		return fmt.Errorf("failed to set file: %w", err)
	}

	return fmt.Errorf("file upload failed after %d retries", maxRetries)
}

// executeKeyboard 执行键盘事件操作
func (p *Player) executeKeyboard(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	key := action.Key
	if key == "" {
		return fmt.Errorf("keyboard action missing key")
	}

	logger.Info(ctx, "Executing keyboard action: %s", key)

	var element *rod.Element
	var err error

	// 如果有选择器，先定位到目标元素并聚焦
	if action.Selector != "" || action.XPath != "" {
		elementInfo, findErr := p.findElementWithContext(ctx, page, action)
		if findErr != nil {
			logger.Warn(ctx, "Failed to find target element for keyboard action, executing on page: %v", findErr)
		} else {
			element = elementInfo.element
			page = elementInfo.page // 使用正确的 page（可能是 iframe 的 frame）

			// 等待元素可见
			if err := element.WaitVisible(); err != nil {
				logger.Warn(ctx, "Element not visible: %v", err)
			}

			// 滚动到元素可见
			if err := element.ScrollIntoView(); err != nil {
				logger.Warn(ctx, "Failed to scroll to element: %v", err)
			}

			// 高亮显示元素
			p.highlightElement(ctx, element)
			defer p.unhighlightElement(ctx, element)

			// 聚焦元素
			if err := element.Focus(); err != nil {
				logger.Warn(ctx, "Failed to focus element: %v", err)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

	// 执行键盘操作
	switch key {
	case "ctrl+a":
		// 全选操作：Ctrl+A (Windows/Linux) 或 Cmd+A (Mac)
		logger.Info(ctx, "Executing select all (Ctrl+A)")

		// 根据操作系统选择不同的实现方式
		if runtime.GOOS == "darwin" {
			// Mac 使用 KeyActions API（更可靠）
			logger.Info(ctx, "Using Command key for Mac with KeyActions")
			err = page.KeyActions().Press(input.MetaLeft).Type(input.KeyA).Release(input.MetaLeft).Do()
			if err != nil {
				return fmt.Errorf("failed to execute Cmd+A: %w", err)
			}
		} else {
			// Windows/Linux 使用原有方法
			keyboard := page.Keyboard
			err = keyboard.Press(input.ControlLeft)
			if err != nil {
				return fmt.Errorf("failed to press Ctrl: %w", err)
			}
			err = keyboard.Type(input.KeyA)
			if err != nil {
				keyboard.Release(input.ControlLeft)
				return fmt.Errorf("failed to press A: %w", err)
			}
			err = keyboard.Release(input.ControlLeft)
			if err != nil {
				return fmt.Errorf("failed to release Ctrl: %w", err)
			}
		}

	case "ctrl+c":
		// 复制操作：Ctrl+C (Windows/Linux) 或 Cmd+C (Mac)
		logger.Info(ctx, "Executing copy (Ctrl+C)")

		// 根据操作系统选择不同的实现方式
		if runtime.GOOS == "darwin" {
			// Mac 使用 KeyActions API（更可靠）
			logger.Info(ctx, "Using Command key for Mac with KeyActions")
			err = page.KeyActions().Press(input.MetaLeft).Type(input.KeyC).Release(input.MetaLeft).Do()
			if err != nil {
				return fmt.Errorf("failed to execute Cmd+C: %w", err)
			}
		} else {
			// Windows/Linux 使用原有方法
			keyboard := page.Keyboard
			err = keyboard.Press(input.ControlLeft)
			if err != nil {
				return fmt.Errorf("failed to press Ctrl: %w", err)
			}
			err = keyboard.Type(input.KeyC)
			if err != nil {
				keyboard.Release(input.ControlLeft)
				return fmt.Errorf("failed to press C: %w", err)
			}
			err = keyboard.Release(input.ControlLeft)
			if err != nil {
				return fmt.Errorf("failed to release Ctrl: %w", err)
			}
		}

	case "ctrl+v":
		// 粘贴操作：Ctrl+V (Windows/Linux) 或 Cmd+V (Mac)
		logger.Info(ctx, "Executing paste (Ctrl+V)")

		// 根据操作系统选择不同的实现方式
		if runtime.GOOS == "darwin" {
			// Mac 使用多种方法尝试
			logger.Info(ctx, "Using Command key for Mac with KeyActions")

			// 先确保元素已聚焦
			if element != nil {
				logger.Info(ctx, "Ensuring element is focused before paste")
				if err := element.Focus(); err != nil {
					logger.Warn(ctx, "Failed to focus element: %v", err)
				}
				time.Sleep(200 * time.Millisecond)
			}

			// 记录粘贴前的内容（如果有目标元素）
			var beforeValue string
			if element != nil {
				valueResult, _ := element.Eval(`() => this.value || this.textContent || this.innerText || ''`)
				if valueResult != nil {
					beforeValue = valueResult.Value.String()
					logger.Info(ctx, "Content before paste: length=%d", len(beforeValue))
				}
			}

			// 方法1: 使用 KeyActions 多次尝试
			pasteSuccess := false
			for attempt := 0; attempt < 3; attempt++ {
				if attempt > 0 {
					logger.Info(ctx, "Retry paste attempt %d", attempt+1)
					time.Sleep(300 * time.Millisecond)
				}

				keyboard := page.Keyboard
				err = keyboard.Press(input.MetaLeft)
				if err != nil {
					return fmt.Errorf("failed to press Cmd: %w", err)
				}
				err = keyboard.Type(input.KeyV)
				if err != nil {
					keyboard.Release(input.MetaLeft)
					return fmt.Errorf("failed to press V: %w", err)
				}
				err = keyboard.Release(input.MetaLeft)
				if err != nil {
					return fmt.Errorf("failed to release Cmd: %w", err)
				}

				// 等待一下看粘贴是否生效
				time.Sleep(500 * time.Millisecond)

				// 检查内容是否发生变化
				if element != nil {
					valueResult, _ := element.Eval(`() => this.value || this.textContent || this.innerText || ''`)
					if valueResult != nil {
						afterValue := valueResult.Value.String()
						// 内容发生变化才认为粘贴成功
						if afterValue != beforeValue {
							pasteSuccess = true
							logger.Info(ctx, "✓ Paste successful via KeyActions, content changed (length: %d -> %d)", len(beforeValue), len(afterValue))
							break
						}
					}
				} else {
					// 没有目标元素，假设成功
					pasteSuccess = true
					logger.Info(ctx, "✓ Paste completed via KeyActions (no target element to verify)")
					break
				}
			}

			// 如果 KeyActions 成功，直接返回，避免重复粘贴
			if pasteSuccess {
				logger.Info(ctx, "✓ Keyboard action completed: %s", key)
				return nil
			}

			// KeyActions 失败，尝试使用 navigator.clipboard API
			logger.Warn(ctx, "KeyActions paste did not change content, trying navigator.clipboard API")

			// 方法2: 使用 navigator.clipboard API 读取剪贴板（支持富文本）
			_, jsErr := page.Eval(`async () => {
					try {
						console.log('[BrowserWing] Attempting to read clipboard...');
						
						// 获取当前聚焦的元素
						const activeElement = document.activeElement;
						if (!activeElement) {
							console.warn('[BrowserWing] No active element');
							return false;
						}
						
						console.log('[BrowserWing] Active element type:', activeElement.tagName, activeElement.contentEditable);
						
						// 尝试读取剪贴板数据（包括富文本）
						let clipboardText = '';
						let clipboardHTML = '';
						
						try {
							// 首先尝试 clipboard.read() 来获取富文本
							const clipboardItems = await navigator.clipboard.read();
							console.log('[BrowserWing] Clipboard items:', clipboardItems.length);
							
							for (const item of clipboardItems) {
								console.log('[BrowserWing] Clipboard item types:', item.types);
								
								// 优先读取 HTML 格式
								if (item.types.includes('text/html')) {
									const blob = await item.getType('text/html');
									clipboardHTML = await blob.text();
									console.log('[BrowserWing] Got HTML from clipboard, length:', clipboardHTML.length);
								}
								
								// 读取纯文本作为后备
								if (item.types.includes('text/plain')) {
									const blob = await item.getType('text/plain');
									clipboardText = await blob.text();
									console.log('[BrowserWing] Got text from clipboard:', clipboardText.substring(0, 50));
								}
							}
						} catch (readErr) {
							console.warn('[BrowserWing] clipboard.read() failed, trying readText():', readErr);
							// 回退到 readText()（只支持纯文本）
							clipboardText = await navigator.clipboard.readText();
							console.log('[BrowserWing] Got text via readText():', clipboardText.substring(0, 50));
						}
						
						// 如果两者都没有，失败
						if (!clipboardHTML && !clipboardText) {
							console.error('[BrowserWing] No clipboard content available');
							return false;
						}
						
						// 根据元素类型粘贴
						if (activeElement.tagName === 'INPUT' || activeElement.tagName === 'TEXTAREA') {
							// 传统输入框：只能插入纯文本
							// 注意：TEXTAREA 永远不支持富文本，只能用纯文本
							console.log('[BrowserWing] Detected INPUT/TEXTAREA, using plain text only');
							
							const start = activeElement.selectionStart || 0;
							const end = activeElement.selectionEnd || 0;
							const currentValue = activeElement.value || '';
							
							// 在光标位置插入文本
							activeElement.value = currentValue.substring(0, start) + clipboardText + currentValue.substring(end);
							
							// 设置新的光标位置
							const newPos = start + clipboardText.length;
							activeElement.setSelectionRange(newPos, newPos);
							
							// 触发事件
							activeElement.dispatchEvent(new Event('input', { bubbles: true }));
							activeElement.dispatchEvent(new Event('change', { bubbles: true }));
							
							console.log('[BrowserWing] Pasted plain text to input/textarea');
							return true;
							
						} else if (activeElement.isContentEditable || activeElement.contentEditable === 'true') {
							// contenteditable 元素：支持富文本
							console.log('[BrowserWing] Detected contenteditable element, attempting rich text paste');
							
							// 对于 React 编辑器，优先使用浏览器原生粘贴事件
							// 而不是直接操作 DOM，避免破坏 React 状态
							
							// 尝试触发原生 paste 事件（最佳，不破坏框架状态）
							try {
								const pasteEvent = new ClipboardEvent('paste', {
									bubbles: true,
									cancelable: true,
									clipboardData: new DataTransfer()
								});
								
								// 设置剪贴板数据
								if (clipboardHTML) {
									pasteEvent.clipboardData.setData('text/html', clipboardHTML);
								}
								pasteEvent.clipboardData.setData('text/plain', clipboardText);
								
								// 触发 paste 事件，让编辑器自己处理
								activeElement.dispatchEvent(pasteEvent);
								
								// ClipboardEvent 已触发，让编辑器处理，直接返回成功
								// 不再执行手动插入逻辑，避免重复粘贴
								console.log('[BrowserWing] Paste event dispatched to editor');
								return true;
								
							} catch (eventErr) {
								console.warn('[BrowserWing] Failed to dispatch paste event, trying manual insertion:', eventErr);
							}
							
							// 如果 ClipboardEvent 触发失败，才使用手动插入（回退方案）
							console.log('[BrowserWing] Fallback to manual HTML insertion');
							
							// 获取当前选区
							const selection = window.getSelection();
							if (!selection || selection.rangeCount === 0) {
								console.warn('[BrowserWing] No selection range');
								// 尝试聚焦元素并创建选区
								activeElement.focus();
								if (selection && selection.rangeCount > 0) {
									console.log('[BrowserWing] Created selection after focus');
								} else {
									// 最后尝试：直接设置 innerHTML
									if (clipboardHTML) {
										activeElement.innerHTML = clipboardHTML;
									} else {
										activeElement.textContent = clipboardText;
									}
									activeElement.dispatchEvent(new Event('input', { bubbles: true }));
									return true;
								}
							}
							
							if (selection && selection.rangeCount > 0) {
								const range = selection.getRangeAt(0);
								range.deleteContents();
								
								if (clipboardHTML) {
									// 插入 HTML 内容（保留格式）
									console.log('[BrowserWing] Inserting HTML content via range');
									const fragment = range.createContextualFragment(clipboardHTML);
									range.insertNode(fragment);
									
									// 移动光标到插入内容之后
									range.collapse(false);
									selection.removeAllRanges();
									selection.addRange(range);
									
								} else {
									// 只有纯文本，使用 insertText
									console.log('[BrowserWing] Inserting plain text via execCommand');
									document.execCommand('insertText', false, clipboardText);
								}
								
								// 触发事件
								activeElement.dispatchEvent(new Event('input', { bubbles: true }));
								activeElement.dispatchEvent(new Event('change', { bubbles: true }));
								
								console.log('[BrowserWing] Pasted to contenteditable successfully');
								return true;
							}
						}
						
						console.warn('[BrowserWing] Element type not supported for paste:', activeElement.tagName);
						return false;
						
					} catch (e) {
						console.error('[BrowserWing] Clipboard API failed:', e);
						return false;
					}
				}`)

			if jsErr != nil {
				return fmt.Errorf("all paste methods failed on Mac: %v", jsErr)
			}
			logger.Info(ctx, "✓ Paste successful using navigator.clipboard API")
		}
		// Mac 粘贴处理完成

		// Windows/Linux 使用原有方法
		if runtime.GOOS != "darwin" {
			keyboard := page.Keyboard
			err = keyboard.Press(input.ControlLeft)
			if err != nil {
				return fmt.Errorf("failed to press Ctrl: %w", err)
			}
			err = keyboard.Type(input.KeyV)
			if err != nil {
				keyboard.Release(input.ControlLeft)
				return fmt.Errorf("failed to press V: %w", err)
			}
			err = keyboard.Release(input.ControlLeft)
			if err != nil {
				return fmt.Errorf("failed to release Ctrl: %w", err)
			}
			logger.Info(ctx, "✓ Paste successful using Ctrl+V")
		}

	case "backspace":
		// Backspace 键
		logger.Info(ctx, "Executing Backspace key")
		keyboard := page.Keyboard
		err = keyboard.Type(input.Backspace)
		if err != nil {
			return fmt.Errorf("failed to press Backspace: %w", err)
		}

	case "tab":
		// Tab 键
		logger.Info(ctx, "Executing Tab key")
		keyboard := page.Keyboard
		err = keyboard.Type(input.Tab)
		if err != nil {
			return fmt.Errorf("failed to press Tab: %w", err)
		}

	case "enter":
		// 回车键
		logger.Info(ctx, "Executing Enter key")
		keyboard := page.Keyboard
		err = keyboard.Type(input.Enter)
		if err != nil {
			return fmt.Errorf("failed to press Enter: %w", err)
		}

	default:
		return fmt.Errorf("unsupported keyboard key: %s", key)
	}

	// 等待一下让操作生效
	time.Sleep(300 * time.Millisecond)

	logger.Info(ctx, "✓ Keyboard action completed: %s", key)
	return nil
}

// executeScreenshot 执行截图操作
func (p *Player) executeScreenshot(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	mode := action.ScreenshotMode
	if mode == "" {
		mode = "viewport" // 默认视口截图
	}

	logger.Info(ctx, "Taking screenshot: mode=%s", mode)

	// 等待页面稳定
	time.Sleep(500 * time.Millisecond)

	// 截图前隐藏AI控制指示器，避免被截入图片
	_, _ = page.Eval(`() => {
		const indicator = document.getElementById('browserwing-ai-indicator');
		if (indicator) {
			indicator.style.display = 'none';
		}
	}`)

	var screenshot []byte
	var err error

	switch mode {
	case "viewport":
		// 当前视口截图
		screenshot, err = page.Screenshot(false, nil)
		if err != nil {
			return fmt.Errorf("failed to take viewport screenshot: %w", err)
		}
		logger.Info(ctx, "Viewport screenshot captured")

	case "fullpage":
		// 完整页面截图
		screenshot, err = page.Screenshot(true, nil)
		if err != nil {
			return fmt.Errorf("failed to take full page screenshot: %w", err)
		}
		logger.Info(ctx, "Full page screenshot captured")

	case "region":
		// 区域截图
		if action.ScreenshotWidth <= 0 || action.ScreenshotHeight <= 0 {
			return fmt.Errorf("invalid region dimensions: width=%d, height=%d", action.ScreenshotWidth, action.ScreenshotHeight)
		}

		// 使用 proto 设置截图区域
		screenshot, err = page.Screenshot(false, &proto.PageCaptureScreenshot{
			Clip: &proto.PageViewport{
				X:      float64(action.X),
				Y:      float64(action.Y),
				Width:  float64(action.ScreenshotWidth),
				Height: float64(action.ScreenshotHeight),
				Scale:  1,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to take region screenshot: %w", err)
		}
		logger.Info(ctx, "Region screenshot captured: x=%d, y=%d, w=%d, h=%d",
			action.X, action.Y, action.ScreenshotWidth, action.ScreenshotHeight)

	default:
		return fmt.Errorf("unsupported screenshot mode: %s", mode)
	}

	// 截图完成后恢复显示AI控制指示器
	_, _ = page.Eval(`() => {
		const indicator = document.getElementById('browserwing-ai-indicator');
		if (indicator) {
			indicator.style.display = 'block';
		}
	}`)

	// 确保下载目录存在
	if p.downloadPath == "" {
		return fmt.Errorf("download path not set")
	}

	// 创建下载目录（如果不存在）
	if err := os.MkdirAll(p.downloadPath, 0o755); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	// 生成唯一的文件名
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("browserwing_screenshot_%s_%s.png", mode, timestamp)

	// 如果有自定义变量名，使用它作为文件名前缀
	if action.VariableName != "" {
		// 清理变量名，移除非法字符
		cleanName := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				return r
			}
			return '_'
		}, action.VariableName)
		fileName = fmt.Sprintf("%s_%s_%s.png", cleanName, mode, timestamp)
	}

	// 构建完整路径
	fullPath := filepath.Join(p.downloadPath, fileName)

	// 保存截图到文件
	if err := os.WriteFile(fullPath, screenshot, 0o644); err != nil {
		return fmt.Errorf("failed to save screenshot to file: %w", err)
	}

	// 存储截图数据
	varName := action.VariableName
	if varName == "" {
		varName = fmt.Sprintf("screenshot_%d", len(p.extractedData))
	}

	// 保存为包含元数据的结构
	screenshotData := map[string]interface{}{
		"path":      fullPath,
		"fileName":  fileName,
		"format":    "png",
		"size":      len(screenshot),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	p.extractedData[varName] = screenshotData

	logger.Info(ctx, "✓ Screenshot saved successfully: %s (path: %s, size: %d bytes)", varName, fullPath, len(screenshot))
	return nil
}

// executeOpenTab 执行打开新标签页操作
func (p *Player) executeOpenTab(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	url := action.URL
	if url == "" {
		return fmt.Errorf("open_tab action requires URL")
	}

	logger.Info(ctx, "Opening new tab with URL: %s", url)

	// 获取浏览器实例
	browser := page.Browser()

	// 创建新页面（新标签页）
	newPage, err := browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return fmt.Errorf("failed to create new tab: %w", err)
	}

	// 等待新页面加载
	if err := newPage.WaitLoad(); err != nil {
		logger.Warn(ctx, "Failed to wait for new tab to load: %v", err)
	}

	// 将新页面添加到 pages map
	p.tabCounter++
	tabIndex := p.tabCounter
	p.pages[tabIndex] = newPage

	// 切换到新标签页
	p.currentPage = newPage

	logger.Info(ctx, "✓ New tab opened (tab index: %d): %s", tabIndex, url)

	// 等待页面稳定
	time.Sleep(1 * time.Second)

	return nil
}

func (p *Player) executeSwitchActiveTab(ctx context.Context) error {
	logger.Info(ctx, "Switching to browser's active tab")

	// 如果没有当前页面，无法获取浏览器实例
	if p.currentPage == nil && len(p.pages) == 0 {
		return fmt.Errorf("no pages available to get browser instance")
	}

	// 获取浏览器实例（从当前页面或任意一个页面）
	var browser *rod.Browser
	if p.currentPage != nil {
		browser = p.currentPage.Browser()
	} else {
		// 如果 currentPage 为空，从 pages map 中获取任意一个页面
		for _, pg := range p.pages {
			browser = pg.Browser()
			break
		}
	}

	if browser == nil {
		return fmt.Errorf("failed to get browser instance")
	}

	// 获取所有的标签页
	pages, err := browser.Pages()
	if err != nil {
		return fmt.Errorf("failed to get browser pages: %w", err)
	}

	if len(pages) == 0 {
		return fmt.Errorf("no pages found in browser")
	}

	logger.Info(ctx, "Found %d pages in browser", len(pages))

	// 找到当前活跃的标签页
	// rod 中，活跃的页面可以通过获取其 TargetInfo 来判断
	var activePage *rod.Page
	for _, page := range pages {
		// 获取页面的 TargetInfo
		targetInfo, err := page.Info()
		if err != nil {
			logger.Warn(ctx, "Failed to get page info: %v", err)
			continue
		}

		// 检查页面类型是否为 "page" 且不是后台页面
		if targetInfo.Type == "page" {
			// 尝试检查页面是否是当前活跃的（attached 状态）
			// 注意：在 Chrome DevTools Protocol 中，活跃的页面通常是 attached 状态
			// 我们可以尝试获取页面的可见性状态
			isVisible, visErr := page.Eval(`() => document.visibilityState === 'visible'`)
			if visErr == nil && isVisible != nil && isVisible.Value.Bool() {
				activePage = page
				logger.Info(ctx, "Found active page: %s", targetInfo.URL)
				break
			}
		}
	}

	// 如果没有找到明确的活跃页面，使用第一个可用的页面
	if activePage == nil {
		logger.Warn(ctx, "Could not determine active page, using first available page")
		activePage = pages[0]
	}

	// 将找到的活跃页面设置为当前页面
	p.currentPage = activePage

	// 同步到 pages map 中（如果该页面不在 map 中，则添加）
	pageFound := false
	for idx, pg := range p.pages {
		if pg == activePage {
			pageFound = true
			logger.Info(ctx, "Active page found in pages map at index: %d", idx)
			break
		}
	}

	if !pageFound {
		// 如果活跃页面不在 pages map 中，添加它
		p.tabCounter++
		p.pages[p.tabCounter] = activePage
		logger.Info(ctx, "Added active page to pages map with index: %d", p.tabCounter)
	}

	// 激活该页面（确保浏览器窗口也切换到该标签页）
	_, err = activePage.Activate()
	if err != nil {
		logger.Warn(ctx, "Failed to activate page: %v", err)
	}

	logger.Info(ctx, "✓ Switched to browser's active tab")
	time.Sleep(500 * time.Millisecond)

	return nil
}

// executeSwitchTab 执行切换标签页操作
func (p *Player) executeSwitchTab(ctx context.Context, action models.ScriptAction) error {
	// 可以通过 action.Value 传递标签页索引
	// 例如 "0" 表示第一个标签页，"1" 表示第二个标签页
	tabIndexStr := action.Value
	if tabIndexStr == "" {
		return fmt.Errorf("switch_tab action requires tab index in value field")
	}

	var tabIndex int
	_, err := fmt.Sscanf(tabIndexStr, "%d", &tabIndex)
	if err != nil {
		return fmt.Errorf("invalid tab index: %s", tabIndexStr)
	}

	targetPage, exists := p.pages[tabIndex]
	if !exists {
		return fmt.Errorf("tab index %d does not exist", tabIndex)
	}

	logger.Info(ctx, "Switching to tab index: %d", tabIndex)
	p.currentPage = targetPage

	// 激活目标页面
	_, err = targetPage.Activate()
	if err != nil {
		logger.Warn(ctx, "Failed to activate tab: %v", err)
	}

	logger.Info(ctx, "✓ Switched to tab %d", tabIndex)
	time.Sleep(500 * time.Millisecond)

	return nil
}

// injectXHRInterceptorForScript 在脚本开始执行时注入XHR拦截器，提前监听所有需要捕获的请求
func (p *Player) injectXHRInterceptorForScript(ctx context.Context, page *rod.Page, actions []models.ScriptAction) error {
	// 收集所有需要监听的 capture_xhr action
	var captureTargets []map[string]string
	for _, action := range actions {
		if action.Type == "capture_xhr" && action.URL != "" && action.Method != "" {
			captureTargets = append(captureTargets, map[string]string{
				"method": action.Method,
				"url":    action.URL,
			})
		}
	}

	// 如果没有 capture_xhr action，不需要注入
	if len(captureTargets) == 0 {
		logger.Info(ctx, "No capture_xhr actions found, skipping XHR interceptor injection")
		return nil
	}

	logger.Info(ctx, "Injecting XHR interceptor for %d capture targets", len(captureTargets))

	// 将目标列表转换为JSON
	targetsJSON, err := json.Marshal(captureTargets)
	if err != nil {
		return fmt.Errorf("failed to marshal capture targets: %w", err)
	}

	// 注入拦截器脚本
	_, err = page.Eval(`(captureTargetsJSON) => {
		if (window.__xhrCaptureInstalled__) {
			console.log('[BrowserWing Player] XHR interceptor already installed');
			return true;
		}
		
		window.__xhrCaptureInstalled__ = true;
		window.__capturedXHRData__ = {};
		
		// 解析需要监听的目标列表
		var captureTargets = JSON.parse(captureTargetsJSON);
		var targetKeys = new Set();
		captureTargets.forEach(function(target) {
			var key = target.method + '|' + target.url;
			targetKeys.add(key);
			console.log('[BrowserWing Player] Will capture:', key);
		});
		
		// 辅助函数：提取域名+路径（不带参数）
		var extractDomainAndPath = function(url) {
			try {
				var fullUrl = url;
				
				// 处理 // 开头的协议相对URL（如 //cdn.example.com/api）
				if (url.indexOf('//') === 0) {
					fullUrl = window.location.protocol + url;
				}
				// 处理相对路径（不包含域名的路径）
				else if (url.indexOf('http') !== 0 && url.indexOf('//') !== 0) {
					// 拼接当前页面的origin
					if (url.startsWith('/')) {
						fullUrl = window.location.origin + url;
					} else {
						fullUrl = window.location.origin + '/' + url;
					}
				}
				
				var urlObj = new URL(fullUrl);
				// 返回 域名+路径（不带参数和hash）
				return urlObj.origin + urlObj.pathname;
			} catch (e) {
				console.warn('[BrowserWing Player] Failed to parse URL:', url, e);
				return url.split('?')[0].split('#')[0];
			}
		};
		
		// 拦截XMLHttpRequest
		var originalXHROpen = XMLHttpRequest.prototype.open;
		var originalXHRSend = XMLHttpRequest.prototype.send;
		
		XMLHttpRequest.prototype.open = function(method, url) {
			this.__xhrInfo__ = {
				method: method,
				url: url,
				domainAndPath: extractDomainAndPath(url)
			};
			return originalXHROpen.apply(this, arguments);
		};
		
		XMLHttpRequest.prototype.send = function(body) {
			var xhr = this;
			var xhrInfo = xhr.__xhrInfo__;
			
			if (xhrInfo) {
				xhr.addEventListener('readystatechange', function() {
					if (xhr.readyState === 4) {
						var key = xhrInfo.method + '|' + xhrInfo.domainAndPath;
						
						// 只存储我们需要监听的请求
						if (!targetKeys.has(key)) {
							return;
						}
						
						var response = null;
						
						try {
							if (xhr.responseType === '' || xhr.responseType === 'text') {
								response = xhr.responseText;
							} else if (xhr.responseType === 'json') {
								response = xhr.response;
							} else {
								response = '[Binary Data]';
							}
						} catch (e) {
							response = '[Error reading response]';
						}
						
						window.__capturedXHRData__[key] = {
							method: xhrInfo.method,
							url: xhrInfo.domainAndPath,
							status: xhr.status,
							statusText: xhr.statusText,
							response: response,
							timestamp: Date.now()
						};
						console.log('[BrowserWing Player] Captured XHR:', key, 'Status:', xhr.status);
					}
				});
			}
			
			return originalXHRSend.apply(this, arguments);
		};
		
		// 拦截Fetch API
		var originalFetch = window.fetch;
		window.fetch = function(input, init) {
			var url = typeof input === 'string' ? input : input.url;
			var method = (init && init.method) || 'GET';
			var domainAndPath = extractDomainAndPath(url);
			var key = method.toUpperCase() + '|' + domainAndPath;
			
			// 只拦截我们需要监听的请求
			if (!targetKeys.has(key)) {
				return originalFetch.apply(this, arguments);
			}
			
			return originalFetch.apply(this, arguments).then(function(response) {
				var clonedResponse = response.clone();
				var contentType = response.headers.get('content-type') || '';
				
				if (contentType.indexOf('application/json') !== -1) {
					clonedResponse.json().then(function(data) {
						window.__capturedXHRData__[key] = {
							method: method.toUpperCase(),
							url: domainAndPath,
							status: response.status,
							statusText: response.statusText,
							response: data,
							timestamp: Date.now()
						};
						console.log('[BrowserWing Player] Captured Fetch:', key);
					}).catch(function(e) {
						console.warn('[BrowserWing Player] Failed to parse Fetch response:', e);
					});
				} else if (contentType.indexOf('text/') !== -1) {
					clonedResponse.text().then(function(text) {
						window.__capturedXHRData__[key] = {
							method: method.toUpperCase(),
							url: domainAndPath,
							status: response.status,
							statusText: response.statusText,
							response: text,
							timestamp: Date.now()
						};
						console.log('[BrowserWing Player] Captured Fetch:', key);
					}).catch(function(e) {
						console.warn('[BrowserWing Player] Failed to read Fetch response:', e);
					});
				}
				
				return response;
			});
		};
		
		console.log('[BrowserWing Player] XHR capture script installed, monitoring', targetKeys.size, 'targets');
		return true;
	}`, string(targetsJSON))
	if err != nil {
		return fmt.Errorf("failed to inject XHR interceptor: %w", err)
	}

	logger.Info(ctx, "✓ XHR interceptor injected successfully")
	return nil
}

// executeCaptureXHR 执行捕获XHR请求操作（回放时等待并获取匹配的XHR响应数据）
func (p *Player) executeCaptureXHR(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	domainAndPath := action.URL
	method := action.Method
	if domainAndPath == "" || method == "" {
		return fmt.Errorf("capture_xhr action requires url and method")
	}

	logger.Info(ctx, "Capturing XHR request: %s %s (domain+path, ignoring query params)", method, domainAndPath)

	// XHR拦截器已经在脚本开始时注入（injectXHRInterceptorForScript），
	// 这里只需要等待和查找数据，不再重复注入

	// 等待指定的XHR请求完成（轮询检查）
	// 使用method + domainAndPath（不带参数）作为匹配key
	key := method + "|" + domainAndPath
	maxWaitTime := 30 * time.Second
	pollInterval := 500 * time.Millisecond
	startTime := time.Now()

	logger.Info(ctx, "Waiting for XHR request to complete (domain+path matching): %s", key)

	for {
		// 检查是否超时
		if time.Since(startTime) > maxWaitTime {
			return fmt.Errorf("timeout waiting for XHR request: %s %s", method, domainAndPath)
		}

		// 检查是否已捕获到目标请求
		result, err := page.Eval(`(key) => {
			if (window.__capturedXHRData__ && window.__capturedXHRData__[key]) {
				return window.__capturedXHRData__[key];
			}
			return null;
		}`, key)
		if err != nil {
			logger.Warn(ctx, "Failed to check captured XHR data: %v", err)
			time.Sleep(pollInterval)
			continue
		}

		// 如果找到了匹配的请求
		if result != nil && !result.Value.Nil() {
			// 解析响应数据
			var xhrData map[string]interface{}
			jsonData, _ := json.Marshal(result.Value)
			if err := json.Unmarshal(jsonData, &xhrData); err != nil {
				logger.Warn(ctx, "Failed to parse XHR data: %v", err)
				return fmt.Errorf("failed to parse XHR response: %w", err)
			}

			// 存储抓取的数据
			varName := action.VariableName
			if varName == "" {
				varName = fmt.Sprintf("xhr_data_%d", len(p.extractedData))
			}
			p.extractedData[varName] = xhrData["response"]

			logger.Info(ctx, "✓ XHR request captured successfully: %s = %v", varName, xhrData["status"])
			logger.Info(ctx, "Response status: %v %v", xhrData["status"], xhrData["statusText"])
			return nil
		}

		// 等待后重试
		time.Sleep(pollInterval)
	}
}

// executeAIControl 执行 AI 控制动作
// 使用录制时保存的提示词和可选的元素 XPath 来启动一个 AI Agent
// Agent 会根据提示词自动操作当前页面
func (p *Player) executeAIControl(ctx context.Context, page *rod.Page, action models.ScriptAction) error {
	if p.agentManager == nil {
		return fmt.Errorf("agent manager is not available")
	}

	userTask := action.AIControlPrompt
	if userTask == "" {
		return fmt.Errorf("AI control prompt is empty")
	}

	logger.Info(ctx, "[executeAIControl] Executing AI control action with user task: %s", userTask)

	// 关键修复：同步当前页面到 Browser Manager 的 activePage
	// 这样 Executor 的 GetActivePage() 才能获取到正确的页面
	if p.browserManager != nil {
		logger.Info(ctx, "[executeAIControl] Syncing current page to Browser Manager's activePage")
		p.browserManager.SetActivePage(page)

		// 验证设置是否成功
		if activePage := p.browserManager.GetActivePage(); activePage == page {
			logger.Info(ctx, "[executeAIControl] ✓ Successfully synced activePage to Browser Manager")
		} else {
			logger.Warn(ctx, "[executeAIControl] ⚠️  Failed to sync activePage - Executor tools may not work correctly")
		}
	} else {
		logger.Warn(ctx, "[executeAIControl] ⚠️  Browser Manager not set - Executor tools may not work correctly")
	}

	// 获取当前页面信息
	var currentURL string
	pageInfo, err := page.Info()
	if err == nil && pageInfo != nil {
		currentURL = pageInfo.URL
		logger.Info(ctx, "[executeAIControl] Current page URL: %s, Title: %s", currentURL, pageInfo.Title)
	} else {
		currentURL = "unknown"
		logger.Warn(ctx, "[executeAIControl] Failed to get page info: %v", err)
	}

	// 构建完整的提示词，包含上下文信息
	var promptBuilder strings.Builder

	// 1. 页面上下文
	promptBuilder.WriteString(fmt.Sprintf("Current active browser page URL: %s\n\n", currentURL))

	// 2. 简要说明（工具详情会自动在上下文中）
	promptBuilder.WriteString("You have access to browser automation tools for interacting with the page.\n")
	promptBuilder.WriteString("Use the available tools to help the user complete the following task:\n\n")

	// 3. 用户任务
	promptBuilder.WriteString(userTask)

	// 4. 如果有元素 XPath，添加到上下文
	if action.AIControlXPath != "" {
		logger.Info(ctx, "AI control target element XPath: %s", action.AIControlXPath)
		// XPath信息已经包含在userTask中（格式为 "任务描述 (xpath: xxx)"），无需重复添加
	}

	prompt := promptBuilder.String()
	logger.Debug(ctx, "Full AI control prompt: %s", prompt)

	// 创建一个唯一的会话ID用于这次 AI 控制
	// SendMessage 会自动创建会话（如果不存在）
	sessionID := fmt.Sprintf("ai_control_%d", time.Now().UnixNano())

	// 创建流式输出通道（使用 any 类型以匹配接口）
	streamChan := make(chan any, 100)

	// 完成标志和错误通道
	doneChan := make(chan error, 1)
	hasContent := false

	// 创建一个带超时的 context 用于 AI 控制执行
	// 5分钟超时，足够完成大多数自动化任务
	aiCtx, aiCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer aiCancel()

	// 在后台接收并记录流式输出
	go func() {
		for chunk := range streamChan {
			// 尝试将 chunk 转换为 map 来处理
			// 由于 AgentManager 实际发送的是 StreamChunk 结构体
			// 我们需要使用类型断言或反射来处理
			switch v := chunk.(type) {
			case map[string]interface{}:
				if chunkType, ok := v["type"].(string); ok {
					if chunkType == "content" || chunkType == "message" {
						if content, ok := v["content"].(string); ok && content != "" {
							hasContent = true
							logger.Info(ctx, "AI control output: %s", content)
						}
					} else if chunkType == "error" {
						if errMsg, ok := v["error"].(string); ok {
							logger.Error(ctx, "AI control error: %s", errMsg)
						}
					} else if chunkType == "done" || chunkType == "complete" {
						logger.Info(ctx, "AI control stream completed")
					}
				}
			default:
				// 对于其他类型，简单记录
				logger.Debug(ctx, "AI control stream: %v", v)
			}
		}
	}()

	// 在后台执行 AI 控制
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(ctx, "AI control panic recovered: %v", r)
				doneChan <- fmt.Errorf("AI control panic: %v", r)
			}
			close(doneChan)
		}()

		logger.Info(ctx, "Starting AI control task execution with session: %s", sessionID)

		// 使用带超时的 context 调用 SendMessageInterface
		// 这样可以确保即使工具调用卡住，也能在超时后返回
		err := p.agentManager.SendMessageInterface(aiCtx, sessionID, prompt, streamChan)

		if err != nil {
			logger.Error(ctx, "AI control task execution error: %v", err)
		} else {
			logger.Info(ctx, "AI control task execution completed without error")
		}

		doneChan <- err
	}()

	// 等待执行完成或超时
	select {
	case err, ok := <-doneChan:
		// 注意：不要在这里关闭 streamChan！
		// streamChan 会由 adapter.go 的 goroutine 通过 defer 自动关闭
		time.Sleep(100 * time.Millisecond) // 等待流处理完成

		if !ok {
			// doneChan 被关闭但没有收到错误，可能是 panic 后恢复
			logger.Error(ctx, "AI control execution interrupted unexpectedly")
			return fmt.Errorf("AI control execution interrupted")
		}

		if err != nil {
			logger.Error(ctx, "AI control execution failed: %v", err)
			return fmt.Errorf("AI control failed: %w", err)
		}

		if hasContent {
			logger.Info(ctx, "✓ AI control completed successfully")
			return nil
		}

		// 即使没有内容输出，只要没有错误就认为成功
		logger.Info(ctx, "✓ AI control completed (no visible output)")
		return nil

	case <-aiCtx.Done():
		// Context 超时或取消
		// 注意：不要在这里关闭 streamChan！
		// streamChan 会由 adapter.go 的 goroutine 通过 defer 自动关闭
		logger.Warn(ctx, "AI control context done, waiting for cleanup...")
		time.Sleep(100 * time.Millisecond) // 等待流处理完成

		if aiCtx.Err() == context.DeadlineExceeded {
			logger.Error(ctx, "❌ AI control execution timeout after 5 minutes - task took too long")
			return fmt.Errorf("AI control execution timeout (5 minutes)")
		}

		logger.Error(ctx, "❌ AI control cancelled: %v", aiCtx.Err())
		return fmt.Errorf("AI control cancelled: %w", aiCtx.Err())
	}
}
