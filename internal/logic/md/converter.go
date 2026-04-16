package md

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cicbyte/daxe/internal/models"
)

// ConvertResult 转换结果
type ConvertResult struct {
	FilePath      string        // 文件路径
	Success       bool          // 是否成功
	Error         error         // 错误信息
	Duration      time.Duration // 处理耗时
	WorkerID      int           // 工作协程ID
	ConvertedCount int          // 转换的问答对数量
	OutputPath    string        // 输出路径
}

// MDConverter JSON转MD转换器
type MDConverter struct {
	config       *MDConfig
	aiConfig     *models.AppConfig
	progressBar  *ProgressBar
	convertedMap sync.Map // 已转换文件映射
	fileLocks    sync.Map // 文件锁映射
}

// NewMDConverter 创建MD转换器
func NewMDConverter(config *MDConfig, aiConfig *models.AppConfig) *MDConverter {
	return &MDConverter{
		config:      config,
		aiConfig:    aiConfig,
		progressBar: &ProgressBar{},
	}
}

// ConvertFiles 转换文件（主入口）
func (c *MDConverter) ConvertFiles(ctx context.Context) error {
	// 获取文件列表
	filePaths, err := c.getFilePaths()
	if err != nil {
		return fmt.Errorf("获取文件列表失败: %w", err)
	}

	if len(filePaths) == 0 {
		fmt.Println("没有找到需要转换的JSON文件")
		return nil
	}

	fmt.Printf("📄 找到 %d 个JSON文件需要转换\n", len(filePaths))

	// 根据线程数选择处理方式
	if c.config.ThreadCount == 1 {
		return c.convertFilesSequentially(ctx, filePaths)
	} else {
		return c.convertFilesConcurrently(ctx, filePaths)
	}
}

// getFilePaths 获取需要转换的文件路径列表
func (c *MDConverter) getFilePaths() ([]string, error) {
	var filePaths []string

	if c.config.InputListPath != "" {
		// 从列表文件加载
		paths, err := c.loadPathsFromList(c.config.InputListPath)
		if err != nil {
			return nil, fmt.Errorf("从列表文件加载路径失败: %w", err)
		}
		filePaths = paths
	} else if c.config.InputPath != "" {
		// 直接路径处理
		paths, err := c.getPathsFromPath(c.config.InputPath)
		if err != nil {
			return nil, fmt.Errorf("从路径获取文件失败: %w", err)
		}
		filePaths = paths
	} else {
		return nil, fmt.Errorf("必须指定输入路径或列表文件")
	}

	// 验证文件存在且为JSON文件
	validPaths := make([]string, 0, len(filePaths))
	for _, path := range filePaths {
		if c.isValidJSONFile(path) {
			// 检查是否已转换过
			if !c.isConvertedFile(path) {
				validPaths = append(validPaths, path)
			}
		}
	}

	return validPaths, nil
}

// loadPathsFromList 从列表文件加载路径
func (c *MDConverter) loadPathsFromList(listPath string) ([]string, error) {
	return LoadFileList(listPath)
}

// getPathsFromPath 从路径获取文件列表
func (c *MDConverter) getPathsFromPath(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("路径访问失败: %w", err)
	}

	if info.IsDir() {
		// 目录：扫描所有JSON文件
		return c.scanJSONFiles(path)
	} else {
		// 单文件
		return []string{path}, nil
	}
}

// scanJSONFiles 扫描目录中的JSON文件
func (c *MDConverter) scanJSONFiles(dirPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".json" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// isValidJSONFile 验证是否为有效的JSON文件
func (c *MDConverter) isValidJSONFile(path string) bool {
	// 检查文件扩展名
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		return false
	}

	// 检查文件是否存在
	_, err := os.Stat(path)
	if err != nil {
		return false
	}

	return true
}

// isConvertedFile 检查文件是否已转换过
func (c *MDConverter) isConvertedFile(path string) bool {
	// 检查内存中的已转换记录
	if _, exists := c.convertedMap.Load(path); exists {
		return true
	}

	return false
}

