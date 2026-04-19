package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cicbyte/daxe/internal/log"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"go.uber.org/zap"
)

// ==================== 数据类型定义 ====================

// SplitMode 拆分模式
type SplitMode int

const (
	ModeRange SplitMode = iota // 按页码范围
	ModeEvery                  // 按数量拆分
	ModePage                   // 提取单页
)

// SplitConfig PDF拆分配置
type SplitConfig struct {
	InputFile string
	OutputDir string
	Mode      SplitMode
	Pages     string // 页码范围字符串
	Every     int    // 每 N 页
	Page      int    // 单页页码
	Prefix    string // 输出文件名前缀
}

// SplitResult PDF拆分结果
type SplitResult struct {
	Success     bool
	InputFile   string
	OutputFiles []string
	TotalPages  int
	FileCount   int
	Duration    float64
}

// ==================== 处理器 ====================

// SplitProcessor PDF拆分处理器
type SplitProcessor struct {
	config *SplitConfig
}

// NewSplitProcessor 创建拆分处理器
func NewSplitProcessor(config *SplitConfig) *SplitProcessor {
	return &SplitProcessor{config: config}
}

// Execute 执行PDF拆分
func (p *SplitProcessor) Execute(ctx context.Context) (*SplitResult, error) {
	startTime := time.Now()
	result := &SplitResult{
		InputFile: p.config.InputFile,
	}

	defer func() {
		result.Duration = time.Since(startTime).Seconds()
	}()

	// 打开源PDF（使用 pdfcpu）
	pdfCtx, err := api.ReadContextFile(p.config.InputFile)
	if err != nil {
		return nil, fmt.Errorf("无法打开PDF文件: %w", err)
	}

	totalPages := pdfCtx.PageCount
	result.TotalPages = totalPages

	// 创建输出目录
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 确定文件名前缀
	baseName := strings.TrimSuffix(filepath.Base(p.config.InputFile), filepath.Ext(p.config.InputFile))
	if p.config.Prefix != "" {
		baseName = p.config.Prefix
	}

	// 根据模式执行拆分
	switch p.config.Mode {
	case ModeRange:
		if err := p.splitByRange(pdfCtx, totalPages, baseName, result); err != nil {
			return nil, err
		}
	case ModeEvery:
		if err := p.splitByEvery(pdfCtx, totalPages, baseName, result); err != nil {
			return nil, err
		}
	case ModePage:
		if err := p.splitByPage(pdfCtx, totalPages, baseName, result); err != nil {
			return nil, err
		}
	}

	result.FileCount = len(result.OutputFiles)
	result.Success = true
	return result, nil
}

// splitByRange 按页码范围拆分
func (p *SplitProcessor) splitByRange(pdfCtx *model.Context, totalPages int, baseName string, result *SplitResult) error {
	pages, err := ParsePageRange(p.config.Pages, totalPages)
	if err != nil {
		return fmt.Errorf("解析页面范围失败: %w", err)
	}

	outputPath := filepath.Join(p.config.OutputDir, baseName+"_split.pdf")
	if err := p.writePages(pdfCtx, pages, outputPath); err != nil {
		return fmt.Errorf("生成PDF失败: %w", err)
	}

	result.OutputFiles = append(result.OutputFiles, outputPath)
	return nil
}

// splitByEvery 按数量拆分
func (p *SplitProcessor) splitByEvery(pdfCtx *model.Context, totalPages int, baseName string, result *SplitResult) error {
	every := p.config.Every
	for start := 1; start <= totalPages; start += every {
		end := start + every - 1
		if end > totalPages {
			end = totalPages
		}

		// 生成本组页码列表
		pages := make([]int, 0, end-start+1)
		for i := start; i <= end; i++ {
			pages = append(pages, i)
		}

		// 文件名: baseName_001.pdf, baseName_002.pdf, ...
		seq := (start-1)/every + 1
		outputPath := filepath.Join(p.config.OutputDir, fmt.Sprintf("%s_%03d.pdf", baseName, seq))

		if err := p.writePages(pdfCtx, pages, outputPath); err != nil {
			return fmt.Errorf("生成第%d卷失败: %w", seq, err)
		}

		result.OutputFiles = append(result.OutputFiles, outputPath)
	}

	return nil
}

// splitByPage 提取单页
func (p *SplitProcessor) splitByPage(pdfCtx *model.Context, totalPages int, baseName string, result *SplitResult) error {
	page := p.config.Page
	if page < 1 || page > totalPages {
		return fmt.Errorf("页码 %d 超出范围 (总页数: %d)", page, totalPages)
	}

	outputPath := filepath.Join(p.config.OutputDir, fmt.Sprintf("%s_page_%03d.pdf", baseName, page))
	if err := p.writePages(pdfCtx, []int{page}, outputPath); err != nil {
		return fmt.Errorf("生成PDF失败: %w", err)
	}

	result.OutputFiles = append(result.OutputFiles, outputPath)
	return nil
}

// writePages 将指定页码写入新PDF文件（使用 pdfcpu）
func (p *SplitProcessor) writePages(pdfCtx *model.Context, pages []int, outputPath string) error {
	newCtx, err := pdfcpu.ExtractPages(pdfCtx, pages, false)
	if err != nil {
		return fmt.Errorf("提取页面失败: %w", err)
	}

	if err := api.WriteContextFile(newCtx, outputPath); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	log.Info("生成PDF文件", zap.String("path", outputPath), zap.Int("pages", len(pages)))
	return nil
}

// ==================== 结果输出 ====================

// PrintSplitResult 打印拆分结果
func PrintSplitResult(result *SplitResult) {
	fmt.Printf("\n拆分完成!\n")
	fmt.Printf("输入文件: %s\n", result.InputFile)
	fmt.Printf("总页数: %d\n", result.TotalPages)
	fmt.Printf("生成文件数: %d\n", result.FileCount)
	fmt.Printf("耗时: %.2f 秒\n", result.Duration)

	if len(result.OutputFiles) > 0 {
		fmt.Printf("\n输出文件:\n")
		for _, f := range result.OutputFiles {
			if info, err := os.Stat(f); err == nil {
				fmt.Printf("  %s (%s)\n", f, formatFileSize(info.Size()))
			} else {
				fmt.Printf("  %s\n", f)
			}
		}
	}
}
