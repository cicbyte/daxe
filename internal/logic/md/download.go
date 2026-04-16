package md

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cicbyte/daxe/internal/log"
	"github.com/cicbyte/daxe/internal/models"
	"github.com/cicbyte/daxe/internal/utils"
	"go.uber.org/zap"
)

// ==================== 数据类型定义 ====================

// DownloadConfig MD图片下载配置
type DownloadConfig struct {
	FilePath string // MD文件路径或目录
	Threads  int    // 并发线程数
}

// DownloadResult 图片下载结果
type DownloadResult struct {
	Success      bool              `json:"success"`
	FilePath     string            `json:"file_path"`
	TotalLinks   int               `json:"total_links"`
	RemoteLinks  int               `json:"remote_links"`
	Downloaded   int               `json:"downloaded"`
	Failed       int               `json:"failed"`
	LinkMappings map[string]string `json:"link_mappings,omitempty"` // 原始URL到本地文件名的映射
	ErrorMessage string            `json:"error_message,omitempty"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     float64           `json:"duration"`
}

// DownloadImageLink 下载图片链接信息
type DownloadImageLink struct {
	Original string // 原始链接（文件中的内容）
	Decoded  string // 解码后的链接
	IsRemote bool   // 是否为远程链接
	FilePath string // 所属的MD文件路径
}

// FileDownloadTask 文件下载任务
type FileDownloadTask struct {
	FilePath string     // MD文件路径
	Content  string     // 文件内容
	Links    []DownloadImageLink // 图片链接列表
}

// ==================== 构造函数 ====================

// NewDownloadProcessor 创建下载处理器
func NewDownloadProcessor(config *DownloadConfig, appConfig *models.AppConfig) (*DownloadProcessor, error) {
	// 获取绝对路径
	absFilePath, err := filepath.Abs(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	processor := &DownloadProcessor{
		config:      config,
		appConfig:   appConfig,
		filePath:    config.FilePath,
		absFilePath: absFilePath,
	}

	// 只有在处理单个文件时才读取内容
	// 处理目录时，文件读取将在具体任务处理时进行
	info, err := os.Stat(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("访问路径失败: %w", err)
	}

	if !info.IsDir() {
		// 单文件处理：读取文件内容
		content, err := os.ReadFile(config.FilePath)
		if err != nil {
			return nil, fmt.Errorf("读取MD文件失败: %w", err)
		}
		processor.content = string(content)
	}

	return processor, nil
}

// DownloadProcessor MD文件图片下载处理器
type DownloadProcessor struct {
	config      *DownloadConfig
	appConfig   *models.AppConfig
	content     string
	filePath    string
	absFilePath string
}

// ==================== 核心处理方法 ====================

// DownloadRemoteImages 下载远程图片到本地
func (p *DownloadProcessor) DownloadRemoteImages() (*DownloadResult, error) {
	result := &DownloadResult{
		FilePath:     p.filePath,
		LinkMappings: make(map[string]string),
		StartTime:    time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	}()

	// 获取所有需要处理的MD文件
	mdFiles, err := p.findMDFiles()
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("查找MD文件失败: %v", err)
		return result, err
	}

	if len(mdFiles) == 0 {
		result.Success = false
		result.ErrorMessage = "没有找到MD文件"
		return result, fmt.Errorf("没有找到MD文件")
	}

	log.Info("找到MD文件", zap.Int("count", len(mdFiles)), zap.Strings("files", mdFiles))

	// 创建文件下载任务
	tasks, err := p.createDownloadTasks(mdFiles)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("创建下载任务失败: %v", err)
		return result, err
	}

	// 统计总链接数
	for _, task := range tasks {
		result.TotalLinks += len(task.Links)
	}

	// 执行并发下载
	if p.config.Threads > 1 {
		err = p.downloadConcurrently(tasks, result)
	} else {
		err = p.downloadSequentially(tasks, result)
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


// isRemoteLink 判断是否为远程链接（已迁移到 common.go IsRemoteLink）
// 保留方法以兼容现有调用
func (p *DownloadProcessor) isRemoteLink(link string) bool {
	return IsRemoteLink(link)
}

// saveFile 保存文件
func (p *DownloadProcessor) saveFile() error {
	return os.WriteFile(p.filePath, []byte(p.content), 0644)
}

// getImageDir 获取图片保存目录
func (p *DownloadProcessor) getImageDir() string {
	return filepath.Join(filepath.Dir(p.absFilePath), strings.TrimSuffix(filepath.Base(p.filePath), ".md"))
}


// replaceImageLink 替换图片链接
func (p *DownloadProcessor) replaceImageLink(link DownloadImageLink, fileName string) {
	p.content = ReplaceImageLinkInContent(p.content, link.Original, fileName)
}

// downloadImage 下载图片
func (p *DownloadProcessor) downloadImage(url, savePath string) error {
	client := &http.Client{
		Timeout: time.Duration(p.appConfig.PicGo.Timeout) * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %s", resp.Status)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	return nil
}

// saveLinkMappings 保存链接映射关系
func (p *DownloadProcessor) saveLinkMappings(imgDir string, mappings map[string]string) error {
	jsonData, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(imgDir, "link_mappings.json"), jsonData, 0644)
}

// ==================== 新增辅助方法 ====================

// findMDFiles 查找所有需要处理的MD文件
func (p *DownloadProcessor) findMDFiles() ([]string, error) {
	// 检查路径是文件还是目录
	info, err := os.Stat(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("访问路径失败: %w", err)
	}

	if info.IsDir() {
		// 处理目录：查找所有MD文件
		return p.findMDFilesInDir(p.filePath)
	} else {
		// 处理单个文件
		if strings.HasSuffix(strings.ToLower(p.filePath), ".md") || strings.HasSuffix(strings.ToLower(p.filePath), ".markdown") {
			return []string{p.filePath}, nil
		}
		return nil, fmt.Errorf("不是有效的MD文件: %s", p.filePath)
	}
}

// findMDFilesInDir 在目录中查找所有MD文件
func (p *DownloadProcessor) findMDFilesInDir(dir string) ([]string, error) {
	var mdFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过子目录（如果需要递归，可以删除这个判断）
		if info.IsDir() && path != dir {
			return nil
		}

		// 检查是否为MD文件
		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".md") || strings.HasSuffix(strings.ToLower(path), ".markdown")) {
			mdFiles = append(mdFiles, path)
		}

		return nil
	})

	return mdFiles, err
}

// createDownloadTasks 创建文件下载任务
func (p *DownloadProcessor) createDownloadTasks(mdFiles []string) ([]FileDownloadTask, error) {
	var tasks []FileDownloadTask

	for _, filePath := range mdFiles {
		// 读取文件内容
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Error("读取文件失败", zap.String("file", filePath), zap.Error(err))
			continue
		}

		// 获取绝对路径
		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			log.Error("获取绝对路径失败", zap.String("file", filePath), zap.Error(err))
			continue
		}

		// 提取图片链接
		links, err := p.extractImageLinksFromContent(string(content), absFilePath)
		if err != nil {
			log.Error("提取图片链接失败", zap.String("file", filePath), zap.Error(err))
			continue
		}

		// 只保留远程图片
		var remoteLinks []DownloadImageLink
		for _, link := range links {
			if link.IsRemote {
				remoteLinks = append(remoteLinks, link)
			}
		}

		if len(remoteLinks) > 0 {
			tasks = append(tasks, FileDownloadTask{
				FilePath: filePath,
				Content:  string(content),
				Links:    remoteLinks,
			})
		}
	}

	return tasks, nil
}

// extractImageLinksFromContent 从内容中提取图片链接
func (p *DownloadProcessor) extractImageLinksFromContent(content, absFilePath string) ([]DownloadImageLink, error) {
	imageLinks := ExtractImageLinks(content)

	var links []DownloadImageLink
	for _, imgLink := range imageLinks {
		if !imgLink.IsRemote {
			continue // download 只关心远程链接
		}
		links = append(links, DownloadImageLink{
			Original: imgLink.Original,
			Decoded:  imgLink.Decoded,
			IsRemote: imgLink.IsRemote,
			FilePath: absFilePath,
		})
	}

	return links, nil
}

// downloadConcurrently 并发下载
func (p *DownloadProcessor) downloadConcurrently(tasks []FileDownloadTask, result *DownloadResult) error {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 创建工作通道
	taskChan := make(chan FileDownloadTask, len(tasks))

	// 启动工作协程
	for i := 0; i < p.config.Threads; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range taskChan {
				p.processDownloadTask(task, result, &mu)
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

// downloadSequentially 顺序下载
func (p *DownloadProcessor) downloadSequentially(tasks []FileDownloadTask, result *DownloadResult) error {
	var mu sync.Mutex

	for _, task := range tasks {
		if err := p.processDownloadTask(task, result, &mu); err != nil {
			return err
		}
	}

	return nil
}

// processDownloadTask 处理单个文件的下载任务
func (p *DownloadProcessor) processDownloadTask(task FileDownloadTask, result *DownloadResult, mu *sync.Mutex) error {
	var newContent string = task.Content
	var downloadedCount int
	var failedCount int

	// 统一使用images文件夹（在第一个MD文件的同级目录下）
	imgDir := p.getUnifiedImagesDir(task.FilePath)
	if err := os.MkdirAll(imgDir, 0755); err != nil {
		log.Error("创建images目录失败", zap.String("dir", imgDir), zap.Error(err))
		return err
	}

	// 处理每个图片链接
	for _, link := range task.Links {
		fileName, err := p.downloadAndReplaceImageForFile(link, imgDir, newContent, task.FilePath)
		if err != nil {
			log.Error("下载图片失败", zap.String("url", link.Original), zap.Error(err))
			failedCount++
			continue
		}

		// 替换链接
		newContent = p.replaceImageLinkInContent(newContent, link, fileName)

		// 更新链接映射
		mu.Lock()
		result.LinkMappings[link.Original] = fileName
		mu.Unlock()

		downloadedCount++
	}

	// 备份原文件
	if _, err := utils.BackupFile(task.FilePath); err != nil {
		log.Warn("备份文件失败", zap.String("file", task.FilePath), zap.Error(err))
		// 备份失败不影响主要功能，继续执行
	}

	// 保存文件
	if err := os.WriteFile(task.FilePath, []byte(newContent), 0644); err != nil {
		log.Error("保存文件失败", zap.String("file", task.FilePath), zap.Error(err))
		return err
	}

	// 保存链接映射关系
	if err := p.saveLinkMappingsForFile(imgDir, result.LinkMappings); err != nil {
		log.Error("保存链接映射失败", zap.String("dir", imgDir), zap.Error(err))
	}

	// 更新统计结果
	mu.Lock()
	result.Downloaded += downloadedCount
	result.Failed += failedCount
	result.RemoteLinks += len(task.Links)
	mu.Unlock()

	log.Info("文件处理完成",
		zap.String("file", task.FilePath),
		zap.Int("downloaded", downloadedCount),
		zap.Int("failed", failedCount))

	return nil
}

// getImageDirForFile 获取文件的图片目录（已弃用，使用getUnifiedImagesDir）
func (p *DownloadProcessor) getImageDirForFile(filePath string) string {
	ext := filepath.Ext(filePath)
	baseName := strings.TrimSuffix(filepath.Base(filePath), ext)
	return filepath.Join(filepath.Dir(filePath), baseName)
}

// getUnifiedImagesDir 获取统一的images目录
func (p *DownloadProcessor) getUnifiedImagesDir(filePath string) string {
	// 统一在当前处理路径的上级目录创建images文件夹
	if p.config.Threads > 1 || len(strings.Split(filepath.Dir(p.filePath), string(filepath.Separator))) > 1 {
		// 多线程或多文件处理时，使用配置路径的上级目录
		parentDir := filepath.Dir(p.filePath)
		return filepath.Join(parentDir, "images")
	} else {
		// 单文件处理时，使用文件所在目录
		return filepath.Join(filepath.Dir(filePath), "images")
	}
}

// getRelativeImagePath 获取从MD文件到图片的相对路径
func (p *DownloadProcessor) getRelativeImagePath(mdFilePath, fileName string) string {
	mdDir := filepath.Dir(mdFilePath)
	imagesDir := filepath.Join(mdDir, "images")

	// 计算相对路径
	relPath, err := filepath.Rel(mdDir, filepath.Join(imagesDir, fileName))
	if err != nil {
		// 如果计算失败，使用简单的相对路径
		return filepath.Join("images", fileName)
	}

	// 确保路径使用正斜杠（跨平台兼容）
	return strings.ReplaceAll(relPath, "\\", "/")
}

// inferExtensionFromURL 从URL推断文件扩展名
func (p *DownloadProcessor) inferExtensionFromURL(url string) string {
	// 检查URL路径中是否包含文件扩展名
	if strings.Contains(url, "/image/png") {
		return ".png"
	} else if strings.Contains(url, "/image/jpeg") {
		return ".jpg"
	} else if strings.Contains(url, "/image/jpg") {
		return ".jpg"
	} else if strings.Contains(url, "/image/gif") {
		return ".gif"
	} else if strings.Contains(url, "/image/webp") {
		return ".webp"
	} else if strings.Contains(url, "/image/svg") {
		return ".svg"
	}

	// 根据URL模式推断
	if strings.Contains(url, "png") || strings.HasSuffix(url, "png") {
		return ".png"
	} else if strings.Contains(url, "jpg") || strings.Contains(url, "jpeg") {
		return ".jpg"
	} else if strings.Contains(url, "gif") {
		return ".gif"
	} else if strings.Contains(url, "webp") {
		return ".webp"
	} else if strings.Contains(url, "svg") {
		return ".svg"
	}

	// 默认使用图片扩展名
	return ".jpg"
}

// downloadAndReplaceImageForFile 为指定文件下载并替换单个图片
func (p *DownloadProcessor) downloadAndReplaceImageForFile(link DownloadImageLink, imgDir, content, mdFilePath string) (string, error) {
	log.Info("处理图片下载",
		zap.String("original", link.Original),
		zap.String("decoded", link.Decoded))

	// 生成文件名 - 如果URL没有扩展名，则根据HTTP响应头获取
	hash := md5.Sum([]byte(link.Decoded))
	baseName := hex.EncodeToString(hash[:])

	// 尝试从URL获取扩展名
	ext := filepath.Ext(link.Decoded)
	if ext == "" {
		// 从Content-Type推断扩展名
		ext = p.inferExtensionFromURL(link.Decoded)
	}

	fileName := baseName + ext
	savePath := filepath.Join(imgDir, fileName)

	log.Info("生成文件名",
		zap.String("fileName", fileName),
		zap.String("savePath", savePath),
		zap.String("inferredExt", ext))

	// 下载图片
	if err := p.downloadImage(link.Decoded, savePath); err != nil {
		return "", err
	}

	// 生成相对路径引用（从MD文件到images文件夹）
	relativePath := p.getRelativeImagePath(mdFilePath, fileName)
	return relativePath, nil
}

// replaceImageLinkInContent 在内容中替换图片链接
func (p *DownloadProcessor) replaceImageLinkInContent(content string, link DownloadImageLink, fileName string) string {
	return ReplaceImageLinkInContent(content, link.Original, fileName)
}

// saveLinkMappingsForFile 保存文件的链接映射关系
func (p *DownloadProcessor) saveLinkMappingsForFile(imgDir string, mappings map[string]string) error {
	jsonData, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(imgDir, "link_mappings.json"), jsonData, 0644)
}