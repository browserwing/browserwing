package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/browserwing/browserwing/models"
	"github.com/google/uuid"
)

// ScriptClient 脚本客户端
type ScriptClient struct {
	client *Client
}

// Script 脚本数据结构(对外暴露)
type Script struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	URL         string                `json:"url"`
	Actions     []models.ScriptAction `json:"actions"`
	Tags        []string              `json:"tags,omitempty"`
	Group       string                `json:"group,omitempty"`
}

// ScriptExecution 脚本执行结果
type ScriptExecution struct {
	ID          string                    `json:"id"`
	ScriptID    string                    `json:"script_id"`
	ScriptName  string                    `json:"script_name"`
	Status      string                    `json:"status"` // success, failed, running
	StartTime   int64                     `json:"start_time"`
	EndTime     int64                     `json:"end_time"`
	Duration    int64                     `json:"duration"` // 毫秒
	Error       string                    `json:"error,omitempty"`
	Result      map[string]interface{}    `json:"result,omitempty"`
	ExtractedData map[string]string       `json:"extracted_data,omitempty"`
}

// Create 创建脚本
func (sc *ScriptClient) Create(ctx context.Context, script *Script) (string, error) {
	if sc.client.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	if script.Name == "" {
		return "", fmt.Errorf("script name is required")
	}

	// 生成脚本 ID
	if script.ID == "" {
		script.ID = uuid.New().String()
	}

	// 转换为内部模型
	dbScript := &models.Script{
		ID:          script.ID,
		Name:        script.Name,
		Description: script.Description,
		URL:         script.URL,
		Actions:     script.Actions,
		Tags:        script.Tags,
		Group:       script.Group,
	}

	// 保存到数据库
	if err := sc.client.db.SaveScript(dbScript); err != nil {
		return "", fmt.Errorf("failed to save script: %w", err)
	}

	return script.ID, nil
}

// Get 获取脚本
func (sc *ScriptClient) Get(ctx context.Context, scriptID string) (*Script, error) {
	if sc.client.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	dbScript, err := sc.client.db.GetScript(scriptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get script: %w", err)
	}

	if dbScript == nil {
		return nil, fmt.Errorf("script not found: %s", scriptID)
	}

	// 转换为 SDK 模型
	return &Script{
		ID:          dbScript.ID,
		Name:        dbScript.Name,
		Description: dbScript.Description,
		URL:         dbScript.URL,
		Actions:     dbScript.Actions,
		Tags:        dbScript.Tags,
		Group:       dbScript.Group,
	}, nil
}

// List 列出所有脚本
func (sc *ScriptClient) List(ctx context.Context) ([]*Script, error) {
	if sc.client.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	dbScripts, err := sc.client.db.ListScripts()
	if err != nil {
		return nil, fmt.Errorf("failed to list scripts: %w", err)
	}

	// 转换为 SDK 模型
	scripts := make([]*Script, 0, len(dbScripts))
	for _, dbScript := range dbScripts {
		scripts = append(scripts, &Script{
			ID:          dbScript.ID,
			Name:        dbScript.Name,
			Description: dbScript.Description,
			URL:         dbScript.URL,
			Actions:     dbScript.Actions,
			Tags:        dbScript.Tags,
			Group:       dbScript.Group,
		})
	}

	return scripts, nil
}

// Update 更新脚本
func (sc *ScriptClient) Update(ctx context.Context, scriptID string, script *Script) error {
	if sc.client.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// 确保 ID 一致
	script.ID = scriptID

	// 转换为内部模型
	dbScript := &models.Script{
		ID:          script.ID,
		Name:        script.Name,
		Description: script.Description,
		URL:         script.URL,
		Actions:     script.Actions,
		Tags:        script.Tags,
		Group:       script.Group,
	}

	// 更新到数据库
	if err := sc.client.db.UpdateScript(dbScript); err != nil {
		return fmt.Errorf("failed to update script: %w", err)
	}

	return nil
}

// Delete 删除脚本
func (sc *ScriptClient) Delete(ctx context.Context, scriptID string) error {
	if sc.client.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if err := sc.client.db.DeleteScript(scriptID); err != nil {
		return fmt.Errorf("failed to delete script: %w", err)
	}

	return nil
}

