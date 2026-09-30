# 路线图

> 本文只列**可交付的节点**与**当前状态**。每个决策的理由与重审条件在
> [`docs/decisions/`](docs/decisions/README.md)，待验证项在
> [`docs/open-decisions.md`](docs/open-decisions.md)。

状态图例：✅ 已完成 · 🚧 进行中 · ⏳ 计划中 · ❓ 待验证后决定 · ❌ 明确不做

---

## P0–P2 · 已完成（截至 2026-09-28）

| 阶段 | 交付 |
|---|---|
| P0 | 技术选型与体积评估（Tauri 2、Milkdown、draw.io 资源路线） |
| P1 | 文档生命周期：打开 / 脏标记 / 冲突 / 编码探测 / 原子写入（44 项前端 + 11 项 Rust 单测） |
| P2 | Markdown 内核（Milkdown 7.22.2）+ **自包含 HTML 导出**（ADR 0008） |

---

## P3 · 工作区化（本次，2026-09-29）

| # | 交付 | 状态 | 备注 |
|---|---|---|---|
| 1 | 多标签工作区 | ✅ | ADR 0010。`core/workspace/store.ts` 组合 N 个 `DocumentStore` |
| 2 | 文件树（懒加载 + 重目录折叠） | ✅ | 打开文件时自动展开祖先链 |
| 3 | 代码内核（CodeMirror 6，10 种语言） | ✅ | ADR 0011 |
| 4 | 「降级到次优」原则 | ✅ | front matter 的 `.md` 从白底 textarea 变为有行号的源码 |
| 5 | 命令注册表 + 快捷键 + 命令面板 | ✅ | ADR 0012 |
| 6 | 查找 / 替换（三个内核共用语义） | ✅ | `$1` 展开在正则模式生效 |
| 7 | 大纲面板 | ✅ | 纯函数提取，不依赖内核 |
| 8 | 主题（宣纸 / 松烟 / 跟随系统）+ 偏好持久化 | ✅ | 颜色只有一处来源 |
| 9 | 工程基建：git / CI / ESLint / 六份对外文档 | ✅ | lint 已**阻塞**（欠账清零，见第 20 项） |
| 10 | 修掉两处真实缺陷 | ✅ | 注册表优先级与注释相反；查找替换不报变更会静默丢数据 |

### P3 收尾（尚未做）

| # | 交付 | 状态 | 阻塞 |
|---|---|---|---|
| 11 | **V3：长文档输入延迟实测** | ⏳ | 需要真实 WebView。**它的结论决定第 19–22 项** |
| 12 | 会话恢复 + 未保存草稿不丢 | ⏳ | 数据安全，优先级最高 |
| 13 | 图片粘贴落盘 + 相对路径策略 | ✅ | 已完成（M6，ADR 0018）。自包含导出的最后一块拼图 |
| 14 | Rust 侧 `watch` 落地（`notify` + debouncer） | ⏳ | 替换「窗口焦点时比对 mtime」的权宜之计 |
| 15 | 设置页 | ⏳ | |
| 16 | PDF 导出（WebView 打印管线） | ⏳ | 与自包含导出共用渲染结果 |
| 17 | i18n 骨架（先抽文案，不急翻译） | ⏳ | 越晚做越贵 |
| 18 | 主包拆分（`React.lazy` 加载内核） | ⏳ | 外壳首屏目标 < 200 kB。**实测数据见下** |
| 19 | 标签页切换保留撤销栈与光标 | ⏳ | 需先实测内存占用（见 ADR 0010 的重审条件） |
| 20 | **lint 清尾** | ✅ | 已完成，记录见下 |

### 主包体积（第 18 项的起点数据）

`npx vite build` 的实测产出（2026-09-29，M6 之后）：

```
dist/assets/index-C79kp17n.js    1,702.57 kB │ gzip: 566.70 kB
dist/assets/index-Bd1vkdXJ.css      25.98 kB │ gzip:   5.78 kB
```

> M5 结束时是 `1,695.08 kB / gzip 564.07 kB`（CSS `24.17 / 5.35`）。
> **M6 的增量是 +7.49 kB / gzip +2.63 kB，CSS +1.81 kB / gzip +0.43 kB** ——
> 这一批**几乎没有新增逻辑**（图片的显示、落盘、灯箱三块加起来不到 700 行，
> 其中一半是注释），增量主要来自「图片的 NodeView + 灯箱的样式」。
> **仍然没有引入任何新依赖。**

**已经超过 Vite 的 500 kB 警戒线。** 这不是「要不要优化」的问题 —— 一个
「轻量桌面编辑器」让 WebView 在启动时解析 1.7 MB JS，与定位直接冲突。

归因（源码体积，非压缩后；带 `.` 前缀的目录是 npm 暂存残留，不计入）：

| 来源 | 体积 | 说明 |
|---|---|---|
| Lezer 语法表 × 10 种语言 | ≈ 1346 KB | `@lezer/rust` 254、`javascript` 241、`markdown` 172、`python` 97、`html` 67、`css` 60、`yaml` 53、`xml` 33、`json` 7，`lang-sql` 自带 104 |
| prosemirror-view / model / state | ≈ 748 KB | Milkdown 的内核 |
| `highlight.js/lib/common` | 已是最优选择 | 用的是 `lib/common`（约 40 种常用语言），不是全量包 |

