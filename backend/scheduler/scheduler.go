package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/storage"
	"github.com/robfig/cron/v3"
)

// TaskExecutor 任务执行器接口
type TaskExecutor interface {
	ExecuteScript(ctx context.Context, task *models.ScheduledTask) (map[string]interface{}, error)
	ExecuteAgent(ctx context.Context, task *models.ScheduledTask) (map[string]interface{}, error)
}

// Scheduler 定时任务调度器
type Scheduler struct {
	db       *storage.BoltDB
	executor TaskExecutor
	cron     *cron.Cron
	mu       sync.RWMutex
	tasks    map[string]cron.EntryID // taskID -> cronEntryID
	stopCh   chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewScheduler 创建新的调度器
func NewScheduler(db *storage.BoltDB, executor TaskExecutor) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建支持秒级的 cron 调度器
	c := cron.New(cron.WithSeconds())

	return &Scheduler{
		db:       db,
		executor: executor,
		cron:     c,
		tasks:    make(map[string]cron.EntryID),
		stopCh:   make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	log.Println("[Scheduler] Starting scheduler...")

	// 加载所有已启用的定时任务
	tasks, err := s.db.ListScheduledTasks()
	if err != nil {
		return fmt.Errorf("failed to load scheduled tasks: %w", err)
	}

	// 添加任务到调度器
	for _, task := range tasks {
		if task.Enabled {
			if err := s.AddTask(&task); err != nil {
				log.Printf("[Scheduler] Failed to add task %s: %v", task.Name, err)
			}
		}
	}

	// 启动 cron 调度器
	s.cron.Start()
	log.Printf("[Scheduler] Scheduler started with %d tasks", len(tasks))

	// 启动一次性任务检查协程
	go s.checkAtTasks()

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	log.Println("[Scheduler] Stopping scheduler...")
	s.cancel()
	close(s.stopCh)

	ctx := s.cron.Stop()
	<-ctx.Done()

	log.Println("[Scheduler] Scheduler stopped")
}

// AddTask 添加任务到调度器
func (s *Scheduler) AddTask(task *models.ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果任务已存在，先移除
	if entryID, exists := s.tasks[task.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.tasks, task.ID)
	}

	if !task.Enabled {
		return nil
	}

	switch task.ScheduleType {
	case models.ScheduleTypeAt:
		// 一次性任务，在 checkAtTasks 中处理
		return s.scheduleAtTask(task)
	case models.ScheduleTypeEvery:
		// 固定间隔任务
		return s.scheduleEveryTask(task)
	case models.ScheduleTypeCron:
		// Cron 表达式任务
		return s.scheduleCronTask(task)
	default:
		return fmt.Errorf("unknown schedule type: %s", task.ScheduleType)
	}
}

// RemoveTask 从调度器移除任务
func (s *Scheduler) RemoveTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.tasks[taskID]; exists {
		s.cron.Remove(entryID)
		delete(s.tasks, taskID)
		log.Printf("[Scheduler] Removed task: %s", taskID)
	}
}

// scheduleAtTask 调度一次性任务
func (s *Scheduler) scheduleAtTask(task *models.ScheduledTask) error {
	// 解析时间
	executeAt, err := time.Parse(time.RFC3339, task.ScheduleConfig)
	if err != nil {
		return fmt.Errorf("invalid at time format: %w", err)
	}

	// 更新下次执行时间
	task.NextExecutionTime = &executeAt
	if err := s.db.UpdateScheduledTask(task); err != nil {
		log.Printf("[Scheduler] Failed to update next execution time for task %s: %v", task.ID, err)
	}

	log.Printf("[Scheduler] Scheduled at task %s (%s) to execute at %s", task.ID, task.Name, executeAt.Format(time.RFC3339))
	return nil
}

// scheduleEveryTask 调度固定间隔任务
func (s *Scheduler) scheduleEveryTask(task *models.ScheduledTask) error {
	// 解析间隔时间
	duration, err := time.ParseDuration(task.ScheduleConfig)
	if err != nil {
		return fmt.Errorf("invalid every duration format: %w", err)
	}

	// 创建 cron 表达式：每隔 N 秒执行一次
	// 注意：cron 不直接支持间隔，我们使用 @every 语法
	cronExpr := fmt.Sprintf("@every %s", task.ScheduleConfig)

	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeTask(task)
	})
	if err != nil {
		return fmt.Errorf("failed to add every task to cron: %w", err)
	}

	s.tasks[task.ID] = entryID

	// 计算下次执行时间
	nextTime := time.Now().Add(duration)
	task.NextExecutionTime = &nextTime
	if err := s.db.UpdateScheduledTask(task); err != nil {
		log.Printf("[Scheduler] Failed to update next execution time for task %s: %v", task.ID, err)
	}

	log.Printf("[Scheduler] Scheduled every task %s (%s) to execute every %s", task.ID, task.Name, duration)
	return nil
}

