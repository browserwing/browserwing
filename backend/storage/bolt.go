package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/browserwing/browserwing/models"
	bolt "go.etcd.io/bbolt"
)

var (
	articlesBucket          = []byte("articles")
	promptsBucket           = []byte("prompts")
	cookiesBucket           = []byte("cookies")
	scriptsBucket           = []byte("scripts")
	llmConfigsBucket        = []byte("llm_configs")
	browserConfigsBucket    = []byte("browser_configs")
	browserInstancesBucket  = []byte("browser_instances")
	scriptExecutionsBucket  = []byte("script_executions")
	recordingConfigsBucket  = []byte("recording_configs")
	agentSessionsBucket     = []byte("agent_sessions")
	agentMessagesBucket     = []byte("agent_messages")
	toolConfigsBucket       = []byte("tool_configs")
	mcpServicesBucket       = []byte("mcp_services")
	usersBucket             = []byte("users")
	apiKeysBucket           = []byte("api_keys")
	scheduledTasksBucket    = []byte("scheduled_tasks")
	taskExecutionsBucket    = []byte("task_executions")
)

type BoltDB struct {
	db *bolt.DB
}

func NewBoltDB(dbPath string) (*BoltDB, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)

	db, err := bolt.Open(dbPath, 0o600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w (directory: %s)", dbPath, err, dir)
	}

	// 创建必要的bucket
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(cookiesBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(scriptsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(promptsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(llmConfigsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(browserConfigsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(browserInstancesBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(scriptExecutionsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(recordingConfigsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(agentSessionsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(agentMessagesBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(toolConfigsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(mcpServicesBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(usersBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(apiKeysBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(scheduledTasksBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(taskExecutionsBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	storage := &BoltDB{db: db}

	return storage, nil
}

func (b *BoltDB) Close() error {
	return b.db.Close()
}

// SaveCookies 保存Cookie
func (b *BoltDB) SaveCookies(cookieStore *models.CookieStore) error {
	cookieStore.UpdatedAt = time.Now()
	if cookieStore.CreatedAt.IsZero() {
		cookieStore.CreatedAt = time.Now()
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(cookiesBucket)
		data, err := cookieStore.ToJSON()
		if err != nil {
			return err
		}
		return bucket.Put([]byte(cookieStore.ID), data)
	})
}

// GetCookies 获取Cookie
func (b *BoltDB) GetCookies(id string) (*models.CookieStore, error) {
	var cookieStore models.CookieStore
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(cookiesBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("cookies not found")
		}
		return cookieStore.FromJSON(data)
	})
	if err != nil {
		return nil, err
	}
	return &cookieStore, nil
}

// DeleteCookies 删除Cookie
func (b *BoltDB) DeleteCookies(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(cookiesBucket)
		return bucket.Delete([]byte(id))
	})
}

// SaveScript 保存脚本
func (b *BoltDB) SaveScript(script *models.Script) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptsBucket)
		data, err := json.Marshal(script)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(script.ID), data)
	})
}

// GetScript 获取脚本
func (b *BoltDB) GetScript(id string) (*models.Script, error) {
	var script models.Script
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptsBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("Script not found")
		}
		return json.Unmarshal(data, &script)
	})
	if err != nil {
		return nil, err
	}
	return &script, nil
}

// ListScripts 列出所有脚本
func (b *BoltDB) ListScripts() ([]*models.Script, error) {
	var scripts []*models.Script
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var script models.Script
			if err := json.Unmarshal(v, &script); err != nil {
				return err
			}
			scripts = append(scripts, &script)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按创建时间倒序排序
	for i := 0; i < len(scripts)-1; i++ {
		for j := i + 1; j < len(scripts); j++ {
			if scripts[i].CreatedAt.Before(scripts[j].CreatedAt) {
				scripts[i], scripts[j] = scripts[j], scripts[i]
			}
		}
	}

	return scripts, nil
}

// UpdateScript 更新脚本
func (b *BoltDB) UpdateScript(script *models.Script) error {
	script.UpdatedAt = time.Now()
	return b.SaveScript(script)
}

// DeleteScript 删除脚本
func (b *BoltDB) DeleteScript(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptsBucket)
		return bucket.Delete([]byte(id))
	})
}

// ============= LLM 配置相关方法 =============

// SaveLLMConfig 保存 LLM 配置
func (b *BoltDB) SaveLLMConfig(config *models.LLMConfigModel) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(llmConfigsBucket)
		data, err := config.ToJSON()
		if err != nil {
			return err
		}
		return bucket.Put([]byte(config.ID), data)
	})
}

