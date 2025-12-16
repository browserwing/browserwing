package tools

import (
	"context"
	"fmt"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/storage"
)

// GetPresetToolsMetadata 获取所有预设工具的元数据
func GetPresetToolsMetadata() []models.PresetToolMetadata {
	return []models.PresetToolMetadata{
		{
			ID:          "fileops",
			Name:        "File Operations",
			Description: "Read, write, and manipulate local files",
			Parameters: []models.PresetToolParameterSchema{
				{
					Name:        "root_directory",
					Type:        "string",
					Description: "Root directory for file operations (safety restriction)",
					Required:    false,
					Default:     "./",
				},
			},
		},
		{
			ID:          "bark",
			Name:        "Bark Push",
			Description: "Send iOS push notifications via Bark service",
			Parameters: []models.PresetToolParameterSchema{
				{
					Name:        "api_key",
					Type:        "string",
					Description: "Bark device key",
					Required:    true,
				},
			},
		},
		{
			ID:          "websearch",
			Name:        "Web Search",
			Description: "Search web pages and return content in markdown format",
			Parameters:  []models.PresetToolParameterSchema{},
		},
		{
			ID:          "git",
			Name:        "Git",
			Description: "Execute Git commands locally",
			Parameters: []models.PresetToolParameterSchema{
				{
					Name:        "default_workdir",
					Type:        "string",
					Description: "Default working directory for git commands",
					Required:    false,
					Default:     "./",
				},
			},
		},
		{
			ID:          "pyexec",
			Name:        "Python Executor",
			Description: "Execute Python code locally",
			Parameters:  []models.PresetToolParameterSchema{},
		},
	}
}

// InitPresetTools 初始化所有预设工具
func InitPresetTools(ctx context.Context, toolReg *tools.Registry, db *storage.BoltDB) error {
	if toolReg == nil {
		return fmt.Errorf("tool registry cannot be empty")
	}

	// 获取所有工具配置
	toolConfigs, err := db.ListToolConfigs()
	if err != nil {
		// 如果数据库为空，初始化默认配置
		toolConfigs = initDefaultToolConfigs(db)
	}

	// 构建配置映射
	configMap := make(map[string]*models.ToolConfig)
	for _, cfg := range toolConfigs {
		if cfg.Type == models.ToolTypePreset {
			configMap[cfg.ID] = cfg
		}
	}

	// 注册所有预设工具（根据配置）
	registerToolIfEnabled(toolReg, "fileops", configMap, func(params map[string]interface{}) interfaces.Tool {
		return &FileOpsTool{RootDir: getStringParam(params, "root_directory", "./")}
	})

	registerToolIfEnabled(toolReg, "bark", configMap, func(params map[string]interface{}) interfaces.Tool {
		return &BarkTool{APIKey: getStringParam(params, "api_key", "")}
	})

	registerToolIfEnabled(toolReg, "websearch", configMap, func(params map[string]interface{}) interfaces.Tool {
		return &WebSearchTool{}
	})

	registerToolIfEnabled(toolReg, "git", configMap, func(params map[string]interface{}) interfaces.Tool {
		return &GitTool{DefaultWorkDir: getStringParam(params, "default_workdir", "./")}
	})

	registerToolIfEnabled(toolReg, "pyexec", configMap, func(params map[string]interface{}) interfaces.Tool {
		return &PyExecTool{}
	})

	return nil
}

// initDefaultToolConfigs 初始化默认工具配置
func initDefaultToolConfigs(db *storage.BoltDB) []*models.ToolConfig {
	metadata := GetPresetToolsMetadata()
	configs := make([]*models.ToolConfig, 0, len(metadata))

	for _, meta := range metadata {
		config := &models.ToolConfig{
			ID:          meta.ID,
			Name:        meta.Name,
			Type:        models.ToolTypePreset,
			Description: meta.Description,
			Enabled:     true, // 默认启用
			Parameters:  make(map[string]interface{}),
		}

		// 保存到数据库
		if err := db.SaveToolConfig(config); err == nil {
			configs = append(configs, config)
		}
	}

	return configs
}

// registerToolIfEnabled 如果工具启用则注册
func registerToolIfEnabled(
	toolReg *tools.Registry,
	toolID string,
	configMap map[string]*models.ToolConfig,
	createFunc func(params map[string]interface{}) interfaces.Tool,
) {
	config, exists := configMap[toolID]
	if !exists || !config.Enabled {
		return
	}

	tool := createFunc(config.Parameters)
	toolReg.Register(tool)
}

// getStringParam 从参数映射中获取字符串参数
func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if params == nil {
		return defaultValue
	}
	if val, ok := params[key].(string); ok && val != "" {
		return val
	}
	return defaultValue
}
