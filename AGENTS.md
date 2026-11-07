# AGENTS.md · 上手手册

> 给 AI 助手与**新加入的人**。目标：十分钟内能安全地改代码，而不是先踩一遍别人踩过的坑。
>
> 人类贡献者的流程在 [`CONTRIBUTING.md`](CONTRIBUTING.md)，决策的理由在
> [`docs/decisions/`](docs/decisions/README.md)。本文只讲**动手时必须知道的事**。

---

## 一、这个项目是什么

一款**纯本地**的桌面编辑器，四类文档（Markdown / 代码文本 / 思维导图 / draw.io 图表）
在同一个工作区里作为一等公民共存。技术栈：**Tauri 2 + React 19 + TypeScript（严格模式）**。

**没有任何网络请求。** 没有服务端、没有账号、没有遥测。
看到「加一个上传/同步/登录」的需求时，先读 ADR 0001 —— 那是有意砍掉的。

---

## 二、硬约束

**这份清单是编号的唯一来源。** 代码注释里写的「硬约束 N」指的就是下面第 N 条 ——
改任何一条之前，先去找它的完整理由（在对应的 ADR、或那一行代码旁边的注释里）。

前三条是地基（破坏它们会让整个架构塌掉），后面十一条是「换个写法就会坏」的具体约束。

### 1. 依赖方向是单向的

```
modules/（内核）  →  core/editor/types.ts（契约）  ←  ui/（外壳）
```

- 外壳**不许** import 任何具体内核（Milkdown / CodeMirror 的名字不许出现在 `ui/` 里）。
- 内核**不许**直接碰文件系统。
- 内核与 UI **不许** import `@tauri-apps/*`（只有 `core/platform/desktop.ts` 可以）。

### 2. 绝不静默

不静默转码、不静默覆盖、不静默丢内容、不静默降级。
任何降级都要在 UI 上说清楚（「N 张图片读不到，只保留了原链接」）。
这是 ADR 0003 与 `docs/format-spec.md` §8 定的红线。

### 3. 打开任何文件都不能失败

`modules/plaintext` 是兜底内核，**不注册进注册表**，而是作为常量直接交给外壳 ——
这样它不可能被某个内核的错误配置遮蔽掉。任何改动都不能破坏这条。

### 4–14

4. **注册顺序即优先级**：`registerAdapter` 是 `push` + 先注册先匹配。
   **备胎必须排在正主后面** —— 现在的顺序是 `markdown` → `mindmap` → `drawio` → `code`，
   而 `code` 认得 `.md`、`.mindmap` 与 `.drawio`（它是三者的源码形态）。
   把它排前面会把两个正主全吃掉：`matches` 先命中、它又没有 `canHandle`，
   注册表直接返回它，后面那个内核**一次都不会被调用**。
   踩过一次（第一版就是 `markdown` → `code` → `mindmap`），
   `modules/wiring.test.ts` 的备胎链测试当场抓住 —— 见 ADR 0025。
5. **Markdown 扩展名名单只有一份**：`core/path.ts` 的 `MARKDOWN_EXTENSIONS`
   （`isMarkdownPath()`）。抄多份的后果是**双重落空** —— 用户拿到白底 textarea，
   而两边代码都没写错。`modules/wiring.test.ts` 跨模块断言钉住它。
6. **`EditorAdapter.sourceView` 不注册进注册表**：注册表按路径挑内核，
   同一路径只能有一个答案（ADR 0014）。源码视图是「同一路径的第二种看法」。
7. **内核里任何绕过键盘事件改文档的能力，必须自己给事务打 `USER_EDIT_META`。**
   `markdownEmitPlugin` 只在「刚发生过用户输入」的窗口内上报，而查找替换 /
   浮层按钮 / 斜杠菜单**不产生 `input` / `paste`**。漏了 = **静默丢数据**
   （文档不变脏，Ctrl+S 写回改动**之前**的文本，界面还显示「已保存」）。
   已踩**六次**。`withUserEdit` 与 `withUserEditOnView` 是同一个东西的两半，不要各写一份。
   **图表预览是唯一不需要打标记的**（只改选区、不改文档）。
8. **`beforeinput` 是内核判定「这次变更是不是用户干的」的唯一可靠依据。**
   ProseMirror **不监听 `input`**（靠 `MutationObserver`，回调是微任务，
   跑在任何事件派发之前）。只挂 `input` 会让**第一次**变更被安静丢掉。
9. **浮层菜单的键盘必须挂在 `window` 的捕获阶段**（`addEventListener(..., true)`）。
   默认监听的话回车会先把段落拆开 —— ProseMirror 的 `handleKeyDown`
   在 `view.dom` 的冒泡路径上先到。配套：菜单项按钮要 `preventDefault` 掉 `mousedown`。
10. **浮层 / 面板不许持有文档坐标。** `setNodeAttribute` **不检查节点类型** ——
    位置过期会安静写到另一个不相干的节点上。用 `resolvePos: () => number | null`，
    **写的那一刻现问**。
11. **外壳不许按类名 / DOM 结构去认内核画的东西。** 正确做法是**加一条契约方法**
    （`EditorMountProps` 上的 `onImageClick` / `onDiagramOpen` …），
    监听挂在内核自己造的元素上。判断依据：**只要需要外壳知道内核内部的 DOM 结构 /
    类名 / 节点类型名，它就该是一条契约方法。**
12. **`joinPath` 把相对部分的分隔符统一成 `base` 的风格。** Markdown 图片地址必须
    正斜杠、Windows 文档路径是反斜杠；不统一则「写入路径」与「回读路径」是**两个字符串**，
    只在**按字符串比较**处出问题（缓存键 / 去重）→「图写进去了但显示不出来」。
    **只换风格**，不折叠重复分隔符、不解析 `..`。
13. **样式表按「属性」划所有权，不按文件。** 内核根节点的**滚动**由内核自己的样式表声明
    （Markdown 是 `.qm-editor`，CodeMirror 是 `.cm-scroller`）；外壳只声明
    「把内核根节点压进 flex 容器」（`flex` / `min-height`）。
    两边都声明同一个属性时**谁赢只看打包顺序**，而 `styles.css` 在产物里靠后、
    会**安静地赢** —— 表现是「打开长文档滚不动」，且**从 `a224ec4` 就存在**
    （jsdom 没有布局引擎，照不出来）。回归测试 `src/ui/styles.test.ts`。
    > 硬约束 11 的同胞：11 说「外壳不许按类名认内核画的东西」，这条说「也不许替内核决定属性」。
14. **构建期注入的常量必须写成 `import.meta.env.X`，不能写成裸标识符。**
    Vite dev server **不替换裸标识符形式的 `define`**（`vite build` 与 vitest 都替换
    → **单测全绿而开发时崩**）。踩过：设置页的版本号渲染时抛 `ReferenceError`，
    没有错误边界，**React 卸掉整棵渲染树** —— 点一下设置应用就没了。
    **凡是依赖构建期替换的东西，都要在真正跑起来的那条路径上验一次。**

---

## 三、代码地图