// GetLLMConfig 获取 LLM 配置
func (b *BoltDB) GetLLMConfig(id string) (*models.LLMConfigModel, error) {
	var config models.LLMConfigModel
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(llmConfigsBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("LLM config not found")
		}
		return config.FromJSON(data)
	})
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ListLLMConfigs 列出所有 LLM 配置
func (b *BoltDB) ListLLMConfigs() ([]*models.LLMConfigModel, error) {
	var configs []*models.LLMConfigModel
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(llmConfigsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var config models.LLMConfigModel
			if err := config.FromJSON(v); err != nil {
				return err
			}
			configs = append(configs, &config)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按创建时间倒序排序
	for i := range len(configs) - 1 {
		for j := i + 1; j < len(configs); j++ {
			if configs[i].CreatedAt.Before(configs[j].CreatedAt) {
				configs[i], configs[j] = configs[j], configs[i]
			}
		}
	}

	return configs, nil
}

// UpdateLLMConfig 更新 LLM 配置
func (b *BoltDB) UpdateLLMConfig(config *models.LLMConfigModel) error {
	config.UpdatedAt = time.Now()
	return b.SaveLLMConfig(config)
}

// DeleteLLMConfig 删除 LLM 配置
func (b *BoltDB) DeleteLLMConfig(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(llmConfigsBucket)
		return bucket.Delete([]byte(id))
	})
}

// GetDefaultLLMConfig 获取默认 LLM 配置
func (b *BoltDB) GetDefaultLLMConfig() (*models.LLMConfigModel, error) {
	var defaultConfig *models.LLMConfigModel
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(llmConfigsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var config models.LLMConfigModel
			if err := config.FromJSON(v); err != nil {
				return err
			}
			if config.IsDefault && config.IsActive {
				defaultConfig = &config
				return nil
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	if defaultConfig == nil {
		return nil, fmt.Errorf("Default LLM config not found")
	}
	return defaultConfig, nil
}

// ClearDefaultLLMConfig 清除所有 LLM 配置的默认状态
func (b *BoltDB) ClearDefaultLLMConfig() error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(llmConfigsBucket)
		// 先收集所有配置
		var configs []*models.LLMConfigModel
		err := bucket.ForEach(func(k, v []byte) error {
			var config models.LLMConfigModel
			if err := config.FromJSON(v); err != nil {
				return err
			}
			if config.IsDefault {
				config.IsDefault = false
				configs = append(configs, &config)
			}
			return nil
		})
		if err != nil {
			return err
		}
		// 更新配置
		for _, config := range configs {
			data, err := config.ToJSON()
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(config.ID), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// ============= 浏览器配置管理 =============

// SaveBrowserConfig 保存浏览器配置
func (db *BoltDB) SaveBrowserConfig(config *models.BrowserConfig) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(browserConfigsBucket)
		if err != nil {
			return err
		}

		// 如果设置为默认配置,先取消其他配置的默认状态
		if config.IsDefault {
			cursor := b.Cursor()
			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				var existingConfig models.BrowserConfig
				if err := json.Unmarshal(v, &existingConfig); err != nil {
					continue
				}
				if existingConfig.ID != config.ID && existingConfig.IsDefault {
					existingConfig.IsDefault = false
					data, _ := json.Marshal(existingConfig)
					b.Put([]byte(existingConfig.ID), data)
				}
			}
		}

		config.UpdatedAt = time.Now()
		if config.CreatedAt.IsZero() {
			config.CreatedAt = time.Now()
		}

		data, err := json.Marshal(config)
		if err != nil {
			return err
		}

		return b.Put([]byte(config.ID), data)
	})
}

// GetBrowserConfig 获取浏览器配置
func (db *BoltDB) GetBrowserConfig(id string) (*models.BrowserConfig, error) {
	var config models.BrowserConfig
	err := db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(browserConfigsBucket)
		if b == nil {
			return fmt.Errorf("browser config bucket not found")
		}

		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("browser config not found")
		}

		return json.Unmarshal(data, &config)
	})

	return &config, err
}

// GetDefaultBrowserConfig 获取默认浏览器配置
func (db *BoltDB) GetDefaultBrowserConfig() (*models.BrowserConfig, error) {
	var config *models.BrowserConfig
	err := db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(browserConfigsBucket)
		if b == nil {
			return fmt.Errorf("browser config bucket not found")
		}

		cursor := b.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var c models.BrowserConfig
			if err := json.Unmarshal(v, &c); err != nil {
				continue
			}
			if c.IsDefault {
				config = &c
				return nil
			}
		}
		return fmt.Errorf("Default browser config not found")
	})

	return config, err
}

