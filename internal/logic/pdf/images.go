package pdf

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cicbyte/daxe/internal/log"
	"github.com/cicbyte/daxe/internal/utils"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
	"go.uber.org/zap"
)

// ==================== 数据类型定义 ====================

// ExtractionResult 图片提取结果
type ExtractionResult struct {
	Success bool `json:"success"`
	Error   string `json:"error,omitempty"`
	InputPath string `json:"input_path"`
	OutputDir  string `json:"output_dir"`
	Images     []ImageInfo `json:"images"`
	TotalPages   int `json:"total_pages"`
	PagesProcessed []int `json:"pages_processed"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  float64   `json:"duration"`
}

// ImageInfo 提取的单个图片信息
type ImageInfo struct {
	FilePath      string    `json:"file_path"`
	PageNumber    int       `json:"page_number"`
	ImageIndex    int       `json:"image_index"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	Format        string    `json:"format"`
	FileSize      int64     `json:"file_size"`
	ExtractionTime time.Time `json:"extraction_time"`
}

// ImageExtractionStats 图片提取统计信息
type ImageExtractionStats struct {
	TotalImages   int            `json:"total_images"`
	SuccessCount  int            `json:"success_count"`
	FailureCount  int            `json:"failure_count"`
	TotalFileSize int64          `json:"total_file_size"`
	ImagesPerPage map[int]int    `json:"images_per_page"`
	ImagesByFormat map[string]int `json:"images_by_format"`
}

// GetStats 计算提取统计信息
func (r *ExtractionResult) GetStats() *ImageExtractionStats {
	stats := &ImageExtractionStats{
		ImagesPerPage:  make(map[int]int),
		ImagesByFormat: make(map[string]int),
	}

	for _, img := range r.Images {
		stats.TotalImages++
		stats.TotalFileSize += img.FileSize

		// 按页面统计
		stats.ImagesPerPage[img.PageNumber]++

		// 按格式统计
		format := img.Format
		if format == "" {
			format = "unknown"
		}
		stats.ImagesByFormat[format]++
	}

	stats.SuccessCount = stats.TotalImages
	stats.FailureCount = 0

	return stats
}

// ==================== 配置类型定义 ====================

// ImageExtractionConfig PDF图片提取配置
type ImageExtractionConfig struct {
	InputPath     string `json:"input_path"`
	OutputDir     string `json:"output_dir"`
	PageRange     string `json:"page_range,omitempty"`
	ImageFormat   string `json:"image_format"`
	Quality       int    `json:"quality,omitempty"`
	CreatePageDirs bool  `json:"create_page_dirs"`
	Overwrite     bool   `json:"overwrite"`
	Threads       int    `json:"threads"`
}

// DefaultImageExtractionConfig 返回默认配置
func DefaultImageExtractionConfig() *ImageExtractionConfig {
	return &ImageExtractionConfig{
		ImageFormat:    "png",
		Quality:        90,
		CreatePageDirs: false,
		Overwrite:      false,
		Threads:        1,
	}
}

// ImagesConfig PDF图片提取命令配置
type ImagesConfig struct {
	InputFiles  []string
	OutputDir   string
	PageRange   string
	ImageFormat string
	Quality     int
	PageDirs    bool
	Overwrite   bool
	Batch       bool
	Quiet       bool
	Threads     int
}

// ImagesResult PDF图片提取结果
type ImagesResult struct {
	Results []*ExtractionResult
	Summary *BatchSummary
}

// BatchSummary 批量处理汇总信息
type BatchSummary struct {
	TotalFiles   int
	SuccessFiles int
	FailedFiles  int
	TotalImages  int
	TotalSize    int64
	ProcessTime  float64
}

// ==================== 核心提取器 ====================

// ImageExtractor PDF图片提取器
type ImageExtractor struct {
	config *ImageExtractionConfig
}

// NewImageExtractor 创建图片提取器
func NewImageExtractor(config *ImageExtractionConfig) *ImageExtractor {
	return &ImageExtractor{
		config: config,
	}
}