| 想改什么 | 去哪 |
|---|---|
| 文档的打开/保存/冲突/编码 | `core/document/`（**先读 `store.ts` 顶部的「核心不变量」注释**） |
| 多标签、文件树 | `core/workspace/` |
| **命令的失败出口（唯一一处）** | `core/commands/runner.ts` 的 `runCommandSafely(command, ctx)` —— 四个入口（菜单 / 活动栏 / 快捷键 / 命令面板）共用它，**不许再各写一份 `.catch(console.warn)`**（此前抄三遍，修一处漏三处）。它是**最后**一层，不是唯一一层：已有七处命令自己处理预期内的失败。**要命令不落到这里，就在命令内部接住并说明，不许在兜底层加白名单**。判据：**一个会失败的异步能力，它的类型不许是 `void`**（`toggleFullscreen` 因此改成 `Promise<void>`）（ADR 0041） |
| **拖入多个文件怎么打开** | `core/workspace/drop.ts`（`openDroppedPaths` / `describeDropOutcome`）。逐个顺序打开、**失败只记账不中断**、已开着的同名文件算成功；`App.tsx` 只接线。否掉并行 `Promise.all`（打开有顺序语义：reveal / 激活哪一个）（ADR 0041） |
| 快捷键、命令、命令面板 | `core/commands/`（能力清单在 `definitions.ts`；**title/category 只声明 key、经 `builtInCommand` 取 `core/i18n/` 的文案**，见 ADR 0031） |
| **用户可见文案** | `core/i18n/`（`messages.ts` 目录 + `index.ts` 的 `t()`；key 写错是编译错误，漏 key 回退到 key 本身） |
| 查找替换的匹配语义 | `core/search/find.ts`（三个内核共用） |
| **往返保真自检**（保存会改写什么） | `core/fidelity.ts`（行级比对，纯函数）+ `EditorHandle.roundTrip`（内核实现，见 ADR 0033） |
| **视图偏好（主题 / 侧栏 / 行号 / 位置记忆开关）** | `core/settings.ts`（`localStorage`，逐字段校验）。**设置页只是它的一个界面** —— 加偏好项改这里，别在 `SettingsDialog.tsx` 里另起一份 |
| **偏好 / 位置 / 最近打开写不进去时怎么说** | `core/settings.ts` 的 `PersistOutcome`（`{ ok: true }` \| `{ ok: false, reason: 'no-storage' \| 'write-failed' }`）+ `ui/App.tsx` 的 `reportPersistFailure(outcome)`。**四处写偏好（设置 / 位置 / 位置清空 / 最近打开）全部走它**，翻会话横幅那一枚 `writeFailed`，不许各弹各的。`no-storage`（浏览器预览 / 隐私模式）**刻意不说话** —— 那是预期内的降级；只有 `write-failed`（配额 / 策略）才说。判据同样是「会失败的能力类型不许是 `void`」（ADR 0042） |
| **存储里的东西读不出来时怎么说** | `core/settings.ts` 的 `LoadOutcome` + `LoadResult<T>`（**ADR 0042 的对偶**）。`loadSettings` / `loadPositions` / `loadRecentFiles` / `loadSession` **四个一律返回 `{ value, outcome }`**，值照样回落（白屏比丢失更糟），但回落**不等于沉默**。三档 reason：`no-storage` 折成「没丢」（预期内降级）、`read-failed`（`getItem` 抛）、`corrupt`（内容用不了）。**判据只有一条：键在、值用不了才算丢失** —— 键不在是版本升级的新字段、夹取是意图已兑现、字体族消毒，都**不报**（实现见 `core/settings.ts` 的 `pickField`：`raw === undefined` 走安静那条）。点名 vs 只报数由**键名**决定：路径能点名（`ui/App.tsx` 的 `READ_LOSS_NAMES_MAX = 3`，全列出来就是一面墙），`sidebarWidth` 这种内部标识只报数；**整份报废时 `lostCount` 一律写死 1**，写 0 等于把一次丢失又藏了回去。`loadSession` **只报整份报废**，逐条丢弃继续走 `droppedDrafts`，同一件事不说两遍。读侧横幅与写侧横幅**分两条**（ADR 0044） |
| **文件监听没起来时怎么说** | `ui/App.tsx` 的 `WatchIssue`（`{ reason, failedPaths }`）+ `foldWatchIssue` / `watchIssueSignature` / `watchReasonOf`，第三条横幅。**这一族的判据不一样**：丢的不是数据，是「**早点知道的机会**」—— 监听一停，目录树不刷新、冲突检测不报警，用户可以在毫无察觉的情况下把别人的改动覆盖掉，而兜底那两条路（下一批事件、窗口聚焦的全量比对）都建立在「事件还会来」上。**它不会自愈**（目录被删 / inotify-kqueue 额度耗尽 = 事件永久不再到达），所以不能只 `console.warn`（发布版里没人看）。两路来源（批次里的 `error`、`watch_replace` 返回的失败路径）**各占一个槽、整份替换**——不累加（严格模式双挂不翻倍），也因此能自愈（正常的一批把 `reason` 清回 `null`）。`reason` 是**三态**：`null` 没问题 / `''` 有问题但拿不出原因 / 原文；压成两态会让「说不出原因」整条消失，**报 0 等于把一次故障藏回去**。两档措辞：整条链可能断 →「外部改动可能发现不了」，只是几条路径没监视上 →「有 {n} 条路径没在监视」（多半只是那几条被删/移了，重话说就是虚惊，虚惊多了用户就开始忽略这一类）。按掉存**指纹**不存布尔（关掉 A 不该把后来的 B 一起关掉）。**Rust 侧零改动**：`error: Option<String>` 与失败路径数组本来就是为此留的，只是前端一直没接。**别再把监听的任何失败写成 `console.warn`**，也**别给 `watchFiles` / `onFilesChanged` 的 promise `void` 掉而不接 catch**（ADR 0045） |
| **说过的话留不留得下来（问题清单）** | `core/health.ts`（纯函数记录本，**只活一次运行、不落盘**）+ `ui/HealthDialog.tsx`（帮助菜单 → 「本次运行的问题」，命令 `diagnostics.issues`，**刻意不给快捷键**）。**这一族回答的是「横幅要求用户正好在看」**：按掉横幅的动机恰恰是「我不想现在处理」，一次运行里先后三四件事横幅一次只占一行，关掉应用就没了。记的是 `HealthLine{key, params}`（= `LocalizedText`）—— **不是句子**（在记账那一刻 `t()` 会把语言烤死，与 ADR 0031 的 `BUILT_IN_COMMANDS` 同一类错）。两条上报通道分开：**横幅族走 `syncFamily`**（有的记上、消失的**标成已恢复**，`resolvedAt` 不许被覆盖），**一次性事件直接 `recordIssue`**（内核加载、写失败、恢复的草稿与文件失败 —— 它们没有「恢复」这个状态）。三条纪律：① **内容没变就什么都不做**（上报点在渲染里，双挂不是又出一次问题，否则「发生过 2 次」对着只发生过一次的事说话）；② **写失败要加 ref 闩**（位置记忆每 600ms 落一次盘，不加就是「发生过 47 次」；闩只能挂在**上报点**上，不能从横幅的 `writeFailed` 推 —— 那个会被「知道了」清掉）；③ **恢复的记录永不删除**，只打标记（自动收起来是这一族里最难看的一种静默）。「**复制全部**」是纯本地应用唯一一条把状态交给别人的路（ADR 0001：没有账号、没有网络、没有遥测），所以那份文本必须脱离界面也读得懂，剪贴板失败要有一张脸。**加了新的一族要补 `AREA_LABEL_KEYS`**（`satisfies` 钉死，忘写标签名编译期就拦）（ADR 0046） |
| **`t()` 的结果会跨渲染存活的地方** | `core/i18n/index.ts` 的 `LocalizedText{key, params}` + `msg()` / `textOf()`（`core/health.ts` 的 `HealthLine` 是它的别名）。**判据不是「这个文件属于 core 还是 ui」，是「`t()` 的结果有没有进 state / store 字段 / 会被存起来的返回值」** —— 进了就等于把语言烤死在调用那一刻。已收干净的四处：`TreeState.error`（`TreeProblem`，不带恒定的 `code`，`detail` 是抛物原话**不许翻**）与 `validateEntryName()` 的八条规则、`ReplaceReport`（片段数组 + 问题行，**句子只在 `renderReplaceReport()` 里拼，连逗号都是一个 key**）、`SettingsDialog.error`、`PasteImageFail.reason`（这一处一交 key，`DocumentStore` 的 `legacyText()` 品牌门就整个删掉了 —— **没有门后的门比没有门更糟**）。**抛物原话当 `params` 走，绝不当句子翻**（翻一句 `EACCES` 得到的是听起来像回事的谎）。**别再把 `t()` 的结果塞进任何活过渲染的结构**（ADR 0047） |
| **新文件该写什么内容 / 新建入口** | `core/new-file-content.ts` 是**唯一的出口**，判据只用 `core/path.ts` 的 `is*Path`，**不许 import 任何 `modules/`**（硬约束 1）。模板**生成**而不是抄字面量（抄的那份会在格式演进时先掉队）：`core/mindmap/template.ts` 的 `emptyMindmapText()`、`core/drawio.ts` 的 `drawioTemplateXml()`（doc-block 的起始内容也复用它 —— **起始内容只有一份**）。用**各自的判据**验，不照抄字面量：`.mindmap` 看 `parseMindmap(...).canonical`（打开→保存无 diff），`.drawio` 看 `missingStencilLibs` 恒为空（随包只内置 bpmn/flowchart，新文件一打开就顶一条缺库警告不是迎接用户的方式）。`core/commands/definitions.ts` 的 `writeNewFileThenOpen` 是**三个新建入口**（`file.new` / `file.newMindmap` / 示例文档）共用的编排，差异只留数据。**加一种有格式的文件类型时三件事都要做**：判据加一条、模板加一份、命令加一个 —— 漏掉的那一处是**静默**给用户一个空文件（ADR 0048） |
| **`.mindmap` / `.drawio` 各自的独立内核** | `modules/mindmap/index.ts`（`canHandle` + `canHandleReason` + `sourceView: codeAdapter`，ADR 0007 / 0025 / 0043）与 `modules/drawio/index.ts`（**刻意不实现 `canHandle`** —— 查看器自己会把「这不是合法的 XML」写进元素里，`renderDrawio` 读出来当 `DrawioRenderFail.message` 比适配层再判一遍**更准**，还省掉 ADR 0043 那张 kind 表的一个上限）。注册顺序 markdown → mindmap → **drawio** → code。三条纪律：① **`CODE_EXTENSIONS` 里 `.drawio` 与 `.mindmap` 必须留着** —— 它们是这两个内核的 `sourceView`，也就是它们的编辑入口兼内核被否决时的备胎；② **只读内核从不调用 `onChange`** —— 不能改文档就没有「改了什么」可报，报了就会把没变的文件标脏并重写一遍；③ **能力缺失写在面板上**，不靠「没那个按钮」让用户自己发现（硬约束 2），且要给出可走的下一步（图纸 → Ctrl+2 源码视图）。`.drawio` 的文档内容就是那段 XML，`onChange` 一接上就是可编辑，**内核契约一个字都不用改**（ADR 0048） |
| **内核自己画的界面文案** | `modules/mindmap/lang-pack.ts` 的 `mindmapLangPack()` 十二键加 `newTopicName`，**现查现变**（模块级常量会在模块被求值那一刻烤死语言，ADR 0031 的坑）。内核在**安装那一刻**把右键菜单十一条烤成 DOM（`MindElixir.js:1394` 的 `innerHTML`）—— 所以 `contextMenu` 传 `true` 只能拿到它硬编码的英文包，**要传对象**才带得上我们的语言包。语言一变只能**重建内核**：`useSyncExternalStore(subscribeLocale, getLocale)` 把 `locale` 放进主 effect 的依赖数组（全项目第一处真的订阅语言变化的消费方）。重建的三条命门，缺一条就是一次静默的坏：① 种子取 `latestRef`（「此刻 Ctrl+S 会写出的那份文本」，拿 `initialValue` 重新解析会让用户的修改**静默回滚**）；② `getValue()` 里记进 `latestRef` 的是**交出去的那个值**不是 `lastText` —— 记错的话下一次保存把用户文件**重排版**，「打开 → 保存逐字节一致」当场破掉；③ 缩放用公开的 `scaleVal` / `scale()` 还原，而选中项 / 专注模式 / 平移**不保**（都不是公开 API，别去撬内部状态，升级即烂）（ADR 0049） |
| **「上次读到哪儿」（光标 / 滚动）** | `core/positions.ts`（存储：独立键 + LRU 200 条）。**位置的坐标系由内核自己定**，外壳只做往返 —— 见 ADR 0021，别想着在外壳里把它换算成「第几个字符」 |
| **滚动比例那四行算术** | `core/editor/position.ts`（`readScrollRatio` / `applyScrollRatio`）。三个内核共用；**别各写一份**（其中两份一定会忘了判除零） |
| 字数 / 阅读时长 | `core/statistics.ts`（Word 口径，内核无关） |
| **哪些文件算 Markdown** | `core/path.ts` 的 `MARKDOWN_EXTENSIONS` / `isMarkdownPath()`。**名单只有这一份**，四个地方在用它 |
| **所有浮层摆在哪**（位置算术） | `ui/floating.ts` 的 `computeFloatingPlacement`（**纯函数**，12 条单测）。四个浮层共用：选中工具条（`center`/上）、斜杠菜单（`start`/下）、表格工具条（`center`/下）、代码块语言面板（`end`/下）。**改摆放规则改这里，不要在组件里再抄一份** —— 它被抄过两遍才提取出来的 |
| 选中浮层工具条（格式按钮、浮层位置） | `ui/SelectionToolbar.tsx`（外壳画浮层）+ `modules/markdown/selection.ts`（内核报选区状态） |
| 斜杠菜单（`/` 插入块） | `ui/SlashMenu.tsx`（外壳画菜单与键盘导航）+ `modules/markdown/selection.ts` 的 `describeSlash`（内核判断触发） |
| 表格工具条（加行加列、对齐、删除） | `ui/TableToolbar.tsx` + `modules/markdown/table.ts` 的 `describeTable`（纯函数，把光标处的表格算成数据）。**对齐是按列的**，见坑 12 |
| 代码块（行号 / 语言标签） | `modules/markdown/code-block-view.ts`（NodeView，**画在文档树里**）+ `ui/CodeBlockPicker.tsx`（外壳画语言面板） |
| **代码高亮**（编辑器与导出共用） | `core/highlight/hljs.ts` + `core/highlight/cache.ts`（**LRU 缓存**）。高亮在**每次文档变更**时整篇重跑（`codeHighlightPlugin` 的 `apply`），是按键路径上的常客 —— 所以**一律走 `highlightCode`**（它带缓存），不要绕开它直接调 hljs。详见 `docs/open-decisions.md` 的「V3 的后续」 |
| **图片存在哪、叫什么名** | `core/document/assets.ts`（目录布局与命名，**纯函数**）。`<文档名>.assets/` + `image-<时间戳>-<盐><后缀>`；Markdown 里一律写相对路径 |
| **图片的读与写** | `core/document/images.ts`（两个**无状态**的文档级动作：`readImageAsUrl` / `writePastedImage`）。**「已存在就换名字重试，绝不覆盖」这条在这里** |
| **图片类型（MIME ↔ 扩展名）** | `core/image.ts`。**只有这一份**，落盘 / 显示 / 导出三处共用 —— 抄成三份的下场是「编辑器里能看、导出后打不开」 |
| 图片显示（NodeView）与灯箱 | `modules/markdown/image-view.ts`（画占位与加载态）+ `ui/Lightbox.tsx`（外壳画全屏预览） |
| **什么算 Mermaid 图表** | `core/mermaid.ts` 的 `isMermaidLanguage` / `MERMAID_STARTER`。**判定只有这一份**，三处共用（NodeView、导出、斜杠菜单插入）—— 抄成三份的下场是「编辑器画得出、导出不画」 |
| **Mermaid 怎么渲染** | `core/mermaid-render.ts`。动态 `import('mermaid')`、**必须传容器**（不传它会往真实 `document.body` 挂临时节点）、容器必须**参与布局**（`display: none` 会让 `getBBox()` 全 0）。jsdom 里画不出图，所以测试用假渲染器 |
| **`doc-block` 怎么解析** | `core/docblock.ts`。属性区 + `---` + 载荷，载荷**逐字节原样**；不认识的键进 `extra` 原样保留（规范 §5 向前兼容，丢了就是单向数据损失）。永远不抛 —— 内容来自用户文件，失败走 `errors` |
| **`.drawio` 缺哪些 stencil 库** | `core/drawio.ts` 的 `stencilLibsOf` / `missingStencilLibs`；插入用的 **doc-block 起始内容**也在这份文件（`drawioDocBlockStarter`，inline 模式 + 文档内唯一 id，两个内核共用 —— 别在内核里各写一份）。**渲染前从 XML 风格串里算**（`shape=mxgraph.<库名>.`），与内置清单求差 —— 不要改成「事后看请求 404」（并发归属不了，XHR 打点是全局猴子补丁） |
| **draw.io 查看器怎么渲染** | `core/drawio-render.ts`。上游产物在 `public/drawio/`（ESLint 全量忽略），**出现图表引用才注入**脚本；注入前十条资源路径必须全部占住（「无网络」的唯一开关，漏一条就静默联网）。渲染器可注入（`DrawIoIo`），与 `MermaidIo` 同一条纪律。**不接主题**（颜色在图的 XML 里，画布恒白） |
| 图 / 源码双态与全屏入口 | `modules/markdown/code-block-view.ts`（`data-mode`）+ `ui/DiagramViewer.tsx`（外壳画全屏） |
| 导出 | `core/export/`（`document.ts` 组装，`inline-html.ts` 加工，`export-document.ts` 动作）。**打印 PDF 与它共用同一份渲染**：`print-document.ts` 动作 + `src-tauri/src/export/print.rs`（内存自定义协议 + 打印子窗口，ADR 0030）。**`print_html` 必须是 async 命令** —— 同步命令在主线程内联执行，里面开窗口要等 WebView2 回调，回调靠消息循环，循环正被命令占着 → 死锁（白屏 + 全应用关不掉，ADR 0030 §6） |
| **写回磁盘时被改写成什么样**（列表符号 / 分隔线 / 表格） | `modules/markdown/setup.ts` 的 `useDashSymbols`（`bullet` / `rule` 都设成 `'-'`）。**`remark-stringify` 的默认是 `'*'`**，改回去会让「打开一份用 `-` 写的文档 → **改一个字** → 保存」把全篇的 `-` 静默换成 `*`。清单与实测见 `docs/open-decisions.md` 的「原文保真的实测与收窄」 |
| Markdown 编辑体验 | `modules/markdown/` |
| **专注模式 / 打字机模式** | 两个视图偏好（ADR 0039）。Markdown 侧是 `modules/markdown/plugin.ts` 的 `composePlugin`——**两个开关共用一个状态对象**；切换走 `setComposeMode`（`modules/markdown/setup.ts`）**只派发一条带 meta 的事务，不重建内核**（重建会丢光标 / 滚动 / 撤销栈，且那条事务必须带 `addToHistory: false`）。代码内核只认打字机，状态是一个闭包对象而非 Compartment（不改编辑行为）。**打字机只写 `scrollTop`、绝不 dispatch**（dispatch → 位置记忆 → 又滚动，停不下来）；滚动算术只有一份，在 `core/editor/position.ts` 的 `typewriterScrollTop`（返回 `null` = 不用动，含「差值 < 1px」那个阈值）。浓淡归内核样式表 `markdown.css` 的 `.qm-dimmed`（硬约束 13）。专注**只看顶层块**；打字机与专注的生效范围不对称，且**范围写在设置页提示里**。新增内核要么支持实时开关，要么在 `core/editor/types.ts` 的 `EditorMountProps` 注释里写明不支持 |
| **文档里的链接怎么激活（Ctrl / Cmd + 点击）** | 内核半边 `modules/markdown/link.ts` 的 `linkClickPlugin`（挂 `props.handleDOMEvents.click`，**`preventDefault()` 无条件** —— 不拦就是整个 webview 导航走）；地址分类是 core 里的纯函数 `core/link.ts` 的 `classifyLink`（五类 `anchor` / `file` / `external` / `empty` / `unsupported`）。**必须读文档模型里 link mark 的 `attrs.href`，不能读 DOM 上的 `href`** —— 渲染白名单会把不安全协议蒸成空串，DOM 会说谎。外壳半边 `ui/App.tsx` 的 `openLink`（ADR 0040，五种去向各说一句话、成功两支安静）。契约是 `EditorMountProps.onLinkClick?` 与 `EditorHandle.goToAnchor?(anchor): boolean`（**返回布尔，不是抛**）。与 `onImageClick` 同族：监听挂在内核自己造的元素上，外壳不按 `a[href]` 事件委托 |
| 代码编辑体验 | `modules/code/` |
| **`.mindmap` 的读写** | `core/mindmap/`（纯函数、**不 import 任何内核** —— 「结构自持，不绑定具体内核」是 §7.1 的设计意图）。字段对应表见 `docs/format-spec.md` §7.1.2 |
| **`.mindmap` ⇄ 内核数据** | `modules/mindmap/adapter.ts`（ADR 0007 后果 2 要的薄适配层）。契约是「**带旧信封进去、拿新信封出来**」：顶层 `extra` 只能从上一份信封继承（内核的 `getData()` 返回写死的字段列表），节点层才是现读的 |
| **导图编辑体验** | `modules/mindmap/MindmapEditor.tsx`。两条变更通道（`operation`/`expandNode` 主通道 + `linkDiv` 防抖兜底，后者是为 `expandNodeAll` 那种**改了数据不报事件**的操作准备的）；`getValue` 在用户没动过时交回**原文** |
| **导出成图片（PNG / SVG，导图与图纸同一条路）** | 内核侧 `MindmapEditor.tsx` 与 `DrawioEditor.tsx` 的 `exportImage`（契约在 `core/editor/types.ts`，可选方法，产物 `{ blob, warnings }`）+ 外壳编排 `core/export/export-image.ts`（与 export-document.ts **并列**，不是它的分支；外壳只认能力不认内核 id —— 图纸接入时**外壳一行没改**）。**三条纪律别碰**：导图的底色换真实解析色要整体重赋值 + `finally` 还原（`mind.theme` 与 `LEVNOTE_THEME` 同一引用）；图纸的序列化必须 `XMLSerializer`（图片形状用 `xlink:href`，`outerHTML` 不补命名空间声明，独立文件按 XML 解析直接报错）；PNG 在 jsdom 不可测。详见 §六本轮段与 ADR 0050 / 0051 |
| **大纲面板（Markdown 与导图两个来源）** | 提取器是纯函数两份：`core/outline/outline.ts`（Markdown，`level`/`line` 都是 1 基行号）与 `core/outline/mindmap-outline.ts`（导图，`line` 是**先序序号不是行号**、条目带 `id`）；跳转契约 `EditorHandle.goToOutlineItem({ level, text, id? })`（`id` 只有导图传 —— 文件层 id 指得准，Markdown 编不出真值就别编）。**导图侧四条纪律**：① `extractMindmapOutline` **必须**复用 `parseMindmap`（自造第二份「什么算一个主题」的判定迟早漂移，跳转会指错地方）；② 内核 `goToOutlineItem` 跳转时展开折叠祖先是**文档变更**（`collapsed` 是文件字段，onChange + 标脏照常，不标才是撒谎）；③ **根不展开**（mind-elixir 的 `expandNode` 只认带气泡的分支节点，对根调用抛 TypeError；根折叠也无视觉效果）与**只读不展开**（展开是编辑）是两个写死的例外；④ `findEle` 对看不见的节点 throw，**捕获交 false**，不白屏。外壳按 `isMindmapPath` 分派提取器与文案（产品范围，不是认内核）。详见 §六本轮段与 ADR 0052 |
| **导图标记（优先级 / 色标 / 符号）** | 词表在 `modules/mindmap/markers.ts`（`MINDMAP_MARKER_GROUPS` 三组共 19 字形：priority `1️⃣`–`9️⃣` 互斥 / color `🟥🟧🟨🟩🟦🟪` 互斥 / symbol `⭐✅❗❓` 不互斥），右键「标记…」开调色盘；选中走 `reshapeNode` → `operation` 事件 → 既有 emit 主通道（标脏 / 保存照常，不单独伺候）。**四条纪律**：① **字形即内容** —— `icons` 存显示字形本身（与 `topic` 存显示文本同理），**不发明语义 id + 双向映射表**；文件里词表外的字形照 §5 原样保留（调色盘只是词表知识，住内核模块，不进格式层语义）；② **全清过 = `[]` 一路到底** —— `JSON.stringify` 会丢 undefined 键，用 `{icons: undefined}` 清空会把「全清过」状态静默丢掉；序列化键序 `style` 之后、`extra` 之前，`[]` 也写回（canonical）；适配层 `icons` 在已知字段清单里双向透传；③ 标记是内核自有 UI（同右键菜单纪律），**不进 `EditorHandle`**；只读时 `contextMenu: false` 自然关门；④ `applyMarkerToggle` 纯函数语义：同字形再点一次 = 取消，互斥组先移同组旧字形再把新字形追加到尾，词表外字形按非互斥 —— 任何路径都不许因词表外形值炸掉。大纲条目把标记内联显示（`OutlineItem.markers`）。详见 §六本轮段与 ADR 0053 |
| **导图节点插图（粘贴进图 / 占位 / 移除）** | 词表与工具函数在 `modules/mindmap/node-images.ts`（收集引用 / 占位图 / 预热 / 剪贴板取图 / 尺寸解码与钳制 / `bytesToDataUrl`，全部纯函数或静态工具），UI 是 `MindmapEditor` 内部弹层（同右键菜单纪律，**不进 `EditorHandle`**）。**五条纪律**：① 格式层 `image` 照 `icons`（0053）先例升格为正字段，形状照内核 `{ url, width, height, fit? }` 不发明中间表示 —— `url` 存**文件 src**、尺寸即内容；整对象照收（不认识的子键原样跟着走），类型不对**整值转 extra**；键序 `icons` 后 `extra` 前，适配层双向直通；② **预热必须在内核创建之前**（`imageProxy` 是同步的、解析是异步的）：`data:` 透传、查得到换 data:、**查不到画占位图**（内嵌 SVG + 路径如实写在里面）——「图丢了是常见状态」，内核的「缺则不画」是静默，占位才是如实；http 外链也走占位（ADR 0001 零联网不破例）；③ 插入走内核 paste 的官方让位口 `pasteHandler`（内核先吃「复制的节点」magic JSON，吃不下才交原始事件）；剪贴板 `files` 与 `items` **两处都看**（平台填充不一致，markdown `imageFilesOf` 的既有教训）；字节交外壳 `onImagePaste`、外壳交回 null 时内核**静默**（外壳已 notice，再报就是两条消息）；④ 移除 = `{image: undefined}`（stringify 干净丢掉）；无图弹层给「按 Ctrl+V」提示 —— 空弹层是静默；⑤ 只读第一行 return —— 图片是编辑，不悄悄放行。详见 §六本轮段与 ADR 0054 |
| **导图节点链接与标签（右键弹层挂取 / 🔗 单击即走）** | 内核原生渲染这两样（`a.hyper-link` 锚点 + `div.tags` chip）但**零 UI**：菜单里的 `link` 项其实是节点连线 `createArrow`，全库锚点无任何点击监听 —— 编辑与激活都归 `MindmapEditor` 内部弹层/监听（同右键菜单纪律，**不进 `EditorHandle`**）。**五条纪律**：① 格式层 `hyperLink`（**非空字符串** —— 内核对假值不渲染，`""` 是画不出来的链接，整值转 extra；原文不 trim 不解码，分类归外壳）与 `tags`（**文字即内容**：`string \| { text, className?, style? }`；未知子键照收、坏形状**整条**转 extra 不逐项过滤、空数组照写 = 全清过）照 0053/0054 先例升格，键序 `image` 后 `extra` 前，适配层双向直通；② 编辑 = 右键「链接…」「标签…」弹层：无链接时**没有**移除按钮（不是置灰）、空输入禁保存、连挂不关弹层（连挂是正常用法）、重复标签 no-op 现值原样返回；`tagAdd`/`tagRemove` 从内核现取 `nodeObj.tags` 算新组（marker 先例，**不依赖弹层快照算**）；落盘统一 `reshapeNode` → `operation` → emit 主通道；③ 🔗 **单击即走**（与 markdown 的 Ctrl+点击 刻意不同：显式 chip 无歧义，XMind 同款）：host **捕获阶段**无条件 `preventDefault`+`stopPropagation`（内核不拦，WebView 不能整页飞走）；`href` 从 `nodeObj` 读**不从 DOM 读**（DOM 会说谎），`text` 读 `innerText`（内核拿它画的；jsdom 的 `textContent` 是空）—— 交外壳 `onLinkClick` → `classifyLink`/`openLink` 现成出口，文件路径可打开工作区另一份 `.mindmap`；④ 激活是读不是编辑 —— 只读也激活，但 `contextMenu: false` 挂取入口随菜单关门；⑤ 导出与大纲不伺候：`exportSvg` 自带 tags 文本与 `<a href>` 零改动（URL 本来就在文件里，是能力不是泄漏）；大纲只取 `topic`（0052 的刻意）。详见 §六本轮段与 ADR 0055 |
| **导图节点备注（📝 徽章 / 双入口 / 清空即移除）** | `note` 是格式层**早已存在的正字段**（键序第三个），但内核**零渲染零 UI**（探针实证 `me-tpc` 里一个字都不画）—— 本轮不升格格式层，动的是「看得见、改得了」，UI 是 `MindmapEditor` 内部徽章 + 弹层（同右键菜单纪律，**不进 `EditorHandle`**）。**五条纪律**：① 可见性 = 自挂徽章：`syncNoteBadges(mind.nodes, open)` 遍历 `me-tpc`，`nodeObj.note` 非空字符串挂 `button.mm-note-badge`（📝）、无则摘 —— **不 fork 内核渲染**（fork 会在内核升级时丢）；同步挂 **init 后一次 + `linkDiv` 监听**两处；② **`operation` 是编辑主通道、`linkDiv` 是重建兜底**：内核 undo/redo 走 `refresh(prev)` 且**不发 operation** —— 徽章被重建抹掉、onChange 不来，两条都由 `linkDiv` 兜底（徽章重同步 + 既有 `scheduleVerify` 300ms 迟到的脏标记 —— 迟到不是缺席，探针在 jsdom 里 Ctrl+Z 整链钉死）；③ 编辑 = 徽章 + 右键「备注…」**双入口**（堵上的设计洞：徽章唯一入口 ⇒ 没备注的节点永远创建不了第一条）；弹层 textarea **普通 Enter 是换行**、Ctrl+Enter/按钮保存、保存写 trim 后文本；**清空 = 移除 `{note: undefined}` 干净丢键** —— 与 0053 `icons` 的 `[]` 纪律**刻意不同**（空数组有视觉语义是「全清过」的记录，note 零渲染空串只是噪声）；落盘照走 `reshapeNode` → `operation` → emit 主通道（undo 历史自动记进去）；④ 大纲联动：`OutlineItem.note` + 面板 📝 指示 —— 指示是「有」，正文不进大纲（大纲是导航不是阅读器）；⑤ 导出不伺候：`exportSvg` 按选择器挑元素、未知类名不选走 ⇒ 徽章天然不进图（导出的是文件内容，徽章是编辑器部件，同右键菜单不进导出）；`EditorSearch` 从 ADR 0058 起搜备注（`TopicHit.field` 分诊，本行第⑤条的旧边界已退役）。只读徽章照显、弹层只读无按钮（读不是编辑，🔗 同款）详见 §六本轮段与 ADR 0056 |
| **导图查找也搜备注（命中带字段 / 写回分通道）** | `collectHits` 同一节点内 topic 先、note 后（两段偏移各以自己那段为基准，塞进同一张表；空串 note 不扫）。`TopicHit` 加 `field: 'topic' \| 'note'` —— 契约零改动（`EditorSearch` 是索引制的，命中表是内核私有的，外壳只见计数）。**五条纪律**：① `reveal` 到 note 命中亮弹层、到 topic 命中收弹层（命中落在弹层里，不亮等于没定位到；只读亮只读弹层）；② `replace` 按 `field` 分诊 —— note 走 `patchNote`（信封同步，`undefined` = 清空即移除沿 0056 纪律）+ `reshapeNode`（内核**同步**重画，0059 实测：`Xt` 通篇无 `await`），写回后必关弹层（草稿挂载播种，留着就是陈旧输入框），topic 路径一行不动；③ `replaceAll` 按「字段 + 节点」分段键控（`${field}:${id}`，混作一段会互踩偏移）、组内倒序（同段多命中的旧纪律），返回值按**命中数**报（一条 topic + 一条 note = 两处）；④ `NotePopover` 挂 `key={nodeId}` —— 键盘流切节点时草稿重新播种（鼠标流有外点自愈，键盘流没有）；⑤ 大纲与导出不伺候（照 0056 旧例：大纲只取 topic，徽章与弹层不进图）。详见 §六本轮段与 ADR 0058 |
| **导图节点样式（文字色 / 底色 / 字号 / 加粗）** | 词表与判定在 `modules/mindmap/node-style.ts`（`STYLE_TEXT_COLORS` 8 个偏墨低饱和色 / `STYLE_FILL_COLORS` 8 个淡彩底 / `STYLE_FONT_SIZES` `[14,16,20,26]` / `activeSwatchIndex` / `activeFontSize` / `isBold`），UI 是 `MindmapEditor` 内部 `StylePopover`（同右键菜单纪律，**不进 `EditorHandle`**）。**与 0053–0056 性质不同的一轮：格式层与适配层零改动** —— `style` 早就是正字段（`types.ts:152-153` 不透明直通、`KNOWN_NODE_KEYS` 里有它、`adapter.ts` 的 `NON_EXTRA_NODE_KEYS` 里有它、`Object.keys().length > 0` 才写出），内核也一直在渲染它（`Se` 把每个键原样塞 CSSOM），缺的只是「写」。**六条纪律**：① 词表**恰好是内核导出认得的那四键**（`LayoutSsr.js:45-47` 只拼 `color` / `background` / `font-size: ${v}px` / `font-weight`）——多一个键就是「编辑器里画得出、导出的图里没有」（ADR 0050）；`branchColor` 刻意不做（它住 `extra` 桶，可编辑 = 一次升格，该单独一轮）；② **`reshapeNode` 是同步的**（`Xt`，`MindElixir.js:507-515` 通篇无 `await`，`operation` 在函数体里同步 fire ⇒ 补丁交出去时文本已写出）⇒ 摘键必须**在补丁里就干净**，不能交空串等 300ms 的 `scheduleVerify` 兜底（会留下把 `"color": ""` 写进文件的窗口）；③ 补丁三种形状：设值 `{style:{[k]:v}}`、摘一键 `{style:{...rest,[k]:undefined}}`（`Object.assign` 是加/改语义，`JSON.stringify` 丢 `undefined`）、摘最后一键或「清除」走 `{style: undefined}` —— 逐键置 `undefined` 会留下 `"style": {}` 空壳（适配层的 `length > 0` 守卫拦不住，§7.1 说等价但**等价不等于该写**）；④ **`fontSize` 存数字、屏幕上补 `px`**（导出按 `${v}px` 拼要数字、CSSOM 忽略无单位长度要单位）——`syncStyleFontSizes(mind.nodes)` 挂 `linkDiv`（init 一次 + 每次重画，覆盖 undo/redo 的 `refresh` 不发 operation 那条路，0056 同款时机），DOM 擦除必须 `removeProperty(cssPropertyName(key))`（`removeProperty` 只认 CSS 属性名，camelCase **静默不删**，转换同 `LayoutSsr.js:62`）；⑤ 词表外的键照 §5 原样保留（单键编辑天然保留），只有用户显式「清除」才整份撤掉；亮不亮由纯函数判 —— **词表外的值（自定义色、别的工具的 `'20px'`）不亮也不假装摘得掉**，点色块不关弹层、点外面 / Escape 关，只读靠 `contextMenu: false` 自然关门（样式无常驻可见物，不需要只读弹层）；⑥ 已知边界如实记：深色主题下浅底对比度**不保证**（样式是文件内容，跟文件走不跟主题翻）、带单位字符串字号导出会拼成非法值（原样保留，不替别人改写）。详见 §六本轮段与 ADR 0059 |
| **导图分支连线颜色（样式弹层第五行「连线」）** | 词表在 `modules/mindmap/node-style.ts`（`BRANCH_COLORS` 与 `LEVNOTE_THEME.palette` **同一份** —— 摘掉 = 回到默认轮转，自洽；`activeBranchIndex` 纯函数判亮），UI 是样式弹层的第五行（同右键菜单纪律，**不进 `EditorHandle`**；色块视觉是「一段线」—— 底边框即色值，实心方块会和背景行分不清）。**五条纪律**：① 格式层照 `hyperLink`（0055）先例升格 `branchColor` 为正字段：**非空字符串**（内核对假值 `||` 回落轮转色，`""` 是画不出来的颜色，整值转 extra）、键序 `tags` 后 `extra` 前、适配层双向直通；② **渲染模型与 style 是两种**：连线是「每次全量重建的 SVG」（`linkDiv`，`MindElixir.js:1331-1367`；`reshapeNode` 内部就是 `Se → linkDiv → fire("operation")` 一条同步链），颜色烤在 path 的 `stroke` **属性**上 ⇒ 摘键后无需 DOM 收尾（0059 的 `removeProperty` 补丁这族没有）、无需 `linkDiv` 监听、jsdom 直接量得出；③ **子树继承是内核合同**（`dt` 从父色起步、自己有值才盖过）—— 改一个分支 = 整棵子树跟着变；**根不渲染该行**（`linkDiv` 只画 `me-main` 里子节点的线，根没有线；手写在根上的值照存不丢）；④ 落盘通道**独立成族**：`branchApplyRef` 与 `styleApplyRef` 同构不合流（`branchColor` 是节点顶层键不是 `style` 子键，两族数据一条通道是 0053 双写教训的温床）；返回**三态结果对象** `{ hex }` / `{}`（摘掉）/ `null`（没落进去）—— 裸 `undefined` 表示「摘掉」会被出口的 `?? null` 误折成失败，快照漏刷新（测试当场抓住过）；「清除」把 `style` 与 `branchColor` 并在**同一次** `reshapeNode` 里撤掉（一步 undo 一起回来）；⑤ 词表外色值照 §5 原样保留（不亮不假装摘得掉，「清除」是唯一出口）。详见 §六本轮段与 ADR 0060 |
| **markdown 内核文案跟随语言（三机制 / 重画不重建）** | markdown 模块此前**零 i18n 导入**（0047 登记的「6 处」实测是两文件 28 处：`code-block-view.ts` 24 + `image-view.ts` 4）。**三条纪律**：① 同一需求、三种回路形状 —— 机制跟着既有回路走，不另造：badge chrome 走 `chromeLocale` 域（locale 变 ⇒ `drawnLabel=null` 重画 title/aria）；图区纸条走 `painted.locale` 失效 → `ensureDiagram` **整链重渲染**（`paintDiagramNote` 签名不改、收已转好的 string —— 纸条是渲染产物，重跑渲染才是正路；重渲染**不重调**渲染器，断言锁 `spy.calls` 仍 0）；图片占位记 `failLine`（`ImageView` 没有重渲染回路，只能原地换字 —— **有回路重跑回路、没回路记键+参数**，`fail()` 改收 `LocalizedText`）；② **重画不重建**（与 0049 导图刻意不同：ProseMirror 状态、undo 栈、选区全丢不起）—— 照主题切换的既有通道 `subscribeLocale` + live 注册表（`$prose` 工厂闭包订阅、NodeView 构造 `add` / `destroy` `delete`、`view.destroy` 退订 —— 每编辑器实例一份，不泄漏）；③ **别人的话不翻**：mermaid 与 `renderDrawio` 的 `result.message`、`DocBlockPreview.reason` 原样转述（用户拿它搜上游 issue，翻译了反而害人）；zh 逐字照抄旧硬编码 ⇒ 存量断言一行不动（**兼容即契约**）。详见 §六本轮段与 ADR 0057 |
| **导图能不能用某个文件** | `modules/mindmap/adapter.ts` 的 `checkKernelSafety`（id 含引号 / id 重复都要拦）+ `modules/mindmap/index.ts` 的 `canHandle`。退路是**代码内核**（有 JSON 着色），不是纯文本 |
| **降级时说哪句话（ADR 0043）** | 契约 `core/editor/types.ts` 的 `canHandleReason?(path, text) → KernelRejectReason \| null` + `KernelRejectReason` 判别联合（**数据，不是翻译好的话** —— 字符串会把语言钉死在计算那一刻）。内核侧的唯一入口是 `modules/mindmap/adapter.ts` 的 `describeMindmapVeto(text)`，`canHandle` 与 `canHandleReason` 各自从它取一半 —— **不许分开实现**（会漂移到「说出来的理由」与「实际走的分支」不一致）。外壳侧是 `ui/DocumentPane.tsx` 的 `reasonOf` / `whyText`：`canHandleReason` 抛错按「没有理由」处理（`ui.pane.whyUnknown`），**绝不因此不开文件**。`resolveAdapter` 的行为不随这条契约改变（否决 → 下一个候选 → 兜底） |
| **块级互操作（提取 / 拉平 / 引用卡片 / 就地编辑 / 摘要）** | 内核半边：`core/editor/types.ts` 的 `blockTransform` + `modules/markdown/transform.ts`；外壳半边：`core/document/block-ops.ts`（依赖注入）+ `core/commands/definitions.ts` 两条命令；卡片：`modules/markdown/code-block-view.ts` 的 `renderRefCard` + `onDocBlockOpen` 契约。**就地编辑 = 同面板钻取 + 返回栈**（`core/workspace/store.ts` 的 `openInPlace` / `goBackDrill` + `TabBar` 的返回 chip，ADR 0036 —— 嵌套内核已否决，别再想往节点视图里塞第二个内核）；**摘要 = 外壳算、内核画**（`core/document/block-preview.ts` + `resolveDocBlockPreview` 契约）。引用路径解析**必须**走 `core/path.ts` 的 `resolveRelative` |
| **依赖许可证清单** | `pnpm run licenses`（`scripts/gen-license-report.mjs`）→ `docs/dependency-licenses.md`。**生成物不要手改**：升级依赖后重跑，严格条款自动复核（双许可可选宽松支不列）。npm 侧只统计生产依赖 |
| **发版前人工冒烟** | `docs/release-smoke.md`。单测测不了的都点名在这：滚动 / 渲染 / 打印（C1）/ 原生对话框与文件监听 / 键盘输入法。每条都写了「为什么只能人工测」—— 没有理由的条目不许加进去 |
| **面板常驻（标签切换不销毁内核）** | `ui/App.tsx` 的 `PaneSlot` + `visitedIds`（访问过才常驻，ADR 0034）。**句柄按面板分桶**（`reportHandle` / `reportViewInfo`）；隐藏面板收浮层的契约在 `DocumentPane` 的 `active` prop。`initialPosition` 逐面板取，不再有全局的 |
| 界面布局 | `ui/App.tsx` + `ui/styles.css`。**样式表按「属性」划所有权**（硬约束 13）：外壳只声明 `flex` / `min-height` 这类布局约束，`overflow` 一律归内核 |
| 活动栏（最左边那条图标带） | `ui/ActivityBar.tsx` + `ui/icons.tsx`（手写的十来个 path，**刻意不引图标库**）。五个入口都走命令，可用性来自 `CommandRegistry` 的 `enabled` |
| **菜单栏（Typora 式文字菜单）** | `ui/MenuBar.tsx`（ADR 0037）。九个菜单（文件/编辑/段落/格式/搜索/视图/编码/主题/帮助），下拉项就是**命令 id**：标题 / 快捷键 / 可用性全从 `CommandRegistry` 现查，菜单不带语义。键盘挂 **window 捕获阶段**（坑 9），收起状态不挂监听。勾选列走三态 `isChecked`（**只画状态不产生语义**，真相在设置快照与视图模式），勾选槽位**按菜单整列保留**（同菜单有一项可勾选，其余项也留空槽对齐）。新加菜单项 = 先有命令、再往 `MENUS` 塞 id；**没有真实能力的项不许进菜单** |
| **编码清单（一份）** | `core/encodings.ts` 的 `ENCODINGS` / `encodingSlug` —— 状态栏下拉与「编码」菜单共用；编码命令由清单生成（`encoding.set.<slug>`，title 就是选项文案），勾选读文档元数据 |
| **内核怎么被加载与注册** | `modules/index.ts`（**动态 import**，ADR 0028）。四个内核按 markdown → mindmap → drawio → code 逐个 await 后注册（顺序即优先级，与解析时序无关）；单个内核加载失败记进 `editorRegistrationFailures()`，外壳出横幅但启动照常。注册由 `main.tsx` 在**首帧之前** await —— 表还空着就渲染，`.md` 会被静默兜到纯文本。兜底内核（plaintext）保持静态：最后防线不能自己也是异步的 |
| **视图三态（富文本 / 源码 / 分屏）** | `core/document/store.ts` 的 `ViewMode`（**枚举，不是两个布尔**）+ `ui/ViewSwitcher.tsx`（状态栏里的三档）。分屏的两栏在 `ui/DocumentPane.tsx`：左 = `primary.sourceView`（可编辑），右 = `primary` 加 `readOnly` |
| **块手柄（悬停命中 / 上移下移 / 转为 / 拖动落点）** | `modules/markdown/block.ts`（纯逻辑 + 几何，**悬停走 DOM 元素、落点走坐标**，两条路的分工写在文件头）+ `modules/markdown/plugin.ts` 的 `blockHoverPlugin`（**不共用浮层骨架**：触发源是鼠标移动）+ `ui/BlockHandle.tsx`（显隐合三个条件） |
| **什么时候清掉「该操作哪一块」** | `ui/BlockHandle.tsx` 里那个 `visible === false` 的 effect。**不是「菜单关了」** —— 清早了第二次打开菜单会「点了没反应」，见 ADR 0024 |
| 某个面板 | `ui/DocumentPane.tsx` / `TabBar.tsx` / `Sidebar.tsx` / `FindBar.tsx` … |
| 设置页 | `ui/SettingsDialog.tsx`。**快捷键表与版本号都不许手写**：前者由 `CommandRegistry` 派生，后者经 `vite.config.ts` 的 `define` 从 `package.json` 注入（**必须写成 `import.meta.env.X`**，见硬约束 14） |
| 文件读写、编码探测 | `src-tauri/src/fs/` |
| **外部文件改动的发现**（监听 / 防抖 / 冲突提示） | `src-tauri/src/watch/`（Rust：notify + 防抖 + 整表替换，ADR 0029）+ `core/workspace/store.ts` 的 `handleFsChanges`（只做点名，mtime 才是判决） |

