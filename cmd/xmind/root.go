/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package xmind

import (
	"github.com/spf13/cobra"
)

// XMind命令的全局变量定义
var (
	xmindMdOutput       string
	xmindMdThreadCount  int
	xmindMdInputList    string
)

// GetXMindCommand 返回XMind主命令，用于在cmd/root.go中注册
func GetXMindCommand() *cobra.Command {
	// 创建XMind主命令
	xmindCmd := &cobra.Command{
		Use:   "xmind",
		Short: "XMind思维导图处理工具",
		Long: `处理XMind思维导图文件，支持转换为Markdown格式。

支持的操作：
- md: 将XMind文件转换为Markdown格式

使用方式:
  daxe xmind md ./mindmap.xmind                    # 转换单个文件
  daxe xmind md ./mindmap.xmind -o output.md        # 指定输出文件

支持参数:
  -o, --output FILE   指定输出文件路径`,
	}

	// 添加所有子命令
	xmindCmd.AddCommand(getXMindMDCommand())

	return xmindCmd
}