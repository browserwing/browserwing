package sdk

import "time"

// BrowserStatus 浏览器状态
type BrowserStatus struct {
	IsRunning bool      `json:"is_running"`
	StartTime time.Time `json:"start_time,omitempty"`
}
