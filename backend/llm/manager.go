package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/browserwing/browserwing/config"
	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/browserwing/browserwing/storage"
)

// Manager LLM 配置管理器,负责配置的热加载和管理
type Manager struct {
	db         *storage.BoltDB
	extractors map[string]*Extractor // name -> Extractor
	mu         sync.RWMutex
}

// NewManager 创建 LLM 管理器
func NewManager(db *storage.BoltDB) *Manager {
	return &Manager{
		db:         db,
		extractors: make(map[string]*Extractor),
	}
}

// LoadFromConfig 从配置文件加载 LLM 配置
func (m *Manager) LoadFromConfig(cfg *config.Config) error {
	// 如果配置文件中有 LLM 配置,迁移到数据库
	if len(cfg.LLMs) > 0 {
		for _, llmCfg := range cfg.LLMs {
			// 检查数据库中是否已存在
			configs, _ := m.db.ListLLMConfigs()
			exists := false
			for _, dbCfg := range configs {
				if dbCfg.Name == llmCfg.Name {
					exists = true
					break
				}
			}

			if !exists {
				// 迁移到数据库
				model := &models.LLMConfigModel{
					ID:        llmCfg.Name, // 使用 name 作为 ID
					Name:      llmCfg.Name,
					Provider:  llmCfg.Provider,
					APIKey:    llmCfg.APIKey,
					Model:     llmCfg.Model,
					BaseURL:   llmCfg.BaseURL,
					IsDefault: llmCfg.Name == "default",
					IsActive:  true,
				}
				if err := m.db.SaveLLMConfig(model); err != nil {
					return fmt.Errorf("failed to migrate LLM config %s: %w", llmCfg.Name, err)
				}
			}
		}
	}

	err := m.InitSystemPrompts()
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize system prompts: %v", err)
	}

	// 从数据库加载所有配置
	return m.LoadAll()
}

func (m *Manager) InitSystemPrompts() error {
	for _, prompt := range models.SystemPrompts {
		// 检查是否存在，不存在则新增
		existing, err := m.db.GetPrompt(prompt.ID)
		if err != nil || existing == nil {
			if err := m.db.SavePrompt(prompt); err != nil {
				return fmt.Errorf("failed to save system prompt %s: %w", prompt.ID, err)
			}
		}
	}
	return nil
}

// LoadAll 从数据库加载所有启用的 LLM 配置
func (m *Manager) LoadAll() error {
	configs, err := m.db.ListLLMConfigs()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空现有的 extractors
	m.extractors = make(map[string]*Extractor)

	// 加载启用的配置
	for _, cfg := range configs {
		if cfg.IsActive {
			llmCfg := &config.LLMConfig{
				Name:     cfg.Name,
				Provider: cfg.Provider,
				APIKey:   cfg.APIKey,
				Model:    cfg.Model,
				BaseURL:  cfg.BaseURL,
			}

			extractor, err := NewExtractor(llmCfg, m.db)
			if err != nil {
				// 记录错误但继续加载其他配置
				logger.Error(context.Background(), "Failed to load LLM config %s: %v", cfg.Name, err)
				continue
			}

			m.extractors[cfg.Name] = extractor
		}
	}

	return nil
}

// Get 获取指定名称的 Extractor
func (m *Manager) Get(name string) (*Extractor, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	extractor, ok := m.extractors[name]
	return extractor, ok
}

// GetDefault 获取默认的 Extractor
func (m *Manager) GetDefault() (*Extractor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 先尝试从内存中找默认配置
	for name, extractor := range m.extractors {
		cfg, err := m.db.GetLLMConfig(name)
		if err == nil && cfg.IsDefault {
			return extractor, nil
		}
	}

	// 如果没有默认配置,返回第一个可用的
	for _, extractor := range m.extractors {
		return extractor, nil
	}

	return nil, fmt.Errorf("no available LLM config")
}

// List 列出所有已加载的 LLM 名称
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.extractors))
	for name := range m.extractors {
		names = append(names, name)
	}
	return names
}

// Add 添加新的 LLM 配置
func (m *Manager) Add(cfg *models.LLMConfigModel) error {
	// 保存到数据库
	if err := m.db.SaveLLMConfig(cfg); err != nil {
		return err
	}

	// 如果设置为默认,清除其他配置的默认状态
	if cfg.IsDefault {
		if err := m.db.ClearDefaultLLMConfig(); err != nil {
			return err
		}
		cfg.IsDefault = true
		if err := m.db.UpdateLLMConfig(cfg); err != nil {
			return err
		}
	}

	// 如果启用,加载到内存
	if cfg.IsActive {
		return m.loadConfig(cfg)
	}

	return nil
}

// Update 更新 LLM 配置
func (m *Manager) Update(cfg *models.LLMConfigModel) error {
	// 如果设置为默认,清除其他配置的默认状态
	if cfg.IsDefault {
		if err := m.db.ClearDefaultLLMConfig(); err != nil {
			return err
		}
		cfg.IsDefault = true
	}

	// 更新数据库
	if err := m.db.UpdateLLMConfig(cfg); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 删除旧的 extractor
	delete(m.extractors, cfg.Name)

	// 如果启用,重新加载
	if cfg.IsActive {
		llmCfg := &config.LLMConfig{
			Name:     cfg.Name,
			Provider: cfg.Provider,
			APIKey:   cfg.APIKey,
			Model:    cfg.Model,
			BaseURL:  cfg.BaseURL,
		}

		extractor, err := NewExtractor(llmCfg, m.db)
		if err != nil {
			return fmt.Errorf("failed to load LLM config %s: %w", cfg.Name, err)
		}

		m.extractors[cfg.Name] = extractor
	}

	return nil
}

// Delete 删除 LLM 配置
func (m *Manager) Delete(id string) error {
	// 从数据库删除
	if err := m.db.DeleteLLMConfig(id); err != nil {
		return err
	}

	// 从内存中删除
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.extractors, id)

	return nil
}

// loadConfig 加载单个配置到内存
func (m *Manager) loadConfig(cfg *models.LLMConfigModel) error {
	llmCfg := &config.LLMConfig{
		Name:     cfg.Name,
		Provider: cfg.Provider,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
		BaseURL:  cfg.BaseURL,
	}

	extractor, err := NewExtractor(llmCfg, m.db)
	if err != nil {
		return fmt.Errorf("failed to load LLM config %s: %w", cfg.Name, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.extractors[cfg.Name] = extractor

	return nil
}