// convertFilesSequentially 顺序转换文件
func (c *MDConverter) convertFilesSequentially(ctx context.Context, filePaths []string) error {
	fmt.Println("🔄 开始顺序转换文件...")

	successCount := 0
	failCount := 0
	totalConverted := 0

	for i, filePath := range filePaths {
		fmt.Printf("📄 [%d/%d] 转换文件: %s\n", i+1, len(filePaths), filepath.Base(filePath))

		result := c.convertSingleFile(ctx, filePath, 1)
		if result.Success {
			successCount++
			totalConverted += result.ConvertedCount
			fmt.Printf("✅ 转换成功 (%.1fs), 生成 %d 个MD文件\n", result.Duration.Seconds(), result.ConvertedCount)
		} else {
			failCount++
			fmt.Printf("❌ 转换失败: %v\n", result.Error)
		}

		// 标记为已转换
		c.convertedMap.Store(filePath, true)

		if i < len(filePaths)-1 {
			fmt.Println(strings.Repeat("-", 50))
		}
	}

	fmt.Printf("\n📊 转换完成! 成功: %d, 失败: %d, 总生成MD文件: %d\n", successCount, failCount, totalConverted)
	return nil
}

// convertFilesConcurrently 并发转换文件
func (c *MDConverter) convertFilesConcurrently(ctx context.Context, filePaths []string) error {
	fmt.Printf("🚀 开始并发转换文件 (线程数: %d)...\n", c.config.ThreadCount)

	// 初始化进度条
	c.progressBar.Total = len(filePaths)
	c.progressBar.StartTime = time.Now()
	c.progressBar.Start()

	// 创建任务通道
	jobs := make(chan string, len(filePaths))
	results := make(chan ConvertResult, len(filePaths))

	// 添加任务
	for _, filePath := range filePaths {
		jobs <- filePath
	}
	close(jobs)

	// 启动工作协程
	var wg sync.WaitGroup
	wg.Add(c.config.ThreadCount)

	for i := 0; i < c.config.ThreadCount; i++ {
		go func(workerID int) {
			defer wg.Done()
			c.worker(ctx, workerID, jobs, results)
		}(i + 1)
	}

	// 启动结果收集
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	successCount := 0
	failCount := 0
	totalConverted := 0

	for result := range results {
		if result.Success {
			successCount++
			totalConverted += result.ConvertedCount
		} else {
			failCount++
		}

		c.progressBar.Increment()
	}

	c.progressBar.Finish()

	elapsed := time.Since(c.progressBar.StartTime)
	fmt.Printf("\n📊 并发转换完成! 成功: %d, 失败: %d, 总生成MD文件: %d, 耗时: %.1fs\n",
		successCount, failCount, totalConverted, elapsed.Seconds())

	return nil
}

// worker 工作协程
func (c *MDConverter) worker(ctx context.Context, workerID int, jobs <-chan string, results chan<- ConvertResult) {
	for filePath := range jobs {
		// 使用文件锁确保每个文件只被处理一次
		if !c.acquireFileLock(filePath) {
			// 文件已被其他worker锁定，跳过
			continue
		}

		result := c.convertSingleFile(ctx, filePath, workerID)
		results <- result

		// 标记为已转换
		c.convertedMap.Store(filePath, true)

		// 释放文件锁
		c.releaseFileLock(filePath)
	}
}

// convertSingleFile 转换单个文件
func (c *MDConverter) convertSingleFile(ctx context.Context, filePath string, workerID int) ConvertResult {
	startTime := time.Now()

	result := ConvertResult{
		FilePath: filePath,
		WorkerID: workerID,
		Success:  true,
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// 读取JSON文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("读取JSON文件失败: %w", err)
		return result
	}

	// 解析JSON
	var qaList QAList
	if err := json.Unmarshal(content, &qaList); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("JSON解析失败: %w", err)
		return result
	}

	if len(qaList.QAItems) == 0 {
		result.Success = false
		result.Error = fmt.Errorf("JSON文件中没有问答对数据")
		return result
	}

	// 确定输出目录
	outputDir := c.getOutputDir(filePath)

	// 转换为MD文件
	convertedCount, outputPath, err := c.convertToMD(qaList, outputDir)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("转换为MD失败: %w", err)
		return result
	}

	result.ConvertedCount = convertedCount
	result.OutputPath = outputPath
	return result
}

