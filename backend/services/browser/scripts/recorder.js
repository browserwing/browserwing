// 防止重复注入
if (window.__browserwingRecorder__) {
	console.log('[BrowserWing] Recorder already initialized');
} else {
	window.__browserwingRecorder__ = true;
	window.__recordedActions__ = [];
	window.__lastInputTime__ = {};
	window.__inputTimers__ = {};
	window.__extractMode__ = false; // 数据抓取模式标志
	window.__extractType__ = 'text'; // 数据抓取类型: 'text', 'html', 'attribute'
	window.__menuTrigger__ = null; // 菜单触发方式: 'button' 或 'contextmenu'
	window.__aiExtractMode__ = false; // AI提取模式标志
	window.__aiFormFillMode__ = false; // AI填充表单模式标志
	window.__recorderUI__ = null; // 录制器 UI 元素
	window.__highlightElement__ = null; // 高亮元素
	window.__highlightLabel__ = null; // 高亮标签元素
	window.__selectedElement__ = null; // AI模式选中的元素
	window.__recordingFloatButton__ = null; // 浮动录制按钮
	window.__isRecordingActive__ = false; // 录制是否激活
	
	// ============= 录制器 UI 相关函数 =============
	
	// 创建统一的录制器控制面板
	var createRecorderUI = function() {
		// 如果已经创建过，直接返回
		if (window.__recorderUI__) {
			console.log('[BrowserWing] Recorder UI already exists');
			return;
		}
		
		// 创建主容器
		var panel = document.createElement('div');
		panel.id = '__browserwing_recorder_panel__';
	panel.style.cssText = 'position:fixed;top:20px;right:20px;z-index:999999;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","SF Pro Display",Helvetica,Arial,sans-serif;width:360px;background:linear-gradient(135deg, #ffffff 0%, #fafbfc 100%);border-radius:16px;box-shadow:0 8px 32px rgba(0,0,0,0.08), 0 2px 8px rgba(0,0,0,0.04);border:1px solid rgba(0,0,0,0.06);overflow:hidden;backdrop-filter:blur(10px);';	// 创建头部区域（可拖动）
	var header = document.createElement('div');
	header.style.cssText = 'padding:18px 20px 16px;background:transparent;cursor:move;user-select:none;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid rgba(0,0,0,0.05);';		var headerLeft = document.createElement('div');
		headerLeft.style.cssText = 'display:flex;align-items:center;gap:10px;';
		
	var statusDot = document.createElement('div');
	statusDot.id = '__browserwing_status_dot__';
	statusDot.style.cssText = 'width:10px;height:10px;border-radius:50%;background:#ef4444;animation:pulse 2s cubic-bezier(0.4,0,0.6,1) infinite;box-shadow:0 0 8px rgba(239,68,68,0.4);';		var statusText = document.createElement('div');
		statusText.id = '__browserwing_status_text__';
		statusText.style.cssText = 'color:#0f172a;font-size:15px;font-weight:600;letter-spacing:-0.01em;';
		statusText.textContent = '{{RECORDING_STATUS}}';
		
		headerLeft.appendChild(statusDot);
		headerLeft.appendChild(statusText);
		
		var actionCount = document.createElement('div');
		actionCount.id = '__browserwing_action_count__';
		actionCount.style.cssText = 'color:#64748b;font-size:13px;font-weight:600;letter-spacing:-0.01em;background:rgba(100,116,139,0.08);padding:4px 10px;border-radius:8px;';
			header.appendChild(actionCount);
			
		// 创建按钮区域
		var buttonArea = document.createElement('div');
		buttonArea.style.cssText = 'padding:16px 20px;display:flex;gap:10px;border-bottom:1px solid rgba(0,0,0,0.05);background:transparent;';
		
		var extractBtn = document.createElement('button');
		extractBtn.id = '__browserwing_extract_btn__';
		extractBtn.style.cssText = 'flex:1;padding:8px 14px;background:white;color:#334155;border:1.5px solid rgba(0,0,0,0.12);border-radius:10px;cursor:pointer;font-size:13px;font-weight:600;letter-spacing:-0.01em;transition:all 0.25s cubic-bezier(0.4,0,0.2,1);box-shadow:0 1px 3px rgba(0,0,0,0.04);';
		extractBtn.textContent = '{{DATA_EXTRACT}}';
		extractBtn.onmouseover = function() {
			if (!window.__extractMode__) {
				this.style.background = '#f8fafc';
				this.style.borderColor = 'rgba(0,0,0,0.18)';
				this.style.transform = 'translateY(-1px)';
				this.style.boxShadow = '0 2px 8px rgba(0,0,0,0.08)';
			}
		};
		extractBtn.onmouseout = function() {
			if (!window.__extractMode__) {
				this.style.background = 'white';
				this.style.borderColor = 'rgba(0,0,0,0.12)';
				this.style.transform = 'translateY(0)';
				this.style.boxShadow = '0 1px 3px rgba(0,0,0,0.04)';
			}
		};
		extractBtn.onclick = function(e) {
			if (panel.__isDragging) return;
			
			if (window.__extractMode__) {
				// 如果已经在抓取模式，退出模式
				toggleExtractMode();
			} else {
				// 显示抓取类型选择菜单
				window.__menuTrigger__ = 'button';
				var menu = window.__recorderUI__.menu;
				menu.style.display = 'block';
				
				// 计算菜单位置（在按钮下方）
				var btnRect = this.getBoundingClientRect();
				menu.style.left = btnRect.left + 'px';
				menu.style.top = (btnRect.bottom + 5) + 'px';
			}
		};
		
		var aiExtractBtn = document.createElement('button');
		aiExtractBtn.id = '__browserwing_ai_extract_btn__';
		aiExtractBtn.style.cssText = 'flex:1;padding:8px 14px;background:linear-gradient(135deg,#1e293b 0%,#0f172a 100%);color:white;border:1.5px solid rgba(0,0,0,0.1);border-radius:10px;cursor:pointer;font-size:13px;font-weight:600;letter-spacing:-0.01em;transition:all 0.25s cubic-bezier(0.4,0,0.2,1);box-shadow:0 2px 8px rgba(15,23,42,0.2), 0 1px 3px rgba(0,0,0,0.1);';
		aiExtractBtn.textContent = '{{AI_EXTRACT}}';
		aiExtractBtn.onmouseover = function() {
			this.style.background = 'linear-gradient(135deg,#0f172a 0%,#020617 100%)';
			this.style.transform = 'translateY(-1px)';
			this.style.boxShadow = '0 4px 12px rgba(15,23,42,0.3), 0 2px 4px rgba(0,0,0,0.15)';
		};
		aiExtractBtn.onmouseout = function() {
			this.style.background = 'linear-gradient(135deg,#1e293b 0%,#0f172a 100%)';
			this.style.transform = 'translateY(0)';
			this.style.boxShadow = '0 2px 8px rgba(15,23,42,0.2), 0 1px 3px rgba(0,0,0,0.1)';
		};
		aiExtractBtn.onclick = function() {
			if (!panel.__isDragging) {
				toggleAIExtractMode();
			}
		};
		
		var aiFormFillBtn = document.createElement('button');
		aiFormFillBtn.id = '__browserwing_ai_formfill_btn__';
		aiFormFillBtn.style.cssText = 'flex:1;padding:8px 14px;background:linear-gradient(135deg,#64748b 0%,#475569 100%);color:white;border:1.5px solid rgba(0,0,0,0.1);border-radius:10px;cursor:pointer;font-size:13px;font-weight:600;letter-spacing:-0.01em;transition:all 0.25s cubic-bezier(0.4,0,0.2,1);box-shadow:0 2px 8px rgba(71,85,105,0.2), 0 1px 3px rgba(0,0,0,0.1);';
		aiFormFillBtn.textContent = '{{AI_FORMFILL}}';
		aiFormFillBtn.onmouseover = function() {
			this.style.background = 'linear-gradient(135deg,#475569 0%,#334155 100%)';
			this.style.transform = 'translateY(-1px)';
			this.style.boxShadow = '0 4px 12px rgba(71,85,105,0.3), 0 2px 4px rgba(0,0,0,0.15)';
		};
		aiFormFillBtn.onmouseout = function() {
			this.style.background = 'linear-gradient(135deg,#64748b 0%,#475569 100%)';
			this.style.transform = 'translateY(0)';
			this.style.boxShadow = '0 2px 8px rgba(71,85,105,0.2), 0 1px 3px rgba(0,0,0,0.1)';
		};
		aiFormFillBtn.onclick = function() {
			if (!panel.__isDragging) {
				toggleAIFormFillMode();
			}
		};
		
		buttonArea.appendChild(extractBtn);
		buttonArea.appendChild(aiExtractBtn);
		buttonArea.appendChild(aiFormFillBtn);		// 创建动作列表区域

		var actionList = document.createElement('div');
		actionList.id = '__browserwing_action_list__';
		actionList.style.cssText = 'max-height:280px;overflow-y:auto;padding:12px;background:transparent;';
		
	var emptyState = document.createElement('div');
	emptyState.id = '__browserwing_empty_state__';
	emptyState.style.cssText = 'padding:32px 24px;text-align:center;color:#94a3b8;font-size:13px;font-weight:500;letter-spacing:-0.01em;';
	emptyState.textContent = '{{EMPTY_STEPS}}';
	
	actionList.appendChild(emptyState);		// 创建当前操作提示区域
		var currentAction = document.createElement('div');
		currentAction.id = '__browserwing_current_action__';
		currentAction.style.cssText = 'display:none;padding:14px 20px;background:rgba(248,250,252,0.8);border-top:1px solid rgba(0,0,0,0.05);color:#475569;font-size:12px;font-weight:500;line-height:1.5;letter-spacing:-0.01em;';
		
		// 添加拖动功能
		var isDragging = false;
		var currentX = 0;
		var currentY = 0;
		var initialX;
		var initialY;
		var xOffset = 0;
		var yOffset = 0;
		
		header.addEventListener('mousedown', function(e) {
			initialX = e.clientX - xOffset;
			initialY = e.clientY - yOffset;
			isDragging = true;
			panel.__isDragging = false;
		});
		
		document.addEventListener('mousemove', function(e) {
			if (isDragging) {
				e.preventDefault();
				currentX = e.clientX - initialX;
				currentY = e.clientY - initialY;
				xOffset = currentX;
				yOffset = currentY;
				
				if (Math.abs(currentX) > 3 || Math.abs(currentY) > 3) {
					panel.__isDragging = true;
				}
				
				panel.style.transform = 'translate(' + currentX + 'px, ' + currentY + 'px)';
			}
		});
		
		document.addEventListener('mouseup', function() {
			if (isDragging) {
				setTimeout(function() {
					panel.__isDragging = false;
				}, 100);
			}
			isDragging = false;
		});
		
		// 组装面板
		panel.appendChild(header);
		panel.appendChild(buttonArea);
		panel.appendChild(actionList);
		panel.appendChild(currentAction);
		
		// 创建结束录制按钮区域
		var stopRecordingArea = document.createElement('div');
		stopRecordingArea.id = '__browserwing_stop_recording_area__';
		stopRecordingArea.style.cssText = 'padding:16px 20px 20px;background:transparent;border-top:1px solid rgba(0,0,0,0.05);';
		
	var stopRecordingBtn = document.createElement('button');
	stopRecordingBtn.id = '__browserwing_stop_recording_btn__';
	stopRecordingBtn.style.cssText = 'width:100%;padding:14px 20px;background:linear-gradient(135deg,#ef4444 0%,#dc2626 100%);color:white;border:none;border-radius:12px;cursor:pointer;font-size:14px;font-weight:600;letter-spacing:-0.01em;transition:all 0.25s cubic-bezier(0.4,0,0.2,1);box-shadow:0 4px 12px rgba(239,68,68,0.25), 0 2px 4px rgba(0,0,0,0.1);';
	stopRecordingBtn.textContent = '{{STOP_RECORDING}}';
	stopRecordingBtn.onmouseover = function() {
		this.style.background = 'linear-gradient(135deg,#dc2626 0%,#b91c1c 100%)'; this.style.transform = 'translateY(-2px)'; this.style.boxShadow = '0 6px 20px rgba(239,68,68,0.35), 0 4px 8px rgba(0,0,0,0.15)';
	};
	stopRecordingBtn.onmouseout = function() {
		this.style.background = 'linear-gradient(135deg,#ef4444 0%,#dc2626 100%)'; this.style.transform = 'translateY(0)'; this.style.boxShadow = '0 4px 12px rgba(239,68,68,0.25), 0 2px 4px rgba(0,0,0,0.1)';
	};
	stopRecordingBtn.onclick = async function() {
		// 使用轮询方式通知后端停止录制,而不是直接调用API
		window.__stopRecordingRequest__ = {
			timestamp: Date.now(),
			action: 'stop'
		};
		console.log('[BrowserWing] Recording stop request set');
		
		// 禁用按钮,防止重复点击
		this.disabled = true;
		this.textContent = '{{STOPPING}}';
		this.style.background = '#9ca3af';
	};		stopRecordingArea.appendChild(stopRecordingBtn);
		panel.appendChild(stopRecordingArea);
		
		// 创建抓取类型选择菜单
		var menu = document.createElement('div');
		menu.id = '__browserwing_extract_menu__';
		menu.style.cssText = 'display:none;position:fixed;background:white;border:1px solid rgba(0,0,0,0.08);border-radius:12px;box-shadow:0 8px 24px rgba(0,0,0,0.12), 0 2px 8px rgba(0,0,0,0.04);z-index:1000000;padding:6px;min-width:180px;backdrop-filter:blur(10px);';
		
		var menuItems = [
			{type: 'text', label: '{{EXTRACT_TEXT}}'},
			{type: 'html', label: '{{EXTRACT_HTML}}'},
			{type: 'attribute', label: '{{EXTRACT_ATTRIBUTE}}'}
		];
		
		for (var i = 0; i < menuItems.length; i++) {
			var item = document.createElement('div');
			item.setAttribute('data-type', menuItems[i].type);
			item.style.cssText = 'padding:10px 14px;cursor:pointer;font-size:13px;font-weight:600;border-radius:8px;color:#334155;letter-spacing:-0.01em;transition:all 0.2s cubic-bezier(0.4,0,0.2,1);';
			item.textContent = menuItems[i].label;
			item.onmouseover = function() { this.style.background = '#f1f5f9'; this.style.color = '#0f172a'; };
			item.onmouseout = function() { this.style.background = 'transparent'; this.style.color = '#334155'; };
			
			// 统一的菜单项点击处理
			item.onclick = function() {
				var extractType = this.getAttribute('data-type');
				var ui = window.__recorderUI__;
				
				if (window.__menuTrigger__ === 'button') {
					// 从按钮点击显示的菜单：设置抓取类型并进入模式
					window.__extractType__ = extractType;
					menu.style.display = 'none';
					
					// 进入抓取模式
					if (!window.__extractMode__) {
						toggleExtractMode();
					}
					
					// 更新提示信息
					var modeText = '{{EXTRACT_MODE_ENABLED}}';
					if (extractType === 'text') {
						modeText = '{{EXTRACT_TEXT_MODE}}';
					} else if (extractType === 'html') {
						modeText = '{{EXTRACT_HTML_MODE}}';
					} else if (extractType === 'attribute') {
						modeText = '{{EXTRACT_ATTRIBUTE_MODE}}';
					}
					showCurrentAction(modeText);
				} else if (window.__menuTrigger__ === 'contextmenu') {
					// 从右键显示的菜单：直接抓取当前元素
					if (extractType === 'attribute') {
						// 弹出对话框让用户输入属性名
						var attrName = prompt('{{PROMPT_ATTRIBUTE}}', 'href');
						if (attrName) {
							recordExtractAction(ui.currentElement, extractType, attrName);
						}
					} else {
						recordExtractAction(ui.currentElement, extractType, null);
					}
					menu.style.display = 'none';
				}
			};
			
			menu.appendChild(item);
		}
		
		// 添加 CSS 动画
		var style = document.createElement('style');
		style.textContent = '@keyframes pulse{0%,100%{opacity:1}50%{opacity:0.5}}';
		document.head.appendChild(style);
		
		document.body.appendChild(panel);
		document.body.appendChild(menu);
		
		window.__recorderUI__ = {
			panel: panel,
			header: header,
			statusDot: statusDot,
			statusText: statusText,
			actionCount: actionCount,
			extractBtn: extractBtn,
			aiExtractBtn: aiExtractBtn,
			aiFormFillBtn: aiFormFillBtn,
			actionList: actionList,
			emptyState: emptyState,
			currentAction: currentAction,
			menu: menu,
			stopRecordingBtn: stopRecordingBtn
		};
	};
	
	// 创建高亮元素
	var createHighlightElement = function() {
		if (window.__highlightElement__) return;
		
		var highlight = document.createElement('div');
		highlight.id = '__browserwing_highlight__';
		highlight.style.cssText = 'position:absolute;pointer-events:none;z-index:999998;border:2px solid #374151;border-radius:4px;box-shadow:0 0 0 2px rgba(55,65,81,0.1);transition:all 0.15s cubic-bezier(0.4,0,0.2,1);display:none;';
		document.body.appendChild(highlight);
		window.__highlightElement__ = highlight;
		
		// 创建信息标签
		var infoLabel = document.createElement('div');
		infoLabel.id = '__browserwing_highlight_label__';
		infoLabel.style.cssText = 'position:absolute;pointer-events:none;z-index:999999;background:linear-gradient(135deg,#1e293b 0%,#0f172a 100%);color:white;padding:6px 12px;border-radius:6px;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;font-size:12px;font-weight:600;box-shadow:0 4px 12px rgba(0,0,0,0.3);display:none;max-width:500px;line-height:1.4;backdrop-filter:blur(10px);border:1px solid rgba(255,255,255,0.1);';
		document.body.appendChild(infoLabel);
		window.__highlightLabel__ = infoLabel;
	};
	
	// 高亮元素
	var highlightElement = function(element) {
		if (!element || !window.__highlightElement__) return;
		
		var rect = element.getBoundingClientRect();
		var highlight = window.__highlightElement__;
		var label = window.__highlightLabel__;
		
		highlight.style.display = 'block';
		highlight.style.left = (rect.left + window.scrollX - 3) + 'px';
		highlight.style.top = (rect.top + window.scrollY - 3) + 'px';
		highlight.style.width = (rect.width + 6) + 'px';
		highlight.style.height = (rect.height + 6) + 'px';
		
		// 抓取模式下使用不同颜色
		if (window.__extractMode__) {
			highlight.style.borderColor = '#82dee4ff';
			highlight.style.boxShadow = '0 0 0 2px rgba(119, 192, 252, 0.39)';
		} else {
			highlight.style.borderColor = '#82dee4ff';
			highlight.style.boxShadow = '0 0 0 2px rgba(119, 192, 252, 0.39)';
		}
		
		// 更新信息标签
		if (label) {
			// 获取元素信息
			var tagName = element.tagName ? element.tagName.toLowerCase() : 'unknown';
			var selectors = getSelector(element);
			var xpath = selectors.xpath || '//unknown';
			
			// 截断过长的 XPath
			var displayXPath = xpath;
			if (xpath.length > 80) {
				displayXPath = xpath.substring(0, 77) + '...';
			}
			
			// 构建标签内容
			var labelHTML = '<div style="margin-bottom:2px;"><span style="color:#60a5fa;font-weight:700;">&lt;' + escapeHtml(tagName) + '&gt;</span></div>';
			labelHTML += '<div style="color:#94a3b8;font-size:11px;word-break:break-all;">' + escapeHtml(displayXPath) + '</div>';
			
			label.innerHTML = labelHTML;
			label.style.display = 'block';
			
			// 计算标签位置（左上角，向上偏移）
			var labelLeft = rect.left + window.scrollX - 3;
			var labelTop = rect.top + window.scrollY - 3;
			
			// 确保标签不会超出视口顶部
			if (labelTop < window.scrollY + 10) {
				labelTop = rect.bottom + window.scrollY + 5; // 如果上方空间不够，放到元素下方
			} else {
				// 获取标签高度后再调整位置
				label.style.left = labelLeft + 'px';
				label.style.top = labelTop + 'px';
				var labelHeight = label.offsetHeight;
				labelTop = labelTop - labelHeight - 8; // 向上偏移
			}
			
			label.style.left = labelLeft + 'px';
			label.style.top = labelTop + 'px';
			
			// 确保标签不会超出视口右侧
			var labelRight = labelLeft + label.offsetWidth;
			var viewportRight = window.scrollX + window.innerWidth;
			if (labelRight > viewportRight) {
				labelLeft = viewportRight - label.offsetWidth - 10;
				label.style.left = labelLeft + 'px';
			}
		}
	};
	
	// 隐藏高亮
	var hideHighlight = function() {
		if (window.__highlightElement__) {
			window.__highlightElement__.style.display = 'none';
		}
		if (window.__highlightLabel__) {
			window.__highlightLabel__.style.display = 'none';
		}
	};
	
	// 创建全屏 Loading 遮罩
	var showFullPageLoading = function(message) {
		// 如果已经存在，先移除
		removeFullPageLoading();
		
		var loadingOverlay = document.createElement('div');
		loadingOverlay.id = '__browserwing_loading_overlay__';
		loadingOverlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(255,255,255,0.9);z-index:9999999;display:flex;align-items:center;justify-content:center;backdrop-filter:blur(8px);';
		
		var loadingBox = document.createElement('div');
		loadingBox.style.cssText = 'background:white;padding:32px 48px;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,0.12);text-align:center;max-width:400px;border:1px solid #e5e7eb;';
		
		var spinner = document.createElement('div');
		spinner.style.cssText = 'width:48px;height:48px;border:3px solid #f3f4f6;border-top-color:#1f2937;border-radius:50%;animation:spin 1s linear infinite;margin:0 auto 20px;';
		
		var loadingText = document.createElement('div');
		loadingText.style.cssText = 'color:#1f2937;font-size:16px;font-weight:600;margin-bottom:8px;';
		loadingText.textContent = message || '{{AI_GENERATING}}';
		
		var loadingTip = document.createElement('div');
		loadingTip.style.cssText = 'color:#6b7280;font-size:13px;';
		loadingTip.textContent = '{{PLEASE_WAIT}}';
		
		loadingBox.appendChild(spinner);
		loadingBox.appendChild(loadingText);
		loadingBox.appendChild(loadingTip);
		loadingOverlay.appendChild(loadingBox);
		
		// 添加旋转动画
		if (!document.getElementById('__browserwing_spin_animation__')) {
			var style = document.createElement('style');
			style.id = '__browserwing_spin_animation__';
			style.textContent = '@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }';
			document.head.appendChild(style);
		}
		
		document.body.appendChild(loadingOverlay);
		window.__loadingOverlay__ = loadingOverlay;
	};
	
	// 移除全屏 Loading
	var removeFullPageLoading = function() {
		if (window.__loadingOverlay__) {
			try {
				window.__loadingOverlay__.remove();
			} catch(e) {}
			window.__loadingOverlay__ = null;
		}
	};
	
	// HTML 转义函数
	var escapeHtml = function(text) {
		if (!text) return '';
		var div = document.createElement('div');
		div.textContent = text;
		return div.innerHTML;
	};
	
	// 更新动作计数
	var updateActionCount = function() {
		if (!window.__recorderUI__) return;
		var count = window.__recordedActions__.length;
		window.__recorderUI__.actionCount.textContent = count + ' {{STEPS_UNIT}}';
	};
	
	// 添加动作到列表
	var addActionToList = function(action, index) {
		if (!window.__recorderUI__) return;
		
		var list = window.__recorderUI__.actionList;
		var emptyState = window.__recorderUI__.emptyState;
		
		// 隐藏空状态
		if (emptyState && emptyState.parentNode) {
			emptyState.style.display = 'none';
		}
		
		var item = document.createElement('div');
		item.setAttribute('data-action-index', index);
		item.style.cssText = 'padding:12px 14px;margin:6px 0;background:white;border:1px solid rgba(0,0,0,0.08);border-radius:10px;font-size:12px;transition:all 0.25s cubic-bezier(0.4,0,0.2,1);box-shadow:0 1px 2px rgba(0,0,0,0.04);position:relative;';
		item.onmouseover = function() {
			this.style.background = '#f9fafb';
			this.style.borderColor = '#d1d5db';
		};
		item.onmouseout = function() {
			this.style.background = 'white';
			this.style.borderColor = '#e5e7eb';
		};
		
		var header = document.createElement('div');
		header.style.cssText = 'display:flex;align-items:center;justify-content:space-between;margin-bottom:4px;';
		
		var leftSection = document.createElement('div');
		leftSection.style.cssText = 'display:flex;align-items:center;gap:8px;flex:1;';
		
		var typeLabel = document.createElement('span');
		typeLabel.style.cssText = 'font-weight:700;color:#0f172a;font-size:13px;letter-spacing:-0.01em;';
		
		var typeText = action.type;
		if (action.type.startsWith('extract_')) {
			typeText = 'Extract ' + action.type.replace('extract_', '');
		} else if (action.type === 'upload_file') {
			typeText = 'Upload File';
		} else if (action.type === 'sleep') {
			typeText = 'Sleep';
		} else if (action.type === 'execute_js') {
			typeText = 'Execute JS';
		}
		typeLabel.textContent = '#' + (index + 1) + ' ' + typeText.charAt(0).toUpperCase() + typeText.slice(1);
		
		var indexLabel = document.createElement('span');
		indexLabel.style.cssText = 'font-size:11px;color:#94a3b8;font-weight:600;letter-spacing:-0.01em;';
		indexLabel.textContent = new Date(action.timestamp).toLocaleTimeString();
		
		leftSection.appendChild(typeLabel);
		leftSection.appendChild(indexLabel);
		
		// 操作按钮区域
		var actionButtons = document.createElement('div');
		actionButtons.style.cssText = 'display:flex;align-items:center;gap:6px;';
		
		// 如果是 execute_js 类型，添加预览按钮
		if (action.type === 'execute_js' && action.js_code) {
			var previewBtn = document.createElement('button');
			previewBtn.setAttribute('data-action-index', index);
			previewBtn.style.cssText = 'padding:5px;background:rgb(51 51 52);color:white;border:none;border-radius:6px;cursor:pointer;transition:all 0.2s;width:23px;height:23px;display:flex;align-items:center;justify-content:center;';
			previewBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>';
			previewBtn.onmouseover = function() {
				this.style.background = 'rgb(51 51 52)';
				this.style.transform = 'scale(1.08)';
			};
			previewBtn.onmouseout = function() {
				this.style.background = 'rgb(51 51 52)';
				this.style.transform = 'scale(1)';
			};
			previewBtn.onclick = function(e) {
				e.stopPropagation();
				var idx = parseInt(this.getAttribute('data-action-index'));
				var act = window.__recordedActions__[idx];
				previewExecuteJSResult(act);
			};
			actionButtons.appendChild(previewBtn);
		}
		
		// 添加删除按钮
		var deleteBtn = document.createElement('button');
		deleteBtn.setAttribute('data-action-index', index);
		deleteBtn.style.cssText = 'padding:5px;background:#6b7280;color:white;border:none;border-radius:6px;cursor:pointer;transition:all 0.2s;width:23px;height:23px;display:flex;align-items:center;justify-content:center;';
		deleteBtn.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M10 11v6M14 11v6"></path></svg>';
		deleteBtn.onmouseover = function() {
			this.style.background = '#4b5563';
			this.style.transform = 'scale(1.08)';
		};
		deleteBtn.onmouseout = function() {
			this.style.background = '#6b7280';
			this.style.transform = 'scale(1)';
		};
		deleteBtn.onclick = function(e) {
			e.stopPropagation();
			var idx = parseInt(this.getAttribute('data-action-index'));
			deleteAction(idx);
		};
		actionButtons.appendChild(deleteBtn);
		
		header.appendChild(leftSection);
		header.appendChild(actionButtons);
		
		var details = document.createElement('div');
		details.style.cssText = 'color:#64748b;font-size:11px;margin-top:4px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-weight:500;';
		
		var detailText = '';
		
		// 特殊处理 sleep action
		if (action.type === 'sleep' && action.duration) {
			detailText = '⏱ {{WAIT_PREFIX}}' + (action.duration / 1000).toFixed(1) + ' {{SECONDS_UNIT}}';
			if (action.description) {
				detailText = action.description;
			}
		} else {
			// 优先显示 xpath，其次显示 selector
			if (action.xpath) {
				detailText = 'XPath: ' + escapeHtml(action.xpath);
			} else if (action.selector) {
				detailText = 'CSS: ' + escapeHtml(action.selector);
			}
			
			// 对于 input 类型，显示输入的值
			if (action.type === 'input' && action.value) {
				detailText += ' → "' + escapeHtml(action.value.substring(0, 30)) + (action.value.length > 30 ? '...' : '') + '"';
			}
			
			// 对于 click 类型，显示点击的文本（如果有）
			if (action.type === 'click' && action.text && action.text.length > 0) {
				detailText += ' ("' + escapeHtml(action.text.substring(0, 20)) + (action.text.length > 20 ? '...' : '') + '")';
			}
			
			// 显示数据抓取的变量名
			if (action.variable_name) {
				detailText += ' → ' + escapeHtml(action.variable_name);
			}
			
			// 显示文件上传的文件名
			if (action.file_names && action.file_names.length > 0) {
				detailText = 'Files: ' + action.file_names.map(function(f) { return escapeHtml(f); }).join(', ');
			}
		}
		
		details.innerHTML = detailText || 'No details';
		
		item.appendChild(header);
		item.appendChild(details);
		list.appendChild(item);
		
		// 自动滚动到底部
		list.scrollTop = list.scrollHeight;
	};
	
	// 显示当前操作提示
	var showCurrentAction = function(actionText) {
		if (!window.__recorderUI__) return;
		
		var currentAction = window.__recorderUI__.currentAction;
		currentAction.textContent = actionText;
		currentAction.style.display = 'block';
		
		// 3秒后自动隐藏
		setTimeout(function() {
			currentAction.style.display = 'none';
		}, 3000);
	};
	
	// 删除指定的操作步骤
	var deleteAction = function(index) {
		if (!window.__recordedActions__ || index < 0 || index >= window.__recordedActions__.length) {
			console.error('[BrowserWing] Invalid action index:', index);
			return;
		}
		
		// 确认删除
		var action = window.__recordedActions__[index];
		var actionDesc = '#' + (index + 1) + ' ' + action.type;
		if (!confirm('{{CONFIRM_DELETE}}\n\n' + actionDesc)) {
			return;
		}
		
		// 从数组中删除
		window.__recordedActions__.splice(index, 1);
		console.log('[BrowserWing] Deleted action at index:', index);
		
		// 更新 sessionStorage
		try {
			sessionStorage.setItem('__browserwing_actions__', JSON.stringify(window.__recordedActions__));
		} catch (e) {
			console.error('[BrowserWing] sessionStorage save error:', e);
		}
		
		// 重新渲染整个列表
		refreshActionList();
		
		// 更新计数
		updateActionCount();
		
		showCurrentAction('{{STEP_DELETED}}');
	};
	
	// 刷新整个操作列表
	var refreshActionList = function() {
		if (!window.__recorderUI__) return;
		
		var list = window.__recorderUI__.actionList;
		var emptyState = window.__recorderUI__.emptyState;
		
		// 收集所有需要删除的节点（除了 emptyState）
		var nodesToRemove = [];
		for (var i = 0; i < list.childNodes.length; i++) {
			var node = list.childNodes[i];
			if (node !== emptyState) {
				nodesToRemove.push(node);
			}
		}
		
		// 删除所有收集到的节点
		for (var j = 0; j < nodesToRemove.length; j++) {
			list.removeChild(nodesToRemove[j]);
		}
		
		// 如果没有操作，显示空状态
		if (window.__recordedActions__.length === 0) {
			if (emptyState) {
				emptyState.style.display = 'block';
			}
			return;
		}
		
		// 隐藏空状态
		if (emptyState) {
			emptyState.style.display = 'none';
		}
		
		// 重新添加所有操作
		for (var k = 0; k < window.__recordedActions__.length; k++) {
			addActionToList(window.__recordedActions__[k], k);
		}
	};
	
	// 预览 execute_js 操作的结果
	var previewExecuteJSResult = function(action) {
		if (!action || !action.js_code) {
			console.error('[BrowserWing] Invalid action for preview');
			return;
		}
		
		showFullPageLoading('{{EXECUTING_CODE}}');
		
		try {
			// 执行 JS 代码
			// 注意：AI 生成的代码通常已经是 IIFE (立即执行函数表达式)，如 (() => {...})()
			// 直接 eval 执行即可，不需要再包装一层 function
			var result = eval(action.js_code);
			
			console.log('[BrowserWing] Execute result:', result);
			
			// 移除 Loading
			removeFullPageLoading();
			
			// 显示结果弹框
			showPreviewDialog(action, result);
		} catch (error) {
			// 移除 Loading
			removeFullPageLoading();
			
			// 显示错误
			alert('{{EXECUTE_ERROR}}\n\n' + error.message);
			console.error('[BrowserWing] Execute JS error:', error);
		}
	};
	
	// 显示预览对话框
	var showPreviewDialog = function(action, result) {
		// 创建遮罩层
		var overlay = document.createElement('div');
		overlay.id = '__browserwing_preview_dialog__';
		overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.5);z-index:10000000;display:flex;align-items:center;justify-content:center;backdrop-filter:blur(4px);';
		
		// 创建对话框
		var dialog = document.createElement('div');
		dialog.style.cssText = 'background:white;border-radius:16px;box-shadow:0 20px 50px rgba(0,0,0,0.3);max-width:800px;width:90%;max-height:80vh;display:flex;flex-direction:column;overflow:hidden;';
		
		// 对话框头部
		var dialogHeader = document.createElement('div');
		dialogHeader.style.cssText = 'padding:20px 24px;border-bottom:1px solid #e5e7eb;display:flex;align-items:center;justify-content:space-between;';
		
		var dialogTitle = document.createElement('div');
		dialogTitle.style.cssText = 'font-size:18px;font-weight:700;color:#0f172a;';
		dialogTitle.textContent = '{{PREVIEW_TITLE}}';
		
		var closeBtn = document.createElement('button');
		closeBtn.style.cssText = 'background:none;border:none;font-size:24px;color:#6b7280;cursor:pointer;width:32px;height:32px;display:flex;align-items:center;justify-content:center;border-radius:6px;transition:all 0.2s;';
		closeBtn.textContent = '×';
		closeBtn.onmouseover = function() {
			this.style.background = '#f3f4f6';
			this.style.color = '#111827';
		};
		closeBtn.onmouseout = function() {
			this.style.background = 'none';
			this.style.color = '#6b7280';
		};
		closeBtn.onclick = function() {
			overlay.remove();
		};
		
		dialogHeader.appendChild(dialogTitle);
		dialogHeader.appendChild(closeBtn);
		
		// 对话框内容
		var dialogContent = document.createElement('div');
		dialogContent.style.cssText = 'padding:24px;overflow-y:auto;flex:1;';
		
		// 变量名显示
		if (action.variable_name) {
			var varName = document.createElement('div');
			varName.style.cssText = 'margin-bottom:16px;padding:12px;background:#f8fafc;border-radius:8px;border:1px solid #e2e8f0;';
			varName.innerHTML = '<strong style="color:#475569;">{{VARIABLE_NAME}}:</strong> <code style="color:#1e293b;font-weight:600;">' + escapeHtml(action.variable_name) + '</code>';
			dialogContent.appendChild(varName);
		}
		
		// 结果标题
		var resultTitle = document.createElement('div');
		resultTitle.style.cssText = 'font-size:14px;font-weight:600;color:#475569;margin-bottom:12px;';
		resultTitle.textContent = '{{EXECUTION_RESULT}}:';
		dialogContent.appendChild(resultTitle);
		
		// 格式化 JSON（提前，以便复制按钮使用）
		var jsonStr = '';
		var hasValidResult = !(result === undefined || result === null);
		
		if (hasValidResult) {
			try {
				jsonStr = JSON.stringify(result, null, 2);
			} catch (e) {
				jsonStr = String(result);
			}
		}
		
		// 检查结果是否为空
		if (!hasValidResult) {
			var emptyNotice = document.createElement('div');
			emptyNotice.style.cssText = 'padding:24px;background:#fef2f2;border:1px solid #fecaca;border-radius:8px;text-align:center;color:#991b1b;font-size:14px;';
			emptyNotice.innerHTML = '<strong>⚠️ {{NO_DATA_RETURNED}}</strong><br><span style="font-size:12px;color:#dc2626;margin-top:8px;display:block;">{{CHECK_CODE_HINT}}</span>';
			dialogContent.appendChild(emptyNotice);
		} else {
			// JSON 显示区域容器（相对定位，用于浮动复制按钮）
			var jsonContainer = document.createElement('div');
			jsonContainer.style.cssText = 'position:relative;';
			
			// 复制按钮（浮动在右上角）
			var copyBtn = document.createElement('button');
			copyBtn.style.cssText = 'position:absolute;top:8px;right:8px;background:rgba(255,255,255,0.15);color:rgba(255,255,255,0.9);border:none;padding:6px 12px;border-radius:6px;cursor:pointer;font-size:12px;font-weight:600;transition:all 0.2s;z-index:10;backdrop-filter:blur(8px);';
			copyBtn.textContent = '{{COPY}}';
			copyBtn.onmouseover = function() {
				this.style.background = 'rgba(255,255,255,0.25)';
				this.style.transform = 'scale(1.05)';
			};
			copyBtn.onmouseout = function() {
				this.style.background = 'rgba(255,255,255,0.15)';
				this.style.transform = 'scale(1)';
			};
			
			// 复制按钮点击事件
			copyBtn.onclick = function() {
				if (!jsonStr) {
					alert('{{NO_DATA_TO_COPY}}');
					return;
				}
				
				// 复制到剪贴板
				try {
					// 使用现代 Clipboard API
					if (navigator.clipboard && navigator.clipboard.writeText) {
						navigator.clipboard.writeText(jsonStr).then(function() {
							// 成功反馈
							var originalText = copyBtn.textContent;
							copyBtn.textContent = '{{COPIED}}';
							copyBtn.style.background = 'rgba(255,255,255,0.3)';
							
							setTimeout(function() {
								copyBtn.textContent = originalText;
								copyBtn.style.background = 'rgba(255,255,255,0.15)';
							}, 2000);
						}).catch(function(err) {
							console.error('[BrowserWing] Copy failed:', err);
							alert('{{COPY_FAILED}}');
						});
					} else {
						// 降级方案：使用 textarea
						var textarea = document.createElement('textarea');
						textarea.value = jsonStr;
						textarea.style.position = 'fixed';
						textarea.style.opacity = '0';
						document.body.appendChild(textarea);
						textarea.select();
						
						try {
							document.execCommand('copy');
							// 成功反馈
							var originalText = copyBtn.textContent;
							copyBtn.textContent = '{{COPIED}}';
							copyBtn.style.background = 'rgba(255,255,255,0.3)';
							
							setTimeout(function() {
								copyBtn.textContent = originalText;
								copyBtn.style.background = 'rgba(255,255,255,0.15)';
							}, 2000);
						} catch (err) {
							console.error('[BrowserWing] Copy failed:', err);
							alert('{{COPY_FAILED}}');
						} finally {
							document.body.removeChild(textarea);
						}
					}
				} catch (err) {
					console.error('[BrowserWing] Copy error:', err);
					alert('{{COPY_FAILED}}');
				}
			};
			
			// JSON 显示区域
			var jsonDisplay = document.createElement('pre');
			jsonDisplay.style.cssText = 'background:#1e293b;color:#e2e8f0;padding:16px;padding-top:40px;border-radius:8px;overflow-x:auto;font-size:13px;line-height:1.6;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;margin:0;box-shadow:inset 0 2px 4px rgba(0,0,0,0.2);';
			jsonDisplay.textContent = jsonStr;
			
			jsonContainer.appendChild(copyBtn);
			jsonContainer.appendChild(jsonDisplay);
			dialogContent.appendChild(jsonContainer);
			
			// 数据统计信息
			var stats = document.createElement('div');
			stats.style.cssText = 'margin-top:16px;padding:12px;background:#ecfdf5;border-radius:8px;border:1px solid #a7f3d0;color:#065f46;font-size:13px;';
			
			var dataType = Array.isArray(result) ? 'Array' : typeof result;
			var dataSize = '';
			if (Array.isArray(result)) {
				dataSize = result.length + ' {{ITEMS}}';
			} else if (typeof result === 'object' && result !== null) {
				dataSize = Object.keys(result).length + ' {{PROPERTIES}}';
			}
			
			stats.innerHTML = '<strong>{{DATA_TYPE}}:</strong> ' + dataType + (dataSize ? ' (' + dataSize + ')' : '');
			dialogContent.appendChild(stats);
		}
		
		// 组装对话框
		dialog.appendChild(dialogHeader);
		dialog.appendChild(dialogContent);
		overlay.appendChild(dialog);
		
		// 添加到页面
		document.body.appendChild(overlay);
		
		// 点击遮罩层关闭
		overlay.onclick = function(e) {
			if (e.target === overlay) {
				overlay.remove();
			}
		};
	};
	
	// 切换抓取模式
	var toggleExtractMode = function() {
		window.__extractMode__ = !window.__extractMode__;
		var ui = window.__recorderUI__;
		
		if (window.__extractMode__) {
			// 开启抓取模式
			var extractType = window.__extractType__ || 'text';
			var modeLabel = '{{EXIT_EXTRACT}}';
			
			// 根据抓取类型显示不同的按钮文本
			if (extractType === 'text') {
				modeLabel = '{{EXIT_EXTRACT_TEXT}}';
			} else if (extractType === 'html') {
				modeLabel = '{{EXIT_EXTRACT_HTML}}';
			} else if (extractType === 'attribute') {
				modeLabel = '{{EXIT_EXTRACT_ATTR}}';
			}
			
			ui.extractBtn.textContent = modeLabel;
			ui.extractBtn.style.background = '#1f2937';
			ui.extractBtn.style.color = 'white';
			ui.extractBtn.style.borderColor = '#1f2937';
			ui.extractBtn.onmouseover = function() {
				this.style.background = '#111827';
			};
			ui.extractBtn.onmouseout = function() {
				this.style.background = '#1f2937';
			};
			document.body.style.cursor = 'crosshair';
			
			// 根据抓取类型显示不同的提示
			var modeText = '{{EXTRACT_MODE_ENABLED}}';
			if (extractType === 'text') {
				modeText = '{{EXTRACT_TEXT_MODE}}';
			} else if (extractType === 'html') {
				modeText = '{{EXTRACT_HTML_MODE}}';
			} else if (extractType === 'attribute') {
				modeText = '{{EXTRACT_ATTRIBUTE_MODE}}';
			}
			showCurrentAction(modeText);
			console.log('[BrowserWing] Extract mode enabled, type:', extractType);
		} else {
			// 关闭抓取模式
			ui.extractBtn.textContent = '{{DATA_EXTRACT}}';
			ui.extractBtn.style.background = 'white';
			ui.extractBtn.style.color = '#374151';
			ui.extractBtn.style.borderColor = '#d1d5db';
			ui.extractBtn.onmouseover = function() {
				this.style.background = '#f8fafc';
				this.style.borderColor = 'rgba(0,0,0,0.18)';
				this.style.transform = 'translateY(-1px)';
				this.style.boxShadow = '0 2px 8px rgba(0,0,0,0.08)';
			};
			ui.extractBtn.onmouseout = function() {
				this.style.background = 'white';
				this.style.borderColor = 'rgba(0,0,0,0.12)';
				this.style.transform = 'translateY(0)';
				this.style.boxShadow = '0 1px 3px rgba(0,0,0,0.04)';
			};
			ui.menu.style.display = 'none';
			document.body.style.cursor = 'default';
			hideHighlight();
			console.log('[BrowserWing] Extract mode disabled');
		}
	};
	
	// 切换 AI 填充表单模式
	var toggleAIFormFillMode = function() {
		window.__aiFormFillMode__ = !window.__aiFormFillMode__;
		var ui = window.__recorderUI__;
		
		if (window.__aiFormFillMode__) {
			// 关闭其他模式
			if (window.__extractMode__) {
				toggleExtractMode();
			}
			if (window.__aiExtractMode__) {
				toggleAIExtractMode();
			}
			
			// 开启 AI 填充表单模式
			ui.aiFormFillBtn.textContent = '{{EXIT_AI_FORMFILL}}';
			ui.aiFormFillBtn.style.background = '#047857';
			ui.aiFormFillBtn.onmouseover = function() {
				this.style.background = '#065f46';
			};
			ui.aiFormFillBtn.onmouseout = function() {
				this.style.background = '#047857';
			};
			document.body.style.cursor = 'crosshair';
			showCurrentAction('{{AI_FORMFILL_MODE_ENABLED}}');
			console.log('[BrowserWing] AI Form Fill mode enabled');
		} else {
			// 关闭 AI 填充表单模式
			ui.aiFormFillBtn.textContent = '{{AI_FORMFILL}}';
			ui.aiFormFillBtn.style.background = '#059669';
			ui.aiFormFillBtn.style.borderColor = '#059669';
			ui.aiFormFillBtn.onmouseover = function() {
				this.style.background = '#047857';
				this.style.borderColor = '#047857';
			};
			ui.aiFormFillBtn.onmouseout = function() {
				this.style.background = '#059669';
				this.style.borderColor = '#059669';
			};
			document.body.style.cursor = 'default';
			window.__selectedElement__ = null;
			hideHighlight();
			console.log('[BrowserWing] AI Form Fill mode disabled');
		}
	};
	
	// 切换 AI 提取模式
	var toggleAIExtractMode = function() {
		window.__aiExtractMode__ = !window.__aiExtractMode__;
		var ui = window.__recorderUI__;
		
		if (window.__aiExtractMode__) {
			// 关闭其他模式
			if (window.__extractMode__) {
				toggleExtractMode();
			}
			if (window.__aiFormFillMode__) {
				toggleAIFormFillMode();
			}
			
			// 开启 AI 提取模式
			ui.aiExtractBtn.textContent = '{{EXIT_AI_EXTRACT}}';
			ui.aiExtractBtn.style.background = '#111827';
			ui.aiExtractBtn.onmouseover = function() {
				this.style.background = '#030712';
			};
			ui.aiExtractBtn.onmouseout = function() {
				this.style.background = '#111827';
			};
			document.body.style.cursor = 'crosshair';
			showCurrentAction('{{AI_EXTRACT_MODE_ENABLED}}');
			console.log('[BrowserWing] AI Extract mode enabled');
		} else {
			// 关闭 AI 提取模式
			ui.aiExtractBtn.textContent = '{{AI_EXTRACT}}';
			ui.aiExtractBtn.style.background = '#1f2937';
			ui.aiExtractBtn.style.borderColor = '#1f2937';
			ui.aiExtractBtn.onmouseover = function() {
				this.style.background = '#111827';
				this.style.borderColor = '#111827';
			};
			ui.aiExtractBtn.onmouseout = function() {
				this.style.background = '#1f2937';
				this.style.borderColor = '#1f2937';
			};
			document.body.style.cursor = 'default';
			window.__selectedElement__ = null;
			hideHighlight();
			console.log('[BrowserWing] AI Extract mode disabled');
		}
	};
	
	// 处理 AI 填充表单元素点击
	var handleAIFormFillClick = async function(element) {
		if (!element || !element.outerHTML) {
			console.error('[BrowserWing] Invalid element for AI form fill');
			return;
		}
		
		window.__selectedElement__ = element;
		
		// 先弹出确认对话框，让用户选择是否添加自定义要求
		var userChoice = confirm('{{AI_FORMFILL_CONFIRM}}\n\n{{CLICK_OK_TO_ADD_PROMPT}}\n{{CLICK_CANCEL_FOR_DEFAULT}}');
		
		var userPrompt = '';
		if (userChoice) {
			// 用户选择添加自定义要求，弹出输入框
			userPrompt = prompt('{{AI_FORMFILL_PROMPT_INPUT}}:', '');
			
			// 用户取消输入，直接返回
			if (userPrompt === null) {
				console.log('[BrowserWing] User cancelled AI form fill');
				return;
			}
		}
		
		// 显示全屏 Loading
		showFullPageLoading('{{AI_ANALYZING_FORM}}');
		
		try {
			// 查找表单元素（可能点击的是表单本身或表单内的元素）
			var formElement = element;
			if (element.tagName.toLowerCase() !== 'form') {
				formElement = element.closest('form');
				if (!formElement) {
					// 如果没有找到 form 标签，使用点击的容器元素
					formElement = element;
				}
			}
			
			// 清理和优化 HTML
			var cleanedHtml = cleanAndSampleHTML(formElement);
			
			// 更新 loading 提示
			if (window.__loadingOverlay__) {
				var loadingText = window.__loadingOverlay__.querySelector('div > div:nth-child(2)');
				if (loadingText) loadingText.textContent = '{{AI_GENERATING_FILL}}';
			}
			
			console.log('[BrowserWing] Submitting AI form fill request via polling...');
			
			// 设置请求到全局变量，让后端轮询处理
			window.__aiExtractionRequest__ = {
				type: 'formfill',
				html: cleanedHtml,
				description: '{{FORMFILL_PROMPT}}',
				user_prompt: userPrompt || ''
			};
			
			// 轮询等待后端处理结果
			var maxWaitTime = 60000;
			var pollInterval = 200;
			var elapsedTime = 0;
			
			var result = await new Promise(function(resolve, reject) {
				var pollTimer = setInterval(function() {
					elapsedTime += pollInterval;
					
					if (window.__aiFormFillResponse__) {
						clearInterval(pollTimer);
						var response = window.__aiFormFillResponse__;
						delete window.__aiFormFillResponse__;
						delete window.__aiExtractionRequest__;
						resolve(response);
					} else if (elapsedTime >= maxWaitTime) {
						clearInterval(pollTimer);
						delete window.__aiExtractionRequest__;
						reject(new Error('{{AI_TIMEOUT}}'));
					}
				}, pollInterval);
			});
			
			// 检查响应是否成功
			if (!result.success) {
				throw new Error(result.error || '{{AI_FORMFILL_FAILED}}');
			}
			
			if (!result.javascript) {
				throw new Error('{{NO_CODE_RECEIVED}}');
			}
			
			console.log('[BrowserWing] AI form fill successful, code length:', result.javascript.length);
			
			// 创建一个 execute_js 操作记录
			var selectors = getSelector(formElement);
			var variableName = 'ai_formfill_' + window.__recordedActions__.length;
			
			var action = {
				type: 'execute_js',
				timestamp: Date.now(),
				selector: selectors.css,
				xpath: selectors.xpath,
				js_code: result.javascript,
				variable_name: variableName,
				tagName: formElement.tagName ? formElement.tagName.toLowerCase() : '',
			description: '{{AI_FORMFILL_DESC}}'
		};
		
		recordAction(action, formElement, 'execute_js');			// 移除 Loading
			removeFullPageLoading();
			
			showCurrentAction('{{AI_FORMFILL_SUCCESS}}' + (result.used_model || 'unknown'));
			console.log('[BrowserWing] AI form fill code added:', variableName);
			
			// 自动退出 AI 填充表单模式
			setTimeout(function() {
				if (window.__aiFormFillMode__) {
					toggleAIFormFillMode();
				}
			}, 2000);
			
		} catch (error) {
			// 移除 Loading
			removeFullPageLoading();
			
			// 发生错误时清理可能残留的全局变量
			delete window.__aiExtractionRequest__;
			delete window.__aiFormFillResponse__;
			
			console.error('[BrowserWing] AI form fill error:', error);
			showCurrentAction('{{AI_FORMFILL_ERROR}}' + error.message);
		}
	};
	
	// 处理 AI 提取元素点击
	var handleAIExtractClick = async function(element) {
		if (!element || !element.outerHTML) {
			console.error('[BrowserWing] Invalid element for AI extraction');
			return;
		}
		
		window.__selectedElement__ = element;
		
		// 先弹出确认对话框，让用户选择是否添加自定义要求
		var userChoice = confirm('{{AI_EXTRACT_CONFIRM}}\n\n{{CLICK_OK_TO_ADD_PROMPT}}\n{{CLICK_CANCEL_FOR_DEFAULT}}');
		
		var userPrompt = '';
		if (userChoice) {
			// 用户选择添加自定义要求，弹出输入框
			userPrompt = prompt('{{AI_EXTRACT_PROMPT_INPUT}}:', '');
			
			// 用户取消输入，直接返回
			if (userPrompt === null) {
				console.log('[BrowserWing] User cancelled AI extraction');
				return;
			}
		}
		
		// 显示全屏 Loading
		showFullPageLoading('{{AI_ANALYZING_PAGE}}');
		
		try {
			// 清理和优化 HTML
			var cleanedHtml = cleanAndSampleHTML(element);
			
		// 更新 loading 提示
		if (window.__loadingOverlay__) {
			var loadingText = window.__loadingOverlay__.querySelector('div > div:nth-child(2)');
			if (loadingText) loadingText.textContent = '{{AI_GENERATING}}';
		}
		
		console.log('[BrowserWing] Submitting AI extraction request via polling...');			// 设置请求到全局变量，让后端轮询处理（避免 CSP 问题）
			window.__aiExtractionRequest__ = {
				html: cleanedHtml,
				description: '{{EXTRACT_PROMPT}}',
				user_prompt: userPrompt || ''
			};
			
			// 轮询等待后端处理结果
			var maxWaitTime = 60000; // 最多等待 60 秒
			var pollInterval = 200; // 每 200ms 检查一次
			var elapsedTime = 0;
			
			var result = await new Promise(function(resolve, reject) {
				var checkResponse = function() {
					if (window.__aiExtractionResponse__) {
						var response = window.__aiExtractionResponse__;
						delete window.__aiExtractionResponse__; // 立即清除响应，防止重复处理
						resolve(response);
						return;
					}
					
					elapsedTime += pollInterval;
					if (elapsedTime >= maxWaitTime) {
						// 超时清理请求，防止后端后续处理
						delete window.__aiExtractionRequest__;
						reject(new Error('{{AI_TIMEOUT}}'));
						return;
					}
					
					setTimeout(checkResponse, pollInterval);
				};
				
				checkResponse();
			});
			
			// 检查响应是否成功
			if (!result.success) {
				throw new Error(result.error || '{{AI_EXTRACT_FAILED}}');
			}
			
			if (!result.javascript) {
				throw new Error('No JavaScript code returned');
			}
			
			console.log('[BrowserWing] AI extraction successful, code length:', result.javascript.length);
			
			// 创建一个 execute_js 操作记录
			var selectors = getSelector(element);
			var variableName = 'ai_data_' + window.__recordedActions__.length;
			
			var action = {
				type: 'execute_js',
				timestamp: Date.now(),
				selector: selectors.css,
				xpath: selectors.xpath,
				js_code: result.javascript,
				variable_name: variableName,
				tagName: element.tagName ? element.tagName.toLowerCase() : '',
			description: '{{AI_EXTRACT_DESC}}'
		};
		
		recordAction(action, window.__selectedElement__, 'execute_js');			// 移除 Loading
			removeFullPageLoading();
			
			showCurrentAction('{{AI_EXTRACT_SUCCESS}}' + (result.used_model || 'unknown'));
			console.log('[BrowserWing] AI extraction code added:', variableName);
			
			// 自动退出 AI 提取模式
			setTimeout(function() {
				if (window.__aiExtractMode__) {
					toggleAIExtractMode();
				}
			}, 2000);
			
		} catch (error) {
			// 移除 Loading
			removeFullPageLoading();
			
			// 发生错误时清理可能残留的全局变量
			delete window.__aiExtractionRequest__;
			delete window.__aiExtractionResponse__;
			
			console.error('[BrowserWing] AI extraction error:', error);
			showCurrentAction('{{AI_EXTRACT_ERROR}}' + error.message);
		}
	};
	
	// 清理和采样 HTML - 移除无关属性并智能提取列表项样本
	var cleanAndSampleHTML = function(element) {
		console.log('[BrowserWing] Starting HTML cleanup and sampling...');
		
		// 克隆元素以避免修改原始 DOM
		var clone = element.cloneNode(true);
		
		// 步骤1: 移除无关属性
		var removeAttributes = [
			'style', 'onclick', 'onmouseover', 'onmouseout', 'onload',
			'data-reactid', 'data-react-checksum', 'data-reactroot',
			'data-v-', // Vue 相关
			'ng-', // Angular 相关
			'_ngcontent-', '_nghost-', // Angular 相关
			'tabindex', 'aria-hidden', 'aria-label', 'aria-describedby',
			'data-spm', 'data-track', 'data-analytics', 'data-ga', // 埋点相关
			'data-test', 'data-testid', 'data-qa', 'data-cy', // 测试相关（保留可能用于定位）
			'draggable', 'contenteditable',
			'autocomplete', 'spellcheck',
			'srcset', 'sizes' // 图片响应式属性
		];
		
		var cleanElement = function(el) {
			if (!el || el.nodeType !== 1) return;
			
			// 移除指定属性
			for (var i = 0; i < removeAttributes.length; i++) {
				var attr = removeAttributes[i];
				if (attr.endsWith('-')) {
					// 前缀匹配（如 data-v-, ng-）
					var attrs = el.attributes;
					for (var j = attrs.length - 1; j >= 0; j--) {
						if (attrs[j].name.startsWith(attr)) {
							el.removeAttribute(attrs[j].name);
						}
					}
				} else {
					el.removeAttribute(attr);
				}
			}
			
			// 简化 class（移除动态生成的类名）
			if (el.className && typeof el.className === 'string') {
				var classes = el.className.split(/\s+/);
				var cleanClasses = [];
				for (var k = 0; k < classes.length; k++) {
					var cls = classes[k];
					// 保留简短的、有意义的类名，排除哈希类名
					if (cls.length > 0 && cls.length < 30 && 
					    !/^[a-f0-9]{8,}$/i.test(cls) && // 排除纯哈希
					    !/--[a-f0-9]{5,}$/i.test(cls)) { // 排除 CSS Modules 哈希
						cleanClasses.push(cls);
					}
				}
				if (cleanClasses.length > 0) {
					el.className = cleanClasses.slice(0, 3).join(' '); // 最多保留3个类名
				} else {
					el.removeAttribute('class');
				}
			}
			
			// 递归清理子元素
			for (var m = 0; m < el.children.length; m++) {
				cleanElement(el.children[m]);
			}
		};
		
		cleanElement(clone);

		// 步骤3: 获取清理后的 HTML
		var cleanedHtml = clone.outerHTML;
		
		// 步骤4: 美化输出（移除多余空白）
		cleanedHtml = cleanedHtml
			.replace(/\s+/g, ' ') // 多个空格合并为一个
			.replace(/>\s+</g, '><') // 移除标签间空白
			.trim();
		
		console.log('[BrowserWing] Original length: ' + element.outerHTML.length + ', Cleaned length: ' + cleanedHtml.length);
		
		// 如果还是太长，截断
		if (cleanedHtml.length > 30000) {
			cleanedHtml = cleanedHtml.substring(0, 30000) + '...[truncated]';
		}
		
		return cleanedHtml;
	};
	
	// 检测列表项 - 识别重复的子元素结构
	var detectListItems = function(container) {
		if (!container || !container.children || container.children.length < 2) {
			return null;
		}
		
		var children = Array.prototype.slice.call(container.children);
		
		// 策略1: 检查是否有相同标签名和类名的子元素（最常见）
		var tagClassMap = {};
		for (var i = 0; i < children.length; i++) {
			var child = children[i];
			var key = child.tagName + '|' + (child.className || '');
			if (!tagClassMap[key]) {
				tagClassMap[key] = [];
			}
			tagClassMap[key].push(child);
		}
		
		// 找出数量最多的组
		var maxCount = 0;
		var maxGroup = null;
		for (var key in tagClassMap) {
			if (tagClassMap[key].length > maxCount) {
				maxCount = tagClassMap[key].length;
				maxGroup = tagClassMap[key];
			}
		}
		
		// 如果有至少2个相同结构的元素，认为是列表项
		if (maxCount >= 2 && maxCount >= children.length * 0.5) {
			return maxGroup;
		}
		
		// 策略2: 递归检查子元素
		for (var j = 0; j < children.length; j++) {
			var subItems = detectListItems(children[j]);
			if (subItems && subItems.length >= 2) {
				return subItems;
			}
		}
		
		return null;
	};
	
	// 简化文本内容 - 将长文本替换为占位符
	var simplifyTextContent = function(element) {
		if (!element) return;
		
		// 遍历所有文本节点
		var walker = document.createTreeWalker(
			element,
			NodeFilter.SHOW_TEXT,
			null,
			false
		);
		
		var textNodesToSimplify = [];
		var node;
		while (node = walker.nextNode()) {
			var text = node.nodeValue.trim();
			if (text.length > 20) {
				textNodesToSimplify.push(node);
			}
		}
		
		// 替换长文本为简短占位符
		for (var i = 0; i < textNodesToSimplify.length; i++) {
			var textNode = textNodesToSimplify[i];
			var originalText = textNode.nodeValue.trim();
			var placeholder = originalText.substring(0, 15) + '...';
			textNode.nodeValue = placeholder;
		}
		
		// 简化属性值（如 alt, title）
		if (element.nodeType === 1) {
			['alt', 'title', 'placeholder'].forEach(function(attr) {
				var value = element.getAttribute(attr);
				if (value && value.length > 20) {
					element.setAttribute(attr, value.substring(0, 15) + '...');
				}
			});
			
			// 递归处理子元素
			for (var j = 0; j < element.children.length; j++) {
				simplifyTextContent(element.children[j]);
			}
		}
	};
	
	// 记录数据抓取操作
	var recordExtractAction = function(element, extractType, attributeName) {
		var selectors = getSelector(element);
		var variableName = 'data_' + window.__recordedActions__.length;
		
		var action = {
			type: 'extract_' + extractType,
			timestamp: Date.now(),
			selector: selectors.css,
			xpath: selectors.xpath,
			extract_type: extractType,
			variable_name: variableName,
			tagName: element.tagName ? element.tagName.toLowerCase() : '',
			text: (element.innerText || element.textContent || '').substring(0, 50)
		};
		
		if (extractType === 'attribute' && attributeName) {
			action.attribute_name = attributeName;
		}
		
		recordAction(action, element, 'extract_' + extractType);
		
		var actionText = 'Extracted ' + extractType + ' from <' + action.tagName + '> as ' + variableName;
		showCurrentAction(actionText);
		
		console.log('[BrowserWing] Recorded extraction:', extractType, variableName);
	};
	
	// 生成更精确和可靠的选择器（支持 CSS 和 XPath）
	var getSelector = function(element) {
		if (!element || !element.tagName) {
			return { css: 'unknown', xpath: '//*' };
		}
		
		try {
			var css = '';
			var xpath = '';
			
			// 策略1: 优先使用稳定的 ID
			if (element.id && element.id.length > 0 && !/^[0-9]/.test(element.id)) {
				css = '#' + element.id;
				xpath = '//*[@id="' + element.id + '"]';
				return { css: css, xpath: xpath };
			}
			
			// 策略2: 使用 name 属性（表单元素常用）
			if (element.name && element.name.length > 0) {
				var tagName = element.tagName.toLowerCase();
				css = tagName + '[name="' + element.name + '"]';
				xpath = '//' + tagName + '[@name="' + element.name + '"]';
				return { css: css, xpath: xpath };
			}
			
			// 策略3: 使用 data-testid 等测试属性（最稳定）
			var stableAttrs = ['data-testid', 'data-test', 'data-qa', 'data-cy', 'aria-label', 'role'];
			for (var i = 0; i < stableAttrs.length; i++) {
				var attr = stableAttrs[i];
				var value = element.getAttribute(attr);
				if (value && value.length > 0) {
					css = element.tagName.toLowerCase() + '[' + attr + '="' + value + '"]';
					xpath = '//' + element.tagName.toLowerCase() + '[@' + attr + '="' + value + '"]';
					return { css: css, xpath: xpath };
				}
			}
			
			// 策略4: 使用 placeholder（输入框常用）
			if (element.placeholder && element.placeholder.length > 0 && element.placeholder.length < 50) {
				css = element.tagName.toLowerCase() + '[placeholder="' + element.placeholder + '"]';
				xpath = '//' + element.tagName.toLowerCase() + '[@placeholder="' + element.placeholder + '"]';
				return { css: css, xpath: xpath };
			}
			
			// 辅助函数：构建完整 XPath 路径（最可靠）
			var getFullXPath = function(el) {
				if (el.id && !/^[0-9]/.test(el.id)) {
					return '//*[@id="' + el.id + '"]';
				}
				
				var path = '';
				for (; el && el.nodeType === 1; el = el.parentNode) {
					var index = 0;
					var tagName = el.tagName.toLowerCase();
					
					// 计算同类型兄弟节点中的位置
					for (var sibling = el.previousSibling; sibling; sibling = sibling.previousSibling) {
						if (sibling.nodeType === 1 && sibling.tagName === el.tagName) {
							index++;
						}
					}
					
					// 计算总共有多少个同类型兄弟节点
					var sameTagCount = 0;
					if (el.parentNode) {
						var children = el.parentNode.children;
						for (var k = 0; k < children.length; k++) {
							if (children[k].tagName === el.tagName) {
								sameTagCount++;
							}
						}
					}
					
					// 如果有多个同类型节点，添加索引
					var pathIndex = (sameTagCount > 1 ? '[' + (index + 1) + ']' : '');
					path = '/' + tagName + pathIndex + path;
					
					// 如果父节点有 ID，就可以停止了
					if (el.parentNode && el.parentNode.nodeType === 1 && el.parentNode.id && !/^[0-9]/.test(el.parentNode.id)) {
						path = '//*[@id="' + el.parentNode.id + '"]' + path;
						break;
					}
				}
				
				return path;
			};
			
			// 策略5: 使用文本内容（但要检查唯一性）
			var textContent = (element.textContent || element.innerText || '').trim();
			textContent = textContent.replace(/[\u200b-\u200d\ufeff]/g, '').replace(/\s+/g, ' ').trim();
			
			if (textContent.length > 0 && textContent.length < 30) {
				var tag = element.tagName.toLowerCase();
				if (tag === 'button' || tag === 'a' || tag === 'span') {
					// 检查是否有多个相同文本的元素
					var textXPath = '//' + tag + '[contains(normalize-space(.), "' + textContent.substring(0, 20) + '")]';
					
					var hasDuplicates = false;
					try {
						var result = document.evaluate(textXPath, document, null, XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null);
						if (result.snapshotLength > 1) {
							hasDuplicates = true;
							console.log('[BrowserWing] Found ' + result.snapshotLength + ' elements with text "' + textContent.substring(0, 20) + '", using full XPath');
						}
					} catch (e) {
						console.warn('[BrowserWing] Failed to check duplicates:', e);
					}

					var hasNumbers = /[0-9]/.test(textContent);
					
					// 如果没有重复且没有数字，使用文本匹配
					if (!hasDuplicates && !hasNumbers) {
						css = '';
						xpath = textXPath;
						return { css: css, xpath: xpath };
					}
					
					// 如果有重复，使用完整 XPath
					xpath = getFullXPath(element);
					css = element.tagName.toLowerCase();
					
					// 尝试添加 nth-of-type 到 CSS
					if (element.parentNode) {
						var siblings = element.parentNode.children;
						var sameTagSiblings = [];
						for (var m = 0; m < siblings.length; m++) {
							if (siblings[m].tagName === element.tagName) {
								sameTagSiblings.push(siblings[m]);
							}
						}
						
						if (sameTagSiblings.length > 1) {
							var elementIndex = sameTagSiblings.indexOf(element);
							if (elementIndex >= 0) {
								css += ':nth-of-type(' + (elementIndex + 1) + ')';
							}
						}
					}
					
					return { css: css, xpath: xpath };
				}
			}
			
			// 策略6: 默认使用完整 XPath
			xpath = getFullXPath(element);
			
			// 策略7: CSS 选择器（包含稳定的 class）
			css = element.tagName.toLowerCase();
			
			// 只使用稳定的 class（不包含随机字符串）
			if (element.className && typeof element.className === 'string') {
				var classes = element.className.trim().split(/\s+/);
				var stableClasses = [];
				
				for (var j = 0; j < classes.length && stableClasses.length < 2; j++) {
					var cls = classes[j];
					// 排除包含随机字符的类名（长度>15或包含多个大写）
					if (cls.length > 0 && cls.length < 20 && !/[A-Z]{2,}/.test(cls) && !/[0-9]{4,}/.test(cls)) {
						stableClasses.push(cls);
					}
				}
				
				if (stableClasses.length > 0) {
					css += '.' + stableClasses.join('.');
				}
			}
			
			// 添加 type 属性
			if (element.type) {
				css += '[type="' + element.type + '"]';
			}
			
			// 添加 contenteditable 属性
			if (element.contentEditable === 'true') {
				css += '[contenteditable="true"]';
			}
			
			return { css: css, xpath: xpath };
		} catch (e) {
			console.error('[BrowserWing] getSelector error:', e);
			return { css: 'body', xpath: '//body' };
		}
	};
	
	// ============= 语义信息提取辅助函数（用于自愈） =============
	
	// 获取元素的隐式角色
	var implicitRole = function(el) {
		if (!el || !el.tagName) return 'generic';
		
		var tag = el.tagName.toLowerCase();
		var type = el.type ? el.type.toLowerCase() : '';
		
		// 常见的隐式角色映射
		if (tag === 'button') return 'button';
		if (tag === 'a') return 'link';
		if (tag === 'input') {
			if (type === 'text' || type === '') return 'textbox';
			if (type === 'email') return 'textbox';
			if (type === 'password') return 'textbox';
			if (type === 'search') return 'searchbox';
			if (type === 'tel') return 'textbox';
			if (type === 'url') return 'textbox';
			if (type === 'checkbox') return 'checkbox';
			if (type === 'radio') return 'radio';
			if (type === 'submit') return 'button';
			if (type === 'button') return 'button';
			if (type === 'file') return 'button';
		}
		if (tag === 'textarea') return 'textbox';
		if (tag === 'select') return 'combobox';
		if (tag === 'img') return 'img';
		if (tag === 'nav') return 'navigation';
		if (tag === 'header') return 'banner';
		if (tag === 'footer') return 'contentinfo';
		if (tag === 'main') return 'main';
		if (tag === 'aside') return 'complementary';
		if (tag === 'form') return 'form';
		if (tag === 'h1' || tag === 'h2' || tag === 'h3' || tag === 'h4' || tag === 'h5' || tag === 'h6') return 'heading';
		if (tag === 'ul' || tag === 'ol') return 'list';
		if (tag === 'li') return 'listitem';
		
		return 'generic';
	};
	
	// 获取元素的 ARIA 角色
	var getRole = function(el) {
		if (!el) return 'generic';
		return el.getAttribute('role') || implicitRole(el);
	};
	
	// 获取 label 元素的文本
	var getLabelText = function(el) {
		if (!el || !el.id) return '';
		var label = document.querySelector('label[for="' + el.id + '"]');
		return label ? label.innerText.trim() : '';
	};
	
	// 获取元素的可访问名称（Accessible Name）
	var getAccessibleName = function(el) {
		if (!el) return '';
		
		// 1. aria-label 优先级最高
		var ariaLabel = el.getAttribute('aria-label');
		if (ariaLabel) return ariaLabel.trim();
		
		// 2. aria-labelledby
		var labelledby = el.getAttribute('aria-labelledby');
		if (labelledby) {
			var labelElement = document.getElementById(labelledby);
			if (labelElement) return labelElement.innerText.trim();
		}
		
		// 3. 关联的 label 元素
		var labelText = getLabelText(el);
		if (labelText) return labelText;
		
		// 4. 元素自身文本（限制长度）
		if (el.innerText) {
			var text = el.innerText.trim();
			if (text.length > 50) text = text.substring(0, 50) + '...';
			return text;
		}
		
		// 5. placeholder
		if (el.placeholder) return el.placeholder.trim();
		
		// 6. title 属性
		if (el.title) return el.title.trim();
		
		// 7. alt 属性（图片等）
		if (el.alt) return el.alt.trim();
		
		// 8. value 属性（按钮等）
		if (el.value && (el.tagName === 'BUTTON' || el.tagName === 'INPUT')) {
			return el.value.trim();
		}
		
		return '';
	};
	
	// 推断操作动词
	var inferVerb = function(eventType, el) {
		if (eventType === 'click') return 'click';
		if (eventType === 'input') return 'input';
		if (eventType === 'change') {
			if (el && el.tagName === 'SELECT') return 'select';
			if (el && el.type === 'checkbox') return 'check';
			if (el && el.type === 'radio') return 'choose';
			return 'change';
		}
		if (eventType === 'submit') return 'submit';
		if (eventType === 'keydown' || eventType === 'keyup' || eventType === 'keypress') return 'type';
		return 'interact';
	};
	
	// 推断操作对象
	var inferObject = function(el) {
		if (!el) return 'element';
		
		// 优先使用可访问名称
		var name = getAccessibleName(el);
		if (name) return name;
		
		// 使用 name 属性
		if (el.name) return el.name;
		
		// 使用 id
		if (el.id) return el.id;
		
		// 使用标签名
		return el.tagName.toLowerCase();
	};
	
	// 获取附近的可见文本（用于上下文定位）
	var getNearbyText = function(el, maxDistance) {
		if (!el) return [];
		
		maxDistance = maxDistance || 100; // 默认 100px
		var rect = el.getBoundingClientRect();
		var centerX = rect.left + rect.width / 2;
		var centerY = rect.top + rect.height / 2;
		
		var nearbyTexts = [];
		var textNodes = [];
		
		// 收集页面上所有可见的文本节点
		var walker = document.createTreeWalker(
			document.body,
			NodeFilter.SHOW_TEXT,
			{
				acceptNode: function(node) {
					// 过滤掉脚本、样式等
					var parent = node.parentElement;
					if (!parent) return NodeFilter.FILTER_REJECT;
					
					var tag = parent.tagName.toLowerCase();
					if (tag === 'script' || tag === 'style' || tag === 'noscript') {
						return NodeFilter.FILTER_REJECT;
					}
					
					// 过滤掉空白文本
					var text = node.textContent.trim();
					if (!text) return NodeFilter.FILTER_REJECT;
					
					// 检查是否可见
					var style = window.getComputedStyle(parent);
					if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') {
						return NodeFilter.FILTER_REJECT;
					}
					
					return NodeFilter.FILTER_ACCEPT;
				}
			},
			false
		);
		
		var node;
		while (node = walker.nextNode()) {
			textNodes.push(node);
		}
		
		// 计算每个文本节点与目标元素的距离
		for (var i = 0; i < textNodes.length; i++) {
			var textNode = textNodes[i];
			var parent = textNode.parentElement;
			
			// 跳过目标元素内部的文本
			if (el.contains(parent)) continue;
			
			var range = document.createRange();
			range.selectNodeContents(textNode);
			var textRect = range.getBoundingClientRect();
			
			var textCenterX = textRect.left + textRect.width / 2;
			var textCenterY = textRect.top + textRect.height / 2;
			
			var distance = Math.sqrt(
				Math.pow(textCenterX - centerX, 2) + 
				Math.pow(textCenterY - centerY, 2)
			);
			
			if (distance <= maxDistance) {
				var text = textNode.textContent.trim();
				if (text.length > 30) text = text.substring(0, 30) + '...';
				nearbyTexts.push(text);
			}
		}
		
		// 去重并限制数量
		var uniqueTexts = [];
		for (var j = 0; j < nearbyTexts.length && uniqueTexts.length < 5; j++) {
			if (uniqueTexts.indexOf(nearbyTexts[j]) === -1) {
				uniqueTexts.push(nearbyTexts[j]);
			}
		}
		
		return uniqueTexts;
	};
	
	// 获取祖先标签列表（用于上下文）
	var getAncestorTags = function(el) {
		if (!el) return [];
		
		var ancestors = [];
		var current = el.parentElement;
		var depth = 0;
		var maxDepth = 10; // 最多向上查找10层
		
		while (current && depth < maxDepth) {
			var tag = current.tagName.toLowerCase();
			ancestors.push(tag);
			
			// 如果遇到 body，停止
			if (tag === 'body') break;
			
			current = current.parentElement;
			depth++;
		}
		
		return ancestors;
	};
	
	// 获取表单提示信息（判断是登录、搜索还是其他）
	var getFormHint = function(el) {
		if (!el) return '';
		
		// 向上查找最近的 form 元素
		var form = el.closest('form');
		if (!form) return '';
		
		// 检查 form 的 id, name, class
		var formId = (form.id || '').toLowerCase();
		var formName = (form.name || '').toLowerCase();
		var formClass = (form.className || '').toLowerCase();
		
		var combined = formId + ' ' + formName + ' ' + formClass;
		
		if (combined.indexOf('login') !== -1 || combined.indexOf('signin') !== -1) return 'login';
		if (combined.indexOf('register') !== -1 || combined.indexOf('signup') !== -1) return 'register';
		if (combined.indexOf('search') !== -1) return 'search';
		if (combined.indexOf('checkout') !== -1 || combined.indexOf('payment') !== -1) return 'checkout';
		if (combined.indexOf('contact') !== -1) return 'contact';
		
		// 检查 form 中的 input 类型
		var inputs = form.querySelectorAll('input');
		var hasPassword = false;
		var hasEmail = false;
		
		for (var i = 0; i < inputs.length; i++) {
			var type = inputs[i].type.toLowerCase();
			if (type === 'password') hasPassword = true;
			if (type === 'email') hasEmail = true;
		}
		
		if (hasPassword && hasEmail) return 'login';
		if (hasPassword) return 'auth';
		
		return 'generic';
	};
	
	// 获取后端 DOM 节点 ID（如果可用）
	var getBackendNodeId = function(el) {
		// 这需要通过 CDP 协议获取，在纯 JS 中无法直接获取
		// 返回 null，由 Go 端在需要时填充
		return null;
	};
	
	// 计算匹配置信度（基于选择器的可靠性）
	var calculateConfidence = function(el, selectors) {
		if (!el || !selectors) return 0.5;
		
		var confidence = 0.5; // 基础置信度
		
		// 如果有 id，置信度高
		if (el.id && selectors.css.indexOf('#' + el.id) !== -1) {
			confidence = 0.95;
		}
		// 如果有 name 属性
		else if (el.name) {
			confidence = 0.85;
		}
		// 如果有 aria-label
		else if (el.getAttribute('aria-label')) {
			confidence = 0.8;
		}
		// 如果有关联的 label
		else if (getLabelText(el)) {
			confidence = 0.75;
		}
		// 如果有稳定的 class
		else if (el.className && typeof el.className === 'string') {
			var classes = el.className.split(' ');
			var hasStableClass = false;
			for (var i = 0; i < classes.length; i++) {
				var cls = classes[i];
				// 检查是否是动态生成的 class（如 css-xxxxx）
				if (cls && !/^(css|jss|sc)-[\w-]+$/.test(cls)) {
					hasStableClass = true;
					break;
				}
			}
			if (hasStableClass) {
				confidence = 0.7;
			} else {
				confidence = 0.4;
			}
		}
		// 只能用 nth-child
		else if (selectors.css.indexOf(':nth-child') !== -1) {
			confidence = 0.3;
		}
		
		return confidence;
	};
	
	// ============= 结束：语义信息提取辅助函数 =============
	
	// 为操作添加语义信息（Intent, Accessibility, Context, Evidence）
	var enrichActionWithSemantics = function(action, element, eventType) {
		if (!element) return action;
		
		try {
			// 获取选择器（如果还没有）
			var selectors = null;
			if (!action.selector || !action.xpath) {
				selectors = getSelector(element);
			}
			
			// 1. 填充 Intent（操作意图）
			action.intent = {
				verb: inferVerb(eventType, element),
				object: inferObject(element)
			};
			
			// 2. 填充 Accessibility（可访问性信息）
			action.accessibility = {
				role: getRole(element),
				name: getAccessibleName(element),
				value: element.value || ''
			};
			
			// 3. 填充 Context（上下文信息）
			action.context = {
				nearby_text: getNearbyText(element, 100),
				ancestor_tags: getAncestorTags(element),
				form_hint: getFormHint(element)
			};
			
			// 4. 填充 Evidence（录制证据）
			action.evidence = {
				backend_dom_node_id: 0, // 将由 Go 端填充（如果需要）
				ax_node_id: '', // 将由 Go 端填充（如果需要）
				confidence: calculateConfidence(element, selectors || {css: action.selector, xpath: action.xpath})
			};
			
		} catch (e) {
			console.error('[BrowserWing] Failed to enrich action with semantics:', e);
		}
		
		return action;
	};
	
	// 记录操作的辅助函数（带去重）
	var recordAction = function(action, element, eventType) {
		// 去重逻辑：检查最近的操作是否与当前操作重复
		if (window.__recordedActions__.length > 0) {
			var lastAction = window.__recordedActions__[window.__recordedActions__.length - 1];
			
			// 如果是 scroll 类型，始终更新最后一个 scroll 操作（而不是添加新的）
			if (action.type === 'scroll' && lastAction.type === 'scroll') {
				console.log('[BrowserWing] ↻ Updated last scroll position: X=' + action.scroll_x + ', Y=' + action.scroll_y);
				lastAction.scroll_x = action.scroll_x;
				lastAction.scroll_y = action.scroll_y;
				lastAction.timestamp = action.timestamp;
				lastAction.description = action.description;
				
				// 更新 sessionStorage
				try {
					sessionStorage.setItem('__browserwing_actions__', JSON.stringify(window.__recordedActions__));
				} catch (e) {
					console.error('[BrowserWing] sessionStorage save error:', e);
				}
				
				// 更新 UI 显示
				updateActionCount();
				return; // 不添加新操作
			}
			
			// 如果是 input 类型，检查是否与最后一个操作重复
			if (action.type === 'input' && lastAction.type === 'input') {
				// 相同选择器、相同标签、相同值，且时间间隔小于 2 秒，认为是重复
				var timeDiff = action.timestamp - lastAction.timestamp;
				var isSameSelector = (action.selector === lastAction.selector || action.xpath === lastAction.xpath);
				var isSameValue = action.value === lastAction.value;
				
				if (isSameSelector && isSameValue && timeDiff < 2000) {
					console.log('[BrowserWing] ⊘ Skipped duplicate input action');
					return; // 跳过重复操作
				}
				
				// 如果选择器相同但值不同，更新最后一个操作的值（而不是添加新操作）
				if (isSameSelector && !isSameValue && timeDiff < 2000) {
					console.log('[BrowserWing] ↻ Updated last input action value');
					lastAction.value = action.value;
					lastAction.timestamp = action.timestamp;
					
					// 更新 sessionStorage
					try {
						sessionStorage.setItem('__browserwing_actions__', JSON.stringify(window.__recordedActions__));
					} catch (e) {
						console.error('[BrowserWing] sessionStorage save error:', e);
					}
					return; // 不添加新操作
				}
			}
			
		// 自动插入 sleep：如果两个操作间隔超过 1 秒，插入 sleep action
		var timeDiff = action.timestamp - lastAction.timestamp;
		if (timeDiff > 1000 && lastAction.type !== 'sleep') {
			var sleepDuration = Math.round(Math.round(timeDiff) / 3);
			// 最长为5秒
			if (sleepDuration > 5000) {
				sleepDuration = 5000;
			}
			
			// 创建 sleep 操作
			var sleepAction = {
				type: 'sleep',
				timestamp: lastAction.timestamp + 1, // 紧跟在上一个操作之后
				duration: sleepDuration,
				description: '{{AUTO_WAIT}}' + (sleepDuration / 1000).toFixed(1) + ' {{SECONDS_UNIT}}'
			};				window.__recordedActions__.push(sleepAction);
				console.log('[BrowserWing] ⏱ Auto-inserted sleep action: ' + sleepDuration + 'ms');
				
				// 更新 UI（添加 sleep action）
				updateActionCount();
				addActionToList(sleepAction, window.__recordedActions__.length - 1);
			}
		}
		
		// 添加语义信息（自愈所需）
		if (element && eventType) {
			action = enrichActionWithSemantics(action, element, eventType);
		}
		
		// 添加新操作
		window.__recordedActions__.push(action);
		console.log('[BrowserWing] Recorded action #' + window.__recordedActions__.length + ':', action.type, 'on', action.tagName);
		
		// 更新 UI
		updateActionCount();
		addActionToList(action, window.__recordedActions__.length - 1);
		
		// 立即将操作保存到 sessionStorage，防止页面刷新丢失
		try {
			sessionStorage.setItem('__browserwing_actions__', JSON.stringify(window.__recordedActions__));
		} catch (e) {
			console.error('[BrowserWing] sessionStorage save error:', e);
		}
	};
	
	// 页面加载时恢复之前的录制（如果有）
	try {
		var savedActions = sessionStorage.getItem('__browserwing_actions__');
		if (savedActions) {
			var parsed = JSON.parse(savedActions);
			if (Array.isArray(parsed) && parsed.length > 0) {
				window.__recordedActions__ = parsed;
				console.log('[BrowserWing] Restored ' + parsed.length + ' previous actions');
				
				// 初始化 UI 后重建动作列表
				setTimeout(function() {
					if (window.__recorderUI__) {
						updateActionCount();
						for (var i = 0; i < parsed.length; i++) {
							addActionToList(parsed[i], i);
						}
					}
				}, 100);
			}
		}
	} catch (e) {
		console.error('[BrowserWing] sessionStorage restore error:', e);
	}
	
	// 监听页面卸载事件，最后保存一次
	window.addEventListener('beforeunload', function() {
		try {
			if (window.__recordedActions__ && window.__recordedActions__.length > 0) {
				sessionStorage.setItem('__browserwing_actions__', JSON.stringify(window.__recordedActions__));
				console.log('[BrowserWing] Saved ' + window.__recordedActions__.length + ' actions before unload');
			}
		} catch (e) {
			console.error('[BrowserWing] beforeunload save error:', e);
		}
	});
	
	// 初始化逻辑 - 根据是否有保存的录制决定显示哪个UI
	var hasRecordedActions = false;
	var wasRecording = false;
	try {
		var savedActions = sessionStorage.getItem('__browserwing_actions__');
		if (savedActions) {
			var parsed = JSON.parse(savedActions);
			hasRecordedActions = Array.isArray(parsed) && parsed.length > 0;
		}
		
		// 检查是否有持久化的录制状态标志（跨页面）
		var recordingState = sessionStorage.getItem('__browserwing_recording_state__');
		if (recordingState === 'active') {
			wasRecording = true;
			console.log('[BrowserWing] Detected active recording state from previous page');
		}
	} catch (e) {
		// 忽略错误
	}
	
	// 检查是否是从页面内启动的录制或从上一个页面继承的录制状态
	var isRecordingMode = window.__browserwingRecordingMode__ === true || wasRecording;
	
	// 如果进入录制模式,保存状态到sessionStorage以便跨页面保持
	if (isRecordingMode) {
		try {
			sessionStorage.setItem('__browserwing_recording_state__', 'active');
			console.log('[BrowserWing] Recording state persisted to sessionStorage');
		} catch (e) {
			console.error('[BrowserWing] Failed to persist recording state:', e);
		}
	}
	
	if (isRecordingMode || hasRecordedActions) {
		// 录制模式：显示录制控制面板
		window.__isRecordingActive__ = true;
		window.__browserwingRecordingMode__ = true; // 确保在当前页面也设置此标志
		createRecorderUI();
		createHighlightElement();
		console.log('[BrowserWing] Recording UI restored after page navigation');
	} else {
		// 非录制模式：只显示浮动录制按钮
		// 注意: 浮动按钮由float_button.js单独注入,这里不需要创建
		console.log('[BrowserWing] Not in recording mode, floating button should be present.');
	}
	
	// 鼠标悬停事件 - 高亮元素（仅在录制模式下）
	document.addEventListener('mouseover', function(e) {
		if (!window.__isRecordingActive__) return;
		
		try {
			var target = e.target || e.srcElement;
			if (!target || !target.tagName) return;
			
		// 忽略录制器 UI 自身
		if (target.id && target.id.indexOf('__browserwing_') === 0) return;
		if (target.closest && target.closest('#__browserwing_recorder_panel__')) return;
		if (target.closest && target.closest('#__browserwing_extract_menu__')) return;
		if (target.closest && target.closest('#__browserwing_preview_dialog__')) return;
		
		highlightElement(target);
		} catch (err) {
			console.error('[BrowserWing] mouseover event error:', err);
		}
	});
	
	document.addEventListener('mouseout', function(e) {
		if (!window.__isRecordingActive__) return;
		hideHighlight();
	});
	
	// 监听点击事件 - 使用capture模式记录操作
	document.addEventListener('click', function(e) {
		if (!window.__isRecordingActive__) return;
		
		try {
			var target = e.target || e.srcElement;
			if (!target || !target.tagName) return;
			
		// 忽略录制器 UI 自身的点击
		if (target.id && target.id.indexOf('__browserwing_') === 0) return;
		if (target.closest && target.closest('#__browserwing_recorder_panel__')) return;
		if (target.closest && target.closest('#__browserwing_extract_menu__')) return;
		if (target.closest && target.closest('#__browserwing_preview_dialog__')) return;
			
			// 如果在 AI 填充表单模式下，阻止默认行为并调用 AI 生成
			if (window.__aiFormFillMode__) {
				e.preventDefault();
				e.stopPropagation();
				handleAIFormFillClick(target);
				return false;
			}
			
			// 如果在 AI 提取模式下，阻止默认行为并调用 AI 生成
			if (window.__aiExtractMode__) {
				e.preventDefault();
				e.stopPropagation();
				handleAIExtractClick(target);
				return false;
			}
			
			// 如果在抓取模式下，阻止默认行为并记录抓取操作
			if (window.__extractMode__) {
				e.preventDefault();
				e.stopPropagation();
				
				// 使用当前选择的抓取类型
				var extractType = window.__extractType__ || 'text';
				if (extractType === 'attribute') {
					// 如果是属性抓取，弹出对话框让用户输入属性名
					var attrName = prompt('{{PROMPT_ATTRIBUTE}}', 'href');
					if (attrName) {
						recordExtractAction(target, extractType, attrName);
					}
				} else {
					recordExtractAction(target, extractType, null);
				}
				return false;
			}
			
			// 普通录制模式：不阻止事件传播，让原来的点击事件正常执行
			
			// 检查是否点击了文件上传按钮（input[type=file] 或者触发文件选择的按钮）
			var isFileInput = target.tagName.toLowerCase() === 'input' && target.type === 'file';
			var fileInput = null;
			
			// 如果点击的不是 file input 本身，检查是否有关联的 file input
			if (!isFileInput) {
				// 1. 检查是否通过 label 关联
				if (target.tagName.toLowerCase() === 'label' && target.htmlFor) {
					var associated = document.getElementById(target.htmlFor);
					if (associated && associated.tagName.toLowerCase() === 'input' && associated.type === 'file') {
						fileInput = associated;
						isFileInput = true;
						console.log('[BrowserWing] Found file input via label.htmlFor');
					}
				}
				
				// 2. 检查是否是包含 file input 的 label
				if (!isFileInput && target.tagName.toLowerCase() === 'label') {
					var inputInLabel = target.querySelector('input[type="file"]');
					if (inputInLabel) {
						fileInput = inputInLabel;
						isFileInput = true;
						console.log('[BrowserWing] Found file input inside label');
					}
				}
				
				// 3. 检查是否点击的元素内部**直接**包含 file input（限制为直接子元素）
				if (!isFileInput) {
					// 只检查直接子元素，避免误判
					var directChildren = target.children;
					for (var i = 0; i < directChildren.length; i++) {
						if (directChildren[i].tagName.toLowerCase() === 'input' && directChildren[i].type === 'file') {
							fileInput = directChildren[i];
							isFileInput = true;
							console.log('[BrowserWing] Found file input as direct child');
							break;
						}
					}
				}
				
				// 4. 查找最近的父元素中的 file input（限制为 2 层，且必须是上传相关的容器）
				if (!isFileInput) {
					var parent = target.parentElement;
					var depth = 0;
					while (parent && depth < 2) {
						// 检查父元素的类名或属性，确保是上传相关的容器
						var className = parent.className || '';
						var isUploadContainer = 
							className.indexOf('upload') !== -1 || 
							className.indexOf('file') !== -1 ||
							parent.getAttribute('data-type') === 'upload' ||
							parent.tagName.toLowerCase() === 'label';
						
						if (isUploadContainer) {
							var inputs = parent.querySelectorAll('input[type="file"]');
							if (inputs.length > 0) {
								fileInput = inputs[0];
								isFileInput = true;
								console.log('[BrowserWing] Found file input in upload container (depth=' + depth + ')');
								break;
							}
						}
						parent = parent.parentElement;
						depth++;
					}
				}
			} else {
				fileInput = target;
				console.log('[BrowserWing] Clicked directly on file input');
			}
			
			// 如果是文件上传，设置监听器等待文件选择
			if (isFileInput && fileInput) {
				// 检查是否已经添加过监听器(防止重复)
				if (fileInput.__browserwing_listener_added__) {
					console.log('[BrowserWing] File input listener already added, skipping');
					return; // 不记录 click 事件
				}
				
				console.log('[BrowserWing] File input detected:', {
					tagName: fileInput.tagName,
					type: fileInput.type,
					name: fileInput.name,
					id: fileInput.id,
					className: fileInput.className
				});
				
				// 标记已添加监听器
				fileInput.__browserwing_listener_added__ = true;
				
				var selectors = getSelector(fileInput);
				
				// 监听 change 事件记录上传的文件
				var changeHandler = function(changeEvent) {
					console.log('[BrowserWing] File input change event fired');
					
					// 标记此事件已被处理，防止全局 change 事件重复处理
					changeEvent.__browserwing_handled__ = true;
					
					var files = changeEvent.target.files;
					if (files && files.length > 0) {
						var fileNames = [];
						for (var i = 0; i < files.length; i++) {
							fileNames.push(files[i].name);
						}
						
						console.log('[BrowserWing] Recording file upload action, files:', fileNames);
						
						var action = {
							type: 'upload_file',
							timestamp: Date.now(),
							selector: selectors.css,
							xpath: selectors.xpath,
							tagName: fileInput.tagName ? fileInput.tagName.toLowerCase() : 'input',
							file_names: fileNames,
							multiple: fileInput.multiple || false,
							accept: fileInput.accept || '',
							description: '{{FILES_SELECTED}}' + fileNames.length + ' {{FILES_COUNT}}' + fileNames.join(', ')
						};
						
						recordAction(action, fileInput, 'change');
						showCurrentAction('{{UPLOAD_FILE}}' + fileNames.join(', '));
					} else {
						console.log('[BrowserWing] No files selected');
					}
					
					// 清除标记，允许下次使用
					delete fileInput.__browserwing_listener_added__;
				};
				
				// 添加 change 事件监听器
				fileInput.addEventListener('change', changeHandler, { once: true });
				
				console.log('[BrowserWing] File input change listener added, waiting for file selection...');
				return; // 不记录 click 事件，等待 change 事件
			}
			
			// 普通点击事件
			var selectors = getSelector(target);
			var action = {
				type: 'click',
				timestamp: Date.now(),
				selector: selectors.css,
				xpath: selectors.xpath,
				text: (target.innerText || target.textContent || '').substring(0, 50),
				tagName: target.tagName ? target.tagName.toLowerCase() : '',
				x: e.clientX || 0,
				y: e.clientY || 0
			};
			
			recordAction(action, target, 'click');
			
			var actionText = 'Clicked <' + action.tagName + '>';
			if (action.text) {
				actionText += ' "' + action.text.substring(0, 20) + '"';
			}
			showCurrentAction(actionText);
		} catch (err) {
			console.error('[BrowserWing] click event error:', err);
		}
	}, true);
	
	// 监听输入事件（使用防抖，避免录制每个字符）
	document.addEventListener('input', function(e) {
		if (!window.__isRecordingActive__) return;
		
		try {
			var target = e.target || e.srcElement;
			if (!target) return;
			
			var tagName = target.tagName ? target.tagName.toUpperCase() : '';
			var isContentEditable = target.contentEditable === 'true' || target.isContentEditable;
			
			// 支持 INPUT、TEXTAREA 和 contenteditable 元素
			if (tagName !== 'INPUT' && tagName !== 'TEXTAREA' && !isContentEditable) return;
			
			// 排除文件输入框（文件选择由 change 事件处理）
			if (tagName === 'INPUT' && target.type === 'file') {
				console.log('[BrowserWing] Ignoring input event on file input');
				return;
			}
			
			var selectors = getSelector(target);
			var selectorKey = selectors.css || selectors.xpath;  // 用作定时器 key
			
			// 检查上一个动作是否是 Ctrl+V 粘贴（针对同一个元素）
			if (window.__recordedActions__.length > 0) {
				var lastAction = window.__recordedActions__[window.__recordedActions__.length - 1];
				// 如果上一个动作是 ctrl+v，且目标元素相同，则跳过 input 录制
				if (lastAction.type === 'keyboard' && lastAction.key === 'ctrl+v') {
					var lastSelector = lastAction.selector || lastAction.xpath;
					if (lastSelector && (lastSelector === selectors.css || lastSelector === selectors.xpath)) {
						console.log('[BrowserWing] Skipping input event after ctrl+v on same element');
						return;
					}
				}
			}
			
			// 清除之前的定时器
			if (window.__inputTimers__[selectorKey]) {
				clearTimeout(window.__inputTimers__[selectorKey]);
			}
			
			// 获取内容：对于 contenteditable 使用 textContent 或 innerText
			var content = '';
			if (isContentEditable) {
				content = target.textContent || target.innerText || '';
			} else {
				content = target.value || '';
			}
			
			// 设置新的定时器，500ms 后记录（防抖）
			window.__inputTimers__[selectorKey] = setTimeout(function() {
				recordAction({
					type: 'input',
					timestamp: Date.now(),
					selector: selectors.css,
					xpath: selectors.xpath,
					value: content,
					tagName: isContentEditable ? 'contenteditable' : tagName.toLowerCase()
				}, target, 'input');
			}, 500);
		} catch (err) {
			console.error('[BrowserWing] input event error:', err);
		}
	}, true);
	
	// 监听焦点事件（记录输入框的最终值）
	document.addEventListener('blur', function(e) {
		if (!window.__isRecordingActive__) return;
		
		try {
			var target = e.target || e.srcElement;
			if (!target) return;
			
			var tagName = target.tagName ? target.tagName.toUpperCase() : '';
			var isContentEditable = target.contentEditable === 'true' || target.isContentEditable;
			
			// 支持 INPUT、TEXTAREA 和 contenteditable 元素
			if (tagName !== 'INPUT' && tagName !== 'TEXTAREA' && !isContentEditable) return;
			
			var selectors = getSelector(target);
			var selectorKey = selectors.css || selectors.xpath;
			
			// 检查上一个动作是否是 Ctrl+V 粘贴（针对同一个元素）
			if (window.__recordedActions__.length > 0) {
				var lastAction = window.__recordedActions__[window.__recordedActions__.length - 1];
				// 如果上一个动作是 ctrl+v，且目标元素相同，则跳过 blur 时的 input 录制
				if (lastAction.type === 'keyboard' && lastAction.key === 'ctrl+v') {
					var lastSelector = lastAction.selector || lastAction.xpath;
					if (lastSelector && (lastSelector === selectors.css || lastSelector === selectors.xpath)) {
						console.log('[BrowserWing] Skipping blur input event after ctrl+v on same element');
						// 清除定时器
						if (window.__inputTimers__[selectorKey]) {
							clearTimeout(window.__inputTimers__[selectorKey]);
							delete window.__inputTimers__[selectorKey];
						}
						return;
					}
				}
			}
			
			// 清除防抖定时器
			if (window.__inputTimers__[selectorKey]) {
				clearTimeout(window.__inputTimers__[selectorKey]);
				delete window.__inputTimers__[selectorKey];
			}
			
			// 立即记录最终值
			var content = '';
			if (isContentEditable) {
				content = target.textContent || target.innerText || '';
			} else {
				content = target.value || '';
			}
			
			// 只在有内容时才记录
			if (content && content.trim().length > 0) {
				recordAction({
					type: 'input',
					timestamp: Date.now(),
					selector: selectors.css,
					xpath: selectors.xpath,
					value: content,
					tagName: isContentEditable ? 'contenteditable' : tagName.toLowerCase()
				}, target, 'blur');
			}
		} catch (err) {
			console.error('[BrowserWing] blur event error:', err);
		}
	}, true);
	
	// 监听 DOMCharacterDataModified 事件（某些富文本编辑器使用）
	document.addEventListener('DOMCharacterDataModified', function(e) {
		if (!window.__isRecordingActive__) return;
		
		try {
			var target = e.target;
			if (!target) return;
			
			// 查找最近的 contenteditable 祖先元素
			var editableParent = target.parentElement;
			while (editableParent && editableParent.contentEditable !== 'true' && !editableParent.isContentEditable) {
				editableParent = editableParent.parentElement;
				if (!editableParent || editableParent === document.body) break;
			}
			
			if (!editableParent || (editableParent.contentEditable !== 'true' && !editableParent.isContentEditable)) return;
			
			var selectors = getSelector(editableParent);
			var selectorKey = selectors.css || selectors.xpath;
			
			if (window.__inputTimers__[selectorKey]) {
				clearTimeout(window.__inputTimers__[selectorKey]);
			}
			
			var content = editableParent.textContent || editableParent.innerText || '';
			
			window.__inputTimers__[selectorKey] = setTimeout(function() {
				recordAction({
					type: 'input',
					timestamp: Date.now(),
					selector: selectors.css,
					xpath: selectors.xpath,
					value: content,
					tagName: 'contenteditable'
				}, editableParent, 'input');
			}, 500);
		} catch (err) {
			console.error('[BrowserWing] DOMCharacterDataModified event error:', err);
		}
	}, true);
	
	// 监听选择事件
	document.addEventListener('change', function(e) {
		if (!window.__isRecordingActive__) return;
		
		try {
			var target = e.target || e.srcElement;
			if (!target) return;
			
			var tagName = target.tagName ? target.tagName.toUpperCase() : '';
			
			// 文件上传 - 这里作为备份处理(主要由 click 事件中的监听器处理)
			if (tagName === 'INPUT' && target.type === 'file') {
				// 检查事件是否已被 click 监听器处理过
				if (e.__browserwing_handled__) {
					console.log('[BrowserWing] File upload already handled by click listener, skipping');
					return;
				}
				
				console.log('[BrowserWing] Global change event detected file input (backup handler)');
				var files = target.files;
				if (files && files.length > 0) {
					var fileNames = [];
					for (var i = 0; i < files.length; i++) {
						fileNames.push(files[i].name);
					}
					
					console.log('[BrowserWing] Recording file upload from global change, files:', fileNames);
					
					var selectors = getSelector(target);
					var action = {
						type: 'upload_file',
						timestamp: Date.now(),
						selector: selectors.css,
						xpath: selectors.xpath,
						tagName: 'input',
						file_names: fileNames,
						multiple: target.multiple || false,
						accept: target.accept || '',
						description: '{{FILES_SELECTED}}' + fileNames.length + ' {{FILES_COUNT}}' + fileNames.join(', ')
					};
					
					recordAction(action, target, 'change');
					showCurrentAction('{{UPLOAD_FILE}}' + fileNames.join(', '));
				}
				return;
			}
			
			if (tagName === 'SELECT') {
				var selectedText = '';
				if (target.options && target.selectedIndex >= 0 && target.options[target.selectedIndex]) {
					selectedText = target.options[target.selectedIndex].text || '';
				}
				
				var selectors = getSelector(target);
				recordAction({
					type: 'select',
					timestamp: Date.now(),
					selector: selectors.css,
					xpath: selectors.xpath,
					value: target.value || '',
					text: selectedText,
					tagName: 'select'
				}, target, 'change');
			} else if (tagName === 'INPUT' && (target.type === 'checkbox' || target.type === 'radio')) {
				// 记录复选框和单选框的变化
				var selectors = getSelector(target);
				recordAction({
					type: 'input',
					timestamp: Date.now(),
					selector: selectors.css,
					xpath: selectors.xpath,
					value: target.checked ? 'checked' : 'unchecked',
					tagName: tagName.toLowerCase()
				}, target, 'change');
			}
		} catch (err) {
			console.error('[BrowserWing] change event error:', err);
		}
	}, true);

	// ============= 右键菜单支持（抓取模式） =============
	document.addEventListener('contextmenu', function(e) {
		if (!window.__extractMode__) return;
		
		var target = e.target || e.srcElement;
		if (!target || !target.tagName) return;
		
		// 忽略录制器 UI 自身
		if (target.id && target.id.indexOf('__browserwing_') === 0) return;
		if (target.closest && target.closest('#__browserwing_recorder_panel__')) return;
		if (target.closest && target.closest('#__browserwing_extract_menu__')) return;
		if (target.closest && target.closest('#__browserwing_preview_dialog__')) return;
		
		e.preventDefault();
		e.stopPropagation();
		
		var ui = window.__recorderUI__;
		ui.currentElement = target;
		
		// 设置菜单触发方式为右键
		window.__menuTrigger__ = 'contextmenu';
		
		// 显示菜单
		ui.menu.style.display = 'block';
		ui.menu.style.left = e.pageX + 'px';
		ui.menu.style.top = e.pageY + 'px';
		
		return false;
	}, true);
	
	// ============= 键盘事件监听 =============
	// 监听键盘事件 - 支持 Ctrl+C、Ctrl+V、Enter
	document.addEventListener('keydown', function(e) {
		if (!window.__isRecordingActive__) return;
		
		try {
			var target = e.target || e.srcElement;
			if (!target) return;
			
		// 忽略录制器 UI 自身的键盘事件
		if (target.id && target.id.indexOf('__browserwing_') === 0) return;
		if (target.closest && target.closest('#__browserwing_recorder_panel__')) return;
		if (target.closest && target.closest('#__browserwing_preview_dialog__')) return;
			
			var keyAction = null;
			
			// 检测 Ctrl+A (Windows/Linux) 或 Cmd+A (Mac)
			if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
				keyAction = {
					type: 'keyboard',
					timestamp: Date.now(),
					key: 'ctrl+a',
					description: '{{KEYBOARD_SELECT_ALL}}',
					tagName: target.tagName ? target.tagName.toLowerCase() : ''
				};
				
				// 如果在输入框或contenteditable中，记录选择器
				if (target.tagName) {
					var tagName = target.tagName.toLowerCase();
					if (tagName === 'input' || tagName === 'textarea' || target.contentEditable === 'true') {
						var selectors = getSelector(target);
						keyAction.selector = selectors.css;
						keyAction.xpath = selectors.xpath;
					}
				}
			}
			// 检测 Ctrl+C (Windows/Linux) 或 Cmd+C (Mac)
			else if ((e.ctrlKey || e.metaKey) && e.key === 'c') {
				keyAction = {
					type: 'keyboard',
					timestamp: Date.now(),
					key: 'ctrl+c',
					description: '{{KEYBOARD_COPY}}',
					tagName: target.tagName ? target.tagName.toLowerCase() : ''
				};
				
				// 如果在输入框或contenteditable中，记录选择器
				if (target.tagName) {
					var tagName = target.tagName.toLowerCase();
					if (tagName === 'input' || tagName === 'textarea' || target.contentEditable === 'true') {
						var selectors = getSelector(target);
						keyAction.selector = selectors.css;
						keyAction.xpath = selectors.xpath;
					}
				}
			}
			// 检测 Ctrl+V (Windows/Linux) 或 Cmd+V (Mac)
			else if ((e.ctrlKey || e.metaKey) && e.key === 'v') {
				keyAction = {
					type: 'keyboard',
					timestamp: Date.now(),
					key: 'ctrl+v',
					description: '{{KEYBOARD_PASTE}}',
					tagName: target.tagName ? target.tagName.toLowerCase() : ''
				};
				
				// 如果在输入框或contenteditable中，记录选择器
				if (target.tagName) {
					var tagName = target.tagName.toLowerCase();
					if (tagName === 'input' || tagName === 'textarea' || target.contentEditable === 'true') {
						var selectors = getSelector(target);
						keyAction.selector = selectors.css;
						keyAction.xpath = selectors.xpath;
					}
				}
			}
			// 检测 Backspace 键
			else if (e.key === 'Backspace' || e.keyCode === 8) {
				keyAction = {
					type: 'keyboard',
					timestamp: Date.now(),
					key: 'backspace',
					description: '{{KEYBOARD_BACKSPACE}}',
					tagName: target.tagName ? target.tagName.toLowerCase() : ''
				};
				
				// 如果在输入框或contenteditable中，记录选择器
				if (target.tagName) {
					var tagName = target.tagName.toLowerCase();
					if (tagName === 'input' || tagName === 'textarea' || target.contentEditable === 'true') {
						var selectors = getSelector(target);
						keyAction.selector = selectors.css;
						keyAction.xpath = selectors.xpath;
					}
				}
			}
			// 检测 Tab 键
			else if (e.key === 'Tab' || e.keyCode === 9) {
				keyAction = {
					type: 'keyboard',
					timestamp: Date.now(),
					key: 'tab',
					description: '{{KEYBOARD_TAB}}',
					tagName: target.tagName ? target.tagName.toLowerCase() : ''
				};
				
				// 记录目标元素的选择器
				if (target.tagName) {
					var selectors = getSelector(target);
					keyAction.selector = selectors.css;
					keyAction.xpath = selectors.xpath;
				}
			}
			// 检测 Enter 键
			else if (e.key === 'Enter' || e.keyCode === 13) {
				keyAction = {
					type: 'keyboard',
					timestamp: Date.now(),
					key: 'enter',
					description: '{{KEYBOARD_ENTER}}',
					tagName: target.tagName ? target.tagName.toLowerCase() : ''
				};
				
				// 记录目标元素的选择器
				if (target.tagName) {
					var selectors = getSelector(target);
					keyAction.selector = selectors.css;
					keyAction.xpath = selectors.xpath;
				}
			}
			
			// 如果识别到需要记录的按键，记录动作
			if (keyAction) {
				recordAction(keyAction, target, 'keydown');
				showCurrentAction(keyAction.description);
				console.log('[BrowserWing] Recorded keyboard action:', keyAction.key);
			}
		} catch (err) {
			console.error('[BrowserWing] keydown event error:', err);
		}
	}, true);
	
	// 点击其他地方关闭菜单
	document.addEventListener('click', function(e) {
		if (window.__recorderUI__) {
			var menu = window.__recorderUI__.menu;
			if (menu && menu.style.display !== 'none') {
				// 如果点击的不是菜单项，关闭菜单
				if (!e.target.closest('#__browserwing_extract_menu__')) {
					menu.style.display = 'none';
				}
			}
		}
	}, true);

	// ============= 滚动事件监听（防抖） =============
	var scrollDebounceTimer = null;
	var lastScrollX = window.scrollX || window.pageXOffset || 0;
	var lastScrollY = window.scrollY || window.pageYOffset || 0;
	
	document.addEventListener('scroll', function(e) {
		if (!window.__isRecordingActive__) return;
		
		// 清除之前的定时器
		if (scrollDebounceTimer) {
			clearTimeout(scrollDebounceTimer);
		}
		
		// 设置新的定时器，500ms 后记录滚动位置
		scrollDebounceTimer = setTimeout(function() {
			try {
				var currentScrollX = window.scrollX || window.pageXOffset || 0;
				var currentScrollY = window.scrollY || window.pageYOffset || 0;
				
				// 只有当滚动位置真正变化时才记录
				if (currentScrollX !== lastScrollX || currentScrollY !== lastScrollY) {
					var action = {
						type: 'scroll',
						timestamp: Date.now(),
						scroll_x: Math.round(currentScrollX),
						scroll_y: Math.round(currentScrollY),
					description: '{{SCROLL_TO}}' + ' X:' + Math.round(currentScrollX) + ', Y:' + Math.round(currentScrollY)
				};
				
				recordAction(action, document.documentElement || document.body, 'scroll');
				showCurrentAction('{{SCROLL_TO}}' + ' X:' + Math.round(currentScrollX) + ', Y:' + Math.round(currentScrollY));					lastScrollX = currentScrollX;
					lastScrollY = currentScrollY;
				}
			} catch (err) {
				console.error('[BrowserWing] scroll event error:', err);
			}
		}, 500); // 500ms 防抖延迟
	}, true);

	console.log('[BrowserWing] Recorder initialized successfully');
	console.log('[BrowserWing] Monitoring: click, input, select, checkbox, radio, contenteditable, scroll');
	console.log('[BrowserWing] Extract mode available');
}
