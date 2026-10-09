# 0005 · draw.io 集成路线：内置查看器 + 编辑器按需下载

日期：2026-09-28（V1 实测复核：2026-10-02）
状态：**已采纳**（V1 实测通过 —— 见文末「V1 实测记录」）
关联：0001（桌面端优先）、开放问题 R1（体积）

## 背景

`.drawio` 是四类文档之一，但集成 draw.io 的资源体积与「轻」的产品定位正面冲突。
实测数据（2026-09-28，取自上游仓库与官方镜像）：

`src/main/webapp/js/` 主要产物：

| 文件 | 大小 |
|---|---|
| `integrate.min.js`（完整编辑器） | 22.7 MB |
| `app.min.js` | 9.8 MB |
| `stencils.min.js` | 7.7 MB |
| `viewer-static.min.js` | 4.2 MB |
| `extensions.min.js` | 3.9 MB |
| **`viewer.min.js`**（只读查看器） | **2.7 MB** |
| `shapes-14-6-5.min.js` | 1.5 MB |
| `orgchart.min.js` | 0.55 MB |

仅这几个大文件合计 ≈ **53 MB**；加上 `stencils/`、`shapes/`、`images/`、`templates/` 等目录，
整个 webapp 约 **60–100 MB**（旁证：官方 Docker 镜像 116.4 MB）。

对照：Tauri 2 应用本体典型体积 **3.2 MB**（2026-07 基准，同基准 Electron 为 85 MB）。

## 决策

1. **默认构建内置只读查看器**（`viewer.min.js` + 必要 stencils，约 3 MB）。Markdown 里的
   `doc-block` 引用在编辑器中**直接渲染成真图**，且完全离线可用。
2. **完整编辑器按需下载**：用户首次点「编辑图表」时才下载（约 60–100 MB），存到应用数据目录，
   下载后完全离线可用。下载是**显式**的，必须让用户看到体积与进度。
3. **另提供一个完整离线构建目标**（全部资源打包），供内网/涉密环境使用。
4. **不采用 maxGraph 自建编辑器。**

## 理由

- **查看高频、编辑低频。** 引用 `.drawio` 的绝大多数场景是「看」。花 3 MB 让引用块渲染成真图，
  是 P6「块级互操作」体验的关键，代价几乎为零。
- **按需下载把 60–100 MB 变成用户的选择**，而不是替他决定。这与项目的编码策略同源（ADR 0003：
  绝不静默转码）—— 凡是有重大代价的操作，都要显式、可见、可撤销。
- **不选 maxGraph 是这道题的核心判断。** maxGraph（Apache-2.0，1,141★，是已归档的
  `jgraph/mxgraph` 的活跃继任者）是**图形库不是编辑器**。选它意味着形状面板、模板、stencil 库、
  导出、多页处理全部自建，并把 `format-spec.md` §7.2 的四条 `.drawio` 往返硬要求，
  从「上游的既有正确性」变成「我们要永远自己保证的正确性」。
  这是用**有界且已知的成本（体积）**，去换**无界且持续的成本（兼容性维护）**。

## 许可与商标边界（取自上游 README 原文）

- 源码 Apache-2.0。**图标集、stencil 库、模板另有限制**：不得作为软件资源用于 / 随附分发 /
  集成进 **Atlassian 产品**或经 Atlassian 市场与插件生态分发的产品，未经书面许可。
  该限制**明确覆盖衍生品**（格式转换、描摹复制、实质相似的视觉表现、以这些图标作参考或训练输入的 AI 生成图）。
  - 射程判断：轻笺不是 Atlassian 产品，字面不在禁止范围内。**商业发行前建议过一遍法务。**
  - **不要**用这些图标做训练素材或 AI 生成。
  - 例外：用户导出的图表成品不受限制。
- **商标**：`draw.io` 是注册 EU 商标（**#018062448**）。不得以暗示关联/背书的方式使用其名称与 logo，
  不得用于自己的产品、项目、域名，不得修改其 logo。
  - 落地：UI 只能写「支持 `.drawio` 格式」或「集成 draw.io 编辑器」，不得命名为「轻笺 draw.io」，不使用其 logo。
- **上游不接受 PR**（完全由核心团队开发）。→ **不要 fork 后长期维护分支**；使用上游**未改动**的
  构建产物，改动限制在配置层（`PreConfig.js` / `PostConfig.js`）。

## 后果

- 安装包从 3.2 MB 增至约 **7 MB**。
- 首次编辑图表需要一次联网下载。必须先实现在线检测与失败态（下载中断、校验失败、磁盘不足），
  不能出现「点了没反应」。
- 需要维护一份资源版本清单与校验和。
- **macOS 系统要求被这条决策抬高**：draw.io 要求 Safari 17.5+，而 WKWebView 无法脱离系统独立升级
  → 内嵌 draw.io 等于**最低 macOS 14.5**。只内置 viewer 时下限可更低，这属于本决策的额外收益。

