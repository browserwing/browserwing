package models

import "time"

// BrowserConfig 浏览器配置
type BrowserConfig struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`         // 配置名称
	Description string    `json:"description"`  // 配置描述
	IsDefault   bool      `json:"is_default"`   // 是否为默认配置
	
	// 网站匹配规则
	URLPattern  string    `json:"url_pattern"`  // URL正则匹配模式，为空表示默认配置
	
	// 浏览器行为配置
	UserAgent   string    `json:"user_agent"`   // User Agent，为空使用默认
	UseStealth  *bool     `json:"use_stealth"`  // 是否使用 Stealth 模式，nil表示使用默认
	Headless    *bool     `json:"headless"`     // 是否使用 Headless 模式，nil表示使用默认(false)
	LaunchArgs  []string  `json:"launch_args"`  // 启动参数，为空使用默认
	
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
