package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/storage"
)

// toolResultContextKey 用于在 context 中存储工具执行结果
type toolResultContextKey struct{}

// ToolResultStore 存储工具执行结果
type ToolResultStore struct {
	mu      sync.RWMutex
	results map[string]string // toolName -> result
}

// GlobalToolResultStore 全局工具结果存储
var GlobalToolResultStore = &ToolResultStore{
	results: make(map[string]string),
}

// SetResult 设置工具执行结果
func (s *ToolResultStore) SetResult(toolName, result string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[toolName] = result
}

// GetResult 获取并清除工具执行结果
func (s *ToolResultStore) GetResult(toolName string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.results[toolName]
	delete(s.results, toolName) // 获取后清除
	return result
}

// ToolWithSchema 定义带有 InputSchema 方法的工具接口
type ToolWithSchema interface {
	interfaces.Tool
	InputSchema() map[string]interface{}
}

// ToolWrapper 工具包装器，为所有工具添加 instructions 参数
type ToolWrapper struct {
	originalTool interfaces.Tool
}

// InputSchema 重写 InputSchema 方法，添加 instructions 参数
func (w *ToolWrapper) InputSchema() map[string]interface{} {
	logger.Info(context.Background(), "[ToolWrapper.InputSchema] Called for tool: %s", w.Name())

	var originalSchema map[string]interface{}

	// 尝试获取原始 schema
	if toolWithSchema, ok := w.originalTool.(ToolWithSchema); ok {
		originalSchema = toolWithSchema.InputSchema()
		logger.Info(context.Background(), "[ToolWrapper.InputSchema] Got original schema from tool")
	} else {
		// 如果工具没有 InputSchema，创建一个基本的
		originalSchema = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
		logger.Info(context.Background(), "[ToolWrapper.InputSchema] Created basic schema")
	}

	// 复制 schema
	schema := make(map[string]interface{})
	for k, v := range originalSchema {
		schema[k] = v
	}

	// 添加 instructions 参数到 properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		properties = make(map[string]interface{})
		schema["properties"] = properties
	}

	// 添加 instructions 字段
	properties["instructions"] = map[string]interface{}{
		"type":        "string",
		"description": instructionsDescription,
	}

	// 将 instructions 添加到 required 列表
	required, ok := schema["required"].([]interface{})
	if !ok {
		required = []interface{}{}
	}
	required = append(required, "instructions")
	schema["required"] = required

	logger.Info(context.Background(), "[ToolWrapper.InputSchema] Final schema has %d properties, required: %v",
		len(properties), required)

	return schema
}

const instructionsDescription = "Please briefly explain: 1) Why you are calling this tool 2) What information or task you expect to accomplish with this tool. This explanation will be shown to users to help them understand the AI's thinking process. 3) In the explanation, use the specific tool name instead of saying 'this tool'."

// Parameters 重写 Parameters 方法，添加 instructions 参数
func (w *ToolWrapper) Parameters() map[string]interfaces.ParameterSpec {

	// 获取原始参数
	originalParams := w.originalTool.Parameters()

	// 复制参数
	params := make(map[string]interfaces.ParameterSpec)
	for k, v := range originalParams {
		params[k] = v
	}

	// 添加 instructions 参数
	params["instructions"] = interfaces.ParameterSpec{
		Type:        "string",
		Description: instructionsDescription,
		Required:    true,
	}

	return params
}

// Name 返回工具名称
func (w *ToolWrapper) Name() string {
	return w.originalTool.Name()
}

// Description 返回工具描述
func (w *ToolWrapper) Description() string {
	return w.originalTool.Description()
}

// Run 执行工具（从参数中移除 instructions 后调用原始工具）
func (w *ToolWrapper) Run(ctx context.Context, input string) (string, error) {
	toolName := w.Name()

	// 记录执行开始

	result, err := w.originalTool.Run(ctx, input)

	// 记录执行结果
	if err != nil {
		// 保存错误信息
		GlobalToolResultStore.SetResult(toolName, fmt.Sprintf("Error: %v", err))
	} else {
		resultPreview := result
		if len(resultPreview) > 200 {
			resultPreview = result[:200] + "..."
		}
		// 保存结果到全局存储
		GlobalToolResultStore.SetResult(toolName, result)
	}

	return result, err
}

