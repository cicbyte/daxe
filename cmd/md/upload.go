/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package md

import (
	"fmt"

	"github.com/cicbyte/daxe/internal/common"
	"github.com/cicbyte/daxe/internal/logic/md"
	"github.com/spf13/cobra"
)

// 全局变量
var (
	upPicGoServer string
	upThreadCount int
)

// getUploadCommand 返回upload子命令
func getUploadCommand() *cobra.Command {
	uploadCmd := &cobra.Command{
		Use:   "upload <path>",
		Short: "上传MD文件中的本地图片到云存储",
		Long: `上传MD文件中的本地图片到云存储服务。

该命令会：
1. 扫描MD文件中的所有图片链接（支持文件夹批量处理）
2. 筛选出本地图片（非HTTP链接）
3. 并发上传本地图片到PicGo服务
4. 自动替换MD文件中的图片链接为云端地址
5. 创建原文件的时间戳备份

功能特性：
- 支持单文件或文件夹批量处理
- 支持Markdown格式: ![alt](local-image.png)
- HTML格式: <img src="local-image.png">
- 多线程并发上传提高效率
- 自动备份原始文件
- 显示详细的处理统计信息

参数说明：
<path>  MD文件路径或包含MD文件的目录
-t, --threads N     并发线程数（默认1，最大100）
--picgo-server      PicGo服务器地址（默认使用配置文件）

使用方式:
  daxe md upload ./document.md                    # 上传单个文件
  daxe md upload ./documents/                     # 批量处理文件夹中所有MD文件
  daxe md upload ./document.md -t 4               # 使用4个线程并发上传
  daxe md upload ./documents/ --picgo-server http://192.168.1.100:36677  # 指定PicGo服务器

输出:
- 上传成功后会显示处理统计信息
- 原文件会自动备份，文件名包含时间戳
- 失败时会保留处理进度，继续处理其他图片`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// 参数验证
			if err := validateUploadParams(args, upPicGoServer, upThreadCount); err != nil {
				fmt.Printf("❌ 参数验证失败: %v\n", err)
				return
			}

			// 限制最大线程数
			if upThreadCount > 100 {
				upThreadCount = 100
			}
			if upThreadCount < 1 {
				upThreadCount = 1
			}

			// 创建处理配置
			config := &md.UploadConfig{
				FilePath:    args[0],
				PicGoServer: upPicGoServer,
				Threads:     upThreadCount,
			}

			fmt.Printf("🚀 开始上传图片...\n")
			fmt.Printf("📄 路径: %s\n", config.FilePath)
			fmt.Printf("🧵 并发线程: %d\n", config.Threads)
			if upPicGoServer != "" {
				fmt.Printf("🌐 PicGo服务器: %s\n", upPicGoServer)
			}

			// 创建上传处理器
			processor, err := md.NewUploadProcessor(config, common.AppConfigModel)
			if err != nil {
				fmt.Printf("❌ 创建处理器失败: %v\n", err)
				return
			}

			// 执行上传（支持文件夹批量处理）
			result, err := processor.UploadLocalImages()
			if err != nil {
				fmt.Printf("❌ 上传失败: %v\n", err)
				return
			}

			// 只有完全成功（没有失败）时才输出详细结果
			if result.Success && result.Failed == 0 {
				printUploadResult(result)
			} else if result.Failed > 0 {
				// 有部分失败时，显示简短的失败信息
				fmt.Printf("❌ 上传失败: %d 个图片上传失败\n", result.Failed)
			}
		},
	}

	// 参数定义
	uploadCmd.Flags().StringVar(&upPicGoServer, "picgo-server", "", "PicGo服务器地址（默认使用配置文件）")
	uploadCmd.Flags().IntVarP(&upThreadCount, "threads", "t", 1, "并发线程数（默认1，最大100）")

	return uploadCmd
}

// printUploadResult 打印上传结果
func printUploadResult(result *md.UploadResult) {
	fmt.Printf("\n📊 上传完成!\n")
	fmt.Printf("📁 处理文件: %s\n", result.FilePath)
	fmt.Printf("🔗 总链接数: %d\n", result.TotalLinks)
	fmt.Printf("📁 本地图片: %d\n", result.LocalLinks)
	fmt.Printf("✅ 成功上传: %d\n", result.Uploaded)
	fmt.Printf("❌ 失败数量: %d\n", result.Failed)
	fmt.Printf("⏱️  耗时: %.2f 秒\n", result.Duration)

	if result.Uploaded > 0 {
		fmt.Printf("💡 原文件已自动备份\n")
		fmt.Printf("✨ MD文件中的本地图片链接已替换为云端地址\n")
	}

	if result.Failed > 0 {
		fmt.Printf("⚠️  部分图片上传失败，请检查日志\n")
	}
}