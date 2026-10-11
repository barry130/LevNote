# ADR 0043 · 降级要说出理由

- 日期：2026-10-11
- 状态：已接受
- 关联：硬约束 2（绝不静默）、ADR 0003、ADR 0040（失败要有出口）、ADR 0041、ADR 0042

## 背景

ADR 0041 清的是**命令**这条线，ADR 0042 清的是**持久化**这条线。这一轮照着同一条结论
再扫一遍「用户的动作没有任何反馈」，扫出来的是第三种形态：

**降级的理由在注册表边界上被丢掉了。**

`EditorAdapter.canHandle(path, text)` 的返回类型是 `boolean`。内核否决自己时，
它知道为什么吗？有一部分内核**知道得很清楚**：

- `modules/mindmap/adapter.ts` 的 `checkKernelSafety(document)` 返回的
  `KernelSafety` 里就带着一串写给人看的中文理由
  （「节点 id `a"b` 里有引号或反斜杠，内核的查找会失败」）；
- 而 `canRenderInKernel(document)` 是它的窄化包装 —— `return checkKernelSafety(document).ok`，
  **那串理由在这一行被扔了**，只剩布尔值流到注册表。

于是外壳那边能说出来的只有一句写死的文案。`ui/DocumentPane.tsx` 的降级横幅
拼的是 `ui.pane.frontMatterWhy1/2/3` —— **整整三行只讲 front matter 这一种原因**。
后果很具体：

> 用户打开一份 id 重复的 `.mindmap`，他的导图被退到了代码内核（JSON 源码）。
> 横幅上写的是「这个文件的开头是 YAML front matter……」。

这句话与眼前的文件**毫无关系**。而「什么都不说」其实更好一点 —— 至少不会骗人。
这正是硬约束 2 要拦的：降级发生了，UI 上也确实有一条横幅，
但那条横幅说的是**别人的理由**，用户拿它去改文件只会越改越远。

三条观察决定了后面怎么修：

1. **理由必须是数据，不能是翻译好的字符串。** `checkKernelSafety` 现在返回中文，
   那是模块加载时（甚至解析时）就定下来的文本。照搬这个形状去做契约，
   会把语言钉死在内核算出来的那一刻 —— 与「组件里的静态清单只存 key、渲染时才 `t()`」
   是同一个坑（ADR 0031 复评记录二）。
2. **理由在内核里算、在外壳里说。** 只有 mindmap 内核知道「哪个 id 重复了」，
   只有外壳知道「现在该用中文还是英文」。所以契约上跑的是一条带参数的**判别联合**，
   外壳负责把它翻成句子。
3. **`resolveAdapter` 的行为一个字节都不能变。** 否决 → 继续找下一个候选 →
   最后才兜底。`canHandle` 抛错仍然按否决处理 + `console.warn`。
   这次加的是**旁支信息**，不是新分支。

## 决策

### 1. 契约上加一条 `canHandleReason`，`canHandle` 保持布尔

```ts
// src/core/editor/types.ts
export type KernelRejectReason =
  | { kind: 'front-matter' }
  | { kind: 'mindmap-invalid' }
  | { kind: 'mindmap-unsafe-id'; id: string }
  | { kind: 'mindmap-duplicate-id'; id: string };

interface EditorAdapter {
  canHandle?(path: string, text: string): boolean;
  canHandleReason?(path: string, text: string): KernelRejectReason | null;
}
```

`canHandleReason` 返回 `null` = 「我接得住」。**刻意不把它并入 `canHandle`**
（即不让 `canHandle` 返回 `{ok:true} | {ok:false, reason}`）：

- 现有七处调用点（注册表、测试里的 stub、接线测试）全部按布尔写，合并会让它们
  一个个改过去，而**那些改动没有任何一行是这次想要的**；
- 更重要的是，两个方法会**各算一遍**（一次判定、一次取理由）——
  而 mindmap 那次要解析整篇 JSON。判据与理由必须是**同一次计算的两半**，
  否则「说出来的理由」迟早会与「实际走过的分支」不一样。

所以内核侧的形状是：**一个 `describe…` 纯函数一次算完，两个契约方法各自从结果里取值**
（`modules/mindmap/index.ts` 的 `describeMindmapVeto(text)`）。

