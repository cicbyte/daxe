# daxe xmind

XMind 思维导图处理工具集。

## 命令

### md — XMind 转 Markdown

将 XMind 文件转换为 Markdown 格式，保持层级关系和链接信息。

```bash
daxe xmind md mindmap.xmind                   # 转换单个文件
daxe xmind md mindmap.xmind -o output.md      # 指定输出文件
daxe xmind md ./xmind_files/ -o ./output/ -t 4  # 批量转换
daxe xmind md -i files.json -o ./output/      # 从文件列表转换
```

| 参数 | 说明 |
|------|------|
| `-t, --threads N` | 并发线程数（默认1，最大100） |
| `-l, --list FILE` | 文件列表路径（JSON/TXT） |
| `-o, --output PATH` | 输出路径（单文件时为文件路径，多文件时为目录） |

输入文件列表格式：
- **TXT**：每行一个文件路径或目录路径
- **JSON**：字符串数组，如 `["/path/to/file1.xmind", "/path/to/dir/"]`

输出使用 Markdown 标题层级（H1-H6），子主题使用列表结构，保留原始链接和标记信息。
