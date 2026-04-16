/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package md

import (
	"github.com/spf13/cobra"
)

// MD命令的全局变量定义
var (
	fixInputList     string
	fixThreadCount   int
	convInputList    string
	convThreadCount  int
	convOutputDir    string
)

// GetMDCommand 返回MD主命令，用于在cmd/root.go中注册
func GetMDCommand() *cobra.Command {
	// 创建MD主命令
	mdCmd := &cobra.Command{
		Use:   "md",
		Short: "Markdown文件处理工具",
		Long: `处理Markdown文件的专业工具集。

支持的操作：
- fix: 修复MD文件中的格式和语法错误
- convert: 将JSON格式的问答对转换为MD结构
- upload: 上传MD文件中的本地图片到云存储
- download: 下载MD文件中的远程图片到本地
- extractLinks: 提取MD文件中的所有图片链接

使用方式:
  daxe md fix ./file.md                          # 修复MD文件格式错误
  daxe md convert ./qa.json                       # 将JSON转换为MD结构
  daxe md convert ./qa_files/                     # 转换目录下所有JSON文件
  daxe md convert ./qa.json -o ./output           # 转换并输出到指定目录
  daxe md upload ./document.md                    # 上传本地图片到云存储
  daxe md download ./document.md                  # 下载远程图片到本地
  daxe md extractLinks ./document.md              # 提取所有图片链接
  daxe md extractLinks ./document.md -f json      # 以JSON格式提取链接

支持参数:
  -t, --threads N     并发线程数（默认1，最大100）
  -l, --list FILE     从文件列表读取路径
  -o, --output DIR    指定输出目录（仅convert命令）
  -f, --format FMT   输出格式：json或md（仅extractLinks命令）
  --picgo-server     PicGo服务器地址（仅upload命令）`,
	}

	// 添加所有子命令
	mdCmd.AddCommand(getFixCommand())
	mdCmd.AddCommand(getConvertCommand())
	mdCmd.AddCommand(getUploadCommand())
	mdCmd.AddCommand(getDownloadCommand())
	mdCmd.AddCommand(getExtractLinksCommand())

	return mdCmd
}