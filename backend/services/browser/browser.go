package browser

import (
	"context"
	"encoding/json"

	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
)

type Browser interface {
	Close()
	NewPage() *rod.Page
}

func NewBrowserV2(ctx context.Context, headless bool, options ...OptionV2) Browser {
	cfg := &Config{}
	for _, opt := range options {
		opt(cfg)
	}

	opts := []OptionV2{
		WithHeadless(headless),
	}
	if cfg.ChromeBinPath != "" {
		opts = append(opts, WithChromeBinPath(cfg.ChromeBinPath))
	}
	if cfg.UserDataDir != "" {
		opts = append(opts, WithUserDataDir(cfg.UserDataDir))
	}

	return New(ctx, opts...)
}

// Browser represents a headless browser instance with an underlying rod.Browser and launcher.
type BrowserV2 struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
}

// Config holds the configuration options for the browser.
type Config struct {
	Headless      bool   // Whether to run browser in headless mode
	UserAgent     string // Custom user agent string
	Cookies       string // JSON string of cookies to set
	ChromeBinPath string // Custom Chrome/Chromium executable path
	UserDataDir   string // User data directory path (not implemented yet)

	Trace bool // Whether to enable tracing (not implemented yet)
}

// Option is a functional option for configuring the browser.
type OptionV2 func(*Config)

// newDefaultConfig returns a new Config with default values.
func newDefaultConfig() *Config {
	return &Config{
		Headless:      true,
		UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		Cookies:       "",
		ChromeBinPath: "", // Empty means auto-detect
		Trace:         false,
	}
}

// WithHeadless sets whether the browser should run in headless mode.
func WithHeadless(headless bool) OptionV2 {
	return func(c *Config) {
		c.Headless = headless
	}
}

func WithUserDataDir(dir string) OptionV2 {
	return func(c *Config) {
		c.UserDataDir = dir
	}
}

// WithUserAgent sets a custom user agent string for the browser.
func WithUserAgent(userAgent string) OptionV2 {
	return func(c *Config) {
		c.UserAgent = userAgent
	}
}

// WithCookies sets cookies for the browser from a JSON string.
// The cookies should be in the format expected by proto.NetworkCookie.
func WithCookies(cookies string) OptionV2 {
	return func(c *Config) {
		c.Cookies = cookies
	}
}

// WithChromeBinPath sets a custom Chrome/Chromium executable path.
// If not set or empty, launcher will auto-detect or download a browser.
// Common paths:
//   - macOS: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
//   - Linux: "/usr/bin/google-chrome" or "/usr/bin/chromium"
//   - Windows: "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
func WithChromeBinPath(path string) OptionV2 {
	return func(c *Config) {
		c.ChromeBinPath = path
	}
}

func WithTrace() OptionV2 {
	return func(c *Config) {
		c.Trace = true
	}
}

// New creates a new Browser instance with the provided options.
// It initializes a Chrome browser with stealth mode enabled.
func New(ctx context.Context, options ...OptionV2) *BrowserV2 {
	cfg := newDefaultConfig()
	for _, option := range options {
		option(cfg)
	}

	l := launcher.New().
		Headless(cfg.Headless).
		Leakless(false).
		UserDataDir(cfg.UserDataDir).
		Set("--no-sandbox").
		Set(
			"user-agent", cfg.UserAgent,
		)

	// Set custom Chrome binary path if provided
	if cfg.ChromeBinPath != "" {
		l = l.Bin(cfg.ChromeBinPath)
	}

	url := l.MustLaunch()

	browser := rod.New().
		ControlURL(url).
		Trace(cfg.Trace).
		MustConnect()

	// 加载 cookies
	if cfg.Cookies != "" {
		var cookies []*proto.NetworkCookie
		if err := json.Unmarshal([]byte(cfg.Cookies), &cookies); err != nil {
			logger.Warn(ctx, "failed to unmarshal cookies: %v", err)
		} else {
			browser.MustSetCookies(cookies...)
		}
	}

	return &BrowserV2{
		browser:  browser,
		launcher: l,
	}
}

func (b *BrowserV2) SaveCookies(ctx context.Context, page *rod.Page) ([]*proto.NetworkCookie, error) {
	cookies, err := page.Browser().GetCookies()
	if err != nil {
		logger.Error(ctx, "failed to get cookies: %v", err)
		return nil, err
	}

	return cookies, nil
}

// Close closes the browser and cleans up resources.
func (b *BrowserV2) Close() {
	b.browser.MustClose()
	b.launcher.Cleanup()
}

// NewPage creates a new page with stealth mode enabled.
// The returned page can be used to navigate and interact with web content.
func (b *BrowserV2) NewPage() *rod.Page {
	return stealth.MustPage(b.browser)
}
