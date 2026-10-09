# 0038 · 项目改名：轻墨 Qingmo → 轻笺 LevNote

日期：2026-10-09
状态：已采纳
关联：0021（位置记忆与设置）、0025（导图内核接入）、0004（内核选型）、0028（主包拆分）、0031（i18n 骨架）

## 背景

用户要求把项目改名：中文名 **轻笺**，英文名 **LevNote**（0.3.28）。

这不是改一行标题那么简单。旧名 `qingmo` / `轻墨` 一共出现在 **95 个文件、234 处**，
且分布在四类语义完全不同的位置 —— 混在一起改会踩到两种完全不同性质的坑：

1. **改了没人察觉，但用户的东西读不到了**（localStorage 键、`.mindmap` 的 `format` 字段）；
2. **改了立刻炸**（包名、crate 名、应用标识符、TS 路径映射）。

更要紧的是时机：这个项目**还没发布**。`git remote -v` 为空，没有上游二进制，
ROADMAP 上 P7 的发版项（32/33/39）全部待办，仓库里也没有任何一个已跟踪的
`.mindmap` 文件。也就是说**改名此刻的成本最低** —— 一旦分发出去，
下面第 2 条的兼容问题就再也绕不开了。

## 决策

### 1. 一次改到底，包括内部标识符

不保留 `qingmo` 兼容层、不留双名过渡期、不做映射表。四类内容一起改：

| 类别 | 例子 | 改了会怎样 |
|---|---|---|
| 界面文案 | `index.html` 的 `<title>`、`core/i18n/messages.ts` 的 `ui.app.appName` | 界面上还是旧名 |
| **持久化键** | `levnote.settings.v1` / `.positions.v1` / `.session.v1` / `.recentFiles` | **老用户设置、位置、会话、最近文件全部读不到** |
| **文件格式 id** | `MINDMAP_FORMAT = 'levnote.mindmap'` | **老 `.mindmap` 打不开，静默降级到代码内核**（`parse.ts` 的 `WRONG_FORMAT` 是致命项） |
| 构建与应用身份 | `package.json` / `Cargo.toml` / `identifier` / `qingmo-print` 协议 / `@levnote/interop-spec` | 构建期或运行期直接失败 |

**为什么不留兼容层**：兼容层（读旧键、认旧 `format`）本身是有长期成本的代码 ——
每个读键的地方都要多一次 fallback，每个解析器都要多认一个格式 id，
而它保护的**是一批还不存在的用户**。等项目真发出去、真有老用户时再补也不迟，
那时补的成本和现在一样清楚。

**执行方式**：批量替换写成一个可复核的一次性脚本而不是手改 234 处 ——
大小写敏感（仓库里同时有 `qingmo` / `QINGMO` / `Qingmo` 三种写法），
批量改可复核、可重跑、可 dry-run，比手点 95 个文件可靠。脚本带 UTF-8 BOM
（宿主的 shell 是 Windows PowerShell 5.1，无 BOM 的中文脚本会被当 ANSI 读成乱码）。
脚本**刻意跳过** 5 个含持久化键与格式 id 的文件（`settings.ts` / `positions.ts` /
`session.ts` / `recent.ts` / `mindmap/types.ts`），那些逐个人工确认 ——
「改了会静默丢用户数据」的地方不该由一条正则批量处理。
脚本在改完、验证并提交后即删除（它是一次性工具，留在仓库里只会让下一次改名
有人「跑一下试试」，而它硬编码的是这一次的替换表）。

### 2. 存储键与格式 id 也改，接受「零兼容」

第 2 条是本 ADR 唯一真正有代价的取舍，明确记下来：

- 改名后，**改名之前**用过的 `qingmo.*` 键与 `format: "qingmo.mindmap"` 的文件
  不会被识别。设置回到默认、位置与会话不恢复、旧导图打不开。
- 依据：此刻没有用户。发版是下一步的事，兼容包袱留到有真实用户时再评估。

