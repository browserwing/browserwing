package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
)

// BarkTool Bark push notification tool
type BarkTool struct {
	APIKey string // 默认的API Key
}

// Name tool name
func (t *BarkTool) Name() string {
	return "bark"
}

// Description tool description
func (t *BarkTool) Description() string {
	return "Send iOS push notifications (based on Bark service)"
}

// InputSchema input parameter schema
func (t *BarkTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"key": map[string]interface{}{
				"type":        "string",
				"description": "Bark device key",
				"required":    true,
			},
			"body": map[string]interface{}{
				"type":        "string",
				"description": "Push content",
				"required":    true,
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Push title",
			},
			"subtitle": map[string]interface{}{
				"type":        "string",
				"description": "Push subtitle",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to open when notification is clicked",
			},
			"group": map[string]interface{}{
				"type":        "string",
				"description": "Push group name",
			},
			"icon": map[string]interface{}{
				"type":        "string",
				"description": "Push icon URL (iOS 15+ support)",
			},
			"sound": map[string]interface{}{
				"type":        "string",
				"description": "Push sound",
			},
			"level": map[string]interface{}{
				"type":        "string",
				"description": "Notification level: active, timeSensitive, passive",
			},
		},
	}
}

// Parameters parameter specification
func (t *BarkTool) Parameters() map[string]interfaces.ParameterSpec {
	return map[string]interfaces.ParameterSpec{
		"key": {
			Type:        "string",
			Description: "Bark device key",
			Required:    true,
		},
		"body": {
			Type:        "string",
			Description: "Push content",
			Required:    true,
		},
		"title": {
			Type:        "string",
			Description: "Push title",
			Required:    false,
		},
		"subtitle": {
			Type:        "string",
			Description: "Push subtitle",
			Required:    false,
		},
		"url": {
			Type:        "string",
			Description: "URL to open when notification is clicked",
			Required:    false,
		},
		"group": {
			Type:        "string",
			Description: "Push group name",
			Required:    false,
		},
		"icon": {
			Type:        "string",
			Description: "Push icon URL (iOS 15+ support)",
			Required:    false,
		},
		"sound": {
			Type:        "string",
			Description: "Push sound",
			Required:    false,
		},
		"level": {
			Type:        "string",
			Description: "Notification level: active, timeSensitive, passive",
			Required:    false,
		},
	}
}

// Execute execute tool
func (t *BarkTool) Execute(ctx context.Context, input string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("failed to parse input parameters: %w", err)
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return "", fmt.Errorf("missing required parameter: key")
	}

	body, ok := args["body"].(string)
	if !ok || body == "" {
		return "", fmt.Errorf("missing required parameter: body")
	}

	// Build Bark API URL
	baseURL := "https://api.day.app"

	// Build URL path
	var path strings.Builder
	path.WriteString(fmt.Sprintf("/%s", key))

	title, _ := args["title"].(string)
	subtitle, _ := args["subtitle"].(string)

	if title != "" && subtitle != "" {
		path.WriteString(fmt.Sprintf("/%s/%s/%s", url.PathEscape(title), url.PathEscape(subtitle), url.PathEscape(body)))
	} else if title != "" {
		path.WriteString(fmt.Sprintf("/%s/%s", url.PathEscape(title), url.PathEscape(body)))
	} else {
		path.WriteString(fmt.Sprintf("/%s", url.PathEscape(body)))
	}

	// Build query parameters
	queryParams := url.Values{}

	if u, ok := args["url"].(string); ok && u != "" {
		queryParams.Add("url", u)
	}

	if group, ok := args["group"].(string); ok && group != "" {
		queryParams.Add("group", group)
	}

	if icon, ok := args["icon"].(string); ok && icon != "" {
		queryParams.Add("icon", icon)
	}

	if sound, ok := args["sound"].(string); ok && sound != "" {
		queryParams.Add("sound", sound)
	}

	if level, ok := args["level"].(string); ok && level != "" {
		queryParams.Add("level", level)
	}

	// Full URL
	fullURL := fmt.Sprintf("%s%s", baseURL, path.String())
	if len(queryParams) > 0 {
		fullURL = fmt.Sprintf("%s?%s", fullURL, queryParams.Encode())
	}

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to send push request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("push request failed with status code: %d", resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If parsing fails, return original response directly
		return fmt.Sprintf("Push notification sent successfully\nResponse: %s", string(respBody)), nil
	}

	if errMsg, ok := result["message"].(string); ok {
		return fmt.Sprintf("Push notification sent successfully: %s", errMsg), nil
	}

	return "Push notification sent successfully", nil
}

// Run execute tool (compatible with old interface)
func (t *BarkTool) Run(ctx context.Context, input string) (string, error) {
	return t.Execute(ctx, input)
}