### 内核能力怎么加（不要破坏外壳的边界）

外壳**不认识任何具体内核**。要给内核加一种能力，走这条路：

1. 加在 `core/editor/types.ts` 的 `EditorAdapter` / `EditorHandle` 上（**契约**）；
2. 各内核按需实现（不实现的要能被外壳识别为「没有这个能力」）；
3. 外壳只问契约，不问内核。

已有八例可以照着看：

- `EditorHandle.search` / `goToOutlineItem` —— 可选能力，**缺了显示为禁用**；
- `EditorAdapter.sourceView` —— 源码视图，**不注册进注册表**
  （同一路径只能有一个内核答案，见 ADR 0014），缺了显示为禁用；
- `EditorHandle.formatting` + `EditorMountProps.onSelectionChange` —— 选中浮层工具条，
  **缺了就不显示浮层**（不是禁用）。与上面两例的差别是刻意的：
  源码视图是「有，但这份文档用不上」，格式化是「这个内核根本没有」——
  一个纯文本编辑器弹出加粗按钮，用户点下去只会什么都没发生。
- `EditorMountProps.onSlashChange` + `EditorFormatting.insertBlock` —— 斜杠菜单。
  这里有一个容易做错的分工：**触发判定（`describeSlash`）必须留在内核**，
  因为「光标在不在段落末尾」「有没有选中文字」是文档模型的问题；
  **菜单长什么样、有哪些项、选中哪一项、键盘怎么走全在外壳**，
  因为那是用户界面。`insertBlock` 与 `toggleBlock` 分开，是因为前者多一件事要做：
  先删掉 `/query` —— 要删哪几个字符是文档坐标，只有内核知道。
