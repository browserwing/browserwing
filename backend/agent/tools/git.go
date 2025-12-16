package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
)

// GitTool Git command tool
type GitTool struct {
	DefaultWorkDir string // 默认工作目录
}

// Name tool name
func (t *GitTool) Name() string {
	return "git"
}

// Description tool description
func (t *GitTool) Description() string {
	return "Execute Git commands locally (requires Git to be installed)"
}

// InputSchema input parameter schema
func (t *GitTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Git command to execute (e.g., status, log, commit)",
				"required":    true,
			},
			"args": map[string]interface{}{
				"type":        "array",
				"description": "List of command arguments",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"workdir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory path",
			},
		},
	}
}

// Parameters parameter specification
func (t *GitTool) Parameters() map[string]interfaces.ParameterSpec {
	return map[string]interfaces.ParameterSpec{
		"command": {
			Type:        "string",
			Description: "Git command to execute (e.g., status, log, commit)",
			Required:    true,
		},
		"args": {
			Type:        "array",
			Description: "List of command arguments",
			Required:    false,
		},
		"workdir": {
			Type:        "string",
			Description: "Working directory path",
			Required:    false,
		},
	}
}

// Execute execute tool
func (t *GitTool) Execute(ctx context.Context, input string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("failed to parse input parameters: %w", err)
	}

	command, ok := args["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("missing required parameter: command")
	}

	// Check if Git is available
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("git command is not available, please ensure Git is installed")
	}

	// Build complete command arguments
	cmdArgs := []string{command}
	if argsList, ok := args["args"].([]interface{}); ok {
		for _, arg := range argsList {
			if argStr, ok := arg.(string); ok {
				cmdArgs = append(cmdArgs, argStr)
			}
		}
	}

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)

	// Set working directory
	if workdir, ok := args["workdir"].(string); ok && workdir != "" {
		cmd.Dir = workdir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute Git command: %w\nstderr: %s", err, stderr.String())
	}

	result := stdout.String()
	if result == "" {
		result = "Execution successful, but no output"
	}

	return result, nil
}

// Run execute tool (compatible with old interface)
func (t *GitTool) Run(ctx context.Context, input string) (string, error) {
	return t.Execute(ctx, input)
}

// IsAvailable check if tool is available
func (t *GitTool) IsAvailable() bool {
	// Check if git is available
	_, err := exec.LookPath("git")
	return err == nil
}