// ListBrowserConfigs 列出所有浏览器配置
func (db *BoltDB) ListBrowserConfigs() ([]models.BrowserConfig, error) {
	var configs []models.BrowserConfig
	err := db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(browserConfigsBucket)
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var config models.BrowserConfig
			if err := json.Unmarshal(v, &config); err != nil {
				return err
			}
			configs = append(configs, config)
			return nil
		})
	})

	return configs, err
}

// DeleteBrowserConfig 删除浏览器配置
func (db *BoltDB) DeleteBrowserConfig(id string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(browserConfigsBucket)
		if b == nil {
			return fmt.Errorf("browser config bucket not found")
		}

		// 检查是否是默认配置
		data := b.Get([]byte(id))
		if data != nil {
			var config models.BrowserConfig
			if err := json.Unmarshal(data, &config); err == nil && config.IsDefault {
				return fmt.Errorf("Cannot delete default browser config")
			}
		}

		return b.Delete([]byte(id))
	})
}

// SavePrompt 保存提示词
func (b *BoltDB) SavePrompt(prompt *models.Prompt) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(promptsBucket)
		data, err := json.Marshal(prompt)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(prompt.ID), data)
	})
}

// GetPrompt 获取提示词
func (b *BoltDB) GetPrompt(id string) (*models.Prompt, error) {
	var prompt models.Prompt
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(promptsBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("prompt not found")
		}
		return json.Unmarshal(data, &prompt)
	})
	if err != nil {
		return nil, err
	}
	return &prompt, nil
}

// ListPrompts 列出所有提示词
func (b *BoltDB) ListPrompts() ([]*models.Prompt, error) {
	var prompts []*models.Prompt
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(promptsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var prompt models.Prompt
			if err := json.Unmarshal(v, &prompt); err != nil {
				return err
			}
			prompts = append(prompts, &prompt)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按创建时间倒序排序
	for i := 0; i < len(prompts)-1; i++ {
		for j := i + 1; j < len(prompts); j++ {
			if prompts[i].CreatedAt.Before(prompts[j].CreatedAt) {
				prompts[i], prompts[j] = prompts[j], prompts[i]
			}
		}
	}

	return prompts, nil
}

// UpdatePrompt 更新提示词
func (b *BoltDB) UpdatePrompt(prompt *models.Prompt) error {
	prompt.UpdatedAt = time.Now()
	return b.SavePrompt(prompt)
}

// DeletePrompt 删除提示词
func (b *BoltDB) DeletePrompt(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(promptsBucket)
		return bucket.Delete([]byte(id))
	})
}

// CheckAndUpdateSystemPrompts 检查并更新系统提示词
// 只更新用户未手动修改过且版本落后的系统prompt
func (b *BoltDB) CheckAndUpdateSystemPrompts() error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(promptsBucket)
		
		// 遍历所有系统prompt
		for _, systemPrompt := range models.SystemPrompts {
			// 获取数据库中的prompt
			data := bucket.Get([]byte(systemPrompt.ID))
			
			if data == nil {
				// 数据库中不存在，直接保存新的
				promptData, err := json.Marshal(systemPrompt)
				if err != nil {
					return err
				}
				if err := bucket.Put([]byte(systemPrompt.ID), promptData); err != nil {
					return err
				}
				continue
			}
			
			// 解析数据库中的prompt
			var dbPrompt models.Prompt
			if err := json.Unmarshal(data, &dbPrompt); err != nil {
				return err
			}
			
			// 检查是否需要更新
			if dbPrompt.NeedsUpdate(systemPrompt) {
				// 保留原始的CreatedAt，更新其他字段
				systemPrompt.CreatedAt = dbPrompt.CreatedAt
				systemPrompt.UpdatedAt = time.Now()
				
				promptData, err := json.Marshal(systemPrompt)
				if err != nil {
					return err
				}
				if err := bucket.Put([]byte(systemPrompt.ID), promptData); err != nil {
					return err
				}
			}
		}
		
		return nil
	})
}

// ============= 脚本执行记录相关方法 =============

// SaveScriptExecution 保存脚本执行记录
func (b *BoltDB) SaveScriptExecution(execution *models.ScriptExecution) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptExecutionsBucket)
		data, err := json.Marshal(execution)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(execution.ID), data)
	})
}

