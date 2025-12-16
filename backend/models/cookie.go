package models

import (
	"encoding/json"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// CookieStore Cookie存储模型
type CookieStore struct {
	ID        string                 `json:"id"`         // 存储ID，通常使用平台名称如 "xiaohongshu"
	Platform  string                 `json:"platform"`   // 平台名称
	Cookies   []*proto.NetworkCookie `json:"cookies"`    // Cookie列表
	CreatedAt time.Time              `json:"created_at"` // 创建时间
	UpdatedAt time.Time              `json:"updated_at"` // 更新时间
}

// ToJSON 将CookieStore转换为JSON
func (c *CookieStore) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON 从JSON解析CookieStore
func (c *CookieStore) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}
