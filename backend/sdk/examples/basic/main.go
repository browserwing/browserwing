package main

import (
	"context"
	"log"
	"time"

	"github.com/browserwing/browserwing/models"
	"github.com/browserwing/browserwing/sdk"
)

func main() {
	// 创建 SDK 客户端 - 仅启用浏览器和脚本功能
	client, err := sdk.New(&sdk.Config{
		DatabasePath:  "./data/browserwing.db",
		EnableBrowser: true,
		EnableScript:  true,
		EnableAgent:   false, // 不启用 Agent
		LogLevel:      "info",
	})
	if err != nil {
		log.Fatalf("Failed to create SDK client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 1. 启动浏览器
	log.Println("Starting browser...")
	if err := client.Browser().Start(ctx); err != nil {
		log.Fatalf("Failed to start browser: %v", err)
	}
	defer func() {
		log.Println("Stopping browser...")
		client.Browser().Stop()
	}()

	// 2. 访问页面
	log.Println("Opening page...")
	if err := client.Browser().OpenPage(ctx, "https://www.example.com"); err != nil {
		log.Fatalf("Failed to open page: %v", err)
	}

	// 等待页面加载
	time.Sleep(2 * time.Second)

	// 3. 创建一个简单的脚本
	log.Println("Creating script...")
	script := &sdk.Script{
		Name:        "Example Script",
		Description: "A simple example script",
		URL:         "https://www.example.com",
		Actions: []models.ScriptAction{
			{
				Type: "navigate",
				URL:  "https://www.example.com",
			},
			{
				Type:     "wait",
				Duration: 2000, // 等待 2 秒
			},
			{
				Type:         "extract_text",
				Selector:     "h1",
				ExtractType:  "text",
				VariableName: "title",
				Description:  "提取页面标题",
			},
		},
		Tags:  []string{"example", "test"},
		Group: "examples",
	}

	scriptID, err := client.Script().Create(ctx, script)
	if err != nil {
		log.Fatalf("Failed to create script: %v", err)
	}
	log.Printf("✓ Script created with ID: %s", scriptID)

	// 4. 列出所有脚本
	log.Println("\nListing all scripts...")
	scripts, err := client.Script().List(ctx)
	if err != nil {
		log.Fatalf("Failed to list scripts: %v", err)
	}
	log.Printf("Found %d scripts:", len(scripts))
	for _, s := range scripts {
		log.Printf("  - %s: %s", s.Name, s.Description)
	}

	// 5. 执行脚本
	log.Println("\nPlaying script...")
	result, err := client.Script().Play(ctx, scriptID)
	if err != nil {
		log.Printf("Warning: Failed to play script: %v", err)
	} else {
		log.Printf("✓ Script execution result:")
		log.Printf("  Status: %s", result.Status)
		log.Printf("  Duration: %d ms", result.Duration)
		if len(result.ExtractedData) > 0 {
			log.Printf("  Extracted data:")
			for key, value := range result.ExtractedData {
				log.Printf("    %s: %s", key, value)
			}
		}
	}

	// 6. 获取执行历史
	log.Println("\nListing script executions...")
	executions, err := client.Script().ListExecutions(ctx, scriptID)
	if err != nil {
		log.Printf("Warning: Failed to list executions: %v", err)
	} else {
		log.Printf("Found %d executions:", len(executions))
		for _, e := range executions {
			log.Printf("  - %s: %s (duration: %d ms)", e.ID, e.Status, e.Duration)
		}
	}

	log.Println("\n✓ Example completed successfully!")
}
