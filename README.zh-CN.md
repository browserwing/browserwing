<p align="center">
  <img width="600" alt="BrowserWing" src="https://raw.githubusercontent.com/browserwing/browserwing/main/docs/assets/banner.svg">
</p>

<p align="center">
  <a href="./README.md">English</a> · 简体中文 · <a href="./README.ja.md">日本語</a> · <a href="./README.es.md">Español</a> · <a href="./README.pt.md">Português</a>
</p>

<p align="center"><a href="https://browserwing.com">browserwing.com</a></p>

## 特性亮点

**原生浏览器自动化平台 + AI 集成**

- **完整浏览器控制**：26+ HTTP API 端点，提供全功能浏览器自动化能力
- **内置 AI Agent**：直接通过对话界面进行浏览器自动化任务
- **通用 AI 工具集成**：原生支持 MCP & Skills 协议 - 兼容任何支持这些标准的 AI 工具
- **可视化脚本录制**：记录浏览器操作，可视化编辑，精确重放
- **灵活导出选项**：将录制的脚本转换为 MCP 命令或 Skills 文件，用于 AI 工具集成
- **智能数据提取**：LLM 驱动的语义提取，支持 OpenAI、Claude、DeepSeek 等
- **会话管理**：强大的 Cookie 和存储处理，保证稳定、认证的浏览会话

## 环境要求

- 环境中需安装 Google Chrome 或 Chromium，并可正常访问。

## 截图

<img width="600" alt="BrowserWing Homepage" src="https://raw.githubusercontent.com/browserwing/browserwing/main/docs/assets/screenshot_homepage.png">

## 快速开始

### 方式 A — 通过包管理器安装（推荐）

**使用 npm：**
```bash
npm install -g browserwing
browserwing --port 8080
```

**使用 pnpm：**
```bash
pnpm add -g browserwing
browserwing --port 8080
```

npm 包在安装时会自动测试 GitHub 和 Gitee 镜像，选择最快的源进行下载。

**⚠️ macOS 用户注意：**  
如果运行时遇到 "killed" 错误，请执行以下命令修复：
```bash
xattr -d com.apple.quarantine $(which browserwing)
```
详细说明请参见：[macOS 安装问题修复指南](docs/MACOS_INSTALLATION_FIX.md)

**使用 Homebrew (macOS/Linux)：**
```bash
# 即将支持
brew install browserwing
```

### 方式 B — 一键安装脚本

**Linux / macOS：**
```bash
curl -fsSL https://raw.githubusercontent.com/browserwing/browserwing/main/install.sh | bash
```

**Windows (PowerShell)：**
```powershell
iwr -useb https://raw.githubusercontent.com/browserwing/browserwing/main/install.ps1 | iex
```

脚本将自动：
- 检测系统和架构
- 测试 GitHub 和 Gitee 镜像，选择最快的源
- 下载并解压二进制文件
- 添加到 PATH

**然后启动 BrowserWing：**
```bash
browserwing --port 8080
# 在浏览器中打开 http://localhost:8080
```

**国内用户友好：** 安装脚本会自动检测并使用 Gitee 镜像（如果 GitHub 较慢）。

### 方式 C — 手动下载