// Extract 执行图片提取
func (e *ImageExtractor) Extract() *ExtractionResult {
	result := &ExtractionResult{
		InputPath: e.config.InputPath,
		OutputDir: e.config.OutputDir,
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	}()

	// 配置验证已在CMD层完成

	// 打开PDF文件
	file, err := os.Open(e.config.InputPath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("无法打开PDF文件: %v", err)
		return result
	}
	defer file.Close()

	// 创建PDF阅读器
	pdfReader, err := model.NewPdfReader(file)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("创建PDF阅读器失败: %v", err)
		return result
	}

	// 获取总页数
	totalPages, err := pdfReader.GetNumPages()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("获取PDF页数失败: %v", err)
		return result
	}
	result.TotalPages = totalPages

	// 解析页面范围
	targetPages, err := ParsePageRange(e.config.PageRange, totalPages)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("解析页面范围失败: %v", err)
		return result
	}
	result.PagesProcessed = targetPages

	// 创建输出目录
	if err := os.MkdirAll(e.config.OutputDir, 0755); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("创建输出目录失败: %v", err)
		return result
	}

	// 根据线程数选择处理方式
	if e.config.Threads == 1 {
		return e.extractSequentially(pdfReader, targetPages, result)
	} else {
		return e.extractConcurrently(pdfReader, targetPages, result)
	}
}

// extractSequentially 顺序提取图片
func (e *ImageExtractor) extractSequentially(pdfReader *model.PdfReader, targetPages []int, result *ExtractionResult) *ExtractionResult {
	var mu sync.Mutex // 顺序模式下也需要传入mutex参数
	for _, pageNum := range targetPages {
		if err := e.extractPageImages(pdfReader, pageNum, result, &mu); err != nil {
			log.Error("提取页面图片失败", zap.Int("page", pageNum), zap.Error(err))
			continue
		}
	}

	result.Success = true
	return result
}

// extractConcurrently 并发提取图片
func (e *ImageExtractor) extractConcurrently(pdfReader *model.PdfReader, targetPages []int, result *ExtractionResult) *ExtractionResult {
	var mutex sync.Mutex
	jobs := make(chan int, len(targetPages))

	// 添加任务
	for _, pageNum := range targetPages {
		jobs <- pageNum
	}
	close(jobs)

	// 启动工作协程
	var wg sync.WaitGroup
	wg.Add(e.config.Threads)

	for i := 0; i < e.config.Threads; i++ {
		go func() {
			defer wg.Done()
			for pageNum := range jobs {
				if err := e.extractPageImages(pdfReader, pageNum, result, &mutex); err != nil {
					log.Error("提取页面图片失败", zap.Int("page", pageNum), zap.Error(err))
				}
			}
		}()
	}

	wg.Wait()

	// 对结果排序
	mutex.Lock()
	sort.Slice(result.Images, func(i, j int) bool {
		if result.Images[i].PageNumber != result.Images[j].PageNumber {
			return result.Images[i].PageNumber < result.Images[j].PageNumber
		}
		return result.Images[i].ImageIndex < result.Images[j].ImageIndex
	})
	mutex.Unlock()

	result.Success = true
	return result
}

// extractPageImages 提取单个页面的图片
func (e *ImageExtractor) extractPageImages(pdfReader *model.PdfReader, pageNum int, result *ExtractionResult, mu *sync.Mutex) error {
	// 获取页面
	page, err := pdfReader.GetPage(pageNum)
	if err != nil {
		return fmt.Errorf("获取页面%d失败: %w", pageNum, err)
	}

	// 创建页面提取器
	pageExtractor, err := extractor.New(page)
	if err != nil {
		return fmt.Errorf("创建页面%d提取器失败: %w", pageNum, err)
	}

	// 提取页面图片
	extractionResult, err := pageExtractor.ExtractPageImages(nil)
	if err != nil {
		return fmt.Errorf("提取页面%d图片失败: %w", pageNum, err)
	}

	// 处理提取到的图片
	for i, extractedImage := range extractionResult.Images {
		if err := e.saveExtractedImage(&extractedImage, pageNum, i, result, mu); err != nil {
			log.Error("保存图片失败", zap.Int("page", pageNum), zap.Int("index", i), zap.Error(err))
			continue
		}
	}

	return nil
}

