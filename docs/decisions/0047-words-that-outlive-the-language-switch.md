# 0047 · 切了语言，话还是旧的：把烤死在返回值里的 `t()` 交回 key

- 日期：2026-10-13
- 状态：已接受
- 关联：硬约束 2（绝不静默）、ADR 0031（命令标题与内置文案为什么不能是 `t(...)`）、
  ADR 0046（记录本存 key + 参数，不存成品句子）、ADR 0043（降级要说出理由 ——
  说出来的**那半句**自己也得是活的）

## 背景

ADR 0031 复评的时候，我们把「一个 `t()` 的结果被烤进一个活过渲染的结构」这一族
记在了 `core/document/store.ts` 的文件头里（「本轮的已知残留」），一共七条。
它们的共同形状是：

```
某个 core 函数 return t('...')        // 或者更常见：
某个 core 函数把 t('...') 塞进一个 state / store 字段
```

`state` 与 `store` 字段**跨渲染活下来**。所以用户在运行途中切语言时，
这些句子停在旧语言上——而**它们旁边**的一切（菜单、面板、状态栏）都跟着换了。
这比「没有 i18n」更糟：没有 i18n 的用户至少知道自己看的是一种语言；
一半中文一半英文的界面读起来像坏了。

七条里没有一条是「忘了翻译」，每一条翻译都在、每一条都查表了——
只是查表发生得太早，早在句子被存进去之前。

## 判据

不是「文件属于 core 还是 ui」，而是这两条：

1. **`t()` 的结果是否跨渲染存活**（进了 state、store 字段、或任何会被存起来的返回值）；
2. **能不能在不改内核契约的前提下修**。

第 2 条把白名单当成了「**允许动的边界**」而不是「必须改的清单」——
`DocumentPane.tsx` 的 `editorError` 在白名单里，但它存的是 `thrown.message`
（内核抛出的原话），不是 `t()` 的结果，所以**不在这一族**；
`App.tsx` 的 `resolveDocBlockPreview` 里的 `reason` 不在白名单，但它要改就得动
`core/editor/types.ts`，所以这一轮**够不着**。

## 取舍

### 1. 一个共享形状：`LocalizedText`

```ts
export interface LocalizedText { readonly key: MessageKey; readonly params: Readonly<Record<string, string | number>> }
export function msg(key, params = {}): LocalizedText
export function textOf(line: LocalizedText): string   // 只在渲染的那一刻调
```

字段名取 **`params`**——与 `t(key, params)` 和 `Notice` 的 key 变体一致。
ADR 0046 自己起了一个 `HealthLine { key, values }`；同一个想法有两种写法
就是「抄两份」，所以 `core/health.ts` 里的 `HealthLine` 现在是
`type HealthLine = LocalizedText`（别名），`values` → `params` 一并改名。

`msg()` 在没有参数时返回**同一个** `{}` 引用——ADR 0046 的 `resolveIssue`
靠引用相等做幂等（`syncFamily` 的「这一帧什么都没变就不重渲染」），
换了形状不能顺手把那个性质弄丢。

### 2. 抛物的原话当**参数**走，绝不当句子翻

`detail` / `lastReason` / `thrown.message` 这类东西一律进 `params`：
翻一个 `EACCES` 或者一段 Rust 的 IO 错误，只会得到一句听起来像那么回事、
实际上什么都不是的话（ADR 0043 的「降级要说出理由」：理由必须是真理由）。

### 3. 句子在哪里拼，就在哪里翻——**只留一处**

跨文件替换的汇报原来是 `describeReplaceOutcome()` 拼好的一句话
（`t()` 拼的片段 + 硬编码的 `'，'`）。改成：

```ts
export interface ReplaceReport { readonly headline: readonly LocalizedText[]; readonly problems: readonly ReplaceProblem[] }
```

`renderReplaceReport()` 是**唯一**把汇报变成句子的地方。注意逗号也是一个 key
（`core.replace.join`：中文 `'，'`、英文 `', '`）——把已经翻好的片段用硬编码标点
串起来，得到的还是一句各说各话的句子。

问题行**先失败后跳过**：失败比跳过更需要被看见。

### 4. `TreeProblem` 不留 `code`

树这一栏的十一种失败**全是 `IO`**。一个恒定的值不是数据，留着它会让人以为
将来要按它分支。（`openError` 的 `AppErrorShape` 保持原样——那是另一族、
另一份消费方，它们要的是能接 `asAppError` 的原样句子。）

### 5. `legacyText()` 连同品牌门一起删掉

`DocumentStore` 的 `Notice` 曾经是一个结构化联合：`{ key, params } | LegacyTextNotice`，
`renderNotice` 是它的渲染器。那扇门当初只为 `images.ts` 留着——
后者把六句话烤成了字符串，而 `LegacyTextNotice` 是「这些是原话，别翻译」的**品牌**。

