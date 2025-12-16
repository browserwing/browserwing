//go:build !embed
// +build !embed

package main

import (
	"io/fs"
	"os"
)

// GetFrontendFS 开发模式下返回本地文件系统
func GetFrontendFS() (fs.FS, error) {
	// 开发模式下不提供静态文件服务，使用 Vite 开发服务器
	return os.DirFS("../frontend/dist"), nil
}

// IsEmbedMode 返回是否为嵌入模式
func IsEmbedMode() bool {
	return false
}
