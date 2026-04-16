package md

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cicbyte/daxe/internal/common"
	"github.com/cicbyte/daxe/internal/log"
	"github.com/cicbyte/daxe/internal/models"
	"github.com/cicbyte/daxe/internal/utils"
	"go.uber.org/zap"
)

// MDFixer MD文件修复器
type MDFixer struct {
	config       *MDConfig
	aiConfig     *models.AppConfig
	progressBar  *ProgressBar
	processedMap sync.Map // 已处理文件映射
	fileLocks    sync.Map // 文件锁映射
}

// FixResult 修复结果
type FixResult struct {
	FilePath      string        // 文件路径
	Success       bool          // 是否成功
	Error         error         // 错误信息
	Duration      time.Duration // 处理耗时
	WorkerID      int           // 工作协程ID
	FixedCount    int           // 修复的问题数量
}

// NewMDFixer 创建MD修复器
func NewMDFixer(config *MDConfig, aiConfig *models.AppConfig) *MDFixer {
	return &MDFixer{
		config:      config,
		aiConfig:    aiConfig,
		progressBar: &ProgressBar{},
	}
}

// FixFiles 修复文件（主入口）
func (p *MDFixer) FixFiles(ctx context.Context) error {
	// 验证AI配置
	if err := utils.ValidateAIConfig(p.aiConfig); err != nil {
		return fmt.Errorf("AI配置验证失败: %w", err)
	}

	// 获取文件列表
	filePaths, err := p.getFilePaths()
	if err != nil {
		return fmt.Errorf("获取文件列表失败: %w", err)
	}

	if len(filePaths) == 0 {
		fmt.Println("没有找到需要修复的MD文件")
		return nil
	}

	fmt.Printf("📄 找到 %d 个MD文件需要修复\n", len(filePaths))

	fmt.Printf("🤖 修复模式: %d个文件, %d线程\n",
		len(filePaths), p.config.ThreadCount)

	// 根据线程数选择处理方式
	if p.config.ThreadCount == 1 {
		return p.fixFilesSequentially(ctx, filePaths)
	} else {
		return p.fixFilesConcurrently(ctx, filePaths)
	}
}

// getFilePaths 获取需要处理的文件路径列表
func (p *MDFixer) getFilePaths() ([]string, error) {
	var filePaths []string

	if p.config.InputListPath != "" {
		// 从列表文件加载
		paths, err := p.loadPathsFromList(p.config.InputListPath)
		if err != nil {
			return nil, fmt.Errorf("从列表文件加载路径失败: %w", err)
		}
		filePaths = paths
	} else if p.config.InputPath != "" {
		// 直接路径处理
		paths, err := p.getPathsFromPath(p.config.InputPath)
		if err != nil {
			return nil, fmt.Errorf("从路径获取文件失败: %w", err)
		}
		filePaths = paths
	} else {
		return nil, fmt.Errorf("必须指定输入路径或列表文件")
	}

	// 验证文件存在且为MD文件
	validPaths := make([]string, 0, len(filePaths))
	for _, path := range filePaths {
		if p.isValidMDFile(path) {
			validPaths = append(validPaths, path)
		}
	}

	return validPaths, nil
}

// loadPathsFromList 从列表文件加载路径
func (p *MDFixer) loadPathsFromList(listPath string) ([]string, error) {
	return LoadFileList(listPath)
}

// getPathsFromPath 从路径获取文件列表
func (p *MDFixer) getPathsFromPath(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("路径访问失败: %w", err)
	}

	if info.IsDir() {
		// 目录：扫描所有MD文件
		return p.scanMDFiles(path)
	} else {
		// 单文件
		return []string{path}, nil
	}
}