`images.ts` 交回 key 之后（它自己的重审条件就是这么写的），那扇门背后
**什么都没有了**。留着一扇没有门后的门的门，比删掉它更糟：
下一个遇到「这个联合有第二个变体」的人会以为第二个变体还有存在理由。

`FindBar` 与 `DocumentPane` 的 `notice` 随之从 `Notice` 收成 `LocalizedText`。
`localizedError` / `localizedConflict` 那两个 getter **留着**：
`AppErrorShape` 的消费方（`openError`、`core/search/workspace.ts`）
不在这一轮的边界内。

## 这一族现在的账

| 处 | 存的东西 | 谁翻 |
|---|---|---|
| `TreeState.error`（`workspace/store.ts`） | `TreeProblem { message: LocalizedText; detail: string \| null }` | `Sidebar.tsx:142` |
| `validateEntryName`（`path.ts`） | `LocalizedText \| null`（8 条规则逐条 `msg(...)`） | 同上 |
| `ReplaceReport`（`search/replace.ts`） | 片段数组 + 问题行 | `renderReplaceReport()` |
| `SettingsDialog.error` | `LocalizedText` | 对话框渲染处 |
| `PasteImageFail.reason`（`document/images.ts`） | `LocalizedText` | `DocumentPane` |
| `HealthLine`（`health.ts`） | `= LocalizedText` | `HealthDialog` |

### 登记不做（连同理由一起钉住，别让人重新扫一遍）

- **`core/document/block-preview.ts`**：`buildDocBlockPreview()` 的 `reason`
  （三处 `msg`）与 `firstLines()` 截断尾拼进 `snippet` 的那句 `t(...)`。
- **`core/katex.ts` / `core/mermaid-render.ts` / `core/drawio-render.ts`**：
  三个 `Render.message` 出口。实测切 en 之后同一个结果对象里的字仍然是中文。

这两族落在**内核的绘制侧**与 `core/editor/types.ts` 上，要改就是改契约——
那是另一个爆炸半径，该由一次专门的 ADR 来定，不该顺手塞进一次「外壳侧」的清理里。

### 两处「看着像、其实不是」

- `App.tsx` 的 `resolveDocBlockPreview` 回调里的 `reason: t('ui.app.refNoBaseReason')`：
  要改得动契约（白名单外），**而且那个分支在生产里不可达**——标签页只在
  `store.open()` 成功之后才入 `tabs`，活动面板的 `doc` 恒非 null，
  唯一置 null 的 `close()` 在生产代码里无人调用。
- `DocumentPane.tsx` 的 `editorError`：存的是 `thrown.message`，内核抛出的原话。
  判据落在**来源**上，不是文件归属。

## 后果

- `core/i18n/index.ts` 加 `LocalizedText` / `msg` / `textOf` 三件套。
- i18n 删 2 个 key（`core.replace.unreadable` / `unwritable`）、
  加 4 个（`readFailed` / `writeFailed` / `join` / `problemLine`），× 两种语言。
- `workspace/store.ts`：`failTreeAction` 的签名从 `string` 变成 `LocalizedText`，
  十处调用点跟着改；`toggleDir` 里直接写 `error` 的一处改成同一个形状。
- `core/health.ts` 的 `HealthLine` 收成别名，`values` 改名 `params`（约 25 处）。
- `document/store.ts` 净减 179 行：删掉的是那扇没有门后的门。
- 新增测试 12 条，其中 6 条是**切语言的外部用例**：
  「同一份 store 字段 / state，重画一次就换了字」——
  这类用例在修复之前全部是红的，因为那时存的就是句子。

## 重审条件

- **`textOf` 调在了渲染之外。** 它是「翻译要推迟到最后一刻」这条规则的**唯一执行点**，
  一旦有人在别处调它用来「先算一遍」，推迟就名存实亡了。将来若要做富文本 / 多行排版，
  要么新增一条只给渲染用的路径，要么把这个函数挪进渲染层。
- **`LocalizedText` 开始长第三个变体**（比如带 `when` 的条件句、
  或者带复数规则的句子）时，先别加第四个 key 了——那说明句子不是被烤死的，
  而是**键本身不够**。届时该重审的是 catalog 的形状，不是这一族。
- **两个 `Render.message` 出口或 block-preview 那一族**：一旦有人准备动
  `core/editor/types.ts` 的契约，就把它们一起收掉——它们和本 ADR 是同一族，
  只是落点在内核那一侧。别让它们再各自排队一轮。
- **`AppErrorShape` 与 `LocalizedText` 的合并**：`localizedError` /
  `localizedConflict` 两个 getter 的重审条件（见 `document/store.ts`）
  仍然有效——`openError` 与 `core/search/workspace.ts` 的消费方哪天都进了边界，
  getter 就该退场，两个形状该合成一个。