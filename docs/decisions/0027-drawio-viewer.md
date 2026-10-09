# 0027 · 内置只读 draw.io 查看器（`doc-block` 图表引用落地）

日期：2026-10-04
状态：已采纳（P5 第 25 项落地）
关联：0001（无网络）、0002（doc-block 与四类文档）、0005（draw.io 查看器的 V1 实测与四条硬要求）、
0018（图片的 `data:` 通道）、0020（Mermaid 的动态加载）、`docs/format-spec.md` §2–§5

## 背景

`.drawio` 的查看器在 V1（ADR 0005）已经**用真实浏览器实测过**：资源路径怎么占、
`createViewerForElement` 怎么驱动、stencil 缺失长什么样，全部有结论。本 ADR 记录的是
**把那些结论接进编辑器**时的实现决策 —— V1 回答「行不行」，这里回答「怎么放、怎么画、
失败说什么」。

图的来源是 `doc-block`（ADR 0002 定的块级引用格式）：信息串为 `doc-block` 的围栏代码块，
属性区声明 `kind: diagram`，图的内容要么在 `src` 引用的外部 `.drawio` 文件里
（`mode: reference`，默认），要么就在载荷里（`mode: inline`）。

## 决策

### 1. 上游产物放 `public/drawio/`，运行时脚本注入，**不进 bundle**

`viewer.min.js` 2.654 MiB 是 draw.io 官方**未改动**的构建产物（ADR 0005：上游不接受
PR，我们只动配置层）。静态 import 会让每个打开轻笺的人先解析 2.6 MB —— 哪怕一篇
文档一张图都没有。

所以：文件按**原样**放 `public/`（Vite 的 `publicDir`，产物里 URL 稳定、不带哈希、
同源 —— CSP 的 `script-src 'self'` 天然放行），`core/drawio-render.ts` 只在文档里
**真的出现 `kind: diagram` 的 doc-block 时**才用 `<script>` 注入一次。加载去重缓存
的是 Promise（失败不缓存 —— 资源补齐后下一次渲染要重试），与 `mermaid-render` 的
`cachedLoader` 同一条规则。

ESLint 对 `public/drawio/**` 全量忽略：那是**被观测的对象**，不是我们的代码
（与 `.v1-out/**` 同一条理由）。

### 2. 十条资源路径在脚本加载**之前**全部占住 —— 这是「无网络」的唯一开关

CDN 版产物把每条资源路径写成 `window.X || "https://viewer.diagrams.net/…"`。
加载前必须全部指到本地（`SHAPES_PATH` / `STENCIL_PATH` / `STYLE_PATH` /
`GRAPH_IMAGE_PATH` / `RESOURCES_PATH` / `RESOURCE_BASE` / `mxImageBasePath` /
`mxBasePath` 八条给本地路径，`PROXY_URL` / `DRAW_MATH_URL` / `DRAWIO_BASE_URL` /
`DRAWIO_LOG_URL` 四条给空串走「没配置」分支），再顺手用 `onDrawioViewerLoad = noop`
拦掉脚本末尾的自动 `processElements()`。**漏一条就是一次静默联网**，与 ADR 0001
直接冲突 —— 所以单测把这条清单**逐条**钉住（`drawio-render.test.ts`）。

### 3. 缺哪些 stencil 库：**渲染前从 XML 里算**，不是事后看请求

V1 实测结论 4：mxGraph 取不到 stencil 时**不报错**，形状画成矩形、标签照旧 ——
静默降级。硬约束 2 要求把这件事说出来，但「事后看哪些请求 404」有两个问题：
多图并发时请求归属不了，XHR 打点又是个全局猴子补丁。

而需要哪些库**写在 XML 里**：stencil 形状的风格串是 `shape=mxgraph.<库名>.<形状名>`。
渲染前扫一遍风格串、与内置清单（`bpmn`、`flowchart`，ADR 0005 按库带方案）求差，
就得到**这一张图**缺什么 —— 纯函数、逐图准确、无竞态、可直接单测。
缺失时图**照画**，图区里补一条警示（「未内置的形状库：… —— 相关形状已按矩形显示」）。

刻意只认 `mxgraph.` 前缀这一种写法（V1 样本压过的主流形态）；漏网的 exotic 写法
退回 mxGraph 的静默矩形 —— 与内置查看器出现之前一样，不会更糟。

