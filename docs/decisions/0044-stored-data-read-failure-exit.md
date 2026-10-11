# 0044 · 存储里的东西读不出来：把「上次存的这次没了」说出来

- 日期：2026-10-12
- 状态：已接受
- 关联：硬约束 2（绝不静默）、ADR 0003、ADR 0021（位置记忆）、
  ADR 0023（会话恢复：草稿丢了要交回外壳 —— 本 ADR 是它的另一半）、
  ADR 0042（写侧出口 —— 本 ADR 是它的对偶）、ADR 0043（降级要说出理由）

## 背景

ADR 0042 清掉了**写侧**的静默：写入一律返回 `PersistOutcome`，
`write-failed` 要说话，`no-storage` 安静。

它在「刻意排除的静默类别」里写下了一句排除理由，本轮就是来撤销它的：

> **读失败回落**：`loadSettings` / `loadPositions` / `loadRecentFiles` 抓到异常
> 就回落默认值。**读有默认值可依，写没有** —— 读不出来最坏是「回到默认」，
> 而那个默认值用户看得见、能再改一次；写不进去是「改了、看不见、无从补救」。

这句推理在**偏好**上成立，在**会话**上不成立：

| 读不出来的是 | 回落成 | 用户的损失 |
|---|---|---|
| 偏好 | 默认值 | 看得出变了，能再改一次 —— 可补救 |
| 最近打开 | 空列表 | 菜单里少几条，重新打开就有 —— 可重建 |
| 位置记忆 | 文档开头 | 「上次读到哪儿」没了 —— 可重建 |
| **会话** | **空会话** | **上次开着的标签页、当时的路径、以及其中未保存的草稿全没了** |

最后一行是 `core/session.ts` 里 `droppedDrafts` 那一族要解决的**同一件事**：
0023 已经决定「草稿丢了要说」，而 `droppedDrafts` 只在**逐条校验**时计数。
存储里那一整段直接不是 JSON 时，四个读取器一齐 `catch → 回落到空会话`，
**连计数都进不去** —— 最贵的一次丢失，是唯一一次完全无声的。

所以本 ADR 的取法与 0042 对称：**四份存储一起改，不是只改偏好那一份**。
只改偏好会留下一条刺眼的对比 ——「主题坏了会说话，上次的草稿坏了不会」。

## 决策

### 1. 读侧与写侧对称的一套出口：`LoadOutcome`

`src/core/settings.ts`，紧挨着 `persistFailed`：

```ts
export type LoadFailure = 'no-storage' | 'read-failed' | 'corrupt';
export type LoadOutcome =
  | { readonly ok: true }
  | {
      readonly ok: false;
      readonly reason: LoadFailure;
      readonly lostKeys: readonly string[];
      readonly lostCount: number;
    };
export interface LoadResult<T> { readonly value: T; readonly outcome: LoadOutcome }
```

四个 `load*` 全部改成返回 `LoadResult<T>`：`loadSettings`、`loadPositions`、
`loadRecentFiles`、`loadSession`。

`lostKeys`（丢了哪些键）与 `lostCount`（一共丢了多少）是两件事，分开是因为
**键名不一定点得出来名**（见决策 3）。

> 判断依据与 0042 决策 1 同一条：**一个会失败的能力，它的类型不许把失败抹掉。**
> 原来四个 `load*` 的返回类型都是裸的 `T`，失败在类型上就无处可去。

### 2. 三档 reason，`no-storage` 继续安静

沿用 0042 决策 2 的分档，并给读侧补了第三档：

- **`no-storage`**：`storage` 参数为 `null`。**预期内的降级**，折成「没丢」——
  压根没有存储 = 从来没存过东西 = 没有东西可丢。浏览器预览里每开一次文档就弹一句
  「没读回来」，那种横幅三次之后就没人看了。
- **`read-failed`**：有存储，但 `getItem` 抛了（隐私模式 / 企业策略 / 配额）。
  这一档是 0042 排除那段里点名过的「启动白屏」路径 —— 回落层保留，
  但**回落不等于沉默**。
