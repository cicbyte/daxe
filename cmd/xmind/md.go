/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package xmind

import (
	"fmt"
	"os"

	"github.com/cicbyte/daxe/internal/common"
	"github.com/cicbyte/daxe/internal/logic/xmind"
	"github.com/spf13/cobra"
)

// getXMindMDCommand 返回XMind MD子命令
func getXMindMDCommand() *cobra.Command {
	mdCmd := &cobra.Command{
		Use:   "md [路径]",
		Short: "将XMind文件转换为Markdown格式",
		Long: `将XMind思维导图文件转换为Markdown格式，支持多层级结构转换。

该命令会：
1. 解析XMind文件的content.json内容
2. 递归转换思维导图结构为Markdown
3. 保持层级关系和链接信息
4. 支持标记和注释信息
5. 支持文件夹批量处理和多线程并发
6. 支持从文件列表读取路径

使用方式:
  daxe xmind md mindmap.xmind                        # 转换单个文件
  daxe xmind md mindmap.xmind -o output.md            # 指定输出文件
  daxe xmind md ./xmind_files/                        # 批量处理目录下所有XMind文件
  daxe xmind md ./xmind_files/ -o ./output/           # 批量处理并输出到指定目录
  daxe xmind md ./xmind_files/ -t 4                   # 使用4个线程并发处理
  daxe xmind md -i files.json                         # 从JSON文件列表处理
  daxe xmind md -i files.txt                          # 从TXT文件列表处理
  daxe xmind md -i files.json -o ./output/ -t 4       # 从文件列表处理并输出到指定目录

输入文件格式:
- TXT: 每行一个XMind文件路径或目录路径
- JSON: 字符串数组格式，如 ["/path/to/file1.xmind", "/path/to/dir/"]

输出格式:
- 使用Markdown标题层级（H1-H6）
- 子主题使用列表结构
- 保持原始链接和标记信息
- 文件名保持一致，扩展名改为.md`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// 参数验证
			if err := validateXMindMDParams(args, xmindMdInputList, xmindMdOutput); err != nil {
				fmt.Printf("❌ 参数验证失败: %v\n", err)
				return
			}

			// 限制最大线程数
			if xmindMdThreadCount > 100 {
				xmindMdThreadCount = 100
			}

			// 设置输入路径
			inputPath := ""
			if len(args) > 0 {
				inputPath = args[0]
			}

			// 创建处理配置
			config := &xmind.XMindMDConfig{
				InputPath:     inputPath,
				InputListPath: xmindMdInputList,
				OutputDir:     xmindMdOutput,
				Threads:       xmindMdThreadCount,
			}

			// 创建处理器
			processor, err := xmind.NewXMindMDProcessor(config, common.AppConfigModel)
			if err != nil {
				fmt.Printf("❌ 创建处理器失败: %v\n", err)
				return
			}

			// 执行转换（支持文件夹批量处理和文件列表）
			fmt.Printf("🚀 开始转换XMind文件...\n")
			if inputPath != "" {
				fmt.Printf("📄 路径: %s\n", inputPath)
			}
			if xmindMdInputList != "" {
				fmt.Printf("📋 列表文件: %s\n", xmindMdInputList)
			}
			fmt.Printf("🧵 并发线程: %d\n", xmindMdThreadCount)
			if xmindMdOutput != "" {
				fmt.Printf("📁 输出目录: %s\n", xmindMdOutput)
			}

			result, err := processor.ConvertToMarkdown()
			if err != nil {
				fmt.Printf("❌ 转换失败: %v\n", err)
				return
			}

			// 输出结果
			printXMindMDResult(result)
		},
	}

	// 添加命令参数
	mdCmd.Flags().StringVarP(&xmindMdInputList, "list", "l", "", "文件列表路径（支持JSON/TXT格式）")
	mdCmd.Flags().StringVarP(&xmindMdOutput, "output", "o", "", "输出目录路径（单文件时为输出文件路径）")
	mdCmd.Flags().IntVarP(&xmindMdThreadCount, "threads", "t", 1, "并发线程数（默认1，最大100）")

	return mdCmd
}

// validateXMindMDParams 验证XMind MD命令参数
func validateXMindMDParams(args []string, inputList, outputDir string) error {
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
		return fmt.Errorf("只能指定一种输入方式：路径或列表文件")
	}

	// 验证输入路径（如果指定）
	if len(args) > 0 {
		inputPath := args[0]
		// 检查输入路径是否存在
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			return fmt.Errorf("输入路径不存在: %s", inputPath)
		}
	}

	// 验证输入列表文件（如果指定）
	if inputList != "" {
		// 检查输入列表文件是否存在
		if _, err := os.Stat(inputList); os.IsNotExist(err) {
			return fmt.Errorf("输入列表文件不存在: %s", inputList)
		}
	}

	// 如果指定了输出目录，检查输出目录是否存在
	if outputDir != "" {
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			return fmt.Errorf("输出目录不存在: %s", outputDir)
		}
	}

	return nil
}

// printXMindMDResult 打印转换结果
func printXMindMDResult(result *xmind.XMindMDResult) {
	fmt.Printf("\n📊 转换完成!\n")
	fmt.Printf("📁 输入路径: %s\n", result.InputPath)
	if result.OutputDir != "" {
		fmt.Printf("📁 输出目录: %s\n", result.OutputDir)
	}
	fmt.Printf("📄 文件数量: %d\n", result.FileCount)
	fmt.Printf("✅ 成功文件: %d\n", result.SuccessFiles)
	fmt.Printf("❌ 失败文件: %d\n", result.FailedFiles)
	fmt.Printf("📋 总画布数: %d\n", result.TotalSheets)
	fmt.Printf("🔗 总主题数: %d\n", result.TotalTopics)
	fmt.Printf("⏱️  耗时: %.2f 秒\n", result.Duration)

	if result.SuccessFiles > 0 {
		fmt.Printf("💡 所有成功文件已转换为Markdown格式\n")
	}
}