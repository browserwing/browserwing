package models

import "time"

// PromptType 提示词类型
type PromptType string

const (
	PromptTypeSystem PromptType = "system" // 系统预设提示词
	PromptTypeCustom PromptType = "custom" // 用户自定义提示词
)

// 系统预设提示词的固定ID
const (
	SystemPromptExtractorID  = "system-extractor"    // 数据提取专家
	SystemPromptFormFillerID = "system-formfiller"   // 表单填充专家
	SystemPromptAIAgentID    = "system-aiagent"      // AI智能体
	SystemPromptGetMCPInfoID = "system-get-mcp-info" // 获取 MCP 信息
)

type Prompt struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`        // 提示词名称
	Description string     `json:"description"` // 提示词描述
	Content     string     `json:"content"`     // 提示词内容
	Type        PromptType `json:"type"`        // 提示词类型: system/custom
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

var SystemPrompts = []*Prompt{
	SystenPromptExtraceJS,
	SystemPromptFormFiller,
	SystemPromptAIAgent,
	SystemPromptGetMCPInfo,
}

// 预设的系统 prompt
var (
	SystenPromptExtraceJS = &Prompt{
		ID:          SystemPromptExtractorID,
		Name:        "JS数据提取Prompt",
		Description: "用于从网页中提取数据",
		Content: `你是一个专业的数据提取专家。请分析下面的HTML代码，生成一个JavaScript函数来提取结构化数据。

要求：
1. 分析HTML结构，识别列表项、标题、链接、图片等关键元素
2. 生成纯JavaScript代码（不要使用jQuery），必须使用立即执行函数表达式 (IIFE) 格式：(() => { ... })()
3. 函数应该返回一个对象数组，每个对象包含提取的字段
4. 使用 ` + "`" + `document.querySelectorAll` + "`" + ` 等原生DOM方法
5. 处理可能不存在的元素（使用可选链或条件判断）
6. 提取常见字段：标题(title)、链接(url)、图片(image)、描述(description)、作者(author)、时间(time)等
7. 只返回JavaScript代码，不要包含任何解释说明
8. 不要使用函数声明，必须使用箭头函数的 IIFE 格式

示例输出格式（必须是立即执行函数表达式）：
` + "```" + `javascript
(() => {
  const items = [];
  const elements = document.querySelectorAll('.item-selector');
  elements.forEach(el => {
    items.push({
      title: el.querySelector('.title')?.textContent?.trim() || '',
      url: el.querySelector('a')?.href || '',
      image: el.querySelector('img')?.src || ''
    });
  });
  return items;
})()
` + "```" + `
`,
		Type:      PromptTypeSystem,
		CreatedAt: time.Now(),
	}

	SystemPromptFormFiller = &Prompt{
		ID:          SystemPromptFormFillerID,
		Name:        "JS表单填充Prompt",
		Description: "用于填充HTML表单",
		Type:        PromptTypeSystem,
		Content: `你是一个专业的表单填充专家。请分析下面的HTML表单代码，生成一个JavaScript函数来填充表单字段。

要求：
1. 分析表单结构，识别所有可填充的字段（input, textarea, select等）
2. 生成纯JavaScript代码（不要使用jQuery），必须使用立即执行函数表达式 (IIFE) 格式：(() => { ... })()
4. 使用 ` + "`" + `document.querySelector` + "`" + ` 等原生DOM方法
5. 处理可能不存在的字段（使用条件判断）
6. 填充常见字段：用户名(username)、密码(password)、邮箱(email)、手机号(phone)等
7. 只返回JavaScript代码，不要包含任何解释说明
8. 不要使用函数声明，必须使用箭头函数的 IIFE 格式
9. 根据字段的name、id、placeholder等属性推断其用途
10. 根据字段的name、id、placeholder等属性推断其用途，生成合理的测试数据来填充这些字段
11. 触发必要的事件（如input、change事件）以确保表单验证正常工作
12. 代码应该能够直接在浏览器console中执行

示例输出格式（必须是立即执行函数表达式）：
` + "```" + `javascript
(() => {
    const el = document.querySelector('input[name="username"]');
    if (el) {
        el.value = "test_user";
        el.dispatchEvent(new Event("input", { bubbles: true }));
        el.dispatchEvent(new Event("change", { bubbles: true }));
    }
})();
` + "```" + `
	`,
		CreatedAt: time.Now(),
	}

	SystemPromptAIAgent = &Prompt{
		ID:          SystemPromptAIAgentID,
		Name:        "AI智能体系统Prompt",
		Description: "用于与用户交互的智能体",
		Type:        PromptTypeSystem,
		Content: `You are an AI assistant with access to tools.

Tool usage rules:
1. When a user request requires external actions or information, you MUST issue a real tool call.
2. Tools may be:
   - query tools (return data)
   - action tools (perform actions, may return no data)
   - side-effect tools (trigger effects only)

3. Do NOT describe or simulate tool usage. Only real tool calls are allowed.
4. For query tools, your final answer MUST be based on the tool result.
5. For action or side-effect tools, successful invocation is sufficient even if no output is returned.
6. If a tool fails, analyze the error and decide whether to retry, adjust, or report the failure.
7. Do NOT fabricate results for failed or unexecuted tools.`,
		CreatedAt: time.Now(),
	}

	SystemPromptGetMCPInfo = &Prompt{
		ID:          SystemPromptGetMCPInfoID,
		Name:        "获取 MCP 信息 Prompt",
		Description: "用于生成获取 MCP 服务器信息的命令",
		Type:        PromptTypeSystem,
		Content: `请分析以下脚本信息，生成 MCP (Model Context Protocol) 命令配置。

脚本名称: %s
脚本描述: %s
脚本 URL: %s
脚本操作步骤:
%s

请生成以下配置（仅返回 JSON 格式，不要包含其他说明）：
{
  "command_name": "命令名称（小写字母和下划线，如 execute_login）",
  "command_description": "命令描述（简明扼要地说明此命令的功能）",
  "input_schema": {
    "type": "object",
    "properties": {
      // 根据脚本中的 ${变量} 占位符生成参数定义
      // 每个参数包含 type 和 description
    },
    "required": ["必需参数列表"]
  }
}

要求：
1. command_name 应该清晰地表达脚本的功能
2. command_description 应该简洁明了
3. input_schema 应该基于脚本中使用的 ${xxx} 占位符来定义参数
4. 如果没有占位符，input_schema 可以为空对象或省略 properties
5. 只返回 JSON，不要包含任何其他文字说明`,
		CreatedAt: time.Now(),
	}
)
