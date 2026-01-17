# Executor API Claude Skills 导出功能

## 概述

现在你可以一键导出 Executor API 的完整 Claude Skills 文档（SKILL.md 格式），让 Claude 可以直接使用 BrowserPilot 进行浏览器自动化。

## 导出接口

### 端点
```
GET /api/v1/executor/export/skill
```

### 请求
```bash
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL.md
```

### 响应
- **Content-Type**: `text/markdown; charset=utf-8`
- **Content-Disposition**: `attachment; filename=EXECUTOR_SKILL_<timestamp>.md`
- **内容**: 完整的 SKILL.md 文件

---

## 生成的 SKILL.md 包含什么？

### 1. YAML Frontmatter
```yaml
---
name: browserpilot-executor
description: Control browser automation through HTTP API. Supports page navigation, element interaction (click, type, select), data extraction, semantic tree analysis, screenshot, JavaScript execution, and batch operations.
---
```

### 2. 概述信息
- BrowserPilot Executor 的功能介绍
- API 基础 URL
- 认证方式说明
- 核心能力列表

### 3. API 端点列表

#### 发现命令
- `GET /help` - 查看所有可用命令
- `GET /help?command=<name>` - 查看特定命令详情

#### 页面语义树
- `GET /semantic-tree` - 获取页面结构（⭐ 关键接口）

#### 常用操作示例
- Navigate（导航）
- Click（点击）
- Type（输入）
- Extract（提取数据）
- Wait（等待元素）
- Batch（批量操作）

### 4. 元素定位方式

详细说明了 5 种元素定位方法：
- **语义树索引**（推荐）: `[1]`, `Input Element [1]`
- **CSS 选择器**: `#id`, `.class`
- **文本内容**: `Login`, `Sign Up`
- **XPath**: `//button[@id='login']`
- **ARIA Label**: 自动搜索

### 5. 使用说明（Instructions）

6 个步骤的标准工作流：
1. 发现命令（如果不确定）
2. 导航到页面
3. 分析页面结构（semantic-tree）
4. 与元素交互
5. 提取数据
6. 呈现结果

### 6. 完整示例

**登录示例**：
- 7 个详细步骤
- 包含完整的 API 调用
- 展示如何使用语义树索引

**批量操作示例**：
- 表单填写场景
- 一次请求完成多个操作
- 展示 stop_on_error 用法

### 7. 关键命令速查表

按功能分类的命令列表：
- Navigation（4个命令）
- Element Interaction（6个命令）
- Data Extraction（6个命令）
- Page Analysis（3个命令）
- Advanced（5个命令）

### 8. 最佳实践

7 条实用建议：
- 如何发现命令
- 为什么要先调用 semantic-tree
- 为什么推荐语义树索引
- 如何处理动态内容
- 何时使用批量操作
- 如何处理错误
- 如何验证结果

### 9. 常见场景

4 种典型使用场景的详细步骤：
- 表单填写
- 数据抓取
- 搜索操作
- 登录自动化

### 10. 故障排除

针对常见问题的解决方案：
- 元素未找到
- 超时错误
- 提取返回空

### 11. 快速参考

常用命令的快速查询格式。

### 12. 响应格式说明

标准响应和错误响应的结构。

---

## 使用流程

### 步骤 1: 导出 SKILL.md

```bash
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL.md
```

这会下载一个名为 `EXECUTOR_SKILL_<timestamp>.md` 的文件。

### 步骤 2: 在 Claude 中加载

1. 打开 Claude（Desktop 或 Web）
2. 进入 Skills 设置
3. 点击 "Add Skill"
4. 选择刚才下载的 `EXECUTOR_SKILL.md` 文件
5. 启用这个 Skill

### 步骤 3: 开始使用

现在你可以直接让 Claude 控制浏览器了！

