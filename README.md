# daxe

> **d**ata + **axe** — 快刀斩乱麻，专业的文件处理 CLI 工具集。

支持 Markdown 处理、PDF 图片提取和 XMind 思维导图转换。

## 安装

**下载预编译二进制**（推荐）

从 [Releases](https://github.com/cicbyte/daxe/releases) 下载对应平台的可执行文件。

**从源码编译**

```bash
go build -o daxe
```

**go install**

```bash
go install github.com/cicbyte/daxe@latest
```

## 功能

| 模块 | 说明 | 文档 |
|------|------|------|
| `daxe md` | Markdown 处理（修复、转换、图片管理、链接提取） | [docs/md.md](docs/md.md) |
| `daxe pdf` | PDF 图片提取 | [docs/pdf.md](docs/pdf.md) |
| `daxe xmind` | XMind 转 Markdown | [docs/xmind.md](docs/xmind.md) |

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

## License

[MIT](LICENSE)
