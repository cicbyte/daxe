package cmd

import (
	"fmt"

	"github.com/cicbyte/daxe/internal/common"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			commit := common.GitCommit
			if len(commit) > 7 {
				commit = commit[:7]
			}
			fmt.Printf("daxe v%s (%s) %s\n", common.Version, commit, common.BuildTime)
		},
	}
}