// scanMDFiles 扫描目录中的MD文件
func (p *MDFixer) scanMDFiles(dirPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".md" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// isValidMDFile 验证是否为有效的MD文件
func (p *MDFixer) isValidMDFile(path string) bool {
	// 检查文件扩展名
	if strings.ToLower(filepath.Ext(path)) != ".md" {
		return false
	}

	// 检查文件是否存在
	_, err := os.Stat(path)
	if err != nil {
		return false
	}

	return true
}

// fixFilesSequentially 顺序修复文件
func (p *MDFixer) fixFilesSequentially(ctx context.Context, filePaths []string) error {
	fmt.Println("🔄 开始顺序修复文件...")

	successCount := 0
	failCount := 0

	for i, filePath := range filePaths {
		fmt.Printf("📄 [%d/%d] 修复文件: %s\n", i+1, len(filePaths), filepath.Base(filePath))

		result := p.fixSingleFile(ctx, filePath, 1)
		if result.Success {
			successCount++
			fmt.Printf("✅ 修复成功，修复问题数: %d (%.1fs)\n", result.FixedCount, result.Duration.Seconds())
		} else {
			failCount++
			fmt.Printf("❌ 修复失败: %v\n", result.Error)
		}

		// 标记为已处理
		p.processedMap.Store(filePath, true)

		if i < len(filePaths)-1 {
			fmt.Println(strings.Repeat("-", 50))
		}
	}

	fmt.Printf("\n📊 修复完成! 成功: %d, 失败: %d\n", successCount, failCount)
	return nil
}

// fixFilesConcurrently 并发修复文件
func (p *MDFixer) fixFilesConcurrently(ctx context.Context, filePaths []string) error {
	fmt.Printf("🚀 开始并发修复文件 (线程数: %d)...\n", p.config.ThreadCount)

	// 初始化进度条
	p.progressBar.Total = len(filePaths)
	p.progressBar.StartTime = time.Now()
	p.progressBar.Start()

	// 创建任务通道
	jobs := make(chan string, len(filePaths))
	results := make(chan FixResult, len(filePaths))

	// 添加任务
	for _, filePath := range filePaths {
		jobs <- filePath
	}
	close(jobs)

	// 启动工作协程
	var wg sync.WaitGroup
	wg.Add(p.config.ThreadCount)

	for i := 0; i < p.config.ThreadCount; i++ {
		go func(workerID int) {
			defer wg.Done()
			p.worker(ctx, workerID, jobs, results)
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
	totalFixed := 0

	for result := range results {
		if result.Success {
			successCount++
			totalFixed += result.FixedCount
		} else {
			failCount++
		}

		p.progressBar.Increment()
	}

	p.progressBar.Finish()

	elapsed := time.Since(p.progressBar.StartTime)
	fmt.Printf("\n📊 并发修复完成! 成功: %d, 失败: %d, 总修复: %d, 耗时: %.1fs\n",
		successCount, failCount, totalFixed, elapsed.Seconds())

	return nil
}

// worker 工作协程
func (p *MDFixer) worker(ctx context.Context, workerID int, jobs <-chan string, results chan<- FixResult) {
	for filePath := range jobs {
		// 使用文件锁确保每个文件只被处理一次
		if !p.acquireFileLock(filePath) {
			// 文件已被其他worker锁定，跳过
			continue
		}

		result := p.fixSingleFile(ctx, filePath, workerID)
		results <- result

		// 标记为已处理
		p.processedMap.Store(filePath, true)

		// 释放文件锁
		p.releaseFileLock(filePath)
	}
}

// fixSingleFile 修复单个文件
func (p *MDFixer) fixSingleFile(ctx context.Context, filePath string, workerID int) FixResult {
	startTime := time.Now()

	result := FixResult{
		FilePath: filePath,
		WorkerID: workerID,
		Success:  true,
	}

	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("读取文件失败: %w", err)
		return result
	}

	originalContent := string(content)
	if originalContent == "" {
		result.Success = false
		result.Error = fmt.Errorf("文件内容为空")
		return result
	}

	// 使用AI进行全量修复
	fixedContent, fixedCount := p.fixWithAI(originalContent)

	// 如果没有修复内容，直接返回
	if fixedCount == 0 {
		result.FixedCount = 0
		return result
	}

	// 原子性写入修复后的内容
	if err := p.writeFileAtomically(filePath, fixedContent); err != nil {
		result.Success = false
		result.Error = fmt.Errorf("写入文件失败: %w", err)
		return result
	}

	result.FixedCount = fixedCount
	return result
}

// fixWithAI 使用AI进行修复
func (p *MDFixer) fixWithAI(content string) (string, int) {
	// 从嵌入的文件系统加载提示词
	promptContent, err := common.GetAssetFile("assets/prompts/fix.md")
	if err != nil {
		log.Error("加载修复提示词失败", zap.Error(err))
		return content, 0
	}

	// 构建系统提示词
	systemPrompt := string(promptContent)

	// 构建用户提示词
	userPrompt := fmt.Sprintf(`请修复以下MD文件中的格式和语法错误：

%s

**重要提醒：必须主动进行修复，不要因为内容看起来可读就跳过修复！**

请分析并修复以下问题：
1. **Markdown语法错误**：标题格式、列表格式、代码块格式等
2. **HTML标签清理**：将所有HTML标签转换为对应的Markdown语法或移除
3. **格式不一致**：统一格式，提高可读性
4. **链接格式问题**：修复链接语法，将HTML链接转换为Markdown链接
5. **列表格式问题**：修复列表项格式，移除HTML包装
6. **代码块格式问题**：为代码块添加语言标识符，转换HTML代码标签
7. **表格格式问题**：将HTML表格转换为Markdown表格格式
8. **Mermaid图表语法错误**：修复流程图语法
9. **样式清理**：移除所有内联样式、class、id属性
10. **注释和脚本清理**：移除HTML注释、script标签、style标签
11. **空白字符优化**：移除多余空行和空格

**修复要求：**
- 发现HTML标签必须转换为Markdown或移除
- 发现格式错误必须修复
- 发现样式问题必须清理
- 不要保留原有的HTML格式

请只返回修复后的完整内容，不要添加额外的解释。`, content)

	// 调用AI进行修复
	ctx := context.Background()
	fixedContent, err := utils.CallAISync(ctx, p.aiConfig, systemPrompt, userPrompt)
	if err != nil {
		log.Error("AI修复失败", zap.Error(err))
		return content, 0
	}

	// 计算修复数量（更智能的比较）
	var fixedCount int
	if fixedContent != content {
		// 内容有变化，至少修复了1个问题
		fixedCount = 1

		// 统计可能修复的问题类型
		originalLower := strings.ToLower(content)
		fixedLower := strings.ToLower(fixedContent)

		if strings.Contains(originalLower, "<") && !strings.Contains(fixedLower, "<div") {
			fixedCount += 2 // HTML标签清理
		}
		if strings.Contains(content, "#") && strings.Contains(fixedContent, "# ") {
			fixedCount += 1 // 标题格式修复
		}
		if strings.Contains(content, "-") && strings.Contains(fixedContent, "- ") {
			fixedCount += 1 // 列表格式修复
		}
		if strings.Contains(originalLower, "```") && !strings.Contains(fixedLower, "```\n```") {
			fixedCount += 1 // 代码块修复
		}
		if strings.Contains(originalLower, "<a href") || strings.Contains(originalLower, "<img") {
			fixedCount += 1 // 链接/图片格式修复
		}
		if strings.Contains(originalLower, "<table") || strings.Contains(originalLower, "<script") {
			fixedCount += 2 // 表格/脚本清理
		}
	} else {
		fixedCount = 0
	}

	return fixedContent, fixedCount
}

// acquireFileLock 获取文件锁
func (p *MDFixer) acquireFileLock(filePath string) bool {
	// 先检查是否已经处理过
	if _, processed := p.processedMap.Load(filePath); processed {
		return false
	}

	// 使用LoadOrStore实现原子性的锁获取
	_, loaded := p.fileLocks.LoadOrStore(filePath, true)
	if loaded {
		// 文件已被锁定
		return false
	}
	return true
}

// releaseFileLock 释放文件锁
func (p *MDFixer) releaseFileLock(filePath string) {
	p.fileLocks.Delete(filePath)
}

// writeFileAtomically 原子性写入文件
func (p *MDFixer) writeFileAtomically(filePath, content string) error {
	return WriteFileAtomically(filePath, content)
}