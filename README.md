# daxe

[![Release](https://img.shields.io/github/v/release/cicbyte/daxe?style=flat-square)](https://github.com/cicbyte/daxe/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/cicbyte/daxe?style=flat-square)](https://goreportcard.com/report/github.com/cicbyte/daxe)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/cicbyte/daxe/release.yml?style=flat-square)](https://github.com/cicbyte/daxe/actions)

> **d**ata + **axe** — 快刀斩乱麻，专业的文件处理 CLI 工具集。

支持 Markdown 处理、PDF 图片提取/拆分和 XMind 思维导图转换。Go 编译，单文件分发，无运行时依赖。

## 功能特性

### Markdown 处理

- **AI 智能修复** — 自动修复格式错误、语法问题、Mermaid 图表、表格、标点符号
- **JSON 转 MD** — 将 JSON 问答对转换为结构化 Markdown
- **图片上传** — 批量上传本地图片至 PicGo，自动替换链接并备份原文件
- **图片下载** — 批量下载远程图片到本地，MD5 哈希去重
- **链接提取** — 提取所有图片链接，支持文本/JSON 输出

### PDF 处理

- **图片提取** — 从 PDF 提取图片，支持 PNG/JPEG、页面范围、批量处理
- **PDF 拆分** — 按页码范围、固定页数、单页提取三种模式拆分 PDF

### XMind 处理

- **XMind 转 MD** — 将 XMind 思维导图转换为 Markdown，保持层级和链接

## 安装

**从 Release 下载**（推荐）

从 [Releases](https://github.com/cicbyte/daxe/releases) 下载对应平台的可执行文件，支持 Windows / Linux / macOS。

**go install**

```bash
go install github.com/cicbyte/daxe@latest
```

**从源码编译**

```bash
git clone https://github.com/cicbyte/daxe.git
cd daxe
go build -o daxe
```

## 快速开始

```bash
# AI 修复 Markdown 格式
daxe md fix ./document.md

# 提取 PDF 中的图片
daxe pdf images input.pdf -o ./images --pages 1,3,5-8

# 每 10 页拆分 PDF
daxe pdf split input.pdf -o ./output --every 10

# XMind 转 Markdown
daxe xmind md mindmap.xmind
```

## 命令参考

| 模块 | 命令 | 说明 | 文档 |
|------|------|------|------|
| `daxe md` | `fix` | AI 智能修复 Markdown 格式 | [docs/md.md](docs/md.md) |
| | `convert` | JSON 问答对转 Markdown | |
| | `upload` | 上传本地图片至 PicGo | |
| | `download` | 下载远程图片到本地 | |
| | `extractLinks` | 提取图片链接 | |
| `daxe pdf` | `images` | 从 PDF 提取图片 | [docs/pdf.md](docs/pdf.md) |
| | `split` | 拆分 PDF 文件 | |
| `daxe xmind` | `md` | XMind 转 Markdown | [docs/xmind.md](docs/xmind.md) |

```bash
daxe --version                # 查看版本
daxe <command> --help         # 查看命令帮助
daxe <command> <sub> --help   # 查看子命令帮助
```

## 配置

配置文件路径：`~/.cicbyte/daxe/config/config.yaml`，首次运行自动创建。

```yaml
ai:
  provider: "openai"          # openai / ollama
  base_url: "https://open.bigmodel.cn/api/paas/v4/"
  api_key: ""
  model: "GLM-4-Flash-250414"
  max_tokens: 2048
  temperature: 0.8

database:
  type: "sqlite"

picgo:
  server: "http://127.0.0.1:36677"
```

## 技术栈

- [Cobra](https://github.com/spf13/cobra) — CLI 命令框架
- [UniPDF](https://github.com/unidoc/unipdf) — PDF 图片提取
- [pdfcpu](https://github.com/pdfcpu/pdfcpu) — PDF 拆分
- [Eino](https://github.com/cloudwego/eino) — AI 模型调用
- [GORM](https://gorm.io) + [SQLite](https://github.com/glebarez/sqlite) — 数据持久化（纯 Go，无 CGO）
- [Zap](https://github.com/uber-go/zap) — 结构化日志

## License

[MIT](LICENSE)
