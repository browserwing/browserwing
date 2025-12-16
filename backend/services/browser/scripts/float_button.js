// 浮动录制按钮脚本 - 只在OpenPage时注入
if (!window.__browserwingFloatButton__) {
	createFloatingRecordButton()
}

function createFloatingRecordButton() {
    window.__browserwingFloatButton__ = true;
	
	// 创建主面板 - 类似录制面板风格
	var panel = document.createElement('div');
	panel.id = '__browserwing_float_panel__';
	panel.style.cssText = 'position: fixed !important;top: 20px !important;right: 20px !important;z-index: 2147483647 !important;font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif !important;width: 280px !important;background: white !important;border-radius: 8px !important;box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15) !important;border: 1px solid #e5e7eb !important;overflow: hidden !important;opacity: 1 !important;visibility: visible !important;';
	
	// 创建头部区域（可拖动）
	var header = document.createElement('div');
	header.style.cssText = 'padding: 14px 16px !important;background: #fafafa !important;cursor: move !important;user-select: none !important;display: flex !important;align-items: center !important;justify-content: center !important;border-bottom: 1px solid #e5e7eb !important;';
	
	var title = document.createElement('div');
	title.style.cssText = 'color: #1f2937 !important;font-size: 14px !important;font-weight: 600 !important;opacity: 1 !important;visibility: visible !important;';
	title.textContent = '{{TITLE}}';
	
	header.appendChild(title);
	
	// 创建按钮区域
	var buttonArea = document.createElement('div');
	buttonArea.style.cssText = 'padding: 16px !important;background: white !important;opacity: 1 !important;visibility: visible !important;';
	
	// 开始录制按钮
	var startBtn = document.createElement('button');
	startBtn.id = '__browserwing_start_record_btn__';
	startBtn.style.cssText = 'width: 100% !important;padding: 12px 16px !important;background: #dc2626 !important;color: white !important;border: none !important;border-radius: 6px !important;cursor: pointer !important;font-size: 14px !important;font-weight: 600 !important;transition: all 0.2s !important;display: flex !important;align-items: center !important;justify-content: center !important;gap: 8px !important;opacity: 1 !important;visibility: visible !important;';
	
	// 录制图标
	var icon = document.createElement('div');
	icon.style.cssText = 'width: 8px !important;height: 8px !important;border-radius: 50% !important;background: white !important;opacity: 1 !important;visibility: visible !important;flex-shrink: 0 !important;';
	
	var btnText = document.createElement('span');
	btnText.style.cssText = 'opacity: 1 !important;visibility: visible !important;';
	btnText.textContent = '{{START_RECORD}}';
	
	startBtn.appendChild(icon);
	startBtn.appendChild(btnText);
	
	// 悬停效果
	startBtn.onmouseover = function() {
		this.style.background = '#b91c1c';
	};
	startBtn.onmouseout = function() {
		this.style.background = '#dc2626';
	};
	
	// 点击事件
	startBtn.onclick = function() {
		if (!panel.__isDragging) {
			// 使用轮询方式通知后端,而不是直接调用API
			window.__startRecordingRequest__ = {
				timestamp: Date.now(),
				action: 'start'
			};
			console.log('[BrowserWing] Recording start request set');
			
			// 隐藏面板
			panel.style.display = 'none';
		}
	};
	
	buttonArea.appendChild(startBtn);
	
	// 组装面板
	panel.appendChild(header);
	panel.appendChild(buttonArea);
	
	// 拖动功能
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
		e.preventDefault();
	});
	
	document.addEventListener('mousemove', function(e) {
		if (isDragging) {
			e.preventDefault();
			currentX = e.clientX - initialX;
			currentY = e.clientY - initialY;
			xOffset = currentX;
			yOffset = currentY;
			
			// 设置移动标志，防止拖动时触发点击
			if (Math.abs(currentX) > 5 || Math.abs(currentY) > 5) {
				panel.__isDragging = true;
			}
			
			panel.style.transform = 'translate(' + currentX + 'px, ' + currentY + 'px)';
		}
	});
	
	document.addEventListener('mouseup', function() {
		if (isDragging) {
			// 延迟重置拖动标志，防止立即触发点击
			setTimeout(function() {
				panel.__isDragging = false;
			}, 100);
		}
		isDragging = false;
	});
	
	document.body.appendChild(panel);
	console.log('[BrowserWing] Browserwing Pilot panel initialized');
}