- `EditorMountProps.onTableChange` + `EditorHandle.table` —— 表格工具条。
  `describeTable` 把「光标在哪一格、这张表几行几列、能不能删」算成纯数据，
  外壳只负责画。**注意它的生命周期与浮层不同**：`TableState` 非 null 的含义是
  「光标在表格里」，会跟着光标来去。
- `EditorMountProps.onCodeBlockChange` + `EditorHandle.codeBlock` —— 代码块语言面板。
  它的状态非 null 的含义是「**面板开着**」（点击触发），不是「光标在代码块里」。
  另外注意这里**不是纯外壳画**：代码块的行号与语言标签是 NodeView 画进文档树的，
  外壳只画那个弹出的面板。
- `EditorMountProps.onDiagramOpen` —— Mermaid 图表的全屏查看。交出来的是
  **已经画好的那份 SVG 字符串**（不是路径：图没有文件，只有此刻屏幕上的这一份）。
  与 `onImageClick` 同属「监听必须挂在内核自己造的元素上」那一类。
- `EditorMountProps.resolveImageUrl` / `onImagePaste` / `onImageClick` —— 图片的显示、
  落盘、灯箱。前两个是**异步**的（读盘 / 写盘），第三个是同步上报。
  这里有一个容易做错的分工：**内核只做它唯一能做的那部分**（从剪贴板里把字节取出来），
  「存到哪、叫什么名、绝不覆盖」全在外壳 —— 内核连自己正在编辑哪个文件都不该知道。
  灯箱为什么由内核上报「点了哪张图」而不是外壳按类名做事件委托：见下面的坑 14。
- `EditorHandle.getPosition` / `setPosition` + `EditorMountProps.onPositionChange` ——
  位置记忆。这是**最不像浮层的一例**：前六例都是「内核报事件、外壳画界面」，
  这一例是「**内核读写状态、外壳只记住**」。
  两处容易做错的地方：① `offset` 的坐标系**由内核自己定**，外壳只做往返 ——
  想在外壳里把它换算成「第几个可见字符」等于重新实现一遍 ProseMirror 的
  `textBetween`，两份实现一定会在边界上分叉（见 ADR 0021）；
  ② 滚动容器归内核所有（`.qm-editor` / `.cm-scroller` / textarea 自身），
  所以滚动**只能**由内核读写 —— 外壳自己去 `querySelector` 就是按类名认 DOM。
  另外它**不是**「离开时问一次」：那一刻编辑器可能已经销毁，
  依赖卸载时序的东西会在某次重构里安静地坏掉。
- `EditorMountProps.onBlockHandleChange` + `EditorHandle.blockHandle` —— 块手柄。
  **它与前七例的触发源都不一样**：前面几个都挂在「选区 / 文档 / storedMarks 变了」上，
  而这一例挂在**鼠标移动**上（鼠标移动完全不碰 state，那套骨架一辈子不会被唤醒），
  所以它有自己的插件 `blockHoverPlugin`，不共用 `floatingReporterPlugin`。
  三条容易做错的地方（完整版见 ADR 0024）：
  ① 悬停命中**走 DOM 元素**（`event.target` → `posAtDOM`），不走 `posAtCoords` ——
  后者依赖 `document.elementFromPoint`，而 jsdom 没有它，走坐标等于把
  「上移 / 删除 / 转为」全部命令挡在测试之外；拖动算落点**反过来必须用坐标**，
  因为那时指针下面压着的是手柄自己；
  ② **目标块由内核记，但清它的时机在外壳** —— 判据是「手柄彻底不可交互」，
  **不是「菜单关了」**（菜单关掉时指针还在手柄上，清早了第二次打开菜单就「点了没反应」）；
  ③ **按住鼠标时不许换目标块**（判据 `event.buttons !== 0`）——
  拖动期间 `mousemove` 照常发，不挡的话用户拖第 1 块、松手却搬走了第 3 块。

**反例**（不要这样做）：在外壳里写 `if (adapter.id === 'markdown')` 或
`resolveAdapterById('code')`。那等于把内核知识漏进外壳，
换内核时外壳要跟着改 —— 那正是这套抽象要避免的事。

**另一条容易忘的**：内核里任何**绕过键盘事件**去改文档的能力，
都必须自己给事务打上「这是用户发起的编辑」标记（`USER_EDIT_META`），
否则文档不会变脏、用户按 `Ctrl+S` 写回磁盘的是改动**之前**的内容。

已经踩过四次：查找替换（`setup.ts` 的 `replace` / `replaceAll`，逐个事务打标记）、
选中浮层工具条与斜杠菜单（`asUserEdit`）、**表格的每一个操作**、
**代码块改语言**。后三者的统一入口是 `withUserEdit(editor, run)` ——
它临时替换 `view.dispatch`，给这期间派发的每个事务都盖上标记，`finally` 里还原。

图片插入（落盘回来时早已过了那个窗口，所以另有一个 `withUserEditOnView`）踩过第六次。

**Mermaid 预览是唯一一个「不需要打标记」的**：它只改**选区**（点图 → 光标进块），
不改文档 —— 而「文档没变就不该变脏」正是这个标记要保证的事。
所以判断依据不是「是不是用户动作」，而是**「这次事务改了文档没有」**。
新加一条这类能力时，先问自己这一句。

---

## 四、十六个已经踩过的坑（省下你的两小时）

### 1. Markdown 内核的「变更上报」不能用防抖

`@milkdown/plugin-listener` 的 `markdownUpdated` 是 **200ms 防抖**触发的。
我们在 dispatch 时设的「这次是程序写入」标志，等到 200ms 后回调才去读，早就清掉了。
后果是「从磁盘重新加载后，一个字都没改的文档立刻显示成脏」，用户被引着去保存，
一保存就把文件里他没碰过的排版改掉。

**上报规则是「只有用户交互之后的变更才算用户改动」**（见 `modules/markdown/plugin.ts`）。
键盘输入、输入法上屏、粘贴、剪切、拖放、撤销/重做会「上膛」；
初始化、`setMarkdown`、插件自己补段落的事务都不上报。

**另外**：由用户动作触发但**不经过键盘事件**的编辑（查找替换、将来的格式化）
必须显式打 `USER_EDIT_META`。少了它，替换结果不会上报，
用户按保存时写回磁盘的是替换前的旧内容 —— **这是静默丢数据**。

### 2. 打开路径上的两个「静默毁文件」问题

- **mdast 里「声明为字符串、实际是 null」的字段必须收敛成空字符串**
  （`modules/markdown/normalize-ast.ts`）。不加这一步，**没有标题的图片会被静默丢出文档**。
  注册必须走 `config`（早于 parser 创建），不能等编辑器建好再挂。
- **带 YAML front matter 的文件不能让富文本内核碰**。它会把 `---` 当分隔线、
  把 `键: 值` 当正文，保存时元数据就没了。走 `canHandle` 否决，
  注册表会继续找下一个候选（现在是代码内核接住）。

### 3. `EditorHandle.setValue` 必须避免回声

程序写入的内容变化**不能再回调 `onChange`**，否则「外壳设值 → 内核报变更 → 外壳设值」成环。
两个内核各自有机制：Markdown 用 `SILENT_META`，代码内核用 `SILENT` 注解。

同理，外壳那边用 `lastReportedRef` / `lastPushedRef` 两个基准去重（见 `DocumentPane.tsx`）。

### 4. 内核是异步创建的，句柄可能指向已销毁的实例

Markdown 内核是 `await Editor.make()...create()`。换文件、**切视图**时新的内核还没就绪，
而同步的 effect 已经跑了 —— 只凭「句柄存在」就往上写，会写到一个已经销毁的内核上。

所以 `DocumentPane` 里有 `handleKeyRef`：句柄必须带上它属于**哪一次内核实例**，
标识是 `会话号:视图模式`。

> **别把它简化回 `会话号`。** 切视图时会话号**不变**，只用会话号的话，
> 那一段异步窗口里旧句柄会被误认为「有效」。同一个标识也用在失败标记
> （`failedKey`）上，因此一次内核启动失败不会把用户永久锁在兜底内核上。

### 5. vitest 默认 `css: false`，会让「CSS 是否真的内联」这类断言永远通过

导出测试要验证「版式 CSS 被内联进单文件 HTML」，而那条链路走
`import styles from './document-styles.css?raw'`。不开 `css: true` 的话，
CSS 模块被替换成空实现，`?raw` 也拿不到内容 —— **测试假绿**。
已经在 `vite.config.ts` 的 `test` 里显式开了。

### 6. `cargo check` 通过但 `cargo test` 报 `LNK1181`

手删过 `target/debug/build/levnote-*/out` 会导致这个：那里除了 codegen 资产，
还有 `tauri-build` 生成的 `resource.lib`。`cargo check` 不链接所以看不出来。
**用 `cargo clean -p levnote`，不要手删 cargo 的输出目录。**

### 7. jsdom 缺三个 DOM 接口，报错位置与被测代码无关

`Range.prototype.getClientRects`、`Range.prototype.getBoundingClientRect`、
`HTMLElement.prototype.scrollIntoView` **在 jsdom 30 里都没有**
（`Element.prototype.getClientRects` 倒是有）。

不补的话，任何触发 ProseMirror `scrollIntoView()` 的操作 —— 查找跳转、
浮层工具条格式化 —— 都会抛 `target.getClientRects is not a function`，
栈顶停在 `prosemirror-view` 里，**看起来像编辑器坏了**。
补丁在 `src/test/jsdom-gaps.ts`（`vite.config.ts` 的 `test.setupFiles` 全局加载）。
它只补缺口、不覆盖 jsdom 已有的实现 —— 否则将来 jsdom 补上了，
我们反而用一个更假的版本盖掉它，而且没人会注意到。

### 8. 在测试里造 ProseMirror 选区，有两个坑

写「选中文字」的测试时走**真实 DOM 选区** + `document.dispatchEvent(new Event('selectionchange'))`
是对的（这正是浏览器里发生的事）。但有两处会让用例**假绿**：

1. **不能只 `selection.removeAllRanges()`**。ProseMirror 会看到「焦点不在自己身上」，
   直接跳过这次事件，连事务都不派发 —— 内核根本不知道选区没了。
   要模拟「用户点掉选中」，得在编辑器里放一个**塌缩的光标**。
2. **塌缩点不能落在文档最开头**。`DOMObserver.flush()` 里有一条：若「刚获得焦点 200ms 内」
   +「选区塌缩」+「塌缩到 `Selection.near(doc.resolve(0))`」，它认定这是**浏览器自作主张
   把选区重置到了文档开头**，于是把 state 里的选区写回 DOM、丢弃这次塌缩。
   这是真实存在的浏览器行为，PM 处理得对；但测试正好踩上就会看到
   「选了、取消了、浮层还在」，然后去怀疑一个根本没坏的实现。

（两条都记在 `App.test.tsx` 的 `clearSelection` 注释里，改那里之前先读它。）

### 9. 浮层菜单的键盘必须挂在 `window` 的**捕获阶段**

菜单打开时焦点**还在编辑器里**（用户正在打字），所以 `keydown` 的事件目标是
contenteditable。ProseMirror 的 `handleKeyDown` 挂在 `view.dom` 上，
在**冒泡路径上先到**：默认方式（`window` 冒泡）监听的话，等到我们收到回车时，
ProseMirror 已经把段落拆开了 —— 结果是「插入了块，但也多出一个空行」。

捕获阶段从 `window` 往下走，先于目标元素，所以能先一步
`preventDefault()` + `stopPropagation()` 把这一下拿走。
`stopPropagation` 还顺带拦住了外壳自己的全局快捷键（`App.tsx` 也在 `window` 上监听），
否则 `Esc` 会顺手关掉查找栏。

**配套的另一条**：菜单项按钮上要 `preventDefault` 掉 `mousedown`，
否则浏览器先把焦点挪到按钮上、ProseMirror 的选区随之丢掉，
菜单会在 `click` 到达之前被卸载 —— 表现成「按钮点不动」。
（链接输入框相反：它必须能拿到焦点。）

### 10. 测试里「打字」必须复现浏览器的链路，不能直接塞事务

jsdom 没有真实键盘。直接往 `view.state` 里塞事务能造出文档状态，但会**绕过**
整条输入链路（DOM 变更 → `MutationObserver` → `DOMObserver.flush()` →
`readDOMChange` → `input` 事件 → `markdownEmitPlugin`）——
而斜杠菜单、输入规则、变更上报全都挂在这条链上，绕过它测试就只验证了同义反复。

正确做法在 `src/test/typing.ts`（`typeInto`），**三个顺序不能乱**：

1. **先派发 `beforeinput`，再改 DOM。** 这是浏览器的顺序，也是内核判定
   「这次变更是用户干的」的**唯一可靠依据** —— 理由见坑 11。