// Play 执行脚本
func (sc *ScriptClient) Play(ctx context.Context, scriptID string) (*ScriptExecution, error) {
	if sc.client.browserManager == nil {
		return nil, fmt.Errorf("browser manager not initialized")
	}

	if !sc.client.browserManager.IsRunning() {
		return nil, fmt.Errorf("browser is not running, please start browser first")
	}

	// 获取脚本
	dbScript, err := sc.client.db.GetScript(scriptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get script: %w", err)
	}

	if dbScript == nil {
		return nil, fmt.Errorf("script not found: %s", scriptID)
	}

	// 执行脚本
	result, err := sc.client.browserManager.PlayScript(ctx, dbScript)
	if err != nil {
		return nil, fmt.Errorf("failed to play script: %w", err)
	}

	// 构建执行结果
	execution := &ScriptExecution{
		ID:         uuid.New().String(),
		ScriptID:   scriptID,
		ScriptName: dbScript.Name,
		Status:     "success",
		StartTime:  time.Now().Unix(),
		EndTime:    time.Now().Unix(),
		Duration:   0,
	}

	if result.Success {
		execution.Status = "success"
		// 转换 ExtractedData
		if result.ExtractedData != nil {
			strData := make(map[string]string)
			for k, v := range result.ExtractedData {
				if str, ok := v.(string); ok {
					strData[k] = str
				}
			}
			execution.ExtractedData = strData
		}
	} else {
		execution.Status = "failed"
		execution.Error = result.Message
	}

	// 保存执行记录到数据库
	dbExecution := &models.ScriptExecution{
		ID:            execution.ID,
		ScriptID:      execution.ScriptID,
		ScriptName:    execution.ScriptName,
		StartTime:     time.Unix(execution.StartTime, 0),
		EndTime:       time.Unix(execution.EndTime, 0),
		Duration:      execution.Duration,
		Success:       result.Success,
		Message:       result.Message,
		ErrorMsg:      execution.Error,
		ExtractedData: result.ExtractedData,
		CreatedAt:     time.Now(),
	}

	if err := sc.client.db.SaveScriptExecution(dbExecution); err != nil {
		// 执行成功但保存记录失败,仅记录警告
		fmt.Printf("Warning: Failed to save execution record: %v\n", err)
	}

	return execution, nil
}

// GetExecution 获取执行记录
func (sc *ScriptClient) GetExecution(ctx context.Context, executionID string) (*ScriptExecution, error) {
	if sc.client.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	dbExecution, err := sc.client.db.GetScriptExecution(executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	if dbExecution == nil {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	// 转换 ExtractedData
	strData := make(map[string]string)
	if dbExecution.ExtractedData != nil {
		for k, v := range dbExecution.ExtractedData {
			if str, ok := v.(string); ok {
				strData[k] = str
			}
		}
	}

	status := "success"
	if !dbExecution.Success {
		status = "failed"
	}

	return &ScriptExecution{
		ID:            dbExecution.ID,
		ScriptID:      dbExecution.ScriptID,
		ScriptName:    dbExecution.ScriptName,
		Status:        status,
		StartTime:     dbExecution.StartTime.Unix(),
		EndTime:       dbExecution.EndTime.Unix(),
		Duration:      dbExecution.Duration,
		Error:         dbExecution.ErrorMsg,
		ExtractedData: strData,
	}, nil
}

// ListExecutions 列出脚本的执行记录
func (sc *ScriptClient) ListExecutions(ctx context.Context, scriptID string) ([]*ScriptExecution, error) {
	if sc.client.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	dbExecutions, err := sc.client.db.ListScriptExecutions(scriptID)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	// 过滤指定脚本的执行记录
	executions := make([]*ScriptExecution, 0)
	for _, dbExec := range dbExecutions {
		if scriptID == "" || dbExec.ScriptID == scriptID {
			// 转换 ExtractedData
			strData := make(map[string]string)
			if dbExec.ExtractedData != nil {
				for k, v := range dbExec.ExtractedData {
					if str, ok := v.(string); ok {
						strData[k] = str
					}
				}
			}

			status := "success"
			if !dbExec.Success {
				status = "failed"
			}

			executions = append(executions, &ScriptExecution{
				ID:            dbExec.ID,
				ScriptID:      dbExec.ScriptID,
				ScriptName:    dbExec.ScriptName,
				Status:        status,
				StartTime:     dbExec.StartTime.Unix(),
				EndTime:       dbExec.EndTime.Unix(),
				Duration:      dbExec.Duration,
				Error:         dbExec.ErrorMsg,
				ExtractedData: strData,
			})
		}
	}

	return executions, nil
}
