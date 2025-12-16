package models

import (
	"encoding/json"
	"time"
)

// LLMConfigModel LLM配置数据库模型
type LLMConfigModel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`       // LLM 名称标识
	Provider  string    `json:"provider"`   // openai, anthropic, custom 等
	APIKey    string    `json:"api_key"`    // API密钥
	Model     string    `json:"model"`      // 模型名称，如 gpt-4, claude-3
	BaseURL   string    `json:"base_url"`   // 自定义 API 地址
	IsDefault bool      `json:"is_default"` // 是否为默认配置
	IsActive  bool      `json:"is_active"`  // 是否启用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToJSON 转换为JSON字节
func (l *LLMConfigModel) ToJSON() ([]byte, error) {
	return json.Marshal(l)
}

// FromJSON 从JSON字节解析
func (l *LLMConfigModel) FromJSON(data []byte) error {
	return json.Unmarshal(data, l)
}