- **`corrupt`**：读到了，内容用不了。这是本 ADR 的新增档，也是绝大多数真机情形
  （写到一半被打断、手改坏、换版本改了结构）。

「读有默认值可依」那句话在 `settings` 上仍然有参考价值，所以**值照样回落** ——
本 ADR 从头到尾没有改任何一处回落行为，改的只是「回落后说不说」。

### 3. 什么才算「丢失」—— 一条规则，不靠临场发挥

这是本 ADR 最吃重的一条，也是最容易做错的一半：

> **键在、值用不了 = 丢失；键不在 = 从来没有过，不是丢失。**

三条推论：

- **版本升级新增的字段不报。** 存储里只有 `theme`，其余 16 项是这一版才加的 ——
  那些字段是**从来没存过**，不是「存了没读回来」。若把缺失也算丢失，
  以后每加一个偏好项就得给全体用户推一次告警。
- **夹取（clamping）不报。** `splitRatio: 9` 被夹回上限、`editorFontSize: 999`
  被夹回 28 —— 用户的**意图已部分兑现**，说「丢了」反而是撒谎。
  `core/positions.ts` 里比例越界也照样留条目，是同一条理由。
- **字体族消毒不报。** `'x} y' → 'x y'` 是消毒，条目还在。

落到实现上是一条 `pickField`：

```ts
function pickField<T>(key, raw, parse, fallback, lost): T {
  if (raw === undefined) return fallback;   // 键不在：安静
  const parsed = parse(raw, lost);
  if (parsed !== null) return parsed;       // 用得上（含夹取）：安静
  lost.push(key);                           // 键在、用不了：点名
  return fallback;
}
```

「键在、值用不了」与「键不在」在 JS 里就是 `undefined` 与「不是 `undefined`」
的区别 —— 整条规则因此可以由类型钉住，而不是靠每个调用点自己判断。

### 4. 点名还是只报数，由键名决定

外壳 `src/ui/App.tsx` 的 `foldReadLoss`：

- **位置**：`lostKeys` 是**文件路径**，人能读懂 → 点名（`basename` 之后）。
  但位置能记 200 条，所以封顶 `READ_LOSS_NAMES_MAX = 3`，其余进计数
  （文案 `等 {n} 份`）。**全列出来就是一面墙，墙和沉默在效果上一样。**
- **偏好**：`lostKeys` 是 `sidebarWidth` / `keybindings.bold` 这类内部标识，
  用户看到只会更困惑 → **只报数**。
- **最近打开**：条目本身没有名字（「第 3 项」对用户没有意义）→ 只报数。
- **会话**：整份不可用就是整份不可用 → 一个布尔。

这条规则与 ADR 0041「成功的那支安静、失败的那支点名」是同一条纪律的延伸：
**点名是为了让用户能补救**。点不出名的，报数也要报 —— 见下一条。

### 5. 整份读不出来时，`lostCount` 一律写死 1

`loadLost` 的第四个参数是「显式报数」。四个调用点在整份报废时**一律传 1**，
且有单测钉住。

理由：整份报废时 `lostKeys` 必然是空的（点不出名），若让 `lostCount` 默认成
`lostKeys.length`，用户那边读到的就是「**丢了 0 项**」——
那等于把一次丢失又藏了回去，而且比原来的沉默更坏：屏幕上有话，话说的是假的。

### 6. 会话只报「整份报废」，不与 `droppedDrafts` 重复

`loadSession` 的逐条丢弃继续走 `droppedDrafts`（它有自己那条 0023 的横幅）。
若 `LoadOutcome` 再说一次，同一件事会有两句话。

`loadSession` 因此**只**在三种情况下报：不是 JSON、不是对象、`tabs` 不是数组 ——
这三种都是整份报废，与逐条无关。

### 7. 与写侧**分两条**横幅，不合并

