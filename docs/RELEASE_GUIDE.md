# BrowserWing Release Guide

本文档说明如何发布 BrowserWing 的新版本。

## 发布流程

### 1. 构建所有平台的二进制文件

```bash
# 构建所有平台
make release
```

这个命令会：
- 构建前端并嵌入到后端
- 编译 6 个平台的二进制文件（darwin 和 mac 别名）
- 生成压缩包（`.tar.gz` 和 `.zip`）
- 将所有文件复制到 `build/release/` 目录

**生成的文件包括：**

二进制文件（直接可执行）：
- `browserwing-darwin-amd64` (macOS Intel)
- `browserwing-darwin-arm64` (macOS Apple Silicon)
- `browserwing-mac-amd64` (darwin 别名，用户友好)
- `browserwing-mac-arm64` (darwin 别名，用户友好)
- `browserwing-linux-amd64` (Linux x86_64)
- `browserwing-linux-arm64` (Linux ARM64)
- `browserwing-windows-amd64.exe` (Windows x86_64)
- `browserwing-windows-arm64.exe` (Windows ARM64)

压缩包（用于快速下载）：
- `browserwing-darwin-amd64.tar.gz`
- `browserwing-darwin-arm64.tar.gz`
- `browserwing-mac-amd64.tar.gz`
- `browserwing-mac-arm64.tar.gz`
- `browserwing-linux-amd64.tar.gz`
- `browserwing-linux-arm64.tar.gz`
- `browserwing-windows-amd64.zip`
- `browserwing-windows-arm64.zip`

### 2. 创建 GitHub Release 和 Gitee Release

#### 2.1 GitHub Release

**方式 A - 使用 GitHub CLI（推荐）：**

```bash
# 安装 gh（如果还没有）
# macOS: brew install gh
# Linux: https://github.com/cli/cli/blob/trunk/docs/install_linux.md
# Windows: https://github.com/cli/cli#installation

# 登录
gh auth login

# 创建 Release 并上传所有文件
gh release create v0.0.1 \
  --title "v0.0.1" \
  --notes "Release notes here" \
  build/release/*
```

**方式 B - 使用 GitHub Web 界面：**

1. 访问 https://github.com/browserwing/browserwing/releases/new
2. 填写：
   - **Tag**: `v0.0.1` (必须以 v 开头)
   - **Title**: `v0.0.1` 或 `BrowserWing v0.0.1`
   - **Description**: 描述本次更新内容
3. 上传 `build/release/` 目录下的所有文件（二进制 + 压缩包）
4. 点击 "Publish release"

#### 2.2 Gitee Release（国内镜像）

为国内用户提供 Gitee 镜像：

1. 访问 https://gitee.com/browserwing/browserwing/releases/new
2. 填写：
   - **标签名称**: `v0.0.1`
   - **发行版标题**: `v0.0.1`
   - **发行说明**: 同 GitHub
3. 上传 `build/release/` 目录下的所有文件（与 GitHub 相同）
4. 点击 "发布"

**重要：** 确保 GitHub 和 Gitee 的 Release 文件完全一致，以便安装脚本可以在两个镜像之间自动切换。

### 3. 发布到 npm

```bash
cd npm

# 更新版本号（与 GitHub tag 一致，不带 v 前缀）
npm version 0.0.1 --no-git-tag-version

# 发布
./publish.sh

# 或手动发布
npm publish
```

发布前确保：
- ✅ 已登录 npm：`npm login`
- ✅ GitHub Release 已创建并上传了所有文件
- ✅ 版本号匹配（npm 用 `0.0.1`，GitHub 用 `v0.0.1`）

### 4. 验证安装

测试所有安装方式是否正常：

```bash
# 测试 curl 安装脚本
curl -fsSL https://raw.githubusercontent.com/browserwing/browserwing/main/install.sh | bash

# 测试 npm 安装
npm install -g browserwing

# 验证运行
browserwing --version
```

## 版本号规则

使用语义化版本（Semantic Versioning）：

- **MAJOR.MINOR.PATCH** (例如：1.2.3)
  - **MAJOR**: 不兼容的 API 变更
  - **MINOR**: 向后兼容的新功能
  - **PATCH**: 向后兼容的 bug 修复

**示例：**
- `0.0.1` → `0.0.2` (bug 修复)
- `0.0.2` → `0.1.0` (新功能)
- `0.1.0` → `1.0.0` (重大更新)

## 文件名规范

**必须严格遵守以下命名规则：**

**二进制文件：**
```
browserwing-darwin-amd64           # macOS Intel (标准名)
browserwing-darwin-arm64           # macOS Apple Silicon (标准名)
browserwing-mac-amd64              # macOS Intel (用户友好别名)
browserwing-mac-arm64              # macOS Apple Silicon (用户友好别名)
browserwing-linux-amd64            # Linux x86_64
browserwing-linux-arm64            # Linux ARM64
browserwing-windows-amd64.exe      # Windows x86_64
browserwing-windows-arm64.exe      # Windows ARM64
```

