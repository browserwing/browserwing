# LLM API 配置指南

## 为什么需要配置 LLM API？

BrowserWing 使用大语言模型（LLM）来自动生成文章内容。要使用 AI 写作功能，你需要配置一个 LLM 提供商的 API Key。

**注意**：即使没有配置 LLM API，你仍然可以使用其他功能（数据抓取、手动编辑、发布等）。

## 支持的 LLM 提供商

- ✅ **OpenAI** (GPT-3.5, GPT-4)
- ✅ **DeepSeek** (DeepSeek-Chat)
- ✅ 其他兼容 OpenAI API 格式的提供商

## 快速配置

### 方式 1: 环境变量（推荐）

```bash
# 设置 API Key
export LLM_API_KEY="your-api-key-here"

# 启动应用
./start.sh
```

### 方式 2: 配置文件

编辑 `backend/config.toml`：

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": "8080"
  },
  "database": {
    "path": "./data/browserwing.db"
  },
  "llm": {
    "provider": "openai",
    "api_key": "your-api-key-here",
    "model": "gpt-3.5-turbo"
  }
}
```

**安全提示**：不要将包含真实 API Key 的配置文件提交到 Git！

## 详细配置步骤

### 使用 OpenAI

#### 1. 获取 API Key

1. 访问 [OpenAI Platform](https://platform.openai.com/)
2. 注册/登录账号
3. 进入 [API Keys](https://platform.openai.com/api-keys) 页面
4. 点击 "Create new secret key"
5. 复制生成的 API Key

#### 2. 配置

```bash
# 设置环境变量
export LLM_API_KEY="sk-..."

# 或编辑 config.toml
{
  "llm": {
    "provider": "openai",
    "api_key": "sk-...",
    "model": "gpt-3.5-turbo"
  }
}
```

#### 3. 可用模型

- `gpt-3.5-turbo` - 快速、经济（推荐）
- `gpt-4` - 更强大，但更贵
- `gpt-4-turbo` - GPT-4 的快速版本

### 使用 DeepSeek

#### 1. 获取 API Key

1. 访问 [DeepSeek Platform](https://platform.deepseek.com/)
2. 注册/登录账号
3. 获取 API Key

#### 2. 配置

```bash
# 环境变量
export LLM_API_KEY="your-deepseek-key"

# config.toml
{
  "llm": {
    "provider": "deepseek",
    "api_key": "your-deepseek-key",
    "model": "deepseek-chat"
  }
}
```

### 使用其他提供商

如果你的提供商兼容 OpenAI API 格式，可以这样配置：

```json
{
  "llm": {
    "provider": "openai",
    "api_key": "your-key",
    "model": "your-model"
  }
}
```

## 验证配置

### 1. 启动后端

```bash
cd backend
export LLM_API_KEY="your-key"
go run main.go
```

查看日志，应该看到：
```
✓ LLM写作器初始化成功
```

如果看到警告：
```
警告: 初始化LLM写作器失败: ... (将无法生成文章)
```

说明配置有问题。

### 2. 测试生成文章

1. 访问前端: http://localhost:5173
2. 点击 "生成文章"
3. 选择抓取器和输入提示词
4. 点击 "生成文章"

如果配置正确，应该能成功生成文章。

如果看到错误：
```
⚠️ AI 写作服务未配置
```

说明需要配置 LLM API Key。

## 常见问题

### Q1: 没有 API Key 可以使用吗？

**A**: 可以！没有 API Key 时，你仍然可以：
- ✅ 使用数据抓取功能
- ✅ 手动创建和编辑文章
- ✅ 发布文章到各平台
- ❌ 无法使用 AI 自动生成文章

### Q2: API Key 安全吗？

**A**: 请注意：
- ✅ 使用环境变量（不会提交到 Git）
- ✅ 不要将 API Key 硬编码到代码中
- ✅ 不要将包含 API Key 的配置文件提交到版本控制
- ✅ 定期轮换 API Key

### Q3: 如何选择模型？

**A**: 建议：
- **开发/测试**: `gpt-3.5-turbo` (快速、便宜)
- **生产环境**: `gpt-4` (质量更高)
- **预算有限**: `deepseek-chat` (性价比高)

### Q4: API 调用失败怎么办？

**A**: 检查：
1. API Key 是否正确
2. 账户是否有余额
3. 网络是否能访问 API 服务器
4. 是否超过了速率限制

查看后端日志获取详细错误信息。

### Q5: 如何估算成本？

**A**: 成本取决于：
- 使用的模型（GPT-4 > GPT-3.5）
- 生成的文章长度
- 调用频率

示例（OpenAI GPT-3.5-turbo）：
- 输入: $0.0015 / 1K tokens
- 输出: $0.002 / 1K tokens
- 一篇 1000 字文章约 $0.01-0.02

## 配置示例

### 开发环境

```bash
# .env 或 .bashrc
export LLM_API_KEY="sk-test-..."
export LLM_MODEL="gpt-3.5-turbo"
```

### 生产环境

```bash
# 使用环境变量（推荐）
export LLM_API_KEY="sk-prod-..."
export LLM_MODEL="gpt-4"

# 或使用配置管理工具
# - Docker secrets
# - Kubernetes secrets
# - AWS Secrets Manager
```

### Docker 部署

```dockerfile
# Dockerfile
ENV LLM_API_KEY=""

# docker-compose.yml
services:
  backend:
    environment:
      - LLM_API_KEY=${LLM_API_KEY}
```

```bash
# 运行时传入
docker run -e LLM_API_KEY="your-key" browserwing-backend
```

## 故障排查

### 错误: "API key is invalid"

**原因**: API Key 错误或已失效

**解决**:
1. 检查 API Key 是否正确复制
2. 确认 API Key 未过期
3. 重新生成新的 API Key

### 错误: "Insufficient quota"

**原因**: 账户余额不足

**解决**:
1. 检查账户余额
2. 充值或升级账户
3. 切换到其他提供商

### 错误: "Rate limit exceeded"

**原因**: 超过调用频率限制

**解决**:
1. 降低调用频率
2. 升级账户等级
3. 实现请求队列和重试机制

### 错误: "Connection timeout"

**原因**: 网络连接问题

**解决**:
1. 检查网络连接
2. 配置代理（如需要）
3. 检查防火墙设置

## 最佳实践

1. **使用环境变量**: 不要硬编码 API Key
2. **监控使用量**: 定期检查 API 使用情况
3. **设置预算**: 在提供商平台设置使用限额
4. **错误处理**: 实现优雅的降级策略
5. **日志记录**: 记录 API 调用但不记录 API Key

## 相关资源

- [OpenAI API 文档](https://platform.openai.com/docs)
- [DeepSeek API 文档](https://platform.deepseek.com/docs)
- [llmhub 库文档](https://github.com/gotoailab/llmhub)

## 获取帮助

如果遇到问题：
1. 查看后端日志
2. 检查 API 提供商的状态页面
3. 查阅本项目的 Issues
4. 联系 API 提供商支持

---

**配置完成后，重启后端服务即可使用 AI 写作功能！** 🎉