写侧的 `sessionWarning` 横幅讲的是「你刚才那个改动下次启动就没了 / 存不下」；
读侧这条讲的是「上次存的东西这次读不出来」。**这是两件事**：

- 「该保存草稿」与「该检查存储」是完全不同的处置；
- 合成一条会逼用户在两者之间猜是哪一件。

两条并排，且按**丢失的代价**排序：会话（未保存的草稿）→ 位置 → 偏好 → 最近打开。
会话那条同时是 `error` 配色，其余是 `warn`。

### 8. 内核交数据，外壳才翻成话

沿用 ADR 0043 的分工：`core/` 里的四个 `load*` 只产出 `LoadOutcome`，
**一个字的用户可见文案都没有**；key → 话的映射全在 `src/core/i18n/messages.ts`，
决定「说哪几条、按什么顺序」全在 `src/ui/App.tsx`。

## 后果

- **类型新增**（`core/settings.ts`）：`LoadFailure` / `LoadOutcome` / `LOADED` /
  `LoadResult` / `loaded` / `loadLost`。
- **四个函数改签名**：`loadSettings` / `loadPositions` / `loadRecentFiles` /
  `loadSession`，从裸 `T` 变成 `LoadResult<T>`。**这是破坏性签名变更**，
  四个调用点全在 `App.tsx`。
- **`settings.ts` 内部重写**：私有的 `clampSidebar` / `asBoolean` /
  `clampSplitRatio` / `asKeybindingOverrides` 换成返回 `T | null` 的
  `parse*` 系列 + `pickField`。**数组按整份报废算**（`[1,2,3]` 会把下标当成
  路径键，报出来的是 `0`、`1`、`2` 这种没人看得懂的名字）。
- **外壳新增**：`readLossRef` + `foldReadLoss` + `hasReadLoss` +
  `describePositionNames`，以及第二条横幅与它的 dismiss 标记。
  `readLossRef` 放在设置初始化器**之前**，且取值的时机放在位置读取**之后**
  （位置是四份里最后读的）。
- **文案新增 10 枚 × 2 语言**：`ui.app.readLostSession*` /
  `readLostPositions*` / `readLostPositionNames` / `readLostMore` /
  `readLostSettings*` / `readLostRecent*`。
- **测试**：core 侧四份把「静默回落」的老断言改成断言 `value` **与** `outcome`，
  并各补 1–3 条 outcome 断言（首启静默 / 缺键静默 / 夹取静默 / 整份报废报 1 /
  逐条点名）；外壳侧新增 `mount` 选项 `readSettingsFromStorage`
  （不注入 `initialSettings`，让 App 走真实的 `loadSettings`）与一组 14 条
  「存储里的东西读不出来（ADR 0044）」。

## 重审条件

- **「有东西没读回来」需要不同的处置时**（比如「位置没读回来」其实完全无所谓，
  值得说的只有会话与偏好）：那时才按 `ReadArea` 拆开横幅 —— 拆的依据是
  **处置不同**，不是「想知道是哪一份丢了」。这与 0042 重审条件第一条同形。
- **`readLossDismissed` 被抱怨「关掉了又回来」时**：现在它是本次会话一次性
  的，重渲染不会翻起来（`readLoss` 存在 ref 里，不再读存储）。若将来加了
  「运行中检测到存储坏了也要报」，那时才需要按次区分。
- **存储迁移到别处时**（Tauri 的 store 插件 / 文件落地）：`LoadOutcome` 这一层
  不变，变的是 `StorageLike` 的实现。`read-failed` / `corrupt` 的划分要跟着
  重新问一遍 —— 文件系统有「临时打不开」和「内容真的坏了」两种吗？
- **偏好项需要「必须落地」时**（比如工作区布局同步到云端）：那时读失败就不再是
  「横幅说一句」，而是要**阻止进入**或**从别处恢复**。那属于 0003 那一族，
  另开一条记录，不要往这里加参数。