### 2. 外壳按 `kind` 选文案，front matter 那一条保持原样

`DocumentPane` 的 `resolveAdapter` 旁多问一次首选内核（按扩展名匹配的那一个）
的 `canHandleReason`，把 reason 存进 `downgraded`。横幅分两支：

- `kind === 'front-matter'` → 现有的三行（它讲的是「保存会毁掉元数据」，值得说透）；
- 其余 → 一句通用句「这份文件不能用「⟨内核名⟩」打开：⟨理由⟩。」 +
  「现在以「⟨内核名⟩」打开，内容不会有任何改动。」

新增 key：`ui.pane.downgradeWhy` / `ui.pane.downgradeWhyFrontMatter` /
`ui.pane.whyUnsafeId` / `ui.pane.whyDuplicateId` / `ui.pane.whyInvalidMindmap`。
带 `id` 的两种把 id 用 `{id}` 插值出来 —— **点名那个 id 才让用户知道去改哪里**。

`canHandleReason` 抛错按「没有理由」处理（横幅退回通用句），
**绝不因此不开文件** —— 与注册表里 `canHandle` 抛错的取舍一致。

### 3. mindmap 内核：`checkKernelSafety` 改为返回判别联合，理由**不再是中文字符串**

```ts
export type KernelSafety =
  | { ok: true }
  | { ok: false; reason: KernelRejectReason };
```

`checkKernelSafety` 的两条 walk 分支现在返回 `{ kind: 'mindmap-unsafe-id', id }` 与
`{ kind: 'mindmap-duplicate-id', id }`。**这是本轮唯一一处「删掉中文」的改动** ——
那两句中文原本就没活着出去过（被 `canRenderInKernel` 的 `.ok` 滤掉了），
现在它们以 `MessageKey` 参数的形式重新出现在 UI 上，而且有了英文版。

`KernelRejectReason` 定义在 `core/editor/types.ts`（契约层），
mindmap 内核 import 它 —— 方向仍然是 `modules/` → `core/`（硬约束 1）。
Reason 里带 mindmap 专属的 kind 不违反「契约不认识具体内核」：
契约认识的是**一类降级原因**（与 `LinkTarget` 认识 `anchor` / `file` / `external`
是同一种写法），不是 mind-elixir 的任何东西。

## 后果

- 好：任何一次 `canHandle` 否决都有话可说，且说的是**这一份文件**的原因。
- 好：理由是可单测的纯数据（`describeMindmapVeto` 直接断言 `kind`），
  不需要跑 UI 才能验证「说对了没有」。
- 好：文案随语言切换（`kind` 是数据，渲染时才 `t()`）。
- 代价：`canHandle` 与 `canHandleReason` 是两次调用。mindmap 侧靠
  `describeMindmapVeto` 返回的**同一个对象**避免解析两遍 JSON ——
  真正的成本只是一次字符串比较。
- 代价：契约上多了一个可选方法。**不实现 = 「没有理由可说」**，
  外壳退回通用句 —— 与 `EditorHandle.search` 那类可选能力的处理一致。

## 重审条件

- **出现第三个会否决自己的内核**（比如将来的 draw.io 独立内核）：
  它的 reason kind 要往 `KernelRejectReason` 里加 —— 加到**五个以上**时，
  把「理由 → 文案」的映射从 `DocumentPane` 里挪成 `core/i18n/` 旁边的一张表，
  别让组件里堆一串三元。
- **外壳开始按 `kind` 分支做事而不只是选文案**（比如「id 重复」给一个「去修 id」按钮）：
  那就越过了「外壳只问契约、不问内核」的线 ——
  那时应当把动作也做成契约（`EditorAdapter.repair?`），而不是在外壳里认 kind。
- **有用户报告「降级横幅说的与实际不符」**：说明 `canHandle` 与 `canHandleReason`
  在某个内核里算歪了 —— 先去查那个内核是不是没走「一次计算两处取值」的形状。
- **理由开始需要带参数之外的东西**（比如「第 12 行有问题」）：
  在 `KernelRejectReason` 上加字段，不要退化成返回翻译好的字符串 ——
  那会把语言钉死在计算那一刻（本 ADR 背景里的观察 1）。
