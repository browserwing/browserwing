package models

import "time"

// ToolType 工具类型
type ToolType string

const (
	ToolTypePreset ToolType = "preset" // 预设工具
	ToolTypeScript ToolType = "script" // 脚本工具
)

// ToolConfig 工具配置
type ToolConfig struct {
	ID          string                 `json:"id"`          // 工具唯一标识
	Name        string                 `json:"name"`        // 工具名称
	Type        ToolType               `json:"type"`        // 工具类型: preset | script
	Description string                 `json:"description"` // 工具描述
	Enabled     bool                   `json:"enabled"`     // 是否启用
	Parameters  map[string]interface{} `json:"parameters"`  // 工具参数配置
	ScriptID    string                 `json:"script_id"`   // 关联的脚本ID (仅 script 类型)
	CreatedAt   time.Time              `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time              `json:"updated_at"`  // 更新时间
}

// PresetToolMetadata 预设工具元数据（用于描述工具支持的参数）
type PresetToolMetadata struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Parameters  []PresetToolParameterSchema `json:"parameters"` // 支持的参数定义
}

// PresetToolParameterSchema 预设工具参数Schema
type PresetToolParameterSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, number, boolean
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}
