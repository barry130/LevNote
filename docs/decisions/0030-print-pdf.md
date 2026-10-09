# 0030 · 打印 / 导出 PDF：交给打印管线，不自造排版引擎

日期：2026-10-04
状态：已采纳（P3 第 16 项落地）
关联：0001（自包含导出是唯一出口）、`core/export/document.ts`
（`buildSelfContainedHtml`，两种导出共用）、`src-tauri/src/export/print.rs`、
`ROADMAP.md` 第 16 项

## 背景

`export/mod.rs` 的模块注释从 P2 起就写着第三条导出路线：**Markdown → PDF
走 WebView 的打印管线**，复用渲染结果而不是另做一套排版引擎。第 16 项
要回答的是这条管线具体长什么样：

- tauri 2.12 / wry 0.57 **没有**把 WebView2 的程序化打印（`ICoreWebView2_16::PrintToPdf`）
  暴露出来 —— 只有 `print()`（弹系统打印对话框）。想直接产出 PDF 字节，
  得 unsafe 地够 COM 接口，且绑死 wry 内部结构，macOS / Linux 还要各写一条路。
- 自包含 HTML 内联了 base64 图片，几 MB 很平常。开个窗口展示这份 HTML 有两条路：
  **临时文件**（要管清理、权限、路径编码）或**内存里的自定义协议**（无大小限制）。

## 决策

### 1. 渲染完全复用：与「导出为自包含 HTML」共用 `buildSelfContainedHtml`

两种导出是「**共用渲染，分道出口**」：同一份 `buildSelfContainedHtml` 产物，
HTML 导出落盘成文件，打印交给打印窗口。导出 HTML 里那份 `@media print`
纸张样式（去纸边、投影、防截断分页）就是为这条管线准备的 —— 任何一方改
渲染，另一方自动跟上。

### 2. 打印对话框是出口，不是降级

前端把 HTML 交给 Rust 命令 `print_html` → Rust 开一个子窗口 → HTML 经
**自定义协议 `levnote-print://`** 从内存表喂给 WebView → 加载完成后注入脚本
调 `window.print()` 弹系统打印对话框 → 「另存为 PDF」（WebView2 内置目的地）
或选真打印机。

选对话框而不是程序化 PrintToPdf 的理由：wry 没暴露；unsafe COM 会绑死
wry 内部；而打印对话框是三平台共有的「打印管线」，还顺带给了真打印机
出口。代价是**输出路径与文件名由系统对话框接管**，前端不经过
`pickSavePath` —— 这与 HTML 导出的交互差异写进了 CHANGELOG。

### 3. HTML 走内存自定义协议，不落临时文件

`register_uri_scheme_protocol("levnote-print")` 在 `run()` 注册一次；打印
窗口的 URL 是 `levnote-print://localhost/<id>.html`，协议处理器按 id 从
`PRINT_PAGES` 内存表（`Mutex<BTreeMap>`）取 HTML。**窗口销毁（`Destroyed`）
即摘表项**；窗口创建失败也摘。404（表里没有）返回一个说清楚「打印会话
已失效，回主窗口重来」的页面 —— 硬约束 2，不静默。

平台差异收在处理器里只看 `uri().path()`：Windows 实际请求是
`http://levnote-print.localhost/<id>.html`，macOS / Linux 是
`levnote-print://localhost/<id>.html`，**path 两边长一个样**。

### 4. 打印窗口自带一条引导条（`@media print` 隐藏）

注入脚本在页面顶部加一条引导：「在打印对话框里选另存为 PDF；对话框没
自动出现就按 Ctrl+P」。它同时是 `window.print()` 失败时的**可见兜底**
（老 WebView2 运行时不支持时，用户仍知道怎么手动打印），打印时隐藏、
不会印到纸上。

### 5. 动作层与平台解耦：`printHtml` 注入

`core/export/print-document.ts` 是与 `exportHtml` 同构的动作：组装输入 →
`buildSelfContainedHtml` → 出口。出口能力 `printHtml(html, title)` 是注入
参数（provider 的同一条理由），桌面端由 `core/platform` 提供实现（唯一能
碰 `@tauri-apps/*` 的地方），测试注入假实现。缺项通报与 HTML 导出同一套
口径（读不到的图片 / 网络图片 / 渲染失败的公式，在完成提示里说出来）。

命令 `file.exportPdf` 在浏览器预览里**禁用**（`enabled` 同时要求
`isDesktop()`），比「点了报错」诚实；`printHtml` 里还有一道明确拒绝，
防别的入口直接调动作。

### 6. `print_html` 必须是 async 命令（2026-10-06 补，实测踩坑）

第一版把命令写成同步的（`pub fn print_html`），在 Windows 上炸出了本
feature 最刁的一次事故：**打印窗口一片空白、整个应用死掉、子窗口连 X
都关不掉**，只能任务管理器杀进程。

根因是**主线程死锁**：Tauri 的同步命令在**主线程**上内联执行，而
`WebviewWindowBuilder::build()` 要等 WebView2 的「环境已创建」回调 ——
那个回调得靠消息循环派发，消息循环此刻正被本命令占着，谁也等不到谁。
窗口框架（带标题）先建出来了，webview 永远初始化不完 → 白屏 +
事件循环死亡 → 什么都关不掉。

实测定位的过程也值得记一笔：先怀疑协议通道（体积、CSP、URL 改写），
给打印窗口注入探针脚本把页面真实状态回传，结果小页面秒开、大页面也
能开 —— 唯独**走真实 invoke 链路**时 `exit(0)` 都发不出去（事件循环
死了），把命令改成 async 后同一条链路 3 MB 照样加载。判断依据：
**在 Tauri 里开窗口的命令必须 async（或从非主线程调用）**——同步命令
的「主线程内联执行」与「窗口创建要等消息循环」天然互斥。

## 后果

- 用户第一次拿到「PDF 出口」：命令面板 → 「打印 / 导出为 PDF」→ 打印
  对话框 → 另存为 PDF。所见即所得由「与编辑器/导出同源的渲染」保证。
- 打印窗口是**真实子窗口**：可调大小、可反复打印（Ctrl+P）、关掉即清。
- PDF 的页眉页脚、纸张、缩放都由系统打印对话框控制 —— 我们不重复造
  这些轮子，也不在 HTML 里跟它抢。

## 重审条件

- **wry 暴露 PrintToPdf 的那天**（或 tauri 提供等价 API）：程序化产出
  字节 + `pickSavePath` 命名的直出流程比对话框更顺 —— 届时把
  `print.rs` 的出口换成它，前端动作与命令形状不变（注入点就是为此留的）。
- **引导条被打断**：如果实测打印对话框总是可靠弹出、引导条反而碍眼，
  可以把它收成只在不支持 `window.print()` 的环境里注入。
- **超大文档**：WebView2 的打印预览对几百页文档的内存占用如果实测有问题
  （先量再改），考虑提示用户分卷导出，而不是自己实现分页。