// scheduleCronTask 调度 Cron 任务
func (s *Scheduler) scheduleCronTask(task *models.ScheduledTask) error {
	entryID, err := s.cron.AddFunc(task.ScheduleConfig, func() {
		s.executeTask(task)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron task: %w", err)
	}

	s.tasks[task.ID] = entryID

	// 计算下次执行时间
	entry := s.cron.Entry(entryID)
	nextTime := entry.Next
	task.NextExecutionTime = &nextTime
	if err := s.db.UpdateScheduledTask(task); err != nil {
		log.Printf("[Scheduler] Failed to update next execution time for task %s: %v", task.ID, err)
	}

	log.Printf("[Scheduler] Scheduled cron task %s (%s) with expression: %s, next run at %s",
		task.ID, task.Name, task.ScheduleConfig, nextTime.Format(time.RFC3339))
	return nil
}

// checkAtTasks 检查并执行一次性任务
func (s *Scheduler) checkAtTasks() {
	ticker := time.NewTicker(10 * time.Second) // 每 10 秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processAtTasks()
		}
	}
}

// processAtTasks 处理一次性任务
func (s *Scheduler) processAtTasks() {
	tasks, err := s.db.ListScheduledTasks()
	if err != nil {
		log.Printf("[Scheduler] Failed to list tasks: %v", err)
		return
	}

	now := time.Now()
	for _, task := range tasks {
		if !task.Enabled || task.ScheduleType != models.ScheduleTypeAt {
			continue
		}

		if task.NextExecutionTime != nil && task.NextExecutionTime.Before(now) {
			// 执行任务
			go s.executeTask(&task)

			// 执行后禁用任务
			task.Enabled = false
			task.NextExecutionTime = nil
			if err := s.db.UpdateScheduledTask(&task); err != nil {
				log.Printf("[Scheduler] Failed to disable at task %s: %v", task.ID, err)
			}
		}
	}
}

// executeTask 执行任务
func (s *Scheduler) executeTask(task *models.ScheduledTask) {
	log.Printf("[Scheduler] Executing task %s (%s), type: %s", task.ID, task.Name, task.ExecutionType)

	// 创建执行记录
	execution := &models.TaskExecution{
		ID:            generateID(),
		TaskID:        task.ID,
		TaskName:      task.Name,
		StartTime:     time.Now(),
		ExecutionType: task.ExecutionType,
		CreatedAt:     time.Now(),
	}

	var resultData map[string]interface{}
	var err error

	// 执行任务
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 5分钟超时
	defer cancel()

	switch task.ExecutionType {
	case models.ExecutionTypeScript:
		execution.ScriptID = task.ScriptID
		resultData, err = s.executor.ExecuteScript(ctx, task)
	case models.ExecutionTypeAgent:
		execution.AgentSessionID = task.AgentSessionID
		resultData, err = s.executor.ExecuteAgent(ctx, task)
	default:
		err = fmt.Errorf("unknown execution type: %s", task.ExecutionType)
	}

	// 记录执行结果
	execution.EndTime = time.Now()
	execution.Duration = execution.EndTime.Sub(execution.StartTime).Milliseconds()
	execution.Success = err == nil
	execution.ResultData = resultData

	if err != nil {
		execution.ErrorMsg = err.Error()
		execution.Message = fmt.Sprintf("task.messages.failed: %v", err)
		log.Printf("[Scheduler] Task %s failed: %v", task.Name, err)
	} else {
		execution.Message = "task.messages.success"
		log.Printf("[Scheduler] Task %s completed successfully", task.Name)
	}

	// 保存执行记录
	if err := s.db.CreateTaskExecution(execution); err != nil {
		log.Printf("[Scheduler] Failed to save execution record: %v", err)
	}

	// 更新任务统计
	s.updateTaskStats(task, execution.Success)

	// 更新下次执行时间（对于重复任务）
	s.updateNextExecutionTime(task)
}

// updateTaskStats 更新任务统计信息
func (s *Scheduler) updateTaskStats(task *models.ScheduledTask, success bool) {
	// 重新从数据库加载任务以获取最新状态
	latestTask, err := s.db.GetScheduledTask(task.ID)
	if err != nil {
		log.Printf("[Scheduler] Failed to load task for stats update: %v", err)
		return
	}

	now := time.Now()
	latestTask.LastExecutionTime = &now
	latestTask.ExecutionCount++

	if success {
		latestTask.SuccessCount++
		latestTask.LastExecutionStatus = "success"
	} else {
		latestTask.FailedCount++
		latestTask.LastExecutionStatus = "failed"
	}

	if err := s.db.UpdateScheduledTask(latestTask); err != nil {
		log.Printf("[Scheduler] Failed to update task stats: %v", err)
	}
}

// updateNextExecutionTime 更新下次执行时间
func (s *Scheduler) updateNextExecutionTime(task *models.ScheduledTask) {
	s.mu.RLock()
	entryID, exists := s.tasks[task.ID]
	s.mu.RUnlock()

	if !exists {
		return
	}

	// 获取下次执行时间
	entry := s.cron.Entry(entryID)
	if !entry.Next.IsZero() {
		nextTime := entry.Next
		task.NextExecutionTime = &nextTime

		if err := s.db.UpdateScheduledTask(task); err != nil {
			log.Printf("[Scheduler] Failed to update next execution time: %v", err)
		}
	}
}

// ReloadTask 重新加载任务（用于任务更新后）
func (s *Scheduler) ReloadTask(taskID string) error {
	task, err := s.db.GetScheduledTask(taskID)
	if err != nil {
		return err
	}

	// 移除旧任务
	s.RemoveTask(taskID)

	// 添加新任务
	if task.Enabled {
		return s.AddTask(task)
	}

	return nil
}

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// MarshalResultData 序列化结果数据
func MarshalResultData(data interface{}) (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}