**根因是接线方式，不是依赖选得不对**：`src/modules/index.ts` **静态**引入
`markdownAdapter` 与 `codeAdapter`，于是打开**任何**文件（哪怕是个 `.txt`）
都会把两套内核、10 种语言的语法表、highlight.js 全部加载进来。

三条可做的（按收益/风险排序）：

1. **内核级代码分割**：`registerBuiltInEditors()` 改成动态 `import()`。
   注意这会引入一个**真实的语义变化** —— 注册是异步的，如果用户在注册完成前
   就打开了文件，注册表里还没有候选，会落到兜底内核。那不是崩溃（plaintext
   永不失败），但是「用错了内核」，属于本项目的**静默降级**红线。
   所以这一条**必须配套一个「等待注册完成」的状态**，不能只改 import 就完事。
2. **语言支持按扩展名懒加载**：`@codemirror/lang-*` 用动态 import，
   由 CodeMirror 的 `Compartment` 在扩展名确定后重配置。
   代价是首次打开会多一次重解析，需与 V3 的延迟结论一起看。
3. 把 `@lezer/markdown`（172 KB）与 Milkdown 自带的 markdown 解析器对齐 ——
   当前 markdown 解析能力在包里存在两份。

**第 1 条与 V3 的结论有耦合**（都落在「打开文件到能打字」这条路径上），
所以排在第 11 项之后做。

### lint 清尾（第 20 项）—— 已完成

`ESLint` 已从「非阻塞」转为**阻塞**：`npm run check` 与 CI 都会因 lint 失败而失败。

首轮全量扫描 **33 条**，现在 **0 条**。逐条的去向：

| 规则 | 数量 | 怎么处理的 |
|---|---|---|
| `react-hooks/exhaustive-deps`「规则不存在」 | 4 | 装上 `eslint-plugin-react-hooks`。**只开 `rules-of-hooks` + `exhaustive-deps`**，不开它的 `recommended` —— 那会把整套 React Compiler 规则（`purity` / `immutability` / `set-state-in-render` …）一起打开，瞬间产出上百条意见，等于把 lint 重新变成噪声 |
| `only-throw-error` | 11 | 生产代码引入 `AppError` 类（`extends Error`，与 Rust 的 `AppError::new` 对称）；测试里的 `FakeProvider` **保留抛普通对象**并加豁免 —— 那是在模拟 Tauri 边界的 serde 输出，抛对象才是保真 |
| `no-unnecessary-type-assertion` | 7 | `lint:fix` 自动去除（顺带发现 `store.ts` 的 `TargetLineEnding` 变成未使用导入，已删） |
| `no-base-to-string` | 3 | 改成 `typeof` 收窄。`String(对象)` 会得到 `'[object Object]'`，那种「看起来有值其实是垃圾」的结果比空串更难查 |
| `no-useless-assignment` | 2 | 去掉 `let x = <初值>` 的死初值（try/catch 两条路径都会赋值） |
| `consistent-type-imports` | 2 | `lint:fix` 自动修；`registry.test.ts` 的 `typeof import(...)` 改成静态 `import type`（类型被完全擦除，不影响 `vi.resetModules()`） |
| `restrict-plus-operands` | 1 | 提取 `findHeadingPos()` 辅助函数。**这条最有信息量**，见下 |
| `no-console`（脚本） | 2 | `scripts/**` 的 override 里关掉 —— 命令行脚本的产出就是 stdout |
| 未使用的 disable 指令 | 2 | 自动清除，并清掉留下的纯空白行 |

**另外还多做了一件：把一条原本被关掉的规则重新打开。**

`@typescript-eslint/unbound-method` 在接入时是 `off` 的 —— 它报的是
「把对象上的方法拆下来单传」，而 React 里回调 prop 用**方法简写**声明时会被误报：

```ts
interface SidebarProps { onOpenFolder(): void }                    // 方法签名 → 被认为「有 this」
export function Sidebar({ onOpenFolder }: SidebarProps) { ... }     // 解构 → 报错
```

关掉它只是把噪声藏起来。真正的修法是把 11 个文件里的回调统一改成**属性签名**
`onOpenFolder: () => void`，改完之后规则不再误报，于是打开。

顺带的好处是**类型安全变严了**：方法参数是双变的，属性参数是逆变的
（`strictFunctionTypes`），所以「回调参数类型必须真的对得上」从此被编译器检查。
实测这次改动**没有暴露任何新的类型错误**，说明原来的标注本来就是对的 ——
这也算是一次免费的验证。

**留下的配套约定**（写在 `eslint.config.js` 里）：options / props 里的**回调**用属性签名；
**返回实例上的方法**（`EditorHandle`、`MarkdownEditorInstance`）保持方法简写 ——
它们只会以 `instance.foo()` 调用，不会被拆下来。

**`restrict-plus-operands` 暴露的是一件比它本身重要得多的事**：
同一行代码，**TS 6 算出 `never`，TS 7 算出 `number`**。

用 TS 6 的编译器 API 逐点查过（`src/modules/markdown/setup.ts`）：

