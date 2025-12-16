# AI 填充表单功能

## 功能概述

AI 填充表单功能允许用户在录制脚本时，选择表单区域后通过 AI 自动生成填充该表单的 JavaScript 代码。

## 使用方法

1. **开启录制器**：在浏览器中打开录制器面板

2. **点击 "AI填充" 按钮**：进入 AI 填充表单模式
   - 按钮位于录制器面板顶部，绿色背景
   - 点击后按钮文字变为 "退出AI填充"

3. **选择表单区域**：
   - 鼠标悬停在表单元素上会看到高亮效果
   - 点击表单容器或 `<form>` 元素
   - 也可以点击表单内的任意元素，系统会自动查找最近的 `<form>` 标签

4. **AI 生成代码**：
   - 系统显示 "AI 正在分析表单结构..." 的加载提示
   - LLM 分析表单 HTML 结构
   - 生成填充表单的 JavaScript 代码
   - 自动添加到脚本的操作列表中

5. **查看生成的代码**：
   - 生成的代码会作为 `execute_js` 类型的操作添加到脚本中
   - 可以在操作列表中查看和编辑生成的代码

## 实现原理

### 前端部分 (recorder.js)

1. **UI 按钮**：
   - 添加了 "AI填充" 按钮（绿色主题）
   - 与 "AI提取" 按钮并列显示

2. **模式切换**：
   - `toggleAIFormFillMode()` 函数控制模式的开启/关闭
   - 开启时其他模式（普通抓取、AI提取）会自动关闭
   - 光标变为十字形状，提示用户选择元素

3. **表单选择与处理**：
   - `handleAIFormFillClick(element)` 处理表单点击
   - 查找最近的 `<form>` 元素或使用点击的容器元素
   - 使用 `cleanAndSampleHTML()` 清理和优化 HTML
   - 通过 `window.__aiExtractionRequest__` 设置请求（type: 'formfill'）

4. **轮询响应**：
   - 前端通过轮询检查 `window.__aiFormFillResponse__`
   - 超时时间 60 秒，轮询间隔 200ms
   - 收到响应后创建 `execute_js` 操作记录

### 后端部分

#### 1. extractor.go

新增结构体：
```go
type FormFillRequest struct {
    HTML        string `json:"html"`
    Description string `json:"description"`
}

type FormFillResult struct {
    JavaScript string `json:"javascript"`
    UsedModel  string `json:"used_model"`
}
```

核心方法：
```go
func (e *Extractor) GenerateFormFillingJS(ctx context.Context, req FormFillRequest) (*FormFillResult, error)
```

- 调用 LLM 生成表单填充代码
- 使用温度 0.3 确保代码生成稳定
- 从 markdown 代码块中提取纯 JavaScript

#### 2. recorder.go

新增方法：
```go
func (r *Recorder) handleFormFillRequest(ctx context.Context, html, description string)
```

- 检测请求类型（通过 `type` 字段区分 "extract" 和 "formfill"）
- 调用 `extractor.GenerateFormFillJS()` 生成代码
- 将结果设置到 `window.__aiFormFillResponse__`

#### 3. models/prompt.go

新增常量：
```go
const SystemPromptFormFillerID = "system-formfiller"
```

用于从数据库获取表单填充的系统提示词。

## AI Prompt 设计

默认的表单填充提示词要求 LLM：

1. **识别表单字段**：分析 input、textarea、select 等元素
2. **推断字段用途**：根据 name、id、placeholder 等属性理解字段含义
3. **生成测试数据**：为每个字段生成合理的测试数据
4. **返回纯代码**：只返回可执行的 JavaScript，不包含解释
5. **使用标准选择器**：使用 `document.querySelector` 定位元素
6. **处理特殊控件**：
   - select 元素选择合适选项
   - checkbox/radio 合理勾选
7. **触发事件**：触发 input、change 事件确保表单验证正常工作

## 生成代码示例

对于典型的登录表单：

```html
<form id="login-form">
  <input type="text" name="username" placeholder="用户名">
  <input type="password" name="password" placeholder="密码">
  <button type="submit">登录</button>
</form>
```

AI 可能生成：

```javascript
// 填充用户名
const usernameInput = document.querySelector('input[name="username"]');
if (usernameInput) {
  usernameInput.value = 'testuser@example.com';
  usernameInput.dispatchEvent(new Event('input', { bubbles: true }));
  usernameInput.dispatchEvent(new Event('change', { bubbles: true }));
}

// 填充密码
const passwordInput = document.querySelector('input[name="password"]');
if (passwordInput) {
  passwordInput.value = 'Test@123456';
  passwordInput.dispatchEvent(new Event('input', { bubbles: true }));
  passwordInput.dispatchEvent(new Event('change', { bubbles: true }));
}
```

## 注意事项

1. **数据隐私**：
   - 生成的是测试数据，不应用于真实账号
   - 建议在测试环境使用

2. **表单复杂度**：
   - 简单表单效果最好
   - 复杂的动态表单可能需要手动调整生成的代码

3. **验证规则**：
   - AI 会尝试生成符合常见验证规则的数据
   - 特殊验证规则可能需要在描述中说明

4. **生成代码审查**：
   - 建议查看生成的代码确保符合预期
   - 可以在脚本编辑器中修改生成的代码

## 与 AI 提取功能的对比

| 特性 | AI 提取 | AI 填充 |
|------|---------|---------|
| 按钮颜色 | 深灰色 | 绿色 |
| 主要用途 | 从页面提取数据 | 填充表单 |
| 生成代码类型 | 数据提取代码 | 表单填充代码 |
| 典型应用场景 | 抓取列表、表格数据 | 自动化表单测试 |
| 系统提示词 ID | system-extractor | system-formfiller |

## 后续优化建议

1. **支持用户自定义填充数据**：
   - 允许用户在点击表单后输入期望的填充内容
   - 添加一个输入框让用户描述需要填充的数据

2. **表单字段映射**：
   - 提供界面让用户预览将要填充的数据
   - 允许用户调整每个字段的填充值

3. **数据集成**：
   - 支持从变量或配置文件读取填充数据
   - 与脚本的输入参数集成

4. **智能识别**：
   - 识别常见表单类型（登录、注册、搜索等）
   - 根据表单类型生成更合理的数据

5. **验证码处理**：
   - 识别表单中的验证码字段
   - 提示用户手动处理或集成验证码服务
