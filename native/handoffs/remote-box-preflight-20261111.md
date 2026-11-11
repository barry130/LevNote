# DSH 远程 box 只读预检 · 2026-11-11

## 范围与结论

- 任务：`task-1`，由 `remote-preflight` 按 revision 1 认领，认领后 revision 2；只负责 DSH 辅助预检，不代替 Codex 原主会话 `01a1b6b1-3caf-76b1-81e2-183c090b7b86`。
- 已完整读取 [Codex 接续计划](<F:/AIproject/LevNote/native/handoffs/codex-continuation-current-route-20261111.md#L1-L61>)，执行其第 59–61 行的预检请求。该说明文件为 61 行、6149 字节，读取时 SHA-256 为 `207ec0773bb0365d9edd25d65f8fda7dae52b877217d219c3b4b8e8401cb7612`。
- 唯一远程目标为 `main / cb-698f697fed7c`。所有远程观察均通过已连接 `mcp__box__box_status`、`mcp__box__shell`、`mcp__box__list_dir`，每次显式传入该 ID；没有新建、更换或升级 box。
- 本机仅通过文件工具读取项目说明、检查报告路径并新建本报告；未运行本机门禁。创建前精确路径 `ops_stat` 返回 `exists=false; revision=absent`，使用 `expectedRevision=absent` 创建，不覆盖既有报告。
- **预检观察已完成，但执行前守卫尚未完成。共享库和字体可用不等于 MyGo 已选择有效文本后端，不等于 UI 或 Windows 验收通过。**
- 没有执行测试、vet、构建、产品启动、安装、Go 更换、源码修改、凭据读取或原字节文件迁移；未创建远端运行目录。旧 Windows 证据及已接受 `.29` 交付未触碰。

## 已证实的远程事实

三个 shell 请求均返回工具级 `[exit status 0]`。下列 UTC 是各请求在远端打印的起始时间，不伪作完整结束时间。

| 项目 | 实测结果 | 证据时间 / 方法 |
|---|---|---|
| box 身份 | `box_id=cb-698f697fed7c; name=main; state=running; image=sbxdesk:v15` | 定向 `box_status` |
| OS / 架构 | `Ubuntu 24.04.5 LTS`、`VERSION_ID=24.04`、`noble`；`uname -sm` 为 `Linux x86_64`，Debian 包架构 `amd64` | `02:31:29Z` / `02:32:26Z` |
| user / home / cwd | `user=user; HOME=/home/user; cwd=/workspace` | `02:31:29Z` |
| DISPLAY | `DISPLAY=:0`；`/tmp/.X11-unix/X0` 存在且为 socket | `02:31:29Z`；只查环境和文件元数据，未连接显示服务器 |
| Go 本地状态 | PATH 无 `go`；`dpkg-query` 没有匹配 `golang*` 的包；下列 8 个常规候选目录均不存在 | `02:31:29Z`；未执行任何 Go 二进制 |
| 运行时共享库 | 下列 13 个精确缓存路径均成功 `ctypes.CDLL(path)` | `02:32:26Z`；仅加载库，未初始化 GTK / 产品 |
| 开发元数据工具 | `shutil.which('pkg-config')` 返回 `None` | `02:32:26Z`；未安装，未尝试编译探测 |
| 字体 | `fc-list` 返回 161 个 face 记录、135 个不同字体文件、35 个不同 family 字段值；中文匹配筛选有 30 个 face 记录 | `02:32:26Z`；使用 fontconfig 查询，未渲染产品 |
| 远端工作区 | `/workspace` 顶层只有 `flows/`；下列 7 个 LevNote 候选目录均不存在 | `02:31:29Z`；没有扫描无关目录 / 全盘 |
| 磁盘 | `/workspace`、`/home/user`、`/tmp` 均在同一 `overlay` 挂载 `/`；总 `20957446144`、已用 `231669760`、可用 `20708999168` 字节，使用率 `2%` | `02:31:29Z`；可用量不可将三条重复相加 |

### Go 1.27.3 官方 Linux 归档

从该 box 直接读取 [Go 官方发布清单](<https://go.dev/dl/?mode=json&include=all>) 并对 [官方 Linux amd64 归档](<https://go.dev/dl/go1.27.3.linux-amd64.tar.gz>) 发出 HEAD；UTC `2026-11-11T02:32:27Z`。

```text
official_manifest HTTP=200
matching_record_count=1
filename=go1.27.3.linux-amd64.tar.gz
version=go1.27.3; os=linux; arch=amd64; kind=archive
size=70588678
sha256=65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40
manifest_matches_parent_SHA256=True
archive_HEAD HTTP=200
final_url=https://dl.google.com/go/go1.27.3.linux-amd64.tar.gz
Content-Length=70588678
Content-Type=application/x-gzip
archive_body_downloaded=False
archive_SHA256_computed=False
filesystem_written=False
Go_executed=False
```

**以上证明官方清单存在相应记录、远端可达该归档 HEAD，且清单值与父级提供值一致；没有证明已下载归档的实际字节哈希或安装后的工具链身份。**

### Linux 共享库与字体

以下为实际 `ldconfig` 缓存路径解析后的真实路径，13 项加载结果均为 `OK`；不据此推断 Go 的平台选中文件或 MyGo 后端选择。

```text
libpango-1.0.so.0       /usr/lib/x86_64-linux-gnu/libpango-1.0.so.0.5200.1
libpangocairo-1.0.so.0  /usr/lib/x86_64-linux-gnu/libpangocairo-1.0.so.0.5200.1
libglib-2.0.so.0        /usr/lib/x86_64-linux-gnu/libglib-2.0.so.0.8000.0
libgobject-2.0.so.0     /usr/lib/x86_64-linux-gnu/libgobject-2.0.so.0.8000.0
libgio-2.0.so.0         /usr/lib/x86_64-linux-gnu/libgio-2.0.so.0.8000.0
libgtk-3.so.0           /usr/lib/x86_64-linux-gnu/libgtk-3.so.0.2409.32
libgdk-3.so.0           /usr/lib/x86_64-linux-gnu/libgdk-3.so.0.2409.32
libcairo.so.2           /usr/lib/x86_64-linux-gnu/libcairo.so.2.11800.0
libcairo-gobject.so.2   /usr/lib/x86_64-linux-gnu/libcairo-gobject.so.2.11800.0
libepoxy.so.0           /usr/lib/x86_64-linux-gnu/libepoxy.so.0.0.0
libharfbuzz.so.0        /usr/lib/x86_64-linux-gnu/libharfbuzz.so.0.60830.0
libfontconfig.so.1      /usr/lib/x86_64-linux-gnu/libfontconfig.so.1.12.1
libfreetype.so.6        /usr/lib/x86_64-linux-gnu/libfreetype.so.6.20.1
```

定向包查询退出码 0；以下每项均为 `amd64 / install ok installed`：

```text
libpango-1.0-0       1.52.1+ds-1build1
libpangocairo-1.0-0  1.52.1+ds-1build1
libglib2.0-0t64      2.80.0-6ubuntu3.9
libgtk-3-0t64        3.24.41-4ubuntu1.3
libcairo2           1.18.0-3build1
libepoxy0           1.5.10-1build1
libharfbuzz0b        8.3.0-2build2
libfontconfig1      2.15.0-1.1ubuntu2
libfreetype6        2.13.2+dfsg-1ubuntu0.2
```

实际字体匹配，三次 `fc-match` 退出码均为 0：

```text
sans-serif              DejaVu Sans      /usr/share/fonts/truetype/dejavu/DejaVuSans.ttf
monospace               DejaVu Sans Mono /usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf
sans-serif:lang=zh-cn    Noto Sans CJK SC  /usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc
```

fontconfig 中文筛选样本还包含 Noto Sans Mono CJK SC 等家族；此处不罗列全部字体。未做 shaping、IME、截图或产品视觉检查。

### 限定路径观察与远端输入清单状态

以下路径仅检查存在性，全部 `exists=false`，不含递归扫描：

```text
Go candidates:
/usr/local/go
/usr/lib/go
/usr/lib/go-1.27
/opt/go
/home/user/go
/home/user/.local/go
/workspace/go
/home/user/tools/go

LevNote source candidates:
/workspace/LevNote
/workspace/levnote
/workspace/native
/home/user/LevNote
/home/user/levnote
/home/user/AIproject/LevNote
/home/user/projects/LevNote

mcp__box__list_dir(/workspace):
/workspace:
flows/
```

- 已观察的 `/workspace` 不存在 LevNote 原生源码树；上述 home 常规候选也不存在。未将“这些路径未发现”扩大为“整个 box 绝无源码 / Go”。未扫描 `flows/`、其他 home 子目录、隐藏配置或凭据目录。
- 本轮没有定位到可绑定的远端 LevNote 运行根，因此没有远端源码、vendor、冻结交接或完整输入清单身份可核对。迁移及完整清单核对 **尚未执行**，不能把缺少远端清单表述为已通过身份守卫。
- [Codex 接续计划的五文件身份](<F:/AIproject/LevNote/native/handoffs/codex-continuation-current-route-20261111.md#L13-L25>)及 [26 项补丁守卫基线](<F:/AIproject/LevNote/native/handoffs/codex-continuation-current-route-20261111.md#L41-L49>)仅为本机既有交接，不是本轮远端哈希结果。

## 最小实际命令与调用证据

除 `box_status` 和一次 `/workspace` 顶层目录读取外，只执行三个只读 shell 请求。以下保留实际命令及实际 Python 调用的关键原语；输出对应上文，不是待执行的产品脚本。

第一请求（`02:31:29Z`）：

```bash
date -u '+%Y-%m-%dT%H:%M:%SZ'
uname -sm
pwd
id -un
printf 'HOME=%s\nDISPLAY=%s\n' "$HOME" "${DISPLAY-<unset>}"
command -v go
dpkg-query -W -f='${binary:Package}\t${Version}\t${Status}\n' 'golang*'
df -B1 /workspace /home/user /tmp
```

```text
go_on_PATH=<absent>
dpkg-query: no packages found matching golang*
dpkg_go_query_exit=1
```

Go 查询退出码 1 是上述“没有匹配包”的原始结果；未隐去或改成成功。其余存在性、OS 字段和 socket 元数据由同一请求的 `python3 -B` 使用下列原语读取：

```python
pathlib.Path('/etc/os-release').read_text()
os.path.lexists(p)
(d / 'bin/go').is_file()
stat.S_ISSOCK(os.stat('/tmp/.X11-unix/X0').st_mode)
```

第二请求（`02:32:26Z`，在 `python3 -B` 内执行）：

```python
subprocess.check_output(['dpkg', '--print-architecture'], text=True)
subprocess.check_output(['/sbin/ldconfig', '-p'], text=True)
ctypes.CDLL(paths[0])  # 逐个传入上述13项的精确缓存路径
subprocess.run(['dpkg-query', '-W', '-f=${binary:Package}\t${Version}\t${Architecture}\t${Status}\n', *packages], capture_output=True, text=True)
shutil.which('pkg-config')  # None；因此未执行pkg-config子命令
subprocess.run(['fc-list', '--format', '%{file}\t%{family}\n'], capture_output=True, text=True)
subprocess.run(['fc-list', ':lang=zh-cn', '--format', '%{family}\n'], capture_output=True, text=True)
subprocess.run(['fc-match', '--format', '%{family}\t%{file}\n', pattern], capture_output=True, text=True)
```

其中 `packages` 是上述 9 个包名；`pattern` 分别为三个已列出的字体模式。第三请求（`02:32:27Z`，在 `python3 -B` 内执行）：

```python
manifest_url = 'https://go.dev/dl/?mode=json&include=all'
archive_url = 'https://go.dev/dl/go1.27.3.linux-amd64.tar.gz'
urllib.request.urlopen(manifest_url, timeout=20)  # 只读JSON，筛选go1.27.3/linux/amd64/archive
urllib.request.urlopen(urllib.request.Request(archive_url, method='HEAD'), timeout=20)
```

## 未证实项与限制

1. **工具链未就绪**：没有发现可用 Go；没有下载、校验归档字节或安装。不排除未检查的非标准位置。没有 Go 版本执行结果或实际工具链身份。
2. **运行输入未就绪**：没有传入源码 / vendor / fixtures / helpers / 冻结记录，没有覆盖完整构建输入的路径、长度、SHA-256 清单，没有前后身份或 26 项补丁远端核对。
3. **平台与开发依赖未验**：PATH 无 pkg-config；开发包、头文件、编译链接配置及 Linux 选中文件未核验。不能仅因运行时库存在便宣称构建条件齐备，也不能断言 pkg-config 缺失一定会阻塞尚未选择的 MyGo 路径。
4. **显示与 UI 未验**：仅证实环境变量 / X socket 元数据和字体匹配，没有 X 连接、GTK 初始化、MyGo backend / stub 选择、实际文本绘制或 IME 验收；未读取 Xauthority 或其他凭据。
5. **原字节文件通道未实证**：已证实的是按 box ID 定向的 shell / 目录查询和结果读取。没有执行本机实际文件的原字节上传或远端结果文件下载，未证明字节 / 行尾保真、大小限制或端到端哈希一致。不能把文本参数 / 文本输出能力称作已验证的文件迁移通道。没有读取密钥，也没有使用公共上传站点。
6. **Windows 交付仍待定**：没有 Windows `.30` GUI PE、hidden-host 生命周期、物理 Windows IME 或五次构建结果；Linux 库加载 / 字体查询不是 Windows 或 UI 验收。现有历史结果不能标成 2026-11-11 新远程结果。
7. **磁盘仅为快照**：可用字节是观察时值；完整输入、工具链解包、缓存和证据尺寸尚未测算，不据此保证未来运行容量。

## 回交原 Codex 的待执行清单（只制定，未授权 / 未执行）

后续产品操作及平台取舍均回交原会话；本报告不扩大 DSH 预检任务范围。

- 先取得受控原字节传输能力及明确后续授权；在唯一 box 的新独立运行目录传入当前实际工作区所需源码、测试、vendor、模块记录、补丁、fixtures / helpers 和冻结交接。不能只 clone Git 或把五文件清单当作完整输入清单；逐项核对路径、字节数、SHA-256，保留原始行尾。
- 冻结五文件身份按 [接续计划第 19–25 行](<F:/AIproject/LevNote/native/handoffs/codex-continuation-current-route-20261111.md#L19-L25>)逐项核对；其权威 [冻结清单](<F:/AIproject/LevNote/native/handoffs/open-document-picker-key-correction-evidence-20261110/source-frozen.json>)为 748 字节，既有 SHA-256 `AB90451D3003E713E6A5260087FA36B0E32DC2B3EDA23DD517F8B0502E9B59F6`，不是本轮重算。
- 固定官方 Linux amd64 Go 1.27.3；后续下载时必须实算归档 SHA-256，与上述官方值严格比对，再记录安装后工具链身份。失败即停止，不降级、不自动换版。
- 独立包装器不得修改冻结源码；保持 `GOTOOLCHAIN=local`、`GOWORK=off`、`GOENV=off`、明确 `-mod=vendor`；工具链 / 缓存 / 临时目录 / 遥测状态隔离在远端本次目录。禁止替代输入参数、tidy、mod vendor 或依赖升级。
- 固定 MyGo `v0.8.3`；核对 startup 标记和禁止恢复的 cleanUpdateLeftovers 调用；补丁清单既有 SHA-256 `9113120BD2EBCD9A86842153239CDDFF84D1F1BD3F46F5D4F0F152EB4560E16A` 及全部 26 项均需在远端实核，vet 同样受守卫约束。
- 再核对实际 Linux 平台选择和非 stub 文本路径；Windows 路径断言 / build tags 差异如实保留，不能删除断言、加跳过或改冻结源码来换取通过。
- 只有上述守卫通过并获得产品执行阶段授权后，才由原会话安排一次目标 app test / vet；保留 UTC、分离的原始 stdout / stderr、退出码、平台选中文件和输入前后身份到新远端证据目录，不覆盖旧 Windows 证据。本轮没有执行这一步。