// saveExtractedImage 保存提取的图片
func (e *ImageExtractor) saveExtractedImage(extractedImg *extractor.ImageMark, pageNum, imageIndex int, result *ExtractionResult, mu *sync.Mutex) error {
	// 转换为Go标准图像格式
	goImage, err := extractedImg.Image.ToGoImage()
	if err != nil {
		return fmt.Errorf("转换第%d张图片失败: %w", imageIndex+1, err)
	}

	// 生成文件名
	filename := fmt.Sprintf("img_%03d_page_%03d.%s", len(result.Images)+1, pageNum, e.config.ImageFormat)

	var filePath string
	if e.config.CreatePageDirs {
		// 创建页面子目录
		pageDir := filepath.Join(e.config.OutputDir, fmt.Sprintf("page_%03d", pageNum))
		if err := os.MkdirAll(pageDir, 0755); err != nil {
			return fmt.Errorf("创建页面目录失败: %w", err)
		}
		filePath = filepath.Join(pageDir, filename)
	} else {
		// 平铺目录结构
		filePath = filepath.Join(e.config.OutputDir, filename)
	}

	// 检查文件是否存在
	if !e.config.Overwrite {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("文件已存在: %s", filePath)
		}
	}

	// 编码为指定格式
	var buf bytes.Buffer
	var imgFormat string

	switch strings.ToLower(e.config.ImageFormat) {
	case "png":
		err = png.Encode(&buf, goImage)
		imgFormat = "png"
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, goImage, &jpeg.Options{Quality: e.config.Quality})
		imgFormat = "jpeg"
	default:
		return fmt.Errorf("不支持的图片格式: %s", e.config.ImageFormat)
	}

	if err != nil {
		return fmt.Errorf("编码图片失败: %w", err)
	}

	// 保存文件
	if err := e.writeFileAtomically(filePath, buf.Bytes()); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	// 记录图片信息（使用mutex保护）
	mu.Lock()
	result.Images = append(result.Images, ImageInfo{
		FilePath:      filePath,
		PageNumber:    pageNum,
		ImageIndex:    imageIndex,
		Width:         goImage.Bounds().Dx(),
		Height:        goImage.Bounds().Dy(),
		Format:        imgFormat,
		FileSize:      int64(buf.Len()),
		ExtractionTime: time.Now(),
	})
	mu.Unlock()

	return nil
}

// ==================== 处理器 ====================

// ImagesProcessor PDF图片提取处理器
type ImagesProcessor struct {
	config *ImagesConfig
}

// NewImagesProcessor 创建图片提取处理器
func NewImagesProcessor(config *ImagesConfig) *ImagesProcessor {
	return &ImagesProcessor{
		config: config,
	}
}

// Execute 执行图片提取（主入口）
func (p *ImagesProcessor) Execute(ctx context.Context) (*ImagesResult, error) {
	if p.config.Batch {
		return p.executeBatch(ctx)
	} else {
		return p.executeSingle(ctx)
	}
}

// executeSingle 执行单文件处理
func (p *ImagesProcessor) executeSingle(ctx context.Context) (*ImagesResult, error) {
	if len(p.config.InputFiles) == 0 {
		return nil, fmt.Errorf("必须指定输入PDF文件")
	}

	pdfFile := p.config.InputFiles[0]

	// 检查文件是否存在
	if _, err := os.Stat(pdfFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("PDF文件不存在: %s", pdfFile)
	}

	if !p.config.Quiet {
		log.Info("开始处理PDF文件", zap.String("file", pdfFile))
	}

	// 提取图片
	result, err := p.extractImages(pdfFile)
	if err != nil {
		return nil, err
	}

	// 输出结果
	if !p.config.Quiet {
		p.printSingleResult(result)
	}

	return &ImagesResult{
		Results: []*ExtractionResult{result},
	}, nil
}