2. **改完 DOM 要先让出一个宏任务，再派发 `input`。**
   `MutationObserver` 的回调是微任务，ProseMirror 靠它把变更记进队列；
   `input` 处理器只负责「立刻结算队列」。顺序反了队列是空的，
   这次输入被**完全忽略而且不报错** —— 表现为「测试里打字没反应」。
3. **之后还要等一帧。** 浮层上报是用 `requestAnimationFrame` 合并的
   （拖选时每个 mousemove 都触发一次，逐次去量会强制布局），
   `setTimeout` 走不到它。

同理，**任何断言「内核上报了什么」的用例都要先等一帧** ——
不等就会断言到上一个状态，那是假绿。

### 11. ProseMirror **不监听 `input`** —— 少一次 `beforeinput` 就静默丢字

这一条是 M5 做表格测试时发现的**真实数据丢失缺陷**，不是测试问题。

现象：**用户敲的第一个字存不进磁盘**。界面显示「已保存」，文件里却没有那个字。

根因：ProseMirror 靠 `MutationObserver` 读 DOM 变更并**自己派发事务**，
而 MutationObserver 的回调是**微任务**，会跑在任何事件派发之前。
于是 `markdownEmitPlugin` 的 `armedByUser` 在 `input` 到达时**还是 false**，
那次变更被当成「程序写入」安静丢掉。

它最坏的地方是**只错第一次**：从第二次输入起一切正常（标志已经上膛），
所以人工测试几乎不可能发现 —— 只有「打开文件 → 敲一个字 → 保存」
这条最短路径会暴露它。

修法：`plugin.ts` 的 `markdownEmitPlugin` 改为监听 **`beforeinput`**
（按规范在 DOM 修改**之前**派发，因此一定早于那次事务），`input` 保留作补充
（不是所有输入路径都发 `beforeinput`）。

> **这个缺陷在 M3/M4 时就已经存在**，只是没有任何一条测试踩到它。
> 结论：**「测试全绿」不等于「没有数据丢失」** —— 测试覆盖的是我们想到的路径。

### 12. Milkdown 的表格 schema 有三条读文档读不出来的约束

三条都是**实测**出来的，写错了不会报错，只会**静默丢内容**：

1. **最小表格是 2 行**（`table_header_row` + 至少一个 `table_row`）。
   空表头行在 schema 上**仍然合法**，但 `tableHeaderRowSchema.toMarkdown`
   遇到空内容直接 `return` —— 于是「只有表头的表格」存盘后**一行都不剩**，
   没有任何报错。所以删行的下限是 `map.height >= 3`，与光标在哪一行**无关**。
2. **对齐是按列、且由表头决定的**。`tableSchema.toMarkdown` 用**第一行**的单元格
   拼出对齐串，`keepTableAlignPlugin` 再把表头的 `alignment` 抄给同列的每一格。
   所以「给某一列设对齐」必须**写整列** —— 只写光标所在那格，
   下一次事务就被改回表头的值，用户会觉得「设了没生效」。
3. **`TableRole` 里没有 `header_row`**（只有 `table | row | cell | header_cell`）。
   表头行和正文行共用 `row`，想按「这是第几行」判断必须自己数。

完整取舍见 [ADR 0017](docs/decisions/0017-m5-tables-and-code-blocks.md)。

### 13. `setNodeAttribute` **不检查节点类型** —— 过期的位置会写到别的节点上

`tr.setNodeAttribute(pos, 'language', lang)` 只认位置，不认节点是什么。
位置一旦过期（面板开着的时候用户在别处打字、或者 `setMarkdown` 换掉了整篇文档），
它会**安静地把属性写到另一个不相干的节点上**，而且不报错。

所以 `CodeBlockChromeRequest` 里给的不是 `pos` 数字，而是
`resolvePos: () => number | null` —— **写的那一刻现问**，拿不到就放弃。
给函数就没有「记下的坐标过期」这个窗口。

> 这条是踩出来的：第一版存的是裸 `pos`，`setMarkdown` 之后设语言写到了
> 一个无关段落上，测试表现为「什么都没发生」，查起来像是面板没生效。
>
> **推广**：任何「面板 / 浮层持有文档坐标」的设计都要问一句 ——
> 这段时间里文档会不会变？会的话就存函数，不要存数字。

### 14. 外壳想按类名从 DOM 里认出内核画的东西 —— 那是耦合，不是捷径

图片的灯箱要「点一张图 → 开全屏预览」。最省事的写法是外壳在编辑器容器上挂
一个 `click`，命中 `img.qm-img-el` 就开。**不要这么做。**

那要求外壳认识**内核画的类名**，而依赖方向是
`modules/` → `core/editor/types.ts` ← `ui/`：**外壳不许 import 任何具体内核**。
把类名挪进 `core/` 只是把耦合换个地方藏起来 —— 那份类名本质上是内核的渲染细节，
它随内核改而改，而外壳会在毫无察觉的情况下跟着坏。

正确做法是**让内核上报**：契约加 `EditorMountProps.onImageClick(image)`，
监听挂在内核自己造的那个元素上。内核知道「点了哪个节点、它的 src 是什么、
解析出来的 URL 是什么」，外壳只需要画。

> **判断依据**：如果一件事需要外壳知道内核内部的 DOM 结构、类名、或者节点类型名，
> 那它就该是一条契约方法。已经按这条走过六次（选区、斜杠、表格、代码块、图片、图表），
> 每次都加在 `core/editor/types.ts` 上。
>
> **顺带一条**：`onImageClick` 给的是**已经解析好的 URL**，不是让外壳拿 `src`
> 再解析一次 —— 再解析意味着把同一个文件读第二遍，而那份字节此刻就在 DOM 上。
> 同理 `onDiagramOpen` 给的是**已经画好的那份 SVG 字符串**：图没有文件路径，
> 只有「此刻屏幕上的这一份」，重新渲染一遍既慢又可能画出不一样的图。

### 15. `nodeView.update()` 比 `PluginView.update()` **早一步** —— 在它里面判断选区会差一帧

Mermaid 图表块要在「光标进块」时显示源码、「光标离块」时显示图。
最自然的地方是在 `update()` 里判断 —— 而 `update()` 只在**节点本身**变了时被调用
（选区变化不是文档变化），所以本来就够不着。于是改用插件持一份活跃视图表，
在 `PluginView.update()` 里推给每个视图。

**坑在后面**：一个事务里 ProseMirror **先**跑 `nodeView.update()`、**后**跑
`PluginView.update()`。所以如果在 `update()` 里顺手做一次「要不要画图」的判断，
它看到的 `editing` 是**上一帧**的值 —— 表现是「在图表块里打第一个字时会多画一张
没人看的图」（第一次错，后面就对了，因为标志已经跟上）。

**修法**：图区的对齐**统一放在插件推过来的那个回调里**，`update()` 只管节点本身。
一句话：**在 `update()` 里读任何与选区有关的状态，都要先问一句「这个值是不是上一帧的」。**

### 16. `getBBox()` 在「没被渲染」的元素上给不出数字 —— 也决定了单测能证明什么

Mermaid 靠 `getBBox()` / `getComputedTextLength()` 量文字尺寸来决定节点框多大。
这两个 API 只在元素**真的参与布局**时才给得出数字：`display: none` 或者游离
（detached）节点会得到全 0（Firefox 上直接抛错），画出来是一片挤在一起、
互相重叠的框。所以给它量尺寸的容器必须挂在文档里、且用 `position: fixed`
挪到屏幕外，**不能用 `display: none` 藏**。

**同一个事实的另一半在测试里**：jsdom 没有布局引擎，`getBBox()` 一律返回 0，
**mermaid 在单测里画不出任何东西**。所以「渲染」这一层必须做成**可注入**的
（`MermaidIo` / `renderMermaid` 的注入点），单测钉住的是**我们这一层的契约**
（谁被换掉、失败留下什么、主题传对了没有、容器摘没摘），
而不是「图画得好不好看」—— 后者只能人工看。

> 判断依据：**只要一个库的渲染依赖真实的排版引擎，它就必须能被替换掉**，
> 否则那个功能在单测里等于没测。

---

## 五、工作流

```bash
pnpm run check        # = typecheck + lint + test。提交前必须全绿
pnpm run lint         # 单独跑 ESLint（check 与 CI 都已包含）
```

> **包管理器是 pnpm**（版本钉在 `package.json` 的 `packageManager` 字段），
> 锁文件只有 `pnpm-lock.yaml`。别用 npm/yarn 装 —— 会生成第二份锁文件，
> 而「到底哪份是真的」这个问题一旦出现就没法便宜地回答。

> **lint 是阻塞的。** 它曾经以「非阻塞」引入（首轮有 33 条历史问题），
> 欠账清零后已转为强制门禁。「有规则但不执行」比「没有规则」更糟 ——
> 前者会让人以为已经守住了。
>
> 配置里**刻意只开能抓真问题的规则**：不开风格规则，也不开
> `eslint-plugin-react-hooks` 的 `recommended`（它会把整套 React Compiler
> 规则一起打开）。要加规则，先想清楚它抓的是哪一类真 bug。

> **注意 `package.json` 里有两个 TypeScript 大版本**（`typescript` 指向 TS 6、
> `@typescript/native` 指向 TS 7）。这是为了让 `typescript-eslint` 能跑起来，
> 理由与重审条件写在 `eslint.config.js` 头部。**不要「顺手」把它改回 TS 7。**
>
> 代价要知道：**lint 跑在 TS 6 上，`tsc` 跑在 TS 7 上，两者可能对同一段代码
> 给出不同的类型**（已实测遇到过：TS 6 把闭包赋值后的变量收窄成 `never`，
> TS 7 正确得到 `number`）。所以「`tsc` 通过」不等于「lint 通过」。
> 撞上这种分歧时改写法，不要赌哪个版本更聪明。

改完代码之后：

1. **纯逻辑放 `core/`，配 `*.test.ts`。** core 里的每个模块都该有测试。
2. **有长期影响的取舍 → 新增一条 ADR**（`docs/decisions/NNNN-*.md`），
   并更新索引。**ADR 必须写「重审条件」** —— 没有重审条件的决策记录会变成后人不敢碰的祖训。
3. **注释写「为什么」。** 判断标准：这条注释是否记录了某次踩坑、
   某个反直觉的取舍、或某个「换个写法就会坏」的约束？
   代码里已有的注释密度是有意的，不要以「太啰嗦」为由删掉。
4. 用户可见的变化 → 更新 `CHANGELOG.md`；阶段性的 → 更新 `ROADMAP.md`。

---

## 六、现在的状态（2026-10-04）

**已完成**：P0–P3 的大部分 —— 文档生命周期、Markdown 内核、自包含导出、
多标签工作区、文件树、代码内核、命令面板、查找替换、大纲、主题。

**Markdown 完善（P3.5，对标 horseMD）**：字数统计、源码模式切换（Ctrl+/）、
**选中浮层工具条**、**斜杠菜单（`/` 插入块）**、
**表格可视化编辑 + 代码块语言选择器 / 行号**、
**图片粘贴落盘 + 相对路径 + 灯箱**、**LaTeX 数学公式**、**Mermaid 图表**、
**位置记忆 + 设置页**、**VSCode 式外壳 + 视图三态（含源码/预览分屏）**、
**块手柄（拖动 / 快捷菜单，ADR 0024）**已交付 —— P3.5 清单全部落地。
清单在 `ROADMAP.md` 的 P3.5。
**前六项都没有引入新依赖** —— `@milkdown/kit` 已经带齐大部分积木，
表格与代码块是自研的（官方那套是 Vue 3，见 ADR 0017），
图片的显示与灯箱也是自己画的（见 ADR 0018），
M10 的图标也是手写的十来个 path（`ui/icons.tsx`）。
后两项（KaTeX、Mermaid）破了这条：`katex` + `remark-math`、`mermaid`，
两者都走动态加载、不进主包（ADR 0019 / 0020）。

测试：**1,577 条前端 + 66 条 Rust**，`tsc` / `lint` / `vite build` / `cargo test` 全绿。
主包 382.99 kB（gzip 120.26 kB）—— 第 18 项主包拆分已落地（ADR 0028）：
三个内核与 hljs 拆成独立 chunk，首帧前并行加载；导出按需加载。
打包产物里另有 mermaid 的懒加载 chunk 3.4 MB（ADR 0020）与 draw.io 查看器
2.654 MiB（`public/drawio/`，按需注入、不进 bundle，ADR 0027）。

**P4 思维导图已全部完成（2026-10-04）**：`.mindmap` 的读写、适配层、内核接入、
**工作区全文检索**（第 23 项，见 ADR 0026）全部落地，四类一等公民里
**Markdown、思维导图、代码文本三类已可独立使用**。

**P5-25 内置只读 draw.io 查看器已完成（2026-10-04，ADR 0027）**：Markdown 里的
`kind: diagram` doc-block 渲染成真图，reference / inline 两种模式，四类一等公民里
draw.io 已可在 Markdown 里引用查看。P5 剩下的按需下载（26）与离线构建（27）
需要真实分发环境先行。

**P3-14 文件监听已完成（2026-10-04，ADR 0029）**：`watch/mod.rs` 空壳转正，
notify 8 + notify-debouncer-mini 0.7（主版本锁死）。事件是通知、mtime 才是
判决；整表替换 API（`watch_replace`），焦点全量比对保留作兜底。

**P3-16 打印 / 导出 PDF 已完成（2026-10-04，ADR 0030）**：与自包含 HTML
共用 `buildSelfContainedHtml` 渲染，出口是系统打印对话框（wry 未暴露
程序化 PrintToPdf）；HTML 经内存自定义协议 `levnote-print://` 供给打印
子窗口，窗口销毁即清。

**P3-17 i18n 骨架已完成（2026-10-04，ADR 0031）**：`core/i18n/` 目录 +
类型安全 `t()` + 回退链；首批文案 = 命令面（title/category）。
**新命令的文案写进 `core/i18n/messages.ts`**：文案取 key（`command.<命令 id>.title`
/ `command.category.<分组>`），`definitions.ts` 里**只声明 key、经
`builtInCommand({ titleKey, categoryKey })` 注册** —— 不要再写中文裸字符串。
**不许**绕过它直接往 `BUILT_IN_COMMANDS` 数组里塞
`{ title: t('command.…'), category: t('command.…') }` 这种裸对象。
**为什么这条要紧，以及编译期为什么拦不住**（本仓实测，2026-10-10）：
裸对象里的 `t()` 在**模块加载那一刻**求值（locale 还是默认 zh-CN）并烤进
`BUILT_IN_COMMANDS` 这个模块级常量，运行中切语言时条目纹丝不动；
而 `BuiltInCommandSpec` 只约束 `builtInCommand(...)` 那一个入口，
**管不着数组字面量** —— 照那写法塞进去，`tsc`(TS7) / `tsc6` / `eslint` /
`vite build` **全 rc 0 不报错**，会红的是两处测试：
`core/commands/definitions.test.ts` 钉命令对象本身；
`ui/App.test.tsx` 的「命令面文案跟随语言」一组钉界面三处（菜单下拉项 /
命令面板条目 / 快捷键表），**且两向都验**（切过去要变、切回来也要变）。
别被 `t()` 的类型安全误导：它拦得住**写错的 key**，
拦不住**写对 key 却烤死在数组字面量里** —— 类型系统不会替你记这件事。
**en 目录已落地（2026-10-07，ADR 0031 复评）**：`settings.locale` +
设置页语言下拉。**命令标题是惰性求值的（2026-10-09，复评记录三）**：
`definitions.ts` 里内置命令只声明 `titleKey` / `categoryKey`，命令对象上的
`title` / `category` 是**读的那一刻才查表**的 getter —— 切语言只需要重渲染，
**不要再写「重注册一遍」那种刷新**（`BUILT_IN_COMMANDS` 是模块级常量，
重注册拿到的是同一批对象，曾经因此让命令面形同被钉死在启动语言上）。
**E2 批次（同日，复评记录二）**：
外壳 ui/ 全部组件的界面文案已抽进目录（约 180 条，en 同量）。
**E3 批次（同日，复评记录二补记）**：core 层全部用户可见消息已迁完 ——
保存 notice、检索与替换汇总、AppError 文案、`.mindmap` 解析降级提示、
图表渲染失败原因、平台对话框标题、浏览器预览虚拟盘消息。
**组件里的静态清单存 key、渲染时才 `t()`**（ViewSwitcher / BlockHandle /
SlashMenu / SelectionToolbar / TableToolbar / CodeBlockPicker，以及
mermaid / drawio 渲染的失败消息）——
模块加载时求值会把语言钉死在启动那一刻；memo 里的条目同理，只存 key。
刻意**不迁**的：示例文档正文（文档数据）、命令 `keywords`（搜索数据）、
`console.warn`（开发者看的）。

**P6-31 自定义快捷键已完成（2026-10-04，ADR 0032）**：设置页每行可录制，
冲突显式顶掉（覆盖表三态：缺省 = 默认 / 写法 = 覆盖 / `''` = 清除）。
**给命令加默认快捷键仍写在 definitions.ts**；覆盖表是用户偏好，
`App` 的 effect 负责把 `settings.keybindings` 整表推给注册表。

**P6-29 往返保真度自检已完成（2026-10-05，ADR 0033）**：命令
`diagnostics.roundTrip`（只读不写），`EditorHandle.roundTrip` 可选契约 +
`core/fidelity.ts` 行级比对。**「解析 → 序列化」必须由内核实现**（复刻
trailing 规则），外壳另起 remark 管线量出来的差异是假的。