### 4. 谁去读 `src` 引用的文件：**外壳**（契约 `resolveDocBlock`）

「`figs/a.drawio` 在哪、怎么读」是文档模型之外的事 —— 内核连自己正在编辑哪个文件
都不该知道（与 `resolveImageUrl` 完全同构）。所以契约加在
`EditorMountProps.resolveDocBlock` 上：`src` 进去，文件原文或 `null` 出来；
外壳的实现是 `store.resolveDocBlock` —— 相对当前文档目录，经 `joinPath`
（分隔符统一成文档路径的风格，硬约束 12）归并后走 `readTextFile`。
**不缓存**、**不抛**：调用链尽头是 NodeView 的一次渲染，未捕获的拒绝只会变成
一个没人处理的 Promise。

### 5. 所有失败都落成图区里**带原因的一句话**

`kind: diagram` 的块解析失败、引用读不到、查看器加载失败、XML 不合法 —— 全部
占据图区并写明原因（「缺少结束行」「读不到引用的文件：…」「查看器脚本加载失败」），
而不是退回源码态：图区被 CSS 藏着，退回去等于把原因藏起来。这与 mermaid 失败时
占着图区的理由相同，是硬约束 2 在查看器上的全部落点。inline 载荷为空也给**可操作**
的话（「图的内容应该写在 `---` 之后」），不用查看器那句笼统的「内容是空的」。

只有一种降级**不说失败**：`kind` 不是 `diagram` 的 doc-block（比如 `kind: code`）
显示为普通代码块 —— 那是 ADR 0002 定的「不认识就显示源码」，语义本来就是对的。

### 6. 渲染**不接主题**；「放大看」交的是渲染用的那份 XML

mermaid 的颜色是渲染时按主题定的，所以 `renderMermaid` 必须收主题；drawio 的颜色
写在图自己的 XML 里，画布恒白 —— 换明暗重画只会得到同一张图。刻意不设 `theme`
参数：与 `renderMermaid`「不设默认值」是同一条纪律，方向相反。

「放大看」的 `source` 交的是**渲染用的那份 XML**（引用模式下是文件原文，与块文本
不是一回事）—— 重渲染一遍既慢又可能画出不一样的图，与 `onDiagramOpen` 交 SVG
快照是同一条理由的两半。

### 7. 渲染器做成可注入的（`DrawIoIo`）

jsdom 没有布局引擎也没有 rAF，真查看器在单测里什么都证明不了。与 `MermaidIo`
同一条纪律：`loadViewer` / `settle` 注入，单测钉**我们这一层**的契约 ——
资源路径占位、加载去重、失败不缓存、错误说出来、SVG 快照、超时兜底 ——
「图画得好不好」只能人工看。

## 后果

- 应用带 2.654 MiB 的查看器 + 0.26 MiB 的 stencil/shape（ADR 0005 按库带方案），
  全部按需注入，主包不受影响（构建实测主包 1,892 kB，viewer 不在其中）。
- 查看器是**只读**的：多页切换、缩放可用（`nav: true`），编辑图要走导出回写，
  不在本项射程内。
- `.drawio` 作为**独立打开**的文档类型仍未支持（那是 P5 第 26 项及以后的事）；
  本项只覆盖 Markdown 里 `kind: diagram` 的 doc-block 引用。

## 重审条件

1. **stencil 覆盖不足**：真实使用里 `missingStencilLibs` 的警示频繁出现（比如用户
   群集中在某个云厂商的形状库），按库补清单或改用 `viewer-static.min.js`
   （ADR 0005 建议 5，自带 832 个形状类）—— 上游若有更新先重跑 V1 的三判据。
2. **上游产物的路径机制变了**：`window.X || 默认值` 这个可覆盖写法若在后续
   viewer.min.js 里消失（改成闭包内常量），第 2 条的「唯一开关」失效，
   必须重新实测取路径清单 —— 在那之前**不要升级 viewer.min.js**。
3. **`public/` 供给方式失效**：若 Tauri 的资源打包对 `public/` 产物做出改动
   （哈希、scope、协议变化）导致同源脚本注不进来，改走自定义协议时必须同步
   复核 CSP 与 ADR 0018 的白名单理由。
4. **「按库算缺」粒度不够**：若「库在清单里但具体形状缺失」成为常见反馈，
   再考虑按形状建清单 —— 那份清单要跟着上游版本走，没有证据前不加这个维护负担。
