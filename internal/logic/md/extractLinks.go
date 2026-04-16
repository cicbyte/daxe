package md

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cicbyte/daxe/internal/models"
)

// ExtractLinksConfig MD链接提取配置
type ExtractLinksConfig struct {
	FilePath string // MD文件路径
}

// ExtractLinksResult 链接提取结果
type ExtractLinksResult struct {
	Success      bool              `json:"success"`
	FilePath     string            `json:"file_path"`
	TotalLinks   int               `json:"total_links"`
	LocalLinks   int               `json:"local_links"`
	RemoteLinks  int               `json:"remote_links"`
	ImageLinks   []ExtractImageLink `json:"image_links"`
	ErrorMessage string            `json:"error_message,omitempty"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     float64           `json:"duration"`
}

// ExtractImageLink 提取图片链接信息
type ExtractImageLink struct {
	Original string `json:"original"` // 原始链接（文件中的内容）
	Decoded  string `json:"decoded"`  // 解码后的链接
	IsRemote bool   `json:"is_remote"` // 是否为远程链接
	AbsPath  string `json:"abs_path"`  // 本地文件的绝对路径
}

// NewExtractLinksProcessor 创建链接提取处理器
func NewExtractLinksProcessor(config *ExtractLinksConfig, appConfig *models.AppConfig) (*ExtractLinksProcessor, error) {
	content, err := os.ReadFile(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("读取MD文件失败: %w", err)
	}

	absFilePath, err := filepath.Abs(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	return &ExtractLinksProcessor{
		config:      config,
		appConfig:   appConfig,
		content:     string(content),
		filePath:    config.FilePath,
		absFilePath: absFilePath,
	}, nil
}

// ExtractLinksProcessor MD文件链接提取处理器
type ExtractLinksProcessor struct {
	config      *ExtractLinksConfig
	appConfig   *models.AppConfig
	content     string
	filePath    string
	absFilePath string
}

// ExtractLinks 提取所有图片链接
func (p *ExtractLinksProcessor) ExtractLinks() (*ExtractLinksResult, error) {
	result := &ExtractLinksResult{
		FilePath: p.filePath,
		StartTime: time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).Seconds()
	}()

	// 提取图片链接
	links, err := p.extractImageLinks()
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	result.ImageLinks = links
	result.TotalLinks = len(links)

	for _, link := range links {
		if link.IsRemote {
			result.RemoteLinks++
		} else {
			result.LocalLinks++
		}
	}

	result.Success = true
	return result, nil
}

// extractImageLinks 提取图片链接
func (p *ExtractLinksProcessor) extractImageLinks() ([]ExtractImageLink, error) {
	imageLinks := ExtractImageLinks(p.content)

	var links []ExtractImageLink
	for _, imgLink := range imageLinks {
		var absPath string
		if !imgLink.IsRemote {
			absPath = filepath.Join(filepath.Dir(p.absFilePath), imgLink.Decoded)
		}

		links = append(links, ExtractImageLink{
			Original: imgLink.Original,
			Decoded:  imgLink.Decoded,
			IsRemote: imgLink.IsRemote,
			AbsPath:  absPath,
		})
	}

	return links, nil
}
