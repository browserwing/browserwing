package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/PuerkitoBio/goquery"
)

// WebSearchTool web search tool
type WebSearchTool struct{}

// Name tool name
func (t *WebSearchTool) Name() string {
	return "websearch"
}

// Description tool description
func (t *WebSearchTool) Description() string {
	return "Search web pages and return content in markdown format"
}

// InputSchema input parameter schema
func (t *WebSearchTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query term",
				"required":    true,
			},
			"num_results": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results to return",
				"default":     5,
			},
		},
	}
}

// Parameters parameter specification
func (t *WebSearchTool) Parameters() map[string]interfaces.ParameterSpec {
	return map[string]interfaces.ParameterSpec{
		"query": {
			Type:        "string",
			Description: "Search query term",
			Required:    true,
		},
		"num_results": {
			Type:        "integer",
			Description: "Number of results to return",
			Required:    false,
		},
	}
}

// Execute execute tool
func (t *WebSearchTool) Execute(ctx context.Context, input string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("failed to parse input parameters: %w", err)
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing required parameter: query")
	}

	numResults := 5
	if n, ok := args["num_results"].(float64); ok {
		numResults = int(n)
	}

	// Using DuckDuckGo search API (no API key required)
	searchURL := fmt.Sprintf("https://duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	resp, err := client.Get(searchURL)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search request failed with status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse search results: %w", err)
	}

	var results []string
	results = append(results, fmt.Sprintf("# 搜索结果: %s", query))
	results = append(results, "")

	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		if i >= numResults {
			return
		}

		title := s.Find(".result__title").Text()
		link := s.Find(".result__a").AttrOr("href", "")
		snippet := s.Find(".result__snippet").Text()

		// Clean up link
		if strings.HasPrefix(link, "/l/") {
			link = "https://duckduckgo.com" + link
		}

		results = append(results, fmt.Sprintf("## %d. %s", i+1, title))
		results = append(results, fmt.Sprintf("**链接:** %s", link))
		results = append(results, fmt.Sprintf("**摘要:** %s", snippet))
		results = append(results, "")
	})

	if len(results) <= 2 {
		return "No relevant search results found", nil
	}

	return strings.Join(results, "\n"), nil
}

// Run execute tool (compatible with old interface)
func (t *WebSearchTool) Run(ctx context.Context, input string) (string, error) {
	return t.Execute(ctx, input)
}
