//go:build embed
// +build embed

package main

import (
	"embed"
	"io/fs"
)

//go:embed dist
var frontendFS embed.FS

// GetFrontendFS 返回嵌入的前端文件系统
func GetFrontendFS() (fs.FS, error) {
	return fs.Sub(frontendFS, "dist")
}

// IsEmbedMode 返回是否为嵌入模式
func IsEmbedMode() bool {
	return true
}