## 重审条件

- **P0 的 1 天实测结论**：若 `viewer.min.js` 无法独立渲染（必须依赖完整 bundle），
  或无法正确渲染多页与压缩变体的 `.drawio`，则退化为「全部按需下载」（连 viewer 也下载）。
- 若实测 viewer + 必要 stencils 明显超过 6 MB，重新评估是否值得内置。
- 若上游改变许可条款或停止维护，立即重审。

---

## V1 实测记录（2026-10-02）

**两条重审条件都没有命中 —— 决策维持。** 但实测翻出了**两条原决策没预料到的风险**，
它们直接改变 P5 的实现要求（见下面「结论 4 / 5」）。

### 怎么测的

`scripts/v1-run.mjs`（驱动器）+ `scripts/v1-page.html`（测试页）+ `scripts/v1-samples.mjs`（样本）。
产物从官方 CDN 下到 `.v1-out/`（**本机只有 Python urllib 能过代理，且必须显式设 `User-Agent`** ——
`viewer.diagrams.net` 对默认 UA 回 403；`curl` 走 Schannel 握手直接失败、Node 的 `fetch` 不认 `http_proxy`）。

样本是 **8 份 `.drawio`**，压的是从**查看器源码读出来的**变体（不是想出来的）：

| 样本 | 形态 |
|---|---|
| `child-uncompressed` | 3 页，`<mxGraphModel>` 作为 `<diagram>` 子元素 |
| `compressed` | 3 页，`base64(deflateRaw(encodeURIComponent(xml)))` |
| `mixed` | 2 页，一页未压缩一页压缩 |
| `empty-page` | 2 页，其中一页 `<diagram>` 里什么都没有 |
| `bare-model` | 根直接是 `<mxGraphModel>`，没有 `<mxfile>` 外壳 |
| `stencil-required` | BPMN + flowchart 形状（**不在** mxGraph 核心里） |
| `plain-rectangles` | **对照**：与上一条几何完全相同，只是没用 stencil |
| `escaped-text` | **负样本**：证伪过的变体，见下 |

五轮对照（每轮只改一个变量）：A 占位+无 stencils / B 占位+真 stencils / C 占位+预加载
`stencils.min.js` / D **不占位** / E 换 `viewer-static.min.js`。

### 结论 1 —— **能独立运行，但必须自己占住资源路径**

CDN 版把每一条资源路径硬写成绝对地址并可覆盖：
`window.SHAPES_PATH = window.SHAPES_PATH || "https://viewer.diagrams.net/shapes"`
（`STENCIL_PATH` / `STYLE_PATH` / `PROXY_URL` / `DRAW_MATH_URL` / `GRAPH_IMAGE_PATH` /
`RESOURCES_PATH` / `RESOURCE_BASE` / `mxImageBasePath` / `mxBasePath` 同）。

- **占位后**：不带 stencil 的样本 **0 个网络请求**（服务器日志 + CDP + 页面内打点 + Resource Timing
  四条读数一致，且有一条**正对照**证明探针本身是通的）。
- **不占位（D 轮）**：那 5 个请求**全部发往 `https://viewer.diagrams.net/...` 并成功取回**。

→ **P5 必须把这 10 个路径全部指到本地**。否则一个「纯本地、无网络请求」的产品
（ADR 0001）会在打开一张带 stencil 的图时**静默联网**。

### 结论 2 —— **多页与压缩变体都能正确渲染**

| 样本 | 页数 | 标签 | 请求 |
|---|---|---|---|
| `child-uncompressed` | 3/3 ✅ | 逐页切过去都 ✅ | 0 |
| `compressed` | 3/3 ✅ | 逐页 ✅ | 0 |
| `mixed` | 2/2 ✅ | 逐页 ✅ | 0 |
| `empty-page` | 2/2 ✅ | 第 1 页可切、不画东西 | 0 |
| `bare-model` | **无「页」概念**（页集合 0） | 内容画得出 ✅ | 0 |

判据不是「有没有 svg」，而是**逐页切过去把标签文字读回来**跟样本的期望值比 ——
「页数对」不等于「每页都画对了」。

**`bare-model` 是个新事实**：没有 `<mxfile>` 外壳时 `extractGraphModel` 接受 `mxGraphModel` 当根，
内容照画，但 `GraphViewer` 的页集合是 `getElementsByTagName('diagram')` → **为 0**。
P5 读这种文件时**不能假设「页」存在**。

### 结论 3 —— **体积：单个查看器过线，「viewer + 全量 stencils」超线**

