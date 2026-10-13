# 0057 · markdown 内核自己画的字，也得跟着语言走

- 状态：已采纳
- 日期：2026-10-13
- 关联：[ADR 0047](./0047-words-that-outlive-the-language-switch.md)（它登记过「block-preview 与三个 `Render.message` 出口暂不做（落点在内核绘制侧）」—— 本轮补的就是这块）、[ADR 0049](./0049-mindmap-ui-follows-the-language.md)（内核 UI 跟随语言的首例，导图走重建；本轮是 markdown，走重画）
- 承接：ADR 0031 的 i18n 骨架、ADR 0047 的 `t()` 跨渲染判据

## 背景

ADR 0049 让导图内核的右键菜单跟随语言；markdown 内核却整模块**零 i18n 导入**。它自己画的 chrome —— 代码块的行号槽与语言标签、图区（mermaid / doc-block / 引用卡片）的加载与失败纸条、图片的占位说明 —— 全是硬编码中文。英文界面里这些字还是中文：`t()` 的完整性红线（0047 的判据：**内核自己画的字也是 UI**）被击穿。

grep 实证（落码前）：`code-block-view.ts` 24 条、`image-view.ts` 4 条用户可见硬编码。0047 当年登记在案的「6 条」只是子集 —— 纸条、卡片、摘要区全漏了。

## 判据

1. **三处重画回路形状不同，机制只能各自配**：
   - badge（语言标签）：构造器同步画，`renderChrome` 有 `drawnLabel` 短路 —— 加一个 `chromeLocale` 域即可；
   - 图区纸条：`painted` 盖章短机制已有（`kind` / `source` / mermaid 的 `theme`）—— 加 `locale` 成员，`paintedFresh` 判据跟着失效 → `ensureDiagram` 整链重渲染；
   - 图片占位：`ImageView` **没有重渲染回路**（DOM 是 `load()` 一次性搭的）—— 只能原地换字。
2. **语言切换不重建编辑器**（与 0049 的差异）：markdown 重建成本是 ProseMirror 状态、undo 栈、选区全丢。主题切换已有同构先例 —— `MutationObserver` 看 `data-theme` → 遍历 `live` 重跑 `ensureDiagram`。语言照抄这条通道：`subscribeLocale`（`core/i18n/index.ts:76`，`(listener: () => void) => () => void`）。
3. `setLocale` **同步**触发订阅者 —— chrome 重画可同步断言；图区重渲染走微任务，`flush()` 即可。
4. zh 文案与旧硬编码**逐字相同** ⇒ 存量断言（`code-block.test.ts` 的 `badgeLabel '纯文本'`、`docblock-block.test.ts` 的五条纸条断言、`image.test.ts` 的带路径占位）一行不动 —— 兼容即契约。

## 取舍

1. **26 对键住 `ui.markdown.*`**（zh 逐字照抄旧硬编码；插在 `ui.mindmap` 块后，zh/en 键集与占位符集一致 —— i18n 测试锁）。别人的话除外（取舍 6）。
2. **`paintDiagramNote` 签名不改**（收已转好的 `string`）：语言重画靠 `painted.locale` 失效走重渲染 —— 纸条本来就是渲染产物，重跑渲染才是正路。记 `LocalizedText` 那套留给真正没有回路的图片占位（取舍 4）。
3. **badge 走 `chromeLocale` 域**：locale 变 → `drawnLabel = null` → 重画 `title` / `aria-label`（`textContent` 顺带重设同值，无害）。`wantDiagram` 刻意**不加** locale 判据 —— 它只答「这块内容配不配当图」，不答「画没画过」。
4. **图片占位记 `failLine`（`LocalizedText`）**：`fail()` 改收 `LocalizedText` 存 `failLine`，`onLocaleChange` 按 `state` 原地换字（`is-loading` → 重说 loading；`is-failed` → `textOf(failLine)`）。
5. **两文件同构的 live 注册表**：`$prose` 工厂闭包里 `subscribeLocale` + `Set`，NodeView 构造 `add` / `destroy` `delete`，插件 `view.destroy` 退订 —— 每编辑器实例一份，不泄漏。
6. **别人的话不翻**：mermaid 与 `renderDrawio` 的 `result.message`、`DocBlockPreview` 的 `reason` 原样转述 —— 用户拿它们去搜上游 issue，翻译了反而害人。头注释与分支注释注明这条边界。

## 后果

- **测到的**：三机制各配一条语言切换用例锁住 —— badge `title`/`aria`（`chromeLocale`）、doc-block 失败纸条（`painted.locale`，且 `resolveDocBlock` 被再调一次、渲染器 `spy.calls` 仍 0 —— 锁「重画不重渲」）、图片失败纸条（`failLine`）。87 文件 / 1,842 测试全绿（+3）。
- **没测到的**：真实切换的观感（英文串更宽对 chrome 布局的影响）需 `tauri dev` 冒烟。
- **已知边界**：编辑中的代码块（`mode==='code'`）在语言切换时图区不重画 —— 退出编辑才重画（`ensureDiagram` 的 editing 短路是既有语义，语言跟着它走）；mermaid SVG **内部**的文字（图表自己的节点标签）不归我们翻。
- **工程教训**：`$prose` 工厂闭包是每编辑器实例执行一次 —— 订阅若不在 `view.destroy` 退订，编辑器销毁后监听器泄漏；edit 工具对行首缩进敏感，批量 `old_string` 用唯一子串去前导空格。

## 重审条件

1. 换 markdown 内核时 NodeView chrome 全套跟着重建 —— `ui.markdown.*` 键表退役重审，但三机制的形状（chrome 域 / 渲染盖章 / 占位记 Line）可复用。
2. note 搜索（`EditorSearch` 契约扩展）是另一轮 —— 0056 重审条件已登记，不在本轮。
3. mermaid 图表内的文字（节点标签、边标签）要不要跟随语言 —— 数据还是文案，出现真实需求再议。
4. 上游若提供本地化钩子，「别人的话不翻」重审。
5. 若要求「编辑中的代码块也当场重画纸条」，重审 `ensureDiagram` 的 editing 短路与语言失效的相互作用。