**P3-19 面板常驻已完成（2026-10-05，ADR 0034）**：先实测后改 ——
Markdown 内核每空闲实例 ≈ 0.27 MB、CodeMirror ≈ 1.9 MB（V8 堆，N=30 采样）。
**访问过的标签页面板常驻**（`PaneSlot` + `DocumentPane` 的 `active` prop），
切回即离开时的样子（撤销栈、光标、滚动、分屏状态）。两条新红线：
**句柄与视图信息按面板分桶**（`reportHandle` / `reportViewInfo`，否则关一个
后台标签会把活动面板的句柄一起清掉）；**隐藏面板必须收浮层、不收浮层上报**
（浮层键盘监听在 window 捕获阶段，不收起会截走活动标签的按键）。
内核永远不在隐藏状态下创建（挂载发生在首次激活那一刻，测量才不会全 0）。

**P7-34 依赖许可证清单已完成（2026-10-05）**：`npm run licenses` 生成
`docs/dependency-licenses.md`（302 个 npm 生产依赖 + 452 个 Rust crates）；
版本号三处统一。**项目已采用 MIT 许可证（2026-10-05，用户拍板）**：根目录
`LICENSE` + `package.json` / `packages/interop-spec/package.json` / `Cargo.toml`
的 license 元数据，三处版本号随 0.3.25 统一。

**P6-28 块级互操作第一阶段已完成（2026-10-05，ADR 0035）**：**提取为引用 /
拉平引用**（L3）—— 内核 `EditorHandle.blockTransform` 契约（只有 Markdown 内核实现，
读写都走 Markdown 源，与 roundTrip 同一条理由），外壳编排文件操作
（`core/document/block-ops.ts`，依赖注入可单测）；**引用卡片**（L1）——
markdown/mindmap/text 的 reference 引用画成卡片、点击打开（`onDocBlockOpen` 契约）。
`withUserEdit` 拆进了 `modules/markdown/user-edit.ts`，**不要再各写一份**。
新坑：`joinPath(dirname(docPath), src)` 在目录是裸名时**丢分隔符风格**
（`notes\main.md` → `notes` → 拼成 `/`）—— 解析引用用 `core/path.ts` 的
`resolveRelative`，不要直接拼。

**P6-28 块级互操作第二阶段已完成（2026-10-05，ADR 0036）**：**L2 就地编辑** =
**同面板钻取 + 返回栈** —— 卡片上的「就地编辑」把被引用文档按**正常文档路径**
打开（自己的 store / 脏标记 / Ctrl+S / 文件监听），来路记在目标标签页的
`drilledFrom` 上，标签栏出现「↩ 返回」chip（`workspace.openInPlace` /
`goBackDrill`；普通 open 命中已有标签页会清来路，closeTab 顺手清掉指着自己的）。
**嵌套内核否决**（六条理由在 ADR，核心是「同一份文件只有一个编辑面」的
不变量），跨文档分屏立项时才重审。**引用摘要**：外壳算（`block-preview.ts`，
`resolveDocBlockPreview` 契约）内核画 —— markdown 给标题树 + 字数、text 给
行数 + 片段、mindmap 给根主题 + 节点数，读不到把原因当数据画出来；
**markdown 刻意不渲染成 HTML**（半渲染器 ≠ 编辑器，比不显示更糟）。
卡片的动作按钮**只在对应能力挂着时才画**（setup.ts 里 handler 必须按
undefined 原样传，包一层永远存在的函数会画出假按钮）。

**菜单栏已完成（2026-10-06，ADR 0037）**：Typora 式文字菜单九个
（文件 / 编辑 / 段落 / 格式 / 搜索 / 视图 / 编码 / 主题 / 帮助），自绘 web 菜单 +
命令注册表当唯一真相。内容充实阶段新增 15 条命令（新建 / 保存全部 /
撤销 / 重做 / 复制为 HTML / 四至六级标题 / 换行 / 行号 / 全屏 / 主题三档
等）与可选契约 `EditorHandle.undo/redo`（两个内核实现，菜单触发的撤销
打 `USER_EDIT_META`，不实现撤销的内核菜单灰着）。**撤销 / 重做刻意
无默认快捷键**（内核 keymap 已占 Mod+Z，全局再注册会双份执行）。
顺带修了代码内核 `setValue` 漏打 `addToHistory.of(false)` 的既有缺陷
（打开文件后第一下 Ctrl+Z 清空整篇）。判断依据多一条：
**菜单里没有真实能力支撑的项不进菜单** —— 参考外部产品的菜单清单时，
没实现的宁可缺，不画点了没反应的占位。

**菜单栏第三轮（同日，ADR 0037 §7）**：**代码内核补上 Markdown 源码级
格式化**（`modules/code/markdown-format.ts`，`EditorFormatting` 的第二个
实现，仅 Markdown 语言）—— 源码 / 分屏视图下「段落 / 格式」不再整体置灰，
菜单不认视图只认能力；CodeMirror 上报走 `updateListener`，**不需要**
`USER_EDIT_META` 上膛。**编码菜单**是状态栏编码下拉的提升（选项清单在
`core/encodings.ts`，命令由清单生成）；**搜索菜单**全是已有命令的第三入口。
**顶部工具栏已整排撤除**（品牌 / 路径 / 徽章 / 六个按钮并入菜单与
状态栏三档切换，`--titlebar-bg` 已改名 `--menubar-bg`）；
**状态栏是中性配色**（VSCode Modern 风格：底色与 chrome 同源 + 上边线，
彩色只留给 `--accent` 选中态与语义徽章 —— 别再把「主题色铺满」加回来）。

**专注 / 打字机已完成（2026-10-09，ADR 0039）**：两个视图偏好（`settings.focusMode` /
`typewriterMode`），设置页两个开关 + 视图菜单两个命令（`view.toggleFocusMode` /
`view.toggleTypewriterMode`，**刻意不给默认快捷键**）。Markdown 侧一个
`composePlugin` 管两件事，代码内核只认打字机。**两条纪律**：切换偏好
**不重建内核**（丢光标 / 滚动 / 撤销栈），只派发一条带 meta 且
`addToHistory: false` 的事务；**打字机绝不 dispatch**，只写 `scrollTop`
（dispatch → 位置记忆上报 → 又滚动，停不下来）。测试只钉插件契约，
滚动那半边 jsdom 里是空操作 —— 已点名进 `docs/release-smoke.md` 第 3a 步。

**链接激活已完成（2026-10-10，ADR 0040）**：Markdown 富文本视图里的 `<a>`
终于可以 Ctrl / Cmd + 点击。此前它不是「点了没反应」，而是**默认行为会让整个
webview 导航走** —— 未保存的改动一起消失，连一句「要不要保存」都不会问；
所以内核第一步是**无条件 `preventDefault()`**，第二步才是把地址分成五类各给
一句话（`core/link.ts` 的 `classifyLink` 是纯函数，30 条单测）。**成功的两支
刻意一句话都不说**，五种失败各有自己的成因与文案 —— 详见 ADR 0040。
两条纪律值得记住：**地址必须读文档模型里 link mark 的 `attrs.href`，不能读
DOM 上的 `href`** —— 渲染白名单会把不安全协议蒸成空串，DOM 会说谎（`App.test.tsx`
的那条用例先断言 DOM 的 `href` 为空，再断言提示里出现协议名）；**锚点跳转
不能断言行号** —— ProseMirror 的位置里没有「空行」，它算出来的行号与 Markdown
源码行号不是同一刻度，钉行号等于把实现内部细节写成契约（改断言「DOM 选区落在
哪个标题里」之后才转绿）。`findAnchorPos('')` 命中 id 默认为空串的节点那个缺陷
是**测试跑出来的**，不是读代码读出来的。

**失败要有出口已完成（2026-10-11，ADR 0041）**：这一轮不是新功能，而是照着上面
那条结论回头清点 —— 既然「清单全部完成」证明不了没有静默，收工时该做的是**再按
硬约束 2 扫一遍**，而不是宣布这一类结束了。扫出来两类都属于「用户的动作没有任何
反馈」：① 命令失败了只 `console.warn`（四个入口**各抄一遍**，所以修一处漏三处 ——
这次的起因正是如此），现在收成 `core/commands/runner.ts` 的 `runCommandSafely`，
弹一句「「⟨命令名⟩」执行时出错：⟨原因⟩」；② 一次拖多个文件只开第一个，现在逐个
打开、**头一个坏不连累后面**、`core/workspace/drop.ts` 只在有失败时点名。
两条纪律：
**一个会失败的异步能力，它的类型不许是 `void`** —— 类型上抹掉 Promise 等于在编译期
宣布「这次失败无处可去」（`toggleFullscreen` 因此改成 `Promise<void>`，浏览器没手势
必 reject，此前被 `void` 掉就是「按 F11 没反应」）；
**它是最后一层，不是唯一一层** —— 想让某条命令不落到这里，在命令内部接住并说明，
**不许在兜底层加白名单**。
形式上最要紧的一条：`runCommandSafely` 不能写成
`void Promise.resolve(command.run(ctx)).catch(...)` —— `run` 在**参数求值时**就跑，
同步 throw 到不了 `.catch`；正确形状是 `try { result = command.run(ctx) } catch { … }`，
**这条是单测逼出来的**（`runner.test.ts` 里那条 `Error: 同步炸了`）。
取舍上的两条：**成功时安静**（新标签页自己看得见，一边干活一边表功是另一种吵），
**失败才点名**（用文件名不用绝对路径 —— 用户拖的是文件，他认的是文件名），
与 ADR 0040 同一取舍。
刻意排除的三类在 ADR 0041 的背景里写了理由：后台监听失败（会自动重试）、无效
快捷键写法（是用户数据写错，该由设置页的冲突提示去说）、消息框自身失败（已无处可去）。

**偏好写不进去也要有出口（2026-10-11，ADR 0042）**：0041 清的是**命令**这条线，
这一轮清的是**持久化** —— 全仓库只有四个模块碰存储（`settings` / `positions` /
`recent` / `session`），会话那份 0023 已经做对（`SaveOutcome`），剩下三份**全是静默的**，
静默的理由是同一句注释「配额用尽 / 被禁用：静默放弃。偏好没存上不是需要打断用户的事」。
**这句话对「打断」是对的，对「不说话」是错的**：界面立刻变成改好的样子，真相是
**只在内存里改好了**，下次启动回到默认值而用户无从把两件事联系起来。
现在写入一律返回 `PersistOutcome`（`{ ok: true }` | `{ ok: false, reason: 'no-storage' |
'write-failed' }`，定义在 `core/settings.ts`），**两个失败值刻意分开**：
`no-storage`（浏览器预览 / 隐私模式）是**预期内的降级、刻意不说话** ——
为它弹一句等于每改一次设置弹一次，那样的横幅三次之后就没人看了；
只有 `write-failed`（配额 / 策略）才翻横幅。
四处写偏好（设置 / 位置 / 位置清空 / 最近打开）全部汇进 `ui/App.tsx` 的
`reportPersistFailure(outcome)`，**翻会话横幅那一枚 `writeFailed`，不另开一条** ——
它们说的是同一件事，分成四条提示等于让用户自己拼出「哦，是存储坏了」；
翻标记**必须幂等**（位置记忆每 600ms 写一次，配额满了次次失败，不幂等就是
每 600ms 重渲染一次整个外壳）。
顺带把副作用从 `setSettings` 的 updater 里搬了出来（StrictMode 会跑两遍 updater、
且吞掉返回值）；`settingsRef` 存当前一份**不进依赖表**，否则 `update` 的身份每次改
设置都变、把整张命令表一起重建。
`AppProps` 新增 `storage?: StorageLike | null`（**只有测试用**，`main.tsx` 未动）——
jsdom 的 localStorage 永远写得进去，没有注入点这条出口就一条用例都碰不到。

**降级要说出理由（2026-10-11，ADR 0043）**：0041 清命令、0042 清持久化，
这一轮清的是**降级** —— `EditorAdapter.canHandle` 的返回类型是 `boolean`，
于是「内核为什么不要这份文件」在注册表边界上**被丢掉了**。最要紧的一处证据是
`modules/mindmap/adapter.ts` 的 `checkKernelSafety` **本来就带着理由**
（一句写好的中文），而 `canRenderInKernel` 只取 `.ok` —— 那句话从来没活着出去过。
于是降级横幅只能写死一种原因（`ui.pane.frontMatterWhy1/2/3`），
打开一份 **id 重复**的 `.mindmap` 时，用户读到的是「这个文件的开头是 YAML front
matter……」：一句与眼前文件**毫无关系**的话。**这比沉默更糟** —— 横幅明明在说，
说的是别人的理由，而用户会拿它去改文件。
现在契约多一条 `canHandleReason?(path, text) → KernelRejectReason | null`，
`KernelRejectReason` 是**判别联合而不是翻译好的字符串**（`{ kind: 'front-matter' }` /
`{ kind: 'mindmap-invalid' }` / `{ kind: 'mindmap-unsafe-id', id }` /
`{ kind: 'mindmap-duplicate-id', id }`）—— 字符串会把语言钉死在内核算出理由的
那一刻（与「静态清单只存 key、渲染时才 `t()`」同一个坑），而带 `id` 的那两种
要能在横幅上**点名那一个节点**，用户才知道去改哪里。
两条纪律：**判定与理由必须同源**（mindmap 侧两个方法都读同一个
`describeMindmapVeto(text)`；分开实现会漂移到「一个说不行、另一个说不出为什么」，
`modules/wiring.test.ts` 逐份文本比对齐它们）；
**`resolveAdapter` 的行为一个字节不改**（否决 → 下一个候选 → 兜底；
`canHandleReason` 抛错按「没有理由」处理，绝不因此不开文件）。
front matter 那一种**保留原来三行** —— 它讲的是「一保存元数据就没了」，值得说透；
其余走一句通用句（`ui.pane.downgradeWhy` + `why*`），内核不说话时退回
`whyUnknown` 而不是让横幅消失（降级是确定的事实）。

**读侧也要有出口（2026-10-12，ADR 0044）**：0042 只清了写侧，并**明确排除了**
读侧的沉默，理由是「读有默认值可依，写没有」。这条理由本身没错，
**结论却下反了**：默认值兜的是「这次没有」，而「上次存的那份这次读不出来」丢的是
**上一次的全部状态** —— 四处存储里最贵的是会话（`levnote.session.v1` 一坏，
**没保存的草稿跟着没**，而 `droppedDrafts` 连这件事都数不出来，它只数逐条丢弃）。
所以 `loadSettings` / `loadPositions` / `loadRecentFiles` / `loadSession`
**四个一律返回 `{ value, outcome }`**（`LoadResult<T>`，`core/settings.ts` 里
与 `PersistOutcome` 并排定义的那个 `LoadOutcome`）：**值照样回落**
（回落是止血，白屏比丢失更糟），但回落**不等于沉默**。
判断的**只有一条**：**键在、值用不了**，才算丢失。由此推出三处刻意的**不报** ——
**键不在**是版本升级新加的字段（老用户的存储里根本没有，报一次等于对着空气说话）；
**夹取**（`sidebarWidth: 9999` → 夹回上限）是意图已兑现，报「没读回来」是撒谎；
**字体族消毒**同理。实现就落在这条判据上：`core/settings.ts` 的 `pickField`
里 `raw === undefined` 走安静那条，`parse` 返回 `null` 才记进 `lostKeys`。
两条纪律：**整份报废时 `lostCount` 一律写死 1**（写 0 的话用户读到的是
「什么都没丢」，那等于把一次丢失又藏了回去）；**点名还是只报数，由键名决定** ——
路径能点名（`READ_LOSS_NAMES_MAX = 3`，全列出来就是一面墙），
`sidebarWidth` 这种内部标识只报数。`loadSession` **只报整份报废**，
逐条丢弃继续走 `droppedDrafts`，同一件事不说两遍。
读侧横幅与写侧横幅**分两条**：处置完全不一样（一个「读的是别人写的那份」、
一个「我写的那份没存上」），合并会让用户猜是哪一边。
测试里补了 `mount` 的 `readSettingsFromStorage` 开关 —— 默认注入是有道理的，
但读侧那一族要验的恰恰是「**从存储读出来的偏好**」，注入了就永远读不到。

