package md

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// ==================== 数据类型定义 ====================

// MDConfig MD处理通用配置
type MDConfig struct {
	ThreadCount   int    // 并发线程数
	InputPath     string // 输入路径
	InputListPath string // 输入文件列表路径
	OutputDir     string // 输出目录
}

// ProgressBar 进度条
type ProgressBar struct {
	Total     int       // 总任务数
	current   int32     // 当前进度
	StartTime time.Time // 开始时间
	started   bool      // 是否已开始
}

// Start 开始进度条
func (p *ProgressBar) Start() {
	p.started = true
	p.StartTime = time.Now()
	p.current = 0
}

// Increment 增加进度
func (p *ProgressBar) Increment() {
	newVal := atomic.AddInt32(&p.current, 1)
	if p.Total > 0 {
		percent := int(float64(newVal) / float64(p.Total) * 100)
		fmt.Printf("\r进度: %d/%d (%d%%)", newVal, p.Total, percent)
	}
}

// Finish 完成进度条
func (p *ProgressBar) Finish() {
	if p.started {
		duration := time.Since(p.StartTime)
		fmt.Printf("\n✅ 完成! 耗时: %v\n", duration.Round(time.Millisecond))
	}
}

// QAItem 问答对项
type QAItem struct {
	Question   string   `json:"question"`   // 问题
	Answer     string   `json:"answer"`     // 答案
	Category   string   `json:"category"`   // 分类
	Tags       []string `json:"tags"`       // 标签
	Difficulty int      `json:"difficulty"` // 难度等级（1-5）
}

// QAList 问答对列表
type QAList struct {
	QAItems []QAItem `json:"qa_items"` // 问答对列表
}

// ==================== 图片链接公共函数 ====================

// ImageLink 通用图片链接信息
type ImageLink struct {
	Original string // 原始链接（文件中的内容）
	Decoded  string // 解码后的链接
	IsRemote bool   // 是否为远程链接
}

// imageLinkPattern 匹配 Markdown 和 HTML 两种图片格式
var imageLinkPattern = regexp.MustCompile(`(!\[.*?\]\((.*?)\)|<img[^>]+src="([^"]+)"[^>]*>)`)

// ExtractImageLinks 从内容中提取所有图片链接
func ExtractImageLinks(content string) []ImageLink {
	matches := imageLinkPattern.FindAllStringSubmatch(content, -1)

	var links []ImageLink
	for _, match := range matches {
		var original string
		switch {
		case len(match[2]) > 0: // Markdown格式
			original = match[2]
		case len(match[3]) > 0: // HTML格式
			original = match[3]
		default:
			continue
		}

		if strings.HasPrefix(original, "(") {
			continue
		}

		decoded, err := url.QueryUnescape(original)
		if err != nil {
			decoded = original
		}

		links = append(links, ImageLink{
			Original: original,
			Decoded:  decoded,
			IsRemote: IsRemoteLink(decoded),
		})
	}

	return links
}

// IsRemoteLink 判断是否为远程链接
func IsRemoteLink(link string) bool {
	return strings.HasPrefix(link, "http://") ||
		strings.HasPrefix(link, "https://") ||
		strings.HasPrefix(link, "ftp://")
}

// ReplaceImageLinkInContent 在内容中替换图片链接
func ReplaceImageLinkInContent(content, original, replacement string) string {
	// 替换HTML格式
	content = strings.ReplaceAll(content,
		fmt.Sprintf(`"%s"`, original),
		fmt.Sprintf(`"%s"`, replacement))
	// 替换Markdown格式
	content = strings.ReplaceAll(content,
		fmt.Sprintf(`(%s)`, original),
		fmt.Sprintf(`(%s)`, replacement))
	return content
}

// ==================== 文件列表加载 ====================

// LoadFileList 从 JSON/TXT 文件加载路径列表
func LoadFileList(listPath string) ([]string, error) {
	data, err := os.ReadFile(listPath)
	if err != nil {
		return nil, fmt.Errorf("读取文件列表失败: %w", err)
	}

	content := strings.TrimSpace(string(data))

	// JSON 格式: 以 [ 开头
	if strings.HasPrefix(content, "[") {
		var paths []string
		if err := json.Unmarshal(data, &paths); err != nil {
			return nil, fmt.Errorf("解析JSON文件列表失败: %w", err)
		}
		return paths, nil
	}

	// TXT 格式: 每行一个路径
	lines := strings.Split(content, "\n")
	var paths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}

// ==================== 原子文件写入 ====================

// WriteFileAtomically 原子写入文件（先写临时文件再 rename）
func WriteFileAtomically(filePath, content string) error {
	tempPath := filePath + ".tmp." + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("重命名文件失败: %w", err)
	}
	return nil
}