```
行 335  回调内 found !== null  => number | null
行 343  守卫 found === null    => null      ← TS 6 没把回调里的赋值算进去
行 345  使用点 found + 1       => never
```

原来的写法是「在 `descendants` 回调里给外部 `let` 赋值 → 外层收窄 → 再 `+ 1`」。
TS 6 的控制流分析对这种模式失效，把 `found` 当成 `null`，排除后成了 `never`；
TS 7 则得到正确的 `number`。所以 `tsc --noEmit` 一直是通过的，只有 lint 报错。

**这是「lint 用 TS 6、tsc 用 TS 7」双编译器并存（ADR 0013）会系统性地暴露出来
的一类分歧：lint 看到的类型不一定等于 tsc 看到的类型。**
修法不是去赌哪个版本的 CFA 更聪明，而是换成不依赖它的写法（提取辅助函数后
`return found` 即可）。这条已写进 `setup.ts` 的注释，因为下一个人很可能再踩。


### 关于 TypeScript 的两个大版本

`package.json` 里 `typescript` 指向 TS 6、`@typescript/native` 指向 TS 7。
这是 `typescript-eslint@8` 的 peer 范围（`<6.1.0`）决定的 —— 它在 TS 7 上**导入时就抛错**
（跟踪 issue：typescript-eslint#10940）。
**重审条件**：typescript-eslint 支持 TS ≥ 7.1 后，把 `typescript` 换回 TS 7、
删掉 `@typescript/native`、重跑 `npm run lint`。

**已知代价（实测，不是推测）**：两个编译器对同一段代码可能给出**不同的类型**。
lint 跑在 TS 6 上，`tsc` 跑在 TS 7 上，所以「`tsc` 通过」不等于「lint 通过」。
第一次清尾时就在 `src/modules/markdown/setup.ts` 撞到过一次
（TS 6 把闭包赋值后的变量收窄成 `never`，TS 7 正确得到 `number`），
细节见第 20 项的记录。**遇到这种分歧，改写法，不要去赌哪个版本更聪明。**

---

## P3.5 · Markdown 完善（对标 horseMD 的功能清单）

