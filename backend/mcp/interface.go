package mcp

import (
	"context"

	"github.com/browserwing/browserwing/models"
)

// IMCPServer MCP 服务器接口
type IMCPServer interface {
	Start() error
	Stop()
	RegisterScript(script *models.Script) error
	UnregisterScript(scriptID string)
	GetStatus() map[string]interface{}
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (interface{}, error)
}

// 确保两个实现都满足接口
var (
	_ IMCPServer = (*MCPServer)(nil)
)
