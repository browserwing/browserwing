// 监听来自 iframe 的消息
(function() {
	if (window.__browserwingIframeListener__) {
		return;
	}
	window.__browserwingIframeListener__ = true;
	
	window.addEventListener('message', function(event) {
		try {
			if (event.data && event.data.type === '__browserwing_iframe_action__') {
				var action = event.data.action;
				if (action && window.__recordedActions__) {
					// 去重逻辑：检查最近的操作是否与当前操作重复
					var shouldRecord = true;
					
					if (window.__recordedActions__.length > 0) {
						var lastAction = window.__recordedActions__[window.__recordedActions__.length - 1];
						
						// 如果是 input 类型，检查是否与最后一个操作重复
						if (action.type === 'input' && lastAction.type === 'input') {
							var timeDiff = action.timestamp - lastAction.timestamp;
							var isSameSelector = (action.selector === lastAction.selector || action.xpath === lastAction.xpath);
							var isSameValue = action.value === lastAction.value;
							
							// 相同选择器、相同值，且时间间隔小于 2 秒，认为是重复
							if (isSameSelector && isSameValue && timeDiff < 2000) {
								console.log('[BrowserWing] ⊘ Skipped duplicate iframe input action');
								shouldRecord = false;
							}
							// 如果选择器相同但值不同，更新最后一个操作的值
							else if (isSameSelector && !isSameValue && timeDiff < 2000) {
								console.log('[BrowserWing] ↻ Updated last iframe input action value');
								lastAction.value = action.value;
								lastAction.timestamp = action.timestamp;
								
								// 更新 sessionStorage
								try {
									sessionStorage.setItem('__browserwing_actions__', JSON.stringify(window.__recordedActions__));
								} catch (e) {
									console.error('[BrowserWing] sessionStorage update error:', e);
								}
								shouldRecord = false;
							}
						}
					}
					
					// 记录新操作
					if (shouldRecord) {
						window.__recordedActions__.push(action);
						console.log('[BrowserWing] ✓ Recorded iframe action #' + window.__recordedActions__.length + ':', action.type);
						
						// 保存到 sessionStorage
						try {
							sessionStorage.setItem('__browserwing_actions__', JSON.stringify(window.__recordedActions__));
						} catch (e) {
							console.error('[BrowserWing] sessionStorage save error:', e);
						}
					}
				}
			}
		} catch (e) {
			console.error('[BrowserWing] iframe message listener error:', e);
		}
	});
	
	console.log('[BrowserWing] iframe message listener initialized');
})();
