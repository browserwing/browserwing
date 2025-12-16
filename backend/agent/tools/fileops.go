package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
)

// FileOpsTool file operations tool
type FileOpsTool struct {
	RootDir string // 根目录限制
}

// Name tool name
func (t *FileOpsTool) Name() string {
	return "fileops"
}

// Description tool description
func (t *FileOpsTool) Description() string {
	return "Local file operations tool, supporting read, write, list files and other operations"
}

// InputSchema input parameter schema
func (t *FileOpsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "Operation type: read, write, list, delete",
				"required":    true,
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File or directory path",
				"required":    true,
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to file (only required when action is write)",
			},
			"overwrite": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to overwrite existing file (only effective when action is write)",
				"default":     false,
			},
		},
	}
}

// Parameters parameter specification
func (t *FileOpsTool) Parameters() map[string]interfaces.ParameterSpec {
	return map[string]interfaces.ParameterSpec{
		"action": {
			Type:        "string",
			Description: "Operation type: read, write, list, delete",
			Required:    true,
		},
		"path": {
			Type:        "string",
			Description: "File or directory path",
			Required:    true,
		},
		"content": {
			Type:        "string",
			Description: "Content to write to file (only required when action is write)",
			Required:    false,
		},
		"overwrite": {
			Type:        "boolean",
			Description: "Whether to overwrite existing file (only effective when action is write)",
			Required:    false,
		},
	}
}

// Execute execute tool
func (t *FileOpsTool) Execute(ctx context.Context, input string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("failed to parse input parameters: %w", err)
	}

	action, ok := args["action"].(string)
	if !ok || action == "" {
		return "", fmt.Errorf("missing required parameter: action")
	}

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing required parameter: path")
	}

	switch action {
	case "read":
		return t.readFile(path)
	case "write":
		content, ok := args["content"].(string)
		if !ok {
			return "", fmt.Errorf("content parameter is required for writing files")
		}
		overwrite := false
		if o, ok := args["overwrite"].(bool); ok {
			overwrite = o
		}
		return t.writeFile(path, content, overwrite)
	case "list":
		return t.listFiles(path)
	case "delete":
		return t.deleteFile(path)
	default:
		return "", fmt.Errorf("unsupported operation type: %s", action)
	}
}

// readFile read file content
func (t *FileOpsTool) readFile(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// writeFile write file content
func (t *FileOpsTool) writeFile(path, content string, overwrite bool) (string, error) {
	// Check if file exists
	if _, err := os.Stat(path); err == nil && !overwrite {
		return "", fmt.Errorf("file already exists, use overwrite=true parameter to overwrite")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := ioutil.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File written successfully: %s", path), nil
}

// listFiles list files in directory
func (t *FileOpsTool) listFiles(path string) (string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var result []string
	result = append(result, fmt.Sprintf("Directory contents: %s", path))
	result = append(result, "")

	for _, file := range files {
		if file.IsDir() {
			result = append(result, fmt.Sprintf("📁 %s", file.Name()))
		} else {
			result = append(result, fmt.Sprintf("📄 %s (%d bytes)", file.Name(), file.Size()))
		}
	}

	if len(result) <= 2 {
		return "Directory is empty", nil
	}

	return strings.Join(result, "\n"), nil
}

// deleteFile delete file
func (t *FileOpsTool) deleteFile(path string) (string, error) {
	if err := os.RemoveAll(path); err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}
	return fmt.Sprintf("File deleted successfully: %s", path), nil
}

// Run execute tool (compatible with old interface)
func (t *FileOpsTool) Run(ctx context.Context, input string) (string, error) {
	return t.Execute(ctx, input)
}
