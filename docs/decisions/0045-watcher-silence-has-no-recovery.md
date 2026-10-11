# 0045 · 监听没起来要说出来：丢的不是数据，是「早点知道的机会」

- 日期：2026-10-12
- 状态：已接受
- 关联：硬约束 2（绝不静默）、ADR 0041（失败要有出口 —— 本 ADR 是它在前端
  最后一条 `console.warn` 上的应用）、ADR 0043（降级要说出理由）、
  ADR 0044（读侧出口 —— 那一族讲的是「上次存的这次没了」，本条讲的是
  「**此刻正在丢的那道安全网没了**」）

## 背景

0041–0044 这一族把 `core/` 里每一处「会失败却静默」的地方都清了。
清完之后再回头扫，剩下三处全在**同一个地方** —— `src/ui/App.tsx` 的文件监听：

```ts
// 事件批次自带的后端错误（后端是故意带上这个字段的）
if (batch.error !== undefined) console.warn('文件监听出错：', batch.error);
// 这次没能监视上的路径
if (failed.length > 0) console.warn('这些路径没能监视…', failed);
// 而这两个 promise 本身都 void 掉了，连一个 catch 都没有
void onFilesChanged(…); void watchFiles(paths).then(…);
```

三处都是同一种错：**发布版里 `console.warn` 等于把话写进一个没人会打开的盒子。**
Rust 侧特意在 `FsChangedBatch` 上留了 `error: Option<String>` 字段（`watch/mod.rs`），
`watch_replace` 也特意把失败路径**作为返回值**交回来 —— 前端把两份交付都扔进了控制台。

这一族别的成员都有「当场可见」的优点：保存失败时用户刚按了保存，
命令失败时用户刚点了一下，**失败会自己跳到眼前**。
监听失败不会。用户正在丢的东西是「磁盘上被别人改了什么」：

- 目录树不刷新，别人新建的文件不出现；
- 冲突检测不再报警，用户可以把别人的改动**整份覆盖掉**。

更要命的是它**不会自愈**。保存失败下一次就可能成功；而监听一旦停摆 ——
目录被删掉、inotify / kqueue 的监视额度耗尽 —— 事件就**永久**不再到达，
兜底那两条路（下一批事件、窗口聚焦时的全量比对）也都建立在
「事件还会来」这个前提上。

「不弹横幅，只说一句原因」这个理由是**从 ADR 0041 抄来的**，
在那里它成立（失败了就当场失败了，提醒只是补充）；
在这里它是错的 —— 没有横幅意味着「安全网没了」这件事可以一直不被知道。

## 决策

### 1. 监听状态是一个值，不是一串副作用

外壳多了一个状态：

```ts
interface WatchIssue {
  readonly reason: string | null;          // 后端报的原因原文；null = 这一路没问题
  readonly failedPaths: readonly string[]; // 没能监视上的路径
}
```

`reason` 与 `failedPaths` **各占一个槽，互不相干**：一次后端报错不该顺手抹掉
「哪些路径没监视上」，反之亦然。

### 2. 两个槽都是整份替换，不是累加

`foldWatchIssue(current, patch)` 只覆盖传进来的那一半。这样有两个好处：

- **不会翻倍**：同一个失败重复收到（例如严格模式把 effect 挂两次）只记一次。
- **能自愈**：下一批正常事件把 `reason` 清回 `null`，一次全成功的
  `watch_replace` 把 `failedPaths` 清空。横幅描述的是**此刻**的状态，
  不是一份故障日志 —— 后者只会越积越长，直到没人看。

### 3. `reason` 是三态，不是两态

`null` = 这一路没问题；`''` = **有问题但拿不出原因**；其余 = 原因原文。

压成两态（`''` 同时表示「没事」与「有事但说不出」）会让「拒绝里没有可读的原因」
整条消失。`watchReasonOf` 因此在取不到可读原因时返回 `''` 而不是 `null`：

> **报 0 等于把一次丢失又藏了回去。** 而且比沉默更坏 —— 屏幕上有话，
> 话说的是「一切正常」。

顺带一条：`String(thrown)` 拼出来的 `[object Object]` 对用户没有任何意义，
所以非 Error、非字符串的拒绝一律折成空串，而不是原样贴上去。

因为 `null` 要表示「清空」，`foldWatchIssue` 里判的是 `patch.reason === undefined`
而**不是** `??` —— 用 `??` 的话 `null` 会落回旧值，而 `null` 正是「这一路好了」。

### 4. 两档措辞，不是一档

| 此刻的状态 | 标题 | 配色 |
|---|---|---|
| 后端报了原因 | 「外部改动可能发现不了」 | `warn` |
| 只有若干路径没监视上 | 「有 {n} 条路径没在监视」 | `warn` |

