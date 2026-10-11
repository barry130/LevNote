# 0042 · 偏好写不进去：把「下次启动就没了」说出来

日期：2026-10-10
状态：已采纳
关联：0003（不静默，是本 ADR 的母题）、0021（位置记忆与设置页）、
0023（会话恢复：写不进去要交回外壳 —— 本 ADR 是它的推广）、
0040 / 0041（成功的分支安静、失败各自点名 —— 本 ADR 沿用同一取舍）
AI助手手册硬约束 2（绝不静默）

## 背景

0041 收尾时留下同一句判断：**那一轮清的是「命令」这一条线，别的线还没清。**

这一轮清点的是**持久化**。全仓库只有四个模块碰存储
（`grep 'getItem(|setItem('`）：`core/settings.ts`、`core/positions.ts`、
`core/recent.ts`、`core/session.ts` —— 没有任何 adapter 碰它，所以边界是清楚的。

其中 `session.ts` 上一轮已经做对了：`SaveOutcome { state, ok }`，外壳读到
`!ok` 就翻横幅。剩下三份**全是静默的**，而且静默的理由是同一句注释：

```ts
catch {
  // 配额用尽 / 被禁用：静默放弃。偏好没存上不是需要打断用户的事。
}
```

**这句话对「打断」是对的，对「不说话」是错的。**

它混淆了两件事：

- **打断**（弹模态框、拦住下一步操作）—— 偏好没存上确实不该打断；
- **沉默**（一句话都没有）—— 用户改了主题，界面立刻变了，于是**他相信改成了**；
  下次启动回到默认值，他只会以为「应用没记住我的设置，大概就这样吧」。

这是一次**静默降级**：界面明明是「改好了」的样子，真相是「只在内存里改好了」。
`core/session.ts` 那份草稿是**用户刚写的字**，写不进去要说没人反对；
而**设置 / 位置 / 最近打开**写不进去，丢的是「用户刚刚做出的选择」——
同一个硬约束，只是代价小一档。小一档影响的是**怎么说**（横幅而不是弹窗），
不是**说不说**。

### 刻意排除的静默类别（写下来，免得下一轮再来一遍）

- **读失败回落**：`loadSettings` / `loadPositions` / `loadRecentFiles` 抓到异常
  就回落默认值。**读有默认值可依，写没有** —— 读不出来最坏是「回到默认」，
  而那个默认值用户看得见、能再改一次；写不进去是「改了、看不见、无从补救」。
  唯一动了一处读：`recent.ts` 的 `getItem` 原来连 try/catch 都没有，
  那个 throw 会从 `App.tsx` 的 `useState` 初始化器穿出去 —— 那是**启动白屏**，
  比写失败严重得多，所以顺手补上（有单测钉住）。
- **后台 / 恢复的 catch**（`core/document/store.ts`、`core/workspace/store.ts`）：
  绝大多数已经 `notify` 了；没说的那几处是「后台自动重试，下次刷新会好」，
  与 0041 里「刻意不动」的第一类同理。
- **几何 / 选区的 catch**（`modules/markdown/block.ts` / `selection.ts` /
  `image-view.ts`、`core/search/find.ts`、`core/highlight/hljs.ts`）：
  它们失败的表现是「这一下没算出坐标 / 没高亮」，**界面立刻就是证据**，
  不需要另一句话来复述用户已经看见的东西。
- **`core/export/*`**：已有自己的兜底与失败出口（0041 决策 4 里点过名）。

## 决策

### 1. 返回 `PersistOutcome`，而不是吞掉

`src/core/settings.ts`：

```ts
export type PersistFailure = 'no-storage' | 'write-failed';
export type PersistOutcome = { readonly ok: true } | { readonly ok: false; readonly reason: PersistFailure };
export const PERSISTED: PersistOutcome = { ok: true };
export function persistFailed(reason: PersistFailure): PersistOutcome
```

三份写入（`saveSettings` / `savePositions` / `saveRecentFiles` / `clearRecentFiles`）
一律改成这个形状，**都不抛**。

> 判断依据（与 0041 决策 5 同一条）：**一个会失败的能力，它的类型不许是 `void`。**
> 类型上抹掉返回值，等于在编译期就宣布「这个失败无处可去」。

### 2. `no-storage` 与 `write-failed` 必须分开 —— 这是本 ADR 最吃重的一条

- **`no-storage`**：`storage` 参数为 `null`（浏览器预览 / 隐私模式 /
  企业策略整体禁用）。**这是预期内的降级**：那个环境本来就不打算持久化，
  轻笺在浏览器预览里也确实照常能用。为它弹一句「没能记住」，
  等于**每改一次设置弹一次**，那样的横幅三次之后就没人看了。
- **`write-failed`**：有存储、但 `setItem` 抛了（配额用尽 / 策略禁写）。
  这才是真的坏了，要说。

合成一个 `ok: false` 会直接毁掉浏览器预览的可用性 ——
**「宁可多一句」在这里不成立**，因为那不是「多一句」，是「每次都有一句」。

（会话那份 `SaveOutcome` 保持只有一个 `ok`，不拆 reason：它只有一处调用方，
且那份存的是草稿，无论哪种失败都值得说。）

### 3. 四处出口汇成一句话，不各弹各的

外壳加一个 `reportPersistFailure(outcome)`（`src/ui/App.tsx`），
设置 / 位置 / 位置清空 / 最近打开**全部**走它：

```ts
const reportPersistFailure = useCallback((outcome: PersistOutcome) => {
  if (outcome.ok || outcome.reason === 'no-storage') return;
  setSessionWarning((warn) => (warn.writeFailed ? warn : { ...warn, writeFailed: true }));
}, []);
```

