package common

import (
	"embed"
	"io/fs"

	"github.com/cicbyte/daxe/internal/models"
)

var (
	AppConfigModel *models.AppConfig
	AssetsFS       embed.FS // 嵌入的资源文件系统
	Version        string
	GitCommit      string
	BuildTime      string
)

// GetAssetFile 获取嵌入的资源文件内容
func GetAssetFile(path string) ([]byte, error) {
	// embed.FS 总是有值的，即使没有嵌入任何文件
	return AssetsFS.ReadFile(path)
}

// AssetExists 检查嵌入的资源文件是否存在
func AssetExists(path string) bool {
	_, err := fs.Stat(AssetsFS, path)
	return err == nil
}