「有几条路径没监视上」多半只是**那几条刚好被删掉或移动了**，
把它也说成「外部改动可能发现不了」是虚惊。而虚惊多了，
用户就会开始忽略这一类横幅 —— 那时真的那一次也没人看。

### 5. 按掉的是「这一份问题」，不是一个布尔

`dismissedWatchIssue` 存的是问题的**指纹**（`watchIssueSignature`：
原因 + 路径，用 NUL 拼接以免撞车），不是一个 `true`。
关掉了 A 不该把之后出现的 B 一起关掉。

与 ADR 0044 的 `readLossDismissed` 只判一次不同，理由是：
**监听故障可以在应用开着的时候发生好几次**，每一次都意味着一段
「安全网没了」的时间。

已知的一处缝隙（问题消失、之后以**完全相同**的样子回来时不再提醒）见重审条件。

### 6. 过期的那一次结果不落地

`watchFiles` 那个 effect 加了 `disposed` 标记。集合换得很快时，
旧的一次 `watch_replace` 可能后到 —— 它的返回值描述的是**上一份**集合，
落进状态就成了「明明全监视上了，却还挂着几条没监视上」。

### 7. 两个 promise 都接上 catch

`onFilesChanged(...)` 与 `watchFiles(paths)` 的拒绝都折成 `reason`。
订阅本身就失败（插件没装上、invoke 被拒）意味着**一个事件都收不到**，
原来那是一条没人处理的拒绝 —— 同样一个字都不说。

### 8. 内核交数据，外壳才翻成话

沿用 ADR 0043：`watch/mod.rs` 只交 `error` 字段与失败路径数组，
一个字的用户可见文案都没有；说哪一档、说哪几条、怎么排序全在 `src/ui/App.tsx`，
key → 话的映射在 `src/core/i18n/messages.ts`。

## 后果

- **外壳新增**（`src/ui/App.tsx`）：`WatchIssue` / `NO_WATCH_ISSUE` /
  `WatchIssuePatch` / `foldWatchIssue` / `hasWatchIssue` /
  `watchIssueSignature` / `watchReasonOf`、两个 `useState`、
  `watchIssueVisible`，以及第三条横幅。
- **两条 effect 改动**：`onFilesChanged` 的 handler 不再 `console.warn`，
  正常批次会把 `reason` 清回 `null`；两个 promise 都接了 `.catch`；
  `watchFiles` 那个 effect 新增 cleanup 置 `disposed`。
- **Rust 侧零改动** —— `error: Option<String>` 与 `watch_replace` 的失败路径
  数组**本来就是为此留的**，只是前端一直没接。这条链上从后端到横幅之间
  唯一缺的就是前端这一段。
- **文案新增 6 枚 × 2 语言**：`ui.app.watchLostTitle` / `watchLostBody` /
  `watchLostBodyNoReason` / `watchLostPaths` / `watchPartialTitle` /
  `watchPartialBody`。说的是**后果**（外部改动不会自己出现）而不是术语
  （inotify / watch）。
- **测试**：`src/ui/App.test.tsx` 新增 9 条「监听没起来（ADR 0045）」，
  含一条**正常时一句都不说** —— 这一族最容易被做坏的地方就是
  「每次开应用都弹一条」。用例用假后端（`mockIPC` + `shouldMockEvents`）
  驱动真实的 effect，退出时把 `__TAURI_INTERNALS__` 两个壳都删掉，
  否则后面的用例会以为自己在桌面环境里。

## 重审条件

- **「同一个问题消失后又原样回来，不再提醒」被抱怨时**：现在指纹相同就被当成
  已处理。补它要么多一次渲染、要么多一个复位点，两个代价都比「少提醒一次
  同样的问题」更贵。真的被抱怨时，用「指纹 + 一个计数」表示，
  而不是退回布尔。
- **Tauri 侧改成插件式监听（或换回轮询）时**：`watch/mod.rs` 里
  `app.emit("fs://changed", batch)` 那句 `let _ =` 仍然是被丢在地上的一句 ——
  「检测到改动、通知发不出去」在本 ADR 之后应该由 Rust 侧自己记日志，
  前端这条横幅只能覆盖「后端知道自己错了」这一半。
- **需要按严重度决定是否拦截保存时**（例如监听已断时对「磁盘上被改过」的文档
  多问一句）：那时这条横幅要升级成 0003 那一族的「先处理再继续」，
  另开一条记录，不要往这里加参数。
- **`watchReasonOf` 的字符串分支被要求单测时**：真机上的 Tauri 确实用**字符串**
  拒绝，而 `@typescript-eslint/prefer-promise-reject-errors` 与 `only-throw-error`
  都禁止在测试里造出这种拒绝 —— 现在用「消息为空的 `Error`」覆盖同一句文案。
  若将来这条分支需要单测，办法是把它导出成一个纯函数单独测，而不是放宽 lint。