// executeBatch 执行批量处理
func (p *ImagesProcessor) executeBatch(ctx context.Context) (*ImagesResult, error) {
	startTime := time.Now()

	// 解析文件通配符
	pdfFiles, err := p.resolveFilePatterns(p.config.InputFiles)
	if err != nil {
		return nil, fmt.Errorf("解析文件模式失败: %w", err)
	}

	if len(pdfFiles) == 0 {
		return nil, fmt.Errorf("未找到任何PDF文件")
	}

	if !p.config.Quiet {
		log.Info("开始批量处理", zap.Int("files", len(pdfFiles)))
	}

	// 批量提取
	results := make([]*ExtractionResult, 0, len(pdfFiles))
	for _, pdfFile := range pdfFiles {
		// 为每个PDF文件创建独立的输出目录
		pdfName := filepath.Base(pdfFile)
		pdfName = strings.TrimSuffix(pdfName, filepath.Ext(pdfName))
		outputDir := filepath.Join(p.config.OutputDir, pdfName)

		result, err := p.extractImagesToDir(pdfFile, outputDir)
		if err != nil {
			// 失败也记录结果，继续处理其他文件
			result = &ExtractionResult{
				Success:   false,
				InputPath: pdfFile,
				OutputDir: outputDir,
				Error:     err.Error(),
			}
		}

		results = append(results, result)
	}

	// 计算汇总信息
	summary := p.calculateBatchSummary(results, time.Since(startTime).Seconds())

	// 输出批量处理结果
	if !p.config.Quiet {
		p.printBatchResults(results, summary)
	}

	return &ImagesResult{
		Results: results,
		Summary: summary,
	}, nil
}

// extractImages 提取单个PDF文件的图片到配置的输出目录
func (p *ImagesProcessor) extractImages(pdfFile string) (*ExtractionResult, error) {
	return p.extractImagesToDir(pdfFile, p.config.OutputDir)
}

// extractImagesToDir 提取单个PDF文件的图片到指定目录
func (p *ImagesProcessor) extractImagesToDir(pdfFile, outputDir string) (*ExtractionResult, error) {
	// 使用默认配置
	config := DefaultImageExtractionConfig()
	config.InputPath = pdfFile
	config.OutputDir = outputDir
	config.PageRange = p.config.PageRange
	config.ImageFormat = p.config.ImageFormat
	config.Quality = p.config.Quality
	config.CreatePageDirs = p.config.PageDirs
	config.Overwrite = p.config.Overwrite
	config.Threads = p.config.Threads

	// 创建提取器并执行提取
	extractor := NewImageExtractor(config)
	result := extractor.Extract()

	if !result.Success {
		return result, fmt.Errorf("图片提取失败: %s", result.Error)
	}

	return result, nil
}

// resolveFilePatterns 解析文件通配符
func (p *ImagesProcessor) resolveFilePatterns(patterns []string) ([]string, error) {
	var pdfFiles []string

	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("解析文件模式失败 '%s': %w", pattern, err)
		}

		// 过滤出PDF文件
		for _, file := range files {
			if strings.ToLower(filepath.Ext(file)) == ".pdf" {
				pdfFiles = append(pdfFiles, file)
			}
		}
	}

	return pdfFiles, nil
}

// calculateBatchSummary 计算批量处理汇总信息
func (p *ImagesProcessor) calculateBatchSummary(results []*ExtractionResult, processTime float64) *BatchSummary {
	summary := &BatchSummary{
		TotalFiles: len(results),
	}

	var totalSize int64
	var totalImages int

	for _, result := range results {
		if result.Success {
			summary.SuccessFiles++
			totalImages += len(result.Images)
			stats := result.GetStats()
			totalSize += stats.TotalFileSize
		} else {
			summary.FailedFiles++
		}
	}

	summary.TotalImages = totalImages
	summary.TotalSize = totalSize
	summary.ProcessTime = processTime

	return summary
}

