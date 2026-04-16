/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package md

import (
	"encoding/json"
	"fmt"

	"github.com/cicbyte/daxe/internal/common"
	"github.com/cicbyte/daxe/internal/logic/md"
	"github.com/spf13/cobra"
)

// 全局变量
var (
	exFilePath string
	exFormat   string // 输出格式：text/json
)

// getExtractLinksCommand 返回extractLinks子命令
func getExtractLinksCommand() *cobra.Command {
	extractLinksCmd := &cobra.Command{
		Use:   "extractLinks <file>",
		Short: "提取MD文件中的所有图片链接",
		Long: `提取MD文件中的所有图片链接，支持本地和远程图片。

该命令会：
1. 扫描MD文件中的所有图片链接
2. 识别本地图片和远程图片
3. 显示每个图片的详细信息
4. 支持不同格式的输出

支持格式：
- Markdown格式: ![alt](src)
- HTML格式: <img src="src">

链接类型：
- [Local] 本地图片文件
- [Remote] 远程图片URL

使用方式:
  daxe md extractLinks ./document.md                    # 文本格式输出
  daxe md extractLinks ./document.md --format json      # JSON格式输出

输出格式:
- text: 人类可读的文本格式
- json: 结构化JSON格式，便于程序处理

信息包含：
- 原始链接（文件中的内容）
- 解码后的链接
- 链接类型（本地/远程）
- 本地文件的绝对路径`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// 参数验证
			if err := validateExtractLinksParams(args, exFormat); err != nil {
				fmt.Printf("❌ 参数验证失败: %v\n", err)
				return
			}

			// 创建处理配置
			config := &md.ExtractLinksConfig{
				FilePath: args[0],
			}

			fmt.Printf("🔍 开始提取图片链接...\n")
			fmt.Printf("📄 文件: %s\n", config.FilePath)

			// 创建提取处理器
			processor, err := md.NewExtractLinksProcessor(config, common.AppConfigModel)
			if err != nil {
				fmt.Printf("❌ 创建处理器失败: %v\n", err)
				return
			}

			// 执行提取
			result, err := processor.ExtractLinks()
			if err != nil {
				fmt.Printf("❌ 提取失败: %v\n", err)
				return
			}

			// 只有成功时才输出结果
			if result.Success {
				if exFormat == "json" {
					printExtractLinksResultJSON(result)
				} else {
					printExtractLinksResultText(result)
				}
			}
		},
	}

	// 添加命令参数
	extractLinksCmd.Flags().StringVarP(&exFormat, "format", "f", "text", "输出格式：text或json")

	return extractLinksCmd
}

// printExtractLinksResultText 打印文本格式的提取结果
func printExtractLinksResultText(result *md.ExtractLinksResult) {
	fmt.Printf("\n📊 提取完成!\n")
	fmt.Printf("📁 处理文件: %s\n", result.FilePath)
	fmt.Printf("🔗 总链接数: %d\n", result.TotalLinks)
	fmt.Printf("📁 本地图片: %d\n", result.LocalLinks)
	fmt.Printf("🌐 远程图片: %d\n", result.RemoteLinks)
	fmt.Printf("⏱️  耗时: %.2f 秒\n", result.Duration)

	if result.TotalLinks > 0 {
		fmt.Printf("\n📋 图片链接详情:\n")
		fmt.Printf("─" + string(make([]rune, 60)) + "─\n")

		for i, link := range result.ImageLinks {
			fmt.Printf("%2d. ", i+1)

			if link.IsRemote {
				fmt.Printf("[Remote] %s\n", link.Decoded)
			} else {
				fmt.Printf("[Local] %s\n", link.Decoded)
				fmt.Printf("        Abs: %s\n", link.AbsPath)
			}
		}
	}
}

// printExtractLinksResultJSON 打印JSON格式的提取结果
func printExtractLinksResultJSON(result *md.ExtractLinksResult) {
	// 转换为更友好的JSON输出
	jsonResult := map[string]interface{}{
		"success":       result.Success,
		"file_path":     result.FilePath,
		"statistics": map[string]interface{}{
			"total_links":  result.TotalLinks,
			"local_links":  result.LocalLinks,
			"remote_links": result.RemoteLinks,
			"duration":     result.Duration,
		},
		"image_links": result.ImageLinks,
	}

	jsonData, err := json.MarshalIndent(jsonResult, "", "  ")
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}