// getOutputDir 获取输出目录
func (c *MDConverter) getOutputDir(inputPath string) string {
	if c.config.OutputDir != "" {
		return c.config.OutputDir
	}

	// 默认在输入文件同目录创建子目录
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	return filepath.Join(filepath.Dir(inputPath), baseName)
}

// convertToMD 转换为MD文件
func (c *MDConverter) convertToMD(qaList QAList, outputDir string) (int, string, error) {
	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 按分类分组
	categorizedItems := make(map[string][]QAItem)
	for _, qa := range qaList.QAItems {
		category := c.sanitizeCategory(qa.Category)
		categorizedItems[category] = append(categorizedItems[category], qa)
	}

	totalConverted := 0

	// 为每个分类创建目录和MD文件
	for category, items := range categorizedItems {
		// 创建分类目录
		categoryDir := filepath.Join(outputDir, category)
		if err := os.MkdirAll(categoryDir, 0755); err != nil {
			return 0, "", fmt.Errorf("创建分类目录失败: %w", err)
		}

		// 为每个问答对创建MD文件
		for i, qa := range items {
			filename := fmt.Sprintf("%03d_%s.md", i+1, c.sanitizeFilename(qa.Question))
			filePath := filepath.Join(categoryDir, filename)

			// 生成MD内容
			mdContent := c.generateMarkdownContent(qa)

			// 原子性写入文件
			if err := c.writeFileAtomically(filePath, mdContent); err != nil {
				return 0, "", fmt.Errorf("写入MD文件失败: %w", err)
			}

			totalConverted++
		}
	}

	return totalConverted, outputDir, nil
}

// sanitizeCategory 清理分类名
func (c *MDConverter) sanitizeCategory(category string) string {
	if category == "" {
		return "未分类"
	}

	// 清理文件系统不允许的字符
	category = strings.ReplaceAll(category, "/", "_")
	category = strings.ReplaceAll(category, "\\", "_")
	category = strings.ReplaceAll(category, ":", "_")
	category = strings.ReplaceAll(category, "*", "_")
	category = strings.ReplaceAll(category, "?", "_")
	category = strings.ReplaceAll(category, "\"", "_")
	category = strings.ReplaceAll(category, "<", "_")
	category = strings.ReplaceAll(category, ">", "_")
	category = strings.ReplaceAll(category, "|", "_")

	return strings.TrimSpace(category)
}

// sanitizeFilename 清理文件名
func (c *MDConverter) sanitizeFilename(filename string) string {
	// 清理文件系统不允许的字符
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "*", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")
	filename = strings.ReplaceAll(filename, "\n", "_")
	filename = strings.ReplaceAll(filename, "\r", "_")

	// 限制文件名长度
	if len(filename) > 100 {
		filename = filename[:100]
	}

	return strings.TrimSpace(filename)
}

// generateMarkdownContent 生成Markdown内容
func (c *MDConverter) generateMarkdownContent(qa QAItem) string {
	var difficultyStars string
	for i := 0; i < qa.Difficulty; i++ {
		difficultyStars += "⭐"
	}

	var tags string
	if len(qa.Tags) > 0 {
		tags = "**标签**: " + strings.Join(qa.Tags, ", ") + "\n"
	}

	return fmt.Sprintf(`# %s

**分类**: %s
**难度**: %s (%d/5)
%s

## 答案

%s

---

*由 daxe 自动转换生成*
`, qa.Question, qa.Category, difficultyStars, qa.Difficulty, tags, qa.Answer)
}

// acquireFileLock 获取文件锁
func (c *MDConverter) acquireFileLock(filePath string) bool {
	// 先检查是否已经转换过
	if _, converted := c.convertedMap.Load(filePath); converted {
		return false
	}

	// 使用LoadOrStore实现原子性的锁获取
	_, loaded := c.fileLocks.LoadOrStore(filePath, true)
	if loaded {
		// 文件已被锁定
		return false
	}
	return true
}

// releaseFileLock 释放文件锁
func (c *MDConverter) releaseFileLock(filePath string) {
	c.fileLocks.Delete(filePath)
}

// writeFileAtomically 原子性写入文件
func (c *MDConverter) writeFileAtomically(filePath, content string) error {
	return WriteFileAtomically(filePath, content)
}