// GetScriptExecution 获取单个脚本执行记录
func (b *BoltDB) GetScriptExecution(id string) (*models.ScriptExecution, error) {
	var execution models.ScriptExecution
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptExecutionsBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("script execution not found")
		}
		return json.Unmarshal(data, &execution)
	})
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func (b *BoltDB) GetLatestScriptExecutionByScriptID(scriptID string) (*models.ScriptExecution, error) {
	var execution models.ScriptExecution
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptExecutionsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var execution models.ScriptExecution
			if err := json.Unmarshal(v, &execution); err != nil {
				return err
			}
			if execution.ScriptID == scriptID {
				return json.Unmarshal(v, &execution)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// ListScriptExecutions 列出所有脚本执行记录（支持按脚本ID过滤）
func (b *BoltDB) ListScriptExecutions(scriptID string) ([]*models.ScriptExecution, error) {
	var executions []*models.ScriptExecution
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptExecutionsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var execution models.ScriptExecution
			if err := json.Unmarshal(v, &execution); err != nil {
				return err
			}
			// 如果指定了 scriptID，只返回该脚本的执行记录
			if scriptID == "" || execution.ScriptID == scriptID {
				executions = append(executions, &execution)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按创建时间倒序排序（最新的在前）
	for i := 0; i < len(executions)-1; i++ {
		for j := i + 1; j < len(executions); j++ {
			if executions[i].StartTime.Before(executions[j].StartTime) {
				executions[i], executions[j] = executions[j], executions[i]
			}
		}
	}

	return executions, nil
}

// DeleteScriptExecution 删除脚本执行记录
func (b *BoltDB) DeleteScriptExecution(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptExecutionsBucket)
		return bucket.Delete([]byte(id))
	})
}

// DeleteScriptExecutionsByScriptID 删除指定脚本的所有执行记录
func (b *BoltDB) DeleteScriptExecutionsByScriptID(scriptID string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scriptExecutionsBucket)
		// 先收集要删除的 key
		var keysToDelete [][]byte
		err := bucket.ForEach(func(k, v []byte) error {
			var execution models.ScriptExecution
			if err := json.Unmarshal(v, &execution); err != nil {
				return err
			}
			if execution.ScriptID == scriptID {
				keysToDelete = append(keysToDelete, append([]byte(nil), k...))
			}
			return nil
		})
		if err != nil {
			return err
		}
		// 删除收集的 key
		for _, key := range keysToDelete {
			if err := bucket.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// ============= 录制配置相关方法 =============

// SaveRecordingConfig 保存录制配置
func (b *BoltDB) SaveRecordingConfig(config *models.RecordingConfig) error {
	config.UpdatedAt = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(recordingConfigsBucket)
		data, err := json.Marshal(config)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(config.ID), data)
	})
}

// GetRecordingConfig 获取录制配置
func (b *BoltDB) GetRecordingConfig(id string) (*models.RecordingConfig, error) {
	var config models.RecordingConfig
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(recordingConfigsBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("recording config not found")
		}
		return json.Unmarshal(data, &config)
	})
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetDefaultRecordingConfig 获取默认录制配置，如果不存在则返回系统默认值
func (b *BoltDB) GetDefaultRecordingConfig() *models.RecordingConfig {
	config, err := b.GetRecordingConfig("default")
	if err != nil {
		// 返回系统默认配置
		defaultConfig := models.GetDefaultRecordingConfig()
		// 尝试保存到数据库
		b.SaveRecordingConfig(defaultConfig)
		return defaultConfig
	}
	return config
}

// ==================== Agent Session 相关方法 ====================

// SaveAgentSession 保存 Agent 会话
func (b *BoltDB) SaveAgentSession(session *models.AgentSession) error {
	session.UpdatedAt = time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(agentSessionsBucket)
		data, err := json.Marshal(session)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(session.ID), data)
	})
}

// GetAgentSession 获取 Agent 会话
func (b *BoltDB) GetAgentSession(id string) (*models.AgentSession, error) {
	var session models.AgentSession
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(agentSessionsBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("agent session not found")
		}
		return json.Unmarshal(data, &session)
	})
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListAgentSessions 列出所有 Agent 会话
func (b *BoltDB) ListAgentSessions() ([]*models.AgentSession, error) {
	var sessions []*models.AgentSession
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(agentSessionsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var session models.AgentSession
			if err := json.Unmarshal(v, &session); err != nil {
				return err
			}
			sessions = append(sessions, &session)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按更新时间倒序排序（最新的在前面）
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// DeleteAgentSession 删除 Agent 会话
func (b *BoltDB) DeleteAgentSession(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(agentSessionsBucket)
		if err := bucket.Delete([]byte(id)); err != nil {
			return err
		}

		// 同时删除该会话的所有消息
		msgBucket := tx.Bucket(agentMessagesBucket)
		return msgBucket.ForEach(func(k, v []byte) error {
			var msg models.AgentMessage
			if err := json.Unmarshal(v, &msg); err != nil {
				return nil // 跳过无效数据
			}
			if msg.SessionID == id {
				return msgBucket.Delete(k)
			}
			return nil
		})
	})
}