> **重审条件**：一旦项目对外分发且产生了真实用户数据（尤其是别人手上已有
> `.mindmap` 文件），本条必须重审 —— 那时要回答的问题变成
> 「双读旧键 / 双认旧 format」的维护成本，是否低于「升级后用户的东西消失」的代价。
> 届时倾向于**只给 `format` 加兼容**（文件会外发给收件人，`WRONG_FORMAT`
> 直接让人打不开），而 localStorage 键可以给一次性迁移。

### 3. 类名 / 标识符一并对齐大小写

`QINGMO_VERSION` → `LEVNOTE_VERSION`、`QINGMO_THEME` → `LEVNOTE_THEME`、
`qingmo-*` 的 PluginKey 与 `SILENT_META` / `USER_EDIT_META` 一并改。
这些是内部标识符，改动**零外部影响**（不落盘、不进文件、不跨进程），
但留着会让「这个名字到底叫什么」在搜索里永远有两个答案。
注意 `SILENT_META` / `USER_EDIT_META` 属于**硬约束 7** 的实现 ——
它们只是事务 meta 的字符串值，不跨进程、不是协议，两端同时改就没有兼容问题。

`qingmo-print` 自定义协议（打印 / 导出 PDF，ADR 0030）改为 `levnote-print`。
它是**应用内部**的：前端用同一个常量生成 URL、Rust 侧注册同一个 scheme，
两端同一次提交里改掉即可。注意 `print_html` 必须保持 async 命令（ADR 0030 §6），
那与协议名无关。

### 4. `AGENTS.md` 里的坑 6 一并更新

手册第 6 条给的是 `cargo clean -p qingmo`。改名后 crate 叫 `levnote`，
**照抄旧名字会让那条命令直接失败**（clean 一个不存在的包）。
这类「文档里写着一个会失效的命令」比过时的事实更危险，所以它在批量替换的范围内。

## 顺带修掉的一个坑

改名过程中撞见 `src-tauri/target/` 里**整片缓存指向项目的旧路径**
（`F:\AIproject\qingmo\...`）—— 目录早已从 `qingmo` 改成 `LevNote`，
而 cargo 的构建产物把绝对路径写进了指纹。表现是 `cargo test` 报
`failed to read plugin permissions: 系统找不到指定的路径`（os error 3），
`pnpm` 的 `.bin` 垫片同理（`NODE_PATH` 写死旧路径，`node_modules\@typescript\native`
是个空目录）。两者都**只能整个重建**：包管理器与构建系统都认为「已是最新」，
不会去修一个指向旧路径的链接。

教训（值得推广）：**项目目录改名后，`node_modules` 与构建缓存目录要一起重建**，
否则会出现「`pnpm install` 说 up to date，但每个命令都报 MODULE_NOT_FOUND」这种
自相矛盾的现象。

## 后果

- 版本号 0.3.27 → **0.3.28**（`package.json` / `Cargo.toml` / `Cargo.lock` /
  `tauri.conf.json` 四处同步，沿用 0.3.25 确立的四处统一）。
- `identifier` 由 `app.qingmo.desktop` 改为 `app.levnote.desktop`
  —— 它决定 Windows 的 AppUserModelID 与安装目录，**发布后不可改**，故与发版项一起处理。
- 内置作用域包 `@qingmo/interop-spec` 改为 `@levnote/interop-spec`
  （`tsconfig.json` 路径映射、`vite.config.ts` alias、`block-ops.ts` 的 import 三处同步）。
- 文档里对旧名的散文引用（历史 ADR、`format-spec.md`、`CHANGELOG.md` 等）一并更新 ——
  ADR 记录的是**当时的决策**，但「当时的那个项目叫什么」不是决策内容，
  留着旧名只会让搜索「轻笺」搜不到自己的决策记录。
- `docs/open-decisions.md` 里 P7-36/37（商标与域名）尚未结项，
  类目与域名检索必须按**新名**重新查一遍。
- 备份与回滚：改动前的 `git diff` 快照存在 `%TEMP%\levnote-pre-rename.patch`。

## 重审条件

- 一旦对外分发并出现真实用户数据 → 第 2 条必须重审（见上）。
- 若引入「按产品名做多实例区分」的需求（如同时装两个版本），
  `identifier` 与 `AppUserModelID` 的关系要重新设计；那时 `identifier` 不能再改。