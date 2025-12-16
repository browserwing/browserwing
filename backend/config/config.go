package config

import (
	"os"

	"github.com/browserwing/browserwing/pkg/logger"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Debug     bool                 `json:"debug" yaml:"debug" toml:"debug"`
	Server    *ServerConfig        `json:"server" yaml:"server" toml:"server"`
	Database  *DatabaseConfig      `json:"database" yaml:"database" toml:"database"`
	LLM       *LLMConfig           `json:"llm" yaml:"llm" toml:"llm"`    // 保留用于默认配置
	LLMs      []LLMConfig          `json:"llms" yaml:"llms" toml:"llms"` // 新增：多个 LLM 配置
	Browser   *BrowserConfig       `json:"browser" yaml:"browser" toml:"browser"`
	AssetsDir string               `json:"assets_dir,omitempty" yaml:"assets_dir,omitempty" toml:"assets_dir,omitempty"`
	Log       *logger.LoggerConfig `json:"log,omitempty" yaml:"log,omitempty" toml:"log,omitempty"`
}

type ServerConfig struct {
	Port string `json:"port" toml:"port"`
	Host string `json:"host" toml:"host"`
}

type DatabaseConfig struct {
	Path string `json:"path" toml:"path"`
}

type LLMConfig struct {
	Name     string `json:"name" toml:"name"` // 新增：LLM 名称标识
	Provider string `json:"provider" toml:"provider"`
	APIKey   string `json:"api_key" toml:"api_key"`
	Model    string `json:"model" toml:"model"`
	BaseURL  string `json:"base_url,omitempty" toml:"base_url,omitempty"` // 新增：自定义 API 地址
}

type BrowserConfig struct {
	BinPath     string `json:"bin_path" toml:"bin_path"`
	UserDataDir string `json:"user_data_dir" toml:"user_data_dir"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		browserCfg := &BrowserConfig{}
		// 根据系统设置默认的 binpath
		chromeBinPath := ""
		if envPath := os.Getenv("CHROME_BIN_PATH"); envPath != "" {
			chromeBinPath = envPath
		} else {
			// 常见的 Chrome/Chromium 安装路径
			commonPaths := []string{
				"/usr/bin/google-chrome",
				"/usr/bin/chromium-browser",
				"/usr/bin/chromium",
				"/usr/bin/google-chrome-stable",
				"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
				"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
				"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
			}
			for _, p := range commonPaths {
				if _, err := os.Stat(p); err == nil {
					chromeBinPath = p
					break
				}
			}
		}
		browserCfg.BinPath = chromeBinPath
		browserCfg.UserDataDir = "./chrome_user_data"
		// 如果本地不存在 data 和 log 目录，则创建
		_, err := os.Stat("./data")
		if os.IsNotExist(err) {
			os.Mkdir("./data", 0o755)
		}
		_, err = os.Stat("./log")
		if os.IsNotExist(err) {
			os.Mkdir("./log", 0o755)
		}
		// 返回默认配置
		defConfig := &Config{
			Server: &ServerConfig{
				Port: "8080",
				Host: "0.0.0.0",
			},
			Database: &DatabaseConfig{
				Path: "./data/browserwing.db",
			},
			LLMs:      make([]LLMConfig, 0),
			AssetsDir: "./data",
			Browser:   browserCfg,
			Log: &logger.LoggerConfig{
				Level: "info",
				File:  "./log/browserwing.log",
			},
		}
		// 如果错误是文件不存在，则将defConfig写到本地的path位置
		if os.IsNotExist(err) {
			cfgData, err := toml.Marshal(defConfig)
			if err == nil {
				os.WriteFile(path, cfgData, 0o644)
			}
		}
		return defConfig, nil
	}

	var cfg Config
	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	// 确保所有必需的配置项都有值
	if cfg.Browser == nil {
		cfg.Browser = &BrowserConfig{}
	}
	if cfg.Log == nil {
		cfg.Log = &logger.LoggerConfig{
			Level: "info",
		}
	}

	// 兼容处理：如果没有配置 LLMs 数组，但配置了单个 LLM，则转换为数组
	if len(cfg.LLMs) == 0 && cfg.LLM != nil {
		cfg.LLMs = []LLMConfig{*cfg.LLM}
		// 确保有名称
		if cfg.LLMs[0].Name == "" {
			cfg.LLMs[0].Name = "default"
		}
	}

	// 从环境变量覆盖API Key
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		if cfg.LLM != nil {
			cfg.LLM.APIKey = apiKey
		}
		// 也更新到 LLMs 数组的第一个
		if len(cfg.LLMs) > 0 {
			cfg.LLMs[0].APIKey = apiKey
		}
	}

	return &cfg, nil
}

// GetLLMConfig 根据名称获取 LLM 配置
func (c *Config) GetLLMConfig(name string) *LLMConfig {
	if len(c.LLMs) == 0 {
		return c.LLM
	}

	for i := range c.LLMs {
		if c.LLMs[i].Name == name {
			return &c.LLMs[i]
		}
	}

	// 如果没找到，返回第一个
	return &c.LLMs[0]
}

// ListLLMs 返回所有可用的 LLM 配置
func (c *Config) ListLLMs() []LLMConfig {
	if len(c.LLMs) > 0 {
		return c.LLMs
	}
	if c.LLM != nil {
		return []LLMConfig{*c.LLM}
	}
	return []LLMConfig{}
}
