# task-2 · 指定 box 原字节传输调查结果

日期：2026-11-11。执行成员：remote-transfer。结论：**辅助调查完成，实际传输未完成；没有上传成功回执。**

## 任务与边界

- 原 Codex 主会话仍为 `01a1b6b1-3caf-76b1-81e2-183c090b7b86`，本任务不替代或启动它。
- 已通过共享任务 list/get 读取完整约束并认领 task-2。唯一目标为官方 MCP server `box` 的 `main / cb-698f697fed7c`。
- 已完整读取[准备结果](<F:/AIproject/LevNote/native/handoffs/remote-box-source-preparation-20261111.md>)、[02 版传输清单](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/transfer-files-02.json>)及[02 版 runbook](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/remote-runbook-02.md>)。没有输出 ZIP 或 244 KB 输入清单的载荷。
- 本机显式写入仅限本报告所在目录；不修改准备目录、源码、项目约定或旧证据。没有重放 Lead 已失败的命令，没有开启新本机 Shell，没有读取 credential/env/auth 文件或令牌。

## 实际核对，不是远端收件证据

本机 `ops_stat` 的 SHA-256 revision 与原始字节数均与 02 清单一致：

| 本机文件 | 实算字节 | 实算 SHA256 |
|---|---:|---|
| [02 ZIP](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/levnote-source-inputs-20261111-02.zip>) | 5227577 | `457503deecbcf0a534cc3240027048454bdf758076e0ee8d3a0a64f7773db7f9` |
| [inputs-02.json](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/inputs-02.json>) | 244715 | `a51734c0e9b40051ad1ba06ad4b5e0acc4d92ef1a3de5261259fa435b5df2e51` |
| [Linux 守卫草稿](<F:/AIproject/LevNote/native/handoffs/remote-box-preparation-20261111/linux-guarded-validation-02.py>) | 33772 | `7d53c6b7374a52c9d3e6749e423449d073778ac1c33c49888a121b97d43e17d7` |

官方 `box_list` 和显式 `box_status(box="cb-698f697fed7c")` 均返回 `name=main, state=running, image=sbxdesk:v15`；连接 URL 为 `https://mcp.box.simular.ai/mcp/{{SECRET_htygbg2l}}

**本任务传输调用数为 0；没有创建远端 inbox，也没有运行远端文件长度或 SHA256 收件验收。三个文件的远端路径、字节数及哈希均未取得，不得用上表代替。**

## 已调查的官方公开证据

1. [box 官网工具清单](<https://box.simular.ai/#tools>)明确将 Shell & files 描述为 “reading, writing and editing files on the box”，列出 `shell/read_file/write_file/edit_file/list_dir`。当前实际工具 schema 中 `write_file` 接收文本 `content`，`shell` 接收远端 `command`，均无本机 `local_path` 或原始二进制 stream 参数。
2. [box 官方 MCP 文档](<https://box.simular.ai/docs/mcp.md>)说明官方 Streamable HTTP MCP 连接与授权；本次读取未找到写入 box inbox 的本机字节上传操作。官方[文档索引](<https://box.simular.ai/docs/llms.txt>)当前只列 Introduction、Install、MCP、Headless、Secrets 和 Troubleshooting，没有传输指南。这是当前文档与接口的有界观察，**不是断言整个产品永远没有上传功能**。
3. 官网链接的 [Sai 官方 Upload 文档](<https://platform.simular.ai/documentation/api-reference/upload>)确实提供二进制上传：`POST /v1/agents/upload`，raw body，`Authorization: Bearer $SAI_API_KEY`，`x-filename`，单文件上限 25 MB，并要求 `--data-binary` 保留换行。其返回的是附加给 `POST /message` 的 attachment 对象，可能改名；文档没有 `cb-698f697fed7c` 或指定 `/home/user` inbox 的参数和绑定证明。它不是当前已授权 box MCP 的上传操作。本任务未调用该 API、未创建或读取 API key、未把 box/view-only token用于它。
4. Native 能力检索“本机二进制文件按路径私有上传到官方 box 连接器，保持原字节及远端 SHA256”返回 `No allowed, implemented capability matched.`。这是本会话可激活能力池的结果，不是全局产品清单。

公开页面均按已发现的官网链接读取，没有猜测写入 endpoint。通用 web_fetch 的 provider 不可用后，改用现有 box 只读网页工具；Markdown MIME 与随后 Chrome CDP 超时后，仅用指定 box 现有 shell 的 `python3 -I -B`/`urllib.request`读取精确公开 URL，没有写文件或导入凭据。

公开文档实测身份（HTTP body，不是源文件）：

| URL | UTC 采样 | HTTP | 原 body 字节 | SHA256 | 远端只读命令 exit |
|---|---|---:|---:|---|---:|
| `https://box.simular.ai/docs/mcp.md` | `2026-11-11T03:25:17.167667+00:00` | 200 | 12901 | `66d92ffbb888017d57497a1cc79a6b7e0de84d0941ddc70e104281fdb06131c9` | 0 |
| `https://platform.simular.ai/documentation/api-reference/upload` | `2026-11-11T03:26:02.469025+00:00` | 200 | 68358 | `57c89008c5b954ed5e344e2d1a26f5e4a92ba3224b691c0b037c7eebe3015c18` | 0 |

此处为工具事实转录及文档 body 身份，不是原始二进制进程流留档，也不是传输 receipt。

## 具体阻塞与最小所缺接口

**缺少一个当前可调用、已授权的 opaque-byte 桥接接口**，同时满足：

1. 按本机路径直接消费三文件原字节，不把 ZIP/大清单全读进模型文本，不经重编码或文本重建。
2. 通过既有官方 box 连接器明确绑定 `cb-698f697fed7c`，不读取凭据、不新建 box、不另起公共或私有中转服务、不改插件或全局配置。
3. create-only 创建全新 `/home/user/levnote-inbox-20261111-<新后缀>`，不覆盖任何旧文件。
4. 传完后允许在同一明确目标内实际核对三个原始文件的精确长度和完整 SHA256，并保留真实收件证据。

本机元数据/UTF-8/hex读取与远端文本写入之间，没有现成的 opaque-byte 引用/流转接口。不能把两端各有文本工具推断成已经具备安全原字节传输；也不能借另一个需 key 的 Sai 附件 API或 view-only token扩大权限。

## 交回与下一步

调查到此停止。回交 lead，由指定 Codex 原会话继续主线。用户已经授权直接上传，不需要再问许可，也不要求用户手动搬文件。只有宿主提供符合上述要求的既有授权字节桥接，才能继续实际迁移。

本次没有工具链下载/安装、guard prepare、语法检查、任何 Go 命令、test/vet/build、产品/worker/测量、公开 source 上传、服务启动、插件/配置变更。产品阶段、.30 和原有 Windows 证据状态不变。共享任务 completed 仅表示这份辅助调查已交付，**不表示三文件上传成功**。
