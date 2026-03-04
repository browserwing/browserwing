package models

import "time"

// BrowserInstance 浏览器实例
type BrowserInstance struct {
	ID          string `json:"id"`
	Name        string `json:"name"`        // 实例名称
	Description string `json:"description"` // 实例描述
	IsDefault   bool   `json:"is_default"`  // 是否为默认实例
	IsActive    bool   `json:"is_active"`   // 是否正在运行

	// 实例类型：local（本地）或 remote（远程）
	Type string `json:"type"` // "local" 或 "remote"

	// 本地浏览器配置
	BinPath     string `json:"bin_path"`      // Chrome 可执行文件路径（本地模式）
	UserDataDir string `json:"user_data_dir"` // 用户数据目录路径（本地模式）

	// 远程浏览器配置
	ControlURL string `json:"control_url"` // 远程 Chrome DevTools URL（远程模式）

	// 浏览器行为配置（可选，如果为空则使用默认配置）
	UserAgent  string   `json:"user_agent,omitempty"`  // User Agent
	UseStealth *bool    `json:"use_stealth,omitempty"` // 是否使用 Stealth 模式
	Headless   *bool    `json:"headless,omitempty"`    // 是否使用 Headless 模式
	NoSandbox  *bool    `json:"no_sandbox,omitempty"`  // 是否禁用沙箱模式
	LaunchArgs []string `json:"launch_args,omitempty"` // 启动参数
	Proxy      string   `json:"proxy,omitempty"`       // 代理地址

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