// printSingleResult 打印单文件处理结果
func (p *ImagesProcessor) printSingleResult(result *ExtractionResult) {
	stats := result.GetStats()

	fmt.Printf("\n📊 提取完成!\n")
	fmt.Printf("📁 输入文件: %s\n", result.InputPath)
	fmt.Printf("📁 输出目录: %s\n", result.OutputDir)
	fmt.Printf("📄 总页数: %d\n", result.TotalPages)
	fmt.Printf("📄 处理页数: %d\n", len(result.PagesProcessed))
	fmt.Printf("🖼️  提取图片数: %d\n", len(result.Images))
	fmt.Printf("⏱️  耗时: %.2f 秒\n", result.Duration)

	if len(result.Images) > 0 {
		fmt.Printf("📋 图片格式统计:\n")
		for format, count := range stats.ImagesByFormat {
			fmt.Printf("  %s: %d 张\n", format, count)
		}
		fmt.Printf("💾 总文件大小: %s\n", formatFileSize(stats.TotalFileSize))
	}
}

// printBatchResults 打印批量处理结果
func (p *ImagesProcessor) printBatchResults(results []*ExtractionResult, summary *BatchSummary) {
	fmt.Printf("\n🎉 批量处理完成!\n")
	fmt.Printf("📊 处理文件数: %d\n", summary.TotalFiles)
	fmt.Printf("✅ 成功处理: %d\n", summary.SuccessFiles)
	fmt.Printf("❌ 失败处理: %d\n", summary.FailedFiles)
	fmt.Printf("🖼️  提取图片总数: %d\n", summary.TotalImages)
	fmt.Printf("💾 总文件大小: %s\n", formatFileSize(summary.TotalSize))
	fmt.Printf("⏱️  总耗时: %.2f 秒\n", summary.ProcessTime)

	// 显示失败的文件
	if summary.FailedFiles > 0 {
		fmt.Printf("\n❌ 失败的文件:\n")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  - %s: %s\n", result.InputPath, result.Error)
			}
		}
	}
}

// ==================== 辅助函数 ====================

// formatFileSize 格式化文件大小
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ParsePageRange 解析页面范围字符串
func ParsePageRange(rangeStr string, totalPages int) ([]int, error) {
	if rangeStr == "" || rangeStr == "all" {
		pages := make([]int, totalPages)
		for i := 0; i < totalPages; i++ {
			pages[i] = i + 1
		}
		return pages, nil
	}

	var pages []int
	parts := strings.Split(rangeStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("无效的页面范围格式: %s", part)
			}

			var startNum, endNum int
			_, err := fmt.Sscanf(rangeParts[0], "%d", &startNum)
			if err != nil {
				return nil, fmt.Errorf("无效的起始页: %s", rangeParts[0])
			}

			_, err = fmt.Sscanf(rangeParts[1], "%d", &endNum)
			if err != nil {
				return nil, fmt.Errorf("无效的结束页: %s", rangeParts[1])
			}

			if startNum < 1 || endNum > totalPages || startNum > endNum {
				return nil, fmt.Errorf("页面范围超出总页数: %s (总页数: %d)", part, totalPages)
			}

			for i := startNum; i <= endNum; i++ {
				pages = append(pages, i)
			}
		} else {
			var pageNum int
			_, err := fmt.Sscanf(part, "%d", &pageNum)
			if err != nil || pageNum < 1 || pageNum > totalPages {
				return nil, fmt.Errorf("无效的页面号: %s (总页数: %d)", part, totalPages)
			}
			pages = append(pages, pageNum)
		}
	}

	return pages, nil
}

// writeFileAtomically 原子性写入文件
func (e *ImageExtractor) writeFileAtomically(filePath string, data []byte) error {
	return utils.WriteFileAtomically(filePath, data)
}