**压缩包：**
```
browserwing-darwin-amd64.tar.gz
browserwing-darwin-arm64.tar.gz
browserwing-mac-amd64.tar.gz
browserwing-mac-arm64.tar.gz
browserwing-linux-amd64.tar.gz
browserwing-linux-arm64.tar.gz
browserwing-windows-amd64.zip
browserwing-windows-arm64.zip
```

**注意：**
- macOS 同时提供 `darwin`（标准名）和 `mac`（别名）两个版本
- Windows 文件必须有 `.exe` 后缀
- Unix 系统使用 `.tar.gz` 压缩，Windows 使用 `.zip` 压缩
- 安装脚本默认下载压缩包，速度更快

## 完整发布清单

### 构建阶段
- [ ] 更新 `Makefile` 中的 `VERSION` 变量
- [ ] 运行 `make release` 构建所有平台
- [ ] 检查 `build/release/` 目录，确认所有文件都存在：
  - [ ] 8 个二进制文件（darwin、mac、linux、windows）
  - [ ] 8 个压缩包（.tar.gz 和 .zip）

### 发布阶段
- [ ] 创建 GitHub Release（tag: `v0.0.1`）
  - [ ] 上传 `build/release/` 下的所有文件
  - [ ] 发布 Release（点击 "Publish release"）
- [ ] 创建 Gitee Release（tag: `v0.0.1`）
  - [ ] 上传相同的所有文件
  - [ ] 发布 Release
- [ ] 发布到 npm
  - [ ] 更新 `npm/package.json` 版本号（`0.0.1`，不带 v）
  - [ ] 运行 `cd npm && ./publish.sh`

### 测试阶段
- [ ] 测试安装脚本（Linux/macOS）：`curl ... | bash`
- [ ] 测试安装脚本（Windows）：`iwr ... | iex`
- [ ] 测试 npm 安装：`npm install -g browserwing`
- [ ] 测试 pnpm 安装：`pnpm add -g browserwing`
- [ ] 验证二进制运行：`browserwing --version`

### 文档阶段
- [ ] 更新 CHANGELOG（如果有的话）
- [ ] 检查 README 中的版本号和链接
- [ ] 更新相关文档

## 常见问题

### Q1: 构建失败

**检查：**
```bash
# 前端依赖
cd frontend && pnpm install

# 后端依赖
cd backend && go mod download

# 清理后重建
make clean
make release
```

### Q2: 安装脚本 404 错误

**原因：** GitHub 或 Gitee Release 中没有对应的文件

**解决：**
1. 确认 Release 已发布（不是 Draft）
2. 确认文件名完全匹配：
   - 二进制文件：`browserwing-darwin-amd64`、`browserwing-mac-amd64` 等
   - 压缩包：`browserwing-darwin-amd64.tar.gz`、`browserwing-windows-amd64.zip` 等
3. 确认 GitHub 和 Gitee 都已上传
4. 等待几分钟（CDN 同步）

**测试下载链接：**
```bash
# 测试 GitHub
curl -I https://github.com/browserwing/browserwing/releases/download/v0.0.1/browserwing-darwin-arm64.tar.gz

# 测试 Gitee
curl -I https://gitee.com/browserwing/browserwing/releases/download/v0.0.1/browserwing-darwin-arm64.tar.gz
```

### Q3: npm 发布权限错误

**解决：**
```bash
# 检查登录状态
npm whoami

# 重新登录
npm logout
npm login

# 如果需要 2FA
# 访问 https://www.npmjs.com/settings/[username]/tfa 启用
```

### Q4: 版本号已存在

**解决：**
```bash
# npm 不允许覆盖已发布版本，需要发布新版本
npm version patch  # 0.0.1 -> 0.0.2
npm publish
```

## 自动化发布（未来）

可以使用 GitHub Actions 自动化发布流程：

1. 创建 Release → 自动触发 CI
2. CI 构建所有平台（二进制 + 压缩包）
3. CI 上传到 GitHub Release
4. CI 自动同步到 Gitee Release（需要配置 Gitee Token）
5. CI 自动发布到 npm

参考 `.github/workflows/npm-publish.yml`（需要配置 `NPM_TOKEN` 和 `GITEE_TOKEN` secrets）

**注意：** Gitee Release 可能需要手动同步，或使用 Gitee API 实现自动化。

## 回滚发布

**撤销 npm 包（72 小时内）：**
```bash
npm unpublish browserwing@0.0.1
```

**弃用版本：**
```bash
npm deprecate browserwing@0.0.1 "有严重 bug，请使用 0.0.2+"
```

**删除 GitHub Release：**
1. 访问 Releases 页面
2. 找到对应版本
3. 点击 "Delete" → 确认

## 参考资源

- Semantic Versioning: https://semver.org/
- npm Publishing: https://docs.npmjs.com/cli/publish
- GitHub Releases: https://docs.github.com/en/repositories/releasing-projects-on-github
