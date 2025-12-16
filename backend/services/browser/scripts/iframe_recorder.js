// 为 iframe 注入录制器
(function() {
	// 检查是否已注入
	if (window.__browserwingIframeRecorder__) {
		return;
	}
	window.__browserwingIframeRecorder__ = true;
	
	console.log('[BrowserWing] iframe Recorder initialized');
	
	// 向父窗口发送录制的操作
	var sendToParent = function(action) {
		try {
			// 标记为来自 iframe
			action.fromIframe = true;
			window.parent.postMessage({
				type: '__browserwing_iframe_action__',
				action: action
			}, '*');
		} catch (e) {
			console.error('[BrowserWing] iframe postMessage error:', e);
		}
	};
	
	// 获取选择器（简化版）
	var getSelector = function(element) {
		if (!element || !element.tagName) {
			return { css: 'unknown', xpath: '//*' };
		}
		
		var css = element.tagName.toLowerCase();
		var xpath = '//' + css;
		
		// ID
		if (element.id) {
			css = '#' + element.id;
			xpath = '//*[@id="' + element.id + '"]';
			return { css: css, xpath: xpath };
		}
		
		// name
		if (element.name) {
			css += '[name="' + element.name + '"]';
			xpath += '[@name="' + element.name + '"]';
			return { css: css, xpath: xpath };
		}
		
		// class
		if (element.className && typeof element.className === 'string') {
			var classes = element.className.trim().split(/\s+/);
			if (classes.length > 0 && classes[0]) {
				css += '.' + classes[0];
			}
		}
		
		return { css: css, xpath: xpath };
	};
	
	// 监听点击
	document.addEventListener('click', function(e) {
		try {
			var target = e.target || e.srcElement;
			if (!target || !target.tagName) return;
			
			var selectors = getSelector(target);
			sendToParent({
				type: 'click',
				timestamp: Date.now(),
				selector: 'iframe ' + selectors.css,
				xpath: '//iframe' + selectors.xpath,
				text: (target.innerText || target.textContent || '').substring(0, 50),
				tagName: target.tagName.toLowerCase()
			});
		} catch (err) {
			console.error('[BrowserWing] iframe click error:', err);
		}
	}, true);
	
	// 监听输入（防抖）
	var inputTimers = {};
	
	// 监听 input 事件（标准输入）
	document.addEventListener('input', function(e) {
		try {
			var target = e.target || e.srcElement;
			if (!target) return;
			
			var tagName = target.tagName ? target.tagName.toUpperCase() : '';
			var isContentEditable = target.contentEditable === 'true' || target.isContentEditable;
			
			if (tagName !== 'INPUT' && tagName !== 'TEXTAREA' && !isContentEditable) return;
			
			var selectors = getSelector(target);
			var selectorKey = selectors.css;
			
			if (inputTimers[selectorKey]) {
				clearTimeout(inputTimers[selectorKey]);
			}
			
			var content = isContentEditable ? (target.textContent || target.innerText || '') : (target.value || '');
			
			inputTimers[selectorKey] = setTimeout(function() {
				sendToParent({
					type: 'input',
					timestamp: Date.now(),
					selector: 'iframe ' + selectors.css,
					xpath: '//iframe' + selectors.xpath,
					value: content,
					tagName: isContentEditable ? 'contenteditable' : tagName.toLowerCase()
				});
			}, 500);
		} catch (err) {
			console.error('[BrowserWing] iframe input error:', err);
		}
	}, true);
	
	// 监听 blur 事件（失去焦点时立即记录最终值）
	document.addEventListener('blur', function(e) {
		try {
			var target = e.target || e.srcElement;
			if (!target) return;
			
			var tagName = target.tagName ? target.tagName.toUpperCase() : '';
			var isContentEditable = target.contentEditable === 'true' || target.isContentEditable;
			
			if (tagName !== 'INPUT' && tagName !== 'TEXTAREA' && !isContentEditable) return;
			
			var selectors = getSelector(target);
			var selectorKey = selectors.css;
			
			// 清除防抖定时器
			if (inputTimers[selectorKey]) {
				clearTimeout(inputTimers[selectorKey]);
				delete inputTimers[selectorKey];
			}
			
			// 立即记录最终值
			var content = isContentEditable ? (target.textContent || target.innerText || '') : (target.value || '');
			
			// 只在有内容时才记录
			if (content && content.trim().length > 0) {
				sendToParent({
					type: 'input',
					timestamp: Date.now(),
					selector: 'iframe ' + selectors.css,
					xpath: '//iframe' + selectors.xpath,
					value: content,
					tagName: isContentEditable ? 'contenteditable' : tagName.toLowerCase()
				});
			}
		} catch (err) {
			console.error('[BrowserWing] iframe blur error:', err);
		}
	}, true);
	
	// 监听 DOMCharacterDataModified 事件（某些富文本编辑器使用）
	document.addEventListener('DOMCharacterDataModified', function(e) {
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
			var selectorKey = selectors.css;
			
			if (inputTimers[selectorKey]) {
				clearTimeout(inputTimers[selectorKey]);
			}
			
			var content = editableParent.textContent || editableParent.innerText || '';
			
			inputTimers[selectorKey] = setTimeout(function() {
				sendToParent({
					type: 'input',
					timestamp: Date.now(),
					selector: 'iframe ' + selectors.css,
					xpath: '//iframe' + selectors.xpath,
					value: content,
					tagName: 'contenteditable'
				});
			}, 500);
		} catch (err) {
			console.error('[BrowserWing] iframe DOMCharacterDataModified error:', err);
		}
	}, true);
	
	// 监听选择
	document.addEventListener('change', function(e) {
		try {
			var target = e.target || e.srcElement;
			if (!target) return;
			
			var tagName = target.tagName ? target.tagName.toUpperCase() : '';
			if (tagName === 'SELECT') {
				var selectedText = target.options && target.selectedIndex >= 0 ? 
					(target.options[target.selectedIndex].text || '') : '';
				
				var selectors = getSelector(target);
				sendToParent({
					type: 'select',
					timestamp: Date.now(),
					selector: 'iframe ' + selectors.css,
					xpath: '//iframe' + selectors.xpath,
					value: target.value || '',
					text: selectedText,
					tagName: 'select'
				});
			}
		} catch (err) {
			console.error('[BrowserWing] iframe change error:', err);
		}
	}, true);
})();