从 [Releases](https://github.com/browserwing/browserwing/releases) 下载对应操作系统的预构建二进制文件：

如果 GitHub 下载遇到问题，可以访问：[Gitee Releases](https://gitee.com/browserwing/browserwing/releases)

```bash
# Linux/macOS
chmod +x ./browserwing
./browserwing --port 8080

# Windows (PowerShell)
./browserwing.exe --port 8080
```

### 方式 D — 源码构建

```bash
# 安装依赖（需要 Go 与 pnpm）
make install

# 构建集成版本（前端嵌入后端）
make build-embedded
./build/browserwing --port 8080

# 或构建全部目标并打包
make build-all
make package
```

## 快速集成到 AI 工具

**三种使用方式：**

### 1. MCP 服务器集成

在任何支持 MCP 的 AI 工具中配置 BrowserWing 为 MCP 服务器：

```json
{
  "mcpServers": {
    "browserwing": {
      "type": "http",
      "url": "http://localhost:8080/api/v1/mcp/message"
    }
  }
}
```

将此配置粘贴到 AI 工具的 MCP 设置中，即可启用浏览器自动化能力。

### 2. Skills 文件集成

下载并导入 Skills 文件到任何支持 Skills 协议的 AI 工具：

1. 启动 BrowserWing
2. 从仓库下载 [SKILL.md](https://raw.githubusercontent.com/browserwing/browserwing/refs/heads/main/SKILL.md)
3. 导入到 AI 工具的 Skills 设置
4. 使用自然语言命令开始自动化

**示例：**
```
"访问淘宝，搜索 'MacBook'，提取前 5 个商品的价格"
```

### 3. 直接使用 AI Agent

使用 BrowserWing 内置的 AI Agent 进行即时浏览器自动化：

1. 打开 BrowserWing 网页界面：`http://localhost:8080`
2. 进入 "AI Agent" 部分
3. 配置 LLM（OpenAI、Claude、DeepSeek 等）
4. 开始对话式浏览器自动化

**导出自定义脚本：**
```bash
# 将录制的脚本导出为 Skills 或 MCP 命令
curl -X POST 'http://localhost:8080/api/v1/scripts/export/skill' \
  -H 'Content-Type: application/json' \
  -d '{"script_ids": []}' \
  -o MY_CUSTOM_SCRIPTS.md
```

## 为什么选择 BrowserWing

**专业浏览器自动化 + AI 集成**

- **通用协议支持**：原生 MCP & Skills 实现，兼容任何支持这些协议的 AI 工具
- **完整自动化 API**：26+ HTTP 端点，提供全面的浏览器控制能力
- **灵活集成选项**：可作为 MCP 服务器、Skills 文件或独立 AI Agent 使用
- **可视化工作流构建器**：无需编写代码即可录制、编辑和重放浏览器操作
- **Token 高效设计**：针对 LLM 使用优化，快速性能且最小 token 消耗
- **生产就绪**：稳定的会话管理、Cookie 处理和错误恢复
- **可扩展架构**：将录制的脚本转换为可复用的 MCP 命令或 Skills 文件
- **多 LLM 支持**：兼容 OpenAI、Anthropic、DeepSeek 等多家服务商
- **企业级应用场景**：数据提取、RPA、测试、监控和智能体驱动的自动化

## 使用指南

### 三步快速开始

1. **选择集成方式**
   - 复制 MCP 服务器配置以集成到 AI 工具
   - 下载 Skills 文件用于支持 Skills 的 AI 工具
   - 或使用内置 AI Agent 立即开始

2. **配置 AI 工具**
   - 将 MCP 配置或 Skills 文件导入到你偏好的 AI 工具
   - 配置 LLM 设置（API 密钥、模型选择）
   - 验证与 BrowserWing 的连接

3. **开始自动化**
   - 通过自然语言命令控制浏览器
   - 录制自定义脚本用于重复任务
   - 将脚本导出为 MCP 命令或 Skills 以便复用

### 高级工作流

**浏览器自动化：**
- 启动和管理多个浏览器实例
- 配置配置文件、代理和浏览器设置
- 处理 Cookie 和认证会话
- 执行复杂的交互序列

**脚本录制：**
- 捕获点击、输入、导航和等待操作
- 在脚本编辑器中可视化编辑动作
- 通过逐步重放进行测试和调试
- 添加变量和条件逻辑

**AI 集成：**
- 将脚本转换为 MCP 命令或 Skills 文件
- 集成多个 LLM 提供商
- 使用语义提取进行数据解析
- 构建智能体驱动的自动化工作流

### HTTP API 参考

BrowserWing 提供 26+ 个 RESTful 端点用于编程式浏览器控制：

**导航与控制**
- 导航到 URL、后退/前进、刷新页面
- 管理浏览器窗口和标签页
- 处理页面加载和超时

**元素交互**
- 点击、输入、选择和悬停操作
- 文件上传和表单提交
- 键盘快捷键和按键操作

**数据提取**
- 提取文本、HTML 和属性
- 使用 LLM 进行语义内容分析
- 截图捕获（整页或元素）

**高级操作**
- 执行自定义 JavaScript
- 管理 Cookie 和本地存储
- 批量操作以提高效率
- 等待条件和元素可见性

**完整文档**：详细的端点规范请参阅 `docs/EXECUTOR_HTTP_API.md`

## 参与贡献

欢迎提交 Issue 和 PR，请附上复现步骤或清晰的动机。新特性建议请在讨论区提出，描述使用场景与预期结果。

## 社区

- **Discord**: [https://discord.gg/BkqcApRj](https://discord.gg/BkqcApRj)
- **Twitter**: [https://x.com/chg80333](https://x.com/chg80333)
- **QQ 群**：点击链接加入群聊[【Browserwing用户群】](https://qun.qq.com/universal-share/share?ac=1&authKey=Wk%2FnSWvWLNO8Cegxo1PFqUmF%2Bntymd9JFl1l1n0GCwpWjeR2Yo7K91PgnugnK8N9&busi_data=eyJncm91cENvZGUiOiIxMDc4MTQwMTU1IiwidG9rZW4iOiJPa1pLeTVqai9EV09DRUpFeHM3dWVwclU5NW5LRDNRaEJ0ZTVld2lMbmFOelgxZWhia2JpZHhsc2hYbmxWdW1RIiwidWluIjoiMzE3NTQyNTQ4MCJ9&data=HbgiLCOhCT4c68pCpyI0whItk4SppgqtsjnQMaiP_zUtfM1O62y6jUFBVH0moLnQ_1ucw9gilYKMuMNux9F-FQ&svctype=4&tempid=h5_group_info)
- **微信交流群**：添加微信 mongorz（备注 browserwing）

扫码加入微信群：

<img width="150" alt="BrowserWing 微信群" src="https://raw.githubusercontent.com/browserwing/browserwing/main/docs/assets/wechat_group.jpg">

## 致谢

灵感源自现代浏览器自动化、智能体工作流与 MCP。

## 许可证

采用 MIT 许可证，详见 `LICENSE`。

## 免责声明

请勿用于任何非法用途或违反网站条款的行为。仅供个人学习与合规自动化使用。