**示例对话**:
```
用户: 帮我在淘宝搜索 "MacBook Pro" 并提取前 10 个商品的标题和价格

Claude:
好的，让我来帮你完成这个任务。

1. 首先导航到淘宝...
   [调用 POST /navigate]

2. 获取页面结构...
   [调用 GET /semantic-tree]
   发现：Input Element [1] 是搜索框

3. 在搜索框输入 "MacBook Pro"...
   [调用 POST /type]

4. 按 Enter 提交搜索...
   [调用 POST /press-key]

5. 等待搜索结果加载...
   [调用 POST /wait]

6. 提取商品数据...
   [调用 POST /extract]

找到 10 个商品：
1. MacBook Pro 14英寸 M3 - ¥14999
2. MacBook Pro 16英寸 M3 Max - ¥24999
...
```

---

## 与脚本导出的区别

### Script Export（`POST /api/v1/scripts/export/skill`）
- 导出**录制的脚本**
- 预设的自动化流程
- 固定的参数和步骤
- 适合重复执行的任务

### Executor Export（`GET /api/v1/executor/export/skill`）
- 导出**通用的浏览器操作 API**
- 灵活的操作能力
- Claude 可以自由组合
- 适合动态和复杂的任务

**推荐**: 两个 Skills 都加载到 Claude 中！
- 使用脚本 Skill 执行常见的固定任务
- 使用 Executor Skill 处理灵活的自定义任务

---

## 导出的 SKILL.md 特点

### ✅ 自包含
- 包含所有需要的信息
- 不依赖外部文档
- 可以独立使用

### ✅ 自动发现
- 引导 Claude 先调用 `/help` 接口
- 动态学习最新的命令
- 不需要手动更新 Skill

### ✅ 详细的指导
- 清晰的步骤说明
- 丰富的使用示例
- 最佳实践建议
- 故障排除指南

### ✅ Claude 友好
- 结构化的说明
- 清晰的工作流
- 具体的示例代码
- 重点内容突出（⭐ 标记）

---

## 高级用法

### 1. 自定义 API Host

导出的 SKILL.md 会自动使用当前的 `c.Request.Host`。如果你想使用不同的地址：

```bash
# 方式 1: 通过反向代理
# 设置 X-Forwarded-Host header

# 方式 2: 手动编辑导出的文件
# 将所有 localhost:8080 替换为你的实际域名
```

### 2. 多环境支持

为不同环境导出不同的 SKILL.md：

```bash
# 开发环境
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -o EXECUTOR_SKILL_DEV.md

# 生产环境
curl -X GET 'https://prod.example.com/api/v1/executor/export/skill' \
  -o EXECUTOR_SKILL_PROD.md
```

### 3. 定期更新

当 API 更新后，重新导出 SKILL.md：

```bash
# 下载最新版本
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL_LATEST.md

# 在 Claude 中替换旧的 Skill
```

---

## 完整的 Claude Skills 设置

推荐同时加载两个 Skills：

### 1. Scripts Skill
```bash
curl -X POST 'http://localhost:8080/api/v1/scripts/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -H 'Content-Type: application/json' \
  -d '{"script_ids": []}' \
  -o SCRIPTS_SKILL.md
```

**用途**: 执行预先录制的脚本
- 搜索固定网站
- 定期数据采集
- 标准化的操作流程

### 2. Executor Skill
```bash
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL.md
```

**用途**: 灵活的浏览器控制
- 动态网站交互
- 自定义数据提取
- 复杂的自动化流程

### Claude 的选择逻辑

Claude 会根据任务选择合适的 Skill：

**使用 Scripts Skill**:
```
用户: "帮我在小红书搜索 MCP"
Claude: 发现有预设的"小红书搜索"脚本，直接调用
```

**使用 Executor Skill**:
```
用户: "帮我在这个陌生网站上找到联系表单并填写"
Claude: 这是动态任务，使用 Executor API 灵活操作
```

---

## SKILL.md 内容预览

导出的文件看起来像这样：

