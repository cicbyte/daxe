package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cicbyte/daxe/internal/log"
	"go.uber.org/zap"
)

// BackupFile 备份文件到本地backup目录
// filePath: 要备份的文件路径
// 返回：备份文件路径和错误信息
func BackupFile(filePath string) (string, error) {
	// 获取文件的绝对路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("获取文件绝对路径失败: %w", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在: %s", absPath)
	}

	// 创建backup目录
	backupDir := filepath.Join(filepath.Dir(absPath), "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("创建backup目录失败: %w", err)
	}

	// 生成备份文件名：原文件名_时间戳.扩展名
	fileName := filepath.Base(absPath)
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("%s_%s%s", baseName, timestamp, ext)
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// 复制文件
	if err := copyFile(absPath, backupFilePath); err != nil {
		return "", fmt.Errorf("复制文件失败: %w", err)
	}

	log.Info("文件备份完成",
		zap.String("原文件", absPath),
		zap.String("备份文件", backupFilePath))

	return backupFilePath, nil
}

// copyFile 复制文件内容
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// 复制文件权限
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}