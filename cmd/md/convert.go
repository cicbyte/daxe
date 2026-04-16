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

// getConvertCommand 返回convert子命令
func getConvertCommand() *cobra.Command {
	convertCmd := &cobra.Command{
		Use:   "convert [路径]",
		Short: "将JSON格式的问答对转换为MD结构",
		Long: `将JSON格式的问答对转换为MD结构，按分类组织输出。

使用方式:
  daxe md convert ./qa.json                        # 转换单个JSON文件
  daxe md convert ./qa_files/                      # 转换目录下所有JSON文件
  daxe md convert ./qa.json -o ./output            # 转换并输出到指定目录
  daxe md convert -i qa_list.json                  # 从JSON文件列表转换
  daxe md convert -i qa_list.txt                   # 从TXT文件列表转换

转换功能:
- 将JSON格式的问答对转换为Markdown格式
- 按分类自动组织输出结构
- 生成可读性强的MD文件
- 支持批量转换和并发处理
- 自动扫描目录下的所有JSON文件
- 递归处理子目录中的JSON文件

输入文件格式:
- TXT: 每行一个JSON文件路径
- JSON: 字符串数组格式，如 ["/path/to/file1.json", "/path/to/file2.json"]

支持参数:
  -t, --threads N     并发线程数（默认1，最大100）
  -l, --list FILE     从文件列表读取JSON文件路径
  -o, --output DIR    指定输出目录

输出:
- 生成按分类组织的MD文件结构
- 每个分类创建独立目录
- 每个问答对生成独立MD文件
- 文件名包含序号和问题标题`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// 参数验证
			if err := validateConvertParams(args, convInputList, convOutputDir, convThreadCount); err != nil {
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
				ThreadCount:   convThreadCount,
				InputPath:     inputPath,
				InputListPath: convInputList,
				OutputDir:     convOutputDir,
			}

			fmt.Printf("🔄 开始转换JSON文件...\n")

			// 创建转换器
			converter := md.NewMDConverter(config, common.AppConfigModel)

			// 执行转换
			ctx := context.Background()
			if err := converter.ConvertFiles(ctx); err != nil {
				fmt.Printf("❌ 转换失败: %v\n", err)
				return
			}

			fmt.Println("🎉 所有文件转换完成!")
		},
	}

	// 添加命令参数
	convertCmd.Flags().StringVarP(&convInputList, "list", "l", "", "JSON文件列表路径（支持JSON/TXT格式）")
	convertCmd.Flags().IntVarP(&convThreadCount, "threads", "t", 1, "并发线程数（默认1，最大100）")
	convertCmd.Flags().StringVarP(&convOutputDir, "output", "o", "", "指定输出目录")

	return convertCmd
}