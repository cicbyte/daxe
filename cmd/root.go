/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/cicbyte/daxe/cmd/md"
	"github.com/cicbyte/daxe/cmd/pdf"
	"github.com/cicbyte/daxe/cmd/xmind"
	"github.com/cicbyte/daxe/internal/common"
	"github.com/cicbyte/daxe/internal/log"
	"github.com/cicbyte/daxe/internal/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "daxe",
	Short: "daxe",
	Long:  `daxe`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// 初始化应用目录
	if err := utils.InitAppDirs(); err != nil {
		fmt.Printf("初始化目录失败: %v\n", err)
		os.Exit(1)
	}
	// 加载配置(会自动创建默认配置)
	common.AppConfigModel = utils.ConfigInstance.LoadConfig()
	// 初始化日志
	if err := log.Init(utils.ConfigInstance.GetLogPath()); err != nil {
		fmt.Printf("日志初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化数据库连接
	if _, err := utils.GetGormDB(); err != nil {
		log.Error("数据库连接失败",
			zap.String("operation", "db init"),
			zap.Error(err))
		os.Exit(1)
	}
	log.Info("数据库连接成功")

	// 添加version命令
	rootCmd.AddCommand(newVersionCmd())

	// 添加MD命令
	rootCmd.AddCommand(md.GetMDCommand())

	// 添加PDF命令
	rootCmd.AddCommand(pdf.GetPDFCommand())

	// 添加XMind命令
	rootCmd.AddCommand(xmind.GetXMindCommand())
}
