# 本工作区开发环境

准备日期：2026-11-19。适用于当前 Linux amd64 Harness 工作区。

- 仓库：<https://github.com/barry130/LevNote>
- 工作目录：`/data/workspace/harness/home/LevNote`
- 分支：`main`
- 拉取提交：`2ebec3d8ef47338f4efa2f913cc64015f61253d6`
- 当前主线：`native/` 下的 Go + MyGo 原生 Surface。根目录 React/Tauri 实现是历史参考。
- 本次只做环境准备和基线检查，未修改产品源码、依赖版本、锁文件、旧门禁或历史证据，未恢复交接中的功能任务，也未提交或推送代码。

继续开发前先阅读 [交接入口](HANDOFF.md)、[当前原生状态](native/STATUS.md) 和 [原生约束](native/AGENTS.md)。本页记录本次新工作区的实际环境，不覆盖这些文档中的历史验证结果。

## 已准备的工具与依赖

| 项目 | 当前准备结果 |
| --- | --- |
| Go | 项目内 `.localtools/go1.27.3`，Linux amd64；保留系统 Go 1.27.1 |
| MyGo | 仓库 vendor 的 v0.8.3；补丁清单及全部 26 个条目的 SHA-256 已核验 |
| Node.js | 当前运行时 v24.22.2 |
| pnpm | 锁定 11.7.0；命令入口在 `.localtools/bin`，由 Corepack 管理 |
| 历史前端依赖 | 已按 `pnpm-lock.yaml` 使用 `--frozen-lockfile` 安装 |
| Linux 原生运行库 | GTK3、GDK、Cairo、Pango、中文字体和 Xvfb 已有；Surface 路径不要求 WebKitGTK |
| 外部服务 | 纯本地项目，无需数据库、后端服务或账号密钥 |

Go 归档来自官方发布清单与下载站。`go1.27.3.linux-amd64.tar.gz` 的已核验 SHA-256 为：

```text
65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40
```

工具链、包依赖、缓存、开发二进制和本次原始日志均放在已忽略的本地目录；重新克隆后需要重新准备。没有替换全局 Go，也没有改变全局 Go telemetry 设置。

## 日常开发命令

以下命令从仓库根目录运行。原 `.localtools/gate.sh` 固定旧 Windows 路径，作为历史交接保留；当前 Linux 使用新建的 `.localtools/native-go.sh`。

```bash
# 让当前 Bash shell 使用项目内 Go 与 pnpm；每个新 shell 执行一次。
source .localtools/env.sh

go version       # go1.27.3 linux/amd64
pnpm --version   # 11.7.0
go telemetry     # off

# 固定工具链、离线 vendor 和补丁校验；build/vet 无参数时使用六个产品根。
bash .localtools/native-go.sh build
bash .localtools/native-go.sh vet

# 按改动选择回归。
bash .localtools/native-go.sh test -count=1 -timeout=300s ./formats/drawio ./modules/graphview -run Clone

# 需要完整基线时使用六根，避免 handoff 中的历史源码快照。
bash .localtools/native-go.sh test -json -count=1 -timeout=300s \
  ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
```

包装器固定 `GOENV=off`、`GOWORK=off`、`GOFLAGS=-mod=vendor`、`GOPROXY=off`、`GOSUMDB=off`、`GOVCS=*:off`、`GOTOOLCHAIN=local`、`CGO_ENABLED=0`。缓存与临时目录位于 `.localtools/linux-cache/`。Go telemetry 使用该目录下的独立 `off` mode 文件，并已通过 `go telemetry` 回读确认。每次执行产品命令前检查 MyGo 补丁；拒绝 `./...`、非 vendor 模式及更换构建输入的参数。

不要执行 `go mod vendor` 覆盖现有补丁，也不要为通过 Linux 测试而削弱 Windows 断言。

### 开发二进制