两条理由：

- **它们说的是同一件事** —— 「你刚才那个改动下次启动就没了」。
  用户不需要知道四份数据里没写进去的是哪一份；分成四条提示，
  他得自己拼出「哦，是存储坏了」。
- **幂等**（`warn.writeFailed ? warn : …`）：位置记忆每 600ms 写一次，
  配额满了会**每一次都失败**。不幂等就是每 600ms 重渲染一次整个外壳，
  而横幅一字不变。有外壳用例钉住「写 N 次、横幅只有一条」。

**复用会话横幅那一枚 `writeFailed`，不另开一条横幅。**
（试过新增 `StorageWarning { writeFailed, at: … }`，插进去又删掉两次。）
两条横幅并排讲同一句「存储坏了」，只会让人以为坏的是两处 ——
而 0023 那条横幅本来就是「存储 / 会话这一族」的出口。

### 4. 副作用从 `setSettings` 的 updater 里搬出来

原来：

```ts
setSettings((previous) => {
  const next = { ...previous, ...patch };
  saveSettings(storage, next);   // ← 副作用在 updater 里
  return next;
});
```

updater 在 StrictMode 下会被**跑两次**（写两次盘、也说两次），
更要紧的是**返回值会被吞掉** —— 而这一轮要的正是那个返回值。

改后：`settingsRef` 存当前一份（`useRef` + 每次渲染同步，不进依赖表，
免得 `update` 的身份每次改设置都变、把整张命令表一起重建），
然后 `算 next → setSettings(next) → saveSettings → reportPersistFailure`。

### 5. 文案扩到「偏好与会话」两类后果

`ui.app.storageBrokenTitle` / `Body` 原来只讲「会话记录 / 未保存的草稿」，
现在这枚标记也管偏好，所以文案改成两条后果都列：

- zh：**偏好和会话都没能记住** ——「（存储被禁用或已写满）。刚才改的设置、
  最近打开的文件、以及未保存的草稿，下次启动都不会在 —— 请先手动保存正在编辑的内容。」
- en：**Your changes could not be remembered** —— 同上。

横幅照旧是 `sessionWarning` 那一族，关掉就没了（不钉在屏幕上）。

### 6. 测试要能走到这条路径 —— 于是给了 `AppProps.storage`

jsdom 的 localStorage **永远写得进去**，而 App 原来自己调 `defaultStorage()`：
没有注入点，这条出口就没有一条用例能碰到它（写了也等于没写）。

新增 `AppProps.storage?: StorageLike | null`，不给时仍是 `defaultStorage()`。
**只有测试用它** —— 生产入口 `main.tsx` 一个字没动。

## 后果

- **类型新增**：`PersistOutcome` / `PersistFailure` / `PERSISTED` / `persistFailed`
  住在 `core/settings.ts`（`StorageLike` 本来就在这里，三处都从它导入）。
- **四个函数改签名**（都从 `void` 变 `PersistOutcome`）：`saveSettings`、
  `savePositions`、`saveRecentFiles`、`clearRecentFiles`。
- **外壳新增 `settingsRef` + `reportPersistFailure`**，五处写偏好全部接上；
  横幅文案两枚（zh + en）扩写。
- **测试**：core 侧三份各补 2–4 条（成功返回 `ok`、`write-failed`、
  `no-storage` 静默，以及 `recent` 读抛不再穿出初始化器）；
  外壳侧新增一组 8 条「偏好写不进去（ADR 0042）」。
- **`clearRecentFiles` 目前没有 UI 调用方**（grep 只命中 `recent.ts` /
  `recent.test.ts`）。它的返回值照样接上出口 —— 将来接上 UI 时不用再想起这一轮。

## 重审条件

- **出现「四份数据要说不同的话」时**：现在它们共用一枚标记、一句文案。
  若将来某一份写失败需要**不同的处置**（比如「会话草稿写不进去」要立刻提示保存，
  而「位置写不进去」完全无所谓），那时才把 `reportPersistFailure` 拆开 ——
  拆的依据是**处置不同**，不是「想知道是哪一份失败了」。
- **横幅被抱怨「关掉了又回来」时**：现在「写失败」翻的是 `sessionWarning` 里
  一枚**持久**的标记，关掉之后下一次写失败会再翻起来（配额满时会一直弹）。
  真到那一步，正确做法是给「本次会话已经说过」加一个一次性标记，
  而不是把 `writeFailed` 改成 transient 提示 —— 后者会让
  「用户没在看的时候失败了」重新变回沉默。
- **`no-storage` 需要被看见时**（比如将来「浏览器预览」要变成一等支持的目标，
  而用户开始抱怨「设置没记住」）：那时才给 `no-storage` 一个**一次性**的、
  更轻的出口（比如状态栏一小行），**不要**把它并进 `write-failed` 的横幅 ——
  那正是本 ADR 决策 2 要挡的事。
- **存储迁移到别处时**（Tauri 的 store 插件 / 文件落地）：`PersistOutcome`
  这一层不变，变的是 `StorageLike` 的实现与 `defaultStorage()`。
  `no-storage` / `write-failed` 的划分要跟着重新问一遍：
  新的存储有没有「预期内不可用」这一档。
- **偏好变成「必须落地」的数据时**（比如工作区布局同步到云端）：
  那时写失败就不再是「横幅说一句」，而是要**阻止**或**重试** ——
  那属于 0003（不静默转码）那一族的议题，另开一条记录，不要往这里加参数。