```markdown
---
name: browserpilot-executor
description: Control browser automation through HTTP API...
---

# BrowserPilot Executor API

## Overview

BrowserPilot Executor provides comprehensive browser automation...

**API Base URL:** `http://localhost:8080/api/v1/executor`

**Authentication:** Use `X-BrowserWing-Key: <api-key>` header...

## Core Capabilities

- **Page Navigation:** Navigate to URLs, go back/forward, reload
- **Element Interaction:** Click, type, select, hover on page elements
- **Data Extraction:** Extract text, attributes, values from elements
...

## API Endpoints

### 1. Discover Available Commands

**IMPORTANT:** Always call this endpoint first to see all available commands...

### 2. Get Semantic Tree

**CRITICAL:** Always call this after navigation to understand page structure...

### 3. Common Operations

#### Navigate to URL
```bash
curl -X POST 'http://localhost:8080/api/v1/executor/navigate' \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com"}'
```

...

## Complete Example

**Scenario:** User wants to login to a website

**Step 1:** Navigate to login page
**Step 2:** Get page structure
**Step 3:** Enter username
...
```

---

## 测试

### 1. 导出文件

```bash
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL.md
```

### 2. 检查文件

```bash
# 查看文件大小
ls -lh EXECUTOR_SKILL.md

# 查看前几行
head -20 EXECUTOR_SKILL.md

# 查看文件内容
cat EXECUTOR_SKILL.md
```

### 3. 验证格式

确保文件包含：
- ✅ YAML frontmatter（`---` 开头和结尾）
- ✅ 主标题和概述
- ✅ API 端点列表
- ✅ 完整示例
- ✅ 使用说明

---

## 在 Claude 中使用

### 步骤 1: 加载 Skill

1. 打开 Claude Desktop 或 Claude.ai
2. 进入 Settings → Skills
3. 点击 "Add Skill" 或 "Import Skill"
4. 选择导出的 `EXECUTOR_SKILL.md` 文件
5. 启用这个 Skill

### 步骤 2: 验证加载

```
用户: 你能帮我控制浏览器吗？

Claude: 
是的！我可以通过 BrowserPilot Executor API 控制浏览器。

我有以下能力：
- 页面导航和浏览器控制
- 点击、输入、选择等元素交互
- 数据提取和页面分析
- 截图和 JavaScript 执行
- 批量操作

我会首先调用 /semantic-tree 来了解页面结构，然后使用元素索引（如 [1], [2]）
来精确定位元素。

你想让我做什么？
```

### 步骤 3: 测试功能

```
用户: 帮我打开 example.com 并告诉我页面上有什么可点击的元素

Claude:
好的，让我来帮你...

1. 导航到 example.com
   [POST /navigate {"url": "https://example.com"}]
   ✅ 成功

2. 获取页面结构
   [GET /semantic-tree]
   ✅ 成功

页面上有以下可点击元素：
1. [1] Login - 登录按钮
2. [2] Sign Up - 注册链接
3. [3] Learn More - 了解更多按钮
4. [4] Contact Us - 联系我们链接

你想点击哪个？
```

---

## 与内置 Help API 的配合

导出的 SKILL.md 会引导 Claude 使用内置的 Help API：

### 在 SKILL.md 中的说明

```markdown
## Instructions

**Step-by-step workflow:**

1. **Discover commands:** Call `GET /help` to see all available operations 
   and their parameters (do this first if unsure).
   
2. **Navigate:** Use `POST /navigate` to open the target webpage.

3. **Analyze page:** Call `GET /semantic-tree` to understand page structure 
   and get element indices.
...
```

### Claude 的实际行为

```
用户: 我想提取数据，但不知道怎么用 extract 命令

Claude: 
让我查询一下 extract 命令的详细用法...

[调用 GET /help?command=extract]

extract 命令用于从页面元素提取数据，支持以下参数：

必需参数：
- selector (string): CSS 选择器

