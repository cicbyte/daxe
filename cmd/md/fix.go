/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package md

import (
	"context"
	"fmt"

	"github.com/cicbyte/daxe/internal/common"
	"github.com/cicbyte/daxe/internal/logic/md"
	"github.com/spf13/cobra"
)

// getFixCommand 返回fix子命令
func getFixCommand() *cobra.Command {
	fixCmd := &cobra.Command{
		Use:   "fix [路径]",
		Short: "修复MD文件中的格式和语法错误",
		Long: `修复MD文件中的格式和语法错误，使用AI进行智能修复。

使用方式:
  daxe md fix ./questions.md                    # 修复单个文件
  daxe md fix ./questions/                      # 批量修复目录下所有MD文件
  daxe md fix -i files.json                     # 从JSON文件列表修复

修复范围:
- Markdown语法错误（标题、列表、链接、代码块等）
- Mermaid图表语法错误
- 格式不一致问题
- 表格语法错误
- 标点符号和编码问题

支持参数:
  -t, --threads N     并发线程数（默认1，最大100）
  -l, --list FILE     从文件列表读取路径

输出:
- 修复后的文件会自动保存，原文件会被覆盖
- 修复日志会显示每个文件的修复情况`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// 参数验证
			if err := validateFixParams(args, fixInputList, fixThreadCount); err != nil {
				fmt.Printf("❌ 参数验证失败: %v\n", err)
				cmd.Help()
				return
			}

			// 设置输入路径
			inputPath := ""
			if len(args) > 0 {
				inputPath = args[0]
			}

			// 创建处理配置
			config := &md.MDConfig{
				ThreadCount:   fixThreadCount,
				InputPath:     inputPath,
				InputListPath: fixInputList,
			}

			fmt.Printf("🔧 开始修复MD文件...\n")

			// 创建修复器
			fixer := md.NewMDFixer(config, common.AppConfigModel)

			// 执行修复
			ctx := context.Background()
			if err := fixer.FixFiles(ctx); err != nil {
				fmt.Printf("❌ 修复失败: %v\n", err)
				return
			}

			fmt.Println("🎉 所有文件修复完成!")
		},
	}

	// 添加命令参数
	fixCmd.Flags().StringVarP(&fixInputList, "list", "l", "", "文件列表路径（支持JSON/TXT格式）")
	fixCmd.Flags().IntVarP(&fixThreadCount, "threads", "t", 1, "并发线程数（默认1，最大100）")

	return fixCmd
}