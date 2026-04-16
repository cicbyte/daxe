# daxe pdf

PDF 文件处理工具集。

## 命令

### images — 图片提取

从 PDF 文件中提取图片并保存到指定目录。

```bash
daxe pdf images input.pdf -o ./images                     # 提取所有图片
daxe pdf images input.pdf -o ./images --pages 1,3,5-8     # 指定页面范围
daxe pdf images input.pdf -o ./images --format jpeg --quality 95
daxe pdf images "*.pdf" -o ./images --batch                # 批量处理
```

| 参数 | 说明 |
|------|------|
| `-o, --output DIR` | 输出目录路径（必需） |
| `-t, --threads N` | 并发线程数（默认1，最大100） |
| `--pages` | 页面范围，如 `1,3,5-8`（默认全部） |
| `--format` | 图片格式：`png`、`jpeg`（默认 `png`） |
| `--quality` | JPEG 质量 1-100（默认 90） |
| `--batch` | 批量处理模式（支持通配符） |
| `--page-dirs` | 为每个页面创建独立子目录 |
| `--overwrite` | 覆盖已存在的图片 |
| `-q, --quiet` | 静默模式 |

页面范围格式：单页 `1,3,5`，范围 `1-5,10-20`，混合 `1,3-5,10`。