可选参数：
- fields (array): 要提取的字段，如 ["text", "href", "src"]
- multiple (boolean): 是否提取多个元素，默认 false
- type (string): 提取类型（text, html, attribute, property）

使用示例：
{
  "selector": ".product-item",
  "fields": ["text", "href"],
  "multiple": true
}

你想提取什么数据？
```

---

## 优势总结

### 🎯 一键导出
- 一个 API 调用即可获得完整的 Claude Skill
- 无需手动编写大量文档
- 自动包含最新的 API 信息

### 📖 完整文档
- 包含所有 25 个命令
- 详细的参数说明
- 丰富的使用示例
- 最佳实践指导

### 🔄 自动发现
- SKILL.md 引导 Claude 调用 `/help` 接口
- Claude 可以动态学习最新命令
- 无需更新 Skill 文件

### 💡 智能指导
- 内置工作流建议
- 元素定位方式说明
- 常见场景示例
- 故障排除指南

### 🚀 立即可用
- 下载后直接导入 Claude
- 无需额外配置
- 开箱即用

---

## 完整示例：从导出到使用

```bash
# 1. 导出 SKILL.md
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL.md

# 2. 检查文件
echo "✅ SKILL.md exported successfully"
head -30 EXECUTOR_SKILL.md

# 3. 在 Claude 中导入这个文件

# 4. 开始使用
echo "Now you can ask Claude to automate any browser task!"
```

**Claude 的能力**:
```
用户: 帮我监控这个页面，如果价格低于 $100 就告诉我

Claude:
好的，我来设置一个监控...

1. 导航到页面
2. 获取语义树找到价格元素
3. 提取当前价格
4. 判断是否低于 $100
5. 如果是，通知你

当前价格：$129
还没有达到你的目标价格，我会继续监控...
```

---

## 相关接口

### 1. Executor Help API
```bash
# 查看所有命令
GET /api/v1/executor/help

# 查看特定命令
GET /api/v1/executor/help?command=click
```

### 2. Scripts Export API
```bash
# 导出脚本 Skills
POST /api/v1/scripts/export/skill
{
  "script_ids": []  # 空数组表示导出所有脚本
}
```

---

## 最佳实践

### 1. 定期更新

当 API 更新后，重新导出：
```bash
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL_$(date +%Y%m%d).md
```

### 2. 版本管理

保存不同版本的 SKILL.md：
```bash
EXECUTOR_SKILL_v1.0.0.md
EXECUTOR_SKILL_v1.1.0.md
EXECUTOR_SKILL_latest.md
```

### 3. 团队共享

将导出的 SKILL.md 分享给团队：
```bash
# 上传到共享位置
cp EXECUTOR_SKILL.md /shared/claude-skills/

# 或通过 Git 管理
git add EXECUTOR_SKILL.md
git commit -m "Update Executor Skill"
git push
```

---

## 相关文档

- [EXECUTOR_HTTP_API.md](./EXECUTOR_HTTP_API.md) - 完整的 HTTP API 文档
- [EXECUTOR_HELP_API.md](./EXECUTOR_HELP_API.md) - Help API 详细说明
- [EXECUTOR_HTTP_API_SUMMARY.md](./EXECUTOR_HTTP_API_SUMMARY.md) - API 总结

---

## 总结

✅ **一键导出**: `GET /api/v1/executor/export/skill`
✅ **完整文档**: 包含所有 25 个命令的详细说明
✅ **自动发现**: 引导 Claude 调用 `/help` 动态学习
✅ **立即可用**: 下载后直接导入 Claude
✅ **持续更新**: 通过 Help API 保持最新

现在你可以一键生成 Claude Skills 文档，让 Claude 立即掌握浏览器自动化能力！🎉

```bash
# 一条命令，搞定一切
curl -X GET 'http://localhost:8080/api/v1/executor/export/skill' \
  -H 'X-BrowserWing-Key: your-api-key' \
  -o EXECUTOR_SKILL.md
```

然后在 Claude 中导入，开始自动化吧！🚀
