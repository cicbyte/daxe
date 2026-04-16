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
	dlThreadCount int
)

// getDownloadCommand 返回download子命令
func getDownloadCommand() *cobra.Command {
	downloadCmd := &cobra.Command{
		Use:   "download <path>",
		Short: "下载MD文件中的远程图片到本地",
		Long: `下载MD文件中的远程图片到本地存储。

该命令会：
1. 扫描MD文件中的所有图片链接
2. 筛选出远程图片（HTTP/HTTPS/FTP链接）
3. 下载远程图片到本地同名目录
4. 自动替换MD文件中的图片链接为本地路径
5. 创建URL到本地文件名的映射关系

本地目录结构：
- document.md
- document/            # 图片目录
  - abc123.png        # 下载的图片
  - def456.jpg
  - link_mappings.json # 链接映射文件

文件命名规则：
- 使用原始URL的MD5哈希值作为文件名
- 保留原始文件扩展名

使用方式:
  daxe md download ./document.md                    # 下载单个文件

输出:
- 下载成功后会显示处理统计信息
- 图片保存在MD文件同名的子目录中
- 生成link_mappings.json文件记录URL映射关系`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// 参数验证
			if err := validateDownloadParams(args, dlThreadCount); err != nil {
				fmt.Printf("❌ 参数验证失败: %v\n", err)
				return
			}

			// 限制最大线程数
			if dlThreadCount > 100 {
				dlThreadCount = 100
			}
			if dlThreadCount < 1 {
				dlThreadCount = 1
			}

			// 创建处理配置
			config := &md.DownloadConfig{
				FilePath: args[0],
				Threads:  dlThreadCount,
			}

			fmt.Printf("🚀 开始下载图片...\n")
			fmt.Printf("📄 路径: %s\n", config.FilePath)
			fmt.Printf("🧵 并发线程: %d\n", config.Threads)

			// 创建下载处理器
			processor, err := md.NewDownloadProcessor(config, common.AppConfigModel)
			if err != nil {
				fmt.Printf("❌ 创建处理器失败: %v\n", err)
				return
			}

			// 执行下载（支持文件夹批量处理）
			result, err := processor.DownloadRemoteImages()
			if err != nil {
				fmt.Printf("❌ 下载失败: %v\n", err)
				return
			}

			// 只有完全成功（没有失败）时才输出详细结果
			if result.Success && result.Failed == 0 {
				printDownloadResult(result)
			} else if result.Failed > 0 {
				// 有部分失败时，显示简短的失败信息
				fmt.Printf("❌ 下载失败: %d 个图片下载失败\n", result.Failed)
			}
		},
	}

	// 参数定义
	downloadCmd.Flags().IntVarP(&dlThreadCount, "threads", "t", 1, "并发线程数（默认1，最大100）")

	return downloadCmd
}

// printDownloadResult 打印下载结果
func printDownloadResult(result *md.DownloadResult) {
	fmt.Printf("\n📊 下载完成!\n")
	fmt.Printf("📁 处理文件: %s\n", result.FilePath)
	fmt.Printf("🔗 总链接数: %d\n", result.TotalLinks)
	fmt.Printf("🌐 远程图片: %d\n", result.RemoteLinks)
	fmt.Printf("✅ 成功下载: %d\n", result.Downloaded)
	fmt.Printf("❌ 失败数量: %d\n", result.Failed)
	fmt.Printf("⏱️  耗时: %.2f 秒\n", result.Duration)

	if result.Downloaded > 0 {
		fmt.Printf("💡 图片已保存到: images/\n")
		fmt.Printf("💡 原文件已自动备份到 backup/ 目录\n")
		fmt.Printf("✨ MD文件中的远程图片链接已替换为本地路径\n")
		fmt.Printf("📋 链接映射已保存到 link_mappings.json\n")
	}

	if result.Failed > 0 {
		fmt.Printf("⚠️  部分图片下载失败，请检查网络连接\n")
	}
}