package xmind

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cicbyte/daxe/internal/log"
	"github.com/cicbyte/daxe/internal/logic/md"
	"github.com/cicbyte/daxe/internal/models"
	"github.com/cicbyte/daxe/internal/utils"
	"go.uber.org/zap"
)

// ==================== 数据类型定义 ====================

// XMindMDConfig XMind转MD配置
type XMindMDConfig struct {
	InputPath     string // 输入路径（文件或目录）
	InputListPath string // 输入列表文件路径
	OutputDir     string // 输出目录路径
	Threads       int    // 并发线程数
}

// XMindMDResult XMind转MD结果
type XMindMDResult struct {
	Success       bool    `json:"success"`
	InputPath     string  `json:"input_path"`
	OutputDir     string  `json:"output_dir"`
	FileCount     int     `json:"file_count"`
	TotalSheets   int     `json:"total_sheets"`
	TotalTopics   int     `json:"total_topics"`
	SuccessFiles  int     `json:"success_files"`
	FailedFiles   int     `json:"failed_files"`
	Duration      float64 `json:"duration"`
	ErrorMessage  string  `json:"error_message,omitempty"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
}

// XMindFileTask XMind文件处理任务
type XMindFileTask struct {
	InputFile  string // 输入XMind文件路径
	OutputFile string // 输出MD文件路径
}

// XMindContent represents the structure of content.json
type XMindContent []struct {
	ID        string `json:"id"`
	Class     string `json:"class"`
	Title     string `json:"title"`
	RootTopic struct {
		ID             string `json:"id"`
		Class          string `json:"class"`
		Title          string `json:"title"`
		Href           string `json:"href"`
		StructureClass string `json:"structureClass"`
		Children       struct {
			Attached []Topic `json:"attached"`
		} `json:"children"`
	} `json:"rootTopic"`
}

type Topic struct {
	Title    string `json:"title"`
	ID       string `json:"id"`
	Href     string `json:"href"`
	Position struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position"`
	Children struct {
		Attached []Topic `json:"attached"`
	} `json:"children"`
	Branch  string `json:"branch"`
	Markers []struct {
		MarkerID string `json:"markerId"`
	} `json:"markers"`
	Summaries []struct {
		Range   string `json:"range"`
		TopicID string `json:"topicId"`
	} `json:"summaries"`
}

// ==================== 构造函数 ====================

// NewXMindMDProcessor 创建XMind转MD处理器
func NewXMindMDProcessor(config *XMindMDConfig, appConfig *models.AppConfig) (*XMindMDProcessor, error) {
	// 获取绝对路径
	absInputPath := ""
	if config.InputPath != "" {
		absPath, err := filepath.Abs(config.InputPath)
		if err != nil {
			return nil, fmt.Errorf("获取输入路径绝对路径失败: %w", err)
		}
		absInputPath = absPath
	}

	absOutputDir := ""
	if config.OutputDir != "" {
		absDir, err := filepath.Abs(config.OutputDir)
		if err != nil {
			return nil, fmt.Errorf("获取输出目录绝对路径失败: %w", err)
		}
		absOutputDir = absDir
	}

	processor := &XMindMDProcessor{
		config:      config,
		appConfig:   appConfig,
		inputPath:   config.InputPath,
		outputDir:   config.OutputDir,
		absInputPath: absInputPath,
		absOutputDir: absOutputDir,
	}

	return processor, nil
}

// XMindMDProcessor XMind转MD处理器
type XMindMDProcessor struct {
	config      *XMindMDConfig
	appConfig   *models.AppConfig
	inputPath   string
	outputDir   string
	absInputPath string
	absOutputDir string
}

// ==================== 核心处理方法 ====================

