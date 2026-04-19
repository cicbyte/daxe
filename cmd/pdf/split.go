package pdf

import (
	"context"
	"fmt"

	"github.com/cicbyte/daxe/internal/logic/pdf"
	"github.com/spf13/cobra"
)

// PDF split 命令的全局变量
var (
	pdfSplitOutput string
	pdfSplitPages  string
	pdfSplitEvery  int
	pdfSplitPage   int
	pdfSplitPrefix string
)

// getSplitCommand 返回split子命令
func getSplitCommand() *cobra.Command {
	splitCmd := &cobra.Command{
		Use:   "split <pdf_file>",
		Short: "拆分PDF文件",
		Long: `将PDF文件按指定策略拆分为多个PDF文件。

支持三种拆分模式（互斥）:
  --pages  按页码范围提取指定页面为一个PDF
  --every  按固定页数拆分为多个PDF
  --page   提取单个页面为一个PDF

页面范围格式：
  单页：1, 3, 5
  范围：1-5, 10-20
  混合：1, 3-5, 10

示例:
  daxe pdf split document.pdf -o ./output --pages 1,3,5-8
  daxe pdf split document.pdf -o ./output --every 10
  daxe pdf split document.pdf -o ./output --page 5
  daxe pdf split document.pdf -o ./output --every 10 --prefix chapter`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := validateSplitParams(args[0], pdfSplitOutput, pdfSplitPages, pdfSplitEvery, pdfSplitPage); err != nil {
				fmt.Printf("❌ 参数验证失败: %v\n", err)
				return
			}

			// 确定拆分模式
			var mode pdf.SplitMode
			switch {
			case pdfSplitPages != "":
				mode = pdf.ModeRange
			case pdfSplitEvery > 0:
				mode = pdf.ModeEvery
			case pdfSplitPage > 0:
				mode = pdf.ModePage
			}

			config := &pdf.SplitConfig{
				InputFile: args[0],
				OutputDir: pdfSplitOutput,
				Mode:      mode,
				Pages:     pdfSplitPages,
				Every:     pdfSplitEvery,
				Page:      pdfSplitPage,
				Prefix:    pdfSplitPrefix,
			}

			processor := pdf.NewSplitProcessor(config)
			result, err := processor.Execute(context.Background())
			if err != nil {
				fmt.Printf("❌ 拆分失败: %v\n", err)
				return
			}

			pdf.PrintSplitResult(result)
		},
	}

	// 添加命令参数
	splitCmd.Flags().StringVarP(&pdfSplitOutput, "output", "o", "", "输出目录路径 (必需)")
	splitCmd.MarkFlagRequired("output")

	splitCmd.Flags().StringVar(&pdfSplitPages, "pages", "", "按页码范围提取 (如: 1,3,5-8)")
	splitCmd.Flags().IntVar(&pdfSplitEvery, "every", 0, "按固定页数拆分 (如: 10 表示每10页一个文件)")
	splitCmd.Flags().IntVar(&pdfSplitPage, "page", 0, "提取单页 (如: 5 表示提取第5页)")
	splitCmd.Flags().StringVar(&pdfSplitPrefix, "prefix", "", "输出文件名前缀 (默认使用源文件名)")

	return splitCmd
}
