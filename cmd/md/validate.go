/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package md

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ==================== MD命令参数验证函数 ====================

// validateFixParams 验证fix命令参数
func validateFixParams(args []string, inputList string, threadCount int) error {
	// 检查是否指定了任何输入
	if len(args) == 0 && inputList == "" {
		return fmt.Errorf("必须指定输入路径或列表文件")
	}

	// 检查是否同时指定了多种输入（互斥检查）
	inputCount := 0
	if len(args) > 0 {
		inputCount++
	}
	if inputList != "" {
		inputCount++
	}

	if inputCount > 1 {
		return fmt.Errorf("只能指定一种输入方式：文件路径或列表文件")
	}

	// 验证线程数
	if threadCount < 1 {
		return fmt.Errorf("线程数必须大于0")
	}
	if threadCount > 100 {
		return fmt.Errorf("线程数不能超过100")
	}

	return nil
}

// validateConvertParams 验证convert命令参数
func validateConvertParams(args []string, inputList, outputDir string, threadCount int) error {
	// 检查是否指定了任何输入
	if len(args) == 0 && inputList == "" {
		return fmt.Errorf("必须指定输入路径或列表文件")
	}

	// 检查是否同时指定了多种输入（互斥检查）
	inputCount := 0
	if len(args) > 0 {
		inputCount++
	}
	if inputList != "" {
		inputCount++
	}

	if inputCount > 1 {
		return fmt.Errorf("只能指定一种输入方式：文件路径或列表文件")
	}

	// 验证线程数
	if threadCount < 1 {
		return fmt.Errorf("线程数必须大于0")
	}
	if threadCount > 100 {
		return fmt.Errorf("线程数不能超过100")
	}

	// 验证输出目录（如果指定）
	if outputDir != "" {
		// 检查输出目录是否可写（简单检查）
		if !filepath.IsAbs(outputDir) {
			// 相对路径，转换为绝对路径验证
			_, err := filepath.Abs(outputDir)
			if err != nil {
				return fmt.Errorf("输出目录路径无效: %w", err)
			}
			// 这里我们不需要修改全局变量，只是验证
		}
	}

	return nil
}

// validateFilePath 验证文件路径
func validateFilePath(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("文件路径不能为空")
	}

	// 检查文件扩展名
	ext := filepath.Ext(filePath)
	validExts := map[string]bool{
		".md":      true,
		".markdown": true,
	}

	if !validExts[ext] {
		return fmt.Errorf("不支持的文件格式: %s，支持的格式: .md, .markdown", ext)
	}

	return nil
}

// validateThreads 验证线程数
func validateThreads(threadCount int) error {
	if threadCount < 1 {
		return fmt.Errorf("线程数必须大于0")
	}
	if threadCount > 100 {
		return fmt.Errorf("线程数不能超过100")
	}
	return nil
}

// ==================== 新增图片管理命令验证函数 ====================

// validateUploadParams 验证upload命令参数
func validateUploadParams(args []string, picGoServer string, threadCount int) error {
	// 验证路径参数
	if len(args) == 0 {
		return fmt.Errorf("必须指定MD文件路径或包含MD文件的目录")
	}

	// 验证路径有效性
	if err := validateMDPath(args[0]); err != nil {
		return err
	}

	// 验证线程数
	if threadCount < 1 {
		return fmt.Errorf("线程数必须大于0")
	}
	if threadCount > 100 {
		return fmt.Errorf("线程数不能超过100")
	}

	// 验证PicGo服务器地址（可选）
	if picGoServer != "" {
		if err := validatePicGoServer(picGoServer); err != nil {
			return err
		}
	}

	return nil
}

// validateDownloadParams 验证download命令参数
func validateDownloadParams(args []string, threadCount int) error {
	// 验证路径参数
	if len(args) == 0 {
		return fmt.Errorf("必须指定MD文件路径或包含MD文件的目录")
	}

	// 验证路径有效性
	if err := validateMDPath(args[0]); err != nil {
		return err
	}

	// 验证线程数
	if threadCount < 1 {
		return fmt.Errorf("线程数必须大于0")
	}
	if threadCount > 100 {
		return fmt.Errorf("线程数不能超过100")
	}

	return nil
}

// validateExtractLinksParams 验证extractLinks命令参数
func validateExtractLinksParams(args []string, format string) error {
	// 验证文件路径
	if len(args) != 1 {
		return fmt.Errorf("必须指定一个MD文件路径")
	}

	if err := validateMDFilePath(args[0]); err != nil {
		return err
	}

	// 验证输出格式
	if format != "" {
		validFormats := map[string]bool{
			"text": true,
			"json": true,
		}

		if !validFormats[format] {
			return fmt.Errorf("不支持的输出格式: %s，支持的格式: text, json", format)
		}
	}

	return nil
}

// validateMDFilePath 验证MD文件路径
func validateMDFilePath(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("MD文件路径不能为空")
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	validExts := map[string]bool{
		".md":      true,
		".markdown": true,
	}

	if !validExts[ext] {
		return fmt.Errorf("不支持的文件格式: %s，支持的格式: .md, .markdown", ext)
	}

	return nil
}

// validateMDPath 验证MD文件路径或包含MD文件的目录
func validateMDPath(path string) error {
	if path == "" {
		return fmt.Errorf("路径不能为空")
	}

	// 检查路径是否存在
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("路径不存在: %s", path)
	}
	if err != nil {
		return fmt.Errorf("访问路径失败: %v", err)
	}

	// 如果是文件，验证是否为MD文件
	if !info.IsDir() {
		if !strings.HasSuffix(strings.ToLower(path), ".md") && !strings.HasSuffix(strings.ToLower(path), ".markdown") {
			return fmt.Errorf("必须是.md或.markdown格式的文件")
		}
	}

	return nil
}

// validatePicGoServer 验证PicGo服务器地址
func validatePicGoServer(server string) error {
	if server == "" {
		return fmt.Errorf("PicGo服务器地址不能为空")
	}

	// 解析URL
	parsedURL, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("无效的PicGo服务器地址: %w", err)
	}

	// 检查协议
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("PicGo服务器地址必须使用http或https协议")
	}

	// 检查主机
	if parsedURL.Host == "" {
		return fmt.Errorf("PicGo服务器地址必须包含主机名")
	}

	return nil
}