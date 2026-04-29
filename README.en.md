# daxe

[![Release](https://img.shields.io/github/v/release/cicbyte/daxe?style=flat-square)](https://github.com/cicbyte/daxe/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/cicbyte/daxe?style=flat-square)](https://goreportcard.com/report/github.com/cicbyte/daxe)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Build](https://img.shields.io/github/actions/workflow/status/cicbyte/daxe/release.yml?style=flat-square)](https://github.com/cicbyte/daxe/actions)

> **d**ata + **axe** — A sharp CLI toolkit for file processing.

Markdown processing, PDF image extraction/splitting, and XMind mindmap conversion. Written in Go, distributed as a single binary, no runtime dependencies.

## Features

### Markdown Processing

- **AI Fix** — Auto-fix formatting errors, syntax issues, Mermaid charts, tables, and punctuation
- **JSON to MD** — Convert JSON Q&A pairs to structured Markdown
- **Image Upload** — Batch upload local images to PicGo, auto-replace links and backup originals
- **Image Download** — Batch download remote images locally with MD5 deduplication
- **Link Extraction** — Extract all image links in text or JSON format

### PDF Processing

- **Image Extraction** — Extract images from PDF with PNG/JPEG support, page ranges, and batch mode
- **PDF Splitting** — Split PDFs by page range, fixed page count, or single page

### XMind Processing

- **XMind to MD** — Convert XMind mindmaps to Markdown, preserving hierarchy and links

## Installation

**Download from Release** (recommended)

Download prebuilt binaries from [Releases](https://github.com/cicbyte/daxe/releases) for Windows / Linux / macOS.

**go install**

```bash
go install github.com/cicbyte/daxe@latest
```

**Build from source**

```bash
git clone https://github.com/cicbyte/daxe.git
cd daxe
go build -o daxe
```

## Quick Start

```bash
# AI-fix Markdown formatting
daxe md fix ./document.md

# Extract images from PDF
daxe pdf images input.pdf -o ./images --pages 1,3,5-8

# Split PDF every 10 pages
daxe pdf split input.pdf -o ./output --every 10

# Convert XMind to Markdown
daxe xmind md mindmap.xmind
```

## Command Reference

| Module | Command | Description | Docs |
|--------|---------|-------------|------|
| `daxe md` | `fix` | AI-powered Markdown format fixing | [docs/md.md](docs/md.md) |
| | `convert` | Convert JSON Q&A to Markdown | |
| | `upload` | Upload local images to PicGo | |
| | `download` | Download remote images locally | |
| | `extractLinks` | Extract image links | |
| `daxe pdf` | `images` | Extract images from PDF | [docs/pdf.md](docs/pdf.md) |
| | `split` | Split PDF files | |
| `daxe xmind` | `md` | Convert XMind to Markdown | [docs/xmind.md](docs/xmind.md) |

```bash
daxe --version                # Show version
daxe <command> --help         # Show command help
daxe <command> <sub> --help   # Show subcommand help
```

## Configuration

Config file: `~/.cicbyte/daxe/config/config.yaml`, auto-created on first run.

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

## Tech Stack

- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [UniPDF](https://github.com/unidoc/unipdf) — PDF image extraction
- [pdfcpu](https://github.com/pdfcpu/pdfcpu) — PDF splitting
- [Eino](https://github.com/cloudwego/eino) — AI model invocation
- [GORM](https://gorm.io) + [SQLite](https://github.com/glebarez/sqlite) — Data persistence (pure Go, no CGO)
- [Zap](https://github.com/uber-go/zap) — Structured logging

## License

[MIT](LICENSE)
