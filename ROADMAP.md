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
| 13 | 图片粘贴落盘 + 相对路径策略 | ⏳ | 自包含导出的最后一块拼图 |
| 14 | Rust 侧 `watch` 落地（`notify` + debouncer） | ⏳ | 替换「窗口焦点时比对 mtime」的权宜之计 |
| 15 | 设置页 | ⏳ | |
| 16 | PDF 导出（WebView 打印管线） | ⏳ | 与自包含导出共用渲染结果 |
| 17 | i18n 骨架（先抽文案，不急翻译） | ⏳ | 越晚做越贵 |
| 18 | 主包拆分（`React.lazy` 加载内核） | ⏳ | 外壳首屏目标 < 200 kB。**实测数据见下** |
| 19 | 标签页切换保留撤销栈与光标 | ⏳ | 需先实测内存占用（见 ADR 0010 的重审条件） |
| 20 | **lint 清尾** | ✅ | 已完成，记录见下 |

### 主包体积（第 18 项的起点数据）

`npx vite build` 的实测产出（2026-09-29，M4 之后）：

```
dist/assets/index-BU_XDXo0.js    1,680.13 kB │ gzip: 559.55 kB
dist/assets/index-JNieFIhK.css      20.63 kB │ gzip:   4.91 kB
```

> M3 结束时是 `1,675.62 kB / gzip 558.41 kB`（CSS `19.30 / 4.67`）。
> M4 的增量是 **+4.51 kB / gzip +1.14 kB**（斜杠菜单组件 + 触发判定 + 菜单样式）。
> **同样没有引入任何新依赖** —— 九个可插入的块全部走已经挂载的
> `commonmark` / `gfm` 预设命令。

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
| M5 | 表格增强 + 代码块（语言选择器 / 行号） | ⏳ | `kit/component/table-block` + `code-block` |
| M6 | 图片：粘贴落盘 + 相对路径解析 + 灯箱 | ⏳ | 与 P3 第 13 项同源 |
| M7 | LaTeX 数学公式（KaTeX） | ⏳ | 新依赖；**必须内联 KaTeX 的 CSS 与字体**才能保证自包含导出 |
| M8 | Mermaid 图表（懒加载 + 全屏查看） | ⏳ | 大依赖，必须动态 import，不能进主包 |
| M9 | 文档位置记忆（光标 / 滚动）+ 设置页 | ⏳ | 与 P3 第 12、15 项同源 |

**M1 – M4 已完成**，共同点是**没有引入任何新依赖** ——
`@milkdown/kit` 已经带齐了大部分能力，M5 同样不需要新包。
只有 M7（KaTeX）与 M8（Mermaid）会真的加依赖，那两项要先解决「自包含导出」与
「不拖累主包体积」这两个约束再动手。

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
