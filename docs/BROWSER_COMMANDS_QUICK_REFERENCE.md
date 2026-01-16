# Browser 命令快速参考

## 完整命令列表（20个）

### 导航 (Navigation) - 3个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_navigate` | 导航到 URL | ✅ 原有 |
| `browser_navigate_back` | 后退 | ✅ 原有 (GoBack) |
| `browser_scroll` | 滚动页面 | ✅ 原有 |

### 交互 (Interaction) - 7个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_click` | 点击元素 | ✅ 原有 |
| `browser_type` | 输入文本 | ✅ 原有 |
| `browser_select_option` | 选择下拉选项 | ✅ 原有 (Select) |
| `browser_hover` | 鼠标悬停 | ✅ 原有 |
| `browser_press_key` | 按键 | ✨ 新增 |
| `browser_drag` | 拖拽元素 | ✨ 新增 |
| `browser_file_upload` | 上传文件 | ✨ 新增 |

### 捕获 (Capture) - 1个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_take_screenshot` | 截图 | ✨ 新增 |

### 脚本 (Scripting) - 2个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_evaluate` | 执行 JS | ✨ 新增 |
| `browser_run_code` | 运行代码 | ✨ 新增 (同 evaluate) |

### 数据 (Data) - 1个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_extract` | 提取数据 | ✅ 原有 |

### 分析 (Analysis) - 2个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_get_semantic_tree` | 获取语义树 | ✅ 原有 |
| `browser_get_page_info` | 获取页面信息 | ✅ 原有 |

### 同步 (Synchronization) - 1个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_wait_for` | 等待元素 | ✅ 原有 |

### 窗口 (Window) - 2个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_resize` | 调整窗口大小 | ✨ 新增 |
| `browser_close` | 关闭页面 | ✨ 新增 |

### 对话框 (Dialog) - 1个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_handle_dialog` | 处理对话框 | ✨ 新增 |

### 调试 (Debug) - 2个
| 命令 | 功能 | 状态 |
|------|------|------|
| `browser_console_messages` | 获取控制台消息 | ✨ 新增 |
| `browser_network_requests` | 获取网络请求 | ✨ 新增 |

---

## 未实现的命令（原列表中）

以下命令在原始列表中，但实际上已通过其他方式实现或不需要单独实现：

1. ❌ `browser_install` - 不需要实现（浏览器管理功能）
2. ✅ `browser_navigate_back` - 已实现为 `GoBack()` 函数
3. ✅ `browser_snapshot` - 已实现为 `browser_take_screenshot`
4. ✅ `browser_tabs` - 通过现有的页面管理功能实现
5. ✅ `browser_fill_form` - 通过组合 `type` 和 `select` 实现

---

## 快速使用示例

### 基础操作
```javascript
// 导航
browser_navigate({ url: "https://example.com" })

// 点击
browser_click({ identifier: "Clickable Element [1]" })

// 输入
browser_type({ identifier: "Input Element [1]", text: "Hello" })

// 按键
browser_press_key({ key: "Enter" })
```

### 高级操作
```javascript
// 截图
browser_take_screenshot({ full_page: true, format: "png" })

// 执行脚本
browser_evaluate({ script: "document.title" })

// 拖拽
browser_drag({ 
  from_identifier: "Element [1]", 
  to_identifier: "Element [2]" 
})

// 上传文件
browser_file_upload({ 
  identifier: "Input Element [1]",
  file_paths: ["/path/to/file.jpg"]
})
```

### 调试操作
```javascript
// 获取控制台消息
browser_console_messages({})

// 获取网络请求
browser_network_requests({})

// 调整窗口大小
browser_resize({ width: 1920, height: 1080 })
```

---

## 统计

- **总命令数**: 20
- **原有命令**: 10
- **新增命令**: 10 ✨
- **覆盖场景**: 
  - ✅ 页面导航
  - ✅ 元素交互
  - ✅ 表单填写
  - ✅ 键盘操作
  - ✅ 拖拽操作
  - ✅ 文件上传
  - ✅ 截图捕获
  - ✅ 脚本执行
  - ✅ 对话框处理
  - ✅ 调试监控

---

## 相关文档

- 详细文档: `BROWSER_COMMANDS_COMPLETED.md`
- 工具配置: `EXECUTOR_TOOLS_CONFIG_INTEGRATION.md`
- API 使用: 参见各命令的 MCP 工具定义
