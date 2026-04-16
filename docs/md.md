# daxe md

Markdown 文件处理工具集。

## 命令

### fix — AI 智能修复

修复 MD 文件中的格式和语法错误。

```bash
daxe md fix ./file.md                          # 修复单个文件
daxe md fix ./files/ -t 4                      # 批量修复，4线程
daxe md fix -i files.json                      # 从文件列表修复
```

| 参数 | 说明 |
|------|------|
| `-t, --threads N` | 并发线程数（默认1，最大100） |
| `-l, --list FILE` | 文件列表路径（JSON/TXT） |

修复范围：Markdown 语法错误、Mermaid 图表语法、格式不一致、表格语法、标点符号和编码问题。

### convert — JSON 转 MD

将 JSON 格式的问答对转换为 MD 结构，按分类组织输出。

```bash
daxe md convert ./qa.json                      # 转换单个文件
daxe md convert ./qa_files/ -o ./output        # 转换目录并指定输出
daxe md convert -i qa_list.json                # 从文件列表转换
```

| 参数 | 说明 |
|------|------|
| `-t, --threads N` | 并发线程数（默认1，最大100） |
| `-l, --list FILE` | 文件列表路径（JSON/TXT） |
| `-o, --output DIR` | 指定输出目录 |

### upload — 图片上传

上传 MD 文件中的本地图片到云存储（PicGo），自动替换链接并备份原文件。

```bash
daxe md upload ./doc.md                        # 上传单个文件
daxe md upload ./documents/ -t 4               # 批量处理，4线程
daxe md upload ./doc.md --picgo-server http://192.168.1.100:36677
```

| 参数 | 说明 |
|------|------|
| `-t, --threads N` | 并发线程数（默认1，最大100） |
| `--picgo-server` | PicGo 服务器地址（默认使用配置文件） |

### download — 图片下载

下载 MD 文件中的远程图片到本地，保存在同名子目录中。

```bash
daxe md download ./doc.md                      # 下载单个文件
```

| 参数 | 说明 |
|------|------|
| `-t, --threads N` | 并发线程数（默认1，最大100） |

图片保存为 URL 的 MD5 哈希文件名，同目录下生成 `link_mappings.json` 记录映射关系。

### extractLinks — 链接提取

提取 MD 文件中的所有图片链接。

```bash
daxe md extractLinks ./doc.md                  # 文本格式输出
daxe md extractLinks ./doc.md -f json          # JSON格式输出
```

| 参数 | 说明 |
|------|------|
| `-f, --format` | 输出格式：`text` 或 `json`（默认 `text`） |
