/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package pdf

import (
	"github.com/spf13/cobra"
)

// PDF命令的全局变量定义
var (
	// PDF图片提取命令参数
	pdfImagesOutput     string
	pdfImagesPages      string
	pdfImagesFormat     string
	pdfImagesQuality    int
	pdfImagesPageDirs   bool
	pdfImagesOverwrite  bool
	pdfImagesBatch      bool
	pdfImagesQuiet      bool
	pdfImagesThreads    int
)

// GetPDFCommand 返回PDF主命令，用于在cmd/root.go中注册
func GetPDFCommand() *cobra.Command {
	// 创建PDF主命令
	pdfCmd := &cobra.Command{
		Use:   "pdf",
		Short: "PDF文件处理工具",
		Long: `PDF文件处理工具集，支持图片提取、文件拆分等功能。

示例:
  daxe pdf images input.pdf -o ./output
  daxe pdf images input.pdf -o ./output --pages 1,3,5-8 --format jpeg --quality 90
  daxe pdf images "*.pdf" -o ./output --batch
  daxe pdf split input.pdf -o ./output --pages 1,3,5-8
  daxe pdf split input.pdf -o ./output --every 10
  daxe pdf split input.pdf -o ./output --page 5`,
	}

	// 添加所有子命令
	pdfCmd.AddCommand(getImagesCommand())
	pdfCmd.AddCommand(getSplitCommand())

	return pdfCmd
}