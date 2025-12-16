package models

import (
	"time"
)

// RecordingConfig 录制配置
type RecordingConfig struct {
	ID        string    `json:"id"`         // 配置 ID（固定为 "default"）
	Enabled   bool      `json:"enabled"`    // 是否启用录制
	FrameRate int       `json:"frame_rate"` // 帧率（默认 15）
	Quality   int       `json:"quality"`    // 质量 0-100（默认 70）
	Format    string    `json:"format"`     // 输出格式（默认 "mp4"）
	OutputDir string    `json:"output_dir"` // 输出目录（默认 "recordings"）
	CreatedAt time.Time `json:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at"` // 更新时间
}

// GetDefaultRecordingConfig 获取默认录制配置
func GetDefaultRecordingConfig() *RecordingConfig {
	return &RecordingConfig{
		ID:        "default",
		Enabled:   false,
		FrameRate: 15,
		Quality:   70,
		Format:    "gif",
		OutputDir: "recordings",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
