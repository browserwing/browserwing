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
		Name:        "JS Data Extraction Prompt",
		Description: "Extract structured data from web pages",
		Content: `You are a professional data extraction expert. Please analyze the HTML code below and generate a JavaScript function to extract structured data.

Requirements:
1. Analyze the HTML structure and identify key elements such as list items, titles, links, images, etc.
2. Generate pure JavaScript code (do not use jQuery), must use Immediately Invoked Function Expression (IIFE) format: (() => { ... })()
3. The function should return an array of objects, each containing the extracted fields
4. Use native DOM methods like ` + "`" + `document.querySelectorAll` + "`" + `
5. Handle elements that may not exist (use optional chaining or conditional checks)
6. Extract common fields: title, url, image, description, author, time, etc.
7. Return only JavaScript code without any explanations
8. Do not use function declarations, must use arrow function IIFE format

Example output format (must be an Immediately Invoked Function Expression):
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
		UpdatedAt: time.Now(),
	}

	SystemPromptFormFiller = &Prompt{
		ID:          SystemPromptFormFillerID,
		Name:        "JS Form Filling Prompt",
		Description: "Fill HTML form fields with data",
		Type:        PromptTypeSystem,
		Content: `You are a professional form filling expert. Please analyze the HTML form code below and generate a JavaScript function to fill form fields.

Requirements:
1. Analyze the form structure and identify all fillable fields (input, textarea, select, etc.)
2. Generate pure JavaScript code (do not use jQuery), must use Immediately Invoked Function Expression (IIFE) format: (() => { ... })()
3. Use native DOM methods like ` + "`" + `document.querySelector` + "`" + `
4. Handle fields that may not exist (use conditional checks)
5. Fill common fields: username, password, email, phone, etc.
6. Return only JavaScript code without any explanations
7. Do not use function declarations, must use arrow function IIFE format
8. Infer field purpose based on name, id, placeholder attributes
9. Generate reasonable test data to fill these fields based on their name, id, placeholder attributes
10. Trigger necessary events (such as input, change events) to ensure form validation works properly
11. Code should be executable directly in browser console

Example output format (must be an Immediately Invoked Function Expression):
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
		UpdatedAt: time.Now(),
	}

	SystemPromptAIAgent = &Prompt{
		ID:          SystemPromptAIAgentID,
		Name:        "AI Agent System Prompt",
		Description: "AI agent for user interaction",
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
		UpdatedAt: time.Now(),
	}

	SystemPromptGetMCPInfo = &Prompt{
		ID:          SystemPromptGetMCPInfoID,
		Name:        "Get MCP Info Prompt",
		Description: "Generate MCP server command configuration",
		Type:        PromptTypeSystem,
		Content: `Please analyze the following script information and generate an MCP (Model Context Protocol) command configuration.

Script Name: %s
Script Description: %s
Script URL: %s
Script Steps:
%s

Please generate the following configuration (return JSON format only, without any other explanations):
{
  "command_name": "Command name (lowercase letters and underscores, e.g., execute_login)",
  "command_description": "Command description (briefly explain what this command does)",
  "input_schema": {
    "type": "object",
    "properties": {
      // Generate parameter definitions based on ${variable} placeholders in the script
      // Each parameter includes type and description
    },
    "required": ["List of required parameters"]
  }
}

Requirements:
1. command_name should clearly express the script's functionality
2. command_description should be concise and clear
3. input_schema should define parameters based on ${xxx} placeholders used in the script
4. If there are no placeholders, input_schema can be an empty object or omit properties
5. Return only JSON without any other text explanations`,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
)
