package pdf

import (
	"fmt"
	"strings"
)

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
