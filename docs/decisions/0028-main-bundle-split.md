# 0028 · 主包拆分：内核走动态 import，注册在首帧之前完成

日期：2026-10-04
状态：已采纳（P3 第 18 项落地）
关联：0003（导出与内核共用高亮管线）、0017（代码块与高亮同源）、
`core/editor/registry.ts`（注册顺序即优先级）、`ROADMAP.md` 第 18 项

## 背景

主包（`index-*.js`）从 1,702 kB 一路涨到 1,892 kB（gzip 626），远超 Vite 的
500 kB 警戒线。一个「轻量桌面编辑器」让 WebView 在画出任何东西之前先解析
1.9 MB JS，与定位直接冲突。

归因（见 ROADMAP 第 18 项的起点数据）：Lezer 语法表十份 ≈ 1.3 MB（源码）、
ProseMirror ≈ 750 KB、highlight.js common —— 全部是**编辑器内核**的东西，
而外壳首屏（活动栏、标签栏、设置页、命令面板）一行都用不到。

## 决策

### 1. 三个专门化内核改成 `import()`，各自成 chunk；兜底内核保持静态

`modules/index.ts` 的 `registerBuiltInEditors()` 从同步函数变成**返回 Promise**
（内部对 markdown / mindmap / code 逐个动态 import）。Milkdown、CodeMirror、
mind-elixir 各自落进独立 chunk，与外壳并行加载。

兜底内核（plaintext）**刻意保持静态 import**：「打开任何文件都不能失败」
（硬约束 3）的最后一道防线不能自己也是异步的 —— 它小（几 KB），静态跟随
外壳没有任何代价。

### 2. 注册顺序由 await 的先后钉死，与模块解析时序无关

「注册顺序即优先级」是硬约束 4，动态化之后这条契约**只能由执行顺序保证**：
`await import(markdown)` → 注册 → `await import(mindmap)` → 注册 →
`await import(code)` → 注册。谁先谁后写在代码里，不依赖 Promise.all 的
解构顺序、更不依赖模块图。`wiring.test.ts` 的备胎链测试继续钉住最终顺序。

### 3. 首帧之前必须注册完成 —— 等待态放在 main.tsx，不放 App 里

`resolveAdapter` 是同步查表。表还空着就渲染第一帧，`.md` 会被**静默兜到
纯文本**上，等内核到了再「跳」回富文本 —— 那是硬约束 3 要杜绝的「被遮蔽
的打开路径」，比打不开更难解释。所以 `main.tsx` 在 `render()` 之前
`await registerBuiltInEditors()`；等待期间用户看到的是 `index.html` 里的
静态引导位（「正在启动轻笺…」），本地磁盘上通常只有几十毫秒。

**放在 main.tsx 而不是 App 的 state 里**：让 App 持一个 `kernelsReady`
布尔，意味着每个渲染 App 的测试都要先等一个异步状态 —— 一千多条测试
全都要改写成异步等待。注册挪到入口、App 保持同步组件，测试只需要在
`beforeAll` 里 await 一次注册。

### 4. 单个内核加载失败 = 降级，不说失败但也不挡启动

动态 import 引入了一种静态时代不存在的失败：**某个内核的 chunk 拿不到**
（安装不完整）。处理分两层：

- `registerBuiltInEditors()` 对每个内核单独 try/catch：失败的记进
  `editorRegistrationFailures()`，**其余内核照常注册** —— 注册表少一个元素
  时，备胎链的降级语义天然成立（正主没了 → 备胎接住 → 实在不行纯文本）。
- App 挂载后读一次 `editorRegistrationFailures()`，非空就出横幅
  （「有 N 个编辑内核没能加载…相关文档会以纯文本打开」）—— 降级不许
  静默（硬约束 2），但也不该把整个应用挡在门外。
- 这层之上如果还有异常（连「失败」都记不下来的意外），`main.tsx` 把错误
  写进 `#boot-status` 如实展示 —— 启动坏了也有一句话，不能白屏。

### 5. 顺带从主包里挪走的两个「低频重物」

拆完内核后主包还有 601 kB，归因发现两个非内核的静态依赖：

- **导出模块**（`core/export/*`）拖着 hljs 与整套 remark（为了代码块与公式
  的 HTML 化），而导出是低频动作。`file.exportHtml` 命令改成执行时
  动态 import —— 导出模块成为独立 chunk（24 kB）。
- **语言选择器的清单**（`ui/CodeBlockPicker.tsx`）在模块顶层调
  `supportedLanguages()`，把 hljs 拉进主包。清单改成**打开面板时**才
  动态加载 —— 数据源不变（仍然是 `hljs.listLanguages()`，不另立清单，
  那条「名单只有一份」的纪律原样保留），变的只是时机；面板打开那一刻
  hljs 已经跟着内核 chunk 进了内存，等待实际不到一帧。

## 结果（vite build 实测，2026-10-04）

| chunk | 体积 | gzip | 何时加载 |
|---|---|---|---|
| `index`（外壳） | **341.92 kB** | **107.53** | 首帧（原 1,892.10 / 626.59） |
| `markdown`（Milkdown + ProseMirror） | 436.99 kB | 134.16 | 注册时（首帧前，并行） |
| `code`（CodeMirror + Lezer ×10） | 748.65 kB | 265.95 | 注册时（首帧前，并行） |
| `mindmap`（mind-elixir） | 103.85 kB | 34.23 | 注册时（首帧前，并行） |
| `hljs`（highlight.js common） | 156.23 kB | — | 注册时（被 markdown 与导出共用） |
| `export-document` | 24.48 kB | 9.31 | 第一次执行导出命令时 |
| mermaid / katex / draw.io viewer | — | — | 维持原有懒加载（ADR 0019/0020/0027） |

主包 gzip 从 626 → **107.5 kB**，跌破 Vite 的 500 kB 警戒线。
**启动时需要解析的总量没有变**（内核 chunk 仍在首帧前并行加载）——
这一步省的是「外壳先解析完、内核并行加载」的流水线与缓存粒度；
把内核也变成「打开对应文件才加载」是下一步的事，需要先把 canHandle
的内容判定挪到加载之后，不在本项射程内。

浏览器实测（`vite preview` + 真实浏览器）：引导位显示 → 三个内核 chunk
与 hljs 并行取回 → React 挂载，外壳完整可用，控制台无错误。

## 重审条件

1. **「按需加载内核」**（打开 `.md` 才加载 Milkdown、打开 `.mindmap` 才加载
   mind-elixir）若立项，本 ADR 的注册时序要重新设计 —— `canHandle` 的内容
   判定（坏 `.mindmap` 落到代码内核）依赖先有内容，而读内容在外壳；
   先画启动画面再按需加载会让「注册表」从一张同步查表变成异步解析，
   硬约束 3 的验证方式要跟着换。
2. **启动可感知变慢**（比如低端设备的 WebView 并行解析四个 chunk 反而卡住
   主线程）则考虑给注册加优先级：先 markdown（最常用），code / mindmap
   延后到空闲时。
3. **`#boot-status` 引导位在真实 Tauri 窗口里不可见**（纯白闪一帧）就把它
   做成品牌启动画面 —— 现在的实现刻意只有几行内联样式，不值得更早优化。