| 组合 | 原始 | gzip |
|---|---|---|
| `viewer.min.js` | 2.654 MiB | 0.684 MiB |
| `viewer-static.min.js` | 4.073 MiB | 0.866 MiB |
| `embed-static.min.js` | 1.284 MiB | 0.318 MiB |
| **`viewer.min.js` + 全量 `stencils.min.js`** | **9.987 MiB** | **6.191 MiB** |
| **`viewer.min.js` + bpmn/flowchart 两个库（5 个文件）** | **2.914 MiB** | **0.708 MiB** |

（与本文档 2026-09-28 那次从上游仓库量的数字吻合：`viewer.min.js` 2.7 MB、`viewer-static` 4.2 MB、
`stencils.min.js` 7.7 MB。）

**「按库带」与「全量带」差 3.4 倍**，而且全量那条在**两个口径上都超 6 MB 判据**
（`stencils.min.js` 几乎压不动：7.333 MiB → gzip 5.507 MiB，因为它是 base64 数据块）。

### 结论 4 —— **stencil 缺失是静默降级（原决策没预料到）**

mxGraph 取不到 stencil 时**不报错、不提示**，把形状画成普通矩形，而**标签照旧画出来**：

| 轮 | 请求 | rect | path | svg 长度 | 形状 |
|---|---|---|---|---|---|
| A 全 404 | 5 | 2 | 3 | 2809 | **回落** |
| B 真文件 | 5 | **1** | **7** | 3365 | ✅ |
| C 预加载 | 3 | 2 | 3 | 2827 | **仍回落** |

→ **P5 必须把「哪些 stencil 库取不到」在 UI 上说清楚**（硬约束 2「绝不静默」）。
只靠「标签都在」这种断言会把静默降级判成通过 —— 本次实测就**被骗过一次**。

### 结论 5 —— **`stencils.min.js` 不足以让形状画对（原决策没预料到）**

C 轮（预加载 7.333 MiB 的 `stencils.min.js`）把请求从 5 降到 3，但**形状仍然回落**。
原因是它**只含 `.xml` 那一半**：文件头就是 `f['alibaba_cloud.xml'] = '…base64…'` 这样一张表，
`registerShape` 出现 **0 次**、`function` 只有 4 次 —— 它是**数据块，不是代码**。
而 BPMN 的形状是 JS 类（`shapes/bpmn/mxBpmnShape2.js` → `mxCellRenderer.registerShape`）画的。

**`shapes/*.js` 是必需的，且它不在 `stencils.min.js` 里。** 任何「只带 stencils.min.js 就离线了」
的方案都是错的。

### 一条被证伪的变体（留证，别再发明一遍）

变体表里原本有一行「未压缩 · 转义文本」—— `<diagram>` 里放 `&lt;mxGraphModel …&gt;`。**实测读不了**，
报 `Failed to execute 'atob' on 'Window': The string to be decoded is not correctly encoded`。

根因在 `Editor.parseDiagramNode`：`<diagram>` 只要**有文本内容**就**无条件**走 `Graph.decompress`，
而 `Graph.decompress` 第一句是 `Graph.stringToArrayBuffer(atob(b))` —— **没有保护**，
也没有「这段文本是不是已经是裸 XML」的分支。

反向也证过：`viewer.min.js` 里**没有任何路径会把转义文本写进 `<diagram>`**
（全文搜 `"<diagram` 只有一处，是 Firefox 拖拽的 hack），`Graph.compress` 默认就是 `deflateRaw`。
→ **这不是「查看器不支持某版本」，而是这种形式根本不是 `.drawio` 的合法变体。**
（转义形式真实存在的地方是别处：HTML 的 `data-mxgraph` 属性、SVG 的 `content` 属性、
PNG 的 `zTXt` 块 —— 那些走另外的抽取函数，不经过 `parseDiagramNode`。）

它在样本表里**留着当负样本**：期望是「明确报错」，而不是「画出一张空白画布」。

### 对 P5 的落地建议

1. 内置 `viewer.min.js`（2.654 MiB），**按库内置** `stencils/*.xml` + `shapes/*.js`
   （bpmn + flowchart 两个库只要 0.26 MiB）。**不要内置全量 `stencils.min.js`**（超判据，且不够用）。
2. 那 10 个路径**全部**指到本地；这是「无网络请求」的**唯一开关**，漏一个就会静默联网。
3. 取不到的 stencil 库 → **显式提示**（「这张图里的 N 个形状用到了未内置的库，已按矩形显示」）。
4. 读 `.drawio` 时**不假设「页」存在**（`bare-model` 形态的页集合为 0）。
5. `viewer-static.min.js`（4.073 MiB）是一条**备选**：它自带 832 个形状类（含 50 处 `mxBpmnShape`），
   实测只剩 `stencils/flowchart.xml` 一个外取 —— 更省事，但多 1.4 MiB 且**仍不是完全离线**。