**监听没起来也要说出来（2026-10-12，ADR 0045）**：这一族扫完之后，`core/` 里
已经**没有一处静默降级**了，于是按上面那条结论再扫一次，剩下三处全在同一个地方
—— `ui/App.tsx` 的文件监听，而且是同一种错：**发布版里 `console.warn` 等于
把话写进一个没人会打开的盒子**（后端特意在批次上留了 `error: Option<String>`、
特意把失败路径作为返回值交回来，前端把两份交付都扔进了控制台，
连两个 promise 的拒绝都 `void` 掉没接 catch）。**这一族的判据与前几条不同**：
丢的**不是数据，是「早点知道的机会」** —— 监听一停，目录树不刷新、冲突检测不报警，
用户可以在毫无察觉的情况下把别人的改动整份覆盖掉。
**而且它不会自愈**：目录被删、inotify / kqueue 监视额度耗尽，事件就**永久**
不再到达，兜底那两条路（下一批事件、窗口聚焦时的全量比对）全都建立在
「事件还会来」这个前提上 —— 0041 那句「失败了当场就失败了，只说一句原因」
在这里是错的，因为它不会当场失败。
所以有了一条 `WatchIssue{ reason, failedPaths }` 横幅：两路来源**各占一个槽、
整份替换**（不累加 ⇒ 严格模式双挂不翻倍；能整份替换 ⇒ 下一批正常事件会把
`reason` 清回去，横幅说的是**此刻**的状态而不是一份越积越长的故障日志）。
`reason` 是**三态**（`null` 没问题 / `''` 有问题但拿不出原因 / 原文）——
压成两态会让「说不出原因」整条消失，而**报 0 等于把一次故障又藏了回去**；
`foldWatchIssue` 因此判的是 `undefined`（不碰）而不是 `??`（`null` 会被落回旧值）。
措辞**分两档**：整条链可能断才说「外部改动可能发现不了」，只是几条路径没监视上
只说「有 n 条路径没在监视」 —— 多半那几条只是被删了/移了，
重话说就是虚惊，而虚惊多了用户就会开始忽略这一类横幅。
按掉的记成**指纹**而不是布尔：关掉 A 不该把之后出现的 B 一起关掉。
**Rust 侧零改动** —— 那两个字段本来就是为此留的，只是前端一直没接；
**这一条要记住的纪律**：别再把监听的任何失败写成 `console.warn`，
也别 `void` 掉一个可能失败的 promise 却不接 catch。

**说过的话要留得住（2026-10-12，ADR 0046）**：0042–0045 这一族让降级**说**了话，
但横幅有个结构性前提 —— **用户正好在看**。而按掉横幅的动机恰恰是
「我不想现在处理」，于是那一次降级在用户的视野里**第二次**消失，比第一次更彻底；
一次运行里先后三四件事，横幅一次只占一行；关掉应用那一条也没了，
而「刚才那一下到底提示了什么」正是用户回来描述问题要说的第一句话。
ADR 0001 那边（没有账号、没有网络、没有遥测）让这件事更硬 ——
**除了一页列表，第二个把内部状态交给别人的办法根本不存在**。
所以有了 `core/health.ts` + 帮助菜单的「本次运行的问题」
（命令 `diagnostics.issues`，**刻意不给快捷键**）。五条要记住的：
① **记的是 `{ key, params }`，不是句子**（记账那一刻调 `t()` 会把语言烤死，
与 ADR 0031 的 `BUILT_IN_COMMANDS` 同一类错）；② **横幅族走 `syncFamily`**、
**一次性事件走 `recordIssue`**（内核加载 / 写失败 / 恢复丢掉的草稿与文件
没有「恢复」这个状态）；③ **内容一字未改且未恢复就什么都不做** ——
上报点在渲染里，严格模式双挂会把「发生过 2 次」写到只发生过一次的事上；
④ **写失败要加 ref 闩**（位置记忆每 600ms 落一次盘；闩只能挂在**上报点**上，
不能从横幅的 `writeFailed` 推 —— 那个会被「知道了」清掉）；
⑤ **恢复的记录永不删除**，只打标记（自动收起来是这一族里最难看的一种静默）。
**刻意不落盘**：跨运行日志会把「上次的记录」混进「本次运行」。
「**复制全部**」是纯本地应用唯一一条把状态交给别人的路，所以那份文本必须
脱离界面也读得懂，剪贴板失败要有一张脸。

**说过的话要跟着语言变（2026-10-11，ADR 0047）**：0046 刚把「记的是 key 不是句子」
这条立起来，紧接着就发现**同一类错还剩四处**—— 文件树顶横幅（`failTreeAction`
的十处调用 + `validateEntryName()` 的八条规则，两处生产者都扫了才没漏）、
跨文件替换的汇报（`describeReplaceOutcome` 交出一句拼好的话，
`SearchPanel` 存进 state）、设置页的 `error`、图片落盘失败的 `reason`。
**这一族回答的是「用户中途切了语言」**：数据还在，可说的话停在旧语言上，
界面于是一半中文一半英文。**判据不落在文件归属上，落在「`t()` 的结果有没有
活过渲染」上** —— 进了 state / store 字段 / 会被存起来的返回值就是这一族；
白名单是**允许动的边界**，不是「必须改的清单」（`DocumentPane.editorError`
在白名单里，但它存的是 `thrown.message`，不是 `t()` 的返回值）。
四条要记住的：① **跨边界的不是句子，是 `{ key, params }`**
（`LocalizedText`，字段名跟 `t(key, params)` 对齐 —— 同一个想法两种拼法
就是代码地图禁的那类「抄两份」）；② **抛物的原话当 `params` 走，绝不当句子翻**
（翻一句 `EACCES` 得到的是听起来像回事的谎）；③ **句子只在渲染那一刻拼**，
`renderReplaceReport()` 里连那个逗号都是一个 key；④ `images.ts` 一交 key，
`DocumentStore` 的 `legacyText()` 品牌门就整个删掉 —— **门后的门比没有门更糟**。
**登记不做**（写进重审条件，别让人重新扫一遍）：`block-preview` 的
`reason` 与 `snippet`、katex / mermaid / drawio 的 `Render.message` ——
它们的落点在内核画的那一侧，修它要动 `core/editor/types.ts` 的契约。

**另外两类文件也是一等公民（2026-10-11，ADR 0048）**：「Markdown 支持得很好」
不等于「导图与图纸支持得很好」—— 本轮之前 `.mindmap` 内核早就在那儿了，
**只是入口为零**（菜单里只有 `file.new`，落一个空文本文件；侧栏新建走 Rust 的
`create_file`，注释明写「创建**空**文件，这里不替调用方预填任何内容」）。
空 `.mindmap` 能打开（`describeMindmapVeto('')` 返回 `null` —— 空文件不是「坏文件」
是「还没开始写」），但用户**没有任何办法主动得到一份**。`.drawio` 更彻底：
连内核都没有，只能在 Markdown 里被引用。四条要记住的：
① **起始内容只有一份出口**：`core/new-file-content.ts` 按 `is*Path` 判，
**不许 import 任何 `modules/`**；模板**生成**而不是抄字面量（抄的那份会在格式演进时先掉队），
用**各自的判据**验 —— `.mindmap` 看 `parseMindmap(...).canonical`（打开→保存无 diff），
`.drawio` 看 `missingStencilLibs` 恒为空。三个新建入口共用
`writeNewFileThenOpen`，差异只留数据。
② **编辑入口的形状就是 `sourceView: codeAdapter`**，内核契约一个字没改 ——
`.drawio` 的文档内容就是那段 XML，`onChange` 一接上就是可编辑。
**只读内核从不调用 `onChange`**：不能改文档就没有「改了什么」可报，
报了会把没变的文件标脏并重写一遍。
③ **能力缺失写在面板上**，并给出可走的下一步（图纸 → Ctrl+2 源码视图），
不靠「没那个按钮」让用户自己发现。**随包的 `viewer.min.js` 跑不起编辑器是有读数的**：
产物里 `Editor` 是 `function(){return new mxCellEditor(this)}` 替身，
`new EditorUi(document.body)` 直接抛 `Cannot set properties of undefined (setting 'lightbox')`。
**别把它当编辑器用**，也别在 README 里承诺做不到的事。
④ **注册顺序变成 markdown → mindmap → drawio → code**（硬约束 4）。
**登记不做**：`modules/markdown/code-block-view.ts` 里 6 处硬编码中文文案
属 ADR 0047 的漏网；侧栏新建不落模板（Rust 那条不改，命令面板那条落模板）。

**内核自己画的字也得跟着语言变（2026-10-11，ADR 0049）**：ADR 0047 管的是
**我们自己的**文案，「切语言后那一句还是旧的」还有另一半 —— **内核自己画的那部分**。
思维导图的右键菜单十一条与新节点占位文字全是英文，来源只有两处，都在
`mind-elixir` 的构造参数里（`contextMenu.locale` / `newTopicName`），都有配置的口子。
三条要记住的：
① **菜单是内核在安装那一刻烤进 DOM 的**（`MindElixir.js:1394` 的 `innerHTML`）——
换语言包对象不会更新已经建好的菜单，**唯一的办法是重建内核**。所以
`contextMenu` 传 `true` 只能拿到它硬编码的英文包，**要传对象**才带得上我们的语言包。
② **重建不能丢内容**：种子取 `latestRef`（「此刻 Ctrl+S 会写出的那份文本」），
拿 `initialValue` 重新解析等于把用户的修改**静默回滚**；`getValue()` 里记进
`latestRef` 的必须是**交出去的那个值**而不是 `lastText` —— 记错的话下一次保存
把用户文件**重排版**，「打开 → 保存逐字节一致」当场破掉。cleanup 里还要结掉
兜底通道那个 300ms 定时器，否则拖完立刻切语言会把那一次改动**丢掉**。
③ **重建会交出一个新句柄**：外壳的 `handleEditorReady` → `reportHandle` → `setHandle(next)`
这条链本来就在跑，所以查找栏会自己重跑，**不用你再接线一次**。
**别用「重注册一遍界面」那种刷新**（ADR 0031 复评记录三已经吃过一次亏）。
视图状态里**只有缩放**用公开 API 还原（`scaleVal` / `scale()`），选中项 / 专注模式 /
平移位置都不是公开 API，**别去撬内部状态** —— 升级即烂，而且撬出来的形状
会静默地丢内容。想要更多的视图状态，得先给内核加公开 API。

**导图画出来的样子归内核（2026-10-12，ADR 0050）**：对比主流导图软件后自派的
需求（用户原话「你自己比较一下现在主流的思维导图软件，看看你还欠缺什么，然后
自己给自己需求继续开发」）—— 缺口是「完全不能把导图导出成图片」。落成的形状：
① 内核契约新增**可选**方法 `exportImage(format)`，产物是 `{ blob, warnings }`；
`toHtml()` 对导图**依旧返回 null** —— 富导出的正解不是让外壳拼语义 HTML
（连线 / 盒子 / 配色长在内核的 DOM 里，外壳没有那部分信息），而是「请内核把自己
画一张」。**别把这两条路混了。** ② `warnings` 是「绝不静默」在跨层消息上的落点：
外链图片节点在 SVG 里只剩引用、在 PNG 里**画不出来**（SVG 作为图片加载时不取
外部资源），这个实话只有内核知道，用 `LocalizedText`（ADR 0047）传出来。
**别删警告，也别升级成拒绝** —— 用户无从修复（内核读不了盘，内联不了）。
③ 底色有个坑：内核把 `theme.cssVar['--bgcolor']`（我们配的是 `var(--surface)`）
**原样**写进独立 SVG 的底板 `fill`（`MindElixir.js:2671`），独立文件里没有
`--surface` 定义就会黑底。修法是画图前用 `getComputedStyle` 拿**真实解析色**
（浅 `#ffffff` / 深 `#1f1f1f`，跟着主题走），解析不出（空串或还含 `var(`，
jsdom 就是）就**保留原样** —— 宁可底色可疑，不编颜色。**两条纪律**：
`mind.theme` 与 `LEVNOTE_THEME` 是**同一个引用**，必须整体重赋值 + `finally`
还原，原地改会污染下一个实例；`exportPng` 第一行是**同步**调 `exportSvg` 拿到
SVG 串再异步转换，所以「换主题 → await → 还原」的窗口是安全的，别改这个顺序。
④ PNG **不自己重画**（内核 `exportPng()` 全链完成，重实现要追内核序列化语义跑，
升级即烂）；文字保留内核默认的 `foreignObject`（保真高，代价是部分老 SVG 查看器
不显示 —— 写在 ADR 的取舍里，等真实抱怨再动）。⑤ 编排**先问后画**
（`core/export/export-image.ts`，与 export-document.ts 并列不是分支）：PNG 位图
编码费时，用户在对话框上点取消不该白画。⑥ 导出家族**统一不覆盖**：
`writeBinaryFile` 是 create_new（HTML 导出的 `writeTextFile({create:true})`
同语义），CONFLICT 原文如实转达 —— **别在图片这条偷偷放宽**，要覆盖是家族级
的决定（见 ADR 0050 重审条件）。⑦ 两条命令 `file.exportPng` / `file.exportSvg`
**按能力启用**（`typeof handle.exportImage === 'function'`，diagnostics.roundTrip
同款形状，**绝不**按内核 id 分支），不给快捷键（低频动作，`Mod+Shift+E` 已归
HTML 导出）。⑧ 测试边界要诚实：**PNG 路径在 jsdom 不可测**（`Image` 永不加载，
只会挂起 —— 这不是可测而未测），像素零测试（jsdom 无布局，画布 200×200、
底色走回退分支）；单测只钉「参数对 / Blob 类型 / 如实失败 / 警告拼接」，
像素靠 `pnpm run tauri dev` 人工冒烟（两格式 × 两主题，独立打开验底色）。
另外 `definitions.test.ts` 里跑命令用动态 `import()` 的，**测试文件头部要
side-effect 预载同一个模块** —— 整跑负载下首次装配不止一个宏任务，只等一个
`setTimeout(0)` 是 flake（本轮踩过）。

**图纸接上同一条图片导出路（2026-10-12，ADR 0051）**：继续对比后自派的另一半
需求 —— `.drawio` 文档此前**完全不能**导出图片，而 draw.io 官方工具把「导出
PNG / SVG」放在第一屏。这轮是 ADR 0050 重审条件（「第三家内核接入时复用同一
契约与 export-image.ts，不另开导出路径」）的兑现，也是检验：`DrawioEditor.tsx`
一实现 `exportImage`，`.drawio` 文档上 `file.exportPng` / `file.exportSvg`
**自动亮起** —— 能力检查、菜单、编排、写盘纪律**外壳一行没改**，契约放
`core/editor/types.ts` 的意义第一次被兑现检验。图纸内核的三个事实定的形状，
**别拿导图的经验去套**：① 查看器（viewer.min.js）**没有任何导出 API** ——
mind-elixir 有现成 `exportPng()` 可借，这里没有，PNG 只能内核层自建
Image+canvas（失败一律交 null 让外壳如实报；将来拿到 draw.io 编辑器资产时
这条自建链退役，借编辑器自己的导出，见 ADR 0051 重审条件）。② 底色**不用换**：
mxGraph 把解析好的底色写进 svg 根的**行内 style**（实证
`this.svg.style.backgroundColor = <色>.cssText`），序列化天然带走；PNG 画布
那侧要自己铺 `getComputedStyle(svg).backgroundColor`（解析出 `rgb` 系且非
全透明才铺，解析不出**不铺也不编色**）。③ 序列化必须 `XMLSerializer`，**不是
`outerHTML`**：mxGraph 图片形状用 `xlink:href`，`outerHTML` 不补命名空间声明，
那样的文件独立打开按 XML 一解析就报错 —— **这条换了写法产物当场废**。
`exportImage` 先 `await renderRef.current`（在途那趟 `renderDrawio`）——
「打开立刻导出」不许拍到半张图；画布上没有 svg（失败 / 空文件）按契约交
`null`。导的是**此刻画布上的那张**（当前页、当前缩放，所见即所得）。
警告三句随文件走（`LocalizedText`）：外链图片 PNG 画不出 / SVG 保留原引用
（`href` / `xlink:href` 非 `data:` 都算，两处都看）、缺 stencil 库的形状
**按矩形显示与屏幕一致**（那句约定离开应用没人解释，得跟着文件走）。
ADR 0048 给 `toHtml()`-null 留的重审条件（「外链图片怎么办」）里的
「**明确标注**」在此兑现；**换成内联未做**（内核没有读盘能力），
`toHtml()` 维持 `null` —— 两条路各答各的，别混。测试边界同导图：PNG 在
jsdom 不可测（`Image` 永不加载）、像素零测试，靠 `pnpm run tauri dev`
人工冒烟；测试形状教训：**这套挂载每个 it 只 mount() 一次** —— 同一 it 里
第二次 mount() 复用同一 root，React 视为「更新」而非重挂载，onReady
（deps `[]`）不再发 handle，`expect(p).resolves` 直接拿到 undefined。

