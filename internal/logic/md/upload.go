package md

import (
	"bytes"
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

// UploadConfig MD图片上传配置
type UploadConfig struct {
	FilePath    string // MD文件路径或目录
	PicGoServer string // PicGo服务器地址，为空时使用配置文件
	Threads     int    // 并发线程数
}

// UploadResult 图片上传结果
type UploadResult struct {
	Success    bool   `json:"success"`
	FilePath   string `json:"file_path"`
	TotalLinks int    `json:"total_links"`
	LocalLinks int    `json:"local_links"`
	Uploaded   int    `json:"uploaded"`
	Failed     int    `json:"failed"`
	ErrorMessage string `json:"error_message,omitempty"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Duration   float64   `json:"duration"`
}

// UploadImageLink 上传图片链接信息
type UploadImageLink struct {
	Original string // 原始链接（文件中的内容）
	Decoded  string // 解码后的链接
	IsRemote bool   // 是否为远程链接
	AbsPath  string // 本地文件的绝对路径
	FilePath string // 所属的MD文件路径
}

// FileUploadTask 文件上传任务
type FileUploadTask struct {
	FilePath string     // MD文件路径
	Content  string     // 文件内容
	Links    []UploadImageLink // 图片链接列表
}

// ==================== 构造函数 ====================

// NewUploadProcessor 创建上传处理器
func NewUploadProcessor(config *UploadConfig, appConfig *models.AppConfig) (*UploadProcessor, error) {
	// 获取绝对路径
	absFilePath, err := filepath.Abs(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	processor := &UploadProcessor{
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

// UploadProcessor MD文件图片上传处理器
type UploadProcessor struct {
	config      *UploadConfig
	appConfig   *models.AppConfig
	content     string
	filePath    string
	absFilePath string
}

// ==================== 核心处理方法 ====================

// UploadLocalImages 上传本地图片到PicGo（支持文件夹和多线程）
func (p *UploadProcessor) UploadLocalImages() (*UploadResult, error) {
	result := &UploadResult{
		FilePath: p.filePath,
		StartTime: time.Now(),
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

	// 创建文件上传任务
	tasks, err := p.createUploadTasks(mdFiles)
	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("创建上传任务失败: %v", err)
		return result, err
	}

	// 统计总链接数
	for _, task := range tasks {
		result.TotalLinks += len(task.Links)
	}

	// 获取PicGo服务器地址
	picgoServer := p.getPicGoServer()

	// 执行并发上传
	if p.config.Threads > 1 {
		err = p.uploadConcurrently(tasks, picgoServer, result)
	} else {
		err = p.uploadSequentially(tasks, picgoServer, result)
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
func (p *UploadProcessor) isRemoteLink(link string) bool {
	return IsRemoteLink(link)
}

// getPicGoServer 获取PicGo服务器地址
func (p *UploadProcessor) getPicGoServer() string {
	// 优先使用命令行参数指定的服务器地址
	if p.config.PicGoServer != "" {
		log.Info("使用命令行指定的PicGo服务器", zap.String("server", p.config.PicGoServer))
		return p.config.PicGoServer
	}

	// 使用配置文件中的默认服务器地址
	server := p.appConfig.PicGo.Server
	log.Info("使用配置文件的PicGo服务器", zap.String("server", server), zap.Bool("isEmpty", server == ""))

	// 如果配置文件中也为空，使用硬编码的默认值
	if server == "" {
		server = "http://127.0.0.1:36677"
		log.Info("配置文件中PicGo服务器为空，使用默认值", zap.String("server", server))
	}

	return server
}


// uploadAndReplaceImage 上传并替换单个图片
func (p *UploadProcessor) uploadAndReplaceImage(link UploadImageLink, picgoServer string) error {
	// 上传图片
	url, err := p.uploadToPicGo(picgoServer, link.AbsPath)
	if err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}

	// 替换链接
	p.replaceImageLink(link, url)
	return nil
}

// replaceImageLink 替换图片链接
func (p *UploadProcessor) replaceImageLink(link UploadImageLink, newUrl string) {
	p.content = ReplaceImageLinkInContent(p.content, link.Original, newUrl)
}

// uploadToPicGo 上传图片到PicGo
func (p *UploadProcessor) uploadToPicGo(server, filePath string) (string, error) {
	log.Info("开始上传图片到PicGo",
		zap.String("server", server),
		zap.String("filePath", filePath))

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 构建请求数据
	dataMap := map[string][]string{
		"list": {filePath},
	}

	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		return "", fmt.Errorf("序列化请求数据失败: %w", err)
	}

	log.Debug("PicGo请求数据", zap.ByteString("data", jsonData))

	// 创建HTTP请求
	requestURL := server + "/upload"
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	log.Debug("PicGo请求",
		zap.String("url", requestURL),
		zap.String("method", req.Method),
		zap.String("contentType", req.Header.Get("Content-Type")))

	// 设置超时
	client := &http.Client{
		Timeout: time.Duration(p.appConfig.PicGo.Timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	log.Info("PicGo响应",
		zap.String("statusCode", resp.Status),
		zap.Int("statusCodeNum", resp.StatusCode))

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	log.Debug("PicGo响应体",
		zap.ByteString("response", responseBody),
		zap.String("responseStr", string(responseBody)))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("上传失败，状态码: %s, 响应: %s", resp.Status, string(responseBody))
	}

	// 解析响应
	var result struct {
		Success bool     `json:"success"`
		Result  []string `json:"result"`
		Message string    `json:"message,omitempty"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	log.Debug("PicGo解析结果",
		zap.Bool("success", result.Success),
		zap.Strings("result", result.Result),
		zap.String("message", result.Message))

	if !result.Success {
		errorMsg := fmt.Sprintf("PicGo服务返回失败: success=%v", result.Success)
		if result.Message != "" {
			errorMsg += fmt.Sprintf(", message=%s", result.Message)
		}
		return "", fmt.Errorf(errorMsg)
	}

	if len(result.Result) == 0 {
		return "", fmt.Errorf("PicGo服务返回空结果")
	}

	log.Info("图片上传成功", zap.String("url", result.Result[0]))
	return result.Result[0], nil
}

// ==================== 新增辅助方法 ====================

// findMDFiles 查找所有需要处理的MD文件
func (p *UploadProcessor) findMDFiles() ([]string, error) {
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
func (p *UploadProcessor) findMDFilesInDir(dir string) ([]string, error) {
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

// createUploadTasks 创建文件上传任务
func (p *UploadProcessor) createUploadTasks(mdFiles []string) ([]FileUploadTask, error) {
	var tasks []FileUploadTask

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

		// 只保留本地图片
		var localLinks []UploadImageLink
		for _, link := range links {
			if !link.IsRemote {
				localLinks = append(localLinks, link)
			}
		}

		if len(localLinks) > 0 {
			tasks = append(tasks, FileUploadTask{
				FilePath: filePath,
				Content:  string(content),
				Links:    localLinks,
			})
		}
	}

	return tasks, nil
}

// extractImageLinksFromContent 从内容中提取图片链接
func (p *UploadProcessor) extractImageLinksFromContent(content, absFilePath string) ([]UploadImageLink, error) {
	imageLinks := ExtractImageLinks(content)

	var links []UploadImageLink
	for _, imgLink := range imageLinks {
		if imgLink.IsRemote {
			continue // upload 只关心本地链接
		}

		absPath := filepath.Join(filepath.Dir(absFilePath), imgLink.Decoded)
		links = append(links, UploadImageLink{
			Original: imgLink.Original,
			Decoded:  imgLink.Decoded,
			IsRemote: imgLink.IsRemote,
			AbsPath:  absPath,
			FilePath: absFilePath,
		})
	}

	return links, nil
}

// uploadConcurrently 并发上传
func (p *UploadProcessor) uploadConcurrently(tasks []FileUploadTask, picgoServer string, result *UploadResult) error {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 创建工作通道
	taskChan := make(chan FileUploadTask, len(tasks))

	// 启动工作协程
	for i := 0; i < p.config.Threads; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range taskChan {
				p.processUploadTask(task, picgoServer, result, &mu)
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

// uploadSequentially 顺序上传
func (p *UploadProcessor) uploadSequentially(tasks []FileUploadTask, picgoServer string, result *UploadResult) error {
	var mu sync.Mutex

	for _, task := range tasks {
		if err := p.processUploadTask(task, picgoServer, result, &mu); err != nil {
			return err
		}
	}

	return nil
}

// processUploadTask 处理单个文件的上传任务
func (p *UploadProcessor) processUploadTask(task FileUploadTask, picgoServer string, result *UploadResult, mu *sync.Mutex) error {
	var newContent string = task.Content
	var uploadedCount int
	var failedCount int

	// 备份文件
	if err := p.createBackupFile(task.FilePath, task.Content); err != nil {
		log.Error("备份文件失败", zap.String("file", task.FilePath), zap.Error(err))
		return err
	}

	// 处理每个图片链接
	for _, link := range task.Links {
		url, err := p.uploadToPicGo(picgoServer, link.AbsPath)
		if err != nil {
			log.Error("上传图片失败", zap.String("path", link.AbsPath), zap.Error(err))
			failedCount++
			continue
		}

		// 替换链接
		newContent = p.replaceImageLinkInContent(newContent, link, url)
		uploadedCount++
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

	// 更新统计结果
	mu.Lock()
	result.Uploaded += uploadedCount
	result.Failed += failedCount
	result.LocalLinks += len(task.Links)
	mu.Unlock()

	log.Info("文件处理完成",
		zap.String("file", task.FilePath),
		zap.Int("uploaded", uploadedCount),
		zap.Int("failed", failedCount))

	return nil
}

// replaceImageLinkInContent 在内容中替换图片链接
func (p *UploadProcessor) replaceImageLinkInContent(content string, link UploadImageLink, newUrl string) string {
	return ReplaceImageLinkInContent(content, link.Original, newUrl)
}

// createBackupFile 备份指定文件
func (p *UploadProcessor) createBackupFile(filePath, content string) error {
	ext := filepath.Ext(filePath)
	backupPath := strings.TrimSuffix(filePath, ext) + "_" + time.Now().Format("20060102150405") + ext
	return os.WriteFile(backupPath, []byte(content), 0644)
}