本次 Linux 宿主与四个独立 worker 位于 `.localtools/build/linux-amd64/`；Windows GUI 交叉编译产物位于 `.localtools/build/windows-amd64/`。

```bash
# 重新生成 Linux 开发二进制（需先存在输出目录）。
bash .localtools/native-go.sh build \
  -o "$PWD/.localtools/build/linux-amd64/" ./cmd/...

# 本机有图形显示会话时运行原生宿主。
.localtools/build/linux-amd64/levnote

# Windows amd64 GUI 交叉编译。
GOOS=windows GOARCH=amd64 bash .localtools/native-go.sh build \
  -ldflags=-H=windowsgui \
  -o "$PWD/.localtools/build/windows-amd64/" ./cmd/...
```

这些是开发二进制，未生成正式发行清单、完整随包布局或新的接受版本。Linux 宿主中的固定引擎包管理与进程运行路径仍有 Windows 平台限制；独立 Linux worker 的命令行成功不代表这些宿主能力已支持 Linux。

### 历史前端

```bash
source .localtools/env.sh
pnpm run typecheck
pnpm run build
# 如需查看历史浏览器预览，再手动启动：
pnpm run dev
```

本次没有保留运行中的前端开发服务器。历史 Tauri Linux 桌面打包所需的 WebKitGTK 开发环境不属于当前原生主线的准备结果。

## 本次验证结果

原始命令、分离的 stdout/stderr、时间和退出码保存在 `.localtools/setup-20261119-linux/`。该目录是本次新建的环境证据，未复用历史运行目录。

| 检查 | 结果 |
| --- | --- |
| MyGo 补丁完整性 | 固定清单及 26 个文件全部匹配 |
| draw.io 克隆定向回归 | 45 PASS、0 FAIL；包含交接中最后失败的两个测试 |
| 原生六根 Linux build | exit 0 |
| 原生六根 Linux vet | exit 0 |
| Linux 五目标开发构建 | exit 0 |
| Windows amd64 五目标 GUI 交叉编译 | exit 0；五个文件均确认为 PE32+ GUI x86-64 |
| 四个独立 Linux worker 进程 | 均 exit 0、协议 status=ok，成功生成 HTML、SVG 或导图场景 |
| Linux 宿主启动 | 在隔离配置和 Xvfb 中观察到原生窗口；通过 SIGTERM 结束并回收所属宿主进程 |
| 历史前端 typecheck/build | 通过；首次 build 因子命令找不到 pnpm 失败，加入本地 PATH 后通过，首份失败日志保留 |
| 原生六根完整测试 | exit 1；2349 PASS / 103 FAIL / 15 SKIP（含子测试），另有 2 个已开始测试没有终态；20 包通过、5 包失败、2 包无测试 |

宿主启动日志明确显示没有 GPU，使用软件绘制；这里只验证创建窗口，不验证正常 UI 关闭、恢复、物理输入、中文 IME、剪贴板、显示缩放或性能。Windows 交叉编译也不能替代 Windows 实机验收。

完整测试执行于 2026-11-19 19:29:20–19:34:43（Asia/Shanghai）。失败包为 `app`、`core/plugins`、`core/session`、`modules/graphview` 和 `modules/markdownview`。已观察到 Windows 绝对路径与固定引擎包平台相关拒绝、符号链接检查失败和图形/Markdown UI 断言失败；Markdown 表格测试出现 nil pointer panic，app 在 300 秒上限触发 timeout，未跑完全部用例。不能把所有失败自动归为平台差异。本次保留失败和原断言，后续应按具体开发任务分别定位。

详细测试身份、包结果、panic/timeout 与统计见 [基线汇总](.localtools/setup-20261119-linux/baseline-summary.json)，原始事件见 [完整测试日志](.localtools/setup-20261119-linux/native-tests.stdout.log)。不将交接中历史 Windows 的通过数当成本工作区结果。`.29` 正式接受包和现有体验包保持原状态，GitHub CI 保持关闭。