> **顺序纪律**：先把 Markdown 这一类做到「完美」，再动思维导图与 draw.io。
> 功能参照 [`BND-1/horseMD`](https://github.com/BND-1/horseMD) 的清单，
> 差距分析见 [`docs/gap-analysis-horsemd.md`](docs/gap-analysis-horsemd.md)。

| # | 交付 | 状态 | 备注 |
|---|---|---|---|
| M1 | 字数统计（行 / 字 / 字符 / 阅读时长） | ✅ | Word 口径；`core/statistics.ts` + 状态栏 |
| M2 | **源码模式切换（Ctrl+/）** | ✅ | ADR 0014。源码视图复用代码内核，**没有引入新依赖** |
| M3 | **选中工具条（bubble menu）** | ✅ | ADR 0015。内核只上报选区状态与屏幕位置，浮层由外壳画；**没有引入新依赖** |
| M4 | **斜杠菜单（`/` 插入块）** | ✅ | ADR 0016。触发判定留在内核、菜单与键盘在外壳；**没有引入新依赖** |
| M5 | 表格增强 + 代码块（语言选择器 / 行号） | ✅ | ADR 0017。**自研两半，不引入官方 Vue 版组件** |
| M6 | 图片：粘贴落盘 + 相对路径解析 + 灯箱 | ✅ | ADR 0018。`<文档名>.assets/` + `data:` URI 显示 + 自研灯箱；**没有引入新依赖** |
| M7 | LaTeX 数学公式（KaTeX） | ✅ | ADR 0019。**不用已废弃的官方插件**，自研节点 + 输入规则 + NodeView；KaTeX 动态加载不进主包；导出内联 CSS 与 20 个 woff2 |
| M8 | Mermaid 图表（懒加载 + 全屏查看） | ✅ | ADR 0020。**图与源码是同一块的两种看法**（光标在块里就是源码）；mermaid 动态加载不进主包（主包仅 +9.8 kB）；导出画成 SVG，永远按浅色 |
| M9 | 文档位置记忆（光标 / 滚动）+ 设置页 | ⏳ | 与 P3 第 12、15 项同源 |

**M1 – M8 已完成**。M1 – M6 的共同点是**没有引入任何新依赖** ——
`@milkdown/kit` 已经带齐了大部分能力，M5 / M6 都没有破例（**这是一个刻意的选择，见下**）。
M7 是第一个真的加依赖的阶段（`katex` + `remark-math`），
它按预先立下的两条约束办的：「自包含导出」（内联 CSS 与字体）与
「不拖累主包体积」（KaTeX 走动态 import，实测主包大小未变）。
M8（Mermaid）沿用同一套约束，代价比 M7 大得多（见下）。

### M7 的收尾状态（LaTeX 数学公式，ADR 0019）

M7 的关键判断是**「打字时的识别」与「重新打开文件时的识别」必须是同一个答案**。
这两处各写一套规则就一定会分叉，而分叉的表现是「打字时看着是公式、
存盘重开变成一段字面文本」。所以 `core/math.ts` 直接把候选串喂给
`remark-math` 跑一遍 —— 同一个解析器，同一个答案。

由此引出三个不显然的坑（都写进了实现）：

- ProseMirror 的 `textBefore` 会被**截断到 500 字符**，长段落里打公式会静默
  认不出来 —— 取整块文本，不取 `textBefore`；
- remark **不会**为末尾空格留文本节点，所以判「公式是否贴着末尾」必须用
  **绝对偏移**（`position.end.offset === text.length`），只看节点类型会误判；
- `$5 到 $` 会被上游当成公式。**不修正** —— 修正就等于在识别上偏离上游，
  第 1 条判断立刻破掉。

#### 修掉的两条「静默丢数据」真 bug

两条都是**探针先证实、再修、再补回归用例**，不是读代码读出来的：

| 现象 | 根因 | 修法 |
|---|---|---|
| 打 `$$ ` 只得到字面量，整次输入无声失败 | 块级公式是块级原子节点，而 `setBlockType` **只接受 textblock**，抛 `RangeError`（未捕获，顺着 `handleTextInput` 冒出来） | 改用 `tr.replaceWith($end.before(), $end.after(), type.create())` |
| 斜杠菜单插入公式会**吃掉整段文字**（`前文/公` → 只剩 `$$\n$$`） | 无条件 `replaceWith($from.before(), $from.after(), node)` | 段落为空才整块替换，有文字就插在后面 |

#### 体积：798 KB 死重（两段弯路）

`katex.min.css` 的每条 `@font-face` 指向 woff2 / woff / ttf 三个文件，
编辑器那份 CSS 一旦 `@import` 它，Vite 就把**三个都**产出：实测 60 个文件、
1.05 MB，其中 **798 KB 永远不会被取用**（目标环境只有 WebView2 与 WKWebView，
两者从 2016 年起就支持 woff2）。

修的过程走了两条弯路，**两条都是「构建成功但产物坏」**：

1. **Vite 的 `transform` 钩子改不到 `@import` 进来的 CSS。** `@import` 由
   `postcss-import` 在 PostCSS 管线内部读盘内联，**不经过 Vite 的模块图** ——
   钩子挂在一个永远不会被经过的路口上（探针证实：`katex.min.css` 从未以
   模块形式出现在 `resolveId` / `load` / `transform` 的任何一次调用里）。
   改用 PostCSS 插件。
2. **把 CSS 重新 `postcss.parse` 一遍会让字体 404。** Vite 解析 `url()` 用的
   importer 是**每个声明自己的** `declaration.source.input.file`；重建节点把
   来源从 `katex.min.css` 改成 `markdown.css`，相对路径的基准就变了 ——
   日志里是 `didn't resolve at build time`，而构建照样成功。
   改成**就地改 `decl.value`**，一个节点都不重建。

修复后：dist 从 3.2 MB 降到 **2.4 MB**，字体从 1.05 MB 降到
**253.7 KB（20 个 woff2）**，woff2 那部分的资源哈希与修复前**逐字节一致**。


### M8 的收尾状态（Mermaid 图表，ADR 0020）

M8 的关键判断是**「图」和「源码」不是两个东西，是同一个代码块的两种看法**。
光标在块里就是源码，离开就是图 —— 而不是「一个块加一个预览面板」。
后者要用户自己记得把预览关掉，结果是文档里留着一堆忘了收起的图。

由此定下四件事，每一件都有一个只能靠实测发现的细节：

- **谁提供渲染容器**。`mermaid.render` 的第三个参数省掉的话，mermaid 会把
  量尺寸用的临时节点挂到**真实页面的 body** 上；而容器又必须真的参与布局
  （`getBBox()` 在 `display: none` 或游离节点上返回全 0，Firefox 直接抛错），
  所以容器用 `position: fixed; left: -99999px`。
- **「光标在不在块里」由插件推给视图**。ProseMirror 给 NodeView 的两个钩子
  都不是「选区变了」，而图的显隐完全由选区决定 —— 插件持一份活跃视图表，
  在 `PluginView.update()` 里逐个推。
- **同一个事务里的顺序**。ProseMirror **先**跑 `nodeView.update()`、**后**跑
  `PluginView.update()`，所以在 `update()` 里判断「要不要画图」会看到**上一帧**的
  `editing`，于是「在块里打第一个字」会多画一张没人看的图。修法是把图区的对齐
  统一放在 `syncSelection` 里。
- **打字期间不渲染**。否则每敲一个键都要让整张图重新布局一遍 —— 而那张图
  用户此刻根本看不见（显示的是源码）。离开块的那一刻才画。

#### 体积：主包只 +9.8 kB，但 `dist/` 大了 3.4 MB

| | M7 之后 | M8 之后 |
|---|---|---|
| 主包 `index-*.js` | 1,724 kB | 1,734 kB |
| `dist/` 总量 | 2.4 MB | 5.8 MB |
| 字体（woff2 / woff / ttf） | 20 / 0 / 0 | 20 / 0 / 0（未回退） |

mermaid 的 12.7 MB 模块图全在懒加载 chunk 里，主包那 +9.8 kB 是我们自己的代码
与 chunk 装载器。**两份 KaTeX**（各 259 kB）不是打包器的问题：mermaid 依赖
`katex@^0.16.47`，我们用 `^0.18.9`，semver 不重叠，npm 装了两份。
降版本能省掉它但会丢 0.18 的修复 —— 不划算，记进 ADR 0020 的重审条件。

#### 单测能证明什么、不能证明什么

**jsdom 没有布局引擎**，`getBBox()` 一律返回 0，**mermaid 在单测里画不出任何东西**。
所以「渲染」这一层是可注入的：编辑器与导出两边的单测都用假渲染器，
钉住的是**我们这一层的契约**（哪一块被换掉、失败留下什么、主题传对了没有、
容器有没有被摘掉），而不是「图画得好不好看」。后者只能人工看。


### M6 的收尾状态（图片：粘贴落盘 + 显示 + 灯箱，ADR 0018）

M6 的每一半都只有一句话，但**四层之间的接缝**才是真正的工作量：

```
ProseMirror 的 handlePaste（内核）
  → 字节交给外壳（EditorMountProps.onImagePaste）
  → 生成文件名并写盘（DocumentStore.pasteImage / core/document/images.ts）
  → 换回来的相对路径插进文档，再由 NodeView 读回来显示
```

#### 图片存哪、叫什么

`<文档名>.assets/`，与文档同级。**不是**全局的 `assets/`（一个目录里十份文档
会共用一个目录，删掉其中一份时没人知道哪些图还属于别人），**也不是**
`assets/<文档名>/`（目录名与文档名的对应关系藏进第二层，按名字排序看不出来）。

文件名是 `image-<时间戳>-<4 位随机盐><扩展名>`。**刻意不列目录去算 `image-1.png`** ——
那会把「生成名字」从一个纯函数变成一次可能失败的异步 IO，中间还夹着
「列完到写入之间别人占了这个名字」的竞态。代价是撞名时换一个盐重试（最多 3 次）。

#### 显示走 `data:` URI，不走 `asset:` 协议

Tauri 有 `convertFileSrc()`，能让 WebView 直接按路径流式加载本地文件。
**没用它**：要打开 `app.security.assetProtocol` 并配一份路径白名单，
那是一块新增的安全面。`data:` 走的是已有的 `read_binary_file` 命令，
权限收口在同一个地方。代价（base64 大 1/3、常驻 DOM）由上游 24MB 的上限兜住。
重审条件写在 ADR 0018。

#### 三条会静默坏掉的路径

1. **`ignoreMutation` 必须返回 `true`**。图片是叶子节点、没有 `contentDOM`，
   少了它 ProseMirror 会把我们写进 `<img>` 的 **data URI 当成用户输入读进文档** ——
   存出来的 Markdown 里多出一大段 base64，而界面上一切正常。
2. **粘贴必须自己声明 `USER_EDIT_META`**。落盘是异步的，等它回来时
   「刚发生过用户输入」的窗口早过了。不声明的话按 Ctrl+S 写回磁盘的是
   **粘贴之前**的文本，而界面上那张图还在。这是本项目**第五次**遇到这条路径
   （前四次：查找替换、浮层格式化、斜杠菜单、表格操作）。
3. **`joinPath` 要统一分隔符风格**。Markdown 里的图片地址必须是正斜杠，
   而 Windows 上的文档路径是反斜杠 —— 不统一的话，「写进去的路径」
   （文件名是裸名 → 反斜杠）与「回读时算出来的路径」（相对路径带正斜杠）
   是**两个不同的字符串**。文件系统不在乎，但按字符串比较的地方
   （缓存键、去重、相等判断）会悄悄失配。已加一条「逐字符相同」的用例钉住。

> 第 3 条是**在写 App 级测试时暴露出来的**：内核单测里 NodeView 的 resolver
> 是假的（返回什么都是我自己给的），所以看不到路径不匹配；只有走到
> 「真写盘 → 真回读」才表现为「图写进去了但显示不出来」。

#### 灯箱

外壳自己画（`ui/Lightbox.tsx`），**内核只报「点了哪张图」**。
一开始想让外壳在编辑器容器上做事件委托、按类名找 `<img>`，但那样外壳就得
认识内核画的类名 —— 而**外壳不许 import 任何具体内核**。把类名挪进 `core/`
只是把耦合换个地方藏起来。所以契约加了 `onImageClick`，监听挂在内核
自己造的那个元素上。

灯箱的底色**恒定是深的，不跟主题走** —— 这是全项目唯一一处刻意如此的地方：
图片的观感取决于周围亮度，浅底会让用户对「这张图有多亮」判断错。

#### 顺手补上的两件小事

- **浏览器预览模式的二进制盘**：`createBrowserProvider` 原本对 `readBinaryFile`
  一律抛 `NOT_SUPPORTED`。现在加了一个内存 `binaries` 表（刷新即失），
  于是 `npm run dev` 里能完整走到「粘一张图 → 占位消失 → 图出现 → 点开灯箱」
  这条链路 —— 而那正是最需要眼睛确认的一段。
- **Rust 侧的二进制写入命令**（`write_binary_file`）：请求体是原始字节
  （`InvokeBody::Raw`），路径走 `x-qm-path` 请求头（percent-encode，
  自带三十行 `percent_decode`，不引 crate）。**目标已存在就拒绝**，
  与「绝不静默覆盖」这条硬约束对齐。4 条 Rust 单测钉住 `percent_decode`。

#### 测试

- `core/document/images.test.ts` **18 条（新建）** —— 读取的六种降级、
  撞名重试、非 CONFLICT 不重试、写入被截断要报出来、`randomSalt` 的形状；
- `core/image.test.ts` **29 条**（上一批建）与 `core/document/assets.test.ts` **38 条**；
- `core/path.test.ts` +4（分隔符统一、重复分隔符不折叠、**写读路径逐字符相同**）；
- `modules/markdown/image.test.ts` **16 条**（内核半边）；
- `App.test.tsx` 图片链路 **10 条（新建）** —— 跨四层的那条端到端路径。

**尚未做、且只有人能做的**：图片在真实 WebView 里的显示效果与灯箱观感
（同 M3 / M4 / M5）—— jsdom 没有布局引擎，也不解码图片。

### M5 的收尾状态（表格增强 + 代码块，ADR 0017）

**先说一件在动手之前查清、并且改变了做法的事**：`ROADMAP` 原先写的是
「用 `kit/component/table-block` + `code-block`」，但去读 `node_modules` 之后发现
**`@milkdown/components` 是 Vue 3 的**。

- 它的 peer 里明确要 `vue@^3.5.20`，`code-block` 的 NodeView 直接 `createApp(...)`，
  还在 ProseMirror 的文档树里内嵌一个 **CodeMirror 6 实例**；
- 还会顺带拉进 `@floating-ui/dom` / `clsx` / `dompurify` / `lodash-es` / `nanoid`，
  以及 `plugin-diff`、`plugin-tooltip` —— 后两者在本项目里是**已经被 ADR 明确否掉的**。

采用它意味着：项目里出现第二套 UI 框架、编辑器的高亮管线被换成另一条
（导出用的还是 hljs，两边从此不同源）、以及一条被否过的依赖回归。
**所以两半都自己写。** 完整取舍与四条重审条件在 ADR 0017。

#### 表格（走装饰，不走 NodeView）

三条结论是**读 Milkdown 源码 + 实测**得到的，都不是推测：

1. **最小表格是 2 行**（`table_header_row` + 至少一个 `table_row`）。
   空表头行在序列化时被**静默丢掉**（`tableHeaderRowSchema.toMarkdown` 里
   `if (node.content.size === 0) return;`）—— 所以「插入表格」不能只给表头。
2. **对齐是按列、且由表头决定的**：`tableSchema.toMarkdown` 用**第一行**的单元格
   拼出对齐串，`keepTableAlignPlugin` 再把表头的 `alignment` 抄给同列的每个单元格。
   因此「给某一列设对齐」必须**写整列**，只写光标所在格会在存盘时丢掉。
3. `TableRole` 只有 `table | row | cell | header_cell`，**没有 `header_row`** ——
   表头行和正文行共用 `row`。想按「这是第几行」判断必须自己数。

界面侧：`describeTable()` 把光标处的表格状态（行列数、当前对齐、能不能删）
算成纯数据交给外壳，`TableToolbar` 只负责画。**内核不认识 React，外壳不认识 ProseMirror。**

#### 代码块（走 NodeView，因为行号需要真 DOM）

行号做不到用行内装饰：装饰只能改字符的样式，造不出块边界，也点不中空行。
语言标签同理 —— 它得**常驻在块上**，而不是随光标出现。

NodeView 里有三处是**不能省的**，写错了会以很难查的方式坏掉：

- **`contentDOM` 必须指向 `<code>`**，否则高亮装饰不会落进代码区（已用探针确认
  装饰仍然照常落在自定义 `contentDOM` 内部 —— 这是选 NodeView 而不是换掉
  高亮管线的前提）；
- **`renderChrome()` 先比差再碰 DOM**：打字时行号会重算，但只有在**行数真的变了**
  时才写 DOM。不然每敲一个字都产生一批 mutation；
- **`ignoreMutation` 要把 `attributes` 一律放过**：我们自己改类名/属性，
  不放过会被 ProseMirror 当成用户改文档。

#### 顺手修掉一条会静默丢数据的路径

语言选择器的**语言标签是个按钮，点它不产生键盘事件** ——
与 M3 的浮层按钮、查找替换是同一个问题：内核的变更上报只在「刚发生过用户输入」
的窗口内工作，所以 `setLanguage` 必须自己打 `USER_EDIT_META`。
否则改完语言按 Ctrl+S，界面显示「已保存」，写回磁盘的是改语言之前的文本。

同时加了一条**空操作守卫**：`setLanguage` 在语言没变时直接返回。
`setNodeAttribute` 即使写入同值也会让文档变脏 —— 点一下「当前语言」那一行
就把文件标成待保存，用户会以为误触了什么。

#### 面板怎么拿到「改哪个块」

面板**不持有文档坐标**。它拿到的是一次性的 `token` 加一个 `resolvePos()`，
真正写之前才重新解析位置（ProseMirror 的 `getPos()`）。
原因是 `setNodeAttribute` **不做类型检查**：位置一旦过期，它会把语言写到
**另一个节点**上，而且**不报错**。

> 这条是踩出来的：第一版存的是裸 `pos`，`setMarkdown` 换掉整篇文档之后，
> 再设语言就写到了一个无关的段落上，测试表现为「什么都没发生」。
> 改成 `resolvePos` 之后这个坑从设计上消失了。

#### 视图偏好不重建内核

「显示行号」是一个 **class 名**（`.qm-editor.qm-no-line-numbers`），
行号槽照建不误、由 CSS 负责显隐。这样翻转开关**不会重建内核** ——
光标、滚动位置、撤销栈全部保留。

#### 测试

- `modules/markdown/code-block.test.ts` **29 条**（`lineCount` 的口径、
  NodeView 的 DOM 与 mutation 行为、面板的开关与写入）；
- `modules/markdown/table.test.ts` **27 条**（`describeTable` 的六种光标位置、
  对齐写整列、序列化往返）；
- `App.test.tsx` 表格工具条 **12 条** + 代码块语言选择器 **14 条**；
- `ui/floating.test.ts` **12 条（新建）** —— 浮层摆放算术此前内联在工具条与斜杠菜单里，
  本次提取成纯函数，第一次有了单测；顺带加了 `horizontal: 'end'`（右对齐）。

**尚未做、且只有人能做的**：面板在真实 WebView 里的几何（同 M3 / M4）——
jsdom 没有布局引擎，`getBoundingClientRect()` 全是 0。

### M4 的收尾状态（斜杠菜单，ADR 0016）

- 契约加了 `onSlashChange`（内核 → 外壳）与 `EditorFormatting.insertBlock`（外壳 → 内核）。
  **与 `toggleBlock` 分开**，因为 `insertBlock` 多一件事要做：先删掉 `/query` ——
  要删哪几个字符是文档坐标，只有内核知道；
- 触发条件收得很紧（`describeSlash`）：光标塌缩 · 在段落里 · 光标在文本末尾 ·
  整段匹配 `/^\/(\S*)$/`。**筛选词不允许含空格**，否则用户写「一半 / 另一半」
  这种正文时会莫名弹出菜单；
- 两个浮层**共用了上报骨架**（`plugin.ts` 的 `floatingReporterPlugin<T>`）、
  **没有共用渲染组件** —— 两者的摆放规则相反（工具条默认在上、菜单默认在下），
  硬抽会变成一堆开关参数。判断依据写进了 ADR 0016；
- 键盘处理挂在 `window` 的**捕获阶段**：菜单打开时焦点还在编辑器里，
  ProseMirror 在冒泡路径上先到，默认监听会让 Enter 先把段落拆开；
- **Esc 只收菜单、不删用户打的字**：靠记住「哪一段筛选词被收起过」，
  而不是把 `/表` 删掉（删掉会改文档内容，用户只是想「算了不插了」）；
  **一个匹配项都没有时回车放行**（让 ProseMirror 拆段落）——
  否则「按回车没反应」，而用户打 `/zzz` 很可能就是想把它当正文；
- 新增测试脚手架 `src/test/typing.ts`：改 DOM → 让出宏任务让 `MutationObserver`
  记进队列 → 派发 `input` → 等一帧。它复现的是浏览器里的真实输入链路，
  不是绕过它直接塞事务 —— 斜杠触发是逐键推进的，绕过链路就等于没测；
- 测试：`modules/markdown/slash` +18、`App` +8。

**顺带查清一件既有行为**（不是 M4 引入的）：空段落套进引用 / 列表之后立刻保存，
文件里会出现 `> <br />`。来源是 Milkdown 的 `remarkPreserveEmptyLinePlugin`
把空段落序列化成 `<br />` 当**空行占位符**（没有它，空引用存盘再打开就消失了）。
它往返稳定，且打字之后占位符就没了。已记进 `docs/open-decisions.md` 待观察。

**尚未做、且只有人能做的**：菜单在真实 WebView 里的几何（同 M3）——
`getBoundingClientRect()` 在 jsdom 里全是 0，摆在哪、会不会被视口边缘切掉只能靠人眼看。

### M3 的收尾状态（选中浮层工具条，ADR 0015）

- 契约加了 `onSelectionChange`（内核 → 外壳）与 `EditorHandle.formatting`（外壳 → 内核），
  两侧都是**可选能力**，缺了就不显示浮层（不是禁用 —— 与 `sourceView` 相反）；
- 浮层是 `.editor-area` 的**兄弟**节点，用 `position: fixed` + 视口坐标摆放，
  上方放不下自动翻到下方、水平夹进视口；
- 格式名在外壳与内核之间翻译一次（`modules/markdown/selection.ts`），
  外壳只认 `strong / emphasis / strikethrough / inlineCode / link` 这五个词 ——
  顺带钉住 gfm 的删除线叫 `strike_through`（下划线写法）这个容易写错的细节；
- 选区上报用 `requestAnimationFrame` 合并，并且**文档变了也要重量**
  （点「加粗」不移动选区，只看选区的话按钮会停在旧的高亮上）；
- **修掉一条会静默丢数据的路径**：浮层按钮不产生 `input` 事件，
  格式化事务必须自己打 `USER_EDIT_META`，否则文档不变脏、Ctrl+S 写回的是格式化之前的文本；
- 测试基础设施补了 jsdom 缺的三个 DOM 接口（`Range.getClientRects` /
  `Range.getBoundingClientRect` / `HTMLElement.scrollIntoView`），
  否则任何触发 `scrollIntoView()` 的用例都会在一个与被测代码无关的地方抛错；
- 测试：`modules/markdown/formatting` +13（走真实内核，断言序列化出来的 markdown）、
  `App` +9（用真实 DOM 选区驱动，覆盖出现 / 消失 / 生效态 / 变脏 / 能力缺失）。

**尚未做、且只有人能做的**：浮层在真实 WebView 里的几何 ——
`coordsAtPos()` 在 jsdom 里没有布局引擎，所以「摆在哪、会不会压住文字、贴边时有没有翻边」
只能靠人眼看。这一条与 V3 同类。

### M2 的收尾状态（源码模式，ADR 0014）

- `EditorAdapter.sourceView` 让外壳不必知道源码视图底下是谁；
- 源码视图**不注册进注册表**（同一路径只能有一个内核答案）；
- `ViewMode` 住在 `StoreState`，所以 `reload()` / `saveAs()` 不会把它重置掉；
- 内核实例标识从 `session` 扩成 `session:viewMode`，
  顺带修掉「一次内核启动失败会把用户永久锁在兜底内核上」；
- Markdown 扩展名名单从三处抄本收敛到 `core/path.ts`（源码视图是第四个需要它的地方）；
- 测试：`path` +4、`store` +7、`modules/wiring` +5（跨模块的名单一致性断言）、
  `App` +8（内核交棒、无损、切标签页不串台）。

**尚未做、且只有人能做的**：在真实 WebView 里目视确认切视图时的观感
（有没有闪一下、光标位置跳不跳）。逻辑上「内容无损」有测试钉住，但视觉过渡没有。

---

## P4 · 思维导图

| # | 交付 | 状态 | 前置 |
|---|---|---|---|
| 20 | **V2：mind-elixir 500 节点压测** | ❓ | 判据：输入到反馈 ≤ 50ms |
| 21 | `.mindmap` 读写（薄适配层，隔离内核版本变化） | ⏳ | 依赖 20 |
| 22 | 导图编辑器接入（`EditorAdapter` 的第三个实现） | ⏳ | |
| 23 | 全文检索（Rust 流式 + 可取消 + 尊重 `.gitignore`） | ⏳ | |

---

## P5 · draw.io 图表

| # | 交付 | 状态 | 前置 |
|---|---|---|---|
| 24 | **V1：`viewer.min.js` 能否独立渲染多页与压缩变体** | ❓ | 体积判据：≤ 6 MB |
| 25 | 内置只读查看器（`doc-block` 引用在编辑器里渲染成真图） | ⏳ | 依赖 24 |
| 26 | 编辑器按需下载（显式、可见进度、可取消） | ⏳ | |
| 27 | 完整离线构建目标（内网 / 涉密环境） | ⏳ | |

**P5 结束是第一个可灰度发布的节点**（四个模块独立可用）。

---

## P6 · 差异化

| # | 交付 | 状态 |
|---|---|---|
| 28 | 块级互操作的完整体验（引用 / 就地编辑 / 结构变换 L1–L3） | ⏳ |
| 29 | 「往返保真度自检」：一个显式的、如实报告差异的动作 | ⏳ |
| 30 | 分屏 / 源码 + 预览双栏 | ⏳ |
| 31 | 自定义快捷键（录制 + 冲突提示） | ⏳ |

---

## P7 · 发布

| # | 交付 | 状态 | 备注 |
|---|---|---|---|
| 32 | macOS 签名与公证 | ⏳ | 需 Apple 开发者账号（99 美元/年），**申请有周期，应提前启动** |
| 33 | Windows 代码签名（OV / EV） | ⏳ | 需企业主体 |
| 34 | 依赖许可证清单 | ⏳ | |
| 35 | 每周 macOS 渲染冒烟 | ❌ | 阻塞：需要一台可用的 Mac（ADR 0006） |
| 36 | 商标查询（第 9 / 42 类） | ⏳ | 中国商标网 |
| 37 | 域名与名称占用检查 | ⏳ | |

---

## 明确不做

| 项 | 理由 |
|---|---|
| 云同步（WebDAV / S3） | 与「本地优先、无服务端、数据在用户手里」的定位冲突（ADR 0001） |
| 移动端（Capacitor） | 只能做只读，投入产出比低，且拖累桌面端迭代 |
| AI 功能 | 没有明确用户诉求前不做 |
| Pandoc 打包（Word / EPUB / LaTeX） | 外部二进制 + 跨平台分发成本高。**若要做，走「检测到 pandoc 才显示」** |
| 图床（PicGo） | 图片一旦上传就走出了用户的文件系统 |
| Web 版 | 需要服务端读写用户文件系统，等于扔掉「本地优先」这个最有价值的前提（ADR 0001） |

---

## 执行纪律

> **`open-decisions.md` 里的 V1–V4 在没有结论之前，不要再新增待验证项。**

这条比任何功能都重要。轻墨的决策记录写得比多数项目清楚，
但 V1–V4 至今一项未跑 —— 一份只写不执行的决策记录会退化成负资产。