// SaveAgentMessage 保存 Agent 消息
func (b *BoltDB) SaveAgentMessage(message *models.AgentMessage) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(agentMessagesBucket)
		data, err := json.Marshal(message)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(message.ID), data)
	})
}

// GetAgentMessage 获取 Agent 消息
func (b *BoltDB) GetAgentMessage(id string) (*models.AgentMessage, error) {
	var message models.AgentMessage
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(agentMessagesBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("agent message not found")
		}
		return json.Unmarshal(data, &message)
	})
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// ListAgentMessages 列出指定会话的所有消息
func (b *BoltDB) ListAgentMessages(sessionID string) ([]*models.AgentMessage, error) {
	var messages []*models.AgentMessage
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(agentMessagesBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var message models.AgentMessage
			if err := json.Unmarshal(v, &message); err != nil {
				return nil // 跳过无效数据
			}
			if message.SessionID == sessionID {
				messages = append(messages, &message)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按时间戳正序排序（最早的在前面）
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	return messages, nil
}

// ============= 工具配置相关方法 =============

// SaveToolConfig 保存工具配置
func (b *BoltDB) SaveToolConfig(config *models.ToolConfig) error {
	config.UpdatedAt = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(toolConfigsBucket)
		data, err := json.Marshal(config)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(config.ID), data)
	})
}

// GetToolConfig 获取工具配置
func (b *BoltDB) GetToolConfig(id string) (*models.ToolConfig, error) {
	var config models.ToolConfig
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(toolConfigsBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("tool config not found: %s", id)
		}
		return json.Unmarshal(data, &config)
	})
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ListToolConfigs 列出所有工具配置
func (b *BoltDB) ListToolConfigs() ([]*models.ToolConfig, error) {
	var configs []*models.ToolConfig
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(toolConfigsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var config models.ToolConfig
			if err := json.Unmarshal(v, &config); err != nil {
				return nil // 跳过无效数据
			}
			configs = append(configs, &config)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按名称排序
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})

	return configs, nil
}

// DeleteToolConfig 删除工具配置
func (b *BoltDB) DeleteToolConfig(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(toolConfigsBucket)
		return bucket.Delete([]byte(id))
	})
}

// DeleteToolConfigByScriptID 删除关联指定脚本ID的所有工具配置
func (b *BoltDB) DeleteToolConfigByScriptID(scriptID string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(toolConfigsBucket)
		// 先收集要删除的 key
		var keysToDelete [][]byte
		err := bucket.ForEach(func(k, v []byte) error {
			var config models.ToolConfig
			if err := json.Unmarshal(v, &config); err != nil {
				return err
			}
			if config.ScriptID == scriptID {
				keysToDelete = append(keysToDelete, append([]byte(nil), k...))
			}
			return nil
		})
		if err != nil {
			return err
		}
		// 删除收集的 key
		for _, key := range keysToDelete {
			if err := bucket.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// ============= MCP服务相关方法 =============

// SaveMCPService 保存MCP服务配置
func (b *BoltDB) SaveMCPService(service *models.MCPService) error {
	service.UpdatedAt = time.Now()
	if service.CreatedAt.IsZero() {
		service.CreatedAt = time.Now()
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(mcpServicesBucket)
		data, err := json.Marshal(service)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(service.ID), data)
	})
}

// GetMCPService 获取MCP服务配置
func (b *BoltDB) GetMCPService(id string) (*models.MCPService, error) {
	var service models.MCPService
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(mcpServicesBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("mcp service not found: %s", id)
		}
		return json.Unmarshal(data, &service)
	})
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// ListMCPServices 列出所有MCP服务配置
func (b *BoltDB) ListMCPServices() ([]*models.MCPService, error) {
	var services []*models.MCPService
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(mcpServicesBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var service models.MCPService
			if err := json.Unmarshal(v, &service); err != nil {
				return nil // 跳过无效数据
			}
			services = append(services, &service)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按名称排序
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}

// DeleteMCPService 删除MCP服务配置
func (b *BoltDB) DeleteMCPService(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(mcpServicesBucket)
		return bucket.Delete([]byte(id))
	})
}

// SaveMCPServiceTools 保存MCP服务发现的工具列表
func (b *BoltDB) SaveMCPServiceTools(serviceID string, tools []models.MCPDiscoveredTool) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(mcpServicesBucket)

		// 先获取现有服务
		data := bucket.Get([]byte(serviceID))
		if data == nil {
			return fmt.Errorf("mcp service not found: %s", serviceID)
		}

		var service models.MCPService
		if err := json.Unmarshal(data, &service); err != nil {
			return err
		}

		// 更新工具数量和状态
		service.ToolCount = len(tools)
		service.UpdatedAt = time.Now()

		// 保存服务
		newData, err := json.Marshal(&service)
		if err != nil {
			return err
		}

		// 保存工具列表到单独的key
		toolsKey := []byte(serviceID + "_tools")
		toolsData, err := json.Marshal(tools)
		if err != nil {
			return err
		}

		if err := bucket.Put(toolsKey, toolsData); err != nil {
			return err
		}

		return bucket.Put([]byte(serviceID), newData)
	})
}

