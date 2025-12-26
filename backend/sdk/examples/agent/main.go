package main

import (
	"context"
	"log"

	"github.com/browserwing/browserwing/sdk"
)

func main() {
	// 创建 SDK 客户端 - 启用所有功能
	client, err := sdk.New(&sdk.Config{
		DatabasePath:  "./data/browserwing.db",
		EnableBrowser: true,
		EnableScript:  true,
		EnableAgent:   true,
		LogLevel:      "info",
		LLMConfig: &sdk.LLMConfig{
			Provider: "openai",
			APIKey:   "your-api-key-here", // ⚠️ 请替换为实际的 API Key
			Model:    "gpt-4",
			BaseURL:  "https://api.openai.com/v1",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create SDK client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 检查 LLM 配置
	if client.Agent() == nil {
		log.Fatal("Agent is not enabled. Please provide valid LLM configuration.")
	}

	// 1. 创建 Agent 会话
	log.Println("Creating agent session...")
	sessionID, err := client.Agent().CreateSession(ctx)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	log.Printf("✓ Session created with ID: %s", sessionID)

	// 2. 发送消息 - 非流式
	log.Println("\n=== Non-streaming example ===")
	log.Println("Sending: Hello! What can you help me with?")
	response, err := client.Agent().SendMessage(ctx, sessionID, "Hello! What can you help me with?")
	if err != nil {
		log.Printf("Warning: Failed to send message: %v", err)
	} else {
		log.Printf("Agent response: %s", response)
	}

	// 3. 发送消息 - 流式
	log.Println("\n=== Streaming example ===")
	log.Print("Agent response (streaming): ")
	err = client.Agent().SendMessageStream(ctx, sessionID, "Tell me a short joke", func(chunk *sdk.MessageChunk) {
		switch chunk.Type {
		case "message":
			// 打印消息内容
			log.Print(chunk.Content)
		case "tool_call":
			// 工具调用
			if chunk.ToolCall != nil {
				log.Printf("\n[Tool Call] %s: %s", chunk.ToolCall.ToolName, chunk.ToolCall.Status)
				if chunk.ToolCall.Message != "" {
					log.Printf("  Message: %s", chunk.ToolCall.Message)
				}
			}
		case "error":
			// 错误
			log.Printf("\n[Error] %s", chunk.Error)
		case "done":
			// 完成
			log.Println("\n[Done]")
		}
	})
	if err != nil {
		log.Printf("Warning: Failed to send message: %v", err)
	}

	// 4. 获取会话历史
	log.Println("\n=== Session history ===")
	session, err := client.Agent().GetSession(ctx, sessionID)
	if err != nil {
		log.Printf("Warning: Failed to get session: %v", err)
	} else {
		log.Printf("Session has %d messages:", len(session.Messages))
		for i, msg := range session.Messages {
			content := msg.Content
			if len(content) > 50 {
				content = content[:50] + "..."
			}
			log.Printf("  %d. [%s] %s", i+1, msg.Role, content)
		}
	}

	// 5. 列出所有会话
	log.Println("\n=== All sessions ===")
	sessions, err := client.Agent().ListSessions(ctx)
	if err != nil {
		log.Printf("Warning: Failed to list sessions: %v", err)
	} else {
		log.Printf("Found %d sessions:", len(sessions))
		for _, s := range sessions {
			log.Printf("  - %s: %d messages", s.ID, len(s.Messages))
		}
	}

	// 6. 使用浏览器脚本工具(如果已配置)
	log.Println("\n=== Using browser automation (optional) ===")
	log.Println("Asking agent to visit a website...")
	err = client.Agent().SendMessageStream(ctx, sessionID, 
		"Please visit https://www.example.com and tell me the page title", 
		func(chunk *sdk.MessageChunk) {
			switch chunk.Type {
			case "message":
				log.Print(chunk.Content)
			case "tool_call":
				if chunk.ToolCall != nil {
					log.Printf("\n[Tool Call] %s: %s", chunk.ToolCall.ToolName, chunk.ToolCall.Status)
					if chunk.ToolCall.Message != "" {
						log.Printf("  %s", chunk.ToolCall.Message)
					}
				}
			case "done":
				log.Println("\n[Done]")
			}
		})
	if err != nil {
		log.Printf("Note: %v (This is normal if MCP tools are not configured)", err)
	}

	log.Println("\n✓ Agent example completed!")
}