// ConvertToMarkdown 转换XMind文件为Markdown
func (p *XMindMDProcessor) ConvertToMarkdown() (*XMindMDResult, error) {
	result := &XMindMDResult{
		InputPath: p.inputPath,
		OutputDir: p.outputDir,
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	}()

	// 获取所有需要处理的XMind文件
	xmindFiles, err := p.getXMindFiles()
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("查找XMind文件失败: %v", err)
		return result, err
	}

	if len(xmindFiles) == 0 {
		result.Success = false
		result.ErrorMessage = "没有找到XMind文件"
		return result, fmt.Errorf("没有找到XMind文件")
	}

	log.Info("找到XMind文件", zap.Int("count", len(xmindFiles)), zap.Strings("files", xmindFiles))

	// 创建文件处理任务
	tasks, err := p.createXMindTasks(xmindFiles)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("创建处理任务失败: %v", err)
		return result, err
	}

	// 统计总文件数
	result.FileCount = len(tasks)

	// 执行并发处理
	if p.config.Threads > 1 {
		err = p.processConcurrently(tasks, result)
	} else {
		err = p.processSequentially(tasks, result)
	}

	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	result.Success = true
	return result, nil
}

// ==================== 辅助方法 ====================


// printTopic 递归打印主题
func (p *XMindMDProcessor) printTopic(topic Topic, level int, output *os.File) {
	// 打印带缩进的主题标题
	fmt.Fprintf(output, "%s- ", p.getIndent(level))

	// 处理带或不带链接的标题
	if topic.Href != "" {
		fmt.Fprintf(output, "[%s](%s)", topic.Title, topic.Href)
	} else {
		fmt.Fprint(output, topic.Title)
	}

	// 显示标记（如果有）
	if len(topic.Markers) > 0 {
		fmt.Fprint(output, " [")
		for i, marker := range topic.Markers {
			if i > 0 {
				fmt.Fprint(output, ", ")
			}
			fmt.Fprint(output, marker.MarkerID)
		}
		fmt.Fprint(output, "]")
	}
	fmt.Fprintln(output)

	// 递归打印子主题
	for _, child := range topic.Children.Attached {
		p.printTopic(child, level+1, output)
	}
}

// getIndent 获取缩进字符串
func (p *XMindMDProcessor) getIndent(level int) string {
	indent := ""
	for i := 0; i < level; i++ {
		indent += "  "
	}
	return indent
}

// countTopics 统计主题数量
func (p *XMindMDProcessor) countTopics(content XMindContent) int {
	count := 0
	for _, sheet := range content {
		count++ // 根主题
		count += p.countTopicChildren(sheet.RootTopic.Children.Attached)
	}
	return count
}

// countTopicChildren 递归统计子主题数量
func (p *XMindMDProcessor) countTopicChildren(topics []Topic) int {
	count := 0
	for _, topic := range topics {
		count++
		count += p.countTopicChildren(topic.Children.Attached)
	}
	return count
}

// ==================== 新增辅助方法 ====================

// getXMindFiles 获取所有需要处理的XMind文件
func (p *XMindMDProcessor) getXMindFiles() ([]string, error) {
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

	// 验证文件存在且为XMind文件
	validPaths := make([]string, 0, len(filePaths))
	for _, path := range filePaths {
		if p.isValidXMindFile(path) {
			validPaths = append(validPaths, path)
		}
	}

	if len(validPaths) == 0 {
		return nil, fmt.Errorf("没有找到有效的XMind文件")
	}

	return validPaths, nil
}

// loadPathsFromList 从列表文件加载路径
func (p *XMindMDProcessor) loadPathsFromList(listPath string) ([]string, error) {
	return md.LoadFileList(listPath)
}

// getPathsFromPath 从单个路径获取文件
func (p *XMindMDProcessor) getPathsFromPath(path string) ([]string, error) {
	// 检查路径是文件还是目录
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("访问路径失败: %w", err)
	}

	if info.IsDir() {
		// 处理目录：查找所有XMind文件
		return p.findXMindFilesInDir(path)
	} else {
		// 处理单个文件
		if strings.HasSuffix(strings.ToLower(path), ".xmind") {
			return []string{path}, nil
		}
		return nil, fmt.Errorf("不是有效的XMind文件: %s", path)
	}
}

