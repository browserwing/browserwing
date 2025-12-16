// 在浏览器 Console 中使用此脚本来测试选择器

// 测试 XPath 选择器
function testXPath(xpath) {
    console.log('测试 XPath:', xpath);
    
    const result = document.evaluate(
        xpath,
        document,
        null,
        XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
        null
    );
    
    console.log('找到 ' + result.snapshotLength + ' 个元素');
    
    const elements = [];
    for (let i = 0; i < result.snapshotLength; i++) {
        const el = result.snapshotItem(i);
        elements.push(el);
        
        // 高亮显示
        el.style.outline = '3px solid red';
        el.style.backgroundColor = 'rgba(255, 0, 0, 0.1)';
        
        console.log(`[${i}]`, {
            element: el,
            tagName: el.tagName,
            text: el.textContent?.substring(0, 50),
            innerText: el.innerText?.substring(0, 50),
            visible: el.offsetParent !== null
        });
    }
    
    return elements;
}

// 测试 CSS 选择器
function testCSS(selector) {
    console.log('测试 CSS:', selector);
    
    try {
        const elements = document.querySelectorAll(selector);
        console.log('找到 ' + elements.length + ' 个元素');
        
        elements.forEach((el, i) => {
            // 高亮显示
            el.style.outline = '3px solid blue';
            el.style.backgroundColor = 'rgba(0, 0, 255, 0.1)';
            
            console.log(`[${i}]`, {
                element: el,
                tagName: el.tagName,
                text: el.textContent?.substring(0, 50),
                innerText: el.innerText?.substring(0, 50),
                visible: el.offsetParent !== null
            });
        });
        
        return Array.from(elements);
    } catch (e) {
        console.error('CSS 选择器无效:', e.message);
        return [];
    }
}

// 清除所有高亮
function clearHighlight() {
    document.querySelectorAll('*').forEach(el => {
        el.style.outline = '';
        el.style.backgroundColor = '';
    });
    console.log('已清除所有高亮');
}

// 分析元素的文本内容（查找隐藏字符）
function analyzeText(element) {
    const text = element.textContent || '';
    const innerText = element.innerText || '';
    
    console.log('=== 文本分析 ===');
    console.log('textContent:', text);
    console.log('innerText:', innerText);
    console.log('textContent 长度:', text.length);
    console.log('innerText 长度:', innerText.length);
    console.log('trim 后:', text.trim());
    console.log('trim 后长度:', text.trim().length);
    
    console.log('\n字符编码:');
    [...text].forEach((char, i) => {
        const code = char.charCodeAt(0);
        console.log(`[${i}] '${char}' (U+${code.toString(16).toUpperCase().padStart(4, '0')}) - ${code}`);
    });
    
    // 检测特殊字符
    const specialChars = {
        '\u200b': '零宽空格 (ZWSP)',
        '\u200c': '零宽非连接符 (ZWNJ)',
        '\u200d': '零宽连接符 (ZWJ)',
        '\ufeff': '零宽无断空格 (BOM)',
        '\u00a0': '不间断空格 (NBSP)',
        '\u2003': '全角空格 (EM SPACE)',
    };
    
    console.log('\n特殊字符检测:');
    Object.entries(specialChars).forEach(([char, name]) => {
        if (text.includes(char)) {
            console.warn(`⚠️ 发现 ${name}`);
        }
    });
}

// 生成更可靠的选择器
function generateSelector(element) {
    if (!element) {
        console.error('元素不存在');
        return;
    }
    
    console.log('=== 生成选择器 ===');
    
    // ID
    if (element.id) {
        console.log('CSS (ID):', `#${element.id}`);
        console.log('XPath (ID):', `//*[@id="${element.id}"]`);
    }
    
    // Name
    if (element.name) {
        console.log('CSS (Name):', `${element.tagName.toLowerCase()}[name="${element.name}"]`);
        console.log('XPath (Name):', `//${element.tagName.toLowerCase()}[@name="${element.name}"]`);
    }
    
    // data-testid
    const testId = element.getAttribute('data-testid');
    if (testId) {
        console.log('CSS (data-testid):', `[data-testid="${testId}"]`);
        console.log('XPath (data-testid):', `//*[@data-testid="${testId}"]`);
    }
    
    // aria-label
    const ariaLabel = element.getAttribute('aria-label');
    if (ariaLabel) {
        console.log('CSS (aria-label):', `[aria-label="${ariaLabel}"]`);
        console.log('XPath (aria-label):', `//*[@aria-label="${ariaLabel}"]`);
    }
    
    // 文本内容（处理特殊字符）
    const text = (element.textContent || '').trim();
    if (text && text.length < 30) {
        const cleanText = text.replace(/[\u200b-\u200d\ufeff]/g, ''); // 移除零宽字符
        console.log('XPath (文本-原始):', `//${element.tagName.toLowerCase()}[contains(text(), "${text}")]`);
        console.log('XPath (文本-清理):', `//${element.tagName.toLowerCase()}[contains(text(), "${cleanText}")]`);
        console.log('XPath (文本-精确):', `//${element.tagName.toLowerCase()}[text()="${text}"]`);
        console.log('XPath (normalize):', `//${element.tagName.toLowerCase()}[contains(normalize-space(text()), "${cleanText}")]`);
    }
    
    // 完整 XPath
    function getFullXPath(el) {
        if (el.id) return `//*[@id="${el.id}"]`;
        
        let path = '';
        for (; el && el.nodeType === 1; el = el.parentNode) {
            let index = 0;
            for (let sibling = el.previousSibling; sibling; sibling = sibling.previousSibling) {
                if (sibling.nodeType === 1 && sibling.tagName === el.tagName) index++;
            }
            const tagName = el.tagName.toLowerCase();
            const pathIndex = index > 0 ? `[${index + 1}]` : '';
            path = `/${tagName}${pathIndex}${path}`;
        }
        return path;
    }
    
    console.log('XPath (完整路径):', getFullXPath(element));
}

// 使用示例
console.log('=== 选择器测试工具已加载 ===');
console.log('使用方法:');
console.log('1. testXPath("//a[contains(text(), \\"热榜\\")]") - 测试 XPath');
console.log('2. testCSS("a.link") - 测试 CSS');
console.log('3. clearHighlight() - 清除高亮');
console.log('4. analyzeText(element) - 分析元素文本');
console.log('5. generateSelector(element) - 生成选择器');
console.log('');
console.log('快速测试（从日志中的 XPath）:');
console.log('testXPath("//a[contains(text(), \\"热榜\\")]")');
console.log('testXPath("//a[contains(text(), \\"进入创作中心\\")]")');
