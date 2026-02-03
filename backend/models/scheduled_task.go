package models

import (
	"time"
)

// ScheduleType 定时任务调度类型
type ScheduleType string

const (
	ScheduleTypeAt    ScheduleType = "at"    // 一次性任务（在指定时间执行）
	ScheduleTypeEvery ScheduleType = "every" // 固定间隔重复任务
	ScheduleTypeCron  ScheduleType = "cron"  // 标准 cron 表达式任务
)

// ExecutionType 执行类型
type ExecutionType string

const (
	ExecutionTypeScript ExecutionType = "script" // 执行脚本
	ExecutionTypeAgent  ExecutionType = "agent"  // 调用 agent
)

// ScheduledTask 定时任务
type ScheduledTask struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"` // 是否启用
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 调度配置
	ScheduleType ScheduleType `json:"schedule_type"` // at, every, cron
	// At 类型：ISO 8601 时间字符串（如 "2024-12-31T23:59:59Z"）
	// Every 类型：间隔字符串（如 "5m", "1h", "2h30m"）
	// Cron 类型：标准 cron 表达式（如 "0 */5 * * * *"，支持秒级）
	ScheduleConfig string `json:"schedule_config"`

	// 执行配置
	ExecutionType ExecutionType `json:"execution_type"` // script, agent

	// 脚本执行配置（当 execution_type 为 script 时使用）
	ScriptID         string            `json:"script_id,omitempty"`          // 脚本 ID
	ScriptName       string            `json:"script_name,omitempty"`        // 脚本名称（冗余字段，便于显示）
	ScriptVariables  map[string]string `json:"script_variables,omitempty"`   // 脚本变量
	BrowserInstanceID string           `json:"browser_instance_id,omitempty"` // 浏览器实例 ID（可选）

	// Agent 执行配置（当 execution_type 为 agent 时使用）
	AgentPrompt   string `json:"agent_prompt,omitempty"`    // Agent 提示词
	AgentLLMID    string `json:"agent_llm_id,omitempty"`    // 使用的 LLM 配置 ID
	AgentLLMName  string `json:"agent_llm_name,omitempty"`  // LLM 配置名称（冗余字段）
	AgentSessionID string `json:"agent_session_id,omitempty"` // 关联的会话 ID（如果需要上下文）

	// 执行状态
	LastExecutionTime *time.Time `json:"last_execution_time,omitempty"` // 上次执行时间
	NextExecutionTime *time.Time `json:"next_execution_time,omitempty"` // 下次执行时间
	LastExecutionStatus string   `json:"last_execution_status,omitempty"` // 上次执行状态：success, failed
	ExecutionCount    int        `json:"execution_count"`                 // 总执行次数
	SuccessCount      int        `json:"success_count"`                   // 成功次数
	FailedCount       int        `json:"failed_count"`                    // 失败次数
}

// TaskExecution 定时任务执行记录
type TaskExecution struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`    // 关联的定时任务 ID
	TaskName  string    `json:"task_name"`  // 任务名称（冗余）
	StartTime time.Time `json:"start_time"` // 开始时间
	EndTime   time.Time `json:"end_time"`   // 结束时间
	Duration  int64     `json:"duration"`   // 执行耗时（毫秒）
	Success   bool      `json:"success"`    // 是否成功
	Message   string    `json:"message"`    // 执行消息
	ErrorMsg  string    `json:"error_msg"`  // 错误信息

	// 执行结果数据
	// - 对于脚本执行：存储 PlayResult 的 ExtractedData
	// - 对于 Agent 执行：存储 Agent 返回的内容
	ResultData map[string]interface{} `json:"result_data,omitempty"` // 执行结果数据

	// 执行类型和关联信息
	ExecutionType ExecutionType `json:"execution_type"` // script, agent
	ScriptID      string        `json:"script_id,omitempty"`
	AgentSessionID string       `json:"agent_session_id,omitempty"`

	CreatedAt time.Time `json:"created_at"` // 记录创建时间
}
