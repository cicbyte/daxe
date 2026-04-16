/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package pdf

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ==================== PDF命令参数验证函数 ====================

// validateImagesParams 验证images子命令参数
func validateImagesParams(inputFiles []string, outputDir, pageRange, imageFormat string, quality, threads int) error {
	// 验证输入文件
	if len(inputFiles) == 0 {
		return fmt.Errorf("必须指定PDF文件")
	}

	// 验证输出目录
	if outputDir == "" {
		return fmt.Errorf("必须指定输出目录")
	}

	// 验证线程数参数
	if threads < 1 || threads > 100 {
		return fmt.Errorf("并发线程数必须在1-100之间")
	}

	// 验证图片格式
	if imageFormat != "" {
		validFormats := map[string]bool{
			"png":  true,
			"jpeg": true,
			"jpg":  true,
		}

		format := strings.ToLower(imageFormat)
		if !validFormats[format] {
			return fmt.Errorf("不支持的图片格式: %s，支持的格式: png, jpeg, jpg", imageFormat)
		}
	}

	// 验证JPEG质量
	if (imageFormat == "jpeg" || imageFormat == "jpg") &&
		(quality < 1 || quality > 100) {
		return fmt.Errorf("JPEG质量必须在1-100之间")
	}

	// 验证页面范围
	if err := validatePageRangeFormat(pageRange); err != nil {
		return fmt.Errorf("页面范围验证失败: %w", err)
	}

	return nil
}

// validatePageRangeFormat 验证页面范围格式
func validatePageRangeFormat(pageRange string) error {
	if pageRange == "" || pageRange == "all" {
		return nil
	}

	parts := strings.Split(pageRange, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return fmt.Errorf("无效的页面范围格式: %s", part)
			}

			var startNum, endNum int
			_, err := fmt.Sscanf(rangeParts[0], "%d", &startNum)
			if err != nil || startNum < 1 {
				return fmt.Errorf("无效的起始页: %s", rangeParts[0])
			}

			_, err = fmt.Sscanf(rangeParts[1], "%d", &endNum)
			if err != nil || endNum < 1 {
				return fmt.Errorf("无效的结束页: %s", rangeParts[1])
			}

			if startNum > endNum {
				return fmt.Errorf("起始页不能大于结束页: %s", part)
			}
		} else {
			var pageNum int
			_, err := fmt.Sscanf(part, "%d", &pageNum)
			if err != nil || pageNum < 1 {
				return fmt.Errorf("无效的页面号: %s", part)
			}
		}
	}

	return nil
}

// validatePDFFiles 验证PDF文件列表
func validatePDFFiles(inputFiles []string) error {
	if len(inputFiles) == 0 {
		return fmt.Errorf("必须指定PDF文件")
	}

	for _, file := range inputFiles {
		if err := validatePDFFile(file); err != nil {
			return fmt.Errorf("PDF文件验证失败 '%s': %w", file, err)
		}
	}

	return nil
}

// validatePDFFile 验证单个PDF文件
func validatePDFFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("PDF文件路径不能为空")
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".pdf" {
		return fmt.Errorf("文件必须是PDF格式，当前格式: %s", ext)
	}

	return nil
}

// validateOutputDir 验证输出目录
func validateOutputDir(outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("输出目录不能为空")
	}

	// 检查路径是否包含非法字符
	if strings.ContainsAny(outputDir, "<>:\"|?*") {
		return fmt.Errorf("输出目录包含非法字符")
	}

	// 如果是相对路径，可以转换为绝对路径验证
	if !filepath.IsAbs(outputDir) {
		_, err := filepath.Abs(outputDir)
		if err != nil {
			return fmt.Errorf("输出目录路径无效: %w", err)
		}
	}

	return nil
}

// validateThreads 验证线程数
func validateThreads(threads int) error {
	if threads < 1 {
		return fmt.Errorf("线程数必须大于0")
	}
	if threads > 100 {
		return fmt.Errorf("线程数不能超过100")
	}
	return nil
}