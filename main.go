/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"embed"

	"github.com/cicbyte/daxe/cmd"
	"github.com/cicbyte/daxe/internal/common"
)

//go:embed assets/*
var AssetsFS embed.FS

// 构建时通过 -ldflags 注入
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	common.Version = Version
	common.GitCommit = GitCommit
	common.BuildTime = BuildTime
	common.AssetsFS = AssetsFS
	cmd.Execute()
}