// GetMCPServiceTools 获取MCP服务的工具列表
func (b *BoltDB) GetMCPServiceTools(serviceID string) ([]models.MCPDiscoveredTool, error) {
	var tools []models.MCPDiscoveredTool
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(mcpServicesBucket)
		toolsKey := []byte(serviceID + "_tools")
		data := bucket.Get(toolsKey)
		if data == nil {
			// 如果没有找到工具数据,返回空列表而不是错误
			return nil
		}
		return json.Unmarshal(data, &tools)
	})
	if err != nil {
		return nil, err
	}
	return tools, nil
}

// ============= 用户管理 =============

// CreateUser 创建用户
func (b *BoltDB) CreateUser(user *models.User) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(user.ID), data)
	})
}

// GetUser 获取用户
func (b *BoltDB) GetUser(id string) (*models.User, error) {
	var user models.User
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("user not found")
		}
		return json.Unmarshal(data, &user)
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (b *BoltDB) GetUserByUsername(username string) (*models.User, error) {
	var user *models.User
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var u models.User
			if err := json.Unmarshal(v, &u); err != nil {
				continue
			}
			if u.Username == username {
				user = &u
				return nil
			}
		}
		return fmt.Errorf("user not found")
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ListUsers 列出所有用户
func (b *BoltDB) ListUsers() ([]*models.User, error) {
	var users []*models.User
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var user models.User
			if err := json.Unmarshal(v, &user); err != nil {
				return err
			}
			users = append(users, &user)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按创建时间排序
	sort.Slice(users, func(i, j int) bool {
		return users[i].CreatedAt.After(users[j].CreatedAt)
	})

	return users, nil
}

// UpdateUser 更新用户
func (b *BoltDB) UpdateUser(user *models.User) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(user.ID), data)
	})
}

// DeleteUser 删除用户
func (b *BoltDB) DeleteUser(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(usersBucket)
		return bucket.Delete([]byte(id))
	})
}

// ============= ApiKey 管理 =============

// CreateApiKey 创建API密钥
func (b *BoltDB) CreateApiKey(apiKey *models.ApiKey) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		data, err := json.Marshal(apiKey)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(apiKey.ID), data)
	})
}

// GetApiKey 获取API密钥
func (b *BoltDB) GetApiKey(id string) (*models.ApiKey, error) {
	var apiKey models.ApiKey
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("api key not found")
		}
		return json.Unmarshal(data, &apiKey)
	})
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// GetApiKeyByKey 根据密钥获取API密钥
func (b *BoltDB) GetApiKeyByKey(key string) (*models.ApiKey, error) {
	var apiKey *models.ApiKey
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var ak models.ApiKey
			if err := json.Unmarshal(v, &ak); err != nil {
				continue
			}
			if ak.Key == key {
				apiKey = &ak
				return nil
			}
		}
		return fmt.Errorf("api key not found")
	})
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

// ListApiKeys 列出所有API密钥
func (b *BoltDB) ListApiKeys() ([]*models.ApiKey, error) {
	var apiKeys []*models.ApiKey
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var apiKey models.ApiKey
			if err := json.Unmarshal(v, &apiKey); err != nil {
				return err
			}
			apiKeys = append(apiKeys, &apiKey)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按创建时间排序
	sort.Slice(apiKeys, func(i, j int) bool {
		return apiKeys[i].CreatedAt.After(apiKeys[j].CreatedAt)
	})

	return apiKeys, nil
}

// ListApiKeysByUser 列出用户的所有API密钥
func (b *BoltDB) ListApiKeysByUser(userID string) ([]*models.ApiKey, error) {
	var apiKeys []*models.ApiKey
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var apiKey models.ApiKey
			if err := json.Unmarshal(v, &apiKey); err != nil {
				return err
			}
			if apiKey.UserID == userID {
				apiKeys = append(apiKeys, &apiKey)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// 按创建时间排序
	sort.Slice(apiKeys, func(i, j int) bool {
		return apiKeys[i].CreatedAt.After(apiKeys[j].CreatedAt)
	})

	return apiKeys, nil
}

// UpdateApiKey 更新API密钥
func (b *BoltDB) UpdateApiKey(apiKey *models.ApiKey) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		data, err := json.Marshal(apiKey)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(apiKey.ID), data)
	})
}