// isValidXMindFile 验证是否为有效的XMind文件
func (p *XMindMDProcessor) isValidXMindFile(filePath string) bool {
	if filePath == "" {
		return false
	}

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".xmind" {
		return false
	}

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}

	// 确保是文件而不是目录
	return !info.IsDir()
}

// findXMindFilesInDir 在目录中查找所有XMind文件
func (p *XMindMDProcessor) findXMindFilesInDir(dir string) ([]string, error) {
	var xmindFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过子目录（如果需要递归，可以删除这个判断）
		if info.IsDir() && path != dir {
			return nil
		}

		// 检查是否为XMind文件
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".xmind") {
			xmindFiles = append(xmindFiles, path)
		}

		return nil
	})

	return xmindFiles, err
}

// createXMindTasks 创建XMind文件处理任务
func (p *XMindMDProcessor) createXMindTasks(xmindFiles []string) ([]XMindFileTask, error) {
	var tasks []XMindFileTask

	for _, xmindFile := range xmindFiles {
		// 确定输出文件路径
		outputFile := p.getOutputFilePath(xmindFile)

		tasks = append(tasks, XMindFileTask{
			InputFile:  xmindFile,
			OutputFile: outputFile,
		})
	}

	return tasks, nil
}

// getOutputFilePath 获取输出文件路径
func (p *XMindMDProcessor) getOutputFilePath(inputFile string) string {
	// 如果指定了输出目录，则输出到指定目录
	if p.outputDir != "" {
		fileName := filepath.Base(inputFile)
		outputName := strings.TrimSuffix(fileName, ".xmind") + ".md"
		return filepath.Join(p.outputDir, outputName)
	}

	// 否则在输入文件同目录生成
	return strings.TrimSuffix(inputFile, ".xmind") + ".md"
}

// processSequentially 顺序处理
func (p *XMindMDProcessor) processSequentially(tasks []XMindFileTask, result *XMindMDResult) error {
	var mu sync.Mutex

	for _, task := range tasks {
		if err := p.processXMindTask(task, result, &mu); err != nil {
			return err
		}
	}

	return nil
}

// processConcurrently 并发处理
func (p *XMindMDProcessor) processConcurrently(tasks []XMindFileTask, result *XMindMDResult) error {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 创建工作通道
	taskChan := make(chan XMindFileTask, len(tasks))

	// 启动工作协程
	for i := 0; i < p.config.Threads; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range taskChan {
				p.processXMindTask(task, result, &mu)
			}
		}(i)
	}

	// 发送任务
	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	// 等待所有工作完成
	wg.Wait()

	return nil
}

// processXMindTask 处理单个XMind文件任务
func (p *XMindMDProcessor) processXMindTask(task XMindFileTask, result *XMindMDResult, mu *sync.Mutex) error {
	// 创建输出目录
	outputDir := filepath.Dir(task.OutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Error("创建输出目录失败", zap.String("dir", outputDir), zap.Error(err))
		return err
	}

	// 解析XMind文件
	content, err := p.parseXMindFileFromPath(task.InputFile)
	if err != nil {
		log.Error("解析XMind文件失败", zap.String("file", task.InputFile), zap.Error(err))
		mu.Lock()
		result.FailedFiles++
		mu.Unlock()
		return err
	}

	// 转换为Markdown
	err = p.generateMarkdownToFile(content, task.OutputFile)
	if err != nil {
		log.Error("生成Markdown文件失败", zap.String("file", task.OutputFile), zap.Error(err))
		mu.Lock()
		result.FailedFiles++
		mu.Unlock()
		return err
	}

	// 更新统计结果
	sheetCount := len(content)
	topicCount := p.countTopics(content)

	mu.Lock()
	result.SuccessFiles++
	result.TotalSheets += sheetCount
	result.TotalTopics += topicCount
	mu.Unlock()

	log.Info("文件处理完成",
		zap.String("input", task.InputFile),
		zap.String("output", task.OutputFile),
		zap.Int("sheets", sheetCount),
		zap.Int("topics", topicCount))

	return nil
}