// Execute 执行工具（从参数中移除 instructions 后调用原始工具）
func (w *ToolWrapper) Execute(ctx context.Context, input string) (string, error) {
	toolName := w.Name()

	// 记录执行开始

	result, err := w.originalTool.Execute(ctx, input)

	// 记录执行结果
	if err != nil {
		// 保存错误信息
		GlobalToolResultStore.SetResult(toolName, fmt.Sprintf("Error: %v", err))
	} else {
		resultPreview := result
		if len(resultPreview) > 200 {
			resultPreview = result[:200] + "..."
		}
		// 保存结果到全局存储
		GlobalToolResultStore.SetResult(toolName, result)
	}

	return result, err
}

// WrapTool 包装工具以添加 instructions 参数
func WrapTool(tool interfaces.Tool) interfaces.Tool {
	return &ToolWrapper{
		originalTool: tool,
	}
}

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
		{
			ID:          "webfetch",
			Name:        "Web Fetch",
			Description: "Fetch a web page and convert it to specified format (html or markdown)",
			Parameters:  []models.PresetToolParameterSchema{},
		},
	}
}

// InitPresetTools 初始化所有预设工具
func InitPresetTools(ctx context.Context, toolReg *tools.Registry, db *storage.BoltDB) error {
	if toolReg == nil {
		return fmt.Errorf("tool registry cannot be empty")
	}

	// 定义所有已实现的预设工具
	implementedTools := map[string]func(params map[string]interface{}) interfaces.Tool{
		"fileops": func(params map[string]interface{}) interfaces.Tool {
			return &FileOpsTool{RootDir: getStringParam(params, "root_directory", "./")}
		},
		"bark": func(params map[string]interface{}) interfaces.Tool {
			return &BarkTool{APIKey: getStringParam(params, "api_key", "")}
		},
		"git": func(params map[string]interface{}) interfaces.Tool {
			return &GitTool{DefaultWorkDir: getStringParam(params, "default_workdir", "./")}
		},
		"pyexec": func(params map[string]interface{}) interfaces.Tool {
			return &PyExecTool{}
		},
		"webfetch": func(params map[string]interface{}) interfaces.Tool {
			return &WebFetchTool{}
		},
	}

	// 获取所有工具配置
	toolConfigs, err := db.ListToolConfigs()
	if err != nil {
		// 如果数据库为空，初始化默认配置
		toolConfigs = initDefaultToolConfigs(db, implementedTools)
	} else {
		// 清理未实现的工具配置
		cleanupUnimplementedTools(db, toolConfigs, implementedTools)
		// 重新获取配置
		toolConfigs, _ = db.ListToolConfigs()
	}

	// 构建配置映射
	configMap := make(map[string]*models.ToolConfig)
	for _, cfg := range toolConfigs {
		if cfg.Type == models.ToolTypePreset {
			configMap[cfg.ID] = cfg
		}
	}

	// 注册所有预设工具（根据配置）
	for toolID, createFunc := range implementedTools {
		registerToolIfEnabled(toolReg, toolID, configMap, createFunc)
	}

	return nil
}

// initDefaultToolConfigs 初始化默认工具配置
func initDefaultToolConfigs(db *storage.BoltDB, implementedTools map[string]func(params map[string]interface{}) interfaces.Tool) []*models.ToolConfig {
	metadata := GetPresetToolsMetadata()
	configs := make([]*models.ToolConfig, 0, len(metadata))

	for _, meta := range metadata {
		// 只为已实现的工具创建配置
		if _, implemented := implementedTools[meta.ID]; !implemented {
			continue
		}

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

// cleanupUnimplementedTools 清理未实现的工具配置
func cleanupUnimplementedTools(db *storage.BoltDB, toolConfigs []*models.ToolConfig, implementedTools map[string]func(params map[string]interface{}) interfaces.Tool) {
	for _, cfg := range toolConfigs {
		// 只处理预设工具类型
		if cfg.Type != models.ToolTypePreset {
			continue
		}

		// 如果工具未实现，删除其配置
		if _, implemented := implementedTools[cfg.ID]; !implemented {
			_ = db.DeleteToolConfig(cfg.ID)
		}
	}
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
	// 包装工具以添加 instructions 参数
	wrappedTool := WrapTool(tool)
	toolReg.Register(wrappedTool)
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