// DeleteApiKey 删除API密钥
func (b *BoltDB) DeleteApiKey(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiKeysBucket)
		return bucket.Delete([]byte(id))
	})
}

// ==================== 浏览器实例管理 ====================

// SaveBrowserInstance 保存浏览器实例
func (b *BoltDB) SaveBrowserInstance(instance *models.BrowserInstance) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(browserInstancesBucket)
		
		// 如果设置为默认实例，需要先取消其他实例的默认状态
		if instance.IsDefault {
			cursor := bucket.Cursor()
			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				var existingInstance models.BrowserInstance
				if err := json.Unmarshal(v, &existingInstance); err != nil {
					continue
				}
				if existingInstance.ID != instance.ID && existingInstance.IsDefault {
					existingInstance.IsDefault = false
					existingInstance.UpdatedAt = time.Now()
					data, _ := json.Marshal(existingInstance)
					bucket.Put([]byte(existingInstance.ID), data)
				}
			}
		}
		
		data, err := json.Marshal(instance)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(instance.ID), data)
	})
}

// GetBrowserInstance 获取浏览器实例
func (b *BoltDB) GetBrowserInstance(id string) (*models.BrowserInstance, error) {
	var instance models.BrowserInstance
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(browserInstancesBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("browser instance not found")
		}
		return json.Unmarshal(data, &instance)
	})
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// ListBrowserInstances 列出所有浏览器实例
func (b *BoltDB) ListBrowserInstances() ([]models.BrowserInstance, error) {
	var instances []models.BrowserInstance
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(browserInstancesBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var instance models.BrowserInstance
			if err := json.Unmarshal(v, &instance); err != nil {
				return err
			}
			instances = append(instances, instance)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	
	// 按创建时间排序（默认实例排在前面）
	sort.Slice(instances, func(i, j int) bool {
		if instances[i].IsDefault != instances[j].IsDefault {
			return instances[i].IsDefault
		}
		return instances[i].CreatedAt.Before(instances[j].CreatedAt)
	})
	
	return instances, nil
}

// GetDefaultBrowserInstance 获取默认浏览器实例
func (b *BoltDB) GetDefaultBrowserInstance() (*models.BrowserInstance, error) {
	var instance *models.BrowserInstance
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(browserInstancesBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var inst models.BrowserInstance
			if err := json.Unmarshal(v, &inst); err != nil {
				return err
			}
			if inst.IsDefault {
				instance = &inst
				return nil
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("default browser instance not found")
	}
	return instance, nil
}

// UpdateBrowserInstance 更新浏览器实例
func (b *BoltDB) UpdateBrowserInstance(id string, instance *models.BrowserInstance) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(browserInstancesBucket)
		
		// 检查实例是否存在
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("browser instance not found")
		}
		
		// 如果设置为默认实例，需要先取消其他实例的默认状态
		if instance.IsDefault {
			cursor := bucket.Cursor()
			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				var existingInstance models.BrowserInstance
				if err := json.Unmarshal(v, &existingInstance); err != nil {
					continue
				}
				if existingInstance.ID != id && existingInstance.IsDefault {
					existingInstance.IsDefault = false
					existingInstance.UpdatedAt = time.Now()
					updatedData, _ := json.Marshal(existingInstance)
					bucket.Put([]byte(existingInstance.ID), updatedData)
				}
			}
		}
		
		instance.UpdatedAt = time.Now()
		newData, err := json.Marshal(instance)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(id), newData)
	})
}

// DeleteBrowserInstance 删除浏览器实例
func (b *BoltDB) DeleteBrowserInstance(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(browserInstancesBucket)
		
		// 检查是否为默认实例
		data := bucket.Get([]byte(id))
		if data != nil {
			var instance models.BrowserInstance
			if err := json.Unmarshal(data, &instance); err == nil {
				if instance.IsDefault {
					return fmt.Errorf("cannot delete default browser instance")
				}
			}
		}
		
		return bucket.Delete([]byte(id))
	})
}

// ================== Scheduled Tasks ==================

// CreateScheduledTask 创建定时任务
func (db *BoltDB) CreateScheduledTask(task *models.ScheduledTask) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scheduledTasksBucket)
		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(task.ID), data)
	})
}

// GetScheduledTask 获取定时任务
func (db *BoltDB) GetScheduledTask(id string) (*models.ScheduledTask, error) {
	var task models.ScheduledTask
	err := db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scheduledTasksBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("scheduled task not found")
		}
		return json.Unmarshal(data, &task)
	})
	return &task, err
}

// UpdateScheduledTask 更新定时任务
func (db *BoltDB) UpdateScheduledTask(task *models.ScheduledTask) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scheduledTasksBucket)
		// 检查任务是否存在
		if bucket.Get([]byte(task.ID)) == nil {
			return fmt.Errorf("scheduled task not found")
		}
		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(task.ID), data)
	})
}