// parseXMindFileFromPath 从指定路径解析XMind文件
func (p *XMindMDProcessor) parseXMindFileFromPath(filePath string) (XMindContent, error) {
	// 打开XMind文件（实际上是zip文件）
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开XMind文件失败: %w", err)
	}
	defer r.Close()

	// 读取content.json
	var contentBytes []byte
	for _, f := range r.File {
		if f.Name == "content.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("打开content.json失败: %w", err)
			}
			defer rc.Close()

			contentBytes, err = io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("读取content.json失败: %w", err)
			}
			break
		}
	}

	if contentBytes == nil {
		return nil, fmt.Errorf("content.json在XMind文件中未找到")
	}

	// 解析JSON
	var content XMindContent
	err = json.Unmarshal(contentBytes, &content)
	if err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	return content, nil
}

// generateMarkdownToFile 生成Markdown文件到指定路径（原子性写入）
func (p *XMindMDProcessor) generateMarkdownToFile(content XMindContent, outputPath string) error {
	// 使用strings.Builder构建内容
	var buf strings.Builder

	// 为每个画布生成Markdown
	for _, sheet := range content {
		// 画布标题作为H1
		buf.WriteString(fmt.Sprintf("# %s\n\n", sheet.Title))

		// 根主题标题作为H2
		buf.WriteString(fmt.Sprintf("## %s\n", sheet.RootTopic.Title))
		if sheet.RootTopic.Href != "" {
			buf.WriteString(fmt.Sprintf("[%s](%s)\n", sheet.RootTopic.Title, sheet.RootTopic.Href))
		}
		buf.WriteString("\n")

		// 第一级主题作为H3
		for _, topic := range sheet.RootTopic.Children.Attached {
			buf.WriteString(fmt.Sprintf("### %s\n", topic.Title))
			if topic.Href != "" {
				buf.WriteString(fmt.Sprintf("[%s](%s)\n", topic.Title, topic.Href))
			}

			// 显示标记（如果有）
			if len(topic.Markers) > 0 {
				buf.WriteString("Markers: ")
				for i, marker := range topic.Markers {
					if i > 0 {
						buf.WriteString(", ")
					}
					buf.WriteString(marker.MarkerID)
				}
				buf.WriteString("\n")
			}

			// 更深层级作为列表
			for _, child := range topic.Children.Attached {
				p.printTopicToString(child, 0, &buf)
			}
			buf.WriteString("\n") // 主题间添加空格
		}
	}

	// 原子性写入文件
	return p.writeFileAtomically(outputPath, buf.String())
}

// printTopicToString 递归打印主题到字符串构建器
func (p *XMindMDProcessor) printTopicToString(topic Topic, level int, buf *strings.Builder) {
	// 打印带缩进的主题标题
	buf.WriteString(p.getIndent(level))
	buf.WriteString("- ")

	// 处理带或不带链接的标题
	if topic.Href != "" {
		buf.WriteString(fmt.Sprintf("[%s](%s)", topic.Title, topic.Href))
	} else {
		buf.WriteString(topic.Title)
	}

	// 打印标记
	if len(topic.Markers) > 0 {
		buf.WriteString(" ")
		for _, marker := range topic.Markers {
			buf.WriteString(marker.MarkerID)
		}
	}

	buf.WriteString("\n")

	// 递归打印子主题
	for _, child := range topic.Children.Attached {
		p.printTopicToString(child, level+1, buf)
	}
}

// writeFileAtomically 原子性写入文件
func (p *XMindMDProcessor) writeFileAtomically(filePath, content string) error {
	return utils.WriteFileAtomically(filePath, []byte(content))
}