**导图的大纲面板（2026-10-12，ADR 0052）**：对比 XMind 后自派的第三轮需求 ——
大纲视图在导图产品里是第一屏功能，而 LevNote 的活动栏大纲按钮此前只认
Markdown。这轮定了四件事，**换内核时最容易烂的是第一条**：
① 大纲提取器 `core/outline/mindmap-outline.ts` **必须复用 `parseMindmap`**
（`level` 是树深、`line` 是先序序号、条目带文件层 id）—— 面板与内核看到
「什么算一个主题」靠同一份判定；自造第二份迟早漂移，跳转会悄悄指错地方。
② 契约 `goToOutlineItem({ level, text, id? })` 的 `id` **只有导图传**
（Markdown 编不出真值就别编，缺着比假值诚实）；主题可重名，id 才指得准。
③ 内核跳转（`MindmapEditor.tsx`）的两个**写死的例外**：**根不展开** ——
mind-elixir 的 `expandNode` 读 `i.children[1]`（+/− 气泡），根没有气泡，
对根调用**抛 TypeError**；且根折叠在视觉上不生效（子分支永远渲染，
jsdom 探测实证）。**只读不展开**（展开是编辑，交 false；可见的照跳，
`selectNode` 不发事件，选中不是编辑）。`findEle` 对看不见的节点 throw
（`FindEle: Node … not found, maybe it's collapsed.`），**捕获交 false**。
④ 展开折叠祖先是**文档变更**：`collapsed` 是文件字段，onChange + 标脏照常
—— 跳转把看不见的地方变可见，本来就该存进文件；不标脏才是撒谎
（用户看到展开了，保存重开却还是折着的）。面板按 `isMindmapPath` 分派
提取器与文案（外壳知道「哪些文件是导图」是产品范围，不知道用哪个内核）。
测试边界：jsdom 无布局，`scrollIntoView` 的真实滚动不断言，断言返回值 +
展开链经 `getValue` 落回文本；人工冒烟（开 `.mindmap` → 开大纲 → 点深层条目）。
工具教训一条：PowerShell `Set-Content` 不带 `-Encoding` 会写出 **UTF-16**，
把 TS 文件整个毁掉（read 报 invalid UTF-8）—— 造文件用
`New-Item` 占位 + read + `write` 工具，不要用 shell 重定向写源码文件。

**导图标记（2026-10-12，ADR 0053）**：继续对比 XMind 后自派的第四轮需求 ——
优先级标记在导图产品里是第一屏核心能力，而 LevNote 的 `.mindmap` 节点此前
没有任何标记字段（`icons` 在格式层只是不透明透传，没有 UI 碰它）。
这轮定了四件事：
① **字形即内容**：格式层把 `icons` 升格为已知字段（序列化键序 `style` 之后、
`extra` 之前，`[]` 也写回；适配层已知字段清单双向透传），存**显示字形本身**
（与 `topic` 存显示文本同理）—— **不发明语义 id + 双向映射表**：那要永久维护
两张表，且与 extra 桶透传的 `icons` 形成双表示冲突；字形还最内核无关
（能排文本的内核就能显示）；词表外字形照 §5 原样保留（调色盘只是词表知识，
住内核模块，不进格式层语义）。
② **词表**：`modules/mindmap/markers.ts` 三组 19 字形 —— 优先级 `1️⃣`–`9️⃣`
互斥、颜色 `🟥🟧🟨🟩🟦🟪` 互斥、符号 `⭐✅❗❓` 不互斥；`applyMarkerToggle`
纯函数：同字形再点一次取消，互斥组先移同组旧字形再追加到尾，词表外字形按
非互斥 —— 任何路径都不许因词表外形值炸掉。
③ **内核 UI**：右键菜单 `contextMenu.extend` 追加「标记…」（extend 项在菜单
尾部，onclick 里 `mind.currentNode` 即右键到的节点 —— 内核在弹菜单前已
`selectNode`），打开调色盘；选中走 `reshapeNode` → `operation` 事件 → 既有
emit 主通道，标脏 / 保存不用单独伺候；内核 `be` 转义 `& < "`（字形不可注入）；
`getData` 是 `JSON.parse` 深拷贝、icons 照带。
④ **全清过 = `[]` 一路到底**：`JSON.stringify` 丢 undefined 键，用
`{icons: undefined}` 清空会把「全清过」状态**静默丢掉**（空数组是状态本身）。
边界：icons 类型不对整值转 `extra`（格式层不猜）；只读 `contextMenu: false`
自然关门；大纲条目把标记内联显示（`OutlineItem.markers`）。
测试边界：jsdom 三个坑踩过写下 —— 右键必须是 `button === 2`、内核
`setTimeout(200)` 后才出菜单、topic 选择器是 `me` 前缀（`data-nodeid="me"+id`）；
观感（字形大小 / 调色盘样式）需 `tauri dev` 人工冒烟。
**86 文件 / 1,775 条测试**（markers 12 + MindmapEditor 7 + parse 3 +
serialize 2 + adapter 1）。
工具教训：PowerShell `Get-Content | Set-Content` 回环在 zh-CN 下把无 BOM
文件**按 GBK 双重编码整个毁掉**（恢复 = `git restore --source=HEAD` + `write`
工具重写；教训已入运行时记忆）；`.at(-1)` 要求 lib ≥ es2022，tsc 报 TS2550，
写 `arr[arr.length - 1] ?? ''`。

**导图节点插图（2026-10-12，ADR 0054）**：第五轮自派需求 —— XMind 的主题
插图是第一屏能力，而 LevNote 导图此前零图片能力。**接 Markdown 已有的整套
图片机制，不另造一套**（`resolveImageUrl` / `onImagePaste` 早就在 `EditorProps`
里、`DocumentPane` 对每个内核统一传 —— 内核只解构，外壳零改动）：
① **格式层**：`image` 照 `icons`（0053）先例升格为正字段，形状照内核
`{ url, width, height, fit? }` 不发明中间表示；`url` 存**文件 src**（相对路径
或 data:），尺寸即内容（不存它下次打开就是内核的默认猜值）；**整对象照收**
（不认识的子键原样跟着走），类型不对**整值转 extra**；键序 `icons` 之后、
`extra` 之前；适配层双向直通。
② **显示 = 预热 + `imageProxy` 同步查表**：`imageProxy` 是同步的而解析是异步
的 —— 预热必须在内核创建**之前**（收集 `image.url` → `Promise.all` 逐个
`resolveImageUrl` → 填 Map → 再装内核）；`data:` 透传、查得到换 data:、
**查不到画占位图**（内嵌 SVG 虚线框 + 🖼 + 路径如实写在里面）——「图丢了
是常见状态」，内核的「缺则不画」是静默，占位才是如实；http 外链也走占位
（ADR 0001 零联网不破例）；语言切换重建即重跑预热（0049 的命门顺带解决）；
`setValue` 换文档补预热后对带图节点 `reshapeNode(el, {})` 空 patch 重渲染。
③ **插入 = 粘贴**：内核 paste 的官方让位口 `pasteHandler`（内核先吃「复制的
节点」magic JSON，吃不下才交原始事件）→ 剪贴板 `files` 与 `items` **两处都看**
（平台填充不一致，markdown `imageFilesOf` 的既有教训）→ 没选中主题如实说
（`onError`，绝不静默）→ 字节交 `onImagePaste` → 文件 src → 内核层
`decodeImageSize`（`createObjectURL` + `Image`）→ 最长边 320 缩好 →
`reshapeNode({ image })` 落既有主通道；外壳交回 null 时内核**静默**（外壳已
notice，再报就是两条消息）；只读第一行 return —— 图片是编辑，不悄悄放行。
④ **移除 = 右键「图片…」弹层**：有图显示引用 + 显示尺寸 + 移除
（`{image: undefined}`，stringify 干净丢掉、文件里不剩半句）；无图显示
「按 Ctrl+V」提示 —— 空弹层是静默。`fit` 刻意不进弹层（纯展示，别先造编辑 UI）。
⑤ **工程教训**：预热是异步的 ⇒ effect 的销毁可能抢在创建之前 —— effect 体
改成 `boot` 闭包返回 teardown，`cancelled` 标志守卫预热回调。jsdom 的
`DataTransfer` 是「files 有货、items 是空壳」的填充方式（恰好钉住两处都看）；
`Image` 不解码，`decodeImageSize` 测试必须 `vi.stubGlobal`。
**87 文件 / 1,812 条测试**（node-images 20 + MindmapEditor 9 + parse 6 +
serialize 1 + adapter 1）；像素观感需 `tauri dev` 人工冒烟（开带图的
`.mindmap`、粘贴截图、挪走图片文件看占位）。

**导图节点链接与标签（2026-10-12，ADR 0055）**：第六轮自派需求 —— XMind
第一屏的最后两块（第三个候选 `note` 内核完全不渲染，刻意跳过，重审条件
单列）。要点全在 §三那张表的行里，这里只记工程教训：**① 内核对
hyperLink 零 UI 是实测出来的**（菜单里的 `link` 项是节点连线
`createArrow`，grep 全库锚点无任何点击监听）—— 「内核有这个字段」和
「内核有这套交互」是两回事，落码前 grep 一遍再说；**② jsdom 实现了
`innerText` 但 `textContent` 是空**（探针实证）—— 内核拿 `innerText` 画
🔗，读回就得顺「写进」的那条 API 走，别想当然换成姐妹属性；**③ eslint
对 `?.` 上的方法引用报 unbound-method，cast 也不认** —— 仓库既有惯例是
`Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.bind(input)`
（App/SearchPanel/Sidebar 测试里四处同款），别再发明第五种写法；
**④ tsconfig 的 target 不到 es2022 就没有 `.at(-1)`**（TS2550）—— 照旧写
`arr[arr.length - 1] ?? ''`。测试：**87 文件 / 1,830 条**（MindmapEditor
弹层/激活/只读链路 +8、parse 升格回归 +7、serialize +2、adapter +1）；
观感与真 WebView 的导航拦截需 `tauri dev` 人工冒烟（挂一条链接点它、指向
另一份 `.mindmap` 再点、挂两个标签导出 SVG 用浏览器打开）。

**导图节点备注（2026-10-13，ADR 0056）**：第七轮自派需求 —— 上轮刻意跳过的
`note` 这轮补上。要点全在 §三那张表的行里，这里只记工程教训：**① note 内核
零渲染是探针实证的**（`me-tpc` 里一个字不画）—— 可见性自己建，但**不 fork
内核渲染**（fork 会在内核升级时整个丢掉），同步挂 init 后一次 + `linkDiv`
监听两处；**② undo/redo 走 `refresh(prev)` 且不发 `operation`** ——
`operation` 是编辑主通道、`linkDiv` 是重建兜底，两条腿缺一不可（徽章重同步
+ `scheduleVerify` 300ms 迟到的脏标记，迟到不是缺席）；**③ jsdom 里 Ctrl+Z
必须派在 `.map-container` 上**（内核监听挂 `e.container`，host 派发够不着
—— 事件不向下传播）；**④ undo 恢复的文本不是逐字节**（内核规范化补
direction）—— 判据用语义不用字节；**⑤ 设计洞自查**：徽章唯一入口 ⇒
没备注的节点永远创建不了第一条，双入口（徽章 + 右键「备注…」）堵上。
测试：**87 文件 / 1,839 条**（MindmapEditor 备注 describe +7、mindmap-outline
+1、App 大纲 📝 +1）；观感需 `tauri dev` 人工冒烟（长备注弹层高度、
多主题徽章对齐、undo 后徽章重现）。

**markdown 内核文案跟随语言（2026-10-13，ADR 0057）**：第八轮自派需求 ——
0047 当年登记「落点在内核绘制侧，暂不做」的那块，这轮补上。这里只记工程教训：
**① 注册在案的清单不可信**：0047 只数了 6 处，grep 实证是两文件 28 处 ——
登记只记了标签，漏了纸条、卡片与摘要区；范围靠 grep 全量重数，不靠旧账。
**② 机制跟着既有回路走**：badge 有 `drawnLabel` 短路就加 `chromeLocale` 域、
图区有 `painted` 盖章短机制就给盖章加 `locale` 成员、`ImageView` 没有重画
回路才发明 `failLine` —— 三种形状不是三种发明，是对三种既有回路的顺应。
**③ 订阅的生与死要对称**：`$prose` 工厂闭包每编辑器实例跑一次 —— 订阅在
闭包里、退订必须在 `view.destroy`；NodeView 构造 `add`、`destroy` `delete`，
否则编辑器销毁后监听器泄漏。**④ 别人的话不翻**：mermaid 与 `renderDrawio`
的 `result.message`、`DocBlockPreview.reason` 原样转述 —— 用户拿它去搜上游
issue，翻译了反而害人。**⑤ zh 逐字照抄旧硬编码** ⇒ 存量断言一行不动
（兼容即契约，0047 的「逐字节一致」纪律在 i18n 上的对偶）。
测试：**87 文件 / 1,842 条**（badge title / doc-block 失败纸条 / image
失败纸条三机制各 +1）；en 文案的字体宽度对 chrome 布局的影响需
`tauri dev` 人工冒烟。

**导图查找也搜备注（2026-10-14，ADR 0058）**：第九轮自派需求 ——
0056 取舍 ⑤ 留的边界、0057 重审条件 2 登记的「另一轮」，本轮兑现。这里只记工程教训：
**① 注释过期要改写**：0056 的「只搜 topic 因为内核不渲染 note」在徽章落地那一刻
已经是错话 —— 真正缺的是 `replace` 的写回路由；错注释比没注释更能把后人带偏，
动手前先把理由重写成「偏移是段内的，分通道才搜得全」。
**② `key` 是最便宜的换状态手段**：弹层草稿挂载播种（`useState` 初值），鼠标流有
外点自愈、键盘流（查找栏 `reveal` 连跳）没有 —— 挂 `key={nodeId}` 换挂载点，
别补同步 effect。
**③ 两段独立文本必须分段键控**：`replaceAll` 按 `${field}:${id}` 分组、组内倒序、
报命中数 —— 混作一段会互踩偏移，数节点在分家后说不通。
测试：**87 文件 / 1,847 条**（MindmapEditor 备注搜索 describe +5）；
真 WebView 里键盘连跳 note 命中的观感需 `tauri dev` 人工冒烟。

**最该先做的一件事**：ROADMAP 上本地可完成的项已全部落地。剩下的都需要
真实世界资源：P5-26（draw.io 编辑器资产，需联网取一次）/27（分发环境）、
P7-32/33（签名主体）、P7-35（Mac）、P7-36/37（商标与域名）。

> **✅ 纪律已解除（2026-10-02）**：`open-decisions.md` 里的 V1–V4 全部有结论，
> 待验证清单已清空，**不再有「出结论之前不许新增待验证项」这条约束**。
> 但「**先量再改**」这条不解除 —— 见 `ROADMAP.md` 的六条方法教训。
>
> V1（draw.io 查看器）2026-10-02 已测完 —— 三条判据全过、决策维持，
> 但翻出两条新风险（**stencil 缺失静默降级** / **`stencils.min.js` 不含 `shapes/*.js`**）。
> **给 P5 的四条硬要求写在 ADR 0005 文末**，动手前先读那一段。
> V2（mind-elixir 500 节点）判据修正为 `refresh`，结论是「成本由顶层分支数决定」，
> 维持 mind-elixir（ADR 0007 复评记录）；
> V3（Milkdown 长文档输入延迟）已测完并落地（ADR 0004）；
> V4（表格可视化编辑的成本）已由 M5 回答 —— 自研可行、成本可接受。
>
> **P4 接入内核时新踩的四个坑**（都写进 ADR 0025 与 `PITFALLS.md`）：
> 注册顺序把 `code` 排在了导图前面（它认得 `.mindmap`，于是全被它吃掉）；
> 「没改过就不写」的基准取错了时机（内核跑完首次布局会给顶层节点补 `direction`）；
> jsdom 没有 `matchMedia` 而内核在构造函数里就要它；空文件的「原文」不能当回退目标。
>
> **P4 第 23 项（全文检索）又踩的一个坑**（写进 ADR 0026 决策 7）：
> **React 19 的 StrictMode 会把「模拟卸载→重挂」推迟到挂载之后的某个异步点**。
> 外壳握着句柄「戳一次」的定位刚好落在重挂之前，随后那次重挂把选区与
> 状态栏的光标读数一起销毁，而外壳早已把待办清掉 —— 再也没有人补那一次定位。
> 修法是把「跳到某一行」变成与 `initialPosition` 同构的东西：
> 请求作为 prop 传给面板，**面板在 `handleEditorReady` 里应用**，
> 任何一次内核重建都会重新落地。判断依据由此多一条：
> **凡是「等内核就绪才该落地」的意图，都走 ready 回调 + prop，不走命令式戳一下。**

**空壳模块**（返回 `NOT_SUPPORTED`，等对应阶段落地）：
`src-tauri/src/drawio/mod.rs`（`watch` 已于 2026-10-04 转正，见 ADR 0029）。

---

## 七、明确不要做的事

| 做法 | 原因 |
|---|---|
| 参考 Typora 的代码 | 闭源商业软件。只做功能对标 |
| 参考 Notepad++ 的代码 | GPLv3，会传染整个项目 |
| 用 draw.io 的名称/图标做产品标识 | 商标约定（ADR 0005） |
| 引入 GPL 系依赖 | 同上 |
| 用 `.mindmap` 的完整内核 schema | 规范刻意只固定最小骨架，绑定内核等于让换内核时用户的文件失效 |
| 引入 `@milkdown/components`（官方表格 / 代码块组件） | 它是 **Vue 3** 写的，会带进第二个 UI 框架并替换掉编辑器的高亮管线（编辑器与导出从此不同源）。理由与重审条件见 [ADR 0017](docs/decisions/0017-m5-tables-and-code-blocks.md) |
| 打开 Tauri 的 `asset:` 协议来显示本地图片 | 它要配一份**路径白名单**（scope），是一块需要长期维护的安全面。`data:` 走已有的 `read_binary_file`，权限收口在同一个地方。代价有上界（上游卡了 24MB）。重审条件见 [ADR 0018](docs/decisions/0018-m6-images.md) |
| 为「可能被否的方案」先写代码 | 先跑验证。ADR 0005 的 R1 就是这么处理的 |