// DeleteScheduledTask 删除定时任务
func (db *BoltDB) DeleteScheduledTask(id string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scheduledTasksBucket)
		return bucket.Delete([]byte(id))
	})
}

// ListScheduledTasks 列出所有定时任务
func (db *BoltDB) ListScheduledTasks() ([]models.ScheduledTask, error) {
	var tasks []models.ScheduledTask
	err := db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(scheduledTasksBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var task models.ScheduledTask
			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}
			tasks = append(tasks, task)
			return nil
		})
	})
	
	// 按创建时间降序排序
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
	
	return tasks, err
}

// ListScheduledTasksWithPagination 分页列出定时任务
func (db *BoltDB) ListScheduledTasksWithPagination(page, pageSize int, searchQuery string) ([]models.ScheduledTask, int, error) {
	allTasks, err := db.ListScheduledTasks()
	if err != nil {
		return nil, 0, err
	}

	// 过滤
	var filteredTasks []models.ScheduledTask
	for _, task := range allTasks {
		if searchQuery == "" {
			filteredTasks = append(filteredTasks, task)
			continue
		}
		// 简单的名称和描述搜索
		if strings.Contains(strings.ToLower(task.Name), strings.ToLower(searchQuery)) || 
		   strings.Contains(strings.ToLower(task.Description), strings.ToLower(searchQuery)) {
			filteredTasks = append(filteredTasks, task)
		}
	}

	total := len(filteredTasks)

	// 分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return []models.ScheduledTask{}, total, nil
	}
	if end > total {
		end = total
	}

	return filteredTasks[start:end], total, nil
}

// ================== Task Executions ==================

// CreateTaskExecution 创建任务执行记录
func (db *BoltDB) CreateTaskExecution(execution *models.TaskExecution) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(taskExecutionsBucket)
		data, err := json.Marshal(execution)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(execution.ID), data)
	})
}

// GetTaskExecution 获取任务执行记录
func (db *BoltDB) GetTaskExecution(id string) (*models.TaskExecution, error) {
	var execution models.TaskExecution
	err := db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(taskExecutionsBucket)
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("task execution not found")
		}
		return json.Unmarshal(data, &execution)
	})
	return &execution, err
}

// DeleteTaskExecution 删除任务执行记录
func (db *BoltDB) DeleteTaskExecution(id string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(taskExecutionsBucket)
		return bucket.Delete([]byte(id))
	})
}

// ListTaskExecutions 列出所有任务执行记录
func (db *BoltDB) ListTaskExecutions() ([]models.TaskExecution, error) {
	var executions []models.TaskExecution
	err := db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(taskExecutionsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var execution models.TaskExecution
			if err := json.Unmarshal(v, &execution); err != nil {
				return err
			}
			executions = append(executions, execution)
			return nil
		})
	})
	
	// 按开始时间降序排序
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].StartTime.After(executions[j].StartTime)
	})
	
	return executions, err
}

// ListTaskExecutionsWithPagination 分页列出任务执行记录
func (db *BoltDB) ListTaskExecutionsWithPagination(page, pageSize int, taskID, searchQuery string, successFilter string) ([]models.TaskExecution, int, error) {
	allExecutions, err := db.ListTaskExecutions()
	if err != nil {
		return nil, 0, err
	}

	// 过滤
	var filteredExecutions []models.TaskExecution
	for _, execution := range allExecutions {
		// 按任务 ID 过滤
		if taskID != "" && execution.TaskID != taskID {
			continue
		}
		
		// 按搜索关键字过滤
		if searchQuery != "" {
			if !strings.Contains(strings.ToLower(execution.TaskName), strings.ToLower(searchQuery)) && 
			   !strings.Contains(strings.ToLower(execution.Message), strings.ToLower(searchQuery)) {
				continue
			}
		}
		
		// 按成功状态过滤
		if successFilter == "success" && !execution.Success {
			continue
		}
		if successFilter == "failed" && execution.Success {
			continue
		}
		
		filteredExecutions = append(filteredExecutions, execution)
	}

	total := len(filteredExecutions)

	// 分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return []models.TaskExecution{}, total, nil
	}
	if end > total {
		end = total
	}

	return filteredExecutions[start:end], total, nil
}

// BatchDeleteTaskExecutions 批量删除任务执行记录
func (db *BoltDB) BatchDeleteTaskExecutions(ids []string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(taskExecutionsBucket)
		for _, id := range ids {
			if err := bucket.Delete([]byte(id)); err != nil {
				return err
			}
		}
		return nil
	})
}
