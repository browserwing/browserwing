package models

import (
	"time"
)

// ScriptExecution 脚本执行记录
type ScriptExecution struct {
	ID          string    `json:"id"`           // 执行记录 ID
	ScriptID    string    `json:"script_id"`    // 关联的脚本 ID
	ScriptName  string    `json:"script_name"`  // 脚本名称（冗余，方便查询）
	StartTime   time.Time `json:"start_time"`   // 开始时间
	EndTime     time.Time `json:"end_time"`     // 结束时间
	Duration    int64     `json:"duration"`     // 执行耗时（毫秒）
	Success     bool      `json:"success"`      // 是否成功
	Message     string    `json:"message"`      // 执行消息
	ErrorMsg    string    `json:"error_msg"`    // 错误信息
	
	// 步骤统计
	TotalSteps   int `json:"total_steps"`   // 总步骤数
	SuccessSteps int `json:"success_steps"` // 成功步骤数
	FailedSteps  int `json:"failed_steps"`  // 失败步骤数
	
	// 抓取数据
	ExtractedData map[string]interface{} `json:"extracted_data,omitempty"` // 抓取到的数据
	
	// 录制视频
	VideoPath string `json:"video_path,omitempty"` // 录制视频路径
	
	CreatedAt time.Time `json:"created_at"` // 记录创建时间
}
