/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package pdf

import (
	"context"
	"fmt"
	"os"

	"github.com/cicbyte/daxe/internal/logic/pdf"
	"github.com/spf13/cobra"
)

// getImagesCommand 返回images子命令
func getImagesCommand() *cobra.Command {
	imagesCmd := &cobra.Command{
		Use:   "images <pdf_file|pattern>",
		Short: "从PDF文件提取图片",
		Long: `从PDF文件中提取所有图片并保存到指定目录。

支持的图片格式：
- PNG (默认)
- JPEG (可设置质量)

页面范围格式：
- 单页：1, 3, 5
- 范围：1-5, 10-20
- 混合：1, 3-5, 10

示例:
  daxe pdf images document.pdf -o ./images
  daxe pdf images document.pdf -o ./images --pages 1-5
  daxe pdf images document.pdf -o ./images --format jpeg --quality 95
  daxe pdf images "*.pdf" -o ./images --batch`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// 参数验证
			if err := validateImagesParams(args, pdfImagesOutput, pdfImagesPages, pdfImagesFormat, pdfImagesQuality, pdfImagesThreads); err != nil {
				fmt.Printf("❌ 参数验证失败: %v\n", err)
				return
			}

			// 直接创建处理器配置
			config := &pdf.ImagesConfig{
				InputFiles:  args,
				OutputDir:   pdfImagesOutput,
				PageRange:   pdfImagesPages,
				ImageFormat: pdfImagesFormat,
				Quality:     pdfImagesQuality,
				PageDirs:    pdfImagesPageDirs,
				Overwrite:   pdfImagesOverwrite,
				Batch:       pdfImagesBatch,
				Quiet:       pdfImagesQuiet,
				Threads:     pdfImagesThreads,
			}

			// 直接创建处理器并执行
			processor := pdf.NewImagesProcessor(config)
			ctx := context.Background()
			if _, err := processor.Execute(ctx); err != nil {
				fmt.Printf("错误: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// 添加命令参数
	imagesCmd.Flags().StringVarP(&pdfImagesOutput, "output", "o", "", "输出目录路径 (必需)")
	imagesCmd.MarkFlagRequired("output")

	imagesCmd.Flags().StringVar(&pdfImagesPages, "pages", "", "页面范围 (如: 1,3,5-8)，默认所有页面")
	imagesCmd.Flags().StringVar(&pdfImagesFormat, "format", "png", "图片格式 (png, jpeg, jpg)")
	imagesCmd.Flags().IntVar(&pdfImagesQuality, "quality", 90, "JPEG图片质量 (1-100)")
	imagesCmd.Flags().BoolVar(&pdfImagesPageDirs, "page-dirs", false, "为每个页面创建独立子目录（默认使用平铺结构）")
	imagesCmd.Flags().BoolVar(&pdfImagesOverwrite, "overwrite", false, "覆盖已存在的图片文件")
	imagesCmd.Flags().BoolVar(&pdfImagesBatch, "batch", false, "批量处理模式 (支持文件通配符)")
	imagesCmd.Flags().BoolVarP(&pdfImagesQuiet, "quiet", "q", false, "静默模式，减少输出信息")
	imagesCmd.Flags().IntVarP(&pdfImagesThreads, "threads", "t", 1, "并发线程数（默认1，最大100）")

	return imagesCmd
}