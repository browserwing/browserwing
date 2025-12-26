package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/browserwing/browserwing/models"
	"github.com/google/uuid"
)

// BrowserClient 浏览器客户端
type BrowserClient struct {
	client *Client
}

// Start 启动浏览器
func (bc *BrowserClient) Start(ctx context.Context) error {
	if bc.client.browserManager == nil {
		return fmt.Errorf("browser manager not initialized")
	}

	return bc.client.browserManager.Start(ctx)
}

// Stop 停止浏览器
func (bc *BrowserClient) Stop() error {
	if bc.client.browserManager == nil {
		return fmt.Errorf("browser manager not initialized")
	}

	return bc.client.browserManager.Stop()
}

// IsRunning 检查浏览器是否正在运行
func (bc *BrowserClient) IsRunning() bool {
	if bc.client.browserManager == nil {
		return false
	}

	return bc.client.browserManager.IsRunning()
}

// OpenPage 打开指定页面
func (bc *BrowserClient) OpenPage(ctx context.Context, url string) error {
	if bc.client.browserManager == nil {
		return fmt.Errorf("browser manager not initialized")
	}

	if !bc.IsRunning() {
		return fmt.Errorf("browser is not running, please start browser first")
	}

	return bc.client.browserManager.OpenPage(url, "")
}

// GetStatus 获取浏览器状态
func (bc *BrowserClient) GetStatus() (*BrowserStatus, error) {
	if bc.client.browserManager == nil {
		return nil, fmt.Errorf("browser manager not initialized")
	}

	return &BrowserStatus{
		IsRunning: bc.client.browserManager.IsRunning(),
		StartTime: time.Now(), // 简化实现
	}, nil
}

// SaveCookies 保存当前浏览器的 Cookie
// id: Cookie 存储 ID
// platform: 平台名称
// 返回: Cookie ID
func (bc *BrowserClient) SaveCookies(ctx context.Context, id, platform string) (string, error) {
	if bc.client.browserManager == nil {
		return "", fmt.Errorf("browser manager not initialized")
	}

	if !bc.IsRunning() {
		return "", fmt.Errorf("browser is not running")
	}

	// 生成 ID
	if id == "" {
		id = uuid.New().String()
	}

	// 注意: 当前简化实现,需要通过 Web API 或其他方式保存 Cookie
	// 这里返回ID表示操作已接受,实际保存需要扩展实现
	return id, fmt.Errorf("save cookies: please use browser manager's save cookie API")
}

// ImportCookies 导入 Cookie 到浏览器
func (bc *BrowserClient) ImportCookies(ctx context.Context, cookieID string) error {
	if bc.client.browserManager == nil {
		return fmt.Errorf("browser manager not initialized")
	}

	if !bc.IsRunning() {
		return fmt.Errorf("browser is not running")
	}

	// 从数据库获取 Cookie
	cookieStore, err := bc.client.db.GetCookies(cookieID)
	if err != nil {
		return fmt.Errorf("failed to get cookies from database: %w", err)
	}

	if cookieStore == nil {
		return fmt.Errorf("cookies not found: %s", cookieID)
	}

	// 注意: 当前简化实现,需要通过 Web API 或扩展 browser.Manager
	// 提供公开方法来导入 cookies
	return fmt.Errorf("import cookies: please use browser manager's import cookie API")
}

// ListCookies 列出所有保存的 Cookie 配置
// 注意: 当前实现不支持列表操作
func (bc *BrowserClient) ListCookies() ([]*models.CookieStore, error) {
	return nil, fmt.Errorf("list cookies not implemented")
}

// GetCookies 获取指定的 Cookie 配置
func (bc *BrowserClient) GetCookies(cookieID string) (*models.CookieStore, error) {
	if bc.client.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	cookieStore, err := bc.client.db.GetCookies(cookieID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	return cookieStore, nil
}

// DeleteCookies 删除指定的 Cookie 配置
// 注意: 当前实现不支持删除操作
func (bc *BrowserClient) DeleteCookies(cookieID string) error {
	return fmt.Errorf("delete cookies not implemented")
}
