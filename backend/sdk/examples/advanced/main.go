package main

import (
	"context"
	"log"
	"time"

	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/sdk"
)

func main() {
	// 创建 SDK 客户端
	client, err := sdk.New(&sdk.Config{
		DatabasePath:  "./data/browserwing.db",
		EnableBrowser: true,
		EnableScript:  true,
		EnableAgent:   false,
		LogLevel:      "info",
	})
	if err != nil {
		log.Fatalf("Failed to create SDK client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 启动浏览器
	log.Println("Starting browser...")
	if err := client.Browser().Start(ctx); err != nil {
		log.Fatalf("Failed to start browser: %v", err)
	}
	defer func() {
		log.Println("Stopping browser...")
		client.Browser().Stop()
	}()

	// 场景 1: 数据抓取脚本
	log.Println("\n=== Scenario 1: Data extraction ===")
	extractScript := &sdk.Script{
		Name:        "Extract Example.com Content",
		Description: "Extract title and heading from example.com",
		URL:         "https://www.example.com",
		Actions: []models.ScriptAction{
			{
				Type: "navigate",
				URL:  "https://www.example.com",
			},
			{
				Type:     "wait",
				Duration: 2000,
			},
			{
				Type:         "extract_text",
				Selector:     "h1",
				ExtractType:  "text",
				VariableName: "heading",
				Description:  "提取页面标题",
			},
			{
				Type:         "extract_text",
				Selector:     "p",
				ExtractType:  "text",
				VariableName: "paragraph",
				Description:  "提取第一段内容",
			},
		},
		Group: "scraping",
		Tags:  []string{"example", "extraction"},
	}

	scriptID1, err := client.Script().Create(ctx, extractScript)
	if err != nil {
		log.Fatalf("Failed to create extract script: %v", err)
	}
	log.Printf("✓ Created script: %s", scriptID1)

	result, err := client.Script().Play(ctx, scriptID1)
	if err != nil {
		log.Printf("Warning: Failed to play script: %v", err)
	} else {
		log.Printf("Extraction result:")
		for key, value := range result.ExtractedData {
			log.Printf("  %s: %s", key, value)
		}
	}

	// 场景 2: 批量访问页面
	log.Println("\n=== Scenario 2: Batch page visits ===")
	urls := []string{
		"https://www.example.com",
		"https://www.example.org",
		"https://www.example.net",
	}

	for i, url := range urls {
		log.Printf("Visiting page %d: %s", i+1, url)
		
		script := &sdk.Script{
			Name:        "Visit " + url,
			Description: "Simple page visit",
			URL:         url,
			Actions: []models.ScriptAction{
				{
					Type: "navigate",
					URL:  url,
				},
				{
					Type:     "wait",
					Duration: 1000,
				},
			},
			Group: "batch",
			Tags:  []string{"visit"},
		}

		scriptID, err := client.Script().Create(ctx, script)
		if err != nil {
			log.Printf("  Error creating script: %v", err)
			continue
		}

		result, err := client.Script().Play(ctx, scriptID)
		if err != nil {
			log.Printf("  Error playing script: %v", err)
		} else {
			log.Printf("  ✓ Status: %s, Duration: %dms", result.Status, result.Duration)
		}
		
		// 短暂延迟
		time.Sleep(500 * time.Millisecond)
	}

	// 场景 3: 定期监控
	log.Println("\n=== Scenario 3: Periodic monitoring ===")
	
	monitorScript := &sdk.Script{
		Name:        "Monitor Example.com",
		Description: "Check if example.com is accessible",
		URL:         "https://www.example.com",
		Actions: []models.ScriptAction{
			{
				Type: "navigate",
				URL:  "https://www.example.com",
			},
			{
				Type:     "wait",
				Duration: 1000,
			},
			{
				Type:         "extract_text",
				Selector:     "h1",
				ExtractType:  "text",
				VariableName: "status",
				Description:  "检查页面状态",
			},
		},
		Group: "monitoring",
		Tags:  []string{"health-check"},
	}

	scriptID2, err := client.Script().Create(ctx, monitorScript)
	if err != nil {
		log.Fatalf("Failed to create monitor script: %v", err)
	}

	// 执行 3 次,模拟定期检查
	for i := 1; i <= 3; i++ {
		log.Printf("\nMonitoring check #%d:", i)
		result, err := client.Script().Play(ctx, scriptID2)
		if err != nil {
			log.Printf("  ✗ Error: %v", err)
			continue
		}
		
		log.Printf("  ✓ Status: %s", result.Status)
		log.Printf("  Duration: %d ms", result.Duration)
		if len(result.ExtractedData) > 0 {
			log.Printf("  Data: %v", result.ExtractedData)
		}
		
		if i < 3 {
			time.Sleep(2 * time.Second)
		}
	}

	// 场景 4: 查看所有脚本和执行历史
	log.Println("\n=== Scenario 4: Script overview ===")
	
	// 列出所有脚本
	scripts, err := client.Script().List(ctx)
	if err != nil {
		log.Printf("Error listing scripts: %v", err)
	} else {
		log.Printf("Total scripts: %d", len(scripts))
		for _, s := range scripts {
			log.Printf("  - [%s] %s (Group: %s)", s.ID[:8], s.Name, s.Group)
		}
	}

	// 查看执行历史
	executions, err := client.Script().ListExecutions(ctx, "")
	if err != nil {
		log.Printf("Error listing executions: %v", err)
	} else {
		log.Printf("\nTotal executions: %d", len(executions))
		
		// 统计成功和失败
		success := 0
		failed := 0
		for _, e := range executions {
			if e.Status == "success" {
				success++
			} else {
				failed++
			}
		}
		log.Printf("  Success: %d, Failed: %d", success, failed)
		
		// 显示最近的几次执行
		log.Println("\nRecent executions:")
		count := len(executions)
		if count > 5 {
			count = 5
		}
		for i := 0; i < count; i++ {
			e := executions[i]
			log.Printf("  - [%s] %s: %s (%dms)",
				e.ID[:8], e.ScriptName, e.Status, e.Duration)
		}
	}

	// 场景 5: 清理示例脚本
	log.Println("\n=== Scenario 5: Cleanup (optional) ===")
	log.Println("Note: To clean up example scripts, you can delete them using client.Script().Delete(ctx, scriptID)")

	log.Println("\n✓ Advanced example completed!")
}
