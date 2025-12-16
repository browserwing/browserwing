package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
)

// PyExecTool Python execution tool
type PyExecTool struct{}

// Name tool name
func (t *PyExecTool) Name() string {
	return "pyexec"
}

// Description tool description
func (t *PyExecTool) Description() string {
	return "Execute Python code locally (requires Python to be installed)"
}

// InputSchema input parameter schema
func (t *PyExecTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"code": map[string]interface{}{
				"type":        "string",
				"description": "Python code to execute",
				"required":    true,
			},
			"version": map[string]interface{}{
				"type":        "string",
				"description": "Python version (python or python3)",
				"default":     "python3",
			},
		},
	}
}

// Parameters parameter specification
func (t *PyExecTool) Parameters() map[string]interfaces.ParameterSpec {
	return map[string]interfaces.ParameterSpec{
		"code": {
			Type:        "string",
			Description: "Python code to execute",
			Required:    true,
		},
		"version": {
			Type:        "string",
			Description: "Python version (python or python3)",
			Required:    false,
		},
	}
}

// Execute execute tool
func (t *PyExecTool) Execute(ctx context.Context, input string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("failed to parse input parameters: %w", err)
	}

	code, ok := args["code"].(string)
	if !ok || code == "" {
		return "", fmt.Errorf("missing required parameter: code")
	}

	version := "python3"
	if v, ok := args["version"].(string); ok && v != "" {
		version = v
	}

	// Check if Python is available
	if _, err := exec.LookPath(version); err != nil {
		return "", fmt.Errorf("%s command is not available, please ensure Python is installed", version)
	}

	cmd := exec.CommandContext(ctx, version, "-c", code)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute Python code: %w\nstderr: %s", err, stderr.String())
	}

	result := stdout.String()
	if result == "" {
		result = "Execution successful, but no output"
	}

	return result, nil
}

// Run execute tool (compatible with old interface)
func (t *PyExecTool) Run(ctx context.Context, input string) (string, error) {
	return t.Execute(ctx, input)
}

// IsAvailable check if tool is available
func (t *PyExecTool) IsAvailable() bool {
	// Check if python3 or python is available
	for _, cmd := range []string{"python3", "python"} {
		if _, err := exec.LookPath(cmd); err == nil {
			return true
		}
	}
	return false
}
